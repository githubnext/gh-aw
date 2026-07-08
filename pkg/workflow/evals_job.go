// Package workflow - BinEval evaluation job assembler.
package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var evalsJobLog = logger.New("workflow:evals_job")

// buildEvalsJob creates a separate evals job that runs after the safe_outputs job
// (or directly after the agent job if safe_outputs is not configured).
// The job downloads the agent artifact to access output files, runs a BinEval
// multi-question evaluation via an agentic engine, and uploads evals.jsonl as an artifact.
// Returns nil if evals are not declared in the workflow frontmatter.
func (c *Compiler) buildEvalsJob(data *WorkflowData) (*Job, error) {
	if !data.Evals.HasEvals() {
		evalsJobLog.Print("No evals declared; skipping evals job")
		return nil, nil
	}
	evalsJobLog.Print("Building evals job")

	var steps []string

	// Add setup action steps (installs the agentic engine helper scripts).
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		// For dev mode (local action path), checkout the actions folder first.
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		// Reuse the activation job trace ID so all jobs share one OTLP trace.
		evalsTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		evalsParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, evalsTraceID, evalsParentSpanID)...)
	}

	// Download agent output artifact to access output files (prompt.txt, agent_output.json).
	// Use activation-derived prefix since this job always depends on activation.
	agentArtifactPrefix := artifactPrefixExprForDownstreamJob(data)
	steps = append(steps, buildAgentOutputDownloadSteps(agentArtifactPrefix, c.getActionPin)...)

	// Download experiment artifact so the evals agent can read the current variant assignments.
	steps = append(steps, buildExperimentArtifactDownloadSteps(data, c.getActionPin)...)

	// Add all evals steps: engine install, engine execution, parse, redact, upload.
	steps = append(steps, c.buildEvalsJobSteps(data)...)

	// Determine job dependencies.
	// Evals runs after safe_outputs when it is configured; otherwise directly after agent.
	var needs []string
	if data.SafeOutputs != nil {
		needs = []string{string(constants.SafeOutputsJobName), string(constants.ActivationJobName)}
	} else {
		needs = []string{string(constants.AgentJobName), string(constants.ActivationJobName)}
	}
	evalsJobLog.Printf("Evals job dependencies resolved: needs=%v", needs)

	// Evals job condition: always run but skip if the upstream job was skipped.
	// This matches the detection job pattern so conclusion still sees a non-skipped evals result.
	var upstreamJobName string
	if data.SafeOutputs != nil {
		upstreamJobName = string(constants.SafeOutputsJobName)
	} else {
		upstreamJobName = string(constants.AgentJobName)
	}
	alwaysFunc := BuildFunctionCall("always")
	upstreamNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", upstreamJobName)),
		BuildStringLiteral("skipped"),
	)
	jobConditionNode := BuildAnd(alwaysFunc, upstreamNotSkipped)
	jobCondition := RenderCondition(jobConditionNode)

	// Determine runs-on: use evals override if set, otherwise ubuntu-latest.
	runsOn := "runs-on: ubuntu-latest"
	if data.Evals != nil && data.Evals.RunsOn != "" {
		runsOn = normalizeRunsOnSnippet(data.Evals.RunsOn)
	}

	// Determine permissions for the evals job (same rationale as the detection job).
	copilotRequestsEnabled := hasCopilotRequestsWritePermission(data)
	perms := NewPermissionsContentsRead()
	if copilotRequestsEnabled {
		perms.Set(PermissionCopilotRequests, PermissionWrite)
	}
	if data.EngineConfig != nil && data.EngineConfig.Auth != nil && data.EngineConfig.Auth.Type == "github-oidc" {
		perms.Set(PermissionIdToken, PermissionWrite)
	}
	if hasOTLPGitHubOIDCAuth(data.ParsedFrontmatter, data.RawFrontmatter) {
		perms.Set(PermissionIdToken, PermissionWrite)
	}
	permissions := perms.RenderToYAML()

	job := &Job{
		Name:        string(constants.EvalsJobName),
		Needs:       needs,
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(runsOn, "    "),
		Environment: c.indentYAMLLines(data.Environment, "    "),
		Permissions: permissions,
		Steps:       steps,
	}

	return job, nil
}
