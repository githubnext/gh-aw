package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

// activationJobBuildContext carries mutable state while composing the activation job.
// It is created once by newActivationJobBuildContext, then incrementally mutated by
// helper methods in buildActivationJob, and discarded after the final Job is assembled.
type activationJobBuildContext struct {
	data                     *WorkflowData
	preActivationJob         bool
	workflowRunRepoSafety    string
	lockFilename             string
	steps                    []string
	outputs                  map[string]string
	engine                   CodingAgentEngine
	hasReaction              bool
	reactionIssues           bool
	reactionPullRequests     bool
	reactionDiscussions      bool
	hasStatusComment         bool
	statusCommentIssues      bool
	statusCommentPRs         bool
	statusCommentDiscussions bool
	hasLabelCommand          bool
	shouldRemoveLabel        bool
	filteredLabelEvents      []string
	needsAppTokenForAccess   bool

	customJobsBeforeActivation []string
	activationNeeds            []string
	activationCondition        string

	// activationAllScripts holds the `run` scripts extracted from jobs.activation.pre-steps,
	// cached to avoid repeated extraction. Only pre-steps are honored for built-in jobs;
	// jobs.activation.steps and jobs.activation.post-steps are not injected by the compiler.
	activationAllScripts []string
	// activationInferredPerms holds the permissions inferred from activationAllScripts,
	// cached here to avoid repeated inference.
	activationInferredPerms map[PermissionScope]PermissionLevel
}

// newActivationJobBuildContext initializes activation-job state with setup, aw_info, and base outputs.
func (c *Compiler) newActivationJobBuildContext(
	data *WorkflowData,
	preActivationJobCreated bool,
	workflowRunRepoSafety string,
	lockFilename string,
) (*activationJobBuildContext, error) {
	compilerActivationJobLog.Printf("Initializing activation job build context: pre_activation=%t, lock=%s", preActivationJobCreated, lockFilename)
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" {
		compilerActivationJobLog.Print("Failed to resolve setup action reference for activation job")
		return nil, errors.New("failed to resolve setup action reference; ensure ./actions/setup exists and is accessible")
	}

	ctx := newActivationBuildContext(data, preActivationJobCreated, workflowRunRepoSafety, lockFilename)
	if err := cacheActivationPreStepPermissions(ctx); err != nil {
		return nil, err
	}
	c.addActivationSetupAndWorkflowCallSteps(ctx, setupActionRef)

	engine, err := c.getAgenticEngine(data.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to get agentic engine: %w", err)
	}
	c.addActivationEngineOutputs(ctx, engine)

	return ctx, nil
}

func newActivationBuildContext(data *WorkflowData, preActivationJobCreated bool, workflowRunRepoSafety, lockFilename string) *activationJobBuildContext {
	ctx := &activationJobBuildContext{
		data:                     data,
		preActivationJob:         preActivationJobCreated,
		workflowRunRepoSafety:    workflowRunRepoSafety,
		lockFilename:             lockFilename,
		outputs:                  map[string]string{},
		hasReaction:              data.AIReaction != "" && data.AIReaction != "none",
		reactionIssues:           shouldIncludeIssueReactions(data),
		reactionPullRequests:     shouldIncludePullRequestReactions(data),
		reactionDiscussions:      shouldIncludeDiscussionReactions(data),
		hasStatusComment:         data.StatusComment != nil && *data.StatusComment,
		statusCommentIssues:      shouldIncludeIssueStatusComments(data),
		statusCommentPRs:         shouldIncludePullRequestStatusComments(data),
		statusCommentDiscussions: shouldIncludeDiscussionStatusComments(data),
		hasLabelCommand:          len(data.LabelCommand) > 0,
		filteredLabelEvents:      FilterLabelCommandEvents(data.LabelCommandEvents),
		needsAppTokenForAccess:   data.ActivationGitHubApp != nil && !data.StaleCheckDisabled,
	}
	ctx.shouldRemoveLabel = ctx.hasLabelCommand && data.LabelCommandRemoveLabel
	return ctx
}

func cacheActivationPreStepPermissions(ctx *activationJobBuildContext) error {
	// Cache scripts from setup/pre-steps and inferred permissions once to avoid redundant
	// extraction and inference calls in buildActivationPermissions and
	// addActivationFeedbackAndValidationSteps.
	// Only setup/pre-steps are honored for built-in jobs: applyBuiltinJobPreSteps (compiler_jobs.go)
	// inserts only jobs.<name>.setup-steps / jobs.<name>.pre-steps; jobs.<name>.steps and jobs.<name>.post-steps are
	// ignored for built-in jobs, so scanning them would cause false-positive errors or
	// unneeded permission grants.
	activationJobName := string(constants.ActivationJobName)
	ctx.activationAllScripts = extractRunScriptsFromJobSection(ctx.data.Jobs, activationJobName, "setup-steps")
	ctx.activationAllScripts = append(ctx.activationAllScripts, extractRunScriptsFromJobSection(ctx.data.Jobs, activationJobName, "pre-steps")...)
	if len(ctx.activationAllScripts) > 0 {
		inferredPerms, err := inferPermissionsFromShellScripts(ctx.activationAllScripts)
		if err != nil {
			return err
		}
		ctx.activationInferredPerms = inferredPerms
	}
	return nil
}

func (c *Compiler) addActivationSetupAndWorkflowCallSteps(ctx *activationJobBuildContext, setupActionRef string) {
	ctx.steps = append(ctx.steps, c.generateCheckoutActionsFolder(ctx.data)...)
	activationSetupTraceID, activationSetupParentSpanID := buildActivationSetupParentSpans(ctx.preActivationJob)
	enableArtifactClient := hasMaxDailyAICGuardrail(ctx.data)
	artifactClientCondition := ""
	if enableArtifactClient {
		artifactClientCondition = maxDailyAICreditsConfiguredIfExpr
	}
	ctx.steps = append(ctx.steps, c.generateSetupStepWithArtifactClientCondition(
		ctx.data,
		setupActionRef,
		SetupActionDestination,
		enableArtifactClient,
		activationSetupTraceID,
		activationSetupParentSpanID,
		artifactClientCondition,
	)...)
	ctx.outputs["setup-trace-id"] = "${{ steps.setup.outputs.trace-id }}"
	ctx.outputs["setup-span-id"] = "${{ steps.setup.outputs.span-id }}"
	ctx.outputs["setup-parent-span-id"] = "${{ steps.setup.outputs.parent-span-id || steps.setup.outputs.span-id }}"
	c.addActivationWorkflowCallResolutionSteps(ctx)
}

func buildActivationSetupParentSpans(preActivationJobCreated bool) (traceID string, parentSpanID string) {
	if !preActivationJobCreated {
		return "", ""
	}
	return fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.PreActivationJobName),
		setupParentSpanNeedsExpr(constants.PreActivationJobName)
}

