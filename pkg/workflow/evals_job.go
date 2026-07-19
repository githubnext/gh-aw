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

// buildEvalsJob creates a separate evals job that runs after the agent job (and detection
// job when enabled), allowing it to run in parallel with safe_outputs.
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

	steps = append(steps, c.buildEvalsSetupSteps(data)...)
	agentArtifactPrefix := artifactPrefixExprForDownstreamJob(data)
	steps = append(steps, buildAgentOutputDownloadSteps(agentArtifactPrefix, c.getActionPin)...)
	steps = append(steps, buildExperimentArtifactDownloadSteps(data, c.getActionPin)...)
	steps = append(steps, c.buildEvalsJobSteps(data)...)

	needs := evalsJobNeeds(data)
	evalsJobLog.Printf("Evals job dependencies resolved: needs=%v", needs)

	job := &Job{
		Name:        string(constants.EvalsJobName),
		Needs:       needs,
		If:          evalsJobCondition(),
		RunsOn:      c.indentYAMLLines(evalsJobRunsOn(data), "    "),
		Environment: c.indentYAMLLines(data.Environment, "    "),
		Permissions: evalsJobPermissions(data),
		Steps:       steps,
	}

	return job, nil
}

func (c *Compiler) buildEvalsSetupSteps(data *WorkflowData) []string {
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" && !c.actionMode.IsScript() {
		return nil
	}
	var steps []string
	steps = append(steps, c.generateCheckoutActionsFolder(data)...)
	evalsTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
	evalsParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
	return append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, evalsTraceID, evalsParentSpanID)...)
}

func evalsJobNeeds(data *WorkflowData) []string {
	needs := []string{string(constants.AgentJobName), string(constants.ActivationJobName)}
	if IsDetectionJobEnabled(data.SafeOutputs) {
		needs = append(needs, string(constants.DetectionJobName))
	}
	return needs
}

func evalsJobCondition() string {
	alwaysFunc := BuildFunctionCall("always")
	upstreamNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)
	return RenderCondition(BuildAnd(alwaysFunc, upstreamNotSkipped))
}

func evalsJobRunsOn(data *WorkflowData) string {
	if data.Evals != nil && data.Evals.RunsOn != "" {
		return normalizeRunsOnSnippet(data.Evals.RunsOn)
	}
	return "runs-on: ubuntu-latest"
}

func evalsJobPermissions(data *WorkflowData) string {
	perms := NewPermissionsContentsRead()
	if hasCopilotRequestsWritePermission(data) {
		perms.Set(PermissionCopilotRequests, PermissionWrite)
	}
	if data.EngineConfig != nil && data.EngineConfig.Auth != nil && data.EngineConfig.Auth.Type == "github-oidc" {
		perms.Set(PermissionIdToken, PermissionWrite)
	}
	if hasOTLPGitHubOIDCAuth(data.ParsedFrontmatter, data.RawFrontmatter) {
		perms.Set(PermissionIdToken, PermissionWrite)
	}
	return perms.RenderToYAML()
}

// buildPushEvalsStateJob creates a job that downloads the evals artifact and commits it to a
// git branch ("evals/{sanitizedID}") so eval results can be read even when artifacts are absent.
func (c *Compiler) buildPushEvalsStateJob(data *WorkflowData) (*Job, error) {
	if data.Evals == nil || !data.Evals.HasEvals() {
		return nil, nil
	}

	evalsJobLog.Printf("Building push_evals_state job (branch=%s)", evalsBranchName(data.WorkflowID))

	var steps []string

	steps = append(steps, c.buildEvalsSetupSteps(data)...)
	steps = append(steps, buildPushEvalsCheckoutSteps()...)
	steps = append(steps, c.generateGitConfigurationSteps()...)
	steps = append(steps, c.buildPushEvalsDownloadSteps(data)...)
	branchName := evalsBranchName(data.WorkflowID)
	steps = append(steps, buildPushEvalsStateSteps(data, branchName)...)

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

func buildPushEvalsCheckoutSteps() []string {
	return []string{
		"      - name: Checkout repository\n",
		fmt.Sprintf("        uses: %s\n", getActionPin("actions/checkout")),
		"        with:\n",
		"          persist-credentials: false\n",
		"          sparse-checkout: .\n",
	}
}

func (c *Compiler) buildPushEvalsDownloadSteps(data *WorkflowData) []string {
	evalsArtifactName := artifactPrefixExprForDownstreamJob(data) + constants.EvalsArtifactName
	return []string{
		"      - name: Download evals artifact\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/download-artifact")),
		"        continue-on-error: true\n",
		"        with:\n",
		fmt.Sprintf("          name: %s\n", evalsArtifactName),
		fmt.Sprintf("          path: %s\n", evalsStateDir),
	}
}

func buildPushEvalsStateSteps(data *WorkflowData, branchName string) []string {
	return []string{
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
		"            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('" + SetupActionDestination + "/push_experiment_state.cjs');\n",
		"            await main();\n",
	}
}
