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

// TestBuildEvalsJobStepsRenderSummary verifies that the evals job includes the
// "Render evals results to step summary" step and that it runs after the redact step.
func TestBuildEvalsJobStepsRenderSummary(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "copilot",
		Evals: &EvalsConfig{
			Questions: []EvalDefinition{
				{ID: "builds", Question: "Does the code build?"},
			},
		},
	}

	steps := compiler.buildEvalsJobSteps(data)
	if len(steps) == 0 {
		t.Fatal("expected non-empty steps")
	}
	allSteps := strings.Join(steps, "")

	// The render summary step must be present.
	if !strings.Contains(allSteps, "- name: Render evals results to step summary") {
		t.Errorf("expected 'Render evals results to step summary' step in evals job;\ngot:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "id: redact_evals_results") {
		t.Errorf("expected redact step id in evals job;\ngot:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "redact_evals_results.cjs") {
		t.Errorf("expected redact_evals_results.cjs reference in evals job steps;\ngot:\n%s", allSteps)
	}

	// The render summary step must call render_evals_summary.cjs.
	if !strings.Contains(allSteps, "render_evals_summary.cjs") {
		t.Errorf("expected render_evals_summary.cjs reference in evals job steps;\ngot:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "if: steps.redact_evals_results.outcome == 'success'") {
		t.Errorf("expected redact outcome gating for render/upload steps;\ngot:\n%s", allSteps)
	}

	// The render step must appear after the redact step (redact before publish to step summary).
	redactIdx := strings.Index(allSteps, "- name: Redact secrets in evals results")
	renderIdx := strings.Index(allSteps, "- name: Render evals results to step summary")
	uploadIdx := strings.Index(allSteps, "- name: Upload evals results")
	if redactIdx < 0 {
		t.Error("expected 'Redact secrets in evals results' step")
	}
	if renderIdx < 0 {
		t.Error("expected 'Render evals results to step summary' step")
	}
	if uploadIdx < 0 {
		t.Error("expected 'Upload evals results' step")
	}
	if redactIdx >= 0 && renderIdx >= 0 && renderIdx <= redactIdx {
		t.Errorf("expected render step to appear after redact step; redactIdx=%d renderIdx=%d", redactIdx, renderIdx)
	}
	if renderIdx >= 0 && uploadIdx >= 0 && renderIdx >= uploadIdx {
		t.Errorf("expected render step to appear before upload step; renderIdx=%d uploadIdx=%d", renderIdx, uploadIdx)
	}

}

func TestBuildEvalsJobStepsRedactionUsesEvalsSecretReferences(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "copilot",
		Evals: &EvalsConfig{
			Model: "${{ secrets.EVALS_MODEL_SECRET }}",
			Questions: []EvalDefinition{
				{ID: "builds", Question: "Does it use ${{ secrets.EVALS_PROMPT_SECRET }}?"},
			},
		},
	}

	steps := strings.Join(compiler.buildRedactEvalsSecretsStep(data), "")
	if !strings.Contains(steps, "GH_AW_SECRET_NAMES") {
		t.Fatalf("expected GH_AW_SECRET_NAMES in redact step:\n%s", steps)
	}
	if !strings.Contains(steps, "SECRET_EVALS_MODEL_SECRET: ${{ secrets.EVALS_MODEL_SECRET }}") {
		t.Errorf("expected model secret env binding in redact step:\n%s", steps)
	}
	if !strings.Contains(steps, "SECRET_EVALS_PROMPT_SECRET: ${{ secrets.EVALS_PROMPT_SECRET }}") {
		t.Errorf("expected prompt secret env binding in redact step:\n%s", steps)
	}
}

func TestBuildParseEvalsResultsStepUsesResolvedExecutionModel(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "claude",
		EngineConfig: &EngineConfig{
			ID:    "claude",
			Model: "claude-sonnet-4.6",
		},
		Evals: &EvalsConfig{
			Questions: []EvalDefinition{
				{ID: "builds", Question: "Does it build?"},
			},
		},
	}

	steps := strings.Join(compiler.buildParseEvalsResultsStep(data), "")
	if !strings.Contains(steps, `GH_AW_EVALS_MODEL: "claude-sonnet-4.6"`) {
		t.Errorf("expected parse step to record resolved execution model; got:\n%s", steps)
	}
	if strings.Contains(steps, `GH_AW_EVALS_MODEL: "small"`) {
		t.Errorf("expected parse step to avoid default 'small' when engine model is resolved; got:\n%s", steps)
	}
	if !strings.Contains(steps, "GITHUB_RUN_ID: ${{ github.run_id }}") {
		t.Errorf("expected parse step to pass github.run_id through to the eval record writer; got:\n%s", steps)
	}
}

// TestBuildEvalsJobStepsCodexNoDuplicateContainerDownload verifies that a codex
// workflow with evals does not generate a duplicate "Download container images" step.
// Regression: buildEvalsJobSteps previously called buildPullAWFContainersStep
// unconditionally while buildEvalsEngineSteps → generateMCPSetup also emitted the
// step for codex, producing two identical steps and tripping duplicate-step validation.
func TestBuildEvalsJobStepsCodexNoDuplicateContainerDownload(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "codex",
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Type: SandboxTypeAWF,
			},
		},
		Evals: &EvalsConfig{
			Questions: []EvalDefinition{
				{ID: "task-completed", Question: "Did the agent complete the assigned task?"},
			},
		},
	}

	steps := compiler.buildEvalsJobSteps(data)
	allSteps := strings.Join(steps, "")

	count := strings.Count(allSteps, "name: Download container images")
	if count != 1 {
		t.Errorf("expected exactly one 'Download container images' step for codex evals job, got %d\n%s", count, allSteps)
	}
}

