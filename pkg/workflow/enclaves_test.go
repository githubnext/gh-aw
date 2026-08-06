package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enclaveWorkflowData(script, agent bool, scriptTimeout, agentTimeout int) *WorkflowData {
	data := &WorkflowData{
		Tools: map[string]any{},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:               "awf",
				NetworkIsolation: true,
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Enclaves: &EnclavesConfig{
			Enabled: true,
			PrivateRepos: []*EnclavePrivateRepo{{
				Repo: "octo-org/private-service", Sensitivity: "confidential",
			}},
			Executors: &EnclaveExecutorsConfig{},
		},
	}
	if script {
		data.Enclaves.Executors.Script = &ScriptEnclaveExecutorConfig{
			Enabled: true, Timeout: scriptTimeout,
		}
	}
	if agent {
		data.Enclaves.Executors.Agent = &AgentEnclaveExecutorConfig{
			Enabled: true, Model: "gpt-5", Timeout: agentTimeout,
		}
	}
	return data
}

func TestEnabledEnclaveToolsAndTimeout(t *testing.T) {
	tests := []struct {
		name                  string
		script, agent         bool
		scriptTime, agentTime int
		wantTools             []string
		wantTimeout           int
	}{
		{"script only defaults", true, false, 0, 0, []string{"enclave_run_script"}, 60},
		{"agent only defaults", false, true, 0, 0, []string{"enclave_run_agent"}, 150},
		{"both use maximum", true, true, 200, 90, []string{"enclave_run_script", "enclave_run_agent"}, 230},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := enclaveWorkflowData(tt.script, tt.agent, tt.scriptTime, tt.agentTime)
			assert.Equal(t, tt.wantTools, enabledEnclaveTools(data))
			assert.Equal(t, tt.wantTimeout, enclaveToolTimeout(data))
			assert.Contains(t, collectMCPTools(data), enclaveMCPServerName)
		})
	}

	disabled := enclaveWorkflowData(true, true, 30, 120)
	disabled.Enclaves.Enabled = false
	assert.Empty(t, enabledEnclaveTools(disabled))
	assert.NotContains(t, collectMCPTools(disabled), enclaveMCPServerName)
}

func TestValidateEnclavesRequiresNetworkIsolation(t *testing.T) {
	data := enclaveWorkflowData(true, false, 30, 0)
	data.SandboxConfig.Agent.NetworkIsolation = false
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires AWF network isolation")
}

func TestValidateEnclavesRejectsBoundedQueries(t *testing.T) {
	data := enclaveWorkflowData(true, false, 30, 0)
	data.ParsedTools = &ToolsConfig{
		GitHub: &GitHubToolConfig{
			BoundedQueries: &BoundedQueriesConfig{},
		},
	}
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined with tools.github.bounded-queries")
}

func TestBuildAWFConfigJSONEnclaves(t *testing.T) {
	data := enclaveWorkflowData(true, true, 45, 180)
	configJSON, err := BuildAWFConfigJSON(AWFCommandConfig{
		EngineName: "copilot", WorkflowData: data,
	})
	require.NoError(t, err)

	var config map[string]any
	require.NoError(t, json.Unmarshal([]byte(configJSON), &config))
	enclaves := config["enclaves"].(map[string]any)
	executors := enclaves["executors"].(map[string]any)
	assert.InDelta(t, 45, executors["script"].(map[string]any)["timeout"], 0)
	assert.Equal(t, "gpt-5", executors["agent"].(map[string]any)["model"])
	assert.Equal(t, []any{"awmg-mcpg"}, config["network"].(map[string]any)["topologyAttach"])
	assert.NotContains(t, configJSON, "boundedQueries")
	assert.NotContains(t, configJSON, "boundedAgents")
}

func TestGenerateEnclaveGatewayContract(t *testing.T) {
	data := enclaveWorkflowData(true, true, 45, 180)
	ensureDefaultMCPGatewayConfig(data)
	var output strings.Builder
	require.NoError(t, generateMCPGatewaySetup(
		&output, data.Tools, []string{enclaveMCPServerName}, NewCopilotEngine(), data, false, nil,
	))
	generated := output.String()

	assert.Contains(t, generated, `"awf-enclave": {`)
	assert.Contains(t, generated, `"url": "http://awf-enclave-mcp:8080/mcp"`)
	assert.Contains(t, generated, `"connectTimeout": 120`)
	assert.Contains(t, generated, `"toolTimeout": 210`)
	assert.Contains(t, generated, `"tools": ["enclave_run_script", "enclave_run_agent"]`)
	assert.Contains(t, generated, `Bearer \${AWF_ENCLAVE_MCP_CAPABILITY}`)
	assert.Contains(t, generated, `openssl rand -hex 32`)
	assert.Contains(t, generated, `::add-mask::${AWF_ENCLAVE_MCP_CAPABILITY}`)
	assert.Contains(t, generated, `--network bridge`)
	assert.Contains(t, generated, `--label com.github.gh-aw.mcpg.run=`)
	assert.Contains(t, generated, `${AWF_ENCLAVE_MCP_GATEWAY_IDENTITY}`)
	assert.Contains(t, generated, `-e AWF_ENCLAVE_MCP_CAPABILITY`)
	assert.Contains(t, generated, `AWF_ENCLAVE_MCP_GATEWAY_ENDPOINT="http://localhost:${MCP_GATEWAY_PORT}/mcp/awf-enclave"`)
	assert.NotRegexp(t, `AWF_ENCLAVE_MCP_CAPABILITY=[0-9a-f]{64}`, generated)

	excluded := ComputeAWFExcludeEnvVarNames(data, nil)
	assert.Contains(t, excluded, enclaveMCPCapabilityEnv)
	assert.Contains(t, excluded, enclaveMCPGatewayIdentityEnv)
}

func TestCompileEnclaveStartupOrdering(t *testing.T) {
	tmp := t.TempDir()
	workflowPath := filepath.Join(tmp, "enclave.md")
	content := `---
on: workflow_dispatch
strict: false
network: defaults
engine: copilot
sandbox:
  agent:
    id: awf
    sudo: false
    version: latest
enclaves:
  enabled: true
  private-repos:
    - repo: octo-org/private-service
      sensitivity: confidential
  executors:
    script:
      enabled: true
      timeout: 45
---

Use the enclave script executor.
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0o600))
	compiler := NewCompiler()
	compiler.SetSkipValidation(true)
	require.NoError(t, compiler.CompileWorkflow(workflowPath))
	lockBytes, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
	require.NoError(t, err)
	lock := string(lockBytes)

	gateway := strings.Index(lock, "- name: Start MCP Gateway")
	awf := strings.Index(lock, "awf --config")
	require.Greater(t, gateway, -1)
	require.Greater(t, awf, -1)
	assert.Less(t, gateway, awf)
	assert.NotContains(t, lock, "Start Enclave MCP")
	assert.NotContains(t, lock, "start_enclave")
}
