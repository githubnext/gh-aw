package workflow

import (
	"strings"
	"testing"
)

func TestCollectRequiredPermissions(t *testing.T) {
	tests := []struct {
		name     string
		toolsets []string
		readOnly bool
		expected map[PermissionScope]PermissionLevel
	}{
		{
			name:     "Context toolset requires no permissions",
			toolsets: []string{"context"},
			readOnly: false,
			expected: map[PermissionScope]PermissionLevel{},
		},
		{
			name:     "Repos toolset in read-write mode",
			toolsets: []string{"repos"},
			readOnly: false,
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionWrite,
			},
		},
		{
			name:     "Repos toolset in read-only mode",
			toolsets: []string{"repos"},
			readOnly: true,
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
			},
		},
		{
			name:     "Issues toolset in read-write mode",
			toolsets: []string{"issues"},
			readOnly: false,
			expected: map[PermissionScope]PermissionLevel{
				PermissionIssues: PermissionWrite,
			},
		},
		{
			name:     "Multiple toolsets",
			toolsets: []string{"repos", "issues", "pull_requests"},
			readOnly: false,
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name:     "Default toolsets in read-write mode",
			toolsets: DefaultGitHubToolsets,
			readOnly: false,
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name:     "Actions toolset (read-only)",
			toolsets: []string{"actions"},
			readOnly: false,
			expected: map[PermissionScope]PermissionLevel{
				PermissionActions: PermissionRead,
			},
		},
		{
			name:     "Code security toolset",
			toolsets: []string{"code_security"},
			readOnly: false,
			expected: map[PermissionScope]PermissionLevel{
				PermissionSecurityEvents: PermissionWrite,
			},
		},
		{
			name:     "Discussions toolset",
			toolsets: []string{"discussions"},
			readOnly: false,
			expected: map[PermissionScope]PermissionLevel{
				PermissionDiscussions: PermissionWrite,
			},
		},
		{
			name:     "Projects toolset",
			toolsets: []string{"projects"},
			readOnly: false,
			expected: map[PermissionScope]PermissionLevel{
				PermissionRepositoryProj: PermissionWrite,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectRequiredPermissions(tt.toolsets, tt.readOnly)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d permissions, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for scope, expectedLevel := range tt.expected {
				actualLevel, found := result[scope]
				if !found {
					t.Errorf("Expected permission %s not found in result", scope)
					continue
				}
				if actualLevel != expectedLevel {
					t.Errorf("Permission %s: expected level %s, got %s", scope, expectedLevel, actualLevel)
				}
			}
		})
	}
}

func TestValidatePermissions_MissingPermissions(t *testing.T) {
	tests := []struct {
		name               string
		permissions        *Permissions
		githubTool         any
		expectMissing      map[PermissionScope]PermissionLevel
		expectMissingCount int
		expectExcessCount  int
		expectHasIssues    bool
	}{
		{
			name:               "No GitHub tool configured",
			permissions:        NewPermissions(),
			githubTool:         nil,
			expectMissing:      map[PermissionScope]PermissionLevel{},
			expectMissingCount: 0,
			expectExcessCount:  0,
			expectHasIssues:    false,
		},
		{
			name:        "Default toolsets with no permissions",
			permissions: NewPermissions(),
			githubTool: map[string]any{
				"toolsets": []string{"default"},
			},
			expectMissingCount: 3, // contents, issues, pull-requests
			expectExcessCount:  0,
			expectHasIssues:    true,
		},
		{
			name: "Default toolsets with all required permissions",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"default"},
				"read-only": false,
			},
			expectMissingCount: 0,
			expectExcessCount:  0,
			expectHasIssues:    false,
		},
		{
			name: "Default toolsets with only read permissions (missing write)",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionRead,
				PermissionPullRequests: PermissionRead,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"default"},
				"read-only": false, // Need write permissions
			},
			expectMissingCount: 3, // All need write
			expectExcessCount:  0,
			expectHasIssues:    true,
		},
		{
			name: "Read-only mode with read permissions",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionRead,
				PermissionPullRequests: PermissionRead,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"default"},
				"read-only": true,
			},
			expectMissingCount: 0,
			expectExcessCount:  0,
			expectHasIssues:    false,
		},
		{
			name: "Specific toolsets with partial permissions",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionWrite,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"repos", "issues"},
				"read-only": false,
			},
			expectMissingCount: 1, // Missing issues: write
			expectExcessCount:  0,
			expectHasIssues:    true,
		},
		{
			name: "Actions toolset with read permission",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionActions: PermissionRead,
			}),
			githubTool: map[string]any{
				"toolsets": []string{"actions"},
			},
			expectMissingCount: 0,
			expectExcessCount:  0,
			expectHasIssues:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePermissions(tt.permissions, tt.githubTool)

			if len(result.MissingPermissions) != tt.expectMissingCount {
				t.Errorf("Expected %d missing permissions, got %d: %v",
					tt.expectMissingCount, len(result.MissingPermissions), result.MissingPermissions)
			}

			if len(result.ExcessPermissions) != tt.expectExcessCount {
				t.Errorf("Expected %d excess permissions, got %d: %v",
					tt.expectExcessCount, len(result.ExcessPermissions), result.ExcessPermissions)
			}

			if result.HasValidationIssues != tt.expectHasIssues {
				t.Errorf("Expected HasValidationIssues=%v, got %v", tt.expectHasIssues, result.HasValidationIssues)
			}

			if tt.expectMissing != nil {
				for scope, expectedLevel := range tt.expectMissing {
					actualLevel, found := result.MissingPermissions[scope]
					if !found {
						t.Errorf("Expected missing permission %s not found", scope)
						continue
					}
					if actualLevel != expectedLevel {
						t.Errorf("Missing permission %s: expected level %s, got %s", scope, expectedLevel, actualLevel)
					}
				}
			}
		})
	}
}

