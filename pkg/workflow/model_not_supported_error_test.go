//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestModelNotSupportedErrorDetectionStep tests that a Copilot engine workflow includes
// the detect-copilot-errors step with model_not_supported_error output in the agent job.
func TestModelNotSupportedErrorDetectionStep(t *testing.T) {
	testDir := testutil.TempDir(t, "test-model-not-supported-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: copilot
---

Test workflow`

	if err := os.WriteFile(workflowFile, []byte(workflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Check that agent job has detect-copilot-errors step
	if !strings.Contains(lockStr, "id: detect-copilot-errors") {
		t.Error("Expected agent job to have detect-copilot-errors step")
	}

	// Check that the agent job exposes model_not_supported_error output
	if !strings.Contains(lockStr, "model_not_supported_error: ${{ steps.detect-copilot-errors.outputs.model_not_supported_error || 'false' }}") {
		t.Error("Expected agent job to have model_not_supported_error output")
	}
}

// TestModelNotSupportedErrorInConclusionJob tests that the conclusion job receives the
// model-not-supported error env var when the Copilot engine is used.
func TestModelNotSupportedErrorInConclusionJob(t *testing.T) {
	testDir := testutil.TempDir(t, "test-model-not-supported-conclusion-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: copilot
safe-outputs:
  add-comment:
    max: 5
---

Test workflow`

	if err := os.WriteFile(workflowFile, []byte(workflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Check that conclusion job receives model not supported error from agent job
	if !strings.Contains(lockStr, "GH_AW_MODEL_NOT_SUPPORTED_ERROR: ${{ needs.agent.outputs.model_not_supported_error }}") {
		t.Error("Expected conclusion job to receive model_not_supported_error from agent job")
	}
}

// TestModelNotSupportedErrorNotInNonCopilotEngine tests that non-Copilot engines
// do NOT include the model_not_supported_error output.
func TestModelNotSupportedErrorNotInNonCopilotEngine(t *testing.T) {
	testDir := testutil.TempDir(t, "test-model-not-supported-claude-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: claude
---

Test workflow`

	if err := os.WriteFile(workflowFile, []byte(workflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Check that non-Copilot engines do NOT have the model_not_supported_error output
	if strings.Contains(lockStr, "model_not_supported_error:") {
		t.Error("Expected non-Copilot engine to NOT have model_not_supported_error output")
	}
}
