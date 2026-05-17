//go:build !integration

package stringutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertSanitizeResult(t *testing.T, functionName, input, got, want string) {
	t.Helper()
	require.Equal(t, want, got, "%s(%q) should return expected output", functionName, input)
}

func assertSanitizeResultWithContext(t *testing.T, functionName, context, got, want string) {
	t.Helper()
	require.Equal(t, want, got, "%s(%s) should return expected output", functionName, context)
}

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
		{
			name:     "message with no secrets",
			message:  "This is a regular error message",
			expected: "This is a regular error message",
		},
		{
			name:     "message with snake_case secret",
			message:  "Error accessing MY_SECRET_KEY",
			expected: "Error accessing [REDACTED]",
		},
		{
			name:     "message with multiple secrets",
			message:  "Failed to use API_TOKEN and DATABASE_PASSWORD",
			expected: "Failed to use [REDACTED] and [REDACTED]",
		},
		{
			name:     "message with PascalCase secret",
			message:  "Invalid GitHubToken provided",
			expected: "Invalid [REDACTED] provided",
		},
		{
			name:     "message with workflow keyword (not redacted)",
			message:  "Error in GITHUB_ACTIONS workflow",
			expected: "Error in [REDACTED] workflow",
		},
		{
			name:     "message with GITHUB keyword (not redacted)",
			message:  "GITHUB is not responding",
			expected: "GITHUB is not responding",
		},
		{
			name:     "message with PATH keyword (not redacted)",
			message:  "PATH variable is not set",
			expected: "PATH variable is not set",
		},
		{
			name:     "complex message with mixed secrets",
			message:  "Failed to authenticate with DEPLOY_KEY and ApiSecret",
			expected: "Failed to authenticate with [REDACTED] and [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.message)
			assertSanitizeResult(t, "SanitizeErrorMessage", tt.message, result, tt.expected)
		})
	}
}

func BenchmarkSanitizeErrorMessage(b *testing.B) {
	message := "Failed to use API_TOKEN and DATABASE_PASSWORD with GitHubToken"
	for b.Loop() {
		SanitizeErrorMessage(message)
	}
}

// Additional edge case tests

func TestSanitizeErrorMessage_AllWorkflowKeywords(t *testing.T) {
	// Test all common workflow keywords that should NOT be redacted
	keywords := []string{
		"GITHUB", "ACTIONS", "WORKFLOW", "RUNNER", "JOB", "STEP",
		"MATRIX", "ENV", "PATH", "HOME", "SHELL", "INPUTS", "OUTPUTS",
		"NEEDS", "STRATEGY", "CONCURRENCY", "IF", "WITH", "USES", "RUN",
		"WORKING_DIRECTORY", "CONTINUE_ON_ERROR", "TIMEOUT_MINUTES",
	}

	for _, keyword := range keywords {
		message := "Error with " + keyword + " configuration"
		result := SanitizeErrorMessage(message)
		assert.Contains(t, result, keyword, "Workflow keyword %q should not be redacted", keyword)
	}
}

func TestSanitizeErrorMessage_MultipleOccurrences(t *testing.T) {
	message := "MY_SECRET is used twice: MY_SECRET here and MY_SECRET there"
	result := SanitizeErrorMessage(message)
	expected := "[REDACTED] is used twice: [REDACTED] here and [REDACTED] there"

	assertSanitizeResult(t, "SanitizeErrorMessage", message, result, expected)
}

func TestSanitizeErrorMessage_MixedCase(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "lowercase not matched",
			message:  "error with my_secret_key",
			expected: "error with my_secret_key",
		},
		{
			name:     "mixed case not matched",
			message:  "error with My_Secret_Key",
			expected: "error with My_Secret_Key",
		},
		{
			name:     "all uppercase matched",
			message:  "error with MY_SECRET_KEY",
			expected: "error with [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.message)
			assertSanitizeResult(t, "SanitizeErrorMessage", tt.message, result, tt.expected)
		})
	}
}

