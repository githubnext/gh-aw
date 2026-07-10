//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initIntegrationTestSetup holds the setup state for init integration tests
type initIntegrationTestSetup struct {
	tempDir    string
	originalWd string
	binaryPath string
	cleanup    func()
}

// setupInitIntegrationTest creates a minimal test environment for init command:
// - temporary directory with git init
// - pre-built gh-aw binary
func setupInitIntegrationTest(t *testing.T) *initIntegrationTestSetup {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "gh-aw-init-integration-*")
	require.NoError(t, err, "Failed to create temp directory")

	originalWd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	err = os.Chdir(tempDir)
	require.NoError(t, err, "Failed to change to temp directory")

	// Initialize git repository (required by init command)
	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = tempDir
	output, err := gitInitCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run git init: %s", string(output))

	// Copy the pre-built binary to this test's temp directory
	binaryPath := filepath.Join(tempDir, "gh-aw")
	err = fileutil.CopyFile(globalBinaryPath, binaryPath)
	require.NoError(t, err, "Failed to copy gh-aw binary to temp directory")

	err = os.Chmod(binaryPath, 0755)
	require.NoError(t, err, "Failed to make binary executable")

	cleanup := func() {
		_ = os.Chdir(originalWd)
		_ = os.RemoveAll(tempDir)
	}

	return &initIntegrationTestSetup{
		tempDir:    tempDir,
		originalWd: originalWd,
		binaryPath: binaryPath,
		cleanup:    cleanup,
	}
}

// TestInitCommandIntegration tests that the init command generates the expected files
// when run through the compiled CLI binary.
func TestInitCommandIntegration(t *testing.T) {
	setup := setupInitIntegrationTest(t)
	defer setup.cleanup()

	// Run: gh-aw init --no-mcp (skip MCP to avoid network requirements)
	cmd := exec.Command(setup.binaryPath, "init", "--no-mcp")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	t.Logf("init command output:\n%s", outputStr)

	require.NoError(t, err, "init command should succeed: %s", outputStr)

	// .gitattributes must exist and contain the lock.yml entry
	gitAttrPath := filepath.Join(setup.tempDir, ".gitattributes")
	content, err := os.ReadFile(gitAttrPath)
	require.NoError(t, err, ".gitattributes should be created")
	assert.Contains(t, string(content), ".github/workflows/*.lock.yml linguist-generated=true merge=ours",
		".gitattributes should mark lock.yml files as generated")

	// Dispatcher skill should be created
	skillPath := filepath.Join(setup.tempDir, ".github", "skills", "agentic-workflows", "SKILL.md")
	_, err = os.Stat(skillPath)
	require.NoError(t, err, "dispatcher skill file should be created at %s", skillPath)

	// Custom agent should be created
	agentPath := filepath.Join(setup.tempDir, ".github", "agents", "agentic-workflows.md")
	_, err = os.Stat(agentPath)
	require.NoError(t, err, "custom agent file should be created at %s", agentPath)

	// VSCode settings should be created
	vscodePath := filepath.Join(setup.tempDir, ".vscode", "settings.json")
	_, err = os.Stat(vscodePath)
	require.NoError(t, err, ".vscode/settings.json should be created")
}

// TestInitCommandNoSkillIntegration tests that --no-skill skips creating the dispatcher skill
func TestInitCommandNoSkillIntegration(t *testing.T) {
	setup := setupInitIntegrationTest(t)
	defer setup.cleanup()

	cmd := exec.Command(setup.binaryPath, "init", "--no-mcp", "--no-skill")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	t.Logf("init --no-skill command output:\n%s", outputStr)

	require.NoError(t, err, "init --no-skill command should succeed: %s", outputStr)

	// .gitattributes must still be created
	gitAttrPath := filepath.Join(setup.tempDir, ".gitattributes")
	_, err = os.Stat(gitAttrPath)
	require.NoError(t, err, ".gitattributes should be created even with --no-skill")

	// Dispatcher skill must NOT be created
	skillPath := filepath.Join(setup.tempDir, ".github", "skills", "agentic-workflows", "SKILL.md")
	_, err = os.Stat(skillPath)
	assert.True(t, os.IsNotExist(err), "dispatcher skill should NOT be created with --no-skill")
}

// TestInitCommandIdempotentIntegration tests that running init twice produces no errors
func TestInitCommandIdempotentIntegration(t *testing.T) {
	setup := setupInitIntegrationTest(t)
	defer setup.cleanup()

	runInit := func(label string) {
		cmd := exec.Command(setup.binaryPath, "init", "--no-mcp")
		cmd.Dir = setup.tempDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s: init command should succeed: %s", label, string(output))
	}

	runInit("first run")
	runInit("second run (idempotent)")

	// Verify files still exist after second run
	gitAttrPath := filepath.Join(setup.tempDir, ".gitattributes")
	content, err := os.ReadFile(gitAttrPath)
	require.NoError(t, err, ".gitattributes should exist after second run")
	assert.Contains(t, string(content), ".github/workflows/*.lock.yml",
		".gitattributes should still have the lock.yml entry after second run")

	// Ensure no duplicate lines were added
	lines := strings.Split(string(content), "\n")
	count := 0
	for _, line := range lines {
		if strings.Contains(line, ".github/workflows/*.lock.yml") {
			count++
		}
	}
	assert.Equal(t, 1, count, ".gitattributes should not have duplicate lock.yml entries")
}
