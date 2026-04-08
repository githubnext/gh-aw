package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var compilerActivationJobLog = logger.New("workflow:compiler_activation_job")

// buildActivationJob creates the activation job that handles timestamp checking, reactions, and locking.
// This job depends on the pre-activation job if it exists, and runs before the main agent job.
func (c *Compiler) buildActivationJob(data *WorkflowData, preActivationJobCreated bool, workflowRunRepoSafety string, lockFilename string) (*Job, error) {
	outputs := map[string]string{}
	var steps []string

	// Team member check is now handled by the separate check_membership job
	// No inline role checks needed in the task job anymore

	// Add setup step to copy activation scripts (required - no inline fallback)
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" {
		return nil, errors.New("setup action reference is required but could not be resolved")
	}

	// For dev mode (local action path), checkout the actions folder first
	steps = append(steps, c.generateCheckoutActionsFolder(data)...)

	// Activation job doesn't need project support (no safe outputs processed here)
	// When a pre-activation job exists, reuse its trace ID so all three jobs (pre_activation,
	// activation, agent) share a single OTLP trace. When no pre-activation job exists, the
	// empty string instructs the setup action to generate a new root trace ID.
	activationSetupTraceID := ""
	if preActivationJobCreated {
		activationSetupTraceID = fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.PreActivationJobName)
	}
	steps = append(steps, c.generateSetupStep(setupActionRef, SetupActionDestination, false, false, activationSetupTraceID)...)
	// Expose the trace ID for cross-job span correlation so downstream jobs can reuse it
	outputs["setup-trace-id"] = "${{ steps.setup.outputs.trace-id }}"

	// Mask OTLP telemetry headers immediately after setup so authentication tokens cannot
	// leak into runner debug logs for any subsequent step in the activation job.
	if isOTLPHeadersPresent(data) {
		steps = append(steps, generateOTLPHeadersMaskStep())
	}

	// When a workflow_call trigger is present, resolve the platform (host) repository before
	// generating aw_info so that target_repo can be included in aw_info.json and used by
	// the checkout step. This is necessary for event-driven relays (e.g. on: issue_comment)
	// where github.event_name is not 'workflow_call', making the previous expression
	// (github.event_name == 'workflow_call' && github.action_repository || github.repository)
	// unreliable. GITHUB_WORKFLOW_REF always reflects the executing workflow's repo regardless
	// of how it was triggered.
	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		compilerActivationJobLog.Print("Adding resolve-host-repo step for workflow_call trigger")
		steps = append(steps, c.generateResolveHostRepoStep())
	}

	// In workflow_call context, compute a unique artifact prefix from a hash of the
	// workflow inputs. This prefix is applied to all artifact names so that multiple
	// callers of the same reusable workflow can run concurrently in the same workflow
	// run without artifact name collisions.
	if hasWorkflowCallTrigger(data.On) {
		compilerActivationJobLog.Print("Adding artifact prefix computation step for workflow_call trigger")
		steps = append(steps, generateArtifactPrefixStep()...)
		outputs[constants.ArtifactPrefixOutputName] = "${{ steps.artifact-prefix.outputs.prefix }}"
	}

	// Generate agentic run info immediately after setup so aw_info.json is ready as early as
	// possible. This step runs before the reaction so that its data is captured even if the
	// reaction step fails.
	engine, err := c.getAgenticEngine(data.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to get agentic engine: %w", err)
	}
	compilerActivationJobLog.Print("Generating aw_info step in activation job")
	var awInfoYaml strings.Builder
	c.generateCreateAwInfo(&awInfoYaml, data, engine)
	steps = append(steps, awInfoYaml.String())
	// Expose the model output from the activation job so downstream jobs can reference it
	outputs["model"] = "${{ steps.generate_aw_info.outputs.model }}"
	// Track whether the lockdown check failed so the conclusion job can surface
	// the configuration error in the failure issue even when the agent never ran.
	outputs["lockdown_check_failed"] = "${{ steps.generate_aw_info.outputs.lockdown_check_failed == 'true' }}"

	// Expose the resolved platform (host) repository and ref so agent and safe_outputs jobs
	// can use needs.activation.outputs.target_repo / target_ref for any checkout that must
	// target the platform repo and branch rather than github.repository (the caller's repo in
	// cross-repo workflow_call scenarios, especially when pinned to a non-default branch).
	if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		outputs["target_repo"] = "${{ steps.resolve-host-repo.outputs.target_repo }}"
		outputs["target_repo_name"] = "${{ steps.resolve-host-repo.outputs.target_repo_name }}"
		outputs["target_ref"] = "${{ steps.resolve-host-repo.outputs.target_ref }}"
	}

	// Compute reaction/comment/label flags early so the app token and reaction steps can be
	// inserted right after generate_aw_info for fast user feedback.
	hasReaction := data.AIReaction != "" && data.AIReaction != "none"
	hasStatusComment := data.StatusComment != nil && *data.StatusComment
	hasLabelCommand := len(data.LabelCommand) > 0
	// shouldRemoveLabel is true when label-command is active AND remove_label is not disabled
	shouldRemoveLabel := hasLabelCommand && data.LabelCommandRemoveLabel
	// Compute filtered label events once and reuse below (permissions + app token scopes)
	filteredLabelEvents := FilterLabelCommandEvents(data.LabelCommandEvents)

	// Mint a single activation app token upfront if a GitHub App is configured and any
	// step in the activation job will need it (reaction, status-comment, or label removal).
	// This avoids minting multiple tokens.
	if data.ActivationGitHubApp != nil && (hasReaction || hasStatusComment || shouldRemoveLabel) {
		// Build the combined permissions needed for all activation steps.
		// For label removal we only add the scopes required by the enabled events.
		appPerms := NewPermissions()
		if hasReaction || hasStatusComment {
			appPerms.Set(PermissionIssues, PermissionWrite)
			appPerms.Set(PermissionPullRequests, PermissionWrite)
			appPerms.Set(PermissionDiscussions, PermissionWrite)
		}
		if shouldRemoveLabel {
			if slices.Contains(filteredLabelEvents, "issues") || slices.Contains(filteredLabelEvents, "pull_request") {
				appPerms.Set(PermissionIssues, PermissionWrite)
			}
			if slices.Contains(filteredLabelEvents, "discussion") {
				appPerms.Set(PermissionDiscussions, PermissionWrite)
			}
		}
		steps = append(steps, c.buildActivationAppTokenMintStep(data.ActivationGitHubApp, appPerms)...)
		// Track whether the token minting succeeded so the conclusion job can surface
		// GitHub App authentication errors in the failure issue.
		outputs["activation_app_token_minting_failed"] = "${{ steps.activation-app-token.outcome == 'failure' }}"
	}

	// Add reaction step right after generate_aw_info so it is shown to the user as fast as
	// possible. generate_aw_info runs first so its data is captured even if the reaction fails.
	// This runs in the activation job so it can use any configured github-token or github-app.
	if hasReaction {
		reactionCondition := BuildReactionCondition()

		steps = append(steps, fmt.Sprintf("      - name: Add %s reaction for immediate feedback\n", data.AIReaction))
		steps = append(steps, "        id: react\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", RenderCondition(reactionCondition)))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

		// Add environment variables
		steps = append(steps, "        env:\n")
		// Quote the reaction value to prevent YAML interpreting +1/-1 as integers
		steps = append(steps, fmt.Sprintf("          GH_AW_REACTION: %q\n", data.AIReaction))

		steps = append(steps, "        with:\n")
		// Use configured github-token or app-minted token; fall back to GITHUB_TOKEN
		steps = append(steps, fmt.Sprintf("          github-token: %s\n", c.resolveActivationToken(data)))
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("add_reaction.cjs"))
	}

	// Add secret validation step before context variable validation.
	// This validates that the required engine secrets are available before any other checks.
	secretValidationStep := engine.GetSecretValidationStep(data)
	if len(secretValidationStep) > 0 {
		for _, line := range secretValidationStep {
			steps = append(steps, line+"\n")
		}
		outputs["secret_verification_result"] = "${{ steps.validate-secret.outputs.verification_result }}"
		compilerActivationJobLog.Printf("Added validate-secret step to activation job")
	} else {
		compilerActivationJobLog.Printf("Skipped validate-secret step (engine does not require secret validation)")
	}

	// Add cross-repo setup guidance when workflow_call is a trigger.
	// This step only runs when secret validation fails in a cross-repo context,
	// providing actionable guidance to the caller team about configuring secrets.
	// Use steps.resolve-host-repo.outputs.target_repo != github.repository instead of
	// github.event_name == 'workflow_call': the latter never fires for event-driven relays
	// (issue_comment/push → workflow_call) where the event_name is the originating event.
	if hasWorkflowCallTrigger(data.On) {
		compilerActivationJobLog.Print("Adding cross-repo setup guidance step for workflow_call trigger")
		steps = append(steps, "      - name: Cross-repo setup guidance\n")
		steps = append(steps, "        if: failure() && steps.resolve-host-repo.outputs.target_repo != github.repository\n")
		steps = append(steps, "        run: |\n")
		steps = append(steps, "          echo \"::error::COPILOT_GITHUB_TOKEN must be configured in the CALLER repository's secrets.\"\n")
		steps = append(steps, "          echo \"::error::For cross-repo workflow_call, secrets must be set in the repository that triggers the workflow.\"\n")
		steps = append(steps, "          echo \"::error::See: https://github.github.com/gh-aw/patterns/central-repo-ops/#cross-repo-setup\"\n")
	}

	// Checkout .github and .agents folders for accessing workflow configurations and runtime imports
	// This is needed for prompt generation which may reference runtime imports from .github folder
	// Always add this checkout in activation job since it needs access to workflow files for runtime imports
	checkoutSteps := c.generateCheckoutGitHubFolderForActivation(data)
	steps = append(steps, checkoutSteps...)

	// Add frontmatter hash check to detect stale lock files using GitHub API.
	// Compares the hash embedded in the lock file against the source .md file to detect stale lock files.
	// No checkout step needed - uses GitHub API to fetch file contents.
	// Skipped when on.stale-check: false is set in the frontmatter.
	if !data.StaleCheckDisabled {
		steps = append(steps, "      - name: Check workflow lock file\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_FILE: \"%s\"\n", lockFilename))
		// Inject the GitHub Actions context workflow_ref expression as GH_AW_CONTEXT_WORKFLOW_REF
		// for check_workflow_timestamp_api.cjs. Note: despite what was previously documented,
		// ${{ github.workflow_ref }} resolves to the CALLER's workflow ref in reusable workflow
		// contexts, not the callee's. The referenced_workflows API lookup in the script is the
		// primary mechanism for resolving the callee's repo; GH_AW_CONTEXT_WORKFLOW_REF serves
		// as a fallback when the API is unavailable or finds no matching entry.
		steps = append(steps, "          GH_AW_CONTEXT_WORKFLOW_REF: \"${{ github.workflow_ref }}\"\n")
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_workflow_timestamp_api.cjs"))
	}

	// Add compile-agentic version update check, unless disabled via check-for-updates: false.
	// The check downloads .github/aw/releases.json from the gh-aw repository and verifies that the
	// compiled version is not blocked and meets the minimum supported version requirement.
	// If the download fails, the check is skipped (soft failure).
	if !data.UpdateCheckDisabled && IsReleasedVersion(c.version) {
		steps = append(steps, "      - name: Check compile-agentic version\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_COMPILED_VERSION: \"%s\"\n", c.version))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("check_version_updates.cjs"))
	}

	// Generate sanitized text/title/body outputs if needed
	// This step computes sanitized versions of the triggering content (issue/PR/comment text, title, body)
	// and makes them available as step outputs.
	//
	// IMPORTANT: These outputs are referenced as steps.sanitized.outputs.{text|title|body} in workflow markdown.
	// Users should use ${{ steps.sanitized.outputs.text }} directly in their workflows.
	// The outputs are also exposed as needs.activation.outputs.* for downstream jobs.
	if data.NeedsTextOutput {
		steps = append(steps, "      - name: Compute current body text\n")
		steps = append(steps, "        id: sanitized\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		if len(data.Bots) > 0 {
			steps = append(steps, "        env:\n")
			steps = append(steps, formatYAMLEnv("          ", "GH_AW_ALLOWED_BOTS", strings.Join(data.Bots, ",")))
		}
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("compute_text.cjs"))

		// Set up outputs - includes text, title, and body
		// These are exposed as needs.activation.outputs.* for downstream jobs
		// and as steps.sanitized.outputs.* within the activation job (where prompts are rendered)
		outputs["text"] = "${{ steps.sanitized.outputs.text }}"
		outputs["title"] = "${{ steps.sanitized.outputs.title }}"
		outputs["body"] = "${{ steps.sanitized.outputs.body }}"
	}

	// Add comment with workflow run link if status comments are explicitly enabled
	if data.StatusComment != nil && *data.StatusComment {
		reactionCondition := BuildReactionCondition()

		steps = append(steps, "      - name: Add comment with workflow run link\n")
		steps = append(steps, "        id: add-comment\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", RenderCondition(reactionCondition)))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

		// Add environment variables
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))

		// Add tracker-id if present
		if data.TrackerID != "" {
			steps = append(steps, fmt.Sprintf("          GH_AW_TRACKER_ID: %q\n", data.TrackerID))
		}

		// Add lock-for-agent status if enabled
		if data.LockForAgent {
			steps = append(steps, "          GH_AW_LOCK_FOR_AGENT: \"true\"\n")
		}

		// Pass custom messages config if present (for custom run-started messages)
		if data.SafeOutputs != nil && data.SafeOutputs.Messages != nil {
			messagesJSON, err := serializeMessagesConfig(data.SafeOutputs.Messages)
			if err != nil {
				return nil, fmt.Errorf("failed to serialize messages config for activation job: %w", err)
			}
			if messagesJSON != "" {
				steps = append(steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
			}
		}

		steps = append(steps, "        with:\n")
		// Use configured github-token or app-minted token if set; omit to use default GITHUB_TOKEN
		commentToken := c.resolveActivationToken(data)
		if commentToken != "${{ secrets.GITHUB_TOKEN }}" {
			steps = append(steps, fmt.Sprintf("          github-token: %s\n", commentToken))
		}
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("add_workflow_run_comment.cjs"))

		// Add comment outputs
		outputs["comment_id"] = "${{ steps.add-comment.outputs.comment-id }}"
		outputs["comment_url"] = "${{ steps.add-comment.outputs.comment-url }}"
		outputs["comment_repo"] = "${{ steps.add-comment.outputs.comment-repo }}"
	}

	// Add lock step if lock-for-agent is enabled
	if data.LockForAgent {
		// Build condition: only lock if event type is 'issues' or 'issue_comment'
		// lock-for-agent can be configured under on.issues or on.issue_comment
		// For issue_comment events, context.issue.number automatically resolves to the parent issue
		lockCondition := BuildOr(
			BuildEventTypeEquals("issues"),
			BuildEventTypeEquals("issue_comment"),
		)

		steps = append(steps, "      - name: Lock issue for agent workflow\n")
		steps = append(steps, "        id: lock-issue\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", RenderCondition(lockCondition)))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("lock-issue.cjs"))

		// Add output for tracking if issue was locked
		outputs["issue_locked"] = "${{ steps.lock-issue.outputs.locked }}"

		// Add lock message to reaction comment if reaction is enabled
		if data.AIReaction != "" && data.AIReaction != "none" {
			compilerActivationJobLog.Print("Adding lock notification to reaction message")
		}
	}

	// Always declare comment_id and comment_repo outputs to avoid actionlint errors
	// These will be empty if no reaction is configured, and the scripts handle empty values gracefully
	// Use plain empty strings (quoted) to avoid triggering security scanners like zizmor
	if _, exists := outputs["comment_id"]; !exists {
		outputs["comment_id"] = `""`
	}
	if _, exists := outputs["comment_repo"]; !exists {
		outputs["comment_repo"] = `""`
	}

	// Add slash_command output if this is a command workflow
	// This output contains the matched command name from check_command_position step
	if len(data.Command) > 0 {
		if preActivationJobCreated {
			// Reference the matched_command output from pre_activation job
			outputs["slash_command"] = fmt.Sprintf("${{ needs.%s.outputs.%s }}", string(constants.PreActivationJobName), constants.MatchedCommandOutput)
		} else {
			// Fallback to steps reference if pre_activation doesn't exist (shouldn't happen for command workflows)
			outputs["slash_command"] = fmt.Sprintf("${{ steps.%s.outputs.%s }}", constants.CheckCommandPositionStepID, constants.MatchedCommandOutput)
		}
	}

	// Add label removal step and label_command output for label-command workflows.
	// When a label-command trigger fires, the triggering label is immediately removed
	// so that the same label can be applied again to trigger the workflow in the future.
	// This step is skipped when remove_label is set to false.
	if shouldRemoveLabel {
		// The removal step only makes sense for actual "labeled" events; for
		// workflow_dispatch we skip it silently via the env-based label check.
		steps = append(steps, "      - name: Remove trigger label\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.RemoveTriggerLabelStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		// Pass label names as a JSON array so the script can validate the label
		labelNamesJSON, err := json.Marshal(data.LabelCommand)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal label-command names: %w", err)
		}
		steps = append(steps, formatYAMLEnv("          ", "GH_AW_LABEL_NAMES", string(labelNamesJSON)))
		steps = append(steps, "        with:\n")
		// Use GitHub App or custom token if configured (avoids needing elevated GITHUB_TOKEN permissions)
		labelToken := c.resolveActivationToken(data)
		if labelToken != "${{ secrets.GITHUB_TOKEN }}" {
			steps = append(steps, fmt.Sprintf("          github-token: %s\n", labelToken))
		}
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("remove_trigger_label.cjs"))

		// Expose the matched label name as a job output for downstream jobs to consume
		outputs["label_command"] = fmt.Sprintf("${{ steps.%s.outputs.label_name }}", constants.RemoveTriggerLabelStepID)
	} else if hasLabelCommand {
		// When remove_label is disabled, emit a github-script step that runs get_trigger_label.cjs
		// (via generateGitHubScriptWithRequire) to safely resolve the triggering command name for
		// both label_command and slash_command events and emit a unified `command_name` output
		// (plus a `label_name` alias).
		steps = append(steps, "      - name: Get trigger label name\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.GetTriggerLabelStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		// Pass the pre-computed matched slash-command (if any) so the script can provide a
		// unified command_name for workflows that have both label_command and slash_command.
		if len(data.Command) > 0 {
			steps = append(steps, "        env:\n")
			if preActivationJobCreated {
				steps = append(steps, fmt.Sprintf("          GH_AW_MATCHED_COMMAND: ${{ needs.%s.outputs.%s }}\n", string(constants.PreActivationJobName), constants.MatchedCommandOutput))
			} else {
				steps = append(steps, fmt.Sprintf("          GH_AW_MATCHED_COMMAND: ${{ steps.%s.outputs.%s }}\n", constants.CheckCommandPositionStepID, constants.MatchedCommandOutput))
			}
		}
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")
		steps = append(steps, generateGitHubScriptWithRequire("get_trigger_label.cjs"))
		outputs["label_command"] = fmt.Sprintf("${{ steps.%s.outputs.label_name }}", constants.GetTriggerLabelStepID)
		outputs["command_name"] = fmt.Sprintf("${{ steps.%s.outputs.command_name }}", constants.GetTriggerLabelStepID)
	}

	// If no steps have been added, add a placeholder step to make the job valid
	// This can happen when the activation job is created only for an if condition
	if len(steps) == 0 {
		steps = append(steps, "      - run: echo \"Activation success\"\n")
	}

	// Build the conditional expression that validates activation status and other conditions
	var activationNeeds []string
	var activationCondition string

	// Find custom jobs that depend on pre_activation - these run before activation
	customJobsBeforeActivation := c.getCustomJobsDependingOnPreActivation(data.Jobs)

	// Find custom jobs whose outputs are referenced in the markdown body but have no explicit needs.
	// These jobs must run before activation so their outputs are available when the activation job
	// builds the prompt. Without this, activation would reference their outputs while they haven't
	// run yet, causing actionlint errors and incorrect prompt substitutions.
	promptReferencedJobs := c.getCustomJobsReferencedInPromptWithNoActivationDep(data)
	for _, jobName := range promptReferencedJobs {
		if !slices.Contains(customJobsBeforeActivation, jobName) {
			customJobsBeforeActivation = append(customJobsBeforeActivation, jobName)
			compilerActivationJobLog.Printf("Added '%s' to activation dependencies: referenced in markdown body and has no explicit needs", jobName)
		}
	}

	if preActivationJobCreated {
		// Activation job depends on pre-activation job and checks the "activated" output
		activationNeeds = []string{string(constants.PreActivationJobName)}

		// Also depend on custom jobs that run after pre_activation but before activation
		activationNeeds = append(activationNeeds, customJobsBeforeActivation...)

		activatedExpr := BuildEquals(
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.%s", string(constants.PreActivationJobName), constants.ActivatedOutput)),
			BuildStringLiteral("true"),
		)

		// If there are custom jobs before activation and the if condition references them,
		// include that condition in the activation job's if clause
		if data.If != "" && c.referencesCustomJobOutputs(data.If, data.Jobs) && len(customJobsBeforeActivation) > 0 {
			// Include the custom job output condition in the activation job
			unwrappedIf := stripExpressionWrapper(data.If)
			ifExpr := &ExpressionNode{Expression: unwrappedIf}
			combinedExpr := BuildAnd(activatedExpr, ifExpr)
			activationCondition = RenderCondition(combinedExpr)
		} else if data.If != "" && !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			// Include user's if condition that doesn't reference custom jobs
			unwrappedIf := stripExpressionWrapper(data.If)
			ifExpr := &ExpressionNode{Expression: unwrappedIf}
			combinedExpr := BuildAnd(activatedExpr, ifExpr)
			activationCondition = RenderCondition(combinedExpr)
		} else {
			activationCondition = RenderCondition(activatedExpr)
		}
	} else {
		// No pre-activation check needed
		// Add custom jobs that would run before activation as dependencies
		activationNeeds = append(activationNeeds, customJobsBeforeActivation...)

		if data.If != "" && c.referencesCustomJobOutputs(data.If, data.Jobs) && len(customJobsBeforeActivation) > 0 {
			// Include the custom job output condition
			activationCondition = data.If
		} else if !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			activationCondition = data.If
		}
	}

	// Apply workflow_run repository safety check exclusively to activation job
	// This check is combined with any existing activation condition
	if workflowRunRepoSafety != "" {
		activationCondition = c.combineJobIfConditions(activationCondition, workflowRunRepoSafety)
	}

	// Generate prompt in the activation job (before agent job runs)
	compilerActivationJobLog.Print("Generating prompt in activation job")
	c.generatePromptInActivationJob(&steps, data, preActivationJobCreated, customJobsBeforeActivation)

	// APM packaging is handled by the separate "apm" job that depends on activation.
	// That job packs the bundle and uploads it as an artifact; the agent job then
	// depends on the apm job to download and restore it.

	// Upload aw_info.json and prompt.txt as the activation artifact for the agent job to download.
	// In workflow_call context the artifact is prefixed to avoid name clashes when multiple callers
	// invoke the same reusable workflow within the same parent workflow run.
	compilerActivationJobLog.Print("Adding activation artifact upload step")
	activationArtifactName := artifactPrefixExprForActivationJob(data) + constants.ActivationArtifactName
	steps = append(steps, "      - name: Upload activation artifact\n")
	steps = append(steps, "        if: success()\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          name: %s\n", activationArtifactName))
	steps = append(steps, "          path: |\n")
	steps = append(steps, "            /tmp/gh-aw/aw_info.json\n")
	steps = append(steps, "            /tmp/gh-aw/aw-prompts/prompt.txt\n")
	steps = append(steps, "            /tmp/gh-aw/"+constants.GithubRateLimitsFilename+"\n")
	steps = append(steps, "          if-no-files-found: ignore\n")
	steps = append(steps, "          retention-days: 1\n")

	// Set permissions - activation job always needs contents:read for GitHub API access
	// Also add reaction/comment permissions if reaction or status-comment is configured
	// Also add issues:write permission if lock-for-agent is enabled (for locking issues)
	permsMap := map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead, // Always needed for GitHub API access to check file commits
	}

	// Add actions:read permission when the hash check API step is emitted.
	// check_workflow_timestamp_api.cjs calls github.rest.actions.getWorkflowRun() which
	// requires the actions:read scope. GitHub Actions enforces explicit permissions when
	// any permissions block is present, so we must add it explicitly.
	if !data.StaleCheckDisabled {
		permsMap[PermissionActions] = PermissionRead
	}

	if hasReaction {
		permsMap[PermissionDiscussions] = PermissionWrite
		permsMap[PermissionIssues] = PermissionWrite
		permsMap[PermissionPullRequests] = PermissionWrite
	}

	// Add write permissions if status comments are enabled (even without a reaction).
	// Status comments post to issues, PRs, and discussions, so write access is required.
	// Assigning write to the map is safe here - it does not downgrade existing permissions.
	if hasStatusComment {
		permsMap[PermissionDiscussions] = PermissionWrite
		permsMap[PermissionIssues] = PermissionWrite
		permsMap[PermissionPullRequests] = PermissionWrite
	}

	// Add issues:write permission if lock-for-agent is enabled (even without reaction)
	if data.LockForAgent {
		permsMap[PermissionIssues] = PermissionWrite
	}

	// Add write permissions for label removal when label_command is configured and remove_label is enabled.
	// Only grant the scopes required by the enabled events:
	// - issues/pull_request events need issues:write (PR labels use the issues REST API)
	// - discussion events need discussions:write
	// When a github-app token is configured, the GITHUB_TOKEN permissions are irrelevant
	// for the label removal step (it uses the app token instead), so we skip them.
	// When remove_label is false, no label removal occurs so these permissions are not needed.
	if shouldRemoveLabel && data.ActivationGitHubApp == nil {
		if slices.Contains(filteredLabelEvents, "issues") || slices.Contains(filteredLabelEvents, "pull_request") {
			permsMap[PermissionIssues] = PermissionWrite
		}
		if slices.Contains(filteredLabelEvents, "discussion") {
			permsMap[PermissionDiscussions] = PermissionWrite
		}
	}

	perms := NewPermissionsFromMap(permsMap)
	permissions := perms.RenderToYAML()

	// Set environment if manual-approval is configured
	var environment string
	if data.ManualApproval != "" {
		// Strip ANSI escape codes from manual-approval environment name
		cleanManualApproval := stringutil.StripANSI(data.ManualApproval)
		environment = "environment: " + cleanManualApproval
	}

	// In script mode, explicitly add a cleanup step (mirrors post.js in dev/release/action mode).
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}

	job := &Job{
		Name:                       string(constants.ActivationJobName),
		If:                         activationCondition,
		HasWorkflowRunSafetyChecks: workflowRunRepoSafety != "", // Mark job as having workflow_run safety checks
		RunsOn:                     c.formatFrameworkJobRunsOn(data),
		Permissions:                permissions,
		Environment:                environment,
		Steps:                      steps,
		Outputs:                    outputs,
		Needs:                      activationNeeds, // Depend on pre-activation job if it exists
	}

	return job, nil
}

