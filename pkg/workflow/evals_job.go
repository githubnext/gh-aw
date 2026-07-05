// Package workflow - BinEval evaluation job assembler.
package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var evalsJobLog = logger.New("workflow:evals_job")

// buildEvalsJob creates the evals job that runs BinEval-style evaluations after
// safe-outputs completes. The job downloads the agent artifact, runs each declared
// question through a small LLM, and stores results as a JSONL artifact and in a
// git branch for historical comparison.
//
// Returns nil when no evals are configured.
func (c *Compiler) buildEvalsJob(data *WorkflowData) (*Job, error) {
	if !data.Evals.HasEvals() {
		evalsJobLog.Print("No evals configured, skipping evals job")
		return nil, nil
	}

	evalsJobLog.Printf("Building evals job with %d questions", len(data.Evals.Questions))

	var steps []string

	// Setup action (installs the agentic engine helpers and CJS scripts).
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		traceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		parentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, traceID, parentSpanID)...)
	}

	// Download the agent output artifact so the harness can access prompt.txt and
	// agent_output.json.
	agentArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	steps = append(steps, buildAgentOutputDownloadSteps(agentArtifactPrefix, c.getActionPin)...)

	// Build the evaluation harness steps (prepare files, run questions, store results).
	steps = append(steps, c.buildEvalsJobSteps(data)...)

	// Upload the eval artifact (JSONL file).
	steps = append(steps, c.buildEvalsArtifactUploadStep(data, agentArtifactPrefix)...)

	// Persist results to git branch evals/<workflow-id>.
	steps = append(steps, c.buildEvalsBranchPersistStep(data)...)

	// Determine which jobs this job depends on.
	// Always depends on agent and activation; also depends on safe_outputs when it is configured.
	needs := []string{string(constants.AgentJobName), string(constants.ActivationJobName)}
	if data.SafeOutputs != nil {
		needs = append(needs, string(constants.SafeOutputsJobName))
	}
	// Deduplicate needs.
	needs = deduplicateStrings(needs)

	// Job condition: run when the agent job completed (not skipped) and is not cancelled.
	alwaysFunc := BuildFunctionCall("always")
	notCancelled := &NotNode{Child: BuildFunctionCall("cancelled")}
	agentNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)
	jobCondition := RenderCondition(BuildAnd(BuildAnd(alwaysFunc, notCancelled), agentNotSkipped))

	// Runner: use evals override if set, otherwise default ubuntu-latest.
	runsOn := "runs-on: ubuntu-latest"
	if data.Evals.RunsOn != "" {
		runsOn = normalizeRunsOnSnippet(data.Evals.RunsOn)
	}

	// Permissions: contents:write for git branch persistence + actions:read for artifacts.
	perms := NewPermissionsContentsRead()
	perms.Set(PermissionContents, PermissionWrite)
	permissions := perms.RenderToYAML()

	job := &Job{
		Name:        string(constants.EvalsJobName),
		Needs:       needs,
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(runsOn, "    "),
		Permissions: permissions,
		Steps:       steps,
	}

	evalsJobLog.Printf("Built evals job with %d steps, depends on: %v", len(steps), needs)
	return job, nil
}

// deduplicateStrings returns a slice with duplicate strings removed while preserving order.
func deduplicateStrings(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
