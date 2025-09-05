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
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with specific commands in claude section (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"echo", "ls"},
					},
				},
			},
			expected: "Bash(echo),Bash(ls),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with nil value (all commands allowed)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": nil,
					},
				},
			},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "regular tools in claude section (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"WebFetch":  nil,
						"WebSearch": nil,
					},
				},
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,WebSearch",
		},
		{
			name: "mcp tools",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues", "create_issue"},
				},
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,mcp__github__create_issue,mcp__github__list_issues",
		},
		{
			name: "mixed claude and mcp tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"WebFetch":  nil,
						"WebSearch": nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,WebSearch,mcp__github__list_issues",
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
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,mcp__custom_server__tool1,mcp__custom_server__tool2",
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
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,mcp__notion",
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
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,mcp__github__create_issue,mcp__github__list_issues,mcp__notion",
		},
		{
			name: "bash with :* wildcard (should ignore other bash tools)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{":*"},
					},
				},
			},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with :* wildcard mixed with other commands (should ignore other commands)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"echo", "ls", ":*", "cat"},
					},
				},
			},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with :* wildcard and other tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":     []any{":*"},
						"WebFetch": nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,mcp__github__list_issues",
		},
		{
			name: "bash with single command should include implicit tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"ls"},
					},
				},
			},
			expected: "Bash(ls),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "explicit KillBash and BashOutput should not duplicate",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"echo"},
					},
				},
			},
			expected: "Bash(echo),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "no bash tools means no implicit tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"WebFetch":  nil,
						"WebSearch": nil,
					},
				},
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,WebSearch",
		},
		// Test cases for new neutral tools format
		{
			name: "neutral bash tool",
			tools: map[string]any{
				"bash": []any{"echo", "ls"},
			},
			expected: "Bash(echo),Bash(ls),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "neutral web-fetch tool",
			tools: map[string]any{
				"web-fetch": nil,
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebFetch",
		},
		{
			name: "neutral web-search tool",
			tools: map[string]any{
				"web-search": nil,
			},
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,WebSearch",
		},
		{
			name: "neutral edit tool",
			tools: map[string]any{
				"edit": nil,
			},
			expected: "Edit,ExitPlanMode,Glob,Grep,LS,MultiEdit,NotebookEdit,NotebookRead,Read,Task,TodoWrite,Write",
		},
		{
			name: "mixed neutral and MCP tools",
			tools: map[string]any{
				"web-fetch": nil,
				"bash":      []any{"git status"},
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expected: "Bash(git status),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite,WebFetch,mcp__github__list_issues",
		},
		{
			name: "all neutral tools together",
			tools: map[string]any{
				"bash":       []any{"echo"},
				"web-fetch":  nil,
				"web-search": nil,
				"edit":       nil,
			},
			expected: "Bash(echo),BashOutput,Edit,ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit,NotebookEdit,NotebookRead,Read,Task,TodoWrite,WebFetch,WebSearch,Write",
		},
		{
			name: "neutral bash with nil value (all commands)",
			tools: map[string]any{
				"bash": nil,
			},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
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
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,Write",
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
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,Write",
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
			expected:    "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite",
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
			expected: "Bash,BashOutput,Edit,ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit,NotebookEdit,NotebookRead,Read,Task,TodoWrite,Write",
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
			expected: "ExitPlanMode,Glob,Grep,LS,NotebookRead,Read,Task,TodoWrite,Write,mcp__github__create_issue,mcp__github__create_pull_request",
		},
		{
			name: "SafeOutputs with neutral tools and create-pull-request",
			tools: map[string]any{
				"bash":      []any{"echo", "ls"},
				"web-fetch": nil,
			},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Max: 1},
			},
			expected: "Bash(echo),Bash(git add:*),Bash(git branch:*),Bash(git checkout:*),Bash(git commit:*),Bash(git merge:*),Bash(git rm:*),Bash(git switch:*),Bash(ls),BashOutput,Edit,ExitPlanMode,Glob,Grep,KillBash,LS,MultiEdit,NotebookEdit,NotebookRead,Read,Task,TodoWrite,WebFetch,Write",
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
