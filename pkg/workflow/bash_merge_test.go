package workflow

import (
	"testing"
)

// TestBashToolsMergeCustomWithDefaults tests that custom bash tools get merged with defaults
func TestBashToolsMergeCustomWithDefaults(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		tools       map[string]any
		safeOutputs *SafeOutputsConfig
		expected    []string
	}{
		{
			name: "bash with make commands should include defaults + make",
			tools: map[string]any{
				"bash": []any{"make:*"},
			},
			safeOutputs: nil,
			expected:    []string{"echo", "ls", "pwd", "cat", "head", "tail", "grep", "wc", "sort", "uniq", "date", "yq", "make:*"},
		},
		{
			name: "bash: true should be converted to wildcard",
			tools: map[string]any{
				"bash": true,
			},
			safeOutputs: nil,
			expected:    []string{"*"},
		},
		{
			name: "bash: false should be removed",
			tools: map[string]any{
				"bash": false,
			},
			safeOutputs: nil,
			expected:    nil, // bash should not exist
		},
		{
			name: "bash: true with safe outputs should use wildcard (not add git commands)",
			tools: map[string]any{
				"bash": true,
			},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expected: []string{"*"},
		},
		{
			name: "bash with multiple commands should include defaults + custom",
			tools: map[string]any{
				"bash": []any{"make:*", "npm:*"},
			},
			safeOutputs: nil,
			expected:    []string{"echo", "ls", "pwd", "cat", "head", "tail", "grep", "wc", "sort", "uniq", "date", "yq", "make:*", "npm:*"},
		},
		{
			name: "bash with empty array should remain empty",
			tools: map[string]any{
				"bash": []any{},
			},
			safeOutputs: nil,
			expected:    []string{},
		},
		{
			name: "bash with make commands and safe outputs should include defaults + make + git",
			tools: map[string]any{
				"bash": []any{"make:*"},
			},
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expected: []string{"echo", "ls", "pwd", "cat", "head", "tail", "grep", "wc", "sort", "uniq", "date", "yq", "make:*", "git checkout:*", "git branch:*", "git switch:*", "git add:*", "git rm:*", "git commit:*", "git merge:*", "git status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply default tools
			toolsConfig := NewTools(tt.tools)
			result := compiler.applyDefaultTools(toolsConfig, tt.safeOutputs)

			// Check the bash tools
			var bashTools []string
			exists := result.Bash != nil
			if exists {
				bashTools = result.Bash.AllowedCommands
			}

			// Handle case where bash should not exist (e.g., bash: false)
			if tt.expected == nil {
				if exists {
					t.Errorf("Expected bash to be removed, but it exists: %v", bashTools)
				}
				return
			}

			if !exists {
				t.Fatalf("Expected bash tools to exist")
			}

			// Compare commands - bashTools is already []string
			if len(bashTools) != len(tt.expected) {
				t.Logf("Actual tools: %v", bashTools)
				t.Logf("Expected tools: %v", tt.expected)
				t.Fatalf("Expected %d bash tools, got %d", len(tt.expected), len(bashTools))
			}

			// Check that all expected commands are present
			for _, expectedCmd := range tt.expected {
				found := false
				for _, actualCmd := range bashTools {
					if actualCmd == expectedCmd {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected command %q not found in actual commands: %v", expectedCmd, bashTools)
				}
			}
		})
	}
}
