package workflow

import (
	"fmt"
	"strings"
)

// GenerateConcurrencyConfig generates the concurrency configuration for a workflow
// based on its trigger types and characteristics.
func GenerateConcurrencyConfig(workflowData *WorkflowData, isCommandTrigger bool) string {
	// Don't override if already set
	if workflowData.Concurrency != "" {
		return workflowData.Concurrency
	}

	// Build concurrency group keys using the original workflow-specific logic
	keys := buildConcurrencyGroupKeys(workflowData, isCommandTrigger)
	groupValue := strings.Join(keys, "-")

	// Build the concurrency configuration
	concurrencyConfig := fmt.Sprintf("concurrency:\n  group: \"%s\"", groupValue)

	// Add cancel-in-progress if appropriate
	if shouldEnableCancelInProgress(workflowData, isCommandTrigger) {
		concurrencyConfig += "\n  cancel-in-progress: true"
	}

	return concurrencyConfig
}

// GenerateJobConcurrencyConfig generates the agent concurrency configuration
// for the agent job based on engine.concurrency field
func GenerateJobConcurrencyConfig(workflowData *WorkflowData) string {
	// If concurrency is explicitly configured in engine, use it
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Concurrency != "" {
		return workflowData.EngineConfig.Concurrency
	}

	// No default concurrency - return empty string
	return ""
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
