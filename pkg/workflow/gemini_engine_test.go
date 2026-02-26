//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiEngine(t *testing.T) {
	engine := NewGeminiEngine()

	t.Run("engine identity", func(t *testing.T) {
		assert.Equal(t, "gemini", engine.GetID(), "Engine ID should be 'gemini'")
		assert.Equal(t, "Google Gemini CLI", engine.GetDisplayName(), "Display name should be 'Google Gemini CLI'")
		assert.NotEmpty(t, engine.GetDescription(), "Description should not be empty")
		assert.False(t, engine.IsExperimental(), "Gemini engine should not be experimental")
	})

	t.Run("capabilities", func(t *testing.T) {
		assert.True(t, engine.SupportsToolsAllowlist(), "Should support tools allowlist")
		assert.False(t, engine.SupportsMaxTurns(), "Should not support max turns")
		assert.False(t, engine.SupportsWebFetch(), "Should not support built-in web fetch")
		assert.False(t, engine.SupportsWebSearch(), "Should not support built-in web search")
		assert.True(t, engine.SupportsFirewall(), "Should support firewall/AWF")
		assert.False(t, engine.SupportsPlugins(), "Should not support plugins")
		assert.Equal(t, 10003, engine.SupportsLLMGateway(), "Should support LLM gateway on port 10003")
	})

	t.Run("required secrets", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test",
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "GEMINI_API_KEY", "Should require GEMINI_API_KEY")
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
		assert.Contains(t, secrets, "GEMINI_API_KEY", "Should require GEMINI_API_KEY")
		assert.Contains(t, secrets, "MCP_GATEWAY_API_KEY", "Should require MCP_GATEWAY_API_KEY when MCP servers present")
		assert.Contains(t, secrets, "GITHUB_MCP_SERVER_TOKEN", "Should require GITHUB_MCP_SERVER_TOKEN for GitHub tool")
	})

	t.Run("declared output files", func(t *testing.T) {
		outputFiles := engine.GetDeclaredOutputFiles()
		require.Len(t, outputFiles, 1, "Should declare one output file path")
		assert.Equal(t, "/tmp/gemini-client-error-*.json", outputFiles[0], "Should declare Gemini error log wildcard path")
	})
}

