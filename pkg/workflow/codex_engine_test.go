//go:build !integration

package workflow

import (
	"fmt"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestCodexEngine(t *testing.T) {
	engine := NewCodexEngine()

	// Test basic properties
	if engine.GetID() != "codex" {
		t.Errorf("Expected ID 'codex', got '%s'", engine.GetID())
	}

	if engine.GetDisplayName() != "Codex" {
		t.Errorf("Expected display name 'Codex', got '%s'", engine.GetDisplayName())
	}

	if engine.IsExperimental() {
		t.Error("Codex engine should not be experimental")
	}

	if !engine.SupportsToolsAllowlist() {
		t.Error("Codex engine should support MCP tools")
	}

	// Test installation steps
	steps := engine.GetInstallationSteps(&WorkflowData{})
	// Secret validation is now in the activation job; installation has Node.js setup + Install Codex = 2 steps
	expectedStepCount := 2
	if len(steps) != expectedStepCount {
		t.Errorf("Expected %d installation steps, got %d", expectedStepCount, len(steps))
	}

	// Verify first step is Node.js setup
	if len(steps) > 0 && len(steps[0]) > 0 {
		if !strings.Contains(steps[0][0], "Setup Node.js") {
			t.Errorf("Expected first step to contain 'Setup Node.js', got '%s'", steps[0][0])
		}
	}

	// Verify second step is Install Codex CLI
	if len(steps) > 1 && len(steps[1]) > 0 {
		if !strings.Contains(steps[1][0], "Install Codex CLI") {
			t.Errorf("Expected second step to contain 'Install Codex CLI', got '%s'", steps[1][0])
		}
	}

	// Test execution steps
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	execSteps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(execSteps) != 1 {
		t.Fatalf("Expected 1 step for Codex execution, got %d", len(execSteps))
	}

	// Check the execution step
	stepContent := strings.Join([]string(execSteps[0]), "\n")

	if !strings.Contains(stepContent, "name: Execute Codex CLI") {
		t.Errorf("Expected step name 'Execute Codex CLI' in step content:\n%s", stepContent)
	}

	if strings.Contains(stepContent, "uses:") {
		t.Errorf("Expected no action for Codex (uses command), got step with 'uses:' in:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "codex") {
		t.Errorf("Expected command to contain 'codex' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "exec") {
		t.Errorf("Expected command to contain 'exec' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "test-log") {
		t.Errorf("Expected command to contain log file name in step content:\n%s", stepContent)
	}

	// Check that pipefail is enabled to preserve exit codes
	if !strings.Contains(stepContent, "set -o pipefail") {
		t.Errorf("Expected command to contain 'set -o pipefail' to preserve exit codes in step content:\n%s", stepContent)
	}

	// Check environment variables
	if !strings.Contains(stepContent, "CODEX_API_KEY: ${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}") {
		t.Errorf("Expected CODEX_API_KEY environment variable in step content:\n%s", stepContent)
	}
}

func TestCodexEngineWithVersion(t *testing.T) {
	engine := NewCodexEngine()

	// Test installation steps without version (should use pinned default version)
	stepsNoVersion := engine.GetInstallationSteps(&WorkflowData{})
	foundNoVersionInstall := false
	expectedPackage := fmt.Sprintf("@openai/codex@%s", constants.DefaultCodexVersion)
	for _, step := range stepsNoVersion {
		for _, line := range step {
			if strings.Contains(line, "npm install") && strings.Contains(line, expectedPackage) {
				foundNoVersionInstall = true
				break
			}
		}
	}
	if !foundNoVersionInstall {
		t.Errorf("Expected npm install command with @%s when no version specified", constants.DefaultCodexVersion)
	}

	// Test installation steps with version
	engineConfig := &EngineConfig{
		ID:      "codex",
		Version: "3.0.1",
	}
	workflowData := &WorkflowData{
		EngineConfig: engineConfig,
	}
	stepsWithVersion := engine.GetInstallationSteps(workflowData)
	foundVersionInstall := false
	for _, step := range stepsWithVersion {
		for _, line := range step {
			if strings.Contains(line, "npm install") && strings.Contains(line, "@openai/codex@3.0.1") {
				foundVersionInstall = true
				break
			}
		}
	}
	if !foundVersionInstall {
		t.Error("Expected versioned npm install command with @openai/codex@3.0.1")
	}
}

func TestCodexEngineExecutionIncludesGitHubAWPrompt(t *testing.T) {
	engine := NewCodexEngine()

	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// Should have at least one step
	if len(steps) == 0 {
		t.Error("Expected at least one execution step")
		return
	}

	// Check that GH_AW_PROMPT environment variable is included
	foundPromptEnv := false
	foundMCPConfigEnv := false
	for _, step := range steps {
		stepContent := strings.Join([]string(step), "\n")
		if strings.Contains(stepContent, "GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt") {
			foundPromptEnv = true
		}
		if strings.Contains(stepContent, "GH_AW_MCP_CONFIG: ${{ runner.temp }}/gh-aw/mcp-config/config.toml") {
			foundMCPConfigEnv = true
		}
	}

	if !foundPromptEnv {
		t.Error("Expected GH_AW_PROMPT environment variable in codex execution steps")
	}

	if !foundMCPConfigEnv {
		t.Error("Expected GH_AW_MCP_CONFIG environment variable in codex execution steps")
	}
}

func TestCodexEngineExecutionUsesWritableCodexHome(t *testing.T) {
	engine := NewCodexEngine()

	steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-workflow"}, "/tmp/gh-aw/test.log")
	if len(steps) == 0 {
		t.Fatal("Expected at least one execution step")
	}

	stepContent := strings.Join([]string(steps[0]), "\n")
	if !strings.Contains(stepContent, "CODEX_HOME: /tmp/gh-aw/mcp-config") {
		t.Errorf("Expected CODEX_HOME to use writable /tmp path, got:\n%s", stepContent)
	}
}

func TestCodexEngineRenderMCPConfig(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name     string
		tools    map[string]any
		mcpTools []string
		expected []string
	}{
		{
			name: "github tool with user_agent",
			tools: map[string]any{
				"github": map[string]any{},
			},
			mcpTools: []string{"github"},
			expected: []string{
				`cat > "${RUNNER_TEMP}/gh-aw/mcp-config/config.toml" << GH_AW_MCP_CONFIG_NORM_EOF`,
				"[history]",
				"persistence = \"none\"",
				"",
				"[shell_environment_policy]",
				"inherit = \"core\"",
				"include_only = [\"CODEX_API_KEY\", \"GITHUB_PERSONAL_ACCESS_TOKEN\", \"HOME\", \"OPENAI_API_KEY\", \"PATH\"]",
				"",
				"[mcp_servers.github]",
				"user_agent = \"test-workflow\"",
				"startup_timeout_sec = 120",
				"tool_timeout_sec = 60",
				fmt.Sprintf("container = \"ghcr.io/github/github-mcp-server:%s\"", constants.DefaultGitHubMCPServerVersion),
				"env = { \"GITHUB_HOST\" = \"$GITHUB_SERVER_URL\", \"GITHUB_PERSONAL_ACCESS_TOKEN\" = \"$GH_AW_GITHUB_TOKEN\", \"GITHUB_READ_ONLY\" = \"1\", \"GITHUB_TOOLSETS\" = \"context,repos,issues,pull_requests\" }",
				"env_vars = [\"GITHUB_HOST\", \"GITHUB_PERSONAL_ACCESS_TOKEN\", \"GITHUB_READ_ONLY\", \"GITHUB_TOOLSETS\"]",
				"GH_AW_MCP_CONFIG_NORM_EOF",
				"",
				"# Generate JSON config for MCP gateway",
				"GH_AW_NODE=$(which node 2>/dev/null || command -v node 2>/dev/null || echo node)",
				"cat << GH_AW_MCP_CONFIG_NORM_EOF | \"$GH_AW_NODE\" \"${RUNNER_TEMP}/gh-aw/actions/start_mcp_gateway.cjs\"",
				"{",
				"\"mcpServers\": {",
				"\"github\": {",
				fmt.Sprintf("\"container\": \"ghcr.io/github/github-mcp-server:%s\",", constants.DefaultGitHubMCPServerVersion),
				"\"env\": {",
				"\"GITHUB_HOST\": \"$GITHUB_SERVER_URL\",",
				"\"GITHUB_PERSONAL_ACCESS_TOKEN\": \"$GITHUB_MCP_SERVER_TOKEN\",",
				"\"GITHUB_READ_ONLY\": \"1\",",
				"\"GITHUB_TOOLSETS\": \"context,repos,issues,pull_requests\"",
				"},",
				"\"guard-policies\": {",
				"\"allow-only\": {",
				"\"min-integrity\": \"$GITHUB_MCP_GUARD_MIN_INTEGRITY\",",
				"\"repos\": \"$GITHUB_MCP_GUARD_REPOS\"",
				"}",
				"}",
				"}",
				"},",
				"\"gateway\": {",
				"\"port\": $MCP_GATEWAY_PORT,",
				"\"domain\": \"${MCP_GATEWAY_DOMAIN}\",",
				"\"apiKey\": \"${MCP_GATEWAY_API_KEY}\",",
				"\"payloadDir\": \"${MCP_GATEWAY_PAYLOAD_DIR}\"",
				"}",
				"}",
				"GH_AW_MCP_CONFIG_NORM_EOF",
				"",
				"# Sync converter output to writable CODEX_HOME for Codex",
				"mkdir -p /tmp/gh-aw/mcp-config",
				"cat > \"/tmp/gh-aw/mcp-config/config.toml\" << GH_AW_CODEX_SHELL_POLICY_NORM_EOF",
				"[shell_environment_policy]",
				"inherit = \"core\"",
				"include_only = [\"CODEX_API_KEY\", \"GITHUB_PERSONAL_ACCESS_TOKEN\", \"HOME\", \"OPENAI_API_KEY\", \"PATH\"]",
				"GH_AW_CODEX_SHELL_POLICY_NORM_EOF",
				"cat \"${RUNNER_TEMP}/gh-aw/mcp-config/config.toml\" >> \"/tmp/gh-aw/mcp-config/config.toml\"",
				"chmod 600 \"/tmp/gh-aw/mcp-config/config.toml\"",
				"mkdir -p \"${CODEX_HOME}\"",
				"if [ \"/tmp/gh-aw/mcp-config/config.toml\" != \"${CODEX_HOME}/config.toml\" ]; then cp \"/tmp/gh-aw/mcp-config/config.toml\" \"${CODEX_HOME}/config.toml\"; fi",
				"chmod 600 \"${CODEX_HOME}/config.toml\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			workflowData := &WorkflowData{Name: "test-workflow"}
			if err := engine.RenderMCPConfig(&yaml, tt.tools, tt.mcpTools, workflowData); err != nil {
				t.Fatalf("RenderMCPConfig returned unexpected error: %v", err)
			}

			result := yaml.String()
			// Normalize randomized heredoc delimiters before comparison
			result = normalizeHeredocDelimiters(result)
			lines := strings.Split(strings.TrimSpace(result), "\n")

			// Remove indentation from both expected and actual lines for comparison
			var normalizedResult []string
			for _, line := range lines {
				normalizedResult = append(normalizedResult, strings.TrimSpace(line))
			}

			var normalizedExpected []string
			for _, line := range tt.expected {
				normalizedExpected = append(normalizedExpected, strings.TrimSpace(line))
			}

			if len(normalizedResult) != len(normalizedExpected) {
				t.Errorf("Expected %d lines, got %d", len(normalizedExpected), len(normalizedResult))
				t.Errorf("Expected:\n%s", strings.Join(normalizedExpected, "\n"))
				t.Errorf("Got:\n%s", strings.Join(normalizedResult, "\n"))
				return
			}

			for i, expectedLine := range normalizedExpected {
				if i < len(normalizedResult) {
					actualLine := normalizedResult[i]
					if actualLine != expectedLine {
						t.Errorf("Line %d mismatch:\nExpected: %s\nActual:   %s", i+1, expectedLine, actualLine)
					}
				}
			}
		})
	}
}

func TestCodexEngineExecutionAddsMountedMCPCLIPathSetup(t *testing.T) {
	engine := NewCodexEngine()
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
	if len(steps) == 0 {
		t.Fatal("Expected execution step")
	}

	stepContent := strings.Join([]string(steps[0]), "\n")
	if !strings.Contains(stepContent, "export PATH=\"${RUNNER_TEMP}/gh-aw/mcp-cli/bin:$PATH\"") {
		t.Errorf("Expected mounted MCP CLI bin directory in AWF command, got:\n%s", stepContent)
	}
}

func TestCodexEngineUserAgentIdentifierConversion(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name         string
		workflowName string
		expectedUA   string
	}{
		{
			name:         "workflow name with spaces",
			workflowName: "Test Codex Create Issue",
			expectedUA:   "test-codex-create-issue",
		},
		{
			name:         "workflow name with underscores",
			workflowName: "Test_Workflow_Name",
			expectedUA:   "test-workflow-name",
		},
		{
			name:         "already identifier format",
			workflowName: "test-workflow",
			expectedUA:   "test-workflow",
		},
		{
			name:         "empty workflow name",
			workflowName: "",
			expectedUA:   "github-agentic-workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			workflowData := &WorkflowData{Name: tt.workflowName}

			tools := map[string]any{"github": map[string]any{}}
			mcpTools := []string{"github"}

			if err := engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData); err != nil {
				t.Fatalf("RenderMCPConfig returned unexpected error: %v", err)
			}

			result := yaml.String()
			expectedUserAgentLine := "user_agent = \"" + tt.expectedUA + "\""

			if !strings.Contains(result, expectedUserAgentLine) {
				t.Errorf("Expected MCP config to contain %q, got:\n%s", expectedUserAgentLine, result)
			}
		})
	}
}