// generatePromptInActivationJob generates the prompt creation steps and adds them to the activation job
// This creates the prompt.txt file that will be uploaded as an artifact and downloaded by the agent job
// beforeActivationJobs is the list of custom job names that run before (i.e., are dependencies of) activation.
// Passing nil or an empty slice means no custom jobs run before activation; expressions referencing any
// custom job will be filtered out of the substitution step to avoid actionlint errors.
func (c *Compiler) generatePromptInActivationJob(steps *[]string, data *WorkflowData, preActivationJobCreated bool, beforeActivationJobs []string) {
	compilerActivationJobLog.Print("Generating prompt steps in activation job")

	// Use a string builder to collect the YAML
	var yaml strings.Builder

	// Call the existing generatePrompt method to get all the prompt steps
	c.generatePrompt(&yaml, data, preActivationJobCreated, beforeActivationJobs)

	// Append the generated YAML content as a single string to steps
	yamlContent := yaml.String()
	*steps = append(*steps, yamlContent)

	compilerActivationJobLog.Print("Prompt generation steps added to activation job")
}

// generateResolveHostRepoStep generates a step that resolves the platform (host) repository
// for the activation job checkout by inspecting GITHUB_WORKFLOW_REF at runtime.
//
// This step replaces the previous compile-time expression
//
//	github.event_name == 'workflow_call' && github.action_repository || github.repository
//
// which only worked when the outermost trigger was workflow_call. For event-driven relays
// (e.g. on: issue_comment, on: push) the event_name is the native event, so the old
// expression always fell back to github.repository (the caller's repo), causing the
// activation job to check out the wrong repository.
//
// GITHUB_WORKFLOW_REF always contains the path of the currently executing workflow file
// (owner/repo/.github/workflows/file.yml@ref), regardless of the triggering event.
// Comparing its owner/repo prefix with GITHUB_REPOSITORY reliably detects cross-repo
// invocations for all relay patterns.
func (c *Compiler) generateResolveHostRepoStep() string {
	var step strings.Builder
	step.WriteString("      - name: Resolve host repo for activation checkout\n")
	step.WriteString("        id: resolve-host-repo\n")
	step.WriteString(fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	step.WriteString("        with:\n")
	step.WriteString("          script: |\n")
	step.WriteString(generateGitHubScriptWithRequire("resolve_host_repo.cjs"))
	return step.String()
}

// generateCheckoutGitHubFolderForActivation generates the checkout step for .github and .agents folders
// specifically for the activation job. Unlike generateCheckoutGitHubFolder, this method doesn't skip
// the checkout when the agent job will have a full repository checkout, because the activation job
// runs before the agent job and needs independent access to workflow files for runtime imports during
// prompt generation.
func (c *Compiler) generateCheckoutGitHubFolderForActivation(data *WorkflowData) []string {
	// Check if action-tag is specified - if so, skip checkout
	if data != nil && data.Features != nil {
		if actionTagVal, exists := data.Features["action-tag"]; exists {
			if actionTagStr, ok := actionTagVal.(string); ok && actionTagStr != "" {
				// action-tag is set, no checkout needed
				compilerActivationJobLog.Print("Skipping .github checkout in activation: action-tag specified")
				return nil
			}
		}
	}

	// Note: We don't check data.Permissions for contents read access here because
	// the activation job ALWAYS gets contents:read added to its permissions (see buildActivationJob
	// around line 720). The workflow's original permissions may not include contents:read,
	// but the activation job will always have it for GitHub API access and runtime imports.
	// The agent job uses only the user-specified permissions (no automatic contents:read augmentation).

	// For workflow_call triggers, checkout the callee (platform) repository using the target_repo
	// output from the resolve-host-repo step. That step parses GITHUB_WORKFLOW_REF at runtime to
	// determine the platform repo, correctly handling event-driven relays where event_name is not
	// 'workflow_call' (e.g. on: issue_comment, on: push).
	//
	// Skip when inlined-imports is enabled: content is embedded at compile time and no
	// runtime-import macros are used, so the callee's .md files are not needed at runtime.
	// In dev mode, actions/setup is referenced via a local workspace path (./actions/setup),
	// so it must be included in the sparse-checkout to preserve it for the post step.
	// In release/script/action modes, the action is in the runner cache and not the workspace.
	var extraPaths []string
	if c.actionMode.IsDev() {
		compilerActivationJobLog.Print("Dev mode: adding actions/setup to sparse-checkout to preserve local action post step")
		extraPaths = append(extraPaths, "actions/setup")
	}

	cm := NewCheckoutManager(nil)
	if data != nil && hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		compilerActivationJobLog.Print("Adding cross-repo-aware .github checkout for workflow_call trigger")
		cm.SetCrossRepoTargetRepo("${{ steps.resolve-host-repo.outputs.target_repo }}")
		cm.SetCrossRepoTargetRef("${{ steps.resolve-host-repo.outputs.target_ref }}")
		return cm.GenerateGitHubFolderCheckoutStep(
			cm.GetCrossRepoTargetRepo(),
			cm.GetCrossRepoTargetRef(),
			GetActionPin,
			extraPaths...,
		)
	}

	// For activation job, always add sparse checkout of .github and .agents folders
	// This is needed for runtime imports during prompt generation
	// sparse-checkout-cone-mode: true ensures subdirectories under .github/ are recursively included
	compilerActivationJobLog.Print("Adding .github and .agents sparse checkout in activation job")
	return cm.GenerateGitHubFolderCheckoutStep("", "", GetActionPin, extraPaths...)
}
