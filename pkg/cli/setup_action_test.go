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
