package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
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
				"COPILOT_VERSION: ${{ env.GH_AW_COPILOT_VERSION || '0.0.369' }}",
				"/opt/gh-aw/actions/install_copilot_cli.sh \"${COPILOT_VERSION}\"",
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
				"COPILOT_VERSION: ${{ env.GH_AW_COPILOT_VERSION || 'v0.0.370' }}",
				"/opt/gh-aw/actions/install_copilot_cli.sh \"${COPILOT_VERSION}\"",
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
				"COPILOT_VERSION: ${{ env.GH_AW_COPILOT_VERSION || '1.2.3' }}",
				"/opt/gh-aw/actions/install_copilot_cli.sh \"${COPILOT_VERSION}\"",
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
			expectedVersion: string(constants.DefaultCopilotVersion), // Should use DefaultCopilotVersion
			shouldContain: []string{
				"COPILOT_VERSION: ${{ env.GH_AW_COPILOT_VERSION || '" + string(constants.DefaultCopilotVersion) + "' }}",
				"/opt/gh-aw/actions/install_copilot_cli.sh \"${COPILOT_VERSION}\"",
			},
			shouldNotContain: []string{
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

			// Verify the version is correctly set as default in the env var
			expectedVersionLine := "COPILOT_VERSION: ${{ env.GH_AW_COPILOT_VERSION || '" + tt.expectedVersion + "' }}"
			if !strings.Contains(stepContent, expectedVersionLine) {
				t.Errorf("Expected version to be set to '%s' in environment variable, but step content was:\n%s", tt.expectedVersion, stepContent)
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
		if strings.Contains(stepContent, "install_copilot_cli.sh") {
			installStep = stepContent
			break
		}
	}

	if installStep == "" {
		t.Fatal("Could not find install step with install_copilot_cli.sh")
	}

	// Should contain the default version from constants as the fallback in the env var
	expectedVersionLine := "COPILOT_VERSION: ${{ env.GH_AW_COPILOT_VERSION || '" + string(constants.DefaultCopilotVersion) + "' }}"
	if !strings.Contains(installStep, expectedVersionLine) {
		t.Errorf("Expected default version %s in install step environment variable, got:\n%s", string(constants.DefaultCopilotVersion), installStep)
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
		if strings.Contains(stepContent, "install_copilot_cli.sh") {
			installStep = stepContent
			break
		}
	}

	if installStep == "" {
		t.Fatal("Could not find install step with install_copilot_cli.sh")
	}

	// Should contain the custom version as the fallback in the env var
	expectedVersionLine := "COPILOT_VERSION: ${{ env.GH_AW_COPILOT_VERSION || '" + customVersion + "' }}"
	if !strings.Contains(installStep, expectedVersionLine) {
		t.Errorf("Expected custom version %s in install step environment variable, got:\n%s", customVersion, installStep)
	}
}
