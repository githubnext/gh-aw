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
// Handler Config Field Tests (limits, labels, reviewers, assignees)
// ========================================

// TestHandlerConfigMaxValues tests max value configuration
func TestHandlerConfigMaxValues(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("10"),
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
				require.NoError(t, err)

				issueConfig, ok := config["create_issue"]
				require.True(t, ok)

				maxVal, ok := issueConfig["max"]
				require.True(t, ok)
				assert.InDelta(t, float64(10), maxVal, 0.0001)
			}
		}
	}
}

// TestHandlerConfigAllowedLabels tests allowed labels configuration
func TestHandlerConfigAllowedLabels(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				SafeOutputAllowedLabelsConfig: SafeOutputAllowedLabelsConfig{AllowedLabels: []string{"bug", "enhancement", "documentation"}},
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
				require.NoError(t, err)

				issueConfig, ok := config["create_issue"]
				require.True(t, ok)

				labels, ok := issueConfig["allowed_labels"]
				require.True(t, ok)

				labelSlice, ok := labels.([]any)
				require.True(t, ok)
				assert.Len(t, labelSlice, 3)
			}
		}
	}
}

// TestHandlerConfigReviewers tests reviewers configuration in create_pull_request
func TestHandlerConfigReviewers(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				Reviewers:     []string{"user1", "user2", "copilot"},
				TeamReviewers: []string{"team-a", "team-b"},
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

				prConfig, ok := config["create_pull_request"]
				require.True(t, ok, "Should have create_pull_request handler")

				reviewers, ok := prConfig["reviewers"]
				require.True(t, ok, "Should have reviewers field")

				reviewerSlice, ok := reviewers.([]any)
				require.True(t, ok, "Reviewers should be an array")
				assert.Len(t, reviewerSlice, 3, "Should have 3 reviewers")
				assert.Equal(t, "user1", reviewerSlice[0])
				assert.Equal(t, "user2", reviewerSlice[1])
				assert.Equal(t, "copilot", reviewerSlice[2])

				teamReviewers, ok := prConfig["team_reviewers"]
				require.True(t, ok, "Should have team_reviewers field")

				teamReviewerSlice, ok := teamReviewers.([]any)
				require.True(t, ok, "team_reviewers should be an array")
				assert.Len(t, teamReviewerSlice, 2, "Should have 2 team reviewers")
				assert.Equal(t, "team-a", teamReviewerSlice[0])
				assert.Equal(t, "team-b", teamReviewerSlice[1])
			}
		}
	}
}

func TestHandlerConfigAddReviewerTeamReviewers(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			AddReviewer: &AddReviewerConfig{
				AllowedReviewers:     []string{"user1"},
				AllowedTeamReviewers: []string{"team-a"},
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

				reviewerConfig, ok := config["add_reviewer"]
				require.True(t, ok, "Should have add_reviewer handler")

				teamReviewers, ok := reviewerConfig["allowed_team_reviewers"]
				require.True(t, ok, "Should have allowed_team_reviewers field")

				teamReviewerSlice, ok := teamReviewers.([]any)
				require.True(t, ok, "allowed_team_reviewers should be an array")
				assert.Len(t, teamReviewerSlice, 1, "Should have 1 allowed team reviewer")
				assert.Equal(t, "team-a", teamReviewerSlice[0])
			}
		}
	}
}

// TestHandlerConfigAssignees tests assignees configuration in create_pull_request
func TestHandlerConfigAssignees(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				Assignees: []string{"user1", "user2"},
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

				prConfig, ok := config["create_pull_request"]
				require.True(t, ok, "Should have create_pull_request handler")

				assignees, ok := prConfig["assignees"]
				require.True(t, ok, "Should have assignees field")

				assigneeSlice, ok := assignees.([]any)
				require.True(t, ok, "Assignees should be an array")
				assert.Len(t, assigneeSlice, 2, "Should have 2 assignees")
				assert.Equal(t, "user1", assigneeSlice[0])
				assert.Equal(t, "user2", assigneeSlice[1])
			}
		}
	}
}

