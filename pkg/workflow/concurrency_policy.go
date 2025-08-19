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

// parseConcurrencyPolicyFromFrontmatter extracts concurrency policy configuration from frontmatter
func parseConcurrencyPolicyFromFrontmatter(frontmatter map[string]any) (*ConcurrencyPolicySet, error) {
	// Look for "concurrency_policy" key in frontmatter
	policyValue, exists := frontmatter["concurrency_policy"]
	if !exists {
		return nil, nil // No policy defined, use defaults
	}

	policyMap, ok := policyValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("concurrency_policy must be an object")
	}

	policySet := &ConcurrencyPolicySet{
		Custom: make(map[string]*ConcurrencyPolicy),
	}

	// Parse each policy context
	for key, value := range policyMap {
		valueMap, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("concurrency_policy.%s must be an object", key)
		}

		policy, err := parseSinglePolicy(valueMap)
		if err != nil {
			return nil, fmt.Errorf("invalid concurrency_policy.%s: %w", key, err)
		}

		// Assign to the appropriate field based on key
		switch key {
		case "*":
			policySet.Default = policy
		case "issues":
			policySet.Issues = policy
		case "pull_requests":
			policySet.PullRequest = policy
		case "schedule":
			policySet.Schedule = policy
		case "workflow_dispatch":
			policySet.Manual = policy
		default:
			policySet.Custom[key] = policy
		}
	}

	return policySet, nil
}

// parseSinglePolicy parses a single policy definition
func parseSinglePolicy(policyMap map[string]any) (*ConcurrencyPolicy, error) {
	policy := &ConcurrencyPolicy{}

	// Parse group (can be either "group" or "id" for backwards compatibility)
	if group, exists := policyMap["group"]; exists {
		if groupStr, ok := group.(string); ok {
			policy.Group = groupStr
		} else {
			return nil, fmt.Errorf("group must be a string")
		}
	} else if id, exists := policyMap["id"]; exists {
		// Support "id" for backwards compatibility (mentioned in issue)
		if idStr, ok := id.(string); ok {
			policy.Group = idStr
		} else {
			return nil, fmt.Errorf("id must be a string")
		}
	}

	// Parse node
	if node, exists := policyMap["node"]; exists {
		if nodeStr, ok := node.(string); ok {
			policy.Node = nodeStr
		} else {
			return nil, fmt.Errorf("node must be a string")
		}
	}

	// Parse cancel-in-progress
	if cancel, exists := policyMap["cancel-in-progress"]; exists {
		if cancelBool, ok := cancel.(bool); ok {
			policy.CancelInProgress = &cancelBool
		} else {
			return nil, fmt.Errorf("cancel-in-progress must be a boolean")
		}
	}

	return policy, nil
}

// computeConcurrencyPolicy merges policies from different sources and computes the final configuration
func computeConcurrencyPolicy(workflowData *WorkflowData, isAliasTrigger bool, userPolicySet *ConcurrencyPolicySet) (*ComputedConcurrencyPolicy, error) {
	// Start with default policies based on workflow characteristics
	defaultPolicySet := getDefaultPolicySet(isAliasTrigger)

	// Merge user-defined policies if provided
	finalPolicySet := mergePolicySets(defaultPolicySet, userPolicySet)

	// Determine which specific policy to use based on workflow triggers
	selectedPolicy := selectPolicyForWorkflow(workflowData, isAliasTrigger, finalPolicySet)

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

// mergePolicySets merges user-defined policies with default policies
func mergePolicySets(base, override *ConcurrencyPolicySet) *ConcurrencyPolicySet {
	if override == nil {
		return base
	}

	result := &ConcurrencyPolicySet{
		Custom: make(map[string]*ConcurrencyPolicy),
	}

	// Copy base policies
	if base.Default != nil {
		result.Default = copyPolicy(base.Default)
	}
	if base.Issues != nil {
		result.Issues = copyPolicy(base.Issues)
	}
	if base.PullRequest != nil {
		result.PullRequest = copyPolicy(base.PullRequest)
	}
	if base.Schedule != nil {
		result.Schedule = copyPolicy(base.Schedule)
	}
	if base.Manual != nil {
		result.Manual = copyPolicy(base.Manual)
	}
	for k, v := range base.Custom {
		result.Custom[k] = copyPolicy(v)
	}

	// Override with user-defined policies
	if override.Default != nil {
		result.Default = mergePolicy(result.Default, override.Default)
	}
	if override.Issues != nil {
		result.Issues = mergePolicy(result.Issues, override.Issues)
	}
	if override.PullRequest != nil {
		result.PullRequest = mergePolicy(result.PullRequest, override.PullRequest)
	}
	if override.Schedule != nil {
		result.Schedule = mergePolicy(result.Schedule, override.Schedule)
	}
	if override.Manual != nil {
		result.Manual = mergePolicy(result.Manual, override.Manual)
	}
	for k, v := range override.Custom {
		result.Custom[k] = mergePolicy(result.Custom[k], v)
	}

	return result
}

// mergePolicy merges two policies, with override taking precedence
func mergePolicy(base, override *ConcurrencyPolicy) *ConcurrencyPolicy {
	if override == nil {
		return base
	}
	if base == nil {
		return copyPolicy(override)
	}

	result := copyPolicy(base)

	// Override fields if specified
	if override.Group != "" {
		result.Group = override.Group
	}
	if override.Node != "" {
		result.Node = override.Node
	}
	if override.CancelInProgress != nil {
		result.CancelInProgress = override.CancelInProgress
	}

	return result
}

// copyPolicy creates a deep copy of a policy
func copyPolicy(policy *ConcurrencyPolicy) *ConcurrencyPolicy {
	if policy == nil {
		return nil
	}

	result := &ConcurrencyPolicy{
		Group: policy.Group,
		Node:  policy.Node,
	}

	if policy.CancelInProgress != nil {
		cancel := *policy.CancelInProgress
		result.CancelInProgress = &cancel
	}

	return result
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
