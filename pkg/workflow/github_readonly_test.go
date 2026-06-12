//go:build !integration

package workflow

import "testing"

func TestGetGitHubReadOnly(t *testing.T) {
	// getGitHubReadOnly always returns true; the GitHub MCP server is unconditionally read-only.
	result := getGitHubReadOnly()
	if !result {
		t.Errorf("getGitHubReadOnly() = %v, want true", result)
	}
}
