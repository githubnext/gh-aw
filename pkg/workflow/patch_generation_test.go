package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPullRequestPatchGeneration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "patch-generation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

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
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Check that git patch generation step is included in main job
	if !strings.Contains(lockStr, "Generate git patch") {
		t.Error("Expected 'Generate git patch' step in workflow")
	}

	// Check that it uses git am to apply patches (newer approach)
	if !strings.Contains(lockStr, "git am /tmp/gh-aw/aw.patch") {
		t.Error("Expected 'git am /tmp/gh-aw/aw.patch' command in git patch step")
	}

	// Check that it pushes to origin branch
	if !strings.Contains(lockStr, "git push origin") {
		t.Error("Expected 'git push origin' command in git patch step")
	}

	// Check that it generates patch from format-patch
	if !strings.Contains(lockStr, "git format-patch") {
		t.Error("Expected 'git format-patch' command in git patch step")
	}

	// Check that the create_pull_request job expects the patch file
	if !strings.Contains(lockStr, "No patch file found") {
		t.Error("Expected pull request job to check for patch file existence")
	}

	// Verify the workflow has both main job and pull request job
	if !strings.Contains(lockStr, "create_pull_request:") {
		t.Error("Expected create_pull_request job to be generated")
	}

	t.Logf("Successfully verified patch generation workflow")
}
