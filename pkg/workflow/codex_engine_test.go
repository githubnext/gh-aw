package workflow

import (
	"strings"
	"testing"
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

	if !engine.IsExperimental() {
		t.Error("Codex engine should be experimental")
	}

	if !engine.SupportsToolsWhitelist() {
		t.Error("Codex engine should support MCP tools")
	}

	// Test installation steps
	steps := engine.GetInstallationSteps(nil)
	expectedStepCount := 2 // Setup Node.js and Install Codex
	if len(steps) != expectedStepCount {
		t.Errorf("Expected %d installation steps, got %d", expectedStepCount, len(steps))
	}

	// Verify first step is Setup Node.js
	if len(steps) > 0 && len(steps[0]) > 0 {
		if !strings.Contains(steps[0][0], "Setup Node.js") {
			t.Errorf("Expected first step to contain 'Setup Node.js', got '%s'", steps[0][0])
		}
	}

	// Verify second step is Install Codex
	if len(steps) > 1 && len(steps[1]) > 0 {
		if !strings.Contains(steps[1][0], "Install Codex") {
			t.Errorf("Expected second step to contain 'Install Codex', got '%s'", steps[1][0])
		}
	}

	// Test execution config
	config := engine.GetExecutionConfig("test-workflow", "test-log", nil, false)
	if config.StepName != "Run Codex" {
		t.Errorf("Expected step name 'Run Codex', got '%s'", config.StepName)
	}

	if config.Action != "" {
		t.Errorf("Expected empty action for Codex (uses command), got '%s'", config.Action)
	}

	if !strings.Contains(config.Command, "codex exec") {
		t.Errorf("Expected command to contain 'codex exec', got '%s'", config.Command)
	}

	if !strings.Contains(config.Command, "test-log") {
		t.Errorf("Expected command to contain log file name, got '%s'", config.Command)
	}

	// Check environment variables
	if config.Environment["OPENAI_API_KEY"] != "${{ secrets.OPENAI_API_KEY }}" {
		t.Errorf("Expected OPENAI_API_KEY environment variable, got '%s'", config.Environment["OPENAI_API_KEY"])
	}
}

func TestCodexEngineWithVersion(t *testing.T) {
	engine := NewCodexEngine()

	// Test installation steps without version
	stepsNoVersion := engine.GetInstallationSteps(nil)
	foundNoVersionInstall := false
	for _, step := range stepsNoVersion {
		for _, line := range step {
			if strings.Contains(line, "npm install -g @openai/codex") && !strings.Contains(line, "@openai/codex@") {
				foundNoVersionInstall = true
				break
			}
		}
	}
	if !foundNoVersionInstall {
		t.Error("Expected default npm install command without version")
	}

	// Test installation steps with version
	engineConfig := &EngineConfig{
		ID:      "codex",
		Version: "3.0.1",
	}
	stepsWithVersion := engine.GetInstallationSteps(engineConfig)
	foundVersionInstall := false
	for _, step := range stepsWithVersion {
		for _, line := range step {
			if strings.Contains(line, "npm install -g @openai/codex@3.0.1") {
				foundVersionInstall = true
				break
			}
		}
	}
	if !foundVersionInstall {
		t.Error("Expected versioned npm install command with @openai/codex@3.0.1")
	}
}
