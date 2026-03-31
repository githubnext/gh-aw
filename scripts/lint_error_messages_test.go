//go:build !integration

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckErrorQuality(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		shouldPass  bool
		description string
	}{
		{
			name:        "good validation error with example",
			message:     `invalid time delta format: +%s. Expected format like +25h, +3d, +1w, +1mo. Example: +3d`,
			shouldPass:  true,
			description: "Has 'invalid', 'Expected', and 'Example'",
		},
		{
			name:        "good type error with example",
			message:     `manual-approval value must be a string, got %T. Example: manual-approval: "production"`,
			shouldPass:  true,
			description: "Has 'must be', and 'Example'",
		},
		{
			name:        "good enum error with example",
			message:     `invalid engine: %s. Valid engines are: copilot, claude, codex, custom. Example: engine: copilot`,
			shouldPass:  true,
			description: "Has 'invalid', 'Valid engines', and 'Example'",
		},
		{
			name:        "bad validation error without example",
			message:     `invalid format`,
			shouldPass:  false,
			description: "Has 'invalid' but no example",
		},
		{
			name:        "bad type error without example",
			message:     `manual-approval value must be a string`,
			shouldPass:  false,
			description: "Has 'must be' but no example",
		},
		{
			name:        "wrapped error should pass",
			message:     `failed to parse configuration: %w`,
			shouldPass:  true,
			description: "Wrapped errors are allowed to skip quality check",
		},
		{
			name:        "error with doc link should pass",
			message:     `unsupported feature. See https://docs.example.com/features`,
			shouldPass:  true,
			description: "Errors with documentation links can skip examples",
		},
		{
			name:        "short simple error should pass",
			message:     `not found`,
			shouldPass:  true,
			description: "Very short errors can be self-explanatory",
		},
		{
			name:        "duplicate error should pass",
			message:     `duplicate unit 'd' in time delta`,
			shouldPass:  true,
			description: "Self-explanatory duplicate error",
		},
		{
			name:        "missing required field with example",
			message:     `tool 'my-tool' missing required 'command' field. Example: tools:\n  my-tool:\n    command: "node server.js"`,
			shouldPass:  true,
			description: "Has 'missing required' and 'Example'",
		},
		{
			name:        "config error without example",
			message:     `tool 'my-tool' mcp configuration must specify either 'command' or 'container'`,
			shouldPass:  false,
			description: "Configuration error without example should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := checkErrorQuality(tt.message, 1)

			if tt.shouldPass {
				assert.Nil(t, issue, "%s: message %q should pass quality check", tt.description, tt.message)
			} else {
				assert.NotNil(t, issue, "%s: message %q should fail quality check", tt.description, tt.message)
			}
		})
	}
}

func TestShouldSkipQualityCheck(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		shouldSkip bool
	}{
		{
			name:       "wrapped error",
			message:    "failed to parse: %w",
			shouldSkip: true,
		},
		{
			name:       "doc link",
			message:    "see https://docs.example.com",
			shouldSkip: true,
		},
		{
			name:       "very short",
			message:    "not found",
			shouldSkip: true,
		},
		{
			name:       "duplicate error",
			message:    "duplicate unit",
			shouldSkip: true,
		},
		{
			name:       "empty string",
			message:    "empty time delta",
			shouldSkip: true,
		},
		{
			name:       "validation error should not skip",
			message:    "invalid engine configuration that is longer than fifty characters",
			shouldSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipQualityCheck(tt.message)
			assert.Equal(t, tt.shouldSkip, result, "shouldSkipQualityCheck(%q) should return %v", tt.message, tt.shouldSkip)
		})
	}
}

