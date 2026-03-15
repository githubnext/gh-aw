//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeriveWriteSinkGuardPolicyFromWorkflow tests the helper that derives guard policies from workflow data
func TestDeriveWriteSinkGuardPolicyFromWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expectNil    bool
		description  string
		expectedKey  string
	}{
		{
			name:         "nil workflow data",
			workflowData: nil,
			expectNil:    true,
			description:  "nil workflowData should return nil",
		},
		{
			name:         "nil tools",
			workflowData: &WorkflowData{},
			expectNil:    true,
			description:  "no tools should return nil",
		},
		{
			name: "no github tool",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"playwright": map[string]any{},
				},
			},
			expectNil:   true,
			description: "no github tool means no guard policy",
		},
		{
			name: "github tool without guard policy",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"toolsets": []string{"default"},
					},
				},
			},
			expectNil:   true,
			description: "github tool without repos/min-integrity has no guard policy",
		},
		{
			name: "github tool with repos=all",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"repos":         "all",
						"min-integrity": "none",
					},
				},
			},
			expectNil:   false,
			expectedKey: "write-sink",
			description: "github guard policy with repos=all should produce write-sink policy",
		},
		{
			name: "github tool with specific repo",
			workflowData: &WorkflowData{
				Tools: map[string]any{
					"github": map[string]any{
						"repos":         "myorg/myrepo",
						"min-integrity": "approved",
					},
				},
			},
			expectNil:   false,
			expectedKey: "write-sink",
			description: "github guard policy with specific repo should produce write-sink policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveWriteSinkGuardPolicyFromWorkflow(tt.workflowData)
			if tt.expectNil {
				assert.Nil(t, result, "Expected nil result for: %s", tt.description)
			} else {
				require.NotNil(t, result, "Expected non-nil result for: %s", tt.description)
				assert.Contains(t, result, tt.expectedKey, "Expected write-sink key in policies for: %s", tt.description)
			}
		})
	}
}

// TestRenderSharedMCPConfigWithGuardPoliciesJSON tests that guard policies are rendered correctly in JSON format
func TestRenderCustomToolWithGuardPoliciesJSON(t *testing.T) {
	guardPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"*"},
		},
	}

	toolConfig := map[string]any{
		"url": "https://example.com/mcp",
	}

	var output strings.Builder
	renderer := MCPConfigRenderer{
		IndentLevel:   "                ",
		Format:        "json",
		GuardPolicies: guardPolicies,
	}

	err := renderSharedMCPConfig(&output, "my-tool", toolConfig, renderer)
	require.NoError(t, err, "renderSharedMCPConfig should succeed")

	result := output.String()
	// The url field should have a trailing comma (guard policies follow)
	assert.Contains(t, result, "\"url\": \"https://example.com/mcp\",", "url field should have trailing comma")
	// Guard policies should be rendered
	assert.Contains(t, result, "\"guard-policies\"", "guard-policies should be rendered")
	assert.Contains(t, result, "\"write-sink\"", "write-sink should be rendered")
	assert.Contains(t, result, "\"accept\"", "accept should be rendered")
}

// TestRenderSharedMCPConfigWithGuardPoliciesTOML tests that guard policies are rendered correctly in TOML format
func TestRenderCustomToolWithGuardPoliciesTOML(t *testing.T) {
	guardPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"private:myorg/myrepo"},
		},
	}

	toolConfig := map[string]any{
		"url": "https://example.com/mcp",
	}

	var output strings.Builder
	renderer := MCPConfigRenderer{
		IndentLevel:   "          ",
		Format:        "toml",
		GuardPolicies: guardPolicies,
	}

	err := renderSharedMCPConfig(&output, "my-tool", toolConfig, renderer)
	require.NoError(t, err, "renderSharedMCPConfig should succeed")

	result := output.String()
	// TOML guard policies are in separate sections
	assert.Contains(t, result, "[mcp_servers.my-tool.\"guard-policies\"]", "TOML guard-policies section should be present")
	assert.Contains(t, result, "write-sink", "write-sink should be rendered")
	assert.Contains(t, result, "accept", "accept should be rendered")
	assert.Contains(t, result, "\"private:myorg/myrepo\"", "accept pattern should be rendered")
}

// TestRenderSharedMCPConfigWithoutGuardPoliciesJSON tests that when no guard policies are set, no comma is added
func TestRenderCustomToolWithoutGuardPoliciesJSON(t *testing.T) {
	toolConfig := map[string]any{
		"url": "https://example.com/mcp",
	}

	var output strings.Builder
	renderer := MCPConfigRenderer{
		IndentLevel: "                ",
		Format:      "json",
		// No GuardPolicies set
	}

	err := renderSharedMCPConfig(&output, "my-tool", toolConfig, renderer)
	require.NoError(t, err, "renderSharedMCPConfig should succeed")

	result := output.String()
	// The url field should NOT have a trailing comma (it's the last field)
	assert.NotContains(t, result, "\"url\": \"https://example.com/mcp\",", "url field should not have trailing comma")
	// No guard policies
	assert.NotContains(t, result, "guard-policies", "guard-policies should not be rendered")
}

