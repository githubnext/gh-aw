//go:build !integration

package setutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/setutil"
)

// TestSpec_PublicAPI_Contains validates the documented behavior of Contains as
// described in the setutil README.md specification.
func TestSpec_PublicAPI_Contains(t *testing.T) {
	t.Run("reports membership for a present string key", func(t *testing.T) {
		seen := map[string]struct{}{"foo": {}, "bar": {}}
		assert.True(t, setutil.Contains(seen, "foo"), "Contains should return true for a documented present key")
	})

	t.Run("reports non-membership for an absent string key", func(t *testing.T) {
		seen := map[string]struct{}{"foo": {}, "bar": {}}
		assert.False(t, setutil.Contains(seen, "baz"), "Contains should return false for an absent key")
	})

	t.Run("works with other comparable key types", func(t *testing.T) {
		seen := map[int]struct{}{1: {}, 2: {}}
		assert.True(t, setutil.Contains(seen, 1), "Contains should support generic comparable key types")
		assert.False(t, setutil.Contains(seen, 3), "Contains should report false for an absent comparable key")
	})

	t.Run("does not mutate the input set", func(t *testing.T) {
		seen := map[string]struct{}{"foo": {}, "bar": {}}
		beforeLen := len(seen)

		_ = setutil.Contains(seen, "foo")
		_ = setutil.Contains(seen, "baz")

		assert.Len(t, seen, beforeLen, "Contains should not change the size of the input set")
		_, hasFoo := seen["foo"]
		_, hasBar := seen["bar"]
		assert.True(t, hasFoo, "Contains should not remove existing keys from the input set")
		assert.True(t, hasBar, "Contains should not remove existing keys from the input set")
	})
}
