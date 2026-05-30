//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEventTypes(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid event string form",
			frontmatter: map[string]any{
				"on": "push",
			},
			wantErr: false,
		},
		{
			name: "valid events in list form",
			frontmatter: map[string]any{
				"on": []any{"push", "pull_request"},
			},
			wantErr: false,
		},
		{
			name: "valid events in map form",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push":              nil,
					"workflow_dispatch": nil,
				},
			},
			wantErr: false,
		},
		{
			name: "typo pus suggests push",
			frontmatter: map[string]any{
				"on": []any{"pus"},
			},
			wantErr:     true,
			errContains: "Did you mean",
		},
		{
			name: "typo pull_rquest suggests pull_request",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_rquest": nil,
				},
			},
			wantErr:     true,
			errContains: "Did you mean",
		},
		{
			name: "unknown event with no close match is silently skipped",
			frontmatter: map[string]any{
				"on": []any{"completely_unknown_event_xyz"},
			},
			wantErr: false,
		},
		{
			name:        "no on section is valid",
			frontmatter: map[string]any{},
			wantErr:     false,
		},
		{
			name: "error message contains valid event types",
			frontmatter: map[string]any{
				"on": []any{"pus"},
			},
			wantErr:     true,
			errContains: "push",
		},
		{
			name: "gh-aw synthetic key needs is silently skipped",
			frontmatter: map[string]any{
				"on": map[string]any{
					"needs": nil,
				},
			},
			wantErr: false,
		},
		{
			name: "case-only typo Push suggests push",
			frontmatter: map[string]any{
				"on": []any{"Push"},
			},
			wantErr:     true,
			errContains: "push",
		},
		{
			name: "case-only typo Pull_Request suggests pull_request",
			frontmatter: map[string]any{
				"on": map[string]any{
					"Pull_Request": nil,
				},
			},
			wantErr:     true,
			errContains: "pull_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEventTypes(tt.frontmatter)
			if tt.wantErr {
				require.Error(t, err, "ValidateEventTypes should return an error")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains,
						"error should contain %q", tt.errContains)
				}
			} else {
				assert.NoError(t, err, "ValidateEventTypes should not return an error")
			}
		})
	}
}
