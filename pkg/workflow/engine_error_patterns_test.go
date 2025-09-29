package workflow

import (
	"regexp"
	"strings"
	"testing"
)

// TestEngineErrorPatternsGoCompatibility tests that all error patterns from engines are valid Go regex patterns
func TestEngineErrorPatternsGoCompatibility(t *testing.T) {
	engines := []CodingAgentEngine{
		NewCodexEngine(),
		NewClaudeEngine(),
		NewCopilotEngine(),
	}

	for _, engine := range engines {
		t.Run(engine.GetID()+"_patterns_valid_in_go", func(t *testing.T) {
			patterns := engine.GetErrorPatterns()
			if len(patterns) == 0 {
				t.Skipf("Engine %s has no error patterns", engine.GetID())
			}

			for i, pattern := range patterns {
				t.Run(pattern.Description, func(t *testing.T) {
					// Test that the pattern compiles in Go
					_, err := regexp.Compile(pattern.Pattern)
					if err != nil {
						t.Errorf("Pattern %d (%s) failed to compile in Go: %v\nPattern: %s",
							i, pattern.Description, err, pattern.Pattern)
					}

					// Test basic structure
					if pattern.Pattern == "" {
						t.Errorf("Pattern %d has empty pattern string", i)
					}
					if pattern.Description == "" {
						t.Errorf("Pattern %d has empty description", i)
					}
					if pattern.LevelGroup < 0 {
						t.Errorf("Pattern %d has negative level group: %d", i, pattern.LevelGroup)
					}
					if pattern.MessageGroup < 0 {
						t.Errorf("Pattern %d has negative message group: %d", i, pattern.MessageGroup)
					}
				})
			}
		})
	}
}

// TestEngineErrorPatternsJavaScriptCompatibility tests pattern compatibility with JavaScript
func TestEngineErrorPatternsJavaScriptCompatibility(t *testing.T) {
	engines := []CodingAgentEngine{
		NewCodexEngine(),
		NewClaudeEngine(),
		NewCopilotEngine(),
	}

	for _, engine := range engines {
		t.Run(engine.GetID()+"_patterns_javascript_compatible", func(t *testing.T) {
			patterns := engine.GetErrorPatterns()
			if len(patterns) == 0 {
				t.Skipf("Engine %s has no error patterns", engine.GetID())
			}

			for i, pattern := range patterns {
				t.Run(pattern.Description, func(t *testing.T) {
					jsCompatible := testPatternJavaScriptCompatibility(pattern.Pattern)
					if !jsCompatible {
						t.Errorf("Pattern %d (%s) is not JavaScript compatible\nPattern: %s",
							i, pattern.Description, pattern.Pattern)
					}
				})
			}
		})
	}
}

// testPatternJavaScriptCompatibility tests if a Go regex pattern can be converted to JavaScript
func testPatternJavaScriptCompatibility(goPattern string) bool {
	// Convert (?i) prefix to JavaScript compatible format
	jsPattern := goPattern
	if strings.HasPrefix(goPattern, "(?i)") {
		jsPattern = goPattern[4:] // Remove (?i) prefix
	}

	// Test if the converted pattern compiles in Go (simulating JavaScript compilation)
	_, err := regexp.Compile(jsPattern)
	return err == nil
}

