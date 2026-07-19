package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var concurrencyLog = logger.New("workflow:concurrency")

// GenerateConcurrencyConfig generates the concurrency configuration for a workflow
// based on its trigger types and characteristics.
func GenerateConcurrencyConfig(workflowData *WorkflowData, isCommandTrigger bool) string {
	concurrencyLog.Printf("Generating concurrency config: isCommandTrigger=%v", isCommandTrigger)

	// Don't override if already set
	if workflowData.Concurrency != "" {
		concurrencyLog.Print("Using existing concurrency configuration from workflow data")
		return workflowData.Concurrency
	}

	// Build concurrency group keys using the original workflow-specific logic
	keys := buildConcurrencyGroupKeys(workflowData, isCommandTrigger)
	groupValue := strings.Join(keys, "-")
	concurrencyLog.Printf("Built concurrency group: %s", groupValue)

	// Build the concurrency configuration
	concurrencyConfig := fmt.Sprintf("concurrency:\n  group: \"%s\"", groupValue)

	// Add cancel-in-progress if appropriate
	if shouldEnableCancelInProgress(workflowData, isCommandTrigger) {
		concurrencyLog.Print("Enabling cancel-in-progress for concurrency group")
		concurrencyConfig += "\n  cancel-in-progress: true"
	}

	return concurrencyConfig
}

// GenerateJobConcurrencyConfig generates the agent concurrency configuration
// for the agent job based on engine.concurrency field
func GenerateJobConcurrencyConfig(workflowData *WorkflowData) string {
	concurrencyLog.Print("Generating job-level concurrency config")

	// If concurrency is explicitly configured in engine, use it
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Concurrency != "" {
		concurrencyLog.Print("Using engine-configured concurrency")
		return workflowData.EngineConfig.Concurrency
	}

	// Check if this workflow has special trigger handling (issues, PRs, discussions, push, command,
	// or workflow_dispatch-only). For these cases, no default concurrency should be applied at agent level
	if hasSpecialTriggers(workflowData) {
		concurrencyLog.Print("Workflow has special triggers, skipping default job concurrency")
		return ""
	}

	// For remaining generic triggers like schedule, apply default concurrency
	// Pattern: gh-aw-{engine-id}-${{ github.workflow }}
	engineID := ""
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.ID != "" {
		engineID = workflowData.EngineConfig.ID
	}

	if engineID == "" {
		// If no engine ID is available, skip default concurrency
		return ""
	}

	// Build the default concurrency configuration.
	// For workflow_call workers, github.workflow resolves to the caller workflow
	// name, so use the compile-time workflow ID to avoid cross-worker collisions.
	groupValue := fmt.Sprintf("gh-aw-%s-${{ github.workflow }}", engineID)
	if hasWorkflowCallTrigger(workflowData.On) {
		groupValue = fmt.Sprintf("gh-aw-%s-%s", engineID, workflowCallBaseKey(workflowData))
	}
	// If the user specified a job-discriminator, append it so that concurrent
	// runs with different inputs (fan-out pattern) do not share the same group.
	if workflowData.ConcurrencyJobDiscriminator != "" {
		concurrencyLog.Printf("Appending job discriminator to job-level concurrency group: %s", workflowData.ConcurrencyJobDiscriminator)
		groupValue = fmt.Sprintf("%s-%s", groupValue, workflowData.ConcurrencyJobDiscriminator)
	}
	concurrencyConfig := fmt.Sprintf("concurrency:\n  group: \"%s\"", groupValue)
	if isGroupConcurrencyQueueEnabled(workflowData) {
		concurrencyConfig += "\n  queue: max"
	}

	return concurrencyConfig
}

// hasSpecialTriggers checks if the workflow has special trigger types that require
// workflow-level concurrency handling (issues, PRs, discussions, push, command,
// slash_command, or workflow_dispatch-only)
func hasSpecialTriggers(workflowData *WorkflowData) bool {
	on := workflowData.On
	return isIssueWorkflow(on) ||
		isPullRequestWorkflow(on) ||
		isDiscussionWorkflow(on) ||
		isPushWorkflow(on) ||
		isSlashCommandWorkflow(on) ||
		// workflow_dispatch-only represents explicit user intent — top-level concurrency is sufficient
		isWorkflowDispatchOnly(on)
}

// isPullRequestWorkflow checks if a workflow's "on" section contains pull_request triggers
func isPullRequestWorkflow(on string) bool {
	return strings.Contains(on, "pull_request")
}

// isIssueWorkflow checks if a workflow's "on" section contains issue-related triggers
func isIssueWorkflow(on string) bool {
	return strings.Contains(on, "issues") || strings.Contains(on, "issue_comment")
}

// isDiscussionWorkflow checks if a workflow's "on" section contains discussion-related triggers
func isDiscussionWorkflow(on string) bool {
	return strings.Contains(on, "discussion")
}

