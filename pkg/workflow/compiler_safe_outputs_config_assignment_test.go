//go:build !integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// Auto-Enabled Handlers and Assignment Tests
// ========================================

// TestAutoEnabledHandlers tests that missing_tool and missing_data
// are automatically enabled even when not explicitly configured.
// Note: noop is NOT included here because it is always processed by a dedicated
// standalone step (see notify_comment.go) and should never be in the handler manager config.
func TestAutoEnabledHandlers(t *testing.T) {
	tests := []struct {
		name         string
		safeOutputs  *SafeOutputsConfig
		expectedKeys []string
	}{
		{
			name: "missing_tool auto-enabled",
			safeOutputs: &SafeOutputsConfig{
				MissingTool: &MissingToolConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			expectedKeys: []string{"missing_tool"},
		},
		{
			name: "missing_data auto-enabled",
			safeOutputs: &SafeOutputsConfig{
				MissingData: &MissingDataConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			expectedKeys: []string{"missing_data"},
		},
		{
			name: "all auto-enabled handlers together",
			safeOutputs: &SafeOutputsConfig{
				MissingTool: &MissingToolConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
				MissingData: &MissingDataConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			expectedKeys: []string{"missing_tool", "missing_data"},
		},
		{
			name: "auto-enabled with other handlers",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Test] ",
				},
				MissingTool: &MissingToolConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			expectedKeys: []string{"create_issue", "missing_tool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: tt.safeOutputs,
			}

			var steps []string
			compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

			require.NotEmpty(t, steps, "Steps should be generated")

			// Extract and validate JSON
			for _, step := range steps {
				if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
					parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
					if len(parts) == 2 {
						jsonStr := strings.TrimSpace(parts[1])
						jsonStr = strings.Trim(jsonStr, "\"")
						jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

						var config map[string]map[string]any
						err := json.Unmarshal([]byte(jsonStr), &config)
						require.NoError(t, err, "Config JSON should be valid")

						// Check that all expected keys are present
						for _, key := range tt.expectedKeys {
							_, ok := config[key]
							assert.True(t, ok, "Expected auto-enabled handler: "+key)
						}
					}
				}
			}
		})
	}
}