func (c *Compiler) addActivationWorkflowCallResolutionSteps(ctx *activationJobBuildContext) {
	if isOTLPHeadersPresent(ctx.data) {
		ctx.steps = append(ctx.steps, generateOTLPHeadersMaskStep())
	}
	if isOTLPAttributesPresent(ctx.data) {
		ctx.steps = append(ctx.steps, generateOTLPAttributesMaskStep())
	}
	if hasWorkflowCallTrigger(ctx.data.On) && !ctx.data.InlinedImports {
		compilerActivationJobLog.Print("Adding resolve-host-repo step for workflow_call trigger")
		ctx.steps = append(ctx.steps, c.generateResolveHostRepoStep(ctx.data))
	}
	if hasWorkflowCallTrigger(ctx.data.On) {
		compilerActivationJobLog.Print("Adding artifact prefix computation step for workflow_call trigger")
		ctx.steps = append(ctx.steps, generateArtifactPrefixStep()...)
		ctx.outputs[constants.ArtifactPrefixOutputName] = "${{ steps.artifact-prefix.outputs.prefix }}"
	}
}

func (c *Compiler) addActivationEngineOutputs(ctx *activationJobBuildContext, engine CodingAgentEngine) {
	ctx.engine = engine
	compilerActivationJobLog.Print("Generating aw_info step in activation job")
	var awInfoYAML strings.Builder
	c.generateCreateAwInfo(&awInfoYAML, ctx.data, engine)
	ctx.steps = append(ctx.steps, awInfoYAML.String())
	ctx.outputs["engine_id"] = "${{ steps.generate_aw_info.outputs.engine_id }}"
	ctx.outputs["model"] = "${{ steps.generate_aw_info.outputs.model }}"
	ctx.outputs["lockdown_check_failed"] = "${{ steps.generate_aw_info.outputs.lockdown_check_failed == 'true' }}"
	if !ctx.data.StaleCheckDisabled {
		ctx.outputs["stale_lock_file_failed"] = "${{ steps.check-lock-file.outputs.stale_lock_file_failed == 'true' }}"
	}
	if hasWorkflowCallTrigger(ctx.data.On) && !ctx.data.InlinedImports {
		ctx.outputs["target_repo"] = "${{ steps.resolve-host-repo.outputs.target_repo }}"
		ctx.outputs["target_repo_name"] = "${{ steps.resolve-host-repo.outputs.target_repo_name }}"
		// target_ref: dispatch-compatible branch/tag ref (e.g. refs/heads/main) parsed from
		// job.workflow_ref. Used by dispatch_workflow safe outputs as the `ref` argument to
		// createWorkflowDispatch. The GitHub workflow dispatch API does not accept commit SHAs.
		ctx.outputs["target_ref"] = "${{ steps.resolve-host-repo.outputs.target_ref }}"
		// target_checkout_ref: immutable commit SHA from job.workflow_sha. Used by actions/checkout
		// in the activation job to pin to the exact executing revision.
		ctx.outputs["target_checkout_ref"] = "${{ steps.resolve-host-repo.outputs.target_checkout_ref }}"
	}
}

// addActivationFeedbackAndValidationSteps appends token minting, reactions, secret validation, and guidance.
func (c *Compiler) addActivationFeedbackAndValidationSteps(ctx *activationJobBuildContext) error {
	data := ctx.data
	compilerActivationJobLog.Printf("Adding activation feedback/validation steps: reaction=%t, status_comment=%t, remove_label=%t, app_token_for_access=%t",
		ctx.hasReaction, ctx.hasStatusComment, ctx.shouldRemoveLabel, ctx.needsAppTokenForAccess)
	c.maybeAddActivationAppTokenMintStep(ctx)
	if hasMaxDailyAICGuardrail(data) {
		ctx.steps = append(ctx.steps, c.buildActivationDailyAICGuardrailStep(data)...)
		ctx.outputs["daily_ai_credits_exceeded"] = "${{ steps.daily-effective-workflow-guardrail.outputs.daily_ai_credits_exceeded == 'true' }}"
		ctx.outputs["daily_ai_credits_total_effective_tokens"] = "${{ steps.daily-effective-workflow-guardrail.outputs.daily_ai_credits_total_effective_tokens || '' }}"
		ctx.outputs["daily_ai_credits_threshold"] = "${{ steps.daily-effective-workflow-guardrail.outputs.daily_ai_credits_threshold || '' }}"
	}
	c.addActivationReactionStep(ctx)
	c.addActivationSecretValidationStep(ctx)
	c.addActivationCrossRepoGuidanceStep(ctx)
	return nil
}

func (c *Compiler) maybeAddActivationAppTokenMintStep(ctx *activationJobBuildContext) {
	if !activationJobNeedsAppToken(ctx) {
		return
	}
	appPerms := buildActivationAppTokenPermissions(ctx)
	ctx.steps = append(ctx.steps, c.buildActivationAppTokenMintStep(ctx.data.ActivationGitHubApp, appPerms)...)
	ctx.outputs["activation_app_token_minting_failed"] = "${{ steps.activation-app-token.outcome == 'failure' }}"
}

// activationJobNeedsAppToken gates app-token minting and must stay in sync with
// buildActivationAppTokenPermissions. Any new trigger added here (reaction,
// status-comment, remove-label, repo-access, guardrail) must also add the
// corresponding permission grants there; drift causes either unnecessary minting
// or runtime 403s from missing scopes. TestActivationJobNeedsAppToken locks the
// gate behavior for these triggers.
func activationJobNeedsAppToken(ctx *activationJobBuildContext) bool {
	if ctx.data.ActivationGitHubApp == nil {
		return false
	}
	return ctx.hasReaction ||
		ctx.hasStatusComment ||
		ctx.shouldRemoveLabel ||
		ctx.needsAppTokenForAccess ||
		hasMaxDailyAICGuardrail(ctx.data)
}

