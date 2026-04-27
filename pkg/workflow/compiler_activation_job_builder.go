package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
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
}

// newActivationJobBuildContext initializes activation-job state with setup, aw_info, and base outputs.
func (c *Compiler) newActivationJobBuildContext(
	data *WorkflowData,
	preActivationJobCreated bool,
	workflowRunRepoSafety string,
	lockFilename string,
) (*activationJobBuildContext, error) {
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" {
		return nil, errors.New("failed to resolve setup action reference; ensure ./actions/setup exists and is accessible")
	}

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

	ctx.steps = append(ctx.steps, c.generateCheckoutActionsFolder(data)...)
	activationSetupTraceID := ""
	if preActivationJobCreated {
		activationSetupTraceID = fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.PreActivationJobName)
	}
	ctx.steps = append(ctx.steps, c.generateSetupStep(setupActionRef, SetupActionDestination, false, activationSetupTraceID)...)
	ctx.outputs["setup-trace-id"] = "${{ steps.setup.outputs.trace-id }}"

	if isOTLPHeadersPresent(data) {
		ctx.steps = append(ctx.steps, generateOTLPHeadersMaskStep())
	}
	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		compilerActivationJobLog.Print("Adding resolve-host-repo step for workflow_call trigger")
		ctx.steps = append(ctx.steps, c.generateResolveHostRepoStep(data))
	}
	if hasWorkflowCallTrigger(data.On) {
		compilerActivationJobLog.Print("Adding artifact prefix computation step for workflow_call trigger")
		ctx.steps = append(ctx.steps, generateArtifactPrefixStep()...)
		ctx.outputs[constants.ArtifactPrefixOutputName] = "${{ steps.artifact-prefix.outputs.prefix }}"
	}

	engine, err := c.getAgenticEngine(data.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to get agentic engine: %w", err)
	}
	ctx.engine = engine
	compilerActivationJobLog.Print("Generating aw_info step in activation job")
	var awInfoYAML strings.Builder
	c.generateCreateAwInfo(&awInfoYAML, data, engine)
	ctx.steps = append(ctx.steps, awInfoYAML.String())
	ctx.outputs["engine_id"] = "${{ steps.generate_aw_info.outputs.engine_id }}"
	ctx.outputs["model"] = "${{ steps.generate_aw_info.outputs.model }}"
	ctx.outputs["lockdown_check_failed"] = "${{ steps.generate_aw_info.outputs.lockdown_check_failed == 'true' }}"
	if !data.StaleCheckDisabled {
		ctx.outputs["stale_lock_file_failed"] = "${{ steps.check-lock-file.outputs.stale_lock_file_failed == 'true' }}"
	}
	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		ctx.outputs["target_repo"] = "${{ steps.resolve-host-repo.outputs.target_repo }}"
		ctx.outputs["target_repo_name"] = "${{ steps.resolve-host-repo.outputs.target_repo_name }}"
		ctx.outputs["target_ref"] = "${{ steps.resolve-host-repo.outputs.target_ref }}"
	}

	return ctx, nil
}

