package recipe

import (
	"context"
	_ "embed"
	"sync"

	cnserrors "github.com/NVIDIA/cloud-native-stack/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed data/data-v1.yaml
	recipeData []byte

	storeOnce   sync.Once
	cachedStore *Store
	cachedErr   error
)

// loadStore loads and caches the recipe store from embedded data.
// Because the data is embedded at build time, it is safe (and simpler) to parse it once
// and reuse the in-memory representation for the lifetime of the process.
func loadStore(_ context.Context) (*Store, error) {
	storeOnce.Do(func() {
		var store Store
		if err := yaml.Unmarshal(recipeData, &store); err != nil {
			cachedErr = err
			return
		}
		cachedStore = &store
	})

	if cachedErr != nil {
		return nil, cachedErr
	}
	if cachedStore == nil {
		return nil, cnserrors.New(cnserrors.ErrCodeInternal, "recipe store not initialized")
	}
	return cachedStore, nil
}