func TestValidatePermissions_ExcessPermissions(t *testing.T) {
	tests := []struct {
		name               string
		permissions        *Permissions
		githubTool         any
		expectExcessCount  int
		expectMissingCount int
		expectHasIssues    bool
	}{
		{
			name: "No excess permissions when using 'all' toolset",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionWrite,
				PermissionIssues:   PermissionWrite,
				PermissionActions:  PermissionRead,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"all"},
				"read-only": false,
			},
			expectExcessCount: 0, // 'all' disables excess checking
			// But we still expect missing permissions for toolsets not covered
			expectMissingCount: 4, // discussions, pull-requests, repository-projects, security-events
			expectHasIssues:    true,
		},
		{
			name: "Excess permissions for specific toolsets",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionWrite,
				PermissionIssues:   PermissionWrite,
				PermissionActions:  PermissionRead,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"repos"},
				"read-only": false,
			},
			expectExcessCount:  2, // issues and actions not needed
			expectMissingCount: 0,
			expectHasIssues:    true,
		},
		{
			name: "Write permission when only read needed",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionActions: PermissionWrite,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"actions"},
				"read-only": false,
			},
			expectExcessCount:  1, // actions only needs read
			expectMissingCount: 0,
			expectHasIssues:    true,
		},
		{
			name: "No excess in read-only mode with read permissions",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"repos"},
				"read-only": true,
			},
			expectExcessCount:  0,
			expectMissingCount: 0,
			expectHasIssues:    false,
		},
		{
			name: "Excess write permission in read-only mode",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionWrite,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"repos"},
				"read-only": true,
			},
			expectExcessCount:  1, // write not needed in read-only
			expectMissingCount: 0,
			expectHasIssues:    true,
		},
		{
			name: "Default toolsets with extra permissions",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
				PermissionDiscussions:  PermissionWrite,
				PermissionActions:      PermissionRead,
			}),
			githubTool: map[string]any{
				"toolsets":  []string{"default"},
				"read-only": false,
			},
			expectExcessCount:  2, // discussions and actions not in default
			expectMissingCount: 0,
			expectHasIssues:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePermissions(tt.permissions, tt.githubTool)

			if len(result.ExcessPermissions) != tt.expectExcessCount {
				t.Errorf("Expected %d excess permissions, got %d: %v",
					tt.expectExcessCount, len(result.ExcessPermissions), result.ExcessPermissions)
			}

			if len(result.MissingPermissions) != tt.expectMissingCount {
				t.Errorf("Expected %d missing permissions, got %d: %v",
					tt.expectMissingCount, len(result.MissingPermissions), result.MissingPermissions)
			}

			if result.HasValidationIssues != tt.expectHasIssues {
				t.Errorf("Expected HasValidationIssues=%v, got %v", tt.expectHasIssues, result.HasValidationIssues)
			}
		})
	}
}

