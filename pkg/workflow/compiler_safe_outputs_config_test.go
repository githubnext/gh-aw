//go:build !integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddHandlerManagerConfigEnvVar tests handler config JSON generation
func TestAddHandlerManagerConfigEnvVar(t *testing.T) {
	tests := []struct {
		name          string
		safeOutputs   *SafeOutputsConfig
		checkContains []string
		checkJSON     bool
		expectedKeys  []string
	}{
		{
			name: "create issue config",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
					AllowedLabels: []string{"bug", "feature"},
					Labels:        []string{"ai-generated"},
					TitlePrefix:   "[AI] ",
					Assignees:     []string{"user1"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_issue"},
		},
		{
			name: "add comment config",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("3"),
					},
					Target:            "issue",
					HideOlderComments: testStringPtr("true"),
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"add_comment"},
		},
		{
			name: "create discussion config",
			safeOutputs: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("2"),
					},
					Category:              "general",
					TitlePrefix:           "[Discussion] ",
					Labels:                []string{"ai"},
					CloseOlderDiscussions: testStringPtr("true"),
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_discussion"},
		},
		{
			name: "close issue config",
			safeOutputs: &SafeOutputsConfig{
				CloseIssues: &CloseEntityConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("10"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"close_issue"},
		},
		{
			name: "add labels config",
			safeOutputs: &SafeOutputsConfig{
				AddLabels: &AddLabelsConfig{
					Allowed: []string{"bug", "enhancement", "documentation"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"add_labels"},
		},
		{
			name: "update issue config",
			safeOutputs: &SafeOutputsConfig{
				UpdateIssues: &UpdateIssuesConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: strPtr("5"),
						},
					},
					Status: testBoolPtr(true),
					Title:  testBoolPtr(true),
					Body:   testBoolPtr(true),
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"update_issue"},
		},
		{
			name: "create pull request config",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("3"),
					},
					TitlePrefix: "[PR] ",
					Labels:      []string{"automated"},
					Draft:       testStringPtr("true"),
					IfNoChanges: "skip",
					AllowEmpty:  testStringPtr("true"),
					Expires:     7,
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_pull_request"},
		},
		{
			name: "create pull request with reviewers",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
					Reviewers: []string{"user1", "user2"},
					Labels:    []string{"automated"},
					Draft:     testStringPtr("false"),
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_pull_request"},
		},
		{
			name: "push to PR branch config",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
					Target:            "pull_request",
					TitlePrefix:       "[Update] ",
					Labels:            []string{"update"},
					IfNoChanges:       "skip",
					CommitTitleSuffix: " - Auto Update",
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"push_to_pull_request_branch"},
		},
		{
			name: "push to PR branch staged config",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Staged: true,
					},
					Target:      "*",
					IfNoChanges: "warn",
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"push_to_pull_request_branch"},
		},
		{
			name: "close pull request staged config",
			safeOutputs: &SafeOutputsConfig{
				ClosePullRequests: &ClosePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max:    strPtr("1"),
						Staged: true,
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"close_pull_request"},
		},
		{
			name: "multiple safe output types",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Issue] ",
				},
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("3"),
					},
				},
				AddLabels: &AddLabelsConfig{
					Allowed: []string{"bug"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_issue", "add_comment", "add_labels"},
		},
		{
			name: "config with target-repo",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					TargetRepoSlug: "org/repo",
					TitlePrefix:    "[Test] ",
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_issue"},
		},
		{
			name: "config with allowed repos",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					AllowedRepos: []string{"org/repo1", "org/repo2"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_issue"},
		},
		{
			name: "call_workflow config",
			safeOutputs: &SafeOutputsConfig{
				CallWorkflow: &CallWorkflowConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
					Workflows:     []string{"worker-a", "worker-b"},
					WorkflowFiles: map[string]string{"worker-a": "./.github/workflows/worker-a.lock.yml", "worker-b": "./.github/workflows/worker-b.lock.yml"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"call_workflow"},
		},
		{
			name: "submit_pull_request_review config",
			safeOutputs: &SafeOutputsConfig{
				SubmitPullRequestReview: &SubmitPullRequestReviewConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"submit_pull_request_review"},
		},
		{
			name: "reply_to_pull_request_review_comment config",
			safeOutputs: &SafeOutputsConfig{
				ReplyToPullRequestReviewComment: &ReplyToPullRequestReviewCommentConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"reply_to_pull_request_review_comment"},
		},
		{
			name: "resolve_pull_request_review_thread config",
			safeOutputs: &SafeOutputsConfig{
				ResolvePullRequestReviewThread: &ResolvePullRequestReviewThreadConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("10"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"resolve_pull_request_review_thread"},
		},
		{
			name: "create_code_scanning_alert config",
			safeOutputs: &SafeOutputsConfig{
				CreateCodeScanningAlerts: &CreateCodeScanningAlertsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("3"),
					},
					Driver: "Test Scanner",
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_code_scanning_alert"},
		},
		{
			name: "remove_labels config",
			safeOutputs: &SafeOutputsConfig{
				RemoveLabels: &RemoveLabelsConfig{
					Allowed: []string{"bug", "wontfix"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"remove_labels"},
		},
		{
			name: "update_pull_request config",
			safeOutputs: &SafeOutputsConfig{
				UpdatePullRequests: &UpdatePullRequestsConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: strPtr("1"),
						},
					},
					Title: testBoolPtr(true),
					Body:  testBoolPtr(true),
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"update_pull_request"},
		},
		{
			name: "update_project config",
			safeOutputs: &SafeOutputsConfig{
				UpdateProjects: &UpdateProjectConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"update_project"},
		},
		{
			name: "create_project config",
			safeOutputs: &SafeOutputsConfig{
				CreateProjects: &CreateProjectsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_project"},
		},
		{
			name: "create_project_status_update config",
			safeOutputs: &SafeOutputsConfig{
				CreateProjectStatusUpdates: &CreateProjectStatusUpdateConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_project_status_update"},
		},
		{
			name: "link_sub_issue config",
			safeOutputs: &SafeOutputsConfig{
				LinkSubIssue: &LinkSubIssueConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"link_sub_issue"},
		},
		{
			name: "dispatch_workflow config",
			safeOutputs: &SafeOutputsConfig{
				DispatchWorkflow: &DispatchWorkflowConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
					Workflows: []string{"worker-a"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"dispatch_workflow"},
		},
		{
			name: "update_discussion config",
			safeOutputs: &SafeOutputsConfig{
				UpdateDiscussions: &UpdateDiscussionsConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: strPtr("1"),
						},
					},
					Title: testBoolPtr(true),
					Body:  testBoolPtr(true),
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"update_discussion"},
		},
		{
			name: "close_discussion config",
			safeOutputs: &SafeOutputsConfig{
				CloseDiscussions: &CloseEntityConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"close_discussion"},
		},
		{
			name: "mark_pull_request_as_ready_for_review config",
			safeOutputs: &SafeOutputsConfig{
				MarkPullRequestAsReadyForReview: &MarkPullRequestAsReadyForReviewConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"mark_pull_request_as_ready_for_review"},
		},
		{
			name: "create_pull_request_review_comment config",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("10"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_pull_request_review_comment"},
		},
		{
			name: "autofix_code_scanning_alert config",
			safeOutputs: &SafeOutputsConfig{
				AutofixCodeScanningAlert: &AutofixCodeScanningAlertConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("10"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"autofix_code_scanning_alert"},
		},
		{
			name: "add_reviewer config",
			safeOutputs: &SafeOutputsConfig{
				AddReviewer: &AddReviewerConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("3"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"add_reviewer"},
		},
		{
			name: "assign_milestone config",
			safeOutputs: &SafeOutputsConfig{
				AssignMilestone: &AssignMilestoneConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"assign_milestone"},
		},
		{
			name: "assign_to_agent config",
			safeOutputs: &SafeOutputsConfig{
				AssignToAgent: &AssignToAgentConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
					DefaultAgent: "copilot",
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"assign_to_agent"},
		},
		{
			name: "upload_asset config",
			safeOutputs: &SafeOutputsConfig{
				UploadAssets: &UploadAssetsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"upload_asset"},
		},
		{
			name: "upload_artifact config",
			safeOutputs: &SafeOutputsConfig{
				UploadArtifact: &UploadArtifactConfig{
					MaxUploads:   1,
					MaxSizeBytes: 104857600,
					AllowedPaths: []string{"output/**"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"upload_artifact"},
		},
		{
			name: "update_release config",
			safeOutputs: &SafeOutputsConfig{
				UpdateRelease: &UpdateReleaseConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: strPtr("1"),
						},
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"update_release"},
		},
		{
			name: "create_agent_session config",
			safeOutputs: &SafeOutputsConfig{
				CreateAgentSessions: &CreateAgentSessionConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"create_agent_session"},
		},
		{
			name: "hide_comment config",
			safeOutputs: &SafeOutputsConfig{
				HideComment: &HideCommentConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"hide_comment"},
		},
		{
			name: "set_issue_type config",
			safeOutputs: &SafeOutputsConfig{
				SetIssueType: &SetIssueTypeConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"set_issue_type"},
		},
		{
			name: "noop config",
			safeOutputs: &SafeOutputsConfig{
				NoOp: &NoOpConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"noop"},
		},
		{
			name: "assign_to_user config",
			safeOutputs: &SafeOutputsConfig{
				AssignToUser: &AssignToUserConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
					Allowed: []string{"user1", "user2"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"assign_to_user"},
		},
		{
			name: "unassign_from_user config",
			safeOutputs: &SafeOutputsConfig{
				UnassignFromUser: &UnassignFromUserConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"unassign_from_user"},
		},
		{
			name: "missing_tool config",
			safeOutputs: &SafeOutputsConfig{
				MissingTool: &MissingToolConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"missing_tool"},
		},
		{
			name: "missing_data config",
			safeOutputs: &SafeOutputsConfig{
				MissingData: &MissingDataConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"missing_data"},
		},
		{
			name: "report_incomplete config",
			safeOutputs: &SafeOutputsConfig{
				ReportIncomplete: &ReportIncompleteConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"report_incomplete"},
		},
		{
			name: "merge_pull_request config",
			safeOutputs: &SafeOutputsConfig{
				MergePullRequest: &MergePullRequestConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
					RequiredLabels:  []string{"automerge"},
					AllowedBranches: []string{"feature/*", "fix/*"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
			checkJSON:    true,
			expectedKeys: []string{"merge_pull_request"},
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

			require.NotEmpty(t, steps)

			stepsContent := strings.Join(steps, "")

			for _, expected := range tt.checkContains {
				assert.Contains(t, stepsContent, expected, "Expected to find: "+expected)
			}

			// Extract and validate JSON if requested
			if tt.checkJSON {
				// Extract JSON from the env var line
				for _, step := range steps {
					if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
						// Extract the JSON value
						parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
						if len(parts) == 2 {
							jsonStr := strings.TrimSpace(parts[1])
							jsonStr = strings.Trim(jsonStr, "\"")
							jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

							var config map[string]map[string]any
							err := json.Unmarshal([]byte(jsonStr), &config)
							require.NoError(t, err, "Config JSON should be valid")

							// Check expected keys
							for _, key := range tt.expectedKeys {
								assert.Contains(t, config, key, "Expected config key: "+key)
							}
						}
					}
				}
			}
		})
	}
}

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
				AllowedLabels: []string{"bug", "enhancement", "documentation"},
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
				Reviewers:     []string{"user1"},
				TeamReviewers: []string{"team-a"},
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
					HideOlderComments: testStringPtr("true"),
				},
			},
			checkField: "add_comment",
			checkKey:   "hide_older_comments",
			expected:   true,
		},
		{
			name: "close older discussions",
			safeOutputs: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{
					CloseOlderDiscussions: testStringPtr("true"),
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
					AllowEmpty: testStringPtr("true"),
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
					Draft: testStringPtr("true"),
				},
			},
			checkField: "create_pull_request",
			checkKey:   "draft",
			expected:   true, // AddTemplatableBool converts "true" string to JSON boolean
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
				Status: testBoolPtr(true),
				Title:  testBoolPtr(true),
				Body:   testBoolPtr(true),
			},
			expectedKeys: []string{"allow_status", "allow_title", "allow_body"},
		},
		{
			name: "only status",
			config: &UpdateIssuesConfig{
				Status: testBoolPtr(true),
			},
			expectedKeys: []string{"allow_status"},
		},
		{
			name: "title and body",
			config: &UpdateIssuesConfig{
				Title: testBoolPtr(true),
				Body:  testBoolPtr(true),
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
		name         string
		updateBranch *bool
		expected     bool
	}{
		{
			name:         "defaults update_branch to false",
			updateBranch: nil,
			expected:     false,
		},
		{
			name:         "sets update_branch true when configured",
			updateBranch: testBoolPtr(true),
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					UpdatePullRequests: &UpdatePullRequestsConfig{
						UpdateBranch: tt.updateBranch,
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
						assert.Equal(t, tt.expected, updateBranchValue)
					}
				}
			}

			require.True(t, foundHandlerConfig, "Expected GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG in generated steps")
		})
	}
}

