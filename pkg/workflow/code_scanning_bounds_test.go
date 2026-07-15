//go:build !integration

package workflow

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitHubMCPToolsPrompt_CodeScanningAlertsBounds is a regression test ensuring
// that the runtime GitHub MCP tools prompt template always instructs agents to bound
// list_code_scanning_alerts calls with state:open and severity:critical,high.
//
// Background: oversized list_code_scanning_alerts responses (caused by missing query
// bounds) are the highest-retrieval failure pattern in the Hippo memory store.
// Encoding the guard in the template prevents token-bloat failures across workflows.
func TestGitHubMCPToolsPrompt_CodeScanningAlertsBounds(t *testing.T) {
	paths := []string{
		"../../actions/setup/md/github_mcp_tools_prompt.md",
		"../../actions/setup/md/github_mcp_tools_with_safeoutputs_prompt.md",
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			data, err := os.ReadFile(p)
			require.NoError(t, err, "should read MCP tools prompt template")

			content := string(data)

			assert.Contains(t, content, "list_code_scanning_alerts",
				"prompt template must mention list_code_scanning_alerts so agents know the bound rule")
			assert.Contains(t, content, "state: open",
				"prompt template must require state:open bound for list_code_scanning_alerts")
			assert.Contains(t, content, "severity: critical,high",
				"prompt template must require severity:critical,high bound for list_code_scanning_alerts")
		})
	}
}
