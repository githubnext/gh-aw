package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestGitCommandsIntegrationWithCreatePullRequest(t *testing.T) {
	// Create a simple workflow with create-pull-request enabled
	workflowContent := `---
name: Test Git Commands Integration
tools:
  claude:
    allowed:
      Read: null
      Write: null
safe-outputs:
  create-pull-request:
    max: 1
---

This is a test workflow that should automatically get Git commands when create-pull-request is enabled.
`

	compiler := NewCompiler(false, "", "test")
	engine := NewClaudeEngine()

	// Parse the workflow content and compile it
	result, err := compiler.parseWorkflowMarkdownContent(workflowContent)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Check that Git commands were automatically added to the tools
	claudeSection, hasClaudeSection := result.Tools["claude"]
	if !hasClaudeSection {
		t.Fatal("Expected claude section to be present")
	}

	claudeConfig, ok := claudeSection.(map[string]any)
	if !ok {
		t.Fatal("Expected claude section to be a map")
	}

	allowed, hasAllowed := claudeConfig["allowed"]
	if !hasAllowed {
		t.Fatal("Expected claude section to have allowed tools")
	}

	allowedMap, ok := allowed.(map[string]any)
	if !ok {
		t.Fatal("Expected allowed to be a map")
	}

	bashTool, hasBash := allowedMap["Bash"]
	if !hasBash {
		t.Fatal("Expected Bash tool to be present when create-pull-request is enabled")
	}

	// Verify that Git commands are present
	bashCommands, ok := bashTool.([]any)
	if !ok {
		t.Fatal("Expected Bash tool to have command list")
	}

	gitCommandsFound := 0
	expectedGitCommands := []string{"git checkout:*", "git add:*", "git commit:*", "git branch:*", "git switch:*", "git rm:*", "git merge:*"}

	for _, cmd := range bashCommands {
		if cmdStr, ok := cmd.(string); ok {
			for _, expectedCmd := range expectedGitCommands {
				if cmdStr == expectedCmd {
					gitCommandsFound++
					break
				}
			}
		}
	}

	if gitCommandsFound != len(expectedGitCommands) {
		t.Errorf("Expected %d Git commands, found %d. Commands: %v", len(expectedGitCommands), gitCommandsFound, bashCommands)
	}

	// Verify allowed tools include the Git commands
	allowedToolsStr := engine.computeAllowedClaudeToolsString(result.Tools, result.SafeOutputs)
	if !strings.Contains(allowedToolsStr, "Bash(git checkout:*)") {
		t.Errorf("Expected allowed tools to contain Git commands, got: %s", allowedToolsStr)
	}
}

func TestGitCommandsNotAddedWithoutPullRequestOutput(t *testing.T) {
	// Create a workflow with only create-issue (no PR-related outputs)
	workflowContent := `---
name: Test No Git Commands
tools:
  claude:
    allowed:
      Read: null
      Write: null
safe-outputs:
  create-issue:
    max: 1
---

This workflow should NOT get Git commands since it doesn't use create-pull-request or push-to-branch.
`

	compiler := NewCompiler(false, "", "test")
	engine := NewClaudeEngine()

	// Parse the workflow content
	result, err := compiler.parseWorkflowMarkdownContent(workflowContent)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Check that Git commands were NOT automatically added
	claudeSection, hasClaudeSection := result.Tools["claude"]
	if !hasClaudeSection {
		t.Fatal("Expected claude section to be present")
	}

	claudeConfig, ok := claudeSection.(map[string]any)
	if !ok {
		t.Fatal("Expected claude section to be a map")
	}

	allowed, hasAllowed := claudeConfig["allowed"]
	if !hasAllowed {
		t.Fatal("Expected claude section to have allowed tools")
	}

	allowedMap, ok := allowed.(map[string]any)
	if !ok {
		t.Fatal("Expected allowed to be a map")
	}

	// Bash tool should NOT be present since no Git commands were needed
	_, hasBash := allowedMap["Bash"]
	if hasBash {
		t.Error("Did not expect Bash tool to be present when only create-issue is enabled")
	}

	// Verify allowed tools do not include Git commands
	allowedToolsStr := engine.computeAllowedClaudeToolsString(result.Tools, result.SafeOutputs)
	if strings.Contains(allowedToolsStr, "Bash(git") {
		t.Errorf("Did not expect allowed tools to contain Git commands, got: %s", allowedToolsStr)
	}
}

