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

	// NOTE: Patch generation happens in the agent job (not in MCP server)
	// The patch is generated as a GitHub Actions step where child_process is available,
	// then uploaded as an artifact for safe-output jobs to download.

	// Check that the dedicated "Generate git patch" step IS in the main job
	if !strings.Contains(lockStr, "Generate git patch") {
		t.Error("Expected 'Generate git patch' step in main job")
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
	if !strings.Contains(lockStr, "create_pull_request:") {
		t.Error("Expected create_pull_request job to be generated")
	}

	t.Logf("Successfully verified patch generation workflow (patch generated in agent job)")
}
