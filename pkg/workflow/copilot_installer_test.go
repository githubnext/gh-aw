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
			steps := GenerateCopilotInstallerSteps(tt.version, tt.stepName, false)

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

func TestCopilotEngineWithVersion(t *testing.T) {
	// engine.version must be honored: when an explicit version is set it should be
	// passed to the installer and compat.json resolution must be skipped.
	engine := NewCopilotEngine()

	customVersion := "1.0.0"
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			Version: customVersion,
		},
	}

	steps := engine.GetInstallationSteps(workflowData)

	// EngineConfig.Version must remain as the user-specified value.
	if workflowData.EngineConfig.Version != customVersion {
		t.Fatalf("Expected engine config version to remain %q, got: %q", customVersion, workflowData.EngineConfig.Version)
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

	// Should pass the user-specified version to the installer (compat.json skipped).
	if !strings.Contains(installStep, `install_copilot_cli.sh" `+customVersion) {
		t.Errorf("Expected user-specified version %q in install step, got:\n%s", customVersion, installStep)
	}
	if strings.Contains(installStep, `install_copilot_cli.sh" `+string(constants.DefaultCopilotVersion)) {
		t.Errorf("Expected user-specified version, not default version, in install step:\n%s", installStep)
	}

	// Must pin GH_HOST: github.com to prevent workflow-level GHES overrides from
	// leaking into the Copilot CLI install step.
	if !strings.Contains(installStep, "GH_HOST: github.com") {
		t.Errorf("Install step should pin GH_HOST: github.com to prevent GHES workflow-level overrides, got:\n%s", installStep)
	}
}

func TestCopilotEngineWithoutVersion(t *testing.T) {
	// When engine.version is not set, the default pinned version must be used and
	// EngineConfig.Version must be normalized to the effective installed value.
	engine := NewCopilotEngine()

	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{},
	}

	steps := engine.GetInstallationSteps(workflowData)

	// EngineConfig.Version must be normalized to the default version.
	if workflowData.EngineConfig.Version != string(constants.DefaultCopilotVersion) {
		t.Fatalf("Expected engine config version to be normalized to default Copilot version %q, got: %q", constants.DefaultCopilotVersion, workflowData.EngineConfig.Version)
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

	// Should use the default pinned version.
	if !strings.Contains(installStep, `install_copilot_cli.sh" `+string(constants.DefaultCopilotVersion)) {
		t.Errorf("Expected default Copilot version in install step, got:\n%s", installStep)
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
			steps := GenerateCopilotInstallerSteps(tt.version, "Install GitHub Copilot CLI", false)

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

func TestGenerateCopilotInstallerSteps_Rootless(t *testing.T) {
	steps := GenerateCopilotInstallerSteps("1.2.3", "Install Copilot CLI", true)

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	stepContent := strings.Join(steps[0], "\n")

	if !strings.Contains(stepContent, "--rootless") {
		t.Errorf("Expected step to contain --rootless flag, got:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "install_copilot_cli.sh") {
		t.Errorf("Expected step to use install_copilot_cli.sh, got:\n%s", stepContent)
	}
}

func TestGenerateCopilotInstallerSteps_RootlessWithExpression(t *testing.T) {
	steps := GenerateCopilotInstallerSteps("${{ inputs.version }}", "Install Copilot CLI", true)

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	stepContent := strings.Join(steps[0], "\n")

	if !strings.Contains(stepContent, "--rootless") {
		t.Errorf("Expected step to contain --rootless flag, got:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, `"${ENGINE_VERSION}"`) {
		t.Errorf("Expected step to use ENGINE_VERSION env var, got:\n%s", stepContent)
	}
}

func TestCopilotEngineWithExpressionVersion(t *testing.T) {
	// expression engine.version must be honored: the value must flow through env-var
	// injection (not embedded directly in the shell command) and compat.json must be skipped.
	engine := NewCopilotEngine()

	expressionVersion := "${{ inputs.engine-version }}"
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			Version: expressionVersion,
		},
	}

	steps := engine.GetInstallationSteps(workflowData)

	// EngineConfig.Version must remain as the expression value.
	if workflowData.EngineConfig.Version != expressionVersion {
		t.Fatalf("Expected engine config version to remain %q, got: %q", expressionVersion, workflowData.EngineConfig.Version)
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

	// Should use env var injection (not embed expression directly in shell command).
	if !strings.Contains(installStep, "ENGINE_VERSION: "+expressionVersion) {
		t.Errorf("Expected ENGINE_VERSION env var with expression, got:\n%s", installStep)
	}
	if !strings.Contains(installStep, `"${ENGINE_VERSION}"`) {
		t.Errorf(`Expected step to reference "$ENGINE_VERSION" in run command, got:\n%s`, installStep)
	}
	if strings.Contains(installStep, "install_copilot_cli.sh "+expressionVersion) {
		t.Errorf("Expression version should NOT be embedded directly in shell command, got:\n%s", installStep)
	}
}

func TestCopilotEngineWithVersionAndByokFeature(t *testing.T) {
	// engine.version must be honored even when the BYOK feature flag is enabled.
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

	if !strings.Contains(installStep, `install_copilot_cli.sh" 1.0.0`) {
		t.Errorf("Expected user-specified version in install step, got:\n%s", installStep)
	}
	if strings.Contains(installStep, `install_copilot_cli.sh" `+string(constants.DefaultCopilotVersion)) {
		t.Errorf("Expected user-specified version, not default version, in install step:\n%s", installStep)
	}
}
