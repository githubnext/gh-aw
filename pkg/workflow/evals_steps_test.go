//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

// TestBuildEvalsEngineStepsArcDindTopology verifies that the evals job
// correctly propagates arc-dind runner topology from the main workflow data.
// Regression: before the fix, RunnerConfig was not propagated to evalsData,
// so isArcDindTopology(evalsData) was always false — the Copilot staging step
// was never emitted and the engine was spawned as /usr/local/bin/copilot (ENOENT inside
// the AWF chroot which uses the dind daemon's filesystem).
func TestBuildEvalsEngineStepsArcDindTopology(t *testing.T) {
	compiler := NewCompiler()

	t.Run("arc-dind: emits daemon-visible staging step and uses RUNNER_TEMP copilot path", func(t *testing.T) {
		data := &WorkflowData{
			AI: "copilot",
			RunnerConfig: &RunnerConfig{
				Topology: RunnerTopologyArcDind,
			},
			Evals: &EvalsConfig{
				Questions: []EvalDefinition{
					{ID: "test", Question: "Does the code work?"},
				},
			},
		}

		steps := compiler.buildEvalsEngineSteps(data)
		if len(steps) == 0 {
			t.Fatal("expected non-empty steps")
		}
		allSteps := strings.Join(steps, "")

		// The staging step copies the Copilot CLI to a daemon-visible path.
		if !strings.Contains(allSteps, "Copy Copilot CLI to daemon-visible path") {
			t.Errorf("expected 'Copy Copilot CLI to daemon-visible path' step in evals job for arc-dind;\ngot:\n%s", allSteps)
		}

		// The copilot_harness.cjs invocation must use the daemon-visible path specifically.
		// Note: constants.GhAwRootDirShell+"/bin/copilot" also appears in the staging step's
		// copy command ("cp /usr/local/bin/copilot ..."), so checking the harness line
		// directly avoids a false positive from the staging step.
		harnessArcDindPath := "copilot_harness.cjs " + constants.GhAwRootDirShell + "/bin/copilot"
		if !strings.Contains(allSteps, harnessArcDindPath) {
			t.Errorf("expected copilot_harness.cjs to be invoked with daemon-visible path %q for arc-dind;\ngot:\n%s", harnessArcDindPath, allSteps)
		}
		if strings.Contains(allSteps, "copilot_harness.cjs "+constants.CopilotBinaryPath) {
			t.Errorf("copilot_harness.cjs must NOT be invoked with %q for arc-dind (ENOENT inside chroot);\ngot:\n%s", constants.CopilotBinaryPath, allSteps)
		}
	})

	t.Run("non-arc-dind: no staging step and uses /usr/local/bin/copilot", func(t *testing.T) {
		data := &WorkflowData{
			AI: "copilot",
			// RunnerConfig is nil → default topology
			Evals: &EvalsConfig{
				Questions: []EvalDefinition{
					{ID: "test", Question: "Does the code work?"},
				},
			},
		}

		steps := compiler.buildEvalsEngineSteps(data)
		if len(steps) == 0 {
			t.Fatal("expected non-empty steps")
		}
		allSteps := strings.Join(steps, "")

		// No daemon-visible staging step for standard runners.
		if strings.Contains(allSteps, "Copy Copilot CLI to daemon-visible path") {
			t.Errorf("unexpected 'Copy Copilot CLI to daemon-visible path' step for non-arc-dind evals job;\ngot:\n%s", allSteps)
		}

		// Standard runners use the installed binary directly via the harness.
		if !strings.Contains(allSteps, "copilot_harness.cjs "+constants.CopilotBinaryPath) {
			t.Errorf("expected evals execution to use copilot_harness.cjs with %q for non-arc-dind;\ngot:\n%s", constants.CopilotBinaryPath, allSteps)
		}
	})
}
