package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cloudHypervisorAgentAcceptanceSetupActionSHA = "26e5d7e7d2345ae4c8dfa3d9017ca2ff9bbf2c82"

func cloudHypervisorAgentAcceptanceData() *WorkflowData {
	data := cloudHypervisorEnclaveWorkflowData(false, true)
	data.Features = map[string]any{cloudHypervisorAgentAcceptanceFeature: true}
	data.EngineConfig = &EngineConfig{ID: cloudHypervisorAgentAcceptanceEngine}
	data.WorkflowID = cloudHypervisorAgentAcceptanceWorkflowID
	data.If = cloudHypervisorAgentAcceptanceCondition
	data.RawFrontmatter = map[string]any{"on": map[string]any{"workflow_dispatch": nil}}
	data.SandboxConfig.Agent.Runtime = AgentRuntimeCloudHypervisor
	data.NetworkPermissions.Firewall.Version = cloudHypervisorAgentAcceptanceAWFVersion
	enclave := data.Enclaves[0]
	enclave.Agent = &AgentEnclaveConfig{
		Engine:           cloudHypervisorAgentAcceptanceEngine,
		Model:            cloudHypervisorAgentAcceptanceModel,
		MaxTaskBytes:     cloudHypervisorAgentAcceptanceMaxTaskBytes,
		MaxModelRequests: cloudHypervisorAgentAcceptanceMaxModelRequests,
		MaxModelTokens:   cloudHypervisorAgentAcceptanceMaxModelTokens,
	}
	enclave.Repos = []*EnclaveRepository{{Repo: cloudHypervisorAgentAcceptanceRepo, Sensitivity: "public"}}
	enclave.Timeout = cloudHypervisorAgentAcceptanceTimeout
	enclave.MaxOutputBytes = cloudHypervisorAgentAcceptanceMaxOutputBytes
	enclave.MaxInvocations = cloudHypervisorAgentAcceptanceMaxInvocations
	return data
}

