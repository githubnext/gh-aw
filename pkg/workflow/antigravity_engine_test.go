//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAntigravityEngine(t *testing.T) {
	engine := NewAntigravityEngine()

	t.Run("engine identity", func(t *testing.T) {
		assert.Equal(t, "antigravity", engine.GetID(), "Engine ID should be 'antigravity'")
		assert.Equal(t, "Antigravity CLI", engine.GetDisplayName(), "Display name should be 'Antigravity CLI'")
		assert.NotEmpty(t, engine.GetDescription(), "Description should not be empty")
		assert.True(t, engine.IsExperimental(), "Antigravity engine should be experimental")
	})

	t.Run("capabilities", func(t *testing.T) {
		capabilities := engine.GetCapabilities()
		assert.True(t, capabilities.ToolsAllowlist, "Should support tools allowlist")
		assert.True(t, capabilities.MaxTurns, "Should support max turns")
		assert.False(t, capabilities.WebSearch, "Should not support built-in web search")
	})

	t.Run("required secrets", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:        "test",
			ParsedTools: &ToolsConfig{},
			Tools:       map[string]any{},
		}
		secrets := engine.GetRequiredSecretNames(workflowData)
		assert.Contains(t, secrets, "ANTIGRAVITY_API_KEY", "Should require ANTIGRAVITY_API_KEY")
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
		assert.Contains(t, secrets, "ANTIGRAVITY_API_KEY", "Should require ANTIGRAVITY_API_KEY")
		assert.Contains(t, secrets, "MCP_GATEWAY_API_KEY", "Should require MCP_GATEWAY_API_KEY when MCP servers present")
		assert.Contains(t, secrets, "GITHUB_MCP_SERVER_TOKEN", "Should require GITHUB_MCP_SERVER_TOKEN for GitHub tool")
	})

	t.Run("declared output files", func(t *testing.T) {
		outputFiles := engine.GetDeclaredOutputFiles()
		require.Len(t, outputFiles, 1, "Should declare one output file path")
		assert.Equal(t, "/tmp/gh-aw/antigravity-client-error-*.json", outputFiles[0], "Should declare Antigravity error log wildcard path under /tmp/gh-aw/")
	})

	t.Run("pre-bundle steps move files to /tmp/gh-aw/", func(t *testing.T) {
		workflowData := &WorkflowData{Name: "test-workflow"}
		steps := engine.GetPreBundleSteps(workflowData)
		require.Len(t, steps, 1, "Should return exactly one pre-bundle step")

		stepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, stepContent, "Move Antigravity error files", "Step name should describe move operation")
		assert.Contains(t, stepContent, "mv /tmp/antigravity-client-error-*.json /tmp/gh-aw/", "Step should move files to /tmp/gh-aw/")
		assert.Contains(t, stepContent, "if: always()", "Step should run always so files are captured on failure")
	})
}

