package workflow

import (
	"strings"
	"testing"
)

func TestClaudeEngineComputeAllowedTools(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name     string
		tools    map[string]any
		expected string
	}{
		{
			name:     "empty tools",
			tools:    map[string]any{},
			expected: "",
		},
		{
			name: "bash with specific commands in claude section (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":       []any{"echo", "ls"},
						"BashOutput": nil,
						"KillBash":   nil,
					},
				},
			},
			expected: "Bash(echo),Bash(ls),BashOutput,KillBash",
		},
		{
			name: "bash with nil value (all commands allowed)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":       nil,
						"BashOutput": nil,
						"KillBash":   nil,
					},
				},
			},
			expected: "Bash,BashOutput,KillBash",
		},
		{
			name: "regular tools in claude section (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read":  nil,
						"Write": nil,
					},
				},
			},
			expected: "Read,Write",
		},
		{
			name: "mcp tools",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues", "create_issue"},
				},
			},
			expected: "mcp__github__create_issue,mcp__github__list_issues",
		},
		{
			name: "mixed claude and mcp tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"LS":   nil,
						"Read": nil,
						"Edit": nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expected: "Edit,LS,Read,mcp__github__list_issues",
		},
		{
			name: "custom mcp servers with new format",
			tools: map[string]any{
				"custom_server": map[string]any{
					"mcp": map[string]any{
						"type": "stdio",
					},
					"allowed": []any{"tool1", "tool2"},
				},
			},
			expected: "mcp__custom_server__tool1,mcp__custom_server__tool2",
		},
		{
			name: "mcp server with wildcard access",
			tools: map[string]any{
				"notion": map[string]any{
					"mcp": map[string]any{
						"type": "stdio",
					},
					"allowed": []any{"*"},
				},
			},
			expected: "mcp__notion",
		},
		{
			name: "mixed mcp servers - one with wildcard, one with specific tools",
			tools: map[string]any{
				"notion": map[string]any{
					"mcp":     map[string]any{"type": "stdio"},
					"allowed": []any{"*"},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues", "create_issue"},
				},
			},
			expected: "mcp__github__create_issue,mcp__github__list_issues,mcp__notion",
		},
		{
			name: "bash with :* wildcard (should ignore other bash tools)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":       []any{":*"},
						"BashOutput": nil,
						"KillBash":   nil,
					},
				},
			},
			expected: "Bash,BashOutput,KillBash",
		},
		{
			name: "bash with :* wildcard mixed with other commands (should ignore other commands)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":       []any{"echo", "ls", ":*", "cat"},
						"BashOutput": nil,
						"KillBash":   nil,
					},
				},
			},
			expected: "Bash,BashOutput,KillBash",
		},
		{
			name: "bash with :* wildcard and other tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":       []any{":*"},
						"Read":       nil,
						"BashOutput": nil,
						"KillBash":   nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expected: "Bash,BashOutput,KillBash,Read,mcp__github__list_issues",
		},
		{
			name: "bash with single command should include implicit tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":       []any{"ls"},
						"BashOutput": nil,
						"KillBash":   nil,
					},
				},
			},
			expected: "Bash(ls),BashOutput,KillBash",
		},
		{
			name: "explicit KillBash and BashOutput should not duplicate",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":       []any{"echo"},
						"KillBash":   nil,
						"BashOutput": nil,
					},
				},
			},
			expected: "Bash(echo),BashOutput,KillBash",
		},
		{
			name: "no bash tools means no implicit tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read":  nil,
						"Write": nil,
					},
				},
			},
			expected: "Read,Write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.computeAllowedClaudeToolsString(tt.tools, nil)

			// Parse expected and actual results into sets for comparison
			expectedTools := make(map[string]bool)
			if tt.expected != "" {
				for _, tool := range strings.Split(tt.expected, ",") {
					expectedTools[strings.TrimSpace(tool)] = true
				}
			}

			actualTools := make(map[string]bool)
			if result != "" {
				for _, tool := range strings.Split(result, ",") {
					actualTools[strings.TrimSpace(tool)] = true
				}
			}

			// Check if both sets have the same tools
			if len(expectedTools) != len(actualTools) {
				t.Errorf("Expected %d tools, got %d tools. Expected: '%s', Actual: '%s'",
					len(expectedTools), len(actualTools), tt.expected, result)
				return
			}

			for expectedTool := range expectedTools {
				if !actualTools[expectedTool] {
					t.Errorf("Expected tool '%s' not found in result: '%s'", expectedTool, result)
				}
			}

			for actualTool := range actualTools {
				if !expectedTools[actualTool] {
					t.Errorf("Unexpected tool '%s' found in result: '%s'", actualTool, result)
				}
			}
		})
	}
}

