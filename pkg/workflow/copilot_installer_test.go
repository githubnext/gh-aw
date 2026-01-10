package workflow

import (
	"strings"
	"testing"
)

func TestGenerateCopilotInstallerSteps(t *testing.T) {
	tests := []struct {
		name             string
		version          string
		stepName         string
		expectedVersion  string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:            "version without v prefix",
			version:         "0.0.369",
			stepName:        "Install GitHub Copilot CLI",
			expectedVersion: "0.0.369",
			shouldContain: []string{
				"export VERSION=0.0.369",
				"https://raw.githubusercontent.com/github/copilot-cli/main/install.sh",
				"sudo bash /tmp/copilot-install.sh",
				"copilot --version",
				"name: Install GitHub Copilot CLI",
			},
			shouldNotContain: []string{
				"gh.io/copilot-install | sudo bash", // Should not pipe directly to bash
			},
		},
		{
			name:            "version with v prefix",
			version:         "v0.0.370",
			stepName:        "Install GitHub Copilot CLI",
			expectedVersion: "v0.0.370",
			shouldContain: []string{
				"export VERSION=v0.0.370",
				"https://raw.githubusercontent.com/github/copilot-cli/main/install.sh",
				"sudo bash /tmp/copilot-install.sh",
				"copilot --version",
			},
			shouldNotContain: []string{
				"gh.io/copilot-install | sudo bash",
			},
		},
		{
			name:            "custom version",
			version:         "1.2.3",
			stepName:        "Custom Install Step",
			expectedVersion: "1.2.3",
			shouldContain: []string{
				"export VERSION=1.2.3",
				"https://raw.githubusercontent.com/github/copilot-cli/main/install.sh",
				"sudo bash /tmp/copilot-install.sh",
				"copilot --version",
				"name: Custom Install Step",
			},
			shouldNotContain: []string{
				"gh.io/copilot-install | sudo bash",
			},
		},
		{
			name:            "empty version uses default",
			version:         "",
			stepName:        "Install GitHub Copilot CLI",
			expectedVersion: "0.0.376", // Should use DefaultCopilotVersion
			shouldContain: []string{
				"export VERSION=0.0.376",
				"https://raw.githubusercontent.com/github/copilot-cli/main/install.sh",
				"sudo bash /tmp/copilot-install.sh",
				"copilot --version",
			},
			shouldNotContain: []string{
				"export VERSION= &&", // Should not have empty version
				"gh.io/copilot-install | sudo bash",
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

			// Check expected content
			for _, expected := range tt.shouldContain {
				if !strings.Contains(stepContent, expected) {
					t.Errorf("Expected step to contain '%s', but it didn't.\nStep content:\n%s", expected, stepContent)
				}
			}

			// Check content that should not be present
			for _, notExpected := range tt.shouldNotContain {
				if strings.Contains(stepContent, notExpected) {
					t.Errorf("Expected step NOT to contain '%s', but it did.\nStep content:\n%s", notExpected, stepContent)
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
		t.Fatal("Could not find install step with export VERSION")
	}

	// Should contain the default version from constants
	if !strings.Contains(installStep, "export VERSION=0.0.376") {
		t.Errorf("Expected default version 0.0.376 in install step, got:\n%s", installStep)
	}

	// Should use the official install.sh script
	if !strings.Contains(installStep, "https://raw.githubusercontent.com/github/copilot-cli/main/install.sh") {
		t.Errorf("Expected official install.sh script in install step, got:\n%s", installStep)
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
		t.Fatal("Could not find install step with export VERSION")
	}

	// Should contain the custom version
	if !strings.Contains(installStep, "export VERSION="+customVersion) {
		t.Errorf("Expected custom version %s in install step, got:\n%s", customVersion, installStep)
	}

	// Should download to file before executing
	if !strings.Contains(installStep, "sudo bash /tmp/copilot-install.sh") {
		t.Errorf("Expected script to be downloaded to file before execution, got:\n%s", installStep)
	}
}
