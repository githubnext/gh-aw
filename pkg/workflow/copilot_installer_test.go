package workflow

import (
	"strings"
	"testing"
)

func TestGenerateCopilotInstallerSteps(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		stepName        string
		expectedVersion string
		shouldContain   []string
	}{
		{
			name:            "version without v prefix",
			version:         "0.0.369",
			stepName:        "Install GitHub Copilot CLI",
			expectedVersion: "0.0.369",
			shouldContain: []string{
				"export VERSION=0.0.369",
				"curl -fsSL https://gh.io/copilot-install",
				"sudo bash",
				"copilot --version",
			},
		},
		{
			name:            "version with v prefix",
			version:         "v0.0.370",
			stepName:        "Install GitHub Copilot CLI",
			expectedVersion: "v0.0.370",
			shouldContain: []string{
				"export VERSION=v0.0.370",
				"curl -fsSL https://gh.io/copilot-install",
				"sudo bash",
				"copilot --version",
			},
		},
		{
			name:            "custom version",
			version:         "1.2.3",
			stepName:        "Custom Install Step",
			expectedVersion: "1.2.3",
			shouldContain: []string{
				"export VERSION=1.2.3",
				"curl -fsSL https://gh.io/copilot-install",
				"sudo bash",
				"copilot --version",
				"name: Custom Install Step",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := GenerateCopilotInstallerSteps(tt.version, tt.stepName)

			if len(steps) != 1 {
				t.Errorf("Expected 1 step, got %d", len(steps))
				return
			}

			stepContent := strings.Join(steps[0], "\n")

			for _, expected := range tt.shouldContain {
				if !strings.Contains(stepContent, expected) {
					t.Errorf("Expected step to contain '%s', but it didn't.\nStep content:\n%s", expected, stepContent)
				}
			}

			// Verify the VERSION is correctly set
			if !strings.Contains(stepContent, "export VERSION="+tt.expectedVersion) {
				t.Errorf("Expected VERSION to be set to '%s', but step content was:\n%s", tt.expectedVersion, stepContent)
			}
		})
	}
}

func TestCopilotInstallerVersionPassthrough(t *testing.T) {
	// Test that version from constants is correctly passed through
	engine := NewCopilotEngine()

	// Test with default version (no engine config)
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	steps := engine.GetInstallationSteps(workflowData)

	// Should have at least 2 steps (secret validation + install)
	if len(steps) < 2 {
		t.Fatalf("Expected at least 2 steps, got %d", len(steps))
	}

	// Find the install step
	var installStep string
	for _, step := range steps {
		stepContent := strings.Join(step, "\n")
		if strings.Contains(stepContent, "export VERSION=") {
			installStep = stepContent
			break
		}
	}

	if installStep == "" {
		t.Fatal("Could not find install step with VERSION")
	}

	// Should contain the default version from constants
	if !strings.Contains(installStep, "export VERSION=0.0.369") {
		t.Errorf("Expected default version 0.0.369 in install step, got:\n%s", installStep)
	}
}

func TestCopilotInstallerCustomVersion(t *testing.T) {
	// Test that custom version from engine config is used
	engine := NewCopilotEngine()

	customVersion := "1.0.0"
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			Version: customVersion,
		},
	}

	steps := engine.GetInstallationSteps(workflowData)

	// Find the install step
	var installStep string
	for _, step := range steps {
		stepContent := strings.Join(step, "\n")
		if strings.Contains(stepContent, "export VERSION=") {
			installStep = stepContent
			break
		}
	}

	if installStep == "" {
		t.Fatal("Could not find install step with VERSION")
	}

	// Should contain the custom version
	if !strings.Contains(installStep, "export VERSION="+customVersion) {
		t.Errorf("Expected custom version %s in install step, got:\n%s", customVersion, installStep)
	}
}
