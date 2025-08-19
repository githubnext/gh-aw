package workflow

import (
	"testing"
)

func TestParseConcurrencyPolicyFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    *ConcurrencyPolicySet
		shouldError bool
		description string
	}{
		{
			name:        "empty frontmatter returns nil",
			frontmatter: map[string]any{},
			expected:    nil,
			shouldError: false,
			description: "No concurrency_policy key should return nil",
		},
		{
			name: "basic policy with default",
			frontmatter: map[string]any{
				"concurrency_policy": map[string]any{
					"*": map[string]any{
						"group": "workflow",
					},
				},
			},
			expected: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{
					Group: "workflow",
				},
				Custom: make(map[string]*ConcurrencyPolicy),
			},
			shouldError: false,
			description: "Basic default policy should parse correctly",
		},
		{
			name: "complete policy set",
			frontmatter: map[string]any{
				"concurrency_policy": map[string]any{
					"*": map[string]any{
						"group": "workflow",
					},
					"issues": map[string]any{
						"group":              "workflow",
						"node":               "issue.number",
						"cancel-in-progress": true,
					},
					"pull_requests": map[string]any{
						"group":              "workflow",
						"node":               "pull_request.number",
						"cancel-in-progress": true,
					},
				},
			},
			expected: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{
					Group: "workflow",
				},
				Issues: &ConcurrencyPolicy{
					Group:            "workflow",
					Node:             "issue.number",
					CancelInProgress: boolPtr(true),
				},
				PullRequest: &ConcurrencyPolicy{
					Group:            "workflow",
					Node:             "pull_request.number",
					CancelInProgress: boolPtr(true),
				},
				Custom: make(map[string]*ConcurrencyPolicy),
			},
			shouldError: false,
			description: "Complete policy set should parse correctly",
		},
		{
			name: "policy with backwards compatible id field",
			frontmatter: map[string]any{
				"concurrency_policy": map[string]any{
					"*": map[string]any{
						"id": "workflow", // using "id" instead of "group"
					},
				},
			},
			expected: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{
					Group: "workflow",
				},
				Custom: make(map[string]*ConcurrencyPolicy),
			},
			shouldError: false,
			description: "Backwards compatible 'id' field should work",
		},
		{
			name: "invalid policy type",
			frontmatter: map[string]any{
				"concurrency_policy": "invalid",
			},
			expected:    nil,
			shouldError: true,
			description: "Invalid policy type should return error",
		},
		{
			name: "invalid cancel-in-progress type",
			frontmatter: map[string]any{
				"concurrency_policy": map[string]any{
					"*": map[string]any{
						"group":              "workflow",
						"cancel-in-progress": "invalid",
					},
				},
			},
			expected:    nil,
			shouldError: true,
			description: "Invalid cancel-in-progress type should return error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseConcurrencyPolicyFromFrontmatter(tt.frontmatter)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !comparePolicySets(result, tt.expected) {
				t.Errorf("Policy sets don't match.\nGot: %+v\nExpected: %+v", result, tt.expected)
			}
		})
	}
}

func TestComputeConcurrencyPolicy(t *testing.T) {
	tests := []struct {
		name           string
		workflowData   *WorkflowData
		isAliasTrigger bool
		userPolicySet  *ConcurrencyPolicySet
		expected       *ComputedConcurrencyPolicy
		description    string
	}{
		{
			name: "default policy for basic workflow",
			workflowData: &WorkflowData{
				On: "push:",
			},
			isAliasTrigger: false,
			userPolicySet:  nil,
			expected: &ComputedConcurrencyPolicy{
				Group: "gh-aw-${{ github.workflow }}",
			},
			description: "Basic workflow should use default policy",
		},
		{
			name: "pull request workflow with cancellation",
			workflowData: &WorkflowData{
				On: "pull_request:\n  types: [opened]",
			},
			isAliasTrigger: false,
			userPolicySet:  nil,
			expected: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}-${{ github.ref }}",
				CancelInProgress: boolPtr(true),
			},
			description: "PR workflow should use PR policy with cancellation",
		},
		{
			name: "alias workflow without cancellation",
			workflowData: &WorkflowData{
				On: "issues:\n  types: [opened]",
			},
			isAliasTrigger: true,
			userPolicySet:  nil,
			expected: &ComputedConcurrencyPolicy{
				Group: "gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.event.pull_request.number }}",
			},
			description: "Alias workflow should not use cancellation",
		},
		{
			name: "user override policy",
			workflowData: &WorkflowData{
				On: "push:",
			},
			isAliasTrigger: false,
			userPolicySet: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{
					Group:            "custom-group",
					CancelInProgress: boolPtr(true),
				},
				Custom: make(map[string]*ConcurrencyPolicy),
			},
			expected: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-custom-group",
				CancelInProgress: boolPtr(true),
			},
			description: "User override should take precedence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := computeConcurrencyPolicy(tt.workflowData, tt.isAliasTrigger, tt.userPolicySet)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !compareComputedPolicies(result, tt.expected) {
				t.Errorf("Computed policies don't match.\nGot: %+v\nExpected: %+v", result, tt.expected)
			}
		})
	}
}