func TestCodexEngineRenderMCPConfigUserAgentFromConfig(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name         string
		workflowName string
		configuredUA string
		expectedUA   string
		description  string
	}{
		{
			name:         "configured user_agent overrides workflow name",
			workflowName: "Test Workflow Name",
			configuredUA: "my-custom-agent",
			expectedUA:   "my-custom-agent",
			description:  "When user_agent is configured, it should be used instead of the converted workflow name",
		},
		{
			name:         "configured user_agent with spaces",
			workflowName: "test-workflow",
			configuredUA: "My Custom User Agent",
			expectedUA:   "My Custom User Agent",
			description:  "Configured user_agent should be used as-is, without identifier conversion",
		},
		{
			name:         "empty configured user_agent falls back to workflow name",
			workflowName: "Test Workflow",
			configuredUA: "",
			expectedUA:   "test-workflow",
			description:  "Empty configured user_agent should fall back to workflow name conversion",
		},
		{
			name:         "no workflow name and no configured user_agent uses default",
			workflowName: "",
			configuredUA: "",
			expectedUA:   "github-agentic-workflow",
			description:  "Should use default when neither workflow name nor user_agent is configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder

			engineConfig := &EngineConfig{
				ID: "codex",
			}
			if tt.configuredUA != "" {
				engineConfig.UserAgent = tt.configuredUA
			}

			workflowData := &WorkflowData{
				Name:         tt.workflowName,
				EngineConfig: engineConfig,
			}

			tools := map[string]any{"github": map[string]any{}}
			mcpTools := []string{"github"}

			if err := engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData); err != nil {
				t.Fatalf("RenderMCPConfig returned unexpected error: %v", err)
			}

			result := yaml.String()
			expectedUserAgentLine := "user_agent = \"" + tt.expectedUA + "\""

			if !strings.Contains(result, expectedUserAgentLine) {
				t.Errorf("Test case: %s\nExpected MCP config to contain %q, got:\n%s", tt.description, expectedUserAgentLine, result)
			}
		})
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name with spaces",
			input:    "Test Codex Create Issue",
			expected: "test-codex-create-issue",
		},
		{
			name:     "name with underscores",
			input:    "Test_Workflow_Name",
			expected: "test-workflow-name",
		},
		{
			name:     "name with mixed separators",
			input:    "Test Workflow_Name With Spaces",
			expected: "test-workflow-name-with-spaces",
		},
		{
			name:     "name with special characters",
			input:    "Test@Workflow#With$Special%Characters!",
			expected: "testworkflowwithspecialcharacters",
		},
		{
			name:     "name with multiple spaces",
			input:    "Test   Multiple    Spaces",
			expected: "test-multiple-spaces",
		},
		{
			name:     "empty name",
			input:    "",
			expected: "github-agentic-workflow",
		},
		{
			name:     "name with only special characters",
			input:    "@#$%!",
			expected: "github-agentic-workflow",
		},
		{
			name:     "already lowercase with hyphens",
			input:    "already-lowercase-name",
			expected: "already-lowercase-name",
		},
		{
			name:     "name with leading/trailing spaces",
			input:    "  Test Workflow  ",
			expected: "test-workflow",
		},
		{
			name:     "name with hyphens and underscores",
			input:    "Test-Workflow_Name",
			expected: "test-workflow-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeIdentifier(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCodexEngineRenderMCPConfigUserAgentWithHyphen(t *testing.T) {
	engine := NewCodexEngine()

	// Test that "user-agent" field name works
	tests := []struct {
		name             string
		engineConfigFunc func() *EngineConfig
		expectedUA       string
		description      string
	}{
		{
			name: "user-agent field gets parsed as user_agent (hyphen)",
			engineConfigFunc: func() *EngineConfig {
				// This simulates the parsing of "user-agent" from frontmatter
				// which gets stored in the UserAgent field
				return &EngineConfig{
					ID:        "codex",
					UserAgent: "custom-agent-hyphen",
				}
			},
			expectedUA:  "custom-agent-hyphen",
			description: "user-agent field with hyphen should be parsed and work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder

			workflowData := &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: tt.engineConfigFunc(),
			}

			tools := map[string]any{"github": map[string]any{}}
			mcpTools := []string{"github"}

			if err := engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData); err != nil {
				t.Fatalf("RenderMCPConfig returned unexpected error: %v", err)
			}

			result := yaml.String()
			expectedUserAgentLine := "user_agent = \"" + tt.expectedUA + "\""

			if !strings.Contains(result, expectedUserAgentLine) {
				t.Errorf("Test case: %s\nExpected MCP config to contain %q, got:\n%s", tt.description, expectedUserAgentLine, result)
			}
		})
	}
}