func TestAdditionalClaudeToolsIntegrationWithCreatePullRequest(t *testing.T) {
	// Create a simple workflow with create-pull-request enabled
	workflowContent := `---
name: Test Additional Claude Tools Integration
tools:
  claude:
    allowed:
      Read: null
      Task: null
safe-outputs:
  create-pull-request:
    max: 1
---

This is a test workflow that should automatically get additional Claude tools when create-pull-request is enabled.
`

	compiler := NewCompiler(false, "", "test")
	engine := NewClaudeEngine()

	// Parse the workflow content and compile it
	result, err := compiler.parseWorkflowMarkdownContent(workflowContent)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Check that additional Claude tools were automatically added
	claudeSection, hasClaudeSection := result.Tools["claude"]
	if !hasClaudeSection {
		t.Fatal("Expected claude section to be present")
	}

	claudeConfig, ok := claudeSection.(map[string]any)
	if !ok {
		t.Fatal("Expected claude section to be a map")
	}

	allowed, hasAllowed := claudeConfig["allowed"]
	if !hasAllowed {
		t.Fatal("Expected claude section to have allowed tools")
	}

	allowedMap, ok := allowed.(map[string]any)
	if !ok {
		t.Fatal("Expected allowed to be a map")
	}

	// Verify that additional Claude tools are present
	expectedAdditionalTools := []string{"Edit", "MultiEdit", "Write", "NotebookEdit"}
	for _, expectedTool := range expectedAdditionalTools {
		if _, exists := allowedMap[expectedTool]; !exists {
			t.Errorf("Expected additional Claude tool %s to be present", expectedTool)
		}
	}

	// Verify that pre-existing tools are still there
	if _, exists := allowedMap["Read"]; !exists {
		t.Error("Expected pre-existing Read tool to be preserved")
	}
	if _, exists := allowedMap["Task"]; !exists {
		t.Error("Expected pre-existing Task tool to be preserved")
	}

	// Verify allowed tools include the additional Claude tools
	allowedToolsStr := engine.computeAllowedClaudeToolsString(result.Tools, result.SafeOutputs)
	for _, expectedTool := range expectedAdditionalTools {
		if !strings.Contains(allowedToolsStr, expectedTool) {
			t.Errorf("Expected allowed tools to contain %s, got: %s", expectedTool, allowedToolsStr)
		}
	}
}

func TestAdditionalClaudeToolsIntegrationWithPushToBranch(t *testing.T) {
	// Create a simple workflow with push-to-branch enabled
	workflowContent := `---
name: Test Additional Claude Tools Integration with Push to Branch
tools:
  claude:
    allowed:
      Read: null
safe-outputs:
  push-to-branch:
    branch: "feature-branch"
---

This is a test workflow that should automatically get additional Claude tools when push-to-branch is enabled.
`

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow content and compile it
	result, err := compiler.parseWorkflowMarkdownContent(workflowContent)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Check that additional Claude tools were automatically added
	claudeSection, hasClaudeSection := result.Tools["claude"]
	if !hasClaudeSection {
		t.Fatal("Expected claude section to be present")
	}

	claudeConfig, ok := claudeSection.(map[string]any)
	if !ok {
		t.Fatal("Expected claude section to be a map")
	}

	allowed, hasAllowed := claudeConfig["allowed"]
	if !hasAllowed {
		t.Fatal("Expected claude section to have allowed tools")
	}

	allowedMap, ok := allowed.(map[string]any)
	if !ok {
		t.Fatal("Expected allowed to be a map")
	}

	// Verify that additional Claude tools are present
	expectedAdditionalTools := []string{"Edit", "MultiEdit", "Write", "NotebookEdit"}
	for _, expectedTool := range expectedAdditionalTools {
		if _, exists := allowedMap[expectedTool]; !exists {
			t.Errorf("Expected additional Claude tool %s to be present", expectedTool)
		}
	}
}

// Helper function to parse workflow content like parseWorkflowFile but from string
func (c *Compiler) parseWorkflowMarkdownContent(content string) (*WorkflowData, error) {
	// This would normally be in parseWorkflowFile, but we'll extract the core logic for testing
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return nil, err
	}
	engine := NewClaudeEngine()

	// Extract SafeOutputs early
	safeOutputs := c.extractSafeOutputsConfig(result.Frontmatter)

	// Extract and process tools
	topTools := extractToolsFromFrontmatter(result.Frontmatter)
	topTools = c.applyDefaultGitHubMCPTools(topTools)
	tools := engine.applyDefaultClaudeTools(topTools, safeOutputs)

	// Build basic workflow data for testing
	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		Tools:       tools,
		SafeOutputs: safeOutputs,
		AI:          "claude",
	}

	return workflowData, nil
}
