//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileMaxTurnsFromSharedImport(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcMainPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-max-turns-imports.md")
	srcImportPath := filepath.Join(projectRoot, "pkg/cli/workflows/shared/max-turns-import.md")

	dstMainPath := filepath.Join(setup.workflowsDir, "test-max-turns-imports.md")
	dstImportDir := filepath.Join(setup.workflowsDir, "shared")
	dstImportPath := filepath.Join(dstImportDir, "max-turns-import.md")

	require.NoError(t, os.MkdirAll(dstImportDir, 0755), "Failed to create shared import directory")

	srcMainContent, err := os.ReadFile(srcMainPath)
	require.NoError(t, err, "Failed to read source workflow fixture")
	require.NoError(t, os.WriteFile(dstMainPath, srcMainContent, 0644), "Failed to write main workflow fixture")

	srcImportContent, err := os.ReadFile(srcImportPath)
	require.NoError(t, err, "Failed to read source imported workflow fixture")
	require.NoError(t, os.WriteFile(dstImportPath, srcImportContent, 0644), "Failed to write imported workflow fixture")

	cmd := exec.Command(setup.binaryPath, "compile", dstMainPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Compile failed:\n%s", string(output))

	lockFilePath := filepath.Join(setup.workflowsDir, "test-max-turns-imports.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read lock file")

	lockContentStr := string(lockContent)
	assert.Contains(t, lockContentStr, "GH_AW_MAX_TURNS: 9", "compiled workflow should include merged GH_AW_MAX_TURNS from shared import")
	assert.Contains(t, lockContentStr, "--max-turns 9", "compiled workflow should include merged Claude --max-turns flag from shared import")
}