// TestCodexEngineMCPScriptsSecrets verifies that mcp-scripts secrets are passed to the execution step
func TestCodexEngineMCPScriptsSecrets(t *testing.T) {
	engine := NewCodexEngine()

	// Create workflow data with mcp-scripts that have env secrets
	workflowData := &WorkflowData{
		Name: "test-workflow-with-mcp-scripts",
		Features: map[string]any{
			"mcp-scripts": true, // Feature flag is optional now
		},
		MCPScripts: &MCPScriptsConfig{
			Tools: map[string]*MCPScriptToolConfig{
				"gh": {
					Name:        "gh",
					Description: "Execute gh CLI command",
					Run:         "gh $INPUT_ARGS",
					Env: map[string]string{
						"GH_TOKEN": "${{ github.token }}",
					},
				},
				"api-call": {
					Name:        "api-call",
					Description: "Call an API",
					Script:      "return fetch(url);",
					Env: map[string]string{
						"API_KEY": "${{ secrets.API_KEY }}",
					},
				},
			},
		},
	}

	// Get execution steps
	execSteps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
	if len(execSteps) == 0 {
		t.Fatal("Expected at least one execution step")
	}

	// Join all step lines to check content
	stepContent := strings.Join(execSteps[0], "\n")

	// Verify GH_TOKEN is in the env section
	if !strings.Contains(stepContent, "GH_TOKEN: ${{ github.token }}") {
		t.Errorf("Expected GH_TOKEN environment variable in step content:\n%s", stepContent)
	}

	// Verify API_KEY is in the env section
	if !strings.Contains(stepContent, "API_KEY: ${{ secrets.API_KEY }}") {
		t.Errorf("Expected API_KEY environment variable in step content:\n%s", stepContent)
	}
}

