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
// Staged Mode Tests
// ========================================

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
						Staged: templatableBoolPtr("true"),
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
						Staged: templatableBoolPtr("true"),
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
						Staged: templatableBoolPtr("true"),
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
						Staged: templatableBoolPtr("true"),
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
						Staged: templatableBoolPtr("true"),
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
							Staged: templatableBoolPtr("true"),
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
							Staged: templatableBoolPtr("true"),
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
							Staged: templatableBoolPtr("true"),
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
						Staged: templatableBoolPtr("true"),
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
						Staged: templatableBoolPtr("true"),
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
						Staged: templatableBoolPtr("true"),
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
					stagedBool, ok := stagedVal.(bool)
					require.True(t, ok, "staged field should be a bool")
					assert.True(t, stagedBool, "staged field should be true")
				}
			}

		})
	}
}

func TestHandlerConfigStagedExpression(t *testing.T) {
	t.Parallel()

	compiler := NewCompiler()
	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Staged: templatableBoolPtr("${{ inputs.staged }}"),
				},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)
	require.NotEmpty(t, steps, "Steps should not be empty")

	for _, step := range steps {
		if !strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			continue
		}

		parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
		require.Len(t, parts, 2, "Should have two parts")

		jsonStr := strings.TrimSpace(parts[1])
		jsonStr = strings.Trim(jsonStr, "\"")
		jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

		var config map[string]map[string]any
		err := json.Unmarshal([]byte(jsonStr), &config)
		require.NoError(t, err, "Handler config JSON should be valid")

		handlerConfig, ok := config["create_issue"]
		require.True(t, ok, "Should have create_issue handler")
		assert.Equal(t, "${{ inputs.staged }}", handlerConfig["staged"], "staged expression should pass through to handler config JSON unchanged")
		return
	}

	t.Fatal("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG not found")
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
