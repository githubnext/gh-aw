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
			name: "empty allowed-labels entry fails",
			config: &SafeOutputsConfig{
				MergePullRequest: &MergePullRequestConfig{
					AllowedLabels: []string{"release", ""},
				},
			},
			wantErr: "safe-outputs.merge-pull-request.allowed-labels[1] cannot be empty",
		},
		{
			name: "non-empty labels pass",
			config: &SafeOutputsConfig{
				MergePullRequest: &MergePullRequestConfig{
					RequiredLabels: []string{"safe-to-merge"},
					AllowedLabels:  []string{"release", "automerge"},
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
			assert.Contains(t, err.Error(), tt.wantErr, "expected validation error to include field-specific message")
		})
	}
}
