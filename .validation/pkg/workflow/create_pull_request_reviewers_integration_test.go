//go:build integration

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
  issues: read
  pull-requests: read
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
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	compiledBytes, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledContent := string(compiledBytes)

	// Verify safe_outputs job exists with handler manager step
	if !strings.Contains(compiledContent, "safe_outputs:") {
		t.Error("Expected safe_outputs job in compiled workflow")
	}
	if !strings.Contains(compiledContent, "id: process_safe_outputs") {
		t.Error("Expected handler manager (process_safe_outputs) step in compiled workflow")
	}

	// Verify actions/github-script is used
	if !strings.Contains(compiledContent, "actions/github-script") {
		t.Error("Expected actions/github-script for PR creation")
	}

	// Verify reviewers are mentioned in the workflow
	if !strings.Contains(compiledContent, "user1") || !strings.Contains(compiledContent, "user2") {
		t.Error("Expected reviewers to be referenced in compiled workflow")
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
  issues: read
  pull-requests: read
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
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	outputFile := filepath.Join(tmpDir, "test-single.lock.yml")
	compiledBytes, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledContent := string(compiledBytes)

	// Verify safe_outputs job exists with handler manager step
	if !strings.Contains(compiledContent, "safe_outputs:") {
		t.Error("Expected safe_outputs job in compiled workflow")
	}
	if !strings.Contains(compiledContent, "id: process_safe_outputs") {
		t.Error("Expected handler manager (process_safe_outputs) step in compiled workflow")
	}

	// Verify reviewer is mentioned somewhere in the workflow
	if !strings.Contains(compiledContent, "single-reviewer") {
		t.Error("Expected single-reviewer reference in compiled workflow")
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
  issues: read
  pull-requests: read
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
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
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

// TestCreatePullRequestWorkflowCompilationWithAssignees tests end-to-end workflow compilation with assignees
func TestCreatePullRequestWorkflowCompilationWithAssignees(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "assignees-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	workflowContent := `---
on: push
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
safe-outputs:
  create-pull-request:
    title-prefix: "[ci] "
    reviewers: [user1]
    assignees: [user1, user2]
    protected-files: fallback-to-issue
---

# Test Workflow

Create a pull request with assignees for fallback issues.
`

	workflowPath := filepath.Join(tmpDir, "test-assignees.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "test-assignees.lock.yml")
	compiledBytes, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledContent := string(compiledBytes)

	if !strings.Contains(compiledContent, "safe_outputs:") {
		t.Error("Expected safe_outputs job in compiled workflow")
	}
	// Verify assignees are present in the handler config
	if !strings.Contains(compiledContent, "user1") || !strings.Contains(compiledContent, "user2") {
		t.Error("Expected assignees to be referenced in compiled workflow")
	}
	if !strings.Contains(compiledContent, `"assignees"`) {
		t.Error("Expected assignees key in compiled safe outputs config")
	}
}

// TestCreatePullRequestWorkflowCompilationWithSingleStringAssignee tests workflow with a single string assignee
func TestCreatePullRequestWorkflowCompilationWithSingleStringAssignee(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "single-assignee-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	workflowContent := `---
on: push
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
safe-outputs:
  create-pull-request:
    assignees: single-assignee
---

# Test Workflow

Create a pull request with a single string assignee.
`

	workflowPath := filepath.Join(tmpDir, "test-single-assignee.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "test-single-assignee.lock.yml")
	compiledBytes, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read compiled output: %v", err)
	}

	compiledContent := string(compiledBytes)

	if !strings.Contains(compiledContent, "safe_outputs:") {
		t.Error("Expected safe_outputs job in compiled workflow")
	}
	if !strings.Contains(compiledContent, "single-assignee") {
		t.Error("Expected single-assignee reference in compiled workflow")
	}
}
