package workflow

import (
	"regexp"
	"strings"
	"testing"
)

// TestErrorPatternsNoInfiniteLoopPotential tests that error patterns cannot cause infinite loops
// in JavaScript when used with the global flag. This is critical because patterns that match
// zero-width can cause regex.exec() to never advance lastIndex, resulting in an infinite loop.
func TestErrorPatternsNoInfiniteLoopPotential(t *testing.T) {
	engines := []CodingAgentEngine{
		NewCodexEngine(),
		NewClaudeEngine(),
		NewCopilotEngine(),
	}

	// Test strings that could trigger problematic behavior
	problematicStrings := []string{
		"",                        // Empty string - critical test case
		"a",                       // Single character
		"error",                   // Common word
		"error error error",       // Repeated words
		strings.Repeat("x", 1000), // Long string
	}

	for _, engine := range engines {
		t.Run(engine.GetID()+"_no_infinite_loop_patterns", func(t *testing.T) {
			patterns := engine.GetErrorPatterns()
			if len(patterns) == 0 {
				t.Skipf("Engine %s has no error patterns", engine.GetID())
			}

			for i, pattern := range patterns {
				t.Run(pattern.Description, func(t *testing.T) {
					// Convert to JavaScript-compatible pattern
					jsPattern := pattern.Pattern
					if strings.HasPrefix(pattern.Pattern, "(?i)") {
						jsPattern = pattern.Pattern[4:]
					}

					// Compile pattern
					regex, err := regexp.Compile(jsPattern)
					if err != nil {
						t.Fatalf("Pattern %d failed to compile: %v", i, err)
					}

					// Test each problematic string
					for _, testStr := range problematicStrings {
						// Simulate JavaScript's global regex behavior
						matches := regex.FindAllString(testStr, -1)

						// Check for zero-width matches
						for _, match := range matches {
							if match == "" {
								t.Errorf("Pattern can match empty string (zero-width match):\n"+
									"  Pattern: %s\n"+
									"  Test string: %q\n"+
									"  This can cause infinite loops in JavaScript with global flag!\n"+
									"  Pattern must match at least one character.",
									pattern.Pattern, testStr)
							}
						}

						// Additional check: test if pattern structure could match zero-width
						if hasZeroWidthMatchPotential(jsPattern) {
							// Verify it doesn't actually match zero-width
							if regex.MatchString("") {
								t.Errorf("Pattern matches empty string:\n"+
									"  Pattern: %s\n"+
									"  This WILL cause infinite loops in JavaScript!",
									pattern.Pattern)
							}
						}
					}
				})
			}
		})
	}
}

// hasZeroWidthMatchPotential checks if a pattern has structural elements that could
// potentially match zero-width (like *, ?, {0,n}, etc.)
func hasZeroWidthMatchPotential(pattern string) bool {
	// Check for patterns that are just .* or .*something.* without anchors or required content
	if pattern == ".*" || pattern == ".*?" {
		return true
	}

	// Check for patterns that start with .* without any required prefix
	if strings.HasPrefix(pattern, ".*") && !strings.Contains(pattern[2:], ".+") {
		return true
	}

	// Check for patterns with only optional quantifiers
	hasRequired := false
	for i := 0; i < len(pattern); i++ {
		// Look for required characters (not followed by *, ?, or {0,...)
		if i < len(pattern)-1 {
			next := pattern[i+1]
			if next != '*' && next != '?' && next != '{' {
				if pattern[i] != '(' && pattern[i] != ')' && pattern[i] != '|' && pattern[i] != '[' && pattern[i] != ']' {
					hasRequired = true
					break
				}
			}
		}
	}

	return !hasRequired
}