func buildActivationAppTokenPermissions(ctx *activationJobBuildContext) *Permissions {
	appPerms := NewPermissions()
	addActivationInteractionPermissions(
		appPerms,
		activationInteractionPermissionsOptions{
			onSection:                         ctx.data.On,
			hasReaction:                       ctx.hasReaction,
			reactionIncludesIssues:            ctx.reactionIssues,
			reactionIncludesPullRequests:      ctx.reactionPullRequests,
			reactionIncludesDiscussions:       ctx.reactionDiscussions,
			hasStatusComment:                  ctx.hasStatusComment,
			statusCommentIncludesIssues:       ctx.statusCommentIssues,
			statusCommentIncludesPullRequests: ctx.statusCommentPRs,
			statusCommentIncludesDiscussions:  ctx.statusCommentDiscussions,
		},
	)
	if ctx.data.CommandCentralized && (ctx.hasReaction || ctx.hasStatusComment) {
		syntheticOn := buildCentralizedCommandOnSection(ctx.data.CommandEvents)
		if syntheticOn != "" {
			addActivationInteractionPermissions(
				appPerms,
				activationInteractionPermissionsOptions{
					onSection:                         syntheticOn,
					hasReaction:                       ctx.hasReaction,
					reactionIncludesIssues:            ctx.reactionIssues,
					reactionIncludesPullRequests:      ctx.reactionPullRequests,
					reactionIncludesDiscussions:       ctx.reactionDiscussions,
					hasStatusComment:                  ctx.hasStatusComment,
					statusCommentIncludesIssues:       ctx.statusCommentIssues,
					statusCommentIncludesPullRequests: ctx.statusCommentPRs,
					statusCommentIncludesDiscussions:  ctx.statusCommentDiscussions,
				},
			)
		}
	}
	if hasWorkflowCallTrigger(ctx.data.On) && (ctx.hasReaction || ctx.hasStatusComment) {
		addActivationInteractionPermissions(
			appPerms,
			activationInteractionPermissionsOptions{
				hasReaction:                       ctx.hasReaction,
				reactionIncludesIssues:            ctx.reactionIssues,
				reactionIncludesPullRequests:      ctx.reactionPullRequests,
				reactionIncludesDiscussions:       ctx.reactionDiscussions,
				hasStatusComment:                  ctx.hasStatusComment,
				statusCommentIncludesIssues:       ctx.statusCommentIssues,
				statusCommentIncludesPullRequests: ctx.statusCommentPRs,
				statusCommentIncludesDiscussions:  ctx.statusCommentDiscussions,
			},
		)
	}
	// Keep this aligned with addActivationLabelPermissions: app-token scopes are
	// computed separately from GITHUB_TOKEN scopes because app-token permissions
	// only apply to steps using the minted app token, while label permissions in
	// addActivationLabelPermissions are only for GITHUB_TOKEN execution paths.
	// This intentionally mirrors addActivationLabelPermissions without the
	// ActivationGitHubApp == nil guard because this function runs only when
	// activationJobNeedsAppToken confirms app-token minting is enabled.
	if ctx.shouldRemoveLabel {
		if slices.Contains(ctx.filteredLabelEvents, "issues") || slices.Contains(ctx.filteredLabelEvents, "pull_request") {
			appPerms.Set(PermissionIssues, PermissionWrite)
		}
		if slices.Contains(ctx.filteredLabelEvents, "discussion") {
			appPerms.Set(PermissionDiscussions, PermissionWrite)
		}
	}
	if ctx.needsAppTokenForAccess {
		appPerms.Set(PermissionContents, PermissionRead)
	}
	if hasMaxDailyAICGuardrail(ctx.data) {
		appPerms.Set(PermissionActions, PermissionRead)
	}
	// Add GitHub App-only permissions inferred from activation job gh CLI commands so the
	// minted App token includes the scopes those commands require (e.g. codespaces: read
	// for `gh codespace list`). Only App-only scopes are passed here.
	for scope, level := range ctx.activationInferredPerms {
		if IsGitHubAppOnlyScope(scope) {
			appPerms.Set(scope, level)
		}
	}
	return appPerms
}

func (c *Compiler) addActivationReactionStep(ctx *activationJobBuildContext) {
	if !ctx.hasReaction {
		return
	}
	reactionCondition := BuildReactionConditionForTargets(
		ctx.reactionIssues,
		ctx.reactionPullRequests,
		ctx.reactionDiscussions,
		ctx.data.CommandCentralized,
	)
	ctx.steps = append(ctx.steps, fmt.Sprintf("      - name: Add %s reaction for immediate feedback\n", ctx.data.AIReaction))
	ctx.steps = append(ctx.steps, "        id: react\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("        if: %s\n", RenderCondition(reactionCondition)))
	ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", ctx.data)))
	ctx.steps = append(ctx.steps, "        env:\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_REACTION: %q\n", ctx.data.AIReaction))
	ctx.steps = append(ctx.steps, "        with:\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("          github-token: %s\n", c.resolveActivationToken(ctx.data)))
	ctx.steps = append(ctx.steps, "          script: |\n")
	ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("add_reaction.cjs"))
}

func (c *Compiler) addActivationSecretValidationStep(ctx *activationJobBuildContext) {
	secretValidationStep := ctx.engine.GetSecretValidationStep(ctx.data)
	if len(secretValidationStep) == 0 {
		compilerActivationJobLog.Printf("Skipped validate-secret step (engine does not require secret validation)")
		return
	}
	for _, line := range secretValidationStep {
		ctx.steps = append(ctx.steps, line+"\n")
	}
	ctx.outputs["secret_verification_result"] = "${{ steps.validate-secret.outputs.verification_result }}"
	compilerActivationJobLog.Printf("Added validate-secret step to activation job")
}

func (c *Compiler) addActivationCrossRepoGuidanceStep(ctx *activationJobBuildContext) {
	if !hasWorkflowCallTrigger(ctx.data.On) || ctx.data.InlinedImports {
		return
	}
	compilerActivationJobLog.Print("Adding cross-repo setup guidance step for workflow_call trigger")
	ctx.steps = append(ctx.steps, "      - name: Print cross-repo setup guidance\n")
	ctx.steps = append(ctx.steps, "        if: failure() && steps.resolve-host-repo.outputs.target_repo != github.repository\n")
	ctx.steps = append(ctx.steps, "        run: |\n")
	ctx.steps = append(ctx.steps, "          echo \"::error::COPILOT_GITHUB_TOKEN must be configured in the CALLER repository's secrets.\"\n")
	ctx.steps = append(ctx.steps, "          echo \"::error::For cross-repo workflow_call, secrets must be set in the repository that triggers the workflow.\"\n")
	ctx.steps = append(ctx.steps, "          echo \"::error::See: https://github.github.com/gh-aw/patterns/central-repo-ops/#cross-repo-setup\"\n")
}

