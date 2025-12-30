package networkoperator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/common"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

const (
	configSubtype = "config"
)

// Bundler generates Network Operator deployment bundles.
type Bundler struct {
	// Config for customization
	cfg *config.Config
}

// NewBundler creates a new Network Operator bundler.
func NewBundler(cfg *config.Config) *Bundler {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &Bundler{
		cfg: cfg,
	}
}

// Make generates a Network Operator bundle from a recipe.
func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*common.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	// Validate recipe has required measurements
	if err := b.validateRecipe(r); err != nil {
		return nil, fmt.Errorf("invalid recipe: %w", err)
	}

	// Create result tracker
	result := common.NewResult(common.BundleTypeNetworkOperator)

	// Create output directory structure
	dirManager := common.NewDirectoryManager()
	bundleDir, subdirs, err := dirManager.CreateBundleStructure(outputDir, "network-operator")
	if err != nil {
		return nil, fmt.Errorf("failed to create bundle structure: %w", err)
	}

	// Build configuration map from recipe and bundler config
	configMap := b.buildConfigMap(r)

	// Initialize utilities
	fileWriter := common.NewFileWriter(result)
	contextChecker := common.NewContextChecker()
	templateRenderer := common.NewTemplateRenderer(GetTemplate)

	// Generate all bundle components
	if err := b.generateHelmValues(ctx, r, configMap, bundleDir, contextChecker, templateRenderer, fileWriter); err != nil {
		return nil, fmt.Errorf("failed to generate helm values: %w", err)
	}

	if err := b.generateNicClusterPolicy(ctx, r, configMap, subdirs["manifests"], contextChecker, templateRenderer, fileWriter); err != nil {
		return nil, fmt.Errorf("failed to generate NicClusterPolicy: %w", err)
	}

	if err := b.generateScripts(ctx, r, configMap, subdirs["scripts"], contextChecker, templateRenderer, fileWriter); err != nil {
		return nil, fmt.Errorf("failed to generate scripts: %w", err)
	}

	if err := b.generateReadme(ctx, r, configMap, bundleDir, contextChecker, templateRenderer, fileWriter); err != nil {
		return nil, fmt.Errorf("failed to generate README: %w", err)
	}

	// Generate checksums file last
	if err := b.generateChecksums(ctx, bundleDir, result); err != nil {
		return nil, fmt.Errorf("failed to generate checksums: %w", err)
	}

	// Mark as successful
	result.MarkSuccess()

	return result, nil
}

// validateRecipe checks if recipe has required measurements.
func (b *Bundler) validateRecipe(r *recipe.Recipe) error {
	if r == nil {
		return fmt.Errorf("recipe is nil")
	}

	// Check for required K8s measurements
	hasK8s := false
	for _, m := range r.Measurements {
		if m.Type == measurement.TypeK8s {
			hasK8s = true
			break
		}
	}

	if !hasK8s {
		return fmt.Errorf("recipe missing required Kubernetes measurements")
	}

	return nil
}

// buildConfigMap extracts configuration from recipe and bundler config.
func (b *Bundler) buildConfigMap(r *recipe.Recipe) map[string]string {
	configMap := make(map[string]string)

	// Add bundler config values
	if b.cfg.HelmRepository != "" {
		configMap["helm_repository"] = b.cfg.HelmRepository
	}
	if b.cfg.HelmChartVersion != "" {
		configMap["helm_chart_version"] = b.cfg.HelmChartVersion
	}
	if b.cfg.Namespace != "" {
		configMap["namespace"] = b.cfg.Namespace
	}

	// Add custom labels and annotations using common utilities
	for k, v := range b.cfg.CustomLabels {
		configMap["label_"+k] = v
	}
	for k, v := range b.cfg.CustomAnnotations {
		configMap["annotation_"+k] = v
	}

	// Extract values from recipe measurements
	for _, m := range r.Measurements {
		switch m.Type {
		case measurement.TypeK8s:
			for _, st := range m.Subtypes {
				if st.Name == "image" {
					// Extract Network Operator version
					if val, ok := st.Data["network-operator"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["network_operator_version"] = s
						}
					}
					// Extract OFED driver version
					if val, ok := st.Data["ofed-driver"]; ok {
						if s, ok := val.Any().(string); ok {
							configMap["ofed_version"] = s
						}
					}
				}

				if st.Name == configSubtype {
					// Extract RDMA setting
					if val, ok := st.Data["rdma"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_rdma"] = fmt.Sprintf("%t", b)
						}
					}
					// Extract SR-IOV setting
					if val, ok := st.Data["sr-iov"]; ok {
						if b, ok := val.Any().(bool); ok {
							configMap["enable_sriov"] = fmt.Sprintf("%t", b)
						}
					}
				}
			}
		case measurement.TypeSystemD, measurement.TypeOS, measurement.TypeGPU:
			// Not used for Network Operator configuration
			continue
		}
	}

	return configMap
}

