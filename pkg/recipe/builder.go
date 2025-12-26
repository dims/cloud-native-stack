package recipe

import (
	"context"
	_ "embed"
	"fmt"
	"sync"
	"time"

	"github.com/NVIDIA/cloud-native-stack/pkg/measurement"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed data/data-v1.yaml
	recipeData []byte

	storeOnce   sync.Once
	cachedStore *Store
	storeErr    error

	defaultBuilder = &Builder{}
)

// Option is a functional option for configuring Builder instances.
type Option func(*Builder)

// WithVersion returns an Option that sets the Builder version string.
// The version is included in recipe metadata for tracking purposes.
func WithVersion(version string) Option {
	return func(b *Builder) {
		b.Version = version
	}
}

// NewBuilder creates a new Builder instance with the provided functional options.
func NewBuilder(opts ...Option) *Builder {
	b := &Builder{}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Builder constructs Recipe payloads based on Query specifications.
// It loads recommendation data, applies overlays based on query matching,
// and generates tailored configuration recipes.
type Builder struct {
	Version string
}

// BuildRecipe creates a Recipe based on the provided query using a shared default Builder.
// This is a convenience function for simple use cases.
// For custom configuration or control over Builder settings, create a Builder instance directly.
func BuildRecipe(ctx context.Context, q *Query) (*Recipe, error) {
	return defaultBuilder.Build(ctx, q)
}

// Build creates a Recipe payload for the provided query.
// It loads the recipe store, applies matching overlays, and returns
// a Recipe with base measurements merged with overlay-specific configurations.
// Context is included in the response only if Query.IncludeContext is true.
func (b *Builder) Build(ctx context.Context, q *Query) (*Recipe, error) {
	if q == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}

	// Track overall build duration
	start := time.Now()
	defer func() {
		recipeBuiltDuration.Observe(time.Since(start).Seconds())
	}()

	store, err := loadStore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load recipe store: %w", err)
	}

	r := &Recipe{
		Request:      q,
		MatchedRules: make([]string, 0),
	}

	merged := cloneMeasurements(store.Base)
	index := indexMeasurementsByType(merged)

	for _, overlay := range store.Overlays {
		// overlays use Query as key, so matching queries inherit overlay-specific measurements
		if overlay.Key.IsMatch(q) {
			merged, index = mergeOverlayMeasurements(merged, index, overlay.Types)
			r.MatchedRules = append(r.MatchedRules, overlay.Key.String())
			recipeRuleMatchTotal.WithLabelValues("matched").Inc()
		}
	}

	r.Measurements = merged

	// Strip context if not requested
	if !q.IncludeContext {
		stripContext(r.Measurements)
	}

	return r, nil
}

// stripContext removes context metadata from all measurements
func stripContext(measurements []*measurement.Measurement) {
	for _, m := range measurements {
		if m == nil {
			continue
		}
		for i := range m.Subtypes {
			m.Subtypes[i].Context = nil
		}
	}
}

func loadStore(_ context.Context) (*Store, error) {
	storeOnce.Do(func() {
		var store Store
		if err := yaml.Unmarshal(recipeData, &store); err != nil {
			storeErr = fmt.Errorf("failed to unmarshal recommendation data: %w", err)
			return
		}
		cachedStore = &store
	})
	return cachedStore, storeErr
}

// cloneMeasurements creates deep copies of all measurements so we never mutate
// the shared store payload while tailoring responses.
func cloneMeasurements(list []*measurement.Measurement) []*measurement.Measurement {
	if len(list) == 0 {
		return nil
	}
	cloned := make([]*measurement.Measurement, 0, len(list))
	for _, m := range list {
		if m == nil {
			continue
		}
		cloned = append(cloned, cloneMeasurement(m))
	}
	return cloned
}

// cloneMeasurement duplicates a single measurement including all of its
// subtypes to protect original data from in-place updates.
func cloneMeasurement(m *measurement.Measurement) *measurement.Measurement {
	if m == nil {
		return nil
	}
	clone := &measurement.Measurement{
		Type:     m.Type,
		Subtypes: make([]measurement.Subtype, len(m.Subtypes)),
	}
	for i := range m.Subtypes {
		clone.Subtypes[i] = cloneSubtype(m.Subtypes[i])
	}
	return clone
}

// cloneSubtype duplicates an individual subtype and its key/value readings.
func cloneSubtype(st measurement.Subtype) measurement.Subtype {
	cloned := measurement.Subtype{
		Name: st.Name,
	}
	if len(st.Data) > 0 {
		cloned.Data = make(map[string]measurement.Reading, len(st.Data))
		for k, v := range st.Data {
			cloned.Data[k] = v
		}
	}
	if len(st.Context) > 0 {
		cloned.Context = make(map[string]string, len(st.Context))
		for k, v := range st.Context {
			cloned.Context[k] = v
		}
	}
	return cloned
}

// indexMeasurementsByType builds an index for O(1) lookup when merging
// overlays by measurement type.
func indexMeasurementsByType(measurements []*measurement.Measurement) map[measurement.Type]*measurement.Measurement {
	index := make(map[measurement.Type]*measurement.Measurement, len(measurements))
	for _, m := range measurements {
		if m == nil {
			continue
		}
		index[m.Type] = m
	}
	return index
}

// mergeOverlayMeasurements folds overlay measurements into the base slice,
// appending new types and delegating to subtype merging when the type already exists.
func mergeOverlayMeasurements(base []*measurement.Measurement, index map[measurement.Type]*measurement.Measurement, overlays []*measurement.Measurement) ([]*measurement.Measurement, map[measurement.Type]*measurement.Measurement) {
	if len(overlays) == 0 {
		return base, index
	}
	for _, overlay := range overlays {
		if overlay == nil {
			continue
		}
		if target, ok := index[overlay.Type]; ok {
			mergeMeasurementSubtypes(target, overlay)
			continue
		}
		cloned := cloneMeasurement(overlay)
		base = append(base, cloned)
		index[cloned.Type] = cloned
	}
	return base, index
}

// mergeMeasurementSubtypes walks all subtypes so overlay data augments or
// overrides existing subtype readings. Now uses the built-in Merge() method
// from the measurement package for consistency.
func mergeMeasurementSubtypes(target, overlay *measurement.Measurement) {
	if target == nil || overlay == nil {
		return
	}

	// Use the built-in Merge method which handles data merging
	if err := target.Merge(overlay); err != nil {
		// This shouldn't happen in practice since we check types when building,
		// but log it just in case
		return
	}

	// Handle context merging (not part of Merge())
	for _, overlaySubtype := range overlay.Subtypes {
		targetSubtype := target.GetSubtype(overlaySubtype.Name)
		if targetSubtype == nil {
			continue
		}
		// Merge context fields
		if targetSubtype.Context == nil && len(overlaySubtype.Context) > 0 {
			targetSubtype.Context = make(map[string]string)
		}
		for key, value := range overlaySubtype.Context {
			targetSubtype.Context[key] = value
		}
	}
}
