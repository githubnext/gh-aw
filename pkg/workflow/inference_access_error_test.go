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

// TestInferenceAccessErrorDetectionStep tests that a Copilot engine workflow exposes
// inference_access_error from the detect-agent-errors step.
func TestInferenceAccessErrorDetectionStep(t *testing.T) {
	testDir := testutil.TempDir(t, "test-inference-access-error-*")
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

	// Check that agent job has the primary execution step
	if !strings.Contains(lockStr, "id: agentic_execution") {
		t.Error("Expected agent job to have agentic_execution step")
	}

	// Check that a separate detection step is generated on the host runner
	if !strings.Contains(lockStr, "id: detect-agent-errors") {
		t.Error("Expected agent job to have a separate detect-agent-errors step")
	}

	// Check that the agent job exposes inference_access_error output from the detection step
	if !strings.Contains(lockStr, "inference_access_error: ${{ steps.detect-agent-errors.outputs.inference_access_error || 'false' }}") {
		t.Error("Expected agent job to have inference_access_error output from detect-agent-errors step")
	}
}

// TestInferenceAccessErrorInConclusionJob tests that the conclusion job receives the inference access error
// env var when the Copilot engine is used.
func TestInferenceAccessErrorInConclusionJob(t *testing.T) {
	testDir := testutil.TempDir(t, "test-inference-access-error-conclusion-*")
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

	// Check that conclusion job receives inference access error from agent job
	if !strings.Contains(lockStr, "GH_AW_INFERENCE_ACCESS_ERROR: ${{ needs.agent.outputs.inference_access_error }}") {
		t.Error("Expected conclusion job to receive inference_access_error from agent job")
	}
}

// TestInferenceAccessErrorNotInNonCopilotEngine tests that non-Copilot engines
// do NOT include Copilot-specific error outputs.
func TestInferenceAccessErrorNotInNonCopilotEngine(t *testing.T) {
	testDir := testutil.TempDir(t, "test-inference-access-error-claude-*")
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

	// Check that non-Copilot engines do NOT have the detect-agent-errors step
	if strings.Contains(lockStr, "id: detect-agent-errors") {
		t.Error("Expected non-Copilot engine to NOT have detect-agent-errors step")
	}

	// Check that non-Copilot engines do NOT have the inference_access_error output
	if strings.Contains(lockStr, "inference_access_error:") {
		t.Error("Expected non-Copilot engine to NOT have inference_access_error output")
	}
}
