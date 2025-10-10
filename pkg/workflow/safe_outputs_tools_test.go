package workflow

import (
	"encoding/json"
	"testing"
)

func TestSafeOutputsToolsJSON(t *testing.T) {
	// Test that we can parse the embedded tools JSON
	tools, err := GetSafeOutputsToolsDefinitions()
	if err != nil {
		t.Fatalf("Failed to parse tools JSON: %v", err)
	}

	// Verify we have the expected number of tools
	expectedTools := []string{
		"create_issue",
		"create_discussion",
		"add_comment",
		"create_pull_request",
		"create_pull_request_review_comment",
		"create_code_scanning_alert",
		"add_labels",
		"update_issue",
		"push_to_pull_request_branch",
		"upload_asset",
		"missing_tool",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	// Verify each expected tool is present
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("Expected tool %q not found", expected)
		}
	}

	// Verify each tool has required fields
	for _, tool := range tools {
		if tool.Name == "" {
			t.Errorf("Tool has empty name")
		}
		if tool.Description == "" {
			t.Errorf("Tool %q has empty description", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Errorf("Tool %q has nil inputSchema", tool.Name)
		}
	}
}

func TestGenerateFilteredToolsJSON(t *testing.T) {
	tests := []struct {
		name          string
		safeOutputs   *SafeOutputsConfig
		expectedTools []string
	}{
		{
			name:          "No safe outputs",
			safeOutputs:   nil,
			expectedTools: []string{},
		},
		{
			name: "Single output - create-issue",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
				},
				MissingTool: &MissingToolConfig{},
			},
			expectedTools: []string{"create_issue", "missing_tool"},
		},
		{
			name: "Multiple outputs",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
				},
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
				},
				CreatePullRequests: &CreatePullRequestsConfig{},
				MissingTool:        &MissingToolConfig{},
			},
			expectedTools: []string{"create_issue", "add_comment", "create_pull_request", "missing_tool"},
		},
		{
			name: "Upload assets",
			safeOutputs: &SafeOutputsConfig{
				UploadAssets: &UploadAssetsConfig{},
				MissingTool:  &MissingToolConfig{},
			},
			expectedTools: []string{"upload_asset", "missing_tool"},
		},
	}

	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{
				SafeOutputs: tt.safeOutputs,
			}

			toolsJSON, err := compiler.GenerateFilteredToolsJSON(data)
			if err != nil {
				t.Fatalf("Failed to generate filtered tools JSON: %v", err)
			}

			// Parse the JSON
			var tools map[string]SafeOutputToolDefinition
			if err := json.Unmarshal([]byte(toolsJSON), &tools); err != nil {
				t.Fatalf("Failed to parse tools JSON: %v", err)
			}

			// Verify expected tools are present
			if len(tools) != len(tt.expectedTools) {
				t.Errorf("Expected %d tools, got %d", len(tt.expectedTools), len(tools))
			}

			for _, expected := range tt.expectedTools {
				if _, ok := tools[expected]; !ok {
					t.Errorf("Expected tool %q not found in filtered tools", expected)
				}
			}

			// Verify no unexpected tools
			for toolName := range tools {
				found := false
				for _, expected := range tt.expectedTools {
					if toolName == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected tool %q found in filtered tools", toolName)
				}
			}
		})
	}
}

func TestGenerateFilteredToolsJSONWithHandlers(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true)

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests:       &CreatePullRequestsConfig{},
			PushToPullRequestBranch:  &PushToPullRequestBranchConfig{},
			UploadAssets:             &UploadAssetsConfig{},
			MissingTool:              &MissingToolConfig{},
		},
	}

	toolsJSON, err := compiler.GenerateFilteredToolsJSON(data)
	if err != nil {
		t.Fatalf("Failed to generate filtered tools JSON: %v", err)
	}

	// Parse the JSON
	var tools map[string]SafeOutputToolDefinition
	if err := json.Unmarshal([]byte(toolsJSON), &tools); err != nil {
		t.Fatalf("Failed to parse tools JSON: %v", err)
	}

	// Verify hasHandler flag is set for tools that have handlers
	toolsWithHandlers := []string{"create_pull_request", "push_to_pull_request_branch", "upload_asset"}
	
	for _, toolName := range toolsWithHandlers {
		tool, ok := tools[toolName]
		if !ok {
			t.Errorf("Expected tool %q not found", toolName)
			continue
		}
		if !tool.HasHandler {
			t.Errorf("Tool %q should have hasHandler=true", toolName)
		}
	}

	// Verify missing_tool does not have hasHandler flag
	if tool, ok := tools["missing_tool"]; ok {
		if tool.HasHandler {
			t.Errorf("Tool missing_tool should not have hasHandler flag")
		}
	}
}
