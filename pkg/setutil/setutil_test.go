//go:build !integration

package setutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpec_PublicAPI_Contains validates the documented behavior of Contains as
// described in the setutil README.md specification.
func TestSpec_PublicAPI_Contains(t *testing.T) {
	t.Run("returns true when key is present", func(t *testing.T) {
		set := map[string]struct{}{"a": {}, "b": {}}
		assert.True(t, Contains(set, "a"), "Contains should return true for a present key")
	})

	t.Run("returns false when key is absent", func(t *testing.T) {
		set := map[string]struct{}{"a": {}, "b": {}}
		assert.False(t, Contains(set, "c"), "Contains should return false for an absent key")
	})

	t.Run("returns false for nil set", func(t *testing.T) {
		var set map[string]struct{}
		assert.False(t, Contains(set, "a"), "Contains should return false for a nil set")
	})

	t.Run("returns false for empty set", func(t *testing.T) {
		set := map[string]struct{}{}
		assert.False(t, Contains(set, "a"), "Contains should return false for an empty set")
	})

	t.Run("works with non-string comparable key types", func(t *testing.T) {
		set := map[int]struct{}{1: {}, 2: {}}
		assert.True(t, Contains(set, 1), "Contains should work with int keys")
		assert.False(t, Contains(set, 3), "Contains should return false for absent int key")
	})
}
