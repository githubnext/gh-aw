package workflow

import (
	"regexp"
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

	if !engine.IsExperimental() {
		t.Error("Expected copilot engine to be experimental")
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
	if len(steps) != 3 {
		t.Errorf("Expected 3 installation steps, got %d", len(steps))
	}

	// Test with version
	workflowDataWithVersion := &WorkflowData{
		EngineConfig: &EngineConfig{Version: "1.0.0"},
	}
	stepsWithVersion := engine.GetInstallationSteps(workflowDataWithVersion)
	if len(stepsWithVersion) != 3 {
		t.Errorf("Expected 3 installation steps with version, got %d", len(stepsWithVersion))
	}
}

func TestCopilotEngineExecutionSteps(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 3 {
		t.Fatalf("Expected 3 steps (upload config + copilot execution + log capture), got %d", len(steps))
	}

	// Check the execution step (second step)
	stepContent := strings.Join([]string(steps[1]), "\n")

	if !strings.Contains(stepContent, "name: Execute GitHub Copilot CLI") {
		t.Errorf("Expected step name 'Execute GitHub Copilot CLI' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "copilot --add-dir /tmp/ --log-level debug --log-dir") {
		t.Errorf("Expected command to contain 'copilot --add-dir /tmp/ --log-level debug --log-dir' in step content:\n%s", stepContent)
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

	if len(steps) != 3 {
		t.Fatalf("Expected 3 steps (upload config + copilot execution + log capture) with output, got %d", len(steps))
	}

	// Check the execution step (second step)
	stepContent := strings.Join([]string(steps[1]), "\n")

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

	if len(steps) != 3 {
		t.Fatalf("Expected 3 steps (upload config + copilot execution + log capture), got %d", len(steps))
	}

	// Check the upload config step (first step)
	uploadStepContent := strings.Join([]string(steps[0]), "\n")
	if !strings.Contains(uploadStepContent, "name: Upload config") {
		t.Errorf("Expected first step to be upload config step:\n%s", uploadStepContent)
	}

	if !strings.Contains(uploadStepContent, "uses: actions/upload-artifact@v4") {
		t.Errorf("Expected upload step to use actions/upload-artifact@v4:\n%s", uploadStepContent)
	}

	if !strings.Contains(uploadStepContent, "name: config") {
		t.Errorf("Expected artifact name to be 'config':\n%s", uploadStepContent)
	}

	if !strings.Contains(uploadStepContent, "path: /tmp/.copilot/") {
		t.Errorf("Expected artifact path to be '/tmp/.copilot/':\n%s", uploadStepContent)
	}

	// Check the execution step contains tool arguments (second step)
	stepContent := strings.Join([]string(steps[1]), "\n")

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

func TestCopilotEngineUploadConfigStep(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	if len(steps) != 3 {
		t.Fatalf("Expected 3 steps (upload config + copilot execution + log capture), got %d", len(steps))
	}

	// Check the upload config step is present and correct
	uploadStepContent := strings.Join([]string(steps[0]), "\n")

	if !strings.Contains(uploadStepContent, "name: Upload config") {
		t.Errorf("Expected upload config step name in:\n%s", uploadStepContent)
	}

	if !strings.Contains(uploadStepContent, "if: always()") {
		t.Errorf("Expected upload config step to have 'if: always()' condition in:\n%s", uploadStepContent)
	}

	if !strings.Contains(uploadStepContent, "uses: actions/upload-artifact@v4") {
		t.Errorf("Expected upload config step to use 'actions/upload-artifact@v4' in:\n%s", uploadStepContent)
	}

	if !strings.Contains(uploadStepContent, "name: config") {
		t.Errorf("Expected artifact name 'config' in:\n%s", uploadStepContent)
	}

	if !strings.Contains(uploadStepContent, "path: /tmp/.copilot/") {
		t.Errorf("Expected artifact path '/tmp/.copilot/' in:\n%s", uploadStepContent)
	}

	if !strings.Contains(uploadStepContent, "if-no-files-found: ignore") {
		t.Errorf("Expected 'if-no-files-found: ignore' in:\n%s", uploadStepContent)
	}
}

func TestCopilotEngineErrorPatterns(t *testing.T) {
	engine := NewCopilotEngine()
	patterns := engine.GetErrorPatterns()

	// Test that error patterns can match content from the sample log
	tests := []struct {
		expectedPattern string
		expectedMatch   bool
		sampleText      string
	}{
		{
			expectedPattern: "Copilot CLI timestamped ERROR messages",
			expectedMatch:   true,
			sampleText:      "2024-09-16T18:10:36.123Z [ERROR] Failed to save final output: Permission denied",
		},
		{
			expectedPattern: "NPM error messages during Copilot CLI installation or execution",
			expectedMatch:   true,
			sampleText:      "npm ERR! Could not install package @github/copilot",
		},
		{
			expectedPattern: "Copilot CLI command-level error messages",
			expectedMatch:   true,
			sampleText:      "copilot: error: Invalid authentication token provided",
		},
		{
			expectedPattern: "Fatal error messages from Copilot CLI",
			expectedMatch:   true,
			sampleText:      "Fatal error: Unable to complete workflow execution",
		},
		{
			expectedPattern: "Generic warning messages from Copilot CLI",
			expectedMatch:   true,
			sampleText:      "Warning: Some tools may not be available in restricted mode",
		},
		{
			expectedPattern: "Copilot CLI shell command execution errors",
			expectedMatch:   true,
			sampleText:      "2024-09-16T18:10:33.500Z [ERROR] Shell command failed: command not found",
		},
		{
			expectedPattern: "Copilot CLI MCP server connection errors",
			expectedMatch:   true,
			sampleText:      "2024-09-16T18:10:31.200Z [ERROR] Failed to connect to broken_server MCP server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expectedPattern, func(t *testing.T) {
			found := false
			for _, pattern := range patterns {
				if pattern.Description == tt.expectedPattern {
					found = true
					// Test if the pattern matches the sample text
					re, err := regexp.Compile(pattern.Pattern)
					if err != nil {
						t.Fatalf("Invalid regex pattern '%s': %v", pattern.Pattern, err)
					}

					matches := re.FindStringSubmatch(tt.sampleText)
					if tt.expectedMatch && len(matches) == 0 {
						t.Errorf("Pattern '%s' should match text '%s' but didn't", pattern.Pattern, tt.sampleText)
					} else if !tt.expectedMatch && len(matches) > 0 {
						t.Errorf("Pattern '%s' should not match text '%s' but did", pattern.Pattern, tt.sampleText)
					}

					// Test that message extraction works correctly
					if tt.expectedMatch && len(matches) > 0 {
						messageGroup := pattern.MessageGroup
						if messageGroup > 0 && messageGroup < len(matches) {
							message := strings.TrimSpace(matches[messageGroup])
							if message == "" {
								t.Errorf("Pattern '%s' extracted empty message from '%s'", pattern.Pattern, tt.sampleText)
							}
						}
					}
					break
				}
			}
			if !found {
				t.Errorf("Expected error pattern with description '%s' not found", tt.expectedPattern)
			}
		})
	}

	// Verify we have a reasonable number of patterns
	if len(patterns) < 8 {
		t.Errorf("Expected at least 8 error patterns, got %d", len(patterns))
	}
}
