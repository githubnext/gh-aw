package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn: "runs-on: ubuntu-slim",
		},
		{
			name: "custom runs-on string",
			frontmatter: `---
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
			tmpDir, err := os.MkdirTemp("", "workflow-runs-on-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test.md")
			err = os.WriteFile(testFile, []byte(tt.frontmatter), 0644)
			if err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
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
	tmpDir, err := os.MkdirTemp("", "workflow-runs-on-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.md")
	err = os.WriteFile(testFile, []byte(frontmatter), 0644)
	if err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
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
	expectedJobs := []string{"create_issue:", "create_issue_comment:", "add_labels:", "update_issue:"}
	for _, jobName := range expectedJobs {
		if strings.Contains(yamlStr, jobName) {
			// Find the job section
			jobStart := strings.Index(yamlStr, jobName)
			if jobStart != -1 {
				// Look for runs-on within the next 500 characters of this job
				jobSection := yamlStr[jobStart : jobStart+500]
				if strings.Contains(jobSection, "runs-on: ubuntu-slim") {
					t.Errorf("Job %q still uses default 'runs-on: ubuntu-slim' instead of custom runner.\nJob section:\n%s", jobName, jobSection)
				}
				if !strings.Contains(jobSection, expectedRunsOn) {
					t.Errorf("Job %q does not use expected %q.\nJob section:\n%s", jobName, expectedRunsOn, jobSection)
				}
			}
		}
	}
}

func TestFormatSafeOutputsRunsOnEdgeCases(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name           string
		safeOutputs    *SafeOutputsConfig
		expectedRunsOn string
	}{
		{
			name:           "nil safe outputs config",
			safeOutputs:    nil,
			expectedRunsOn: "runs-on: ubuntu-slim",
		},
		{
			name: "safe outputs config with nil runs-on",
			safeOutputs: &SafeOutputsConfig{
				RunsOn: "",
			},
			expectedRunsOn: "runs-on: ubuntu-slim",
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
