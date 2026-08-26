//go:build !integration

package typeutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeStringSlice(t *testing.T) {
	t.Parallel()

	t.Run("wraps a scalar string", func(t *testing.T) {
		assert.Equal(t, []string{"build"}, NormalizeStringSlice("build"))
		assert.Equal(t, []string{""}, NormalizeStringSlice(""))
	})

	t.Run("copies a string slice", func(t *testing.T) {
		input := []string{"a", "b"}
		result := NormalizeStringSlice(input)
		assert.Equal(t, input, result)
		result[0] = "mutated"
		assert.Equal(t, []string{"a", "b"}, input)
	})

	t.Run("keeps string elements from an any slice", func(t *testing.T) {
		assert.Equal(t, []string{"a", "b"}, NormalizeStringSlice([]any{"a", 42, "b", nil}))
		assert.Equal(t, []string{}, NormalizeStringSlice([]any{}))
		assert.Equal(t, []string{}, NormalizeStringSlice([]any{1, true}))
	})

	t.Run("returns nil for unsupported values", func(t *testing.T) {
		assert.Nil(t, NormalizeStringSlice(nil))
		assert.Nil(t, NormalizeStringSlice(42))
		assert.Nil(t, NormalizeStringSlice(map[string]any{"a": "b"}))
	})
}
