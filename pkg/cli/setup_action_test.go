//go:build !integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupActionCopiesAntigravityLogParser(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")

	projectRoot := filepath.Join(wd, "..", "..")
	setupScript := filepath.Join(projectRoot, "actions", "setup", "setup.sh")
	sourceParser := filepath.Join(projectRoot, "actions", "setup", "js", "parse_antigravity_log.cjs")

	if _, err := os.Stat(sourceParser); err != nil {
		t.Fatalf("parse_antigravity_log.cjs not found at %s: %v", sourceParser, err)
	}

	runnerTemp := filepath.Join(t.TempDir(), "runner-temp")
	require.NoError(t, os.MkdirAll(runnerTemp, 0o755), "Failed to create runner temp directory")

	destination := filepath.Join(runnerTemp, "gh-aw", "actions")
	githubOutput := filepath.Join(runnerTemp, "github-output.txt")

	cmd := exec.Command("bash", setupScript)
	cmd.Env = append(os.Environ(),
		"RUNNER_TEMP="+runnerTemp,
		"INPUT_DESTINATION="+destination,
		"GITHUB_OUTPUT="+githubOutput,
		// Never reset /tmp/gh-aw from a test: the test may run inside an agentic
		// workflow whose activation prompt already lives there.
		"GH_AW_SKIP_TMP_RESET=1",
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "setup.sh should succeed: %s", output)

	copiedParser := filepath.Join(destination, "parse_antigravity_log.cjs")
	assert.FileExists(t, copiedParser, "setup.sh should copy parse_antigravity_log.cjs into the runner payload")

	sourceContent, err := os.ReadFile(sourceParser)
	require.NoError(t, err, "Failed to read source parser")

	copiedContent, err := os.ReadFile(copiedParser)
	require.NoError(t, err, "Failed to read copied parser")

	assert.Equal(t, string(sourceContent), string(copiedContent), "Copied parser should match the source parser")
}

// TestSetupActionPreservesExistingActivationPrompt verifies that setup.sh does not wipe
// /tmp/gh-aw when an activation prompt is already present there. Re-running setup.sh from
// inside a live workflow (e.g. a workflow step that runs this repository's test suite)
// previously deleted /tmp/gh-aw/aw-prompts/prompt.txt, making the agent fail with
// "failed to read prompt file /tmp/gh-aw/aw-prompts/prompt.txt: ENOENT".
func TestSetupActionPreservesExistingActivationPrompt(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")

	projectRoot := filepath.Join(wd, "..", "..")
	setupScript := filepath.Join(projectRoot, "actions", "setup", "setup.sh")

	promptDir := "/tmp/gh-aw/aw-prompts"
	promptFile := filepath.Join(promptDir, "prompt.txt")
	if _, err := os.Stat(promptFile); err == nil {
		t.Skip("an activation prompt already exists at /tmp/gh-aw/aw-prompts/prompt.txt; not touching it")
	}
	require.NoError(t, os.MkdirAll(promptDir, 0o755), "Failed to create prompt directory")
	t.Cleanup(func() { _ = os.RemoveAll(promptDir) })
	require.NoError(t, os.WriteFile(promptFile, []byte("test prompt\n"), 0o644), "Failed to write prompt file")

	runnerTemp := filepath.Join(t.TempDir(), "runner-temp")
	require.NoError(t, os.MkdirAll(runnerTemp, 0o755), "Failed to create runner temp directory")

	cmd := exec.Command("bash", setupScript)
	cmd.Env = append(os.Environ(),
		"RUNNER_TEMP="+runnerTemp,
		"INPUT_DESTINATION="+filepath.Join(runnerTemp, "gh-aw", "actions"),
		"GITHUB_OUTPUT="+filepath.Join(runnerTemp, "github-output.txt"),
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "setup.sh should succeed: %s", output)

	assert.FileExists(t, promptFile, "setup.sh must not delete an existing activation prompt")
}