// TestEmptySafeOutputsConfig tests behavior with no safe outputs
func TestEmptySafeOutputsConfig(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: nil,
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	// Should not add any steps when safe outputs is nil
	assert.Empty(t, steps)
}

// TestHandlerConfigTargetRepo tests target-repo configuration
func TestHandlerConfigTargetRepo(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				TargetRepoSlug: "org/target-repo",
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

				targetRepo, ok := issueConfig["target-repo"]
				require.True(t, ok)
				assert.Equal(t, "org/target-repo", targetRepo)
			}
		}
	}
}

// TestHandlerConfigPatchSize tests max patch size configuration
func TestHandlerConfigPatchSize(t *testing.T) {
	tests := []struct {
		name         string
		maxPatchSize int
		expectedSize int
	}{
		{
			name:         "default patch size",
			maxPatchSize: 0,
			expectedSize: 1024,
		},
		{
			name:         "custom patch size",
			maxPatchSize: 2048,
			expectedSize: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					MaximumPatchSize: tt.maxPatchSize,
					CreatePullRequests: &CreatePullRequestsConfig{
						TitlePrefix: "[PR] ",
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

						prConfig, ok := config["create_pull_request"]
						require.True(t, ok)

						maxSize, ok := prConfig["max_patch_size"]
						require.True(t, ok)
						assert.InDelta(t, float64(tt.expectedSize), maxSize, 0.0001)
					}
				}
			}
		})
	}
}

