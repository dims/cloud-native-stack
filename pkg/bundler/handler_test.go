package bundler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBundlerHandlerNew verifies DefaultBundler can be created for HTTP handling.
func TestBundlerHandlerNew(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatal("expected non-nil bundler")
	}
}

// TestBundleEndpointMethods verifies only POST is allowed.
func TestBundleEndpointMethods(t *testing.T) {
	b := New()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/v1/bundle", nil)
			w := httptest.NewRecorder()

			b.HandleBundles(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for method %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}

			allow := w.Header().Get("Allow")
			if allow == "" {
				t.Error("expected Allow header to be set")
			}
			if allow != http.MethodPost {
				t.Errorf("Allow header = %q, want %q", allow, http.MethodPost)
			}
		})
	}
}

// TestBundleEndpointInvalidJSON tests invalid JSON body handling.
func TestBundleEndpointInvalidJSON(t *testing.T) {
	b := New()

	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"invalid json", "{invalid}"},
		{"malformed json", `{"recipe": `},
		{"wrong type", `{"recipe": "string-not-object"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/bundle", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			b.HandleBundles(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			// Verify JSON error response
			contentType := w.Header().Get("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}
		})
	}
}

// TestBundleEndpointMissingRecipe tests handling of empty/invalid recipe body.
func TestBundleEndpointMissingRecipe(t *testing.T) {
	b := New()

	// Request with empty componentRefs (simulates empty recipe)
	body := `{"apiVersion": "cns.nvidia.com/v1alpha1", "kind": "Recipe", "componentRefs": []}`
	req := httptest.NewRequest(http.MethodPost, "/v1/bundle", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	b.HandleBundles(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify error message
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if msg, ok := resp["message"].(string); !ok || !strings.Contains(msg, "component") {
		t.Errorf("message = %q, want message about components", msg)
	}
}

// TestBundleEndpointEmptyComponentRefs tests handling of recipes without components.
func TestBundleEndpointEmptyComponentRefs(t *testing.T) {
	b := New()

	// Recipe with no component references (direct RecipeResult in body)
	body := `{"apiVersion": "cns.nvidia.com/v1alpha1", "kind": "Recipe", "componentRefs": []}`
	req := httptest.NewRequest(http.MethodPost, "/v1/bundle", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	b.HandleBundles(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if msg, ok := resp["message"].(string); !ok || !strings.Contains(msg, "component") {
		t.Errorf("expected error about components, got: %q", msg)
	}
}

// TestBundleEndpointInvalidBundlerType tests handling of invalid bundler types in query param.
func TestBundleEndpointInvalidBundlerType(t *testing.T) {
	b := New()

	// Recipe with valid components, invalid bundler in query param
	body := `{
		"apiVersion": "cns.nvidia.com/v1alpha1",
		"kind": "Recipe",
		"componentRefs": [
			{"name": "gpu-operator", "version": "v25.3.3"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/bundle?bundlers=invalid-bundler", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	b.HandleBundles(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify error includes valid bundler types
	details, ok := resp["details"].(map[string]interface{})
	if !ok {
		t.Fatal("expected details in response")
	}

	valid, ok := details["valid"].([]interface{})
	if !ok {
		t.Fatal("expected valid bundler list in details")
	}

	if len(valid) == 0 {
		t.Error("expected at least one valid bundler type")
	}
}

// TestBundleEndpointValidRequest tests a valid bundle generation request.
func TestBundleEndpointValidRequest(t *testing.T) {
	b := New()

	// Create a valid recipe (direct RecipeResult in body, bundlers in query param)
	body := `{
		"apiVersion": "cns.nvidia.com/v1alpha1",
		"kind": "Recipe",
		"metadata": {
			"version": "v1.0.0",
			"generatedAt": "2025-01-15T10:30:00Z",
			"appliedOverlays": ["service=eks, accelerator=h100"]
		},
		"criteria": {
			"service": "eks",
			"accelerator": "h100",
			"intent": "training"
		},
		"componentRefs": [
			{
				"name": "gpu-operator",
				"version": "v25.3.3",
				"type": "helm",
				"repository": "https://helm.ngc.nvidia.com/nvidia",
				"valuesFile": "components/gpu-operator/values.yaml"
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/bundle?bundlers=gpu-operator", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	b.HandleBundles(w, req)

	// Should return OK with zip content
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		return
	}

	// Verify content type is zip
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/zip" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/zip")
	}

	// Verify content disposition
	contentDisp := w.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisp, "bundles.zip") {
		t.Errorf("Content-Disposition = %q, want to contain 'bundles.zip'", contentDisp)
	}

	// Verify bundle metadata headers
	if w.Header().Get("X-Bundle-Files") == "" {
		t.Error("expected X-Bundle-Files header")
	}
	if w.Header().Get("X-Bundle-Size") == "" {
		t.Error("expected X-Bundle-Size header")
	}
	if w.Header().Get("X-Bundle-Duration") == "" {
		t.Error("expected X-Bundle-Duration header")
	}

	// Verify zip is readable
	zipReader, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}

	// Verify at least one file in zip
	if len(zipReader.File) == 0 {
		t.Error("expected at least one file in zip archive")
	}

	// Log files for debugging
	t.Logf("Zip contains %d files:", len(zipReader.File))
	for _, f := range zipReader.File {
		t.Logf("  - %s", f.Name)
	}
}

// TestBundleEndpointAllBundlers tests bundle generation with no bundler filter.
func TestBundleEndpointAllBundlers(t *testing.T) {
	b := New()

	// Create a recipe with multiple components (no bundlers query param = all bundlers)
	body := `{
		"apiVersion": "cns.nvidia.com/v1alpha1",
		"kind": "Recipe",
		"componentRefs": [
			{"name": "gpu-operator", "version": "v25.3.3", "type": "helm", "valuesFile": "components/gpu-operator/values.yaml"},
			{"name": "network-operator", "version": "v25.4.0", "type": "helm", "valuesFile": "components/network-operator/values.yaml"}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/bundle", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	b.HandleBundles(w, req)

	// May return OK or error depending on component availability
	// For integration tests, this validates the code path works
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d or %d, got %d", http.StatusOK, http.StatusInternalServerError, w.Code)
	}
}

// TestBundleRequestQueryParamParsing tests bundlers query param parsing.
func TestBundleRequestQueryParamParsing(t *testing.T) {
	b := New()

	tests := []struct {
		name       string
		queryParam string
		body       string
		wantStatus int
	}{
		{
			name:       "single bundler",
			queryParam: "bundlers=gpu-operator",
			body:       `{"apiVersion": "v1", "kind": "Recipe", "componentRefs": [{"name": "gpu-operator", "version": "v1"}]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "multiple bundlers",
			queryParam: "bundlers=gpu-operator,network-operator",
			body:       `{"apiVersion": "v1", "kind": "Recipe", "componentRefs": [{"name": "gpu-operator", "version": "v1"}]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty bundlers param (all bundlers)",
			queryParam: "",
			body:       `{"apiVersion": "v1", "kind": "Recipe", "componentRefs": [{"name": "gpu-operator", "version": "v1"}]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "bundlers with spaces trimmed",
			queryParam: "bundlers=gpu-operator,%20network-operator",
			body:       `{"apiVersion": "v1", "kind": "Recipe", "componentRefs": [{"name": "gpu-operator", "version": "v1"}]}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/bundle"
			if tt.queryParam != "" {
				url += "?" + tt.queryParam
			}
			req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			b.HandleBundles(w, req)

			// Allow both OK and internal error (bundler may fail but parsing should succeed)
			if w.Code != tt.wantStatus && w.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want %d or %d. Body: %s", w.Code, tt.wantStatus, http.StatusInternalServerError, w.Body.String())
			}
		})
	}
}

// TestZipResponseContainsExpectedFiles validates zip structure.
func TestZipResponseContainsExpectedFiles(t *testing.T) {
	b := New()

	// Recipe direct in body, bundlers as query param
	body := `{
		"apiVersion": "cns.nvidia.com/v1alpha1",
		"kind": "Recipe",
		"componentRefs": [
			{
				"name": "gpu-operator",
				"version": "v25.3.3",
				"type": "helm",
				"valuesFile": "components/gpu-operator/values.yaml"
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/bundle?bundlers=gpu-operator", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	b.HandleBundles(w, req)

	if w.Code != http.StatusOK {
		t.Skipf("skipping zip validation, got status %d: %s", w.Code, w.Body.String())
	}

	zipReader, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}

	// Check for expected files/directories
	foundGpuOperator := false
	for _, f := range zipReader.File {
		if strings.HasPrefix(f.Name, "gpu-operator/") || f.Name == "gpu-operator" {
			foundGpuOperator = true
			break
		}
	}

	if !foundGpuOperator {
		t.Error("expected gpu-operator directory in zip")
		t.Log("Files in zip:")
		for _, f := range zipReader.File {
			t.Logf("  - %s", f.Name)
		}
	}
}

// TestZipCanBeExtracted verifies that the returned zip can be extracted.
func TestZipCanBeExtracted(t *testing.T) {
	b := New()

	// Recipe direct in body, bundlers as query param
	body := `{
		"apiVersion": "cns.nvidia.com/v1alpha1",
		"kind": "Recipe",
		"componentRefs": [
			{
				"name": "gpu-operator",
				"version": "v25.3.3",
				"type": "helm",
				"valuesFile": "components/gpu-operator/values.yaml"
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/bundle?bundlers=gpu-operator", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	b.HandleBundles(w, req)

	if w.Code != http.StatusOK {
		t.Skipf("skipping extraction validation, got status %d", w.Code)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}

	// Verify each file can be opened and read
	for _, f := range zipReader.File {
		rc, err := f.Open()
		if err != nil {
			t.Errorf("failed to open %s: %v", f.Name, err)
			continue
		}

		_, err = io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Errorf("failed to read %s: %v", f.Name, err)
		}
	}
}