func TestSanitizeErrorMessage_PascalCaseVariants(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		shouldRedact bool
	}{
		{"Token suffix", "Invalid GitHubToken", true},
		{"Key suffix", "Missing ApiKey", true},
		{"Secret suffix", "Bad DeploySecret", true},
		{"Password suffix", "Wrong DatabasePassword", true},
		{"Credential suffix", "Invalid AwsCredential", true},
		{"Auth suffix", "Failed BasicAuth", true},
		{"No suffix", "Invalid GitHubActions", false},
		{"lowercase", "Invalid githubtoken", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.message)
			containsRedacted := strings.Contains(result, "[REDACTED]")
			assert.Equal(t, tt.shouldRedact, containsRedacted, "SanitizeErrorMessage(%q) redaction state should match expectation", tt.message)
		})
	}
}

func TestSanitizeErrorMessage_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "very long message",
			message:  "Error: " + strings.Repeat("MY_SECRET_KEY ", 100),
			expected: "Error: " + strings.Repeat("[REDACTED] ", 100),
		},
		{
			name:     "only secrets",
			message:  "API_KEY DATABASE_PASSWORD GitHubToken",
			expected: "[REDACTED] [REDACTED] [REDACTED]",
		},
		{
			name:     "secrets at start and end",
			message:  "MY_API_KEY in the middle DATABASE_SECRET",
			expected: "[REDACTED] in the middle [REDACTED]",
		},
		{
			name:     "secret with numbers",
			message:  "Error with API_KEY_V2 and SECRET_123",
			expected: "Error with [REDACTED] and [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.message)
			assertSanitizeResult(t, "SanitizeErrorMessage", tt.message, result, tt.expected)
		})
	}
}

func TestSanitizeErrorMessage_GhAwVariables(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "GH_AW_SKIP_NPX_VALIDATION not redacted",
			message:  "Alternatively, disable validation by setting GH_AW_SKIP_NPX_VALIDATION=true",
			expected: "Alternatively, disable validation by setting GH_AW_SKIP_NPX_VALIDATION=true",
		},
		{
			name:     "GH_AW_SKIP_UV_VALIDATION not redacted",
			message:  "Alternatively, disable validation by setting GH_AW_SKIP_UV_VALIDATION=true",
			expected: "Alternatively, disable validation by setting GH_AW_SKIP_UV_VALIDATION=true",
		},
		{
			name:     "GH_AW_SKIP_PIP_VALIDATION not redacted",
			message:  "Alternatively, disable validation by setting GH_AW_SKIP_PIP_VALIDATION=true",
			expected: "Alternatively, disable validation by setting GH_AW_SKIP_PIP_VALIDATION=true",
		},
		{
			name:     "generic GH_AW prefix not redacted",
			message:  "Set GH_AW_SOME_OPTION to configure this feature",
			expected: "Set GH_AW_SOME_OPTION to configure this feature",
		},
		{
			name:     "non-GH_AW still redacted",
			message:  "Error accessing MY_SECRET_KEY",
			expected: "Error accessing [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.message)
			assertSanitizeResult(t, "SanitizeErrorMessage", tt.message, result, tt.expected)
		})
	}
}

func TestSanitizeErrorMessage_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "GitHub Actions error",
			message:  "Failed to authenticate: GITHUB_TOKEN is invalid",
			expected: "Failed to authenticate: [REDACTED] is invalid",
		},
		{
			name:     "AWS credentials error",
			message:  "AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are required",
			expected: "[REDACTED] and [REDACTED] are required",
		},
		{
			name:     "Database connection error",
			message:  "Could not connect using DB_PASSWORD: connection refused",
			expected: "Could not connect using [REDACTED]: connection refused",
		},
		{
			name:     "API error with token",
			message:  "Request failed with ApiToken: 401 Unauthorized",
			expected: "Request failed with [REDACTED]: 401 Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.message)
			assertSanitizeResult(t, "SanitizeErrorMessage", tt.message, result, tt.expected)
		})
	}
}