func TestAntigravityEngineInstallation(t *testing.T) {
	engine := NewAntigravityEngine()

	t.Run("standard installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetInstallationSteps(workflowData)
		require.NotEmpty(t, steps, "Should generate installation steps")

		// Should have exactly 1 step: Install Antigravity CLI
		// (no Node.js setup needed; agy is a GCS binary, not an npm package)
		assert.GreaterOrEqual(t, len(steps), 1, "Should have at least 1 installation step")

		// Verify first step is Install Antigravity CLI
		if len(steps) > 0 && len(steps[0]) > 0 {
			stepContent := strings.Join(steps[0], "\n")
			assert.Contains(t, stepContent, "Install Antigravity CLI", "First step should install Antigravity CLI")
			assert.Contains(t, stepContent, "install_antigravity_cli.sh", "Should use the Antigravity CLI install script")
			assert.Contains(t, stepContent, `"${ENGINE_VERSION}"`, "Should pass version via ENGINE_VERSION env var")
			assert.NotContains(t, stepContent, "NPM_CONFIG_MIN_RELEASE_AGE", "Antigravity installation should not set npm release-age cooldown")
		}
	})

	t.Run("custom command skips installation", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/custom/antigravity",
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

func TestAntigravityEngineExecution(t *testing.T) {
	engine := NewAntigravityEngine()

	t.Run("basic execution", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		// steps[0] = Write Antigravity Config, steps[1] = Execute Antigravity CLI
		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "name: Execute Antigravity CLI", "Should have correct step name")
		assert.Contains(t, stepContent, "id: agentic_execution", "Should have agentic_execution ID")
		assert.Contains(t, stepContent, "agy --dangerously-skip-permissions", "Should invoke agy with auto-approval enabled")
		assert.Contains(t, stepContent, "--dangerously-skip-permissions", "Should include auto-approval flag for tool executions")
		assert.NotContains(t, stepContent, "--yolo", "Should not include unsupported --yolo flag")
		assert.NotContains(t, stepContent, "--skip-trust", "Should not include unsupported --skip-trust flag")
		assert.NotContains(t, stepContent, "--output-format stream-json", "Should not include unsupported output format flag")
		assert.Contains(t, stepContent, `--prompt "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`, "Should include prompt argument with correct shell quoting")
		assert.Contains(t, stepContent, "/tmp/test.log", "Should include log file")
		assert.Contains(t, stepContent, "ANTIGRAVITY_API_KEY: ${{ secrets.ANTIGRAVITY_API_KEY }}", "Should set ANTIGRAVITY_API_KEY env var")
		assert.Contains(t, stepContent, "GEMINI_API_KEY: ${{ secrets.ANTIGRAVITY_API_KEY }}", "Should map GEMINI_API_KEY to the Antigravity secret for proxy auth")
	})

	t.Run("with model", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Model: "antigravity-1.5-pro",
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// Model is passed via the native ANTIGRAVITY_MODEL env var (not as a --model flag)
		assert.Contains(t, stepContent, "ANTIGRAVITY_MODEL: antigravity-1.5-pro", "Should set ANTIGRAVITY_MODEL env var")
		assert.NotContains(t, stepContent, "--model antigravity-1.5-pro", "Should not embed model in command")
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

		// Antigravity CLI reads MCP config from .antigravity/settings.json, not --mcp-config flag
		assert.NotContains(t, stepContent, "--mcp-config", "Should NOT include --mcp-config flag (Antigravity CLI does not support it)")
		assert.Contains(t, stepContent, "GH_AW_MCP_CONFIG: ${{ github.workspace }}/.antigravity/settings.json", "Should set MCP config env var to Antigravity settings.json path")
	})

	t.Run("with custom command", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Command: "/custom/antigravity",
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "/custom/antigravity", "Should use custom command")
	})

	t.Run("environment variables", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		assert.Contains(t, stepContent, "ANTIGRAVITY_API_KEY:", "Should include ANTIGRAVITY_API_KEY")
		assert.Contains(t, stepContent, "GEMINI_API_KEY:", "Should include GEMINI_API_KEY alias for proxy auth")
		assert.Contains(t, stepContent, "GH_AW_PROMPT:", "Should include GH_AW_PROMPT")
		assert.Contains(t, stepContent, "GITHUB_WORKSPACE:", "Should include GITHUB_WORKSPACE")
		assert.Contains(t, stepContent, "DEBUG: antigravity-cli:*", "Should include DEBUG env var for verbose diagnostics")
		assert.Contains(t, stepContent, "ANTIGRAVITY_CLI_TRUST_WORKSPACE: true", "Should include ANTIGRAVITY_CLI_TRUST_WORKSPACE")
	})

	t.Run("model environment variables", func(t *testing.T) {
		// When model is not configured, no model env var should be set (let Antigravity CLI use its default)
		noModelWorkflow := &WorkflowData{
			Name:        "no-model",
			SafeOutputs: &SafeOutputsConfig{},
		}

		steps := engine.GetExecutionSteps(noModelWorkflow, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")
		stepContent := strings.Join(steps[1], "\n")
		assert.NotContains(t, stepContent, "GH_AW_MODEL_DETECTION_ANTIGRAVITY", "Should not include detection model env var when model is unconfigured")
		assert.NotContains(t, stepContent, "GH_AW_MODEL_AGENT_ANTIGRAVITY", "Should not include agent model env var when model is unconfigured")
		assert.NotContains(t, stepContent, "ANTIGRAVITY_MODEL", "Should not include ANTIGRAVITY_MODEL when model is unconfigured")

		// When model is configured, use the native ANTIGRAVITY_MODEL env var
		modelWorkflow := &WorkflowData{
			Name: "model-configured",
			EngineConfig: &EngineConfig{
				Model: "antigravity-2.0-flash",
			},
		}

		steps = engine.GetExecutionSteps(modelWorkflow, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")
		stepContent = strings.Join(steps[1], "\n")
		assert.Contains(t, stepContent, "ANTIGRAVITY_MODEL: antigravity-2.0-flash", "Should set ANTIGRAVITY_MODEL when model is explicitly configured")
	})

	t.Run("engine env overrides default token expression", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"ANTIGRAVITY_API_KEY": "${{ secrets.MY_ORG_ANTIGRAVITY_KEY }}",
				},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "Should generate settings step and execution step")

		stepContent := strings.Join(steps[1], "\n")

		// The user-provided value should override the default token expression
		assert.Contains(t, stepContent, "ANTIGRAVITY_API_KEY: ${{ secrets.MY_ORG_ANTIGRAVITY_KEY }}", "engine.env should override the default ANTIGRAVITY_API_KEY expression")
		assert.NotContains(t, stepContent, "ANTIGRAVITY_API_KEY: ${{ secrets.ANTIGRAVITY_API_KEY }}", "Default ANTIGRAVITY_API_KEY expression should be replaced by engine.env")
		assert.Contains(t, stepContent, "GEMINI_API_KEY: ${{ secrets.MY_ORG_ANTIGRAVITY_KEY }}", "GEMINI_API_KEY alias should track the effective Antigravity key expression")
	})

	t.Run("validates ANTIGRAVITY_API_KEY secret", func(t *testing.T) {
		step := engine.GetSecretValidationStep(&WorkflowData{Name: "test-workflow"})
		require.NotEmpty(t, step, "Should generate secret validation step")

		stepContent := strings.Join(step, "\n")
		assert.Contains(t, stepContent, "Validate ANTIGRAVITY_API_KEY secret", "Should validate the Antigravity secret")
		assert.NotContains(t, stepContent, "GEMINI_API_KEY", "Should not reference the removed Gemini secret")
		assert.NotContains(t, stepContent, "engine: gemini is no longer supported", "Should not carry legacy Gemini compatibility logic in the validation step")
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

		assert.Contains(t, settingsContent, "Write Antigravity Config", "First step should be Write Antigravity Config")
		assert.Contains(t, settingsContent, "includeDirectories", "Settings step should set includeDirectories")
		assert.Contains(t, settingsContent, "/tmp/", "Settings step should include /tmp/ in include directories")
		assert.Contains(t, execContent, "Execute Antigravity CLI", "Second step should be Execute Antigravity CLI")
	})
}