// TestPlaywrightMCPWithGuardPoliciesJSON tests that playwright gets write-sink guard policies in JSON format
func TestPlaywrightMCPWithGuardPoliciesJSON(t *testing.T) {
	guardPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"*"},
		},
	}

	var output strings.Builder
	renderPlaywrightMCPConfigWithOptions(&output, nil, true, false, false, guardPolicies)

	result := output.String()
	assert.Contains(t, result, "\"guard-policies\"", "playwright should have guard-policies in JSON")
	assert.Contains(t, result, "\"write-sink\"", "playwright should have write-sink in JSON")
}

// TestPlaywrightMCPWithoutGuardPoliciesJSON tests that playwright without guard policies is unchanged
func TestPlaywrightMCPWithoutGuardPoliciesJSON(t *testing.T) {
	var output strings.Builder
	renderPlaywrightMCPConfigWithOptions(&output, nil, true, false, false, nil)

	result := output.String()
	assert.NotContains(t, result, "guard-policies", "playwright without guard policies should not have guard-policies")
}

// TestSerenaMCPWithGuardPoliciesJSON tests that serena gets write-sink guard policies in JSON format
func TestSerenaMCPWithGuardPoliciesJSON(t *testing.T) {
	guardPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"private:myorg"},
		},
	}

	var output strings.Builder
	renderSerenaMCPConfigWithOptions(&output, nil, true, false, false, guardPolicies)

	result := output.String()
	assert.Contains(t, result, "\"guard-policies\"", "serena should have guard-policies in JSON")
	assert.Contains(t, result, "\"write-sink\"", "serena should have write-sink in JSON")
	assert.Contains(t, result, "\"private:myorg\"", "serena should have accept pattern")
}

// TestMCPScriptsMCPWithGuardPoliciesJSON tests that mcp-scripts gets write-sink guard policies in JSON format
func TestMCPScriptsMCPWithGuardPoliciesJSON(t *testing.T) {
	guardPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"*"},
		},
	}

	var output strings.Builder
	renderMCPScriptsMCPConfigWithOptions(&output, nil, true, false, nil, guardPolicies)

	result := output.String()
	assert.Contains(t, result, "\"guard-policies\"", "mcp-scripts should have guard-policies in JSON")
	assert.Contains(t, result, "\"write-sink\"", "mcp-scripts should have write-sink in JSON")
	// The headers section should have a trailing comma
	assert.Contains(t, result, "},\n", "headers closing brace should have trailing comma when guard policies follow")
}

// TestAgenticWorkflowsMCPWithGuardPoliciesJSON tests that agentic-workflows gets write-sink guard policies in JSON format
func TestAgenticWorkflowsMCPWithGuardPoliciesJSON(t *testing.T) {
	guardPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"*"},
		},
	}

	var output strings.Builder
	renderAgenticWorkflowsMCPConfigWithOptions(&output, true, false, ActionModeRelease, guardPolicies)

	result := output.String()
	assert.Contains(t, result, "\"guard-policies\"", "agentic-workflows should have guard-policies in JSON")
	assert.Contains(t, result, "\"write-sink\"", "agentic-workflows should have write-sink in JSON")
}

// TestWebFetchMCPWithGuardPoliciesJSON tests that web-fetch gets write-sink guard policies in JSON format
func TestWebFetchMCPWithGuardPoliciesJSON(t *testing.T) {
	guardPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"*"},
		},
	}

	var output strings.Builder
	renderMCPFetchServerConfig(&output, "json", "              ", true, false, guardPolicies)

	result := output.String()
	assert.Contains(t, result, "\"guard-policies\"", "web-fetch should have guard-policies in JSON")
	assert.Contains(t, result, "\"write-sink\"", "web-fetch should have write-sink in JSON")
	// container should have trailing comma
	assert.Contains(t, result, "\"container\": \"mcp/fetch\",", "container field should have trailing comma when guard policies follow")
}

// TestWebFetchMCPWithGuardPoliciesTOML tests that web-fetch gets write-sink guard policies in TOML format
func TestWebFetchMCPWithGuardPoliciesTOML(t *testing.T) {
	guardPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"*"},
		},
	}

	var output strings.Builder
	renderMCPFetchServerConfig(&output, "toml", "          ", false, false, guardPolicies)

	result := output.String()
	assert.Contains(t, result, "guard-policies", "web-fetch TOML should have guard-policies section")
	assert.Contains(t, result, "write-sink", "web-fetch TOML should have write-sink")
	assert.Contains(t, result, "accept", "web-fetch TOML should have accept")
}

