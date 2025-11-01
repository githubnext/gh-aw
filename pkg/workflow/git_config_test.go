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

	// Verify we get expected number of lines (12 lines with expanded env block)
	if len(steps) != 12 {
		t.Errorf("Expected 12 lines in git configuration steps, got %d", len(steps))
	}

	// Verify the content of the steps
	expectedContents := []string{
		"Configure Git credentials",
		"env:",
		"REPO_NAME:",
		"SERVER_URL:",
		"GITHUB_TOKEN:",
		"run: |",
		"git config --global user.email",
		"git config --global user.name",
		"git remote set-url origin",
		"x-access-token",
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

// TestGitConfigurationNoTemplateInjection verifies that template expressions are properly moved to env block
func TestGitConfigurationNoTemplateInjection(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	steps := compiler.generateGitConfigurationSteps()
	fullContent := strings.Join(steps, "")

	// Split content into env section and run section
	parts := strings.Split(fullContent, "run: |")
	if len(parts) != 2 {
		t.Fatal("Expected steps to contain exactly one 'run: |' section")
	}

	envSection := parts[0]
	runSection := parts[1]

	// Verify that env section contains the template expressions
	envExpectedPatterns := []string{
		"${{ github.repository }}",
		"${{ github.server_url }}",
		"${{ github.token }}",
	}

	for _, pattern := range envExpectedPatterns {
		if !strings.Contains(envSection, pattern) {
			t.Errorf("Expected env section to contain template expression: %s", pattern)
		}
	}

	// Verify that run section does NOT contain template expressions (security fix)
	runForbiddenPatterns := []string{
		"${{ github.server_url }}",
		"${{ github.token }}",
		"${{ github.repository }}",
	}

	for _, pattern := range runForbiddenPatterns {
		if strings.Contains(runSection, pattern) {
			t.Errorf("Run section should NOT contain template expression (template injection vulnerability): %s", pattern)
		}
	}

	// Verify that run section uses environment variables instead
	runExpectedPatterns := []string{
		"${SERVER_URL",
		"${GITHUB_TOKEN}",
		"${REPO_NAME}",
	}

	for _, pattern := range runExpectedPatterns {
		if !strings.Contains(runSection, pattern) {
			t.Errorf("Expected run section to use environment variable: %s", pattern)
		}
	}
}
