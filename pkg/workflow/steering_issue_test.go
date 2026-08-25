package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDefaultToolsSteerEnablesIssueMCPAccess(t *testing.T) {
	compiler := NewCompiler()
	tools := map[string]any{
		"github": map[string]any{
			"toolsets": []any{"repos"},
			"allowed":  []any{"get_file_contents"},
		},
	}

	result := compiler.applyDefaultTools(tools, steeringTestData().SafeOutputs, nil, nil)
	githubTool, ok := result["github"].(map[string]any)
	require.True(t, ok)

	toolsets := parseStringSliceAny(githubTool["toolsets"], nil)
	assert.Contains(t, toolsets, "issues")
	allowed, _ := parseGitHubAllowedToolsAndLimits(githubTool["allowed"])
	assert.Contains(t, allowed, "issue_read")
}

func TestApplyDefaultToolsSteerReEnablesGitHubMCP(t *testing.T) {
	compiler := NewCompiler()
	result := compiler.applyDefaultTools(map[string]any{"github": false}, steeringTestData().SafeOutputs, nil, nil)
	githubTool, ok := result["github"].(map[string]any)
	require.True(t, ok)

	parsed := NewTools(result)
	require.NotNil(t, parsed.GitHub)
	assert.Contains(t, ParseGitHubToolsets(parsed.GitHub.GetToolsets()), "issues")
	assert.NotContains(t, githubTool, "toolsets")
}
