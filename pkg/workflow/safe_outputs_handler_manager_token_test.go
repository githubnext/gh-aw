//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHandlerManagerGitHubTokenEnvVarForCrossRepo verifies that GITHUB_TOKEN is exposed as
// an environment variable in the consolidated safe outputs handler step when create-pull-request
// or push-to-pull-request-branch is configured with a custom token. This is required so that
// the JavaScript handler's git CLI operations (dynamic checkout in multi-repo scenarios) can
// authenticate with the custom token instead of the default repo-scoped GITHUB_TOKEN.
func TestHandlerManagerGitHubTokenEnvVarForCrossRepo(t *testing.T) {
	tests := []struct {
		name                    string
		frontmatter             map[string]any
		expectedGitHubTokenLine string
		shouldHaveGitHubToken   bool
	}{
		{
			name: "create-pull-request with safe-outputs github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"github-token": "${{ secrets.CROSS_REPO_PAT }}",
					"create-pull-request": map[string]any{
						"max":           10,
						"base-branch":   "main",
						"allowed-repos": []any{"Org/repo-a", "Org/repo-b"},
					},
				},
			},
			expectedGitHubTokenLine: "GITHUB_TOKEN: ${{ secrets.CROSS_REPO_PAT }}",
			shouldHaveGitHubToken:   true,
		},
		{
			name: "create-pull-request with per-config github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"create-pull-request": map[string]any{
						"github-token":  "${{ secrets.PR_PAT }}",
						"max":           5,
						"allowed-repos": []any{"Org/repo-a"},
					},
				},
			},
			expectedGitHubTokenLine: "GITHUB_TOKEN: ${{ secrets.PR_PAT }}",
			shouldHaveGitHubToken:   true,
		},
		{
			name: "push-to-pull-request-branch with safe-outputs github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"github-token": "${{ secrets.PUSH_PAT }}",
					"push-to-pull-request-branch": map[string]any{
						"max": 3,
					},
				},
			},
			expectedGitHubTokenLine: "GITHUB_TOKEN: ${{ secrets.PUSH_PAT }}",
			shouldHaveGitHubToken:   true,
		},
		{
			name: "create-pull-request without custom token - no GITHUB_TOKEN override",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"create-pull-request": map[string]any{
						"max": 1,
					},
				},
			},
			shouldHaveGitHubToken: false,
		},
		{
			name: "push-to-pull-request-branch per-config token takes precedence over safe-outputs token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"github-token": "${{ secrets.SAFE_OUTPUTS_TOKEN }}",
					"push-to-pull-request-branch": map[string]any{
						"github-token": "${{ secrets.PUSH_PAT }}",
						"max":          2,
					},
				},
			},
			expectedGitHubTokenLine: "GITHUB_TOKEN: ${{ secrets.PUSH_PAT }}",
			shouldHaveGitHubToken:   true,
		},
		{
			name: "add-comment without patches - no GITHUB_TOKEN override",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"github-token": "${{ secrets.SOME_PAT }}",
					"add-comment": map[string]any{
						"max": 5,
					},
				},
			},
			shouldHaveGitHubToken: false,
		},
		{
			name: "create-pull-request with github-app - uses minted app token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"github-app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
					"create-pull-request": map[string]any{
						"max":           5,
						"allowed-repos": []any{"Org/repo-a"},
					},
				},
			},
			expectedGitHubTokenLine: "GITHUB_TOKEN: ${{ steps.safe-outputs-app-token.outputs.token }}",
			shouldHaveGitHubToken:   true,
		},
		{
			name: "per-config github-token overrides github-app token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"github-app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
					"create-pull-request": map[string]any{
						"github-token":  "${{ secrets.CREATE_PR_PAT }}",
						"max":           5,
						"allowed-repos": []any{"Org/repo-a"},
					},
				},
			},
			expectedGitHubTokenLine: "GITHUB_TOKEN: ${{ secrets.CREATE_PR_PAT }}",
			shouldHaveGitHubToken:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Name:        "test-workflow",
				SafeOutputs: compiler.extractSafeOutputsConfig(tt.frontmatter),
			}

			steps := compiler.buildHandlerManagerStep(workflowData)
			yamlStr := strings.Join(steps, "")

			if tt.shouldHaveGitHubToken {
				assert.Contains(t, yamlStr, tt.expectedGitHubTokenLine,
					"Expected GITHUB_TOKEN env var %q to be set in handler manager step for cross-repo git operations",
					tt.expectedGitHubTokenLine)
			} else {
				assert.NotContains(t, yamlStr, "GITHUB_TOKEN:",
					"Expected GITHUB_TOKEN to NOT be explicitly set when no custom checkout token is configured")
			}
		})
	}
}

