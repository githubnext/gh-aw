//go:build !integration

package sliceutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Additional edge case tests for better coverage

func TestFilter(t *testing.T) {
	tests := []struct {
		name      string
		slice     []string
		predicate func(string) bool
		expected  []string
	}{
		{
			name:      "nil slice returns empty slice",
			slice:     nil,
			predicate: func(s string) bool { return len(s) > 3 },
			expected:  []string{},
		},
		{
			name:      "empty slice returns empty slice",
			slice:     []string{},
			predicate: func(s string) bool { return len(s) > 3 },
			expected:  []string{},
		},
		{
			name:      "no elements match predicate",
			slice:     []string{"a", "b", "c"},
			predicate: func(s string) bool { return len(s) > 3 },
			expected:  []string{},
		},
		{
			name:      "some elements match predicate",
			slice:     []string{"apple", "fig", "banana", "kiwi"},
			predicate: func(s string) bool { return len(s) > 3 },
			expected:  []string{"apple", "banana", "kiwi"},
		},
		{
			name:      "all elements match predicate",
			slice:     []string{"apple", "banana", "cherry"},
			predicate: func(s string) bool { return len(s) > 3 },
			expected:  []string{"apple", "banana", "cherry"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Filter(tt.slice, tt.predicate)
			assert.Equal(t, tt.expected, result,
				"Filter should return correct elements for slice %v", tt.slice)
		})
	}
}

func TestMap(t *testing.T) {
	tests := []struct {
		name      string
		slice     []string
		transform func(string) int
		expected  []int
	}{
		{
			name:      "nil slice returns empty slice",
			slice:     nil,
			transform: func(s string) int { return len(s) },
			expected:  []int{},
		},
		{
			name:      "empty slice returns empty slice",
			slice:     []string{},
			transform: func(s string) int { return len(s) },
			expected:  []int{},
		},
		{
			name:      "transforms each element",
			slice:     []string{"apple", "fig", "banana"},
			transform: func(s string) int { return len(s) },
			expected:  []int{5, 3, 6},
		},
		{
			name:      "single element",
			slice:     []string{"hello"},
			transform: func(s string) int { return len(s) },
			expected:  []int{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Map(tt.slice, tt.transform)
			assert.Equal(t, tt.expected, result,
				"Map should transform all elements in slice %v", tt.slice)
		})
	}
}

func TestDeduplicate(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		expected []string
	}{
		{
			name:     "nil slice returns empty slice",
			slice:    nil,
			expected: []string{},
		},
		{
			name:     "empty slice returns empty slice",
			slice:    []string{},
			expected: []string{},
		},
		{
			name:     "no duplicates returns same elements in order",
			slice:    []string{"apple", "banana", "cherry"},
			expected: []string{"apple", "banana", "cherry"},
		},
		{
			name:     "partial duplicates removed preserving first occurrence order",
			slice:    []string{"apple", "banana", "apple", "cherry", "banana"},
			expected: []string{"apple", "banana", "cherry"},
		},
		{
			name:     "all duplicates returns single element",
			slice:    []string{"apple", "apple", "apple"},
			expected: []string{"apple"},
		},
		{
			name:     "single element",
			slice:    []string{"apple"},
			expected: []string{"apple"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Deduplicate(tt.slice)
			assert.Equal(t, tt.expected, result,
				"Deduplicate should remove duplicates from slice %v", tt.slice)
		})
	}
}

func TestMapToSlice(t *testing.T) {
	t.Run("nil map returns empty slice", func(t *testing.T) {
		result := MapToSlice[string, int](nil)
		assert.Empty(t, result, "MapToSlice should return empty slice for nil map")
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		result := MapToSlice(map[string]int{})
		assert.Empty(t, result, "MapToSlice should return empty slice for empty map")
	})

	t.Run("returns all keys in any order", func(t *testing.T) {
		m := map[string]int{"apple": 1, "banana": 2, "cherry": 3}
		result := MapToSlice(m)
		assert.ElementsMatch(t, []string{"apple", "banana", "cherry"}, result,
			"MapToSlice should return all keys from map")
	})

	t.Run("single entry map", func(t *testing.T) {
		m := map[string]bool{"only": true}
		result := MapToSlice(m)
		assert.Equal(t, []string{"only"}, result,
			"MapToSlice should return the single key from a one-entry map")
	})
}

