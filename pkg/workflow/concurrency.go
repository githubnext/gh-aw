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

	// Build the default concurrency configuration
	groupValue := fmt.Sprintf("gh-aw-%s-${{ github.workflow }}", engineID)
	concurrencyConfig := fmt.Sprintf("concurrency:\n  group: \"%s\"", groupValue)

	return concurrencyConfig
}

// hasSpecialTriggers checks if the workflow has special trigger types that require
// workflow-level concurrency handling (issues, PRs, discussions, push, command,
// or workflow_dispatch-only)
func hasSpecialTriggers(workflowData *WorkflowData) bool {
	// Check for specific trigger types that have special concurrency handling
	on := workflowData.On

	// Check for issue-related triggers
	if isIssueWorkflow(on) {
		return true
	}

	// Check for pull request triggers
	if isPullRequestWorkflow(on) {
		return true
	}

	// Check for discussion triggers
	if isDiscussionWorkflow(on) {
		return true
	}

	// Check for push triggers
	if isPushWorkflow(on) {
		return true
	}

	// workflow_dispatch-only workflows represent explicit user intent, so the
	// top-level workflow concurrency group is sufficient – no engine-level group needed
	if isWorkflowDispatchOnly(on) {
		return true
	}

	// If none of the special triggers are detected, return false
	// This means other generic triggers (e.g. schedule) will get default concurrency
	return false
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
func isWorkflowDispatchOnly(on string) bool {
	if !strings.Contains(on, "workflow_dispatch") {
		return false
	}
	// If any other common trigger is present as a YAML key, this is not a
	// workflow_dispatch-only workflow. We check for the trigger name followed by
	// ':' (YAML key in object form) or as the sole inline value to avoid false
	// matches from input parameter names (e.g., "push_branch" ≠ "push" trigger).
	otherTriggers := []string{
		"push", "pull_request", "pull_request_review", "pull_request_review_comment",
		"pull_request_target", "issues", "issue_comment", "discussion",
		"discussion_comment", "schedule", "repository_dispatch", "workflow_run",
		"create", "delete", "release", "deployment", "fork", "gollum",
		"label", "milestone", "page_build", "public", "registry_package",
		"status", "watch", "merge_group", "check_run", "check_suite",
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

// buildConcurrencyGroupKeys builds an array of keys for the concurrency group
func buildConcurrencyGroupKeys(workflowData *WorkflowData, isCommandTrigger bool) []string {
	keys := []string{"gh-aw", "${{ github.workflow }}"}

	if isCommandTrigger {
		// For command workflows: use issue/PR number
		keys = append(keys, "${{ github.event.issue.number || github.event.pull_request.number }}")
	} else if isPullRequestWorkflow(workflowData.On) && isIssueWorkflow(workflowData.On) {
		// Mixed workflows with both issue and PR triggers: use issue/PR number
		keys = append(keys, "${{ github.event.issue.number || github.event.pull_request.number }}")
	} else if isPullRequestWorkflow(workflowData.On) && isDiscussionWorkflow(workflowData.On) {
		// Mixed workflows with PR and discussion triggers: use PR/discussion number
		keys = append(keys, "${{ github.event.pull_request.number || github.event.discussion.number }}")
	} else if isIssueWorkflow(workflowData.On) && isDiscussionWorkflow(workflowData.On) {
		// Mixed workflows with issue and discussion triggers: use issue/discussion number
		keys = append(keys, "${{ github.event.issue.number || github.event.discussion.number }}")
	} else if isPullRequestWorkflow(workflowData.On) {
		// Pure PR workflows: use PR number if available, otherwise fall back to ref for compatibility
		keys = append(keys, "${{ github.event.pull_request.number || github.ref }}")
	} else if isIssueWorkflow(workflowData.On) {
		// Issue workflows: use issue number
		keys = append(keys, "${{ github.event.issue.number }}")
	} else if isDiscussionWorkflow(workflowData.On) {
		// Discussion workflows: use discussion number
		keys = append(keys, "${{ github.event.discussion.number }}")
	} else if isPushWorkflow(workflowData.On) {
		// Push workflows: use ref to differentiate between branches
		keys = append(keys, "${{ github.ref }}")
	}

	return keys
}

// shouldEnableCancelInProgress determines if cancel-in-progress should be enabled
func shouldEnableCancelInProgress(workflowData *WorkflowData, isCommandTrigger bool) bool {
	// Never enable cancellation for command workflows
	if isCommandTrigger {
		return false
	}

	// Enable cancellation for pull request workflows (including mixed workflows)
	return isPullRequestWorkflow(workflowData.On)
}
