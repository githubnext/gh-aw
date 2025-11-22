package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateFilteredToolsJSON(t *testing.T) {
	tests := []struct {
		name          string
		safeOutputs   *SafeOutputsConfig
		expectedTools []string
	}{
		{
			name:          "nil safe outputs returns empty array",
			safeOutputs:   nil,
			expectedTools: []string{},
		},
		{
			name:          "empty safe outputs returns empty array",
			safeOutputs:   &SafeOutputsConfig{},
			expectedTools: []string{},
		},
		{
			name: "create issues enabled",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 5},
				},
			},
			expectedTools: []string{"create_issue"},
		},
		{
			name: "create agent tasks enabled",
			safeOutputs: &SafeOutputsConfig{
				CreateAgentTasks: &CreateAgentTaskConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 3},
				},
			},
			expectedTools: []string{"create_agent_task"},
		},
		{
			name: "create discussions enabled",
			safeOutputs: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 2},
				},
			},
			expectedTools: []string{"create_discussion"},
		},
		{
			name: "add comments enabled",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10},
				},
			},
			expectedTools: []string{"add_comment"},
		},
		{
			name: "create pull requests enabled",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectedTools: []string{"create_pull_request"},
		},
		{
			name: "create pull request review comments enabled",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 5},
				},
			},
			expectedTools: []string{"create_pull_request_review_comment"},
		},
		{
			name: "create code scanning alerts enabled",
			safeOutputs: &SafeOutputsConfig{
				CreateCodeScanningAlerts: &CreateCodeScanningAlertsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 100},
				},
			},
			expectedTools: []string{"create_code_scanning_alert"},
		},
		{
			name: "add labels enabled",
			safeOutputs: &SafeOutputsConfig{
				AddLabels: &AddLabelsConfig{
					Max: 5,
				},
			},
			expectedTools: []string{"add_labels"},
		},
		{
			name: "update issues enabled",
			safeOutputs: &SafeOutputsConfig{
				UpdateIssues: &UpdateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 3},
				},
			},
			expectedTools: []string{"update_issue"},
		},
		{
			name: "push to pull request branch enabled",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
				},
			},
			expectedTools: []string{"push_to_pull_request_branch"},
		},
		{
			name: "upload assets enabled",
			safeOutputs: &SafeOutputsConfig{
				UploadAssets: &UploadAssetsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10},
				},
			},
			expectedTools: []string{"upload_asset"},
		},
		{
			name: "missing tool enabled",
			safeOutputs: &SafeOutputsConfig{
				MissingTool: &MissingToolConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 5},
				},
			},
			expectedTools: []string{"missing_tool"},
		},
		{
			name: "multiple tools enabled",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 5}},
				AddComments:  &AddCommentsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10}},
				AddLabels:    &AddLabelsConfig{Max: 3},
			},
			expectedTools: []string{"create_issue", "add_comment", "add_labels"},
		},
		{
			name: "all tools enabled",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues:                    &CreateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 5}},
				CreateAgentTasks:                &CreateAgentTaskConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 3}},
				CreateDiscussions:               &CreateDiscussionsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 2}},
				AddComments:                     &AddCommentsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10}},
				CreatePullRequests:              &CreatePullRequestsConfig{},
				CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 5}},
				CreateCodeScanningAlerts:        &CreateCodeScanningAlertsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 100}},
				AddLabels:                       &AddLabelsConfig{Max: 3},
				AddReviewer:                     &AddReviewerConfig{Max: 3},
				UpdateIssues:                    &UpdateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 3}},
				PushToPullRequestBranch:         &PushToPullRequestBranchConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1}},
				UploadAssets:                    &UploadAssetsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10}},
				MissingTool:                     &MissingToolConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 5}},
			},
			expectedTools: []string{
				"create_issue",
				"create_agent_task",
				"create_discussion",
				"add_comment",
				"create_pull_request",
				"create_pull_request_review_comment",
				"create_code_scanning_alert",
				"add_labels",
				"add_reviewer",
				"update_issue",
				"push_to_pull_request_branch",
				"upload_asset",
				"missing_tool",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				SafeOutputs: tt.safeOutputs,
			}

			result, err := generateFilteredToolsJSON(workflowData)
			require.NoError(t, err, "generateFilteredToolsJSON should not error")

			// Parse the JSON result
			var tools []map[string]any
			err = json.Unmarshal([]byte(result), &tools)
			require.NoError(t, err, "Result should be valid JSON")

			// Extract tool names from the result
			var actualTools []string
			for _, tool := range tools {
				if name, ok := tool["name"].(string); ok {
					actualTools = append(actualTools, name)
				}
			}

			// Check that the expected tools are present
			assert.ElementsMatch(t, tt.expectedTools, actualTools, "Tool names should match")

			// Verify each tool has required fields
			for _, tool := range tools {
				assert.Contains(t, tool, "name", "Tool should have name field")
				assert.Contains(t, tool, "description", "Tool should have description field")
				assert.Contains(t, tool, "inputSchema", "Tool should have inputSchema field")
			}
		})
	}
}

