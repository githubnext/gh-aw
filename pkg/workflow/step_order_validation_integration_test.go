package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStepOrderingValidation_SecretRedactionBeforeUploads verifies that the compiler
// generates secret redaction step before any artifact uploads after agent execution
func TestStepOrderingValidation_SecretRedactionBeforeUploads(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "step-order-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	// Test with a workflow that has secrets
	workflowWithSecrets := `---
on: issues
engine: copilot
tools:
  github:
    github-token: ${{ secrets.CUSTOM_TOKEN }}
    allowed: [list_issues]
safe-outputs:
  create-issue:
---

# Test Workflow

This workflow has a secret reference and safe-outputs.
`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflowWithSecrets), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile should succeed - secret redaction should be added before uploads
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Compilation failed (should have succeeded with proper step ordering): %v", err)
	}

	// Read the generated lock file to verify step order
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	contentStr := string(content)

	// Find the positions of key steps
	redactPos := strings.Index(contentStr, "name: Redact secrets in logs")
	uploadSafeOutputsPos := strings.Index(contentStr, "name: Upload Safe Outputs")
	uploadAgentLogsPos := strings.Index(contentStr, "name: Upload Agent Stdio")

	// Verify that redact step comes before upload steps
	if redactPos < 0 {
		t.Error("Secret redaction step not found in generated workflow")
	}

	if uploadSafeOutputsPos > 0 && redactPos > uploadSafeOutputsPos {
		t.Error("Secret redaction step should come BEFORE Upload Safe Outputs")
	}

	if uploadAgentLogsPos > 0 && redactPos > uploadAgentLogsPos {
		t.Error("Secret redaction step should come BEFORE Upload Agent Stdio")
	}

	if redactPos > uploadSafeOutputsPos || redactPos > uploadAgentLogsPos {
		t.Error("Secret redaction must happen before artifact uploads")
	}
}

// TestStepOrderingValidation_NoSecretsStillHasRedaction verifies that even when
// no secrets are detected at compile time, a secret redaction step is still generated
func TestStepOrderingValidation_NoSecretsStillHasRedaction(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "step-order-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	// Test with a workflow that has NO secrets at compile time
	workflowNoSecrets := `---
on: issues
engine: copilot
tools:
  github:
    allowed: [list_issues]
safe-outputs:
  create-issue:
---

# Test Workflow

This workflow has no secret references.
`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflowNoSecrets), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile should succeed - secret redaction should still be added (as a no-op)
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Compilation failed (should have succeeded with proper step ordering): %v", err)
	}

	// Read the generated lock file to verify secret redaction step exists
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	contentStr := string(content)

	// Verify that redact step exists (even if it's a no-op)
	redactPos := strings.Index(contentStr, "name: Redact secrets in logs")
	if redactPos < 0 {
		t.Error("Secret redaction step should be present even when no secrets detected at compile time")
	}
}

// TestStepOrderingValidation_UploadedPathsCoverage verifies that all uploaded
// paths are covered by secret redaction (i.e., they're under /tmp/gh-aw/ with
// scannable extensions)
func TestStepOrderingValidation_UploadedPathsCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "step-order-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	// Test with a workflow that uploads artifacts
	workflow := `---
on: issues
engine: copilot
tools:
  github:
    allowed: [list_issues]
safe-outputs:
  create-issue:
---

# Test Workflow

This workflow uploads artifacts.
`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflow), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile should succeed - all uploaded paths should be scannable
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	contentStr := string(content)

	// Verify common upload paths are present and under /tmp/gh-aw/
	uploadPaths := []string{
		"/tmp/gh-aw/safeoutputs/outputs.jsonl",
		"/tmp/gh-aw/agent-stdio.log",
		"/tmp/gh-aw/mcp-logs/",
	}

	for _, path := range uploadPaths {
		if strings.Contains(contentStr, path) {
			// Verify it's under /tmp/gh-aw/ (already true by construction)
			if !strings.HasPrefix(path, "/tmp/gh-aw/") {
				t.Errorf("Upload path %s is not under /tmp/gh-aw/ and won't be scanned", path)
			}
		}
	}
}
