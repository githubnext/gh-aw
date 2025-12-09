//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSafeOutputsMCPBundlerIntegration tests that the safe-outputs MCP server
// bundler correctly merges destructured imports from child_process module
func TestSafeOutputsMCPBundlerIntegration(t *testing.T) {
	t.Run("create_pull_request with merged child_process imports in draft mode", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "bundler-integration-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a workflow with safe-outputs create-pull-request in draft/staged mode
		workflowContent := `---
name: Test Bundler Integration
on: issues

safe-outputs:
  staged: true
  create-pull-request: {}
---

Test workflow to verify child_process imports are merged correctly.
`

		workflowFile := filepath.Join(tmpDir, "test-bundler.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		if err := compiler.CompileWorkflow(workflowFile); err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read the compiled workflow
		lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
		lockBytes, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}
		lockContent := string(lockBytes)

		// Verify that the bundled safe-outputs MCP server contains merged child_process imports
		// The bundler should merge:
		// - const { execSync } = require("child_process"); (from get_current_branch.cjs)
		// - const { execFile } = require("child_process"); (from mcp_handler_shell.cjs)
		// Into: const { execFile, execSync } = require("child_process");
		//    or: const { execSync, execFile } = require("child_process");
		// Note: generate_git_patch.cjs is NOT in the MCP server anymore - it's in the agent job

		if !strings.Contains(lockContent, `require("child_process")`) {
			t.Fatal("Compiled workflow does not contain child_process require statement")
		}

		// Check for merged imports (order may vary)
		hasExecFileAndExecSync := strings.Contains(lockContent, `const { execFile, execSync } = require("child_process")`) ||
			strings.Contains(lockContent, `const { execSync, execFile } = require("child_process")`)

		if !hasExecFileAndExecSync {
			t.Error("Compiled workflow does not contain merged child_process imports (execFile and execSync)")

			// Debug: Find what we actually got
			lines := strings.Split(lockContent, "\n")
			for _, line := range lines {
				if strings.Contains(line, `require("child_process")`) {
					t.Logf("Found child_process require: %s", strings.TrimSpace(line))
				}
			}
		}

		// Verify both execSync and execFile are used in the code
		if !strings.Contains(lockContent, "execSync(") {
			t.Error("Compiled workflow does not use execSync function")
		}

		if !strings.Contains(lockContent, "execFile(") {
			t.Error("Compiled workflow does not use execFile function")
		}

		// Count how many times child_process is required
		// Should be exactly 2: once in MCP server, once in agent job patch generation step
		count := strings.Count(lockContent, `require("child_process")`)
		if count != 2 {
			t.Errorf("Expected exactly 2 child_process require statements (MCP server + agent job), got %d", count)
		}

		// Verify staged mode is enabled (from the frontmatter)
		if !strings.Contains(lockContent, `GH_AW_SAFE_OUTPUTS_STAGED: "true"`) {
			t.Error("Expected staged mode to be enabled in compiled workflow")
		}

		t.Logf("✓ Successfully verified merged child_process imports in safe-outputs MCP server")
	})

	t.Run("create_pull_request without staged mode", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "bundler-integration-test-no-draft")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a workflow with safe-outputs create-pull-request without staged mode
		workflowContent := `---
name: Test Bundler Integration No Draft
on: issues

safe-outputs:
  create-pull-request: {}
---

Test workflow to verify child_process imports are merged correctly without draft mode.
`

		workflowFile := filepath.Join(tmpDir, "test-bundler-no-draft.md")
		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		if err := compiler.CompileWorkflow(workflowFile); err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		// Read the compiled workflow
		lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
		lockBytes, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}
		lockContent := string(lockBytes)

		// Verify merged imports still work without staged mode
		hasExecFileAndExecSync := strings.Contains(lockContent, `const { execFile, execSync } = require("child_process")`) ||
			strings.Contains(lockContent, `const { execSync, execFile } = require("child_process")`)

		if !hasExecFileAndExecSync {
			t.Error("Compiled workflow does not contain merged child_process imports (execFile and execSync)")
		}

		// Verify staged mode is NOT enabled
		if strings.Contains(lockContent, `GH_AW_SAFE_OUTPUTS_STAGED: "true"`) {
			t.Error("Expected staged mode to NOT be enabled in compiled workflow")
		}

		t.Logf("✓ Successfully verified merged child_process imports without staged mode")
	})
}
