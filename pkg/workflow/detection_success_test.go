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
