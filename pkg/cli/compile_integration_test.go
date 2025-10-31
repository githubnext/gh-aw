//go:build integration

package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

// integrationTestSetup holds the setup state for integration tests
type integrationTestSetup struct {
	tempDir      string
	originalWd   string
	binaryPath   string
	workflowsDir string
	cleanup      func()
}

// setupIntegrationTest creates a temporary directory and builds the gh-aw binary
// This is the equivalent of @Before in Java - common setup for all integration tests
func setupIntegrationTest(t *testing.T) *integrationTestSetup {
	t.Helper()

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "gh-aw-compile-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Save current working directory and change to temp directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Build the gh-aw binary
	binaryPath := filepath.Join(tempDir, "gh-aw")
	projectRoot := filepath.Join(originalWd, "..", "..")
	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = projectRoot
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gh-aw binary: %v", err)
	}

	// Copy binary to temp directory (use copy instead of move to avoid cross-device link issues)
	srcBinary := filepath.Join(projectRoot, "gh-aw")
	if err := copyFile(srcBinary, binaryPath); err != nil {
		t.Fatalf("Failed to copy gh-aw binary to temp directory: %v", err)
	}

	// Make the binary executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		t.Fatalf("Failed to make binary executable: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := ".github/workflows"
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Setup cleanup function
	cleanup := func() {
		err := os.Chdir(originalWd)
		if err != nil {
			t.Fatalf("Failed to change back to original working directory: %v", err)
		}
		err = os.RemoveAll(tempDir)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}

	return &integrationTestSetup{
		tempDir:      tempDir,
		originalWd:   originalWd,
		binaryPath:   binaryPath,
		workflowsDir: workflowsDir,
		cleanup:      cleanup,
	}
}

// TestCompileIntegration tests the compile command by executing the gh-aw CLI binary
func TestCompileIntegration(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Create a test markdown workflow file
	testWorkflow := `---
name: Integration Test Workflow
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Integration Test Workflow

This is a simple integration test workflow.

Please check the repository for any open issues and create a summary.
`

	testWorkflowPath := filepath.Join(setup.workflowsDir, "test.md")
	if err := os.WriteFile(testWorkflowPath, []byte(testWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow file: %v", err)
	}

	// Run the compile command
	cmd := exec.Command(setup.binaryPath, "compile", testWorkflowPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI compile command failed: %v\nOutput: %s", err, string(output))
	}

	// Check that the compiled .lock.yml file was created
	lockFilePath := filepath.Join(setup.workflowsDir, "test.lock.yml")
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		t.Fatalf("Expected lock file %s was not created", lockFilePath)
	}

	// Read and verify the generated lock file contains expected content
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)
	if !strings.Contains(lockContentStr, "name: \"Integration Test Workflow\"") {
		t.Errorf("Lock file should contain the workflow name")
	}

	if !strings.Contains(lockContentStr, "workflow_dispatch") {
		t.Errorf("Lock file should contain the trigger event")
	}

	if !strings.Contains(lockContentStr, "jobs:") {
		t.Errorf("Lock file should contain jobs section")
	}

	t.Logf("Integration test passed - successfully compiled workflow to %s", lockFilePath)
}