// testBoolPtr is a helper function for bool pointers in config tests
func testBoolPtr(b bool) *bool {
	return &b
}

// testStringPtr is a helper function for string pointers in config tests
func testStringPtr(s string) *string {
	return &s
}

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
		expectedBaseBranch               string
		shouldHaveBaseBranchKey          bool
		expectedAllowedBaseBranches      []string
		shouldHaveAllowedBaseBranchesKey bool
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
			expectedBaseBranch:               "main",
			shouldHaveBaseBranchKey:          true,
			expectedAllowedBaseBranches:      []string{"release/*", "main"},
			shouldHaveAllowedBaseBranchesKey: true,
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
					}
				}
			}
		})
	}
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
				Allowed: []string{"user1", "user2", "copilot"},
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
				assert.Equal(t, true, unassignFirst, "unassign_first should be true")
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
				Allowed: []string{"githubactionagent", "bot-user"},
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
				Blocked: []string{"copilot", "*[bot]"},
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
				Blocked: []string{"copilot", "*[bot]"},
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

// TestHandlerConfigStagedMode tests that per-handler staged: true is included in handler config JSON
func TestHandlerConfigStagedMode(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		handlerKey  string
	}{
		{
			name: "push_to_pull_request_branch staged",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Staged: true,
					},
					Target:      "*",
					IfNoChanges: "warn",
				},
			},
			handlerKey: "push_to_pull_request_branch",
		},
		{
			name: "close_pull_request staged",
			safeOutputs: &SafeOutputsConfig{
				ClosePullRequests: &ClosePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max:    strPtr("1"),
						Staged: true,
					},
				},
			},
			handlerKey: "close_pull_request",
		},
		{
			name: "create_issue staged",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Staged: true,
					},
				},
			},
			handlerKey: "create_issue",
		},
		{
			name: "add_comment staged",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Staged: true,
					},
				},
			},
			handlerKey: "add_comment",
		},
		{
			name: "create_pull_request staged",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Staged: true,
					},
				},
			},
			handlerKey: "create_pull_request",
		},
		{
			name: "update_issue staged",
			safeOutputs: &SafeOutputsConfig{
				UpdateIssues: &UpdateIssuesConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Staged: true,
						},
					},
				},
			},
			handlerKey: "update_issue",
		},
		{
			name: "update_pull_request staged",
			safeOutputs: &SafeOutputsConfig{
				UpdatePullRequests: &UpdatePullRequestsConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Staged: true,
						},
					},
				},
			},
			handlerKey: "update_pull_request",
		},
		{
			name: "update_discussion staged",
			safeOutputs: &SafeOutputsConfig{
				UpdateDiscussions: &UpdateDiscussionsConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Staged: true,
						},
					},
				},
			},
			handlerKey: "update_discussion",
		},
		{
			name: "add_labels staged",
			safeOutputs: &SafeOutputsConfig{
				AddLabels: &AddLabelsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Staged: true,
					},
				},
			},
			handlerKey: "add_labels",
		},
		{
			name: "dispatch_workflow staged",
			safeOutputs: &SafeOutputsConfig{
				DispatchWorkflow: &DispatchWorkflowConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Staged: true,
					},
					Workflows: []string{"my-workflow"},
				},
			},
			handlerKey: "dispatch_workflow",
		},
		{
			name: "call_workflow staged",
			safeOutputs: &SafeOutputsConfig{
				CallWorkflow: &CallWorkflowConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Staged: true,
					},
					Workflows: []string{"my-workflow"},
				},
			},
			handlerKey: "call_workflow",
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

			require.NotEmpty(t, steps, "Steps should not be empty")

			for _, step := range steps {
				if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
					parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
					require.Len(t, parts, 2, "Should have two parts")

					jsonStr := strings.TrimSpace(parts[1])
					jsonStr = strings.Trim(jsonStr, "\"")
					jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

					var config map[string]map[string]any
					err := json.Unmarshal([]byte(jsonStr), &config)
					require.NoError(t, err, "Handler config JSON should be valid")

					handlerConfig, ok := config[tt.handlerKey]
					require.True(t, ok, "Should have %s handler", tt.handlerKey)

					stagedVal, ok := handlerConfig["staged"]
					require.True(t, ok, "Handler config should include 'staged' field when staged: true is set")
					assert.Equal(t, true, stagedVal, "staged field should be true")
				}
			}
		})
	}
}

