//go:build !integration

package workflow_test

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
)

// knownFieldsEnabledTools is the set of GitHub MCP tools that support the optional
// `fields` parameter as of github-mcp-server v1.6.0. Tests reference this set to
// catch typos in GitHubMCPDefaultFields keys without embedding the full MCP tool
// registry.
var knownFieldsEnabledTools = map[string]bool{
	"list_pull_requests":   true,
	"search_pull_requests": true,
	"list_issues":          true,
	"search_issues":        true,
	"search_code":          true,
	"list_commits":         true,
	"get_file_contents":    true,
	"list_releases":        true,
}

// TestGitHubMCPDefaultFields_KeysAreKnownTools asserts that every key in
// GitHubMCPDefaultFields is a recognized fields-enabled tool name. This catches
// typos introduced when adding new entries (e.g. "list_pullrequests" instead of
// "list_pull_requests").
func TestGitHubMCPDefaultFields_KeysAreKnownTools(t *testing.T) {
	for toolName, fields := range workflow.GitHubMCPDefaultFields {
		if !knownFieldsEnabledTools[toolName] {
			t.Errorf("GitHubMCPDefaultFields contains unknown tool name %q; update knownFieldsEnabledTools or fix the typo", toolName)
		}
		if len(fields) == 0 {
			t.Errorf("GitHubMCPDefaultFields[%q] is empty; each tool must have at least one recommended field", toolName)
		}
		for _, f := range fields {
			if f == "" {
				t.Errorf("GitHubMCPDefaultFields[%q] contains an empty field name", toolName)
			}
		}
	}
}

// TestGitHubMCPDefaultFields_AllKnownToolsHaveDefaults asserts that every
// fields-enabled tool has a recommended default, so new additions to the server
// are not silently omitted from the map.
func TestGitHubMCPDefaultFields_AllKnownToolsHaveDefaults(t *testing.T) {
	for toolName := range knownFieldsEnabledTools {
		if _, ok := workflow.GitHubMCPDefaultFields[toolName]; !ok {
			t.Errorf("knownFieldsEnabledTools lists %q but GitHubMCPDefaultFields has no entry for it; add a recommended field set", toolName)
		}
	}
}
