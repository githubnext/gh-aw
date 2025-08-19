package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
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
// This is the new entry point that accepts frontmatter for policy parsing
func GenerateConcurrencyConfigWithFrontmatter(workflowData *WorkflowData, isAliasTrigger bool, frontmatter map[string]any, verbose bool) string {
	// Don't override if already set by user
	if workflowData.Concurrency != "" {
		return workflowData.Concurrency
	}

	// Parse concurrency policy from frontmatter
	userPolicySet, err := parseConcurrencyPolicyFromFrontmatter(frontmatter)
	if err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage("Failed to parse concurrency policy, using defaults: " + err.Error()))
		}
		// Fall back to legacy behavior
		return generateConcurrencyWithPolicySystem(workflowData, isAliasTrigger)
	}

	// Compute the final policy
	computed, err := computeConcurrencyPolicy(workflowData, isAliasTrigger, userPolicySet)
	if err != nil {
		if verbose {
			fmt.Println(console.FormatWarningMessage("Failed to compute concurrency policy, using defaults: " + err.Error()))
		}
		// Fall back to legacy behavior
		return generateConcurrencyWithPolicySystem(workflowData, isAliasTrigger)
	}

	// Generate YAML
	yaml := generateConcurrencyYAML(computed)
	if yaml == "" {
		// Fall back to legacy behavior
		return generateConcurrencyWithPolicySystem(workflowData, isAliasTrigger)
	}

	return yaml
}

// generateConcurrencyWithPolicySystem uses the new policy system but with default policies only
func generateConcurrencyWithPolicySystem(workflowData *WorkflowData, isAliasTrigger bool) string {
	// Compute policy using defaults only (no user override)
	computed, err := computeConcurrencyPolicy(workflowData, isAliasTrigger, nil)
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
