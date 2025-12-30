package gpuoperator

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/common"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// Bundler creates GPU Operator application bundles based on recipes.
type Bundler struct {
	config *config.Config
}

// NewBundler creates a new GPU Operator bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	if conf == nil {
		conf = config.NewConfig()
	}

	return &Bundler{
		config: conf,
	}
}

// Make generates the GPU Operator bundle based on the provided recipe.
func (b *Bundler) Make(ctx context.Context, recipe *recipe.Recipe, dir string) (*common.Result, error) {
	// Check for required measurements
	if err := recipe.ValidateMeasurementExists(measurement.TypeK8s); err != nil {
		return nil, fmt.Errorf("measurements are required for GPU Operator bundling: %w", err)
	}

	// Check for GPU measurements (optional but recommended)
	if err := recipe.ValidateMeasurementExists(measurement.TypeGPU); err != nil {
		slog.Warn("GPU measurements not found in recipe", "warning", err)
	}

	start := time.Now()
	result := common.NewResult(common.BundleTypeGpuOperator)

	slog.Debug("generating GPU Operator bundle",
		"output_dir", dir,
		"namespace", b.config.Namespace,
	)

	// Create bundle directory structure
	dirManager := common.NewDirectoryManager()
	bundleDir, subdirs, err := dirManager.CreateBundleStructure(dir, "gpu-operator")
	if err != nil {
		return result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	scriptsDir := subdirs["scripts"]
	manifestsDir := subdirs["manifests"]

	// Prepare configuration map
	configMap := b.buildConfigMap()

	// Initialize utilities
	fileWriter := common.NewFileWriter(result)
	contextChecker := common.NewContextChecker()
	templateRenderer := common.NewTemplateRenderer(GetTemplate)

	// Generate Helm values
	if err := b.generateHelmValues(ctx, recipe, bundleDir, configMap, contextChecker, templateRenderer, fileWriter); err != nil {
		return result, err
	}

	// Generate ClusterPolicy manifest
	if err := b.generateClusterPolicy(ctx, recipe, manifestsDir, configMap, contextChecker, templateRenderer, fileWriter); err != nil {
		return result, err
	}

	// Generate installation scripts
	if b.config.IncludeScripts {
		if err := b.generateScripts(ctx, recipe, scriptsDir, configMap, contextChecker, templateRenderer, fileWriter); err != nil {
			return result, err
		}
	}

	// Generate README
	if b.config.IncludeReadme {
		if err := b.generateReadme(ctx, recipe, bundleDir, configMap, contextChecker, templateRenderer, fileWriter); err != nil {
			return result, err
		}
	}

	// Generate checksums file
	if b.config.IncludeChecksums {
		if err := b.generateChecksums(ctx, bundleDir, result); err != nil {
			return result, err
		}
	}

	result.Duration = time.Since(start)

	// Mark the result as successful
	result.MarkSuccess()

	slog.Info("GPU Operator bundle generated",
		"files", len(result.Files),
		"size_bytes", result.Size,
		"duration", result.Duration.Round(time.Millisecond),
	)

	return result, nil
}

// buildConfigMap creates a configuration map from bundler config.
func (b *Bundler) buildConfigMap() map[string]string {
	config := make(map[string]string)
	config["namespace"] = b.config.Namespace
	config["helm_repository"] = b.config.HelmRepository
	config["helm_chart_version"] = b.config.HelmChartVersion

	// Add custom labels and annotations
	for k, v := range b.config.CustomLabels {
		config["label_"+k] = v
	}
	for k, v := range b.config.CustomAnnotations {
		config["annotation_"+k] = v
	}

	return config
}

// generateHelmValues generates Helm values file.
func (b *Bundler) generateHelmValues(ctx context.Context, recipe *recipe.Recipe,
	bundleDir string, config map[string]string, contextChecker *common.ContextChecker,
	templateRenderer *common.TemplateRenderer, fileWriter *common.FileWriter) error {

	if err := contextChecker.Check(ctx); err != nil {
		return err
	}

	helmValues := GenerateHelmValues(recipe, config)

	if errValidate := helmValues.Validate(); errValidate != nil {
		return errors.Wrap(errors.ErrCodeInvalidRequest, "invalid helm values", errValidate)
	}

	content, err := templateRenderer.Render("values.yaml", helmValues.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render values template", err)
	}

	filePath := filepath.Join(bundleDir, "values.yaml")
	if err := fileWriter.WriteFileString(filePath, content, 0644); err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to write values file", err)
	}

	return nil
}

// generateClusterPolicy generates ClusterPolicy manifest.
func (b *Bundler) generateClusterPolicy(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string, contextChecker *common.ContextChecker,
	templateRenderer *common.TemplateRenderer, fileWriter *common.FileWriter) error {

	if err := contextChecker.Check(ctx); err != nil {
		return err
	}

	manifestData := GenerateManifestData(recipe, config)

	content, err := templateRenderer.Render("clusterpolicy", manifestData.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render clusterpolicy template", err)
	}

	filePath := filepath.Join(dir, "clusterpolicy.yaml")
	if err := fileWriter.WriteFileString(filePath, content, 0644); err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to write clusterpolicy file", err)
	}

	return nil
}

// generateScripts generates installation and uninstallation scripts.
func (b *Bundler) generateScripts(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string, contextChecker *common.ContextChecker,
	templateRenderer *common.TemplateRenderer, fileWriter *common.FileWriter) error {

	if err := contextChecker.Check(ctx); err != nil {
		return err
	}

	scriptData := GenerateScriptData(recipe, config)

	// Generate install script
	installContent, err := templateRenderer.Render("install.sh", scriptData.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render install script", err)
	}

	installPath := filepath.Join(dir, "install.sh")
	if err := fileWriter.WriteFileString(installPath, installContent, 0755); err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to write install script", err)
	}

	// Generate uninstall script
	uninstallContent, err := templateRenderer.Render("uninstall.sh", scriptData.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render uninstall script", err)
	}

	uninstallPath := filepath.Join(dir, "uninstall.sh")
	if err := fileWriter.WriteFileString(uninstallPath, uninstallContent, 0755); err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to write uninstall script", err)
	}

	return nil
}

// generateReadme generates README documentation.
func (b *Bundler) generateReadme(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string, contextChecker *common.ContextChecker,
	templateRenderer *common.TemplateRenderer, fileWriter *common.FileWriter) error {

	if err := contextChecker.Check(ctx); err != nil {
		return err
	}

	scriptData := GenerateScriptData(recipe, config)

	content, err := templateRenderer.Render("README.md", scriptData.ToMap())
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to render README template", err)
	}

	filePath := filepath.Join(dir, "README.md")
	if err := fileWriter.WriteFileString(filePath, content, 0644); err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to write README file", err)
	}

	return nil
}

// generateChecksums generates a checksums file for bundle verification.
func (b *Bundler) generateChecksums(ctx context.Context, dir string, result *common.Result) error {
	generator := common.NewChecksumGenerator(result)
	fileWriter := common.NewFileWriter(result)

	content, err := generator.Generate(dir, "GPU Operator")
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to generate checksums", err)
	}

	filePath := filepath.Join(dir, "checksums.txt")
	if err := fileWriter.WriteFileString(filePath, content, 0644); err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to write checksums file", err)
	}

	return nil
}
