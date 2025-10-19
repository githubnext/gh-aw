package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreatePullRequestWorkflowCompilationWithReviewers tests end-to-end workflow compilation with reviewers
func TestCreatePullRequestWorkflowCompilationWithReviewers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reviewers-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test workflow with reviewers
	workflowContent := `---
on: push
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  create-pull-request:
    title-prefix: "[test] "
    labels: [automation, test]
    reviewers: [user1, user2, copilot]
    draft: false
---

# Test Workflow

Create a pull request with reviewers.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	compiledBytes, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledContent := string(compiledBytes)

	// Verify that reviewer steps are present
	if !strings.Contains(compiledContent, "Add user1 as reviewer") {
		t.Error("Expected reviewer step for user1 in compiled workflow")
	}
	if !strings.Contains(compiledContent, "Add user2 as reviewer") {
		t.Error("Expected reviewer step for user2 in compiled workflow")
	}
	if !strings.Contains(compiledContent, "Add copilot as reviewer") {
		t.Error("Expected reviewer step for copilot in compiled workflow")
	}

	// Verify copilot mapping
	if !strings.Contains(compiledContent, `REVIEWER: "copilot-swe-agent"`) {
		t.Error("Expected copilot to be mapped to copilot-swe-agent")
	}

	// Verify gh pr edit command
	if !strings.Contains(compiledContent, "gh pr edit") {
		t.Error("Expected gh pr edit command in compiled workflow")
	}
	if !strings.Contains(compiledContent, "--add-reviewer") {
		t.Error("Expected --add-reviewer flag in gh pr edit command")
	}

	// Verify checkout step
	if !strings.Contains(compiledContent, "Checkout repository for gh CLI") {
		t.Error("Expected checkout step for gh CLI")
	}
	if !strings.Contains(compiledContent, "uses: actions/checkout@v5") {
		t.Error("Expected checkout to use actions/checkout@v5")
	}

	// Verify conditional execution
	if !strings.Contains(compiledContent, "if: steps.create_pull_request.outputs.pull_request_url != ''") {
		t.Error("Expected conditional execution based on PR URL")
	}

	// Verify PR_URL environment variable
	if !strings.Contains(compiledContent, "PR_URL: ${{ steps.create_pull_request.outputs.pull_request_url }}") {
		t.Error("Expected PR_URL to be set from step output")
	}
}

// TestCreatePullRequestWorkflowCompilationWithSingleStringReviewer tests workflow with single string reviewer
func TestCreatePullRequestWorkflowCompilationWithSingleStringReviewer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "single-reviewer-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test workflow with single string reviewer
	workflowContent := `---
on: push
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  create-pull-request:
    reviewers: single-reviewer
---

# Test Workflow

Create a pull request with a single reviewer.
`

	workflowPath := filepath.Join(tmpDir, "test-single.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-single.lock.yml")
	compiledBytes, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledContent := string(compiledBytes)

	// Verify that reviewer step is present
	if !strings.Contains(compiledContent, "Add single-reviewer as reviewer") {
		t.Error("Expected reviewer step for single-reviewer in compiled workflow")
	}

	// Verify gh pr edit command
	if !strings.Contains(compiledContent, "gh pr edit") {
		t.Error("Expected gh pr edit command in compiled workflow")
	}

	// Verify REVIEWER environment variable
	if !strings.Contains(compiledContent, `REVIEWER: "single-reviewer"`) {
		t.Error("Expected REVIEWER environment variable to be set")
	}
}

// TestCreatePullRequestWorkflowCompilationWithoutReviewers tests workflow without reviewers
func TestCreatePullRequestWorkflowCompilationWithoutReviewers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "no-reviewers-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test workflow without reviewers
	workflowContent := `---
on: push
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  create-pull-request:
    title-prefix: "[test] "
---

# Test Workflow

Create a pull request without reviewers.
`

	workflowPath := filepath.Join(tmpDir, "test-no-reviewers.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-no-reviewers.lock.yml")
	compiledBytes, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledContent := string(compiledBytes)

	// Verify that no reviewer steps are present
	if strings.Contains(compiledContent, "as reviewer") {
		t.Error("Did not expect reviewer steps when no reviewers configured")
	}
	if strings.Contains(compiledContent, "gh pr edit") && strings.Contains(compiledContent, "--add-reviewer") {
		t.Error("Did not expect gh pr edit with --add-reviewer when no reviewers configured")
	}
}
