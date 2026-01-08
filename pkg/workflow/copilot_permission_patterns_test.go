package workflow

import (
	"regexp"
	"testing"
)

// TestCopilotPermissionPatternsAsWarnings verifies that permission-related patterns
// in the Copilot engine are properly marked as warnings (not errors)
func TestCopilotPermissionPatternsAsWarnings(t *testing.T) {
	engine := NewCopilotEngine()
	patterns := engine.GetErrorPatterns()

	// Skip if engine has no patterns (patterns are now in JavaScript)
	if len(patterns) == 0 {
		t.Skip("Engine has no Go patterns - patterns are now defined in JavaScript")
	}

	// Test cases with permission-related error messages
	testCases := []struct {
		name            string
		logContent      string
		shouldMatch     bool
		expectedLevel   string
		expectedPattern string
	}{
		{
			name:            "permission_denied_with_error_prefix",
			logContent:      "ERROR: permission denied for this operation",
			shouldMatch:     true,
			expectedLevel:   "warning",
			expectedPattern: "copilot-permission-denied",
		},
		{
			name:            "permission_denied_stream_error",
			logContent:      "stream error: permission denied for user",
			shouldMatch:     true,
			expectedLevel:   "warning",
			expectedPattern: "copilot-permission-denied",
		},
		{
			name:            "error_permission_denied_capitalized",
			logContent:      "Error: Permission Denied",
			shouldMatch:     true,
			expectedLevel:   "warning",
			expectedPattern: "copilot-permission-denied",
		},
		{
			name:            "unauthorized_with_error",
			logContent:      "ERROR: unauthorized access attempt",
			shouldMatch:     true,
			expectedLevel:   "warning",
			expectedPattern: "copilot-unauthorized",
		},
		{
			name:            "stream_error_unauthorized",
			logContent:      "stream error: unauthorized",
			shouldMatch:     true,
			expectedLevel:   "warning",
			expectedPattern: "copilot-unauthorized",
		},
		{
			name:            "forbidden_with_error",
			logContent:      "ERROR: forbidden action",
			shouldMatch:     true,
			expectedLevel:   "warning",
			expectedPattern: "copilot-forbidden",
		},
		{
			name:            "error_forbidden_lowercase",
			logContent:      "error: forbidden",
			shouldMatch:     true,
			expectedLevel:   "warning",
			expectedPattern: "copilot-forbidden",
		},
		// These should NOT match (no error context)
		{
			name:          "permission_denied_informational_no_error",
			logContent:    "permission was denied because token expired",
			shouldMatch:   false,
			expectedLevel: "",
		},
		{
			name:          "permission_denied_plain_text",
			logContent:    "Permission denied for this operation",
			shouldMatch:   false,
			expectedLevel: "",
		},
		{
			name:          "api_returned_permission_denied",
			logContent:    "The API returned permission denied",
			shouldMatch:   false,
			expectedLevel: "",
		},
		{
			name:          "unauthorized_informational",
			logContent:    "401 Unauthorized",
			shouldMatch:   false,
			expectedLevel: "",
		},
		{
			name:          "user_is_unauthorized",
			logContent:    "The user is unauthorized",
			shouldMatch:   false,
			expectedLevel: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			var matchedPattern *ErrorPattern

			for _, pattern := range patterns {
				regex, err := regexp.Compile(pattern.Pattern)
				if err != nil {
					t.Fatalf("Pattern failed to compile: %v\nPattern: %s", err, pattern.Pattern)
				}

				if regex.MatchString(tc.logContent) {
					matched = true
					matchedPattern = &pattern

					// If we expect this to match a specific pattern, verify it
					if tc.shouldMatch && tc.expectedPattern != "" && pattern.ID == tc.expectedPattern {
						break
					}
				}
			}

			if tc.shouldMatch && !matched {
				t.Errorf("Expected log content to match a pattern but it didn't:\n  Content: %s", tc.logContent)
			}

			if !tc.shouldMatch && matched {
				t.Errorf("Expected log content NOT to match any pattern but it matched:\n  Content: %s\n  Pattern: %s (%s)",
					tc.logContent, matchedPattern.Pattern, matchedPattern.Description)
			}

			// Verify severity is "warning" for matched permission patterns
			if matched && matchedPattern != nil && tc.expectedLevel != "" {
				if matchedPattern.Severity != tc.expectedLevel {
					t.Errorf("Expected severity '%s' for pattern %s, got '%s'",
						tc.expectedLevel, matchedPattern.ID, matchedPattern.Severity)
				}
			}
		})
	}
}

// TestCopilotPermissionPatternsRequireErrorContext verifies that permission patterns
// require "error" context and don't match plain informational text
func TestCopilotPermissionPatternsRequireErrorContext(t *testing.T) {
	engine := NewCopilotEngine()
	patterns := engine.GetErrorPatterns()

	// Skip if engine has no patterns (patterns are now in JavaScript)
	if len(patterns) == 0 {
		t.Skip("Engine has no Go patterns - patterns are now defined in JavaScript")
	}

	// Get the permission-related patterns
	permissionPatterns := []ErrorPattern{}
	for _, pattern := range patterns {
		if pattern.ID == "copilot-permission-denied" ||
			pattern.ID == "copilot-unauthorized" ||
			pattern.ID == "copilot-forbidden" {
			permissionPatterns = append(permissionPatterns, pattern)
		}
	}

	if len(permissionPatterns) != 3 {
		t.Fatalf("Expected 3 permission patterns, got %d", len(permissionPatterns))
	}

	// Informational text that should NOT match
	informationalText := []string{
		"permission was denied because token expired",
		"Permission denied for this operation",
		"The API returned permission denied",
		"401 Unauthorized",
		"The user is unauthorized",
		"Access is forbidden without admin privileges",
		"forbidden status code received",
	}

	for _, text := range informationalText {
		for _, pattern := range permissionPatterns {
			regex, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				t.Fatalf("Pattern failed to compile: %v", err)
			}

			if regex.MatchString(text) {
				t.Errorf("Pattern %s should NOT match informational text:\n  Text: %s\n  Pattern: %s",
					pattern.ID, text, pattern.Pattern)
			}
		}
	}

	// Error context text that SHOULD match
	errorText := []string{
		"ERROR: permission denied",
		"error: unauthorized",
		"Error: forbidden",
		"stream error: permission denied for user",
	}

	for _, text := range errorText {
		matched := false
		for _, pattern := range permissionPatterns {
			regex, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				t.Fatalf("Pattern failed to compile: %v", err)
			}

			if regex.MatchString(text) {
				matched = true
				// Verify it's marked as warning
				if pattern.Severity != "warning" {
					t.Errorf("Pattern %s should be marked as 'warning', got '%s'",
						pattern.ID, pattern.Severity)
				}
				break
			}
		}

		if !matched {
			t.Errorf("At least one pattern should match error text: %s", text)
		}
	}
}
