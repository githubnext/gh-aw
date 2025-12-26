package stringutil

import "testing"

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
			if result != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q; want %q", tt.s, tt.maxLen, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("NormalizeWhitespace(%q) = %q; want %q", tt.content, result, tt.expected)
			}
		})
	}
}

func BenchmarkTruncate(b *testing.B) {
	s := "this is a very long string that needs to be truncated for testing purposes"
	for i := 0; i < b.N; i++ {
		Truncate(s, 30)
	}
}

func BenchmarkNormalizeWhitespace(b *testing.B) {
	content := "line1  \nline2\t\nline3   \t\nline4\n\n"
	for i := 0; i < b.N; i++ {
		NormalizeWhitespace(content)
	}
}
