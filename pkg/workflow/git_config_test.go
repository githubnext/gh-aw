//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestGitConfigurationInMainJob verifies that git configuration step is included in the main agentic job
func TestGitConfigurationInMainJob(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "git-config-test")

	// Create a simple test workflow
	testContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
---

# Test Git Configuration

This is a test workflow to verify git configuration is included.
`

	testFile := filepath.Join(tmpDir, "test-git-config.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler()
	compiler.SetSkipValidation(true)

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Generate YAML content
	lockContent, _, _, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Verify git configuration step is present in the compiled workflow
	if !strings.Contains(lockContent, "Configure Git credentials") {
		t.Error("Expected 'Configure Git credentials' step to be present in compiled workflow")
	}

	// Verify the git config commands are present
	if !strings.Contains(lockContent, "git config --global user.email") {
		t.Error("Expected git config email command to be present")
	}

	if !strings.Contains(lockContent, "git config --global user.name") {
		t.Error("Expected git config name command to be present")
	}

	if !strings.Contains(lockContent, "git config --global am.keepcr true") {
		t.Error("Expected git config am.keepcr command to be present")
	}

	if !strings.Contains(lockContent, "github-actions[bot]@users.noreply.github.com") {
		t.Error("Expected github-actions bot email to be present")
	}
}

// TestGitConfigurationStepsHelper tests the generateGitConfigurationSteps helper directly
func TestGitConfigurationStepsHelper(t *testing.T) {
	compiler := NewCompiler()

	steps := compiler.generateGitConfigurationSteps()

	// Verify we get expected number of lines (13 lines with env block including GITHUB_TOKEN)
	if len(steps) != 13 {
		t.Errorf("Expected 13 lines in git configuration steps, got %d", len(steps))
	}

	// Verify the content of the steps
	expectedContents := []string{
		"Configure Git credentials",
		"env:",
		"REPO_NAME:",
		"GITHUB_TOKEN:",
		"run: |",
		"git config --global user.email",
		"git config --global user.name",
		"git config --global am.keepcr true",
		"git remote set-url origin",
		"x-access-token:${GITHUB_TOKEN}",
		"${REPO_NAME}.git",
		"Git configured with standard GitHub Actions identity",
	}

	fullContent := strings.Join(steps, "")

	for _, expected := range expectedContents {
		if !strings.Contains(fullContent, expected) {
			t.Errorf("Expected git configuration steps to contain '%s'", expected)
		}
	}

	// Verify proper indentation (should start with 6 spaces for job step level)
	if !strings.HasPrefix(steps[0], "      - name:") {
		t.Error("Expected first line to have proper indentation for job step (6 spaces)")
	}
}

// TestGitCredentialsCleanerStep verifies that git credentials cleaner step is included before agent execution
func TestGitCredentialsCleanerStep(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "git-cleaner-test")

	// Create a simple test workflow
	testContent := `---
on: push
permissions:
  contents: read
engine: copilot
---

# Test Git Credentials Cleaner

This is a test workflow to verify git credentials cleaner is included.
`

	testFile := filepath.Join(tmpDir, "test-git-cleaner.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler()
	compiler.SetSkipValidation(true)

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Generate YAML content
	lockContent, _, _, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Verify git credentials cleaner step is present
	if !strings.Contains(lockContent, "Clean git credentials") {
		t.Error("Expected 'Clean git credentials' step to be present in compiled workflow")
	}

	// Verify the cleaner script is called
	if !strings.Contains(lockContent, "clean_git_credentials.sh") {
		t.Error("Expected clean_git_credentials.sh script to be called")
	}

	// Verify the cleaner step comes before the agent execution
	// Find the positions of both steps
	cleanerPos := strings.Index(lockContent, "Clean git credentials")
	// The agent execution step is named "Execute GitHub Copilot CLI" (for Copilot engine)
	// or similar names for other engines
	agentPos := strings.Index(lockContent, "Execute GitHub Copilot CLI")
	if agentPos == -1 {
		// Try alternative patterns for other engines
		agentPos = strings.Index(lockContent, "agentic_execution")
	}

	if cleanerPos == -1 {
		t.Fatal("Could not find 'Clean git credentials' step in compiled workflow")
	}

	if agentPos == -1 {
		t.Fatal("Could not find agent execution step in compiled workflow")
	}

	// Verify cleaner comes before agent execution
	if cleanerPos >= agentPos {
		t.Error("Expected 'Clean git credentials' step to come before agent execution step")
	}
}

// TestGitCredentialsCleanerStepsHelper tests the generateGitCredentialsCleanerStep helper directly
func TestGitCredentialsCleanerStepsHelper(t *testing.T) {
	compiler := NewCompiler()

	steps := compiler.generateGitCredentialsCleanerStep()

	// Verify we get expected number of lines (3 lines: name, continue-on-error, and run)
	if len(steps) != 3 {
		t.Errorf("Expected 3 lines in git credentials cleaner steps, got %d", len(steps))
	}

	// Verify the content of the steps
	expectedContents := []string{
		"Clean git credentials",
		"continue-on-error: true",
		"run: bash \"${RUNNER_TEMP}/gh-aw/actions/clean_git_credentials.sh\"",
	}

	fullContent := strings.Join(steps, "")

	for _, expected := range expectedContents {
		if !strings.Contains(fullContent, expected) {
			t.Errorf("Expected git credentials cleaner steps to contain '%s'", expected)
		}
	}

	// Verify proper indentation (should start with 6 spaces for job step level)
	if !strings.HasPrefix(steps[0], "      - name:") {
		t.Error("Expected first line to have proper indentation for job step (6 spaces)")
	}
}

// TestGitConfigurationSkippedWhenCheckoutDisabled verifies that git credential steps
// are not emitted when checkout: false is set in the workflow frontmatter.
func TestGitConfigurationSkippedWhenCheckoutDisabled(t *testing.T) {
	tmpDir := testutil.TempDir(t, "git-config-checkout-false-test")

	testContent := `---
on: issues
permissions:
  issues: read
engine: copilot
checkout: false
---

# Test Workflow (no checkout)

This workflow uses API tools only and does not need the repository to be checked out.
`

	testFile := filepath.Join(tmpDir, "test-no-checkout.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	compiler.SetSkipValidation(true)

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	lockContent, _, _, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// When checkout: false, the agent job must NOT contain "Configure Git credentials"
	// since there is no .git directory and git remote set-url origin would fail.
	if strings.Contains(lockContent, "Configure Git credentials") {
		t.Error("'Configure Git credentials' step must NOT be present when checkout: false (no .git directory)")
	}

	// The "Clean git credentials" step should still be present (resilient, continue-on-error).
	// Assert that the cleaner step block itself contains both the name and continue-on-error
	// to avoid false positives from other steps that also use continue-on-error.
	const cleanerStepBlock = "- name: Clean git credentials\n        continue-on-error: true\n        run: bash \"${RUNNER_TEMP}/gh-aw/actions/clean_git_credentials.sh\""
	if !strings.Contains(lockContent, cleanerStepBlock) {
		t.Error("Expected 'Clean git credentials' step with 'continue-on-error: true' to be present when checkout: false")
	}
}
