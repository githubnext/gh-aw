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

// TestMCPPolicyErrorDetectionStep tests that a Copilot engine workflow includes
// the detect-mcp-policy-error step in the agent job.
func TestMCPPolicyErrorDetectionStep(t *testing.T) {
	testDir := testutil.TempDir(t, "test-mcp-policy-error-*")
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

	// Check that agent job has detect-mcp-policy-error step
	if !strings.Contains(lockStr, "id: detect-mcp-policy-error") {
		t.Error("Expected agent job to have detect-mcp-policy-error step")
	}

	// Check that the detection step calls the shell script
	if !strings.Contains(lockStr, "detect_mcp_policy_error.sh") {
		t.Error("Expected detect-mcp-policy-error step to call detect_mcp_policy_error.sh")
	}

	// Check that the agent job exposes mcp_policy_error output
	if !strings.Contains(lockStr, "mcp_policy_error: ${{ steps.detect-mcp-policy-error.outputs.mcp_policy_error || 'false' }}") {
		t.Error("Expected agent job to have mcp_policy_error output")
	}
}

// TestMCPPolicyErrorInConclusionJob tests that the conclusion job receives the MCP policy error
// env var when the Copilot engine is used.
func TestMCPPolicyErrorInConclusionJob(t *testing.T) {
	testDir := testutil.TempDir(t, "test-mcp-policy-error-conclusion-*")
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

	// Check that conclusion job receives MCP policy error from agent job
	if !strings.Contains(lockStr, "GH_AW_MCP_POLICY_ERROR: ${{ needs.agent.outputs.mcp_policy_error }}") {
		t.Error("Expected conclusion job to receive mcp_policy_error from agent job")
	}
}

// TestMCPPolicyErrorNotInNonCopilotEngine tests that non-Copilot engines
// do NOT include the detect-mcp-policy-error step.
func TestMCPPolicyErrorNotInNonCopilotEngine(t *testing.T) {
	testDir := testutil.TempDir(t, "test-mcp-policy-error-claude-*")
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

	// Check that non-Copilot engines do NOT have the detect-mcp-policy-error step
	if strings.Contains(lockStr, "id: detect-mcp-policy-error") {
		t.Error("Expected non-Copilot engine to NOT have detect-mcp-policy-error step")
	}

	// Check that non-Copilot engines do NOT have the mcp_policy_error output
	if strings.Contains(lockStr, "mcp_policy_error:") {
		t.Error("Expected non-Copilot engine to NOT have mcp_policy_error output")
	}
}