// TestCodexEngineHttpMCPServerRendered verifies that HTTP MCP servers
// are properly rendered in TOML format for Codex
func TestCodexEngineHttpMCPServerRendered(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name          string
		tools         map[string]any
		mcpTools      []string
		shouldContain []string
	}{
		{
			name: "HTTP MCP server should be rendered with url",
			tools: map[string]any{
				"gh-aw": map[string]any{
					"type": "http",
					"url":  "http://localhost:8765",
				},
			},
			mcpTools: []string{"gh-aw"},
			// localhost URLs are rewritten to host.docker.internal when firewall is enabled (default)
			shouldContain: []string{
				"[mcp_servers.gh-aw]",
				"url = \"http://host.docker.internal:8765\"",
			},
		},
		{
			name: "HTTP MCP server inferred from url field",
			tools: map[string]any{
				"my-http-server": map[string]any{
					"url": "https://api.example.com/mcp",
				},
			},
			mcpTools: []string{"my-http-server"},
			shouldContain: []string{
				"[mcp_servers.my-http-server]",
				"url = \"https://api.example.com/mcp\"",
			},
		},
		{
			name: "HTTP MCP server with headers",
			tools: map[string]any{
				"api-server": map[string]any{
					"type": "http",
					"url":  "https://api.example.com/mcp",
					"headers": map[string]any{
						"Authorization": "Bearer token123",
						"X-Custom":      "value",
					},
				},
			},
			mcpTools: []string{"api-server"},
			shouldContain: []string{
				"[mcp_servers.api-server]",
				"url = \"https://api.example.com/mcp\"",
				"http_headers = {",
				"\"Authorization\" = \"Bearer token123\"",
				"\"X-Custom\" = \"value\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			workflowData := &WorkflowData{Name: "test-workflow"}
			if err := engine.RenderMCPConfig(&yaml, tt.tools, tt.mcpTools, workflowData); err != nil {
				t.Fatalf("RenderMCPConfig returned unexpected error: %v", err)
			}

			result := yaml.String()

			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected MCP config to contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestCodexEngineSkipInstallationWithCommand(t *testing.T) {
	engine := NewCodexEngine()

	// Test with custom command - should skip installation
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{Command: "/usr/local/bin/custom-codex"},
	}
	steps := engine.GetInstallationSteps(workflowData)

	if len(steps) != 0 {
		t.Errorf("Expected 0 installation steps when command is specified, got %d", len(steps))
	}
}

func TestCodexEngineEnvOverridesTokenExpression(t *testing.T) {
	engine := NewCodexEngine()

	t.Run("engine env overrides default token expression", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"CODEX_API_KEY": "${{ secrets.MY_ORG_CODEX_KEY }}",
				},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")
		if len(steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(steps))
		}

		stepContent := strings.Join([]string(steps[0]), "\n")

		// engine.env override should replace the default token expression
		if !strings.Contains(stepContent, "CODEX_API_KEY: ${{ secrets.MY_ORG_CODEX_KEY }}") {
			t.Errorf("Expected engine.env to override CODEX_API_KEY, got:\n%s", stepContent)
		}
		if strings.Contains(stepContent, "CODEX_API_KEY: ${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}") {
			t.Errorf("Default CODEX_API_KEY expression should be replaced by engine.env override, got:\n%s", stepContent)
		}
	})

	t.Run("engine env adds extra environment variables", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"CUSTOM_VAR": "custom-value",
				},
			},
		}

		steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")
		if len(steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(steps))
		}

		stepContent := strings.Join([]string(steps[0]), "\n")

		if !strings.Contains(stepContent, "CUSTOM_VAR: custom-value") {
			t.Errorf("Expected engine.env to add CUSTOM_VAR, got:\n%s", stepContent)
		}
	})
}

