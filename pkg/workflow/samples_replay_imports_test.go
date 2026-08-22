//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestFeaturesSamplesFromImportEnablesReplay verifies that `features: { samples: true }`
// declared only in an imported shared workflow still activates samples replay for the
// importing workflow: WorkflowData.UseSamples must be true, threat detection must be
// force-disabled, and the compiled lock file must contain the deterministic replay step
// instead of invoking the agent.
func TestFeaturesSamplesFromImportEnablesReplay(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	sharedPath := filepath.Join(sharedDir, "samples-config.md")
	sharedContent := `---
features:
  samples: true
---

# Shared Samples Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	mainPath := filepath.Join(tmpDir, ".github", "workflows", "main.md")
	mainContent := `---
on: workflow_dispatch
permissions: read-all
engine:
  id: claude
imports:
  - shared/samples-config.md
safe-outputs:
  create-issue:
    samples:
      - title: "Deterministic test issue"
        body: "Issue body emitted by gh-aw samples replay."
---

# Main Workflow

Test that features.samples: true from an import enables samples replay.
`
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main workflow file: %v", err)
	}

	compiler := NewCompiler()

	workflowData, err := compiler.ParseWorkflowFile(mainPath)
	if err != nil {
		t.Fatalf("ParseWorkflowFile failed: %v", err)
	}
	if !workflowData.UseSamples {
		t.Fatal("Expected workflowData.UseSamples to be true from imported features.samples: true")
	}
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.ThreatDetection != nil {
		t.Fatal("Expected threat-detection to be force-disabled when samples replay is enabled via imports")
	}

	if err := compiler.CompileWorkflow(mainPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(mainPath)
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockStr := string(lockContent)

	if !strings.Contains(lockStr, "Replay safe-outputs samples (deterministic)") {
		t.Error("Expected `Replay safe-outputs samples (deterministic)` step in lock file")
	}
	if !strings.Contains(lockStr, "apply_samples.cjs") {
		t.Error("Expected lock file to invoke apply_samples.cjs driver")
	}
	if strings.Contains(lockStr, "\n  detection:\n") {
		t.Error("Expected no `detection:` job when samples replay is enabled via imports")
	}
}
