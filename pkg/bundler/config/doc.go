// Package config provides configuration options for bundler implementations.
//
// This package defines the configuration structure and functional options pattern
// for customizing bundler behavior. All bundlers receive a Config instance that
// controls their output generation.
//
// # Configuration Options
//
// Config controls bundler behavior through various settings:
//   - Namespace: Kubernetes namespace for deployment (default: "gpu-operator")
//   - IncludeScripts: Generate installation/uninstallation scripts
//   - IncludeReadme: Generate deployment documentation
//   - IncludeChecksums: Generate SHA256 checksums.txt file
//   - IncludeManifests: Generate Kubernetes manifest files
//   - CustomValues: Additional key-value pairs for template rendering
//
// # Usage
//
// Create with defaults:
//
//	cfg := config.NewConfig()
//
// Customize with functional options:
//
//	cfg := config.NewConfig(
//	    config.WithNamespace("nvidia-system"),
//	    config.WithIncludeScripts(true),
//	    config.WithIncludeChecksums(true),
//	    config.WithCustomValue("cluster", "production"),
//	)
//
// Access configuration:
//
//	namespace := cfg.Namespace()
//	if cfg.IncludeScripts() {
//	    // Generate scripts
//	}
//	clusterName, ok := cfg.CustomValue("cluster")
//
// # Default Values
//
// The default configuration includes:
//   - Namespace: "gpu-operator"
//   - IncludeScripts: true
//   - IncludeReadme: true
//   - IncludeChecksums: true
//   - IncludeManifests: true
//   - CustomValues: empty map
//
// # Thread Safety
//
// Config is immutable after creation, making it safe for concurrent use by
// multiple bundlers executing in parallel.
//
// # Integration with Bundlers
//
// Bundlers receive Config through their constructor:
//
//	type MyBundler struct {
//	    cfg *config.Config
//	}
//
//	func NewMyBundler(cfg *config.Config) *MyBundler {
//	    return &MyBundler{cfg: cfg}
//	}
//
//	func (b *MyBundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*result.Result, error) {
//	    namespace := b.cfg.Namespace()
//	    if b.cfg.IncludeScripts() {
//	        // Generate scripts
//	    }
//	    // ...
//	}
//
// Or use BaseBundler which embeds Config:
//
//	type MyBundler struct {
//	    *internal.BaseBundler
//	}
//
//	func (b *MyBundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*result.Result, error) {
//	    namespace := b.Config.Namespace()
//	    if b.Config.IncludeScripts() {
//	        // Generate scripts
//	    }
//	    // ...
//	}
package config