// TestAddHandlerManagerConfigEnvVar_CallWorkflow asserts that GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
// contains call_workflow, workflows, and workflow_files when SafeOutputs.CallWorkflow is configured.
func TestAddHandlerManagerConfigEnvVar_CallWorkflow(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows:     []string{"worker-a", "worker-b"},
				WorkflowFiles: map[string]string{"worker-a": "./.github/workflows/worker-a.lock.yml", "worker-b": "./.github/workflows/worker-b.lock.yml"},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	require.NotEmpty(t, steps, "Steps should not be empty")

	var callWorkflowConfig map[string]any
	for _, step := range steps {
		if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
			require.Len(t, parts, 2, "Should have two parts")

			jsonStr := strings.TrimSpace(parts[1])
			jsonStr = strings.Trim(jsonStr, "\"")
			jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

			var config map[string]map[string]any
			err := json.Unmarshal([]byte(jsonStr), &config)
			require.NoError(t, err, "Handler config JSON should be valid")

			cfg, ok := config["call_workflow"]
			require.True(t, ok, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG should contain 'call_workflow' key")
			callWorkflowConfig = cfg
			break
		}
	}

	require.NotNil(t, callWorkflowConfig, "call_workflow config should be present")

	// Verify max
	maxVal, ok := callWorkflowConfig["max"]
	require.True(t, ok, "call_workflow config should have 'max' field")
	assert.InDelta(t, float64(1), maxVal, 0.0001, "max should be 1")

	// Verify workflows list
	workflowsVal, ok := callWorkflowConfig["workflows"]
	require.True(t, ok, "call_workflow config should have 'workflows' field")
	workflowsSlice, ok := workflowsVal.([]any)
	require.True(t, ok, "workflows should be an array")
	assert.Len(t, workflowsSlice, 2, "Should have 2 workflows")
	assert.Contains(t, workflowsSlice, "worker-a", "Should contain worker-a")
	assert.Contains(t, workflowsSlice, "worker-b", "Should contain worker-b")

	// Verify workflow_files map
	filesVal, ok := callWorkflowConfig["workflow_files"]
	require.True(t, ok, "call_workflow config should have 'workflow_files' field")
	filesMap, ok := filesVal.(map[string]any)
	require.True(t, ok, "workflow_files should be a map")
	assert.Equal(t, "./.github/workflows/worker-a.lock.yml", filesMap["worker-a"], "worker-a path should match")
	assert.Equal(t, "./.github/workflows/worker-b.lock.yml", filesMap["worker-b"], "worker-b path should match")
}

