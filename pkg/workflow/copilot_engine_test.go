package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopilotEngine(t *testing.T) {
	engine := NewCopilotEngine()

	// Test basic properties
	if engine.GetID() != "copilot" {
		t.Errorf("Expected copilot engine ID, got '%s'", engine.GetID())
	}

	if engine.GetDisplayName() != "GitHub Copilot CLI" {
		t.Errorf("Expected 'GitHub Copilot CLI' display name, got '%s'", engine.GetDisplayName())
	}

	if engine.IsExperimental() {
		t.Error("Expected copilot engine to not be experimental")
	}

	if !engine.SupportsToolsAllowlist() {
		t.Error("Expected copilot engine to support tools allowlist")
	}

	if !engine.SupportsHTTPTransport() {
		t.Error("Expected copilot engine to support HTTP transport")
	}

	if engine.SupportsMaxTurns() {
		t.Error("Expected copilot engine to not support max-turns yet")
	}
}

func TestCopilotEngineInstallationSteps(t *testing.T) {
	engine := NewCopilotEngine()

	// Test with no version (firewall feature disabled by default)
	workflowData := &WorkflowData{}
	steps := engine.GetInstallationSteps(workflowData)
	// When firewall is disabled: secret validation + Node.js setup + install = 3 steps
	if len(steps) != 3 {
		t.Errorf("Expected 3 installation steps (secret validation + Node.js setup + install), got %d", len(steps))
	}

	// Test with version (firewall feature disabled by default)
	workflowDataWithVersion := &WorkflowData{
		EngineConfig: &EngineConfig{Version: "1.0.0"},
	}
	stepsWithVersion := engine.GetInstallationSteps(workflowDataWithVersion)
	// When firewall is disabled: secret validation + Node.js setup + install = 3 steps
	if len(stepsWithVersion) != 3 {
		t.Errorf("Expected 3 installation steps with version (secret validation + Node.js setup + install), got %d", len(stepsWithVersion))
	}
}

