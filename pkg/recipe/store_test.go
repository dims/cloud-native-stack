package recipe

import (
	"context"
	"errors"
	"sync"
	"testing"

	cnserrors "github.com/NVIDIA/cloud-native-stack/pkg/errors"
)

func TestLoadStore_CachesErrorUntilReset(t *testing.T) {
	originalData := recipeData
	t.Cleanup(func() {
		recipeData = originalData
		storeOnce = sync.Once{}
		cachedStore = nil
		cachedErr = nil
	})

	// 1) First load with invalid YAML should cache the error.
	recipeData = []byte(": this is not valid yaml")
	storeOnce = sync.Once{}
	cachedStore = nil
	cachedErr = nil

	_, err := loadStore(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// 2) Even if data becomes valid, without resetting the cache it should still return the cached error.
	recipeData = []byte("base: []\noverlays: []\n")
	_, err2 := loadStore(context.Background())
	if err2 == nil {
		t.Fatal("expected cached error, got nil")
	}

	// 3) After resetting the cache, it should succeed.
	storeOnce = sync.Once{}
	cachedStore = nil
	cachedErr = nil

	store, err3 := loadStore(context.Background())
	if err3 != nil {
		t.Fatalf("expected success after reset, got error: %v", err3)
	}
	if store == nil {
		t.Fatal("expected store, got nil")
	}
}

func TestLoadStore_NotInitializedReturnsInternalStructuredError(t *testing.T) {
	originalData := recipeData
	t.Cleanup(func() {
		recipeData = originalData
		storeOnce = sync.Once{}
		cachedStore = nil
		cachedErr = nil
	})

	recipeData = []byte("base: []\noverlays: []\n")
	storeOnce = sync.Once{}
	cachedStore = nil
	cachedErr = nil

	// Mark the Once as already done without initializing cachedStore/cachedErr.
	storeOnce.Do(func() {})

	_, err := loadStore(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var se *cnserrors.StructuredError
	if !errors.As(err, &se) {
		t.Fatalf("expected StructuredError, got %T: %v", err, err)
	}
	if se.Code != cnserrors.ErrCodeInternal {
		t.Fatalf("expected code %s, got %s", cnserrors.ErrCodeInternal, se.Code)
	}
}

func TestLoadStore_ConcurrentCallsReturnSamePointer(t *testing.T) {
	originalData := recipeData
	t.Cleanup(func() {
		recipeData = originalData
		storeOnce = sync.Once{}
		cachedStore = nil
		cachedErr = nil
	})

	recipeData = []byte("base: []\noverlays: []\n")
	storeOnce = sync.Once{}
	cachedStore = nil
	cachedErr = nil

	const n = 50
	stores := make([]*Store, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			stores[i], errs[i] = loadStore(context.Background())
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("unexpected error from goroutine %d: %v", i, errs[i])
		}
		if stores[i] == nil {
			t.Fatalf("unexpected nil store from goroutine %d", i)
		}
	}

	first := stores[0]
	for i := 1; i < n; i++ {
		if stores[i] != first {
			t.Fatalf("expected all goroutines to receive same store pointer")
		}
	}
}
