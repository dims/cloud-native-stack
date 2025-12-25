package recipe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleRecipes_MethodNotAllowed(t *testing.T) {
	builder := NewBuilder()
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/recipes", nil)
			w := httptest.NewRecorder()

			builder.HandleRecipes(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}

			if allow := w.Header().Get("Allow"); allow != http.MethodGet {
				t.Errorf("expected Allow header %s, got %s", http.MethodGet, allow)
			}
		})
	}
}

func TestHandleRecipes_InvalidQuery(t *testing.T) {
	builder := NewBuilder()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "invalid os version",
			query: "?osv=bad-version",
		},
		{
			name:  "invalid kernel version",
			query: "?kernel=invalid",
		},
		{
			name:  "invalid k8s version",
			query: "?k8s=bad",
		},
		{
			name:  "invalid os family",
			query: "?os=windows",
		},
		{
			name:  "invalid service type",
			query: "?service=invalid",
		},
		{
			name:  "invalid gpu type",
			query: "?gpu=t4",
		},
		{
			name:  "invalid intent type",
			query: "?intent=testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/recipes"+tt.query, nil)
			w := httptest.NewRecorder()

			builder.HandleRecipes(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestHandleRecipes_Success(t *testing.T) {
	// Set up test data
	t.Cleanup(setRecommendationData(t, `base:
  - type: K8s
    subtypes:
      - subtype: control-plane
        data:
          version: "1.28.3"
overlays: []
`))

	builder := NewBuilder()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "empty query",
			query: "",
		},
		{
			name:  "with os",
			query: "?os=ubuntu",
		},
		{
			name:  "with service",
			query: "?service=eks",
		},
		{
			name:  "full query",
			query: "?os=ubuntu&osv=22.04&service=eks&gpu=h100&intent=training",
		},
		{
			name:  "with context",
			query: "?os=ubuntu&context=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/recipes"+tt.query, nil)
			w := httptest.NewRecorder()

			builder.HandleRecipes(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			// Verify JSON response
			var recipe Recipe
			if err := json.Unmarshal(w.Body.Bytes(), &recipe); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			// Verify basic structure
			if recipe.PayloadVersion != RecipeAPIVersion {
				t.Errorf("expected version %s, got %s", RecipeAPIVersion, recipe.PayloadVersion)
			}

			if recipe.Measurements == nil {
				t.Error("expected measurements to be present")
			}

			if recipe.GeneratedAt.IsZero() {
				t.Error("expected GeneratedAt to be set")
			}
		})
	}
}

func TestHandleRecipes_CachingHeaders(t *testing.T) {
	t.Cleanup(setRecommendationData(t, `base:
  - type: K8s
    subtypes:
      - subtype: control-plane
        data:
          version: "1.28.3"
overlays: []
`))

	builder := NewBuilder()
	req := httptest.NewRequest(http.MethodGet, "/recipes?os=ubuntu", nil)
	w := httptest.NewRecorder()

	builder.HandleRecipes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "public") {
		t.Errorf("expected Cache-Control to contain 'public', got %s", cacheControl)
	}

	if !strings.Contains(cacheControl, "max-age=600") {
		t.Errorf("expected Cache-Control to contain 'max-age=600', got %s", cacheControl)
	}
}

func TestHandleRecipes_ContextParameter(t *testing.T) {
	t.Cleanup(setRecommendationData(t, `base:
  - type: SystemD
    subtypes:
      - subtype: containerd.service
        context:
          source: systemctl
        data:
          CPUAccounting: true
overlays: []
`))

	builder := NewBuilder()

	t.Run("with context=true", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/recipes?os=ubuntu&context=true", nil)
		w := httptest.NewRecorder()

		builder.HandleRecipes(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var recipe Recipe
		if err := json.Unmarshal(w.Body.Bytes(), &recipe); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Verify context is present
		hasContext := false
		for _, m := range recipe.Measurements {
			if m == nil {
				continue
			}
			for _, st := range m.Subtypes {
				if len(st.Context) > 0 {
					hasContext = true
					break
				}
			}
		}

		if !hasContext {
			t.Error("expected context to be present when context=true")
		}
	})

	t.Run("without context (default)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/recipes?os=ubuntu", nil)
		w := httptest.NewRecorder()

		builder.HandleRecipes(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var recipe Recipe
		if err := json.Unmarshal(w.Body.Bytes(), &recipe); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Verify context is stripped
		hasContext := false
		for _, m := range recipe.Measurements {
			if m == nil {
				continue
			}
			for _, st := range m.Subtypes {
				if len(st.Context) > 0 {
					hasContext = true
					break
				}
			}
		}

		if hasContext {
			t.Error("expected context to be stripped when not requested")
		}
	})
}

func TestBuildRecipe_ValidatesNilQuery(t *testing.T) {
	builder := NewBuilder()
	_, err := builder.Build(context.TODO(), nil)
	if err == nil {
		t.Fatal("expected error for nil query")
	}
}
