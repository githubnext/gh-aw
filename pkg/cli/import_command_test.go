//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddImportToWorkflow_AddsImportToExistingList(t *testing.T) {
	content := `---
on: issues
engine: copilot
imports:
  - shared/security-notice.md
---

Workflow body.
`
	updated, added, err := addImportToWorkflow(content, "shared/tool-setup.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected import to be added")
	}
	if !strings.Contains(updated, "shared/tool-setup.md") {
		t.Errorf("expected updated content to contain new import, got:\n%s", updated)
	}
	if !strings.Contains(updated, "shared/security-notice.md") {
		t.Errorf("expected updated content to still contain original import, got:\n%s", updated)
	}
}

func TestAddImportToWorkflow_CreatesImportsField(t *testing.T) {
	content := `---
on: issues
engine: copilot
---

Workflow body.
`
	updated, added, err := addImportToWorkflow(content, "shared/security-notice.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected import to be added")
	}
	if !strings.Contains(updated, "shared/security-notice.md") {
		t.Errorf("expected updated content to contain import, got:\n%s", updated)
	}
}

func TestAddImportToWorkflow_AddsImportToObjectForm(t *testing.T) {
	content := `---
on: issues
engine: copilot
imports:
  aw:
    - shared/security-notice.md
---

Workflow body.
`
	updated, added, err := addImportToWorkflow(content, "shared/tool-setup.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected import to be added")
	}
	if !strings.Contains(updated, "aw:\n    - shared/security-notice.md\n    - shared/tool-setup.md") {
		t.Errorf("expected updated content to append import under imports.aw, got:\n%s", updated)
	}
}

func TestAddImportToWorkflow_NoDuplicates(t *testing.T) {
	content := `---
on: issues
engine: copilot
imports:
  - shared/security-notice.md
---

Workflow body.
`
	_, added, err := addImportToWorkflow(content, "shared/security-notice.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added {
		t.Fatal("expected import NOT to be added (duplicate)")
	}
}

func TestAddImportToWorkflow_NoDuplicateWhenPathObjectMatches(t *testing.T) {
	content := `---
on: issues
engine: copilot
imports:
  aw:
    - path: shared/security-notice.md
---

Workflow body.
`
	updated, added, err := addImportToWorkflow(content, "shared/security-notice.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added {
		t.Fatal("expected import NOT to be added when object path already matches")
	}
	if updated != content {
		t.Fatalf("expected duplicate detection to preserve original content\nwant:\n%s\ngot:\n%s", content, updated)
	}
}

func TestAddImportToWorkflow_NoFrontmatter(t *testing.T) {
	content := "Just markdown, no frontmatter.\n"
	_, _, err := addImportToWorkflow(content, "shared/security-notice.md")
	if err == nil {
		t.Fatal("expected error for content without frontmatter")
	}
}

func TestAddImportToWorkflow_RejectsLeadingBlankLineBeforeFrontmatter(t *testing.T) {
	content := `
---
on: issues
---

Workflow body.
`
	_, _, err := addImportToWorkflow(content, "shared/security-notice.md")
	if err == nil {
		t.Fatal("expected error when frontmatter delimiter is not on the first line")
	}
}

func TestAddImportToWorkflow_ReplacesNullImports(t *testing.T) {
	content := `---
on: issues
engine: copilot
imports: ~
---

Workflow body.
`
	updated, added, err := addImportToWorkflow(content, "shared/security-notice.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected import to be added")
	}
	want := `---
on: issues
engine: copilot
imports:
  - shared/security-notice.md
---

Workflow body.
`
	if updated != want {
		t.Fatalf("expected null imports to be replaced with a list\nwant:\n%s\ngot:\n%s", want, updated)
	}
}

func TestAddImportToWorkflow_PreservesUntouchedBytesOutsideImportsBlock(t *testing.T) {
	content := `---
# keep this comment
on: issues
imports: # keep inline comment
  - shared/security-notice.md
# keep this line too
engine: copilot
---

Workflow body.

`
	updated, added, err := addImportToWorkflow(content, "shared/tool-setup.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Fatal("expected import to be added")
	}
	want := `---
# keep this comment
on: issues
imports: # keep inline comment
  - shared/security-notice.md
  - shared/tool-setup.md
# keep this line too
engine: copilot
---

Workflow body.

`
	if updated != want {
		t.Fatalf("expected only imports block to change\nwant:\n%s\ngot:\n%s", want, updated)
	}
}

func TestImportCommandVerboseUsesRootPersistentFlag(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().Bool("verbose", false, "")
	if err := root.PersistentFlags().Set("verbose", "true"); err != nil {
		t.Fatalf("failed to set verbose flag: %v", err)
	}

	cmd := NewImportCommand()
	root.AddCommand(cmd)

	if !importCommandVerbose(cmd) {
		t.Fatal("expected import command to read verbose from the root persistent flags")
	}
}

func TestInlineYAMLCommentRequiresWhitespaceBeforeHash(t *testing.T) {
	if got := inlineYAMLComment(`imports: "foo#bar"`); got != "" {
		t.Fatalf("expected no comment for hash inside quoted value, got %q", got)
	}
	if got := inlineYAMLComment(`imports: "foo#bar" # keep`); got != "# keep" {
		t.Fatalf("expected trailing comment to be preserved, got %q", got)
	}
}

func TestRunImport_WritesFile(t *testing.T) {
	tmp := t.TempDir()
	workflowFile := filepath.Join(tmp, "my-workflow.md")
	content := "---\non: issues\nengine: copilot\n---\n\nBody.\n"
	if err := os.WriteFile(workflowFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	err := RunImport(ImportOptions{
		WorkflowID: workflowFile,
		ImportPath: "shared/security-notice.md",
	})
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	updated, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}
	if !strings.Contains(string(updated), "shared/security-notice.md") {
		t.Errorf("expected file to contain import, got:\n%s", string(updated))
	}
}

func TestRunImport_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	workflowFile := filepath.Join(tmp, "my-workflow.md")
	content := "---\non: issues\nengine: copilot\nimports:\n  - shared/security-notice.md\n---\n\nBody.\n"
	if err := os.WriteFile(workflowFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run twice; second run should be idempotent
	for range 2 {
		if err := RunImport(ImportOptions{
			WorkflowID: workflowFile,
			ImportPath: "shared/security-notice.md",
		}); err != nil {
			t.Fatalf("RunImport failed: %v", err)
		}
	}

	updated, _ := os.ReadFile(workflowFile)
	count := strings.Count(string(updated), "shared/security-notice.md")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of import, got %d:\n%s", count, string(updated))
	}
}