func TestValidateCloudHypervisorAgentAcceptance(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*WorkflowData)
		want   string
	}{
		{"acceptance", func(*WorkflowData) {}, ""},
		{"string trigger", func(d *WorkflowData) { d.RawFrontmatter["on"] = "workflow_dispatch" }, ""},
		{"wrapped condition", func(d *WorkflowData) { d.If = "${{ " + d.If + " }}" }, ""},
		{"no opt-in", func(d *WorkflowData) { d.Features = nil }, "features.cloud-hypervisor-agent-acceptance"},
		{"false opt-in", func(d *WorkflowData) { d.Features[cloudHypervisorAgentAcceptanceFeature] = false }, "features.cloud-hypervisor-agent-acceptance"},
		{"string opt-in", func(d *WorkflowData) { d.Features[cloudHypervisorAgentAcceptanceFeature] = "true" }, "features.cloud-hypervisor-agent-acceptance"},
		{"wrong identity", func(d *WorkflowData) { d.WorkflowID = "other" }, "requires workflow"},
		{"missing condition", func(d *WorkflowData) { d.If = "" }, "requires workflow"},
		{"fork condition", func(d *WorkflowData) { d.If = strings.ReplaceAll(d.If, "github/gh-aw-firewall", "octo/fork") }, "requires workflow"},
		{"tag condition", func(d *WorkflowData) { d.If = strings.ReplaceAll(d.If, "'branch'", "'tag'") }, "requires workflow"},
		{"no raw triggers", func(d *WorkflowData) { d.RawFrontmatter = nil }, "without other triggers"},
		{"scheduled", func(d *WorkflowData) { d.RawFrontmatter["on"] = "push" }, "without other triggers"},
		{"extra trigger", func(d *WorkflowData) {
			d.RawFrontmatter["on"] = map[string]any{"workflow_dispatch": nil, "push": nil}
		}, "without other triggers"},
		{"extra script", func(d *WorkflowData) {
			d.Enclaves = append(d.Enclaves, &EnclaveConfig{Script: &ScriptEnclaveConfig{}, Runtime: "cloud-hypervisor", Repos: enclaveTestRepos()})
		}, "exactly one agent enclave"},
		{"extra repo", func(d *WorkflowData) {
			d.Enclaves[0].Repos = append(d.Enclaves[0].Repos, &EnclaveRepository{Repo: "github/gh-aw", Sensitivity: "public"})
		}, "static public"},
		{"other repo", func(d *WorkflowData) { d.Enclaves[0].Repos[0].Repo = "github/gh-aw" }, "static public"},
		{"private repo", func(d *WorkflowData) { d.Enclaves[0].Repos[0].Sensitivity = "confidential" }, "static public"},
		{"trusted repo", func(d *WorkflowData) { d.Enclaves[0].Repos[0].Sensitivity = "trusted" }, "static public"},
		{"other primary engine", func(d *WorkflowData) { d.EngineConfig = &EngineConfig{ID: "codex"} }, "Copilot primary"},
		{"other enclave engine", func(d *WorkflowData) { d.Enclaves[0].Agent.Engine = "codex" }, "agent.engine: copilot"},
		{"implicit enclave engine", func(d *WorkflowData) { d.Enclaves[0].Agent.Engine = "" }, "agent.engine: copilot"},
		{"other model", func(d *WorkflowData) { d.Enclaves[0].Agent.Model = "gpt-5" }, "claude-sonnet-5"},
		{"profile", func(d *WorkflowData) { d.Enclaves[0].Agent.Profile = "custom" }, "without a profile override"},
		{"timeout", func(d *WorkflowData) { d.Enclaves[0].Timeout = 91 }, "timeout: 90"},
		{"output", func(d *WorkflowData) { d.Enclaves[0].MaxOutputBytes = 257 }, "max-output-bytes: 256"},
		{"invocations", func(d *WorkflowData) { d.Enclaves[0].MaxInvocations = 2 }, "max-invocations: 1"},
		{"task", func(d *WorkflowData) { d.Enclaves[0].Agent.MaxTaskBytes = 4097 }, "agent.max-task-bytes: 4096"},
		{"requests", func(d *WorkflowData) { d.Enclaves[0].Agent.MaxModelRequests = 9 }, "agent.max-model-requests: 8"},
		{"tokens", func(d *WorkflowData) { d.Enclaves[0].Agent.MaxModelTokens = 2049 }, "agent.max-model-tokens: 2048"},
		{"old version", func(d *WorkflowData) { d.NetworkPermissions.Firewall.Version = "v0.28.61" }, "version: v0.28.62"},
		{"latest version", func(d *WorkflowData) { d.NetworkPermissions.Firewall.Version = "latest" }, "version: v0.28.62"},
		{"branch version", func(d *WorkflowData) { d.NetworkPermissions.Firewall.Version = "main" }, "version: v0.28.62"},
		{"image", func(d *WorkflowData) { d.Enclaves[0].Image = "custom:latest" }, ".image is incompatible"},
		{"dynamic", func(d *WorkflowData) {
			dynamic := dynamicEnclaveWorkflowData().Enclaves[0]
			dynamic.Agent = d.Enclaves[0].Agent
			dynamic.Runtime = "cloud-hypervisor"
			d.Enclaves[0] = dynamic
		}, "static public"},
		{"GitHub profile", func(d *WorkflowData) {
			d.Enclaves[0].Agent.GitHub = &AgentEnclaveGitHubConfig{CLI: enclaveGitHubIssuesProfile}
		}, "scoped executor bearer handoff"},
		{"GitHub tools", func(d *WorkflowData) {
			d.Enclaves[0].Agent.Tools = &AgentEnclaveToolsConfig{GitHub: &AgentEnclaveGitHubToolConfig{Allowed: []string{"issue_read"}}}
		}, "scoped executor bearer handoff"},
		{"Docker enclave", func(d *WorkflowData) { d.Enclaves[0].Runtime = "docker" }, "cannot be mixed"},
		{"implicit enclave runtime", func(d *WorkflowData) { d.Enclaves[0].Runtime = "" }, "cannot be mixed"},
		{"DinD", func(d *WorkflowData) { d.SandboxConfig.Agent.Args = []string{"--enable-dind"} }, "--enable-dind"},
		{"Docker override", func(d *WorkflowData) { d.SandboxConfig.Agent.Args = []string{"--container-runtime=docker"} }, "requires --container-runtime cloud-hypervisor"},
		{"disabled proxy", func(d *WorkflowData) { d.SandboxConfig.Agent.Args = []string{"--no-enable-api-proxy"} }, "requires the API proxy"},
		{"missing route", func(d *WorkflowData) {
			d.EngineConfig = &EngineConfig{ID: "copilot", Env: map[string]string{"COPILOT_GITHUB_TOKEN": ""}}
		}, "configured Copilot API proxy provider route"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ambient opt-ins must not enable this acceptance path.
			t.Setenv("GH_AW_FEATURES", cloudHypervisorAgentAcceptanceFeature)
			data := cloudHypervisorAgentAcceptanceData()
			tt.mutate(data)
			err := validateEnclavesConfig(data)
			if tt.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.want)
			}
		})
	}
}

