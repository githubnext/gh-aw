package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enclaveWorkflowData(script, agent bool, scriptTimeout, agentTimeout int) *WorkflowData {
	data := &WorkflowData{
		Tools: map[string]any{},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID: "awf",
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
	}
	if script {
		data.Enclaves = append(data.Enclaves, &EnclaveConfig{
			Script: &ScriptEnclaveConfig{}, Timeout: scriptTimeout, Repos: enclaveTestRepos(),
		})
	}
	if agent {
		data.Enclaves = append(data.Enclaves, &EnclaveConfig{
			Agent: &AgentEnclaveConfig{Model: "gpt-5"}, Timeout: agentTimeout, Repos: enclaveTestRepos(),
		})
	}
	return data
}

func enclaveTestRepos() []*EnclaveRepository {
	return []*EnclaveRepository{{
		Repo: "octo-org/private-service", Sensitivity: "confidential",
	}}
}

func enclaveGitHubIssuesWorkflowData() *WorkflowData {
	data := enclaveWorkflowData(false, true, 0, 120)
	data.Enclaves[0].Agent.GitHub = &AgentEnclaveGitHubConfig{CLI: enclaveGitHubIssuesProfile}
	data.SandboxConfig.MCP = &MCPGatewayRuntimeConfig{
		Container: constants.DefaultMCPGatewayContainer,
		Version:   string(constants.MCPGEnclaveGitHubIssuesMinVersion),
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
		{"script only defaults cover timing bucket", true, false, 0, 0, []string{"enclave_run_script"}, 4860},
		{"agent only defaults cover timing bucket", false, true, 0, 0, []string{"enclave_run_agent"}, 4860},
		{"45 second custom timeout covers timing bucket", true, false, 45, 0, []string{"enclave_run_script"}, 4860},
		{"4740 second maximum timeout covers timing bucket", false, true, 0, 4740, []string{"enclave_run_agent"}, 4860},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := enclaveWorkflowData(tt.script, tt.agent, tt.scriptTime, tt.agentTime)
			assert.Equal(t, tt.wantTools, enabledEnclaveTools(data))
			assert.Equal(t, tt.wantTimeout, enclaveToolTimeout(data))
			assert.Contains(t, collectMCPTools(data), enclaveMCPServerName)
		})
	}

	disabled := enclaveWorkflowData(false, false, 0, 0)
	assert.Empty(t, enabledEnclaveTools(disabled))
	assert.NotContains(t, collectMCPTools(disabled), enclaveMCPServerName)
}

func TestValidateEnclavesRequiresNetworkIsolation(t *testing.T) {
	data := enclaveWorkflowData(true, false, 30, 0)
	data.SandboxConfig.Agent.Disabled = true
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires AWF network isolation")
}

func TestValidateEnclavesRejectsDuplicateTypes(t *testing.T) {
	data := enclaveWorkflowData(true, false, 30, 0)
	data.Enclaves = append(data.Enclaves, &EnclaveConfig{
		Script: &ScriptEnclaveConfig{}, Repos: enclaveTestRepos(),
	})
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `duplicate executor type "script"`)
}

func TestValidateEnclavesRequiresConsistentRepositorySensitivity(t *testing.T) {
	data := enclaveWorkflowData(true, true, 30, 120)
	data.Enclaves[1].Repos[0].Sensitivity = "sealed"
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must use the same sensitivity across enclave types")
}

func TestValidateEnclavesRequiresAgentModelOnly(t *testing.T) {
	data := enclaveWorkflowData(false, true, 0, 120)
	data.Enclaves[0].Agent.Model = ""
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent.model is required")

	script := enclaveWorkflowData(true, false, 30, 0)
	assert.NoError(t, validateEnclavesConfig(script))
}