func TestAntigravityEngineFirewallIntegration(t *testing.T) {
	engine := NewAntigravityEngine()

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
		// With config file support, domains and apiProxy are in the JSON config
		assert.Contains(t, stepContent, "allowDomains", "Should include allowDomains in config JSON")
		assert.Contains(t, stepContent, `"enabled":true`, "Should include apiProxy enabled in config JSON")
		assert.Contains(t, stepContent, "--exclude-env GEMINI_API_KEY", "Should exclude GEMINI_API_KEY from sandbox env")
		assert.Contains(t, stepContent, "ANTIGRAVITY_API_BASE_URL: http://host.docker.internal:10003", "Should set ANTIGRAVITY_API_BASE_URL to LLM gateway URL")
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
		assert.NotContains(t, stepContent, "ANTIGRAVITY_API_BASE_URL", "Should not set ANTIGRAVITY_API_BASE_URL when firewall is disabled")
	})
}

func TestComputeAntigravityToolsCore(t *testing.T) {
	t.Run("nil tools includes default read-only tools", func(t *testing.T) {
		result := computeAntigravityToolsCore(nil)
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
		result := computeAntigravityToolsCore(map[string]any{})
		assert.Contains(t, result, "read_file", "Should include read_file")
		assert.NotContains(t, result, "run_shell_command", "Should not include run_shell_command")
	})

	t.Run("bash with specific commands maps to run_shell_command entries", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{"grep", "ls", "git"},
		}
		result := computeAntigravityToolsCore(tools)
		assert.Contains(t, result, "run_shell_command(grep)", "Should map grep to run_shell_command(grep)")
		assert.Contains(t, result, "run_shell_command(ls)", "Should map ls to run_shell_command(ls)")
		assert.Contains(t, result, "run_shell_command(git)", "Should map git to run_shell_command(git)")
		assert.NotContains(t, result, "run_shell_command", "Should not include unrestricted run_shell_command")
	})

	t.Run("bash with wildcard * maps to unrestricted run_shell_command", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{"*"},
		}
		result := computeAntigravityToolsCore(tools)
		assert.Contains(t, result, "run_shell_command", "Should include unrestricted run_shell_command for wildcard")
		assert.NotContains(t, result, "run_shell_command(*)", "Should not include run_shell_command(*)")
	})

	t.Run("bash with :* wildcard maps to unrestricted run_shell_command", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{":*"},
		}
		result := computeAntigravityToolsCore(tools)
		assert.Contains(t, result, "run_shell_command", "Should include unrestricted run_shell_command for :* wildcard")
	})

	t.Run("bash with no specific commands (nil) maps to unrestricted run_shell_command", func(t *testing.T) {
		tools := map[string]any{
			"bash": nil,
		}
		result := computeAntigravityToolsCore(tools)
		assert.Contains(t, result, "run_shell_command", "Should include unrestricted run_shell_command when bash has no commands")
	})

	t.Run("edit tool maps to write_file and replace", func(t *testing.T) {
		tools := map[string]any{
			"edit": map[string]any{},
		}
		result := computeAntigravityToolsCore(tools)
		assert.Contains(t, result, "write_file", "Should map edit to write_file")
		assert.Contains(t, result, "replace", "Should map edit to replace")
	})

	t.Run("combined bash and edit tools", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{"grep"},
			"edit": map[string]any{},
		}
		result := computeAntigravityToolsCore(tools)
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
		result := computeAntigravityToolsCore(tools)
		for i := 1; i < len(result); i++ {
			assert.LessOrEqual(t, result[i-1], result[i], "Tools should be sorted alphabetically")
		}
	})

	t.Run("bash tool with trailing space-star is normalized to canonical run_shell_command(cmd)", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{"jq *"},
		}
		result := computeAntigravityToolsCore(tools)
		assert.Contains(t, result, "run_shell_command(jq)", "Should normalize 'jq *' to run_shell_command(jq)")
		assert.NotContains(t, result, "run_shell_command(jq *)", "Should not emit run_shell_command(jq *)")
	})

	t.Run("community-attribution-style wildcard entries normalize to canonical forms", func(t *testing.T) {
		tools := map[string]any{
			"bash": []any{"jq *", "sed *", "awk *", "cat *"},
		}
		result := computeAntigravityToolsCore(tools)
		assert.Contains(t, result, "run_shell_command(jq)", "Should normalize 'jq *'")
		assert.Contains(t, result, "run_shell_command(sed)", "Should normalize 'sed *'")
		assert.Contains(t, result, "run_shell_command(awk)", "Should normalize 'awk *'")
		assert.Contains(t, result, "run_shell_command(cat)", "Should normalize 'cat *'")
		assert.NotContains(t, result, "run_shell_command(jq *)", "Should not emit run_shell_command with wildcard suffix")
	})
}

