package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateIssueWorkflowCompilationWithAssignees tests end-to-end workflow compilation with assignees
func TestCreateIssueWorkflowCompilationWithAssignees(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "assignees-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `---
name: Test Assignees Feature
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: claude
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
    labels: [automation, ai-generated]
    assignees: [user1, user2, bot-helper]
---

# Test Workflow with Assignees

This is a test workflow that should create an issue and assign it to multiple users.
`

	testFile := filepath.Join(tmpDir, "test-assignees.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-assignees.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that create_issue job exists
	if !strings.Contains(compiledStr, "create_issue:") {
		t.Error("Expected create_issue job in compiled workflow")
	}

	// Verify that assignee steps are present
	if !strings.Contains(compiledStr, "Assign issue to user1") {
		t.Error("Expected assignee step for user1 in compiled workflow")
	}
	if !strings.Contains(compiledStr, "Assign issue to user2") {
		t.Error("Expected assignee step for user2 in compiled workflow")
	}
	if !strings.Contains(compiledStr, "Assign issue to bot-helper") {
		t.Error("Expected assignee step for bot-helper in compiled workflow")
	}

	// Verify gh issue edit command
	if !strings.Contains(compiledStr, "gh issue edit") {
		t.Error("Expected gh issue edit command in compiled workflow")
	}

	// Verify --add-assignee flag
	if !strings.Contains(compiledStr, "--add-assignee") {
		t.Error("Expected --add-assignee flag in compiled workflow")
	}

	// Verify ISSUE_NUMBER from step output
	if !strings.Contains(compiledStr, "${{ steps.create_issue.outputs.issue_number }}") {
		t.Error("Expected ISSUE_NUMBER to reference create_issue step output")
	}

	// Verify conditional execution
	if !strings.Contains(compiledStr, "if: steps.create_issue.outputs.issue_number != ''") {
		t.Error("Expected conditional if statement for assignee steps")
	}

	// Verify GH_TOKEN is set with proper token expression
	if !strings.Contains(compiledStr, "GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}") {
		t.Error("Expected GH_TOKEN environment variable with proper token expression in compiled workflow")
	}

	// Verify environment variables for assignees are properly quoted
	if !strings.Contains(compiledStr, `ASSIGNEE: "user1"`) {
		t.Error("Expected quoted ASSIGNEE environment variable for user1")
	}
	if !strings.Contains(compiledStr, `ASSIGNEE: "user2"`) {
		t.Error("Expected quoted ASSIGNEE environment variable for user2")
	}
	if !strings.Contains(compiledStr, `ASSIGNEE: "bot-helper"`) {
		t.Error("Expected quoted ASSIGNEE environment variable for bot-helper")
	}
}

// TestCreateIssueWorkflowCompilationWithoutAssignees tests that workflows without assignees still work
func TestCreateIssueWorkflowCompilationWithoutAssignees(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "no-assignees-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `---
name: Test Without Assignees
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: claude
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
    labels: [automation]
---

# Test Workflow without Assignees

This workflow should compile successfully without assignees configuration.
`

	testFile := filepath.Join(tmpDir, "test-no-assignees.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-no-assignees.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that create_issue job exists
	if !strings.Contains(compiledStr, "create_issue:") {
		t.Error("Expected create_issue job in compiled workflow")
	}

	// Verify that no assignee steps are present
	if strings.Contains(compiledStr, "Assign issue to") {
		t.Error("Did not expect assignee steps in workflow without assignees")
	}
	if strings.Contains(compiledStr, "gh issue edit") {
		t.Error("Did not expect gh issue edit command in workflow without assignees")
	}
}

// TestCreateIssueWorkflowWithCopilotAssignee tests that "copilot" is mapped to "copilot-swe-agent"
func TestCreateIssueWorkflowWithCopilotAssignee(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "copilot-assignee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `---
name: Test Copilot Assignee
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-issue:
    assignees: copilot
---

# Test Workflow

Create an issue and assign to copilot.
`

	testFile := filepath.Join(tmpDir, "test-copilot.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-copilot.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that step name shows "copilot"
	if !strings.Contains(compiledStr, "Assign issue to copilot") {
		t.Error("Expected assignee step name to show 'copilot'")
	}

	// Verify that actual assignee is "copilot-swe-agent"
	if !strings.Contains(compiledStr, `ASSIGNEE: "copilot-swe-agent"`) {
		t.Error("Expected ASSIGNEE to be mapped to 'copilot-swe-agent'")
	}

	// Verify that "copilot" is NOT used as the actual assignee value
	if strings.Contains(compiledStr, `ASSIGNEE: "copilot"`) {
		t.Error("Did not expect 'copilot' to be used directly as assignee value")
	}
}

// TestCreateIssueWorkflowWithStringAssignee tests that single string assignee works
func TestCreateIssueWorkflowWithStringAssignee(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "string-assignee-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `---
name: Test String Assignee
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-issue:
    assignees: single-user
---

# Test Workflow

Create an issue with a single assignee.
`

	testFile := filepath.Join(tmpDir, "test-string.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-string.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that assignee step is created
	if !strings.Contains(compiledStr, "Assign issue to single-user") {
		t.Error("Expected assignee step for single-user")
	}

	// Verify the assignee environment variable
	if !strings.Contains(compiledStr, `ASSIGNEE: "single-user"`) {
		t.Error("Expected ASSIGNEE environment variable for single-user")
	}
}
