package workflow

import (
	"regexp"
	"testing"
)

// TestErrorPatternsNotOverlyAggressive tests that error patterns don't match informational text
// This addresses the issue where patterns like "(?i)unauthorized" were too broad and matched
// any mention of the word, not just actual errors.
func TestErrorPatternsNotOverlyAggressive(t *testing.T) {
	// Test cases with informational text that should NOT match error patterns
	informationalText := []string{
		// Claude reasoning/thinking text
		"I'll check if the user is unauthorized to access this resource.",
		"The API returned 401 Unauthorized, which means we need to authenticate.",
		"Permission was denied because the token expired.",
		"This endpoint is forbidden without admin privileges.",
		"Access is restricted to team members only.",
		"The function is not authorized to perform this operation.",
		"The token is invalid and needs to be refreshed.",

		// API response descriptions
		"Received 403 forbidden status code from the server.",
		"HTTP response: 401 Unauthorized",
		"Status: forbidden (403)",

		// Informational context
		"Checking if access is restricted...",
		"User permissions: denied for write operations",
		"Insufficient permission to modify this resource",
	}

	// Test cases with actual ERROR context that SHOULD match
	errorText := []string{
		"ERROR: permission was denied for this operation",
		"Error: unauthorized access attempt detected",
		"ERROR: forbidden action attempted",
		"Error: access is restricted for this user",
		"ERROR: insufficient permission to proceed",
		"stream error: unauthorized access",
		"[ERROR] token is invalid",
	}

	engines := []CodingAgentEngine{
		NewClaudeEngine(),
		NewCodexEngine(),
		NewCopilotEngine(),
	}

	for _, engine := range engines {
		t.Run(engine.GetID()+"_does_not_match_informational_text", func(t *testing.T) {
			patterns := engine.GetErrorPatterns()

			for _, text := range informationalText {
				for _, pattern := range patterns {
					// Compile the pattern
					regex, err := regexp.Compile(pattern.Pattern)
					if err != nil {
						t.Fatalf("Pattern failed to compile: %v\nPattern: %s", err, pattern.Pattern)
					}

					// Check if it matches informational text (it shouldn't)
					if regex.MatchString(text) {
						// Additional check: if it matches, it should require "error" context
						if !hasErrorContext(pattern.Pattern) {
							t.Errorf("Pattern '%s' (%s) matched informational text:\n  Text: %s\n  This pattern is too aggressive!",
								pattern.Pattern, pattern.Description, text)
						}
					}
				}
			}
		})

		t.Run(engine.GetID()+"_matches_actual_errors", func(t *testing.T) {
			patterns := engine.GetErrorPatterns()

			// Skip if engine has no patterns (patterns are now in JavaScript)
			if len(patterns) == 0 {
				t.Skip("Engine has no Go patterns - patterns are now defined in JavaScript")
			}

			// At least some patterns should match actual error text
			matchedCount := 0
			for _, text := range errorText {
				for _, pattern := range patterns {
					regex, err := regexp.Compile(pattern.Pattern)
					if err != nil {
						continue
					}

					if regex.MatchString(text) {
						matchedCount++
						break // Count each error text once
					}
				}
			}

			if matchedCount == 0 {
				t.Errorf("No patterns matched any of the actual error text - patterns may be too restrictive")
			}
		})
	}
}

// hasErrorContext checks if a pattern requires explicit error context (like "error", "ERROR", etc.)
// This helper function is used to distinguish between contextual patterns (that require error markers)
// and overly broad patterns (that match any occurrence of keywords).
//
// Parameters:
//   - pattern: The regex pattern string to analyze
//
// Returns:
//   - true if the pattern contains explicit error context markers (error, failed, etc.)
//   - false if the pattern is a generic keyword match without error context
func hasErrorContext(pattern string) bool {
	// Patterns that explicitly require "error" somewhere in the match
	errorMarkers := []string{
		"error",
		"ERROR",
		"Error",
		"stream\\s+error",
		"\\[ERROR\\]",
		"failed",
		"authentication failed",
	}

	for _, marker := range errorMarkers {
		if regexp.MustCompile(marker).MatchString(pattern) {
			return true
		}
	}

	return false
}

// TestSpecificPatternExamples tests specific pattern improvements
func TestSpecificPatternExamples(t *testing.T) {
	testCases := []struct {
		name           string
		pattern        string
		shouldMatch    []string
		shouldNotMatch []string
	}{
		{
			name:    "error_permission_denied_requires_error_context",
			pattern: `(?i)error.*permission.*denied`,
			shouldMatch: []string{
				"ERROR: permission was denied",
				"stream error: permission denied for user",
				"Error: Permission Denied",
			},
			shouldNotMatch: []string{
				"permission was denied because token expired",
				"Permission denied for this operation",
				"The API returned permission denied",
			},
		},
		{
			name:    "error_unauthorized_requires_error_context",
			pattern: `(?i)error.*unauthorized`,
			shouldMatch: []string{
				"ERROR: unauthorized access",
				"stream error: unauthorized request",
				"Error: Unauthorized",
			},
			shouldNotMatch: []string{
				"401 Unauthorized",
				"The user is unauthorized",
				"API returned unauthorized status",
			},
		},
		{
			name:    "error_forbidden_requires_error_context",
			pattern: `(?i)error.*forbidden`,
			shouldMatch: []string{
				"ERROR: forbidden action",
				"stream error: forbidden operation",
				"Error: Forbidden",
			},
			shouldNotMatch: []string{
				"403 Forbidden",
				"This action is forbidden",
				"API returned forbidden status",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			regex, err := regexp.Compile(tc.pattern)
			if err != nil {
				t.Fatalf("Pattern failed to compile: %v", err)
			}

			for _, text := range tc.shouldMatch {
				if !regex.MatchString(text) {
					t.Errorf("Pattern should match but didn't:\n  Pattern: %s\n  Text: %s",
						tc.pattern, text)
				}
			}

			for _, text := range tc.shouldNotMatch {
				if regex.MatchString(text) {
					t.Errorf("Pattern should NOT match but did:\n  Pattern: %s\n  Text: %s",
						tc.pattern, text)
				}
			}
		})
	}
}
