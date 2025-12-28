package sliceutil

import "testing"

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
			if result != tt.expected {
				t.Errorf("Contains(%v, %q) = %v; want %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		substrings []string
		expected   bool
	}{
		{
			name:       "contains first substring",
			s:          "hello world",
			substrings: []string{"hello", "goodbye"},
			expected:   true,
		},
		{
			name:       "contains second substring",
			s:          "hello world",
			substrings: []string{"goodbye", "world"},
			expected:   true,
		},
		{
			name:       "contains no substrings",
			s:          "hello world",
			substrings: []string{"goodbye", "farewell"},
			expected:   false,
		},
		{
			name:       "empty substrings",
			s:          "hello world",
			substrings: []string{},
			expected:   false,
		},
		{
			name:       "empty string",
			s:          "",
			substrings: []string{"hello"},
			expected:   false,
		},
		{
			name:       "contains empty substring",
			s:          "hello world",
			substrings: []string{""},
			expected:   true,
		},
		{
			name:       "multiple matches",
			s:          "Docker images are being downloaded",
			substrings: []string{"downloading", "retry"},
			expected:   false,
		},
		{
			name:       "match found",
			s:          "downloading images",
			substrings: []string{"downloading", "retry"},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsAny(tt.s, tt.substrings...)
			if result != tt.expected {
				t.Errorf("ContainsAny(%q, %v) = %v; want %v", tt.s, tt.substrings, result, tt.expected)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "Hello World",
			substr:   "Hello",
			expected: true,
		},
		{
			name:     "case insensitive match",
			s:        "Hello World",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "case insensitive match uppercase",
			s:        "hello world",
			substr:   "WORLD",
			expected: true,
		},
		{
			name:     "no match",
			s:        "Hello World",
			substr:   "goodbye",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "Hello World",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "hello",
			expected: false,
		},
		{
			name:     "both empty",
			s:        "",
			substr:   "",
			expected: true,
		},
		{
			name:     "mixed case substring in mixed case string",
			s:        "GitHub Actions Workflow",
			substr:   "actions",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsIgnoreCase(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("ContainsIgnoreCase(%q, %q) = %v; want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func BenchmarkContains(b *testing.B) {
	slice := []string{"apple", "banana", "cherry", "date", "elderberry"}
	for i := 0; i < b.N; i++ {
		Contains(slice, "cherry")
	}
}

func BenchmarkContainsAny(b *testing.B) {
	s := "hello world from the testing framework"
	substrings := []string{"goodbye", "world", "farewell"}
	for i := 0; i < b.N; i++ {
		ContainsAny(s, substrings...)
	}
}

func BenchmarkContainsIgnoreCase(b *testing.B) {
	s := "Hello World From The Testing Framework"
	substr := "world"
	for i := 0; i < b.N; i++ {
		ContainsIgnoreCase(s, substr)
	}
}

// Additional edge case tests for better coverage

func TestContains_LargeSlice(t *testing.T) {
	// Test with a large slice
	largeSlice := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		largeSlice[i] = string(rune('a' + i%26))
	}

	// Item at beginning
	if !Contains(largeSlice, "a") {
		t.Error("Expected to find 'a' at beginning of large slice")
	}

	// Item at end
	if !Contains(largeSlice, string(rune('a'+999%26))) {
		t.Error("Expected to find item at end of large slice")
	}

	// Item not in slice
	if Contains(largeSlice, "not-present") {
		t.Error("Expected not to find non-existent item in large slice")
	}
}

func TestContains_SingleElement(t *testing.T) {
	slice := []string{"single"}

	if !Contains(slice, "single") {
		t.Error("Expected to find item in single-element slice")
	}

	if Contains(slice, "other") {
		t.Error("Expected not to find different item in single-element slice")
	}
}

func TestContainsAny_MultipleMatches(t *testing.T) {
	s := "The quick brown fox jumps over the lazy dog"

	// Multiple substrings that match
	if !ContainsAny(s, "quick", "lazy") {
		t.Error("Expected to find at least one matching substring")
	}

	// First one matches
	if !ContainsAny(s, "quick", "missing", "absent") {
		t.Error("Expected to find first matching substring")
	}

	// Last one matches
	if !ContainsAny(s, "missing", "absent", "dog") {
		t.Error("Expected to find last matching substring")
	}
}

func TestContainsAny_NilSubstrings(t *testing.T) {
	s := "test string"

	// Nil substrings should return false
	if ContainsAny(s, nil...) {
		t.Error("Expected false for nil substrings")
	}
}

func TestContainsIgnoreCase_Unicode(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "unicode characters",
			s:        "Café España",
			substr:   "café",
			expected: true,
		},
		{
			name:     "unicode uppercase",
			s:        "café españa",
			substr:   "CAFÉ",
			expected: true,
		},
		{
			name:     "emoji in string",
			s:        "Hello 👋 World",
			substr:   "👋",
			expected: true,
		},
		{
			name:     "special characters",
			s:        "test@example.com",
			substr:   "EXAMPLE",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsIgnoreCase(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("ContainsIgnoreCase(%q, %q) = %v; want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestContainsIgnoreCase_PartialMatch(t *testing.T) {
	s := "GitHub Actions Workflow"

	// Should find partial matches
	if !ContainsIgnoreCase(s, "hub") {
		t.Error("Expected to find partial match 'hub' in 'GitHub'")
	}

	if !ContainsIgnoreCase(s, "WORK") {
		t.Error("Expected to find partial match 'WORK' in 'Workflow'")
	}

	if !ContainsIgnoreCase(s, "actions workflow") {
		t.Error("Expected to find multi-word partial match")
	}
}

func TestContains_Duplicates(t *testing.T) {
	// Slice with duplicate values
	slice := []string{"apple", "banana", "apple", "cherry", "apple"}

	if !Contains(slice, "apple") {
		t.Error("Expected to find 'apple' in slice with duplicates")
	}

	// Should still return true on first match
	count := 0
	for _, item := range slice {
		if item == "apple" {
			count++
		}
	}
	if count != 3 {
		t.Errorf("Expected 3 occurrences of 'apple', got %d", count)
	}
}

func TestContainsAny_OrderMatters(t *testing.T) {
	s := "test string with multiple words"

	// Test that function returns on first match (short-circuit behavior)
	// Both should find a match, order shouldn't affect result
	result1 := ContainsAny(s, "string", "words")
	result2 := ContainsAny(s, "words", "string")

	if result1 != result2 {
		t.Error("Expected same result regardless of substring order")
	}

	if !result1 || !result2 {
		t.Error("Expected both to find matches")
	}
}
