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

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step for Copilot CLI execution, got %d", len(steps))
	}

	// Check the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	if !strings.Contains(stepContent, "name: Execute GitHub Copilot CLI") {
		t.Errorf("Expected step name 'Execute GitHub Copilot CLI' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "copilot --log-level debug --log-dir") {
		t.Errorf("Expected command to contain 'copilot --log-level debug --log-dir' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "/tmp/test.log") {
		t.Errorf("Expected command to contain log file name in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "GITHUB_TOKEN: ${{ secrets.GITHUB_COPILOT_CLI_TOKEN }}") {
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

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step for Copilot CLI execution with output, got %d", len(steps))
	}

	// Check the execution step
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
