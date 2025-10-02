package workflow

import (
	"testing"
)

func TestUnquoteYAMLKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		key      string
		expected string
	}{
		{
			name: "unquote 'on' at start of line",
			input: `"on":
  issues:
    types:
    - opened`,
			key: "on",
			expected: `on:
  issues:
    types:
    - opened`,
		},
		{
			name: "unquote 'on' with indentation",
			input: `  "on":
    issues:
      types:
      - opened`,
			key: "on",
			expected: `  on:
    issues:
      types:
      - opened`,
		},
		{
			name: "do not unquote 'on' in middle of line",
			input: `key: "on":value`,
			key: "on",
			expected: `key: "on":value`,
		},
		{
			name: "do not unquote 'on' in string value",
			input: `description: "This is about on: something"`,
			key: "on",
			expected: `description: "This is about on: something"`,
		},
		{
			name: "unquote multiple occurrences at start of lines",
			input: `"on":
  issues:
    types:
    - opened
"on":
  push:
    branches:
    - main`,
			key: "on",
			expected: `on:
  issues:
    types:
    - opened
on:
  push:
    branches:
    - main`,
		},
		{
			name: "unquote other keys",
			input: `"if":
  github.actor == 'bot'`,
			key: "if",
			expected: `if:
  github.actor == 'bot'`,
		},
		{
			name: "handle key with special regex characters",
			input: `"key.with.dots":
  value: test`,
			key: "key.with.dots",
			expected: `key.with.dots:
  value: test`,
		},
		{
			name: "no change when key is not quoted",
			input: `on:
  issues:
    types:
    - opened`,
			key: "on",
			expected: `on:
  issues:
    types:
    - opened`,
		},
		{
			name: "unquote with tabs",
			input: `		"on":
		  issues:`,
			key: "on",
			expected: `		on:
		  issues:`,
		},
		{
			name: "empty string",
			input:    "",
			key:      "on",
			expected: "",
		},
		{
			name: "only newlines",
			input:    "\n\n\n",
			key:      "on",
			expected: "\n\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unquoteYAMLKey(tt.input, tt.key)
			if result != tt.expected {
				t.Errorf("unquoteYAMLKey() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s",
					tt.input, tt.expected, result)
			}
		})
	}
}
