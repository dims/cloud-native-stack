package serializer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testData struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func TestRespondJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	data := testData{
		Message: "success",
		Code:    200,
	}

	RespondJSON(w, http.StatusOK, data)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Verify response body
	var result testData
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.Message != data.Message {
		t.Errorf("expected message %s, got %s", data.Message, result.Message)
	}

	if result.Code != data.Code {
		t.Errorf("expected code %d, got %d", data.Code, result.Code)
	}
}

func TestRespondJSON_DifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"BadRequest", http.StatusBadRequest},
		{"NotFound", http.StatusNotFound},
		{"InternalServerError", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			data := testData{Message: tt.name, Code: tt.statusCode}

			RespondJSON(w, tt.statusCode, data)

			if w.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, w.Code)
			}
		})
	}
}

func TestRespondJSON_EncodingError(t *testing.T) {
	w := httptest.NewRecorder()

	// Create data that cannot be JSON encoded
	// Channels cannot be marshaled to JSON
	badData := make(chan int)

	RespondJSON(w, http.StatusOK, badData)

	// Should return internal server error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d for encoding error, got %d", http.StatusInternalServerError, w.Code)
	}

	// Should have error message
	if w.Body.Len() == 0 {
		t.Error("expected error message in body")
	}
}

func TestRespondJSON_ComplexData(t *testing.T) {
	w := httptest.NewRecorder()

	type nested struct {
		Field1 string
		Field2 int
	}

	complexData := map[string]interface{}{
		"string": "value",
		"number": 42,
		"bool":   true,
		"nested": nested{Field1: "test", Field2: 123},
		"array":  []int{1, 2, 3},
		"null":   nil,
	}

	RespondJSON(w, http.StatusOK, complexData)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal complex response: %v", err)
	}

	// Verify some fields
	if result["string"] != "value" {
		t.Errorf("expected string field to be 'value', got %v", result["string"])
	}

	if result["number"].(float64) != 42 {
		t.Errorf("expected number field to be 42, got %v", result["number"])
	}
}

func TestRespondJSON_EmptyData(t *testing.T) {
	w := httptest.NewRecorder()

	RespondJSON(w, http.StatusOK, nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// nil encodes to "null\n" in JSON
	body := w.Body.String()
	if body != "null\n" {
		t.Errorf("expected 'null\\n', got %q", body)
	}
}

func TestRespondJSON_BuffersBeforeWritingHeaders(t *testing.T) {
	// This test verifies that RespondJSON buffers the JSON
	// before writing headers, so encoding errors don't result
	// in partial responses

	w := httptest.NewRecorder()

	// Bad data that will fail encoding
	badData := make(chan int)

	RespondJSON(w, http.StatusOK, badData)

	// If buffering works correctly, we should get a 500 error
	// If it doesn't buffer, we'd get a 200 with an error body
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected buffering to prevent status %d, got %d", http.StatusOK, w.Code)
	}
}
