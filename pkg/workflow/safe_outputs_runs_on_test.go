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

// TestRunsOnSlimField tests the top-level runs-on-slim field.
func TestRunsOnSlimField(t *testing.T) {
	tests := []struct {
		name             string
		frontmatter      string
		expectedRunsOn   string
		checkJobPatterns []string // job name patterns to check (e.g. "  activation:")
	}{
		{
			name: "runs-on-slim sets runner for activation job",
			frontmatter: `---
on: push
runs-on-slim: self-hosted
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn:   "runs-on: self-hosted",
			checkJobPatterns: []string{"\n  activation:"},
		},
		{
			name: "runs-on-slim without safe-outputs section",
			frontmatter: `---
on: push
runs-on-slim: ubuntu-22.04
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn:   "runs-on: ubuntu-22.04",
			checkJobPatterns: []string{"\n  activation:"},
		},
		{
			name: "safe-outputs.runs-on takes precedence over runs-on-slim",
			frontmatter: `---
on: push
runs-on-slim: ubuntu-22.04
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
  runs-on: self-hosted
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn:   "runs-on: self-hosted",
			checkJobPatterns: []string{"\n  activation:", "\n  safe_outputs:"},
		},
		{
			name: "default used when neither runs-on-slim nor safe-outputs.runs-on is set",
			frontmatter: `---
on: push
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn:   "runs-on: " + constants.DefaultActivationJobRunnerImage,
			checkJobPatterns: []string{"\n  activation:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "workflow-runs-on-slim-test")

			testFile := filepath.Join(tmpDir, "test.md")
			if err := os.WriteFile(testFile, []byte(tt.frontmatter), 0644); err != nil {
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

			for _, jobPattern := range tt.checkJobPatterns {
				jobStart := strings.Index(yamlStr, jobPattern)
				if jobStart == -1 {
					t.Logf("Job pattern %q not found in lock file (may not be generated for this config)", jobPattern)
					continue
				}
				jobSection := yamlStr[jobStart:min(jobStart+500, len(yamlStr))]
				if !strings.Contains(jobSection, tt.expectedRunsOn) {
					t.Errorf("Job matching %q does not use expected runs-on %q.\nJob section:\n%s", jobPattern, tt.expectedRunsOn, jobSection)
				}
			}
		})
	}
}

// TestFormatFrameworkJobRunsOn tests the formatFrameworkJobRunsOn helper directly.
func TestFormatFrameworkJobRunsOn(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		data           *WorkflowData
		expectedRunsOn string
	}{
		{
			name:           "nil WorkflowData returns default",
			data:           nil,
			expectedRunsOn: "runs-on: " + constants.DefaultActivationJobRunnerImage,
		},
		{
			name:           "empty WorkflowData returns default",
			data:           &WorkflowData{},
			expectedRunsOn: "runs-on: " + constants.DefaultActivationJobRunnerImage,
		},
		{
			name: "runs-on-slim used when safe-outputs.runs-on is empty",
			data: &WorkflowData{
				RunsOnSlim: "self-hosted",
			},
			expectedRunsOn: "runs-on: self-hosted",
		},
		{
			name: "safe-outputs.runs-on takes precedence over runs-on-slim",
			data: &WorkflowData{
				RunsOnSlim:  "ubuntu-22.04",
				SafeOutputs: &SafeOutputsConfig{RunsOn: "self-hosted"},
			},
			expectedRunsOn: "runs-on: self-hosted",
		},
		{
			name: "safe-outputs.runs-on used when runs-on-slim is empty",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{RunsOn: "windows-latest"},
			},
			expectedRunsOn: "runs-on: windows-latest",
		},
		{
			name: "default when safe-outputs present but runs-on is empty",
			data: &WorkflowData{
				RunsOnSlim:  "",
				SafeOutputs: &SafeOutputsConfig{},
			},
			expectedRunsOn: "runs-on: " + constants.DefaultActivationJobRunnerImage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.formatFrameworkJobRunsOn(tt.data)
			if result != tt.expectedRunsOn {
				t.Errorf("formatFrameworkJobRunsOn() = %q, want %q", result, tt.expectedRunsOn)
			}
		})
	}
}

// TestSafeOutputsCustomRunnerNodeSetup tests that Node.js v24 is emitted before
// actions/setup in the safe_outputs job when custom tokens are needed and a custom
// image runner is configured. setup.sh runs `npm install @actions/github` when
// safe-output-custom-tokens is enabled, so Node.js must be on PATH.
func TestSafeOutputsCustomRunnerNodeSetup(t *testing.T) {
	tests := []struct {
		name            string
		frontmatter     string
		expectNodeSetup bool
	}{
		{
			name: "custom runner with assign-to-agent (needs @actions/github) gets Node.js setup",
			frontmatter: `---
on: push
runs-on-slim: self-hosted
safe-outputs:
  assign-to-agent:
---

# Test Workflow`,
			expectNodeSetup: true,
		},
		{
			name: "custom runner with per-handler github-token gets Node.js setup",
			frontmatter: `---
on: push
runs-on-slim: enterprise-runner
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
    github-token: "${{ secrets.MY_TOKEN }}"
---

# Test Workflow`,
			expectNodeSetup: true,
		},
		{
			name: "standard GitHub-hosted runner with assign-to-agent does NOT get extra Node.js setup",
			frontmatter: `---
on: push
safe-outputs:
  assign-to-agent:
---

# Test Workflow`,
			expectNodeSetup: false,
		},
		{
			name: "custom runner without custom tokens does NOT get Node.js setup",
			frontmatter: `---
on: push
runs-on: self-hosted
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
---

# Test Workflow`,
			expectNodeSetup: false,
		},
		{
			name: "safe-outputs runs-on override (custom) with assign-to-agent gets Node.js setup",
			frontmatter: `---
on: push
safe-outputs:
  runs-on: enterprise-runner
  assign-to-agent:
---

# Test Workflow`,
			expectNodeSetup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "safe-outputs-node-setup-test")
			testFile := filepath.Join(tmpDir, "test.md")
			if err := os.WriteFile(testFile, []byte(tt.frontmatter), 0644); err != nil {
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

			safeOutputsSection := extractJobSection(yamlStr, "safe_outputs")

			// Find the position of Node.js setup and actions/setup within the safe_outputs job
			nodeSetupIdx := strings.Index(safeOutputsSection, "Setup Node.js")
			actionSetupIdx := strings.Index(safeOutputsSection, "Setup Scripts")

			if tt.expectNodeSetup {
				if nodeSetupIdx == -1 {
					t.Errorf("Expected 'Setup Node.js' in safe_outputs job, but not found.\nJob section:\n%s", safeOutputsSection)
					return
				}
				if !strings.Contains(safeOutputsSection, "actions/setup-node@") {
					t.Errorf("Expected 'actions/setup-node@' in safe_outputs job, but not found.\nJob section:\n%s", safeOutputsSection)
					return
				}
				// Node.js setup must come BEFORE actions/setup
				if nodeSetupIdx > actionSetupIdx {
					t.Errorf("Node.js setup step must appear before Setup Scripts step in safe_outputs job.\nnodeSetupIdx=%d actionSetupIdx=%d\nJob section:\n%s", nodeSetupIdx, actionSetupIdx, safeOutputsSection)
				}
			} else {
				// On a standard runner or without custom tokens, we should NOT add an extra Node.js setup
				// (the agent job may still have one, but the safe_outputs job should not)
				if nodeSetupIdx != -1 {
					t.Errorf("Did not expect 'Setup Node.js' in safe_outputs job, but found one.\nJob section:\n%s", safeOutputsSection)
				}
			}
		})
	}
}
