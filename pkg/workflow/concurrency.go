package workflow

import (
	"strings"
)

// GenerateConcurrencyConfig generates the concurrency configuration for a workflow
// based on its trigger types and characteristics. Now supports advanced policy computation.
func GenerateConcurrencyConfig(workflowData *WorkflowData, isAliasTrigger bool) string {
	// Don't override if already set by user
	if workflowData.Concurrency != "" {
		return workflowData.Concurrency
	}

	// Try to use the new policy system first
	return generateConcurrencyWithPolicySystem(workflowData, isAliasTrigger)
}

// GenerateConcurrencyConfigWithFrontmatter generates concurrency config using the policy system
// This function maintains the same interface but no longer parses frontmatter for policies
func GenerateConcurrencyConfigWithFrontmatter(workflowData *WorkflowData, isAliasTrigger bool, frontmatter map[string]any, verbose bool) string {
	// Don't override if already set by user
	if workflowData.Concurrency != "" {
		return workflowData.Concurrency
	}

	// Use the policy system with code-based rules only
	return generateConcurrencyWithPolicySystem(workflowData, isAliasTrigger)
}

// generateConcurrencyWithPolicySystem uses the policy system with code-based rules only
func generateConcurrencyWithPolicySystem(workflowData *WorkflowData, isAliasTrigger bool) string {
	// Compute policy using code-based rules only
	computed, err := computeConcurrencyPolicy(workflowData, isAliasTrigger)
	if err != nil {
		// Fall back to legacy behavior if policy system fails
		return generateLegacyConcurrency(workflowData, isAliasTrigger)
	}

	yaml := generateConcurrencyYAML(computed)
	if yaml == "" {
		return generateLegacyConcurrency(workflowData, isAliasTrigger)
	}

	return yaml
}

// generateLegacyConcurrency provides the original concurrency generation logic as fallback
func generateLegacyConcurrency(workflowData *WorkflowData, isAliasTrigger bool) string {
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
