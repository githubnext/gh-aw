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

func TestEnclaveGitHubMCPAgentPolicy(t *testing.T) {
	tests := []struct {
		name             string
		data             *WorkflowData
		wantTools        []string
		wantRepos        []string
		wantMinIntegrity string
	}{
		{
			name: "legacy profile defaults",
			data: func() *WorkflowData {
				data := enclaveGitHubIssuesWorkflowData()
				data.Enclaves[0].Repos = []*EnclaveRepository{
					{Repo: "octo-org/trusted-service", Sensitivity: "trusted"},
					{Repo: "octo-org/public-docs", Sensitivity: "public"},
				}
				return data
			}(),
			wantTools:        []string{"list_issues", "issue_read"},
			wantRepos:        []string{"octo-org/trusted-service", "octo-org/public-docs"},
			wantMinIntegrity: "approved",
		},
		{
			name: "agent tools config overrides defaults",
			data: func() *WorkflowData {
				data := enclaveGitHubToolsWorkflowData()
				data.Enclaves[0].Repos = []*EnclaveRepository{
					{Repo: "octo-org/private-service", Sensitivity: "confidential"},
					{Repo: "octo-org/public-docs", Sensitivity: "public"},
				}
				data.Enclaves[0].Agent.Tools.GitHub.AllowedRepos = GitHubReposScope{"octo-org/public-docs"}
				return data
			}(),
			wantTools:        []string{"list_issues", "issue_read"},
			wantRepos:        []string{"octo-org/public-docs"},
			wantMinIntegrity: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := enclaveGitHubMCPAgentPolicy(tt.data)
			assert.Equal(t, []string{"github"}, policy.Servers)
			assert.Equal(t, map[string][]string{"github": tt.wantTools}, policy.Tools)
			assert.Equal(t, map[string]any{
				"repos":         tt.wantRepos,
				"min-integrity": tt.wantMinIntegrity,
			}, policy.AllowOnly)
		})
	}
}

func TestEnclaveGitHubMCPGatewayConfiguration(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	data.Tools["github"] = map[string]any{}
	data.SafeOutputs = &SafeOutputsConfig{AddComments: &AddCommentsConfig{}}
	config := buildMCPGatewayConfig(data)

	assert.Empty(t, config.AgentID)
	assert.Equal(t, []string{"${MCP_GATEWAY_AGENT_ID}", "${AWF_ENCLAVE_GITHUB_MCP_AGENT_ID}"}, config.AgentIDs)
	assert.Equal(t, []string{enclaveMCPServerName, "github", constants.SafeOutputsMCPServerID.String()}, config.AgentPolicies["${MCP_GATEWAY_AGENT_ID}"].Servers)
	assert.Equal(t, []string{"github"}, config.AgentPolicies["${AWF_ENCLAVE_GITHUB_MCP_AGENT_ID}"].Servers)

	generatedServers := make(map[string]struct{})
	for _, server := range collectMCPServersForManifest(data) {
		generatedServers[server.Name] = struct{}{}
	}
	for agentID, policy := range config.AgentPolicies {
		for _, server := range policy.Servers {
			assert.Contains(t, generatedServers, server, "policy for %s references an unknown MCP server", agentID)
		}
	}
}

func TestToolsWithEnclaveGitHubIssuesUnionsTypedToolsets(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	tools := map[string]any{
		"github": map[string]any{"toolsets": []string{"context"}},
	}

	updated := toolsWithEnclaveGitHubIssues(tools, data)

	assert.Equal(t, []string{"context", "issues"}, updated["github"].(map[string]any)["toolsets"])
	assert.Equal(t, []string{"context"}, tools["github"].(map[string]any)["toolsets"], "original tools must remain unchanged")
}

func TestCompileEnclaveGitHubSharedGateway(t *testing.T) {
	tmp := t.TempDir()
	workflowPath := filepath.Join(tmp, "enclave-github.md")
	content := `---
on: workflow_dispatch
strict: false
network: defaults
engine: copilot
tools:
  github:
    toolsets: [context]
safe-outputs:
  add-comment:
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
	assert.Contains(t, lock, `"safeoutputs": {`)
	assert.Contains(t, lock, `"awf-enclave": {`)
	assert.NotContains(t, lock, `"required": false`)
	assert.Contains(t, lock, `"GITHUB_TOOLSETS": "context,issues"`)
	assert.Contains(t, lock, `"${MCP_GATEWAY_AGENT_ID}":{"servers":["awf-enclave","github","safeoutputs"],"tools":{"github":["get_me"]}}`)
	assert.NotContains(t, lock, `"servers":["awf-enclave","github","safe-outputs"]`)
	assert.Contains(t, lock, `"agentPolicies": {"${AWF_ENCLAVE_GITHUB_MCP_AGENT_ID}":{"servers":["github"],"tools":{"github":["list_issues","issue_read"]},"allow-only":{"min-integrity":"approved","repos":["octo-org/private-service"]}}`)
	assert.Contains(t, lock, `AWF_ENCLAVE_GITHUB_MCP_AGENT_ID=$(openssl rand -base64 45 | tr -d '/+=')`)
	assert.Contains(t, lock, `printf '%s=%s\n' AWF_ENCLAVE_GITHUB_MCP_AGENT_ID "$AWF_ENCLAVE_GITHUB_MCP_AGENT_ID"`)
	assert.Contains(t, lock, `MCP_GATEWAY_API_KEY: ${{ steps.start-mcp-gateway.outputs.gateway-api-key }}`)
	assert.Contains(t, lock, `--exclude-env MCP_GATEWAY_API_KEY`)
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

func TestEnclaveGitHubToolsVersionGates(t *testing.T) {
	data := enclaveGitHubToolsWorkflowData()
	data.NetworkPermissions.Firewall.Version = string(constants.AWFEnclaveGitHubIssuesMinVersion)
	require.NoError(t, validateEnclavesConfig(data))

	data.NetworkPermissions.Firewall.Version = "v0.28.8"
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), string(constants.AWFEnclaveGitHubIssuesMinVersion))

	data = enclaveGitHubToolsWorkflowData()
	data.SandboxConfig.MCP.Version = "v0.4.14"
	err = validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), string(constants.MCPGEnclaveAgentToolsMinVersion))
}
