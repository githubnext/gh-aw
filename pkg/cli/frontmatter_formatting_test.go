package cli

import (
	"strings"
	"testing"
)

// TestFormattingPreservation tests that frontmatter operations preserve comments, blank lines, and formatting
func TestFormattingPreservation(t *testing.T) {
	originalContent := `---
on:
    workflow_dispatch:
    # This is a standalone comment
    schedule:
        # Run daily at 2am UTC
        - cron: "0 2 * * 1-5"
    stop-after: +48h # inline comment

timeout_minutes: 30

permissions: read-all

engine: claude
---

# Test Workflow

This is test content.`

	t.Run("RemoveFieldFromOnTrigger preserves formatting", func(t *testing.T) {
		result, err := RemoveFieldFromOnTrigger(originalContent, "stop-after")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check that comments are preserved
		if !strings.Contains(result, "# This is a standalone comment") {
			t.Error("Standalone comment was not preserved")
		}
		if !strings.Contains(result, "# Run daily at 2am UTC") {
			t.Error("Comment in schedule block was not preserved")
		}

		// Check that blank lines are preserved
		if !strings.Contains(result, "\n\n") {
			t.Error("Blank lines were not preserved")
		}

		// Check that indentation is preserved
		if !strings.Contains(result, "    workflow_dispatch:") {
			t.Error("Indentation was not preserved for workflow_dispatch")
		}
		if !strings.Contains(result, "        - cron:") {
			t.Error("Indentation was not preserved for cron expression")
		}

		// Check that cron expression is still quoted
		if !strings.Contains(result, `"0 2 * * 1-5"`) {
			t.Error("Cron expression was unquoted")
		}

		// Check that field was removed
		if strings.Contains(result, "stop-after:") {
			t.Error("Field was not removed")
		}
	})

	t.Run("SetFieldInOnTrigger preserves formatting", func(t *testing.T) {
		result, err := SetFieldInOnTrigger(originalContent, "stop-after", "+72h")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check that comments are preserved
		if !strings.Contains(result, "# This is a standalone comment") {
			t.Error("Standalone comment was not preserved")
		}
		if !strings.Contains(result, "# Run daily at 2am UTC") {
			t.Error("Comment in schedule block was not preserved")
		}
		if !strings.Contains(result, "# inline comment") {
			t.Error("Inline comment was not preserved")
		}

		// Check that blank lines are preserved
		if !strings.Contains(result, "\n\n") {
			t.Error("Blank lines were not preserved")
		}

		// Check that indentation is preserved
		if !strings.Contains(result, "    workflow_dispatch:") {
			t.Error("Indentation was not preserved for workflow_dispatch")
		}

		// Check that cron expression is still quoted
		if !strings.Contains(result, `"0 2 * * 1-5"`) {
			t.Error("Cron expression was unquoted")
		}

		// Check that field was updated with new value
		if !strings.Contains(result, "stop-after: +72h") {
			t.Error("Field was not updated with new value")
		}
	})

	t.Run("UpdateFieldInFrontmatter preserves formatting", func(t *testing.T) {
		result, err := UpdateFieldInFrontmatter(originalContent, "source", "test/repo@v1.0.0")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check that all comments are preserved
		if !strings.Contains(result, "# This is a standalone comment") {
			t.Error("Standalone comment was not preserved")
		}
		if !strings.Contains(result, "# Run daily at 2am UTC") {
			t.Error("Comment in schedule block was not preserved")
		}
		if !strings.Contains(result, "# inline comment") {
			t.Error("Inline comment was not preserved")
		}

		// Check that blank lines are preserved
		if !strings.Contains(result, "\n\n") {
			t.Error("Blank lines were not preserved")
		}

		// Check that indentation is preserved
		if !strings.Contains(result, "    workflow_dispatch:") {
			t.Error("Indentation was not preserved for workflow_dispatch")
		}

		// Check that new field was added
		if !strings.Contains(result, "source: test/repo@v1.0.0") {
			t.Error("Source field was not added")
		}
	})
}

// TestRemoveFieldFromOnTriggerEdgeCases tests edge cases for field removal
func TestRemoveFieldFromOnTriggerEdgeCases(t *testing.T) {
	t.Run("remove field that doesn't exist", func(t *testing.T) {
		content := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
---

# Test`
		result, err := RemoveFieldFromOnTrigger(content, "stop-after")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		// Content should be unchanged
		if result != content {
			t.Error("Content was modified when field didn't exist")
		}
	})

	t.Run("remove field from workflow without on block", func(t *testing.T) {
		content := `---
permissions:
  contents: read
---

# Test`
		result, err := RemoveFieldFromOnTrigger(content, "stop-after")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		// Content should be unchanged
		if result != content {
			t.Error("Content was modified when on block didn't exist")
		}
	})
}