func TestCodexEngineWebSearch(t *testing.T) {
	engine := NewCodexEngine()

	t.Run("web search disabled by default when tool not specified", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}
		steps := engine.GetExecutionSteps(workflowData, "test-log")
		if len(steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(steps))
		}
		stepContent := strings.Join([]string(steps[0]), "\n")
		if !strings.Contains(stepContent, `-c web_search="disabled"`) {
			t.Errorf(`Expected -c web_search="disabled" config when web-search tool is not specified, got:\n%s`, stepContent)
		}
		if strings.Contains(stepContent, "--no-search") {
			t.Errorf("Expected no --no-search flag (it does not exist), got:\n%s", stepContent)
		}
		if strings.Contains(stepContent, "--search") {
			t.Errorf("Expected no --search flag (it does not exist), got:\n%s", stepContent)
		}
	})

	t.Run("web search enabled when tool is specified", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			ParsedTools: &ToolsConfig{
				WebSearch: &WebSearchToolConfig{},
			},
		}
		steps := engine.GetExecutionSteps(workflowData, "test-log")
		if len(steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(steps))
		}
		stepContent := strings.Join([]string(steps[0]), "\n")
		if strings.Contains(stepContent, `-c web_search="disabled"`) {
			t.Errorf(`Expected no -c web_search="disabled" config when web-search tool is specified, got:\n%s`, stepContent)
		}
		if strings.Contains(stepContent, "--no-search") {
			t.Errorf("Expected no --no-search flag (it does not exist), got:\n%s", stepContent)
		}
		if strings.Contains(stepContent, "--search") {
			t.Errorf("Expected no --search flag (it does not exist), got:\n%s", stepContent)
		}
	})
}