// TestHandlerManagerProjectGitHubTokenEnvVar verifies that GH_AW_PROJECT_GITHUB_TOKEN
// is exposed as an environment variable in the consolidated safe outputs handler step
// when any project-related safe output is configured
func TestHandlerManagerProjectGitHubTokenEnvVar(t *testing.T) {
	tests := []struct {
		name                string
		frontmatter         map[string]any
		expectedEnvVarValue string
		expectedWithToken   string
		shouldHaveToken     bool
	}{
		{
			name: "update-project with custom github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"update-project": map[string]any{
						"github-token": "${{ secrets.PROJECTS_PAT }}",
						"project":      "https://github.com/orgs/myorg/projects/1",
					},
				},
			},
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.PROJECTS_PAT }}",
			expectedWithToken:   "github-token: ${{ secrets.PROJECTS_PAT }}",
			shouldHaveToken:     true,
		},
		{
			name: "update-project without custom github-token (uses GH_AW_PROJECT_GITHUB_TOKEN)",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"update-project": map[string]any{
						"project": "https://github.com/orgs/myorg/projects/1",
					},
				},
			},
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
			expectedWithToken:   "github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}",
			shouldHaveToken:     true,
		},
		{
			name: "update-project with safe-outputs github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"github-token": "${{ secrets.SAFE_OUTPUTS_TOKEN }}",
					"update-project": map[string]any{
						"project": "https://github.com/orgs/myorg/projects/1",
					},
				},
			},
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.SAFE_OUTPUTS_TOKEN }}",
			expectedWithToken:   "github-token: ${{ secrets.SAFE_OUTPUTS_TOKEN }}",
			shouldHaveToken:     true,
		},
		{
			name: "create-project-status-update with custom github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"create-project-status-update": map[string]any{
						"github-token": "${{ secrets.STATUS_PAT }}",
						"project":      "https://github.com/orgs/myorg/projects/2",
					},
				},
			},
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.STATUS_PAT }}",
			expectedWithToken:   "github-token: ${{ secrets.STATUS_PAT }}",
			shouldHaveToken:     true,
		},
		{
			name: "create-project with custom github-token (no project URL)",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"create-project": map[string]any{
						"github-token": "${{ secrets.CREATE_PAT }}",
					},
				},
			},
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.CREATE_PAT }}",
			expectedWithToken:   "github-token: ${{ secrets.CREATE_PAT }}",
			shouldHaveToken:     true,
		},
		{
			name: "multiple project configs - update-project takes precedence",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"update-project": map[string]any{
						"github-token": "${{ secrets.UPDATE_PAT }}",
						"project":      "https://github.com/orgs/myorg/projects/1",
					},
					"create-project-status-update": map[string]any{
						"github-token": "${{ secrets.STATUS_PAT }}",
						"project":      "https://github.com/orgs/myorg/projects/2",
					},
					"create-project": map[string]any{
						"github-token": "${{ secrets.CREATE_PAT }}",
					},
				},
			},
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.UPDATE_PAT }}",
			expectedWithToken:   "github-token: ${{ secrets.UPDATE_PAT }}",
			shouldHaveToken:     true,
		},
		{
			name: "no project configs - no token set",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{
						"max": 5,
					},
				},
			},
			shouldHaveToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			// Parse frontmatter
			workflowData := &WorkflowData{
				Name:        "test-workflow",
				SafeOutputs: compiler.extractSafeOutputsConfig(tt.frontmatter),
			}

			// Build the handler manager step
			steps := compiler.buildHandlerManagerStep(workflowData)
			yamlStr := strings.Join(steps, "")

			if tt.shouldHaveToken {
				// Check that the environment variable is present with the expected value
				assert.Contains(t, yamlStr, tt.expectedEnvVarValue,
					"Expected environment variable %q to be set in handler manager step",
					tt.expectedEnvVarValue)

				// Check that the github-script token matches the effective project token
				assert.Contains(t, yamlStr, tt.expectedWithToken,
					"Expected github-script token %q to be set in handler manager step",
					tt.expectedWithToken)
			} else {
				// Check that GH_AW_PROJECT_GITHUB_TOKEN is NOT set
				assert.NotContains(t, yamlStr, "GH_AW_PROJECT_GITHUB_TOKEN",
					"Expected GH_AW_PROJECT_GITHUB_TOKEN to NOT be set when no project configs are present")
			}
		})
	}
}