func TestGeminiEngineInstallation(t *testing.T) {
	engine := NewGeminiEngine()

	t.Run("standard installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetInstallationSteps(workflowData)
		require.NotEmpty(t, steps, "Should generate installation steps")

		// Should have at least: Node.js setup + Install Gemini
		// (secret validation is now in the activation job via GetSecretValidationStep)
		assert.GreaterOrEqual(t, len(steps), 2, "Should have at least 2 installation steps")

		// Verify first step is Node.js setup
		if len(steps) > 0 && len(steps[0]) > 0 {
			stepContent := strings.Join(steps[0], "\n")
			assert.Contains(t, stepContent, "Setup Node.js", "First step should setup Node.js")
		}

		// Verify second step is Install Gemini CLI
		if len(steps) > 1 && len(steps[1]) > 0 {
			stepContent := strings.Join(steps[1], "\n")
			assert.Contains(t, stepContent, "Install Gemini CLI", "Second step should install Gemini CLI")
			assert.Contains(t, stepContent, "@google/gemini-cli", "Should install @google/gemini-cli package")
		}
	})

	t.Run("custom command skips installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/custom/gemini",
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

func TestGeminiEngineExecution(t *testing.T) {
	engine := NewGeminiEngine()

	t.Run("basic execution", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		// steps[0] = Write Gemini settings, steps[1] = Execute Gemini CLI
		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "name: Execute Gemini CLI", "Should have correct step name")
		assert.Contains(t, stepContent, "id: agentic_execution", "Should have agentic_execution ID")
		assert.Contains(t, stepContent, "gemini", "Should invoke gemini command")
		assert.Contains(t, stepContent, "--yolo", "Should include --yolo flag for auto-approving tool executions")
		assert.Contains(t, stepContent, "--output-format stream-json", "Should use streaming JSON output format")
		assert.Contains(t, stepContent, `--prompt "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`, "Should include prompt argument with correct shell quoting")
		assert.Contains(t, stepContent, "/tmp/test.log", "Should include log file")
		assert.Contains(t, stepContent, "GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}", "Should set GEMINI_API_KEY env var")
	})

	t.Run("with model", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Model: "gemini-1.5-pro",
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// Model is passed via the native GEMINI_MODEL env var (not as a --model flag)
		assert.Contains(t, stepContent, "GEMINI_MODEL: gemini-1.5-pro", "Should set GEMINI_MODEL env var")
		assert.NotContains(t, stepContent, "--model gemini-1.5-pro", "Should not embed model in command")
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
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// Gemini CLI reads MCP config from .gemini/settings.json, not --mcp-config flag
		assert.NotContains(t, stepContent, "--mcp-config", "Should NOT include --mcp-config flag (Gemini CLI does not support it)")
		assert.Contains(t, stepContent, "GH_AW_MCP_CONFIG: ${{ github.workspace }}/.gemini/settings.json", "Should set MCP config env var to Gemini settings.json path")
	})

	t.Run("with custom command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/custom/gemini",
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "/custom/gemini", "Should use custom command")
	})

	t.Run("environment variables", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "GEMINI_API_KEY:", "Should include GEMINI_API_KEY")
		assert.Contains(t, stepContent, "GH_AW_PROMPT:", "Should include GH_AW_PROMPT")
		assert.Contains(t, stepContent, "GITHUB_WORKSPACE:", "Should include GITHUB_WORKSPACE")
		assert.Contains(t, stepContent, "DEBUG: gemini-cli:*", "Should include DEBUG env var for verbose diagnostics")
	})

	t.Run("model environment variables", func(t *testing.T) {
		// When model is not configured, no model env var should be set (let Gemini CLI use its default)
		noModelWorkflow := &WorkflowData{
			Name:        "no-model",
			SafeOutputs: &SafeOutputsConfig{},
		}

		steps := engine.GetExecutionSteps(noModelWorkflow, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")
		stepContent := strings.Join(steps[1], "\n")
		assert.NotContains(t, stepContent, "GH_AW_MODEL_DETECTION_GEMINI", "Should not include detection model env var when model is unconfigured")
		assert.NotContains(t, stepContent, "GH_AW_MODEL_AGENT_GEMINI", "Should not include agent model env var when model is unconfigured")
		assert.NotContains(t, stepContent, "GEMINI_MODEL", "Should not include GEMINI_MODEL when model is unconfigured")

		// When model is configured, use the native GEMINI_MODEL env var
		modelWorkflow := &WorkflowData{
			Name: "model-configured",
			EngineConfig: &EngineConfig{
				Model: "gemini-2.0-flash",
			},
		}

		steps = engine.GetExecutionSteps(modelWorkflow, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")
		stepContent = strings.Join(steps[1], "\n")
		assert.Contains(t, stepContent, "GEMINI_MODEL: gemini-2.0-flash", "Should set GEMINI_MODEL when model is explicitly configured")
	})

	t.Run("engine env overrides default token expression", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"GEMINI_API_KEY": "${{ secrets.MY_ORG_GEMINI_KEY }}",
				},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// The user-provided value should override the default token expression
		assert.Contains(t, stepContent, "GEMINI_API_KEY: ${{ secrets.MY_ORG_GEMINI_KEY }}", "engine.env should override the default GEMINI_API_KEY expression")
		assert.NotContains(t, stepContent, "GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}", "Default GEMINI_API_KEY expression should be replaced by engine.env")
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
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "CUSTOM_VAR: custom-value", "engine.env non-secret vars should be included")
	})

	t.Run("settings step is first", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		settingsContent := strings.Join(steps[0], "\n")
		execContent := strings.Join(steps[1], "\n")

		assert.Contains(t, settingsContent, "Write Gemini settings", "First step should be Write Gemini settings")
		assert.Contains(t, settingsContent, "includeDirectories", "Settings step should set includeDirectories")
		assert.Contains(t, settingsContent, "/tmp/", "Settings step should include /tmp/ in include directories")
		assert.Contains(t, execContent, "Execute Gemini CLI", "Second step should be Execute Gemini CLI")
	})
}

func TestGeminiEngineFirewallIntegration(t *testing.T) {
	engine := NewGeminiEngine()

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
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// Should use AWF command
		assert.Contains(t, stepContent, "awf", "Should use AWF when firewall is enabled")
		assert.Contains(t, stepContent, "--allow-domains", "Should include allow-domains flag")
		assert.Contains(t, stepContent, "--enable-api-proxy", "Should include --enable-api-proxy flag")
		assert.Contains(t, stepContent, "GEMINI_API_BASE_URL: http://host.docker.internal:10003", "Should set GEMINI_API_BASE_URL to LLM gateway URL")
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
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// Should use simple command without AWF
		assert.Contains(t, stepContent, "set -o pipefail", "Should use simple command with pipefail")
		assert.NotContains(t, stepContent, "awf", "Should not use AWF when firewall is disabled")
		assert.NotContains(t, stepContent, "GEMINI_API_BASE_URL", "Should not set GEMINI_API_BASE_URL when firewall is disabled")
	})
}

