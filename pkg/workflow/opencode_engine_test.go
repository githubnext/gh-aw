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

	t.Run("engine identity", func(t *testing.T) {
		assert.Equal(t, "opencode", engine.GetID(), "Engine ID should be 'opencode'")
		assert.Equal(t, "OpenCode", engine.GetDisplayName(), "Display name should be 'OpenCode'")
		assert.NotEmpty(t, engine.GetDescription(), "Description should not be empty")
		assert.True(t, engine.IsExperimental(), "OpenCode engine should be experimental")
	})

	t.Run("capabilities", func(t *testing.T) {
		assert.False(t, engine.SupportsToolsAllowlist(), "Should not support tools allowlist")
		assert.False(t, engine.SupportsMaxTurns(), "Should not support max turns")
		assert.False(t, engine.SupportsWebSearch(), "Should not support built-in web search")
		assert.Equal(t, constants.OpenCodeLLMGatewayPort, engine.SupportsLLMGateway(), "Should support LLM gateway on port 10004")
	})

	t.Run("model env var name", func(t *testing.T) {
		assert.Equal(t, "OPENCODE_MODEL", engine.GetModelEnvVarName(), "Should return OPENCODE_MODEL")
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
		assert.NotContains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should not require COPILOT_GITHUB_TOKEN when copilot-requests is enabled")
	})

	t.Run("required secrets with MCP servers", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test",
			ParsedTools: &ToolsConfig{
				GitHub: &GitHubToolConfig{},
			},
			Tools: map[string]any{
				"github": map[string]any{},
			},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should require COPILOT_GITHUB_TOKEN for Copilot routing")
		assert.Contains(t, secrets, "MCP_GATEWAY_API_KEY", "Should require MCP_GATEWAY_API_KEY when MCP servers present")
		assert.Contains(t, secrets, "GITHUB_MCP_SERVER_TOKEN", "Should require GITHUB_MCP_SERVER_TOKEN for GitHub tool")
	})

	t.Run("required secrets with env override", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test",
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"ANTHROPIC_API_KEY": "${{ secrets.ANTHROPIC_API_KEY }}",
				},
			},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN", "Should still require COPILOT_GITHUB_TOKEN for Copilot routing")
		assert.Contains(t, secrets, "ANTHROPIC_API_KEY", "Should add ANTHROPIC_API_KEY from engine.env")
	})

	t.Run("declared output files", func(t *testing.T) {
		outputFiles := engine.GetDeclaredOutputFiles()
		assert.Empty(t, outputFiles, "Should have no declared output files")
	})

	t.Run("agent manifest files", func(t *testing.T) {
		files := engine.GetAgentManifestFiles()
		assert.Contains(t, files, "opencode.jsonc", "Should include opencode.jsonc config file")
		assert.Contains(t, files, "AGENTS.md", "Should include cross-engine AGENTS.md")
	})

	t.Run("agent manifest path prefixes", func(t *testing.T) {
		prefixes := engine.GetAgentManifestPathPrefixes()
		assert.Contains(t, prefixes, ".opencode/", "Should include .opencode/ config directory")
	})

	t.Run("secret validation step without copilot-requests", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test",
		}
		step := engine.GetSecretValidationStep(workflowData)
		stepContent := strings.Join(step, "\n")
		assert.Contains(t, stepContent, "COPILOT_GITHUB_TOKEN", "Should validate COPILOT_GITHUB_TOKEN")
	})

	t.Run("secret validation step with copilot-requests", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test",
			Features: map[string]any{
				"copilot-requests": true,
			},
		}
		step := engine.GetSecretValidationStep(workflowData)
		assert.Empty(t, step, "Should skip secret validation when copilot-requests is enabled")
	})
}

func TestOpenCodeEngineInstallation(t *testing.T) {
	engine := NewOpenCodeEngine()

	t.Run("standard installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetInstallationSteps(workflowData)
		require.NotEmpty(t, steps, "Should generate installation steps")

		// Should have at least: Node.js setup + Install OpenCode
		assert.GreaterOrEqual(t, len(steps), 2, "Should have at least 2 installation steps")
	})

	t.Run("custom command skips installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/custom/opencode",
			},
		}

		steps := engine.GetInstallationSteps(workflowData)
		assert.Empty(t, steps, "Should skip installation when custom command is specified")
	})

	t.Run("with firewall", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		steps := engine.GetInstallationSteps(workflowData)
		require.NotEmpty(t, steps, "Should generate installation steps")

		// Should include AWF installation step
		hasAWFInstall := false
		for _, step := range steps {
			stepContent := strings.Join(step, "\n")
			if strings.Contains(stepContent, "awf") || strings.Contains(stepContent, "firewall") {
				hasAWFInstall = true
				break
			}
		}
		assert.True(t, hasAWFInstall, "Should include AWF installation step when firewall is enabled")
	})
}