// addActivationFeedbackAndValidationSteps appends token minting, reactions, secret validation, and guidance.
func (c *Compiler) addActivationFeedbackAndValidationSteps(ctx *activationJobBuildContext) error {
	data := ctx.data
	if data.ActivationGitHubApp != nil && (ctx.hasReaction || ctx.hasStatusComment || ctx.shouldRemoveLabel || ctx.needsAppTokenForAccess) {
		appPerms := NewPermissions()
		addActivationInteractionPermissions(
			appPerms,
			data.On,
			ctx.hasReaction,
			ctx.reactionIssues,
			ctx.reactionPullRequests,
			ctx.reactionDiscussions,
			ctx.hasStatusComment,
			ctx.statusCommentIssues,
			ctx.statusCommentPRs,
			ctx.statusCommentDiscussions,
		)
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
		ctx.steps = append(ctx.steps, c.buildActivationAppTokenMintStep(data.ActivationGitHubApp, appPerms)...)
		ctx.outputs["activation_app_token_minting_failed"] = "${{ steps.activation-app-token.outcome == 'failure' }}"
	}

	if ctx.hasReaction {
		reactionCondition := BuildReactionConditionForTargets(
			ctx.reactionIssues,
			ctx.reactionPullRequests,
			ctx.reactionDiscussions,
		)
		ctx.steps = append(ctx.steps, fmt.Sprintf("      - name: Add %s reaction for immediate feedback\n", data.AIReaction))
		ctx.steps = append(ctx.steps, "        id: react\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("        if: %s\n", RenderCondition(reactionCondition)))
		ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
		ctx.steps = append(ctx.steps, "        env:\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_REACTION: %q\n", data.AIReaction))
		ctx.steps = append(ctx.steps, "        with:\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("          github-token: %s\n", c.resolveActivationToken(data)))
		ctx.steps = append(ctx.steps, "          script: |\n")
		ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("add_reaction.cjs"))
	}

	secretValidationStep := ctx.engine.GetSecretValidationStep(data)
	if len(secretValidationStep) > 0 {
		for _, line := range secretValidationStep {
			ctx.steps = append(ctx.steps, line+"\n")
		}
		ctx.outputs["secret_verification_result"] = "${{ steps.validate-secret.outputs.verification_result }}"
		compilerActivationJobLog.Printf("Added validate-secret step to activation job")
	} else {
		compilerActivationJobLog.Printf("Skipped validate-secret step (engine does not require secret validation)")
	}

	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		compilerActivationJobLog.Print("Adding cross-repo setup guidance step for workflow_call trigger")
		ctx.steps = append(ctx.steps, "      - name: Print cross-repo setup guidance\n")
		ctx.steps = append(ctx.steps, "        if: failure() && steps.resolve-host-repo.outputs.target_repo != github.repository\n")
		ctx.steps = append(ctx.steps, "        run: |\n")
		ctx.steps = append(ctx.steps, "          echo \"::error::COPILOT_GITHUB_TOKEN must be configured in the CALLER repository's secrets.\"\n")
		ctx.steps = append(ctx.steps, "          echo \"::error::For cross-repo workflow_call, secrets must be set in the repository that triggers the workflow.\"\n")
		ctx.steps = append(ctx.steps, "          echo \"::error::See: https://github.github.com/gh-aw/patterns/central-repo-ops/#cross-repo-setup\"\n")
	}

	return nil
}