// TestCreatePullRequestBaseBranch tests the base-branch field configuration
func TestCreatePullRequestBaseBranch(t *testing.T) {
	tests := []struct {
		name                             string
		baseBranch                       string
		allowedBaseBranches              []string
		allowedBranches                  []string
		expectedBaseBranch               string
		shouldHaveBaseBranchKey          bool
		expectedAllowedBaseBranches      []string
		shouldHaveAllowedBaseBranchesKey bool
		expectedAllowedBranches          []string
		shouldHaveAllowedBranchesKey     bool
	}{
		{
			name:                    "custom base branch",
			baseBranch:              "vnext",
			expectedBaseBranch:      "vnext",
			shouldHaveBaseBranchKey: true,
		},
		{
			name:                    "default base branch - no key in config",
			baseBranch:              "",
			expectedBaseBranch:      "",
			shouldHaveBaseBranchKey: false, // JS resolves dynamically
		},
		{
			name:                    "branch with slash",
			baseBranch:              "release/v1.0",
			expectedBaseBranch:      "release/v1.0",
			shouldHaveBaseBranchKey: true,
		},
		{
			name:                             "allowed base branches list",
			baseBranch:                       "main",
			allowedBaseBranches:              []string{"release/*", "main"},
			allowedBranches:                  []string{"feature/*", "fix/*"},
			expectedBaseBranch:               "main",
			shouldHaveBaseBranchKey:          true,
			expectedAllowedBaseBranches:      []string{"release/*", "main"},
			shouldHaveAllowedBaseBranchesKey: true,
			expectedAllowedBranches:          []string{"feature/*", "fix/*"},
			shouldHaveAllowedBranchesKey:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreatePullRequests: &CreatePullRequestsConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: strPtr("1"),
						},
						BaseBranch:          tt.baseBranch,
						AllowedBaseBranches: tt.allowedBaseBranches,
						AllowedBranches:     tt.allowedBranches,
					},
				},
			}

			var steps []string
			compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

			require.NotEmpty(t, steps, "Steps should be generated")

			// Extract and validate JSON
			for _, step := range steps {
				if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
					parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
					if len(parts) == 2 {
						jsonStr := strings.TrimSpace(parts[1])
						jsonStr = strings.Trim(jsonStr, "\"")
						jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

						var config map[string]map[string]any
						err := json.Unmarshal([]byte(jsonStr), &config)
						require.NoError(t, err, "Config JSON should be valid")

						prConfig, ok := config["create_pull_request"]
						require.True(t, ok, "create_pull_request config should exist")

						baseBranch, ok := prConfig["base_branch"]
						if tt.shouldHaveBaseBranchKey {
							require.True(t, ok, "base_branch should be in config")
							assert.Equal(t, tt.expectedBaseBranch, baseBranch, "base_branch should match expected value")
						} else {
							require.False(t, ok, "base_branch should NOT be in config when no custom value set")
						}

						allowedBaseBranches, ok := prConfig["allowed_base_branches"]
						if tt.shouldHaveAllowedBaseBranchesKey {
							require.True(t, ok, "allowed_base_branches should be in config")
							allowedSlice, ok := allowedBaseBranches.([]any)
							require.True(t, ok, "allowed_base_branches should be an array")
							require.Len(t, allowedSlice, len(tt.expectedAllowedBaseBranches), "allowed_base_branches length should match")
							for i, expected := range tt.expectedAllowedBaseBranches {
								assert.Equal(t, expected, allowedSlice[i], "allowed_base_branches element should match")
							}
						} else {
							require.False(t, ok, "allowed_base_branches should NOT be in config when no values set")
						}

						allowedBranches, ok := prConfig["allowed_branches"]
						if tt.shouldHaveAllowedBranchesKey {
							require.True(t, ok, "allowed_branches should be in config")
							allowedSlice, ok := allowedBranches.([]any)
							require.True(t, ok, "allowed_branches should be an array")
							require.Len(t, allowedSlice, len(tt.expectedAllowedBranches), "allowed_branches length should match")
							for i, expected := range tt.expectedAllowedBranches {
								assert.Equal(t, expected, allowedSlice[i], "allowed_branches element should match")
							}
						} else {
							require.False(t, ok, "allowed_branches should NOT be in config when no values set")
						}
					}
				}
			}
		})
	}
}

func TestCreatePullRequestFallbackLabels(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				FallbackLabels: []string{"failure", "automated"},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)
	require.NotEmpty(t, steps, "Steps should be generated")
	validated := false

	for _, step := range steps {
		if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
			if len(parts) != 2 {
				continue
			}

			jsonStr := strings.TrimSpace(parts[1])
			jsonStr = strings.Trim(jsonStr, "\"")
			jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

			var config map[string]map[string]any
			err := json.Unmarshal([]byte(jsonStr), &config)
			require.NoError(t, err, "Config JSON should be valid")

			prConfig, ok := config["create_pull_request"]
			require.True(t, ok, "create_pull_request config should exist")

			fallbackLabelsRaw, ok := prConfig["fallback_labels"]
			require.True(t, ok, "fallback_labels should be in config")

			fallbackLabels, ok := fallbackLabelsRaw.([]any)
			require.True(t, ok, "fallback_labels should be an array")
			require.Len(t, fallbackLabels, 2, "fallback_labels should have expected length")
			assert.Equal(t, "failure", fallbackLabels[0], "first fallback label should match")
			assert.Equal(t, "automated", fallbackLabels[1], "second fallback label should match")
			validated = true
			break
		}
	}

	require.True(t, validated, "fallback_labels validation should run when handler config env var is present")
}

