package workflow

import (
	"testing"
)

func TestCountErrorsAndWarningsWithPatterns(t *testing.T) {
	tests := []struct {
		name           string
		logContent     string
		patterns       []ErrorPattern
		expectedErrors int
		expectedWarns  int
	}{
		{
			name:           "empty patterns",
			logContent:     "some log content with error and warning",
			patterns:       []ErrorPattern{},
			expectedErrors: 0,
			expectedWarns:  0,
		},
		{
			name: "codex style patterns",
			logContent: `[2024-01-01T10:00:00] stream error: authentication failed
[2024-01-01T10:00:01] ERROR: processing failed
[2024-01-01T10:00:02] WARN: deprecated feature used
[2024-01-01T10:00:03] WARNING: memory usage high`,
			patterns: []ErrorPattern{
				{
					Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+stream\s+(error):\s+(.+)`,
					LevelGroup:   2,
					MessageGroup: 3,
					Description:  "Codex stream errors",
				},
				{
					Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+(ERROR):\s+(.+)`,
					LevelGroup:   2,
					MessageGroup: 3,
					Description:  "Codex ERROR messages",
				},
				{
					Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+(WARN|WARNING):\s+(.+)`,
					LevelGroup:   2,
					MessageGroup: 3,
					Description:  "Codex warning messages",
				},
			},
			expectedErrors: 2,
			expectedWarns:  2,
		},
		{
			name: "copilot style patterns",
			logContent: `2024-01-01T10:00:00.123Z [ERROR] Connection failed
2024-01-01T10:00:01.456Z [WARN] Retrying connection
copilot: error: Invalid token
Warning: Feature deprecated`,
			patterns: []ErrorPattern{
				{
					Pattern:      `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[(WARN|WARNING)\]\s+(.+)`,
					LevelGroup:   2,
					MessageGroup: 3,
					Description:  "Copilot timestamped WARNING messages",
				},
				{
					Pattern:      `copilot:\s+(error):\s+(.+)`,
					LevelGroup:   1,
					MessageGroup: 2,
					Description:  "Copilot CLI errors",
				},
				{
					Pattern:      `(Warning):\s+(.+)`,
					LevelGroup:   1,
					MessageGroup: 2,
					Description:  "Generic warnings",
				},
			},
			expectedErrors: 1,
			expectedWarns:  2,
		},
		{
			name: "pattern with explicit level groups",
			logContent: `npm ERR! Authentication failed
npm WARN deprecated package
Some random error occurred
Another warning message`,
			patterns: []ErrorPattern{
				{
					Pattern:      `npm (ERR!)(.+)`,
					LevelGroup:   1, // Use capture group 1 which contains "ERR!"
					MessageGroup: 2,
					Description:  "NPM errors",
				},
				{
					Pattern:      `npm (WARN)(.+)`,
					LevelGroup:   1, // Use capture group 1 which contains "WARN"
					MessageGroup: 2,
					Description:  "NPM warnings",
				},
				{
					Pattern:      `(.*)error(.*)`,
					LevelGroup:   0, // Should infer "error" from content
					MessageGroup: 0, // Use full match
					Description:  "Generic error pattern",
				},
				{
					Pattern:      `(.*)warning(.*)`,
					LevelGroup:   0, // Should infer "warning" from content
					MessageGroup: 0, // Use full match
					Description:  "Generic warning pattern",
				},
			},
			expectedErrors: 2, // npm ERR! + random error
			expectedWarns:  2, // npm WARN + warning message
		},
		{
			name: "invalid regex pattern",
			logContent: `error: something went wrong
warning: be careful`,
			patterns: []ErrorPattern{
				{
					Pattern:      `[invalid regex`,
					LevelGroup:   1,
					MessageGroup: 2,
					Description:  "Invalid pattern - should be skipped",
				},
				{
					Pattern:      `(error):\s+(.+)`,
					LevelGroup:   1,
					MessageGroup: 2,
					Description:  "Valid error pattern",
				},
				{
					Pattern:      `(warning):\s+(.+)`,
					LevelGroup:   1,
					MessageGroup: 2,
					Description:  "Valid warning pattern",
				},
			},
			expectedErrors: 1,
			expectedWarns:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := CountErrorsAndWarningsWithPatterns(tt.logContent, tt.patterns)

			errorCount := CountErrors(errors)
			if errorCount != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d", tt.expectedErrors, errorCount)
			}

			warningCount := CountWarnings(errors)
			if warningCount != tt.expectedWarns {
				t.Errorf("Expected %d warnings, got %d", tt.expectedWarns, warningCount)
			}
		})
	}
}

func TestExtractLevelFromMatchCompiled(t *testing.T) {
	tests := []struct {
		name     string
		match    []string
		cp       compiledPattern
		expected string
	}{
		{
			name:     "valid level group",
			match:    []string{"[ERROR] message", "timestamp", "ERROR", "message"},
			cp:       compiledPattern{levelGroup: 2},
			expected: "error", // Level is normalized to lowercase
		},
		{
			name:     "level group out of bounds",
			match:    []string{"error message"},
			cp:       compiledPattern{levelGroup: 5},
			expected: "error", // Should infer from content
		},
		{
			name:     "no level group, infer error",
			match:    []string{"something with error in it"},
			cp:       compiledPattern{levelGroup: 0},
			expected: "error",
		},
		{
			name:     "no level group, infer warning",
			match:    []string{"warning: be careful"},
			cp:       compiledPattern{levelGroup: 0},
			expected: "warning",
		},
		{
			name:     "no level group, unknown",
			match:    []string{"debug: some info"},
			cp:       compiledPattern{levelGroup: 0},
			expected: "unknown",
		},
		{
			name:     "empty match",
			match:    []string{},
			cp:       compiledPattern{levelGroup: 1},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLevelFromMatchCompiled(tt.match, tt.cp)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
