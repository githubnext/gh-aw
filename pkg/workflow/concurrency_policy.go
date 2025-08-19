package workflow

import (
	"fmt"
	"strings"
)

// ConcurrencyPolicy represents a single concurrency policy definition
type ConcurrencyPolicy struct {
	Group            string `json:"group" yaml:"group"`
	Node             string `json:"node" yaml:"node"`
	CancelInProgress *bool  `json:"cancel-in-progress,omitempty" yaml:"cancel-in-progress,omitempty"`
}

// ConcurrencyPolicySet represents a set of policies for different contexts
type ConcurrencyPolicySet struct {
	Default     *ConcurrencyPolicy            `json:"*,omitempty" yaml:"*,omitempty"`
	Issues      *ConcurrencyPolicy            `json:"issues,omitempty" yaml:"issues,omitempty"`
	PullRequest *ConcurrencyPolicy            `json:"pull_requests,omitempty" yaml:"pull_requests,omitempty"`
	Schedule    *ConcurrencyPolicy            `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	Manual      *ConcurrencyPolicy            `json:"workflow_dispatch,omitempty" yaml:"workflow_dispatch,omitempty"`
	Custom      map[string]*ConcurrencyPolicy `json:"-" yaml:"-"` // for any other trigger types
}

// ComputedConcurrencyPolicy represents the final computed concurrency configuration
type ComputedConcurrencyPolicy struct {
	Group            string
	CancelInProgress *bool
}

// computeConcurrencyPolicy computes the final concurrency configuration based on workflow characteristics
func computeConcurrencyPolicy(workflowData *WorkflowData, isAliasTrigger bool) (*ComputedConcurrencyPolicy, error) {
	// Get default policies based on workflow characteristics
	policySet := getDefaultPolicySet(isAliasTrigger)

	// Determine which specific policy to use based on workflow triggers
	selectedPolicy := selectPolicyForWorkflow(workflowData, isAliasTrigger, policySet)

	// Compute the final concurrency configuration
	computed := &ComputedConcurrencyPolicy{}

	if selectedPolicy != nil {
		// Build the group identifier
		computed.Group = buildGroupIdentifier(selectedPolicy, workflowData)
		computed.CancelInProgress = selectedPolicy.CancelInProgress
	} else {
		// Fallback to basic group
		computed.Group = "gh-aw-${{ github.workflow }}"
	}

	return computed, nil
}

// getDefaultPolicySet returns the default policy set based on workflow characteristics
func getDefaultPolicySet(isAliasTrigger bool) *ConcurrencyPolicySet {
	policySet := &ConcurrencyPolicySet{
		Custom: make(map[string]*ConcurrencyPolicy),
	}

	// Default policy for all workflows
	policySet.Default = &ConcurrencyPolicy{
		Group: "workflow",
		Node:  "",
	}

	// Issues policy with cancel
	cancelTrue := true
	policySet.Issues = &ConcurrencyPolicy{
		Group:            "workflow",
		Node:             "issue.number || github.event.pull_request.number", // Support both issue and PR for alias
		CancelInProgress: &cancelTrue,
	}

	// Pull request policy with cancel (use ref for backwards compatibility with existing tests)
	policySet.PullRequest = &ConcurrencyPolicy{
		Group:            "workflow",
		Node:             "github.ref", // Use ref instead of pull_request.number for compatibility
		CancelInProgress: &cancelTrue,
	}

	// For alias triggers, override to not use cancellation
	if isAliasTrigger {
		policySet.Issues.CancelInProgress = nil
		policySet.PullRequest.CancelInProgress = nil
	}

	return policySet
}

// selectPolicyForWorkflow selects the most appropriate policy for the given workflow
func selectPolicyForWorkflow(workflowData *WorkflowData, isAliasTrigger bool, policySet *ConcurrencyPolicySet) *ConcurrencyPolicy {
	if isAliasTrigger {
		// For alias workflows, prefer issues policy if available
		if policySet.Issues != nil {
			return policySet.Issues
		}
	}

	// Check if this is a pull request workflow
	if isPullRequestWorkflow(workflowData.On) {
		if policySet.PullRequest != nil {
			return policySet.PullRequest
		}
	}

	// Check for schedule workflows
	if strings.Contains(workflowData.On, "schedule") {
		if policySet.Schedule != nil {
			return policySet.Schedule
		}
	}

	// Check for manual workflows
	if strings.Contains(workflowData.On, "workflow_dispatch") {
		if policySet.Manual != nil {
			return policySet.Manual
		}
	}

	// Check for issues workflows
	if strings.Contains(workflowData.On, "issues") {
		if policySet.Issues != nil {
			return policySet.Issues
		}
	}

	// Check for custom trigger types in the policy custom map
	for triggerType, customPolicy := range policySet.Custom {
		if strings.Contains(workflowData.On, triggerType) {
			return customPolicy
		}
	}

	// Fall back to default policy
	return policySet.Default
}

// buildGroupIdentifier constructs the final group identifier string
func buildGroupIdentifier(policy *ConcurrencyPolicy, workflowData *WorkflowData) string {
	if policy == nil {
		return "gh-aw-${{ github.workflow }}"
	}

	// Start with the base group
	var parts []string

	// Always include the gh-aw prefix
	parts = append(parts, "gh-aw")

	// Add the workflow identifier
	if policy.Group == "workflow" {
		parts = append(parts, "${{ github.workflow }}")
	} else {
		// Use custom group identifier
		parts = append(parts, policy.Group)
	}

	// Add the node identifier if specified
	if policy.Node != "" {
		var nodeExpr string
		switch policy.Node {
		case "issue.number":
			nodeExpr = "${{ github.event.issue.number }}"
		case "pull_request.number":
			nodeExpr = "${{ github.event.pull_request.number }}"
		case "github.ref":
			nodeExpr = "${{ github.ref }}"
		case "issue.number || github.event.pull_request.number":
			// Special case for alias workflows
			nodeExpr = "${{ github.event.issue.number || github.event.pull_request.number }}"
		default:
			// Custom node expression
			if strings.HasPrefix(policy.Node, "${{") {
				nodeExpr = policy.Node
			} else {
				nodeExpr = fmt.Sprintf("${{ %s }}", policy.Node)
			}
		}
		parts = append(parts, nodeExpr)
	}

	return strings.Join(parts, "-")
}

// generateConcurrencyYAML generates the final YAML for the concurrency section
func generateConcurrencyYAML(computed *ComputedConcurrencyPolicy) string {
	if computed == nil {
		return ""
	}

	var lines []string
	lines = append(lines, "concurrency:")
	lines = append(lines, fmt.Sprintf("  group: \"%s\"", computed.Group))

	if computed.CancelInProgress != nil && *computed.CancelInProgress {
		lines = append(lines, "  cancel-in-progress: true")
	}

	return strings.Join(lines, "\n")
}
