//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newHarnessEngineDefinition returns a minimal EngineDefinition with harness-script set.
func newHarnessEngineDefinition() *EngineDefinition {
	return &EngineDefinition{
		ID:          "testharness",
		DisplayName: "TestHarness",
		Description: "A test engine with a harness script",
		Behaviors: &EngineBehaviorDefinition{
			Execution: &EngineExecutionDefinition{
				CommandName: "testharness-cli",
				Args:        []string{"run"},
				StepName:    "Execute TestHarness CLI",
			},
			HarnessScript: `"use strict";
// Minimal test harness
const cmd = process.argv[2];
const { spawnSync } = require("child_process");
spawnSync(cmd, process.argv.slice(3), { stdio: "inherit" });
`,
		},
	}
}

// TestBehaviorDefinedEngineHarnessScript verifies that harness-script is wired correctly
// into the engine's execution steps.
func TestBehaviorDefinedEngineHarnessScript(t *testing.T) {
	def := newHarnessEngineDefinition()
	engine, err := NewBehaviorDefinedEngine(def)
	require.NoError(t, err)

	t.Run("harness_write_step_included", func(t *testing.T) {
		workflowData := &WorkflowData{Name: "test"}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		// Steps: [harness write, execution]
		require.Len(t, steps, 2, "should generate harness-write step and execution step")

		harnessStepContent := strings.Join(steps[0], "\n")
		assert.Contains(t, harnessStepContent, "Write TestHarness harness script", "step name should include engine display name")
		assert.Contains(t, harnessStepContent, "testharness_harness.cjs", "step should write the correct harness filename")
		assert.Contains(t, harnessStepContent, "gh-aw/actions", "step should write to the setup action destination directory")
		assert.Contains(t, harnessStepContent, harnessScriptHeredocDelimiter, "step should use heredoc delimiter")
		assert.Contains(t, harnessStepContent, "use strict", "step should embed harness script content")
	})

	t.Run("execution_step_uses_node_harness", func(t *testing.T) {
		workflowData := &WorkflowData{Name: "test"}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.Len(t, steps, 2, "should generate harness-write step and execution step")

		execStepContent := strings.Join(steps[1], "\n")
		assert.Contains(t, execStepContent, "id: agentic_execution", "execution step should have agentic_execution ID")
		assert.Contains(t, execStepContent, "GH_AW_NODE_EXEC", "execution should use node runtime resolution command")
		assert.Contains(t, execStepContent, "testharness_harness.cjs", "execution should invoke the harness file")
		assert.Contains(t, execStepContent, "testharness-cli", "execution should pass command name to harness")
		// When harness is set, inline prompt substitution must NOT appear
		assert.NotContains(t, execStepContent, `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`, "harness execution must not include inline prompt substitution")
	})

	t.Run("awf_reflect_enabled_when_firewall_on", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test",
			NetworkPermissions: &NetworkPermissions{
				Allowed:  []string{"defaults"},
				Firewall: &FirewallConfig{Enabled: true},
			},
		}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.GreaterOrEqual(t, len(steps), 2, "should have at least harness-write and execution steps")

		// AWF_REFLECT_ENABLED must be set when firewall is on and harness-script is present
		execStepContent := strings.Join(steps[len(steps)-1], "\n")
		assert.Contains(t, execStepContent, "AWF_REFLECT_ENABLED: 1", "AWF_REFLECT_ENABLED must be set when harness-script and firewall are both active")
	})

	t.Run("awf_forced_and_reflect_enabled_without_explicit_firewall", func(t *testing.T) {
		// harness-script always forces AWF so the harness can read /reflect from the API proxy.
		// AWF_REFLECT_ENABLED must therefore be present even when no explicit firewall is configured.
		workflowData := &WorkflowData{Name: "test"}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.GreaterOrEqual(t, len(steps), 2, "should have at least harness-write and execution steps")

		execStepContent := strings.Join(steps[len(steps)-1], "\n")
		assert.Contains(t, execStepContent, "AWF_REFLECT_ENABLED: 1", "AWF_REFLECT_ENABLED must be set when harness-script forces AWF execution")
		assert.Contains(t, execStepContent, "sudo", "execution step must use AWF when harness-script is present")
	})

	t.Run("env_vars_still_set", func(t *testing.T) {
		workflowData := &WorkflowData{Name: "test"}
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		require.GreaterOrEqual(t, len(steps), 2)

		execStepContent := strings.Join(steps[len(steps)-1], "\n")
		assert.Contains(t, execStepContent, "GH_AW_PROMPT:", "GH_AW_PROMPT must be set for harness to read prompt file")
		assert.Contains(t, execStepContent, "RUNNER_TEMP:", "RUNNER_TEMP must be set")
	})

	t.Run("harness_filename", func(t *testing.T) {
		assert.Equal(t, "testharness_harness.cjs", engine.harnessScriptFilename())
	})

	t.Run("heredoc_delimiter_collision_skips_harness_write_step", func(t *testing.T) {
		// An engine whose harness-script contains the heredoc delimiter at line start must
		// not generate a harness write step (to avoid premature heredoc termination).
		collisionDef := &EngineDefinition{
			ID:          "collision",
			DisplayName: "Collision",
			Behaviors: &EngineBehaviorDefinition{
				Execution: &EngineExecutionDefinition{
					CommandName: "collision-cli",
					StepName:    "Execute",
				},
				HarnessScript: "// legit JS\n" + harnessScriptHeredocDelimiter + "\nconsole.log('hi');",
			},
		}
		eng, err := NewBehaviorDefinedEngine(collisionDef)
		require.NoError(t, err)
		harnessStep := eng.buildHarnessWriteStep()
		assert.Nil(t, harnessStep, "harness write step must be skipped when script contains the heredoc delimiter")
	})
}

// TestBehaviorDefinedEngineNoHarnessScript verifies that engines without harness-script
// continue to use the direct command execution path (inline prompt substitution).
func TestBehaviorDefinedEngineNoHarnessScript(t *testing.T) {
	def := &EngineDefinition{
		ID:          "noharness",
		DisplayName: "NoHarness",
		Behaviors: &EngineBehaviorDefinition{
			Execution: &EngineExecutionDefinition{
				CommandName: "noharness-cli",
				Args:        []string{"run"},
				StepName:    "Execute NoHarness CLI",
			},
		},
	}
	engine, err := NewBehaviorDefinedEngine(def)
	require.NoError(t, err)

	workflowData := &WorkflowData{Name: "test"}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// No config-file, no harness: only the execution step
	require.Len(t, steps, 1, "should generate only execution step when no harness-script and no config-file")

	execStepContent := strings.Join(steps[0], "\n")
	assert.Contains(t, execStepContent, "noharness-cli run", "should invoke the command directly")
	assert.Contains(t, execStepContent, `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`, "direct execution must include inline prompt substitution")
	assert.NotContains(t, execStepContent, "GH_AW_NODE_EXEC", "should not use node harness when harness-script is absent")
	assert.NotContains(t, execStepContent, "GHAW_HARNESS_SCRIPT_EOF", "no harness write step should be present")
}
