//go:build !integration

package cli

import (
	"os"
	"testing"
)

func TestExtractIssueNumberFromURL(t *testing.T) {
	// All of these cases run with the default host (no GH_HOST set), so
	// getGitHubHost() returns https://github.com.
	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Valid GitHub issue URL",
			url:      "https://github.com/github/releases/issues/6818",
			expected: "6818",
		},
		{
			name:     "Another valid issue URL",
			url:      "https://github.com/github/gh-aw-trial/issues/123",
			expected: "123",
		},
		{
			name:     "Issue URL with single digit",
			url:      "https://github.com/user/repo/issues/5",
			expected: "5",
		},
		{
			name:     "Invalid URL - not GitHub",
			url:      "https://gitlab.com/user/repo/issues/123",
			expected: "",
		},
		{
			name:     "Invalid URL - not an issue",
			url:      "https://github.com/user/repo/pulls/123",
			expected: "",
		},
		{
			name:     "Invalid URL - missing issue number",
			url:      "https://github.com/user/repo/issues/",
			expected: "",
		},
		{
			name:     "Invalid URL - non-numeric issue number",
			url:      "https://github.com/user/repo/issues/abc",
			expected: "",
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "URL with query parameters",
			url:      "https://github.com/user/repo/issues/456?tab=comments",
			expected: "456",
		},
		{
			name:     "URL with fragment",
			url:      "https://github.com/user/repo/issues/789#issuecomment-123456",
			expected: "789",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure no GH_HOST is set so getGitHubHost() returns the public github.com default.
			t.Setenv("GITHUB_SERVER_URL", "")
			t.Setenv("GITHUB_ENTERPRISE_HOST", "")
			t.Setenv("GITHUB_HOST", "")
			t.Setenv("GH_HOST", "")
			result := parseIssueSpec(tc.url)
			if result != tc.expected {
				t.Errorf("parseIssueSpec(%q) = %q, expected %q", tc.url, result, tc.expected)
			}
		})
	}
}

func TestExtractIssueNumberFromURL_GHES(t *testing.T) {
	// When GH_HOST points to a GitHub Enterprise Server, parseIssueSpec must accept
	// issue URLs on that host and must reject github.com URLs (which belong to a
	// different host in this configuration).
	t.Setenv("GITHUB_SERVER_URL", "")
	t.Setenv("GITHUB_ENTERPRISE_HOST", "")
	t.Setenv("GITHUB_HOST", "")
	t.Setenv("GH_HOST", "example.ghe.com")

	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GHES issue URL",
			url:      "https://example.ghe.com/owner/repo/issues/42",
			expected: "42",
		},
		{
			name:     "GHES issue URL with query parameters",
			url:      "https://example.ghe.com/owner/repo/issues/99?tab=comments",
			expected: "99",
		},
		{
			name:     "GHES issue URL with fragment",
			url:      "https://example.ghe.com/owner/repo/issues/7#issuecomment-1",
			expected: "7",
		},
		{
			name:     "github.com issue URL rejected when GH_HOST is GHES",
			url:      "https://github.com/owner/repo/issues/123",
			expected: "",
		},
		{
			name:     "other host rejected",
			url:      "https://gitlab.com/owner/repo/issues/123",
			expected: "",
		},
		{
			name:     "GHES non-issue URL rejected",
			url:      "https://example.ghe.com/owner/repo/pulls/5",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseIssueSpec(tc.url)
			if result != tc.expected {
				t.Errorf("parseIssueSpec(%q) = %q, expected %q", tc.url, result, tc.expected)
			}
		})
	}
}

func TestTrialWorkflowSpecParsing(t *testing.T) {
	// Test that workflow spec parsing still works with the new trial functionality
	testCases := []struct {
		name         string
		spec         string
		expectedRepo string
		expectedName string
		shouldError  bool
	}{
		{
			name:         "GitHub URL workflow spec",
			spec:         "github/gh-aw-trial/.github/workflows/release-issue-linker.md",
			expectedRepo: "github/gh-aw-trial",
			expectedName: "release-issue-linker",
			shouldError:  false,
		},
		{
			name:         "Simple workflow spec",
			spec:         "user/repo/workflow-name",
			expectedRepo: "user/repo",
			expectedName: "workflow-name",
			shouldError:  false,
		},
		{
			name:         "Invalid workflow spec",
			spec:         "invalid-spec",
			expectedRepo: "",
			expectedName: "",
			shouldError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := parseWorkflowSpec(tc.spec)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for spec %q, but got none", tc.spec)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for spec %q: %v", tc.spec, err)
				return
			}

			if spec.RepoSlug != tc.expectedRepo {
				t.Errorf("Expected repo %q, got %q", tc.expectedRepo, spec.RepoSlug)
			}

			if spec.WorkflowName != tc.expectedName {
				t.Errorf("Expected workflow name %q, got %q", tc.expectedName, spec.WorkflowName)
			}
		})
	}
}

func TestWorkflowDeclaresDispatchInput(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		input    string
		expected bool
	}{
		{
			name: "declares issue_number input",
			content: `on:
  workflow_dispatch:
    inputs:
      issue_number:
        description: "Issue number"
        required: false
        type: string
`,
			input:    "issue_number",
			expected: true,
		},
		{
			name: "does not declare issue_number input",
			content: `on:
  workflow_dispatch:
    inputs:
      aw_context:
        description: "Agent caller context"
        required: false
        type: string
`,
			input:    "issue_number",
			expected: false,
		},
		{
			name: "workflow_dispatch without inputs",
			content: `on:
  workflow_dispatch:
`,
			input:    "issue_number",
			expected: false,
		},
		{
			name: "no workflow_dispatch trigger",
			content: `on:
  schedule:
    - cron: "0 0 * * *"
`,
			input:    "issue_number",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			lockFile := dir + "/workflow.lock.yml"
			if err := os.WriteFile(lockFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to write lock file: %v", err)
			}

			if got := workflowDeclaresDispatchInput(lockFile, tc.input); got != tc.expected {
				t.Errorf("workflowDeclaresDispatchInput() = %v, want %v", got, tc.expected)
			}
		})
	}

	t.Run("missing file returns false", func(t *testing.T) {
		if workflowDeclaresDispatchInput("/nonexistent/path.lock.yml", "issue_number") {
			t.Errorf("expected false for missing file")
		}
	})
}
