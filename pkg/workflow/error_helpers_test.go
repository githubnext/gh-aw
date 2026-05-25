//go:build !integration

package workflow

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError(t *testing.T) {
	t.Run("basic validation error", func(t *testing.T) {
		err := NewValidationError("title", "", "cannot be empty", "Provide a non-empty title")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Validation failed for field 'title'")
		assert.Contains(t, err.Error(), "Reason: cannot be empty")
		assert.Contains(t, err.Error(), "Suggestion: Provide a non-empty title")

		// Check timestamp is included
		assert.Contains(t, err.Error(), "[")
		assert.Contains(t, err.Error(), "T")
	})

	t.Run("validation error with long value", func(t *testing.T) {
		longValue := strings.Repeat("a", 200)
		err := NewValidationError("body", longValue, "too long", "Shorten the body")

		require.Error(t, err)
		// Value should be truncated
		assert.Contains(t, err.Error(), "...")
		assert.Less(t, len(err.Error()), len(longValue)+200)
	})

	t.Run("validation error without suggestion", func(t *testing.T) {
		err := NewValidationError("labels", "invalid", "not allowed", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Validation failed")
		assert.NotContains(t, err.Error(), "Suggestion:")
	})
}

func TestNewValidationErrorWithLocation(t *testing.T) {
	t.Run("error message omits timestamp when line is set", func(t *testing.T) {
		loc := FieldLocation{File: "workflow.md", Line: 10, Column: 1}
		err := NewValidationErrorWithLocation("engine", "copiliot", "not a valid engine", "Did you mean 'copilot'?", loc)

		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, "Validation failed for field 'engine'")
		assert.Contains(t, msg, "Value: copiliot")
		assert.Contains(t, msg, "Reason: not a valid engine")
		assert.Contains(t, msg, "Suggestion: Did you mean 'copilot'?")
		// No timestamp prefix when location is set
		assert.NotContains(t, msg, "[20")
		assert.Equal(t, "workflow.md", err.File)
		assert.Equal(t, 10, err.Line)
		assert.Equal(t, 1, err.Column)
	})

	t.Run("location fields are stored", func(t *testing.T) {
		loc := FieldLocation{File: "/path/to/wf.md", Line: 5, Column: 3}
		err := NewValidationErrorWithLocation("concurrency", "", "invalid value", "", loc)

		require.NotNil(t, err)
		assert.Equal(t, "/path/to/wf.md", err.File)
		assert.Equal(t, 5, err.Line)
		assert.Equal(t, 3, err.Column)
	})

	t.Run("zero-line location keeps timestamp", func(t *testing.T) {
		// When Line is 0, behavior should match NewValidationError (includes timestamp)
		loc := FieldLocation{File: "workflow.md", Line: 0}
		err := NewValidationErrorWithLocation("engine", "bad", "reason", "", loc)

		require.Error(t, err)
		// Line == 0 means no location info → timestamp should appear
		assert.Contains(t, err.Error(), "[")
	})

	t.Run("error with value truncation", func(t *testing.T) {
		longValue := strings.Repeat("x", 200)
		loc := FieldLocation{File: "workflow.md", Line: 3}
		err := NewValidationErrorWithLocation("field", longValue, "too long", "", loc)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "...")
	})
}

func TestFieldLocationAlias(t *testing.T) {
	loc := FieldLocation{File: "workflow.md", Line: 12, Column: 4}

	pos := loc
	assert.Equal(t, "workflow.md", pos.File)
	assert.Equal(t, 12, pos.Line)
	assert.Equal(t, 4, pos.Column)
}

func TestOperationError(t *testing.T) {
	t.Run("basic operation error", func(t *testing.T) {
		cause := errors.New("API error")
		err := NewOperationError("update", "issue", "123", cause, "Check permissions")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to update issue #123")
		assert.Contains(t, err.Error(), "Underlying error: API error")
		assert.Contains(t, err.Error(), "Suggestion: Check permissions")
	})

	t.Run("operation error without entity ID", func(t *testing.T) {
		cause := errors.New("Network error")
		err := NewOperationError("create", "PR", "", cause, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to create PR")
		assert.NotContains(t, err.Error(), "#")
		// Should have default suggestion
		assert.Contains(t, err.Error(), "Check that the PR exists")
	})

	t.Run("operation error unwrap", func(t *testing.T) {
		cause := errors.New("original error")
		err := NewOperationError("delete", "comment", "456", cause, "")

		unwrapped := errors.Unwrap(err)
		assert.Equal(t, cause, unwrapped)
	})

	t.Run("operation error with timestamp", func(t *testing.T) {
		cause := errors.New("failed")
		err := NewOperationError("operation", "entity", "1", cause, "")

		assert.Contains(t, err.Error(), "[")
		assert.Contains(t, err.Error(), "T")
	})
}

func TestConfigurationError(t *testing.T) {
	t.Run("basic configuration error", func(t *testing.T) {
		err := NewConfigurationError("safe-outputs.max", "abc", "must be an integer", "Use a numeric value")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Configuration error in 'safe-outputs.max'")
		assert.Contains(t, err.Error(), "Value: abc")
		assert.Contains(t, err.Error(), "Reason: must be an integer")
		assert.Contains(t, err.Error(), "Suggestion: Use a numeric value")
	})

	t.Run("configuration error with default suggestion", func(t *testing.T) {
		err := NewConfigurationError("safe-outputs.target", "invalid", "not a valid target", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Configuration error")
		assert.Contains(t, err.Error(), "Check the safe-outputs configuration")
	})

	t.Run("configuration error with long value", func(t *testing.T) {
		longValue := strings.Repeat("x", 200)
		err := NewConfigurationError("config.field", longValue, "invalid", "")

		require.Error(t, err)
		// Value should be truncated
		assert.Contains(t, err.Error(), "...")
	})
}
