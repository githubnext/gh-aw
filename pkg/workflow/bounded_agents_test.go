package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/constants"
)

func TestBoundedAgentsConfig(t *testing.T) {
	timeout, pids, output, task, invocations := 120, 128, 8192, 4096, 8
	config := &BoundedAgentsConfig{
		PrivateRepos: []*BoundedAgentPrivateRepo{{Repo: "my-org/internal-service", Sensitivity: "internal"}},
		Runtime:      "gvisor", Engine: "copilot", Model: "gpt-4o-mini",
		Timeout: &timeout, MemoryLimit: "512m", CPULimit: "1", PidsLimit: &pids, TmpfsLimit: "64m",
		MaxOutputBytes: &output, MaxTaskBytes: &task, MaxInvocations: &invocations,
	}
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Version: string(constants.AWFBoundedAgentsMinVersion)}},
		ParsedTools:   &ToolsConfig{GitHub: &GitHubToolConfig{BoundedAgents: config}},
	}
	require.NoError(t, validateBoundedAgentsConfig(workflowData))

	extracted := extractBoundedAgentsConfig(workflowData)
	require.NotNil(t, extracted)
	assert.Equal(t, "copilot", extracted.Engine)
	assert.Equal(t, "gpt-4o-mini", extracted.Model)

	jsonConfig, err := BuildAWFConfigJSON(AWFCommandConfig{WorkflowData: workflowData})
	require.NoError(t, err)
	var actual map[string]any
	require.NoError(t, json.Unmarshal([]byte(jsonConfig), &actual))
	bounded := actual["boundedAgents"].(map[string]any)
	assert.Equal(t, true, bounded["enabled"])
	assert.Equal(t, "copilot", bounded["engine"])
	assert.Equal(t, "gpt-4o-mini", bounded["model"])
	assert.Contains(t, actual["apiProxy"].(map[string]any)["targets"].(map[string]any), "copilot")
}

func TestBoundedAgentsRejectUnsafeConfiguration(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf", Version: string(constants.AWFBoundedAgentsMinVersion)}},
		ParsedTools: &ToolsConfig{GitHub: &GitHubToolConfig{BoundedAgents: &BoundedAgentsConfig{
			PrivateRepos: []*BoundedAgentPrivateRepo{{Repo: "Org/Repo", Sensitivity: "${{ inputs.sensitivity }}"}},
			Runtime:      "sbx", Engine: "claude", Model: "${{ inputs.model }}",
		}}},
	}
	err := validateBoundedAgentsConfig(workflowData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sensitivity")
}

func TestParseBoundedAgentsConfig(t *testing.T) {
	config := parseGitHubTool(map[string]any{
		"bounded-agents": map[string]any{
			"private-repos": []any{map[string]any{
				"repo":        "my-org/internal-service",
				"sensitivity": "internal",
			}},
			"engine": "copilot",
			"model":  "gpt-4o-mini",
		},
	})

	require.NotNil(t, config.BoundedAgents)
	assert.Equal(t, "copilot", config.BoundedAgents.Engine)
	assert.Equal(t, "gpt-4o-mini", config.BoundedAgents.Model)
	require.Len(t, config.BoundedAgents.PrivateRepos, 1)
	assert.Equal(t, "my-org/internal-service", config.BoundedAgents.PrivateRepos[0].Repo)
}
