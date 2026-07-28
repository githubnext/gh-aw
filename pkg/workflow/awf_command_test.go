//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainAgentRunUsesStandardCreditsExpressionNotDetectionExpression verifies that
// a standard (non-detection) main-agent run emits the main-agent credits expression
// (vars.GH_AW_DEFAULT_MAX_AI_CREDITS) and not the detection-specific one, so a future
// refactor that accidentally sets IsDetectionRun on main-agent data will be caught.
func TestMainAgentRunUsesStandardCreditsExpressionNotDetectionExpression(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "claude",
			// MaxAICredits is zero (not set in frontmatter) to trigger runtime expression injection.
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		// IsDetectionRun is false by default — this is a main-agent run.
	}

	engine := NewClaudeEngine()
	steps := engine.GetExecutionSteps(workflowData, "test.log")
	require.NotEmpty(t, steps, "should produce execution steps")

	stepContent := strings.Join(steps[0], "\n")

	assert.Contains(t, stepContent, "vars.GH_AW_DEFAULT_MAX_AI_CREDITS",
		"main-agent run should use standard credits expression")
	assert.NotContains(t, stepContent, "vars.GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS",
		"main-agent run must not use detection credits expression")
}

func TestBuildAWFCommand_ServicePortsRequireLegacy(t *testing.T) {
	t.Run("emits --allow-host-service-ports when legacy-security is enabled", func(t *testing.T) {
		config := AWFCommandConfig{
			WorkflowData: &WorkflowData{
				Name:                   "test-workflow",
				EngineConfig:           &EngineConfig{ID: "copilot"},
				ServicePortExpressions: "${{ job.services.db.ports['5432'] }}",
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID:             "awf",
						LegacySecurity: true,
					},
				},
			},
			EngineName:    "copilot",
			EngineCommand: "copilot-agent",
		}
		cmd := BuildAWFCommand(config)
		assert.Contains(t, cmd, "--allow-host-service-ports", "Should emit --allow-host-service-ports in legacy mode")
	})

	t.Run("skips --allow-host-service-ports in strict mode", func(t *testing.T) {
		config := AWFCommandConfig{
			WorkflowData: &WorkflowData{
				Name:                   "test-workflow",
				EngineConfig:           &EngineConfig{ID: "copilot"},
				ServicePortExpressions: "${{ job.services.db.ports['5432'] }}",
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID: "awf",
					},
				},
			},
			EngineName:    "copilot",
			EngineCommand: "copilot-agent",
		}
		cmd := BuildAWFCommand(config)
		assert.NotContains(t, cmd, "--allow-host-service-ports", "Should NOT emit --allow-host-service-ports in strict mode")
	})
}
