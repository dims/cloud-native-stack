# Bundler Common Package

This package provides shared utilities for all bundler implementations in the Cloud Native Stack project. It consolidates duplicate code from individual bundlers into reusable components, making bundler development and maintenance easier.

## Purpose

The common package centralizes patterns that were previously duplicated across bundler implementations (GPU Operator, Network Operator). This refactoring:

- **Reduces code duplication** - Common patterns are implemented once and reused
- **Improves maintainability** - Bug fixes and enhancements benefit all bundlers
- **Enforces consistency** - All bundlers use the same approach for common tasks
- **Simplifies new bundlers** - New bundlers can leverage existing utilities

## Components

### Configuration Utilities (`config.go`)

Helper functions for extracting and managing configuration values from recipes:

- **`ConfigValue`** - Struct that wraps a value with optional context/source information
- **`GetConfigValue()`** - Retrieve configuration value with fallback to default
- **`GetSubtypeContext()`** - Extract general context from subtype metadata
- **`GetFieldContext()`** - Get field-specific context with fallback to subtype context
- **`ExtractCustomLabels()`** - Extract custom labels from config (keys prefixed with `label_`)
- **`ExtractCustomAnnotations()`** - Extract custom annotations from config (keys prefixed with `annotation_`)

**Example usage:**
```go
// Extract a config value with default
registry := common.GetConfigValue(config, "driver_registry", "nvcr.io/nvidia")

// Extract custom labels
labels := common.ExtractCustomLabels(config)

// Get field context for traceability
subtypeCtx := common.GetSubtypeContext(subtype.Context)
fieldCtx := common.GetFieldContext(subtype.Context, "gpu-operator", subtypeCtx)
```

### Bundle Generation Utilities (`helpers.go`)

Specialized classes for common bundler operations:

#### TemplateRenderer
Renders embedded templates with data:
```go
renderer := common.NewTemplateRenderer(templatesFS)
content, err := renderer.Render("values.yaml.tmpl", data)
```

#### FileWriter
Writes files and tracks operations in results:
```go
writer := common.NewFileWriter(result)
err := writer.WriteFileString(filepath, content, 0644)
err := writer.MakeExecutable(filepath)
```

#### DirectoryManager
Creates directory structures:
```go
dirMgr := common.NewDirectoryManager()
dirs, err := dirMgr.CreateBundleStructure(outputDir, "gpu-operator")
```

#### ContextChecker
Checks for context cancellation:
```go
checker := common.NewContextChecker()
if err := checker.Check(ctx); err != nil {
    return err
}
```

#### ChecksumGenerator
Computes SHA256 checksums for files:
```go
checksumGen := common.NewChecksumGenerator(result)
checksums, err := checksumGen.Generate(outputDir, result.Files)
```

### Result Tracking (`result.go`)

Track bundle generation progress and outcomes:

- **`Result`** - Tracks files, checksums, size, duration, errors for a single bundle
- **`Output`** - Aggregates results from multiple bundlers
- **`BundleType`** - Enum for bundler types (gpu-operator, network-operator)

**Example usage:**
```go
result := common.NewResult(common.BundleTypeGpuOperator)
result.AddFile("values.yaml", 1024)
result.AddChecksum("values.yaml", checksum)
result.MarkSuccess()

// Access results
if result.Success {
    fmt.Printf("Generated %d files (%s)\n", len(result.Files), result.TotalSizeFormatted())
}
```

## Integration with Bundlers

### Before Refactoring

Each bundler had duplicate implementations of:
- Template rendering logic
- File writing with permission management  
- Directory creation
- Checksum computation
- Context checking
- Configuration value extraction

### After Refactoring

Bundlers use common utilities:

```go
// GPU Operator example
func (b *Bundler) Make(ctx context.Context, r *recipe.Recipe, outputDir string) (*common.Result, error) {
    result := common.NewResult(common.BundleTypeGpuOperator)
    
    // Create directory structure
    dirMgr := common.NewDirectoryManager()
    dirs, err := dirMgr.CreateBundleStructure(outputDir, bundlerType.String())
    if err != nil {
        return result, err
    }
    
    // Check context before proceeding
    checker := common.NewContextChecker()
    if err := checker.Check(ctx); err != nil {
        return result, err
    }
    
    // Create utilities
    renderer := common.NewTemplateRenderer(templatesFS)
    writer := common.NewFileWriter(result)
    
    // Generate files using utilities
    // ... bundle-specific logic ...
    
    // Generate checksums
    checksumGen := common.NewChecksumGenerator(result)
    checksums, err := checksumGen.Generate(outputDir, result.Files)
    
    result.MarkSuccess()
    return result, nil
}
```

## Development Guidelines

### Creating New Bundlers

When creating a new bundler:

1. Import the common package: `"github.com/NVIDIA/cloud-native-stack/pkg/bundler/common"`
2. Use `common.ConfigValue` for all configuration fields that need context tracking
3. Use utility classes for file operations, template rendering, etc.
4. Use `common.Result` to track bundle generation progress
5. Follow the pattern established by existing bundlers (GPU Operator, Network Operator)

### Adding New Utilities

When adding utilities to common:

1. Ensure the utility is truly common across multiple bundlers
2. Write comprehensive unit tests (`*_test.go`)
3. Document the utility with godoc comments
4. Update this README with usage examples

### Testing

Run all common package tests:
```bash
go test ./pkg/bundler/common/
```

Run all bundler tests:
```bash
go test ./pkg/bundler/...
```

## Architecture Benefits

### Single Responsibility
Each utility class has a focused responsibility:
- TemplateRenderer: template operations
- FileWriter: file I/O with result tracking
- DirectoryManager: directory structure creation
- ContextChecker: context validation
- ChecksumGenerator: checksum computation

### Dependency Injection
Utilities accept dependencies (Result, embed.FS) making them:
- Testable with mocks
- Reusable across contexts
- Easy to compose

### Error Handling
All utilities:
- Return errors for proper handling
- Provide context in error messages
- Track errors in Result when appropriate

## Performance Considerations

- **Parallel execution** - Bundlers run concurrently via errgroup
- **Efficient I/O** - FileWriter tracks operations without buffering
- **Minimal allocations** - Utilities reuse buffers and maps where possible
- **Context awareness** - All long operations respect context cancellation

## Related Packages

- `pkg/bundler/config` - Bundle configuration management
- `pkg/bundler/gpuoperator` - GPU Operator bundler implementation
- `pkg/bundler/networkoperator` - Network Operator bundler implementation
- `pkg/recipe` - Recipe data structures and generation logic
- `pkg/measurement` - System measurement types

## Maintenance

### Code Quality
- All public functions have godoc comments
- Unit tests achieve >80% coverage
- golangci-lint passes with no warnings
- Follows Go best practices and idioms

### Versioning
The common package is versioned with the Cloud Native Stack project. Breaking changes should be avoided, but when necessary:
1. Deprecate old APIs first
2. Provide migration path
3. Update all bundlers using deprecated APIs
4. Document changes in release notes
