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
				"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh 0.0.369",
				"name: Install GitHub Copilot CLI",
			},
			shouldNotContain: []string{
				"gh.io/copilot-install | sudo bash", // Should not pipe directly to bash
				"GH_HOST: github.com",               // Should not hardcode GH_HOST (breaks GHEC)
			},
		},
		{
			name:            "version with v prefix",
			version:         "v0.0.370",
			stepName:        "Install GitHub Copilot CLI",
			expectedVersion: "v0.0.370",
			shouldContain: []string{
				"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh v0.0.370",
			},
			shouldNotContain: []string{
				"gh.io/copilot-install | sudo bash",
				"GH_HOST: github.com", // Should not hardcode GH_HOST (breaks GHEC)
			},
		},
		{
			name:            "custom version",
			version:         "1.2.3",
			stepName:        "Custom Install Step",
			expectedVersion: "1.2.3",
			shouldContain: []string{
				"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh 1.2.3",
				"name: Custom Install Step",
			},
			shouldNotContain: []string{
				"gh.io/copilot-install | sudo bash",
				"GH_HOST: github.com", // Should not hardcode GH_HOST (breaks GHEC)
			},
		},
		{
			name:            "empty version uses default",
			version:         "",
			stepName:        "Install GitHub Copilot CLI",
			expectedVersion: string(constants.DefaultCopilotVersion), // Should use DefaultCopilotVersion
			shouldContain: []string{
				"${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh " + string(constants.DefaultCopilotVersion),
			},
			shouldNotContain: []string{
				"gh.io/copilot-install | sudo bash",
				"GH_HOST: github.com", // Should not hardcode GH_HOST (breaks GHEC)
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
			expectedVersionLine := "${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh " + tt.expectedVersion
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
	expectedVersionLine := "${RUNNER_TEMP}/gh-aw/actions/install_copilot_cli.sh " + customVersion
	if !strings.Contains(installStep, expectedVersionLine) {
		t.Errorf("Expected custom version %s in install step, got:\n%s", customVersion, installStep)
	}

	// Should NOT hardcode GH_HOST: github.com — hardcoding it breaks GHEC.
	// The install script uses curl with hardcoded URLs and does not use gh CLI,
	// so GH_HOST is irrelevant to the download. The correct host is already set
	// in GITHUB_ENV by configure_gh_for_ghe.sh (or by the Derive GH_HOST step).
	if strings.Contains(installStep, "GH_HOST: github.com") {
		t.Errorf("Install step should NOT hardcode GH_HOST: github.com (breaks GHEC), got:\n%s", installStep)
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
