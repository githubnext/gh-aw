package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestAddCommentWithDiscussionSupport(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "add-comment-discussion-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with add-comment configuration and issues trigger
	// Note: We test with issues trigger since discussion events may not be fully supported yet
	// but the add-comment job should now include discussion support in its conditional
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
  discussions: write
engine: claude
safe-outputs:
  add-comment:
---

# Test Add Comment with Discussion

This workflow tests add-comment support for discussions.
`

	// Write test workflow file
	testFile := tmpDir + "/test-workflow.md"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create compiler and compile
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read generated lock file
	lockFilePath := tmpDir + "/test-workflow.lock.yml"
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify add_comment job includes discussion permission
	if !strings.Contains(lockContentStr, "discussions: write") {
		t.Error("Expected discussions: write permission in add_comment job")
	}

	// Verify job has conditional execution including discussion.number
	expectedConditionParts := []string{
		"github.event.issue.number",
		"github.event.pull_request.number",
		"github.event.discussion.number",
	}
	for _, part := range expectedConditionParts {
		if !strings.Contains(lockContentStr, part) {
			t.Errorf("Expected add_comment job condition to include '%s'", part)
		}
	}
}