// TestProtectedFilesExclude verifies that the _protected_files_exclude sentinel key is
// used at compile time to filter manifest files and is NOT forwarded to the runtime config.
func TestProtectedFilesExclude(t *testing.T) {
	tests := []struct {
		name               string
		excludeFiles       []string
		wantExcludedFromPF []string // files that must NOT be in the final protected_files list
		wantPresentInPF    []string // files that must still be in the protected_files list
	}{
		{
			name:               "exclude AGENTS.md from create-pull-request",
			excludeFiles:       []string{"AGENTS.md"},
			wantExcludedFromPF: []string{"AGENTS.md"},
			wantPresentInPF:    []string{"package.json", "go.mod", "CODEOWNERS"},
		},
		{
			name:               "exclude multiple files",
			excludeFiles:       []string{"AGENTS.md", "CLAUDE.md"},
			wantExcludedFromPF: []string{"AGENTS.md", "CLAUDE.md"},
			wantPresentInPF:    []string{"package.json", "go.mod"},
		},
		{
			name:               "empty exclude list leaves defaults intact",
			excludeFiles:       nil,
			wantExcludedFromPF: nil,
			wantPresentInPF:    []string{"package.json", "go.mod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreatePullRequests: &CreatePullRequestsConfig{
						BaseSafeOutputConfig:  BaseSafeOutputConfig{Max: strPtr("1")},
						ProtectedFilesExclude: tt.excludeFiles,
					},
				},
			}

			var steps []string
			compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)
			require.NotEmpty(t, steps, "should produce config steps")

			stepsContent := strings.Join(steps, "")
			require.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", "should produce handler config")

			// Extract JSON
			var configJSON string
			for _, step := range steps {
				if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
					parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
					require.Len(t, parts, 2, "should be able to split env var line")
					configJSON = strings.TrimSpace(parts[1])
					configJSON = strings.Trim(configJSON, "\"")
					configJSON = strings.ReplaceAll(configJSON, "\\\"", "\"")
				}
			}
			require.NotEmpty(t, configJSON, "should have extracted JSON")

			var config map[string]map[string]any
			require.NoError(t, json.Unmarshal([]byte(configJSON), &config), "config JSON should be valid")

			prConfig, ok := config["create_pull_request"]
			require.True(t, ok, "should have create_pull_request config")

			// Sentinel key must NOT appear in the final runtime config
			_, hasSentinel := prConfig["_protected_files_exclude"]
			assert.False(t, hasSentinel, "_protected_files_exclude sentinel must not appear in runtime config")

			// Check protected_files list
			pfRaw, ok := prConfig["protected_files"]
			require.True(t, ok, "should have protected_files field")
			pfAny, ok := pfRaw.([]any)
			require.True(t, ok, "protected_files should be a slice")
			pfStrings := make([]string, 0, len(pfAny))
			for _, v := range pfAny {
				if s, ok := v.(string); ok {
					pfStrings = append(pfStrings, s)
				}
			}

			for _, excluded := range tt.wantExcludedFromPF {
				assert.NotContains(t, pfStrings, excluded,
					"excluded file %q should not appear in protected_files", excluded)
			}
			for _, present := range tt.wantPresentInPF {
				assert.Contains(t, pfStrings, present,
					"non-excluded file %q should still appear in protected_files", present)
			}
		})
	}
}

