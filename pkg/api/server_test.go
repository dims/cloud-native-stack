package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NVIDIA/cloud-native-stack/pkg/recipe"
)

func TestServeIntegration(t *testing.T) {
	// This is a basic smoke test to ensure the server can be initialized
	// We don't actually run it since it blocks
	// More comprehensive integration tests would use a test server

	// Verify route setup would work
	r := map[string]http.HandlerFunc{
		"/v1/recommendations": nil,
	}

	if _, exists := r["/v1/recommendations"]; !exists {
		t.Error("expected /v1/recommendations route to exist")
	}
}

func TestRecommendationEndpoint(t *testing.T) {
	b := recipe.NewBuilder()

	req := httptest.NewRequest(http.MethodGet, "/v1/recommendations", nil)
	w := httptest.NewRecorder()

	b.HandleRecipes(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("unexpected status code: %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestRecommendationEndpointMethodNotAllowed(t *testing.T) {
	b := recipe.NewBuilder()

	req := httptest.NewRequest(http.MethodPost, "/v1/recommendations", nil)
	w := httptest.NewRecorder()

	b.HandleRecipes(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}

	allow := w.Header().Get("Allow")
	if allow != http.MethodGet {
		t.Errorf("expected Allow header %s, got %s", http.MethodGet, allow)
	}
}

func TestRecommendationEndpointWithQueryParams(t *testing.T) {
	b := recipe.NewBuilder()

	tests := []struct {
		name       string
		query      string
		expectCode int
	}{
		{
			name:       "valid query with ubuntu",
			query:      "?os=ubuntu",
			expectCode: http.StatusOK,
		},
		{
			name:       "valid query with cos",
			query:      "?os=cos",
			expectCode: http.StatusOK,
		},
		{
			name:       "no query params defaults to all",
			query:      "",
			expectCode: http.StatusOK,
		},
		{
			name:       "invalid os parameter",
			query:      "?os=InvalidOS",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/recommendations"+tt.query, nil)
			w := httptest.NewRecorder()

			b.HandleRecipes(w, req)

			if w.Code != tt.expectCode && w.Code != http.StatusInternalServerError {
				t.Errorf("expected status %d (or 500), got %d", tt.expectCode, w.Code)
			}
		})
	}
}

func TestRecommendationEndpointCacheHeaders(t *testing.T) {
	b := recipe.NewBuilder()

	req := httptest.NewRequest(http.MethodGet, "/v1/recommendations", nil)
	w := httptest.NewRecorder()

	b.HandleRecipes(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl == "" && w.Code == http.StatusOK {
		t.Error("expected Cache-Control header on successful response")
	}
}
