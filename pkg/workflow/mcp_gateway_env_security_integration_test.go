//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compileMCPGatewayWorkflowLock(t *testing.T, workflowContent string) string {
	t.Helper()

	tmpDir := testutil.TempDir(t, "mcp-gateway-env-security")
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0o644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath))

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	return string(lockContent)
}

func extractMCPGatewayStepSection(jobSection, stepName string) string {
	lines := strings.Split(jobSection, "\n")
	var stepLines []string
	var stepPrefix string
	inStep := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inStep {
			if trimmed != "- name: "+stepName {
				continue
			}
			inStep = true
			stepPrefix = line[:strings.Index(line, "- name: ")]
			stepLines = append(stepLines, line)
			continue
		}

		if strings.HasPrefix(line, stepPrefix+"- ") {
			break
		}
		stepLines = append(stepLines, line)
	}

	if len(stepLines) == 0 {
		return ""
	}

	return strings.Join(stepLines, "\n")
}

func TestMCPGatewayCustomEnvIntegrationUsesTransportVariables(t *testing.T) {
	lockContent := compileMCPGatewayWorkflowLock(t, `---
on: workflow_dispatch
strict: false
engine: copilot
tools:
  github:
    toolsets: [repos]
sandbox:
  mcp:
    env:
      API_TOKEN: custom-token
      BASH_ENV: "$(touch /tmp/pwned)"
---

# Test Workflow

Verify MCP gateway custom env transport.
`)

	agentSection := extractJobSection(lockContent, "agent")
	require.NotEmpty(t, agentSection)

	gatewayStep := extractMCPGatewayStepSection(agentSection, "Start MCP Gateway")
	require.NotEmpty(t, gatewayStep)

	assert.Contains(t, gatewayStep, `GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: "[\"API_TOKEN\",\"BASH_ENV\"]"`)
	assert.Contains(t, gatewayStep, `GH_AW_MCP_GATEWAY_ENV_0: "custom-token"`)
	assert.Contains(t, gatewayStep, `GH_AW_MCP_GATEWAY_ENV_1: "$(touch /tmp/pwned)"`)
	assert.NotContains(t, gatewayStep, "\n          API_TOKEN:")
	assert.NotContains(t, gatewayStep, "\n          BASH_ENV:")
	assert.NotContains(t, gatewayStep, "export API_TOKEN=")
	assert.NotContains(t, gatewayStep, "export BASH_ENV=")
}

func TestMCPGatewayCustomEnvIntegrationKeepsMetadataNameUnique(t *testing.T) {
	tmpDir := testutil.TempDir(t, "mcp-gateway-env-collision")
	sharedMCPPath := filepath.Join(tmpDir, "shared-mcp.md")
	require.NoError(t, os.WriteFile(sharedMCPPath, []byte(`---
on: workflow_dispatch
mcp-servers:
  collision:
    url: "https://example.com/mcp"
    env:
      FORWARDED_NAMES: ${{ env.GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES }}
---`), 0o644))

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(`---
on: workflow_dispatch
strict: false
engine: copilot
imports:
  - shared-mcp.md
tools:
  github:
    toolsets: [repos]
sandbox:
  mcp:
    env:
      API_TOKEN: custom-token
---

# Test Workflow

Verify MCP gateway reserved env name handling.
`), 0o644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath))

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	lockContent := string(lockBytes)
	assert.Contains(t, lockContent, `"collision": {`)
	assert.NotContains(t, lockContent, "FORWARDED_NAMES")

	agentSection := extractJobSection(lockContent, "agent")
	require.NotEmpty(t, agentSection)

	gatewayStep := extractMCPGatewayStepSection(agentSection, "Start MCP Gateway")
	require.NotEmpty(t, gatewayStep)

	assert.Equal(t, 1, strings.Count(gatewayStep, mcpGatewayCustomEnvNamesVar+":"))
	assert.Contains(t, gatewayStep, `GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES: "[\"API_TOKEN\"]"`)
	assert.Contains(t, gatewayStep, `GH_AW_MCP_GATEWAY_ENV_0: "custom-token"`)
}
