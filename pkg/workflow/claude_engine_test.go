package workflow

import (
	"fmt"
	"strings"
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
	installSteps := engine.GetInstallationSteps(&WorkflowData{})
	if len(installSteps) != 0 {
		t.Errorf("Expected no installation steps for Claude, got %v", installSteps)
	}

	// Test execution steps
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepLines := []string(executionStep)

	// Check step name
	found := false
	for _, line := range stepLines {
		if strings.Contains(line, "name: Execute Claude Code Action") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected step name 'Execute Claude Code Action' in step lines: %v", stepLines)
	}

	// Check action usage
	found = false
	expectedAction := fmt.Sprintf("anthropics/claude-code-base-action@%s", DefaultClaudeActionVersion)
	for _, line := range stepLines {
		if strings.Contains(line, "uses: "+expectedAction) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected action '%s' in step lines: %v", expectedAction, stepLines)
	}

	// Check that required inputs are present
	stepContent := strings.Join(stepLines, "\n")
	if !strings.Contains(stepContent, "prompt_file: /tmp/aw-prompts/prompt.txt") {
		t.Errorf("Expected prompt_file input in step: %s", stepContent)
	}

	if !strings.Contains(stepContent, "anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}") {
		t.Errorf("Expected anthropic_api_key input in step: %s", stepContent)
	}

	if !strings.Contains(stepContent, "mcp_config: /tmp/mcp-config/mcp-servers.json") {
		t.Errorf("Expected mcp_config input in step: %s", stepContent)
	}

	// claude_env should not be present when hasOutput=false (security improvement)
	if strings.Contains(stepContent, "claude_env:") {
		t.Errorf("Expected no claude_env input for security reasons, but got it in step: %s", stepContent)
	}

	// Check that special fields are present but empty (will be filled during generation)
	if !strings.Contains(stepContent, "allowed_tools:") {
		t.Error("Expected allowed_tools input to be present in step")
	}

	if !strings.Contains(stepContent, "timeout_minutes:") {
		t.Error("Expected timeout_minutes input to be present in step")
	}

	// max_turns should NOT be present when not specified in engine config
	if strings.Contains(stepContent, "max_turns:") {
		t.Error("Expected max_turns input to NOT be present when not specified in engine config")
	}
}

func TestClaudeEngineWithOutput(t *testing.T) {
	engine := NewClaudeEngine()

	// Test execution steps with hasOutput=true
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		SafeOutputs: &SafeOutputsConfig{}, // non-nil means hasOutput=true
	}
	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepContent := strings.Join([]string(executionStep), "\n")

	// Should include GITHUB_AW_SAFE_OUTPUTS when hasOutput=true, but no GH_TOKEN for security
	expectedClaudeEnv := "claude_env: |\n            GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}"
	if !strings.Contains(stepContent, expectedClaudeEnv) {
		t.Errorf("Expected claude_env input with output '%s' in step content:\n%s", expectedClaudeEnv, stepContent)
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
			steps := engine.GetExecutionSteps(workflowData, tc.logFile)
			if len(steps) != 2 {
				t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
			}

			// Check the main execution step
			executionStep := steps[0]
			stepContent := strings.Join([]string(executionStep), "\n")

			// Verify the step contains expected content regardless of input
			if !strings.Contains(stepContent, "name: Execute Claude Code Action") {
				t.Errorf("Expected step name 'Execute Claude Code Action' in step content")
			}

			expectedAction := fmt.Sprintf("anthropics/claude-code-base-action@%s", DefaultClaudeActionVersion)
			if !strings.Contains(stepContent, "uses: "+expectedAction) {
				t.Errorf("Expected action '%s' in step content", expectedAction)
			}

			// Verify all required inputs are present (except claude_env when hasOutput=false for security)
			// max_turns is only present when specified in engine config
			requiredInputs := []string{"prompt_file", "anthropic_api_key", "mcp_config", "allowed_tools", "timeout_minutes"}
			for _, input := range requiredInputs {
				if !strings.Contains(stepContent, input+":") {
					t.Errorf("Expected input '%s' to be present in step content", input)
				}
			}

			// claude_env should not be present when hasOutput=false (security improvement)
			if strings.Contains(stepContent, "claude_env:") {
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

	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepContent := strings.Join([]string(executionStep), "\n")

	// Check that the version is correctly used in the action
	expectedAction := "anthropics/claude-code-base-action@v1.2.3"
	if !strings.Contains(stepContent, "uses: "+expectedAction) {
		t.Errorf("Expected action '%s' in step content:\n%s", expectedAction, stepContent)
	}

	// Check that model is set
	if !strings.Contains(stepContent, "model: claude-3-5-sonnet-20241022") {
		t.Errorf("Expected model 'claude-3-5-sonnet-20241022' in step content:\n%s", stepContent)
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

	steps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps (execution + log capture), got %d", len(steps))
	}

	// Check the main execution step
	executionStep := steps[0]
	stepContent := strings.Join([]string(executionStep), "\n")

	// Check that default version is used
	expectedAction := fmt.Sprintf("anthropics/claude-code-base-action@%s", DefaultClaudeActionVersion)
	if !strings.Contains(stepContent, "uses: "+expectedAction) {
		t.Errorf("Expected action '%s' in step content:\n%s", expectedAction, stepContent)
	}
}

func TestClaudeEngineConvertStepToYAMLWithSection(t *testing.T) {
	engine := NewClaudeEngine()

	// Test step with 'with' section to ensure keys are sorted
	stepMap := map[string]any{
		"name": "Test step with sorted with section",
		"uses": "actions/checkout@v4",
		"with": map[string]any{
			"zebra": "value-z",
			"alpha": "value-a",
			"beta":  "value-b",
			"gamma": "value-g",
		},
	}

	yaml, err := engine.convertStepToYAML(stepMap)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify that the with keys are in alphabetical order
	lines := strings.Split(yaml, "\n")
	withSection := false
	withKeyOrder := []string{}

	for _, line := range lines {
		if strings.TrimSpace(line) == "with:" {
			withSection = true
			continue
		}
		if withSection && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			// End of with section if we hit another top-level key
			break
		}
		if withSection && strings.Contains(line, ":") {
			// Extract the key (before the colon)
			parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
			if len(parts) > 0 {
				withKeyOrder = append(withKeyOrder, strings.TrimSpace(parts[0]))
			}
		}
	}

	expectedOrder := []string{"alpha", "beta", "gamma", "zebra"}
	if len(withKeyOrder) != len(expectedOrder) {
		t.Errorf("Expected %d with keys, got %d", len(expectedOrder), len(withKeyOrder))
	}

	for i, key := range expectedOrder {
		if i >= len(withKeyOrder) || withKeyOrder[i] != key {
			t.Errorf("Expected with key at position %d to be '%s', got '%s'. Full order: %v", i, key, withKeyOrder[i], withKeyOrder)
		}
	}
}
