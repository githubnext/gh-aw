//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsesPatchesAndCheckouts(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		expected    bool
	}{
		{
			name:        "returns false for nil SafeOutputsConfig",
			safeOutputs: nil,
			expected:    false,
		},
		{
			name:        "returns false for empty SafeOutputsConfig",
			safeOutputs: &SafeOutputsConfig{},
			expected:    false,
		},
		{
			name: "returns true when CreatePullRequests is set",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expected: true,
		},
		{
			name: "returns true when PushToPullRequestBranch is set",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			expected: true,
		},
		{
			name: "returns true when both CreatePullRequests and PushToPullRequestBranch are set",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests:      &CreatePullRequestsConfig{},
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			expected: true,
		},
		{
			name: "returns false when only CreateIssues is set",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expected: false,
		},
		{
			name: "returns false when only AddComments is set",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{},
			},
			expected: false,
		},
		{
			name: "returns false when only UpdatePullRequests is set",
			safeOutputs: &SafeOutputsConfig{
				UpdatePullRequests: &UpdatePullRequestsConfig{},
			},
			expected: false,
		},
		{
			name: "returns true when CreatePullRequests is set alongside other outputs",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
				CreateIssues:       &CreateIssuesConfig{},
				AddComments:        &AddCommentsConfig{},
			},
			expected: true,
		},
		{
			name: "returns true when PushToPullRequestBranch is set alongside other outputs",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
				CreateIssues:            &CreateIssuesConfig{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usesPatchesAndCheckouts(tt.safeOutputs)
			assert.Equal(t, tt.expected, result, "usesPatchesAndCheckouts should return expected value")
		})
	}
}
