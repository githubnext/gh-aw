package workflow

import "testing"

func TestShellEscapeArg(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple argument without special characters",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "argument with parentheses",
			input:    "shell(git add:*)",
			expected: "'shell(git add:*)'",
		},
		{
			name:     "argument with brackets",
			input:    "pattern[abc]",
			expected: "'pattern[abc]'",
		},
		{
			name:     "argument with spaces",
			input:    "hello world",
			expected: "'hello world'",
		},
		{
			name:     "argument with single quote",
			input:    "don't",
			expected: "'don'\"'\"'t'",
		},
		{
			name:     "argument with asterisk",
			input:    "*.txt",
			expected: "'*.txt'",
		},
		{
			name:     "argument with dollar sign",
			input:    "$HOME",
			expected: "'$HOME'",
		},
		{
			name:     "simple flag",
			input:    "--allow-tool",
			expected: "--allow-tool",
		},
		{
			name:     "already double-quoted argument should not be escaped",
			input:    "\"$INSTRUCTION\"",
			expected: "\"$INSTRUCTION\"",
		},
		{
			name:     "already single-quoted argument should not be escaped",
			input:    "'hello world'",
			expected: "'hello world'",
		},
		{
			name:     "partial double quote should be escaped",
			input:    "hello\"world",
			expected: "'hello\"world'",
		},
		{
			name:     "empty double quotes should not be escaped",
			input:    "\"\"",
			expected: "\"\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellEscapeArg(tt.input)
			if result != tt.expected {
				t.Errorf("shellEscapeArg(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShellJoinArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "simple arguments",
			input:    []string{"git", "status"},
			expected: "git status",
		},
		{
			name:     "arguments with special characters",
			input:    []string{"--allow-tool", "shell(git add:*)", "--allow-tool", "shell(git commit:*)"},
			expected: "--allow-tool 'shell(git add:*)' --allow-tool 'shell(git commit:*)'",
		},
		{
			name:     "mixed arguments",
			input:    []string{"copilot", "--add-dir", "/tmp/", "--allow-tool", "shell(*.txt)"},
			expected: "copilot --add-dir /tmp/ --allow-tool 'shell(*.txt)'",
		},
		{
			name:     "prompt with pre-quoted instruction should not be escaped",
			input:    []string{"copilot", "--add-dir", "/tmp/", "--prompt", "\"$INSTRUCTION\""},
			expected: "copilot --add-dir /tmp/ --prompt \"$INSTRUCTION\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellJoinArgs(tt.input)
			if result != tt.expected {
				t.Errorf("shellJoinArgs(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
