package recipe

import (
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
)

const (
	// RecipeAPIVersion is the current API version for recipes
	RecipeAPIVersion = "v1"
)

// Recipe represents the recipe response structure.
type Recipe struct {
	Request        Query                      `json:"request" yaml:"request"`
	MatchedRules   []string                   `json:"matchedRuleId" yaml:"matchedRuleId"`
	PayloadVersion string                     `json:"payloadVersion" yaml:"payloadVersion"`
	GeneratedAt    time.Time                  `json:"generatedAt" yaml:"generatedAt"`
	Measurements   []*measurement.Measurement `json:"measurements" yaml:"measurements"`
}

// Store holds base measurements for recipes.
type Store struct {
	Base     []*measurement.Measurement `json:"base" yaml:"base"`
	Overlays []*Overlay                 `json:"overlays" yaml:"overlays"`
}

// Overlay represents overlay measurements for specific scenarios.
type Overlay struct {
	Key   Query                      `json:"key" yaml:"key"`
	Types []*measurement.Measurement `json:"types" yaml:"types"`
}
