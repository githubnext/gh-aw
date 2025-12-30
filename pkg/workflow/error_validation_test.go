package workflow

import (
	"regexp"
	"strings"
	"testing"
)

func TestErrorPatternStruct(t *testing.T) {
	tests := []struct {
		name        string
		pattern     ErrorPattern
		expectValid bool
	}{
		{
			name: "valid error pattern with level and message groups",
			pattern: ErrorPattern{
				Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+(ERROR):\s+(.+)`,
				LevelGroup:   2,
				MessageGroup: 3,
				Description:  "Test error pattern",
			},
			expectValid: true,
		},
		{
			name: "valid pattern with zero groups (defaults)",
			pattern: ErrorPattern{
				Pattern:     `ERROR: .+`,
				Description: "Simple error pattern",
			},
			expectValid: true,
		},
		{
			name: "empty pattern",
			pattern: ErrorPattern{
				Pattern:     "",
				Description: "Empty pattern",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pattern.Pattern == "" && tt.expectValid {
				t.Errorf("Expected valid pattern but got empty pattern")
			}
			if tt.pattern.Pattern != "" && !tt.expectValid {
				// This test case is for checking empty patterns, which we consider invalid
				return
			}
			// Additional validation could be added here
		})
	}
}

func TestCodingAgentEngineErrorValidation(t *testing.T) {
	// Test BaseEngine default behavior
	t.Run("BaseEngine_defaults", func(t *testing.T) {
		base := BaseEngine{}

		patterns := base.GetErrorPatterns()
		if len(patterns) != 0 {
			t.Errorf("BaseEngine should return empty error patterns, got %d", len(patterns))
		}
	})

	// Test CodexEngine error validation support
	t.Run("CodexEngine_error_validation", func(t *testing.T) {
		engine := NewCodexEngine()

		patterns := engine.GetErrorPatterns()
		if len(patterns) == 0 {
			t.Error("CodexEngine should return error patterns")
		}

		// Verify patterns have expected content
		foundError := false
		foundWarning := false

		for _, pattern := range patterns {
			if pattern.Description == "Codex ERROR messages with timestamp" {
				foundError = true
			}
			if pattern.Description == "Codex warning messages with timestamp" {
				foundWarning = true
			}
		}

		if !foundError {
			t.Error("Missing ERROR pattern")
		}
		if !foundWarning {
			t.Error("Missing warning pattern")
		}
	})

	// Test ClaudeEngine error validation support (uses common patterns only)
	t.Run("ClaudeEngine_error_validation", func(t *testing.T) {
		engine := NewClaudeEngine()

		patterns := engine.GetErrorPatterns()
		if len(patterns) == 0 {
			t.Error("ClaudeEngine should return common error patterns")
		}

		// Verify common patterns are present (inherited from GetCommonErrorPatterns)
		foundGitHubError := false
		foundGenericError := false

		for _, pattern := range patterns {
			if strings.Contains(pattern.Description, "GitHub Actions workflow command") {
				foundGitHubError = true
			}
			if strings.Contains(pattern.Description, "Generic ERROR") {
				foundGenericError = true
			}
		}

		if !foundGitHubError {
			t.Error("Missing GitHub Actions workflow command pattern")
		}
		if !foundGenericError {
			t.Error("Missing generic ERROR pattern")
		}
	})

	// Test CopilotEngine detects common patterns (command not found is detected by generic ERROR pattern)
	t.Run("CopilotEngine_detects_command_not_found", func(t *testing.T) {
		engine := NewCopilotEngine()
		patterns := engine.GetErrorPatterns()

		// Test logs with vitest errors - these should now be FILTERED as benign
		testLogs := []string{
			"Error: Cannot find module 'vitest'",
			"ERROR: vitest command not found",
		}

		for _, logLine := range testLogs {
			errors := CountErrorsAndWarningsWithPatterns(logLine, patterns)
			errorCount := CountErrors(errors)
			// Vitest errors should be filtered out as benign
			if errorCount != 0 {
				t.Errorf("Vitest error should be filtered as benign, but was detected in log line: %q", logLine)
			}
		}

		// Test that non-vitest errors are still detected
		actualErrors := []string{
			"Error: Cannot find module 'express'",
			"ERROR: npm command not found",
		}

		for _, logLine := range actualErrors {
			errors := CountErrorsAndWarningsWithPatterns(logLine, patterns)
			errorCount := CountErrors(errors)
			if errorCount == 0 {
				t.Errorf("Failed to detect actual error in log line: %q", logLine)
			}
		}
	})

	// Test CopilotEngine error validation support (timestamp patterns only)
	t.Run("CopilotEngine_error_validation", func(t *testing.T) {
		engine := NewCopilotEngine()

		patterns := engine.GetErrorPatterns()
		if len(patterns) == 0 {
			t.Error("CopilotEngine should return error patterns")
		}

		// Verify patterns have expected content
		foundTimestampedError := false
		foundTimestampedWarning := false
		foundBracketedError := false

		for _, pattern := range patterns {
			switch pattern.Description {
			case "Copilot CLI timestamped ERROR messages":
				foundTimestampedError = true
				if pattern.LevelGroup != 2 || pattern.MessageGroup != 3 {
					t.Errorf("Copilot timestamped error pattern has wrong groups: level=%d, message=%d", pattern.LevelGroup, pattern.MessageGroup)
				}
			case "Copilot CLI timestamped WARNING messages":
				foundTimestampedWarning = true
				if pattern.LevelGroup != 2 || pattern.MessageGroup != 3 {
					t.Errorf("Copilot timestamped warning pattern has wrong groups: level=%d, message=%d", pattern.LevelGroup, pattern.MessageGroup)
				}
			case "Copilot CLI bracketed critical/error messages with timestamp":
				foundBracketedError = true
				if pattern.LevelGroup != 2 || pattern.MessageGroup != 3 {
					t.Errorf("Copilot bracketed error pattern has wrong groups: level=%d, message=%d", pattern.LevelGroup, pattern.MessageGroup)
				}
			}
		}

		if !foundTimestampedError {
			t.Error("CopilotEngine should have timestamped error pattern")
		}
		if !foundTimestampedWarning {
			t.Error("CopilotEngine should have timestamped warning pattern")
		}
		if !foundBracketedError {
			t.Error("CopilotEngine should have bracketed error pattern")
		}
	})

	// Test new error patterns with real-world examples (using generic ERROR pattern)
	t.Run("CopilotEngine_detects_new_error_types", func(t *testing.T) {
		engine := NewCopilotEngine()
		patterns := engine.GetErrorPatterns()

		// Test logs with ERROR prefix - should be detected by common pattern
		testLogs := []string{
			"ERROR: API rate limit exceeded, please try again later",
			"Error: Too many requests",
			"Error: received 429 status code",
			"ERROR: quota exceeded for API calls",
			"Error: Request timeout after 30 seconds",
			"ERROR: Operation timed out",
			"Error: deadline exceeded",
			"ERROR: Connection refused: ECONNREFUSED",
			"Error: connection failed to api.github.com",
			"ERROR: Network error: ETIMEDOUT",
			"Error: DNS resolution failed: ENOTFOUND",
			"ERROR: token expired, please refresh your credentials",
			"Error: Fatal error: maximum call stack size exceeded",
			"ERROR: heap out of memory",
			"Error: spawn ENOMEM: not enough memory",
		}

		for _, logLine := range testLogs {
			counts := CountErrorsAndWarningsWithPatterns(logLine, patterns)
			if CountErrors(counts) == 0 {
				t.Errorf("Failed to detect error in log line: %q", logLine)
			}
		}
	})

	// Test that patterns don't match informational text without ERROR context
	t.Run("CopilotEngine_does_not_match_informational_quota_and_timeout_text", func(t *testing.T) {
		engine := NewCopilotEngine()
		patterns := engine.GetErrorPatterns()

		// These should NOT match because they lack ERROR: or Error: prefix
		informationalText := []string{
			"quota will be exceeded tomorrow",
			"avoid timeout issues by increasing the limit",
			"timeout configuration is set to 30 seconds",
			"the deadline is next week",
			"connection to the database was successful",
			"rate limit is 5000 requests per hour",
		}

		for _, text := range informationalText {
			counts := CountErrorsAndWarningsWithPatterns(text, patterns)
			if CountErrors(counts) > 0 {
				t.Errorf("Pattern incorrectly matched informational text: %q", text)
			}
		}
	})
}

func TestErrorPatternSerialization(t *testing.T) {
	pattern := ErrorPattern{
		Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+(ERROR):\s+(.+)`,
		LevelGroup:   2,
		MessageGroup: 3,
	}

	// Test JSON serialization - this would be used in the workflow compiler
	// We just verify the struct can be used with json operations
	if pattern.Pattern == "" {
		t.Error("Pattern should not be empty")
	}

	if pattern.LevelGroup < 1 {
		t.Error("LevelGroup should be >= 1")
	}

	if pattern.MessageGroup < 1 {
		t.Error("MessageGroup should be >= 1")
	}
}

func TestCodexEngine401UnauthorizedDetection(t *testing.T) {
	// Test case for GitHub issue #668: Codex fails to report failure if unauthorised
	// Updated to test new Rust format
	engine := NewCodexEngine()
	patterns := engine.GetErrorPatterns()

	// Log content in new Rust format (converted from issue #668)
	logContent := `2025-09-10T17:55:15.123Z ERROR exceeded retry limit, last status: 401 Unauthorized`

	// Test that patterns can detect the errors
	foundErrorMessages := 0

	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			t.Errorf("Invalid regex pattern '%s': %v", pattern.Pattern, err)
			continue
		}

		matches := regex.FindAllString(logContent, -1)
		if len(matches) > 0 {
			if pattern.Description == "Codex ERROR messages with timestamp" {
				foundErrorMessages = len(matches)
			}
		}
	}

	// Should detect 1 ERROR message from issue #668
	if foundErrorMessages != 1 {
		t.Errorf("Expected 1 ERROR message from issue #668, found %d", foundErrorMessages)
	}

	// Verify the patterns specifically match 401 unauthorized content
	// Find the Codex ERROR pattern by description
	var errorPattern ErrorPattern
	found := false
	for _, pattern := range patterns {
		if pattern.Description == "Codex ERROR messages with timestamp" {
			errorPattern = pattern
			found = true
			break
		}
	}

	if !found {
		t.Fatal("Could not find 'Codex ERROR messages with timestamp' pattern")
	}

	regex, _ := regexp.Compile(errorPattern.Pattern)
	match := regex.FindStringSubmatch("2025-09-10T17:55:15.123Z ERROR exceeded retry limit, last status: 401 Unauthorized")

	if len(match) < 4 {
		t.Error("ERROR pattern should capture timestamp, level, and message groups")
	} else {
		if match[errorPattern.LevelGroup] != "ERROR" {
			t.Errorf("Expected level 'ERROR', got '%s'", match[errorPattern.LevelGroup])
		}
		if !strings.Contains(match[errorPattern.MessageGroup], "401 Unauthorized") {
			t.Errorf("Expected message to contain '401 Unauthorized', got '%s'", match[errorPattern.MessageGroup])
		}
	}
}

