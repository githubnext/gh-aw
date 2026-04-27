//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrushEngine(t *testing.T) {
	engine := NewCrushEngine()

	t.Run("engine identity", func(t *testing.T) {
		assert.Equal(t, "crush", engine.GetID(), "Engine ID should be 'crush'")
		assert.Equal(t, "Crush", engine.GetDisplayName(), "Display name should be 'Crush'")
		assert.NotEmpty(t, engine.GetDescription(), "Description should not be empty")
		assert.True(t, engine.IsExperimental(), "Crush engine should be experimental")
	})

	t.Run("capabilities", func(t *testing.T) {
		assert.False(t, engine.SupportsToolsAllowlist(), "Should not support tools allowlist")
		assert.False(t, engine.SupportsMaxTurns(), "Should not support max turns")
		assert.False(t, engine.SupportsWebSearch(), "Should not support built-in web search")
	})

	t.Run("model env var name", func(t *testing.T) {
		assert.Equal(t, "CRUSH_MODEL", engine.GetModelEnvVarName(), "Should return CRUSH_MODEL")
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
				Model: "anthropic/claude-sonnet-4-20250514",
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
		assert.Contains(t, files, ".crush.json", "Should include .crush.json config file")
		assert.Contains(t, files, "AGENTS.md", "Should include cross-engine AGENTS.md")
	})

	t.Run("agent manifest path prefixes", func(t *testing.T) {
		prefixes := engine.GetAgentManifestPathPrefixes()
		assert.Contains(t, prefixes, ".crush/", "Should include .crush/ config directory")
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

func TestCrushEngineInstallation(t *testing.T) {
	engine := NewCrushEngine()

	t.Run("standard installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetInstallationSteps(workflowData)
		require.NotEmpty(t, steps, "Should generate installation steps")

		// Should have at least: Node.js setup + Install Crush
		assert.GreaterOrEqual(t, len(steps), 2, "Should have at least 2 installation steps")
	})

	t.Run("custom command skips installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/custom/crush",
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

func TestCrushEngineExecution(t *testing.T) {
	engine := NewCrushEngine()

	t.Run("basic execution", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		// steps[0] = Write Crush config, steps[1] = Execute Crush CLI
		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "name: Execute Crush CLI", "Should have correct step name")
		assert.Contains(t, stepContent, "id: agentic_execution", "Should have agentic_execution ID")
		assert.Contains(t, stepContent, "crush run", "Should invoke crush run command")
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

		// Model is passed via the native CRUSH_MODEL env var
		assert.Contains(t, stepContent, "CRUSH_MODEL: anthropic/claude-sonnet-4-20250514", "Should set CRUSH_MODEL env var")
	})

	t.Run("without model no model env var", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.NotContains(t, stepContent, "CRUSH_MODEL", "Should not include CRUSH_MODEL when model is unconfigured")
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

		assert.Contains(t, stepContent, "GH_AW_MCP_CONFIG: ${{ github.workspace }}/.crush.json", "Should set MCP config env var")
	})

	t.Run("with custom command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/custom/crush",
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate config step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "/custom/crush", "Should use custom command")
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

		assert.Contains(t, configContent, "Write Crush Config", "First step should be Write Crush Config")
		assert.Contains(t, configContent, ".crush.json", "Config step should reference .crush.json")
		assert.Contains(t, configContent, `"permission"`, "Config step should use 'permission' (singular, not 'permissions')")
		assert.Contains(t, configContent, `"external_directory":"allow"`, "Config step should allow external_directory for non-interactive CI")
		assert.NotContains(t, configContent, `"permissions"`, "Config step must NOT use 'permissions' (plural) — silently ignored by OpenCode)")
		assert.Contains(t, execContent, "Execute Crush CLI", "Second step should be Execute Crush CLI")
	})
}

func TestCrushEngineFirewallIntegration(t *testing.T) {
	engine := NewCrushEngine()

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
		assert.Contains(t, stepContent, "GITHUB_COPILOT_BASE_URL: http://host.docker.internal:10002", "Should route copilot/* fallback through Copilot LLM gateway URL")
	})

	t.Run("firewall enabled adds mounted MCP CLI path setup", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			Features: map[string]any{
				"mcp-cli": true,
			},
			ParsedTools: &ToolsConfig{
				MountAsCLIs: true,
			},
			Tools: map[string]any{
				"bash": []any{"echo"},
				"my-mcp-cli": map[string]any{
					"command": "node",
					"args":    []any{"index.js"},
				},
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

		stepContent := strings.Join(steps[1], "\n")
		assert.Contains(t, stepContent, "export PATH=\"${RUNNER_TEMP}/gh-aw/mcp-cli/bin:$PATH\"", "Should add mounted MCP CLI bin directory to PATH in AWF mode")
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
		provider, err := extractProviderFromModel("anthropic/claude-sonnet-4-20250514")
		require.NoError(t, err)
		assert.Equal(t, "anthropic", provider)

		provider, err = extractProviderFromModel("openai/gpt-4.1")
		require.NoError(t, err)
		assert.Equal(t, "openai", provider)

		provider, err = extractProviderFromModel("google/gemini-2.5-pro")
		require.NoError(t, err)
		assert.Equal(t, "google", provider)
	})

	t.Run("empty model returns empty provider", func(t *testing.T) {
		provider, err := extractProviderFromModel("")
		require.NoError(t, err)
		assert.Empty(t, provider)
	})

	t.Run("no slash returns empty provider", func(t *testing.T) {
		provider, err := extractProviderFromModel("claude-sonnet-4-20250514")
		require.NoError(t, err)
		assert.Empty(t, provider)
	})

	t.Run("case insensitive provider", func(t *testing.T) {
		provider, err := extractProviderFromModel("OpenAI/gpt-4.1")
		require.NoError(t, err)
		assert.Equal(t, "openai", provider)
	})

	t.Run("leading slash returns error", func(t *testing.T) {
		_, err := extractProviderFromModel("/gpt-4.1")
		require.Error(t, err, "Leading slash (empty provider prefix) must return an error")
		assert.Contains(t, err.Error(), "provider prefix is empty")
	})
}
