package workflow

import (
	"regexp"
	"strings"
	"testing"
)

// TestActionPinsExist verifies that all action pinning entries exist
func TestActionPinsExist(t *testing.T) {
	expectedActions := []string{
		"actions/checkout@v5",
		"actions/github-script@v8",
		"actions/upload-artifact@v4",
		"actions/download-artifact@v5",
		"actions/cache@v4",
		"actions/setup-node@v4",
		"actions/setup-python@v5",
		"actions/setup-go@v5",
		"actions/setup-java@v4",
		"actions/setup-dotnet@v4",
		"erlef/setup-beam@v1",
		"haskell-actions/setup@v2",
		"ruby/setup-ruby@v1",
		"astral-sh/setup-uv@v5",
	}

	for _, action := range expectedActions {
		pin, exists := actionPins[action]
		if !exists {
			t.Errorf("Missing action pin for %s", action)
			continue
		}

		// Verify the pin has a valid SHA (40 character hex string)
		if !isValidSHA(pin.SHA) {
			t.Errorf("Invalid SHA for %s: %s (expected 40-character hex string)", action, pin.SHA)
		}
	}
}

// TestGetActionPinReturnsValidSHA tests that GetActionPin returns valid SHA references
func TestGetActionPinReturnsValidSHA(t *testing.T) {
	tests := []struct {
		repo    string
		version string
		wantSHA bool
	}{
		{"actions/checkout", "v5", true},
		{"actions/github-script", "v8", true},
		{"actions/upload-artifact", "v4", true},
		{"actions/download-artifact", "v5", true},
		{"actions/cache", "v4", true},
		{"actions/setup-node", "v4", true},
		{"actions/setup-python", "v5", true},
		{"actions/setup-go", "v5", true},
		{"actions/setup-java", "v4", true},
		{"actions/setup-dotnet", "v4", true},
		{"erlef/setup-beam", "v1", true},
		{"haskell-actions/setup", "v2", true},
		{"ruby/setup-ruby", "v1", true},
		{"astral-sh/setup-uv", "v5", true},
	}

	for _, tt := range tests {
		t.Run(tt.repo+"@"+tt.version, func(t *testing.T) {
			result := GetActionPin(tt.repo, tt.version)
			
			// Check that the result contains a SHA (40-char hex after @)
			parts := strings.Split(result, "@")
			if len(parts) != 2 {
				t.Errorf("GetActionPin(%s, %s) = %s, expected format repo@sha", tt.repo, tt.version, result)
				return
			}

			if tt.wantSHA {
				if !isValidSHA(parts[1]) {
					t.Errorf("GetActionPin(%s, %s) = %s, expected SHA to be 40-char hex", tt.repo, tt.version, result)
				}
			}
		})
	}
}

// TestGetActionPinFallback tests that GetActionPin falls back to version if no pin exists
func TestGetActionPinFallback(t *testing.T) {
	result := GetActionPin("unknown/action", "v1")
	expected := "unknown/action@v1"
	if result != expected {
		t.Errorf("GetActionPin(unknown/action, v1) = %s, want %s", result, expected)
	}
}

// isValidSHA checks if a string is a valid 40-character hexadecimal SHA
func isValidSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	matched, _ := regexp.MatchString("^[0-9a-f]{40}$", s)
	return matched
}
