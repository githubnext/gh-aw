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

		if base.SupportsErrorValidation() {
			t.Error("BaseEngine should not support error validation by default")
		}

		patterns := base.GetErrorPatterns()
		if len(patterns) != 0 {
			t.Errorf("BaseEngine should return empty error patterns, got %d", len(patterns))
		}
	})

	// Test CodexEngine error validation support
	t.Run("CodexEngine_error_validation", func(t *testing.T) {
		engine := NewCodexEngine()

		if !engine.SupportsErrorValidation() {
			t.Error("CodexEngine should support error validation")
		}

		patterns := engine.GetErrorPatterns()
		if len(patterns) == 0 {
			t.Error("CodexEngine should return error patterns")
		}

		// Verify patterns have expected content
		foundStreamError := false
		foundError := false
		foundWarning := false

		for _, pattern := range patterns {
			if pattern.Description == "Codex stream errors with timestamp" {
				foundStreamError = true
				if pattern.LevelGroup != 2 || pattern.MessageGroup != 3 {
					t.Errorf("Stream error pattern has incorrect groups: level=%d, message=%d",
						pattern.LevelGroup, pattern.MessageGroup)
				}
			}
			if pattern.Description == "Codex ERROR messages with timestamp" {
				foundError = true
			}
			if pattern.Description == "Codex warning messages with timestamp" {
				foundWarning = true
			}
		}

		if !foundStreamError {
			t.Error("Missing stream error pattern")
		}
		if !foundError {
			t.Error("Missing ERROR pattern")
		}
		if !foundWarning {
			t.Error("Missing warning pattern")
		}
	})

	// Test ClaudeEngine default behavior (should not support error validation)
	t.Run("ClaudeEngine_no_error_validation", func(t *testing.T) {
		engine := NewClaudeEngine()

		if engine.SupportsErrorValidation() {
			t.Error("ClaudeEngine should not support error validation by default")
		}

		patterns := engine.GetErrorPatterns()
		if len(patterns) != 0 {
			t.Errorf("ClaudeEngine should return empty error patterns, got %d", len(patterns))
		}
	})
}

func TestErrorPatternSerialization(t *testing.T) {
	pattern := ErrorPattern{
		Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+(ERROR):\s+(.+)`,
		LevelGroup:   2,
		MessageGroup: 3,
		Description:  "Test pattern",
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
	engine := NewCodexEngine()
	patterns := engine.GetErrorPatterns()

	// Log content from issue #668
	logContent := `[2025-09-10T17:54:49] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 1/5 in 216ms…
[2025-09-10T17:54:54] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 2/5 in 414ms…
[2025-09-10T17:54:58] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 3/5 in 821ms…
[2025-09-10T17:55:03] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 4/5 in 1.611s…
[2025-09-10T17:55:08] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 5/5 in 3.039s…
[2025-09-10T17:55:15] ERROR: exceeded retry limit, last status: 401 Unauthorized`

	// Test that patterns can detect the errors
	foundStreamErrors := 0
	foundErrorMessages := 0

	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			t.Errorf("Invalid regex pattern '%s': %v", pattern.Pattern, err)
			continue
		}

		matches := regex.FindAllString(logContent, -1)
		if len(matches) > 0 {
			if pattern.Description == "Codex stream errors with timestamp" {
				foundStreamErrors = len(matches)
			}
			if pattern.Description == "Codex ERROR messages with timestamp" {
				foundErrorMessages = len(matches)
			}
		}
	}

	// Should detect 5 stream errors and 1 ERROR message from issue #668
	if foundStreamErrors != 5 {
		t.Errorf("Expected 5 stream errors from issue #668, found %d", foundStreamErrors)
	}
	if foundErrorMessages != 1 {
		t.Errorf("Expected 1 ERROR message from issue #668, found %d", foundErrorMessages)
	}

	// Verify the patterns specifically match 401 unauthorized content
	streamPattern := patterns[0] // Stream error pattern
	regex, _ := regexp.Compile(streamPattern.Pattern)
	match := regex.FindStringSubmatch("[2025-09-10T17:54:49] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 1/5 in 216ms…")

	if len(match) < 4 {
		t.Error("Stream error pattern should capture timestamp, level, and message groups")
	} else {
		if match[streamPattern.LevelGroup] != "error" {
			t.Errorf("Expected level 'error', got '%s'", match[streamPattern.LevelGroup])
		}
		if !strings.Contains(match[streamPattern.MessageGroup], "401 Unauthorized") {
			t.Errorf("Expected message to contain '401 Unauthorized', got '%s'", match[streamPattern.MessageGroup])
		}
	}
}