func BenchmarkSanitizeErrorMessage_NoSecrets(b *testing.B) {
	message := "This is a regular error message with no secrets to redact"
	for b.Loop() {
		SanitizeErrorMessage(message)
	}
}

func BenchmarkSanitizeErrorMessage_ManySecrets(b *testing.B) {
	message := "Error with API_KEY, DATABASE_PASSWORD, AWS_SECRET, GitHubToken, and DeploySecret"
	for b.Loop() {
		SanitizeErrorMessage(message)
	}
}

func TestSanitizeIdentifierName(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		extraAllowed func(rune) bool
		expected     string
	}{
		{
			name:     "default behavior uses underscores",
			input:    "my-workflow.name",
			expected: "my_workflow_name",
		},
		{
			name:     "prefix underscore when starting with number",
			input:    "123name",
			expected: "_123name",
		},
		{
			name:         "allows extra characters when provided",
			input:        "$param",
			extraAllowed: func(r rune) bool { return r == '$' },
			expected:     "$param",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIdentifierName(tt.input, tt.extraAllowed)
			assertSanitizeResultWithContext(
				t,
				"SanitizeIdentifierName",
				fmt.Sprintf("%q, extraAllowedProvided=%t", tt.input, tt.extraAllowed != nil),
				result,
				tt.expected,
			)
		})
	}
}

func TestSanitizeParameterName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dash-separated",
			input:    "my-param",
			expected: "my_param",
		},
		{
			name:     "dot-separated",
			input:    "my.param",
			expected: "my_param",
		},
		{
			name:     "starts with number",
			input:    "123param",
			expected: "_123param",
		},
		{
			name:     "already valid",
			input:    "valid_name",
			expected: "valid_name",
		},
		{
			name:     "with dollar sign",
			input:    "$special",
			expected: "$special",
		},
		{
			name:     "mixed special chars",
			input:    "param-name.test",
			expected: "param_name_test",
		},
		{
			name:     "spaces",
			input:    "my param",
			expected: "my_param",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special chars",
			input:    "---",
			expected: "___",
		},
		{
			name:     "number with underscore",
			input:    "123_param",
			expected: "_123_param",
		},
		{
			name:     "camelCase preserved",
			input:    "myParam",
			expected: "myParam",
		},
		{
			name:     "with at sign",
			input:    "my@param",
			expected: "my_param",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeParameterName(tt.input)
			assertSanitizeResult(t, "SanitizeParameterName", tt.input, result, tt.expected)
		})
	}
}

func TestSanitizePythonVariableName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dash-separated",
			input:    "my-param",
			expected: "my_param",
		},
		{
			name:     "dot-separated",
			input:    "my.param",
			expected: "my_param",
		},
		{
			name:     "starts with number",
			input:    "123param",
			expected: "_123param",
		},
		{
			name:     "already valid",
			input:    "valid_name",
			expected: "valid_name",
		},
		{
			name:     "with dollar sign (invalid in Python)",
			input:    "$special",
			expected: "_special",
		},
		{
			name:     "mixed special chars",
			input:    "param-name.test",
			expected: "param_name_test",
		},
		{
			name:     "spaces",
			input:    "my param",
			expected: "my_param",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special chars",
			input:    "---",
			expected: "___",
		},
		{
			name:     "number with underscore",
			input:    "123_param",
			expected: "_123_param",
		},
		{
			name:     "camelCase preserved",
			input:    "myParam",
			expected: "myParam",
		},
		{
			name:     "with at sign",
			input:    "my@param",
			expected: "my_param",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePythonVariableName(tt.input)
			assertSanitizeResult(t, "SanitizePythonVariableName", tt.input, result, tt.expected)
		})
	}
}

