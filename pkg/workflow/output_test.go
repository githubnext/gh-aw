package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputConfigParsing(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "output-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with output.issue configuration
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
engine: claude
output:
  issue:
    title-prefix: "[genai] "
    labels: [copilot, automation]
---

# Test Output Configuration

This workflow tests the output configuration parsing.
`

	testFile := filepath.Join(tmpDir, "test-output-config.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow data
	workflowData, err := compiler.parseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Unexpected error parsing workflow with output config: %v", err)
	}

	// Verify output configuration is parsed correctly
	if workflowData.Output == nil {
		t.Fatal("Expected output configuration to be parsed")
	}

	if workflowData.Output.Issue == nil {
		t.Fatal("Expected issue configuration to be parsed")
	}

	// Verify title prefix
	expectedPrefix := "[genai] "
	if workflowData.Output.Issue.TitlePrefix != expectedPrefix {
		t.Errorf("Expected title prefix '%s', got '%s'", expectedPrefix, workflowData.Output.Issue.TitlePrefix)
	}

	// Verify labels
	expectedLabels := []string{"copilot", "automation"}
	if len(workflowData.Output.Issue.Labels) != len(expectedLabels) {
		t.Errorf("Expected %d labels, got %d", len(expectedLabels), len(workflowData.Output.Issue.Labels))
	}

	for i, expectedLabel := range expectedLabels {
		if i >= len(workflowData.Output.Issue.Labels) || workflowData.Output.Issue.Labels[i] != expectedLabel {
			t.Errorf("Expected label '%s' at index %d, got '%s'", expectedLabel, i, workflowData.Output.Issue.Labels[i])
		}
	}
}

func TestOutputConfigEmpty(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "output-config-empty-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case without output configuration
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
engine: claude
---

# Test No Output Configuration

This workflow has no output configuration.
`

	testFile := filepath.Join(tmpDir, "test-no-output.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow data
	workflowData, err := compiler.parseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Unexpected error parsing workflow without output config: %v", err)
	}

	// Verify output configuration is nil
	if workflowData.Output != nil {
		t.Error("Expected output configuration to be nil when not specified")
	}
}

func TestOutputIssueJobGeneration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "output-issue-job-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with output.issue configuration
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine: claude
output:
  issue:
    title-prefix: "[genai] "
    labels: [copilot]
---

# Test Output Issue Job Generation

This workflow tests the create_output_issue job generation.
`

	testFile := filepath.Join(tmpDir, "test-output-issue.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with output issue: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-output-issue.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify create_output_issue job exists
	if !strings.Contains(lockContent, "create_output_issue:") {
		t.Error("Expected 'create_output_issue' job to be in generated workflow")
	}

	// Verify job properties
	if !strings.Contains(lockContent, "timeout-minutes: 10") {
		t.Error("Expected 10-minute timeout in create_output_issue job")
	}

	if !strings.Contains(lockContent, "permissions:\n      contents: read\n      issues: write") {
		t.Error("Expected correct permissions in create_output_issue job")
	}

	// Verify the job uses github-script
	if !strings.Contains(lockContent, "uses: actions/github-script@v7") {
		t.Error("Expected github-script action to be used in create_output_issue job")
	}

	// Verify JavaScript content includes environment variables for configuration
	if !strings.Contains(lockContent, "GITHUB_AW_ISSUE_TITLE_PREFIX: \"[genai] \"") {
		t.Error("Expected title prefix to be set as environment variable")
	}

	if !strings.Contains(lockContent, "GITHUB_AW_ISSUE_LABELS: \"copilot\"") {
		t.Error("Expected copilot label to be set as environment variable")
	}

	// Verify job dependencies
	if !strings.Contains(lockContent, "needs: test-output-issue") {
		t.Error("Expected create_output_issue job to depend on main job")
	}

	t.Logf("Generated workflow content:\n%s", lockContent)
}

func TestOutputCommentConfigParsing(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "output-comment-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with output.comment configuration
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
  pull-requests: write
engine: claude
output:
  comment: {}
---

# Test Output Comment Configuration

This workflow tests the output.comment configuration parsing.
`

	testFile := filepath.Join(tmpDir, "test-output-comment.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow data
	workflowData, err := compiler.parseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Unexpected error parsing workflow with output comment config: %v", err)
	}

	// Verify output configuration is parsed correctly
	if workflowData.Output == nil {
		t.Fatal("Expected output configuration to be parsed")
	}

	if workflowData.Output.Comment == nil {
		t.Fatal("Expected comment configuration to be parsed")
	}
}

func TestOutputCommentJobGeneration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "output-comment-job-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with output.comment configuration
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
  pull-requests: write
tools:
  github:
    allowed: [get_issue]
engine: claude
output:
  comment: {}
---

# Test Output Comment Job Generation

This workflow tests the create_issue_comment job generation.
`

	testFile := filepath.Join(tmpDir, "test-output-comment.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with output comment: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-output-comment.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify create_issue_comment job exists
	if !strings.Contains(lockContent, "create_issue_comment:") {
		t.Error("Expected 'create_issue_comment' job to be in generated workflow")
	}

	// Verify job properties
	if !strings.Contains(lockContent, "timeout-minutes: 10") {
		t.Error("Expected 10-minute timeout in create_issue_comment job")
	}

	if !strings.Contains(lockContent, "permissions:\n      contents: read\n      issues: write\n      pull-requests: write") {
		t.Error("Expected correct permissions in create_issue_comment job")
	}

	// Verify the job uses github-script
	if !strings.Contains(lockContent, "uses: actions/github-script@v7") {
		t.Error("Expected github-script action to be used in create_issue_comment job")
	}

	// Verify job has conditional execution
	if !strings.Contains(lockContent, "if: github.event.issue.number || github.event.pull_request.number") {
		t.Error("Expected create_issue_comment job to have conditional execution")
	}

	// Verify job dependencies
	if !strings.Contains(lockContent, "needs: test-output-comment") {
		t.Error("Expected create_issue_comment job to depend on main job")
	}

	// Verify JavaScript content includes environment variable for agent output
	if !strings.Contains(lockContent, "AGENT_OUTPUT_CONTENT:") {
		t.Error("Expected agent output content to be passed as environment variable")
	}

	t.Logf("Generated workflow content:\n%s", lockContent)
}

func TestOutputCommentJobSkippedForNonIssueEvents(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "output-comment-skip-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with output.comment configuration but push trigger (not issue/PR)
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
  pull-requests: write
engine: claude
output:
  comment: {}
---

# Test Output Comment Job Skipping

This workflow tests that comment job is skipped for non-issue/PR events.
`

	testFile := filepath.Join(tmpDir, "test-comment-skip.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with output comment: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-comment-skip.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify create_issue_comment job exists (it should be generated regardless of trigger)
	if !strings.Contains(lockContent, "create_issue_comment:") {
		t.Error("Expected 'create_issue_comment' job to be in generated workflow")
	}

	// Verify job has conditional execution to skip when not in issue/PR context
	if !strings.Contains(lockContent, "if: github.event.issue.number || github.event.pull_request.number") {
		t.Error("Expected create_issue_comment job to have conditional execution for skipping")
	}

	t.Logf("Generated workflow content:\n%s", lockContent)
}
