// Package recipe provides recipe building and matching functionality.
package recipe

import (
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

//go:embed data/*.yaml data/components/*/*.yaml
var dataFS embed.FS

// RecipeInput is an interface that both Recipe and RecipeResult implement.
// This allows bundlers to work with either format during the transition period.
type RecipeInput interface {
	// GetMeasurements returns the measurements for bundler configuration.
	GetMeasurements() []*measurement.Measurement

	// GetComponentRef returns the component reference for a given component name.
	// Returns nil if the component is not found.
	GetComponentRef(name string) *ComponentRef

	// GetValuesForComponent returns the values map for a given component.
	// For Recipe, this extracts values from measurements.
	// For RecipeResult, this loads values from the component's valuesFile.
	GetValuesForComponent(name string) (map[string]interface{}, error)
}

// Ensure Recipe implements RecipeInput
var _ RecipeInput = (*Recipe)(nil)

// GetMeasurements returns the measurements from a Recipe.
func (r *Recipe) GetMeasurements() []*measurement.Measurement {
	return r.Measurements
}

// GetComponentRef returns nil for Recipe (v1 format doesn't have components).
func (r *Recipe) GetComponentRef(name string) *ComponentRef {
	return nil
}

// GetValuesForComponent extracts values from measurements for Recipe.
// This maintains backward compatibility with the legacy measurements-based format.
func (r *Recipe) GetValuesForComponent(name string) (map[string]interface{}, error) {
	// For legacy Recipe, values are embedded in measurements
	// This is a no-op - bundlers extract their own values from measurements
	return make(map[string]interface{}), nil
}

// Ensure RecipeResult implements RecipeInput
var _ RecipeInput = (*RecipeResult)(nil)

// GetMeasurements returns an empty slice for RecipeResult.
// The new format uses ComponentRefs with valuesFiles instead of measurements.
func (r *RecipeResult) GetMeasurements() []*measurement.Measurement {
	return nil
}

// GetComponentRef returns the component reference for a given component name.
func (r *RecipeResult) GetComponentRef(name string) *ComponentRef {
	for i := range r.ComponentRefs {
		if r.ComponentRefs[i].Name == name {
			return &r.ComponentRefs[i]
		}
	}
	return nil
}

// GetValuesForComponent loads values from the component's valuesFile.
func (r *RecipeResult) GetValuesForComponent(name string) (map[string]interface{}, error) {
	ref := r.GetComponentRef(name)
	if ref == nil {
		return nil, fmt.Errorf("component %q not found in recipe", name)
	}

	if ref.ValuesFile == "" {
		// No values file, return empty map
		return make(map[string]interface{}), nil
	}

	// Load values from embedded filesystem
	valuesPath := fmt.Sprintf("data/%s", ref.ValuesFile)
	data, err := dataFS.ReadFile(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file %q: %w", ref.ValuesFile, err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values file %q: %w", ref.ValuesFile, err)
	}

	return values, nil
}

// HasComponentRefs checks if the input is a RecipeResult with component references.
func HasComponentRefs(input RecipeInput) bool {
	_, ok := input.(*RecipeResult)
	return ok
}

// ToRecipeResult converts a RecipeInput to RecipeResult if possible.
// Returns nil if the input is a legacy measurements-based Recipe.
func ToRecipeResult(input RecipeInput) *RecipeResult {
	if result, ok := input.(*RecipeResult); ok {
		return result
	}
	return nil
}

// ToRecipe converts a RecipeInput to Recipe if possible.
// Returns nil if the input is a RecipeResult with component references.
func ToRecipe(input RecipeInput) *Recipe {
	if recipe, ok := input.(*Recipe); ok {
		return recipe
	}
	return nil
}
