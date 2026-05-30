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

// TestCompileEngineEnvNeedsExpression verifies that engine.env values containing
// needs.<job>.outputs.* expressions cause the referenced custom job to be added
// as a direct dependency of the agent job.
func TestCompileEngineEnvNeedsExpression(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-engine-env-needs.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-engine-env-needs.md")

	srcContent, err := os.ReadFile(srcPath)
	require.NoError(t, err, "Failed to read source workflow fixture")
	require.NoError(t, os.WriteFile(dstPath, srcContent, 0644), "Failed to write workflow fixture")

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Compile failed:\n%s", string(output))

	lockFilePath := filepath.Join(setup.workflowsDir, "test-engine-env-needs.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read lock file")

	var workflow map[string]any
	require.NoError(t, goyaml.Unmarshal(lockContent, &workflow), "Lock file should be valid YAML")

	jobs, ok := workflow["jobs"].(map[string]any)
	require.True(t, ok, "Compiled workflow should include jobs map")

	agentJob, ok := jobs["agent"].(map[string]any)
	require.True(t, ok, "Compiled workflow should include agent job")

	needsRaw, ok := agentJob["needs"].([]any)
	require.True(t, ok, "agent job should have a needs list")

	needs := make([]string, 0, len(needsRaw))
	for _, need := range needsRaw {
		require.IsType(t, "", need, "agent needs entries should be strings")
		needs = append(needs, need.(string))
	}

	assert.Contains(t, needs, "provide_value_to_agent",
		"agent job must depend on provide_value_to_agent referenced in engine.env")
	assert.Contains(t, needs, "activation",
		"agent job must still depend on activation")
}
