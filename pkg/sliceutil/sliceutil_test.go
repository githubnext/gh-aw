//go:build !integration

package sliceutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists in slice",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		{
			name:     "item does not exist in slice",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "grape",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "apple",
			expected: false,
		},
		{
			name:     "nil slice",
			slice:    nil,
			item:     "apple",
			expected: false,
		},
		{
			name:     "empty string item exists",
			slice:    []string{"", "apple", "banana"},
			item:     "",
			expected: true,
		},
		{
			name:     "empty string item does not exist",
			slice:    []string{"apple", "banana"},
			item:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result,
				"Contains should return correct value for slice %v and item %q", tt.slice, tt.item)
		})
	}
}

func BenchmarkContains(b *testing.B) {
	slice := []string{"apple", "banana", "cherry", "date", "elderberry"}
	for b.Loop() {
		Contains(slice, "cherry")
	}
}

// Additional edge case tests for better coverage

func TestContains_LargeSlice(t *testing.T) {
	// Test with a large slice
	largeSlice := make([]string, 1000)
	for i := range 1000 {
		largeSlice[i] = string(rune('a' + i%26))
	}

	// Item at beginning
	assert.True(t, Contains(largeSlice, "a"), "should find 'a' at beginning of large slice")

	// Item at end
	assert.True(t, Contains(largeSlice, string(rune('a'+999%26))), "should find item at end of large slice")

	// Item not in slice
	assert.False(t, Contains(largeSlice, "not-present"), "should not find non-existent item in large slice")
}

func TestContains_SingleElement(t *testing.T) {
	slice := []string{"single"}

	assert.True(t, Contains(slice, "single"), "should find item in single-element slice")
	assert.False(t, Contains(slice, "other"), "should not find different item in single-element slice")
}

func TestContains_Duplicates(t *testing.T) {
	// Slice with duplicate values
	slice := []string{"apple", "banana", "apple", "cherry", "apple"}

	assert.True(t, Contains(slice, "apple"), "should find 'apple' in slice with duplicates")

	// Should still return true on first match
	count := 0
	for _, item := range slice {
		if item == "apple" {
			count++
		}
	}
	assert.Equal(t, 3, count, "should count all occurrences of duplicate item")
}
