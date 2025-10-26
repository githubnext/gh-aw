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
		// Additional edge cases
		{
			name:     "argument with curly braces",
			input:    "file{1,2,3}.txt",
			expected: "'file{1,2,3}.txt'",
		},
		{
			name:     "argument with question mark",
			input:    "file?.txt",
			expected: "'file?.txt'",
		},
		{
			name:     "argument with pipe",
			input:    "cmd1|cmd2",
			expected: "'cmd1|cmd2'",
		},
		{
			name:     "argument with ampersand",
			input:    "cmd1&cmd2",
			expected: "'cmd1&cmd2'",
		},
		{
			name:     "argument with semicolon",
			input:    "cmd1;cmd2",
			expected: "'cmd1;cmd2'",
		},
		{
			name:     "argument with less than",
			input:    "input<file",
			expected: "'input<file'",
		},
		{
			name:     "argument with greater than",
			input:    "output>file",
			expected: "'output>file'",
		},
		{
			name:     "argument with backslash",
			input:    "path\\to\\file",
			expected: "'path\\to\\file'",
		},
		{
			name:     "argument with backtick",
			input:    "cmd`date`",
			expected: "'cmd`date`'",
		},
		{
			name:     "argument with tab",
			input:    "hello\tworld",
			expected: "'hello\tworld'",
		},
		{
			name:     "argument with newline",
			input:    "hello\nworld",
			expected: "'hello\nworld'",
		},
		{
			name:     "multiple single quotes",
			input:    "it's can't won't",
			expected: "'it'\"'\"'s can'\"'\"'t won'\"'\"'t'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single character flag",
			input:    "-v",
			expected: "-v",
		},
		{
			name:     "path without special characters",
			input:    "/usr/bin/env",
			expected: "/usr/bin/env",
		},
		{
			name:     "path with spaces",
			input:    "/path/with spaces/file",
			expected: "'/path/with spaces/file'",
		},
		{
			name:     "command substitution pattern",
			input:    "$(date)",
			expected: "'$(date)'",
		},
		{
			name:     "variable expansion pattern",
			input:    "${VAR}",
			expected: "'${VAR}'",
		},
		{
			name:     "already quoted with extra chars should be escaped",
			input:    "\"test\"extra",
			expected: "'\"test\"extra'",
		},
		{
			name:     "single quote at start only",
			input:    "'incomplete",
			expected: "''\"'\"'incomplete'",
		},
		{
			name:     "single quote at end only",
			input:    "incomplete'",
			expected: "'incomplete'\"'\"''",
		},
		{
			name:     "double quote at start only",
			input:    "\"incomplete",
			expected: "'\"incomplete'",
		},
		{
			name:     "double quote at end only",
			input:    "incomplete\"",
			expected: "'incomplete\"'",
		},
		{
			name:     "only single quotes",
			input:    "''",
			expected: "''",
		},
		{
			name:     "nested parentheses",
			input:    "func((arg))",
			expected: "'func((arg))'",
		},
		{
			name:     "mixed brackets and parentheses",
			input:    "pattern[a-z](test)",
			expected: "'pattern[a-z](test)'",
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
			input:    []string{"copilot", "--add-dir", "/tmp/gh-aw/", "--allow-tool", "shell(*.txt)"},
			expected: "copilot --add-dir /tmp/gh-aw/ --allow-tool 'shell(*.txt)'",
		},
		{
			name:     "prompt with pre-quoted instruction should not be escaped",
			input:    []string{"copilot", "--add-dir", "/tmp/gh-aw/", "--prompt", "\"$INSTRUCTION\""},
			expected: "copilot --add-dir /tmp/gh-aw/ --prompt \"$INSTRUCTION\"",
		},
		// Additional edge cases
		{
			name:     "empty array",
			input:    []string{},
			expected: "",
		},
		{
			name:     "single argument",
			input:    []string{"command"},
			expected: "command",
		},
		{
			name:     "single argument with spaces",
			input:    []string{"hello world"},
			expected: "'hello world'",
		},
		{
			name:     "multiple arguments with spaces",
			input:    []string{"echo", "hello world", "foo bar"},
			expected: "echo 'hello world' 'foo bar'",
		},
		{
			name:     "arguments with various quote types",
			input:    []string{"cmd", "\"quoted\"", "'single'", "mixed\"quote"},
			expected: "cmd \"quoted\" 'single' 'mixed\"quote'",
		},
		{
			name:     "arguments with command substitution patterns",
			input:    []string{"echo", "$(date)", "`whoami`"},
			expected: "echo '$(date)' '`whoami`'",
		},
		{
			name:     "long command with many arguments",
			input:    []string{"npx", "@github/copilot", "--allow-tool", "shell(cat)", "--allow-tool", "shell(grep)", "--allow-tool", "shell(sed)", "--log-level", "debug"},
			expected: "npx @github/copilot --allow-tool 'shell(cat)' --allow-tool 'shell(grep)' --allow-tool 'shell(sed)' --log-level debug",
		},
		{
			name:     "arguments with special shell operators",
			input:    []string{"cmd", "arg1|arg2", "arg3&arg4", "arg5;arg6"},
			expected: "cmd 'arg1|arg2' 'arg3&arg4' 'arg5;arg6'",
		},
		{
			name:     "arguments with wildcards",
			input:    []string{"ls", "*.txt", "file?.doc", "test[0-9].log"},
			expected: "ls '*.txt' 'file?.doc' 'test[0-9].log'",
		},
		{
			name:     "mixed flags and values",
			input:    []string{"-v", "--verbose", "-f", "file name.txt", "--output", "result.log"},
			expected: "-v --verbose -f 'file name.txt' --output result.log",
		},
		{
			name:     "arguments with dollar signs",
			input:    []string{"echo", "$HOME", "$USER", "$PATH"},
			expected: "echo '$HOME' '$USER' '$PATH'",
		},
		{
			name:     "pre-quoted arguments mixed with unquoted",
			input:    []string{"cmd", "\"arg1\"", "arg2", "'arg3'", "arg 4"},
			expected: "cmd \"arg1\" arg2 'arg3' 'arg 4'",
		},
		{
			name:     "arguments with backslashes",
			input:    []string{"echo", "\\n", "\\t", "path\\to\\file"},
			expected: "echo '\\n' '\\t' 'path\\to\\file'",
		},
		{
			name:     "arguments with parentheses and brackets",
			input:    []string{"tool", "func(arg)", "pattern[a-z]", "test{1,2}"},
			expected: "tool 'func(arg)' 'pattern[a-z]' 'test{1,2}'",
		},
		{
			name:     "empty string arguments",
			input:    []string{"cmd", "", "arg"},
			expected: "cmd  arg",
		},
		{
			name:     "arguments with newlines and tabs",
			input:    []string{"echo", "line1\nline2", "col1\tcol2"},
			expected: "echo 'line1\nline2' 'col1\tcol2'",
		},
		{
			name:     "single quotes in multiple arguments",
			input:    []string{"echo", "it's", "can't", "won't"},
			expected: "echo 'it'\"'\"'s' 'can'\"'\"'t' 'won'\"'\"'t'",
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

func TestShellEscapeCommandString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple command without special characters",
			input:    "echo hello",
			expected: "\"echo hello\"",
		},
		{
			name:     "command with single-quoted arguments",
			input:    "npx --allow-tool 'shell(cat)' --allow-tool 'shell(ls)'",
			expected: "\"npx --allow-tool 'shell\\(cat\\)' --allow-tool 'shell\\(ls\\)'\"",
		},
		{
			name:     "command with double quotes",
			input:    "echo \"hello world\"",
			expected: "\"echo \\\"hello world\\\"\"",
		},
		{
			name:     "command with dollar sign (command substitution)",
			input:    "echo $(date)",
			expected: "\"echo \\$(date)\"",
		},
		{
			name:     "command with backticks",
			input:    "echo `date`",
			expected: "\"echo \\`date\\`\"",
		},
		{
			name:     "command with backslashes",
			input:    "echo \\n\\t",
			expected: "\"echo \\\\n\\\\t\"",
		},
		{
			name:     "complex copilot command",
			input:    "npx -y @github/copilot@0.0.351 --allow-tool 'github(list_workflows)' --prompt \"$(cat /tmp/prompt.txt)\"",
			expected: "\"npx -y @github/copilot@0.0.351 --allow-tool 'github\\(list_workflows\\)' --prompt \\\"\\$(cat /tmp/prompt.txt)\\\"\"",
		},
		// Command substitution edge cases
		{
			name:     "nested command substitution",
			input:    "echo $(echo $(date))",
			expected: "\"echo \\$(echo \\$(date))\"",
		},
		{
			name:     "multiple command substitutions",
			input:    "echo $(date) and $(whoami)",
			expected: "\"echo \\$(date) and \\$(whoami)\"",
		},
		{
			name:     "command substitution with pipes",
			input:    "echo $(cat file | grep pattern)",
			expected: "\"echo \\$(cat file | grep pattern)\"",
		},
		{
			name:     "command substitution with parentheses in arguments",
			input:    "echo $(grep 'pattern(test)' file)",
			expected: "\"echo \\$(grep 'pattern\\(test\\)' file)\"",
		},
		{
			name:     "command substitution at the end",
			input:    "prompt \"$(cat /tmp/file)\"",
			expected: "\"prompt \\\"\\$(cat /tmp/file)\\\"\"",
		},
		// Regular parentheses (not command substitution)
		{
			name:     "standalone parentheses should be escaped",
			input:    "echo (test)",
			expected: "\"echo \\(test\\)\"",
		},
		{
			name:     "multiple levels of parentheses",
			input:    "echo ((test))",
			expected: "\"echo \\(\\(test\\)\\)\"",
		},
		{
			name:     "parentheses in quoted strings",
			input:    "echo 'test(ing)' and \"more(stuff)\"",
			expected: "\"echo 'test\\(ing\\)' and \\\"more\\(stuff\\)\\\"\"",
		},
		// Mixed scenarios
		{
			name:     "command substitution and regular parentheses",
			input:    "echo $(date) and func(arg) and $(cat file)",
			expected: "\"echo \\$(date) and func\\(arg\\) and \\$(cat file)\"",
		},
		{
			name:     "parentheses before dollar sign (not command substitution)",
			input:    "echo (prefix)$VAR",
			expected: "\"echo \\(prefix\\)\\$VAR\"",
		},
		// Backslash edge cases
		{
			name:     "existing backslash before parenthesis",
			input:    "echo \\(already escaped\\)",
			expected: "\"echo \\\\\\(already escaped\\\\\\)\"",
		},
		{
			name:     "backslash before dollar sign",
			input:    "echo \\$HOME",
			expected: "\"echo \\\\\\$HOME\"",
		},
		// Dollar sign edge cases
		{
			name:     "dollar sign without parentheses (variable)",
			input:    "echo $HOME",
			expected: "\"echo \\$HOME\"",
		},
		{
			name:     "dollar sign with braces",
			input:    "echo ${HOME}",
			expected: "\"echo \\${HOME}\"",
		},
		{
			name:     "multiple dollar signs",
			input:    "echo $VAR1 $VAR2 $(cmd)",
			expected: "\"echo \\$VAR1 \\$VAR2 \\$(cmd)\"",
		},
		// Quote edge cases
		{
			name:     "mixed quotes",
			input:    "echo \"test\" 'more' $(cmd)",
			expected: "\"echo \\\"test\\\" 'more' \\$(cmd)\"",
		},
		{
			name:     "escaped quotes in input",
			input:    "echo \\\"already escaped\\\"",
			expected: "\"echo \\\\\\\"already escaped\\\\\\\"\"",
		},
		// Real-world copilot scenarios
		{
			name:     "copilot with multiple shell tools and command substitution",
			input:    "npx @github/copilot --allow-tool 'shell(cat)' --allow-tool 'shell(grep)' --allow-tool 'shell(sed)' --prompt \"$(cat /tmp/prompt.txt)\"",
			expected: "\"npx @github/copilot --allow-tool 'shell\\(cat\\)' --allow-tool 'shell\\(grep\\)' --allow-tool 'shell\\(sed\\)' --prompt \\\"\\$(cat /tmp/prompt.txt)\\\"\"",
		},
		{
			name:     "copilot with environment variable and command substitution",
			input:    "npx @github/copilot --log-dir $LOG_DIR --prompt \"$(cat $PROMPT_FILE)\"",
			expected: "\"npx @github/copilot --log-dir \\$LOG_DIR --prompt \\\"\\$(cat \\$PROMPT_FILE)\\\"\"",
		},
		// Empty and whitespace
		{
			name:     "empty string",
			input:    "",
			expected: "\"\"",
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: "\"   \"",
		},
		// Special characters combination
		{
			name:     "all special characters",
			input:    "cmd \"quoted\" 'single' $VAR $(subst) `backtick` \\backslash (paren)",
			expected: "\"cmd \\\"quoted\\\" 'single' \\$VAR \\$(subst) \\`backtick\\` \\\\backslash \\(paren\\)\"",
		},
		// Edge case: $ at end of string
		{
			name:     "dollar sign at end",
			input:    "echo test$",
			expected: "\"echo test\\$\"",
		},
		// Edge case: Opening paren at start
		{
			name:     "opening paren at start",
			input:    "(command)",
			expected: "\"\\(command\\)\"",
		},
		// Edge case: Command substitution with escaped characters inside
		{
			name:     "command substitution with special chars inside",
			input:    "echo $(echo \"test\" | grep 'pattern')",
			expected: "\"echo \\$(echo \\\"test\\\" | grep 'pattern')\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellEscapeCommandString(tt.input)
			if result != tt.expected {
				t.Errorf("shellEscapeCommandString(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
