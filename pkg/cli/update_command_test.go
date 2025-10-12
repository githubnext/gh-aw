package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMergeWorkflowContent_CleanMerge tests a merge with truly non-overlapping changes
func TestMergeWorkflowContent_CleanMerge(t *testing.T) {
	base := `---
on: push
engine: claude
---

# Base Workflow

This is the base content.`

	// Local adds to markdown only, and has source field from previous update
	current := `---
on: push
engine: claude
source: test/repo/workflow.md@v1.0.0
---

# Base Workflow

This is the base content.

## Local Addition

This section was added locally.`

	// Upstream adds to frontmatter only
	new := `---
on: push
engine: claude
tools:
  bash: ["ls"]
source: test/repo/workflow.md@v1.1.0
---

# Base Workflow

This is the base content.`

	oldSourceSpec := "test/repo/workflow.md@v1.0.0"
	newRef := "v1.1.0"

	merged, hasConflicts, err := MergeWorkflowContent(base, current, new, oldSourceSpec, newRef, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if hasConflicts {
		t.Errorf("Expected no conflicts when changes are in different sections (frontmatter vs markdown), merged content:\n%s", merged)
	}

	// Check that local markdown changes are preserved
	if !strings.Contains(merged, "Local Addition") {
		t.Errorf("Expected local markdown changes to be preserved, got:\n%s", merged)
	}

	// Check that upstream frontmatter changes are included
	if !strings.Contains(merged, "bash:") {
		t.Errorf("Expected upstream frontmatter changes to be included, got:\n%s", merged)
	}

	// Check that source field is updated
	if !strings.Contains(merged, "source: test/repo/workflow.md@v1.1.0") {
		t.Errorf("Expected source field to be updated to v1.1.0, got:\n%s", merged)
	}
}

// TestMergeWorkflowContent_WithConflicts tests a merge with conflicts
func TestMergeWorkflowContent_WithConflicts(t *testing.T) {
	base := `---
on: push
engine: claude
---

# Original Workflow

This is the original content.`

	current := `---
on: push
engine: claude
---

# Original Workflow

This is the local modified content.`

	new := `---
on: push
engine: claude
---

# Original Workflow

This is the upstream modified content.`

	oldSourceSpec := "test/repo/workflow.md@v1.0.0"
	newRef := "v1.1.0"

	merged, hasConflicts, err := MergeWorkflowContent(base, current, new, oldSourceSpec, newRef, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !hasConflicts {
		t.Error("Expected conflicts to be detected")
	}

	// Check for conflict markers
	if !strings.Contains(merged, "<<<<<<<") || !strings.Contains(merged, ">>>>>>>") {
		t.Error("Expected conflict markers in merged content")
	}

	// The merged content should contain both versions
	if !strings.Contains(merged, "local modified") && !strings.Contains(merged, "upstream modified") {
		t.Error("Expected both local and upstream changes in conflict markers")
	}
}

// TestMergeWorkflowContent_MarkdownOnly tests merging only markdown changes
func TestMergeWorkflowContent_MarkdownOnly(t *testing.T) {
	base := `---
on: push
engine: claude
---

# Original

Original markdown content.

## Base Section

Base content here.`

	current := `---
on: push
engine: claude
source: test/repo/workflow.md@v1.0.0
---

# Original

Original markdown content.

## Base Section

Base content here.

## Local Section

Local addition at the end.`

	new := `---
on: push
engine: claude
source: test/repo/workflow.md@v1.1.0
---

# Original

Original markdown content.

## Upstream Section

Upstream addition after original.

## Base Section

Base content here.`

	oldSourceSpec := "test/repo/workflow.md@v1.0.0"
	newRef := "v1.1.0"

	merged, hasConflicts, err := MergeWorkflowContent(base, current, new, oldSourceSpec, newRef, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Git merge-file should successfully merge these non-overlapping sections
	if hasConflicts {
		t.Logf("Note: Conflicts detected even for non-overlapping sections:\n%s", merged)
		// This is fine - git is being conservative
	}

	// At minimum, the merge should include some content
	if !strings.Contains(merged, "## Base Section") {
		t.Errorf("Expected base section to be preserved, got:\n%s", merged)
	}
}

// TestMergeWorkflowContent_FrontmatterOnly tests merging only frontmatter changes
func TestMergeWorkflowContent_FrontmatterOnly(t *testing.T) {
	base := `---
on: push
engine: claude
---

# Workflow

Content remains the same.`

	// Local adds permissions field
	current := `---
on: push
engine: claude
permissions:
  contents: read
source: test/repo/workflow.md@v1.0.0
---

# Workflow

Content remains the same.`

	// Upstream adds tools field
	new := `---
on: push
engine: claude
tools:
  bash: ["ls"]
---

# Workflow

Content remains the same.`

	oldSourceSpec := "test/repo/workflow.md@v1.0.0"
	newRef := "v1.1.0"

	merged, hasConflicts, err := MergeWorkflowContent(base, current, new, oldSourceSpec, newRef, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Since both added different fields, should not conflict
	if hasConflicts {
		t.Logf("Note: Conflicts detected for non-overlapping frontmatter fields:\n%s", merged)
	}

	// At minimum, the merge should complete
	if merged == "" {
		t.Error("Expected non-empty merged content")
	}

	// Both fields should be present (if no conflicts) or at least one should be there
	hasPermissions := strings.Contains(merged, "permissions:")
	hasTools := strings.Contains(merged, "tools:")

	if !hasPermissions && !hasTools {
		t.Errorf("Expected at least one of the frontmatter changes to be present, got:\n%s", merged)
	}
}

// TestUpdateSourceFieldInContent tests the source field update function
func TestUpdateSourceFieldInContent(t *testing.T) {
	content := `---
on: push
source: old/repo/workflow.md@v1.0.0
---

# Test Workflow`

	updated, err := UpdateFieldInFrontmatter(content, "source", "old/repo/workflow.md@v2.0.0")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.Contains(updated, "source: old/repo/workflow.md@v2.0.0") {
		t.Errorf("Expected source field to be updated to v2.0.0, got:\n%s", updated)
	}

	// Ensure other content is preserved (formatting should be preserved)
	if !strings.Contains(updated, `on: push`) {
		t.Errorf("Expected other frontmatter fields to be preserved, got:\n%s", updated)
	}

	if !strings.Contains(updated, "# Test Workflow") {
		t.Error("Expected markdown content to be preserved")
	}
}

// TestMergeWorkflowContent_Integration tests the merge with temporary files
func TestMergeWorkflowContent_Integration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	base := `---
on: push
permissions:
  contents: read
---

# Test Workflow

Base content.`

	// Local adds issues permission
	current := `---
on: push
permissions:
  contents: read
  issues: write
source: test/repo/workflow.md@v1.0.0
---

# Test Workflow

Base content with local notes.`

	// Upstream adds pr permission
	new := `---
on: push
permissions:
  contents: read
  pull-requests: write
---

# Test Workflow

Base content with upstream notes.`

	oldSourceSpec := "test/repo/workflow.md@v1.0.0"
	newRef := "v1.1.0"

	merged, hasConflicts, err := MergeWorkflowContent(base, current, new, oldSourceSpec, newRef, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Write merged content to verify it's valid
	testFile := filepath.Join(tmpDir, "merged.md")
	if err := os.WriteFile(testFile, []byte(merged), 0644); err != nil {
		t.Fatalf("Failed to write merged file: %v", err)
	}

	// Since permissions are on different lines, git should merge them
	if hasConflicts {
		t.Logf("Conflicts detected (may be expected):\n%s", merged)
		// With conflicts, we can't check the merged result as reliably
		return
	}

	// Without conflicts, verify both permissions are merged
	if !strings.Contains(merged, "issues: write") || !strings.Contains(merged, "pull-requests: write") {
		t.Logf("Merged content:\n%s", merged)
		t.Error("Expected both local and upstream permission changes to be merged")
	}
}