// TestAllNonGitHubMCPServersGetGuardPoliciesViaRenderer tests that the MCPConfigRendererUnified
// propagates WriteSinkGuardPolicies to all non-GitHub MCP server render methods
func TestAllNonGitHubMCPServersGetGuardPoliciesViaRenderer(t *testing.T) {
	guardPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"*"},
		},
	}

	t.Run("playwright JSON", func(t *testing.T) {
		renderer := NewMCPConfigRenderer(MCPRendererOptions{
			Format:                 "json",
			IsLast:                 true,
			WriteSinkGuardPolicies: guardPolicies,
		})
		var output strings.Builder
		renderer.RenderPlaywrightMCP(&output, nil)
		assert.Contains(t, output.String(), "guard-policies", "playwright JSON should have guard-policies")
	})

	t.Run("playwright TOML", func(t *testing.T) {
		renderer := NewMCPConfigRenderer(MCPRendererOptions{
			Format:                 "toml",
			WriteSinkGuardPolicies: guardPolicies,
		})
		var output strings.Builder
		renderer.RenderPlaywrightMCP(&output, nil)
		assert.Contains(t, output.String(), "[mcp_servers.playwright.\"guard-policies\"]", "playwright TOML should have guard-policies section")
	})

	t.Run("serena JSON", func(t *testing.T) {
		renderer := NewMCPConfigRenderer(MCPRendererOptions{
			Format:                 "json",
			IsLast:                 true,
			WriteSinkGuardPolicies: guardPolicies,
		})
		var output strings.Builder
		renderer.RenderSerenaMCP(&output, nil)
		assert.Contains(t, output.String(), "guard-policies", "serena JSON should have guard-policies")
	})

	t.Run("serena TOML", func(t *testing.T) {
		renderer := NewMCPConfigRenderer(MCPRendererOptions{
			Format:                 "toml",
			WriteSinkGuardPolicies: guardPolicies,
		})
		var output strings.Builder
		renderer.RenderSerenaMCP(&output, nil)
		assert.Contains(t, output.String(), "[mcp_servers.serena.\"guard-policies\"]", "serena TOML should have guard-policies section")
	})

	t.Run("agentic-workflows JSON", func(t *testing.T) {
		renderer := NewMCPConfigRenderer(MCPRendererOptions{
			Format:                 "json",
			IsLast:                 true,
			WriteSinkGuardPolicies: guardPolicies,
		})
		var output strings.Builder
		renderer.RenderAgenticWorkflowsMCP(&output)
		assert.Contains(t, output.String(), "guard-policies", "agentic-workflows JSON should have guard-policies")
	})

	t.Run("agentic-workflows TOML", func(t *testing.T) {
		renderer := NewMCPConfigRenderer(MCPRendererOptions{
			Format:                 "toml",
			WriteSinkGuardPolicies: guardPolicies,
		})
		var output strings.Builder
		renderer.RenderAgenticWorkflowsMCP(&output)
		result := output.String()
		// The TOML section ID for agentic-workflows uses the constant
		assert.Contains(t, result, "guard-policies", "agentic-workflows TOML should have guard-policies")
	})
}

// TestNonGitHubMCPServersNoGuardPoliciesWhenGitHubNotConfigured verifies that servers
// do not get guard policies when the GitHub tool has no guard policy configured
func TestNonGitHubMCPServersNoGuardPoliciesWhenGitHubNotConfigured(t *testing.T) {
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"github": map[string]any{
				"toolsets": []string{"default"},
			},
			"playwright": nil,
		},
	}

	policies := deriveWriteSinkGuardPolicyFromWorkflow(workflowData)
	assert.Nil(t, policies, "no guard policies when GitHub has no guard policy configured")

	// Verify playwright JSON rendering has no guard-policies
	var output strings.Builder
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		Format:                 "json",
		IsLast:                 true,
		WriteSinkGuardPolicies: policies,
	})
	renderer.RenderPlaywrightMCP(&output, nil)
	assert.NotContains(t, output.String(), "guard-policies", "playwright should not have guard-policies when GitHub has no guard policy")
}

// TestNonGitHubMCPServersGetGuardPoliciesWhenGitHubConfigured verifies the end-to-end flow:
// when GitHub has repos=all, all non-GitHub MCP servers get write-sink: {accept: ["*"]}
func TestNonGitHubMCPServersGetGuardPoliciesWhenGitHubConfigured(t *testing.T) {
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"github": map[string]any{
				"repos":         "all",
				"min-integrity": "none",
			},
			"playwright": nil,
		},
	}

	policies := deriveWriteSinkGuardPolicyFromWorkflow(workflowData)
	require.NotNil(t, policies, "guard policies should be derived when GitHub has guard policy")

	expectedPolicies := map[string]any{
		"write-sink": map[string]any{
			"accept": []string{"*"},
		},
	}
	assert.Equal(t, expectedPolicies, policies, "policies should match expected write-sink with accept=*")

	// Verify playwright JSON rendering has guard-policies
	var output strings.Builder
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		Format:                 "json",
		IsLast:                 true,
		WriteSinkGuardPolicies: policies,
	})
	renderer.RenderPlaywrightMCP(&output, nil)
	result := output.String()
	assert.Contains(t, result, "\"guard-policies\"", "playwright should have guard-policies when GitHub has guard policy")
	assert.Contains(t, result, "\"write-sink\"", "playwright should have write-sink policy")
	assert.Contains(t, result, "\"accept\"", "playwright should have accept field")
	assert.Contains(t, result, "\"*\"", "playwright should accept all patterns")
}
