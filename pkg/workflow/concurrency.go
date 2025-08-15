package workflow

import (
	"strings"
)

// GenerateConcurrencyConfig generates the concurrency configuration for a workflow
// based on its trigger types and characteristics.
func GenerateConcurrencyConfig(workflowData *WorkflowData, isAliasTrigger bool) string {
	// Don't override if already set
	if workflowData.Concurrency != "" {
		return workflowData.Concurrency
	}

	// Generate concurrency configuration based on workflow type
	// Note: Check alias trigger first since alias workflows also contain pull_request events
	if isAliasTrigger {
		// For alias workflows: use issue/PR number for concurrency but do NOT enable cancellation
		return `concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.event.pull_request.number }}"`
	} else if isPullRequestWorkflow(workflowData.On) {
		// For PR workflows: include ref and enable cancellation
		return `concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.ref }}"
  cancel-in-progress: true`
	} else {
		// For other workflows: use static concurrency without cancellation
		return `concurrency:
  group: "gh-aw-${{ github.workflow }}"`
	}
}

// isPullRequestWorkflow checks if a workflow's "on" section contains pull_request triggers
func isPullRequestWorkflow(on string) bool {
	return strings.Contains(on, "pull_request")
}
