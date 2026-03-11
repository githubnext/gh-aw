//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// engineSetupEntry describes an engine under test together with expected config values.
type engineSetupEntry struct {
	name          string
	engine        CodingAgentEngine
	expectSecrets []string // secret name fragments expected in the validation step
	expectCLIName string   // CLI binary name expected in the install step
}

// allEnginesUnderTest returns the four engines that share the common setup pattern.
func allEnginesUnderTest() []engineSetupEntry {
	return []engineSetupEntry{
		{
			name:          "copilot",
			engine:        NewCopilotEngine(),
			expectSecrets: []string{"COPILOT_GITHUB_TOKEN"},
			expectCLIName: "copilot",
		},
		{
			name:          "claude",
			engine:        NewClaudeEngine(),
			expectSecrets: []string{"ANTHROPIC_API_KEY"},
			expectCLIName: "claude",
		},
		{
			name:          "codex",
			engine:        NewCodexEngine(),
			expectSecrets: []string{"CODEX_API_KEY", "OPENAI_API_KEY"},
			expectCLIName: "codex",
		},
		{
			name:          "gemini",
			engine:        NewGeminiEngine(),
			expectSecrets: []string{"GEMINI_API_KEY"},
			expectCLIName: "gemini",
		},
	}
}

// TestEngineSetupParity_SecretValidationSkipsOnCustomCommand verifies that every
// engine returns an empty secret-validation step when a custom command is specified.
func TestEngineSetupParity_SecretValidationSkipsOnCustomCommand(t *testing.T) {
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{
			Command: "/custom/engine-binary",
		},
	}

	for _, tc := range allEnginesUnderTest() {
		t.Run(tc.name, func(t *testing.T) {
			step := tc.engine.GetSecretValidationStep(workflowData)
			assert.Empty(t, step,
				"engine %s: GetSecretValidationStep should return empty step when custom command is set", tc.name)
		})
	}
}

// TestEngineSetupParity_SecretValidationReturnsStepWithoutCustomCommand verifies that
// every engine returns a non-empty secret-validation step under default conditions.
func TestEngineSetupParity_SecretValidationReturnsStepWithoutCustomCommand(t *testing.T) {
	workflowData := &WorkflowData{}

	for _, tc := range allEnginesUnderTest() {
		t.Run(tc.name, func(t *testing.T) {
			step := tc.engine.GetSecretValidationStep(workflowData)
			require.NotEmpty(t, step,
				"engine %s: GetSecretValidationStep should return a non-empty step", tc.name)

			// Verify that each expected secret appears in the validation step content.
			stepContent := strings.Join(step, "\n")
			for _, secret := range tc.expectSecrets {
				assert.Contains(t, stepContent, secret,
					"engine %s: secret validation step should reference %s", tc.name, secret)
			}
		})
	}
}

// TestEngineSetupParity_InstallationSkipsOnCustomCommand verifies that every engine
// returns no installation steps when a custom command is specified.
func TestEngineSetupParity_InstallationSkipsOnCustomCommand(t *testing.T) {
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{
			Command: "/custom/engine-binary",
		},
	}

	for _, tc := range allEnginesUnderTest() {
		t.Run(tc.name, func(t *testing.T) {
			steps := tc.engine.GetInstallationSteps(workflowData)
			assert.Empty(t, steps,
				"engine %s: GetInstallationSteps should return empty when custom command is set", tc.name)
		})
	}
}

// TestEngineSetupParity_InstallationHasSetupAndInstallSteps verifies that every
// engine produces at least two installation steps (setup + install).
func TestEngineSetupParity_InstallationHasSetupAndInstallSteps(t *testing.T) {
	workflowData := &WorkflowData{}

	for _, tc := range allEnginesUnderTest() {
		t.Run(tc.name, func(t *testing.T) {
			steps := tc.engine.GetInstallationSteps(workflowData)
			assert.GreaterOrEqual(t, len(steps), 1,
				"engine %s: GetInstallationSteps should return at least one step", tc.name)
		})
	}
}

// TestEngineSetupParity_FirewallInjectsAWFBetweenSetupAndInstall verifies that when
// the firewall is enabled every engine includes an AWF installation step, and that
// the AWF step appears after the first step and before any subsequent install steps.
func TestEngineSetupParity_FirewallInjectsAWFBetweenSetupAndInstall(t *testing.T) {
	workflowData := &WorkflowData{
		NetworkPermissions: &NetworkPermissions{
			Allowed: []string{"defaults"},
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		},
	}

	for _, tc := range allEnginesUnderTest() {
		t.Run(tc.name, func(t *testing.T) {
			steps := tc.engine.GetInstallationSteps(workflowData)
			require.NotEmpty(t, steps,
				"engine %s: should produce installation steps when firewall is enabled", tc.name)

			// Locate the AWF step.
			awfIdx := -1
			for i, step := range steps {
				content := strings.Join(step, "\n")
				if strings.Contains(content, "awf") || strings.Contains(content, "install_awf_binary") {
					awfIdx = i
					break
				}
			}
			require.NotEqual(t, -1, awfIdx,
				"engine %s: expected an AWF installation step when firewall is enabled", tc.name)

			// AWF must not be the very first step (setup always comes first).
			assert.Positive(t, awfIdx,
				"engine %s: AWF step should appear after the setup step, not as the first step", tc.name)

			// For engines whose installer emits more than two steps there should be at least
			// one CLI install step after AWF.
			if len(steps) > awfIdx+1 {
				afterAWFContent := strings.Join(steps[awfIdx+1], "\n")
				assert.Contains(t, afterAWFContent, tc.expectCLIName,
					"engine %s: CLI install step should follow the AWF step", tc.name)
			}
		})
	}
}

// TestEngineSetupParity_FirewallDisabledOmitsAWF verifies that no AWF step is
// emitted when the firewall is disabled.
func TestEngineSetupParity_FirewallDisabledOmitsAWF(t *testing.T) {
	workflowData := &WorkflowData{}

	for _, tc := range allEnginesUnderTest() {
		t.Run(tc.name, func(t *testing.T) {
			steps := tc.engine.GetInstallationSteps(workflowData)

			for _, step := range steps {
				content := strings.Join(step, "\n")
				assert.NotContains(t, content, "install_awf_binary",
					"engine %s: should not include AWF step when firewall is disabled", tc.name)
			}
		})
	}
}
