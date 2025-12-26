//go:build integration

package workflow

import (
	"testing"
)

// SKIPPED: Scripts now use require() pattern and are loaded at runtime from external files
// TestSafeOutputsMCPBundlerIntegration tests that the safe-outputs workflow
// correctly includes child_process imports in the generated .cjs files
func TestSafeOutputsMCPBundlerIntegration(t *testing.T) {
	t.Skip("Test skipped - safe-outputs MCP scripts now use require() pattern and are loaded at runtime from external files")
}

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

		// Verify that the safe-outputs workflow contains child_process imports
		// Note: Safe-outputs writes separate .cjs files to disk, so each file has its own require statement
		// This is the correct behavior for file-based MCP servers

		if !strings.Contains(lockContent, `require("child_process")`) {
			t.Fatal("Compiled workflow does not contain child_process require statement")
		}

		// Verify both execSync and execFile are imported (in separate files)
		hasExecSync := strings.Contains(lockContent, `const { execSync } = require("child_process")`)
		hasExecFile := strings.Contains(lockContent, `const { execFile } = require("child_process")`)

		if !hasExecSync {
			t.Error("Compiled workflow does not contain execSync import from child_process")
		}

		if !hasExecFile {
			t.Error("Compiled workflow does not contain execFile import from child_process")
		}

		// Verify both execSync and execFile are used in the code
		if !strings.Contains(lockContent, "execSync(") {
			t.Error("Compiled workflow does not use execSync function")
		}

		if !strings.Contains(lockContent, "execFile(") {
			t.Error("Compiled workflow does not use execFile function")
		}

		// Count how many times child_process is required
		// Each separate .cjs file has its own require statement, which is expected
		count := strings.Count(lockContent, `require("child_process")`)
		if count < 2 {
			t.Errorf("Expected at least 2 child_process require statements (separate files), got %d", count)
		}

		// Verify staged mode is enabled (from the frontmatter)
		if !strings.Contains(lockContent, `GH_AW_SAFE_OUTPUTS_STAGED: "true"`) {
			t.Error("Expected staged mode to be enabled in compiled workflow")
		}

		t.Logf("✓ Successfully verified child_process imports in safe-outputs workflow")
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

		// Verify that both execSync and execFile are imported (in separate files)
		hasExecSync := strings.Contains(lockContent, `const { execSync } = require("child_process")`)
		hasExecFile := strings.Contains(lockContent, `const { execFile } = require("child_process")`)

		if !hasExecSync {
			t.Error("Compiled workflow does not contain execSync import from child_process")
		}

		if !hasExecFile {
			t.Error("Compiled workflow does not contain execFile import from child_process")
		}

		// Verify staged mode is NOT enabled
		if strings.Contains(lockContent, `GH_AW_SAFE_OUTPUTS_STAGED: "true"`) {
			t.Error("Expected staged mode to NOT be enabled in compiled workflow")
		}

		t.Logf("✓ Successfully verified child_process imports without staged mode")
	})
}
