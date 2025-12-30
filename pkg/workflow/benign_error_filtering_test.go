package workflow

import (
	"regexp"
	"testing"
)

// TestBenignErrorPatternsDoNotMatchInCounts tests that benign errors are filtered out
func TestBenignErrorPatternsDoNotMatchInCounts(t *testing.T) {
	engine := NewCopilotEngine()
	patterns := engine.GetErrorPatterns()

	// Test log with benign errors (vitest not found)
	testLog := `Running make test
sh: 1: vitest: not found
make: *** [Makefile:100: test-js] Error 127
make: *** Waiting for unfinished jobs....

Error: Cannot find module 'vitest'
Require stack:
- /home/runner/work/gh-aw/gh-aw/pkg/workflow/js/test.js

There's a minor issue with the JavaScript tests failing because vitest is not found, but this is
not something we introduced - it's a pre-existing issue in the environment.
`

	errors := CountErrorsAndWarningsWithPatterns(testLog, patterns)

	errorCount := CountErrors(errors)
	t.Logf("Detected %d errors after filtering benign patterns", errorCount)

	// Log all detected errors for debugging
	for i, err := range errors {
		t.Logf("  %d. [%s] Line %d: %s (pattern: %s)", i+1, err.Type, err.Line, err.Message, err.PatternID)
	}

	// Benign vitest errors should be filtered out
	// We should not detect vitest-related errors
	for _, err := range errors {
		if containsVitest(err.Message) {
			t.Errorf("Vitest-related error was not filtered out: %s (pattern: %s)", err.Message, err.PatternID)
		}
	}

	// Check for pre-existing issue patterns
	vitestNotFoundCount := 0
	preExistingMentionCount := 0
	for i := 0; i < len(testLog); i++ {
		line := testLog[i:minInt(i+50, len(testLog))]
		if containsVitest(line) {
			vitestNotFoundCount++
		}
		if containsSubstr(line, "pre-existing") || containsSubstr(line, "not something we introduced") {
			preExistingMentionCount++
		}
	}

	t.Logf("Log contains %d vitest mentions and %d pre-existing issue mentions", vitestNotFoundCount, preExistingMentionCount)
}

// TestBenignErrorPatternsCompile tests that all benign error patterns compile correctly
func TestBenignErrorPatternsCompile(t *testing.T) {
	patterns := GetBenignErrorPatterns()

	if len(patterns) == 0 {
		t.Fatal("No benign error patterns defined")
	}

	for i, pattern := range patterns {
		// Verify pattern has IsBenignError flag set
		if !pattern.IsBenignError {
			t.Errorf("Pattern %d (%s) should have IsBenignError=true", i, pattern.Description)
		}

		// Verify pattern compiles
		if _, err := compileErrorPattern(pattern.Pattern); err != nil {
			t.Errorf("Pattern %d (%s) failed to compile: %v\nPattern: %s",
				i, pattern.Description, err, pattern.Pattern)
		}

		// Verify pattern has description
		if pattern.Description == "" {
			t.Errorf("Pattern %d has no description", i)
		}

		// Verify pattern has ID
		if pattern.ID == "" {
			t.Errorf("Pattern %d (%s) has no ID", i, pattern.Description)
		}
	}

	t.Logf("All %d benign error patterns compiled successfully", len(patterns))
}

// TestBenignErrorPatternsMatchExpectedErrors tests that benign patterns match their intended errors
func TestBenignErrorPatternsMatchExpectedErrors(t *testing.T) {
	testCases := []struct {
		name           string
		logLine        string
		shouldBeBenign bool
	}{
		{
			name:           "vitest_command_not_found",
			logLine:        "sh: 1: vitest: not found",
			shouldBeBenign: true,
		},
		{
			name:           "vitest_module_not_found",
			logLine:        "Error: Cannot find module 'vitest'",
			shouldBeBenign: true,
		},
		{
			name:           "make_test_js_error_127",
			logLine:        "make: *** [Makefile:100: test-js] Error 127",
			shouldBeBenign: false, // This is a generic make error, not vitest-specific in pattern
		},
		{
			name:           "dev_dependencies_not_installed",
			logLine:        "ERROR: development dependencies not installed",
			shouldBeBenign: true,
		},
		{
			name:           "pre_existing_issue_mentioned",
			logLine:        "This is a pre-existing issue in the environment",
			shouldBeBenign: true,
		},
		{
			name:           "not_introduced_by_changes",
			logLine:        "This error is not something we introduced",
			shouldBeBenign: true,
		},
		{
			name:           "actual_error_should_not_be_benign",
			logLine:        "ERROR: Failed to compile workflow",
			shouldBeBenign: false,
		},
	}

	benignPatterns := GetBenignErrorPatterns()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			for _, pattern := range benignPatterns {
				if matchesErrorPattern(pattern.Pattern, tc.logLine) {
					matched = true
					break
				}
			}

			if tc.shouldBeBenign && !matched {
				t.Errorf("Expected benign error pattern to match: %s", tc.logLine)
			} else if !tc.shouldBeBenign && matched {
				t.Errorf("Benign pattern should NOT match: %s", tc.logLine)
			}
		})
	}
}

// TestEnginesIncludeBenignPatterns tests that all engines include benign error patterns
func TestEnginesIncludeBenignPatterns(t *testing.T) {
	engines := []CodingAgentEngine{
		NewCopilotEngine(),
		NewClaudeEngine(),
		NewCodexEngine(),
	}

	benignPatternIDs := make(map[string]bool)
	for _, p := range GetBenignErrorPatterns() {
		benignPatternIDs[p.ID] = true
	}

	for _, engine := range engines {
		t.Run(engine.GetID()+"_has_benign_patterns", func(t *testing.T) {
			patterns := engine.GetErrorPatterns()

			foundBenignCount := 0
			for _, pattern := range patterns {
				if pattern.IsBenignError {
					foundBenignCount++
				}
			}

			if foundBenignCount == 0 {
				t.Errorf("Engine %s has no benign error patterns - all engines should include GetBenignErrorPatterns()",
					engine.GetID())
			} else {
				t.Logf("Engine %s has %d benign error patterns", engine.GetID(), foundBenignCount)
			}

			// Verify that at least some common benign patterns are present
			foundCommonBenign := 0
			for _, pattern := range patterns {
				if benignPatternIDs[pattern.ID] {
					foundCommonBenign++
				}
			}

			if foundCommonBenign == 0 {
				t.Errorf("Engine %s does not include common benign patterns from GetBenignErrorPatterns()",
					engine.GetID())
			}
		})
	}
}

// Helper functions

func containsVitest(s string) bool {
	return containsSubstr(s, "vitest")
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func compileErrorPattern(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile(pattern)
}

func matchesErrorPattern(pattern, text string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(text)
}
