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

	// Verify actions/github-script is used
	if !strings.Contains(compiledStr, "actions/github-script") {
		t.Error("Expected actions/github-script to be used in compiled workflow")
	}

	// Verify exec.exec is used for gh CLI
	if !strings.Contains(compiledStr, "exec.exec") {
		t.Error("Expected exec.exec to be used in assign script")
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

	// Verify checkout step is present
	if !strings.Contains(compiledStr, "Checkout repository for gh CLI") {
		t.Error("Expected checkout step for gh CLI in compiled workflow")
	}

	if !strings.Contains(compiledStr, "uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8") {
		t.Error("Expected checkout to use actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8 in compiled workflow")
	}

	// Verify checkout step is conditional on issue creation
	checkoutPattern := "Checkout repository for gh CLI"
	checkoutIndex := strings.Index(compiledStr, checkoutPattern)
	if checkoutIndex != -1 {
		// Check that conditional appears after the checkout step name
		afterCheckout := compiledStr[checkoutIndex:]
		if !strings.Contains(afterCheckout[:200], "if: steps.create_issue.outputs.issue_number != ''") {
			t.Error("Expected checkout step to be conditional on issue creation")
		}
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

// TestCreateIssueWorkflowWithCopilotAssignee tests that "copilot" uses GitHub API with copilot[bot]
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

	// Verify copilot uses GitHub API with copilot[bot]
	if !strings.Contains(compiledStr, "copilot[bot]") {
		t.Error("Expected copilot to use copilot[bot] via GitHub API")
	}

	// Verify that it uses the GitHub API (not gh issue edit)
	if !strings.Contains(compiledStr, "gh api --method POST") {
		t.Error("Expected GitHub API call for copilot assignee")
	}

	// Verify that it uses the correct API endpoint
	if !strings.Contains(compiledStr, "/assignees") {
		t.Error("Expected /assignees API endpoint")
	}

	// Find the assignee step section (after "Assign issue to copilot")
	assigneeStepIndex := strings.Index(compiledStr, "Assign issue to copilot")
	if assigneeStepIndex == -1 {
		t.Fatal("Could not find assignee step")
	}
	assigneeStepContent := compiledStr[assigneeStepIndex:]

	// Find the next step or end of content (limit to this step only)
	nextStepIndex := strings.Index(assigneeStepContent[len("Assign issue to copilot"):], "- name:")
	if nextStepIndex != -1 {
		assigneeStepContent = assigneeStepContent[:len("Assign issue to copilot")+nextStepIndex]
	}

	// Verify that gh issue edit is NOT used for copilot within the assignee step
	if strings.Contains(assigneeStepContent, "gh issue edit") {
		t.Error("Should not use gh issue edit for copilot assignee")
	}

	// Verify that actions/github-script is NOT used for copilot within the assignee step
	if strings.Contains(assigneeStepContent, "actions/github-script") {
		t.Error("Should not use actions/github-script for copilot assignee")
	}

	// Verify GH_TOKEN uses Copilot token precedence within the assignee step
	if !strings.Contains(assigneeStepContent, "GH_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN || secrets.GH_AW_GITHUB_TOKEN }}") {
		t.Error("Expected GH_TOKEN to use Copilot token precedence without GITHUB_TOKEN fallback")
	}

	// Verify GITHUB_TOKEN is NOT in the fallback chain for copilot assignees within the assignee step
	if strings.Contains(assigneeStepContent, "|| secrets.GITHUB_TOKEN }}") {
		t.Errorf("Did not expect GITHUB_TOKEN in fallback chain for copilot assignees. Content: %s", assigneeStepContent)
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
