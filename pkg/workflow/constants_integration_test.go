package workflow

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestConstantsIntegration verifies that constants can be accessed from the workflow package
func TestConstantsIntegration(t *testing.T) {
	// Test that DefaultGitHubTools constant is accessible and not empty
	if len(constants.DefaultGitHubTools) == 0 {
		t.Error("DefaultGitHubTools constant should not be empty")
	}

	// Test that it contains expected tools
	expectedTools := []string{
		"get_issue",
		"list_issues",
		"search_repositories",
		"get_commit",
		"get_file_contents",
	}

	toolsMap := make(map[string]bool)
	for _, tool := range constants.DefaultGitHubTools {
		toolsMap[tool] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolsMap[expectedTool] {
			t.Errorf("Expected tool '%s' not found in DefaultGitHubTools", expectedTool)
		}
	}
}

// TestClaudeCanAccessGitHubTools demonstrates that Claude engine can access the GitHub tools constant
func TestClaudeCanAccessGitHubTools(t *testing.T) {
	engine := NewClaudeEngine()
	if engine == nil {
		t.Fatal("Failed to create Claude engine")
	}

	// Demonstrate that Claude can access the constant
	gitHubTools := constants.DefaultGitHubTools
	if len(gitHubTools) == 0 {
		t.Error("Claude engine should be able to access DefaultGitHubTools constant")
	}

	// Verify specific tools that would be useful for Claude
	toolsMap := make(map[string]bool)
	for _, tool := range gitHubTools {
		toolsMap[tool] = true
	}

	claudeRelevantTools := []string{
		"get_issue",
		"get_pull_request",
		"search_code",
		"list_commits",
	}

	for _, tool := range claudeRelevantTools {
		if !toolsMap[tool] {
			t.Errorf("Claude-relevant tool '%s' not found in DefaultGitHubTools", tool)
		}
	}
}

// TestDefaultClaudeTools verifies that DefaultClaudeTools constant contains expected tools
func TestDefaultClaudeTools(t *testing.T) {
	// Verify constant is accessible and not empty
	claudeTools := constants.DefaultClaudeTools
	if len(claudeTools) == 0 {
		t.Error("DefaultClaudeTools constant should not be empty")
	}

	// Verify expected tools are present
	expectedTools := []string{
		"Task",
		"Glob",
		"Grep",
		"ExitPlanMode",
		"TodoWrite",
		"LS",
		"Read",
		"NotebookRead",
	}

	toolsMap := make(map[string]bool)
	for _, tool := range claudeTools {
		toolsMap[tool] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolsMap[expectedTool] {
			t.Errorf("Expected Claude tool '%s' not found in DefaultClaudeTools", expectedTool)
		}
	}

	// Verify that the constant has exactly the expected tools (no more, no less)
	if len(claudeTools) != len(expectedTools) {
		t.Errorf("Expected %d tools in DefaultClaudeTools, got %d", len(expectedTools), len(claudeTools))
	}
}

// TestDefaultCopilotTools verifies that DefaultCopilotTools constant contains expected tools
func TestDefaultCopilotTools(t *testing.T) {
	// Verify constant is accessible and not empty
	copilotTools := constants.DefaultCopilotTools
	if len(copilotTools) == 0 {
		t.Error("DefaultCopilotTools constant should not be empty")
	}

	// Verify expected tools are present
	expectedTools := []string{
		"read",
		"write",
		"shell",
		"web-fetch",
		"web-search",
	}

	toolsMap := make(map[string]bool)
	for _, tool := range copilotTools {
		toolsMap[tool] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolsMap[expectedTool] {
			t.Errorf("Expected Copilot tool '%s' not found in DefaultCopilotTools", expectedTool)
		}
	}

	// Verify that the constant has exactly the expected tools (no more, no less)
	if len(copilotTools) != len(expectedTools) {
		t.Errorf("Expected %d tools in DefaultCopilotTools, got %d", len(expectedTools), len(copilotTools))
	}
}