func (c *Compiler) buildActivationDailyAICGuardrailStep(data *WorkflowData) []string {
	var steps []string
	// Prepend cache restore step so cached AIC values from prior runs are available
	// when the guardrail script runs, allowing it to skip artifact downloads.
	if data.WorkflowID != "" {
		sanitized := SanitizeWorkflowIDForCacheKey(data.WorkflowID)
		cacheKeyPrefix := fmt.Sprintf("agentic-workflow-usage-%s-", sanitized)
		steps = append(steps, "      - name: Restore daily AIC usage cache\n")
		steps = append(steps, "        id: restore-daily-aic-cache\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", maxDailyAICreditsConfiguredIfExpr))
		steps = append(steps, "        continue-on-error: true\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/cache/restore", data)))
		steps = append(steps, "        with:\n")
		steps = append(steps, fmt.Sprintf("          key: %s${{ github.run_id }}\n", cacheKeyPrefix))
		steps = append(steps, fmt.Sprintf("          restore-keys: %s\n", cacheKeyPrefix))
		steps = append(steps, "          path: /tmp/gh-aw/agentic-workflow-usage-cache.jsonl\n")
		// Artifact-based fallback for cross-branch cache misses.
		// GitHub Actions actions/cache is branch-scoped: caches written by the conclusion job
		// on one PR branch are invisible to the activation job running on a different PR branch.
		// This step downloads the most recent aic-usage-cache artifact uploaded by a prior
		// conclusion job so that the guardrail script can skip per-run artifact downloads.
		// Cache-miss detection is performed inside restore_aic_usage_cache_fallback.cjs using
		// the cache restore outputs forwarded via env vars.
		steps = append(steps, "      - name: Restore daily AIC usage cache (artifact fallback)\n")
		steps = append(steps, "        id: restore-daily-aic-cache-fallback\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", maxDailyAICreditsConfiguredIfExpr))
		steps = append(steps, "        continue-on-error: true\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
		steps = append(steps, "        env:\n")
		steps = append(steps, "          GH_AW_RESTORE_DAILY_AIC_CACHE_HIT: ${{ steps.restore-daily-aic-cache.outputs.cache-hit }}\n")
		steps = append(steps, "          GH_AW_RESTORE_DAILY_AIC_CACHE_MATCHED_KEY: ${{ steps.restore-daily-aic-cache.outputs.cache-matched-key }}\n")
		steps = append(steps, "        with:\n")
		steps = append(steps, fmt.Sprintf("          github-token: %s\n", c.resolveActivationToken(data)))
		steps = append(steps, "          script: |\n")
		steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
		steps = append(steps, "            setupGlobals(core, github, context, exec, io, getOctokit);\n")
		steps = append(steps, "            const { main } = require('"+SetupActionDestination+"/restore_aic_usage_cache_fallback.cjs');\n")
		steps = append(steps, "            await main();\n")
	}
	steps = append(steps, "      - name: Check daily workflow token guardrail\n")
	steps = append(steps, "        id: daily-effective-workflow-guardrail\n")
	steps = append(steps, fmt.Sprintf("        if: %s\n", maxDailyAICreditsConfiguredIfExpr))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_ID: %q\n", data.WorkflowID))
	steps = append(steps, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
	steps = append(steps, "          GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT: ${{ github.event.inputs.aw_context || '' }}\n")
	steps = append(steps, fmt.Sprintf("          GH_AW_HAS_SLASH_COMMAND: %q\n", strconv.FormatBool(len(data.Command) > 0)))
	steps = append(steps, fmt.Sprintf("          GH_AW_HAS_LABEL_COMMAND: %q\n", strconv.FormatBool(len(data.LabelCommand) > 0)))
	steps = append(steps, fmt.Sprintf("          GH_AW_GITHUB_TOKEN: %s\n", c.resolveActivationToken(data)))
	steps = append(steps, buildTemplatableIntEnvVar(maxDailyAICreditsEnvVar, data.MaxDailyAICredits)...)
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          github-token: %s\n", c.resolveActivationToken(data)))
	steps = append(steps, "          script: |\n")
	steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
	steps = append(steps, "            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	steps = append(steps, "            const { main } = require('"+SetupActionDestination+"/check_daily_aic_workflow_guardrail.cjs');\n")
	steps = append(steps, "            await main();\n")
	return steps
}

func buildRuntimeFeaturesSummaryStep() []string {
	return []string{
		"      - name: Log runtime features\n",
		"        if: ${{ contains(toJSON(vars), '\"GH_AW_RUNTIME_FEATURES\":') }}\n",
		"        run: bash \"${RUNNER_TEMP}/gh-aw/actions/log_runtime_features_summary.sh\"\n",
	}
}

func buildPolicyStrictEnforcementStep() []string {
	return []string{
		"      - name: Enforce strict mode policy\n",
		fmt.Sprintf("        if: ${{ vars.%s == 'true' }}\n", compilerenv.PolicyStrict),
		"        run: |\n",
		fmt.Sprintf("          echo \"::error::%s=true but this workflow was not compiled in strict mode. Recompile with --strict or strict: true.\"\n", compilerenv.PolicyStrict),
		"          exit 1\n",
	}
}

// addActivationRepositoryAndOutputSteps appends checkout, validation, sanitization, comment, and lock steps.
func (c *Compiler) addActivationRepositoryAndOutputSteps(ctx *activationJobBuildContext) error {
	data := ctx.data
	compilerActivationJobLog.Printf("Adding activation repository/output steps: stale_check_disabled=%t, needs_text_output=%t, lock_for_agent=%t",
		data.StaleCheckDisabled, data.NeedsTextOutput, data.LockForAgent)
	c.addActivationCheckoutAndBaseRestoreStep(ctx)
	c.addActivationLockFileStep(ctx)
	c.addActivationVersionCheckStep(ctx)
	if err := c.addActivationTextOutputStep(ctx); err != nil {
		return err
	}
	if err := c.addActivationStatusCommentStep(ctx); err != nil {
		return err
	}
	c.addActivationIssueLockStep(ctx)
	ensureActivationCommentOutputs(ctx)
	return nil
}

func (c *Compiler) addActivationCheckoutAndBaseRestoreStep(ctx *activationJobBuildContext) {
	data := ctx.data
	checkoutSteps := c.generateCheckoutGitHubFolderForActivation(data)
	ctx.steps = append(ctx.steps, checkoutSteps...)
	if len(checkoutSteps) > 0 {
		compilerActivationJobLog.Print("Adding step to save agent config folders for base branch restoration")
		registry := GetGlobalEngineRegistry()
		ctx.steps = append(ctx.steps, generateSaveBaseGitHubFoldersStep(
			registry.GetAllAgentManifestFolders(),
			registry.GetAllAgentManifestFiles(),
		)...)
	}
}

func (c *Compiler) addActivationLockFileStep(ctx *activationJobBuildContext) {
	if ctx.data.StaleCheckDisabled {
		return
	}
	ctx.steps = append(ctx.steps, "      - name: Check workflow lock file\n")
	ctx.steps = append(ctx.steps, "        id: check-lock-file\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", ctx.data)))
	ctx.steps = append(ctx.steps, "        env:\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_WORKFLOW_FILE: \"%s\"\n", ctx.lockFilename))
	ctx.steps = append(ctx.steps, "          GH_AW_CONTEXT_WORKFLOW_REF: \"${{ github.workflow_ref }}\"\n")
	if ctx.data.StaleCheckFull {
		ctx.steps = append(ctx.steps, "          GH_AW_STALE_CHECK_FULL: \"true\"\n")
	}
	ctx.steps = append(ctx.steps, "        with:\n")
	hashToken := c.resolveActivationToken(ctx.data)
	if hashToken != "${{ secrets.GITHUB_TOKEN }}" {
		ctx.steps = append(ctx.steps, fmt.Sprintf("          github-token: %s\n", hashToken))
	}
	ctx.steps = append(ctx.steps, "          script: |\n")
	ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("check_workflow_timestamp_api.cjs"))
}