func TestGenerateFilteredToolsJSONValidStructure(t *testing.T) {
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 5}},
			AddComments:  &AddCommentsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10}},
		},
	}

	result, err := generateFilteredToolsJSON(workflowData)
	require.NoError(t, err)

	// Parse the JSON result
	var tools []map[string]any
	err = json.Unmarshal([]byte(result), &tools)
	require.NoError(t, err)

	// Verify create_issue tool structure
	var createIssueTool map[string]any
	for _, tool := range tools {
		if tool["name"] == "create_issue" {
			createIssueTool = tool
			break
		}
	}
	require.NotNil(t, createIssueTool, "create_issue tool should be present")

	// Check inputSchema structure
	inputSchema, ok := createIssueTool["inputSchema"].(map[string]any)
	require.True(t, ok, "inputSchema should be a map")

	assert.Equal(t, "object", inputSchema["type"], "inputSchema type should be object")

	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "properties should be a map")

	// Verify required properties exist
	assert.Contains(t, properties, "title", "Should have title property")
	assert.Contains(t, properties, "body", "Should have body property")

	// Verify required field
	required, ok := inputSchema["required"].([]any)
	require.True(t, ok, "required should be an array")
	assert.Contains(t, required, "title", "title should be required")
	assert.Contains(t, required, "body", "body should be required")
}

func TestGetSafeOutputsToolsJSON(t *testing.T) {
	// Test that the embedded JSON can be retrieved and parsed
	toolsJSON := GetSafeOutputsToolsJSON()
	require.NotEmpty(t, toolsJSON, "Tools JSON should not be empty")

	// Parse the JSON to ensure it's valid
	var tools []map[string]any
	err := json.Unmarshal([]byte(toolsJSON), &tools)
	require.NoError(t, err, "Tools JSON should be valid")
	require.NotEmpty(t, tools, "Tools array should not be empty")

	// Verify all expected tools are present
	expectedTools := []string{
		"create_issue",
		"create_agent_task",
		"create_discussion",
		"close_discussion",
		"close_issue",
		"add_comment",
		"create_pull_request",
		"create_pull_request_review_comment",
		"create_code_scanning_alert",
		"add_labels",
		"add_reviewer",
		"assign_milestone",
		"update_issue",
		"push_to_pull_request_branch",
		"upload_asset",
		"update_release",
		"missing_tool",
		"noop",
	}

	var actualTools []string
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok {
			actualTools = append(actualTools, name)
		}
	}

	assert.ElementsMatch(t, expectedTools, actualTools, "All expected tools should be present")

	// Verify each tool has the required structure
	for _, tool := range tools {
		name := tool["name"].(string)
		t.Run("tool_"+name, func(t *testing.T) {
			assert.Contains(t, tool, "name", "Tool should have name")
			assert.Contains(t, tool, "description", "Tool should have description")
			assert.Contains(t, tool, "inputSchema", "Tool should have inputSchema")

			// Verify inputSchema structure
			inputSchema, ok := tool["inputSchema"].(map[string]any)
			require.True(t, ok, "inputSchema should be a map")
			assert.Equal(t, "object", inputSchema["type"], "inputSchema type should be object")
			assert.Contains(t, inputSchema, "properties", "inputSchema should have properties")
		})
	}
}
