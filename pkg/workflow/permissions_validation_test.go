package workflow

import (
	"strings"
	"testing"
)

func TestValidatePermissionScopes(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name:        "no permissions specified is valid",
			frontmatter: map[string]any{},
			wantErr:     false,
		},
		{
			name: "shorthand permissions are valid",
			frontmatter: map[string]any{
				"permissions": "read-all",
			},
			wantErr: false,
		},
		{
			name: "valid permission scopes",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"contents":        "read",
					"issues":          "write",
					"pull-requests":   "write",
					"discussions":     "write",
					"security-events": "write",
				},
			},
			wantErr: false,
		},
		{
			name: "repository-projects permission is invalid",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"repository-projects": "read",
				},
			},
			wantErr:     true,
			errContains: "repository-projects",
		},
		{
			name: "repository-projects with write is invalid",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"repository-projects": "write",
					"contents":            "read",
				},
			},
			wantErr:     true,
			errContains: "repository-projects",
		},
		{
			name: "repository-projects with other valid permissions is still invalid",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"contents":            "read",
					"issues":              "write",
					"repository-projects": "read",
				},
			},
			wantErr:     true,
			errContains: "repository-projects",
		},
		{
			name: "all read with repository-projects is invalid",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"all":                 "read",
					"repository-projects": "write",
				},
			},
			wantErr:     true,
			errContains: "repository-projects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePermissionScopes(tt.frontmatter)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePermissionScopes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validatePermissionScopes() error = %v, should contain %q", err, tt.errContains)
				}
				// Verify the error message contains helpful information
				if !strings.Contains(err.Error(), "classic GitHub Projects") {
					t.Errorf("validatePermissionScopes() error should explain about classic GitHub Projects, got: %v", err)
				}
				if !strings.Contains(err.Error(), "https://github.com/orgs/community/discussions/54538") {
					t.Errorf("validatePermissionScopes() error should include reference link, got: %v", err)
				}
			}
		})
	}
}
