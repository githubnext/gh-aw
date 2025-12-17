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
		shouldNotContain []string
	}{
		{
			name:            "version without v prefix",
			version:         "0.0.369",
			stepName:        "Install GitHub Copilot CLI",
			expectedVersion: "v0.0.369",
			shouldContain: []string{
				"COPILOT_VERSION=\"v0.0.369\"",
				"COPILOT_REPO=\"github/copilot-cli\"",
				"releases/download",
				"checksums.txt",
				"sha256sum",
				"Checksum verification",
				"copilot --version",
				"name: Install GitHub Copilot CLI",
			},
			shouldNotContain: []string{
				"gh.io/copilot-install",  // Should not use installer script
			},
		},
		{
			name:            "version with v prefix",
			version:         "v0.0.370",
			stepName:        "Install GitHub Copilot CLI",
			expectedVersion: "v0.0.370",
			shouldContain: []string{
				"COPILOT_VERSION=\"v0.0.370\"",
				"COPILOT_REPO=\"github/copilot-cli\"",
				"releases/download",
				"checksums.txt",
				"sha256sum",
				"Checksum verification",
				"copilot --version",
			},
			shouldNotContain: []string{
				"gh.io/copilot-install",
			},
		},
		{
			name:            "custom version",
			version:         "1.2.3",
			stepName:        "Custom Install Step",
			expectedVersion: "v1.2.3",
			shouldContain: []string{
				"COPILOT_VERSION=\"v1.2.3\"",
				"COPILOT_REPO=\"github/copilot-cli\"",
				"releases/download",
				"checksums.txt",
				"sha256sum",
				"copilot --version",
				"name: Custom Install Step",
			},
			shouldNotContain: []string{
				"gh.io/copilot-install",
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

			// Verify the VERSION is correctly set (with v prefix)
			if !strings.Contains(stepContent, "COPILOT_VERSION=\""+tt.expectedVersion+"\"") {
				t.Errorf("Expected COPILOT_VERSION to be set to '%s', but step content was:\n%s", tt.expectedVersion, stepContent)
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
		if strings.Contains(stepContent, "COPILOT_VERSION=") {
			installStep = stepContent
			break
		}
	}

	if installStep == "" {
		t.Fatal("Could not find install step with COPILOT_VERSION")
	}

	// Should contain the default version from constants (with v prefix added)
	if !strings.Contains(installStep, "COPILOT_VERSION=\"v0.0.369\"") {
		t.Errorf("Expected default version v0.0.369 in install step, got:\n%s", installStep)
	}
	
	// Should contain checksum verification
	if !strings.Contains(installStep, "sha256sum") {
		t.Errorf("Expected checksum verification in install step, got:\n%s", installStep)
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
		if strings.Contains(stepContent, "COPILOT_VERSION=") {
			installStep = stepContent
			break
		}
	}

	if installStep == "" {
		t.Fatal("Could not find install step with COPILOT_VERSION")
	}

	// Should contain the custom version (with v prefix added)
	if !strings.Contains(installStep, "COPILOT_VERSION=\"v"+customVersion+"\"") {
		t.Errorf("Expected custom version v%s in install step, got:\n%s", customVersion, installStep)
	}
	
	// Should contain checksum verification
	if !strings.Contains(installStep, "checksums.txt") {
		t.Errorf("Expected checksums.txt download in install step, got:\n%s", installStep)
	}
}
