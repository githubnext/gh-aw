package workflow

import (
	"encoding/json"
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

func TestBuildEnclaveGitHubProxyPolicyJSON(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	data.Enclaves[0].Repos = append(data.Enclaves[0].Repos, &EnclaveRepository{
		Repo: "octo-org/public-docs", Sensitivity: "public",
	})

	policyJSON, err := buildEnclaveGitHubProxyPolicyJSON(data, "123456")
	require.NoError(t, err)
	assert.Contains(t, policyJSON, `"version":1`)
	assert.Contains(t, policyJSON, `"audience":"gh-aw-enclave-github"`)
	assert.Contains(t, policyJSON, `"repositories":[`)
	assert.Contains(t, policyJSON, `"max_capability_ttl_seconds":600`)
	assert.NotContains(t, policyJSON, "assigned_repositories")
	assert.NotContains(t, policyJSON, "max_ttl_seconds")

	var policy enclaveGitHubProxyPolicy
	require.NoError(t, json.Unmarshal([]byte(policyJSON), &policy))
	assert.Equal(t, 1, policy.Version)
	assert.Equal(t, "123456", policy.WorkflowRunID)
	assert.Equal(t, enclaveGitHubIssuesProfile, policy.Profile)
	assert.Equal(t, enclaveGitHubProxyAudience, policy.Audience)
	assert.Equal(t, "approved", policy.PublicMinimumIntegrity)
	assert.Equal(t, []string{"issues.comments.list", "issues.get", "issues.list"}, policy.AllowedOperations)
	assert.Equal(t, enclaveGitHubProxyMaxTTL, policy.MaxCapabilityTTL)
	assert.Equal(t, []enclaveGitHubPolicyRepository{
		{Repo: "octo-org/private-service", Sensitivity: "confidential"},
		{Repo: "octo-org/public-docs", Sensitivity: "public"},
	}, policy.Repositories)
}

func TestEnclaveGitHubProxyPolicyInheritsPrimaryIntegrity(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	data.Tools["github"] = map[string]any{"min-integrity": "merged"}

	policyJSON, err := buildEnclaveGitHubProxyPolicyJSON(data, "123456")
	require.NoError(t, err)
	assert.Contains(t, policyJSON, `"public_min_integrity":"merged"`)

	data.Tools = map[string]any{}
	data.ParsedTools = &ToolsConfig{
		GitHub: &GitHubToolConfig{MinIntegrity: GitHubIntegrityUnapproved},
	}
	policyJSON, err = buildEnclaveGitHubProxyPolicyJSON(data, "123456")
	require.NoError(t, err)
	assert.Contains(t, policyJSON, `"public_min_integrity":"unapproved"`)

	data.Tools["github"] = map[string]any{"min-integrity": "merged"}
	policyJSON, err = buildEnclaveGitHubProxyPolicyJSON(data, "123456")
	require.NoError(t, err)
	assert.Contains(t, policyJSON, `"public_min_integrity":"unapproved"`)
}

func TestGenerateEnclaveGitHubProxySetup(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	var yaml strings.Builder
	require.NoError(t, (&Compiler{}).generateStartEnclaveGitHubProxyStep(&yaml, data))
	generated := yaml.String()

	assert.Contains(t, generated, "- name: Start Enclave GitHub Proxy")
	assert.Contains(t, generated, "GH_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN }}")
	assert.Contains(t, generated, enclaveGitHubProxyAliasEnv+": "+enclaveGitHubProxyNetworkAlias)
	assert.Contains(t, generated, "ENCLAVE_GITHUB_PROXY_POLICY_TEMPLATE:")
	assert.Contains(t, generated, `"allowed_operations":["issues.comments.list","issues.get","issues.list"]`)
	assert.Contains(t, generated, "start_enclave_github_proxy.sh")
	assert.NotContains(t, generated, enclaveGitHubProxyRootKeyEnv+":")
	assert.NotContains(t, generated, enclaveGitHubProxyContainerEnv+":")
}

func TestGenerateEnclaveGitHubProxyStopAlwaysRuns(t *testing.T) {
	data := enclaveGitHubIssuesWorkflowData()
	var yaml strings.Builder
	(&Compiler{}).generateStopEnclaveGitHubProxyStep(&yaml, data)
	generated := yaml.String()

	assert.Contains(t, generated, "- name: Stop Enclave GitHub Proxy")
	assert.Contains(t, generated, "if: always()")
	assert.Contains(t, generated, "continue-on-error: true")
	assert.Contains(t, generated, "stop_enclave_github_proxy.sh")
}

func TestEnclaveGitHubProxyScriptsEnforceDedicatedBridgeContract(t *testing.T) {
	startScript, err := os.ReadFile(filepath.Join("..", "..", "actions", "setup", "sh", "start_enclave_github_proxy.sh"))
	require.NoError(t, err)
	start := string(startScript)

	keyGeneration := strings.Index(start, "openssl rand -hex 32")
	keyMask := strings.Index(start, "::add-mask::${MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY}")
	containerStart := strings.Index(start, "docker run -d")
	require.GreaterOrEqual(t, keyGeneration, 0)
	require.Greater(t, keyMask, keyGeneration)
	require.Greater(t, containerStart, keyMask)

	assert.Contains(t, start, `^[0-9a-f]{64}$`)
	assert.Contains(t, start, "--network bridge")
	assert.NotContains(t, start, "\n  -p ")
	assert.Contains(t, start, `com.github.gh-aw.enclave-github.run`)
	assert.Contains(t, start, `openssl dgst -sha256 -r`)
	assert.Contains(t, start, `PROXY_IDENTITY="gh-aw-egh-${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}-${JOB_HASH}"`)
	assert.Contains(t, start, `^[a-z0-9][a-z0-9-]{0,63}$`)
	assert.NotContains(t, start, `-${GITHUB_JOB}"`)
	assert.Contains(t, start, `--arg workflow_run_id "$PROXY_IDENTITY"`)
	assert.NotContains(t, start, `--arg workflow_run_id "$GITHUB_RUN_ID"`)
	assert.NotContains(t, start, "--enclave-profile")
	assert.Contains(t, start, `PORT="18443"`)
	assert.Contains(t, start, `MCP_LOG_DIR="${RUNNER_TEMP:-/tmp}/gh-aw/enclave-github-proxy-logs"`)
	assert.Contains(t, start, `-e MCP_GATEWAY_ENCLAVE_POLICY_JSON`)
	assert.Contains(t, start, `-e MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY`)
	assert.Contains(t, start, `rm -rf "${MCP_LOG_DIR}/proxy-tls"`)
	assert.NotContains(t, start, "--policy")
	assert.NotContains(t, start, "--tls-dir")
	assert.Contains(t, start, `PROXY_ALIAS="${ENCLAVE_GITHUB_PROXY_ALIAS:-}"`)
	assert.Contains(t, start, `--tls-dns-name "$PROXY_ALIAS"`)
	assert.Equal(t, "awf-enclave-github-proxy", enclaveGitHubProxyNetworkAlias)
	assert.Contains(t, start, `proxy-tls/ca.crt`)
	assert.Contains(t, start, `AWF_ENCLAVE_GITHUB_PROXY_CONTAINER`)
	assert.Contains(t, start, `AWF_ENCLAVE_GITHUB_PROXY_IDENTITY`)
	assert.Contains(t, start, `AWF_ENCLAVE_GITHUB_PROXY_CA_CERT`)
	assert.Contains(t, start, `MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY`)
	assert.Contains(t, start, `::add-mask::${POLICY_TEMPLATE}`)
	assert.Contains(t, start, `::add-mask::${MCP_GATEWAY_ENCLAVE_POLICY_JSON}`)
	assert.Contains(t, start, `--resolve "${PROXY_ALIAS}:${PORT}:${PROXY_IP}"`)
	assert.Contains(t, start, `--cacert "$CA_CERT"`)
	assert.NotContains(t, start, `curl -skf`)

	stopScript, err := os.ReadFile(filepath.Join("..", "..", "actions", "setup", "sh", "stop_enclave_github_proxy.sh"))
	require.NoError(t, err)
	assert.Contains(t, string(stopScript), "docker rm -f awmg-enclave-github-proxy")
	assert.Contains(t, string(stopScript), `MCP_LOG_DIR="${RUNNER_TEMP:-/tmp}/gh-aw/enclave-github-proxy-logs"`)
	assert.NotContains(t, string(stopScript), "/tmp/gh-aw/enclave-github-proxy-logs")
	assert.Contains(t, string(stopScript), "MCP_GATEWAY_ENCLAVE_CAPABILITY_KEY=\\n")
}

func TestEnclaveGitHubProxyVersionGates(t *testing.T) {
	t.Run("minimum versions accepted", func(t *testing.T) {
		data := enclaveGitHubIssuesWorkflowData()
		data.NetworkPermissions.Firewall.Version = string(constants.AWFEnclaveGitHubIssuesMinVersion)
		require.NoError(t, validateEnclavesConfig(data))
	})

	t.Run("old AWF rejected", func(t *testing.T) {
		data := enclaveGitHubIssuesWorkflowData()
		data.NetworkPermissions.Firewall.Version = "v0.28.5"
		err := validateEnclavesConfig(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), string(constants.AWFEnclaveGitHubIssuesMinVersion))
	})

	t.Run("old MCPG rejected", func(t *testing.T) {
		data := enclaveGitHubIssuesWorkflowData()
		data.SandboxConfig.MCP.Version = "v0.4.10"
		err := validateEnclavesConfig(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), string(constants.MCPGEnclaveGitHubIssuesMinVersion))
	})

	t.Run("nil MCP config uses supporting default", func(t *testing.T) {
		data := enclaveGitHubIssuesWorkflowData()
		data.SandboxConfig.MCP = nil
		require.NoError(t, validateEnclavesConfig(data))
	})

	t.Run("omitted profile keeps compatibility", func(t *testing.T) {
		data := enclaveWorkflowData(false, true, 0, 120)
		data.NetworkPermissions.Firewall.Version = "v0.25.3"
		data.SandboxConfig.MCP = &MCPGatewayRuntimeConfig{Version: "v0.1.0"}
		require.NoError(t, validateEnclavesConfig(data))
	})
}

