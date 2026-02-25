//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"

	"github.com/github/gh-aw/pkg/constants"
)

func TestSafeOutputsRunsOnConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		frontmatter    string
		expectedRunsOn string
	}{
		{
			name: "default runs-on when not specified",
			frontmatter: `---
on: push
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn: "runs-on: " + constants.DefaultActivationJobRunnerImage,
		},
		{
			name: "custom runs-on string",
			frontmatter: `---
on: push
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
  runs-on: windows-latest
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn: "runs-on: windows-latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and file
			tmpDir := testutil.TempDir(t, "workflow-runs-on-test")

			testFile := filepath.Join(tmpDir, "test.md")
			var err error
			err = os.WriteFile(testFile, []byte(tt.frontmatter), 0644)
			if err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the compiled lock file
			lockFile := filepath.Join(tmpDir, "test.lock.yml")
			yamlContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			yamlStr := string(yamlContent)
			if !strings.Contains(yamlStr, tt.expectedRunsOn) {
				t.Errorf("Expected compiled YAML to contain %q, but it didn't.\nYAML content:\n%s", tt.expectedRunsOn, yamlStr)
			}
		})
	}
}

func TestSafeOutputsRunsOnAppliedToAllJobs(t *testing.T) {
	frontmatter := `---
on: push
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
  add-comment:
  add-labels:
  update-issue:
  runs-on: self-hosted
---

# Test Workflow

This is a test workflow.`

	// Create temporary directory and file
	tmpDir := testutil.TempDir(t, "workflow-runs-on-test")

	testFile := filepath.Join(tmpDir, "test.md")
	var err error
	err = os.WriteFile(testFile, []byte(frontmatter), 0644)
	if err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled lock file
	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	yamlContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(yamlContent)

	// Check that all safe-outputs jobs use the custom runs-on
	expectedRunsOn := "runs-on: self-hosted"

	// Count occurrences - should appear for safe-outputs jobs + activation/membership jobs
	count := strings.Count(yamlStr, expectedRunsOn)
	if count < 1 { // At least one job should use the custom runner
		t.Errorf("Expected at least 1 occurrence of %q in compiled YAML, found %d.\nYAML content:\n%s", expectedRunsOn, count, yamlStr)
	}

	// Check specifically that the expected safe-outputs jobs use the custom runner
	// Use a pattern that matches YAML job definitions at the correct indentation level
	// to avoid matching JavaScript object properties inside bundled scripts
	expectedJobs := []string{"safe_outputs:"}
	for _, jobName := range expectedJobs {
		// Look for the job name at YAML indentation level (2 spaces under 'jobs:')
		yamlJobPattern := "\n  " + jobName
		jobStart := strings.Index(yamlStr, yamlJobPattern)
		if jobStart != -1 {
			// Look for runs-on within the next 500 characters of this job
			jobSection := yamlStr[jobStart : jobStart+500]
			defaultRunsOn := "runs-on: " + constants.DefaultActivationJobRunnerImage
			if strings.Contains(jobSection, defaultRunsOn) {
				t.Errorf("Job %q still uses default %q instead of custom runner.\nJob section:\n%s", jobName, defaultRunsOn, jobSection)
			}
			if !strings.Contains(jobSection, expectedRunsOn) {
				t.Errorf("Job %q does not use expected %q.\nJob section:\n%s", jobName, expectedRunsOn, jobSection)
			}
		}
	}
}

func TestFormatSafeOutputsRunsOnEdgeCases(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		safeOutputs    *SafeOutputsConfig
		expectedRunsOn string
	}{
		{
			name:           "nil safe outputs config",
			safeOutputs:    nil,
			expectedRunsOn: "runs-on: " + constants.DefaultActivationJobRunnerImage,
		},
		{
			name: "safe outputs config with nil runs-on",
			safeOutputs: &SafeOutputsConfig{
				RunsOn: "",
			},
			expectedRunsOn: "runs-on: " + constants.DefaultActivationJobRunnerImage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runsOn := compiler.formatSafeOutputsRunsOn(tt.safeOutputs)
			if runsOn != tt.expectedRunsOn {
				t.Errorf("Expected runs-on to be %q, got %q", tt.expectedRunsOn, runsOn)
			}
		})
	}
}

func TestUnlockJobUsesRunsOn(t *testing.T) {
	frontmatter := `---
on:
  issues:
    types: [opened]
    lock-for-agent: true
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
  runs-on: self-hosted
---

# Test Workflow

This is a test workflow.`

	tmpDir := testutil.TempDir(t, "workflow-unlock-runs-on-test")

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	yamlContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(yamlContent)

	// Verify the unlock job uses the safe-outputs runs-on value
	expectedRunsOn := "runs-on: self-hosted"
	unlockJobPattern := "\n  unlock:"
	unlockStart := strings.Index(yamlStr, unlockJobPattern)
	if unlockStart == -1 {
		t.Fatal("Expected unlock job to be present in compiled YAML")
	}

	unlockSection := yamlStr[unlockStart : unlockStart+500]
	defaultRunsOn := "runs-on: " + constants.DefaultActivationJobRunnerImage
	if strings.Contains(unlockSection, defaultRunsOn) {
		t.Errorf("Unlock job uses default %q instead of safe-outputs runner.\nUnlock section:\n%s", defaultRunsOn, unlockSection)
	}
	if !strings.Contains(unlockSection, expectedRunsOn) {
		t.Errorf("Unlock job does not use expected %q.\nUnlock section:\n%s", expectedRunsOn, unlockSection)
	}
}
