//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestValidateGitHubAppOnlyPermissions(t *testing.T) {
	tests := []struct {
		name          string
		permissions   string
		parsedTools   *ToolsConfig
		safeOutputs   *SafeOutputsConfig
		activationApp *GitHubAppConfig
		shouldError   bool
		errorContains string
	}{
		{
			name:        "no permissions - should pass",
			permissions: "",
			shouldError: false,
		},
		{
			name:        "GitHub Actions-only permissions - should pass without github-app",
			permissions: "permissions:\n  contents: read\n  issues: write",
			shouldError: false,
		},
		{
			name:          "organization-projects (App-only) without github-app - should error",
			permissions:   "permissions:\n  organization-projects: write",
			shouldError:   true,
			errorContains: "GitHub App-only permissions require a GitHub App",
		},
		{
			name:          "members permission without github-app - should error",
			permissions:   "permissions:\n  members: read",
			shouldError:   true,
			errorContains: "GitHub App-only permissions require a GitHub App",
		},
		{
			name:          "administration permission without github-app - should error",
			permissions:   "permissions:\n  administration: read",
			shouldError:   true,
			errorContains: "GitHub App-only permissions require a GitHub App",
		},
		{
			name:          "secrets permission without github-app - should error",
			permissions:   "permissions:\n  secrets: read",
			shouldError:   true,
			errorContains: "GitHub App-only permissions require a GitHub App",
		},
		{
			name:        "members permission with tools.github.github-app - should pass",
			permissions: "permissions:\n  members: read",
			parsedTools: &ToolsConfig{
				GitHub: &GitHubToolConfig{
					GitHubApp: &GitHubAppConfig{
						AppID:      "${{ vars.APP_ID }}",
						PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
			shouldError: false,
		},
		{
			name:        "members permission with safe-outputs.github-app - should pass",
			permissions: "permissions:\n  members: read",
			safeOutputs: &SafeOutputsConfig{
				GitHubApp: &GitHubAppConfig{
					AppID:      "${{ vars.APP_ID }}",
					PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
				},
			},
			shouldError: false,
		},
		{
			name:        "members permission with activation github-app - should pass",
			permissions: "permissions:\n  members: read",
			activationApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
			shouldError: false,
		},
		{
			name:          "organization-secrets permission without github-app - should error",
			permissions:   "permissions:\n  organization-secrets: read",
			shouldError:   true,
			errorContains: "organization-secrets",
		},
		{
			name:          "workflows permission without github-app - should error",
			permissions:   "permissions:\n  workflows: write",
			shouldError:   true,
			errorContains: "workflows",
		},
		{
			name:          "vulnerability-alerts permission without github-app - should error",
			permissions:   "permissions:\n  vulnerability-alerts: read",
			shouldError:   true,
			errorContains: "vulnerability-alerts",
		},
		{
			name:        "mixed Actions and App-only permissions with github-app - should pass",
			permissions: "permissions:\n  contents: read\n  members: read\n  administration: read",
			parsedTools: &ToolsConfig{
				GitHub: &GitHubToolConfig{
					GitHubApp: &GitHubAppConfig{
						AppID:      "${{ vars.APP_ID }}",
						PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
			shouldError: false,
		},
		{
			name:          "mixed Actions and App-only permissions without github-app - should error",
			permissions:   "permissions:\n  contents: read\n  members: read",
			shouldError:   true,
			errorContains: "GitHub App-only permissions require a GitHub App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Permissions:         tt.permissions,
				ParsedTools:         tt.parsedTools,
				SafeOutputs:         tt.safeOutputs,
				ActivationGitHubApp: tt.activationApp,
			}

			err := validateGitHubAppOnlyPermissions(workflowData)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestIsGitHubAppOnlyScope(t *testing.T) {
	tests := []struct {
		scope    PermissionScope
		expected bool
	}{
		// GitHub Actions scopes - should NOT be GitHub App-only
		{PermissionActions, false},
		{PermissionChecks, false},
		{PermissionContents, false},
		{PermissionDeployments, false},
		{PermissionIssues, false},
		{PermissionPackages, false},
		{PermissionPages, false},
		{PermissionPullRequests, false},
		{PermissionSecurityEvents, false},
		{PermissionStatuses, false},
		{PermissionDiscussions, false},
		// organization-projects is a GitHub App-only scope (not in GitHub Actions GITHUB_TOKEN)
		{PermissionOrganizationProj, true},
		// GitHub App-only scopes - should return true
		{PermissionAdministration, true},
		{PermissionMembers, true},
		{PermissionOrganizationAdministration, true},
		{PermissionSecrets, true},
		{PermissionEnvironments, true},
		{PermissionGitSigning, true},
		{PermissionTeamDiscussions, true},
		{PermissionVulnerabilityAlerts, true},
		{PermissionWorkflows, true},
		{PermissionRepositoryHooks, true},
		{PermissionOrganizationHooks, true},
		{PermissionOrganizationMembers, true},
		{PermissionOrganizationPackages, true},
		{PermissionOrganizationSecrets, true},
		{PermissionOrganizationSelfHostedRunners, true},
		{PermissionSingleFile, true},
		{PermissionCodespaces, true},
		{PermissionDependabotSecrets, true},
		{PermissionEmailAddresses, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			result := IsGitHubAppOnlyScope(tt.scope)
			if result != tt.expected {
				t.Errorf("IsGitHubAppOnlyScope(%q) = %v, want %v", tt.scope, result, tt.expected)
			}
		})
	}
}

func TestGetAllGitHubAppOnlyScopes(t *testing.T) {
	scopes := GetAllGitHubAppOnlyScopes()
	if len(scopes) == 0 {
		t.Error("GetAllGitHubAppOnlyScopes should return at least one scope")
	}

	// Verify some key scopes are included
	keyScopes := []PermissionScope{
		PermissionAdministration,
		PermissionMembers,
		PermissionOrganizationAdministration,
		PermissionSecrets,
		PermissionEnvironments,
		PermissionWorkflows,
		PermissionVulnerabilityAlerts,
		PermissionOrganizationSecrets,
		PermissionOrganizationPackages,
	}

	scopeSet := make(map[PermissionScope]bool)
	for _, s := range scopes {
		scopeSet[s] = true
	}

	for _, expected := range keyScopes {
		if !scopeSet[expected] {
			t.Errorf("Expected scope %q to be in GetAllGitHubAppOnlyScopes()", expected)
		}
	}

	// Verify that GitHub Actions scopes are NOT included
	actionScopes := []PermissionScope{
		PermissionContents,
		PermissionIssues,
		PermissionPullRequests,
		PermissionChecks,
		PermissionIdToken,
	}
	for _, notExpected := range actionScopes {
		if scopeSet[notExpected] {
			t.Errorf("Expected scope %q to NOT be in GetAllGitHubAppOnlyScopes()", notExpected)
		}
	}
}

func TestGitHubAppOnlyPermissionsRenderToYAML(t *testing.T) {
	tests := []struct {
		name          string
		permissions   *Permissions
		shouldContain []string
		shouldSkip    []string
	}{
		{
			name: "members permission is skipped in GitHub Actions YAML",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionMembers, PermissionRead)
				return p
			}(),
			shouldSkip: []string{"members: read"},
		},
		{
			name: "administration permission is skipped in GitHub Actions YAML",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionAdministration, PermissionRead)
				return p
			}(),
			shouldSkip: []string{"administration: read"},
		},
		{
			name: "contents permission IS included in GitHub Actions YAML",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionContents, PermissionRead)
				return p
			}(),
			shouldContain: []string{"contents: read"},
		},
		{
			name: "mixed permissions - only Actions scopes rendered",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionContents, PermissionRead)
				p.Set(PermissionIssues, PermissionRead)
				p.Set(PermissionMembers, PermissionRead)
				p.Set(PermissionAdministration, PermissionRead)
				return p
			}(),
			shouldContain: []string{"contents: read", "issues: read"},
			shouldSkip:    []string{"members: read", "administration: read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := tt.permissions.RenderToYAML()

			for _, expected := range tt.shouldContain {
				if !strings.Contains(yaml, expected) {
					t.Errorf("Expected YAML to contain %q, but got:\n%s", expected, yaml)
				}
			}

			for _, notExpected := range tt.shouldSkip {
				if strings.Contains(yaml, notExpected) {
					t.Errorf("Expected YAML to NOT contain %q, but got:\n%s", notExpected, yaml)
				}
			}
		})
	}
}

