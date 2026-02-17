//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomActionCopilotTokenFallback tests that custom actions use the correct
// Copilot token fallback when no custom token is provided
func TestCustomActionCopilotTokenFallback(t *testing.T) {
	compiler := NewCompiler()

	// Register a test custom action
	testScript := `console.log('test');`
	actionPath := "./actions/test-action"
	err := DefaultScriptRegistry.RegisterWithAction("test_handler", testScript, RuntimeModeGitHubScript, actionPath)
	require.NoError(t, err)

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	// Test with UseCopilotRequestsToken=true and no custom token
	config := GitHubScriptStepConfig{
		StepName:                "Test Custom Action",
		StepID:                  "test",
		CustomToken:             "", // No custom token
		UseCopilotRequestsToken: true,
	}

	steps := compiler.buildCustomActionStep(workflowData, config, "test_handler")
	stepsContent := strings.Join(steps, "")

	t.Logf("Generated steps:\n%s", stepsContent)

	// Should use COPILOT_GITHUB_TOKEN directly (no fallback chain)
	// Note: COPILOT_GITHUB_TOKEN is the recommended token for Copilot operations
	// and does NOT have a fallback to GITHUB_TOKEN because GITHUB_TOKEN lacks
	// permissions for agent sessions and bot assignments
	assert.Contains(t, stepsContent, "secrets.COPILOT_GITHUB_TOKEN", "Should use COPILOT_GITHUB_TOKEN")
	assert.NotContains(t, stepsContent, "COPILOT_TOKEN ||", "Should not use deprecated COPILOT_TOKEN")

	// Verify no fallback chain (COPILOT_GITHUB_TOKEN is used directly)
	assert.NotContains(t, stepsContent, "||", "Should not have fallback chain for Copilot token")
}
