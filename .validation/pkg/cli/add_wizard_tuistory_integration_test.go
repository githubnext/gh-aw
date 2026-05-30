//go:build integration

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type addWizardTuistorySetup struct {
	tempDir      string
	binaryPath   string
	workflowPath string
}

func setupAddWizardTuistoryTest(t *testing.T) *addWizardTuistorySetup {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "gh-aw-add-wizard-tuistory-*")
	require.NoError(t, err, "Failed to create temp directory")

	// Initialize git repository required by add-wizard preconditions.
	gitInit := exec.Command("git", "init")
	gitInit.Dir = tempDir
	output, err := gitInit.CombinedOutput()
	require.NoError(t, err, "Failed to initialize git repository: %s", string(output))

	gitConfigName := exec.Command("git", "config", "user.name", "Test User")
	gitConfigName.Dir = tempDir
	_ = gitConfigName.Run()

	gitConfigEmail := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfigEmail.Dir = tempDir
	_ = gitConfigEmail.Run()

	binaryPath := filepath.Join(tempDir, "gh-aw")
	err = fileutil.CopyFile(globalBinaryPath, binaryPath)
	require.NoError(t, err, "Failed to copy gh-aw binary")

	err = os.Chmod(binaryPath, 0755)
	require.NoError(t, err, "Failed to make gh-aw binary executable")

	workflowPath := filepath.Join(tempDir, "local-test-workflow.md")
	workflowContent := `---
name: Local Add Wizard Integration
on:
  workflow_dispatch:
engine: copilot
---

# Local Add Wizard Integration

This workflow is used by add-wizard tuistory integration tests.
`
	err = os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to write local workflow fixture")

	return &addWizardTuistorySetup{
		tempDir:      tempDir,
		binaryPath:   binaryPath,
		workflowPath: workflowPath,
	}
}

func runTuistory(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command("npx", append([]string{"-y", "tuistory"}, args...)...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func waitForTuistoryText(t *testing.T, sessionName string, text string, timeoutMs int) {
	t.Helper()
	output, err := runTuistory(t, "-s", sessionName, "wait", text, "--timeout", fmt.Sprintf("%d", timeoutMs))
	require.NoError(t, err, "Expected tuistory to find %q. Output: %s", text, output)
}

func TestTuistoryAddWizardIntegration(t *testing.T) {
	const launchTimeoutMs = 30000 // 30 seconds

	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available, skipping tuistory add-wizard integration test")
	}

	versionOutput, err := runTuistory(t, "--version")
	if err != nil {
		t.Skipf("tuistory is not usable in this environment: %v (%s)", err, versionOutput)
	}

	setup := setupAddWizardTuistoryTest(t)
	defer func() {
		_ = os.RemoveAll(setup.tempDir)
	}()

	sessionName := fmt.Sprintf("gh-aw-add-wizard-%d", time.Now().UnixNano())
	command := fmt.Sprintf("%s add-wizard ./%s --engine copilot --skip-secret", setup.binaryPath, filepath.Base(setup.workflowPath))

	launchArgs := []string{
		"launch", command,
		"-s", sessionName,
		"--cwd", setup.tempDir,
		"--cols", "140",
		"--rows", "40",
		"--env", "CI=",
		"--env", "GO_TEST_MODE=",
		"--timeout", fmt.Sprintf("%d", launchTimeoutMs),
	}

	launchOutput, err := runTuistory(t, launchArgs...)
	if err != nil {
		t.Skipf("tuistory launch is not usable in this environment: %v (%s)", err, launchOutput)
	}

	defer func() {
		_, _ = runTuistory(t, "-s", sessionName, "close")
	}()

	// No git remote in the test repository forces add-wizard to prompt for owner/repo.
	waitForTuistoryText(t, sessionName, "Enter the target repository (owner/repo):", 120000)

	typeOutput, err := runTuistory(t, "-s", sessionName, "type", "github/gh-aw")
	require.NoError(t, err, "Failed to type repository slug. Output: %s", typeOutput)

	enterOutput, err := runTuistory(t, "-s", sessionName, "press", "enter")
	require.NoError(t, err, "Failed to press enter after repository slug. Output: %s", enterOutput)

	waitForTuistoryText(t, sessionName, "Do you want to proceed with these changes?", 120000)

	cancelOutput, err := runTuistory(t, "-s", sessionName, "press", "ctrl", "c")
	require.NoError(t, err, "Failed to send Ctrl+C to add-wizard session. Output: %s", cancelOutput)

	// Collect complete session output and assert cancellation occurred before changes were applied.
	readOutput, err := runTuistory(t, "-s", sessionName, "read", "--all")
	require.NoError(t, err, "Failed to read tuistory output after cancellation")
	assert.True(t,
		strings.Contains(readOutput, "confirmation failed") || strings.Contains(readOutput, "interrupted"),
		"Expected cancellation-related output, got:\n%s",
		readOutput,
	)

	addedWorkflowPath := filepath.Join(setup.tempDir, ".github", "workflows", filepath.Base(setup.workflowPath))
	_, statErr := os.Stat(addedWorkflowPath)
	assert.ErrorIs(t, statErr, os.ErrNotExist, "Workflow file should not be created when add-wizard is cancelled")
}
