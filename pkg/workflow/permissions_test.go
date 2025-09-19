package workflow

import (
	"testing"
)

func TestParsePermissions(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    *Permissions
		expectError bool
	}{
		{
			name:        "no_permissions",
			frontmatter: map[string]any{},
			expected:    nil, // No permissions section returns nil
			expectError: false,
		},
		{
			name: "global_read",
			frontmatter: map[string]any{
				"permissions": "read",
			},
			expected: &Permissions{
				Global: "read",
			},
			expectError: false,
		},
		{
			name: "global_write",
			frontmatter: map[string]any{
				"permissions": "write",
			},
			expected: &Permissions{
				Global: "write",
			},
			expectError: false,
		},
		{
			name: "global_read_all",
			frontmatter: map[string]any{
				"permissions": "read-all",
			},
			expected: &Permissions{
				Global: "read-all",
			},
			expectError: false,
		},
		{
			name: "global_write_all",
			frontmatter: map[string]any{
				"permissions": "write-all",
			},
			expected: &Permissions{
				Global: "write-all",
			},
			expectError: false,
		},
		{
			name: "individual_permissions",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"contents": "read",
					"issues":   "write",
				},
			},
			expected: &Permissions{
				Contents: "read",
				Issues:   "write",
			},
			expectError: false,
		},
		{
			name: "individual_permissions_with_models",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"contents": "read",
					"models":   "read",
				},
			},
			expected: &Permissions{
				Contents: "read",
				Models:   "read",
			},
			expectError: false,
		},
		{
			name: "empty_permissions_object",
			frontmatter: map[string]any{
				"permissions": map[string]any{},
			},
			expected:    &Permissions{},
			expectError: false,
		},
		{
			name: "nil_permissions",
			frontmatter: map[string]any{
				"permissions": nil,
			},
			expected:    &Permissions{},
			expectError: false,
		},
		{
			name: "invalid_global_permission",
			frontmatter: map[string]any{
				"permissions": "invalid",
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "invalid_permission_value",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"contents": "invalid",
				},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "unknown_permission_key",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"unknown": "read",
				},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "mixed_case_permission_value",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"contents": "READ",
					"issues":   "Write",
				},
			},
			expected: &Permissions{
				Contents: "read",
				Issues:   "write",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePermissions(tt.frontmatter)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !permissionsEqual(result, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestPermissions_HasContentsAccess(t *testing.T) {
	tests := []struct {
		name        string
		permissions *Permissions
		expected    bool
	}{
		{
			name:        "empty_permissions",
			permissions: &Permissions{},
			expected:    false,
		},
		{
			name: "global_read",
			permissions: &Permissions{
				Global: "read",
			},
			expected: true,
		},
		{
			name: "global_write",
			permissions: &Permissions{
				Global: "write",
			},
			expected: true,
		},
		{
			name: "global_read_all",
			permissions: &Permissions{
				Global: "read-all",
			},
			expected: true,
		},
		{
			name: "global_write_all",
			permissions: &Permissions{
				Global: "write-all",
			},
			expected: true,
		},
		{
			name: "contents_read",
			permissions: &Permissions{
				Contents: "read",
			},
			expected: true,
		},
		{
			name: "contents_write",
			permissions: &Permissions{
				Contents: "write",
			},
			expected: true,
		},
		{
			name: "only_issues_write",
			permissions: &Permissions{
				Issues: "write",
			},
			expected: false,
		},
		{
			name: "mixed_with_contents",
			permissions: &Permissions{
				Contents: "read",
				Issues:   "write",
			},
			expected: true,
		},
		{
			name: "mixed_without_contents",
			permissions: &Permissions{
				Issues:       "write",
				PullRequests: "write",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.permissions.HasContentsAccess()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPermissions_IsEmpty(t *testing.T) {
	tests := []struct {
		name        string
		permissions *Permissions
		expected    bool
	}{
		{
			name:        "empty_permissions",
			permissions: &Permissions{},
			expected:    true,
		},
		{
			name: "global_read",
			permissions: &Permissions{
				Global: "read",
			},
			expected: false,
		},
		{
			name: "has_contents",
			permissions: &Permissions{
				Contents: "read",
			},
			expected: false,
		},
		{
			name: "has_issues",
			permissions: &Permissions{
				Issues: "write",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.permissions.IsEmpty()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPermissions_ToYAML(t *testing.T) {
	tests := []struct {
		name        string
		permissions *Permissions
		expected    []string // Multiple valid formats
	}{
		{
			name:        "empty_permissions",
			permissions: &Permissions{},
			expected:    []string{"permissions: {}"},
		},
		{
			name: "global_read",
			permissions: &Permissions{
				Global: "read",
			},
			expected: []string{"permissions: read"},
		},
		{
			name: "global_write",
			permissions: &Permissions{
				Global: "write",
			},
			expected: []string{"permissions: write"},
		},
		{
			name: "global_read_all",
			permissions: &Permissions{
				Global: "read-all",
			},
			expected: []string{"permissions: read-all"},
		},
		{
			name: "global_write_all",
			permissions: &Permissions{
				Global: "write-all",
			},
			expected: []string{"permissions: write-all"},
		},
		{
			name: "individual_permissions",
			permissions: &Permissions{
				Contents: "read",
				Issues:   "write",
			},
			expected: []string{
				"permissions:\n  contents: read\n  issues: write",
				"permissions:\n  issues: write\n  contents: read", // Order may vary
			},
		},
		{
			name: "single_permission",
			permissions: &Permissions{
				Contents: "read",
			},
			expected: []string{"permissions:\n  contents: read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.permissions.ToYAML()
			
			// Check if result matches any of the expected formats
			found := false
			for _, expected := range tt.expected {
				if result == expected {
					found = true
					break
				}
			}
			
			if !found {
				t.Errorf("Expected one of %v, got %s", tt.expected, result)
			}
		})
	}
}

// Helper function to compare Permissions structs
func permissionsEqual(a, b *Permissions) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return a.Global == b.Global &&
		a.Actions == b.Actions &&
		a.Checks == b.Checks &&
		a.Contents == b.Contents &&
		a.Deployments == b.Deployments &&
		a.Discussions == b.Discussions &&
		a.Issues == b.Issues &&
		a.Metadata == b.Metadata &&
		a.Models == b.Models &&
		a.Packages == b.Packages &&
		a.Pages == b.Pages &&
		a.PullRequests == b.PullRequests &&
		a.RepositoryProjects == b.RepositoryProjects &&
		a.SecurityEvents == b.SecurityEvents &&
		a.Statuses == b.Statuses
}