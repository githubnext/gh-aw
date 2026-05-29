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

func TestCompileSandboxModelFallbackWorkflow(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-sandbox-model-fallback.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-sandbox-model-fallback.md")

	srcContent, err := os.ReadFile(srcPath)
	require.NoError(t, err, "should read source workflow file")
	require.NoError(t, os.WriteFile(dstPath, srcContent, 0644), "should write workflow to test dir")

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "compile command should succeed\nOutput: %s", string(output))

	lockPath := filepath.Join(setup.workflowsDir, "test-sandbox-model-fallback.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "should read lock file")
	lockStr := string(lockContent)

	assert.Contains(t, lockStr, `"modelFallback":{"enabled":false}`,
		"compiled lock file should embed sandbox.agent.model-fallback in the AWF config JSON")
}
