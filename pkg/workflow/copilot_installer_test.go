//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
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
				"bash \"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh\" 0.0.369",
				"name: Install GitHub Copilot CLI",
				"GH_HOST: github.com", // Must pin GH_HOST to prevent GHES workflow-level overrides
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
				"bash \"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh\" v0.0.370",
				"GH_HOST: github.com", // Must pin GH_HOST to prevent GHES workflow-level overrides
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
				"bash \"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh\" 1.2.3",
				"name: Custom Install Step",
				"GH_HOST: github.com", // Must pin GH_HOST to prevent GHES workflow-level overrides
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
				"bash \"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh\" " + string(constants.DefaultCopilotVersion),
				"GH_HOST: github.com", // Must pin GH_HOST to prevent GHES workflow-level overrides
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

			// Verify the version is correctly passed to the install script
			expectedVersionLine := "bash \"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh\" " + tt.expectedVersion
			if !strings.Contains(stepContent, expectedVersionLine) {
				t.Errorf("Expected version to be set to '%s', but step content was:\n%s", tt.expectedVersion, stepContent)
			}
		})
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

	// Should contain the custom version
	expectedVersionLine := "bash \"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh\" " + customVersion
	if !strings.Contains(installStep, expectedVersionLine) {
		t.Errorf("Expected custom version %s in install step, got:\n%s", customVersion, installStep)
	}

	// Must pin GH_HOST: github.com to prevent workflow-level GHES overrides from
	// leaking into the Copilot CLI install step. Without this pin, a workflow with
	// env.GH_HOST set to a GHES host would cause the install/auth path to target
	// the wrong host.
	if !strings.Contains(installStep, "GH_HOST: github.com") {
		t.Errorf("Install step should pin GH_HOST: github.com to prevent GHES workflow-level overrides, got:\n%s", installStep)
	}
}

func TestGenerateCopilotInstallerSteps_ExpressionVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		envVar  string
	}{
		{
			name:    "workflow_call input expression",
			version: "${{ inputs.engine-version }}",
			envVar:  "ENGINE_VERSION: ${{ inputs.engine-version }}",
		},
		{
			name:    "github event input expression",
			version: "${{ github.event.inputs.engine-version }}",
			envVar:  "ENGINE_VERSION: ${{ github.event.inputs.engine-version }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := GenerateCopilotInstallerSteps(tt.version, "Install GitHub Copilot CLI")

			if len(steps) != 1 {
				t.Errorf("Expected 1 step, got %d", len(steps))
				return
			}

			stepContent := strings.Join(steps[0], "\n")

			// Should use env var section
			if !strings.Contains(stepContent, "env:") {
				t.Errorf("Expected step to contain 'env:' section for expression version, got:\n%s", stepContent)
			}

			// Should define ENGINE_VERSION env var with the expression
			if !strings.Contains(stepContent, tt.envVar) {
				t.Errorf("Expected step to contain %q, got:\n%s", tt.envVar, stepContent)
			}

			// Should reference ENGINE_VERSION in the run command
			if !strings.Contains(stepContent, `"${ENGINE_VERSION}"`) {
				t.Errorf(`Expected step to use "$ENGINE_VERSION" in run command, got:\n%s`, stepContent)
			}

			// Should NOT embed the expression directly in the shell command
			if strings.Contains(stepContent, "install_copilot_cli.sh "+tt.version) {
				t.Errorf("Expression version should NOT be embedded directly in shell command, got:\n%s", stepContent)
			}
		})
	}
}

func TestCopilotInstallerExpressionVersion_ViaEngineConfig(t *testing.T) {
	// Test that expression version from engine config uses env var injection
	engine := NewCopilotEngine()

	expressionVersion := "${{ inputs.engine-version }}"
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			Version: expressionVersion,
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

	// Should use env var for injection safety
	if !strings.Contains(installStep, "ENGINE_VERSION: ${{ inputs.engine-version }}") {
		t.Errorf("Expected ENGINE_VERSION env var in install step, got:\n%s", installStep)
	}

	// Should reference env var in run command
	if !strings.Contains(installStep, `"${ENGINE_VERSION}"`) {
		t.Errorf(`Expected "$ENGINE_VERSION" in run command, got:\n%s`, installStep)
	}

	// Should NOT embed expression directly in shell command
	if strings.Contains(installStep, "install_copilot_cli.sh "+expressionVersion) {
		t.Errorf("Expression should NOT be embedded directly in shell command, got:\n%s", installStep)
	}
}

func TestCopilotInstallerByokFeatureUsesLatestVersion(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			Version: "1.0.0",
		},
		Features: map[string]any{
			string(constants.ByokCopilotFeatureFlag): true,
		},
	}

	steps := engine.GetInstallationSteps(workflowData)

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

	if !strings.Contains(installStep, `install_copilot_cli.sh" latest`) {
		t.Errorf("Expected byok-copilot to force latest Copilot CLI install, got:\n%s", installStep)
	}
	if strings.Contains(installStep, `install_copilot_cli.sh" 1.0.0`) {
		t.Errorf("Expected byok-copilot to ignore pinned version, got:\n%s", installStep)
	}
}