func TestBuildGroupIdentifier(t *testing.T) {
	tests := []struct {
		name         string
		policy       *ConcurrencyPolicy
		workflowData *WorkflowData
		expected     string
		description  string
	}{
		{
			name: "basic workflow group",
			policy: &ConcurrencyPolicy{
				Group: "workflow",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}",
			description:  "Basic workflow group should include workflow name",
		},
		{
			name: "issue number node",
			policy: &ConcurrencyPolicy{
				Group: "workflow",
				Node:  "issue.number",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}-${{ github.event.issue.number }}",
			description:  "Issue number node should be included",
		},
		{
			name: "pull request number node",
			policy: &ConcurrencyPolicy{
				Group: "workflow",
				Node:  "pull_request.number",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}-${{ github.event.pull_request.number }}",
			description:  "PR number node should be included",
		},
		{
			name: "custom group and node",
			policy: &ConcurrencyPolicy{
				Group: "custom-group",
				Node:  "custom.expr",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-custom-group-${{ custom.expr }}",
			description:  "Custom group and node should be included",
		},
		{
			name: "custom node with expression",
			policy: &ConcurrencyPolicy{
				Group: "workflow",
				Node:  "${{ github.event.custom.field }}",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}-${{ github.event.custom.field }}",
			description:  "Custom expression node should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildGroupIdentifier(tt.policy, tt.workflowData)

			if result != tt.expected {
				t.Errorf("Group identifier doesn't match.\nGot: %s\nExpected: %s", result, tt.expected)
			}
		})
	}
}

func TestGenerateConcurrencyYAML(t *testing.T) {
	tests := []struct {
		name        string
		computed    *ComputedConcurrencyPolicy
		expected    string
		description string
	}{
		{
			name: "basic concurrency without cancel",
			computed: &ComputedConcurrencyPolicy{
				Group: "gh-aw-${{ github.workflow }}",
			},
			expected:    "concurrency:\n  group: \"gh-aw-${{ github.workflow }}\"",
			description: "Basic concurrency should only include group",
		},
		{
			name: "concurrency with cancellation",
			computed: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}-${{ github.ref }}",
				CancelInProgress: boolPtr(true),
			},
			expected:    "concurrency:\n  group: \"gh-aw-${{ github.workflow }}-${{ github.ref }}\"\n  cancel-in-progress: true",
			description: "Concurrency with cancellation should include both fields",
		},
		{
			name: "concurrency with cancellation false",
			computed: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}",
				CancelInProgress: boolPtr(false),
			},
			expected:    "concurrency:\n  group: \"gh-aw-${{ github.workflow }}\"",
			description: "Concurrency with cancel false should not include cancel field",
		},
		{
			name:        "nil computed policy",
			computed:    nil,
			expected:    "",
			description: "Nil policy should return empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateConcurrencyYAML(tt.computed)

			if result != tt.expected {
				t.Errorf("YAML doesn't match.\nGot:\n%s\nExpected:\n%s", result, tt.expected)
			}
		})
	}
}

func TestMergePolicySets(t *testing.T) {
	tests := []struct {
		name        string
		base        *ConcurrencyPolicySet
		override    *ConcurrencyPolicySet
		expected    *ConcurrencyPolicySet
		description string
	}{
		{
			name: "merge with nil override",
			base: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{Group: "workflow"},
				Custom:  make(map[string]*ConcurrencyPolicy),
			},
			override: nil,
			expected: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{Group: "workflow"},
				Custom:  make(map[string]*ConcurrencyPolicy),
			},
			description: "Merge with nil should return base",
		},
		{
			name: "override default policy",
			base: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{Group: "workflow"},
				Custom:  make(map[string]*ConcurrencyPolicy),
			},
			override: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{
					Group:            "custom-workflow",
					CancelInProgress: boolPtr(true),
				},
				Custom: make(map[string]*ConcurrencyPolicy),
			},
			expected: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{
					Group:            "custom-workflow",
					CancelInProgress: boolPtr(true),
				},
				Custom: make(map[string]*ConcurrencyPolicy),
			},
			description: "Override should replace default policy",
		},
		{
			name: "partial override",
			base: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{
					Group: "workflow",
					Node:  "base-node",
				},
				Custom: make(map[string]*ConcurrencyPolicy),
			},
			override: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{
					CancelInProgress: boolPtr(true),
				},
				Custom: make(map[string]*ConcurrencyPolicy),
			},
			expected: &ConcurrencyPolicySet{
				Default: &ConcurrencyPolicy{
					Group:            "workflow",
					Node:             "base-node",
					CancelInProgress: boolPtr(true),
				},
				Custom: make(map[string]*ConcurrencyPolicy),
			},
			description: "Partial override should merge fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergePolicySets(tt.base, tt.override)

			if !comparePolicySets(result, tt.expected) {
				t.Errorf("Merged policy sets don't match.\nGot: %+v\nExpected: %+v", result, tt.expected)
			}
		})
	}
}

// Helper functions for testing

func boolPtr(b bool) *bool {
	return &b
}

func comparePolicySets(a, b *ConcurrencyPolicySet) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return comparePolicies(a.Default, b.Default) &&
		comparePolicies(a.Issues, b.Issues) &&
		comparePolicies(a.PullRequest, b.PullRequest) &&
		comparePolicies(a.Schedule, b.Schedule) &&
		comparePolicies(a.Manual, b.Manual) &&
		compareCustomPolicies(a.Custom, b.Custom)
}

func comparePolicies(a, b *ConcurrencyPolicy) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return a.Group == b.Group &&
		a.Node == b.Node &&
		compareBoolPtrs(a.CancelInProgress, b.CancelInProgress)
}

func compareBoolPtrs(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func compareCustomPolicies(a, b map[string]*ConcurrencyPolicy) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if !comparePolicies(v, b[k]) {
			return false
		}
	}

	return true
}

func compareComputedPolicies(a, b *ComputedConcurrencyPolicy) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return a.Group == b.Group &&
		compareBoolPtrs(a.CancelInProgress, b.CancelInProgress)
}
