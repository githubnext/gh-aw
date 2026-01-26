package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string without special chars",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "string with space",
			input:    "hello world",
			expected: "'hello world'",
		},
		{
			name:     "string with single quote",
			input:    "it's",
			expected: "'it'\\''s'",
		},
		{
			name:     "string with double quote",
			input:    "hello \"world\"",
			expected: "'hello \"world\"'",
		},
		{
			name:     "string with dollar sign",
			input:    "$PATH",
			expected: "'$PATH'",
		},
		{
			name:     "string with backtick",
			input:    "`command`",
			expected: "'`command`'",
		},
		{
			name:     "string with backslash",
			input:    "path\\to\\file",
			expected: "'path\\to\\file'",
		},
		{
			name:     "string with tab",
			input:    "hello\tworld",
			expected: "'hello\tworld'",
		},
		{
			name:     "string with newline",
			input:    "hello\nworld",
			expected: "'hello\nworld'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "complex string with multiple special chars",
			input:    "echo 'hello' \"world\" $VAR `cmd`",
			expected: "'echo '\\''hello'\\'' \"world\" $VAR `cmd`'",
		},
		{
			name:     "command injection attempt with semicolon",
			input:    "file; rm -rf /",
			expected: "'file; rm -rf /'",
		},
		{
			name:     "command injection attempt with pipe",
			input:    "file | cat /etc/passwd",
			expected: "'file | cat /etc/passwd'",
		},
		{
			name:     "multiple single quotes",
			input:    "it's a test's file",
			expected: "'it'\\''s a test'\\''s file'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellQuote(tt.input)
			assert.Equal(t, tt.expected, result, "Shell quote result should match expected")
		})
	}
}

func TestBuildDockerCommandWithExpandableVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple command without GITHUB_WORKSPACE",
			input:    "docker run hello",
			expected: "'docker run hello'",
		},
		{
			name:     "command with single GITHUB_WORKSPACE",
			input:    "docker run -v ${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}",
			expected: "'docker run -v '\"${GITHUB_WORKSPACE}\"':'\"${GITHUB_WORKSPACE}\"''",
		},
		{
			name:     "command with GITHUB_WORKSPACE at the start",
			input:    "${GITHUB_WORKSPACE}/file",
			expected: "''\"${GITHUB_WORKSPACE}\"'/file'",
		},
		{
			name:     "command with GITHUB_WORKSPACE at the end",
			input:    "path/to/${GITHUB_WORKSPACE}",
			expected: "'path/to/'\"${GITHUB_WORKSPACE}\"''",
		},
		{
			name:     "command with multiple GITHUB_WORKSPACE references",
			input:    "${GITHUB_WORKSPACE}/src:${GITHUB_WORKSPACE}/dst",
			expected: "''\"${GITHUB_WORKSPACE}\"'/src:'\"${GITHUB_WORKSPACE}\"'/dst'",
		},
		{
			name:     "command with GITHUB_WORKSPACE and single quote",
			input:    "it's in ${GITHUB_WORKSPACE}",
			expected: "'it'\\''s in '\"${GITHUB_WORKSPACE}\"''",
		},
		{
			name:     "complex docker command",
			input:    "docker run -v ${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:rw image",
			expected: "'docker run -v '\"${GITHUB_WORKSPACE}\"':'\"${GITHUB_WORKSPACE}\"':rw image'",
		},
		{
			name:     "command with spaces and no GITHUB_WORKSPACE",
			input:    "docker run hello world",
			expected: "'docker run hello world'",
		},
		{
			name:     "empty command",
			input:    "",
			expected: "",
		},
		{
			name:     "injection attempt in GITHUB_WORKSPACE context",
			input:    "${GITHUB_WORKSPACE}; rm -rf /",
			expected: "''\"${GITHUB_WORKSPACE}\"'; rm -rf /'",
		},
		{
			name:     "multiple variables mixed with GITHUB_WORKSPACE",
			input:    "${GITHUB_WORKSPACE}/src ${OTHER_VAR}/dst",
			expected: "''\"${GITHUB_WORKSPACE}\"'/src ${OTHER_VAR}/dst'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDockerCommandWithExpandableVars(tt.input)
			assert.Equal(t, tt.expected, result, "Docker command with expandable vars should match expected")
		})
	}
}

func TestBuildDockerCommandWithExpandableVars_PreservesVariableExpansion(t *testing.T) {
	// Test that ${GITHUB_WORKSPACE} is properly preserved for shell expansion
	input := "docker run -v ${GITHUB_WORKSPACE}:/workspace"
	result := buildDockerCommandWithExpandableVars(input)

	// Use require.* for the first critical check (fail-fast)
	require.Contains(t, result, "${GITHUB_WORKSPACE}", "Result should preserve GITHUB_WORKSPACE variable for expansion")

	// Use assert.* for the second check (still runs if we get here)
	assert.Contains(t, result, "\"${GITHUB_WORKSPACE}\"", "GITHUB_WORKSPACE should be in double quotes for safe expansion")
}

func TestBuildDockerCommandWithExpandableVars_UnbracedVariable(t *testing.T) {
	// Test that $GITHUB_WORKSPACE (without braces) is handled
	// The current implementation only handles ${GITHUB_WORKSPACE} (with braces)
	// and treats $GITHUB_WORKSPACE as a regular shell character that gets quoted
	input := "docker run -v $GITHUB_WORKSPACE:/workspace"
	result := buildDockerCommandWithExpandableVars(input)

	// Document current behavior: unbraced $GITHUB_WORKSPACE is quoted normally
	assert.Equal(t, "'docker run -v $GITHUB_WORKSPACE:/workspace'", result,
		"Unbraced $GITHUB_WORKSPACE should be quoted normally (not preserved for expansion)")
}
