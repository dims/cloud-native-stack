package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/types"
)

func TestNewBaseBundler(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		bundlerType types.BundleType
		wantNilCfg  bool
	}{
		{
			name:        "with config",
			config:      config.NewConfig(),
			bundlerType: types.BundleTypeGpuOperator,
			wantNilCfg:  false,
		},
		{
			name:        "nil config creates default",
			config:      nil,
			bundlerType: types.BundleTypeGpuOperator,
			wantNilCfg:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBaseBundler(tt.config, tt.bundlerType)

			if b == nil {
				t.Fatal("NewBaseBundler() returned nil")
			}

			if b.Config == nil {
				t.Error("NewBaseBundler() Config is nil")
			}

			if b.Result == nil {
				t.Error("NewBaseBundler() Result is nil")
			}

			if b.Result.Type != tt.bundlerType {
				t.Errorf("Result.Type = %v, want %v", b.Result.Type, tt.bundlerType)
			}
		})
	}
}

func TestBaseBundler_CreateBundleDir(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	dirs, err := b.CreateBundleDir(tmpDir, "test-bundle")
	if err != nil {
		t.Fatalf("CreateBundleDir() error = %v", err)
	}

	// Verify directory structure
	expectedDirs := []string{dirs.Root, dirs.Scripts, dirs.Manifests, dirs.Configs}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}

	// Verify directory paths are correct
	if dirs.Root != filepath.Join(tmpDir, "test-bundle") {
		t.Errorf("Root dir = %s, want %s", dirs.Root, filepath.Join(tmpDir, "test-bundle"))
	}

	if dirs.Scripts != filepath.Join(dirs.Root, "scripts") {
		t.Errorf("Scripts dir = %s, want %s", dirs.Scripts, filepath.Join(dirs.Root, "scripts"))
	}

	if dirs.Manifests != filepath.Join(dirs.Root, "manifests") {
		t.Errorf("Manifests dir = %s, want %s", dirs.Manifests, filepath.Join(dirs.Root, "manifests"))
	}

	if dirs.Configs != filepath.Join(dirs.Root, "configs") {
		t.Errorf("Configs dir = %s, want %s", dirs.Configs, filepath.Join(dirs.Root, "configs"))
	}
}

func TestBaseBundler_WriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")

	err := b.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File was not created")
	}

	// Verify content
	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("File content = %s, want %s", readContent, content)
	}

	// Verify result was updated
	if len(b.Result.Files) != 1 {
		t.Errorf("Result.Files length = %d, want 1", len(b.Result.Files))
	}

	if b.Result.Size != int64(len(content)) {
		t.Errorf("Result.Size = %d, want %d", b.Result.Size, len(content))
	}
}

func TestBaseBundler_WriteFileString(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test string content"

	err := b.WriteFileString(testFile, content, 0644)
	if err != nil {
		t.Fatalf("WriteFileString() error = %v", err)
	}

	// Verify file was created with correct content
	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("File content = %s, want %s", readContent, content)
	}
}

func TestBaseBundler_RenderTemplate(t *testing.T) {
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	tests := []struct {
		name    string
		tmpl    string
		data    interface{}
		want    string
		wantErr bool
	}{
		{
			name: "simple template",
			tmpl: "Hello, {{.Name}}!",
			data: map[string]string{"Name": "World"},
			want: "Hello, World!",
		},
		{
			name: "template with iteration",
			tmpl: "{{range .Items}}{{.}} {{end}}",
			data: map[string][]string{"Items": {"a", "b", "c"}},
			want: "a b c ",
		},
		{
			name:    "invalid template",
			tmpl:    "{{.Invalid",
			data:    map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.RenderTemplate(tt.tmpl, tt.name, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("RenderTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBaseBundler_RenderAndWriteTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	tmpl := "Hello, {{.Name}}!"
	data := map[string]string{"Name": "World"}
	outputPath := filepath.Join(tmpDir, "output.txt")

	err := b.RenderAndWriteTemplate(tmpl, "test", outputPath, data, 0644)
	if err != nil {
		t.Fatalf("RenderAndWriteTemplate() error = %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	want := "Hello, World!"
	if string(content) != want {
		t.Errorf("File content = %s, want %s", content, want)
	}
}

func TestBaseBundler_GenerateChecksums(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)
	ctx := context.Background()

	// Create bundle directory
	bundleDir := filepath.Join(tmpDir, "test-bundle")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatalf("Failed to create bundle dir: %v", err)
	}

	// Write some test files
	testFiles := []struct {
		name    string
		content string
	}{
		{"file1.txt", "content1"},
		{"file2.txt", "content2"},
	}

	for _, tf := range testFiles {
		path := filepath.Join(bundleDir, tf.name)
		if err := b.WriteFileString(path, tf.content, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Generate checksums
	err := b.GenerateChecksums(ctx, bundleDir)
	if err != nil {
		t.Fatalf("GenerateChecksums() error = %v", err)
	}

	// Verify checksums file exists
	checksumPath := filepath.Join(bundleDir, "checksums.txt")
	if _, err := os.Stat(checksumPath); os.IsNotExist(err) {
		t.Error("Checksums file was not created")
	}

	// Verify checksums content
	content, err := os.ReadFile(checksumPath)
	if err != nil {
		t.Fatalf("Failed to read checksums: %v", err)
	}

	contentStr := string(content)
	for _, tf := range testFiles {
		if !filepath.IsAbs(tf.name) && !contains(contentStr, tf.name) {
			t.Errorf("Checksums file does not contain %s", tf.name)
		}
	}
}

func TestBaseBundler_GenerateChecksums_ContextCancelled(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := b.GenerateChecksums(ctx, tmpDir)
	if err == nil {
		t.Error("GenerateChecksums() should return error for cancelled context")
	}
}

func TestBaseBundler_MakeExecutable(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	testFile := filepath.Join(tmpDir, "script.sh")
	if err := os.WriteFile(testFile, []byte("#!/bin/bash\necho test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := b.MakeExecutable(testFile)
	if err != nil {
		t.Fatalf("MakeExecutable() error = %v", err)
	}

	// Verify file is executable
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	if mode&0111 == 0 {
		t.Error("File is not executable")
	}
}

func TestBaseBundler_Finalize(t *testing.T) {
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	// Add some files to result
	b.Result.AddFile("/tmp/file1.txt", 100)
	b.Result.AddFile("/tmp/file2.txt", 200)

	start := time.Now()
	time.Sleep(10 * time.Millisecond) // Ensure some duration
	b.Finalize(start)

	if !b.Result.Success {
		t.Error("Result.Success should be true after Finalize()")
	}

	if b.Result.Duration == 0 {
		t.Error("Result.Duration should be set after Finalize()")
	}

	if b.Result.Duration < 10*time.Millisecond {
		t.Error("Result.Duration should be at least 10ms")
	}
}

func TestBaseBundler_CheckContext(t *testing.T) {
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "active context",
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := b.CheckContext(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBaseBundler_AddError(t *testing.T) {
	b := NewBaseBundler(nil, types.BundleTypeGpuOperator)

	// Add nil error - should not panic
	b.AddError(nil)
	if len(b.Result.Errors) != 0 {
		t.Error("AddError(nil) should not add to errors")
	}

	// Add real error
	testErr := os.ErrNotExist
	b.AddError(testErr)

	if len(b.Result.Errors) != 1 {
		t.Errorf("Result.Errors length = %d, want 1", len(b.Result.Errors))
	}

	if b.Result.Errors[0] != testErr.Error() {
		t.Errorf("Result.Errors[0] = %s, want %s", b.Result.Errors[0], testErr.Error())
	}
}
