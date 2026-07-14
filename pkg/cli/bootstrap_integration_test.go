//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bootstrapIntegrationSetup struct {
	base       *integrationTestSetup
	repoDir    string
	repoArg    string
	fakeBinDir string
	argsLog    string
	pathEnv    string
}

func setupBootstrapIntegrationTest(t *testing.T) *bootstrapIntegrationSetup {
	t.Helper()

	base := setupIntegrationTest(t)
	repoDir := filepath.Join(base.tempDir, "repo")
	require.NoError(t, os.MkdirAll(repoDir, 0o755))

	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = repoDir
	output, err := gitInitCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run git init: %s", string(output))

	gitNameCmd := exec.Command("git", "config", "user.name", "Bootstrap Test")
	gitNameCmd.Dir = repoDir
	output, err = gitNameCmd.CombinedOutput()
	require.NoError(t, err, "Failed to set git user.name: %s", string(output))

	gitEmailCmd := exec.Command("git", "config", "user.email", "bootstrap@example.com")
	gitEmailCmd.Dir = repoDir
	output, err = gitEmailCmd.CombinedOutput()
	require.NoError(t, err, "Failed to set git user.email: %s", string(output))

	gitRemoteCmd := exec.Command("git", "remote", "add", "origin", "https://github.com/octo/platform-ops.git")
	gitRemoteCmd.Dir = repoDir
	output, err = gitRemoteCmd.CombinedOutput()
	require.NoError(t, err, "Failed to add git remote: %s", string(output))

	fakeBinDir := filepath.Join(base.tempDir, "fake-bin")
	require.NoError(t, os.MkdirAll(fakeBinDir, 0o755))
	argsLog := filepath.Join(base.tempDir, "gh-args.log")
	fakeGH := filepath.Join(fakeBinDir, "gh")
	fakeGHScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"" + argsLog + "\"\n" +
		"if [ \"$1\" = \"auth\" ] && [ \"$2\" = \"status\" ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"repo\" ] && [ \"$2\" = \"view\" ]; then\n" +
		"  printf '%s\\n' 'octo/platform-ops'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf 'unexpected gh invocation: %s\\n' \"$*\" >&2\n" +
		"exit 1\n"
	require.NoError(t, os.WriteFile(fakeGH, []byte(fakeGHScript), 0o755))

	return &bootstrapIntegrationSetup{
		base:       base,
		repoDir:    repoDir,
		repoArg:    "repo",
		fakeBinDir: fakeBinDir,
		argsLog:    argsLog,
		pathEnv:    fakeBinDir + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
}

func TestBootstrapCommandPlanIntegration(t *testing.T) {
	setup := setupBootstrapIntegrationTest(t)
	defer setup.base.cleanup()

	cmd := exec.Command(setup.base.binaryPath, "bootstrap", "--repo", "octo/platform-ops", "--dir", setup.repoArg, "--plan")
	cmd.Dir = setup.base.tempDir
	cmd.Env = append(os.Environ(), "PATH="+setup.pathEnv)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	require.NoError(t, err, "bootstrap --plan should succeed: %s", outputStr)
	assert.Contains(t, outputStr, "Bootstrap plan for octo/platform-ops")
	assert.Contains(t, outputStr, "attach existing checkout at repo")
	assert.Contains(t, outputStr, "initialize repository artifacts")
	assert.NoFileExists(t, filepath.Join(setup.repoDir, ".gitattributes"))

	argsLog, err := os.ReadFile(setup.argsLog)
	require.NoError(t, err)
	assert.Contains(t, string(argsLog), "auth status")
	assert.Contains(t, string(argsLog), "repo view octo/platform-ops --json nameWithOwner --jq .nameWithOwner")
	assert.NotContains(t, string(argsLog), "repo clone")
}

func TestBootstrapCommandInitAndRerunIntegration(t *testing.T) {
	setup := setupBootstrapIntegrationTest(t)
	defer setup.base.cleanup()

	runBootstrap := func(label string) string {
		cmd := exec.Command(setup.base.binaryPath, "bootstrap", "--repo", "octo/platform-ops", "--dir", setup.repoArg, "--yes")
		cmd.Dir = setup.base.tempDir
		cmd.Env = append(os.Environ(), "PATH="+setup.pathEnv)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		require.NoError(t, err, "%s should succeed: %s", label, outputStr)
		return outputStr
	}

	firstOutput := runBootstrap("first bootstrap run")
	assert.Contains(t, firstOutput, "Initialized repository for agentic workflows")
	assert.Contains(t, firstOutput, "Bootstrap completed for octo/platform-ops")

	assert.FileExists(t, filepath.Join(setup.repoDir, ".gitattributes"))
	assert.FileExists(t, filepath.Join(setup.repoDir, ".vscode", "settings.json"))
	assert.FileExists(t, filepath.Join(setup.repoDir, ".github", "skills", "agentic-workflows", "SKILL.md"))
	assert.FileExists(t, filepath.Join(setup.repoDir, ".github", "agents", "agentic-workflows.md"))
	assert.FileExists(t, filepath.Join(setup.repoDir, ".github", "mcp.json"))
	assert.FileExists(t, filepath.Join(setup.repoDir, ".github", "workflows", "copilot-setup-steps.yml"))

	secondOutput := runBootstrap("second bootstrap run")
	assert.Contains(t, secondOutput, "Bootstrap already satisfied for octo/platform-ops")
	assert.NotContains(t, secondOutput, "Initialized repository for agentic workflows")

	argsLog, err := os.ReadFile(setup.argsLog)
	require.NoError(t, err)
	assert.NotContains(t, string(argsLog), "repo clone")
	assert.GreaterOrEqual(t, strings.Count(string(argsLog), "auth status"), 2)
	assert.GreaterOrEqual(t, strings.Count(string(argsLog), "repo view octo/platform-ops --json nameWithOwner --jq .nameWithOwner"), 2)
}