func TestSuggestImprovement(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantContain string
	}{
		{
			name:        "format error",
			message:     "invalid time format",
			wantContain: "format",
		},
		{
			name:        "type error",
			message:     "value must be a string, got %T",
			wantContain: "type",
		},
		{
			name:        "enum error",
			message:     "invalid engine",
			wantContain: "valid options",
		},
		{
			name:        "missing field",
			message:     "missing required field",
			wantContain: "required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := suggestImprovement(tt.message)
			assert.NotEmpty(t, result, "suggestImprovement(%q) should return a non-empty suggestion", tt.message)
			assert.Contains(t, result, tt.wantContain, "suggestion for %q should contain %q", tt.message, tt.wantContain)
		})
	}
}

func TestPatternMatching(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		message string
		want    bool
	}{
		// hasExample
		{
			name:    "hasExample matches with colon and space",
			pattern: "hasExample",
			message: "Example: field: value",
			want:    true,
		},
		{
			name:    "hasExample no match without colon",
			pattern: "hasExample",
			message: "here is an example of usage",
			want:    false,
		},
		// hasExpected
		{
			name:    "hasExpected matches expected",
			pattern: "hasExpected",
			message: "Expected format: YYYY-MM-DD",
			want:    true,
		},
		{
			name:    "hasExpected matches must be",
			pattern: "hasExpected",
			message: "value must be a string",
			want:    true,
		},
		{
			name:    "hasExpected no match",
			pattern: "hasExpected",
			message: "something went wrong",
			want:    false,
		},
		// isValidationError
		{
			name:    "isValidationError matches invalid",
			pattern: "isValidationError",
			message: "invalid configuration",
			want:    true,
		},
		{
			name:    "isValidationError matches must",
			pattern: "isValidationError",
			message: "value must be positive",
			want:    true,
		},
		{
			name:    "isValidationError matches missing",
			pattern: "isValidationError",
			message: "missing required field",
			want:    true,
		},
		{
			name:    "isValidationError no match",
			pattern: "isValidationError",
			message: "operation succeeded",
			want:    false,
		},
		// isFormatError
		{
			name:    "isFormatError matches format",
			pattern: "isFormatError",
			message: "invalid time format",
			want:    true,
		},
		{
			name:    "isFormatError no match",
			pattern: "isFormatError",
			message: "something went wrong",
			want:    false,
		},
		// isTypeError
		{
			name:    "isTypeError matches got %T",
			pattern: "isTypeError",
			message: "value must be a string, got %T",
			want:    true,
		},
		{
			name:    "isTypeError matches must be",
			pattern: "isTypeError",
			message: "field must be an integer",
			want:    true,
		},
		{
			name:    "isTypeError no match",
			pattern: "isTypeError",
			message: "invalid configuration",
			want:    false,
		},
		// isEnumError
		{
			name:    "isEnumError matches valid options",
			pattern: "isEnumError",
			message: "invalid level. Valid options are: info, warn, error",
			want:    true,
		},
		{
			name:    "isEnumError matches one of",
			pattern: "isEnumError",
			message: "must be one of: read, write, admin",
			want:    true,
		},
		{
			name:    "isEnumError no match",
			pattern: "isEnumError",
			message: "invalid configuration format",
			want:    false,
		},
		// isWrappedError
		{
			name:    "isWrappedError matches %w verb",
			pattern: "isWrappedError",
			message: "failed to parse config: %w",
			want:    true,
		},
		{
			name:    "isWrappedError no match",
			pattern: "isWrappedError",
			message: "failed to parse configuration",
			want:    false,
		},
		// hasDocLink
		{
			name:    "hasDocLink matches https URL",
			pattern: "hasDocLink",
			message: "see https://docs.example.com for more info",
			want:    true,
		},
		{
			name:    "hasDocLink matches http URL",
			pattern: "hasDocLink",
			message: "see http://docs.example.com for more info",
			want:    true,
		},
		{
			name:    "hasDocLink no match",
			pattern: "hasDocLink",
			message: "see the documentation for more info",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			switch tt.pattern {
			case "hasExample":
				result = hasExample.MatchString(tt.message)
			case "hasExpected":
				result = hasExpected.MatchString(tt.message)
			case "isValidationError":
				result = isValidationError.MatchString(tt.message)
			case "isFormatError":
				result = isFormatError.MatchString(tt.message)
			case "isTypeError":
				result = isTypeError.MatchString(tt.message)
			case "isEnumError":
				result = isEnumError.MatchString(tt.message)
			case "isWrappedError":
				result = isWrappedError.MatchString(tt.message)
			case "hasDocLink":
				result = hasDocLink.MatchString(tt.message)
			}

			assert.Equal(t, tt.want, result, "pattern %q match on %q should be %v", tt.pattern, tt.message, tt.want)
		})
	}
}

