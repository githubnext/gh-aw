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

// TestDetectionJobHasSuccessOutput verifies that the agent job has detection success/conclusion outputs
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

	// Detection is now inline in the agent job - no separate detection job
	agentSection := extractJobSection(yaml, "agent")
	if agentSection == "" {
		t.Fatal("Agent job not found in compiled YAML")
	}

	// Check that agent job outputs include detection_success and detection_conclusion
	if !strings.Contains(yaml, "detection_success:") {
		t.Error("Agent job missing detection_success output")
	}
	if !strings.Contains(yaml, "detection_conclusion:") {
		t.Error("Agent job missing detection_conclusion output")
	}

	// Check that parse_detection_results step has an ID
	if !strings.Contains(agentSection, "id: parse_detection_results") {
		t.Error("Parse detection results step missing ID")
	}

	// Check that the script uses require to load the parse_threat_detection_results.cjs file
	if !strings.Contains(agentSection, "require('/opt/gh-aw/actions/parse_threat_detection_results.cjs')") {
		t.Error("Parse results step doesn't use require to load parse_threat_detection_results.cjs")
	}

	// Check that setupGlobals is called
	if !strings.Contains(yaml, "setupGlobals(core, github, context, exec, io)") {
		t.Error("Parse results step doesn't call setupGlobals")
	}

	// Check that main() is awaited
	if !strings.Contains(yaml, "await main()") {
		t.Error("Parse results step doesn't await main()")
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

	// Detection is now inline in the agent job - check uses agent outputs
	if !strings.Contains(yaml, "needs.agent.outputs.detection_success == 'true'") {
		t.Error("Safe output jobs don't check detection result via agent outputs")
	}
}
