//go:build !integration

package workflow_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// TestSharedWorkflowWithoutOn tests that a workflow without an 'on' field
// is validated with the main_workflow_schema (with forbidden field checks) and returns a SharedWorkflowError
func TestSharedWorkflowWithoutOn(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-shared-workflow-*")

	// Create a workflow without 'on' field (shared workflow)
	sharedPath := filepath.Join(tempDir, "shared-config.md")
	sharedContent := `---
description: "Shared configuration without on field"
tools:
  playwright:
    version: "v1.41.0"
network:
  allowed:
    - playwright
---

# Shared Configuration

This is a reusable shared workflow component.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Try to parse the workflow - it should return SharedWorkflowError
	compiler := workflow.NewCompiler()
	_, err := compiler.ParseWorkflowFile(sharedPath)

	// Check that we got a SharedWorkflowError
	if err == nil {
		t.Fatal("Expected SharedWorkflowError, got nil")
	}

	var sharedErr *workflow.SharedWorkflowError
	if !errors.As(err, &sharedErr) {
		t.Fatalf("Expected *workflow.SharedWorkflowError, got %T: %v", err, err)
	}

	// Verify the error contains expected information
	errMsg := sharedErr.Error()
	if !strings.Contains(errMsg, "Shared agentic workflow") {
		t.Errorf("Error message should mention 'Shared agentic workflow', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "missing the 'on' field") {
		t.Errorf("Error message should mention missing 'on' field, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Skipping compilation") {
		t.Errorf("Error message should mention skipping compilation, got: %s", errMsg)
	}

	// Verify the path is correct
	if sharedErr.Path != sharedPath {
		t.Errorf("Expected path %s, got %s", sharedPath, sharedErr.Path)
	}
}

// TestSharedWorkflowWithInvalidFields tests that a shared workflow with invalid fields
// still produces a proper validation error
func TestSharedWorkflowWithInvalidFields(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-shared-workflow-invalid-*")

	// Create a shared workflow with invalid fields
	sharedPath := filepath.Join(tempDir, "invalid-shared.md")
	sharedContent := `---
description: "Invalid shared workflow"
invalid_field: "This field should not be allowed"
---

# Invalid Shared Workflow
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Try to parse the workflow - it should return a validation error (not SharedWorkflowError)
	compiler := workflow.NewCompiler()
	_, err := compiler.ParseWorkflowFile(sharedPath)

	// Check that we got an error (validation error, not SharedWorkflowError)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	// It should NOT be a SharedWorkflowError since validation failed
	if errors.As(err, new(*workflow.SharedWorkflowError)) {
		t.Fatal("Should not return SharedWorkflowError when validation fails")
	}

	// The error should mention the invalid field
	errMsg := err.Error()
	if !strings.Contains(errMsg, "invalid_field") && !strings.Contains(errMsg, "Unknown property") {
		t.Errorf("Error message should mention the invalid field, got: %s", errMsg)
	}
}

// TestMainWorkflowWithOn tests that a workflow with an 'on' field
// is validated with the main_workflow_schema
func TestMainWorkflowWithOn(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-main-workflow-*")

	// Create a main workflow with 'on' field
	mainPath := filepath.Join(tempDir, "main-workflow.md")
	mainContent := `---
on: issues
engine: copilot
permissions:
  contents: read
  issues: read
---

# Main Workflow

This is a main workflow with an on trigger.
`
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main workflow file: %v", err)
	}

	// Parse the workflow - it should succeed
	compiler := workflow.NewCompiler()
	workflowData, err := compiler.ParseWorkflowFile(mainPath)

	// Check that we got no error
	if err != nil {
		t.Fatalf("Expected no error for valid main workflow, got: %v", err)
	}

	// Verify we got workflow data back
	if workflowData == nil {
		t.Fatal("Expected workflowData, got nil")
	}

	// Verify the 'on' field was processed
	if workflowData.On == "" {
		t.Error("Expected 'On' field to be populated in WorkflowData")
	}
}

