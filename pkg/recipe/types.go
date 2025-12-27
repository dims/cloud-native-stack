package recipe

import (
	"context"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"github.com/NVIDIA/cloud-native-stack/pkg/recipe/header"
	"github.com/NVIDIA/cloud-native-stack/pkg/snapshotter"
)

const (
	// RecipeAPIVersion is the current API version for recipes
	RecipeAPIVersion = "v1"
)

// Recipe represents the recipe response structure.
type Recipe struct {
	header.Header `json:",inline" yaml:",inline"`

	Request      *Query                     `json:"request,omitempty" yaml:"request,omitempty"`
	MatchedRules []string                   `json:"matchedRules,omitempty" yaml:"matchedRules,omitempty"`
	Measurements []*measurement.Measurement `json:"measurements" yaml:"measurements"`
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

// Recommender defines the interface for generating recommendations based on snapshots and intent.
type Recommender interface {
	Recommend(ctx context.Context, intent IntentType, snap *snapshotter.Snapshot) (*Recipe, error)
}
