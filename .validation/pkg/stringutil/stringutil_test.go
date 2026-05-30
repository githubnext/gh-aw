//go:build !integration

package stringutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
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
			result := Truncate(tt.s, tt.maxLen)
			assert.Equal(t, tt.expected, result, "Truncate(%q, %d) should return expected output", tt.s, tt.maxLen)
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "no trailing whitespace",
			content:  "hello\nworld",
			expected: "hello\nworld\n",
		},
		{
			name:     "trailing spaces on lines",
			content:  "hello  \nworld  ",
			expected: "hello\nworld\n",
		},
		{
			name:     "trailing tabs on lines",
			content:  "hello\t\nworld\t",
			expected: "hello\nworld\n",
		},
		{
			name:     "multiple trailing newlines",
			content:  "hello\nworld\n\n\n",
			expected: "hello\nworld\n",
		},
		{
			name:     "empty string",
			content:  "",
			expected: "",
		},
		{
			name:     "single newline",
			content:  "\n",
			expected: "",
		},
		{
			name:     "mixed whitespace",
			content:  "hello  \t\nworld \t \n\n",
			expected: "hello\nworld\n",
		},
		{
			name:     "content with no newline",
			content:  "hello world",
			expected: "hello world\n",
		},
		{
			name:     "content already normalized",
			content:  "hello\nworld\n",
			expected: "hello\nworld\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWhitespace(tt.content)
			assert.Equal(t, tt.expected, result, "NormalizeWhitespace(%q) should normalize trailing whitespace and newlines", tt.content)
		})
	}
}

func BenchmarkTruncate(b *testing.B) {
	s := "this is a very long string that needs to be truncated for testing purposes"
	for b.Loop() {
		Truncate(s, 30)
	}
}

func BenchmarkNormalizeWhitespace(b *testing.B) {
	content := "line1  \nline2\t\nline3   \t\nline4\n\n"
	for b.Loop() {
		NormalizeWhitespace(content)
	}
}

// Additional edge case tests

func TestTruncate_Unicode(t *testing.T) {
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
			result := Truncate(tt.s, tt.maxLen)
			assert.Equal(t, tt.expected, result, "Truncate(%q, %d) should handle unicode input as expected", tt.s, tt.maxLen)
		})
	}
}

func TestNormalizeWhitespace_OnlyWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "only spaces",
			content:  "   ",
			expected: "", // After trimming trailing spaces and newlines, becomes empty
		},
		{
			name:     "only tabs",
			content:  "\t\t\t",
			expected: "", // After trimming trailing tabs and newlines, becomes empty
		},
		{
			name:     "mixed spaces and tabs",
			content:  "  \t  \t",
			expected: "", // After trimming, becomes empty
		},
		{
			name:     "only newlines",
			content:  "\n\n\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWhitespace(tt.content)
			assert.Equal(t, tt.expected, result, "NormalizeWhitespace(%q) should handle whitespace-only input", tt.content)
		})
	}
}

func TestNormalizeWhitespace_ManyLines(t *testing.T) {
	// Test with many lines
	lines := make([]string, 100)
	for i := range 100 {
		lines[i] = "line with trailing spaces  "
	}
	var content strings.Builder
	for _, line := range lines {
		content.WriteString(line + "\n")
	}

	result := NormalizeWhitespace(content.String())

	// Check that all trailing spaces are removed
	expectedLines := make([]string, 100)
	for i := range 100 {
		expectedLines[i] = "line with trailing spaces"
	}
	var expected strings.Builder
	for _, line := range expectedLines {
		expected.WriteString(line + "\n")
	}

	assert.Equal(t, expected.String(), result, "NormalizeWhitespace should remove trailing spaces from all lines in large inputs")
}

func TestNormalizeWhitespace_PreservesContent(t *testing.T) {
	// Ensure that non-trailing whitespace is preserved
	content := "line1  middle  spaces\nline2\t\tmiddle\t\ttabs\n"
	result := NormalizeWhitespace(content)

	assert.Contains(t, result, "middle  spaces", "NormalizeWhitespace should preserve non-trailing spaces inside lines")
	assert.Contains(t, result, "middle\t\ttabs", "NormalizeWhitespace should preserve non-trailing tabs inside lines")
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

func BenchmarkNormalizeWhitespace_NoChange(b *testing.B) {
	content := "line1\nline2\nline3\n"
	for b.Loop() {
		NormalizeWhitespace(content)
	}
}

func BenchmarkNormalizeWhitespace_ManyChanges(b *testing.B) {
	content := "line1  \t  \nline2  \t  \nline3  \t  \n\n\n"
	for b.Loop() {
		NormalizeWhitespace(content)
	}
}

func TestParseVersionValue(t *testing.T) {
	tests := []struct {
		name     string
		version  any
		expected string
	}{
		// String versions
		{
			name:     "string version",
			version:  "v1.2.3",
			expected: "v1.2.3",
		},
		{
			name:     "numeric string",
			version:  "123",
			expected: "123",
		},
		{
			name:     "empty string",
			version:  "",
			expected: "",
		},
		// Integer versions
		{
			name:     "int version",
			version:  42,
			expected: "42",
		},
		{
			name:     "int64 version",
			version:  int64(100),
			expected: "100",
		},
		{
			name:     "uint64 version",
			version:  uint64(999),
			expected: "999",
		},
		// Float versions
		{
			name:     "float64 simple",
			version:  float64(1.5),
			expected: "1.5",
		},
		{
			name:     "float64 whole number",
			version:  float64(2.0),
			expected: "2",
		},
		{
			name:     "float64 with precision",
			version:  float64(1.234),
			expected: "1.234",
		},
		// Unsupported types
		{
			name:     "nil",
			version:  nil,
			expected: "",
		},
		{
			name:     "bool",
			version:  true,
			expected: "",
		},
		{
			name:     "slice",
			version:  []string{"1", "2"},
			expected: "",
		},
		{
			name:     "map",
			version:  map[string]string{"version": "1.0"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseVersionValue(tt.version)
			assert.Equal(t, tt.expected, result, "ParseVersionValue(%v) should return normalized string representation", tt.version)
		})
	}
}

func TestFormatList(t *testing.T) {
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
			result := FormatList(tt.items)
			assert.Equal(t, tt.expected, result, "FormatList(%v) should return natural-language list formatting", tt.items)
		})
	}
}

func TestNormalizeLeadingWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes consistent leading spaces",
			input:    "          Line 1\n          Line 2\n          Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "handles no leading spaces",
			input:    "Line 1\nLine 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "preserves relative indentation",
			input:    "          Line 1\n            Indented Line 2\n          Line 3",
			expected: "Line 1\n  Indented Line 2\nLine 3",
		},
		{
			name:     "handles empty lines",
			input:    "          Line 1\n\n          Line 3",
			expected: "Line 1\n\nLine 3",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "removes consistent leading tabs",
			input:    "\t\tLine 1\n\t\tLine 2\n\t\tLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "removes consistent mixed tab and space indentation",
			input:    "\t  Line 1\n\t  Line 2\n\t  Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLeadingWhitespace(tt.input)
			assert.Equal(t, tt.expected, result, "NormalizeLeadingWhitespace should normalize indentation for case %q", tt.name)
		})
	}
}

func TestIsPositiveInteger(t *testing.T) {
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
			got := IsPositiveInteger(tt.s)
			assert.Equal(t, tt.want, got, "IsPositiveInteger(%q) should match expected positivity result", tt.s)
		})
	}
}
