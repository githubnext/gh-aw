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
		githubToolConfig   *GitHubToolConfig
		expectMissing      map[PermissionScope]PermissionLevel
		expectMissingCount int
		expectHasIssues    bool
	}{
		{
			name:               "No GitHub tool configured",
			permissions:        NewPermissions(),
			githubToolConfig:   nil,
			expectMissing:      map[PermissionScope]PermissionLevel{},
			expectMissingCount: 0,
			expectHasIssues:    false,
		},
		{
			name:        "Default toolsets with no permissions",
			permissions: NewPermissions(),
			githubToolConfig: &GitHubToolConfig{
				Toolset: []string{"default"},
			},
			expectMissingCount: 3, // contents, issues, pull-requests
			expectHasIssues:    true,
		},
		{
			name: "Default toolsets with all required permissions",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			}),
			githubToolConfig: &GitHubToolConfig{
				Toolset:  []string{"default"},
				ReadOnly: false,
			},
			expectMissingCount: 0,
			expectHasIssues:    false,
		},
		{
			name: "Default toolsets with only read permissions (missing write)",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionRead,
				PermissionPullRequests: PermissionRead,
			}),
			githubToolConfig: &GitHubToolConfig{
				Toolset:  []string{"default"},
				ReadOnly: false, // Need write permissions
			},
			expectMissingCount: 3, // All need write
			expectHasIssues:    true,
		},
		{
			name: "Read-only mode with read permissions",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionRead,
				PermissionPullRequests: PermissionRead,
			}),
			githubToolConfig: &GitHubToolConfig{
				Toolset:  []string{"default"},
				ReadOnly: true,
			},
			expectMissingCount: 0,
			expectHasIssues:    false,
		},
		{
			name: "Specific toolsets with partial permissions",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionWrite,
			}),
			githubToolConfig: &GitHubToolConfig{
				Toolset:  []string{"repos", "issues"},
				ReadOnly: false,
			},
			expectMissingCount: 1, // Missing issues: write
			expectHasIssues:    true,
		},
		{
			name: "Actions toolset with read permission",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionActions: PermissionRead,
			}),
			githubToolConfig: &GitHubToolConfig{
				Toolset: []string{"actions"},
			},
			expectMissingCount: 0,
			expectHasIssues:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePermissions(tt.permissions, tt.githubToolConfig)

			if len(result.MissingPermissions) != tt.expectMissingCount {
				t.Errorf("Expected %d missing permissions, got %d: %v",
					tt.expectMissingCount, len(result.MissingPermissions), result.MissingPermissions)
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
				"Missing required permissions for github toolsets:",
				"contents: write (required by repos)",
				"issues: write (required by issues)",
				"Option 1: Add missing permissions to your workflow frontmatter:",
				"Option 2: Reduce the required toolsets in your workflow:",
			},
			expectNotContains: []string{
				"ERROR:",
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
		name             string
		permissions      *Permissions
		githubToolConfig *GitHubToolConfig
		expectMsg        []string
	}{
		{
			name:        "Shorthand read-all with default toolsets",
			permissions: NewPermissionsReadAll(),
			githubToolConfig: &GitHubToolConfig{
				Toolset:  []string{"default"},
				ReadOnly: false,
			},
			expectMsg: []string{
				"Missing required permissions for github toolsets:",
				"contents: write",
				"issues: write",
				"pull-requests: write",
			},
		},
		{
			name:        "All: read with discussions toolset",
			permissions: NewPermissionsAllRead(),
			githubToolConfig: &GitHubToolConfig{
				Toolset:  []string{"discussions"},
				ReadOnly: false,
			},
			expectMsg: []string{
				"Missing required permissions for github toolsets:",
				"discussions: write",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePermissions(tt.permissions, tt.githubToolConfig)
			message := FormatValidationMessage(result, false)

			for _, expected := range tt.expectMsg {
				if !strings.Contains(message, expected) {
					t.Errorf("Expected message to contain %q, got:\n%s", expected, message)
				}
			}
		})
	}
}

