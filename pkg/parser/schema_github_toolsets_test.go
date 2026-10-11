//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitHubToolsetsSchemaAcceptsMappedToolsets(t *testing.T) {
	for _, toolset := range []string{
		"code_quality", "copilot", "copilot_issue_intents", "copilot_spaces",
		"git", "github_support_docs_search", "governance",
	} {
		t.Run(toolset, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on": "workflow_dispatch",
				"tools": map[string]any{
					"github": map[string]any{"toolsets": []any{toolset}},
				},
			}, "workflow.md")
			require.NoError(t, err)
		})
	}
}
