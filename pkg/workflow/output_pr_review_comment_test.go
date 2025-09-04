package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPRReviewCommentConfigParsing(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "output-pr-review-comment-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("basic PR review comment configuration", func(t *testing.T) {
		// Test case with basic create-pull-request-review-comment configuration
		testContent := `---
on: pull_request
permissions:
  contents: read
  pull-requests: write
engine: claude
safe-outputs:
  create-pull-request-review-comment:
---

# Test PR Review Comment Configuration

This workflow tests the create-pull-request-review-comment configuration parsing.
`

		testFile := filepath.Join(tmpDir, "test-pr-review-comment-basic.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")

		// Parse the workflow data
		workflowData, err := compiler.parseWorkflowFile(testFile)
		if err != nil {
			t.Fatalf("Unexpected error parsing workflow with PR review comment config: %v", err)
		}

		// Verify output configuration is parsed correctly
		if workflowData.SafeOutputs == nil {
			t.Fatal("Expected safe-outputs configuration to be parsed")
		}

		if workflowData.SafeOutputs.CreatePullRequestReviewComments == nil {
			t.Fatal("Expected create-pull-request-review-comment configuration to be parsed")
		}

		// Check default values
		config := workflowData.SafeOutputs.CreatePullRequestReviewComments
		if config.Max != 10 {
			t.Errorf("Expected default max to be 10, got %d", config.Max)
		}

		if config.Side != "RIGHT" {
			t.Errorf("Expected default side to be RIGHT, got %s", config.Side)
		}
	})

	t.Run("PR review comment configuration with custom values", func(t *testing.T) {
		// Test case with custom PR review comment configuration
		testContent := `---
on: pull_request
engine: claude
safe-outputs:
  create-pull-request-review-comment:
    max: 5
    side: "LEFT"
---

# Test PR Review Comment Configuration with Custom Values

This workflow tests custom configuration values.
`

		testFile := filepath.Join(tmpDir, "test-pr-review-comment-custom.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")

		// Parse the workflow data
		workflowData, err := compiler.parseWorkflowFile(testFile)
		if err != nil {
			t.Fatalf("Unexpected error parsing workflow with custom PR review comment config: %v", err)
		}

		// Verify custom configuration values
		if workflowData.SafeOutputs == nil || workflowData.SafeOutputs.CreatePullRequestReviewComments == nil {
			t.Fatal("Expected create-pull-request-review-comment configuration to be parsed")
		}

		config := workflowData.SafeOutputs.CreatePullRequestReviewComments
		if config.Max != 5 {
			t.Errorf("Expected max to be 5, got %d", config.Max)
		}

		if config.Side != "LEFT" {
			t.Errorf("Expected side to be LEFT, got %s", config.Side)
		}
	})

	t.Run("PR review comment configuration with null value", func(t *testing.T) {
		// Test case with null PR review comment configuration
		testContent := `---
on: pull_request
engine: claude
safe-outputs:
  create-pull-request-review-comment: null
---

# Test PR Review Comment Configuration with Null

This workflow tests null configuration.
`

		testFile := filepath.Join(tmpDir, "test-pr-review-comment-null.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")

		// Parse the workflow data
		workflowData, err := compiler.parseWorkflowFile(testFile)
		if err != nil {
			t.Fatalf("Unexpected error parsing workflow with null PR review comment config: %v", err)
		}

		// Verify null configuration is handled correctly (should create default config)
		if workflowData.SafeOutputs == nil || workflowData.SafeOutputs.CreatePullRequestReviewComments == nil {
			t.Fatal("Expected create-pull-request-review-comment configuration to be parsed even with null value")
		}

		config := workflowData.SafeOutputs.CreatePullRequestReviewComments
		if config.Max != 10 {
			t.Errorf("Expected default max to be 10 for null config, got %d", config.Max)
		}

		if config.Side != "RIGHT" {
			t.Errorf("Expected default side to be RIGHT for null config, got %s", config.Side)
		}
	})

	t.Run("PR review comment configuration rejects invalid side values", func(t *testing.T) {
		// Test case with invalid side value (should be rejected by schema validation)
		testContent := `---
on: pull_request
engine: claude
safe-outputs:
  create-pull-request-review-comment:
    max: 2
    side: "INVALID_SIDE"
---

# Test PR Review Comment Configuration with Invalid Side

This workflow tests invalid side value handling.
`

		testFile := filepath.Join(tmpDir, "test-pr-review-comment-invalid-side.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")

		// Parse the workflow data - this should fail due to schema validation
		_, err := compiler.parseWorkflowFile(testFile)
		if err == nil {
			t.Fatal("Expected error parsing workflow with invalid side value, but got none")
		}

		// Verify error message mentions the invalid side value
		if !strings.Contains(err.Error(), "value must be one of 'LEFT', 'RIGHT'") {
			t.Errorf("Expected error message to mention valid side values, got: %v", err)
		}
	})
}

func TestPRReviewCommentJobGeneration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "pr-review-comment-job-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("generate PR review comment job", func(t *testing.T) {
		testContent := `---
on: pull_request
engine: claude
safe-outputs:
  create-pull-request-review-comment:
    max: 3
    side: "LEFT"
---

# Test PR Review Comment Job Generation

This workflow tests job generation for PR review comments.
`

		testFile := filepath.Join(tmpDir, "test-pr-review-comment-job.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		// Check that the output file exists
		outputFile := filepath.Join(tmpDir, "test-pr-review-comment-job.lock.yml")
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Fatal("Expected output file to be created")
		}

		// Read the output content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		workflowContent := string(content)

		// Verify the PR review comment job is generated
		if !strings.Contains(workflowContent, "create_pr_review_comment:") {
			t.Error("Expected create_pr_review_comment job to be generated")
		}

		// Verify job condition is correct for PR context
		if !strings.Contains(workflowContent, "if: github.event.pull_request.number") {
			t.Error("Expected job condition to check for pull request context")
		}

		// Verify correct permissions are set
		if !strings.Contains(workflowContent, "pull-requests: write") {
			t.Error("Expected pull-requests: write permission to be set")
		}

		// Verify environment variables are passed
		if !strings.Contains(workflowContent, "GITHUB_AW_AGENT_OUTPUT:") {
			t.Error("Expected GITHUB_AW_AGENT_OUTPUT environment variable to be passed")
		}

		if !strings.Contains(workflowContent, `GITHUB_AW_PR_REVIEW_COMMENT_SIDE: "LEFT"`) {
			t.Error("Expected GITHUB_AW_PR_REVIEW_COMMENT_SIDE environment variable to be set to LEFT")
		}

		// Verify the JavaScript script is embedded
		if !strings.Contains(workflowContent, "create-pull-request-review-comment") {
			t.Error("Expected PR review comment script to be embedded")
		}
	})
}
