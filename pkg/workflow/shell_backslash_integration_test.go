//go:build integration
// +build integration

package workflow

import (
	"testing"
)

func TestSingleQuoteEscapingPreservesBackslashes(t *testing.T) {
	// Test that the shell escaping function preserves backslashes
	// This is a regression test for the typist workflow control character issue
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "grep with word boundaries",
			input:    "grep -r '\\bany\\b' pkg",
			expected: "'grep -r '\\''\\bany\\b'\\'' pkg'",
		},
		{
			name:     "echo with escape sequences",
			input:    "echo '\\n\\t'",
			expected: "'echo '\\''\\n\\t'\\'''",
		},
		{
			name:     "path with backslashes",
			input:    "path\\to\\file",
			expected: "'path\\to\\file'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := shellEscapeArg(tc.input)
			if result != tc.expected {
				t.Errorf("shellEscapeArg(%q) = %q, expected %q", tc.input, result, tc.expected)
			}

			// Verify no control characters in the result
			for i, ch := range result {
				if ch < 32 && ch != '\n' && ch != '\t' && ch != '\r' {
					t.Errorf("Found control character in result at position %d: %q (0x%02x)", i, ch, ch)
				}
			}
		})
	}
}
