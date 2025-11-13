package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDetectionJobHasSuccessOutput verifies that the detection job has a success output
func TestDetectionJobHasSuccessOutput(t *testing.T) {
	tmpDir := t.TempDir()
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

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Read the compiled YAML
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	yamlBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read compiled YAML: %v", err)
	}
	yaml := string(yamlBytes)

	// Check that detection job exists
	if !strings.Contains(yaml, "detection:") {
		t.Error("Detection job not found in compiled YAML")
	}

	// Check that detection job has outputs section with success output
	if !strings.Contains(yaml, "success: ${{ steps.parse_results.outputs.success }}") {
		t.Error("Detection job missing success output")
	}

	// Check that parse_results step has an ID
	if !strings.Contains(yaml, "id: parse_results") {
		t.Error("Parse results step missing ID")
	}

	// Check that success is set in the script
	if !strings.Contains(yaml, "core.setOutput('success', 'true')") {
		t.Error("Parse results step doesn't set success to true")
	}

	if !strings.Contains(yaml, "core.setOutput('success', 'false')") {
		t.Error("Parse results step doesn't set success to false")
	}
}

// TestSafeOutputJobsCheckDetectionSuccess verifies that safe output jobs check detection success
func TestSafeOutputJobsCheckDetectionSuccess(t *testing.T) {
	tmpDir := t.TempDir()
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

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Read the compiled YAML
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	yamlBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read compiled YAML: %v", err)
	}
	yaml := string(yamlBytes)

	// Check that create_issue job has detection success check in its condition
	if !strings.Contains(yaml, "create_issue:") {
		t.Fatal("create_issue job not found")
	}

	if !strings.Contains(yaml, "needs.detection.outputs.success == 'true'") {
		t.Error("Safe output jobs don't check detection success")
	}
}

// TestParseResultsStepAlwaysExecutes verifies that the parse_results step has if: always()
func TestParseResultsStepAlwaysExecutes(t *testing.T) {
	tmpDir := t.TempDir()
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

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Read the compiled YAML
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	yamlBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read compiled YAML: %v", err)
	}
	yaml := string(yamlBytes)

	// Check that parse_results step has if: always()
	// We look for the pattern: id: parse_results followed by if: always()
	parseResultsIdx := strings.Index(yaml, "id: parse_results")
	if parseResultsIdx == -1 {
		t.Fatal("parse_results step not found")
	}

	// Get a section of the YAML around the parse_results step
	section := yaml[parseResultsIdx : parseResultsIdx+200]
	if !strings.Contains(section, "if: always()") {
		t.Error("parse_results step doesn't have 'if: always()' - it won't execute if earlier steps fail")
	}
}

// TestDetectionSuccessChecksAgentJobResult verifies that the success output checks agent job result
func TestDetectionSuccessChecksAgentJobResult(t *testing.T) {
	tmpDir := t.TempDir()
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

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Read the compiled YAML
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	yamlBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read compiled YAML: %v", err)
	}
	yaml := string(yamlBytes)

	// Check that the parse_results step receives the agent job result as an environment variable
	if !strings.Contains(yaml, "AGENT_JOB_RESULT: ${{ needs.agent.result }}") {
		t.Error("parse_results step doesn't receive agent job result")
	}

	// Check that the script checks the agent job result
	if !strings.Contains(yaml, "process.env.AGENT_JOB_RESULT") {
		t.Error("parse_results script doesn't read AGENT_JOB_RESULT")
	}

	// Check that the script validates agent job succeeded
	if !strings.Contains(yaml, "agentJobResult === 'success'") {
		t.Error("parse_results script doesn't check if agent job succeeded")
	}

	// Check that failure message mentions agent job
	if !strings.Contains(yaml, "Agent job failed or was cancelled") {
		t.Error("parse_results script doesn't fail when agent job didn't succeed")
	}
}
