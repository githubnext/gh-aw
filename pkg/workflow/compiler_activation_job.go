package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
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
	steps = append(steps, c.generateSetupStep(setupActionRef, SetupActionDestination, false)...)

	// Generate agentic run info immediately after setup so aw_info.json is ready as early as possible.
	// This ensures it is available for prompt generation and can be uploaded together with prompt.txt.
	engine, err := c.getAgenticEngine(data.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to get agentic engine: %w", err)
	}
	compilerActivationJobLog.Print("Generating aw_info step in activation job (first step after setup)")
	var awInfoYaml strings.Builder
	c.generateCreateAwInfo(&awInfoYaml, data, engine)
	steps = append(steps, awInfoYaml.String())
	// Expose the model output from the activation job so downstream jobs can reference it
	outputs["model"] = "${{ steps.generate_aw_info.outputs.model }}"

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
	// This step only runs when secret validation fails in a workflow_call context,
	// providing actionable guidance to the caller team about configuring secrets.
	if hasWorkflowCallTrigger(data.On) {
		compilerActivationJobLog.Print("Adding cross-repo setup guidance step for workflow_call trigger")
		steps = append(steps, "      - name: Cross-repo setup guidance\n")
		steps = append(steps, "        if: failure() && github.event_name == 'workflow_call'\n")
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

	// Mint a single activation app token upfront if a GitHub App is configured and either
	// the reaction or status-comment step will need it. This avoids minting multiple tokens.
	hasReaction := data.AIReaction != "" && data.AIReaction != "none"
	hasStatusComment := data.StatusComment != nil && *data.StatusComment
	if data.ActivationGitHubApp != nil && (hasReaction || hasStatusComment) {
		// Build the combined permissions needed for reactions and/or status comments
		appPerms := NewPermissions()
		appPerms.Set(PermissionIssues, PermissionWrite)
		appPerms.Set(PermissionPullRequests, PermissionWrite)
		appPerms.Set(PermissionDiscussions, PermissionWrite)
		steps = append(steps, c.buildActivationAppTokenMintStep(data.ActivationGitHubApp, appPerms)...)
	}

	// Add reaction step for immediate feedback.
	// This runs in the activation job so it can use any configured github-token or github-app.
	if hasReaction {
		reactionCondition := BuildReactionCondition()

		steps = append(steps, fmt.Sprintf("      - name: Add %s reaction for immediate feedback\n", data.AIReaction))
		steps = append(steps, "        id: react\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", reactionCondition.Render()))
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

	// Add timestamp check for lock file vs source file using GitHub API
	// No checkout step needed - uses GitHub API to check commit times
	steps = append(steps, "      - name: Check workflow file timestamps\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_FILE: \"%s\"\n", lockFilename))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")
	steps = append(steps, generateGitHubScriptWithRequire("check_workflow_timestamp_api.cjs"))

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
			steps = append(steps, fmt.Sprintf("          GH_AW_ALLOWED_BOTS: %s\n", strings.Join(data.Bots, ",")))
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
		steps = append(steps, fmt.Sprintf("        if: %s\n", reactionCondition.Render()))
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
		steps = append(steps, fmt.Sprintf("        if: %s\n", lockCondition.Render()))
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
		if !sliceutil.Contains(customJobsBeforeActivation, jobName) {
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
			activationCondition = combinedExpr.Render()
		} else if data.If != "" && !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			// Include user's if condition that doesn't reference custom jobs
			unwrappedIf := stripExpressionWrapper(data.If)
			ifExpr := &ExpressionNode{Expression: unwrappedIf}
			combinedExpr := BuildAnd(activatedExpr, ifExpr)
			activationCondition = combinedExpr.Render()
		} else {
			activationCondition = activatedExpr.Render()
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

	// Upload aw_info.json and prompt.txt as the activation artifact for the agent job to download
	compilerActivationJobLog.Print("Adding activation artifact upload step")
	steps = append(steps, "      - name: Upload activation artifact\n")
	steps = append(steps, "        if: success()\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: activation\n")
	steps = append(steps, "          path: |\n")
	steps = append(steps, "            /tmp/gh-aw/aw_info.json\n")
	steps = append(steps, "            /tmp/gh-aw/aw-prompts/prompt.txt\n")
	steps = append(steps, "          retention-days: 1\n")

	// Set permissions - activation job always needs contents:read for GitHub API access
	// Also add reaction/comment permissions if reaction or status-comment is configured
	// Also add issues:write permission if lock-for-agent is enabled (for locking issues)
	permsMap := map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead, // Always needed for GitHub API access to check file commits
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

	perms := NewPermissionsFromMap(permsMap)
	permissions := perms.RenderToYAML()

	// Set environment if manual-approval is configured
	var environment string
	if data.ManualApproval != "" {
		// Strip ANSI escape codes from manual-approval environment name
		cleanManualApproval := stringutil.StripANSI(data.ManualApproval)
		environment = "environment: " + cleanManualApproval
	}

	job := &Job{
		Name:                       string(constants.ActivationJobName),
		If:                         activationCondition,
		HasWorkflowRunSafetyChecks: workflowRunRepoSafety != "", // Mark job as having workflow_run safety checks
		RunsOn:                     c.formatSafeOutputsRunsOn(data.SafeOutputs),
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

	// For workflow_call triggers, checkout the callee repository using a conditional expression.
	// github.action_repository points to the callee (platform) repo during workflow_call;
	// for other event types the explicit event_name check short-circuits to falsy and we
	// fall back to github.repository. This supports mixed triggers (e.g., workflow_call + workflow_dispatch).
	//
	// Skip when inlined-imports is enabled: content is embedded at compile time and no
	// runtime-import macros are used, so the callee's .md files are not needed at runtime.
	cm := NewCheckoutManager(nil)
	if data != nil && hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
		compilerActivationJobLog.Print("Adding cross-repo-aware .github checkout for workflow_call trigger")
		return cm.GenerateGitHubFolderCheckoutStep(
			"${{ github.event_name == 'workflow_call' && github.action_repository || github.repository }}",
			GetActionPin,
		)
	}

	// For activation job, always add sparse checkout of .github and .agents folders
	// This is needed for runtime imports during prompt generation
	// sparse-checkout-cone-mode: true ensures subdirectories under .github/ are recursively included
	compilerActivationJobLog.Print("Adding .github and .agents sparse checkout in activation job")
	return cm.GenerateGitHubFolderCheckoutStep("", GetActionPin)
}
