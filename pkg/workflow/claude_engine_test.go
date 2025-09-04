package workflow

import (
	"fmt"
	"testing"
)

func TestClaudeEngine(t *testing.T) {
	engine := NewClaudeEngine()

	// Test basic properties
	if engine.GetID() != "claude" {
		t.Errorf("Expected ID 'claude', got '%s'", engine.GetID())
	}

	if engine.GetDisplayName() != "Claude Code" {
		t.Errorf("Expected display name 'Claude Code', got '%s'", engine.GetDisplayName())
	}

	if engine.GetDescription() != "Uses Claude Code with full MCP tool support and allow-listing" {
		t.Errorf("Expected description 'Uses Claude Code with full MCP tool support and allow-listing', got '%s'", engine.GetDescription())
	}

	if engine.IsExperimental() {
		t.Error("Claude engine should not be experimental")
	}

	if !engine.SupportsToolsWhitelist() {
		t.Error("Claude engine should support MCP tools")
	}

	// Test installation steps (should be empty for Claude)
	steps := engine.GetInstallationSteps(&WorkflowData{})
	if len(steps) != 0 {
		t.Errorf("Expected no installation steps for Claude, got %v", steps)
	}

	// Test execution config
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	config := engine.GetExecutionConfig(workflowData, "test-log")
	if config.StepName != "Execute Claude Code Action" {
		t.Errorf("Expected step name 'Execute Claude Code Action', got '%s'", config.StepName)
	}

	if config.Action != fmt.Sprintf("anthropics/claude-code-base-action@%s", DefaultClaudeActionVersion) {
		t.Errorf("Expected action 'anthropics/claude-code-base-action@%s', got '%s'", DefaultClaudeActionVersion, config.Action)
	}

	if config.Command != "" {
		t.Errorf("Expected empty command for Claude (uses action), got '%s'", config.Command)
	}

	// Check that required inputs are present
	if config.Inputs["prompt_file"] != "/tmp/aw-prompts/prompt.txt" {
		t.Errorf("Expected prompt_file input, got '%s'", config.Inputs["prompt_file"])
	}

	if config.Inputs["anthropic_api_key"] != "${{ secrets.ANTHROPIC_API_KEY }}" {
		t.Errorf("Expected anthropic_api_key input, got '%s'", config.Inputs["anthropic_api_key"])
	}

	if config.Inputs["mcp_config"] != "/tmp/mcp-config/mcp-servers.json" {
		t.Errorf("Expected mcp_config input, got '%s'", config.Inputs["mcp_config"])
	}

	// claude_env should not be present when hasOutput=false (security improvement)
	if _, hasClaudeEnv := config.Inputs["claude_env"]; hasClaudeEnv {
		t.Errorf("Expected no claude_env input for security reasons, but got: '%s'", config.Inputs["claude_env"])
	}

	// Check that special fields are present but empty (will be filled during generation)
	if _, hasAllowedTools := config.Inputs["allowed_tools"]; !hasAllowedTools {
		t.Error("Expected allowed_tools input to be present")
	}

	if _, hasTimeoutMinutes := config.Inputs["timeout_minutes"]; !hasTimeoutMinutes {
		t.Error("Expected timeout_minutes input to be present")
	}

	if _, hasMaxTurns := config.Inputs["max_turns"]; !hasMaxTurns {
		t.Error("Expected max_turns input to be present")
	}
}

func TestClaudeEngineWithOutput(t *testing.T) {
	engine := NewClaudeEngine()

	// Test execution config with hasOutput=true
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		SafeOutputs: &SafeOutputsConfig{}, // non-nil means hasOutput=true
	}
	config := engine.GetExecutionConfig(workflowData, "test-log")

	// Should include GITHUB_AW_SAFE_OUTPUTS when hasOutput=true, but no GH_TOKEN for security
	expectedClaudeEnv := "|\n            GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}"
	if config.Inputs["claude_env"] != expectedClaudeEnv {
		t.Errorf("Expected claude_env input with output '%s', got '%s'", expectedClaudeEnv, config.Inputs["claude_env"])
	}
}

func TestClaudeEngineConfiguration(t *testing.T) {
	engine := NewClaudeEngine()

	// Test different workflow names and log files
	testCases := []struct {
		workflowName string
		logFile      string
	}{
		{"simple-workflow", "simple-log"},
		{"complex workflow with spaces", "complex-log"},
		{"workflow-with-hyphens", "workflow-log"},
	}

	for _, tc := range testCases {
		t.Run(tc.workflowName, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name: tc.workflowName,
			}
			config := engine.GetExecutionConfig(workflowData, tc.logFile)

			// Verify the configuration is consistent regardless of input
			if config.StepName != "Execute Claude Code Action" {
				t.Errorf("Expected step name 'Execute Claude Code Action', got '%s'", config.StepName)
			}

			if config.Action != fmt.Sprintf("anthropics/claude-code-base-action@%s", DefaultClaudeActionVersion) {
				t.Errorf("Expected action 'anthropics/claude-code-base-action@%s', got '%s'", DefaultClaudeActionVersion, config.Action)
			}

			// Verify all required inputs are present (except claude_env when hasOutput=false for security)
			requiredInputs := []string{"prompt_file", "anthropic_api_key", "mcp_config", "allowed_tools", "timeout_minutes", "max_turns"}
			for _, input := range requiredInputs {
				if _, exists := config.Inputs[input]; !exists {
					t.Errorf("Expected input '%s' to be present", input)
				}
			}

			// claude_env should not be present when hasOutput=false (security improvement)
			if _, hasClaudeEnv := config.Inputs["claude_env"]; hasClaudeEnv {
				t.Errorf("Expected no claude_env input for security reasons when hasOutput=false")
			}
		})
	}
}

func TestClaudeEngineWithVersion(t *testing.T) {
	engine := NewClaudeEngine()

	// Test with custom version
	engineConfig := &EngineConfig{
		ID:      "claude",
		Version: "v1.2.3",
		Model:   "claude-3-5-sonnet-20241022",
	}

	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: engineConfig,
	}

	config := engine.GetExecutionConfig(workflowData, "test-log")

	// Check that the version is correctly used in the action
	expectedAction := "anthropics/claude-code-base-action@v1.2.3"
	if config.Action != expectedAction {
		t.Errorf("Expected action '%s', got '%s'", expectedAction, config.Action)
	}

	// Check that model is set
	if config.Inputs["model"] != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got '%s'", config.Inputs["model"])
	}
}

func TestClaudeEngineWithoutVersion(t *testing.T) {
	engine := NewClaudeEngine()

	// Test without version (should use default)
	engineConfig := &EngineConfig{
		ID:    "claude",
		Model: "claude-3-5-sonnet-20241022",
	}

	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: engineConfig,
	}

	config := engine.GetExecutionConfig(workflowData, "test-log")

	// Check that default version is used
	expectedAction := fmt.Sprintf("anthropics/claude-code-base-action@%s", DefaultClaudeActionVersion)
	if config.Action != expectedAction {
		t.Errorf("Expected action '%s', got '%s'", expectedAction, config.Action)
	}
}
