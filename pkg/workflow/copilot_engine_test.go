package workflow

import (
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

	// Test with no version
	workflowData := &WorkflowData{}
	steps := engine.GetInstallationSteps(workflowData)
	if len(steps) != 2 {
		t.Errorf("Expected 2 installation steps, got %d", len(steps))
	}

	// Test with version
	workflowDataWithVersion := &WorkflowData{
		EngineConfig: &EngineConfig{Version: "1.0.0"},
	}
	stepsWithVersion := engine.GetInstallationSteps(workflowDataWithVersion)
	if len(stepsWithVersion) != 2 {
		t.Errorf("Expected 2 installation steps with version, got %d", len(stepsWithVersion))
	}
}

func TestCopilotEngineExecutionSteps(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (copilot execution + log capture), got %d", len(steps))
	}

	// Check the execution step (first step)
	stepContent := strings.Join([]string(steps[0]), "\n")

	if !strings.Contains(stepContent, "name: Execute GitHub Copilot CLI") {
		t.Errorf("Expected step name 'Execute GitHub Copilot CLI' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "copilot --add-dir /tmp/ --log-level all --log-dir") {
		t.Errorf("Expected command to contain 'copilot --add-dir /tmp/ --log-level all --log-dir' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "/tmp/test.log") {
		t.Errorf("Expected command to contain log file name in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "GITHUB_TOKEN: ${{ secrets.COPILOT_CLI_TOKEN  }}") {
		t.Errorf("Expected GITHUB_TOKEN environment variable in step content:\n%s", stepContent)
	}

	// Test that GITHUB_AW_SAFE_OUTPUTS is not present when SafeOutputs is nil
	if strings.Contains(stepContent, "GITHUB_AW_SAFE_OUTPUTS") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS to not be present when SafeOutputs is nil")
	}
}

func TestCopilotEngineExecutionStepsWithOutput(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		SafeOutputs: &SafeOutputsConfig{}, // Non-nil to trigger output handling
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (copilot execution + log capture) with output, got %d", len(steps))
	}

	// Check the execution step (first step)
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Test that GITHUB_AW_SAFE_OUTPUTS is present when SafeOutputs is not nil
	if !strings.Contains(stepContent, "GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}") {
		t.Errorf("Expected GITHUB_AW_SAFE_OUTPUTS environment variable when SafeOutputs is not nil in step content:\n%s", stepContent)
	}
}

func TestCopilotEngineGetLogParserScript(t *testing.T) {
	engine := NewCopilotEngine()
	script := engine.GetLogParserScriptId()

	if script != "parse_copilot_log" {
		t.Errorf("Expected 'parse_copilot_log', got '%s'", script)
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
			expected: []string{"--allow-tool", "shell"},
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
			expected: []string{"--allow-tool", "shell"},
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
			expected: []string{},
		},
		{
			name: "github tool as nil (no config)",
			tools: map[string]any{
				"github": nil,
			},
			expected: []string{},
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
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (copilot execution + log capture), got %d", len(steps))
	}

	// Check the execution step contains tool arguments (first step)
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
}

func TestCopilotEngineShellEscaping(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"bash": []any{"git add:*", "git commit:*"},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(steps))
	}

	// Get the full command from the execution step (step 0 is the copilot execution)
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Find the line that contains the copilot command
	lines := strings.Split(stepContent, "\n")
	var copilotCommand string
	for _, line := range lines {
		if strings.Contains(line, "copilot ") && strings.Contains(line, "--allow-tool") {
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
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(steps))
	}

	// Get the full command from the execution step (step 0 is the copilot execution)
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Find the line that contains the copilot command
	lines := strings.Split(stepContent, "\n")
	var copilotCommand string
	for _, line := range lines {
		if strings.Contains(line, "copilot ") && strings.Contains(line, "--prompt") {
			copilotCommand = strings.TrimSpace(line)
			break
		}
	}

	if copilotCommand == "" {
		t.Fatalf("Could not find copilot command in step content:\n%s", stepContent)
	}

	// The $INSTRUCTION should NOT be wrapped in additional single quotes
	if strings.Contains(copilotCommand, `'"$INSTRUCTION"'`) {
		t.Errorf("$INSTRUCTION should not be wrapped in single quotes: %s", copilotCommand)
	}

	// The $INSTRUCTION should remain double-quoted for variable expansion
	if !strings.Contains(copilotCommand, `"$INSTRUCTION"`) {
		t.Errorf("$INSTRUCTION should remain double-quoted: %s", copilotCommand)
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
				`"GITHUB_PERSONAL_ACCESS_TOKEN=${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",`,
				`"ghcr.io/github/github-mcp-server:sha-09deac4"`,
				`"tools": ["*"]`,
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
				`"GITHUB_PERSONAL_ACCESS_TOKEN=${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",`,
				`"ghcr.io/github/github-mcp-server:v1.2.3"`,
				`"tools": ["*"]`,
				`}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			workflowData := &WorkflowData{}
			engine.renderGitHubCopilotMCPConfig(&yaml, tt.githubTool, tt.isLast, workflowData)
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
	var yaml strings.Builder
	engine.RenderMCPConfig(&yaml, workflowData.Tools, mcpTools, workflowData)
	output := yaml.String()

	// Verify the MCP config structure
	expectedStrs := []string{
		"mkdir -p /home/runner/.copilot",
		`cat > /home/runner/.copilot/mcp-config.json << 'EOF'`,
		`"mcpServers": {`,
		`"github": {`,
		`"type": "local",`,
		`"command": "docker",`,
		`"ghcr.io/github/github-mcp-server:custom-version"`,
		`"GITHUB_PERSONAL_ACCESS_TOKEN=${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",`,
		`"tools": ["*"]`,
		"EOF",
		"-------START MCP CONFIG-----------",
		"cat /home/runner/.copilot/mcp-config.json",
		"-------END MCP CONFIG-----------",
	}

	for _, expected := range expectedStrs {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't.\nFull output:\n%s", expected, output)
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
	var yaml strings.Builder
	engine.RenderMCPConfig(&yaml, workflowData.Tools, mcpTools, workflowData)
	output := yaml.String()

	// Verify both tools are configured
	expectedStrs := []string{
		`"github": {`,
		`"type": "local",`,
		`"command": "docker",`,
		`"ghcr.io/github/github-mcp-server:sha-09deac4"`,
		`},`, // GitHub should NOT be last (comma after closing brace)
		`"playwright": {`,
		`"type": "local",`,
		`"command": "npx",`,
	}

	for _, expected := range expectedStrs {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't.\nFull output:\n%s", expected, output)
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
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(steps))
	}

	// Get the full command from the execution step (step 0 is the copilot execution)
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Find the line that contains the copilot command
	lines := strings.Split(stepContent, "\n")
	var copilotCommand string
	for _, line := range lines {
		if strings.Contains(line, "copilot ") && strings.Contains(line, "--allow-tool") {
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
