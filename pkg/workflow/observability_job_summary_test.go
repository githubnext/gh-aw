//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileWorkflow_AlwaysIncludesObservabilitySummaryStep(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "observability-summary.md")
	content := `---
on: push
permissions:
  contents: read
engine: copilot
---

# Test Observability Summary
`

	if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Unexpected compile error: %v", err)
	}

	lockPath := filepath.Join(tmpDir, "observability-summary.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	compiled := string(lockContent)
	if !strings.Contains(compiled, "- name: Generate observability summary") {
		t.Fatal("Expected observability summary step to always be generated")
	}
	if !strings.Contains(compiled, "require('${{ runner.temp }}/gh-aw/actions/generate_observability_summary.cjs')") {
		t.Fatal("Expected generated workflow to load generate_observability_summary.cjs")
	}
}