// addActivationRepositoryAndOutputSteps appends checkout, validation, sanitization, comment, and lock steps.
func (c *Compiler) addActivationRepositoryAndOutputSteps(ctx *activationJobBuildContext) error {
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

	if !data.StaleCheckDisabled {
		ctx.steps = append(ctx.steps, "      - name: Check workflow lock file\n")
		ctx.steps = append(ctx.steps, "        id: check-lock-file\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
		ctx.steps = append(ctx.steps, "        env:\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_WORKFLOW_FILE: \"%s\"\n", ctx.lockFilename))
		ctx.steps = append(ctx.steps, "          GH_AW_CONTEXT_WORKFLOW_REF: \"${{ github.workflow_ref }}\"\n")
		ctx.steps = append(ctx.steps, "        with:\n")
		hashToken := c.resolveActivationToken(data)
		if hashToken != "${{ secrets.GITHUB_TOKEN }}" {
			ctx.steps = append(ctx.steps, fmt.Sprintf("          github-token: %s\n", hashToken))
		}
		ctx.steps = append(ctx.steps, "          script: |\n")
		ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("check_workflow_timestamp_api.cjs"))
	}

	if !data.UpdateCheckDisabled && IsReleasedVersion(c.version) {
		ctx.steps = append(ctx.steps, "      - name: Check compile-agentic version\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
		ctx.steps = append(ctx.steps, "        env:\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_COMPILED_VERSION: \"%s\"\n", c.version))
		ctx.steps = append(ctx.steps, "        with:\n")
		ctx.steps = append(ctx.steps, "          script: |\n")
		ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("check_version_updates.cjs"))
	}

	if data.NeedsTextOutput {
		ctx.steps = append(ctx.steps, "      - name: Compute current body text\n")
		ctx.steps = append(ctx.steps, "        id: sanitized\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
		var domainsStr string
		if data.SafeOutputs != nil && len(data.SafeOutputs.AllowedDomains) > 0 {
			expanded, err := c.computeExpandedAllowedDomainsForSanitization(data)
			if err != nil {
				return err
			}
			domainsStr = expanded
		} else {
			computed, err := c.computeAllowedDomainsForSanitization(data)
			if err != nil {
				return err
			}
			domainsStr = computed
		}
		var envLines []string
		if len(data.Bots) > 0 {
			envLines = append(envLines, formatYAMLEnv("          ", "GH_AW_ALLOWED_BOTS", strings.Join(data.Bots, ",")))
		}
		if domainsStr != "" {
			envLines = append(envLines, formatYAMLEnv("          ", "GH_AW_ALLOWED_DOMAINS", domainsStr))
		}
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
	}

	if data.StatusComment != nil && *data.StatusComment {
		statusCommentCondition := BuildStatusCommentCondition(
			ctx.statusCommentIssues,
			ctx.statusCommentPRs,
			ctx.statusCommentDiscussions,
		)
		ctx.steps = append(ctx.steps, "      - name: Add comment with workflow run link\n")
		ctx.steps = append(ctx.steps, "        id: add-comment\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("        if: %s\n", RenderCondition(statusCommentCondition)))
		ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
		ctx.steps = append(ctx.steps, "        env:\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
		if data.TrackerID != "" {
			ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_TRACKER_ID: %q\n", data.TrackerID))
		}
		if data.LockForAgent {
			ctx.steps = append(ctx.steps, "          GH_AW_LOCK_FOR_AGENT: \"true\"\n")
		}
		if data.SafeOutputs != nil && data.SafeOutputs.Messages != nil {
			messagesJSON, err := serializeMessagesConfig(data.SafeOutputs.Messages)
			if err != nil {
				return fmt.Errorf("failed to serialize messages config for activation job: %w", err)
			}
			if messagesJSON != "" {
				ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
			}
		}
		ctx.steps = append(ctx.steps, "        with:\n")
		commentToken := c.resolveActivationToken(data)
		if commentToken != "${{ secrets.GITHUB_TOKEN }}" {
			ctx.steps = append(ctx.steps, fmt.Sprintf("          github-token: %s\n", commentToken))
		}
		ctx.steps = append(ctx.steps, "          script: |\n")
		ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("add_workflow_run_comment.cjs"))
		ctx.outputs["comment_id"] = "${{ steps.add-comment.outputs.comment-id }}"
		ctx.outputs["comment_url"] = "${{ steps.add-comment.outputs.comment-url }}"
		ctx.outputs["comment_repo"] = "${{ steps.add-comment.outputs.comment-repo }}"
	}

	if data.LockForAgent {
		lockCondition := BuildOr(
			BuildEventTypeEquals("issues"),
			BuildEventTypeEquals("issue_comment"),
		)
		ctx.steps = append(ctx.steps, "      - name: Lock issue for agent workflow\n")
		ctx.steps = append(ctx.steps, "        id: lock-issue\n")
		ctx.steps = append(ctx.steps, fmt.Sprintf("        if: %s\n", RenderCondition(lockCondition)))
		ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))
		ctx.steps = append(ctx.steps, "        with:\n")
		ctx.steps = append(ctx.steps, "          script: |\n")
		ctx.steps = append(ctx.steps, generateGitHubScriptWithRequire("lock-issue.cjs"))
		ctx.outputs["issue_locked"] = "${{ steps.lock-issue.outputs.locked }}"
		if data.AIReaction != "" && data.AIReaction != "none" {
			compilerActivationJobLog.Print("Adding lock notification to reaction message")
		}
	}

	if _, exists := ctx.outputs["comment_id"]; !exists {
		ctx.outputs["comment_id"] = `""`
	}
	if _, exists := ctx.outputs["comment_repo"]; !exists {
		ctx.outputs["comment_repo"] = `""`
	}

	return nil
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
	ctx.steps = append(ctx.steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/upload-artifact")))
	ctx.steps = append(ctx.steps, "        with:\n")
	ctx.steps = append(ctx.steps, fmt.Sprintf("          name: %s\n", activationArtifactName))
	ctx.steps = append(ctx.steps, "          path: |\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/aw_info.json\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/aw-prompts/prompt.txt\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/"+constants.GithubRateLimitsFilename+"\n")
	ctx.steps = append(ctx.steps, "            /tmp/gh-aw/base\n")
	ctx.steps = append(ctx.steps, "          if-no-files-found: ignore\n")
	ctx.steps = append(ctx.steps, "          retention-days: 1\n")
}

// buildActivationPermissions builds activation job permissions from workflow features and selected interactions.
func (c *Compiler) buildActivationPermissions(ctx *activationJobBuildContext) string {
	permsMap := map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead,
	}
	if !ctx.data.StaleCheckDisabled {
		permsMap[PermissionActions] = PermissionRead
	}
	addActivationInteractionPermissionsMap(
		permsMap,
		ctx.data.On,
		ctx.hasReaction,
		ctx.reactionIssues,
		ctx.reactionPullRequests,
		ctx.reactionDiscussions,
		ctx.hasStatusComment,
		ctx.statusCommentIssues,
		ctx.statusCommentPRs,
		ctx.statusCommentDiscussions,
	)
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
	return NewPermissionsFromMap(permsMap).RenderToYAML()
}

// buildActivationEnvironment returns manual-approval environment YAML, with ANSI removed.
func (c *Compiler) buildActivationEnvironment(ctx *activationJobBuildContext) string {
	if ctx.data.ManualApproval == "" {
		return ""
	}
	return "environment: " + stringutil.StripANSI(ctx.data.ManualApproval)
}
