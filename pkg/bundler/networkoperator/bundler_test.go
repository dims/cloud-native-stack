package networkoperator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/bundle"
	"github.com/NVIDIA/cloud-native-stack/pkg/bundler/config"
	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe/header"
)

func TestNewBundler(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "with nil config",
			cfg:  nil,
		},
		{
			name: "with valid config",
			cfg:  &config.Config{Namespace: "test-namespace"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBundler(tt.cfg)
			if b == nil {
				t.Fatal("NewBundler() returned nil")
			}
			if b.cfg == nil {
				t.Error("Bundler config should not be nil")
			}
		})
	}
}

func TestBundler_Make(t *testing.T) {
	tests := []struct {
		name    string
		recipe  *recipe.Recipe
		wantErr bool
	}{
		{
			name:    "valid recipe",
			recipe:  createTestRecipe(),
			wantErr: false,
		},
		{
			name: "invalid recipe",
			recipe: &recipe.Recipe{
				Measurements: []*measurement.Measurement{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			b := NewBundler(nil)
			ctx := context.Background()

			result, err := b.Make(ctx, tt.recipe, tmpDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("Make() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("Make() returned nil result")
					return
				}
				if len(result.Files) == 0 {
					t.Error("Make() returned no files")
				}
			}
		})
	}
}

func TestGetTemplate(t *testing.T) {
	tests := []struct {
		name     string
		tmplName string
		wantOK   bool
	}{
		{
			name:     "values template",
			tmplName: "values.yaml",
			wantOK:   true,
		},
		{
			name:     "nicclusterpolicy template",
			tmplName: "nicclusterpolicy",
			wantOK:   true,
		},
		{
			name:     "install script template",
			tmplName: "install.sh",
			wantOK:   true,
		},
		{
			name:     "uninstall script template",
			tmplName: "uninstall.sh",
			wantOK:   true,
		},
		{
			name:     "README template",
			tmplName: "README.md",
			wantOK:   true,
		},
		{
			name:     "unknown template",
			tmplName: "unknown.yaml",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, ok := GetTemplate(tt.tmplName)
			if ok != tt.wantOK {
				t.Errorf("GetTemplate() ok = %v, want %v", ok, tt.wantOK)
				return
			}
			if tt.wantOK && len(tmpl) == 0 {
				t.Error("GetTemplate() returned empty template for valid name")
			}
		})
	}
}

func TestBundler_validateRecipe(t *testing.T) {
	tests := []struct {
		name    string
		recipe  *recipe.Recipe
		wantErr bool
	}{
		{
			name:    "valid recipe",
			recipe:  createTestRecipe(),
			wantErr: false,
		},
		{
			name:    "nil recipe",
			recipe:  nil,
			wantErr: true,
		},
		{
			name: "empty measurements",
			recipe: &recipe.Recipe{
				Measurements: []*measurement.Measurement{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBundler(nil)
			err := b.validateRecipe(tt.recipe)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRecipe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBundler_writeFile(t *testing.T) {
	tmpDir := t.TempDir()
	b := NewBundler(nil)
	result := bundle.NewResult(bundle.BundleTypeNetworkOperator)

	tests := []struct {
		name    string
		path    string
		content string
		perms   os.FileMode
		wantErr bool
	}{
		{
			name:    "write regular file",
			path:    "test.txt",
			content: "test content",
			perms:   0644,
			wantErr: false,
		},
		{
			name:    "write executable file",
			path:    "test.sh",
			content: "#!/bin/bash\necho test",
			perms:   0755,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := filepath.Join(tmpDir, tt.path)
			err := b.writeFile(fullPath, tt.content, tt.perms, result)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file exists and has correct content
				data, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("failed to read written file: %v", err)
					return
				}
				if string(data) != tt.content {
					t.Errorf("file content = %s, want %s", string(data), tt.content)
				}
			}
		})
	}
}

// Helper function to create a test recipe
func createTestRecipe() *recipe.Recipe {
	r := &recipe.Recipe{
		Measurements: []*measurement.Measurement{
			{
				Type: measurement.TypeK8s,
				Subtypes: []measurement.Subtype{
					{
						Name: "config",
						Data: map[string]measurement.Reading{
							"rdma-enabled":             measurement.Bool(true),
							"sr-iov-enabled":           measurement.Bool(true),
							"ofed-version":             measurement.Str("24.07"),
							"network-operator-version": measurement.Str("25.4.0"),
						},
					},
				},
			},
		},
	}
	r.Init(header.KindRecipe, "v1")
	return r
}