func TestClaudeEngineApplyDefaultClaudeTools(t *testing.T) {
	engine := NewClaudeEngine()
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                     string
		inputTools               map[string]any
		expectedClaudeTools      []string
		expectedTopLevelTools    []string
		shouldNotHaveClaudeTools []string
		hasGitHubTool            bool
	}{
		{
			name: "adds default claude tools when github tool present",
			inputTools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"github", "claude"},
			hasGitHubTool:         true,
		},
		{
			name: "adds default github and claude tools when no github tool",
			inputTools: map[string]any{
				"other": map[string]any{
					"allowed": []any{"some_action"},
				},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"other", "github", "claude"},
			hasGitHubTool:         true,
		},
		{
			name: "preserves existing claude tools when github tool present (new format)",
			inputTools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
				"claude": map[string]any{
					"allowed": map[string]any{
						"Task": map[string]any{
							"custom": "config",
						},
						"Read": map[string]any{
							"timeout": 30,
						},
					},
				},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"github", "claude"},
			hasGitHubTool:         true,
		},
		{
			name: "adds only missing claude tools when some already exist (new format)",
			inputTools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
				"claude": map[string]any{
					"allowed": map[string]any{
						"Task": nil,
						"Grep": nil,
					},
				},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"github", "claude"},
			hasGitHubTool:         true,
		},
		{
			name: "handles empty github tool configuration",
			inputTools: map[string]any{
				"github": map[string]any{},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"github", "claude"},
			hasGitHubTool:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of input tools to avoid modifying the test data
			tools := make(map[string]any)
			for k, v := range tt.inputTools {
				tools[k] = v
			}

			// Apply both default tool functions in sequence
			tools = compiler.applyDefaultGitHubMCPTools(tools)
			result := engine.applyDefaultClaudeTools(tools, nil)

			// Check that all expected top-level tools are present
			for _, expectedTool := range tt.expectedTopLevelTools {
				if _, exists := result[expectedTool]; !exists {
					t.Errorf("Expected top-level tool '%s' to be present in result", expectedTool)
				}
			}

			// Check claude section if we expect claude tools
			if len(tt.expectedClaudeTools) > 0 {
				claudeSection, hasClaudeSection := result["claude"]
				if !hasClaudeSection {
					t.Error("Expected 'claude' section to exist")
					return
				}

				claudeConfig, ok := claudeSection.(map[string]any)
				if !ok {
					t.Error("Expected 'claude' section to be a map")
					return
				}

				// Check that the allowed section exists (new format)
				allowedSection, hasAllowed := claudeConfig["allowed"]
				if !hasAllowed {
					t.Error("Expected 'claude.allowed' section to exist")
					return
				}

				claudeTools, ok := allowedSection.(map[string]any)
				if !ok {
					t.Error("Expected 'claude.allowed' section to be a map")
					return
				}

				// Check that all expected Claude tools are present in the claude.allowed section
				for _, expectedTool := range tt.expectedClaudeTools {
					if _, exists := claudeTools[expectedTool]; !exists {
						t.Errorf("Expected Claude tool '%s' to be present in claude.allowed section", expectedTool)
					}
				}
			}

			// Check that tools that should not be present are indeed absent
			if len(tt.shouldNotHaveClaudeTools) > 0 {
				// Check top-level first
				for _, shouldNotHaveTool := range tt.shouldNotHaveClaudeTools {
					if _, exists := result[shouldNotHaveTool]; exists {
						t.Errorf("Expected tool '%s' to NOT be present at top level", shouldNotHaveTool)
					}
				}

				// Also check claude section doesn't exist or doesn't have these tools
				if claudeSection, hasClaudeSection := result["claude"]; hasClaudeSection {
					if claudeTools, ok := claudeSection.(map[string]any); ok {
						for _, shouldNotHaveTool := range tt.shouldNotHaveClaudeTools {
							if _, exists := claudeTools[shouldNotHaveTool]; exists {
								t.Errorf("Expected tool '%s' to NOT be present in claude section", shouldNotHaveTool)
							}
						}
					}
				}
			}

			// Verify github tool presence matches expectation
			_, hasGitHub := result["github"]
			if hasGitHub != tt.hasGitHubTool {
				t.Errorf("Expected github tool presence to be %v, got %v", tt.hasGitHubTool, hasGitHub)
			}

			// Verify that existing tool configurations are preserved
			if tt.name == "preserves existing claude tools when github tool present (new format)" {
				claudeSection := result["claude"].(map[string]any)
				allowedSection := claudeSection["allowed"].(map[string]any)

				if taskTool, ok := allowedSection["Task"].(map[string]any); ok {
					if custom, exists := taskTool["custom"]; !exists || custom != "config" {
						t.Errorf("Expected Task tool to preserve custom config, got %v", taskTool)
					}
				} else {
					t.Errorf("Expected Task tool to be a map[string]any with preserved config")
				}

				if readTool, ok := allowedSection["Read"].(map[string]any); ok {
					if timeout, exists := readTool["timeout"]; !exists || timeout != 30 {
						t.Errorf("Expected Read tool to preserve timeout config, got %v", readTool)
					}
				} else {
					t.Errorf("Expected Read tool to be a map[string]any with preserved config")
				}
			}
		})
	}
}

