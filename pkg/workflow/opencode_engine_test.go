//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenCodeEngine(t *testing.T) {
	engine := NewOpenCodeEngine()

	t.Run("engine identity and capabilities", func(t *testing.T) {
		assert.Equal(t, "opencode", engine.GetID(), "Engine ID should be 'opencode'")
		assert.Equal(t, "OpenCode", engine.GetDisplayName(), "Display name should be 'OpenCode'")
		assert.True(t, engine.IsExperimental(), "OpenCode engine should be experimental")
		assert.False(t, engine.SupportsToolsAllowlist(), "Should not support tools allowlist")
		assert.False(t, engine.SupportsMaxTurns(), "Should not support max turns")
		assert.False(t, engine.SupportsWebSearch(), "Should not support built-in web search")
	})

	t.Run("model env var name", func(t *testing.T) {
		assert.Equal(t, constants.OpenCodeCLIModelEnvVar, engine.GetModelEnvVarName(), "Should return OPENCODE_MODEL")
	})
}

func TestOpenCodeEngineInstallationAndExecution(t *testing.T) {
	engine := NewOpenCodeEngine()

	t.Run("standard installation", func(t *testing.T) {
		steps := engine.GetInstallationSteps(&WorkflowData{Name: "test-workflow"})
		require.NotEmpty(t, steps, "Should generate installation steps")
		stepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, stepContent, "Setup Node.js", "Should include Node setup")
	})

	t.Run("execution uses opencode command and config", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-workflow"}, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		configContent := strings.Join(steps[0], "\n")
		execContent := strings.Join(steps[1], "\n")
		assert.Contains(t, configContent, "Write OpenCode Config", "Should write OpenCode config first")
		assert.Contains(t, configContent, "opencode.jsonc", "Should reference opencode.jsonc")
		assert.Contains(t, execContent, "Execute OpenCode CLI", "Should execute OpenCode CLI")
		assert.Contains(t, execContent, "opencode run", "Should invoke opencode run")
		assert.Contains(t, execContent, "OPENAI_API_KEY: ${{ secrets.COPILOT_GITHUB_TOKEN }}", "Should default to Copilot token routing")
	})

	t.Run("firewall sets OpenCode gateway base URL", func(t *testing.T) {
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
		require.Len(t, steps, 2, "Should generate config step and execution step")
		execContent := strings.Join(steps[1], "\n")
		assert.Contains(t, execContent, "GITHUB_COPILOT_BASE_URL: http://host.docker.internal:10002", "Should route through Copilot LLM gateway port for copilot/* models")
	})
}

func TestOpenCodeEngineProviderProfiles(t *testing.T) {
	engine := NewOpenCodeEngine()

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
