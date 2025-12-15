package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// This file contains the buildActivationJob function for building the activation job.
// The activation job acts as a preamble barrier that handles runtime conditions, timestamp checks,
// text computation, AI reactions, and workflow activation logic.

// buildActivationJob creates the preamble activation job that acts as a barrier for runtime conditions.
// The workflow_run repository safety check is applied exclusively to this job.
// This job handles:
// 1. Timestamp validation (lock file vs source file)
// 2. Text output computation (for compute-text action)
// 3. AI reactions (emoji reactions on triggering items)
// 4. Issue locking (for lock-for-agent feature)
func (c *Compiler) buildActivationJob(data *WorkflowData, preActivationJobCreated bool, workflowRunRepoSafety string, lockFilename string) (*Job, error) {
	outputs := map[string]string{}
	var steps []string

	// Team member check is now handled by the separate check_membership job
	// No inline role checks needed in the task job anymore

	// Add timestamp check for lock file vs source file using GitHub API
	// No checkout step needed - uses GitHub API to check commit times
	steps = append(steps, "      - name: Check workflow file timestamps\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_FILE: \"%s\"\n", lockFilename))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add the JavaScript script with proper indentation (using API-based version)
	formattedScript := FormatJavaScriptForYAML(checkWorkflowTimestampAPIScript)
	steps = append(steps, formattedScript...)

	// Use inlined compute-text script only if needed (no shared action)
	if data.NeedsTextOutput {
		steps = append(steps, "      - name: Compute current body text\n")
		steps = append(steps, "        id: compute-text\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Inline the JavaScript directly instead of using shared action
		steps = append(steps, FormatJavaScriptForYAML(getComputeTextScript())...)

		// Set up outputs
		outputs["text"] = "${{ steps.compute-text.outputs.text }}"
	}

	// Add reaction step if ai-reaction is configured and not "none"
	if data.AIReaction != "" && data.AIReaction != "none" {
		reactionCondition := buildReactionCondition()

		steps = append(steps, fmt.Sprintf("      - name: Add %s reaction to the triggering item\n", data.AIReaction))
		steps = append(steps, "        id: react\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", reactionCondition.Render()))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

		// Add environment variables
		steps = append(steps, "        env:\n")
		// Quote the reaction value to prevent YAML interpreting +1/-1 as integers
		steps = append(steps, fmt.Sprintf("          GH_AW_REACTION: %q\n", data.AIReaction))
		if data.Command != "" {
			steps = append(steps, fmt.Sprintf("          GH_AW_COMMAND: %s\n", data.Command))
		}
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
				compilerJobsLog.Printf("Warning: failed to serialize messages config for activation job: %v", err)
			} else if messagesJSON != "" {
				steps = append(steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
			}
		}

		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add each line of the script with proper indentation (bundled version with messages.cjs)
		formattedScript := FormatJavaScriptForYAML(getAddReactionAndEditCommentScript())
		steps = append(steps, formattedScript...)

		// Add reaction outputs
		outputs["reaction_id"] = "${{ steps.react.outputs.reaction-id }}"
		outputs["comment_id"] = "${{ steps.react.outputs.comment-id }}"
		outputs["comment_url"] = "${{ steps.react.outputs.comment-url }}"
		outputs["comment_repo"] = "${{ steps.react.outputs.comment-repo }}"
	}

	// Add lock step if lock-for-agent is enabled
	if data.LockForAgent {
		// Build condition: only lock if event type is 'issues' or 'issue_comment'
		// lock-for-agent can be configured under on.issues or on.issue_comment
		// For issue_comment events, context.issue.number automatically resolves to the parent issue
		lockCondition := buildOr(
			BuildEventTypeEquals("issues"),
			BuildEventTypeEquals("issue_comment"),
		)

		steps = append(steps, "      - name: Lock issue for agent workflow\n")
		steps = append(steps, "        id: lock-issue\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", lockCondition.Render()))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add the lock-issue script
		formattedScript := FormatJavaScriptForYAML(lockIssueScript)
		steps = append(steps, formattedScript...)

		// Add output for tracking if issue was locked
		outputs["issue_locked"] = "${{ steps.lock-issue.outputs.locked }}"

		// Add lock message to reaction comment if reaction is enabled
		if data.AIReaction != "" && data.AIReaction != "none" {
			compilerJobsLog.Print("Adding lock notification to reaction message")
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

	// If no steps have been added, add a dummy step to make the job valid
	// This can happen when the activation job is created only for an if condition
	if len(steps) == 0 {
		steps = append(steps, "      - run: echo \"Activation success\"\n")
	}

	// Build the conditional expression that validates activation status and other conditions
	var activationNeeds []string
	var activationCondition string

	// Find custom jobs that depend on pre_activation - these run before activation
	customJobsBeforeActivation := c.getCustomJobsDependingOnPreActivation(data.Jobs)

	if preActivationJobCreated {
		// Activation job depends on pre-activation job and checks the "activated" output
		activationNeeds = []string{constants.PreActivationJobName}

		// Also depend on custom jobs that run after pre_activation but before activation
		activationNeeds = append(activationNeeds, customJobsBeforeActivation...)

		activatedExpr := BuildEquals(
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.%s", constants.PreActivationJobName, constants.ActivatedOutput)),
			BuildStringLiteral("true"),
		)

		// If there are custom jobs before activation and the if condition references them,
		// include that condition in the activation job's if clause
		if data.If != "" && c.referencesCustomJobOutputs(data.If, data.Jobs) && len(customJobsBeforeActivation) > 0 {
			// Include the custom job output condition in the activation job
			unwrappedIf := stripExpressionWrapper(data.If)
			ifExpr := &ExpressionNode{Expression: unwrappedIf}
			combinedExpr := buildAnd(activatedExpr, ifExpr)
			activationCondition = combinedExpr.Render()
		} else if data.If != "" && !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			// Include user's if condition that doesn't reference custom jobs
			unwrappedIf := stripExpressionWrapper(data.If)
			ifExpr := &ExpressionNode{Expression: unwrappedIf}
			combinedExpr := buildAnd(activatedExpr, ifExpr)
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

	// Set permissions - activation job always needs contents:read for GitHub API access
	// Also add reaction permissions if reaction is configured and not "none"
	// Also add issues:write permission if lock-for-agent is enabled (for locking issues)
	permsMap := map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead, // Always needed for GitHub API access to check file commits
	}

	if data.AIReaction != "" && data.AIReaction != "none" {
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
		environment = fmt.Sprintf("environment: %s", data.ManualApproval)
	}

	job := &Job{
		Name:                       constants.ActivationJobName,
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
