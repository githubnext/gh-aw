//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubProxyModeIntegration(t *testing.T) {
	tmpDir := testutil.TempDir(t, "github-mode-gh-proxy-workflow-file-test")
	workflowFile := "../cli/workflows/test-copilot-gh-proxy.md"

	src, err := os.ReadFile(workflowFile)
	require.NoError(t, err, "Failed to read workflow file %s", workflowFile)

	baseName := filepath.Base(workflowFile)
	mdDst := filepath.Join(tmpDir, baseName)
	require.NoError(t, os.WriteFile(mdDst, src, 0o600),
		"Failed to write workflow file to temporary directory")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(mdDst)
	require.NoError(t, err, "Workflow file %s should compile successfully", workflowFile)

	lockName := strings.TrimSuffix(baseName, ".md") + ".lock.yml"
	lockPath := filepath.Join(tmpDir, lockName)
	compiledBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	compiled := string(compiledBytes)

	assert.Contains(t, compiled, "start_cli_proxy.sh",
		"Compiled workflow should include host CLI proxy startup for tools.github.mode=gh-proxy")
	assert.Contains(t, compiled, "stop_cli_proxy.sh",
		"Compiled workflow should include host CLI proxy cleanup for tools.github.mode=gh-proxy")
	assert.Contains(t, compiled, "cli_proxy_prompt.md",
		"Compiled workflow should include CLI proxy guidance prompt for tools.github.mode=gh-proxy")

	assert.NotContains(t, compiled, "github_mcp_tools_prompt.md",
		"Compiled workflow should not include GitHub MCP prompt guidance for tools.github.mode=gh-proxy")
	assert.NotContains(t, compiled, "api.githubcopilot.com/mcp/",
		"Compiled workflow should not register remote GitHub MCP server for tools.github.mode=gh-proxy")
}
