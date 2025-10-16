package workflow

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestGitHubToolsModeSeparation verifies that local and remote GitHub tools lists are properly separated
func TestGitHubToolsModeSeparation(t *testing.T) {
	// Verify both lists exist and are not empty
	if len(constants.DefaultGitHubToolsLocal) == 0 {
		t.Error("DefaultGitHubToolsLocal should not be empty")
	}

	if len(constants.DefaultGitHubToolsRemote) == 0 {
		t.Error("DefaultGitHubToolsRemote should not be empty")
	}

	// Verify backward compatibility - DefaultGitHubTools should point to local
	if len(constants.DefaultGitHubTools) == 0 {
		t.Error("DefaultGitHubTools should not be empty (backward compatibility)")
	}

	// Verify DefaultGitHubTools points to the same data as DefaultGitHubToolsLocal
	if len(constants.DefaultGitHubTools) != len(constants.DefaultGitHubToolsLocal) {
		t.Errorf("DefaultGitHubTools should have same length as DefaultGitHubToolsLocal for backward compatibility")
	}

	// Verify they contain expected core tools
	expectedCoreTools := []string{
		"get_issue",
		"list_issues",
		"get_commit",
		"get_file_contents",
		"search_repositories",
	}

	// Check local tools
	localToolsMap := make(map[string]bool)
	for _, tool := range constants.DefaultGitHubToolsLocal {
		localToolsMap[tool] = true
	}

	for _, expectedTool := range expectedCoreTools {
		if !localToolsMap[expectedTool] {
			t.Errorf("Expected core tool '%s' not found in DefaultGitHubToolsLocal", expectedTool)
		}
	}

	// Check remote tools
	remoteToolsMap := make(map[string]bool)
	for _, tool := range constants.DefaultGitHubToolsRemote {
		remoteToolsMap[tool] = true
	}

	for _, expectedTool := range expectedCoreTools {
		if !remoteToolsMap[expectedTool] {
			t.Errorf("Expected core tool '%s' not found in DefaultGitHubToolsRemote", expectedTool)
		}
	}
}

// TestApplyDefaultToolsUsesCorrectMode verifies that applyDefaultTools uses the correct tool list based on mode
func TestApplyDefaultToolsUsesCorrectMode(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name         string
		tools        map[string]any
		expectedList string // "local" or "remote"
	}{
		{
			name: "Local mode (default)",
			tools: map[string]any{
				"github": map[string]any{},
			},
			expectedList: "local",
		},
		{
			name: "Explicit local mode",
			tools: map[string]any{
				"github": map[string]any{
					"mode": "local",
				},
			},
			expectedList: "local",
		},
		{
			name: "Remote mode",
			tools: map[string]any{
				"github": map[string]any{
					"mode": "remote",
				},
			},
			expectedList: "remote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.applyDefaultTools(tt.tools, nil)

			// Get the allowed tools from the github configuration
			githubConfig, ok := result["github"].(map[string]any)
			if !ok {
				t.Fatal("Expected github configuration to be a map")
			}

			allowed, ok := githubConfig["allowed"].([]any)
			if !ok {
				t.Fatal("Expected allowed to be a slice")
			}

			// Verify that the number of tools matches the expected list
			var expectedCount int
			if tt.expectedList == "local" {
				expectedCount = len(constants.DefaultGitHubToolsLocal)
			} else {
				expectedCount = len(constants.DefaultGitHubToolsRemote)
			}

			if len(allowed) != expectedCount {
				t.Errorf("Expected %d tools for %s mode, got %d", expectedCount, tt.expectedList, len(allowed))
			}
		})
	}
}
