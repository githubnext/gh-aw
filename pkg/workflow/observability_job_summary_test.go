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

// TestCompileWorkflow_MasksOTLPHeadersWhenConfigured verifies that the compiled
// workflow includes a ::add-mask:: step for OTEL_EXPORTER_OTLP_HEADERS in all
// relevant jobs when headers are configured.
func TestCompileWorkflow_MasksOTLPHeadersWhenConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "otlp-with-headers.md")
	content := `---
on: push
permissions:
  contents: read
observability:
  otlp:
    endpoint: https://traces.example.com:4317
    headers: "Authorization=Bearer supersecrettoken"
engine: copilot
---

# Test OTLP Headers Masking
`

	if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Unexpected compile error: %v", err)
	}

	lockPath := filepath.Join(tmpDir, "otlp-with-headers.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	compiled := string(lockContent)

	// The ::add-mask:: step must appear in the compiled YAML
	if !strings.Contains(compiled, "- name: Mask OTLP telemetry headers") {
		t.Fatal("Expected OTLP headers masking step to be generated when headers are configured")
	}
	if !strings.Contains(compiled, "::add-mask::") {
		t.Fatal("Expected ::add-mask:: command for OTEL_EXPORTER_OTLP_HEADERS")
	}
	if !strings.Contains(compiled, "$OTEL_EXPORTER_OTLP_HEADERS") {
		t.Fatal("Expected OTEL_EXPORTER_OTLP_HEADERS env var reference in masking step")
	}

	// The masking step must appear in both the activation job and the agent job.
	// Count occurrences: each job that runs has its own instance of the masking step.
	maskCount := strings.Count(compiled, "- name: Mask OTLP telemetry headers")
	if maskCount < 2 {
		t.Fatalf("Expected masking step in at least 2 jobs (activation + agent), found %d", maskCount)
	}
}

// TestCompileWorkflow_DoesNotMaskOTLPHeadersWhenNotConfigured verifies that no
// masking step is emitted when OTLP headers are not configured.
func TestCompileWorkflow_DoesNotMaskOTLPHeadersWhenNotConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "otlp-no-headers.md")
	content := `---
on: push
permissions:
  contents: read
observability:
  otlp:
    endpoint: https://traces.example.com:4317
engine: copilot
---

# Test No OTLP Headers Masking
`

	if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Unexpected compile error: %v", err)
	}

	lockPath := filepath.Join(tmpDir, "otlp-no-headers.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	compiled := string(lockContent)
	if strings.Contains(compiled, "- name: Mask OTLP telemetry headers") {
		t.Fatal("Did not expect OTLP headers masking step when headers are not configured")
	}
	if strings.Contains(compiled, "Mask OTLP") {
		t.Fatal("Did not expect any OTLP masking when headers are not configured")
	}
}

// TestCompileWorkflow_MasksOTLPHeadersBeforeCheckout verifies that the masking
// step appears before the checkout step in the agent job, so the header value is
// masked as early as possible.
func TestCompileWorkflow_MasksOTLPHeadersBeforeCheckout(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "otlp-mask-order.md")
	content := `---
on: push
permissions:
  contents: read
observability:
  otlp:
    endpoint: https://traces.example.com:4317
    headers: "Authorization=Bearer supersecrettoken"
engine: copilot
---

# Test OTLP Headers Masking Order
`

	if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Unexpected compile error: %v", err)
	}

	lockPath := filepath.Join(tmpDir, "otlp-mask-order.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	compiled := string(lockContent)

	if !strings.Contains(compiled, "- name: Mask OTLP telemetry headers") {
		t.Fatal("Expected OTLP headers masking step")
	}

	// Find the first checkout step in the agent job section (after the activation job)
	agentJobIdx := strings.Index(compiled, "agent:")
	if agentJobIdx < 0 {
		t.Fatal("Expected agent job section")
	}

	checkoutIdxInAgent := strings.Index(compiled[agentJobIdx:], "- name: Checkout repository")
	if checkoutIdxInAgent < 0 {
		t.Skip("No checkout step found in agent job, skipping order check")
	}
	checkoutIdx := agentJobIdx + checkoutIdxInAgent

	// Find the mask step in the agent job section
	maskIdxInAgent := strings.Index(compiled[agentJobIdx:], "- name: Mask OTLP telemetry headers")
	if maskIdxInAgent < 0 {
		t.Fatal("Expected OTLP headers masking step in agent job")
	}
	maskAbsIdx := agentJobIdx + maskIdxInAgent

	if maskAbsIdx >= checkoutIdx {
		t.Fatal("OTLP headers masking step should appear before checkout step in agent job")
	}
}