// generateHelmValues creates the Helm values.yaml file.
func (b *Bundler) generateHelmValues(ctx context.Context, r *recipe.Recipe, configMap map[string]string,
	outputDir string, contextChecker *common.ContextChecker, templateRenderer *common.TemplateRenderer,
	fileWriter *common.FileWriter) error {

	if err := contextChecker.Check(ctx); err != nil {
		return err
	}

	values := GenerateHelmValues(r, configMap)
	if err := values.Validate(); err != nil {
		return fmt.Errorf("invalid helm values: %w", err)
	}

	content, err := templateRenderer.Render("values.yaml", values.ToMap())
	if err != nil {
		return fmt.Errorf("failed to render values template: %w", err)
	}

	path := filepath.Join(outputDir, "values.yaml")
	return fileWriter.WriteFileString(path, content, 0644)
}

// generateNicClusterPolicy creates the NicClusterPolicy manifest.
func (b *Bundler) generateNicClusterPolicy(ctx context.Context, r *recipe.Recipe, configMap map[string]string,
	manifestsDir string, contextChecker *common.ContextChecker, templateRenderer *common.TemplateRenderer,
	fileWriter *common.FileWriter) error {

	if err := contextChecker.Check(ctx); err != nil {
		return err
	}

	data := GenerateManifestData(r, configMap)

	content, err := templateRenderer.Render("nicclusterpolicy", data.ToMap())
	if err != nil {
		return fmt.Errorf("failed to render NicClusterPolicy template: %w", err)
	}

	path := filepath.Join(manifestsDir, "nicclusterpolicy.yaml")
	return fileWriter.WriteFileString(path, content, 0644)
}

// generateScripts creates installation and uninstallation scripts.
func (b *Bundler) generateScripts(ctx context.Context, r *recipe.Recipe, configMap map[string]string,
	scriptsDir string, contextChecker *common.ContextChecker, templateRenderer *common.TemplateRenderer,
	fileWriter *common.FileWriter) error {

	if err := contextChecker.Check(ctx); err != nil {
		return err
	}

	scriptData := GenerateScriptData(r, configMap)
	data := scriptData.ToMap()

	// Generate install script
	installContent, err := templateRenderer.Render("install.sh", data)
	if err != nil {
		return fmt.Errorf("failed to render install script template: %w", err)
	}

	installPath := filepath.Join(scriptsDir, "install.sh")
	if err = fileWriter.WriteFileString(installPath, installContent, 0755); err != nil {
		return err
	}

	// Generate uninstall script
	uninstallContent, err := templateRenderer.Render("uninstall.sh", data)
	if err != nil {
		return fmt.Errorf("failed to render uninstall script template: %w", err)
	}

	uninstallPath := filepath.Join(scriptsDir, "uninstall.sh")
	return fileWriter.WriteFileString(uninstallPath, uninstallContent, 0755)
}

// generateReadme creates the README.md file.
func (b *Bundler) generateReadme(ctx context.Context, r *recipe.Recipe, configMap map[string]string,
	outputDir string, contextChecker *common.ContextChecker, templateRenderer *common.TemplateRenderer,
	fileWriter *common.FileWriter) error {

	if err := contextChecker.Check(ctx); err != nil {
		return err
	}

	scriptData := GenerateScriptData(r, configMap)
	helmValues := GenerateHelmValues(r, configMap)

	data := map[string]interface{}{
		"Script": scriptData.ToMap(),
		"Helm":   helmValues.ToMap(),
	}

	content, err := templateRenderer.Render("README.md", data)
	if err != nil {
		return fmt.Errorf("failed to render README template: %w", err)
	}

	path := filepath.Join(outputDir, "README.md")
	return fileWriter.WriteFileString(path, content, 0644)
}

// generateChecksums creates a checksums.txt file with SHA256 hashes.
func (b *Bundler) generateChecksums(ctx context.Context, outputDir string, result *common.Result) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	generator := common.NewChecksumGenerator(result)
	fileWriter := common.NewFileWriter(result)

	content, err := generator.Generate(outputDir, "Network Operator")
	if err != nil {
		return fmt.Errorf("failed to generate checksums: %w", err)
	}

	path := filepath.Join(outputDir, "checksums.txt")
	return fileWriter.WriteFileString(path, content, 0644)
}