func TestCopilotEngineExecutionSteps(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// GetExecutionSteps only returns the execution step, not Squid logs or cleanup
	if len(steps) != 1 {
		t.Fatalf("Expected 1 step (copilot execution), got %d", len(steps))
	}

	// Check the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	if !strings.Contains(stepContent, "name: Execute GitHub Copilot CLI") {
		t.Errorf("Expected step name 'Execute GitHub Copilot CLI' in step content:\n%s", stepContent)
	}

	// When firewall is disabled, should use 'copilot' command (not npx)
	if !strings.Contains(stepContent, "copilot") || !strings.Contains(stepContent, "--add-dir /tmp/ --add-dir /tmp/gh-aw/ --add-dir /tmp/gh-aw/agent/ --log-level all --log-dir") {
		t.Errorf("Expected command to contain 'copilot' and '--add-dir /tmp/ --add-dir /tmp/gh-aw/ --add-dir /tmp/gh-aw/agent/ --log-level all --log-dir' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "/tmp/gh-aw/test.log") {
		t.Errorf("Expected command to contain log file name in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "GITHUB_TOKEN: ${{ secrets.COPILOT_CLI_TOKEN  }}") {
		t.Errorf("Expected GITHUB_TOKEN environment variable in step content:\n%s", stepContent)
	}

	// Test that GH_AW_SAFE_OUTPUTS is not present when SafeOutputs is nil
	if strings.Contains(stepContent, "GH_AW_SAFE_OUTPUTS") {
		t.Error("Expected GH_AW_SAFE_OUTPUTS to not be present when SafeOutputs is nil")
	}

	// Test that --disable-builtin-mcps flag is present
	if !strings.Contains(stepContent, "--disable-builtin-mcps") {
		t.Errorf("Expected --disable-builtin-mcps flag in command, got:\n%s", stepContent)
	}

	// Test that mkdir commands are present for --add-dir directories
	if !strings.Contains(stepContent, "mkdir -p /tmp/") {
		t.Errorf("Expected 'mkdir -p /tmp/' command in step content:\n%s", stepContent)
	}
	if !strings.Contains(stepContent, "mkdir -p /tmp/gh-aw/") {
		t.Errorf("Expected 'mkdir -p /tmp/gh-aw/' command in step content:\n%s", stepContent)
	}
	if !strings.Contains(stepContent, "mkdir -p /tmp/gh-aw/agent/") {
		t.Errorf("Expected 'mkdir -p /tmp/gh-aw/agent/' command in step content:\n%s", stepContent)
	}
	if !strings.Contains(stepContent, "mkdir -p /tmp/gh-aw/.copilot/logs/") {
		t.Errorf("Expected 'mkdir -p /tmp/gh-aw/.copilot/logs/' command in step content:\n%s", stepContent)
	}
}

func TestCopilotEngineExecutionStepsWithOutput(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		SafeOutputs: &SafeOutputsConfig{}, // Non-nil to trigger output handling
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// GetExecutionSteps only returns the execution step
	if len(steps) != 1 {
		t.Fatalf("Expected 1 step (copilot execution), got %d", len(steps))
	}

	// Check the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Test that GH_AW_SAFE_OUTPUTS is present when SafeOutputs is not nil
	if !strings.Contains(stepContent, "GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}") {
		t.Errorf("Expected GH_AW_SAFE_OUTPUTS environment variable when SafeOutputs is not nil in step content:\n%s", stepContent)
	}
}

func TestCopilotEngineGetLogParserScript(t *testing.T) {
	engine := NewCopilotEngine()
	script := engine.GetLogParserScriptId()

	if script != "parse_copilot_log" {
		t.Errorf("Expected 'parse_copilot_log', got '%s'", script)
	}
}

func TestCopilotEngineGetLogFileForParsing(t *testing.T) {
	engine := NewCopilotEngine()
	logFile := engine.GetLogFileForParsing()

	expected := "/tmp/gh-aw/.copilot/logs/"
	if logFile != expected {
		t.Errorf("Expected '%s', got '%s'", expected, logFile)
	}
}

func TestCopilotEngineComputeToolArguments(t *testing.T) {
	engine := NewCopilotEngine()

	tests := []struct {
		name        string
		tools       map[string]any
		safeOutputs *SafeOutputsConfig
		expected    []string
	}{
		{
			name:     "empty tools",
			tools:    map[string]any{},
			expected: []string{},
		},
		{
			name: "bash with specific commands",
			tools: map[string]any{
				"bash": []any{"echo", "ls"},
			},
			expected: []string{"--allow-tool", "shell(echo)", "--allow-tool", "shell(ls)"},
		},
		{
			name: "bash with wildcard",
			tools: map[string]any{
				"bash": []any{":*"},
			},
			expected: []string{"--allow-all-tools"},
		},
		{
			name: "bash with nil (all commands allowed)",
			tools: map[string]any{
				"bash": nil,
			},
			expected: []string{"--allow-tool", "shell"},
		},
		{
			name: "edit tool",
			tools: map[string]any{
				"edit": nil,
			},
			expected: []string{"--allow-tool", "write"},
		},
		{
			name:  "safe outputs without write (uses MCP)",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expected: []string{"--allow-tool", "safe_outputs"},
		},
		{
			name: "mixed tools",
			tools: map[string]any{
				"bash": []any{"git status", "npm test"},
				"edit": nil,
			},
			expected: []string{"--allow-tool", "shell(git status)", "--allow-tool", "shell(npm test)", "--allow-tool", "write"},
		},
		{
			name: "bash with star wildcard",
			tools: map[string]any{
				"bash": []any{"*"},
			},
			expected: []string{"--allow-all-tools"},
		},
		{
			name: "comprehensive with multiple tools",
			tools: map[string]any{
				"bash": []any{"git status", "npm test"},
				"edit": nil,
			},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expected: []string{"--allow-tool", "safe_outputs", "--allow-tool", "shell(git status)", "--allow-tool", "shell(npm test)", "--allow-tool", "write"},
		},
		{
			name:  "safe outputs with safe_outputs config",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expected: []string{"--allow-tool", "safe_outputs"},
		},
		{
			name:  "safe outputs with safe jobs",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				Jobs: map[string]*SafeJobConfig{
					"my-job": {Name: "test job"},
				},
			},
			expected: []string{"--allow-tool", "safe_outputs"},
		},
		{
			name:  "safe outputs with both safe_outputs and safe jobs",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
				Jobs: map[string]*SafeJobConfig{
					"my-job": {Name: "test job"},
				},
			},
			expected: []string{"--allow-tool", "safe_outputs"},
		},
		{
			name: "github tool with allowed tools",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"get_repository", "list_commits"},
				},
			},
			expected: []string{"--allow-tool", "github(get_repository)", "--allow-tool", "github(list_commits)"},
		},
		{
			name: "github tool with single allowed tool",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"add_issue_comment"},
				},
			},
			expected: []string{"--allow-tool", "github(add_issue_comment)"},
		},
		{
			name: "github tool with wildcard",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"*"},
				},
			},
			expected: []string{"--allow-tool", "github"},
		},
		{
			name: "github tool with wildcard and specific tools",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"*", "get_repository", "list_commits"},
				},
			},
			expected: []string{"--allow-tool", "github", "--allow-tool", "github(get_repository)", "--allow-tool", "github(list_commits)"},
		},
		{
			name: "github tool with empty allowed array",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{},
				},
			},
			expected: []string{},
		},
		{
			name: "github tool without allowed field",
			tools: map[string]any{
				"github": map[string]any{},
			},
			expected: []string{"--allow-tool", "github"},
		},
		{
			name: "github tool as nil (no config)",
			tools: map[string]any{
				"github": nil,
			},
			expected: []string{"--allow-tool", "github"},
		},
		{
			name: "github tool with multiple allowed tools sorted",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"update_issue", "add_issue_comment", "create_issue"},
				},
			},
			expected: []string{"--allow-tool", "github(add_issue_comment)", "--allow-tool", "github(create_issue)", "--allow-tool", "github(update_issue)"},
		},
		{
			name: "github tool with bash and edit tools",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"get_repository", "list_commits"},
				},
				"bash": []any{"echo", "ls"},
				"edit": nil,
			},
			expected: []string{"--allow-tool", "github(get_repository)", "--allow-tool", "github(list_commits)", "--allow-tool", "shell(echo)", "--allow-tool", "shell(ls)", "--allow-tool", "write"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.computeCopilotToolArguments(tt.tools, tt.safeOutputs)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d arguments, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("Expected argument %d to be '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestCopilotEngineGenerateToolArgumentsComment(t *testing.T) {
	engine := NewCopilotEngine()

	tests := []struct {
		name        string
		tools       map[string]any
		safeOutputs *SafeOutputsConfig
		indent      string
		expected    string
	}{
		{
			name:     "empty tools",
			tools:    map[string]any{},
			indent:   "  ",
			expected: "",
		},
		{
			name: "bash with commands",
			tools: map[string]any{
				"bash": []any{"echo", "ls"},
			},
			indent:   "        ",
			expected: "        # Copilot CLI tool arguments (sorted):\n        # --allow-tool shell(echo)\n        # --allow-tool shell(ls)\n",
		},
		{
			name: "edit tool",
			tools: map[string]any{
				"edit": nil,
			},
			indent:   "        ",
			expected: "        # Copilot CLI tool arguments (sorted):\n        # --allow-tool write\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.generateCopilotToolArgumentsComment(tt.tools, tt.safeOutputs, tt.indent)

			if result != tt.expected {
				t.Errorf("Expected comment:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestCopilotEngineExecutionStepsWithToolArguments(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"bash": []any{"echo", "git status"},
			"edit": nil,
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// GetExecutionSteps only returns the execution step
	if len(steps) != 1 {
		t.Fatalf("Expected 1 step (copilot execution), got %d", len(steps))
	}

	// Check the execution step contains tool arguments
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Should contain the tool arguments in the command line
	if !strings.Contains(stepContent, "--allow-tool shell(echo)") {
		t.Errorf("Expected step to contain '--allow-tool shell(echo)' in command:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "--allow-tool shell(git status)") {
		t.Errorf("Expected step to contain '--allow-tool shell(git status)' in command:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "--allow-tool write") {
		t.Errorf("Expected step to contain '--allow-tool write' in command:\n%s", stepContent)
	}

	// Should contain the comment showing the tool arguments
	if !strings.Contains(stepContent, "# Copilot CLI tool arguments (sorted):") {
		t.Errorf("Expected step to contain tool arguments comment:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "# --allow-tool shell(echo)") {
		t.Errorf("Expected step to contain comment for shell(echo):\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "# --allow-tool write") {
		t.Errorf("Expected step to contain comment for write:\n%s", stepContent)
	}

	// Should contain --allow-all-paths for edit tool
	if !strings.Contains(stepContent, "--allow-all-paths") {
		t.Errorf("Expected step to contain '--allow-all-paths' for edit tool:\n%s", stepContent)
	}
}

func TestCopilotEngineEditToolAddsAllowAllPaths(t *testing.T) {
	engine := NewCopilotEngine()

	tests := []struct {
		name       string
		tools      map[string]any
		shouldHave bool
	}{
		{
			name: "edit tool present",
			tools: map[string]any{
				"edit": nil,
			},
			shouldHave: true,
		},
		{
			name: "edit tool with other tools",
			tools: map[string]any{
				"edit": nil,
				"bash": []any{"echo"},
			},
			shouldHave: true,
		},
		{
			name: "no edit tool",
			tools: map[string]any{
				"bash": []any{"echo"},
			},
			shouldHave: false,
		},
		{
			name:       "empty tools",
			tools:      map[string]any{},
			shouldHave: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name:  "test-workflow",
				Tools: tt.tools,
			}
			steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

			// GetExecutionSteps only returns the execution step
			if len(steps) != 1 {
				t.Fatalf("Expected 1 step, got %d", len(steps))
			}

			stepContent := strings.Join([]string(steps[0]), "\n")

			// Check for --allow-all-paths flag
			hasAllowAllPaths := strings.Contains(stepContent, "--allow-all-paths")

			if tt.shouldHave && !hasAllowAllPaths {
				t.Errorf("Expected step to contain '--allow-all-paths' when edit tool is present, but it was missing:\n%s", stepContent)
			}

			if !tt.shouldHave && hasAllowAllPaths {
				t.Errorf("Expected step to NOT contain '--allow-all-paths' when edit tool is absent, but it was present:\n%s", stepContent)
			}

			// When edit tool is present, verify it's in the command line
			if tt.shouldHave {
				lines := strings.Split(stepContent, "\n")
				foundInCommand := false
				for _, line := range lines {
					// When firewall is disabled, it uses 'copilot' instead of 'npx'
					if strings.Contains(line, "copilot") && strings.Contains(line, "--allow-all-paths") {
						foundInCommand = true
						break
					}
				}
				if !foundInCommand {
					t.Errorf("Expected '--allow-all-paths' in copilot command line:\n%s", stepContent)
				}
			}
		})
	}
}

func TestCopilotEngineShellEscaping(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"bash": []any{"git add:*", "git commit:*"},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// GetExecutionSteps only returns the execution step
	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Get the full command from the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Find the line that contains the copilot command
	// When firewall is disabled, it uses 'copilot' instead of 'npx'
	lines := strings.Split(stepContent, "\n")
	var copilotCommand string
	for _, line := range lines {
		if strings.Contains(line, "copilot") && strings.Contains(line, "--allow-tool") {
			copilotCommand = strings.TrimSpace(line)
			break
		}
	}

	if copilotCommand == "" {
		t.Fatalf("Could not find copilot command in step content:\n%s", stepContent)
	}

	// Verify that arguments with special characters are properly quoted
	// This test should fail initially, showing the need for escaping
	t.Logf("Generated command: %s", copilotCommand)

	// The command should contain properly escaped arguments with single quotes
	if !strings.Contains(copilotCommand, "'shell(git add:*)'") {
		t.Errorf("Expected 'shell(git add:*)' to be single-quoted in command: %s", copilotCommand)
	}

	if !strings.Contains(copilotCommand, "'shell(git commit:*)'") {
		t.Errorf("Expected 'shell(git commit:*)' to be single-quoted in command: %s", copilotCommand)
	}
}

func TestCopilotEngineInstructionPromptNotEscaped(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"bash": []any{"git status"},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// GetExecutionSteps only returns the execution step
	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Get the full command from the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Find the line that contains the copilot command
	// When firewall is disabled, it uses 'copilot' instead of 'npx'
	lines := strings.Split(stepContent, "\n")
	var copilotCommand string
	for _, line := range lines {
		if strings.Contains(line, "copilot") && strings.Contains(line, "--prompt") {
			copilotCommand = strings.TrimSpace(line)
			break
		}
	}

	if copilotCommand == "" {
		t.Fatalf("Could not find copilot command in step content:\n%s", stepContent)
	}

	// The $COPILOT_CLI_INSTRUCTION should NOT be wrapped in additional single quotes
	if strings.Contains(copilotCommand, `'"$COPILOT_CLI_INSTRUCTION"'`) {
		t.Errorf("$COPILOT_CLI_INSTRUCTION should not be wrapped in single quotes: %s", copilotCommand)
	}

	// The $COPILOT_CLI_INSTRUCTION should remain double-quoted for variable expansion
	if !strings.Contains(copilotCommand, `"$COPILOT_CLI_INSTRUCTION"`) {
		t.Errorf("$COPILOT_CLI_INSTRUCTION should remain double-quoted: %s", copilotCommand)
	}
}

func TestCopilotEngineRenderGitHubMCPConfig(t *testing.T) {
	engine := NewCopilotEngine()

	tests := []struct {
		name         string
		githubTool   any
		isLast       bool
		expectedStrs []string
	}{
		{
			name:       "GitHub MCP with default version",
			githubTool: nil,
			isLast:     false,
			expectedStrs: []string{
				`"github": {`,
				`"type": "local",`,
				`"command": "docker",`,
				`"args": [`,
				`"run",`,
				`"-i",`,
				`"--rm",`,
				`"-e",`,
				`"GITHUB_PERSONAL_ACCESS_TOKEN",`,
				`"ghcr.io/github/github-mcp-server:v0.19.1"`,
				`"tools": ["*"]`,
				`"env": {`,
				`"GITHUB_PERSONAL_ACCESS_TOKEN": "\${GITHUB_MCP_SERVER_TOKEN}"`,
				`},`,
			},
		},
		{
			name: "GitHub MCP with custom version",
			githubTool: map[string]any{
				"version": "v1.2.3",
			},
			isLast: true,
			expectedStrs: []string{
				`"github": {`,
				`"type": "local",`,
				`"command": "docker",`,
				`"GITHUB_PERSONAL_ACCESS_TOKEN",`,
				`"ghcr.io/github/github-mcp-server:v1.2.3"`,
				`"tools": ["*"]`,
				`"env": {`,
				`"GITHUB_PERSONAL_ACCESS_TOKEN": "\${GITHUB_MCP_SERVER_TOKEN}"`,
				`}`,
			},
		},
		{
			name: "GitHub MCP with allowed tools",
			githubTool: map[string]any{
				"allowed": []string{"list_workflows", "get_repository"},
			},
			isLast: true,
			expectedStrs: []string{
				`"github": {`,
				`"type": "local",`,
				`"tools": [`,
				`"list_workflows"`,
				`"get_repository"`,
				`]`,
				`}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			workflowData := &WorkflowData{}
			var _ *WorkflowData = workflowData
			engine.renderGitHubCopilotMCPConfig(&yaml, tt.githubTool, tt.isLast)
			output := yaml.String()

			for _, expected := range tt.expectedStrs {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't.\nFull output:\n%s", expected, output)
				}
			}

			// Verify proper ending based on isLast
			if tt.isLast {
				if !strings.HasSuffix(strings.TrimSpace(output), "}") {
					t.Errorf("Expected output to end with '}' when isLast=true, got:\n%s", output)
				}
			} else {
				if !strings.HasSuffix(strings.TrimSpace(output), "},") {
					t.Errorf("Expected output to end with '},' when isLast=false, got:\n%s", output)
				}
			}
		})
	}
}

func TestCopilotEngineRenderMCPConfigWithGitHub(t *testing.T) {
	engine := NewCopilotEngine()

	workflowData := &WorkflowData{
		Tools: map[string]any{
			"github": map[string]any{
				"version": "custom-version",
			},
		},
	}

	mcpTools := []string{"github"}
	
	// Build MCP config JSON (the new way)
	jsonConfig, err := BuildMCPConfigJSON(
		workflowData.Tools,
		mcpTools,
		workflowData,
		JSONMCPConfigOptions{
			Renderers: MCPToolRenderers{
				RenderGitHub: func(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData) {
					engine.renderGitHubCopilotMCPConfig(yaml, githubTool, isLast)
				},
				RenderPlaywright:       engine.renderPlaywrightCopilotMCPConfig,
				RenderCacheMemory:      func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {},
				RenderAgenticWorkflows: engine.renderAgenticWorkflowsCopilotMCPConfig,
				RenderSafeOutputs:      engine.renderSafeOutputsCopilotMCPConfig,
				RenderWebFetch: func(yaml *strings.Builder, isLast bool) {
					renderMCPFetchServerConfig(yaml, "json", "              ", isLast, true)
				},
				RenderCustomMCPConfig: engine.renderCopilotMCPConfig,
			},
			FilterTool: func(toolName string) bool {
				return toolName != "cache-memory"
			},
		},
	)
	
	if err != nil {
		t.Fatalf("BuildMCPConfigJSON failed: %v", err)
	}

	// Verify the MCP config structure (JSON is now compacted)
	expectedStrs := []string{
		`"mcpServers":{`,
		`"github":{`,
		`"type":"local"`,
		`"command":"docker"`,
		`"ghcr.io/github/github-mcp-server:custom-version"`,
		`"GITHUB_PERSONAL_ACCESS_TOKEN"`,
		`"env":{`,
		`"GITHUB_PERSONAL_ACCESS_TOKEN":"\${GITHUB_MCP_SERVER_TOKEN}"`,
		`"tools":["*"]`,
	}

	for _, expected := range expectedStrs {
		if !strings.Contains(jsonConfig, expected) {
			t.Errorf("Expected JSON config to contain '%s', but it didn't.\nFull config:\n%s", expected, jsonConfig)
		}
	}
}

func TestCopilotEngineRenderMCPConfigWithGitHubAndPlaywright(t *testing.T) {
	engine := NewCopilotEngine()

	workflowData := &WorkflowData{
		Tools: map[string]any{
			"github":     nil,
			"playwright": nil,
		},
	}

	mcpTools := []string{"github", "playwright"}
	
	// Build MCP config JSON (the new way)
	jsonConfig, err := BuildMCPConfigJSON(
		workflowData.Tools,
		mcpTools,
		workflowData,
		JSONMCPConfigOptions{
			Renderers: MCPToolRenderers{
				RenderGitHub: func(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData) {
					engine.renderGitHubCopilotMCPConfig(yaml, githubTool, isLast)
				},
				RenderPlaywright:       engine.renderPlaywrightCopilotMCPConfig,
				RenderCacheMemory:      func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {},
				RenderAgenticWorkflows: engine.renderAgenticWorkflowsCopilotMCPConfig,
				RenderSafeOutputs:      engine.renderSafeOutputsCopilotMCPConfig,
				RenderWebFetch: func(yaml *strings.Builder, isLast bool) {
					renderMCPFetchServerConfig(yaml, "json", "              ", isLast, true)
				},
				RenderCustomMCPConfig: engine.renderCopilotMCPConfig,
			},
			FilterTool: func(toolName string) bool {
				return toolName != "cache-memory"
			},
		},
	)
	
	if err != nil {
		t.Fatalf("BuildMCPConfigJSON failed: %v", err)
	}

	// Verify both tools are configured (JSON is now compacted, no spaces)
	expectedStrs := []string{
		`"github":{`,
		`"type":"local"`,
		`"command":"docker"`,
		`"ghcr.io/github/github-mcp-server:v0.19.1"`,
		`"playwright":{`,
		`"command":"npx"`,
		`"@playwright/mcp@latest"`,
	}

	for _, expected := range expectedStrs {
		if !strings.Contains(jsonConfig, expected) {
			t.Errorf("Expected JSON config to contain '%s', but it didn't.\nFull config:\n%s", expected, jsonConfig)
		}
	}
}

func TestCopilotEngineGitHubToolsShellEscaping(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"github": map[string]any{
				"allowed": []any{"add_issue_comment", "get_issue"},
			},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// GetExecutionSteps only returns the execution step
	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Get the full command from the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Find the line that contains the copilot command
	// When firewall is disabled, it uses 'copilot' instead of 'npx'
	lines := strings.Split(stepContent, "\n")
	var copilotCommand string
	for _, line := range lines {
		if strings.Contains(line, "copilot") && strings.Contains(line, "--allow-tool") {
			copilotCommand = strings.TrimSpace(line)
			break
		}
	}

	if copilotCommand == "" {
		t.Fatalf("Could not find copilot command in step content:\n%s", stepContent)
	}

	// Verify that GitHub tool arguments are properly single-quoted
	t.Logf("Generated command: %s", copilotCommand)

	// The command should contain properly escaped GitHub tool arguments with single quotes
	if !strings.Contains(copilotCommand, "'github(add_issue_comment)'") {
		t.Errorf("Expected 'github(add_issue_comment)' to be single-quoted in command: %s", copilotCommand)
	}

	if !strings.Contains(copilotCommand, "'github(get_issue)'") {
		t.Errorf("Expected 'github(get_issue)' to be single-quoted in command: %s", copilotCommand)
	}
}

func TestCopilotEngineLogParsingUsesCorrectLogFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "copilot-log-parsing-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with Copilot engine
	testContent := `---
on: push
permissions:
  contents: read
engine: copilot
tools:
  github:
    allowed: [list_issues]
---

# Test Copilot Log Parsing

This workflow tests that Copilot log parsing uses the correct log file path.
`

	testFile := filepath.Join(tmpDir, "test-copilot-log-parsing.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify that the log parsing step uses /tmp/gh-aw/.copilot/logs/ instead of agent-stdio.log
	if !strings.Contains(lockStr, "GH_AW_AGENT_OUTPUT: /tmp/gh-aw/.copilot/logs/") {
		t.Error("Expected GH_AW_AGENT_OUTPUT to be set to '/tmp/gh-aw/.copilot/logs/' for Copilot engine")
	}

	// Verify that it's NOT using the agent-stdio.log path for parsing
	if strings.Contains(lockStr, "GH_AW_AGENT_OUTPUT: /tmp/gh-aw/agent-stdio.log") {
		t.Error("Expected GH_AW_AGENT_OUTPUT to NOT use '/tmp/gh-aw/agent-stdio.log' for Copilot engine")
	}

	t.Log("Successfully verified that Copilot log parsing uses /tmp/gh-aw/.copilot/logs/")
}

func TestExtractAddDirPaths(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "empty args",
			args:     []string{},
			expected: []string{},
		},
		{
			name:     "no add-dir flags",
			args:     []string{"--log-level", "debug", "--model", "gpt-4"},
			expected: []string{},
		},
		{
			name:     "single add-dir",
			args:     []string{"--add-dir", "/tmp/"},
			expected: []string{"/tmp/"},
		},
		{
			name:     "multiple add-dir flags",
			args:     []string{"--add-dir", "/tmp/", "--log-level", "debug", "--add-dir", "/tmp/gh-aw/"},
			expected: []string{"/tmp/", "/tmp/gh-aw/"},
		},
		{
			name:     "add-dir at end of args",
			args:     []string{"--log-level", "debug", "--add-dir", "/tmp/gh-aw/agent/"},
			expected: []string{"/tmp/gh-aw/agent/"},
		},
		{
			name:     "all default copilot args",
			args:     []string{"--add-dir", "/tmp/", "--add-dir", "/tmp/gh-aw/", "--add-dir", "/tmp/gh-aw/agent/", "--log-level", "all", "--log-dir", "/tmp/gh-aw/.copilot/logs/"},
			expected: []string{"/tmp/", "/tmp/gh-aw/", "/tmp/gh-aw/agent/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAddDirPaths(tt.args)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d paths, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("Expected path %d to be '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestCopilotEngineExecutionStepsWithCacheMemory(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		CacheMemoryConfig: &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{
				{ID: "default"},
				{ID: "session"},
				{ID: "logs"},
			},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// GetExecutionSteps only returns the execution step
	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Test that mkdir commands are present for cache-memory directories
	if !strings.Contains(stepContent, "mkdir -p /tmp/gh-aw/cache-memory/") {
		t.Errorf("Expected 'mkdir -p /tmp/gh-aw/cache-memory/' command for default cache in step content:\n%s", stepContent)
	}
	if !strings.Contains(stepContent, "mkdir -p /tmp/gh-aw/cache-memory-session/") {
		t.Errorf("Expected 'mkdir -p /tmp/gh-aw/cache-memory-session/' command for session cache in step content:\n%s", stepContent)
	}
	if !strings.Contains(stepContent, "mkdir -p /tmp/gh-aw/cache-memory-logs/") {
		t.Errorf("Expected 'mkdir -p /tmp/gh-aw/cache-memory-logs/' command for logs cache in step content:\n%s", stepContent)
	}

	// Verify --add-dir flags are present for cache directories
	if !strings.Contains(stepContent, "--add-dir /tmp/gh-aw/cache-memory/") {
		t.Errorf("Expected '--add-dir /tmp/gh-aw/cache-memory/' in copilot args")
	}
	if !strings.Contains(stepContent, "--add-dir /tmp/gh-aw/cache-memory-session/") {
		t.Errorf("Expected '--add-dir /tmp/gh-aw/cache-memory-session/' in copilot args")
	}
	if !strings.Contains(stepContent, "--add-dir /tmp/gh-aw/cache-memory-logs/") {
		t.Errorf("Expected '--add-dir /tmp/gh-aw/cache-memory-logs/' in copilot args")
	}
}

func TestCopilotEngineExecutionStepsWithCustomAddDirArgs(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			Args: []string{"--add-dir", "/custom/path/", "--verbose"},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// GetExecutionSteps only returns the execution step
	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	stepContent := strings.Join([]string(steps[0]), "\n")

	// Test that mkdir commands are present for custom --add-dir path
	if !strings.Contains(stepContent, "mkdir -p /custom/path/") {
		t.Errorf("Expected 'mkdir -p /custom/path/' command for custom add-dir arg in step content:\n%s", stepContent)
	}

	// Verify the custom --add-dir flag is still present in copilot args
	if !strings.Contains(stepContent, "--add-dir /custom/path/") {
		t.Errorf("Expected '--add-dir /custom/path/' in copilot args")
	}
}
