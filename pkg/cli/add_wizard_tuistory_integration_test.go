//go:build integration

package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type addWizardTuistorySetup struct {
	tempDir      string
	binaryPath   string
	workflowPath string
}

type addWizardManifestTuistorySetup struct {
	tempDir     string
	binaryPath  string
	packagePath string
	fakeGHDir   string
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

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s failed: %s", strings.Join(args, " "), string(output))
}

func setupAddWizardManifestTuistoryTest(t *testing.T) *addWizardManifestTuistorySetup {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "gh-aw-add-wizard-manifest-*")
	require.NoError(t, err, "Failed to create temp directory")

	runGitCommand(t, tempDir, "init")
	runGitCommand(t, tempDir, "config", "user.name", "Test User")
	runGitCommand(t, tempDir, "config", "user.email", "test@example.com")

	packagePath := filepath.Join(tempDir, "local-package")
	workflowsDir := filepath.Join(packagePath, "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create local package directories")

	manifestContent := `manifest-version: "1"
name: Local Add Wizard Package
files:
  - workflows/bootstrap.md
config:
  - type: repo-variable
    name: OPTIONAL_BOOTSTRAP_VAR
    prompt: Enter optional bootstrap variable
    description: Leave blank to skip this optional setup value.
    optional: true
  - type: handoff
    message: Post-install handoff should not appear before engine selection.
`
	require.NoError(t, os.WriteFile(filepath.Join(packagePath, "aw.yml"), []byte(manifestContent), 0644), "Failed to write local package manifest")
	require.NoError(t, os.WriteFile(filepath.Join(packagePath, "README.md"), []byte("# Local Add Wizard Package\n"), 0644), "Failed to write local package README")

	workflowContent := `---
name: Local Package Bootstrap Integration
on:
  workflow_dispatch:
engine: copilot
---

# Local Package Bootstrap Integration

This workflow is used by add-wizard manifest integration tests.
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "bootstrap.md"), []byte(workflowContent), 0644), "Failed to write local package workflow")

	runGitCommand(t, tempDir, "add", "local-package")
	runGitCommand(t, tempDir, "commit", "-m", "Add local package fixture")

	fakeGHDir, err := os.MkdirTemp("", "gh-aw-fake-gh-*")
	require.NoError(t, err, "Failed to create fake gh directory")

	fakeGHScript := `#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ge 2 && "$1" == "auth" && "$2" == "status" ]]; then
  echo "github.com"
  exit 0
fi

if [[ "$#" -ge 2 && "$1" == "repo" && "$2" == "view" ]]; then
  echo "octo/example"
  exit 0
fi

