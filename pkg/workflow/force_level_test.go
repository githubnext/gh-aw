package workflow

import (
	"testing"
)

// TestSeverityOverride tests that Severity field correctly overrides level inference
func TestSeverityOverride(t *testing.T) {
	tests := []struct {
		name           string
		logContent     string
		patterns       []ErrorPattern
		expectedErrors int
		expectedWarns  int
	}{
		{
			name:       "permission_denied_forced_to_warning",
			logContent: "ERROR: permission denied for this operation",
			patterns: []ErrorPattern{
				{
					Pattern:      `(?i)\berror\b.*permission.*denied`,
					LevelGroup:   0,
					MessageGroup: 0,
					Severity:     "warning",
					Description:  "Permission denied error forced to warning",
				},
			},
			expectedErrors: 0,
			expectedWarns:  1,
		},
		{
			name:       "authentication_failed_forced_to_warning",
			logContent: "authentication failed for user",
			patterns: []ErrorPattern{
				{
					Pattern:      `(?i)authentication failed`,
					LevelGroup:   0,
					MessageGroup: 0,
					Severity:     "warning",
					Description:  "Authentication failure forced to warning",
				},
			},
			expectedErrors: 0,
			expectedWarns:  1,
		},
		{
			name: "multiple_permission_patterns_as_warnings",
			logContent: `ERROR: permission denied for write
Error: unauthorized access attempt
configuration error: required permissions not specified
repository permission check failed`,
			patterns: []ErrorPattern{
				{
					Pattern:      `(?i)\berror\b.*permission.*denied`,
					LevelGroup:   0,
					MessageGroup: 0,
					Severity:     "warning",
					Description:  "Permission denied",
				},
				{
					Pattern:      `(?i)\berror\b.*unauthorized`,
					LevelGroup:   0,
					MessageGroup: 0,
					Severity:     "warning",
					Description:  "Unauthorized",
				},
				{
					Pattern:      `(?i)configuration error.*required permissions not specified`,
					LevelGroup:   0,
					MessageGroup: 0,
					Severity:     "warning",
					Description:  "Config error",
				},
				{
					Pattern:      `(?i)repository permission check failed`,
					LevelGroup:   0,
					MessageGroup: 0,
					Severity:     "warning",
					Description:  "Permission check failed",
				},
			},
			expectedErrors: 0,
			expectedWarns:  4,
		},
		{
			name:       "force_level_overrides_error_keyword",
			logContent: "ERROR: authentication failed",
			patterns: []ErrorPattern{
				{
					Pattern:      `(?i)authentication failed`,
					LevelGroup:   0,
					MessageGroup: 0,
					Severity:     "warning", // Force to warning despite "ERROR" in content
					Description:  "Auth failure",
				},
			},
			expectedErrors: 0,
			expectedWarns:  1,
		},
		{
			name:       "without_force_level_infers_error",
			logContent: "ERROR: permission denied",
			patterns: []ErrorPattern{
				{
					Pattern:      `(?i)\berror\b.*permission.*denied`,
					LevelGroup:   0,
					MessageGroup: 0,
					// No Severity - should infer as error
					Description: "Permission denied without force level",
				},
			},
			expectedErrors: 1,
			expectedWarns:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := CountErrorsAndWarningsWithPatterns(tt.logContent, tt.patterns)

			errorCount := CountErrors(errors)
			if errorCount != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d\nLog content: %s", tt.expectedErrors, errorCount, tt.logContent)
			}

			warningCount := CountWarnings(errors)
			if warningCount != tt.expectedWarns {
				t.Errorf("Expected %d warnings, got %d\nLog content: %s", tt.expectedWarns, warningCount, tt.logContent)
			}
		})
	}
}

// TestCodexEnginePermissionPatternsAreWarnings verifies that all permission-related patterns
// in the Codex engine are marked as warnings
func TestCodexEnginePermissionPatternsAreWarnings(t *testing.T) {
	engine := NewCodexEngine()
	patterns := engine.GetErrorPatterns()

	permissionKeywords := []string{"permission", "unauthorized", "forbidden", "access"}

	for _, pattern := range patterns {
		// Check if this is a permission-related pattern
		isPermissionPattern := false
		for _, keyword := range permissionKeywords {
			if containsIgnoreCase(pattern.Description, keyword) ||
				containsIgnoreCase(pattern.Pattern, keyword) {
				isPermissionPattern = true
				break
			}
		}

		// If it's a permission pattern (but not a timestamped ERROR/WARN), it should have Severity="warning"
		if isPermissionPattern && pattern.LevelGroup != 2 {
			if pattern.Severity != "warning" {
				t.Errorf("Permission-related pattern should have Severity='warning':\n  Pattern: %s\n  Description: %s\n  Severity: %s",
					pattern.Pattern, pattern.Description, pattern.Severity)
			}
		}
	}
}

// TestCopilotEnginePermissionPatternsAreWarnings verifies that all permission-related patterns
// in the Copilot engine are marked as warnings
func TestCopilotEnginePermissionPatternsAreWarnings(t *testing.T) {
	engine := NewCopilotEngine()
	patterns := engine.GetErrorPatterns()

	permissionKeywords := []string{"permission", "unauthorized", "forbidden", "access", "authentication", "token"}

	for _, pattern := range patterns {
		// Check if this is a permission-related pattern
		isPermissionPattern := false
		for _, keyword := range permissionKeywords {
			if containsIgnoreCase(pattern.Description, keyword) ||
				containsIgnoreCase(pattern.Pattern, keyword) {
				isPermissionPattern = true
				break
			}
		}

		// If it's a permission/auth pattern (but not a timestamped ERROR/WARN), it should have Severity="warning"
		// Exclude patterns with explicit level groups (those are properly categorized by their level group)
		if isPermissionPattern && pattern.LevelGroup == 0 &&
			!containsIgnoreCase(pattern.Pattern, `\[(ERROR|WARN|WARNING)\]`) &&
			!containsIgnoreCase(pattern.Pattern, `(ERROR|WARN|WARNING):`) {
			if pattern.Severity != "warning" {
				t.Errorf("Permission-related pattern should have Severity='warning':\n  Pattern: %s\n  Description: %s\n  Severity: %s",
					pattern.Pattern, pattern.Description, pattern.Severity)
			}
		}
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > 0 && len(substr) > 0 &&
				stringContains(toLower(s), toLower(substr)))
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + ('a' - 'A')
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