func TestComputeGeminiToolsCore(t *testing.T) {
	t.Run("nil tools includes default read-only tools", func(t *testing.T) {
		result := computeGeminiToolsCore(nil)
		assert.Contains(t, result, "glob", "Should include glob")
		assert.Contains(t, result, "grep_search", "Should include grep_search")
		assert.Contains(t, result, "list_directory", "Should include list_directory")
		assert.Contains(t, result, "read_file", "Should include read_file")
		assert.Contains(t, result, "read_many_files", "Should include read_many_files")
		assert.NotContains(t, result, "run_shell_command", "Should not include run_shell_command without bash tool")
		assert.NotContains(t, result, "write_file", "Should not include write_file without edit tool")
		assert.NotContains(t, result, "replace", "Should not include replace without edit tool")
	})

	t.Run("empty tools includes default read-only tools", func(t *testing.T) {
		result := computeGeminiToolsCore(map[string]any{})
		assert.Contains(t, result, "read_file", "Should include read_file")
		assert.NotContains(t, result, "run_shell_command", "Should not include run_shell_command")
	})

	t.Run("bash with specific commands maps to run_shell_command entries", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{"grep", "ls", "git"},
		}
		result := computeGeminiToolsCore(tools)
		assert.Contains(t, result, "run_shell_command(grep)", "Should map grep to run_shell_command(grep)")
		assert.Contains(t, result, "run_shell_command(ls)", "Should map ls to run_shell_command(ls)")
		assert.Contains(t, result, "run_shell_command(git)", "Should map git to run_shell_command(git)")
		assert.NotContains(t, result, "run_shell_command", "Should not include unrestricted run_shell_command")
	})

	t.Run("bash with wildcard * maps to unrestricted run_shell_command", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{"*"},
		}
		result := computeGeminiToolsCore(tools)
		assert.Contains(t, result, "run_shell_command", "Should include unrestricted run_shell_command for wildcard")
		assert.NotContains(t, result, "run_shell_command(*)", "Should not include run_shell_command(*)")
	})

	t.Run("bash with :* wildcard maps to unrestricted run_shell_command", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{":*"},
		}
		result := computeGeminiToolsCore(tools)
		assert.Contains(t, result, "run_shell_command", "Should include unrestricted run_shell_command for :* wildcard")
	})

	t.Run("bash with no specific commands (nil) maps to unrestricted run_shell_command", func(t *testing.T) {
		tools := map[string]any{
			"bash": nil,
		}
		result := computeGeminiToolsCore(tools)
		assert.Contains(t, result, "run_shell_command", "Should include unrestricted run_shell_command when bash has no commands")
	})

	t.Run("edit tool maps to write_file and replace", func(t *testing.T) {
		tools := map[string]any{
			"edit": map[string]any{},
		}
		result := computeGeminiToolsCore(tools)
		assert.Contains(t, result, "write_file", "Should map edit to write_file")
		assert.Contains(t, result, "replace", "Should map edit to replace")
	})

	t.Run("combined bash and edit tools", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{"grep"},
			"edit": map[string]any{},
		}
		result := computeGeminiToolsCore(tools)
		assert.Contains(t, result, "run_shell_command(grep)", "Should include run_shell_command(grep)")
		assert.Contains(t, result, "write_file", "Should include write_file")
		assert.Contains(t, result, "replace", "Should include replace")
		assert.Contains(t, result, "read_file", "Should always include read_file")
	})

	t.Run("result is sorted", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{"zzz", "aaa"},
			"edit": map[string]any{},
		}
		result := computeGeminiToolsCore(tools)
		for i := 1; i < len(result); i++ {
			assert.LessOrEqual(t, result[i-1], result[i], "Tools should be sorted alphabetically")
		}
	})
}

func TestGenerateGeminiSettingsStep(t *testing.T) {
	engine := NewGeminiEngine()

	t.Run("step sets context.includeDirectories to /tmp/", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:  "test-workflow",
			Tools: map[string]any{},
		}
		step := engine.generateGeminiSettingsStep(workflowData)
		content := strings.Join(step, "\n")

		assert.Contains(t, content, "Write Gemini settings", "Should have correct step name")
		assert.Contains(t, content, "/tmp/", "Should include /tmp/ in include directories")
		assert.Contains(t, content, "includeDirectories", "Should set includeDirectories")
		assert.Contains(t, content, ".gemini", "Should reference .gemini directory")
		assert.Contains(t, content, "GITHUB_WORKSPACE", "Should use GITHUB_WORKSPACE")
	})

	t.Run("step includes merge logic for existing settings.json", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:  "test-workflow",
			Tools: map[string]any{},
		}
		step := engine.generateGeminiSettingsStep(workflowData)
		content := strings.Join(step, "\n")

		assert.Contains(t, content, "if [ -f", "Should check for existing settings.json")
		assert.Contains(t, content, "jq", "Should use jq for merging")
		assert.Contains(t, content, "$existing * $base", "Should merge with base taking precedence")
	})

	t.Run("step includes tools.core with bash mapping", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			Tools: map[string]any{
				"bash": []any{"grep", "git"},
			},
		}
		step := engine.generateGeminiSettingsStep(workflowData)
		content := strings.Join(step, "\n")

		assert.Contains(t, content, "run_shell_command(grep)", "Should include run_shell_command(grep) for bash grep")
		assert.Contains(t, content, "run_shell_command(git)", "Should include run_shell_command(git) for bash git")
		assert.Contains(t, content, "core", "Should include tools.core")
	})

	t.Run("step includes tools.core with edit mapping", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			Tools: map[string]any{
				"edit": map[string]any{},
			},
		}
		step := engine.generateGeminiSettingsStep(workflowData)
		content := strings.Join(step, "\n")

		assert.Contains(t, content, "write_file", "Should include write_file for edit tool")
		assert.Contains(t, content, "replace", "Should include replace for edit tool")
	})
}
