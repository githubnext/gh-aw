//go:build integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServerRejectsArgumentBoundaryAttacks(t *testing.T) {
	repo, workflowsDir := setupMCPServerAddRepo(t)
	_, session := setupMCPServerSession(t, repo, testMCPServerAddToolTimeout)
	for _, tc := range []struct {
		tool string
		args map[string]any
		want string
	}{
		{"add", map[string]any{"workflows": []string{"--dir", "../..", "--force", "--no-security-scanner", "https://example.com/payload.md"}}, "must not contain values beginning with '-'"},
		{"update", map[string]any{"workflows": []string{"--dir=../.."}}, "must not contain values beginning with '-'"},
		{"fix", map[string]any{"workflows": []string{"--dir=../.."}, "write": true}, "must not contain values beginning with '-'"},
		{"compile", map[string]any{"workflows": []string{"--approve", "--prior-manifest-file=/tmp/other"}}, "must not contain values beginning with '-'"},
		{"mcp-inspect", map[string]any{"workflow_file": "--help"}, "must not contain values beginning with '-'"},
		{"audit", map[string]any{"run_ids_or_urls": []string{"--help"}}, "must not contain values beginning with '-'"},
		{"audit-diff", map[string]any{"base_run_id": "123", "compare_run_ids": []string{"--help"}}, "must not contain values beginning with '-'"},
		{"add", map[string]any{"workflows": []string{"https://example.com/payload.md"}, "name": "../../pwned"}, "invalid workflow name"},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err != nil {
				require.ErrorContains(t, err, tc.want)
			} else {
				require.NotNil(t, result)
				require.True(t, result.IsError)
				assert.Contains(t, extractTextContent(result), tc.want)
			}
		})
	}
	entries, err := os.ReadDir(workflowsDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.NoFileExists(t, filepath.Join(repo, "pwned.md"))
}