// TestBuildEvalsJobStepsNonCodexIncludesContainerDownload verifies that non-codex
// engines still get the "Download container images" step in the evals job.
// Unlike codex, non-codex engines do not call generateMCPSetup in buildEvalsEngineSteps,
// so buildPullAWFContainersStep must still be called for them.
func TestBuildEvalsJobStepsNonCodexIncludesContainerDownload(t *testing.T) {
	compiler := NewCompiler()

	for _, engine := range []string{"copilot", "claude"} {
		t.Run(engine, func(t *testing.T) {
			data := &WorkflowData{
				AI: engine,
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Type: SandboxTypeAWF,
					},
				},
				Evals: &EvalsConfig{
					Questions: []EvalDefinition{
						{ID: "task-completed", Question: "Did the agent complete the assigned task?"},
					},
				},
			}

			steps := compiler.buildEvalsJobSteps(data)
			allSteps := strings.Join(steps, "")

			if !strings.Contains(allSteps, "Download container images") {
				t.Errorf("expected 'Download container images' step in evals job for %s engine\ngot:\n%s", engine, allSteps)
			}
		})
	}
}

// TestBuildRunScriptEvalsStep verifies that script-type evals produce the
// dedicated "Run script evals" step with the correct env vars and JS call.
func TestBuildRunScriptEvalsStep(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "copilot",
		Evals: &EvalsConfig{
			Questions: []EvalDefinition{
				{ID: "s1", Run: "./scripts/check_output.sh"},
				{ID: "s2", Run: "bash -c 'echo YES'"},
			},
		},
	}

	steps := compiler.buildRunScriptEvalsStep(data)
	allSteps := strings.Join(steps, "")

	if !strings.Contains(allSteps, "- name: Run script evals") {
		t.Errorf("expected 'Run script evals' step name; got:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "GH_AW_EVALS_SCRIPT_DEFS:") {
		t.Errorf("expected GH_AW_EVALS_SCRIPT_DEFS env var; got:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "GH_AW_AGENT_OUTPUT:") {
		t.Errorf("expected GH_AW_AGENT_OUTPUT env var; got:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "GH_AW_SAFE_OUTPUT_ITEMS:") {
		t.Errorf("expected GH_AW_SAFE_OUTPUT_ITEMS env var; got:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "GITHUB_RUN_ID:") {
		t.Errorf("expected GITHUB_RUN_ID env var; got:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "run-scripts") {
		t.Errorf("expected GH_AW_EVALS_PHASE: run-scripts; got:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "run_evals.cjs") {
		t.Errorf("expected run_evals.cjs reference; got:\n%s", allSteps)
	}
}

// TestBuildEvalsJobStepsScriptOnly verifies that script-only evals skip the LLM
// engine install/execute/parse steps entirely and only emit the script evals step.
func TestBuildEvalsJobStepsScriptOnly(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "copilot",
		Evals: &EvalsConfig{
			Questions: []EvalDefinition{
				{ID: "s1", Run: "./scripts/check.sh"},
			},
		},
	}

	steps := compiler.buildEvalsJobSteps(data)
	allSteps := strings.Join(steps, "")

	// Script eval step must be present.
	if !strings.Contains(allSteps, "Run script evals") {
		t.Errorf("expected 'Run script evals' step; got:\n%s", allSteps)
	}

	// LLM-specific steps must be absent.
	if strings.Contains(allSteps, "Setup BinEval evaluations") {
		t.Errorf("unexpected 'Setup BinEval evaluations' step for script-only evals; got:\n%s", allSteps)
	}
	if strings.Contains(allSteps, "Parse BinEval results") {
		t.Errorf("unexpected 'Parse BinEval results' step for script-only evals; got:\n%s", allSteps)
	}
}

// TestBuildEvalsJobStepsMixed verifies that when both question and script evals are
// declared, both the LLM path (setup + engine + parse) and the script step are emitted.
func TestBuildEvalsJobStepsMixed(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "copilot",
		Evals: &EvalsConfig{
			Questions: []EvalDefinition{
				{ID: "q1", Question: "Does the code compile?"},
				{ID: "s1", Run: "./scripts/check.sh"},
			},
		},
	}

	steps := compiler.buildEvalsJobSteps(data)
	allSteps := strings.Join(steps, "")

	if !strings.Contains(allSteps, "Setup BinEval evaluations") {
		t.Errorf("expected 'Setup BinEval evaluations' step for mixed evals; got:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "Parse BinEval results") {
		t.Errorf("expected 'Parse BinEval results' step for mixed evals; got:\n%s", allSteps)
	}
	if !strings.Contains(allSteps, "Run script evals") {
		t.Errorf("expected 'Run script evals' step for mixed evals; got:\n%s", allSteps)
	}
}
