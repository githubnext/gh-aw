//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorEngine(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("cursor")
	require.NoError(t, err)

	t.Run("engine identity and capabilities", func(t *testing.T) {
		capabilities := engine.GetCapabilities()
		assert.Equal(t, "cursor", engine.GetID(), "Engine ID should be 'cursor'")
		assert.Equal(t, "Cursor", engine.GetDisplayName(), "Display name should be 'Cursor'")
		assert.True(t, engine.IsExperimental(), "Cursor engine should be experimental")
		assert.False(t, capabilities.ToolsAllowlist, "Should not support tools allowlist")
		assert.True(t, capabilities.MaxTurns, "Should support max turns")
		assert.False(t, capabilities.WebSearch, "Should not support built-in web search")
	})

	t.Run("model env var name", func(t *testing.T) {
		assert.Equal(t, constants.CursorCLIModelEnvVar, engine.GetModelEnvVarName(), "Should return CURSOR_MODEL")
	})
}

func TestCursorEngineInstallationAndExecution(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("cursor")
	require.NoError(t, err)

	t.Run("standard installation", func(t *testing.T) {
		steps := engine.GetInstallationSteps(&WorkflowData{Name: "test-workflow"})
		require.NotEmpty(t, steps, "Should generate installation steps")
		stepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, stepContent, "Setup Node.js", "Should include Node setup")
	})

	t.Run("execution uses cursor command and config", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-workflow"}, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		configContent := strings.Join(steps[0], "\n")
		execContent := strings.Join(steps[1], "\n")
		assert.Contains(t, configContent, "Write Cursor Config", "Should write Cursor config first")
		assert.Contains(t, configContent, "cursor-agent.jsonc", "Should reference cursor-agent.jsonc")
		assert.Contains(t, configContent, `"awf-proxy"`, "Config should define a custom awf-proxy provider to force @ai-sdk/openai-compatible (Chat Completions only, not Responses API)")
		assert.Contains(t, configContent, "172.30.0.30:10002", "Config should use the internal AWF api-proxy IP to bypass host.docker.internal auth")
		assert.Contains(t, configContent, "awf-copilot-proxy", "Config should use the AWF api-proxy placeholder key accepted by the internal 172.30.0.30 proxy")
		assert.Contains(t, configContent, `"autoupdate": false`, "Config should disable auto-updates to prevent interactive prompts in headless mode")
		assert.Contains(t, configContent, `"disabled_providers"`, "Config should disable unused providers")
		assert.Contains(t, execContent, "Execute Cursor CLI", "Should execute Cursor CLI")
		assert.Contains(t, execContent, "cursor-agent run", "Should invoke cursor-agent run")
		assert.Contains(t, execContent, "OPENAI_API_KEY: ${{ secrets.COPILOT_GITHUB_TOKEN }}", "Should default to Copilot token routing")
		assert.Contains(t, execContent, "XDG_DATA_HOME: /tmp/cursor-agent-data", "Should set XDG_DATA_HOME to prevent persistent DB migrations")
	})

	t.Run("firewall sets Cursor gateway base URL and OPENAI_BASE_URL", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:         "test-workflow",
			Model:        "copilot/gpt-5",
			EngineConfig: &EngineConfig{},
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
		assert.Contains(t, execContent, "OPENAI_BASE_URL: http://host.docker.internal:10002", "Should also set OPENAI_BASE_URL to Copilot gateway so Cursor's openai provider routes correctly")
	})

	t.Run("firewall passes model through awf-proxy prefix rewrite in CURSOR_MODEL", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:         "test-workflow",
			Model:        "copilot/gpt-5",
			EngineConfig: &EngineConfig{},
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
		assert.Contains(t, execContent, "CURSOR_MODEL: awf-proxy/gpt-5", "Should rewrite 'copilot/' to 'awf-proxy/' so Cursor uses the custom awf-proxy provider")
		assert.NotContains(t, execContent, "CURSOR_MODEL: copilot/gpt-5", "Should not pass 'copilot/' prefix through since the copilot provider uses Responses API")
	})
}

func TestCursorEngineProviderProfiles(t *testing.T) {
	engine, err := newBuiltinBehaviorDefinedEngine("cursor")
	require.NoError(t, err)

	t.Run("anthropic model uses anthropic secret", func(t *testing.T) {
		workflowData := &WorkflowData{
			Model:        "anthropic/claude-sonnet-4",
			EngineConfig: &EngineConfig{},
			ParsedTools:  &ToolsConfig{},
			Tools:        map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "ANTHROPIC_API_KEY", "Should require ANTHROPIC_API_KEY for anthropic/* models")
		assert.NotContains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should not require COPILOT_GITHUB_TOKEN for anthropic/* models")
	})

	t.Run("openai model uses codex/openai secrets", func(t *testing.T) {
		workflowData := &WorkflowData{
			Model:        "openai/gpt-4.1",
			EngineConfig: &EngineConfig{},
			ParsedTools:  &ToolsConfig{},
			Tools:        map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "CODEX_API_KEY", "Should require CODEX_API_KEY for openai/* models")
		assert.Contains(t, secrets, "OPENAI_API_KEY", "Should require OPENAI_API_KEY for openai/* models")
	})
}