func TestCloudHypervisorAgentAcceptanceExactLimits(t *testing.T) {
	for _, field := range []string{"timeout", "output", "invocations", "task", "requests", "tokens"} {
		t.Run(field, func(t *testing.T) {
			for _, delta := range []int{-1, 1} {
				data := cloudHypervisorAgentAcceptanceData()
				enclave := data.Enclaves[0]
				switch field {
				case "timeout":
					enclave.Timeout += delta
				case "output":
					enclave.MaxOutputBytes += delta
				case "invocations":
					enclave.MaxInvocations += delta
				case "task":
					enclave.Agent.MaxTaskBytes += delta
				case "requests":
					enclave.Agent.MaxModelRequests += delta
				case "tokens":
					enclave.Agent.MaxModelTokens += delta
				}
				require.Error(t, validateEnclavesConfig(data))
			}
		})
	}
}

func TestCompileCloudHypervisorAgentAcceptance(t *testing.T) {
	content := fmt.Sprintf(`---
on:
  workflow_dispatch:
if: %s
features:
  %s: true
  action-tag: v0.92.3
strict: false
runs-on: ubuntu-24.04
network: defaults
engine:
  id: copilot
  version: 1.0.96
tools:
  github: false
sandbox:
  agent:
    runtime: cloud-hypervisor
    version: %s
  mcp:
    version: v0.4.33
enclaves:
  - agent:
      engine: %s
      model: %s
      max-task-bytes: %d
      max-model-requests: %d
      max-model-tokens: %d
    runtime: cloud-hypervisor
    repos:
      - repo: %s
        sensitivity: public
    timeout: %d
    max-output-bytes: %d
    max-invocations: %d
---
Run one enclave agent invocation.
`, cloudHypervisorAgentAcceptanceCondition,
		cloudHypervisorAgentAcceptanceFeature,
		cloudHypervisorAgentAcceptanceAWFVersion,
		cloudHypervisorAgentAcceptanceEngine,
		cloudHypervisorAgentAcceptanceModel,
		cloudHypervisorAgentAcceptanceMaxTaskBytes,
		cloudHypervisorAgentAcceptanceMaxModelRequests,
		cloudHypervisorAgentAcceptanceMaxModelTokens,
		cloudHypervisorAgentAcceptanceRepo,
		cloudHypervisorAgentAcceptanceTimeout,
		cloudHypervisorAgentAcceptanceMaxOutputBytes,
		cloudHypervisorAgentAcceptanceMaxInvocations)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, cloudHypervisorAgentAcceptanceWorkflowID+".md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	compiler := NewCompiler()
	compiler.SetStrictMode(false)
	compiler.gitRoot = tmpDir
	compiler.GetSharedActionCache().Set("github/gh-aw-actions/setup", "v0.92.3", cloudHypervisorAgentAcceptanceSetupActionSHA)
	require.NoError(t, compiler.CompileWorkflow(path))
	lockBytes, err := os.ReadFile(stringutil.MarkdownToLockFile(path))
	require.NoError(t, err)
	lock := string(lockBytes)
	for _, fragment := range []string{
		`--container-runtime cloud-hypervisor`,
		`\"runtime\":\"cloud-hypervisor\"`,
		`\"engine\":\"copilot\"`, `\"model\":\"claude-sonnet-5\"`,
		`\"maxInvocations\":1`, `\"maxOutputBytes\":256`, `\"maxTaskBytes\":4096`,
		`\"maxModelRequests\":8`, `\"maxModelTokens\":2048`,
		`\"topologyAttach\":[\"awmg-mcpg\"]`,
		`--exclude-env AWF_ENCLAVE_MCP_CAPABILITY`,
		`--exclude-env AWF_CLOUD_HYPERVISOR_ENCLAVE_AGENT_ROOTFS`,
		`GH_AW_AWF_VERSION: v0.28.62`,
		`GH_AW_AWF_VERSION: ${{ steps.cloud-hypervisor-bundle.outputs.release_tag }}`,
		"gh-aw-actions/setup@" + cloudHypervisorAgentAcceptanceSetupActionSHA,
		`1.0.96`, `v0.4.33`,
	} {
		assert.Contains(t, lock, fragment)
	}
	assert.NotContains(t, lock, "--container-runtime docker")
	bundle := strings.Index(lock, "- name: Download and verify cloud-hypervisor bundle")
	artifacts := strings.Index(lock, "- name: Download and verify cloud-hypervisor enclave artifacts")
	gateway := strings.Index(lock, "- name: Start MCP Gateway")
	awf := strings.Index(lock, "awf --config")
	require.GreaterOrEqual(t, bundle, 0)
	assert.Greater(t, artifacts, bundle)
	assert.Greater(t, gateway, artifacts)
	assert.Greater(t, awf, gateway)
}
