//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuggieEngine(t *testing.T) {
	engine := NewAuggieEngine()
	require.NotNil(t, engine)

	assert.Equal(t, "auggie", engine.GetID())
	assert.Equal(t, "Auggie CLI", engine.GetDisplayName())
	assert.Equal(t, "Augment Code Auggie CLI (experimental)", engine.GetDescription())
	assert.True(t, engine.IsExperimental())

	capabilities := engine.GetCapabilities()
	assert.False(t, capabilities.ToolsAllowlist)
	assert.False(t, capabilities.MaxTurns)
	assert.True(t, capabilities.WebSearch)
	assert.False(t, capabilities.MaxContinuations)
	assert.False(t, capabilities.NativeAgentFile)
}

func TestAuggieEngine_GetRequiredSecretNames(t *testing.T) {
	engine := NewAuggieEngine()

	t.Run("basic", func(t *testing.T) {
		secrets := engine.GetRequiredSecretNames(&WorkflowData{Name: "test", ParsedTools: &ToolsConfig{}, Tools: map[string]any{}})
		assert.Contains(t, secrets, constants.AuggieSessionAuthEnvVar)
	})

	t.Run("with github and http mcp secrets", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test",
			ParsedTools: &ToolsConfig{
				GitHub: &GitHubToolConfig{},
			},
			Tools: map[string]any{
				"github": map[string]any{},
				"web-search": map[string]any{
					"type": "http",
					"url":  "https://example.com/mcp",
					"headers": map[string]any{
						"Authorization": "${{ secrets.TAVILY_API_KEY }}",
					},
				},
			},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, constants.AuggieSessionAuthEnvVar)
		assert.Contains(t, secrets, "MCP_GATEWAY_API_KEY")
		assert.Contains(t, secrets, "GITHUB_MCP_SERVER_TOKEN")
		assert.Contains(t, secrets, "TAVILY_API_KEY")
	})
}

func TestAuggieEngine_GetSecretValidationStep(t *testing.T) {
	engine := NewAuggieEngine()
	step := engine.GetSecretValidationStep(&WorkflowData{Name: "test"})
	stepContent := strings.Join(step, "\n")
	assert.Contains(t, stepContent, "Validate AUGMENT_SESSION_AUTH secret")
	assert.Contains(t, stepContent, "AUGMENT_SESSION_AUTH: ${{ secrets.AUGMENT_SESSION_AUTH }}")
}

func TestAuggieEngine_GetInstallationSteps(t *testing.T) {
	engine := NewAuggieEngine()

	t.Run("standard installation", func(t *testing.T) {
		steps := engine.GetInstallationSteps(&WorkflowData{Name: "test", EngineConfig: &EngineConfig{ID: "auggie"}})
		require.NotEmpty(t, steps)

		rendered := renderSteps(steps)
		assert.Contains(t, rendered, "Install Auggie CLI")
		assert.Contains(t, rendered, "@augmentcode/auggie@"+string(constants.DefaultAuggieVersion))
		assert.Contains(t, rendered, "Verify Auggie CLI installation")
		assert.Contains(t, rendered, "auggie --version")
		assert.NotContains(t, rendered, "NPM_CONFIG_MIN_RELEASE_AGE")
	})

	t.Run("custom command skips installation", func(t *testing.T) {
		steps := engine.GetInstallationSteps(&WorkflowData{
			Name:         "test",
			EngineConfig: &EngineConfig{ID: "auggie", Command: "/custom/auggie"},
		})
		assert.Empty(t, steps)
	})
}

func TestAuggieEngine_GetExecutionSteps(t *testing.T) {
	engine := NewAuggieEngine()

	t.Run("basic", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{
			Name:         "test",
			EngineConfig: &EngineConfig{ID: "auggie"},
			ParsedTools:  NewTools(map[string]any{}),
		}, "/tmp/test.log")
		require.Len(t, steps, 1)

		stepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, stepContent, "Execute Auggie CLI")
		assert.Contains(t, stepContent, "auggie")
		assert.Contains(t, stepContent, "--print")
		assert.Contains(t, stepContent, "--quiet")
		assert.Contains(t, stepContent, `${GH_AW_MODEL_AGENT_AUGGIE:+--model "$GH_AW_MODEL_AGENT_AUGGIE"}`)
		assert.Contains(t, stepContent, constants.AuggieSessionAuthEnvVar+": ${{ secrets.AUGMENT_SESSION_AUTH }}")
		assert.Contains(t, stepContent, "GH_AW_MAX_TURNS:")
		assert.Contains(t, stepContent, "GH_AW_PHASE: agent")
	})

	t.Run("with model, args, and mcp", func(t *testing.T) {
		tools := map[string]any{"github": map[string]any{}}
		steps := engine.GetExecutionSteps(&WorkflowData{
			Name: "test",
			EngineConfig: &EngineConfig{
				ID:    "auggie",
				Model: "sonnet",
				Args:  []string{"--extra-flag"},
			},
			ParsedTools: NewTools(tools),
			Tools:       tools,
		}, "/tmp/test.log")
		require.Len(t, steps, 1)

		stepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, stepContent, `--model "$GH_AW_MODEL_AGENT_AUGGIE"`)
		assert.Contains(t, stepContent, "--extra-flag")
		assert.Contains(t, stepContent, `--mcp-config "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json"`)
		assert.Contains(t, stepContent, "GH_AW_MODEL_AGENT_AUGGIE: sonnet")
		assert.Contains(t, stepContent, "GH_AW_MCP_CONFIG: ${{ runner.temp }}/gh-aw/mcp-config/mcp-servers.json")
	})
}

func TestAuggieEngine_AgentManifestMetadata(t *testing.T) {
	engine := NewAuggieEngine()
	assert.Equal(t, []string{"AGENTS.md"}, engine.GetAgentManifestFiles())
	assert.Equal(t, []string{".augment/"}, engine.GetAgentManifestPathPrefixes())
}

func TestAuggieEngine_ImplementsCodingAgentEngine(t *testing.T) {
	var _ CodingAgentEngine = NewAuggieEngine()
}

func renderSteps(steps []GitHubActionStep) string {
	var sb strings.Builder
	for _, step := range steps {
		sb.WriteString(strings.Join(step, "\n"))
		sb.WriteString("\n")
	}
	return sb.String()
}