func TestCompileWithIncludeWithEmptyFrontmatterUnderPty(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Create an include file
	includeContent := `---
---
# Included Workflow

This is an included workflow file.
`
	includeFile := filepath.Join(setup.workflowsDir, "include.md")
	if err := os.WriteFile(includeFile, []byte(includeContent), 0644); err != nil {
		t.Fatalf("Failed to write include file: %v", err)
	}

	// Create a test markdown workflow file
	testWorkflow := `---
name: Integration Test Workflow
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Integration Test Workflow

This is a simple integration test workflow.

Please check the repository for any open issues and create a summary.

@include include.md
`
	testWorkflowPath := filepath.Join(setup.workflowsDir, "test.md")
	if err := os.WriteFile(testWorkflowPath, []byte(testWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow file: %v", err)
	}

	// Run the compile command
	cmd := exec.Command(setup.binaryPath, "compile", testWorkflowPath)
	// Start the command with a TTY attached to stdin/stdout/stderr
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("failed to start PTY: %v", err)
	}
	defer func() { _ = ptmx.Close() }() // Best effort

	// Capture all output from the PTY
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, ptmx) // reads both stdout/stderr via the PTY
		close(done)
	}()

	// Wait for the process to finish
	err = cmd.Wait()

	// Ensure reader goroutine drains remaining output
	select {
	case <-done:
	case <-time.After(750 * time.Millisecond):
	}

	output := buf.String()
	if err != nil {
		t.Fatalf("CLI compile command failed: %v\nOutput:\n%s", err, output)
	}

	// Check that the compiled .lock.yml file was created
	lockFilePath := filepath.Join(setup.workflowsDir, "test.lock.yml")
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		t.Fatalf("Expected lock file %s was not created", lockFilePath)
	}

	// Read and verify the generated lock file contains expected content
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)
	if !strings.Contains(lockContentStr, "name: \"Integration Test Workflow\"") {
		t.Errorf("Lock file should contain the workflow name")
	}

	if !strings.Contains(lockContentStr, "workflow_dispatch") {
		t.Errorf("Lock file should contain the trigger event")
	}

	if !strings.Contains(lockContentStr, "jobs:") {
		t.Errorf("Lock file should contain jobs section")
	}

	if strings.Contains(lockContentStr, "\x1b[") {
		t.Errorf("Lock file must not contain color escape sequences")
	}

	t.Logf("Integration test passed - successfully compiled workflow to %s", lockFilePath)
}

// TestCompileWithZizmor tests the compile command with --zizmor flag
func TestCompileWithZizmor(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Initialize git repository for zizmor to work (it needs git root)
	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = setup.tempDir
	if output, err := gitInitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v\nOutput: %s", err, string(output))
	}

	// Configure git user for the repository
	gitConfigEmail := exec.Command("git", "config", "user.email", "test@test.com")
	gitConfigEmail.Dir = setup.tempDir
	if output, err := gitConfigEmail.CombinedOutput(); err != nil {
		t.Fatalf("Failed to configure git user email: %v\nOutput: %s", err, string(output))
	}

	gitConfigName := exec.Command("git", "config", "user.name", "Test User")
	gitConfigName.Dir = setup.tempDir
	if output, err := gitConfigName.CombinedOutput(); err != nil {
		t.Fatalf("Failed to configure git user name: %v\nOutput: %s", err, string(output))
	}

	// Create a test markdown workflow file
	testWorkflow := `---
name: Zizmor Test Workflow
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
---

# Zizmor Test Workflow

This workflow tests the zizmor security scanner integration.
`

	testWorkflowPath := filepath.Join(setup.workflowsDir, "zizmor-test.md")
	if err := os.WriteFile(testWorkflowPath, []byte(testWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow file: %v", err)
	}

	// First compile without zizmor to create the lock file
	compileCmd := exec.Command(setup.binaryPath, "compile", testWorkflowPath)
	if output, err := compileCmd.CombinedOutput(); err != nil {
		t.Fatalf("Initial compile failed: %v\nOutput: %s", err, string(output))
	}

	// Check that the lock file was created
	lockFilePath := filepath.Join(setup.workflowsDir, "zizmor-test.lock.yml")
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		t.Fatalf("Expected lock file %s was not created", lockFilePath)
	}

	// Now compile with --zizmor flag
	zizmorCmd := exec.Command(setup.binaryPath, "compile", testWorkflowPath, "--zizmor", "--verbose")
	output, err := zizmorCmd.CombinedOutput()

	// The command should succeed even if zizmor finds issues
	if err != nil {
		t.Fatalf("Compile with --zizmor failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	// Note: With the new behavior, if there are 0 warnings, no zizmor output is displayed
	// The test just verifies that the command succeeds with --zizmor flag
	// If there are warnings, they will be shown in the format:
	// "🌈 zizmor X warnings in <filepath>"
	//   - [Severity] finding-type

	// The lock file should still exist after zizmor scan
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		t.Fatalf("Lock file was removed after zizmor scan")
	}

	t.Logf("Integration test passed - zizmor flag works correctly\nOutput: %s", outputStr)
}
