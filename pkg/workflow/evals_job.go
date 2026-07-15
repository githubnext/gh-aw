// Package workflow - BinEval evaluation job assembler.
package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var evalsJobLog = logger.New("workflow:evals_job")

const evalsStateDir = "/tmp/gh-aw/evals-state"

func evalsBranchName(workflowID string) string {
	return WorkflowStateBranchName(constants.EvalsBranchPrefix, workflowID)
}

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

// buildPushEvalsStateJob creates a job that downloads the evals artifact and commits it to a
// git branch ("evals/{sanitizedID}") so eval results can be read even when artifacts are absent.
func (c *Compiler) buildPushEvalsStateJob(data *WorkflowData) (*Job, error) {
	if data.Evals == nil || !data.Evals.HasEvals() {
		return nil, nil
	}

	evalsJobLog.Printf("Building push_evals_state job (branch=%s)", evalsBranchName(data.WorkflowID))

	var steps []string

	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		traceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		parentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, traceID, parentSpanID)...)
	}

	steps = append(steps,
		"      - name: Checkout repository\n",
		fmt.Sprintf("        uses: %s\n", getActionPin("actions/checkout")),
		"        with:\n",
		"          persist-credentials: false\n",
		"          sparse-checkout: .\n",
	)

	steps = append(steps, c.generateGitConfigurationSteps()...)

	evalsArtifactName := artifactPrefixExprForDownstreamJob(data) + constants.EvalsArtifactName
	steps = append(steps,
		"      - name: Download evals artifact\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/download-artifact")),
		"        continue-on-error: true\n",
		"        with:\n",
		fmt.Sprintf("          name: %s\n", evalsArtifactName),
		fmt.Sprintf("          path: %s\n", evalsStateDir),
	)

	branchName := evalsBranchName(data.WorkflowID)
	steps = append(steps,
		"      - name: Push evals results to git\n",
		"        id: push_evals_state\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		"          GH_TOKEN: ${{ github.token }}\n",
		"          GITHUB_RUN_ID: ${{ github.run_id }}\n",
		"          GITHUB_SERVER_URL: ${{ github.server_url }}\n",
		fmt.Sprintf("          GH_AW_STATE_DIR: %s\n", evalsStateDir),
		fmt.Sprintf("          GH_AW_STATE_BRANCH: %s\n", branchName),
		fmt.Sprintf("          GH_AW_STATE_FILES: %s\n", constants.EvalsResultFilename),
		"          GH_AW_STATE_LABEL: evals results\n",
		"        with:\n",
		"          script: |\n",
		"            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('"+SetupActionDestination+"/push_experiment_state.cjs');\n",
		"            await main();\n",
	)

	if c.actionMode.IsDev() {
		steps = append(steps, c.generateRestoreActionsSetupStep())
	}

	evalsFinished := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.EvalsJobName)),
		BuildStringLiteral("skipped"),
	)
	notCancelled := &NotNode{Child: BuildFunctionCall("cancelled")}
	jobCondition := RenderCondition(BuildAnd(BuildAnd(BuildFunctionCall("always"), notCancelled), evalsFinished))

	job := &Job{
		Name:        pushEvalsStateJobName,
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		If:          jobCondition,
		Permissions: "permissions:\n      contents: write",
		Needs:       []string{string(constants.EvalsJobName), string(constants.ActivationJobName)},
		Steps:       steps,
	}

	return job, nil
}