func TestCodexEngineWebFetch(t *testing.T) {
	engine := NewCodexEngine()

	t.Run("fetch tool disabled by default when tool not specified", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
		}
		steps := engine.GetExecutionSteps(workflowData, "test-log")
		if len(steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(steps))
		}
		stepContent := strings.Join([]string(steps[0]), "\n")
		if !strings.Contains(stepContent, `-c fetch="disabled"`) {
			t.Errorf(`Expected -c fetch="disabled" config when web-fetch tool is not specified, got:\n%s`, stepContent)
		}
	})

	t.Run("fetch tool enabled when web-fetch tool is specified", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			ParsedTools: &ToolsConfig{
				WebFetch: &WebFetchToolConfig{},
			},
		}
		steps := engine.GetExecutionSteps(workflowData, "test-log")
		if len(steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(steps))
		}
		stepContent := strings.Join([]string(steps[0]), "\n")
		if strings.Contains(stepContent, `-c fetch="disabled"`) {
			t.Errorf(`Expected no -c fetch="disabled" config when web-fetch tool is specified, got:\n%s`, stepContent)
		}
	})
}

func TestCodexEngineWithExpressionVersion(t *testing.T) {
	engine := NewCodexEngine()

	expressionVersion := "${{ inputs.engine-version }}"
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:      "codex",
			Version: expressionVersion,
		},
	}

	installSteps := engine.GetInstallationSteps(workflowData)

	// Find the npm install step
	var installStep string
	for _, step := range installSteps {
		stepContent := strings.Join(step, "\n")
		if strings.Contains(stepContent, "npm install") {
			installStep = stepContent
			break
		}
	}

	if installStep == "" {
		t.Fatal("Could not find npm install step")
	}

	// Should use ENGINE_VERSION env var for injection safety
	if !strings.Contains(installStep, "ENGINE_VERSION: "+expressionVersion) {
		t.Errorf("Expected ENGINE_VERSION env var in install step, got:\n%s", installStep)
	}

	// Should reference env var in command
	if !strings.Contains(installStep, `"${ENGINE_VERSION}"`) {
		t.Errorf(`Expected "$ENGINE_VERSION" in npm install command, got:\n%s`, installStep)
	}

	// Should NOT embed expression directly in npm install command
	if strings.Contains(installStep, "@openai/codex@"+expressionVersion) {
		t.Errorf("Expression should NOT be embedded directly in npm install command, got:\n%s", installStep)
	}
}
