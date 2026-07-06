// Package workflow - push_evals job assembler.
package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// buildPushEvalsJob creates a job that downloads the evals artifact and commits it
// to a git branch ("evals/{sanitizedWorkflowID}") for durable storage across runs.
// Returns nil when evals are not declared in the workflow frontmatter.
func (c *Compiler) buildPushEvalsJob(data *WorkflowData) (*Job, error) {
	if !data.Evals.HasEvals() {
		return nil, nil
	}

	var steps []string

	// Setup step so the push_evals.cjs script is available.
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		traceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		parentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, traceID, parentSpanID)...)
	}

	// Checkout step – configure git credentials without downloading workspace files.
	var checkoutStep strings.Builder
	checkoutStep.WriteString("      - name: Checkout repository\n")
	fmt.Fprintf(&checkoutStep, "        uses: %s\n", getActionPin("actions/checkout"))
	checkoutStep.WriteString("        with:\n")
	checkoutStep.WriteString("          persist-credentials: false\n")
	checkoutStep.WriteString("          sparse-checkout: .\n")
	steps = append(steps, checkoutStep.String())

	// Git configuration (author, email).
	steps = append(steps, c.generateGitConfigurationSteps()...)

	// Download the evals artifact uploaded by the evals job.
	evalsArtifactName := artifactPrefixExprForAgentDownstreamJob(data) + constants.EvalsArtifactName
	var downloadStep strings.Builder
	downloadStep.WriteString("      - name: Download evals artifact\n")
	fmt.Fprintf(&downloadStep, "        uses: %s\n", c.getActionPin("actions/download-artifact"))
	downloadStep.WriteString("        continue-on-error: true\n")
	downloadStep.WriteString("        with:\n")
	fmt.Fprintf(&downloadStep, "          name: %s\n", evalsArtifactName)
	fmt.Fprintf(&downloadStep, "          path: %s\n", evalsArtifactDownloadDir)
	steps = append(steps, downloadStep.String())

	// Push evals results to the git branch via push_evals.cjs.
	branchName := evalsBranchName(data.WorkflowID)

	var pushStep strings.Builder
	pushStep.WriteString("      - name: Push evals results to git\n")
	pushStep.WriteString("        id: push_evals\n")
	pushStep.WriteString("        if: always()\n")
	fmt.Fprintf(&pushStep, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	pushStep.WriteString("        env:\n")
	pushStep.WriteString("          GH_TOKEN: ${{ github.token }}\n")
	pushStep.WriteString("          GITHUB_RUN_ID: ${{ github.run_id }}\n")
	pushStep.WriteString("          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	fmt.Fprintf(&pushStep, "          GH_AW_EVALS_DIR: %s\n", evalsArtifactDownloadDir)
	fmt.Fprintf(&pushStep, "          GH_AW_EVALS_BRANCH: %s\n", branchName)
	pushStep.WriteString("        with:\n")
	pushStep.WriteString("          script: |\n")
	pushStep.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	pushStep.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	pushStep.WriteString("            const { main } = require('" + SetupActionDestination + "/push_evals.cjs');\n")
	pushStep.WriteString("            await main();\n")
	steps = append(steps, pushStep.String())

	// Restore the checkout in dev mode (same reason as push_repo_memory and push_experiments_state).
	if c.actionMode.IsDev() {
		steps = append(steps, c.generateRestoreActionsSetupStep())
	}

	// The push_evals job runs after the evals job completes (success or failure).
	// It does not block on evals succeeding — we always want to persist whatever results were generated.
	evalsNotFailure := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.EvalsJobName)),
		BuildStringLiteral("failure"),
	)
	notCancelled := &NotNode{Child: BuildFunctionCall("cancelled")}
	jobCondition := RenderCondition(BuildAnd(BuildAnd(BuildFunctionCall("always"), notCancelled), evalsNotFailure))

	job := &Job{
		Name:        pushEvalsJobName,
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		If:          jobCondition,
		Permissions: "permissions:\n      contents: write",
		Needs:       []string{string(constants.EvalsJobName), string(constants.ActivationJobName)},
		Steps:       steps,
	}

	return job, nil
}

// evalsBranchName returns the git branch name used to persist evals results for the given workflow ID.
func evalsBranchName(workflowID string) string {
	return string(constants.EvalsBranchPrefix) + "/" + SanitizeWorkflowIDForCacheKey(workflowID)
}
