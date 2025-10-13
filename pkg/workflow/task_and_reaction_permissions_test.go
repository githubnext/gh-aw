package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestActivationAndAddReactionJobsPermissions(t *testing.T) {
	// Test that activation and add_reaction jobs do not have contents permissions and checkout steps
	tmpDir, err := os.MkdirTemp("", "permissions-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow with both activation job and add_reaction job
	testContent := `---
on:
  issues:
    types: [opened]
  reaction: eyes
tools:
  github:
    allowed: [list_issues]
engine: claude
---

# Test Workflow for Task and Add Reaction

This workflow should generate both activation and add_reaction jobs without contents permissions.

The activation job references text output: "${{ needs.activation.outputs.text }}"
`

	testFile := filepath.Join(tmpDir, "test-permissions.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Calculate the lock file path
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"

	// Read the generated lock file
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Test 1: Verify activation job exists and has no contents permission
	if !strings.Contains(lockContentStr, constants.ActivationJobName+":") {
		t.Error("Expected activation job to be present in generated workflow")
	}

	// Test 2: Verify activation job has no checkout step
	activationJobSection := extractJobSection(lockContentStr, constants.ActivationJobName)
	if strings.Contains(activationJobSection, "actions/checkout") {
		t.Error("Activation job should not contain actions/checkout step")
	}

	// Test 3: Verify activation job has no contents permission
	if strings.Contains(activationJobSection, "contents:") {
		t.Error("Activation job should not have contents permission")
	}

	// Test 4: Verify add_reaction job exists and has no contents permission
	if !strings.Contains(lockContentStr, "add_reaction:") {
		t.Error("Expected add_reaction job to be present in generated workflow")
	}

	// Test 5: Verify add_reaction job has no checkout step
	addReactionJobSection := extractJobSection(lockContentStr, "add_reaction")
	if strings.Contains(addReactionJobSection, "actions/checkout") {
		t.Error("Add_reaction job should not contain actions/checkout step")
	}

	// Test 6: Verify add_reaction job has no contents permission
	if strings.Contains(addReactionJobSection, "contents:") {
		t.Error("Add_reaction job should not have contents permission")
	}

	// Test 7: Verify add_reaction job still has required permissions
	if !strings.Contains(addReactionJobSection, "discussions: write") {
		t.Error("Add_reaction job should have discussions: write permission")
	}
	if !strings.Contains(addReactionJobSection, "issues: write") {
		t.Error("Add_reaction job should still have issues: write permission")
	}
	if !strings.Contains(addReactionJobSection, "pull-requests: write") {
		t.Error("Add_reaction job should still have pull-requests: write permission")
	}
}

// extractJobSection extracts a specific job section from the YAML content
func extractJobSection(yamlContent, jobName string) string {
	lines := strings.Split(yamlContent, "\n")
	var jobLines []string
	inJob := false
	jobPrefix := "  " + jobName + ":"

	for i, line := range lines {
		if strings.HasPrefix(line, jobPrefix) {
			inJob = true
			jobLines = append(jobLines, line)
			continue
		}

		if inJob {
			// If we hit another job at the same level (starts with "  " and ends with ":"), stop
			if strings.HasPrefix(line, "  ") && strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "    ") {
				break
			}
			// If we hit the end of jobs section, stop
			if strings.HasPrefix(line, "jobs:") && i > 0 {
				break
			}
			jobLines = append(jobLines, line)
		}
	}

	return strings.Join(jobLines, "\n")
}
