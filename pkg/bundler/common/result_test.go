package common

import (
	"strings"
	"testing"
	"time"
)

func TestNewResult(t *testing.T) {
	result := NewResult(BundleTypeGpuOperator)

	if result == nil {
		t.Fatal("NewResult() returned nil")
		return
	}

	if result.Type != BundleTypeGpuOperator {
		t.Errorf("Type = %v, want %v", result.Type, BundleTypeGpuOperator)
	}

	if result.Files == nil {
		t.Error("Files should be initialized")
	}

	if result.Errors == nil {
		t.Error("Errors should be initialized")
	}

	if result.Success {
		t.Error("Success should be false initially")
	}
}

func TestResult_AddFile(t *testing.T) {
	result := NewResult(BundleTypeGpuOperator)

	result.AddFile("/path/to/file1.yaml", 100)
	result.AddFile("/path/to/file2.yaml", 200)

	if len(result.Files) != 2 {
		t.Errorf("len(Files) = %d, want 2", len(result.Files))
	}

	if result.Size != 300 {
		t.Errorf("Size = %d, want 300", result.Size)
	}

	if result.Files[0] != "/path/to/file1.yaml" {
		t.Errorf("Files[0] = %s, want /path/to/file1.yaml", result.Files[0])
	}
}

func TestResult_AddError(t *testing.T) {
	result := NewResult(BundleTypeGpuOperator)

	// Add nil error (should not add anything)
	result.AddError(nil)

	// Verify error count is 0 since the error is nil
	if len(result.Errors) != 0 {
		t.Errorf("len(Errors) = %d, want 0", len(result.Errors))
	}

	// Add actual error
	result.AddError(testError{msg: "test error"})

	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(result.Errors))
	}

	if result.Errors[0] != "test error" {
		t.Errorf("Errors[0] = %s, want 'test error'", result.Errors[0])
	}
}

func TestResult_MarkSuccess(t *testing.T) {
	result := NewResult(BundleTypeGpuOperator)

	if result.Success {
		t.Error("Success should be false initially")
	}

	result.MarkSuccess()

	if !result.Success {
		t.Error("Success should be true after MarkSuccess()")
	}
}

func TestOutput_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		output *Output
		want   bool
	}{
		{
			name: "no errors",
			output: &Output{
				Errors: []BundleError{},
			},
			want: false,
		},
		{
			name: "has errors",
			output: &Output{
				Errors: []BundleError{
					{BundlerType: BundleTypeGpuOperator, Error: "failed"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.output.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutput_SuccessCount(t *testing.T) {
	output := &Output{
		Results: []*Result{
			{Type: BundleTypeGpuOperator, Success: true},
			{Type: BundleTypeNetworkOperator, Success: false},
			{Type: BundleTypeGpuOperator, Success: true},
		},
	}

	got := output.SuccessCount()
	want := 2

	if got != want {
		t.Errorf("SuccessCount() = %d, want %d", got, want)
	}
}

func TestOutput_FailureCount(t *testing.T) {
	output := &Output{
		Results: []*Result{
			{Type: BundleTypeGpuOperator, Success: true},
			{Type: BundleTypeNetworkOperator, Success: false},
			{Type: BundleTypeGpuOperator, Success: true},
		},
	}

	got := output.FailureCount()
	want := 1

	if got != want {
		t.Errorf("FailureCount() = %d, want %d", got, want)
	}
}

func TestOutput_Summary(t *testing.T) {
	output := &Output{
		Results: []*Result{
			{Type: BundleTypeGpuOperator, Success: true},
			{Type: BundleTypeNetworkOperator, Success: true},
		},
		TotalFiles:    10,
		TotalSize:     1024 * 1024 * 5, // 5 MB
		TotalDuration: 2500 * time.Millisecond,
	}

	summary := output.Summary()

	// Check key components are present
	if !strings.Contains(summary, "10 files") {
		t.Errorf("Summary missing file count: %s", summary)
	}

	if !strings.Contains(summary, "5.0 MB") {
		t.Errorf("Summary missing size: %s", summary)
	}

	if !strings.Contains(summary, "2.5s") {
		t.Errorf("Summary missing duration: %s", summary)
	}

	if !strings.Contains(summary, "2/2 bundlers") {
		t.Errorf("Summary missing success count: %s", summary)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"bytes", 100, "100 B"},
		{"kilobytes", 1024, "1.0 KB"},
		{"megabytes", 1024 * 1024, "1.0 MB"},
		{"gigabytes", 1024 * 1024 * 1024, "1.0 GB"},
		{"mixed", 1536, "1.5 KB"},
		{"large", 5 * 1024 * 1024, "5.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestOutput_ByType(t *testing.T) {
	output := &Output{
		Results: []*Result{
			{Type: BundleTypeGpuOperator, Success: true},
			{Type: BundleTypeNetworkOperator, Success: false},
		},
	}

	byType := output.ByType()

	if len(byType) != 2 {
		t.Errorf("ByType() returned %d results, want 2", len(byType))
	}

	if r, exists := byType[BundleTypeGpuOperator]; !exists {
		t.Error("ByType() missing gpu-operator result")
	} else if !r.Success {
		t.Error("gpu-operator result should be successful")
	}

	if r, exists := byType[BundleTypeNetworkOperator]; !exists {
		t.Error("ByType() missing network-operator result")
	} else if r.Success {
		t.Error("network-operator result should not be successful")
	}
}

func TestOutput_FailedBundlers(t *testing.T) {
	output := &Output{
		Results: []*Result{
			{Type: BundleTypeGpuOperator, Success: true},
			{Type: BundleTypeNetworkOperator, Success: false},
		},
		Errors: []BundleError{
			{BundlerType: BundleTypeNetworkOperator, Error: "failed"},
		},
	}

	failed := output.FailedBundlers()

	if len(failed) != 1 {
		t.Errorf("FailedBundlers() returned %d bundlers, want 1", len(failed))
	}

	if failed[0] != BundleTypeNetworkOperator {
		t.Errorf("FailedBundlers() = %v, want network-operator", failed[0])
	}
}

func TestOutput_SuccessfulBundlers(t *testing.T) {
	output := &Output{
		Results: []*Result{
			{Type: BundleTypeGpuOperator, Success: true},
			{Type: BundleTypeNetworkOperator, Success: false},
			{Type: BundleTypeGpuOperator, Success: true},
		},
	}

	successful := output.SuccessfulBundlers()

	if len(successful) != 2 {
		t.Errorf("SuccessfulBundlers() returned %d bundlers, want 2", len(successful))
	}

	for _, bundler := range successful {
		if bundler != BundleTypeGpuOperator {
			t.Errorf("SuccessfulBundlers() contains %v, expected only gpu-operator", bundler)
		}
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e testError) Error() string {
	return e.msg
}
