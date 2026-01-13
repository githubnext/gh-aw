package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssignToAgentUsesCopilotToken verifies that assign-to-agent uses the
// correct Copilot token precedence chain instead of the agent token chain.
// This is critical because GITHUB_TOKEN does not have permissions to assign bot agents.
func TestAssignToAgentUsesCopilotToken(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "Test Assign To Agent",
		SafeOutputs: &SafeOutputsConfig{
			AssignToAgent: &AssignToAgentConfig{
				DefaultAgent: "copilot",
			},
		},
	}

	// Build the assign-to-agent step configuration
	config := compiler.buildAssignToAgentStepConfig(workflowData, "agent", false)

	// Verify UseCopilotToken is true
	assert.True(t, config.UseCopilotToken, "AssignToAgent should use UseCopilotToken")
	assert.False(t, config.UseAgentToken, "AssignToAgent should not use UseAgentToken")
}

// TestAssignToAgentTokenPrecedence verifies the actual token precedence
// in the compiled workflow YAML
func TestAssignToAgentTokenPrecedence(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "Test Assign To Agent",
		SafeOutputs: &SafeOutputsConfig{
			AssignToAgent: &AssignToAgentConfig{
				DefaultAgent: "copilot",
			},
		},
	}

	// Build the step using buildConsolidatedSafeOutputStep
	config := compiler.buildAssignToAgentStepConfig(workflowData, "agent", false)
	steps := compiler.buildConsolidatedSafeOutputStep(workflowData, config)
	stepsContent := strings.Join(steps, "")

	t.Logf("Generated steps:\n%s", stepsContent)

	// Should use COPILOT_GITHUB_TOKEN in the fallback chain
	assert.Contains(t, stepsContent, "COPILOT_GITHUB_TOKEN",
		"Should use COPILOT_GITHUB_TOKEN in token fallback")

	// Should NOT include GITHUB_TOKEN in the fallback
	// The Copilot chain is: COPILOT_GITHUB_TOKEN || GH_AW_GITHUB_TOKEN
	// The Agent chain would be: GH_AW_AGENT_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
	assert.NotContains(t, stepsContent, "GH_AW_AGENT_TOKEN",
		"Should not use GH_AW_AGENT_TOKEN (agent token chain)")
	assert.NotContains(t, stepsContent, "|| secrets.GITHUB_TOKEN",
		"Should not fall back to GITHUB_TOKEN (lacks bot assignment permissions)")

	// Verify the complete correct precedence chain
	assert.Contains(t, stepsContent, "secrets.COPILOT_GITHUB_TOKEN || secrets.GH_AW_GITHUB_TOKEN",
		"Should use correct Copilot token precedence: COPILOT_GITHUB_TOKEN || GH_AW_GITHUB_TOKEN")
}

// TestAssignToAgentDefaultAgentEnvVar verifies that the default agent
// environment variable is set when configured
func TestAssignToAgentDefaultAgentEnvVar(t *testing.T) {
	tests := []struct {
		name         string
		defaultAgent string
		shouldHaveEnv bool
	}{
		{
			name:         "with default agent",
			defaultAgent: "copilot",
			shouldHaveEnv: true,
		},
		{
			name:         "without default agent",
			defaultAgent: "",
			shouldHaveEnv: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			workflowData := &WorkflowData{
				Name: "Test Assign To Agent",
				SafeOutputs: &SafeOutputsConfig{
					AssignToAgent: &AssignToAgentConfig{
						DefaultAgent: tt.defaultAgent,
					},
				},
			}

			config := compiler.buildAssignToAgentStepConfig(workflowData, "agent", false)
			steps := compiler.buildConsolidatedSafeOutputStep(workflowData, config)
			stepsContent := strings.Join(steps, "")

			if tt.shouldHaveEnv {
				assert.Contains(t, stepsContent, "GH_AW_AGENT_DEFAULT",
					"Should include GH_AW_AGENT_DEFAULT environment variable")
				assert.Contains(t, stepsContent, `"copilot"`,
					"Should set correct default agent value")
			} else {
				assert.NotContains(t, stepsContent, "GH_AW_AGENT_DEFAULT",
					"Should not include GH_AW_AGENT_DEFAULT when not configured")
			}
		})
	}
}

// TestAssignToAgentMaxCountEnvVar verifies that the max count environment
// variable is set correctly
func TestAssignToAgentMaxCountEnvVar(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "Test Assign To Agent",
		SafeOutputs: &SafeOutputsConfig{
			AssignToAgent: &AssignToAgentConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 5,
				},
			},
		},
	}

	config := compiler.buildAssignToAgentStepConfig(workflowData, "agent", false)
	steps := compiler.buildConsolidatedSafeOutputStep(workflowData, config)
	stepsContent := strings.Join(steps, "")

	assert.Contains(t, stepsContent, "GH_AW_AGENT_MAX_COUNT: 5",
		"Should include GH_AW_AGENT_MAX_COUNT environment variable with correct value")
}

// TestAssignToAgentWithCustomToken verifies that a custom token takes precedence
func TestAssignToAgentWithCustomToken(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	customToken := "${{ secrets.CUSTOM_AGENT_TOKEN }}"
	workflowData := &WorkflowData{
		Name: "Test Assign To Agent",
		SafeOutputs: &SafeOutputsConfig{
			AssignToAgent: &AssignToAgentConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					GitHubToken: customToken,
				},
			},
		},
	}

	config := compiler.buildAssignToAgentStepConfig(workflowData, "agent", false)
	require.Equal(t, customToken, config.Token, "Config should store custom token")

	steps := compiler.buildConsolidatedSafeOutputStep(workflowData, config)
	stepsContent := strings.Join(steps, "")

	// Custom token should be used
	assert.Contains(t, stepsContent, "CUSTOM_AGENT_TOKEN",
		"Should use custom token when provided")
}