// TestProtectedFilesExcludePushToPRBranch verifies the same exclusion logic for
// the push_to_pull_request_branch handler.
func TestProtectedFilesExcludePushToPRBranch(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			PushToPullRequestBranch: &PushToPullRequestBranchConfig{
				ProtectedFilesExclude: []string{"AGENTS.md"},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)
	require.NotEmpty(t, steps, "should produce config steps")

	var configJSON string
	for _, step := range steps {
		if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
			require.Len(t, parts, 2, "should split env var line")
			configJSON = strings.TrimSpace(parts[1])
			configJSON = strings.Trim(configJSON, "\"")
			configJSON = strings.ReplaceAll(configJSON, "\\\"", "\"")
		}
	}
	require.NotEmpty(t, configJSON, "should have extracted JSON")

	var config map[string]map[string]any
	require.NoError(t, json.Unmarshal([]byte(configJSON), &config), "config JSON should be valid")

	pushConfig, ok := config["push_to_pull_request_branch"]
	require.True(t, ok, "should have push_to_pull_request_branch config")

	_, hasSentinel := pushConfig["_protected_files_exclude"]
	assert.False(t, hasSentinel, "_protected_files_exclude sentinel must not appear in runtime config")

	pfRaw, ok := pushConfig["protected_files"]
	require.True(t, ok, "should have protected_files field")
	pfAny, ok := pfRaw.([]any)
	require.True(t, ok, "protected_files should be a slice")
	pfStrings := make([]string, 0, len(pfAny))
	for _, v := range pfAny {
		if s, ok := v.(string); ok {
			pfStrings = append(pfStrings, s)
		}
	}
	assert.NotContains(t, pfStrings, "AGENTS.md", "AGENTS.md should be excluded from protected_files")
	assert.Contains(t, pfStrings, "package.json", "package.json should still be in protected_files")
}