func TestConvertPermissionsToAppTokenFields_GitHubAppOnly(t *testing.T) {
	tests := []struct {
		name           string
		permissions    *Permissions
		expectedFields map[string]string
		absentFields   []string
	}{
		{
			name: "members permission maps to permission-members",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionMembers, PermissionRead)
				return p
			}(),
			expectedFields: map[string]string{
				"permission-members": "read",
			},
		},
		{
			name: "administration permission maps to permission-administration",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionAdministration, PermissionRead)
				return p
			}(),
			expectedFields: map[string]string{
				"permission-administration": "read",
			},
		},
		{
			name: "organization-secrets permission maps correctly",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionOrganizationSecrets, PermissionRead)
				return p
			}(),
			expectedFields: map[string]string{
				"permission-organization-secrets": "read",
			},
		},
		{
			name: "workflows permission maps to permission-workflows",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionWorkflows, PermissionWrite)
				return p
			}(),
			expectedFields: map[string]string{
				"permission-workflows": "write",
			},
		},
		{
			name: "vulnerability-alerts maps correctly",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionVulnerabilityAlerts, PermissionRead)
				return p
			}(),
			expectedFields: map[string]string{
				"permission-vulnerability-alerts": "read",
			},
		},
		{
			name: "models permission is NOT mapped (no GitHub App equivalent)",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionModels, PermissionRead)
				return p
			}(),
			absentFields: []string{"permission-models"},
		},
		{
			name: "id-token permission is NOT mapped (not applicable to GitHub Apps)",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionIdToken, PermissionWrite)
				return p
			}(),
			absentFields: []string{"permission-id-token"},
		},
		{
			name: "organization-packages permission maps correctly",
			permissions: func() *Permissions {
				p := NewPermissions()
				p.Set(PermissionOrganizationPackages, PermissionRead)
				return p
			}(),
			expectedFields: map[string]string{
				"permission-organization-packages": "read",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := convertPermissionsToAppTokenFields(tt.permissions)

			for key, expectedValue := range tt.expectedFields {
				if actualValue, exists := fields[key]; !exists {
					t.Errorf("Expected field %q to be present, but it was not. Got fields: %v", key, fields)
				} else if actualValue != expectedValue {
					t.Errorf("Expected field %q = %q, got %q", key, expectedValue, actualValue)
				}
			}

			for _, absentKey := range tt.absentFields {
				if _, exists := fields[absentKey]; exists {
					t.Errorf("Expected field %q to be absent, but it was present with value %q", absentKey, fields[absentKey])
				}
			}
		})
	}
}