// TestHandlerConfigAssignToUser tests assign_to_user configuration
func TestHandlerConfigAssignToUser(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			AssignToUser: &AssignToUserConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("5"),
				},
				SafeOutputTargetConfig: SafeOutputTargetConfig{
					Target:         "issues",
					TargetRepoSlug: "org/target-repo",
					AllowedRepos:   []string{"org/repo1", "org/repo2"},
				},
				SafeOutputAllowBlockConfig: SafeOutputAllowBlockConfig{
					Allowed: []string{"user1", "user2", "copilot"},
				},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	// Extract and validate JSON
	for _, step := range steps {
		if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
			if len(parts) == 2 {
				jsonStr := strings.TrimSpace(parts[1])
				jsonStr = strings.Trim(jsonStr, "\"")
				jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

				var config map[string]map[string]any
				err := json.Unmarshal([]byte(jsonStr), &config)
				require.NoError(t, err, "Handler config JSON should be valid")

				assignConfig, ok := config["assign_to_user"]
				require.True(t, ok, "Should have assign_to_user handler")

				// Check max
				max, ok := assignConfig["max"]
				require.True(t, ok, "Should have max field")
				assert.InDelta(t, 5.0, max, 0.001, "Max should be 5")

				// Check allowed users
				allowed, ok := assignConfig["allowed"]
				require.True(t, ok, "Should have allowed field")
				allowedSlice, ok := allowed.([]any)
				require.True(t, ok, "Allowed should be an array")
				assert.Len(t, allowedSlice, 3, "Should have 3 allowed users")
				assert.Equal(t, "user1", allowedSlice[0])
				assert.Equal(t, "user2", allowedSlice[1])
				assert.Equal(t, "copilot", allowedSlice[2])

				// Check target
				target, ok := assignConfig["target"]
				require.True(t, ok, "Should have target field")
				assert.Equal(t, "issues", target)

				// Check target-repo
				targetRepo, ok := assignConfig["target-repo"]
				require.True(t, ok, "Should have target-repo field")
				assert.Equal(t, "org/target-repo", targetRepo)

				// Check allowed_repos
				allowedRepos, ok := assignConfig["allowed_repos"]
				require.True(t, ok, "Should have allowed_repos field")
				allowedReposSlice, ok := allowedRepos.([]any)
				require.True(t, ok, "Allowed repos should be an array")
				assert.Len(t, allowedReposSlice, 2, "Should have 2 allowed repos")

				// unassign_first should not be present when false/omitted
				_, hasUnassignFirst := assignConfig["unassign_first"]
				assert.False(t, hasUnassignFirst, "Should not have unassign_first field when false")
			}
		}
	}
}

// TestHandlerConfigAssignToUserWithUnassignFirst tests assign_to_user configuration with unassign_first enabled
func TestHandlerConfigAssignToUserWithUnassignFirst(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			AssignToUser: &AssignToUserConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("3"),
				},
				UnassignFirst: testStringPtr("true"),
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	// Extract and validate JSON
	for _, step := range steps {
		if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
			if len(parts) == 2 {
				jsonStr := strings.TrimSpace(parts[1])
				jsonStr = strings.Trim(jsonStr, "\"")
				jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

				var config map[string]map[string]any
				err := json.Unmarshal([]byte(jsonStr), &config)
				require.NoError(t, err, "Handler config JSON should be valid")

				assignConfig, ok := config["assign_to_user"]
				require.True(t, ok, "Should have assign_to_user handler")

				// Check max
				max, ok := assignConfig["max"]
				require.True(t, ok, "Should have max field")
				assert.InDelta(t, 3.0, max, 0.001, "Max should be 3")

				// Check unassign_first
				unassignFirst, ok := assignConfig["unassign_first"]
				require.True(t, ok, "Should have unassign_first field")
				unassignFirstBool, ok := unassignFirst.(bool)
				require.True(t, ok, "unassign_first should be a bool")
				assert.True(t, unassignFirstBool, "unassign_first should be true")
			}
		}
	}
}