func TestEnclaveGitHubProxyEnvironmentExclusions(t *testing.T) {
	excluded := ComputeAWFExcludeEnvVarNames(enclaveGitHubIssuesWorkflowData(), nil)
	assert.Contains(t, excluded, enclaveGitHubProxyContainerEnv)
	assert.Contains(t, excluded, enclaveGitHubProxyIdentityEnv)
	assert.Contains(t, excluded, enclaveGitHubProxyCACertEnv)
	assert.Contains(t, excluded, enclaveGitHubProxyRootKeyEnv)
	assert.NotContains(t, excluded, enclaveGitHubProxyPolicyEnv)
}

func TestCompileEnclaveGitHubProxyLifecycle(t *testing.T) {
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
    version: v0.4.13
enclaves:
  - agent:
      model: gpt-5
      github:
        cli: issues-read-v1
    repos:
      - repo: octo-org/private-service
        sensitivity: confidential
    timeout: 120
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

	imageTag := strings.TrimPrefix(string(constants.DefaultFirewallVersion), "v")
	for _, imageName := range []string{"enclave-agent", "enclave-mcp-server"} {
		image := constants.DefaultFirewallRegistry + "/" + imageName + ":" + imageTag
		pin, ok := getEmbeddedContainerPin(image)
		require.True(t, ok, "expected embedded pin for %s", image)
		pinnedImage := pin.Image + "@" + pin.Digest
		assert.Contains(t, lock, `"image":"`+pin.Image+`","digest":"`+pin.Digest+`","pinned_image":"`+pinnedImage+`"`)
		assert.Contains(t, lock, "#   - "+pinnedImage)
		assert.NotContains(t, lock, `"repo":"`+constants.DefaultFirewallRegistry+`/`+imageName+`","ref":"`+imageTag+`","error_type":"container_pin_not_found"`)
	}

	proxyStart := strings.Index(lock, "- name: Start Enclave GitHub Proxy")
	gatewayStart := strings.Index(lock, "- name: Start MCP Gateway")
	gatewayKeyHandoff := strings.Index(lock, `printf '%s=%s\n' MCP_GATEWAY_API_KEY "$MCP_GATEWAY_API_KEY"`)
	awf := strings.Index(lock, "awf --config")
	proxyStop := strings.Index(lock, "- name: Stop Enclave GitHub Proxy")
	require.GreaterOrEqual(t, proxyStart, 0)
	require.Greater(t, gatewayStart, proxyStart)
	require.Greater(t, gatewayKeyHandoff, gatewayStart)
	require.Greater(t, awf, gatewayStart)
	require.Less(t, gatewayKeyHandoff, awf)
	require.Greater(t, proxyStop, awf)

	assert.Contains(t, lock, `\"github\":{\"cli\":\"issues-read-v1\"}`)
	assert.Contains(t, lock, "ENCLAVE_GITHUB_PROXY_POLICY_TEMPLATE:")
	assert.NotContains(t, lock, enclaveGitHubProxyPolicyEnv+":")
	assert.Contains(t, lock, "--exclude-env "+enclaveGitHubProxyContainerEnv)
	assert.Contains(t, lock, "--exclude-env "+enclaveGitHubProxyIdentityEnv)
	assert.Contains(t, lock, "--exclude-env "+enclaveGitHubProxyCACertEnv)
	assert.Contains(t, lock, "--exclude-env "+enclaveGitHubProxyRootKeyEnv)
	assert.Contains(t, lock, "--exclude-env MCP_GATEWAY_API_KEY")
	mountStart := strings.Index(lock, "- name: Mount MCP servers as CLIs")
	require.GreaterOrEqual(t, mountStart, 0)
	mountEnd := strings.Index(lock[mountStart:], "\n      - name:")
	require.Positive(t, mountEnd)
	assert.NotContains(t, lock[mountStart:mountStart+mountEnd], "steps.start-mcp-gateway.outputs.gateway-api-key")
	stopStart := strings.Index(lock, "- name: Stop MCP Gateway")
	require.GreaterOrEqual(t, stopStart, 0)
	stopEnd := strings.Index(lock[stopStart:], "\n      - name:")
	require.Positive(t, stopEnd)
	assert.NotContains(t, lock[stopStart:stopStart+stopEnd], "steps.start-mcp-gateway.outputs.gateway-api-key")
}
