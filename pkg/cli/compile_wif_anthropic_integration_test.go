//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCompileClaudeWIFAnthropic verifies that a workflow using Anthropic
// Workload Identity Federation (engine.auth.type=github-oidc with
// provider=anthropic) compiles successfully without requiring ANTHROPIC_API_KEY.
//
// This is acceptance criterion 6 from issue #35937:
// "Integration test: Claude WIF workflow compiles without requiring ANTHROPIC_API_KEY secret"
func TestCompileClaudeWIFAnthropic(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Copy the canonical Anthropic WIF workflow fixture into the test's .github/workflows dir
	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-claude-wif-anthropic.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-claude-wif-anthropic.md")

	srcContent, err := os.ReadFile(srcPath)
	require.NoError(t, err, "Failed to read source workflow file %s", srcPath)
	require.NoError(t, os.WriteFile(dstPath, srcContent, 0644), "Failed to write workflow to test dir")

	// Compile the workflow - it must succeed (exit 0) without ANTHROPIC_API_KEY.
	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Claude WIF Anthropic workflow must compile without error:\n%s", string(output))

	// Verify the lock file was created
	lockFilePath := filepath.Join(setup.workflowsDir, "test-claude-wif-anthropic.lock.yml")
	_, statErr := os.Stat(lockFilePath)
	require.NoError(t, statErr, "Expected lock file %s to be created", lockFilePath)
}
