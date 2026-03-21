//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHasNewlineInStringLiteral verifies that the function correctly detects newlines
// inside single-quoted GitHub Actions expression string literals.
func TestHasNewlineInStringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "no newlines at all",
			input:    "startsWith(github.event.comment.body, '/command ')",
			expected: false,
		},
		{
			name:     "newline outside any string literal",
			input:    "startsWith(body, '/command ')\n|| body == '/command'",
			expected: false,
		},
		{
			name:     "newline inside single-quoted string literal",
			input:    "startsWith(github.event.comment.body, '/command\n')",
			expected: true,
		},
		{
			name:     "multiple string literals, newline in one",
			input:    "startsWith(body, '/command ') || startsWith(body, '/command\n') || body == '/command'",
			expected: true,
		},
		{
			name:     "escaped single quote before newline",
			input:    "startsWith(body, '/it''s\n')",
			expected: true,
		},
		{
			name:     "escaped single quote, no newline inside",
			input:    "startsWith(body, '/it''s a test')",
			expected: false,
		},
		{
			name:     "newline right before closing quote",
			input:    "body == '/command\n'",
			expected: true,
		},
		{
			name:     "multi-condition with event_name and no newline in literal",
			input:    "(github.event_name == 'issue_comment') && (startsWith(body, '/command '))",
			expected: false,
		},
		{
			name:     "bot comment newline detection in realistic condition",
			input:    "((startsWith(github.event.comment.body, '/deploy ')) || (startsWith(github.event.comment.body, '/deploy\n')))",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasNewlineInStringLiteral(tt.input)
			assert.Equal(t, tt.expected, result, "hasNewlineInStringLiteral(%q)", tt.input)
		})
	}
}

// TestEscapeForYAMLDoubleQuoted verifies that the function correctly escapes strings
// for use inside YAML double-quoted scalars.
func TestEscapeForYAMLDoubleQuoted(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no special characters",
			input:    "startsWith(body, '/command')",
			expected: "startsWith(body, '/command')",
		},
		{
			name:     "newline character escaped",
			input:    "startsWith(body, '/command\n')",
			expected: `startsWith(body, '/command\n')`,
		},
		{
			name:     "double quote escaped",
			input:    `startsWith(body, "value")`,
			expected: `startsWith(body, \"value\")`,
		},
		{
			name:     "backslash escaped",
			input:    `path\to\file`,
			expected: `path\\to\\file`,
		},
		{
			name:     "carriage return escaped",
			input:    "value\rend",
			expected: `value\rend`,
		},
		{
			name:     "tab escaped",
			input:    "value\tend",
			expected: `value\tend`,
		},
		{
			name:     "combination of special characters",
			input:    "startsWith(body, '/command\n') || body == '/command'",
			expected: `startsWith(body, '/command\n') || body == '/command'`,
		},
		{
			name:     "realistic bot-comment condition",
			input:    "(startsWith(github.event.comment.body, '/deploy ')) || (startsWith(github.event.comment.body, '/deploy\n'))",
			expected: `(startsWith(github.event.comment.body, '/deploy ')) || (startsWith(github.event.comment.body, '/deploy\n'))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeForYAMLDoubleQuoted(tt.input)
			assert.Equal(t, tt.expected, result, "escapeForYAMLDoubleQuoted(%q)", tt.input)
		})
	}
}
