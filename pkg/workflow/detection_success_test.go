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

// TestDetectionJobHasSuccessOutput verifies that the detection job has detection success/conclusion outputs
func TestDetectionJobHasSuccessOutput(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")

	frontmatter := `---
on: workflow_dispatch
permissions:
  contents: read
engine: claude
safe-outputs:
  create-issue:
---

# Test

Create an issue.
`

	if err := os.WriteFile(workflowPath, []byte(frontmatter), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Read the compiled YAML
	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	yamlBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read compiled YAML: %v", err)
	}
	yaml := string(yamlBytes)

	// Detection is now in a separate detection job
	detectionSection := extractJobSection(yaml, "detection")
	if detectionSection == "" {
		t.Fatal("Detection job not found in compiled YAML")
	}

	// Check that detection job outputs include detection_success and detection_conclusion
	if !strings.Contains(yaml, "detection_success:") {
		t.Error("Detection job missing detection_success output")
	}
	if !strings.Contains(yaml, "detection_conclusion:") {
		t.Error("Detection job missing detection_conclusion output")
	}

	// Check that the combined parse-and-conclude step has ID detection_conclusion
	if !strings.Contains(detectionSection, "id: detection_conclusion") {
		t.Error("Combined parse-and-conclude step missing id: detection_conclusion")
	}

	// Check that the script uses require to load the parse_threat_detection_results.cjs file
	if !strings.Contains(detectionSection, "require('${{ runner.temp }}/gh-aw/actions/parse_threat_detection_results.cjs')") {
		t.Error("Detection conclusion step doesn't use require to load parse_threat_detection_results.cjs")
	}

	// Check that setupGlobals is called
	if !strings.Contains(yaml, "setupGlobals(core, github, context, exec, io, getOctokit)") {
		t.Error("Detection conclusion step doesn't call setupGlobals")
	}

	// Check that main() is awaited
	if !strings.Contains(yaml, "await main()") {
		t.Error("Detection conclusion step doesn't await main()")
	}

	// Verify there is no separate parse_detection_results step (it is now merged into detection_conclusion)
	if strings.Contains(detectionSection, "id: parse_detection_results") {
		t.Error("Separate parse_detection_results step should no longer exist; logic is consolidated in detection_conclusion")
	}
}

// TestSafeOutputJobsCheckDetectionSuccess verifies that safe output jobs check detection success
func TestSafeOutputJobsCheckDetectionSuccess(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")

	frontmatter := `---
on: workflow_dispatch
permissions:
  contents: read
engine: claude
safe-outputs:
  create-issue:
  add-comment:
---

# Test

Create outputs.
`

	if err := os.WriteFile(workflowPath, []byte(frontmatter), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Read the compiled YAML
	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	yamlBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read compiled YAML: %v", err)
	}
	yaml := string(yamlBytes)

	// Check that safe_outputs job has detection success check in its condition
	if !strings.Contains(yaml, "safe_outputs:") {
		t.Fatal("safe_outputs job not found")
	}

	// Detection is now in a separate detection job - check uses detection job result
	// (detection job fails with exit 1 when threats are found, so downstream jobs check job result)
	if !strings.Contains(yaml, "needs.detection.result == 'success'") {
		t.Error("Safe output jobs don't check detection result via detection job result")
	}
}