if [[ "$#" -ge 2 && "$1" == "api" ]]; then
  path=""
  for arg in "$@"; do
    case "$arg" in
      /repos/*|/orgs/*|/user)
        path="$arg"
        break
        ;;
    esac
  done

  case "$path" in
    /repos/octo/example)
      echo "public"
      exit 0
      ;;
    /repos/octo/example/actions/permissions)
      echo '{"enabled":true,"allowed_actions":"all"}'
      exit 0
      ;;
    /repos/octo/example/collaborators/tester/permission)
      echo '{"permission":"write"}'
      exit 0
      ;;
    /repos/octo/example/actions/variables?per_page=100|/repos/octo/example/actions/secrets?per_page=100|/repos/octo/example/actions/secrets|/orgs/octo/actions/secrets)
      exit 0
      ;;
    /user)
      echo "tester"
      exit 0
      ;;
  esac
fi

echo "unexpected gh invocation: $*" >&2
exit 1
`
	fakeGHPath := filepath.Join(fakeGHDir, "gh")
	require.NoError(t, os.WriteFile(fakeGHPath, []byte(fakeGHScript), 0755), "Failed to write fake gh script")

	return &addWizardManifestTuistorySetup{
		tempDir:     tempDir,
		binaryPath:  globalBinaryPath,
		packagePath: packagePath,
		fakeGHDir:   fakeGHDir,
	}
}

func waitForTuistoryText(t *testing.T, sessionName string, text string, timeoutMs int) {
	t.Helper()
	output, err := runTuistory(t, "-s", sessionName, "wait", text, "--timeout", fmt.Sprintf("%d", timeoutMs))
	if err != nil && strings.Contains(output, "requires an interactive terminal") {
		t.Skipf("tuistory session is not interactive in this environment: %s", output)
	}
	require.NoError(t, err, "Expected tuistory to find %q. Output: %s", text, output)
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

type interactivePTYSession struct {
	cmd    *exec.Cmd
	ptmx   *os.File
	output lockedBuffer
	done   chan error
}

func startInteractivePTYSession(t *testing.T, cmd *exec.Cmd) *interactivePTYSession {
	t.Helper()

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 140, Rows: 40})
	require.NoError(t, err, "failed to start PTY")

	session := &interactivePTYSession{
		cmd:  cmd,
		ptmx: ptmx,
		done: make(chan error, 1),
	}

	go func() {
		_, _ = io.Copy(&session.output, ptmx)
	}()
	go func() {
		session.done <- cmd.Wait()
	}()

	return session
}

func (s *interactivePTYSession) readAll() string {
	return s.output.String()
}

func (s *interactivePTYSession) waitForText(text string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(s.readAll(), text) {
			return nil
		}

		select {
		case err := <-s.done:
			if strings.Contains(s.readAll(), text) {
				return nil
			}
			if err == nil {
				return fmt.Errorf("process exited before output contained %q\nOutput:\n%s", text, s.readAll())
			}
			return fmt.Errorf("process exited before output contained %q: %w\nOutput:\n%s", text, err, s.readAll())
		default:
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for %q\nOutput:\n%s", text, s.readAll())
}

func (s *interactivePTYSession) writeString(t *testing.T, text string) {
	t.Helper()
	_, err := io.WriteString(s.ptmx, text)
	require.NoError(t, err, "failed to write to PTY")
}

func (s *interactivePTYSession) interrupt(t *testing.T) {
	t.Helper()
	s.writeString(t, "\x03")
}

func (s *interactivePTYSession) close(t *testing.T) {
	t.Helper()
	_ = s.ptmx.Close()
	select {
	case <-s.done:
	case <-time.After(2 * time.Second):
		_ = s.cmd.Process.Kill()
	}
}

func TestTuistoryAddWizardIntegration(t *testing.T) {
	const launchTimeoutMs = 30000 // 30 seconds

	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available, skipping tuistory add-wizard integration test")
	}
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh not available, skipping tuistory add-wizard integration test")
	}
	authCheck := exec.Command("gh", "auth", "status")
	if output, err := authCheck.CombinedOutput(); err != nil {
		t.Skipf("gh auth is not usable in this environment: %v (%s)", err, string(output))
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
	command := fmt.Sprintf("%s add-wizard ./%s --engine copilot --no-secret", setup.binaryPath, filepath.Base(setup.workflowPath))

	launchArgs := []string{
		"launch", command,
		"-s", sessionName,
		"--cwd", setup.tempDir,
		"--cols", "140",
		"--rows", "40",
		"--env", "CI=",
		"--env", "CONTINUOUS_INTEGRATION=",
		"--env", "GITHUB_ACTIONS=",
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

func TestTuistoryAddWizardManifestBootstrapRunsBeforeEngineSelection(t *testing.T) {
	setup := setupAddWizardManifestTuistoryTest(t)
	defer func() {
		_ = os.RemoveAll(setup.tempDir)
		_ = os.RemoveAll(setup.fakeGHDir)
	}()

	const earlyPrompt = "Enter optional bootstrap variable"
	const enginePrompt = "Which coding agent would you like to use?"
	const lateMessage = "Post-install handoff should not appear before engine selection."

	cmd := exec.Command(setup.binaryPath, "add-wizard", "./local-package", "--no-secret")
	cmd.Dir = setup.tempDir
	cmd.Env = append(os.Environ(),
		"CI=",
		"CONTINUOUS_INTEGRATION=",
		"GITHUB_ACTIONS=",
		"GO_TEST_MODE=",
		"NO_COLOR=1",
		"PAGER=cat",
		"GH_PAGER=cat",
		fmt.Sprintf("PATH=%s%c%s", setup.fakeGHDir, os.PathListSeparator, os.Getenv("PATH")),
	)

	session := startInteractivePTYSession(t, cmd)
	defer session.close(t)

	require.NoError(t, session.waitForText(earlyPrompt, 30*time.Second), "Expected pre-install bootstrap prompt")

	beforeEngineOutput := session.readAll()
	assert.Contains(t, beforeEngineOutput, earlyPrompt, "Expected pre-install bootstrap prompt to be shown")
	assert.NotContains(t, beforeEngineOutput, enginePrompt, "Engine selection should not start before pre-install bootstrap setup")
	assert.NotContains(t, beforeEngineOutput, lateMessage, "Post-install bootstrap steps should not run before engine selection")

	session.writeString(t, "\r")

	require.NoError(t, session.waitForText(enginePrompt, 30*time.Second), "Expected engine selection prompt")

	afterEngineOutput := session.readAll()
	assert.Contains(t, afterEngineOutput, earlyPrompt, "Expected pre-install bootstrap prompt in final session output")
	assert.Contains(t, afterEngineOutput, enginePrompt, "Expected engine selection prompt after pre-install bootstrap setup")
	assert.NotContains(t, afterEngineOutput, lateMessage, "Post-install bootstrap steps should not run before installation")

	earlyPromptIndex := strings.Index(afterEngineOutput, earlyPrompt)
	enginePromptIndex := strings.Index(afterEngineOutput, enginePrompt)
	require.NotEqual(t, -1, earlyPromptIndex, "Expected to find pre-install bootstrap prompt in session output")
	require.NotEqual(t, -1, enginePromptIndex, "Expected to find engine selection prompt in session output")
	assert.Less(t, earlyPromptIndex, enginePromptIndex, "Pre-install bootstrap prompt should appear before engine selection")

	session.interrupt(t)
}
