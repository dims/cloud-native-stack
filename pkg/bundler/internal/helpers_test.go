package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTemplateRenderer_Render(t *testing.T) {
	templates := map[string]string{
		"test": "Hello {{.Name}}!",
	}

	getter := func(name string) (string, bool) {
		tmpl, ok := templates[name]
		return tmpl, ok
	}

	renderer := NewTemplateRenderer(getter)

	tests := []struct {
		name     string
		tmplName string
		data     map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "renders template",
			tmplName: "test",
			data:     map[string]interface{}{"Name": "World"},
			want:     "Hello World!",
			wantErr:  false,
		},
		{
			name:     "template not found",
			tmplName: "missing",
			data:     map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderer.Render(tt.tmplName, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirectoryManager_CreateDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewDirectoryManager()

	dirs := []string{
		filepath.Join(tmpDir, "dir1"),
		filepath.Join(tmpDir, "dir2", "subdir"),
	}

	err := manager.CreateDirectories(dirs, 0755)
	if err != nil {
		t.Fatalf("CreateDirectories() error = %v", err)
	}

	// Verify directories were created
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
}

func TestDirectoryManager_CreateBundleStructure(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewDirectoryManager()

	bundleDir, subdirs, err := manager.CreateBundleStructure(tmpDir, "test-bundle")
	if err != nil {
		t.Fatalf("CreateBundleStructure() error = %v", err)
	}

	// Verify bundle directory
	expectedBundleDir := filepath.Join(tmpDir, "test-bundle")
	if bundleDir != expectedBundleDir {
		t.Errorf("bundleDir = %v, want %v", bundleDir, expectedBundleDir)
	}

	// Verify subdirectories
	expectedSubdirs := map[string]string{
		"root":      expectedBundleDir,
		"scripts":   filepath.Join(expectedBundleDir, "scripts"),
		"manifests": filepath.Join(expectedBundleDir, "manifests"),
	}

	for key, expectedPath := range expectedSubdirs {
		if subdirs[key] != expectedPath {
			t.Errorf("subdirs[%s] = %v, want %v", key, subdirs[key], expectedPath)
		}

		// Verify directory exists
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", expectedPath)
		}
	}
}

func TestContextChecker_Check(t *testing.T) {
	checker := NewContextChecker()

	t.Run("active context", func(t *testing.T) {
		ctx := context.Background()
		err := checker.Check(ctx)
		if err != nil {
			t.Errorf("Check() with active context should not error, got %v", err)
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := checker.Check(ctx)
		if err == nil {
			t.Error("Check() with cancelled context should error")
		}
	})

	t.Run("expired context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		err := checker.Check(ctx)
		if err == nil {
			t.Error("Check() with expired context should error")
		}
	})
}

func TestComputeChecksum(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{
			name:    "empty content",
			content: []byte{},
			want:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "hello world",
			content: []byte("hello world"),
			want:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeChecksum(tt.content)
			if got != tt.want {
				t.Errorf("ComputeChecksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