func (c *Compiler) addActivationVersionCheckStep(ctx *activationJobBuildContext) {
	if ctx.data.UpdateCheckDisabled || !IsReleasedVersion(c.version) {
		return
	}
	ctx.steps = append(ctx.steps, "      - name: Check compile-agentic version\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", ctx.data)))
	ctx.steps = append(ctx.steps, "        env:\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_COMPILED_VERSION: \"%s\"\n", c.version))
	ctx.steps = append(ctx.steps, "        with:\n")
	ctx.steps = append(ctx.steps, "          script: |\n")
	ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("check_version_updates.cjs"))
}

func (c *Compiler) addActivationSkillInstallSteps(ctx *activationJobBuildContext) error {
	if len(ctx.data.Skills) == 0 {
		return nil
	}

	engineID := ""
	if ctx.data.EngineConfig != nil {
		engineID = ctx.data.EngineConfig.ID
	}
	skillDir := GetEngineSkillDir(engineID)
	skillSpecsJSON, err := json.Marshal(ctx.data.Skills)
	if err != nil {
		return fmt.Errorf("marshal activation skill specs: %w", err)
	}
	escapedSkillSpecsJSON := strings.ReplaceAll(string(skillSpecsJSON), "'", "''")

	ctx.steps = append(ctx.steps, "      - name: Upgrade gh CLI for frontmatter skills\n")
	ctx.steps = append(ctx.steps, "        run: |\n")
	ctx.steps = append(ctx.steps, "          set -euo pipefail\n")
	ctx.steps = append(ctx.steps, "          bash \"${RUNNER_TEMP}/gh-aw/actions/install_gh_cli.sh\"\n")
	ctx.steps = append(ctx.steps, "          GH_VERSION=$(gh --version | awk 'NR==1 {print $3}')\n")
	ctx.steps = append(ctx.steps, "          REQUIRED=\"2.90.0\"\n")
	ctx.steps = append(ctx.steps, "          echo \"gh version: ${GH_VERSION}\"\n")
	ctx.steps = append(ctx.steps, "          if ! printf '%s\\n%s\\n' \"$REQUIRED\" \"$GH_VERSION\" | sort -V -C; then\n")
	ctx.steps = append(ctx.steps, "            echo \"::error::gh ${GH_VERSION} is older than required ${REQUIRED} (gh skill support requires v2.90+)\"\n")
	ctx.steps = append(ctx.steps, "            exit 1\n")
	ctx.steps = append(ctx.steps, "          fi\n")

	ctx.steps = append(ctx.steps, "      - name: Install frontmatter skills\n")
	ctx.steps = append(ctx.steps, "        env:\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_TOKEN: %s\n", c.resolveActivationToken(ctx.data)))
	ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_SKILL_DIR: %q\n", skillDir))
	ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_SKILLS_SUMMARY: '%s'\n", escapedSkillSpecsJSON))
	for i, skillSpec := range ctx.data.Skills {
		ctx.steps = append(ctx.steps, formatYAMLEnv("          ", fmt.Sprintf("GH_AW_SKILL_SPEC_%d", i), skillSpec))
	}
	ctx.steps = append(ctx.steps, "        run: |\n")
	ctx.steps = append(ctx.steps, "          set -euo pipefail\n")
	ctx.steps = append(ctx.steps, "          SKILLS_DST=\"/tmp/gh-aw/${GH_AW_SKILL_DIR}\"\n")
	ctx.steps = append(ctx.steps, "          mkdir -p \"${SKILLS_DST}\"\n")
	ctx.steps = append(ctx.steps, "          echo \"Installing frontmatter skills to ${SKILLS_DST}\"\n")
	ctx.steps = append(ctx.steps, "          echo \"Existing skills at destination may be replaced (--force) to ensure pinned refs are up to date\"\n")
	for i := range ctx.data.Skills {
		ctx.steps = append(ctx.steps, fmt.Sprintf("          skill_spec=\"${GH_AW_SKILL_SPEC_%d}\"\n", i))
		ctx.steps = append(ctx.steps, "          echo \"Installing skill reference: ${skill_spec}\"\n")
		// Keep this runtime owner/repo vs owner/repo/path detection aligned with
		// isRepositorySkillSpec so expression-based refs behave the same after resolution.
		ctx.steps = append(ctx.steps, "          skill_base=\"${skill_spec%@*}\"\n")
		ctx.steps = append(ctx.steps, "          install_args=()\n")
		ctx.steps = append(ctx.steps, "          if [[ \"${skill_base}\" == */* && \"${skill_base}\" != */*/* ]]; then\n")
		ctx.steps = append(ctx.steps, "            install_args+=(--all)\n")
		ctx.steps = append(ctx.steps, "          fi\n")
		ctx.steps = append(ctx.steps, "          gh skill install \"${skill_spec}\" \"${install_args[@]}\" --dir \"${SKILLS_DST}\" --force\n")
	}
	ctx.steps = append(ctx.steps, "          SKILL_COUNT=$(find \"${SKILLS_DST}\" -name \"SKILL.md\" | wc -l | tr -d '[:space:]')\n")
	ctx.steps = append(ctx.steps, "          echo \"Installed ${SKILL_COUNT} skill file(s)\"\n")
	ctx.steps = append(ctx.steps, "          core_summary_path=\"${GITHUB_STEP_SUMMARY:-}\"\n")
	ctx.steps = append(ctx.steps, "          if [ -n \"${core_summary_path}\" ]; then\n")
	ctx.steps = append(ctx.steps, "            {\n")
	ctx.steps = append(ctx.steps, "              echo \"### Frontmatter skills installed\"\n")
	ctx.steps = append(ctx.steps, "              echo \"\"\n")
	ctx.steps = append(ctx.steps, "              echo \"- Engine skill directory: \\`${GH_AW_SKILL_DIR}\\`\"\n")
	ctx.steps = append(ctx.steps, "              echo \"- Requested references: \\`${GH_AW_SKILLS_SUMMARY}\\`\"\n")
	ctx.steps = append(ctx.steps, "              echo \"- Installed SKILL.md files: ${SKILL_COUNT}\"\n")
	ctx.steps = append(ctx.steps, "            } >> \"${core_summary_path}\"\n")
	ctx.steps = append(ctx.steps, "          fi\n")
	return nil
}

func (c *Compiler) addActivationTextOutputStep(ctx *activationJobBuildContext) error {
	if !ctx.data.NeedsTextOutput {
		return nil
	}
	ctx.steps = append(ctx.steps, "      - name: Compute current body text\n")
	ctx.steps = append(ctx.steps, "        id: sanitized\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", ctx.data)))
	domainsStr, err := c.computeActivationSanitizationDomains(ctx.data)
	if err != nil {
		return err
	}
	envLines := buildActivationTextOutputEnvLines(ctx.data, domainsStr)
	if len(envLines) > 0 {
		ctx.steps = append(ctx.steps, "        env:\n")
		ctx.steps = append(ctx.steps, envLines...)
	}
	ctx.steps = append(ctx.steps, "        with:\n")
	ctx.steps = append(ctx.steps, "          script: |\n")
	ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("compute_text.cjs"))
	ctx.outputs["text"] = "${{ steps.sanitized.outputs.text }}"
	ctx.outputs["title"] = "${{ steps.sanitized.outputs.title }}"
	ctx.outputs["body"] = "${{ steps.sanitized.outputs.body }}"
	return nil
}