func TestClaudeEngineDefaultClaudeToolsList(t *testing.T) {
	// Test that ensures the default Claude tools list contains the expected tools
	// This test will need to be updated if the default tools list changes
	expectedDefaultTools := []string{
		"Task",
		"Glob",
		"Grep",
		"ExitPlanMode",
		"TodoWrite",
		"LS",
		"Read",
		"NotebookRead",
	}

	engine := NewClaudeEngine()
	compiler := NewCompiler(false, "", "test")

	// Create a minimal tools map with github tool to trigger the default Claude tools logic
	tools := map[string]any{
		"github": map[string]any{
			"allowed": []any{"list_issues"},
		},
	}

	// Apply both default tool functions in sequence
	tools = compiler.applyDefaultGitHubMCPTools(tools)
	result := engine.applyDefaultClaudeTools(tools, nil)

	// Verify the claude section was created
	claudeSection, hasClaudeSection := result["claude"]
	if !hasClaudeSection {
		t.Error("Expected 'claude' section to be created")
		return
	}

	claudeConfig, ok := claudeSection.(map[string]any)
	if !ok {
		t.Error("Expected 'claude' section to be a map")
		return
	}

	// Check that the allowed section exists (new format)
	allowedSection, hasAllowed := claudeConfig["allowed"]
	if !hasAllowed {
		t.Error("Expected 'claude.allowed' section to exist")
		return
	}

	claudeTools, ok := allowedSection.(map[string]any)
	if !ok {
		t.Error("Expected 'claude.allowed' section to be a map")
		return
	}

	// Verify all expected default Claude tools are added to the claude.allowed section
	for _, expectedTool := range expectedDefaultTools {
		if _, exists := claudeTools[expectedTool]; !exists {
			t.Errorf("Expected default Claude tool '%s' to be added, but it was not found", expectedTool)
		}
	}

	// Verify the count matches (github tool + claude section)
	expectedTopLevelCount := 2 // github tool + claude section
	if len(result) != expectedTopLevelCount {
		topLevelNames := make([]string, 0, len(result))
		for name := range result {
			topLevelNames = append(topLevelNames, name)
		}
		t.Errorf("Expected %d top-level tools in result (github + claude section), got %d: %v",
			expectedTopLevelCount, len(result), topLevelNames)
	}

	// Verify the claude section has the right number of tools
	if len(claudeTools) != len(expectedDefaultTools) {
		claudeToolNames := make([]string, 0, len(claudeTools))
		for name := range claudeTools {
			claudeToolNames = append(claudeToolNames, name)
		}
		t.Errorf("Expected %d tools in claude section, got %d: %v",
			len(expectedDefaultTools), len(claudeTools), claudeToolNames)
	}
}