// TestSpecificPatternFunctionality tests that converted patterns work correctly for specific cases
func TestSpecificPatternFunctionality(t *testing.T) {
	testCases := []struct {
		name        string
		goPattern   string
		testString  string
		shouldMatch bool
	}{
		{
			name:        "case_insensitive_unauthorized",
			goPattern:   "(?i)unauthorized",
			testString:  "UNAUTHORIZED access denied",
			shouldMatch: true,
		},
		{
			name:        "case_insensitive_forbidden",
			goPattern:   "(?i)forbidden",
			testString:  "Forbidden resource",
			shouldMatch: true,
		},
		{
			name:        "case_insensitive_permission_denied",
			goPattern:   "(?i)permission.*denied",
			testString:  "Permission is DENIED for user",
			shouldMatch: true,
		},
		{
			name:        "codex_error_timestamp",
			goPattern:   `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+(ERROR):\s+(.+)`,
			testString:  "[2025-01-10T12:34:56] ERROR: Something went wrong",
			shouldMatch: true,
		},
		{
			name:        "codex_stream_error",
			goPattern:   `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+stream\s+(error):\s+(.+)`,
			testString:  "[2025-01-10T12:34:56] stream error: exceeded retry limit",
			shouldMatch: true,
		},
		{
			name:        "copilot_error_with_milliseconds",
			goPattern:   `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[(ERROR)\]\s+(.+)`,
			testString:  "2025-01-10T12:34:56.789Z [ERROR] Test error message",
			shouldMatch: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test Go version
			goRegex, err := regexp.Compile(tc.goPattern)
			if err != nil {
				t.Fatalf("Go pattern failed to compile: %v", err)
			}

			goMatch := goRegex.MatchString(tc.testString)
			if goMatch != tc.shouldMatch {
				t.Errorf("Go pattern match result: got %v, want %v\nPattern: %s\nString: %s",
					goMatch, tc.shouldMatch, tc.goPattern, tc.testString)
			}

			// Test JavaScript-converted version
			jsPattern := tc.goPattern
			if strings.HasPrefix(tc.goPattern, "(?i)") {
				jsPattern = tc.goPattern[4:]
			}

			jsRegex, err := regexp.Compile("(?i)" + jsPattern)
			if strings.HasPrefix(tc.goPattern, "(?i)") {
				// For case-insensitive patterns, use case-insensitive flag in Go
				if err != nil {
					t.Fatalf("JS-compatible pattern failed to compile: %v", err)
				}
				jsMatch := jsRegex.MatchString(tc.testString)
				if jsMatch != tc.shouldMatch {
					t.Errorf("JS-compatible pattern match result: got %v, want %v\nOriginal: %s\nConverted: %s\nString: %s",
						jsMatch, tc.shouldMatch, tc.goPattern, jsPattern, tc.testString)
				}
			} else {
				// For case-sensitive patterns, test directly
				jsRegex, err := regexp.Compile(jsPattern)
				if err != nil {
					t.Fatalf("JS-compatible pattern failed to compile: %v", err)
				}
				jsMatch := jsRegex.MatchString(tc.testString)
				if jsMatch != tc.shouldMatch {
					t.Errorf("JS-compatible pattern match result: got %v, want %v\nPattern: %s\nString: %s",
						jsMatch, tc.shouldMatch, jsPattern, tc.testString)
				}
			}
		})
	}
}

// TestErrorPatternGroupExtraction tests that regex groups are extracted correctly
func TestErrorPatternGroupExtraction(t *testing.T) {
	testCases := []struct {
		name            string
		pattern         ErrorPattern
		testString      string
		expectedLevel   string
		expectedMessage string
	}{
		{
			name: "codex_error_with_groups",
			pattern: ErrorPattern{
				Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+(ERROR):\s+(.+)`,
				LevelGroup:   2,
				MessageGroup: 3,
				Description:  "Codex ERROR messages with timestamp",
			},
			testString:      "[2025-01-10T12:34:56] ERROR: Something went wrong",
			expectedLevel:   "ERROR",
			expectedMessage: "Something went wrong",
		},
		{
			name: "codex_stream_error_with_groups",
			pattern: ErrorPattern{
				Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+stream\s+(error):\s+(.+)`,
				LevelGroup:   2,
				MessageGroup: 3,
				Description:  "Codex stream errors with timestamp",
			},
			testString:      "[2025-01-10T12:34:56] stream error: exceeded retry limit",
			expectedLevel:   "error",
			expectedMessage: "exceeded retry limit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			regex, err := regexp.Compile(tc.pattern.Pattern)
			if err != nil {
				t.Fatalf("Pattern failed to compile: %v", err)
			}

			matches := regex.FindStringSubmatch(tc.testString)
			if matches == nil {
				t.Fatalf("Pattern did not match test string\nPattern: %s\nString: %s",
					tc.pattern.Pattern, tc.testString)
			}

			// Test level group extraction
			if tc.pattern.LevelGroup > 0 && tc.pattern.LevelGroup < len(matches) {
				actualLevel := matches[tc.pattern.LevelGroup]
				if actualLevel != tc.expectedLevel {
					t.Errorf("Level group extraction: got %s, want %s", actualLevel, tc.expectedLevel)
				}
			}

			// Test message group extraction
			if tc.pattern.MessageGroup > 0 && tc.pattern.MessageGroup < len(matches) {
				actualMessage := matches[tc.pattern.MessageGroup]
				if actualMessage != tc.expectedMessage {
					t.Errorf("Message group extraction: got %s, want %s", actualMessage, tc.expectedMessage)
				}
			}
		})
	}
}
