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
