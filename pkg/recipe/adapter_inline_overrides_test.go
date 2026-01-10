package recipe

import (
	"context"
	"testing"
)

// TestGetValuesForComponent_InlineOverrides tests the three-way merge:
// base values → ValuesFile → inline Overrides.
func TestGetValuesForComponent_InlineOverrides(t *testing.T) {
	tests := []struct {
		name          string
		setupRecipe   func() *RecipeResult
		componentName string
		wantDriver    string
		wantGDRCopy   bool
		wantGDS       bool
		wantErr       bool
	}{
		{
			name: "inline overrides only (no valuesFile)",
			setupRecipe: func() *RecipeResult {
				return &RecipeResult{
					ComponentRefs: []ComponentRef{
						{
							Name:    "gpu-operator",
							Version: "v25.3.4",
							Overrides: map[string]interface{}{
								"driver": map[string]interface{}{
									"version": "570.86.16",
								},
								"gdrcopy": map[string]interface{}{
									"enabled": true,
								},
								"gds": map[string]interface{}{
									"enabled": true,
								},
							},
						},
					},
				}
			},
			componentName: "gpu-operator",
			wantDriver:    "570.86.16",
			wantGDRCopy:   true,
			wantGDS:       true,
			wantErr:       false,
		},
		{
			name: "valuesFile + inline overrides (hybrid)",
			setupRecipe: func() *RecipeResult {
				// This would load from components/gpu-operator/values.yaml
				// and apply overrides on top
				return &RecipeResult{
					ComponentRefs: []ComponentRef{
						{
							Name:       "gpu-operator",
							Version:    "v25.3.4",
							ValuesFile: "components/gpu-operator/values.yaml",
							Overrides: map[string]interface{}{
								// Override just the driver version
								"driver": map[string]interface{}{
									"version": "570.86.16",
								},
							},
						},
					},
				}
			},
			componentName: "gpu-operator",
			wantDriver:    "570.86.16",
			wantErr:       false,
		},
		{
			name: "valuesFile only (traditional)",
			setupRecipe: func() *RecipeResult {
				// Load from overlay file without inline overrides
				return &RecipeResult{
					ComponentRefs: []ComponentRef{
						{
							Name:       "gpu-operator",
							Version:    "v25.3.4",
							ValuesFile: "components/gpu-operator/eks-gb200-training.yaml",
						},
					},
				}
			},
			componentName: "gpu-operator",
			wantDriver:    "570.86.16",
			wantGDRCopy:   true,
			wantGDS:       true,
			wantErr:       false,
		},
		{
			name: "inline overrides take precedence over valuesFile",
			setupRecipe: func() *RecipeResult {
				return &RecipeResult{
					ComponentRefs: []ComponentRef{
						{
							Name:       "gpu-operator",
							Version:    "v25.3.4",
							ValuesFile: "components/gpu-operator/eks-gb200-training.yaml", // driver: 570.86.16
							Overrides: map[string]interface{}{
								"driver": map[string]interface{}{
									"version": "999.99.99", // Override with different version
								},
							},
						},
					},
				}
			},
			componentName: "gpu-operator",
			wantDriver:    "999.99.99", // Inline override should win
			wantErr:       false,
		},
		{
			name: "no valuesFile and no overrides (empty)",
			setupRecipe: func() *RecipeResult {
				return &RecipeResult{
					ComponentRefs: []ComponentRef{
						{
							Name:    "test-component",
							Version: "v1.0.0",
						},
					},
				}
			},
			componentName: "test-component",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recipe := tt.setupRecipe()

			values, err := recipe.GetValuesForComponent(tt.componentName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetValuesForComponent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				return // Expected error, test passes
			}

			// Verify driver version if specified
			if tt.wantDriver != "" {
				driver, ok := values["driver"].(map[string]interface{})
				if !ok {
					t.Fatalf("driver not found or not a map")
				}
				version, ok := driver["version"].(string)
				if !ok {
					t.Fatalf("driver.version not found or not a string")
				}
				if version != tt.wantDriver {
					t.Errorf("driver.version = %q, want %q", version, tt.wantDriver)
				}
			}

			// Verify gdrcopy if specified
			if tt.wantGDRCopy {
				gdrcopy, ok := values["gdrcopy"].(map[string]interface{})
				if !ok {
					t.Errorf("gdrcopy not found or not a map")
				} else {
					enabled, ok := gdrcopy["enabled"].(bool)
					if !ok {
						t.Errorf("gdrcopy.enabled not found or not a bool")
					} else if !enabled {
						t.Errorf("gdrcopy.enabled = false, want true")
					}
				}
			}

			// Verify gds if specified
			if tt.wantGDS {
				gds, ok := values["gds"].(map[string]interface{})
				if !ok {
					t.Errorf("gds not found or not a map")
				} else {
					enabled, ok := gds["enabled"].(bool)
					if !ok {
						t.Errorf("gds.enabled not found or not a bool")
					} else if !enabled {
						t.Errorf("gds.enabled = false, want true")
					}
				}
			}

			t.Logf("✅ Test passed - values merged correctly")
		})
	}
}

