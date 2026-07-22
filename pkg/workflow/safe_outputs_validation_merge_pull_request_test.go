//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSafeOutputsMergePullRequestLabelValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *SafeOutputsConfig
		wantErr string
	}{
		{
			name: "empty required-labels entry fails",
			config: &SafeOutputsConfig{
				MergePullRequest: &MergePullRequestConfig{
					RequiredLabels: []string{"safe-to-merge", "   "},
				},
			},
			wantErr: "safe-outputs.merge-pull-request.required-labels[1] cannot be empty",
		},
		{
			name: "non-empty labels pass",
			config: &SafeOutputsConfig{
				MergePullRequest: &MergePullRequestConfig{
					RequiredLabels: []string{"safe-to-merge", "automerge"},
				},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsMergePullRequest(tt.config)
			if tt.wantErr == "" {
				assert.NoError(t, err, "expected merge-pull-request label validation to pass")
				return
			}
			require.Error(t, err, "expected merge-pull-request label validation to fail")
			require.ErrorContains(t, err, tt.wantErr, "expected validation error to include field-specific message")
		})
	}
}

func TestValidateSafeOutputsMergePullRequestAllowedBranchesValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *SafeOutputsConfig
		wantErr string
	}{
		{
			name: "invalid pattern at index 0 fails",
			config: &SafeOutputsConfig{
				MergePullRequest: &MergePullRequestConfig{
					AllowedBranches: []string{"feature branch"},
				},
			},
			wantErr: `invalid glob pattern "feature branch" in safe-outputs.merge-pull-request.allowed-branches[0]`,
		},
		{
			name: "invalid pattern at index 1 reports index 1",
			config: &SafeOutputsConfig{
				MergePullRequest: &MergePullRequestConfig{
					AllowedBranches: []string{"main", "feature branch"},
				},
			},
			wantErr: `invalid glob pattern "feature branch" in safe-outputs.merge-pull-request.allowed-branches[1]`,
		},
		{
			name: "valid patterns pass",
			config: &SafeOutputsConfig{
				MergePullRequest: &MergePullRequestConfig{
					AllowedBranches: []string{"main", "feature/*"},
				},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsMergePullRequest(tt.config)
			if tt.wantErr == "" {
				assert.NoError(t, err, "expected merge-pull-request allowed-branches validation to pass")
				return
			}

			require.Error(t, err, "expected merge-pull-request allowed-branches validation to fail")
			require.ErrorContains(t, err, tt.wantErr, "expected field-specific allowed-branches error")
		})
	}
}