func (c *Compiler) computeActivationSanitizationDomains(data *WorkflowData) (string, error) {
	if data.SafeOutputs != nil && len(data.SafeOutputs.AllowedDomains) > 0 {
		return c.computeExpandedAllowedDomainsForSanitization(data)
	}
	return c.computeAllowedDomainsForSanitization(data)
}

func buildActivationTextOutputEnvLines(data *WorkflowData, domainsStr string) []string {
	var envLines []string
	if len(data.Bots) > 0 {
		envLines = append(envLines, formatYAMLEnv("          ", "GH_AW_ALLOWED_BOTS", strings.Join(data.Bots, ",")))
	}
	if domainsStr != "" {
		envLines = append(envLines, formatYAMLEnv("          ", "GH_AW_ALLOWED_DOMAINS", domainsStr))
	}
	return envLines
}

func (c *Compiler) addActivationStatusCommentStep(ctx *activationJobBuildContext) error {
	if ctx.data.StatusComment == nil || !*ctx.data.StatusComment {
		return nil
	}
	statusCommentCondition := BuildStatusCommentCondition(
		ctx.statusCommentIssues,
		ctx.statusCommentPRs,
		ctx.statusCommentDiscussions,
		ctx.data.CommandCentralized,
	)
	ctx.steps = append(ctx.steps, "      - name: Add comment with workflow run link\n")
	ctx.steps = append(ctx.steps, "        id: add-comment\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("        if: %s\n", RenderCondition(statusCommentCondition)))
	ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", ctx.data)))
	ctx.steps = append(ctx.steps, "        env:\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", ctx.data.Name))
	if ctx.data.TrackerID != "" {
		ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_TRACKER_ID: %q\n", ctx.data.TrackerID))
	}
	if ctx.data.LockForAgent {
		ctx.steps = append(ctx.steps, "          GH_AW_LOCK_FOR_AGENT: \"true\"\n")
	}
	if err := addActivationSafeOutputMessagesEnv(ctx); err != nil {
		return err
	}
	ctx.steps = append(ctx.steps, "        with:\n")
	commentToken := c.resolveActivationToken(ctx.data)
	if commentToken != "${{ secrets.GITHUB_TOKEN }}" {
		ctx.steps = append(ctx.steps, fmt.Sprintf("          github-token: %s\n", commentToken))
	}
	ctx.steps = append(ctx.steps, "          script: |\n")
	ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("add_workflow_run_comment.cjs"))
	ctx.outputs["comment_id"] = "${{ steps.add-comment.outputs.comment-id }}"
	ctx.outputs["comment_url"] = "${{ steps.add-comment.outputs.comment-url }}"
	ctx.outputs["comment_repo"] = "${{ steps.add-comment.outputs.comment-repo }}"
	return nil
}

func addActivationSafeOutputMessagesEnv(ctx *activationJobBuildContext) error {
	if ctx.data.SafeOutputs == nil || ctx.data.SafeOutputs.Messages == nil {
		return nil
	}
	messagesJSON, err := serializeMessagesConfig(ctx.data.SafeOutputs.Messages)
	if err != nil {
		return fmt.Errorf("failed to serialize messages config for activation job: %w", err)
	}
	if messagesJSON != "" {
		ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
	}
	return nil
}

func (c *Compiler) addActivationIssueLockStep(ctx *activationJobBuildContext) {
	if !ctx.data.LockForAgent {
		return
	}
	lockCondition := BuildOr(
		BuildEventTypeEquals("issues"),
		BuildEventTypeEquals("issue_comment"),
	)
	ctx.steps = append(ctx.steps, "      - name: Lock issue for agentic workflow\n")
	ctx.steps = append(ctx.steps, "        id: lock-issue\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("        if: %s\n", RenderCondition(lockCondition)))
	ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", ctx.data)))
	ctx.steps = append(ctx.steps, "        with:\n")
	ctx.steps = append(ctx.steps, "          script: |\n")
	ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("lock-issue.cjs"))
	ctx.outputs["issue_locked"] = "${{ steps.lock-issue.outputs.locked }}"
	if ctx.data.AIReaction != "" && ctx.data.AIReaction != "none" {
		compilerActivationJobLog.Print("Adding lock notification to reaction message")
	}
}

func ensureActivationCommentOutputs(ctx *activationJobBuildContext) {
	if _, exists := ctx.outputs["comment_id"]; !exists {
		ctx.outputs["comment_id"] = `""`
	}
	if _, exists := ctx.outputs["comment_repo"]; !exists {
		ctx.outputs["comment_repo"] = `""`
	}
}

// addActivationCommandAndLabelOutputs appends slash-command and label-command output steps.
func (c *Compiler) addActivationCommandAndLabelOutputs(ctx *activationJobBuildContext) error {
	data := ctx.data

	if len(data.Command) > 0 {
		if ctx.preActivationJob {
			ctx.outputs["slash_command"] = fmt.Sprintf("${{ needs.%s.outputs.%s }}", string(constants.PreActivationJobName), constants.MatchedCommandOutput)
		} else {
			ctx.outputs["slash_command"] = fmt.Sprintf("${{ steps.%s.outputs.%s }}", constants.CheckCommandPositionStepID, constants.MatchedCommandOutput)
		}
	}

	if ctx.shouldRemoveLabel {
		ctx.steps = append(ctx.steps, "      - name: Remove trigger label\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("        id: %s\n", constants.RemoveTriggerLabelStepID))
		ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
		ctx.steps = append(ctx.steps, "        env:\n")
		labelNamesJSON, err := json.Marshal(data.LabelCommand)
		if err != nil {
			return fmt.Errorf("failed to marshal label-command names: %w", err)
		}
		ctx.steps = append(ctx.steps, formatYAMLEnv("          ", "GH_AW_LABEL_NAMES", string(labelNamesJSON)))
		ctx.steps = append(ctx.steps, "        with:\n")
		labelToken := c.resolveActivationToken(data)
		if labelToken != "${{ secrets.GITHUB_TOKEN }}" {
			ctx.steps = append(ctx.steps, fmt.Sprintf("          github-token: %s\n", labelToken))
		}
		ctx.steps = append(ctx.steps, "          script: |\n")
		ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("remove_trigger_label.cjs"))
		ctx.outputs["label_command"] = fmt.Sprintf("${{ steps.%s.outputs.label_name }}", constants.RemoveTriggerLabelStepID)
	} else if ctx.hasLabelCommand {
		ctx.steps = append(ctx.steps, "      - name: Get trigger label name\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("        id: %s\n", constants.GetTriggerLabelStepID))
		ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
		if len(data.Command) > 0 {
			ctx.steps = append(ctx.steps, "        env:\n")
			if ctx.preActivationJob {
				ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_MATCHED_COMMAND: ${{ needs.%s.outputs.%s }}\n", string(constants.PreActivationJobName), constants.MatchedCommandOutput))
			} else {
				ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_MATCHED_COMMAND: ${{ steps.%s.outputs.%s }}\n", constants.CheckCommandPositionStepID, constants.MatchedCommandOutput))
			}
		}
		ctx.steps = append(ctx.steps, "        with:\n")
		ctx.steps = append(ctx.steps, "          script: |\n")
		ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("get_trigger_label.cjs"))
		ctx.outputs["label_command"] = fmt.Sprintf("${{ steps.%s.outputs.label_name }}", constants.GetTriggerLabelStepID)
		ctx.outputs["command_name"] = fmt.Sprintf("${{ steps.%s.outputs.command_name }}", constants.GetTriggerLabelStepID)
	}

	return nil
}