func TestClaudeEngineDefaultClaudeToolsIntegrationWithComputeAllowedTools(t *testing.T) {
	// Test that default Claude tools are properly included in the allowed tools computation
	engine := NewClaudeEngine()
	compiler := NewCompiler(false, "", "test")

	tools := map[string]any{
		"github": map[string]any{
			"allowed": []any{"list_issues", "create_issue"},
		},
	}

	// Apply default tools first
	tools = compiler.applyDefaultGitHubMCPTools(tools)
	toolsWithDefaults := engine.applyDefaultClaudeTools(tools, nil)

	// Verify that the claude section was created with default tools (new format)
	claudeSection, hasClaudeSection := toolsWithDefaults["claude"]
	if !hasClaudeSection {
		t.Error("Expected 'claude' section to be created")
	}

	claudeConfig, ok := claudeSection.(map[string]any)
	if !ok {
		t.Error("Expected 'claude' section to be a map")
	}

	// Check that the allowed section exists
	allowedSection, hasAllowed := claudeConfig["allowed"]
	if !hasAllowed {
		t.Error("Expected 'claude' section to have 'allowed' subsection")
	}

	claudeTools, ok := allowedSection.(map[string]any)
	if !ok {
		t.Error("Expected 'claude.allowed' section to be a map")
	}

	// Verify default tools are present
	expectedClaudeTools := []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"}
	for _, expectedTool := range expectedClaudeTools {
		if _, exists := claudeTools[expectedTool]; !exists {
			t.Errorf("Expected claude.allowed section to contain '%s'", expectedTool)
		}
	}

	// Compute allowed tools
	allowedTools := engine.computeAllowedClaudeToolsString(toolsWithDefaults, nil)

	// Verify that default Claude tools appear in the allowed tools string
	for _, expectedTool := range expectedClaudeTools {
		if !strings.Contains(allowedTools, expectedTool) {
			t.Errorf("Expected allowed tools to contain '%s', but got: %s", expectedTool, allowedTools)
		}
	}

	// Verify github MCP tools are also present
	if !strings.Contains(allowedTools, "mcp__github__list_issues") {
		t.Errorf("Expected allowed tools to contain 'mcp__github__list_issues', but got: %s", allowedTools)
	}
	if !strings.Contains(allowedTools, "mcp__github__create_issue") {
		t.Errorf("Expected allowed tools to contain 'mcp__github__create_issue', but got: %s", allowedTools)
	}
}

func TestClaudeEngineComputeAllowedToolsWithSafeOutputs(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name        string
		tools       map[string]any
		safeOutputs *SafeOutputsConfig
		expected    string
	}{
		{
			name: "SafeOutputs with no tools - should add Write permission",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read": nil,
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Max: 1},
			},
			expected: "Read,Write",
		},
		{
			name: "SafeOutputs with general Write permission - should not add specific Write",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read":  nil,
						"Write": nil,
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Max: 1},
			},
			expected: "Read,Write",
		},
		{
			name: "No SafeOutputs - should not add Write permission",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read": nil,
					},
				},
			},
			safeOutputs: nil,
			expected:    "Read",
		},
		{
			name: "SafeOutputs with multiple output types",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":       nil,
						"BashOutput": nil,
						"KillBash":   nil,
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues:       &CreateIssuesConfig{Max: 1},
				AddIssueComments:   &AddIssueCommentsConfig{Max: 1},
				CreatePullRequests: &CreatePullRequestsConfig{Max: 1},
			},
			expected: "Bash,BashOutput,KillBash,Write",
		},
		{
			name: "SafeOutputs with MCP tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read": nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"create_issue", "create_pull_request"},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Max: 1},
			},
			expected: "Read,Write,mcp__github__create_issue,mcp__github__create_pull_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.computeAllowedClaudeToolsString(tt.tools, tt.safeOutputs)

			// Split both expected and result into slices and check each tool is present
			expectedTools := strings.Split(tt.expected, ",")
			resultTools := strings.Split(result, ",")

			// Check that all expected tools are present
			for _, expectedTool := range expectedTools {
				if expectedTool == "" {
					continue // Skip empty strings
				}
				found := false
				for _, actualTool := range resultTools {
					if actualTool == expectedTool {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected tool '%s' not found in result '%s'", expectedTool, result)
				}
			}

			// Check that no unexpected tools are present
			for _, actual := range resultTools {
				if actual == "" {
					continue // Skip empty strings
				}
				found := false
				for _, expected := range expectedTools {
					if expected == actual {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected tool '%s' found in result '%s'", actual, result)
				}
			}
		})
	}
}

func TestGenerateAllowedToolsComment(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name            string
		allowedToolsStr string
		indent          string
		expected        string
	}{
		{
			name:            "empty allowed tools",
			allowedToolsStr: "",
			indent:          "  ",
			expected:        "",
		},
		{
			name:            "single tool",
			allowedToolsStr: "Bash",
			indent:          "  ",
			expected:        "  # Allowed tools (sorted):\n  # - Bash\n",
		},
		{
			name:            "multiple tools",
			allowedToolsStr: "Bash,Edit,Read",
			indent:          "    ",
			expected:        "    # Allowed tools (sorted):\n    # - Bash\n    # - Edit\n    # - Read\n",
		},
		{
			name:            "tools with special characters",
			allowedToolsStr: "Bash(echo),mcp__github__get_issue,Write",
			indent:          "      ",
			expected:        "      # Allowed tools (sorted):\n      # - Bash(echo)\n      # - mcp__github__get_issue\n      # - Write\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.generateAllowedToolsComment(tt.allowedToolsStr, tt.indent)
			if result != tt.expected {
				t.Errorf("Expected comment:\n%q\nBut got:\n%q", tt.expected, result)
			}
		})
	}
}
