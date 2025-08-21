package mapper

import (
	"errors"
	"strings"
)

// decodeJSONPointer decodes an RFC6901 pointer (e.g. "/jobs/build/steps/0/uses")
// into segments: ["jobs","build","steps","0","uses"].
// Returns empty slice for "" or "/".
func decodeJSONPointer(ptr string) ([]string, error) {
	if ptr == "" || ptr == "/" {
		return []string{}, nil
	}
	if !strings.HasPrefix(ptr, "/") {
		return nil, errors.New("invalid json pointer: must start with '/'")
	}
	parts := strings.Split(ptr[1:], "/")
	for i, p := range parts {
		// Unescape per RFC6901
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		parts[i] = p
	}
	return parts, nil
}

// isIndex determines whether a segment looks like an array index
func isIndex(segment string) bool {
	if len(segment) == 0 {
		return false
	}
	// Don't allow negative numbers as JSON Pointer indices
	if segment[0] == '-' {
		return false
	}
	for _, r := range segment {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