func TestFilterMapKeys(t *testing.T) {
	t.Run("nil map returns empty slice", func(t *testing.T) {
		result := FilterMapKeys[string, int](nil, func(k string, v int) bool { return true })
		assert.Empty(t, result, "FilterMapKeys should return empty slice for nil map")
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		result := FilterMapKeys(map[string]int{}, func(k string, v int) bool { return true })
		assert.Empty(t, result, "FilterMapKeys should return empty slice for empty map")
	})

	t.Run("no keys match predicate", func(t *testing.T) {
		m := map[string]int{"apple": 1, "banana": 2}
		result := FilterMapKeys(m, func(k string, v int) bool { return v > 10 })
		assert.Empty(t, result, "FilterMapKeys should return empty slice when no keys match")
	})

	t.Run("some keys match predicate", func(t *testing.T) {
		m := map[string]int{"apple": 5, "banana": 2, "cherry": 8}
		result := FilterMapKeys(m, func(k string, v int) bool { return v > 4 })
		assert.ElementsMatch(t, []string{"apple", "cherry"}, result,
			"FilterMapKeys should return only keys whose values match the predicate")
	})

	t.Run("all keys match predicate", func(t *testing.T) {
		m := map[string]int{"apple": 5, "banana": 7, "cherry": 9}
		result := FilterMapKeys(m, func(k string, v int) bool { return v > 0 })
		assert.ElementsMatch(t, []string{"apple", "banana", "cherry"}, result,
			"FilterMapKeys should return all keys when all match the predicate")
	})

	t.Run("predicate uses key value", func(t *testing.T) {
		m := map[string]int{"keep-me": 1, "skip-me": 2, "keep-too": 3}
		result := FilterMapKeys(m, func(k string, v int) bool { return strings.HasPrefix(k, "keep") })
		assert.ElementsMatch(t, []string{"keep-me", "keep-too"}, result,
			"FilterMapKeys should support predicates that filter on key names")
	})
}

func TestAny(t *testing.T) {
	tests := []struct {
		name      string
		slice     []int
		predicate func(int) bool
		expected  bool
	}{
		{
			name:      "at least one element matches",
			slice:     []int{1, 2, 3, 4, 5},
			predicate: func(x int) bool { return x > 3 },
			expected:  true,
		},
		{
			name:      "no element matches",
			slice:     []int{1, 2, 3},
			predicate: func(x int) bool { return x > 10 },
			expected:  false,
		},
		{
			name:      "empty slice returns false",
			slice:     []int{},
			predicate: func(x int) bool { return true },
			expected:  false,
		},
		{
			name:      "nil slice returns false",
			slice:     nil,
			predicate: func(x int) bool { return true },
			expected:  false,
		},
		{
			name:      "single element matches",
			slice:     []int{42},
			predicate: func(x int) bool { return x == 42 },
			expected:  true,
		},
		{
			name:      "single element does not match",
			slice:     []int{42},
			predicate: func(x int) bool { return x == 0 },
			expected:  false,
		},
		{
			name:      "all elements match",
			slice:     []int{2, 4, 6, 8},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Any(tt.slice, tt.predicate)
			assert.Equal(t, tt.expected, result,
				"Any should return %v for slice %v", tt.expected, tt.slice)
		})
	}
}

func TestAny_Strings(t *testing.T) {
	secrets := map[string]bool{"SECRET_A": true, "SECRET_B": false}

	// Mirrors the pattern used in engine_secrets.go
	exists := Any([]string{"SECRET_A", "SECRET_C"}, func(alt string) bool {
		return secrets[alt]
	})
	assert.True(t, exists, "Any should return true when one alternative secret exists")

	notExists := Any([]string{"SECRET_C", "SECRET_D"}, func(alt string) bool {
		return secrets[alt]
	})
	assert.False(t, notExists, "Any should return false when no alternative secret exists")
}

func TestAny_StopsEarly(t *testing.T) {
	callCount := 0
	slice := []int{1, 2, 3, 4, 5}
	Any(slice, func(x int) bool {
		callCount++
		return x == 2 // matches at index 1
	})
	assert.Equal(t, 2, callCount, "Any should stop evaluating after first match")
}
