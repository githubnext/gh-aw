package workflow

import (
	"testing"
)

func TestComputeConcurrencyPolicy(t *testing.T) {
	tests := []struct {
		name           string
		workflowData   *WorkflowData
		isAliasTrigger bool
		expected       *ComputedConcurrencyPolicy
		description    string
	}{
		{
			name: "basic workflow without special triggers",
			workflowData: &WorkflowData{
				On: "push:",
			},
			isAliasTrigger: false,
			expected: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}",
				CancelInProgress: nil,
			},
			description: "Basic workflow should use simple group",
		},
		{
			name: "pull request workflow",
			workflowData: &WorkflowData{
				On: "pull_request:",
			},
			isAliasTrigger: false,
			expected: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}-${{ github.ref }}",
				CancelInProgress: &[]bool{true}[0],
			},
			description: "Pull request workflow should include ref and enable cancellation",
		},
		{
			name: "issues workflow",
			workflowData: &WorkflowData{
				On: "issues:",
			},
			isAliasTrigger: false,
			expected: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.event.pull_request.number }}",
				CancelInProgress: &[]bool{true}[0],
			},
			description: "Issues workflow should include issue number and enable cancellation",
		},
		{
			name: "alias workflow",
			workflowData: &WorkflowData{
				On: "issues:",
			},
			isAliasTrigger: true,
			expected: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.event.pull_request.number }}",
				CancelInProgress: nil,
			},
			description: "Alias workflow should not enable cancellation",
		},
		{
			name: "schedule workflow",
			workflowData: &WorkflowData{
				On: "schedule:",
			},
			isAliasTrigger: false,
			expected: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}",
				CancelInProgress: nil,
			},
			description: "Schedule workflow should use default policy",
		},
		{
			name: "workflow_dispatch workflow",
			workflowData: &WorkflowData{
				On: "workflow_dispatch:",
			},
			isAliasTrigger: false,
			expected: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}",
				CancelInProgress: nil,
			},
			description: "Manual workflow should use default policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := computeConcurrencyPolicy(tt.workflowData, tt.isAliasTrigger)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Group != tt.expected.Group {
				t.Errorf("Group mismatch.\nGot: %s\nExpected: %s", result.Group, tt.expected.Group)
			}

			if !compareCancelInProgress(result.CancelInProgress, tt.expected.CancelInProgress) {
				t.Errorf("CancelInProgress mismatch.\nGot: %v\nExpected: %v", result.CancelInProgress, tt.expected.CancelInProgress)
			}
		})
	}
}

func TestGenerateConcurrencyYAML(t *testing.T) {
	tests := []struct {
		name     string
		computed *ComputedConcurrencyPolicy
		expected string
	}{
		{
			name: "basic group without cancellation",
			computed: &ComputedConcurrencyPolicy{
				Group: "gh-aw-${{ github.workflow }}",
			},
			expected: `concurrency:
  group: "gh-aw-${{ github.workflow }}"`,
		},
		{
			name: "group with cancellation enabled",
			computed: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}-ref",
				CancelInProgress: &[]bool{true}[0],
			},
			expected: `concurrency:
  group: "gh-aw-${{ github.workflow }}-ref"
  cancel-in-progress: true`,
		},
		{
			name: "group with cancellation disabled",
			computed: &ComputedConcurrencyPolicy{
				Group:            "gh-aw-${{ github.workflow }}-ref",
				CancelInProgress: &[]bool{false}[0],
			},
			expected: `concurrency:
  group: "gh-aw-${{ github.workflow }}-ref"`,
		},
		{
			name:     "nil policy returns empty string",
			computed: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateConcurrencyYAML(tt.computed)
			if result != tt.expected {
				t.Errorf("YAML output mismatch.\nGot:\n%s\nExpected:\n%s", result, tt.expected)
			}
		})
	}
}

func TestGetDefaultPolicySet(t *testing.T) {
	tests := []struct {
		name           string
		isAliasTrigger bool
		description    string
	}{
		{
			name:           "normal workflow policy set",
			isAliasTrigger: false,
			description:    "Normal workflows should have cancellation enabled for issues and PR triggers",
		},
		{
			name:           "alias workflow policy set",
			isAliasTrigger: true,
			description:    "Alias workflows should not have cancellation enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policySet := getDefaultPolicySet(tt.isAliasTrigger)

			// Verify basic structure
			if policySet == nil {
				t.Error("Policy set should not be nil")
				return
			}

			if policySet.Default == nil {
				t.Error("Default policy should not be nil")
			}

			if policySet.Issues == nil {
				t.Error("Issues policy should not be nil")
			}

			if policySet.PullRequest == nil {
				t.Error("PullRequest policy should not be nil")
			}

			// Verify cancellation behavior based on alias trigger
			if tt.isAliasTrigger {
				if policySet.Issues.CancelInProgress != nil {
					t.Error("Alias workflow issues policy should not have cancellation enabled")
				}
				if policySet.PullRequest.CancelInProgress != nil {
					t.Error("Alias workflow PR policy should not have cancellation enabled")
				}
			} else {
				if policySet.Issues.CancelInProgress == nil || !*policySet.Issues.CancelInProgress {
					t.Error("Normal workflow issues policy should have cancellation enabled")
				}
				if policySet.PullRequest.CancelInProgress == nil || !*policySet.PullRequest.CancelInProgress {
					t.Error("Normal workflow PR policy should have cancellation enabled")
				}
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
	}{
		{
			name: "basic workflow group",
			policy: &ConcurrencyPolicy{
				Group: "workflow",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}",
		},
		{
			name: "custom group",
			policy: &ConcurrencyPolicy{
				Group: "custom",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-custom",
		},
		{
			name: "with issue number node",
			policy: &ConcurrencyPolicy{
				Group: "workflow",
				Node:  "issue.number",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}-${{ github.event.issue.number }}",
		},
		{
			name: "with pull request number node",
			policy: &ConcurrencyPolicy{
				Group: "workflow",
				Node:  "pull_request.number",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}-${{ github.event.pull_request.number }}",
		},
		{
			name: "with github ref node",
			policy: &ConcurrencyPolicy{
				Group: "workflow",
				Node:  "github.ref",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}-${{ github.ref }}",
		},
		{
			name: "with custom node expression",
			policy: &ConcurrencyPolicy{
				Group: "workflow",
				Node:  "custom.expression",
			},
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}-${{ custom.expression }}",
		},
		{
			name:         "nil policy returns fallback",
			policy:       nil,
			workflowData: &WorkflowData{},
			expected:     "gh-aw-${{ github.workflow }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildGroupIdentifier(tt.policy, tt.workflowData)
			if result != tt.expected {
				t.Errorf("Group identifier mismatch.\nGot: %s\nExpected: %s", result, tt.expected)
			}
		})
	}
}

// Helper functions

func compareCancelInProgress(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
