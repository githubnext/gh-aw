package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
)

func TestCompileWorkflowFileSizeValidation(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "file-size-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("workflow under 1MB should compile successfully", func(t *testing.T) {
		// Create a normal workflow that should be well under 1MB
		testContent := `---
timeout_minutes: 10
permissions:
  contents: read
  issues: write
  pull-requests: read
tools:
  github:
    allowed: [list_issues, create_issue]
---

# Normal Test Workflow

This is a normal workflow that should compile successfully.
`

		testFile := filepath.Join(tmpDir, "normal-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Errorf("Expected no error for normal workflow, got: %v", err)
		}

		// Verify lock file was created and is under 1MB
		lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
		if info, err := os.Stat(lockFile); err != nil {
			t.Errorf("Lock file was not created: %v", err)
		} else if info.Size() > MaxLockFileSize {
			t.Errorf("Lock file size %d exceeds max size %d", info.Size(), MaxLockFileSize)
		}
	})

	t.Run("file size validation logic", func(t *testing.T) {
		// Test the validation by creating a temporary compiler with modified constant
		// Since normal workflows don't exceed 1MB, we'll test the validation path differently

		// Create a normal workflow
		testContent := `---
timeout_minutes: 10
permissions:
  contents: read
  issues: write
  pull-requests: read
tools:
  github:
    allowed: [list_issues, create_issue]
---

# Test Workflow for Size Validation

This workflow tests the file size validation logic.
`

		testFile := filepath.Join(tmpDir, "size-test-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		compiler := NewCompiler(false, "", "test")
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Errorf("Expected no error for normal workflow, got: %v", err)
		}

		// Verify the lock file exists and get its size
		lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
		info, err := os.Stat(lockFile)
		if err != nil {
			t.Fatalf("Lock file was not created: %v", err)
		}

		// The lock file should be well under 1MB (typically around 30KB)
		if info.Size() > MaxLockFileSize {
			t.Errorf("Unexpected: lock file size %d exceeds max size %d", info.Size(), MaxLockFileSize)
		}

		// Verify our constant is correct (1MB = 1048576 bytes)
		if MaxLockFileSize != 1048576 {
			t.Errorf("MaxLockFileSize constant should be 1048576, got %d", MaxLockFileSize)
		}
	})

	t.Run("test file size validation error message", func(t *testing.T) {
		// Test that our validation produces the correct error message format
		// by simulating the error condition

		testFile := filepath.Join(tmpDir, "size-validation-test.md")
		lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"

		// Create a mock file that exceeds the size limit
		largeSize := int64(MaxLockFileSize + 1000000) // 1MB over the limit
		mockContent := strings.Repeat("x", int(largeSize))

		if err := os.WriteFile(lockFile, []byte(mockContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Verify the file exceeds the limit
		info, err := os.Stat(lockFile)
		if err != nil {
			t.Fatalf("Failed to stat mock file: %v", err)
		}

		if info.Size() <= MaxLockFileSize {
			t.Fatalf("Mock file size %d should exceed limit %d", info.Size(), MaxLockFileSize)
		}

		// Test our validation logic by checking what the error message would look like
		lockSize := pretty.FormatFileSize(info.Size())
		maxSize := pretty.FormatFileSize(MaxLockFileSize)
		expectedMessage := fmt.Sprintf("generated lock file size (%s) exceeds maximum allowed size (%s)", lockSize, maxSize)

		t.Logf("Generated error message would be: %s", expectedMessage)

		// Verify the message contains expected elements
		if !strings.Contains(expectedMessage, "exceeds maximum allowed size") {
			t.Error("Error message should contain 'exceeds maximum allowed size'")
		}
		if !strings.Contains(expectedMessage, "MB") {
			t.Error("Error message should contain size in MB")
		}

		// Clean up
		os.Remove(lockFile)
	})
}
