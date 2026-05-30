//go:build !integration

package console

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptSecretInput(t *testing.T) {
	t.Run("function signature", func(t *testing.T) {
		// Verify the function exists and has the right signature
		_ = PromptSecretInput
	})

	t.Run("validates parameters", func(t *testing.T) {
		title := "Enter Secret"
		description := "Secret value will be masked"

		// Function exists and parameters are accepted
		_, err := PromptSecretInput(title, description)
		// Will error in test environment (no TTY), but that's expected
		require.Error(t, err, "Should error when not in TTY")
		assert.Contains(t, err.Error(), "not a TTY", "Error should mention TTY")
	})
}
