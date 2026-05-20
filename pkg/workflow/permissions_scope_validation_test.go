//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePermissionScopeNames(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty yaml is valid",
			yaml:    "",
			wantErr: false,
		},
		{
			name: "valid scope names",
			yaml: `contents: read
issues: write
pull-requests: read`,
			wantErr: false,
		},
		{
			name:    "read-all shorthand is valid",
			yaml:    "read-all",
			wantErr: false,
		},
		{
			name: "permissions prefix with valid scopes",
			yaml: `permissions:
  contents: read
  issues: write`,
			wantErr: false,
		},
		{
			name:        "typo in scope name suggests correction",
			yaml:        `contnts: read`,
			wantErr:     true,
			errContains: "Did you mean",
		},
		{
			name:        "typo pull-rquests suggests pull-requests",
			yaml:        `pull-rquests: write`,
			wantErr:     true,
			errContains: "Did you mean",
		},
		{
			name:    "completely unknown scope with no close match is silently skipped",
			yaml:    `xyz_totally_unknown_scope: read`,
			wantErr: false,
		},
		{
			name:    "all meta-key is valid",
			yaml:    `all: read`,
			wantErr: false,
		},
		{
			name:        "case-only typo Contents suggests contents",
			yaml:        `Contents: read`,
			wantErr:     true,
			errContains: "contents",
		},
		{
			name:        "case-only typo Issues suggests issues",
			yaml:        `Issues: write`,
			wantErr:     true,
			errContains: "issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePermissionScopeNames(tt.yaml)
			if tt.wantErr {
				require.Error(t, err, "ValidatePermissionScopeNames should return an error")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains,
						"error should contain %q", tt.errContains)
				}
			} else {
				assert.NoError(t, err, "ValidatePermissionScopeNames should not return an error")
			}
		})
	}
}
