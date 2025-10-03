package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscussionFeature(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "discussion-test*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("discussion enabled by default", func(t *testing.T) {
		workflowContent := `---
on: push
engine: claude
---

# Test Workflow

This workflow tests default discussion creation.`

		workflowFile := filepath.Join(tmpDir, "default-discussion.md")
		err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		compiler := NewCompiler(false, "", "test")
		err = compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Verify discussion creation step exists
		if !strings.Contains(lockContentStr, "Create discussion to track workflow run") {
			t.Error("Expected discussion creation step in activation job")
		}

		// Verify discussion outputs
		if !strings.Contains(lockContentStr, "discussion-id") {
			t.Error("Expected discussion-id output")
		}
		if !strings.Contains(lockContentStr, "discussion-number") {
			t.Error("Expected discussion-number output")
		}
		if !strings.Contains(lockContentStr, "discussion-url") {
			t.Error("Expected discussion-url output")
		}

		// Verify discussions: write permission
		if !strings.Contains(lockContentStr, "discussions: write") {
			t.Error("Expected discussions: write permission in activation job")
		}

		// Verify environment variables - category should be empty (let JavaScript resolve it)
		if !strings.Contains(lockContentStr, "GITHUB_AW_DISCUSSION_CATEGORY: \"\"") {
			t.Error("Expected empty category (JavaScript will resolve it)")
		}
	})

	t.Run("discussion explicitly disabled", func(t *testing.T) {
		workflowContent := `---
on: push
engine: claude
discussion: false
---

# Test Workflow

This workflow tests disabled discussion creation.`

		workflowFile := filepath.Join(tmpDir, "disabled-discussion.md")
		err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		compiler := NewCompiler(false, "", "test")
		err = compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Verify discussion creation step does NOT exist
		if strings.Contains(lockContentStr, "Create discussion to track workflow run") {
			t.Error("Should not have discussion creation step when disabled")
		}

		// Verify discussions: write permission is NOT present
		if strings.Contains(lockContentStr, "discussions: write") {
			t.Error("Should not have discussions: write permission when disabled")
		}
	})

	t.Run("discussion with custom category", func(t *testing.T) {
		workflowContent := `---
on: push
engine: claude
discussion: "My Custom Category"
---

# Test Workflow

This workflow tests custom discussion category.`

		workflowFile := filepath.Join(tmpDir, "custom-category.md")
		err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		compiler := NewCompiler(false, "", "test")
		err = compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Verify discussion creation step exists
		if !strings.Contains(lockContentStr, "Create discussion to track workflow run") {
			t.Error("Expected discussion creation step in activation job")
		}

		// Verify custom category name
		if !strings.Contains(lockContentStr, "GITHUB_AW_DISCUSSION_CATEGORY: \"My Custom Category\"") {
			t.Error("Expected custom category 'My Custom Category'")
		}

		// Verify discussions: write permission
		if !strings.Contains(lockContentStr, "discussions: write") {
			t.Error("Expected discussions: write permission in activation job")
		}
	})

	t.Run("discussion with null value (enabled with default)", func(t *testing.T) {
		workflowContent := `---
on: push
engine: claude
discussion:
---

# Test Workflow

This workflow tests null discussion value.`

		workflowFile := filepath.Join(tmpDir, "null-discussion.md")
		err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}

		compiler := NewCompiler(false, "", "test")
		err = compiler.CompileWorkflow(workflowFile)
		if err != nil {
			t.Fatalf("Failed to compile workflow: %v", err)
		}

		lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		lockContentStr := string(lockContent)

		// Verify discussion creation step exists
		if !strings.Contains(lockContentStr, "Create discussion to track workflow run") {
			t.Error("Expected discussion creation step when discussion is null")
		}

		// Verify default category - should be empty (let JavaScript resolve it)
		if !strings.Contains(lockContentStr, "GITHUB_AW_DISCUSSION_CATEGORY: \"\"") {
			t.Error("Expected empty category (JavaScript will resolve it)")
		}
	})
}
