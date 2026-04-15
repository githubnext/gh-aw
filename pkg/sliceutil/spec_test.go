//go:build !integration

package sliceutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpec_PublicAPI_Filter validates the documented behavior of Filter as
// described in the sliceutil README.md specification.
func TestSpec_PublicAPI_Filter(t *testing.T) {
	t.Run("returns only elements matching predicate", func(t *testing.T) {
		numbers := []int{1, 2, 3, 4, 5}
		evens := Filter(numbers, func(n int) bool { return n%2 == 0 })
		assert.Equal(t, []int{2, 4}, evens, "Filter should return only even numbers")
	})

	t.Run("does not modify input slice", func(t *testing.T) {
		original := []int{1, 2, 3, 4, 5}
		input := make([]int, len(original))
		copy(input, original)
		_ = Filter(input, func(n int) bool { return n%2 == 0 })
		assert.Equal(t, original, input, "Filter must not modify the input slice")
	})
}

// TestSpec_PublicAPI_Map validates the documented behavior of Map as described
// in the sliceutil README.md specification.
func TestSpec_PublicAPI_Map(t *testing.T) {
	t.Run("applies transform to every element", func(t *testing.T) {
		names := []string{"alice", "bob"}
		upper := Map(names, strings.ToUpper)
		assert.Equal(t, []string{"ALICE", "BOB"}, upper, "Map should transform all elements")
	})

	t.Run("does not modify input slice", func(t *testing.T) {
		original := []string{"alice", "bob"}
		input := make([]string, len(original))
		copy(input, original)
		_ = Map(input, strings.ToUpper)
		assert.Equal(t, original, input, "Map must not modify the input slice")
	})
}

// TestSpec_PublicAPI_MapToSlice validates the documented behavior of MapToSlice
// as described in the sliceutil README.md specification.
func TestSpec_PublicAPI_MapToSlice(t *testing.T) {
	t.Run("returns all map keys as a slice", func(t *testing.T) {
		m := map[string]int{"a": 1, "b": 2}
		keys := MapToSlice(m)
		assert.ElementsMatch(t, []string{"a", "b"}, keys, "MapToSlice should return all map keys (order not guaranteed)")
	})

	t.Run("returns empty slice for empty map", func(t *testing.T) {
		m := map[string]int{}
		keys := MapToSlice(m)
		assert.Empty(t, keys, "MapToSlice of empty map should return empty slice")
	})
}

// TestSpec_PublicAPI_FilterMapKeys validates the documented behavior of
// FilterMapKeys as described in the sliceutil README.md specification.
func TestSpec_PublicAPI_FilterMapKeys(t *testing.T) {
	t.Run("returns keys whose predicate returns true", func(t *testing.T) {
		scores := map[string]int{"alice": 90, "bob": 50, "carol": 80}
		passed := FilterMapKeys(scores, func(name string, score int) bool {
			return score >= 75
		})
		assert.ElementsMatch(t, []string{"alice", "carol"}, passed, "FilterMapKeys should return keys where score >= 75 (order not guaranteed)")
	})

	t.Run("returns empty slice when no keys match predicate", func(t *testing.T) {
		scores := map[string]int{"alice": 40, "bob": 50}
		passed := FilterMapKeys(scores, func(_ string, score int) bool {
			return score >= 75
		})
		assert.Empty(t, passed, "FilterMapKeys should return empty slice when no keys match")
	})
}

// TestSpec_PublicAPI_Any validates the documented behavior of Any as described
// in the sliceutil README.md specification.
func TestSpec_PublicAPI_Any(t *testing.T) {
	t.Run("returns true when at least one element matches predicate", func(t *testing.T) {
		words := []string{"hello", "world"}
		result := Any(words, func(w string) bool { return w == "world" })
		assert.True(t, result, "Any should return true when a matching element exists")
	})

	t.Run("returns false when no element matches predicate", func(t *testing.T) {
		words := []string{"hello", "world"}
		result := Any(words, func(w string) bool { return w == "missing" })
		assert.False(t, result, "Any should return false when no matching element exists")
	})

	t.Run("returns false for nil slice", func(t *testing.T) {
		var nilSlice []string
		result := Any(nilSlice, func(w string) bool { return true })
		assert.False(t, result, "Any should return false for nil slice")
	})

	t.Run("returns false for empty slice", func(t *testing.T) {
		result := Any([]string{}, func(w string) bool { return true })
		assert.False(t, result, "Any should return false for empty slice")
	})
}

// TestSpec_PublicAPI_Deduplicate validates the documented behavior of
// Deduplicate as described in the sliceutil README.md specification.
func TestSpec_PublicAPI_Deduplicate(t *testing.T) {
	t.Run("removes duplicates preserving first occurrence order", func(t *testing.T) {
		items := []string{"a", "b", "a", "c", "b"}
		unique := Deduplicate(items)
		assert.Equal(t, []string{"a", "b", "c"}, unique, "Deduplicate should preserve order of first occurrence")
	})

	t.Run("does not modify input slice", func(t *testing.T) {
		original := []string{"a", "b", "a", "c"}
		input := make([]string, len(original))
		copy(input, original)
		_ = Deduplicate(input)
		assert.Equal(t, original, input, "Deduplicate must not modify the input slice")
	})

	t.Run("returns same elements for slice with no duplicates", func(t *testing.T) {
		items := []string{"x", "y", "z"}
		unique := Deduplicate(items)
		assert.Equal(t, []string{"x", "y", "z"}, unique, "Deduplicate of slice with no duplicates should return same elements in same order")
	})
}
