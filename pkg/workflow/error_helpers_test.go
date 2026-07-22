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
		require.ErrorContains(t, err, "Validation failed for field 'title'")
		require.ErrorContains(t, err, "Reason: cannot be empty")
		require.ErrorContains(t, err, "Suggestion: Provide a non-empty title")

		// Check timestamp is included
		require.ErrorContains(t, err, "[")
		require.ErrorContains(t, err, "T")
	})

	t.Run("validation error with long value", func(t *testing.T) {
		longValue := strings.Repeat("a", 200)
		err := NewValidationError("body", longValue, "too long", "Shorten the body")

		require.Error(t, err)
		// Value should be truncated
		require.ErrorContains(t, err, "...")
		assert.Less(t, len(err.Error()), len(longValue)+200)
	})

	t.Run("validation error without suggestion", func(t *testing.T) {
		err := NewValidationError("labels", "invalid", "not allowed", "")

		require.Error(t, err)
		require.ErrorContains(t, err, "Validation failed")
		assert.NotContains(t, err.Error(), "Suggestion:")
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
		require.ErrorContains(t, err, "Failed to update issue #123")
		require.ErrorContains(t, err, "Underlying error: API error")
		require.ErrorContains(t, err, "Suggestion: Check permissions")
	})

	t.Run("operation error without entity ID", func(t *testing.T) {
		cause := errors.New("Network error")
		err := NewOperationError("create", "PR", "", cause, "")

		require.Error(t, err)
		require.ErrorContains(t, err, "Failed to create PR")
		assert.NotContains(t, err.Error(), "#")
		// Should have default suggestion
		require.ErrorContains(t, err, "Check that the PR exists")
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

		require.ErrorContains(t, err, "[")
		assert.ErrorContains(t, err, "T")
	})
}

func TestConfigurationError(t *testing.T) {
	t.Run("basic configuration error", func(t *testing.T) {
		err := NewConfigurationError("safe-outputs.max", "abc", "must be an integer", "Use a numeric value")

		require.Error(t, err)
		require.ErrorContains(t, err, "Configuration error in 'safe-outputs.max'")
		require.ErrorContains(t, err, "Value: abc")
		require.ErrorContains(t, err, "Reason: must be an integer")
		require.ErrorContains(t, err, "Suggestion: Use a numeric value")
	})

	t.Run("configuration error with default suggestion", func(t *testing.T) {
		err := NewConfigurationError("safe-outputs.target", "invalid", "not a valid target", "")

		require.Error(t, err)
		require.ErrorContains(t, err, "Configuration error")
		require.ErrorContains(t, err, "Check the safe-outputs configuration")
	})

	t.Run("configuration error with long value", func(t *testing.T) {
		longValue := strings.Repeat("x", 200)
		err := NewConfigurationError("config.field", longValue, "invalid", "")

		require.Error(t, err)
		// Value should be truncated
		require.ErrorContains(t, err, "...")
	})
}
