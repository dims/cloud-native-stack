package gpuoperator

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/internal"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/result"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
	"github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

// Bundler creates GPU Operator application bundles based on recipes.
type Bundler struct {
	*internal.BaseBundler
}

// NewBundler creates a new GPU Operator bundler instance.
func NewBundler(conf *config.Config) *Bundler {
	return &Bundler{
		BaseBundler: internal.NewBaseBundler(conf, types.BundleTypeGpuOperator),
	}
}

// Make generates the GPU Operator bundle based on the provided recipe.
func (b *Bundler) Make(ctx context.Context, recipe *recipe.Recipe, dir string) (*result.Result, error) {
	// Check for required measurements
	if err := recipe.ValidateMeasurementExists(measurement.TypeK8s); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInvalidRequest,
			"measurements are required for GPU Operator bundling", err)
	}

	// Check for GPU measurements (optional but recommended)
	if err := recipe.ValidateMeasurementExists(measurement.TypeGPU); err != nil {
		slog.Warn("GPU measurements not found in recipe", "warning", err)
	}

	start := time.Now()

	slog.Debug("generating GPU Operator bundle",
		"output_dir", dir,
		"namespace", b.Config.Namespace(),
	)

	// Create bundle directory structure
	dirs, err := b.CreateBundleDir(dir, "gpu-operator")
	if err != nil {
		return b.Result, errors.Wrap(errors.ErrCodeInternal,
			"failed to create bundle directory", err)
	}

	// Prepare configuration map
	configMap := b.buildConfigMap()

	// Generate Helm values
	if err := b.generateHelmValues(ctx, recipe, dirs.Root, configMap); err != nil {
		return b.Result, err
	}

	// Generate ClusterPolicy manifest
	if err := b.generateClusterPolicy(ctx, recipe, dirs.Manifests, configMap); err != nil {
		return b.Result, err
	}

	// Generate installation scripts
	if b.Config.IncludeScripts() {
		if err := b.generateScripts(ctx, recipe, dirs.Scripts, configMap); err != nil {
			return b.Result, err
		}
	}

	// Generate README
	if b.Config.IncludeReadme() {
		if err := b.generateReadme(ctx, recipe, dirs.Root, configMap); err != nil {
			return b.Result, err
		}
	}

	// Generate checksums file
	if b.Config.IncludeChecksums() {
		if err := b.GenerateChecksums(ctx, dirs.Root); err != nil {
			return b.Result, errors.Wrap(errors.ErrCodeInternal,
				"failed to generate checksums", err)
		}
	}

	// Finalize bundle generation
	b.Finalize(start)

	slog.Info("GPU Operator bundle generated",
		"files", len(b.Result.Files),
		"size_bytes", b.Result.Size,
		"duration", b.Result.Duration.Round(time.Millisecond),
	)

	return b.Result, nil
}

// buildConfigMap creates a configuration map from bundler config.
func (b *Bundler) buildConfigMap() map[string]string {
	return b.BuildBaseConfigMap()
}

// generateHelmValues generates Helm values file.
func (b *Bundler) generateHelmValues(ctx context.Context, recipe *recipe.Recipe,
	bundleDir string, config map[string]string) error {

	helmValues := GenerateHelmValues(recipe, config)

	if errValidate := helmValues.Validate(); errValidate != nil {
		return errors.Wrap(errors.ErrCodeInvalidRequest, "invalid helm values", errValidate)
	}

	filePath := filepath.Join(bundleDir, "values.yaml")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "values.yaml",
		filePath, helmValues.ToMap(), 0644)
}

// generateClusterPolicy generates ClusterPolicy manifest.
func (b *Bundler) generateClusterPolicy(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string) error {

	manifestData := GenerateManifestData(recipe, config)
	filePath := filepath.Join(dir, "clusterpolicy.yaml")

	return b.GenerateFileFromTemplate(ctx, GetTemplate, "clusterpolicy",
		filePath, manifestData.ToMap(), 0644)
}

// generateScripts generates installation and uninstallation scripts.
func (b *Bundler) generateScripts(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string) error {

	scriptData := GenerateScriptData(recipe, config)
	data := scriptData.ToMap()

	// Generate install script
	installPath := filepath.Join(dir, "install.sh")
	if err := b.GenerateFileFromTemplate(ctx, GetTemplate, "install.sh",
		installPath, data, 0755); err != nil {
		return err
	}

	// Generate uninstall script
	uninstallPath := filepath.Join(dir, "uninstall.sh")
	return b.GenerateFileFromTemplate(ctx, GetTemplate, "uninstall.sh",
		uninstallPath, data, 0755)
}

// generateReadme generates README documentation.
func (b *Bundler) generateReadme(ctx context.Context, recipe *recipe.Recipe,
	dir string, config map[string]string) error {

	scriptData := GenerateScriptData(recipe, config)
	filePath := filepath.Join(dir, "README.md")

	return b.GenerateFileFromTemplate(ctx, GetTemplate, "README.md",
		filePath, scriptData.ToMap(), 0644)
}
