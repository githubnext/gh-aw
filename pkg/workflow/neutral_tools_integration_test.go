package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNeutralToolsIntegration(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true) // Skip schema validation for this test
	tempDir := t.TempDir()

	workflowContent := `---
on:
  workflow_dispatch:

engine: 
  id: claude

tools:
  bash: ["echo", "ls"]
  web-fetch:
  web-search:
  edit:
  github:
    allowed: ["list_issues"]

safe-outputs:
  create-pull-request:
    title-prefix: "[test] "
---

Test workflow with neutral tools format.
`

	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow file
	lockFilePath := filepath.Join(tempDir, "test-workflow.lock.yml")
	yamlBytes, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	yamlContent := string(yamlBytes)

	// Should contain Claude tools that were converted from neutral tools
	expectedClaudeTools := []string{
		"Bash(echo)",
		"Bash(ls)",
		"BashOutput",
		"KillBash",
		"WebFetch",
		"WebSearch",
		"Edit",
		"MultiEdit",
		"NotebookEdit",
		"Write",
	}

	for _, tool := range expectedClaudeTools {
		if !strings.Contains(yamlContent, tool) {
			t.Errorf("Expected Claude tool '%s' not found in compiled YAML", tool)
		}
	}

	// Should also contain MCP tools
	if !strings.Contains(yamlContent, "mcp__github__list_issues") {
		t.Error("Expected MCP tool 'mcp__github__list_issues' not found in compiled YAML")
	}

	// Should contain Git commands due to safe-outputs create-pull-request
	expectedGitTools := []string{
		"Bash(git add:*)",
		"Bash(git commit:*)",
		"Bash(git checkout:*)",
	}

	for _, tool := range expectedGitTools {
		if !strings.Contains(yamlContent, tool) {
			t.Errorf("Expected Git tool '%s' not found in compiled YAML", tool)
		}
	}

	// Verify that the old format is not present in the compiled output
	if strings.Contains(yamlContent, "bash:") || strings.Contains(yamlContent, "web-fetch:") {
		t.Error("Compiled YAML should not contain neutral tool names directly")
	}
}

func TestBackwardCompatibilityWithClaudeFormat(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true) // Skip schema validation for this test
	tempDir := t.TempDir()

	workflowContent := `---
on:
  workflow_dispatch:

engine: 
  id: claude

tools:
  web-fetch:
  bash: ["echo", "ls"]
  github:
    allowed: ["list_issues"]
---

Test workflow with legacy Claude tools format.
`

	workflowPath := filepath.Join(tempDir, "legacy-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow file
	lockFilePath := filepath.Join(tempDir, "legacy-workflow.lock.yml")
	yamlBytes, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	yamlContent := string(yamlBytes)

	expectedTools := []string{
		"Bash(echo)",
		"Bash(ls)",
		"BashOutput",
		"KillBash",
		"WebFetch",
		"mcp__github__list_issues",
	}

	for _, tool := range expectedTools {
		if !strings.Contains(yamlContent, tool) {
			t.Errorf("Expected tool '%s' not found in compiled YAML", tool)
		}
	}
}