// isWorkflowDispatchOnly returns true when workflow_dispatch is the only trigger in the
// "on" section, indicating the workflow is always started by explicit user intent.
// It handles both rendered YAML (standard GitHub Actions events) and input YAML
// (which may contain synthetic events like slash_command before they are expanded).
func isWorkflowDispatchOnly(on string) bool {
	if !strings.Contains(on, "workflow_dispatch") {
		return false
	}
	// If any other common trigger is present as a YAML key, this is not a
	// workflow_dispatch-only workflow. We check for the trigger name followed by
	// ':' (YAML key in object form) or as the sole inline value to avoid false
	// matches from input parameter names (e.g., "push_branch" ≠ "push" trigger).
	// slash_command is included here because it is a synthetic event that expands
	// to issue_comment + workflow_dispatch at compile time; its presence means the
	// workflow is not triggered solely by explicit user dispatch.
	otherTriggers := []string{
		"push", "pull_request", "pull_request_review", "pull_request_review_comment",
		"pull_request_target", "issues", "issue_comment", "discussion",
		"discussion_comment", "schedule", "repository_dispatch", "workflow_run",
		"create", "delete", "release", "deployment", "fork", "gollum",
		"label", "milestone", "page_build", "public", "registry_package",
		"status", "watch", "merge_group", "check_run", "check_suite",
		"slash_command",
	}
	for _, trigger := range otherTriggers {
		// Trigger in object format: "push:" / "  push:"
		if strings.Contains(on, trigger+":") {
			return false
		}
		// Trigger in inline format: "on: push" (no colon, trigger is the last token)
		if strings.HasSuffix(strings.TrimSpace(on), " "+trigger) {
			return false
		}
	}
	return true
}

// isPushWorkflow checks if a workflow's "on" section contains push triggers
func isPushWorkflow(on string) bool {
	return strings.Contains(on, "push")
}

// isSlashCommandWorkflow checks if a workflow's "on" section contains the slash_command
// synthetic trigger. slash_command is an input-level event that expands to
// issue_comment + workflow_dispatch at compile time. Detecting it here allows
// the concurrency helpers to produce correct results even when they are called
// with the pre-rendered "on" YAML (before the event expansion has taken place).
func isSlashCommandWorkflow(on string) bool {
	return strings.Contains(on, "slash_command")
}

// entityConcurrencyKey builds a ${{ ... }} concurrency-group expression for entity-number
// based workflows. primaryParts are the event-number identifiers (e.g.,
// "github.event.pull_request.number"), tailParts are the trailing fallbacks (e.g.,
// "github.ref", "github.run_id"). When hasItemNumber is true, "inputs.item_number" is
// inserted between the primary identifiers and the tail, providing a stable per-item
// key for manual workflow_dispatch runs triggered via the label trigger shorthand.
func entityConcurrencyKey(primaryParts []string, tailParts []string, hasItemNumber bool) string {
	parts := make([]string, 0, safeAllocationCapacity(len(primaryParts), len(tailParts), 1))
	parts = append(parts, primaryParts...)
	if hasItemNumber {
		parts = append(parts, "inputs.item_number")
	}
	parts = append(parts, tailParts...)
	return "${{ " + strings.Join(parts, " || ") + " }}"
}

// botIsolatedConcurrencyKey builds a ${{ ... }} concurrency-group expression that
// routes GitHub App bot-triggered runs to a unique github.run_id key, preventing
// passive bot-authored comment events from colliding with the primary run's group.
// When contains(github.actor, '[bot]') is true, the expression short-circuits to
// github.run_id so that bot-triggered runs never share a group with human runs.
func botIsolatedConcurrencyKey(primaryParts []string, tailParts []string, hasItemNumber bool) string {
	parts := make([]string, 0, safeAllocationCapacity(len(primaryParts), len(tailParts), 2))
	// Prepend the bot-actor isolation check: bot runs always get a unique key
	parts = append(parts, "contains(github.actor, '[bot]') && github.run_id")
	parts = append(parts, primaryParts...)
	if hasItemNumber {
		parts = append(parts, "inputs.item_number")
	}
	parts = append(parts, tailParts...)
	return "${{ " + strings.Join(parts, " || ") + " }}"
}

// hasIssueCommentTrigger returns true when the workflow has any trigger that uses
// issue_comment events: explicit issue_comment, slash_command, or command triggers.
// These are the only triggers where a GitHub App-authored comment on an issue can
// re-enter the same workflow and match the primary run's workflow-level concurrency
// group, creating the self-cancellation risk described in the issue.
func hasIssueCommentTrigger(workflowData *WorkflowData) bool {
	on := workflowData.On
	return strings.Contains(on, "issue_comment") ||
		strings.Contains(on, "slash_command") ||
		len(workflowData.Command) > 0
}

// hasBotSelfCancelRisk returns true when the workflow's auto-generated concurrency
// configuration can be collided with by a passive GitHub App bot-authored event.
// The dangerous combination is: issue_comment triggers + GitHub App safe-outputs.
// When this combination is present, App-authored comments posted by safe-outputs can
// re-trigger the workflow and resolve to the same workflow-level concurrency group
// as the originating run, causing cancel-in-progress to cancel the original run.
func hasBotSelfCancelRisk(workflowData *WorkflowData) bool {
	if workflowData.SafeOutputs == nil || workflowData.SafeOutputs.GitHubApp == nil {
		return false
	}
	return hasIssueCommentTrigger(workflowData)
}

