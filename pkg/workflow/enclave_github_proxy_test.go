//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enclaveGitHubIssuesWorkflowData() *WorkflowData {
	data := enclaveWorkflowData(false, true, 0, 120)
	data.Enclaves[0].Agent.GitHub = &AgentEnclaveGitHubConfig{CLI: enclaveGitHubIssuesProfile}
	data.SandboxConfig.MCP = &MCPGatewayRuntimeConfig{
		Container: constants.DefaultMCPGatewayContainer,
		Version:   string(constants.MCPGEnclaveGitHubIssuesMinVersion),
	}
	return data
}

func TestEnclaveGitHubMCPAgentPolicy(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	data.Enclaves[0].Repos = append(data.Enclaves[0].Repos,
		&EnclaveRepository{Repo: "octo-org/public-docs", Sensitivity: "public"})

	policy := enclaveGitHubMCPAgentPolicy(data)
	assert.Equal(t, []string{"github"}, policy.Servers)
	assert.Equal(t, map[string][]string{"github": {"list_issues", "issue_read"}}, policy.Tools)
	assert.Equal(t, map[string]any{
		"repos":         []string{"octo-org/private-service", "octo-org/public-docs"},
		"min-integrity": "unapproved",
	}, policy.AllowOnly)
}

func TestEnclaveGitHubMCPGatewayConfiguration(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	config := buildMCPGatewayConfig(data)

	assert.Empty(t, config.AgentID)
	assert.Equal(t, []string{"${MCP_GATEWAY_AGENT_ID}", "${AWF_ENCLAVE_GITHUB_MCP_AGENT_ID}"}, config.AgentIDs)
	assert.Equal(t, []string{enclaveMCPServerName}, config.AgentPolicies["${MCP_GATEWAY_AGENT_ID}"].Servers)
	assert.Equal(t, []string{"github"}, config.AgentPolicies["${AWF_ENCLAVE_GITHUB_MCP_AGENT_ID}"].Servers)
}

func TestCompileEnclaveGitHubSharedGateway(t *testing.T) {
	tmp := t.TempDir()
	workflowPath := filepath.Join(tmp, "enclave-github.md")
	content := `---
on: workflow_dispatch
strict: false
network: defaults
engine: copilot
sandbox:
  agent:
    id: awf
  mcp:
    version: v0.4.15
enclaves:
  - agent:
      model: gpt-5
      github:
        cli: issues-read-v1
    repos:
      - repo: octo-org/private-service
        sensitivity: confidential
---

Read the assigned repository's issues through the enclave.
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0o600))
	compiler := NewCompiler()
	compiler.SetSkipValidation(true)
	require.NoError(t, compiler.CompileWorkflow(workflowPath))
	lockBytes, err := os.ReadFile(strings.TrimSuffix(workflowPath, ".md") + ".lock.yml")
	require.NoError(t, err)
	lock := string(lockBytes)

	assert.Equal(t, 1, strings.Count(lock, "--name awmg-mcpg"))
	assert.Contains(t, lock, `"agentIds": ["${MCP_GATEWAY_AGENT_ID}","${AWF_ENCLAVE_GITHUB_MCP_AGENT_ID}"]`)
	assert.Contains(t, lock, `"agentPolicies": {"${AWF_ENCLAVE_GITHUB_MCP_AGENT_ID}":{"servers":["github"],"tools":{"github":["list_issues","issue_read"]},"allow-only":{"min-integrity":"unapproved","repos":["octo-org/private-service"]}}`)
	assert.Contains(t, lock, `AWF_ENCLAVE_GITHUB_MCP_AGENT_ID=$(openssl rand -base64 45 | tr -d '/+=')`)
	assert.Contains(t, lock, `printf '%s=%s\n' AWF_ENCLAVE_GITHUB_MCP_AGENT_ID "$AWF_ENCLAVE_GITHUB_MCP_AGENT_ID"`)
	assert.Contains(t, lock, "--exclude-env AWF_ENCLAVE_GITHUB_MCP_AGENT_ID")
	assert.NotContains(t, lock, "Enclave GitHub Proxy")
	assert.NotContains(t, lock, "start_enclave_github_proxy")
	assert.NotContains(t, lock, "stop_enclave_github_proxy")
}

func TestEnclaveGitHubMCPVersionGates(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	data.NetworkPermissions.Firewall.Version = string(constants.AWFEnclaveGitHubIssuesMinVersion)
	require.NoError(t, validateEnclavesConfig(data))

	data.SandboxConfig.MCP.Version = "v0.4.14"
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), string(constants.MCPGEnclaveGitHubIssuesMinVersion))
}