// TestHandlerConfigBooleanFields tests boolean field configuration
func TestHandlerConfigBooleanFields(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		checkField  string
		checkKey    string
		expected    any // expected value in JSON (bool or string)
	}{
		{
			name: "hide older comments",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					HideOlderComments: strPtr("true"),
				},
			},
			checkField: "add_comment",
			checkKey:   "hide_older_comments",
			expected:   true,
		},
		{
			name: "add comment discussions opt-in",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					Discussions: new(true),
				},
			},
			checkField: "add_comment",
			checkKey:   "discussions",
			expected:   true,
		},
		{
			name: "close older discussions",
			safeOutputs: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{
					CloseOlderDiscussions: strPtr("true"),
				},
			},
			checkField: "create_discussion",
			checkKey:   "close_older_discussions",
			expected:   true,
		},
		{
			name: "allow empty PR",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					AllowEmpty: strPtr("true"),
				},
			},
			checkField: "create_pull_request",
			checkKey:   "allow_empty",
			expected:   true,
		},
		{
			name: "draft PR",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					Draft: strPtr("true"),
				},
			},
			checkField: "create_pull_request",
			checkKey:   "draft",
			expected:   true, // AddTemplatableBool converts "true" string to JSON boolean
		},
		{
			name: "create discussion minimum body length",
			safeOutputs: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{
					MinBodyLength: 200,
				},
			},
			checkField: "create_discussion",
			checkKey:   "min_body_length",
			expected:   float64(200),
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
						require.NoError(t, err)

						fieldConfig, ok := config[tt.checkField]
						require.True(t, ok, "Expected config for: "+tt.checkField)

						val, ok := fieldConfig[tt.checkKey]
						require.True(t, ok, "Expected key: "+tt.checkKey)
						assert.Equal(t, tt.expected, val)
					}
				}
			}
		})
	}
}

// TestHandlerConfigUpdateFields tests update field configurations
func TestHandlerConfigUpdateFields(t *testing.T) {
	tests := []struct {
		name         string
		config       *UpdateIssuesConfig
		expectedKeys []string
	}{
		{
			name: "all fields enabled",
			config: &UpdateIssuesConfig{
				Status: boolPtr(true),
				Title:  boolPtr(true),
				Body:   boolPtr(true),
			},
			expectedKeys: []string{"allow_status", "allow_title", "allow_body"},
		},
		{
			name: "only status",
			config: &UpdateIssuesConfig{
				Status: boolPtr(true),
			},
			expectedKeys: []string{"allow_status"},
		},
		{
			name: "title and body",
			config: &UpdateIssuesConfig{
				Title: boolPtr(true),
				Body:  boolPtr(true),
			},
			expectedKeys: []string{"allow_title", "allow_body"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					UpdateIssues: tt.config,
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
						require.NoError(t, err)

						updateConfig, ok := config["update_issue"]
						require.True(t, ok)

						for _, key := range tt.expectedKeys {
							_, ok := updateConfig[key]
							assert.True(t, ok, "Expected key: "+key)
						}
					}
				}
			}
		})
	}
}

func TestUpdatePullRequestUpdateBranchHandlerConfig(t *testing.T) {
	tests := []struct {
		name               string
		updateBranch       *bool
		updateBranchStacks *bool
		expectedBranch     bool
		expectedStacks     bool
	}{
		{
			name:               "defaults update_branch to false and update_branch_stacks to true",
			updateBranch:       nil,
			updateBranchStacks: nil,
			expectedBranch:     false,
			expectedStacks:     true,
		},
		{
			name:               "sets update_branch true and update_branch_stacks false when configured",
			updateBranch:       boolPtr(true),
			updateBranchStacks: boolPtr(false),
			expectedBranch:     true,
			expectedStacks:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					UpdatePullRequests: &UpdatePullRequestsConfig{
						UpdateBranch:       tt.updateBranch,
						UpdateBranchStacks: tt.updateBranchStacks,
					},
				},
			}

			var steps []string
			compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)
			foundHandlerConfig := false

			for _, step := range steps {
				if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
					foundHandlerConfig = true
					parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
					if len(parts) == 2 {
						jsonStr := strings.TrimSpace(parts[1])
						jsonStr = strings.Trim(jsonStr, "\"")
						jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

						var config map[string]map[string]any
						err := json.Unmarshal([]byte(jsonStr), &config)
						require.NoError(t, err)

						updatePRConfig, ok := config["update_pull_request"]
						require.True(t, ok, "Expected update_pull_request config")

						updateBranchValue, ok := updatePRConfig["update_branch"]
						require.True(t, ok, "Expected update_branch key in update_pull_request config")
						assert.Equal(t, tt.expectedBranch, updateBranchValue)

						updateBranchStacksValue, ok := updatePRConfig["update_branch_stacks"]
						require.True(t, ok, "Expected update_branch_stacks key in update_pull_request config")
						assert.Equal(t, tt.expectedStacks, updateBranchStacksValue)
					}
				}
			}

			require.True(t, foundHandlerConfig, "Expected GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG in generated steps")
		})
	}
}