// workflowCallBaseKey returns the stable workflow identifier to use for
// workflow_call runs.
func workflowCallBaseKey(workflowData *WorkflowData) string {
	if workflowData.WorkflowID != "" {
		return workflowData.WorkflowID
	}
	concurrencyLog.Print("WARNING: workflow_call trigger without WorkflowID; using github.workflow fallback")
	return "${{ github.workflow }}"
}

// buildConcurrencyGroupKeys builds an array of keys for the concurrency group
func buildConcurrencyGroupKeys(workflowData *WorkflowData, isCommandTrigger bool) []string {
	keys := []string{"gh-aw", concurrencyWorkflowKey(workflowData)}
	hasItemNumber := workflowData.HasDispatchItemNumber
	botRisk := hasBotSelfCancelRisk(workflowData)
	if botRisk {
		concurrencyLog.Print("Bot self-cancel risk detected: applying bot-actor isolation to concurrency group key")
	}
	if entityKey := buildEntityConcurrencyKey(workflowData, isCommandTrigger, hasItemNumber, botRisk); entityKey != "" {
		keys = append(keys, entityKey)
	}
	if hasItemNumber {
		keys = append(keys, "${{ github.event.label.name || github.run_id }}")
	}
	return keys
}

func concurrencyWorkflowKey(workflowData *WorkflowData) string {
	if hasWorkflowCallTrigger(workflowData.On) {
		return workflowCallBaseKey(workflowData) + "-${{ github.run_id }}"
	}
	return "${{ github.workflow }}"
}

func buildEntityConcurrencyKey(workflowData *WorkflowData, isCommandTrigger, hasItemNumber, botRisk bool) string {
	entityKey := func(primaryParts []string, tailParts []string) string {
		if botRisk {
			return botIsolatedConcurrencyKey(primaryParts, tailParts, hasItemNumber)
		}
		return entityConcurrencyKey(primaryParts, tailParts, hasItemNumber)
	}
	switch {
	case isCommandTrigger || isSlashCommandWorkflow(workflowData.On):
		return commandConcurrencyKey(botRisk)
	case isPullRequestWorkflow(workflowData.On) && isIssueWorkflow(workflowData.On):
		concurrencyLog.Print("Building concurrency key for mixed PR+issue workflow")
		return entityKey([]string{"github.event.issue.number", "github.event.pull_request.number"}, []string{"github.run_id"})
	case isPullRequestWorkflow(workflowData.On) && isDiscussionWorkflow(workflowData.On):
		return entityConcurrencyKey([]string{"github.event.pull_request.number", "github.event.discussion.number"}, []string{"github.run_id"}, hasItemNumber)
	case isIssueWorkflow(workflowData.On) && isDiscussionWorkflow(workflowData.On):
		return entityKey([]string{"github.event.issue.number", "github.event.discussion.number"}, []string{"github.run_id"})
	case isPullRequestWorkflow(workflowData.On):
		concurrencyLog.Print("Building concurrency key for PR workflow")
		return entityConcurrencyKey([]string{"github.event.pull_request.number"}, []string{"github.ref", "github.run_id"}, hasItemNumber)
	case isIssueWorkflow(workflowData.On):
		concurrencyLog.Print("Building concurrency key for issue workflow")
		return entityKey([]string{"github.event.issue.number"}, []string{"github.run_id"})
	case isDiscussionWorkflow(workflowData.On):
		return entityConcurrencyKey([]string{"github.event.discussion.number"}, []string{"github.run_id"}, hasItemNumber)
	case isPushWorkflow(workflowData.On):
		concurrencyLog.Print("Building concurrency key for push workflow")
		return "${{ github.ref || github.run_id }}"
	default:
		return ""
	}
}

func commandConcurrencyKey(botRisk bool) string {
	concurrencyLog.Print("Building concurrency key for command/slash_command workflow")
	if botRisk {
		return "${{ contains(github.actor, '[bot]') && github.run_id || github.event.issue.number || github.event.pull_request.number || github.run_id }}"
	}
	return "${{ github.event.issue.number || github.event.pull_request.number || github.run_id }}"
}

// shouldEnableCancelInProgress determines if cancel-in-progress should be enabled
func shouldEnableCancelInProgress(workflowData *WorkflowData, isCommandTrigger bool) bool {
	// Never enable cancellation for command workflows
	if isCommandTrigger {
		concurrencyLog.Print("cancel-in-progress disabled: command trigger workflow")
		return false
	}

	// Enable cancellation for pull request workflows (including mixed workflows)
	result := isPullRequestWorkflow(workflowData.On)
	concurrencyLog.Printf("cancel-in-progress=%v for workflow on=%s", result, workflowData.On)
	return result
}