// TestHandlerConfigUnassignFromUser tests unassign_from_user configuration
func TestHandlerConfigUnassignFromUser(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			UnassignFromUser: &UnassignFromUserConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("10"),
				},
				SafeOutputTargetConfig: SafeOutputTargetConfig{
					Target:         "issues",
					TargetRepoSlug: "org/target-repo",
					AllowedRepos:   []string{"org/repo1"},
				},
				SafeOutputAllowBlockConfig: SafeOutputAllowBlockConfig{
					Allowed: []string{"githubactionagent", "bot-user"},
				},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	// Extract and validate JSON
	for _, step := range steps {
		if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
			if len(parts) == 2 {
				jsonStr := strings.TrimSpace(parts[1])
				jsonStr = strings.Trim(jsonStr, "\"")
				jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

				var config map[string]map[string]any
				err := json.Unmarshal([]byte(jsonStr), &config)
				require.NoError(t, err, "Handler config JSON should be valid")

				unassignConfig, ok := config["unassign_from_user"]
				require.True(t, ok, "Should have unassign_from_user handler")

				// Check max
				max, ok := unassignConfig["max"]
				require.True(t, ok, "Should have max field")
				assert.InDelta(t, 10.0, max, 0.001, "Max should be 10")

				// Check allowed users
				allowed, ok := unassignConfig["allowed"]
				require.True(t, ok, "Should have allowed field")
				allowedSlice, ok := allowed.([]any)
				require.True(t, ok, "Allowed should be an array")
				assert.Len(t, allowedSlice, 2, "Should have 2 allowed users")
				assert.Equal(t, "githubactionagent", allowedSlice[0])
				assert.Equal(t, "bot-user", allowedSlice[1])

				// Check target
				target, ok := unassignConfig["target"]
				require.True(t, ok, "Should have target field")
				assert.Equal(t, "issues", target)

				// Check target-repo
				targetRepo, ok := unassignConfig["target-repo"]
				require.True(t, ok, "Should have target-repo field")
				assert.Equal(t, "org/target-repo", targetRepo)

				// Check allowed_repos
				allowedRepos, ok := unassignConfig["allowed_repos"]
				require.True(t, ok, "Should have allowed_repos field")
				allowedReposSlice, ok := allowedRepos.([]any)
				require.True(t, ok, "Allowed repos should be an array")
				assert.Len(t, allowedReposSlice, 1, "Should have 1 allowed repo")
			}
		}
	}
}

// TestHandlerConfigAssignToUserWithBlocked tests that blocked patterns are included in assign_to_user handler config
func TestHandlerConfigAssignToUserWithBlocked(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			AssignToUser: &AssignToUserConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				SafeOutputTargetConfig: SafeOutputTargetConfig{
					Target:         "*",
					TargetRepoSlug: "microsoft/vscode",
				},
				SafeOutputAllowBlockConfig: SafeOutputAllowBlockConfig{
					Blocked: []string{"copilot", "*[bot]"},
				},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	for _, step := range steps {
		if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
			if len(parts) == 2 {
				jsonStr := strings.TrimSpace(parts[1])
				jsonStr = strings.Trim(jsonStr, "\"")
				jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

				var config map[string]map[string]any
				err := json.Unmarshal([]byte(jsonStr), &config)
				require.NoError(t, err, "Handler config JSON should be valid")

				assignConfig, ok := config["assign_to_user"]
				require.True(t, ok, "Should have assign_to_user handler")

				blocked, ok := assignConfig["blocked"]
				require.True(t, ok, "Should have blocked field")
				blockedSlice, ok := blocked.([]any)
				require.True(t, ok, "Blocked should be an array")
				assert.Len(t, blockedSlice, 2, "Should have 2 blocked patterns")
				assert.Equal(t, "copilot", blockedSlice[0], "First blocked pattern should be copilot")
				assert.Equal(t, "*[bot]", blockedSlice[1], "Second blocked pattern should be *[bot]")
			}
		}
	}
}

// TestHandlerConfigUnassignFromUserWithBlocked tests that blocked patterns are included in unassign_from_user handler config
func TestHandlerConfigUnassignFromUserWithBlocked(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			UnassignFromUser: &UnassignFromUserConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("2"),
				},
				SafeOutputTargetConfig: SafeOutputTargetConfig{
					Target:         "*",
					TargetRepoSlug: "microsoft/vscode",
				},
				SafeOutputAllowBlockConfig: SafeOutputAllowBlockConfig{
					Blocked: []string{"copilot", "*[bot]"},
				},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	for _, step := range steps {
		if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
			if len(parts) == 2 {
				jsonStr := strings.TrimSpace(parts[1])
				jsonStr = strings.Trim(jsonStr, "\"")
				jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

				var config map[string]map[string]any
				err := json.Unmarshal([]byte(jsonStr), &config)
				require.NoError(t, err, "Handler config JSON should be valid")

				unassignConfig, ok := config["unassign_from_user"]
				require.True(t, ok, "Should have unassign_from_user handler")

				blocked, ok := unassignConfig["blocked"]
				require.True(t, ok, "Should have blocked field")
				blockedSlice, ok := blocked.([]any)
				require.True(t, ok, "Blocked should be an array")
				assert.Len(t, blockedSlice, 2, "Should have 2 blocked patterns")
				assert.Equal(t, "copilot", blockedSlice[0], "First blocked pattern should be copilot")
				assert.Equal(t, "*[bot]", blockedSlice[1], "Second blocked pattern should be *[bot]")
			}
		}
	}
}
