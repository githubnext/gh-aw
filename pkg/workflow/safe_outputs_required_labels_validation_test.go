//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSafeOutputsRequiredLabelsWithTitlePrefix(t *testing.T) {
	tests := []struct {
		name    string
		config  *SafeOutputsConfig
		wantErr string
	}{
		{
			name: "close-issue requires labels when title prefix is set",
			config: &SafeOutputsConfig{
				CloseIssues: &CloseIssuesConfig{
					SafeOutputFilterConfig: SafeOutputFilterConfig{
						RequiredTitlePrefix: "[bot] ",
					},
				},
			},
			wantErr: "safe-outputs.close-issue.required-labels",
		},
		{
			name: "close-issue passes when labels present",
			config: &SafeOutputsConfig{
				CloseIssues: &CloseIssuesConfig{
					SafeOutputFilterConfig: SafeOutputFilterConfig{
						RequiredTitlePrefix: "[bot] ",
						RequiredLabels:      []string{"automated"},
					},
				},
			},
		},
		{
			name: "close-pull-request requires labels when title prefix is set",
			config: &SafeOutputsConfig{
				ClosePullRequests: &ClosePullRequestsConfig{
					SafeOutputFilterConfig: SafeOutputFilterConfig{
						RequiredTitlePrefix: "[bot] ",
					},
				},
			},
			wantErr: "safe-outputs.close-pull-request.required-labels",
		},
		{
			name: "close-pull-request passes when labels present",
			config: &SafeOutputsConfig{
				ClosePullRequests: &ClosePullRequestsConfig{
					SafeOutputFilterConfig: SafeOutputFilterConfig{
						RequiredTitlePrefix: "[bot] ",
						RequiredLabels:      []string{"automated"},
					},
				},
			},
		},
		{
			name: "push-to-pull-request-branch requires required-labels when title prefix is set",
			config: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					TitlePrefix: "[bot] ",
				},
			},
			wantErr: "safe-outputs.push-to-pull-request-branch.required-labels",
		},
		{
			name: "push-to-pull-request-branch passes when required-labels present",
			config: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					TitlePrefix:    "[bot] ",
					RequiredLabels: []string{"automated"},
				},
			},
		},
		{
			name: "mark-ready requires labels when title prefix is set",
			config: &SafeOutputsConfig{
				MarkPullRequestAsReadyForReview: &MarkPullRequestAsReadyForReviewConfig{
					SafeOutputFilterConfig: SafeOutputFilterConfig{
						RequiredTitlePrefix: "[bot] ",
					},
				},
			},
			wantErr: "safe-outputs.mark-pull-request-as-ready-for-review.required-labels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsRequiredLabelsWithTitlePrefix(tt.config)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