// TestSharedWorkflowWithEngineOnly tests that a workflow with only engine config
// (no 'on' field) is treated as a shared workflow
func TestSharedWorkflowWithEngineOnly(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-shared-engine-*")

	// Create a shared workflow with only engine configuration
	sharedPath := filepath.Join(tempDir, "shared-engine.md")
	sharedContent := `---
engine:
  id: codex
  env:
    MODEL_VERSION: "gpt-4"
steps:
  - name: Codex step
    run: echo "test"
---

# Shared Engine Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Try to parse the workflow - it should return SharedWorkflowError
	compiler := workflow.NewCompiler()
	_, err := compiler.ParseWorkflowFile(sharedPath)

	// Check that we got a SharedWorkflowError
	if err == nil {
		t.Fatal("Expected SharedWorkflowError, got nil")
	}

	if !errors.As(err, new(*workflow.SharedWorkflowError)) {
		t.Fatalf("Expected *workflow.SharedWorkflowError, got %T: %v", err, err)
	}
}

// TestSharedWorkflowWithMCPServers tests that a shared workflow with MCP server config
// (no 'on' field) is treated as a shared workflow
func TestSharedWorkflowWithMCPServers(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-shared-mcp-*")

	// Create a shared workflow with MCP server configuration
	sharedPath := filepath.Join(tempDir, "shared-mcp.md")
	sharedContent := `---
mcp-servers:
  deepwiki:
    url: "https://mcp.deepwiki.com/sse"
    allowed:
      - read_wiki_structure
      - read_wiki_contents
---

# Shared MCP Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Try to parse the workflow - it should return SharedWorkflowError
	compiler := workflow.NewCompiler()
	_, err := compiler.ParseWorkflowFile(sharedPath)

	// Check that we got a SharedWorkflowError
	if err == nil {
		t.Fatal("Expected SharedWorkflowError, got nil")
	}

	if !errors.As(err, new(*workflow.SharedWorkflowError)) {
		t.Fatalf("Expected *workflow.SharedWorkflowError, got %T: %v", err, err)
	}
}

