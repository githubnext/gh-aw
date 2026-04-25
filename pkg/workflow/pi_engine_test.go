//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPiEngine(t *testing.T) {
	engine := NewPiEngine()

	t.Run("engine identity", func(t *testing.T) {
		assert.Equal(t, "pi", engine.GetID(), "Engine ID should be 'pi'")
		assert.Equal(t, "Pi", engine.GetDisplayName(), "Display name should be 'Pi'")
		assert.NotEmpty(t, engine.GetDescription(), "Description should not be empty")
		assert.True(t, engine.IsExperimental(), "Pi engine should be experimental")
	})

	t.Run("capabilities", func(t *testing.T) {
		assert.False(t, engine.SupportsToolsAllowlist(), "Should not support tools allowlist")
		assert.False(t, engine.SupportsMaxTurns(), "Should not support max turns")
		assert.False(t, engine.SupportsWebSearch(), "Should not support built-in web search")
		assert.Equal(t, constants.PiLLMGatewayPort, engine.SupportsLLMGateway(), "Should support LLM gateway on the Pi port")
	})

	t.Run("model env var name", func(t *testing.T) {
		assert.Empty(t, engine.GetModelEnvVarName(), "Pi uses --model CLI flag, not a native env var")
	})

	t.Run("required secrets basic", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test",
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should require COPILOT_GITHUB_TOKEN for Copilot routing")
	})

	t.Run("required secrets with anthropic model", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test",
			EngineConfig: &EngineConfig{
				Model: "anthropic/claude-sonnet-4",
			},
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "ANTHROPIC_API_KEY", "Should require ANTHROPIC_API_KEY for anthropic/* models")
		assert.NotContains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should not require COPILOT_GITHUB_TOKEN for anthropic/* models")
	})

	t.Run("required secrets with openai model", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test",
			EngineConfig: &EngineConfig{
				Model: "openai/gpt-4.1",
			},
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "CODEX_API_KEY", "Should require CODEX_API_KEY for openai/* models")
		assert.Contains(t, secrets, "OPENAI_API_KEY", "Should require OPENAI_API_KEY for openai/* models")
	})

	t.Run("required secrets with copilot-requests feature", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test",
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
			Features: map[string]any{
				"copilot-requests": true,
			},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.NotContains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should not require COPILOT_GITHUB_TOKEN when copilot-requests feature is enabled")
	})

	t.Run("agent manifest files", func(t *testing.T) {
		files := engine.GetAgentManifestFiles()
		assert.Contains(t, files, "AGENTS.md", "Should include AGENTS.md as manifest file")
		assert.Contains(t, files, "CLAUDE.md", "Should include CLAUDE.md as manifest file")
	})

	t.Run("agent manifest path prefixes", func(t *testing.T) {
		prefixes := engine.GetAgentManifestPathPrefixes()
		assert.Contains(t, prefixes, ".pi/", "Should include .pi/ as manifest path prefix")
	})
}

func TestPiEngineInstallationAndExecution(t *testing.T) {
	engine := NewPiEngine()

	t.Run("standard installation", func(t *testing.T) {
		steps := engine.GetInstallationSteps(&WorkflowData{Name: "test-workflow"})
		require.NotEmpty(t, steps, "Should generate installation steps")
		stepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, stepContent, "Setup Node.js", "Should include Node setup")
	})

	t.Run("custom command skips installation", func(t *testing.T) {
		steps := engine.GetInstallationSteps(&WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/usr/local/bin/pi",
			},
		})
		assert.Empty(t, steps, "Should skip installation when custom command is specified")
	})

	t.Run("execution uses print mode and no-session", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-workflow"}, "/tmp/test.log")
		require.Len(t, steps, 1, "Pi should generate a single execution step (no separate config step needed)")

		execContent := strings.Join(steps[0], "\n")
		assert.Contains(t, execContent, "Execute Pi coding agent", "Should execute Pi coding agent")
		assert.Contains(t, execContent, "--print", "Should use print mode for non-interactive execution")
		assert.Contains(t, execContent, "--no-session", "Should disable session saving in CI")
		assert.Contains(t, execContent, "OPENAI_API_KEY: ${{ secrets.COPILOT_GITHUB_TOKEN }}", "Should default to Copilot token routing")
	})

	t.Run("model passed via CLI flag when configured", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Model: "anthropic/claude-sonnet-4",
			},
		}, "/tmp/test.log")
		require.Len(t, steps, 1, "Should generate a single execution step")

		execContent := strings.Join(steps[0], "\n")
		assert.Contains(t, execContent, "--model anthropic/claude-sonnet-4", "Should pass model via --model CLI flag")
	})

	t.Run("firewall sets Pi gateway base URL for copilot model", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Model: "copilot/gpt-5",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 1, "Should generate a single execution step")
		execContent := strings.Join(steps[0], "\n")
		assert.Contains(t, execContent, "GITHUB_COPILOT_BASE_URL: http://host.docker.internal:10002", "Should route through Copilot LLM gateway port for copilot/* models")
	})
}

func TestPiEngineProviderProfiles(t *testing.T) {
	engine := NewPiEngine()

	t.Run("anthropic model uses anthropic secret", func(t *testing.T) {
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{Model: "anthropic/claude-sonnet-4"},
			ParsedTools:  &ToolsConfig{},
			Tools:        map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "ANTHROPIC_API_KEY", "Should require ANTHROPIC_API_KEY for anthropic/* models")
		assert.NotContains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should not require COPILOT_GITHUB_TOKEN for anthropic/* models")
	})

	t.Run("openai model uses codex/openai secrets", func(t *testing.T) {
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{Model: "openai/gpt-4.1"},
			ParsedTools:  &ToolsConfig{},
			Tools:        map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "CODEX_API_KEY", "Should require CODEX_API_KEY for openai/* models")
		assert.Contains(t, secrets, "OPENAI_API_KEY", "Should require OPENAI_API_KEY for openai/* models")
	})
}