func TestAnalyzeFile(t *testing.T) {
	t.Run("compliant error messages", func(t *testing.T) {
		src := `package example

import "fmt"

func goodErrors() error {
	return fmt.Errorf("invalid time delta format: +%s. Expected format like +25h, +3d. Example: +3d")
}
`
		path := writeTempGoFile(t, src)
		stats := analyzeFile(path)

		require.NotNil(t, stats, "analyzeFile should return stats")
		assert.Equal(t, 1, stats.Total, "should detect one error message")
		assert.Equal(t, 1, stats.Compliant, "compliant error should be counted")
		assert.Empty(t, stats.Issues, "no quality issues expected for compliant error")
	})

	t.Run("non-compliant error messages", func(t *testing.T) {
		src := `package example

import "fmt"

func badErrors() error {
	return fmt.Errorf("invalid engine configuration that needs an example")
}
`
		path := writeTempGoFile(t, src)
		stats := analyzeFile(path)

		require.NotNil(t, stats, "analyzeFile should return stats")
		assert.Equal(t, 1, stats.Total, "should detect one error message")
		assert.Equal(t, 0, stats.Compliant, "non-compliant error should not be counted")
		assert.Len(t, stats.Issues, 1, "should report one quality issue")
	})

	t.Run("mixed compliant and non-compliant errors", func(t *testing.T) {
		src := `package example

import "fmt"

func mixedErrors() {
	_ = fmt.Errorf("invalid engine: %s. Valid engines: copilot, claude. Example: engine: copilot")
	_ = fmt.Errorf("invalid mode without example")
	_ = fmt.Errorf("wrapped config error: %w")
}
`
		path := writeTempGoFile(t, src)
		stats := analyzeFile(path)

		require.NotNil(t, stats, "analyzeFile should return stats")
		assert.Equal(t, 3, stats.Total, "should detect three error messages")
		assert.Equal(t, 2, stats.Compliant, "two messages should be compliant")
		assert.Len(t, stats.Issues, 1, "should report one quality issue")
	})

	t.Run("file with no error messages", func(t *testing.T) {
		src := `package example

func noErrors() string {
	return "hello world"
}
`
		path := writeTempGoFile(t, src)
		stats := analyzeFile(path)

		require.NotNil(t, stats, "analyzeFile should return stats even with no errors")
		assert.Equal(t, 0, stats.Total, "should detect no error messages")
		assert.Equal(t, 0, stats.Compliant, "no compliant messages expected")
		assert.Empty(t, stats.Issues, "no issues expected")
	})

	t.Run("invalid go file returns empty stats", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "invalid.go")
		err := os.WriteFile(path, []byte("this is not valid go code }{"), 0600)
		require.NoError(t, err, "should create invalid Go file")

		stats := analyzeFile(path)

		require.NotNil(t, stats, "analyzeFile should return stats even for invalid file")
		assert.Equal(t, 0, stats.Total, "invalid file should have no messages")
	})
}

// writeTempGoFile writes Go source code to a temporary file and returns its path.
func writeTempGoFile(t *testing.T, src string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(path, []byte(src), 0600)
	require.NoError(t, err, "should write temporary Go source file")
	return path
}