func TestGenerateAntigravitySettingsStep(t *testing.T) {
	engine := NewAntigravityEngine()

	t.Run("step sets context.includeDirectories to /tmp/", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:  "test-workflow",
			Tools: map[string]any{},
		}
		step := engine.generateAntigravitySettingsStep(workflowData)
		content := strings.Join(step, "\n")

		assert.Contains(t, content, "Write Antigravity Config", "Should have correct step name")
		assert.Contains(t, content, "/tmp/", "Should include /tmp/ in include directories")
		assert.Contains(t, content, "includeDirectories", "Should set includeDirectories")
		assert.Contains(t, content, ".antigravity", "Should reference .antigravity directory")
		assert.Contains(t, content, "GITHUB_WORKSPACE", "Should use GITHUB_WORKSPACE")
	})

	t.Run("step includes merge logic for existing settings.json", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:  "test-workflow",
			Tools: map[string]any{},
		}
		step := engine.generateAntigravitySettingsStep(workflowData)
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
		step := engine.generateAntigravitySettingsStep(workflowData)
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
		step := engine.generateAntigravitySettingsStep(workflowData)
		content := strings.Join(step, "\n")

		assert.Contains(t, content, "write_file", "Should include write_file for edit tool")
		assert.Contains(t, content, "replace", "Should include replace for edit tool")
	})

	t.Run("GH_AW_ANTIGRAVITY_BASE_CONFIG env var is single-quoted for valid YAML", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:  "test-workflow",
			Tools: map[string]any{},
		}
		step := engine.generateAntigravitySettingsStep(workflowData)
		content := strings.Join(step, "\n")

		// The JSON value must be single-quoted so YAML doesn't treat it as an object
		assert.Contains(t, content, "GH_AW_ANTIGRAVITY_BASE_CONFIG: '", "JSON env var value must be single-quoted for valid YAML")
	})

	t.Run("step includes web_fetch in tools.core when web-fetch tool is specified", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			Tools: map[string]any{
				"web-fetch": nil,
			},
		}
		step := engine.generateAntigravitySettingsStep(workflowData)
		content := strings.Join(step, "\n")

		assert.Contains(t, content, "web_fetch", "Should include web_fetch in tools.core when web-fetch is specified")
	})

	t.Run("step does not include web_fetch in tools.core when web-fetch tool is not specified", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:  "test-workflow",
			Tools: map[string]any{},
		}
		step := engine.generateAntigravitySettingsStep(workflowData)
		content := strings.Join(step, "\n")

		assert.NotContains(t, content, "web_fetch", "Should not include web_fetch in tools.core when web-fetch is not specified")
	})

	t.Run("step includes mounted mcp cli commands in restricted bash allowlist", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			Tools: map[string]any{
				"bash":       []any{"echo"},
				"cli-proxy":  true,
				"playwright": true,
				"mymcp": map[string]any{
					"command": "npx",
					"args":    []any{"-y", "@acme/mcp-server"},
				},
			},
			SafeOutputs: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
			},
		}
		step := engine.generateAntigravitySettingsStep(workflowData)
		content := strings.Join(step, "\n")

		assert.Contains(t, content, "run_shell_command(echo)", "Should include original restricted bash command")
		assert.Contains(t, content, "run_shell_command(mymcp:*)", "Should include mounted custom MCP CLI command")
		assert.Contains(t, content, "run_shell_command(playwright:*)", "Should include mounted playwright CLI command")
		assert.Contains(t, content, "run_shell_command(safeoutputs:*)", "Should include mounted safeoutputs CLI command")
	})
}

