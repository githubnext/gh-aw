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
