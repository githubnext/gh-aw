//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	goyaml "github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileSafeOutputsNeedsMergedWithImports(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcMainPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-safe-outputs-needs-imports.md")
	srcImportPath := filepath.Join(projectRoot, "pkg/cli/workflows/shared/safe-outputs-needs-import.md")

	dstMainPath := filepath.Join(setup.workflowsDir, "test-safe-outputs-needs-imports.md")
	dstImportDir := filepath.Join(setup.workflowsDir, "shared")
	dstImportPath := filepath.Join(dstImportDir, "safe-outputs-needs-import.md")

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

	lockFilePath := filepath.Join(setup.workflowsDir, "test-safe-outputs-needs-imports.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read lock file")

	var workflow map[string]any
	require.NoError(t, goyaml.Unmarshal(lockContent, &workflow), "Lock file should be valid YAML")

	jobs, ok := workflow["jobs"].(map[string]any)
	require.True(t, ok, "Compiled workflow should include jobs map")
	safeOutputsJob, ok := jobs["safe_outputs"].(map[string]any)
	require.True(t, ok, "Compiled workflow should include safe_outputs job")
	needsRaw, ok := safeOutputsJob["needs"].([]any)
	require.True(t, ok, "safe_outputs job should include needs array")

	needs := make([]string, 0, len(needsRaw))
	for _, need := range needsRaw {
		require.IsType(t, "", need, "safe_outputs needs entries should be strings")
		needs = append(needs, need.(string))
	}

	assert.Contains(t, needs, "main_job", "safe_outputs.needs should include top-level custom dependency")
	assert.Contains(t, needs, "imported_job", "safe_outputs.needs should include imported custom dependency")
	assert.Contains(t, needs, "shared_job", "safe_outputs.needs should include imported custom dependency")

	importedJobCount := 0
	for _, need := range needs {
		if need == "imported_job" {
			importedJobCount++
		}
	}
	assert.Equal(t, 1, importedJobCount, "safe_outputs.needs should dedupe duplicate custom dependencies")
}
