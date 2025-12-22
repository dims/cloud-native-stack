package measurement

import "strings"

// FilterOut returns a new map with keys filtered out based on the provided patterns.
// Supports wildcard patterns:
//   - "prefix*" matches keys starting with "prefix"
//   - "*suffix" matches keys ending with "suffix"
//   - "*contains*" matches keys containing "contains"
//   - "exact" matches keys exactly
func FilterOut(readings map[string]Reading, keys []string) map[string]Reading {
	result := make(map[string]Reading)

	for key, value := range readings {
		omit := false
		for _, pattern := range keys {
			if matchesPattern(key, pattern) {
				omit = true
				break
			}
		}
		if !omit {
			result[key] = value
		}
	}

	return result
}

// matchesPattern checks if a key matches a wildcard pattern.
func matchesPattern(key, pattern string) bool {
	// No wildcard - exact match
	if !strings.Contains(pattern, "*") {
		return key == pattern
	}

	// *suffix* - contains match
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		substr := strings.Trim(pattern, "*")
		return strings.Contains(key, substr)
	}

	// *suffix - ends with match
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(key, suffix)
	}

	// prefix* - starts with match
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(key, prefix)
	}

	return false
}
