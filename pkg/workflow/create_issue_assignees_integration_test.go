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
  issues: read
  pull-requests: read
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

	// After the change, assignee steps are only generated when "copilot" is in the list
	// Since the assignees list doesn't include copilot, no assignee steps should be present
	if strings.Contains(compiledStr, "Assign issue to") {
		t.Error("Did not expect assignee steps when copilot is not in the assignees list")
	}

	if strings.Contains(compiledStr, "Checkout repository for gh CLI") {
		t.Error("Did not expect checkout step when copilot is not in the assignees list")
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
  issues: read
  pull-requests: read
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

// TestCreateIssueWorkflowWithCopilotAssignee tests that "copilot" is mapped to "@copilot"
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
  issues: read
  pull-requests: read
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

	// Verify that actual assignee is "@copilot" (gh CLI special value)
	if !strings.Contains(compiledStr, `ASSIGNEE: "@copilot"`) {
		t.Error("Expected ASSIGNEE to be mapped to '@copilot'")
	}

	// Verify that "copilot" without @ is NOT used as the actual assignee value
	if strings.Contains(compiledStr, `ASSIGNEE: "copilot"`) && !strings.Contains(compiledStr, `ASSIGNEE: "@copilot"`) {
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
  issues: read
  pull-requests: read
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

	// After the change, assignee steps are only generated when "copilot" is in the list
	// Since the assignee is not copilot, no assignee steps should be generated
	if strings.Contains(compiledStr, "Assign issue to") {
		t.Error("Did not expect assignee steps when copilot is not in the assignees list")
	}

	if strings.Contains(compiledStr, "Checkout repository for gh CLI") {
		t.Error("Did not expect checkout step when copilot is not in the assignees list")
	}
}
