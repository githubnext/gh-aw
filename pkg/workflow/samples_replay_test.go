//go:build integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

// TestUseSamplesReplacesAgentStep verifies that compiling with
// SetUseSamples(true) replaces the engine `Execute coding agent` step
// with the deterministic `Replay safe-outputs samples` step driven by
// apply_samples.cjs.
func TestUseSamplesReplacesAgentStep(t *testing.T) {
	const md = `---
on:
  workflow_dispatch:
permissions: read-all
engine:
  id: claude
safe-outputs:
  create-issue:
    samples:
      - title: "Deterministic test issue"
        body: "Issue body emitted by gh-aw samples replay."
---

Trivial workflow whose only job is to be compiled with --use-samples.
`

	tmpFile, err := os.CreateTemp("", "use-samples-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(md); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	t.Run("Default Mode", func(t *testing.T) {
		compiler := NewCompiler()
		if err := compiler.CompileWorkflow(tmpFile.Name()); err != nil {
			t.Fatalf("compile failed: %v", err)
		}
		lockPath := strings.TrimSuffix(tmpFile.Name(), ".md") + ".lock.yml"
		defer os.Remove(lockPath)
		b, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("read lock: %v", err)
		}
		lockContent := string(b)
		if strings.Contains(lockContent, "Replay safe-outputs samples") {
			t.Error("Did not expect samples replay step in default mode")
		}
		if strings.Contains(lockContent, "apply_samples.cjs") {
			t.Error("Did not expect apply_samples driver in default mode")
		}
	})

	t.Run("Use Samples Mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.SetUseSamples(true)
		if err := compiler.CompileWorkflow(tmpFile.Name()); err != nil {
			t.Fatalf("compile failed: %v", err)
		}
		workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseWorkflowFile failed: %v", err)
		}
		if !workflowData.UseSamples {
			t.Fatal("Expected workflowData.UseSamples to be true after SetUseSamples(true)")
		}
		lockPath := strings.TrimSuffix(tmpFile.Name(), ".md") + ".lock.yml"
		defer os.Remove(lockPath)
		b, _ := os.ReadFile(lockPath)
		lockContent := string(b)
		if !strings.Contains(lockContent, "Replay safe-outputs samples (deterministic)") {
			t.Error("Expected `Replay safe-outputs samples (deterministic)` step in lock file")
		}
		if !strings.Contains(lockContent, "apply_samples.cjs") {
			t.Error("Expected lock file to invoke apply_samples.cjs driver")
		}
		if !strings.Contains(lockContent, "GH_AW_SAMPLES:") {
			t.Error("Expected GH_AW_SAMPLES env var in lock file")
		}
		if !strings.Contains(lockContent, `"tool":"create_issue"`) {
			t.Error("Expected JSON-encoded create_issue tool entry in lock file")
		}
		if !strings.Contains(lockContent, "Deterministic test issue") {
			t.Error("Expected sample title in lock file")
		}
		if !strings.Contains(lockContent, "id: agentic_execution") {
			t.Error("Expected id: agentic_execution on the replay step")
		}
		// Threat detection must be force-disabled under --use-samples so the
		// deterministic replay isn't perturbed by an LLM-backed detection job.
		if strings.Contains(lockContent, "\n  detection:\n") {
			t.Error("Expected no `detection:` job under --use-samples")
		}
	})
}