// configureActivationNeedsAndCondition computes and sets activation dependencies and final job condition.
// This helper mutates the context but only derives values from workflow data and has no error paths.
func (c *Compiler) configureActivationNeedsAndCondition(ctx *activationJobBuildContext) {
	data := ctx.data
	compilerActivationJobLog.Printf("Configuring activation needs and condition: pre_activation=%t, has_if=%t", ctx.preActivationJob, data.If != "")
	customJobsBeforeActivation := c.getCustomJobsDependingOnPreActivation(data.Jobs)
	for _, jobName := range data.OnNeeds {
		if !slices.Contains(customJobsBeforeActivation, jobName) {
			customJobsBeforeActivation = append(customJobsBeforeActivation, jobName)
		}
	}
	promptReferencedJobs := c.getCustomJobsReferencedInPromptWithNoActivationDep(data)
	for _, jobName := range promptReferencedJobs {
		if !slices.Contains(customJobsBeforeActivation, jobName) {
			customJobsBeforeActivation = append(customJobsBeforeActivation, jobName)
			compilerActivationJobLog.Printf("Added '%s' to activation dependencies: referenced in markdown body and has no explicit needs", jobName)
		}
	}
	ctx.customJobsBeforeActivation = customJobsBeforeActivation

	if ctx.preActivationJob {
		ctx.activationNeeds = []string{string(constants.PreActivationJobName)}
		ctx.activationNeeds = append(ctx.activationNeeds, customJobsBeforeActivation...)
		activatedExpr := BuildEquals(
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.%s", string(constants.PreActivationJobName), constants.ActivatedOutput)),
			BuildStringLiteral("true"),
		)
		if data.If != "" && c.referencesCustomJobOutputs(data.If, data.Jobs) && len(customJobsBeforeActivation) > 0 {
			unwrappedIf := stripExpressionWrapper(data.If)
			ifExpr := &ExpressionNode{Expression: unwrappedIf}
			ctx.activationCondition = RenderCondition(BuildAnd(activatedExpr, ifExpr))
		} else if data.If != "" && !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			unwrappedIf := stripExpressionWrapper(data.If)
			ifExpr := &ExpressionNode{Expression: unwrappedIf}
			ctx.activationCondition = RenderCondition(BuildAnd(activatedExpr, ifExpr))
		} else {
			ctx.activationCondition = RenderCondition(activatedExpr)
		}
	} else {
		ctx.activationNeeds = append(ctx.activationNeeds, customJobsBeforeActivation...)
		if data.If != "" && c.referencesCustomJobOutputs(data.If, data.Jobs) && len(customJobsBeforeActivation) > 0 {
			ctx.activationCondition = data.If
		} else if !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			ctx.activationCondition = data.If
		}
	}

	if ctx.workflowRunRepoSafety != "" {
		ctx.activationCondition = c.combineJobIfConditions(ctx.activationCondition, ctx.workflowRunRepoSafety)
	}
}

// addActivationArtifactUploadStep appends the activation artifact upload step for downstream jobs.
func (c *Compiler) addActivationArtifactUploadStep(ctx *activationJobBuildContext) {
	compilerActivationJobLog.Print("Adding activation artifact upload step")
	activationArtifactName := artifactPrefixExprForActivationJob(ctx.data) + constants.ActivationArtifactName
	ctx.steps = append(ctx.steps, "      - name: Upload activation artifact\n")
	ctx.steps = append(ctx.steps, "        if: success()\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/upload-artifact")))
	ctx.steps = append(ctx.steps, "        with:\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("          name: %s\n", activationArtifactName))
	ctx.steps = append(ctx.steps, "          include-hidden-files: true\n")
	ctx.steps = append(ctx.steps, "          path: |\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/aw_info.json\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/models.json\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/aw-prompts/prompt.txt\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/aw-prompts/prompt-template.txt\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/aw-prompts/prompt-import-tree.json\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/"+constants.GithubRateLimitsFilename+"\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/base\n")
	engineID := ""
	if ctx.data.EngineConfig != nil {
		engineID = ctx.data.EngineConfig.ID
	}
	// Include the engine-specific sub-agent staging directory only when inline agents are enabled.
	if isFeatureEnabled(constants.FeatureFlag("inline-agents"), ctx.data) {
		subAgentDir := GetEngineSubAgentDir(engineID)
		ctx.steps = append(ctx.steps, fmt.Sprintf("            /tmp/gh-aw/%s\n", subAgentDir))
	}
	// Always include the engine-specific skill directory when either inline skills are enabled
	// or frontmatter skills are configured.
	if isFeatureEnabled(constants.FeatureFlag("inline-agents"), ctx.data) || len(ctx.data.Skills) > 0 {
		skillDir := GetEngineSkillDir(engineID)
		ctx.steps = append(ctx.steps, fmt.Sprintf("            /tmp/gh-aw/%s\n", skillDir))
	}
	ctx.steps = append(ctx.steps, "          if-no-files-found: ignore\n")
	ctx.steps = append(ctx.steps, "          retention-days: 1\n")
}

// buildActivationPermissions builds activation job permissions from workflow features and selected interactions.
// Returns an error if any activation job step section contains write gh CLI commands that would require write permissions.
func (c *Compiler) buildActivationPermissions(ctx *activationJobBuildContext) (string, error) {
	permsMap := c.buildActivationBasePermissions(ctx)
	c.addCentralizedCommandActivationPermissions(permsMap, ctx)
	c.addWorkflowCallActivationPermissions(permsMap, ctx)
	c.addActivationLabelPermissions(permsMap, ctx)
	if err := c.addActivationScriptPermissions(permsMap, ctx); err != nil {
		return "", err
	}
	return NewPermissionsFromMap(permsMap).RenderToYAML(), nil
}