func TestExtractErrorPatternsFromEngineConfig(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    []ErrorPattern
	}{
		{
			name: "no error_patterns field in engine",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
				},
			},
			expected: []ErrorPattern{},
		},
		{
			name: "valid error patterns in engine config",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
					"error_patterns": []any{
						map[string]any{
							"pattern":       `ERROR:\s+(.+)`,
							"level_group":   0,
							"message_group": 1,
							"description":   "Simple error pattern",
						},
						map[string]any{
							"pattern":       `\[(\d{4}-\d{2}-\d{2})\]\s+(WARN):\s+(.+)`,
							"level_group":   2,
							"message_group": 3,
							"description":   "Warning pattern with timestamp",
						},
					},
				},
			},
			expected: []ErrorPattern{
				{
					Pattern:      `ERROR:\s+(.+)`,
					LevelGroup:   0,
					MessageGroup: 1,
					Description:  "Simple error pattern",
				},
				{
					Pattern:      `\[(\d{4}-\d{2}-\d{2})\]\s+(WARN):\s+(.+)`,
					LevelGroup:   2,
					MessageGroup: 3,
					Description:  "Warning pattern with timestamp",
				},
			},
		},
		{
			name: "pattern with float64 groups (from YAML parsing)",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
					"error_patterns": []any{
						map[string]any{
							"pattern":       `ERROR:\s+(.+)`,
							"level_group":   float64(0),
							"message_group": float64(1),
							"description":   "Float64 group indices",
						},
					},
				},
			},
			expected: []ErrorPattern{
				{
					Pattern:      `ERROR:\s+(.+)`,
					LevelGroup:   0,
					MessageGroup: 1,
					Description:  "Float64 group indices",
				},
			},
		},
		{
			name: "pattern without optional fields",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
					"error_patterns": []any{
						map[string]any{
							"pattern": `CRITICAL.*`,
						},
					},
				},
			},
			expected: []ErrorPattern{
				{
					Pattern:      `CRITICAL.*`,
					LevelGroup:   0,
					MessageGroup: 0,
					Description:  "",
				},
			},
		},
		{
			name: "invalid patterns should be skipped",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
					"error_patterns": []any{
						map[string]any{
							// Missing required pattern field
							"level_group": 1,
							"description": "Invalid - no pattern",
						},
						map[string]any{
							"pattern":     `VALID:\s+(.+)`,
							"description": "Valid pattern",
						},
					},
				},
			},
			expected: []ErrorPattern{
				{
					Pattern:      `VALID:\s+(.+)`,
					LevelGroup:   0,
					MessageGroup: 0,
					Description:  "Valid pattern",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, engineConfig := compiler.ExtractEngineConfig(tt.frontmatter)

			var patterns []ErrorPattern
			if engineConfig != nil {
				patterns = engineConfig.ErrorPatterns
			}

			if len(patterns) != len(tt.expected) {
				t.Errorf("Expected %d patterns, got %d", len(tt.expected), len(patterns))
				return
			}

			for i, expected := range tt.expected {
				if i >= len(patterns) {
					t.Errorf("Missing pattern %d", i)
					continue
				}
				actual := patterns[i]

				if actual.Pattern != expected.Pattern {
					t.Errorf("Pattern %d: expected pattern '%s', got '%s'", i, expected.Pattern, actual.Pattern)
				}
				if actual.LevelGroup != expected.LevelGroup {
					t.Errorf("Pattern %d: expected level_group %d, got %d", i, expected.LevelGroup, actual.LevelGroup)
				}
				if actual.MessageGroup != expected.MessageGroup {
					t.Errorf("Pattern %d: expected message_group %d, got %d", i, expected.MessageGroup, actual.MessageGroup)
				}
				if actual.Description != expected.Description {
					t.Errorf("Pattern %d: expected description '%s', got '%s'", i, expected.Description, actual.Description)
				}
			}
		})
	}
}

