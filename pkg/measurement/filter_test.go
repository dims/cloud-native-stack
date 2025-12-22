package measurement

import "testing"

func TestFilterOut(t *testing.T) {
	// Create test data
	readings := map[string]Reading{
		"root_user":       Str("admin"),
		"root_password":   Str("secret"),
		"user_root":       Str("value1"),
		"some_root_value": Str("value2"),
		"normal_key":      Str("value3"),
		"another_key":     Int(42),
		"root":            Bool(true),
	}

	tests := []struct {
		name     string
		patterns []string
		wantKeys []string
	}{
		{
			name:     "exact match",
			patterns: []string{"root"},
			wantKeys: []string{"root_user", "root_password", "user_root", "some_root_value", "normal_key", "another_key"},
		},
		{
			name:     "prefix wildcard - root*",
			patterns: []string{"root*"},
			wantKeys: []string{"user_root", "some_root_value", "normal_key", "another_key"},
		},
		{
			name:     "suffix wildcard - *root",
			patterns: []string{"*root"},
			wantKeys: []string{"root_user", "root_password", "some_root_value", "normal_key", "another_key"},
		},
		{
			name:     "contains wildcard - *root*",
			patterns: []string{"*root*"},
			wantKeys: []string{"normal_key", "another_key"},
		},
		{
			name:     "multiple patterns",
			patterns: []string{"root*", "*key"},
			wantKeys: []string{"user_root", "some_root_value"},
		},
		{
			name:     "no patterns",
			patterns: []string{},
			wantKeys: []string{"root_user", "root_password", "user_root", "some_root_value", "normal_key", "another_key", "root"},
		},
		{
			name:     "non-matching pattern",
			patterns: []string{"nonexistent*"},
			wantKeys: []string{"root_user", "root_password", "user_root", "some_root_value", "normal_key", "another_key", "root"},
		},
		{
			name:     "multiple wildcards",
			patterns: []string{"*_*"},
			wantKeys: []string{"root"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterOut(readings, tt.patterns)

			// Check that result has the expected number of keys
			if len(result) != len(tt.wantKeys) {
				t.Errorf("FilterOut() returned %d keys, want %d", len(result), len(tt.wantKeys))
			}

			// Check that all expected keys are present
			for _, wantKey := range tt.wantKeys {
				if _, exists := result[wantKey]; !exists {
					t.Errorf("FilterOut() missing expected key %q", wantKey)
				}
			}

			// Check that no unexpected keys are present
			for key := range result {
				found := false
				for _, wantKey := range tt.wantKeys {
					if key == wantKey {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("FilterOut() contains unexpected key %q", key)
				}
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		pattern string
		want    bool
	}{
		// Exact matches
		{"exact match - same", "root", "root", true},
		{"exact match - different", "root", "admin", false},

		// Prefix wildcards
		{"prefix wildcard - matches", "root_user", "root*", true},
		{"prefix wildcard - no match", "user_root", "root*", false},
		{"prefix wildcard - empty prefix", "anything", "*", true},

		// Suffix wildcards
		{"suffix wildcard - matches", "user_root", "*root", true},
		{"suffix wildcard - no match", "root_user", "*root", false},

		// Contains wildcards
		{"contains wildcard - matches", "some_root_value", "*root*", true},
		{"contains wildcard - at start", "root_value", "*root*", true},
		{"contains wildcard - at end", "value_root", "*root*", true},
		{"contains wildcard - no match", "value", "*root*", false},

		// Edge cases
		{"empty pattern", "key", "", false},
		{"empty key", "", "pattern", false},
		{"both empty", "", "", true},
		{"multiple asterisks in middle", "abc", "a*c", false}, // Not supported, treated as exact
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesPattern(tt.key, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.key, tt.pattern, got, tt.want)
			}
		})
	}
}
