package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDefaultToolsSteerEnablesPullRequestMCPAccess(t *testing.T) {
	compiler := NewCompiler()
	tools := map[string]any{
		"github": map[string]any{
			"toolsets": []any{"repos"},
			"allowed":  []any{"get_file_contents"},
		},
	}
	safeOutputs := &SafeOutputsConfig{
		CreatePullRequests: &CreatePullRequestsConfig{
			Steer: true,
		},
	}

	result := compiler.applyDefaultTools(tools, safeOutputs, nil, nil)
	githubTool, ok := result["github"].(map[string]any)
	require.True(t, ok, "github tool should be configured")

	toolsets := parseStringSliceAny(githubTool["toolsets"], nil)
	assert.Contains(t, toolsets, "repos")
	assert.Contains(t, toolsets, "pull_requests")

	allowed, _ := parseGitHubAllowedToolsAndLimits(githubTool["allowed"])
	assert.Contains(t, allowed, "get_file_contents")
	assert.Contains(t, allowed, "pull_request_read")
}

func TestApplyDefaultToolsSteerReEnablesGitHubMCP(t *testing.T) {
	compiler := NewCompiler()
	tools := map[string]any{"github": false}
	safeOutputs := &SafeOutputsConfig{
		CreatePullRequests: &CreatePullRequestsConfig{
			Steer: true,
		},
	}

	result := compiler.applyDefaultTools(tools, safeOutputs, nil, nil)
	githubTool, ok := result["github"].(map[string]any)
	require.True(t, ok, "github tool should be re-enabled for pre-create steering")

	assert.NotContains(t, githubTool, "toolsets", "default GitHub toolsets should be preserved when not explicitly configured")
	parsed := NewTools(result)
	require.NotNil(t, parsed.GitHub)
	assert.Contains(t, ParseGitHubToolsets(parsed.GitHub.GetToolsets()), "pull_requests")
}
