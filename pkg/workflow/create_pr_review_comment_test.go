package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestPRReviewCommentConfigParsing(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "output-pr-review-comment-test")

	t.Run("basic PR review comment configuration", func(t *testing.T) {
		// Test case with basic create-pull-request-review-comment configuration
		testContent := `---
on: pull_request
permissions:
  contents: read
  pull-requests: write
  issues: read
engine: claude
strict: false
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
		// Use release mode to test with inline JavaScript (no local action checkouts)
		compiler.SetActionMode(ActionModeRelease)

		// Parse the workflow data
		workflowData, err := compiler.ParseWorkflowFile(testFile)
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
strict: false
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
		// Use release mode to test with inline JavaScript (no local action checkouts)
		compiler.SetActionMode(ActionModeRelease)

		// Parse the workflow data
		workflowData, err := compiler.ParseWorkflowFile(testFile)
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
strict: false
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
		// Use release mode to test with inline JavaScript (no local action checkouts)
		compiler.SetActionMode(ActionModeRelease)

		// Parse the workflow data
		workflowData, err := compiler.ParseWorkflowFile(testFile)
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

	t.Run("PR review comment configuration with target", func(t *testing.T) {
		// Test case with target configuration
		testContent := `---
on: pull_request
engine: claude
strict: false
safe-outputs:
  create-pull-request-review-comment:
    max: 5
    target: "*"
---

# Test PR Review Comment Configuration with Target

This workflow tests target configuration.
`

		testFile := filepath.Join(tmpDir, "test-pr-review-comment-target.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		// Use release mode to test with inline JavaScript (no local action checkouts)
		compiler.SetActionMode(ActionModeRelease)

		// Parse the workflow data
		workflowData, err := compiler.ParseWorkflowFile(testFile)
		if err != nil {
			t.Fatalf("Unexpected error parsing workflow with target config: %v", err)
		}

		// Verify target configuration value
		if workflowData.SafeOutputs == nil || workflowData.SafeOutputs.CreatePullRequestReviewComments == nil {
			t.Fatal("Expected create-pull-request-review-comment configuration to be parsed")
		}

		config := workflowData.SafeOutputs.CreatePullRequestReviewComments
		if config.Target != "*" {
			t.Errorf("Expected target to be '*', got %s", config.Target)
		}
	})

	t.Run("PR review comment configuration rejects invalid side values", func(t *testing.T) {
		// Test case with invalid side value (should be rejected by schema validation)
		testContent := `---
on: pull_request
engine: claude
strict: false
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
		// Use release mode to test with inline JavaScript (no local action checkouts)
		compiler.SetActionMode(ActionModeRelease)

		// Parse the workflow data - this should fail due to schema validation
		_, err := compiler.ParseWorkflowFile(testFile)
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
	tmpDir := testutil.TempDir(t, "pr-review-comment-job-test")

	t.Run("generate PR review comment job", func(t *testing.T) {
		testContent := `---
on: pull_request
engine: claude
strict: false
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
		// Use release mode to test with inline JavaScript (no local action checkouts)
		compiler.SetActionMode(ActionModeRelease)

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

		// Verify the safe_outputs job is generated (consolidated job approach)
		if !strings.Contains(workflowContent, "safe_outputs:") {
			t.Error("Expected safe_outputs job to be generated (consolidated approach)")
		}

		// Verify the create_pr_review_comment step is present within the safe_outputs job
		if !strings.Contains(workflowContent, "name: Create PR Review Comment") {
			t.Error("Expected Create PR Review Comment step to be generated")
		}
		if !strings.Contains(workflowContent, "id: create_pr_review_comment") {
			t.Error("Expected create_pr_review_comment step ID")
		}

		// Verify step condition uses BuildSafeOutputType
		if !strings.Contains(workflowContent, "contains(needs.agent.outputs.output_types, 'create_pull_request_review_comment')") {
			t.Error("Expected step condition to contain safe-output type check")
		}

		// Verify correct permissions are set
		if !strings.Contains(workflowContent, "pull-requests: write") {
			t.Error("Expected pull-requests: write permission to be set")
		}

		// Verify environment variables are passed
		if !strings.Contains(workflowContent, "GH_AW_AGENT_OUTPUT:") {
			t.Error("Expected GH_AW_AGENT_OUTPUT environment variable to be passed")
		}

		if !strings.Contains(workflowContent, `GH_AW_PR_REVIEW_COMMENT_SIDE: "LEFT"`) {
			t.Error("Expected GH_AW_PR_REVIEW_COMMENT_SIDE environment variable to be set to LEFT")
		}

		// Verify the JavaScript script is embedded
		if !strings.Contains(workflowContent, "create-pull-request-review-comment") {
			t.Error("Expected PR review comment script to be embedded")
		}
	})

	t.Run("generate PR review comment job with target", func(t *testing.T) {
		testContent := `---
on: pull_request
engine: claude
strict: false
safe-outputs:
  create-pull-request-review-comment:
    max: 3
    target: "*"
---

# Test PR Review Comment Job Generation with Target

This workflow tests job generation for PR review comments with target configuration.
`

		testFile := filepath.Join(tmpDir, "test-pr-review-comment-job-target.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		// Use release mode to test with inline JavaScript (no local action checkouts)
		compiler.SetActionMode(ActionModeRelease)

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected error compiling workflow: %v", err)
		}

		// Check that the output file exists
		outputFile := filepath.Join(tmpDir, "test-pr-review-comment-job-target.lock.yml")
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Fatal("Expected output file to be created")
		}

		// Read the output content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}

		workflowContent := string(content)

		// Verify the safe_outputs job is generated (consolidated job approach)
		if !strings.Contains(workflowContent, "safe_outputs:") {
			t.Error("Expected safe_outputs job to be generated (consolidated approach)")
		}

		// Verify the create_pr_review_comment step is present
		if !strings.Contains(workflowContent, "name: Create PR Review Comment") {
			t.Error("Expected Create PR Review Comment step to be generated")
		}
		if !strings.Contains(workflowContent, "id: create_pr_review_comment") {
			t.Error("Expected create_pr_review_comment step ID")
		}

		// Verify environment variables are passed
		if !strings.Contains(workflowContent, `GH_AW_PR_REVIEW_COMMENT_TARGET: "*"`) {
			t.Error("Expected GH_AW_PR_REVIEW_COMMENT_TARGET environment variable to be set to '*'")
		}

		// Verify the step condition contains the safe-output type check
		if !strings.Contains(workflowContent, "contains(needs.agent.outputs.output_types, 'create_pull_request_review_comment')") {
			t.Error("Expected step condition to contain safe-output type check")
		}
	})
}