func TestFormatValidationMessage(t *testing.T) {
	tests := []struct {
		name              string
		result            *PermissionsValidationResult
		strict            bool
		expectContains    []string
		expectNotContains []string
	}{
		{
			name: "No validation issues",
			result: &PermissionsValidationResult{
				HasValidationIssues: false,
			},
			strict:         false,
			expectContains: []string{},
		},
		{
			name: "Missing permissions message",
			result: &PermissionsValidationResult{
				HasValidationIssues: true,
				MissingPermissions: map[PermissionScope]PermissionLevel{
					PermissionContents: PermissionWrite,
					PermissionIssues:   PermissionWrite,
				},
				MissingToolsetDetails: map[string][]PermissionScope{
					"repos":  {PermissionContents},
					"issues": {PermissionIssues},
				},
			},
			strict: false,
			expectContains: []string{
				"ERROR: Missing required permissions",
				"contents: write",
				"issues: write",
				"Required by toolsets:",
				"repos: needs contents",
				"issues: needs issues",
				"Suggested fix:",
			},
		},
		{
			name: "Excess permissions warning",
			result: &PermissionsValidationResult{
				HasValidationIssues: true,
				ExcessPermissions: map[PermissionScope]PermissionLevel{
					PermissionActions:     PermissionRead,
					PermissionDiscussions: PermissionWrite,
				},
			},
			strict: false,
			expectContains: []string{
				"WARNING: Over-provisioned permissions",
				"actions: read (not required",
				"discussions: write (not required",
				"Principle of least privilege",
			},
		},
		{
			name: "Excess permissions warning in strict mode",
			result: &PermissionsValidationResult{
				HasValidationIssues: true,
				ExcessPermissions: map[PermissionScope]PermissionLevel{
					PermissionActions: PermissionRead,
				},
			},
			strict: true,
			expectContains: []string{
				"WARNING: Over-provisioned permissions",
				"actions: read",
			},
			expectNotContains: []string{
				"ERROR:",
			},
		},
		{
			name: "Both missing and excess permissions",
			result: &PermissionsValidationResult{
				HasValidationIssues: true,
				MissingPermissions: map[PermissionScope]PermissionLevel{
					PermissionContents: PermissionWrite,
				},
				ExcessPermissions: map[PermissionScope]PermissionLevel{
					PermissionActions: PermissionRead,
				},
			},
			strict: false,
			expectContains: []string{
				"ERROR: Missing required permissions",
				"contents: write",
				"WARNING: Over-provisioned permissions",
				"actions: read",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := FormatValidationMessage(tt.result, tt.strict)

			if !tt.result.HasValidationIssues {
				if message != "" {
					t.Errorf("Expected empty message for no issues, got: %s", message)
				}
				return
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(message, expected) {
					t.Errorf("Expected message to contain %q, got:\n%s", expected, message)
				}
			}

			for _, notExpected := range tt.expectNotContains {
				if strings.Contains(message, notExpected) {
					t.Errorf("Expected message NOT to contain %q, got:\n%s", notExpected, message)
				}
			}
		})
	}
}

func TestToolsetPermissionsMapping(t *testing.T) {
	// Verify that all toolsets are properly defined
	expectedToolsets := []string{
		"context", "repos", "issues", "pull_requests", "actions",
		"code_security", "dependabot", "discussions", "experiments",
		"gists", "labels", "notifications", "orgs", "projects",
		"secret_protection", "security_advisories", "stargazers",
		"users", "search",
	}

	for _, toolset := range expectedToolsets {
		if _, exists := toolsetPermissionsMap[toolset]; !exists {
			t.Errorf("Toolset %q not defined in toolsetPermissionsMap", toolset)
		}
	}

	// Verify that default toolsets are valid
	for _, toolset := range DefaultGitHubToolsets {
		if _, exists := toolsetPermissionsMap[toolset]; !exists {
			t.Errorf("Default toolset %q not defined in toolsetPermissionsMap", toolset)
		}
	}
}

func TestValidatePermissions_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name        string
		permissions *Permissions
		githubTool  any
		expectMsg   []string
	}{
		{
			name:        "Shorthand read-all with default toolsets",
			permissions: NewPermissionsReadAll(),
			githubTool: map[string]any{
				"toolsets":  []string{"default"},
				"read-only": false,
			},
			expectMsg: []string{
				"ERROR: Missing required permissions",
				"contents: write",
				"issues: write",
				"pull-requests: write",
			},
		},
		{
			name:        "Shorthand write-all with specific toolsets",
			permissions: NewPermissionsWriteAll(),
			githubTool: map[string]any{
				"toolsets":  []string{"repos"},
				"read-only": false,
			},
			expectMsg: []string{
				"WARNING: Over-provisioned permissions",
			},
		},
		{
			name:        "All: read with discussions toolset",
			permissions: NewPermissionsAllRead(),
			githubTool: map[string]any{
				"toolsets":  []string{"discussions"},
				"read-only": false,
			},
			expectMsg: []string{
				"ERROR: Missing required permissions",
				"discussions: write",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePermissions(tt.permissions, tt.githubTool)
			message := FormatValidationMessage(result, false)

			for _, expected := range tt.expectMsg {
				if !strings.Contains(message, expected) {
					t.Errorf("Expected message to contain %q, got:\n%s", expected, message)
				}
			}
		})
	}
}
