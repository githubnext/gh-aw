//go:build !integration

package workflow

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatCompilerError tests the formatCompilerError helper function
func TestFormatCompilerError(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		errType     string
		message     string
		cause       error
		wantContain []string
	}{
		{
			name:     "error type with simple message, no cause",
			filePath: "/path/to/workflow.md",
			errType:  "error",
			message:  "validation failed",
			cause:    nil,
			wantContain: []string{
				"/path/to/workflow.md",
				"1:1",
				"error",
				"validation failed",
			},
		},
		{
			name:     "warning type with detailed message, no cause",
			filePath: "/path/to/workflow.md",
			errType:  "warning",
			message:  "missing required permission",
			cause:    nil,
			wantContain: []string{
				"/path/to/workflow.md",
				"1:1",
				"warning",
				"missing required permission",
			},
		},
		{
			name:     "error with underlying cause",
			filePath: "/path/to/workflow.md",
			errType:  "error",
			message:  "failed to parse YAML",
			cause:    errors.New("syntax error at line 42"),
			wantContain: []string{
				"/path/to/workflow.md",
				"1:1",
				"error",
				"failed to parse YAML",
			},
		},
		{
			name:     "lock file path",
			filePath: "/path/to/workflow.lock.yml",
			errType:  "error",
			message:  "failed to write lock file",
			cause:    nil,
			wantContain: []string{
				"/path/to/workflow.lock.yml",
				"1:1",
				"error",
				"failed to write lock file",
			},
		},
		{
			name:     "formatted message with error details and cause",
			filePath: "test.md",
			errType:  "error",
			message:  "failed to generate YAML: syntax error",
			cause:    errors.New("underlying error"),
			wantContain: []string{
				"test.md",
				"1:1",
				"error",
				"failed to generate YAML: syntax error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatCompilerError(tt.filePath, tt.errType, tt.message, tt.cause)
			require.Error(t, err, "formatCompilerError should return an error")

			errStr := err.Error()
			for _, want := range tt.wantContain {
				assert.Contains(t, errStr, want, "Error message should contain: %s", want)
			}

			// If cause is provided, verify error wrapping
			if tt.cause != nil {
				assert.ErrorIs(t, err, tt.cause, "Error should wrap the cause")
			}
		})
	}
}

// TestFormatCompilerError_OutputFormat verifies the output format remains consistent
func TestFormatCompilerError_OutputFormat(t *testing.T) {
	err := formatCompilerError("/test/workflow.md", "error", "test message", nil)
	require.Error(t, err)

	errStr := err.Error()

	// Verify the error format contains the standard compiler error structure
	assert.Contains(t, errStr, "/test/workflow.md", "Should contain file path")
	// formatCompilerError always uses 1:1 so IDE tooling can navigate to the file
	assert.Contains(t, errStr, "1:1", "Should contain line:column for IDE integration")
	assert.Contains(t, errStr, "error", "Should contain error type")
	assert.Contains(t, errStr, "test message", "Should contain message")
}

// TestFormatCompilerError_ErrorVsWarning tests differentiation between error and warning types
func TestFormatCompilerError_ErrorVsWarning(t *testing.T) {
	errorErr := formatCompilerError("test.md", "error", "error message", nil)
	warningErr := formatCompilerError("test.md", "warning", "warning message", nil)

	require.Error(t, errorErr)
	require.Error(t, warningErr)

	assert.Contains(t, errorErr.Error(), "error", "Error type should be present")
	assert.Contains(t, warningErr.Error(), "warning", "Warning type should be present")

	// Ensure they produce different outputs
	assert.NotEqual(t, errorErr.Error(), warningErr.Error(), "Error and warning should have different outputs")
}

// TestFormatCompilerMessage tests the formatCompilerMessage helper function
func TestFormatCompilerMessage(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		msgType     string
		message     string
		wantContain []string
	}{
		{
			name:     "warning message",
			filePath: "/path/to/workflow.md",
			msgType:  "warning",
			message:  "container image validation failed",
			wantContain: []string{
				"/path/to/workflow.md",
				"warning",
				"container image validation failed",
			},
		},
		{
			name:     "error message as string",
			filePath: "test.md",
			msgType:  "error",
			message:  "validation error",
			wantContain: []string{
				"test.md",
				"error",
				"validation error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := formatCompilerMessage(tt.filePath, tt.msgType, tt.message)

			for _, want := range tt.wantContain {
				assert.Contains(t, msg, want, "Message should contain: %s", want)
			}
		})
	}
}

// TestFormatCompilerError_ErrorWrapping verifies that error wrapping preserves error chains
func TestFormatCompilerError_ErrorWrapping(t *testing.T) {
	// Create an underlying error
	underlyingErr := errors.New("underlying validation error")

	// Wrap it with formatCompilerError
	wrappedErr := formatCompilerError("test.md", "error", "validation failed", underlyingErr)

	require.Error(t, wrappedErr)

	// Verify error chain is preserved
	require.ErrorIs(t, wrappedErr, underlyingErr, "Should preserve error chain with %w")

	// Verify formatted message is in the error string
	assert.Contains(t, wrappedErr.Error(), "test.md")
	assert.Contains(t, wrappedErr.Error(), "validation failed")
	// Verify the formatted string does NOT include the cause text (no duplication)
	assert.NotContains(t, wrappedErr.Error(), "underlying validation error", "Error() should not duplicate cause text")
}

// TestFormatCompilerError_SameMessageAndCause verifies the common pattern where err.Error()
// is passed as both message and cause: the displayed string stays clean and errors.Is still works.
func TestFormatCompilerError_SameMessageAndCause(t *testing.T) {
	underlying := errors.New("yaml syntax error")

	// This is the most common call pattern in compiler.go:
	//   return formatCompilerError(path, "error", err.Error(), err)
	wrappedErr := formatCompilerError("test.md", "error", underlying.Error(), underlying)

	require.Error(t, wrappedErr)

	// errors.Is must still work even though message == cause.Error()
	require.ErrorIs(t, wrappedErr, underlying, "Should preserve error chain even when message == cause.Error()")

	// The Error() string should contain the message exactly once
	errStr := wrappedErr.Error()
	assert.Contains(t, errStr, "yaml syntax error", "Should contain message")
	count := strings.Count(errStr, "yaml syntax error")
	assert.Equal(t, 1, count, "Message should appear exactly once in error string (no duplication)")
}

// TestFormatCompilerError_NilCause verifies that nil cause creates a new error
func TestFormatCompilerError_NilCause(t *testing.T) {
	err := formatCompilerError("test.md", "error", "validation error", nil)

	require.Error(t, err)

	// Verify error message contains expected content
	assert.Contains(t, err.Error(), "test.md")
	assert.Contains(t, err.Error(), "validation error")

	// Verify it's a new error (not wrapping anything)
	// This is a validation error, so it should not wrap
	dummyErr := errors.New("some other error")
	assert.NotErrorIs(t, err, dummyErr, "Should not wrap any error when cause is nil")
}