func TestAntigravityEngineWithExpressionVersion(t *testing.T) {
	engine := NewAntigravityEngine()

	expressionVersion := "${{ inputs.engine-version }}"
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:      "antigravity",
			Version: expressionVersion,
		},
	}

	installSteps := engine.GetInstallationSteps(workflowData)

	// Find the install step (uses install_antigravity_cli.sh)
	var installStep string
	for _, step := range installSteps {
		stepContent := strings.Join([]string(step), "\n")
		if strings.Contains(stepContent, "install_antigravity_cli.sh") {
			installStep = stepContent
			break
		}
	}

	if installStep == "" {
		t.Fatal("Could not find antigravity install step")
	}

	// Should use ENGINE_VERSION env var for injection safety
	if !strings.Contains(installStep, "ENGINE_VERSION: "+expressionVersion) {
		t.Errorf("Expected ENGINE_VERSION env var in install step, got:\n%s", installStep)
	}

	// Should reference env var in command
	if !strings.Contains(installStep, `"${ENGINE_VERSION}"`) {
		t.Errorf(`Expected "${ENGINE_VERSION}" in install command, got:\n%s`, installStep)
	}

	// Should NOT embed expression directly in install command
	if strings.Contains(installStep, "install_antigravity_cli.sh "+expressionVersion) {
		t.Errorf("Expression should NOT be embedded directly in install command, got:\n%s", installStep)
	}
}
