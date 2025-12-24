package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestPullRequestPatchGeneration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "patch-generation-test")

	// Test case with create-pull-request configuration
	testContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
safe-outputs:
  create-pull-request:
    title-prefix: "[test] "
---

# Test Pull Request Patch Generation

This workflow tests how patches are generated automatically.
`

	testFile := filepath.Join(tmpDir, "test-pr-patch.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
// Use release mode to test with inline JavaScript (no local action checkouts)
compiler.SetActionMode(ActionModeRelease)

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// NOTE: Patch generation has been moved to the safe-outputs MCP server
	// The patch is now generated when create_pull_request or push_to_pull_request_branch
	// tools are called within the MCP server, not as a separate workflow step.

	// Check that the dedicated "Generate git patch" step is NOT in the main job anymore
	if strings.Contains(lockStr, "Generate git patch") {
		t.Error("Did not expect 'Generate git patch' step in main job (now handled by MCP server)")
	}

	// Check that patch application still happens in the create_pull_request job
	if !strings.Contains(lockStr, "git am /tmp/gh-aw/aw.patch") {
		t.Error("Expected 'git am /tmp/gh-aw/aw.patch' command in create_pull_request job")
	}

	// Check that it pushes to origin branch in the create_pull_request job
	if !strings.Contains(lockStr, "git push origin") {
		t.Error("Expected 'git push origin' command in create_pull_request job")
	}

	// Check that the create_pull_request job expects the patch file
	if !strings.Contains(lockStr, "No patch file found") {
		t.Error("Expected create_pull_request job to check for patch file existence")
	}

	// Verify the workflow has both main job and create_pull_request job
	if !strings.Contains(lockStr, "safe_outputs:") {
		t.Error("Expected create_pull_request job to be generated")
	}

	t.Logf("Successfully verified patch generation workflow (patch now generated in MCP server)")
}
