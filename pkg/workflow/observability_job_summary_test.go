//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileWorkflow_IncludesObservabilitySummaryStepWhenOTLPEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "observability-summary.md")
	content := `---
on: push
permissions:
  contents: read
observability:
  otlp:
    endpoint: https://traces.example.com:4317
engine: copilot
---

# Test Observability Summary with OTLP
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
		t.Fatal("Expected observability summary step to be generated when OTLP is enabled")
	}
	if !strings.Contains(compiled, "require('${{ runner.temp }}/gh-aw/actions/generate_observability_summary.cjs')") {
		t.Fatal("Expected generated workflow to load generate_observability_summary.cjs")
	}
}

func TestCompileWorkflow_DoesNotIncludeObservabilitySummaryStepWithoutOTLP(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "no-observability-summary.md")
	content := `---
on: push
permissions:
  contents: read
engine: copilot
---

# Test No Observability Summary
`

	if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Unexpected compile error: %v", err)
	}

	lockPath := filepath.Join(tmpDir, "no-observability-summary.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	compiled := string(lockContent)
	if strings.Contains(compiled, "- name: Generate observability summary") {
		t.Fatal("Did not expect observability summary step when OTLP is not configured")
	}
	if strings.Contains(compiled, "GH_AW_OBSERVABILITY_JOB_SUMMARY") {
		t.Fatal("Did not expect GH_AW_OBSERVABILITY_JOB_SUMMARY env var in compiled workflow")
	}
}

func TestCompileWorkflow_IncludesObservabilitySummaryStepWhenOTLPEnabledViaImport(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an imported workflow with OTLP configured
	importedPath := filepath.Join(tmpDir, "shared-otlp.md")
	importedContent := `---
observability:
  otlp:
    endpoint: https://traces.example.com:4317
---
`
	if err := os.WriteFile(importedPath, []byte(importedContent), 0o644); err != nil {
		t.Fatalf("Failed to write imported workflow: %v", err)
	}

	// Main workflow imports the shared OTLP config but has no observability section itself
	workflowPath := filepath.Join(tmpDir, "main-import-otlp.md")
	content := `---
on: push
permissions:
  contents: read
engine: copilot
imports:
  - ./shared-otlp.md
---

# Test Observability Summary via Import
`
	if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write main workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Unexpected compile error: %v", err)
	}

	lockPath := filepath.Join(tmpDir, "main-import-otlp.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	compiled := string(lockContent)
	if !strings.Contains(compiled, "- name: Generate observability summary") {
		t.Fatal("Expected observability summary step when OTLP is enabled via import")
	}
	if !strings.Contains(compiled, "OTEL_EXPORTER_OTLP_ENDPOINT") {
		t.Fatal("Expected OTEL_EXPORTER_OTLP_ENDPOINT env var to be injected when OTLP is configured via import")
	}
}