func TestGenerateErrorValidationWithEngineConfigPatterns(t *testing.T) {
	compiler := NewCompiler(false, "", "")
	engine := NewClaudeEngine() // Claude doesn't support error validation by default

	// Test with engine config defined error patterns
	data := &WorkflowData{
		EngineConfig: &EngineConfig{
			ID: "claude",
			ErrorPatterns: []ErrorPattern{
				{
					Pattern:      `ERROR:\s+(.+)`,
					LevelGroup:   0,
					MessageGroup: 1,
					Description:  "Custom error pattern from engine config",
				},
			},
		},
	}

	var yamlBuilder strings.Builder
	compiler.generateErrorValidation(&yamlBuilder, engine, data)

	generated := yamlBuilder.String()

	// Should generate error validation step even though Claude doesn't support it natively
	if !strings.Contains(generated, "Validate agent logs for errors") {
		t.Error("Should generate error validation step with frontmatter patterns")
	}

	if !strings.Contains(generated, "GH_AW_ERROR_PATTERNS") {
		t.Error("Should include error patterns environment variable")
	}

	// Should contain the custom pattern
	if !strings.Contains(generated, "Custom error pattern from engine config") {
		t.Error("Should include custom pattern description in JSON")
	}

	// Test with empty engine config patterns but engine that supports validation
	codexEngine := NewCodexEngine()
	dataEmpty := &WorkflowData{
		EngineConfig: &EngineConfig{
			ID:            "codex",
			ErrorPatterns: []ErrorPattern{},
		},
	}

	var yamlBuilder2 strings.Builder
	compiler.generateErrorValidation(&yamlBuilder2, codexEngine, dataEmpty)

	generated2 := yamlBuilder2.String()

	// Should fall back to engine patterns
	if !strings.Contains(generated2, "Validate agent logs for errors") {
		t.Error("Should generate error validation step with engine patterns")
	}

	// Test with no engine config but engine that has built-in error patterns (like Claude)
	dataEmpty2 := &WorkflowData{
		EngineConfig: nil,
	}

	var yamlBuilder3 strings.Builder
	compiler.generateErrorValidation(&yamlBuilder3, engine, dataEmpty2)

	generated3 := yamlBuilder3.String()

	// Should generate validation step with engine's built-in patterns since Claude now supports error validation
	if !strings.Contains(generated3, "Validate agent logs for errors") {
		t.Error("Should generate error validation step with engine's built-in patterns")
	}
}
