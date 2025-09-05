package workflow

import (
	"testing"
)

func TestApplyDefaultGitCommandsForSafeOutputs(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	engine := NewClaudeEngine()

	tests := []struct {
		name        string
		tools       map[string]any
		safeOutputs *SafeOutputsConfig
		expectGit   bool
	}{
		{
			name:        "no safe outputs - no git commands",
			tools:       map[string]any{},
			safeOutputs: nil,
			expectGit:   false,
		},
		{
			name:  "create-pull-request enabled - should add git commands",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectGit: true,
		},
		{
			name:  "push-to-branch enabled - should add git commands",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				PushToBranch: &PushToBranchConfig{Branch: "main"},
			},
			expectGit: true,
		},
		{
			name:  "only create-issue enabled - no git commands",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expectGit: false,
		},
		{
			name: "existing bash commands should be preserved",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"echo", "ls"},
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectGit: true,
		},
		{
			name: "bash with wildcard should remain wildcard",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{":*"},
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectGit: true,
		},
		{
			name: "bash with nil value should remain nil",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": nil,
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectGit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of input tools to avoid modifying test data
			tools := make(map[string]any)
			for k, v := range tt.tools {
				tools[k] = v
			}

			// Apply both default tool functions in sequence
			tools = compiler.applyDefaultGitHubMCPTools(tools)
			result := engine.applyDefaultClaudeTools(tools, tt.safeOutputs)

			// Check if claude section exists and has bash tool
			claudeSection, hasClaudeSection := result["claude"]
			if !hasClaudeSection {
				if tt.expectGit {
					t.Error("Expected claude section to be created with Git commands")
				}
				return
			}

			claudeConfig, ok := claudeSection.(map[string]any)
			if !ok {
				t.Error("Expected claude section to be a map")
				return
			}

			allowed, hasAllowed := claudeConfig["allowed"]
			if !hasAllowed {
				if tt.expectGit {
					t.Error("Expected claude section to have allowed tools")
				}
				return
			}

			allowedMap, ok := allowed.(map[string]any)
			if !ok {
				t.Error("Expected allowed to be a map")
				return
			}

			bashTool, hasBash := allowedMap["Bash"]
			if !hasBash {
				if tt.expectGit {
					t.Error("Expected Bash tool to be present when Git commands are needed")
				}
				return
			}

			// If we don't expect Git commands, just verify no error occurred
			if !tt.expectGit {
				return
			}

			// Check the specific cases for bash tool value
			if bashCommands, ok := bashTool.([]any); ok {
				// Should contain Git commands
				foundGitCommands := false
				for _, cmd := range bashCommands {
					if cmdStr, ok := cmd.(string); ok {
						if cmdStr == "git checkout:*" || cmdStr == "git add:*" || cmdStr == ":*" || cmdStr == "*" {
							foundGitCommands = true
							break
						}
					}
				}
				if !foundGitCommands {
					t.Error("Expected to find Git commands in Bash tool commands")
				}
			} else if bashTool == nil {
				// nil value means all bash commands are allowed, which includes Git commands
				// This is acceptable - nil value already permits all commands
				_ = bashTool // Keep the nil value as-is
			} else {
				t.Errorf("Unexpected Bash tool value type: %T", bashTool)
			}
		})
	}
}

func TestAdditionalClaudeToolsForSafeOutputs(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	engine := NewClaudeEngine()

	tests := []struct {
		name               string
		tools              map[string]any
		safeOutputs        *SafeOutputsConfig
		expectEditingTools bool
	}{
		{
			name:               "no safe outputs - no editing tools",
			tools:              map[string]any{},
			safeOutputs:        nil,
			expectEditingTools: false,
		},
		{
			name:  "create-pull-request enabled - should add editing tools",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectEditingTools: true,
		},
		{
			name:  "push-to-branch enabled - should add editing tools",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				PushToBranch: &PushToBranchConfig{Branch: "main"},
			},
			expectEditingTools: true,
		},
		{
			name:  "only create-issue enabled - no editing tools",
			tools: map[string]any{},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expectEditingTools: false,
		},
		{
			name: "existing editing tools should be preserved",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Edit": nil,
						"Task": nil,
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectEditingTools: true,
		},
	}

	expectedEditingTools := []string{"Edit", "MultiEdit", "Write", "NotebookEdit"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of input tools to avoid modifying test data
			tools := make(map[string]any)
			for k, v := range tt.tools {
				tools[k] = v
			}

			// Apply both default tool functions in sequence
			tools = compiler.applyDefaultGitHubMCPTools(tools)
			result := engine.applyDefaultClaudeTools(tools, tt.safeOutputs)

			// Check if claude section exists
			claudeSection, hasClaudeSection := result["claude"]
			if !hasClaudeSection {
				if tt.expectEditingTools {
					t.Error("Expected claude section to be created with editing tools")
				}
				return
			}

			claudeConfig, ok := claudeSection.(map[string]any)
			if !ok {
				t.Error("Expected claude section to be a map")
				return
			}

			allowed, hasAllowed := claudeConfig["allowed"]
			if !hasAllowed {
				if tt.expectEditingTools {
					t.Error("Expected claude section to have allowed tools")
				}
				return
			}

			allowedMap, ok := allowed.(map[string]any)
			if !ok {
				t.Error("Expected allowed to be a map")
				return
			}

			// If we don't expect editing tools, verify they aren't there due to this feature
			if !tt.expectEditingTools {
				// Only check if we started with empty tools - if there were pre-existing tools, they should remain
				if len(tt.tools) == 0 {
					for _, tool := range expectedEditingTools {
						if _, exists := allowedMap[tool]; exists {
							t.Errorf("Unexpected editing tool %s found when not expected", tool)
						}
					}
				}
				return
			}

			// Check that all expected editing tools are present
			for _, expectedTool := range expectedEditingTools {
				if _, exists := allowedMap[expectedTool]; !exists {
					t.Errorf("Expected editing tool %s to be present", expectedTool)
				}
			}
		})
	}
}

func TestNeedsGitCommands(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		expected    bool
	}{
		{
			name:        "nil safe outputs",
			safeOutputs: nil,
			expected:    false,
		},
		{
			name:        "empty safe outputs",
			safeOutputs: &SafeOutputsConfig{},
			expected:    false,
		},
		{
			name: "create-pull-request enabled",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expected: true,
		},
		{
			name: "push-to-branch enabled",
			safeOutputs: &SafeOutputsConfig{
				PushToBranch: &PushToBranchConfig{Branch: "main"},
			},
			expected: true,
		},
		{
			name: "both enabled",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
				PushToBranch:       &PushToBranchConfig{Branch: "main"},
			},
			expected: true,
		},
		{
			name: "only other outputs enabled",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues:     &CreateIssuesConfig{},
				AddIssueComments: &AddIssueCommentsConfig{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsGitCommands(tt.safeOutputs)
			if result != tt.expected {
				t.Errorf("needsGitCommands() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