func TestParseTopLevelKeyedEnclaves(t *testing.T) {
	config, err := ParseFrontmatterConfig(map[string]any{
		"enclaves": []any{
			map[string]any{
				"script": nil,
				"repos": []any{
					map[string]any{"repo": "octo-org/private-service", "sensitivity": "confidential"},
				},
				"timeout": 45,
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, config.Enclaves, 1)
	require.NotNil(t, config.Enclaves[0].Script)
	assert.Equal(t, 45, config.Enclaves[0].Timeout)
	require.Len(t, config.Enclaves[0].Repos, 1)
}

func TestEnclaveConfigRejectsAmbiguousDiscriminator(t *testing.T) {
	data := enclaveWorkflowData(false, false, 0, 0)
	data.Enclaves = EnclavesConfig{{
		Script: &ScriptEnclaveConfig{},
		Agent:  &AgentEnclaveConfig{Model: "gpt-5"},
		Repos:  enclaveTestRepos(),
	}}
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of script or agent")
}

func TestBuildAWFConfigJSONEnclaves(t *testing.T) {
	data := enclaveWorkflowData(true, true, 45, 180)
	configJSON, err := BuildAWFConfigJSON(AWFCommandConfig{
		EngineName: "copilot", WorkflowData: data,
	})
	require.NoError(t, err)

	var config map[string]any
	require.NoError(t, json.Unmarshal([]byte(configJSON), &config))
	enclaves := config["enclaves"].([]any)
	script := enclaves[0].(map[string]any)
	agent := enclaves[1].(map[string]any)
	scriptConfig := script["script"].(map[string]any)
	agentConfig := agent["agent"].(map[string]any)
	assert.Empty(t, scriptConfig)
	assert.InDelta(t, 45, script["timeout"], 0)
	assert.Equal(t, "gpt-5", agentConfig["model"])
	assert.Contains(t, script, "repos")
	assert.NotContains(t, script, "enabled")
	assert.NotContains(t, script, "network")
	assert.NotContains(t, script, "interpreter")
	assert.Equal(t, []any{"awmg-mcpg"}, config["network"].(map[string]any)["topologyAttach"])
	assert.NotContains(t, configJSON, "boundedQueries")
	assert.NotContains(t, configJSON, "boundedAgents")
}

func TestBuildAWFConfigJSONEnclaveGitHubIssues(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	configJSON, err := BuildAWFConfigJSON(AWFCommandConfig{
		EngineName: "copilot", WorkflowData: data,
	})
	require.NoError(t, err)

	var config map[string]any
	require.NoError(t, json.Unmarshal([]byte(configJSON), &config))
	enclaves := config["enclaves"].([]any)
	agent := enclaves[0].(map[string]any)["agent"].(map[string]any)
	assert.Equal(t, map[string]any{"cli": enclaveGitHubIssuesProfile}, agent["github"])
}

func TestValidateEnclaveGitHubIssuesRepositoryLimit(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	data.Enclaves[0].Repos = append(data.Enclaves[0].Repos, &EnclaveRepository{
		Repo: "octo-org/another-private-service", Sensitivity: "internal",
	})
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "supports at most one non-public repository")

	data.Enclaves[0].Repos[1].Sensitivity = "public"
	require.NoError(t, validateEnclavesConfig(data))
}

func TestValidateEnclaveGitHubIssuesRepositoryLimitTreatsTrustedAsPublic(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	data.Enclaves[0].Repos = []*EnclaveRepository{
		{Repo: "octo-org/trusted-service", Sensitivity: "trusted"},
		{Repo: "octo-org/public-service", Sensitivity: "public"},
	}
	require.NoError(t, validateEnclavesConfig(data))
}

func TestValidateEnclaveTrustedSensitivityRequiresAWFVersion(t *testing.T) {
	data := enclaveWorkflowData(false, true, 0, 120)
	data.Enclaves[0].Repos[0].Sensitivity = "trusted"
	data.NetworkPermissions.Firewall.Version = "v0.28.12"
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires AWF v0.28.13 or newer")
}

func TestValidateEnclaveGitHubIssuesRepositoryLimitScopesToGitHubEntry(t *testing.T) {
	data := enclaveWorkflowData(true, true, 30, 120)
	data.Enclaves[0].Repos = []*EnclaveRepository{{
		Repo: "octo-org/private-a", Sensitivity: "confidential",
	}}
	data.Enclaves[1].Agent.GitHub = &AgentEnclaveGitHubConfig{CLI: enclaveGitHubIssuesProfile}
	data.Enclaves[1].Repos = []*EnclaveRepository{{
		Repo: "octo-org/private-b", Sensitivity: "confidential",
	}}
	data.NetworkPermissions.Firewall.Version = string(constants.AWFEnclaveGitHubIssuesMinVersion)
	data.SandboxConfig.MCP = &MCPGatewayRuntimeConfig{
		Version: string(constants.MCPGEnclaveGitHubIssuesMinVersion),
	}

	require.NoError(t, validateEnclavesConfig(data))
}

func TestValidateEnclaveGitHubIssuesMode(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	data.Enclaves[0].Agent.GitHub.CLI = "read-only"
	err := validateEnclavesConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `must be "issues-read-v1"`)
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
	assert.Contains(t, generated, `"toolTimeout": 4860`)
	assert.Contains(t, generated, `"tools": ["enclave_run_script", "enclave_run_agent"]`)
	assert.Contains(t, generated, `Bearer \${AWF_ENCLAVE_MCP_CAPABILITY}`)
	assert.Contains(t, generated, `openssl rand -hex 32`)
	assert.Contains(t, generated, `::add-mask::${AWF_ENCLAVE_MCP_CAPABILITY}`)
	assert.Contains(t, generated, `printf '%s=%s\n' MCP_GATEWAY_AGENT_ID "$MCP_GATEWAY_AGENT_ID"`)
	assert.Contains(t, generated, `--network bridge`)
	assert.Contains(t, generated, `--label com.github.gh-aw.mcpg.run=`)
	assert.Contains(t, generated, `${AWF_ENCLAVE_MCP_GATEWAY_IDENTITY}`)
	assert.Contains(t, generated, `-e AWF_ENCLAVE_MCP_CAPABILITY`)
	assert.Contains(t, generated, `AWF_ENCLAVE_MCP_GATEWAY_ENDPOINT="http://localhost:${MCP_GATEWAY_PORT}/mcp/awf-enclave"`)
	assert.Contains(t, generated, `export GH_AW_MCP_DEFERRED_SERVERS="awf-enclave"`)
	assert.NotContains(t, generated, `printf '%s=%s\n' GH_AW_MCP_DEFERRED_SERVERS`)
	assert.NotContains(t, generated, `"required": false`)
	assert.NotRegexp(t, `AWF_ENCLAVE_MCP_CAPABILITY=[0-9a-f]{64}`, generated)
	gatewayCommand := strings.Index(generated, `export MCP_GATEWAY_DOCKER_COMMAND=`)
	require.Greater(t, gatewayCommand, -1)
	for _, name := range optionalPRHeadEnvVars {
		emptyDefault := `export ` + name + `="${` + name + `:-}"`
		assert.Contains(t, generated, emptyDefault)
		assert.Less(t, strings.Index(generated, emptyDefault), gatewayCommand)
	}
	gatewayKeyMask := strings.Index(generated, `::add-mask::${MCP_GATEWAY_AGENT_ID}`)
	gatewayKeyHandoff := strings.Index(generated, `printf '%s=%s\n' MCP_GATEWAY_AGENT_ID "$MCP_GATEWAY_AGENT_ID"`)
	deferred := strings.Index(generated, `export GH_AW_MCP_DEFERRED_SERVERS="awf-enclave"`)
	gatewayRunner := strings.Index(generated, `| "$GH_AW_NODE" "${RUNNER_TEMP}/gh-aw/actions/start_mcp_gateway.cjs"`)
	require.Greater(t, gatewayKeyMask, -1)
	require.Greater(t, gatewayKeyHandoff, gatewayKeyMask)
	require.Greater(t, deferred, -1)
	require.Greater(t, gatewayRunner, deferred)

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
    version: latest
enclaves:
  - script:
    repos:
      - repo: octo-org/private-service
        sensitivity: confidential
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
	gatewayKeyHandoff := strings.Index(lock, `printf '%s=%s\n' MCP_GATEWAY_AGENT_ID "$MCP_GATEWAY_AGENT_ID"`)
	deferred := strings.Index(lock, `export GH_AW_MCP_DEFERRED_SERVERS="awf-enclave"`)
	awf := strings.Index(lock, "awf --config")
	require.Greater(t, gateway, -1)
	require.Greater(t, gatewayKeyHandoff, gateway)
	require.Greater(t, deferred, gateway)
	require.Greater(t, awf, -1)
	assert.Less(t, gateway, awf)
	assert.Less(t, gatewayKeyHandoff, awf)
	assert.Less(t, deferred, awf)
	assert.Contains(t, lock, `"awf-enclave"`)
	assert.NotContains(t, lock, `"required": false`)
	assert.Contains(t, lock, "--exclude-env MCP_GATEWAY_AGENT_ID")
	if mountStart := strings.Index(lock, "- name: Mount MCP servers as CLIs"); mountStart >= 0 {
		mountEnd := strings.Index(lock[mountStart:], "\n      - name:")
		require.Positive(t, mountEnd)
		assert.NotContains(t, lock[mountStart:mountStart+mountEnd], "steps.start-mcp-gateway.outputs.gateway-agent-id")
	}
	assert.Contains(t, lock, `\"enclaves\":[{\"repos\":[{\"repo\":\"octo-org/private-service\",\"sensitivity\":\"confidential\"}],\"script\":{},\"timeout\":45}]`)
	assert.NotContains(t, lock, "Start Enclave MCP")
	assert.NotContains(t, lock, "start_enclave")
}
