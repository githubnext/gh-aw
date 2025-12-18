package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestCreateIssueWorkflowCompilationWithAssignees tests end-to-end workflow compilation with assignees
func TestCreateIssueWorkflowCompilationWithAssignees(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "assignees-test")

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
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-assignees.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that safe_outputs job exists
	if !strings.Contains(compiledStr, "safe_outputs:") {
		t.Error("Expected safe_outputs job in compiled workflow")
	}

	// Verify that create_issue step is present
	if !strings.Contains(compiledStr, "id: create_issue") {
		t.Error("Expected create_issue step in compiled workflow")
	}

	// Verify actions/github-script is used
	if !strings.Contains(compiledStr, "actions/github-script") {
		t.Error("Expected actions/github-script to be used in compiled workflow")
	}

	// Verify assignees are mentioned in the workflow (in description or config)
	if !strings.Contains(compiledStr, "user1") || !strings.Contains(compiledStr, "user2") {
		t.Error("Expected assignees to be referenced in compiled workflow")
	}

	// Verify GH_TOKEN is set with proper token expression  
	if !strings.Contains(compiledStr, "github-token:") {
		t.Error("Expected github-token to be set in compiled workflow")
	}
}

// TestCreateIssueWorkflowCompilationWithoutAssignees tests that workflows without assignees still work
func TestCreateIssueWorkflowCompilationWithoutAssignees(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "no-assignees-test")

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
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-no-assignees.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that safe_outputs job exists
	if !strings.Contains(compiledStr, "safe_outputs:") {
		t.Error("Expected safe_outputs job in compiled workflow")
	}

	// Verify that no assignee steps are present
	if strings.Contains(compiledStr, "Assign issue to") {
		t.Error("Did not expect assignee steps in workflow without assignees")
	}
	if strings.Contains(compiledStr, "gh issue edit") {
		t.Error("Did not expect gh issue edit command in workflow without assignees")
	}
}

// TestCreateIssueWorkflowWithCopilotAssignee tests that copilot assignment is done
// via a separate step with the agent token (GH_AW_AGENT_TOKEN)
func TestCreateIssueWorkflowWithCopilotAssignee(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "copilot-assignee-test")

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
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-copilot.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledStr := string(compiledContent)

	// Verify that there's a separate step for copilot assignment
	if !strings.Contains(compiledStr, "Assign copilot to created issues") {
		t.Error("Expected separate step 'Assign copilot to created issues' for copilot assignment")
	}

	// Verify that the step uses agent token (GH_AW_AGENT_TOKEN)
	if !strings.Contains(compiledStr, "GH_AW_AGENT_TOKEN") {
		t.Error("Expected copilot assignment step to use GH_AW_AGENT_TOKEN")
	}

	// Verify that the step is conditioned on issues_to_assign_copilot output
	if !strings.Contains(compiledStr, "issues_to_assign_copilot") {
		t.Error("Expected copilot assignment step to reference issues_to_assign_copilot output")
	}

	// Verify GH_AW_ASSIGN_COPILOT env var is set in create_issue step
	if !strings.Contains(compiledStr, "GH_AW_ASSIGN_COPILOT") {
		t.Error("Expected GH_AW_ASSIGN_COPILOT environment variable to be set")
	}
}

// TestCreateIssueWorkflowWithStringAssignee tests that single string assignee works
func TestCreateIssueWorkflowWithStringAssignee(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "string-assignee-test")

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
	if err := compiler.CompileWorkflow(testFile); err != nil {
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