// TestValidatableToolInterface verifies that ValidatableTool interface
// provides type-safe abstraction for tool configuration validation.
// This test ensures type consistency improvements enable compile-time safety.
func TestValidatableToolInterface(t *testing.T) {
	t.Parallel()

	// Test that GitHubToolConfig implements ValidatableTool interface
	var _ ValidatableTool = (*GitHubToolConfig)(nil)

	// Test GetToolsets method
	config := &GitHubToolConfig{
		Toolset: []string{"repos", "issues"},
	}

	toolsets := config.GetToolsets()
	if !strings.Contains(toolsets, "repos") {
		t.Errorf("Expected toolsets to contain 'repos', got %q", toolsets)
	}
	if !strings.Contains(toolsets, "issues") {
		t.Errorf("Expected toolsets to contain 'issues', got %q", toolsets)
	}

	// Test IsReadOnly method
	if config.IsReadOnly() {
		t.Error("Expected IsReadOnly to return false for non-read-only config")
	}

	readOnlyConfig := &GitHubToolConfig{
		Toolset:  []string{"repos"},
		ReadOnly: true,
	}

	if !readOnlyConfig.IsReadOnly() {
		t.Error("Expected IsReadOnly to return true for read-only config")
	}

	// Test nil config handling
	var nilConfig *GitHubToolConfig
	if nilConfig.GetToolsets() != "" {
		t.Error("Expected empty string for nil config GetToolsets")
	}
	if !nilConfig.IsReadOnly() {
		t.Error("Expected nil config to default to read-only for security")
	}
}

// TestValidatableToolInterfaceWithValidation verifies that ValidatableTool
// interface works correctly with the permission validation system.
func TestValidatableToolInterfaceWithValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		permissions     *Permissions
		tool            ValidatableTool
		expectMissing   int
		expectReadOnly  bool
		expectHasIssues bool
	}{
		{
			name:        "Interface with read-write config",
			permissions: NewPermissions(),
			tool: &GitHubToolConfig{
				Toolset:  []string{"repos"},
				ReadOnly: false,
			},
			expectMissing:   1, // Missing contents: write
			expectReadOnly:  false,
			expectHasIssues: true,
		},
		{
			name: "Interface with read-only config and read permissions",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
			}),
			tool: &GitHubToolConfig{
				Toolset:  []string{"repos"},
				ReadOnly: true,
			},
			expectMissing:   0,
			expectReadOnly:  true,
			expectHasIssues: false,
		},
		{
			name:            "Nil interface (no tool configured)",
			permissions:     NewPermissions(),
			tool:            nil,
			expectMissing:   0,
			expectReadOnly:  false,
			expectHasIssues: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePermissions(tt.permissions, tt.tool)

			if len(result.MissingPermissions) != tt.expectMissing {
				t.Errorf("Expected %d missing permissions, got %d",
					tt.expectMissing, len(result.MissingPermissions))
			}

			if result.ReadOnlyMode != tt.expectReadOnly {
				t.Errorf("Expected ReadOnlyMode=%v, got %v",
					tt.expectReadOnly, result.ReadOnlyMode)
			}

			if result.HasValidationIssues != tt.expectHasIssues {
				t.Errorf("Expected HasValidationIssues=%v, got %v",
					tt.expectHasIssues, result.HasValidationIssues)
			}
		})
	}
}

// TestValidatableToolTypeExpansion verifies that the interface method
// GetToolsets correctly expands default toolset references.
func TestValidatableToolTypeExpansion(t *testing.T) {
	t.Parallel()

	// Test that "default" expands to actual toolsets
	config := &GitHubToolConfig{
		Toolset: []string{"default"},
	}

	toolsets := config.GetToolsets()
	// The expandDefaultToolset function should expand "default" to actual toolsets
	if toolsets == "default" {
		t.Error("Expected 'default' to be expanded to actual toolsets")
	}

	// Test multiple toolsets
	multiConfig := &GitHubToolConfig{
		Toolset: []string{"repos", "issues", "pull_requests"},
	}

	multiToolsets := multiConfig.GetToolsets()
	if !strings.Contains(multiToolsets, "repos") {
		t.Error("Expected toolsets to contain 'repos'")
	}
	if !strings.Contains(multiToolsets, "issues") {
		t.Error("Expected toolsets to contain 'issues'")
	}
	if !strings.Contains(multiToolsets, "pull_requests") {
		t.Error("Expected toolsets to contain 'pull_requests'")
	}

	// Test empty toolset (should apply defaults)
	emptyConfig := &GitHubToolConfig{
		Toolset: []string{},
	}

	emptyToolsets := emptyConfig.GetToolsets()
	// Empty toolset should be expanded by expandDefaultToolset
	// The actual behavior depends on expandDefaultToolset implementation
	_ = emptyToolsets // Just verify it doesn't panic
}