func TestSanitizeToolID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "mcp suffix",
			input:    "notion-mcp",
			expected: "notion",
		},
		{
			name:     "mcp prefix",
			input:    "mcp-notion",
			expected: "notion",
		},
		{
			name:     "mcp in middle",
			input:    "some-mcp-server",
			expected: "some-mcp-server",
		},
		{
			name:     "no mcp",
			input:    "github",
			expected: "github",
		},
		{
			name:     "only mcp",
			input:    "mcp",
			expected: "mcp",
		},
		{
			name:     "mcp both prefix and suffix",
			input:    "mcp-server-mcp",
			expected: "server",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple mcp patterns",
			input:    "mcp-mcp-mcp",
			expected: "mcp",
		},
		{
			name:     "mcp as part of word",
			input:    "mcpserver",
			expected: "mcpserver",
		},
		{
			name:     "uppercase MCP",
			input:    "MCP-notion",
			expected: "MCP-notion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeToolID(tt.input)
			assertSanitizeResult(t, "SanitizeToolID", tt.input, result, tt.expected)
		})
	}
}

func BenchmarkSanitizeParameterName(b *testing.B) {
	name := "my-complex-parameter.name"
	for b.Loop() {
		SanitizeParameterName(name)
	}
}

func BenchmarkSanitizePythonVariableName(b *testing.B) {
	name := "my-complex-parameter.name"
	for b.Loop() {
		SanitizePythonVariableName(name)
	}
}

func BenchmarkSanitizeToolID(b *testing.B) {
	toolID := "mcp-notion-server-mcp"
	for b.Loop() {
		SanitizeToolID(toolID)
	}
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		name     string
		slug     string
		expected string
	}{
		{
			name:     "normal slug",
			slug:     "github/gh-aw",
			expected: "github-gh-aw",
		},
		{
			name:     "empty slug",
			slug:     "",
			expected: "clone-mode",
		},
		{
			name:     "slug with multiple slashes",
			slug:     "owner/repo/extra",
			expected: "owner-repo-extra",
		},
		{
			name:     "slug with hyphen",
			slug:     "owner/my-repo",
			expected: "owner-my-repo",
		},
		{
			name:     "multiple slashes",
			slug:     "owner/repo/extra",
			expected: "owner-repo-extra",
		},
		{
			name:     "leading slash",
			slug:     "/owner/repo",
			expected: "-owner-repo",
		},
		{
			name:     "trailing slash",
			slug:     "owner/repo/",
			expected: "owner-repo-",
		},
		{
			name:     "only slashes",
			slug:     "///",
			expected: "---",
		},
		{
			name:     "single character owner and repo",
			slug:     "a/b",
			expected: "a-b",
		},
		{
			name:     "slug with dot and underscore preserved",
			slug:     "my.org/my_repo",
			expected: "my.org-my_repo",
		},
		{
			name:     "slug with special characters replaced",
			slug:     "owner/repo!name",
			expected: "owner-repo-name",
		},
		{
			name:     "slug with space replaced",
			slug:     "owner/repo name",
			expected: "owner-repo-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForFilename(tt.slug)
			assertSanitizeResult(t, "SanitizeForFilename", tt.slug, result, tt.expected)
		})
	}
}

func BenchmarkSanitizeForFilename(b *testing.B) {
	slug := "github/gh-aw"
	for b.Loop() {
		SanitizeForFilename(slug)
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     *SanitizeOptions
		expected string
	}{
		{
			name:     "nil options remove special chars",
			input:    "My Workflow@123",
			opts:     nil,
			expected: "my-workflow123",
		},
		{
			name:  "preserve dot and underscore",
			input: "My.Workflow_Name",
			opts: &SanitizeOptions{
				PreserveSpecialChars: []rune{'.', '_'},
			},
			expected: "my.workflow_name",
		},
		{
			name:  "trim and default when empty",
			input: "@@@",
			opts: &SanitizeOptions{
				TrimHyphens:  true,
				DefaultValue: "default-name",
			},
			expected: "default-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeName(tt.input, tt.opts)
			assertSanitizeResultWithContext(
				t,
				"SanitizeName",
				fmt.Sprintf("%q, opts=%+v", tt.input, tt.opts),
				result,
				tt.expected,
			)
		})
	}
}