// TestSpecificProblematicPatterns tests known problematic pattern structures
func TestSpecificProblematicPatterns(t *testing.T) {
	problematicPatterns := []struct {
		pattern     string
		description string
		shouldFail  bool
	}{
		{
			pattern:     ".*",
			description: "Pure .* matches zero-width",
			shouldFail:  true,
		},
		{
			pattern:     "a*",
			description: "Single char with * matches zero-width",
			shouldFail:  true,
		},
		{
			pattern:     ".*error.*",
			description: ".* surrounding a word can match zero-width at ends",
			shouldFail:  false, // This actually doesn't match empty because 'error' is required
		},
		{
			pattern:     "error.*permission.*denied",
			description: "Required prefix with .* should be safe",
			shouldFail:  false,
		},
		{
			pattern:     "(?:error).*(?:permission).*(?:denied)",
			description: "Non-capturing groups with required text should be safe",
			shouldFail:  false,
		},
	}

	for _, tc := range problematicPatterns {
		t.Run(tc.description, func(t *testing.T) {
			regex, err := regexp.Compile(tc.pattern)
			if err != nil {
				t.Fatalf("Pattern failed to compile: %v", err)
			}

			// Test if it matches empty string
			matchesEmpty := regex.MatchString("")

			if tc.shouldFail && !matchesEmpty {
				t.Logf("Expected pattern to match empty string but it doesn't: %s", tc.pattern)
			} else if !tc.shouldFail && matchesEmpty {
				t.Errorf("Pattern matches empty string (could cause infinite loop):\n"+
					"  Pattern: %s\n"+
					"  Description: %s",
					tc.pattern, tc.description)
			}
		})
	}
}

// TestJavaScriptGlobalFlagBehavior documents and tests the JavaScript global flag behavior
// that can cause infinite loops with certain patterns
func TestJavaScriptGlobalFlagBehavior(t *testing.T) {
	t.Run("document_zero_width_match_issue", func(t *testing.T) {
		// This test documents the issue that validate_errors.cjs needs to handle

		// Patterns that match zero-width at end of string
		zeroWidthPatterns := []string{
			".*",     // Matches everything including empty at end
			"a*",     // Matches zero or more 'a's including empty
			"(x|y)*", // Matches zero or more x or y
		}

		for _, pattern := range zeroWidthPatterns {
			regex, err := regexp.Compile(pattern)
			if err != nil {
				t.Fatalf("Pattern failed to compile: %v", err)
			}

			// These patterns will match at the end of a non-empty string
			testStr := "hello"
			matches := regex.FindAllString(testStr, -1)

			// Check if any match is zero-width
			hasZeroWidth := false
			for _, match := range matches {
				if match == "" {
					hasZeroWidth = true
					break
				}
			}

			if hasZeroWidth {
				t.Logf("Pattern %q has zero-width match potential (validate_errors.cjs must handle this)",
					pattern)
			}
		}
	})
}

// TestAllEnginePatternsSafe is the primary safety test - verifies that NO engine patterns
// can match empty strings, which would cause infinite loops in JavaScript
func TestAllEnginePatternsSafe(t *testing.T) {
	engines := []CodingAgentEngine{
		NewCodexEngine(),
		NewClaudeEngine(),
		NewCopilotEngine(),
	}

	var unsafePatterns []string

	for _, engine := range engines {
		patterns := engine.GetErrorPatterns()
		for _, pattern := range patterns {
			// Convert to JavaScript-compatible pattern
			jsPattern := pattern.Pattern
			if strings.HasPrefix(pattern.Pattern, "(?i)") {
				jsPattern = pattern.Pattern[4:]
			}

			regex, err := regexp.Compile(jsPattern)
			if err != nil {
				continue // Skip invalid patterns (caught by other tests)
			}

			// Critical test: pattern must NOT match empty string
			if regex.MatchString("") {
				unsafePatterns = append(unsafePatterns,
					engine.GetID()+": "+pattern.Description+" - "+pattern.Pattern)
			}
		}
	}

	if len(unsafePatterns) > 0 {
		t.Errorf("Found %d patterns that match empty string (WILL cause infinite loops in JavaScript):\n%s\n\n"+
			"These patterns MUST be fixed by:\n"+
			"  1. Adding a required prefix (e.g., 'error' before .*)\n"+
			"  2. Using .+ instead of .* where appropriate\n"+
			"  3. Ensuring the pattern requires at least one character",
			len(unsafePatterns),
			strings.Join(unsafePatterns, "\n"))
	}
}