func (c *Compiler) buildActivationBasePermissions(ctx *activationJobBuildContext) map[PermissionScope]PermissionLevel {
	permsMap := map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead,
	}
	if !ctx.data.StaleCheckDisabled || hasMaxDailyAICGuardrail(ctx.data) {
		permsMap[PermissionActions] = PermissionRead
	}
	addActivationInteractionPermissionsMap(permsMap, activationInteractionPermissionsOptions{
		onSection:                         ctx.data.On,
		hasReaction:                       ctx.hasReaction,
		reactionIncludesIssues:            ctx.reactionIssues,
		reactionIncludesPullRequests:      ctx.reactionPullRequests,
		reactionIncludesDiscussions:       ctx.reactionDiscussions,
		hasStatusComment:                  ctx.hasStatusComment,
		statusCommentIncludesIssues:       ctx.statusCommentIssues,
		statusCommentIncludesPullRequests: ctx.statusCommentPRs,
		statusCommentIncludesDiscussions:  ctx.statusCommentDiscussions,
	})
	return permsMap
}

func (c *Compiler) addCentralizedCommandActivationPermissions(permsMap map[PermissionScope]PermissionLevel, ctx *activationJobBuildContext) {
	// For centralized slash_command workflows, the compiled "on" section only contains
	// workflow_dispatch, so addActivationInteractionPermissionsMap above cannot detect the
	// original event types and skips write permissions. Supplement with a synthetic section
	// built from the declared command events so reactions and status-comments work correctly.
	if ctx.data.CommandCentralized && (ctx.hasReaction || ctx.hasStatusComment) {
		syntheticOn := buildCentralizedCommandOnSection(ctx.data.CommandEvents)
		if syntheticOn != "" {
			addActivationInteractionPermissionsMap(permsMap, activationInteractionPermissionsOptions{
				onSection:                         syntheticOn,
				hasReaction:                       ctx.hasReaction,
				reactionIncludesIssues:            ctx.reactionIssues,
				reactionIncludesPullRequests:      ctx.reactionPullRequests,
				reactionIncludesDiscussions:       ctx.reactionDiscussions,
				hasStatusComment:                  ctx.hasStatusComment,
				statusCommentIncludesIssues:       ctx.statusCommentIssues,
				statusCommentIncludesPullRequests: ctx.statusCommentPRs,
				statusCommentIncludesDiscussions:  ctx.statusCommentDiscussions,
			})
		}
	}
}

// addWorkflowCallActivationPermissions supplements the activation job's permission map when the
// workflow is triggered via workflow_call (i.e. it is used as a reusable workflow).
//
// At compile time it is impossible to know which GitHub event will fire in the *calling* workflow,
// so the compiler cannot restrict permissions to a specific event type (e.g. "issues" or
// "pull_request"). Instead it falls back to the broad permission set: all permission scopes that
// the configured reactions / status-comments could ever need are granted, respecting the per-type
// opt-out flags (reaction.issues, reaction.pull-requests, etc.).
//
// Because the caller event type is unknown at compile time, this path always uses the broad
// fallback (addBroadActivationInteractionPermissions) instead of event-aware trigger parsing.
func (c *Compiler) addWorkflowCallActivationPermissions(permsMap map[PermissionScope]PermissionLevel, ctx *activationJobBuildContext) {
	if !hasWorkflowCallTrigger(ctx.data.On) {
		return
	}
	if !ctx.hasReaction && !ctx.hasStatusComment {
		return
	}
	compilerActivationJobLog.Print("workflow_call trigger detected; applying broad interaction permissions for reactions/status-comments")
	addBroadActivationInteractionPermissions(permsMap, activationInteractionPermissionsOptions{
		hasReaction:                       ctx.hasReaction,
		reactionIncludesIssues:            ctx.reactionIssues,
		reactionIncludesPullRequests:      ctx.reactionPullRequests,
		reactionIncludesDiscussions:       ctx.reactionDiscussions,
		hasStatusComment:                  ctx.hasStatusComment,
		statusCommentIncludesIssues:       ctx.statusCommentIssues,
		statusCommentIncludesPullRequests: ctx.statusCommentPRs,
		statusCommentIncludesDiscussions:  ctx.statusCommentDiscussions,
	})
}

func (c *Compiler) addActivationLabelPermissions(permsMap map[PermissionScope]PermissionLevel, ctx *activationJobBuildContext) {
	if ctx.data.LockForAgent {
		permsMap[PermissionIssues] = PermissionWrite
	}
	if ctx.shouldRemoveLabel && ctx.data.ActivationGitHubApp == nil {
		if slices.Contains(ctx.filteredLabelEvents, "issues") || slices.Contains(ctx.filteredLabelEvents, "pull_request") {
			permsMap[PermissionIssues] = PermissionWrite
		}
		if slices.Contains(ctx.filteredLabelEvents, "discussion") {
			permsMap[PermissionDiscussions] = PermissionWrite
		}
	}
}

func (c *Compiler) addActivationScriptPermissions(permsMap map[PermissionScope]PermissionLevel, ctx *activationJobBuildContext) error {
	// Infer permissions required by gh CLI calls in jobs.activation step sections
	// (pre-steps, steps, post-steps). This ensures that user-defined steps that call
	// `gh pr diff`, `gh issue view`, etc. get the permissions they need without requiring
	// manual permission declarations.
	// Scripts and inferred permissions are cached in ctx to avoid redundant computation.
	if len(ctx.activationAllScripts) > 0 {
		// Detect write commands first — these are not permitted in the activation job
		// because it intentionally operates with read-only permissions.
		writeCmds, err := detectWriteCommandsInShellScripts(ctx.activationAllScripts)
		if err != nil {
			return err
		}
		if len(writeCmds) > 0 {
			return fmt.Errorf(
				"activation job uses write gh command(s) [%s]; write operations are not permitted in activation job steps because the activation job runs with read-only permissions. Move write operations to the agent job steps or use safe-outputs. See: https://github.github.com/gh-aw/reference/safe-outputs/",
				strings.Join(writeCmds, ", "),
			)
		}
		for scope, level := range ctx.activationInferredPerms {
			if _, exists := permsMap[scope]; !exists {
				permsMap[scope] = level
			}
		}
	}
	return nil
}

// buildActivationEnvironment returns manual-approval environment YAML, with ANSI removed.
func (c *Compiler) buildActivationEnvironment(ctx *activationJobBuildContext) string {
	if ctx.data.ManualApproval == "" {
		return ""
	}
	compilerActivationJobLog.Print("Activation job uses manual-approval environment gate")
	return "environment: " + stringutil.StripANSI(ctx.data.ManualApproval)
}

func buildDailyAICActivationJobEnv(data *WorkflowData) map[string]string {
	if !hasMaxDailyAICGuardrail(data) || !hasMaxDailyAICFrontmatterConfig(data) {
		return nil
	}
	value := strings.TrimSpace(*data.MaxDailyAICredits)
	if value == "" {
		return nil
	}
	if isExpression(value) {
		return map[string]string{maxDailyAICreditsEnvVar: value}
	}
	return map[string]string{maxDailyAICreditsEnvVar: strconv.Quote(value)}
}