func TestOpenCodeEngineExecution(t *testing.T) {
	engine := NewOpenCodeEngine()

	t.Run("basic execution", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		// steps[0] = Write OpenCode config, steps[1] = Execute OpenCode CLI
		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "name: Execute OpenCode CLI", "Should have correct step name")
		assert.Contains(t, stepContent, "id: agentic_execution", "Should have agentic_execution ID")
		assert.Contains(t, stepContent, "opencode run", "Should invoke opencode run command")
		assert.Contains(t, stepContent, `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`, "Should include prompt argument")
		assert.Contains(t, stepContent, "/tmp/test.log", "Should include log file")
		assert.Contains(t, stepContent, "OPENAI_API_KEY: ${{ secrets.COPILOT_GITHUB_TOKEN }}", "Should set OPENAI_API_KEY from COPILOT_GITHUB_TOKEN")
		assert.Contains(t, stepContent, "NO_PROXY: localhost,127.0.0.1", "Should set NO_PROXY env var")
	})

	t.Run("basic execution with copilot-requests", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			Features: map[string]any{
				"copilot-requests": true,
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")
		assert.Contains(t, stepContent, "OPENAI_API_KEY: ${{ github.token }}", "Should set OPENAI_API_KEY from github.token when copilot-requests is enabled")
	})

	t.Run("with model", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Model: "anthropic/claude-sonnet-4-20250514",
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// Model is passed via the native OPENCODE_MODEL env var
		assert.Contains(t, stepContent, "OPENCODE_MODEL: anthropic/claude-sonnet-4-20250514", "Should set OPENCODE_MODEL env var")
	})

	t.Run("without model no model env var", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.NotContains(t, stepContent, "OPENCODE_MODEL", "Should not include OPENCODE_MODEL when model is unconfigured")
	})

	t.Run("with MCP servers", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			ParsedTools: &ToolsConfig{
				GitHub: &GitHubToolConfig{},
			},
			Tools: map[string]any{
				"github": map[string]any{},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "GH_AW_MCP_CONFIG: ${{ github.workspace }}/opencode.jsonc", "Should set MCP config env var")
	})

	t.Run("with custom command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/custom/opencode",
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "/custom/opencode", "Should use custom command")
	})

	t.Run("engine env overrides default token expression", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"OPENAI_API_KEY": "${{ secrets.MY_ORG_OPENAI_KEY }}",
				},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// The user-provided value should override the default token expression
		assert.Contains(t, stepContent, "OPENAI_API_KEY: ${{ secrets.MY_ORG_OPENAI_KEY }}", "engine.env should override the default OPENAI_API_KEY expression")
		assert.NotContains(t, stepContent, "OPENAI_API_KEY: ${{ secrets.COPILOT_GITHUB_TOKEN }}", "Default COPILOT_GITHUB_TOKEN expression should be replaced by engine.env")
	})

	t.Run("engine env adds custom non-secret env vars", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"CUSTOM_VAR": "custom-value",
				},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "CUSTOM_VAR: custom-value", "engine.env non-secret vars should be included")
	})

	t.Run("config step is first", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		configContent := strings.Join(steps[0], "\n")
		execContent := strings.Join(steps[1], "\n")

		assert.Contains(t, configContent, "Write OpenCode configuration", "First step should be Write OpenCode configuration")
		assert.Contains(t, configContent, "opencode.jsonc", "Config step should reference opencode.jsonc")
		assert.Contains(t, configContent, "permissions", "Config step should set permissions")
		assert.Contains(t, execContent, "Execute OpenCode CLI", "Second step should be Execute OpenCode CLI")
	})
}

func TestOpenCodeEngineFirewallIntegration(t *testing.T) {
	engine := NewOpenCodeEngine()

	t.Run("firewall enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// Should use AWF command
		assert.Contains(t, stepContent, "awf", "Should use AWF when firewall is enabled")
		assert.Contains(t, stepContent, "--allow-domains", "Should include allow-domains flag")
		assert.Contains(t, stepContent, "--enable-api-proxy", "Should include --enable-api-proxy flag")
		assert.Contains(t, stepContent, "OPENAI_BASE_URL: http://host.docker.internal:10004", "Should set OPENAI_BASE_URL to LLM gateway URL")
	})

	t.Run("firewall disabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: false,
				},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// Should use simple command without AWF
		assert.Contains(t, stepContent, "set -o pipefail", "Should use simple command with pipefail")
		assert.NotContains(t, stepContent, "awf", "Should not use AWF when firewall is disabled")
		assert.NotContains(t, stepContent, "OPENAI_BASE_URL", "Should not set OPENAI_BASE_URL when firewall is disabled")
	})
}

func TestExtractProviderFromModel(t *testing.T) {
	t.Run("standard provider/model format", func(t *testing.T) {
		assert.Equal(t, "anthropic", extractProviderFromModel("anthropic/claude-sonnet-4-20250514"))
		assert.Equal(t, "openai", extractProviderFromModel("openai/gpt-4.1"))
		assert.Equal(t, "google", extractProviderFromModel("google/gemini-2.5-pro"))
	})

	t.Run("empty model defaults to copilot", func(t *testing.T) {
		assert.Equal(t, "copilot", extractProviderFromModel(""))
	})

	t.Run("no slash defaults to copilot", func(t *testing.T) {
		assert.Equal(t, "copilot", extractProviderFromModel("claude-sonnet-4-20250514"))
	})

	t.Run("case insensitive provider", func(t *testing.T) {
		assert.Equal(t, "openai", extractProviderFromModel("OpenAI/gpt-4.1"))
	})
}
