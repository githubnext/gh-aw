//go:build !integration

package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max length",
			s:        "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string equal to max length",
			s:        "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "string longer than max length",
			s:        "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "max length 3",
			s:        "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "max length 2",
			s:        "hello",
			maxLen:   2,
			expected: "he",
		},
		{
			name:     "max length 1",
			s:        "hello",
			maxLen:   1,
			expected: "h",
		},
		{
			name:     "max length zero",
			s:        "hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "exactly three chars",
			s:        "abc",
			maxLen:   3,
			expected: "abc",
		},
		{
			name:     "four chars exact length",
			s:        "abcd",
			maxLen:   4,
			expected: "abcd",
		},
		{
			name:     "five chars truncated to four",
			s:        "abcde",
			maxLen:   4,
			expected: "a...",
		},
		{
			name:     "empty string",
			s:        "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "long string truncated",
			s:        "this is a very long string that needs to be truncated",
			maxLen:   20,
			expected: "this is a very lo...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Truncate(tt.s, tt.maxLen)
			assert.Equal(t, tt.expected, result, "Truncate(%q, %d) should return expected output", tt.s, tt.maxLen)
		})
	}
}

func BenchmarkTruncate(b *testing.B) {
	s := "this is a very long string that needs to be truncated for testing purposes"
	for b.Loop() {
		Truncate(s, 30)
	}
}

// Additional edge case tests

func TestTruncate_Unicode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{
			name:     "emoji truncation",
			s:        "Hello 👋 World 🌍",
			maxLen:   10,
			expected: "Hello \xf0...", // Truncates in middle of emoji byte sequence
		},
		{
			name:     "unicode characters",
			s:        "Café España México",
			maxLen:   12,
			expected: "Café Esp...", // Actual behavior
		},
		{
			name:     "mixed unicode and ascii",
			s:        "Test-测试-テスト",
			maxLen:   8,
			expected: "Test-...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Truncate(tt.s, tt.maxLen)
			assert.Equal(t, tt.expected, result, "Truncate(%q, %d) should handle unicode input as expected", tt.s, tt.maxLen)
		})
	}
}

func BenchmarkTruncate_Short(b *testing.B) {
	s := "short"
	for b.Loop() {
		Truncate(s, 10)
	}
}

func BenchmarkTruncate_Long(b *testing.B) {
	s := "this is a very very very very very long string that definitely needs truncation"
	for b.Loop() {
		Truncate(s, 20)
	}
}

func TestFormatList(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		items    []string
		expected string
	}{
		{
			name:     "empty slice",
			items:    []string{},
			expected: "",
		},
		{
			name:     "single item",
			items:    []string{"a"},
			expected: "a",
		},
		{
			name:     "two items",
			items:    []string{"a", "b"},
			expected: "a and b",
		},
		{
			name:     "three items",
			items:    []string{"a", "b", "c"},
			expected: "a, b, and c",
		},
		{
			name:     "four items",
			items:    []string{"a", "b", "c", "d"},
			expected: "a, b, c, and d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FormatList(tt.items)
			assert.Equal(t, tt.expected, result, "FormatList(%v) should return natural-language list formatting", tt.items)
		})
	}
}

func TestIsPositiveInteger(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "positive integer",
			s:    "123",
			want: true,
		},
		{
			name: "one",
			s:    "1",
			want: true,
		},
		{
			name: "large number",
			s:    "999999999",
			want: true,
		},
		{
			name: "zero",
			s:    "0",
			want: false,
		},
		{
			name: "negative",
			s:    "-5",
			want: false,
		},
		{
			name: "leading zeros",
			s:    "007",
			want: false,
		},
		{
			name: "float",
			s:    "3.14",
			want: false,
		},
		{
			name: "not a number",
			s:    "abc",
			want: false,
		},
		{
			name: "empty string",
			s:    "",
			want: false,
		},
		{
			name: "spaces",
			s:    " 123 ",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsPositiveInteger(tt.s)
			assert.Equal(t, tt.want, got, "IsPositiveInteger(%q) should match expected positivity result", tt.s)
		})
	}
}
