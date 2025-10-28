package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitConfigurationInMainJob verifies that git configuration step is included in the main agentic job
func TestGitConfigurationInMainJob(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "git-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple test workflow
	testContent := `---
on: push
permissions:
  contents: read
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
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true)

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Generate YAML content
	lockContent, err := compiler.generateYAML(workflowData, testFile)
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

	if !strings.Contains(lockContent, "github-actions[bot]@users.noreply.github.com") {
		t.Error("Expected github-actions bot email to be present")
	}
}

// TestGitConfigurationStepsHelper tests the generateGitConfigurationSteps helper directly
func TestGitConfigurationStepsHelper(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	steps := compiler.generateGitConfigurationSteps()

	// Verify we get expected number of lines (now 9 instead of 5)
	if len(steps) != 9 {
		t.Errorf("Expected 9 lines in git configuration steps, got %d", len(steps))
	}

	// Verify the content of the steps
	expectedContents := []string{
		"Configure Git credentials",
		"run: |",
		"git config --global user.email",
		"git config --global user.name",
		"git remote set-url origin",
		"x-access-token",
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
