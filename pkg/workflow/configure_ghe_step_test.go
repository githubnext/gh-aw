//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

// TestConfigureGHEStepIsEmitted verifies that the compiler always injects a
// "Configure gh CLI for GitHub Enterprise" step into the agent job.
// This step runs configure_gh_for_ghe.sh which detects the GitHub host from
// GITHUB_SERVER_URL and configures GH_HOST for all subsequent steps.
// For github.com it is a no-op; for *.ghe.com instances it sets GH_HOST so
// gh CLI commands in custom steps work without manual per-step configuration.
func TestConfigureGHEStepIsEmitted(t *testing.T) {
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
steps:
  - name: Fetch issues
    env:
      GH_TOKEN: ${{ github.token }}
    run: |
      gh issue list --state open
---

# Test workflow for GHE configure step
`

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "configure-ghe-step-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the .github/workflows directory
	workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Write the test workflow file
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := filepath.Join(workflowsDir, "test-workflow.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockStr := string(lockContent)

	// Extract the agent job section
	agentJobStart := strings.Index(lockStr, "  agent:")
	if agentJobStart == -1 {
		t.Fatal("Could not find 'agent' job in compiled workflow")
	}

	// Find the end of the agent job section (next top-level job)
	agentJobSection := lockStr[agentJobStart:]
	lines := strings.Split(agentJobSection, "\n")
	endIdx := len(agentJobSection)
	for i, line := range lines[1:] { // Skip the "agent:" line
		if len(line) > 2 && line[0] == ' ' && line[1] == ' ' && line[2] != ' ' && line[2] != '\t' {
			endIdx = 0
			for j := range i + 1 {
				endIdx += len(lines[j]) + 1
			}
			break
		}
	}
	agentJobSection = agentJobSection[:endIdx]

	// Verify the configure step is present in the agent job
	if !strings.Contains(agentJobSection, "- name: Configure gh CLI for GitHub Enterprise") {
		t.Error("Expected 'Configure gh CLI for GitHub Enterprise' step in agent job, but it was not found")
		t.Logf("Agent job section:\n%s", agentJobSection[:min(1500, len(agentJobSection))])
	}

	if !strings.Contains(agentJobSection, "run: bash \"${RUNNER_TEMP}/gh-aw/actions/configure_gh_for_ghe.sh\"") {
		t.Error("Expected 'run: bash \"${RUNNER_TEMP}/gh-aw/actions/configure_gh_for_ghe.sh\"' in agent job, but it was not found")
	}

	if !strings.Contains(agentJobSection, "GH_TOKEN: ${{ github.token }}") {
		t.Error("Expected 'GH_TOKEN: ${{ github.token }}' env var in configure step, but it was not found")
	}

	// Verify the configure step comes BEFORE the custom step
	configureIdx := strings.Index(agentJobSection, "Configure gh CLI for GitHub Enterprise")
	customStepIdx := strings.Index(agentJobSection, "Fetch issues")
	if configureIdx == -1 {
		t.Fatal("Configure step not found in agent job")
	}
	if customStepIdx == -1 {
		t.Fatal("Custom step 'Fetch issues' not found in agent job")
	}
	if configureIdx > customStepIdx {
		t.Error("Configure step should appear before custom steps, but it appears after 'Fetch issues'")
	}

	// Verify the configure step comes AFTER the temp directory step
	tempDirIdx := strings.Index(agentJobSection, "Create gh-aw temp directory")
	if tempDirIdx == -1 {
		t.Fatal("'Create gh-aw temp directory' step not found in agent job")
	}
	if tempDirIdx > configureIdx {
		t.Error("Configure step should appear after 'Create gh-aw temp directory', but it appears before")
	}

	t.Logf("✓ Configure gh CLI for GitHub Enterprise step is present before custom steps")
}