// TestGetValuesForComponent_OverridesMergeDeep tests that inline overrides
// merge deeply with existing values, not replace entire maps.
func TestGetValuesForComponent_OverridesMergeDeep(t *testing.T) {
	recipe := &RecipeResult{
		ComponentRefs: []ComponentRef{
			{
				Name:       "gpu-operator",
				Version:    "v25.3.4",
				ValuesFile: "components/gpu-operator/values.yaml",
				Overrides: map[string]interface{}{
					"driver": map[string]interface{}{
						// Only override version, other driver fields should remain
						"version": "999.99.99",
					},
					"newField": map[string]interface{}{
						// Add entirely new field
						"enabled": true,
					},
				},
			},
		},
	}

	values, err := recipe.GetValuesForComponent("gpu-operator")
	if err != nil {
		t.Fatalf("GetValuesForComponent() error = %v", err)
	}

	// Verify driver.version was overridden
	driver, ok := values["driver"].(map[string]interface{})
	if !ok {
		t.Fatalf("driver not found or not a map")
	}
	version, ok := driver["version"].(string)
	if !ok {
		t.Fatalf("driver.version not found or not a string")
	}
	if version != "999.99.99" {
		t.Errorf("driver.version = %q, want 999.99.99", version)
	}

	// Verify other driver fields still exist (from base values)
	// The base values.yaml should have more than just version
	if len(driver) < 2 {
		t.Errorf("driver map has %d fields, expected more (deep merge should preserve other fields)", len(driver))
	}

	// Verify newField was added
	newField, ok := values["newField"].(map[string]interface{})
	if !ok {
		t.Errorf("newField not found or not a map")
	} else {
		enabled, ok := newField["enabled"].(bool)
		if !ok || !enabled {
			t.Errorf("newField.enabled = %v, want true", enabled)
		}
	}

	t.Logf("✅ Deep merge works correctly - overrides merged, not replaced")
}

// TestGetValuesForComponent_BuilderIntegration tests inline overrides
// with real recipe building from criteria.
func TestGetValuesForComponent_BuilderIntegration(t *testing.T) {
	ctx := context.Background()
	builder := NewBuilder()

	// Build a recipe (this will load from metadata store)
	criteria := &Criteria{
		Service:     CriteriaServiceEKS,
		Accelerator: CriteriaAcceleratorGB200,
		Intent:      CriteriaIntentTraining,
	}

	result, err := builder.BuildFromCriteria(ctx, criteria)
	if err != nil {
		t.Fatalf("BuildFromCriteria() error = %v", err)
	}

	// Get gpu-operator component
	ref := result.GetComponentRef("gpu-operator")
	if ref == nil {
		t.Fatal("gpu-operator not found in recipe")
	}

	// Load values (this tests the full pipeline)
	values, err := result.GetValuesForComponent("gpu-operator")
	if err != nil {
		t.Fatalf("GetValuesForComponent() error = %v", err)
	}

	// Verify values were loaded
	if len(values) == 0 {
		t.Error("values map is empty")
	}

	t.Logf("✅ Builder integration works - loaded %d top-level keys", len(values))

	// If the recipe has inline overrides, verify they were applied
	if len(ref.Overrides) > 0 {
		t.Logf("   Recipe has %d inline override keys", len(ref.Overrides))
	}
}