// TestSharedWorkflowWithoutMarkdownContent tests that a shared workflow
// without markdown content (only frontmatter) is allowed
func TestSharedWorkflowWithoutMarkdownContent(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-shared-no-markdown-*")

	// Create a shared workflow with only frontmatter, no markdown content
	sharedPath := filepath.Join(tempDir, "shared-config-only.md")
	sharedContent := `---
mcp-servers:
  deepwiki:
    url: "https://mcp.deepwiki.com/sse"
    allowed:
      - read_wiki_structure
---
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Try to parse the workflow - it should return SharedWorkflowError (not "no markdown content" error)
	compiler := workflow.NewCompiler()
	_, err := compiler.ParseWorkflowFile(sharedPath)

	// Check that we got a SharedWorkflowError
	if err == nil {
		t.Fatal("Expected SharedWorkflowError, got nil")
	}

	var sharedErr *workflow.SharedWorkflowError
	if !errors.As(err, &sharedErr) {
		t.Fatalf("Expected *workflow.SharedWorkflowError, got %T: %v", err, err)
	}

	// Verify it's not a "no markdown content" error
	if strings.Contains(err.Error(), "no markdown content found") {
		t.Error("Should not return 'no markdown content' error for shared workflows")
	}

	// Verify we got the shared workflow info message
	if !strings.Contains(sharedErr.Error(), "Shared agentic workflow") {
		t.Errorf("Expected shared workflow message, got: %s", sharedErr.Error())
	}
}

// TestRedirectOnlyWorkflow tests that a workflow with a redirect field but no 'on' trigger
// is detected as a redirect-only placeholder and returns a RedirectOnlyWorkflowError.
// Regression: `gh aw add githubnext/agentics/daily-repo-status` downloads a file with only
// redirect: and source: fields, which should give a helpful message directing the user to run
// `gh aw update` rather than the confusing "Shared agentic workflow detected" error.
func TestRedirectOnlyWorkflow(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-redirect-only-*")

	// Simulate the content from the agentics repo's daily-repo-status.md
	redirectPath := filepath.Join(tempDir, "daily-repo-status.md")
	redirectContent := `---
redirect: "githubnext/agentics/workflows/repo-status.md@main"
source: githubnext/agentics/workflows/daily-repo-status.md@c7d030cd6d4607b90d9ac3ffc8b24aff4f251632
---
`
	if err := os.WriteFile(redirectPath, []byte(redirectContent), 0644); err != nil {
		t.Fatalf("Failed to write redirect-only workflow file: %v", err)
	}

	// Parse the workflow - it should return RedirectOnlyWorkflowError (not SharedWorkflowError)
	compiler := workflow.NewCompiler()
	_, err := compiler.ParseWorkflowFile(redirectPath)

	if err == nil {
		t.Fatal("Expected RedirectOnlyWorkflowError, got nil")
	}

	// Must be RedirectOnlyWorkflowError, NOT SharedWorkflowError
	var redirectErr *workflow.RedirectOnlyWorkflowError
	if !errors.As(err, &redirectErr) {
		t.Fatalf("Expected *workflow.RedirectOnlyWorkflowError, got %T: %v", err, err)
	}

	if errors.As(err, new(*workflow.SharedWorkflowError)) {
		t.Fatal("Should NOT return SharedWorkflowError for a redirect-only workflow")
	}

	// Verify the error message is helpful and mentions the redirect target
	errMsg := redirectErr.Error()
	if !strings.Contains(errMsg, "Redirect-only workflow") {
		t.Errorf("Error message should mention 'Redirect-only workflow', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "githubnext/agentics/workflows/repo-status.md@main") {
		t.Errorf("Error message should include the redirect target, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "gh aw update") {
		t.Errorf("Error message should suggest 'gh aw update', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Skipping compilation") {
		t.Errorf("Error message should mention skipping compilation, got: %s", errMsg)
	}

	// Verify the path is set correctly
	if redirectErr.Path != redirectPath {
		t.Errorf("Expected path %s, got %s", redirectPath, redirectErr.Path)
	}

	// Verify the redirect target is set correctly
	if redirectErr.Target != "githubnext/agentics/workflows/repo-status.md@main" {
		t.Errorf("Expected target 'githubnext/agentics/workflows/repo-status.md@main', got %q", redirectErr.Target)
	}
}

// TestRedirectOnlyWorkflowWithoutSourceField tests that a redirect-only file with only
// a redirect field (no source field) is also correctly detected.
func TestRedirectOnlyWorkflowWithoutSourceField(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-redirect-no-source-*")

	redirectPath := filepath.Join(tempDir, "moved-workflow.md")
	redirectContent := `---
redirect: "owner/repo/workflows/new-location.md@main"
---
`
	if err := os.WriteFile(redirectPath, []byte(redirectContent), 0644); err != nil {
		t.Fatalf("Failed to write redirect-only workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	_, err := compiler.ParseWorkflowFile(redirectPath)

	if err == nil {
		t.Fatal("Expected RedirectOnlyWorkflowError, got nil")
	}

	var redirectErr *workflow.RedirectOnlyWorkflowError
	if !errors.As(err, &redirectErr) {
		t.Fatalf("Expected *workflow.RedirectOnlyWorkflowError, got %T: %v", err, err)
	}

	if redirectErr.Target != "owner/repo/workflows/new-location.md@main" {
		t.Errorf("Expected redirect target to be set, got %q", redirectErr.Target)
	}
}

// TestWorkflowWithRedirectAndOn tests that a workflow with both redirect and on fields
// is NOT treated as a redirect-only file (it has an 'on' trigger so it's a valid main workflow).
func TestWorkflowWithRedirectAndOn(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-redirect-and-on-*")

	workflowPath := filepath.Join(tempDir, "workflow-with-redirect.md")
	// A workflow with both 'on' and 'redirect' fields (valid: redirect is for update tracking)
	workflowContent := `---
on: issues
redirect: "owner/repo/workflows/new.md@main"
---

# Workflow with redirect

This is a main workflow that also has a redirect for update tracking.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	_, err := compiler.ParseWorkflowFile(workflowPath)

	// Should NOT be a RedirectOnlyWorkflowError or SharedWorkflowError (it has 'on')
	if errors.As(err, new(*workflow.RedirectOnlyWorkflowError)) {
		t.Fatal("Should NOT return RedirectOnlyWorkflowError for a workflow with both redirect and on fields")
	}
	if errors.As(err, new(*workflow.SharedWorkflowError)) {
		t.Fatal("Should NOT return SharedWorkflowError for a workflow with 'on' field")
	}
	// It may return a different error (e.g., if the workflow body is missing context), but not these two
}


func TestMainWorkflowWithoutMarkdownContent(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-main-no-markdown-*")

	// Create a main workflow with 'on' but no markdown content
	mainPath := filepath.Join(tempDir, "main-no-markdown.md")
	mainContent := `---
on: issues
engine: copilot
---
`
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main workflow file: %v", err)
	}

	// Try to parse the workflow - it should fail with "no markdown content" error
	compiler := workflow.NewCompiler()
	_, err := compiler.ParseWorkflowFile(mainPath)

	// Check that we got an error
	if err == nil {
		t.Fatal("Expected error for main workflow without markdown content, got nil")
	}

	// It should be a "no markdown content" error, not SharedWorkflowError
	if errors.As(err, new(*workflow.SharedWorkflowError)) {
		t.Fatal("Should not return SharedWorkflowError for main workflow")
	}

	if !strings.Contains(err.Error(), "no markdown content") {
		t.Errorf("Expected 'no markdown content' error, got: %v", err)
	}
}
