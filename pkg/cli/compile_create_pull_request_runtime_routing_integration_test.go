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

func TestCompileCreatePullRequestRuntimeRoutingWorkflow(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-copilot-create-pull-request-runtime-routing.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-copilot-create-pull-request-runtime-routing.md")
	copyWorkflowFile(t, srcPath, dstPath)

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "compile should succeed:\n%s", string(output))

	lockFilePath := filepath.Join(setup.workflowsDir, "test-copilot-create-pull-request-runtime-routing.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "failed to read compiled lock file")

	lockContentStr := string(lockContent)
	assert.Contains(t, lockContentStr, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", "lock file should include safe outputs handler config")
	assert.Contains(t, lockContentStr, `GH_AW_INPUT_REVIEW_TEAM: ${{ inputs.review-team }}`, "workflow_dispatch input should be preserved for runtime templating")
	assert.Contains(t, lockContentStr, `\"reviewers\":\"${{ github.actor }}\"`, "reviewers expression should remain a runtime string in handler config")
	assert.Contains(t, lockContentStr, `\"team_reviewers\":\"${{ inputs.review-team }}\"`, "team_reviewers expression should remain a runtime string in handler config")
	assert.Contains(t, lockContentStr, `\"assignees\":\"${{ github.actor }}\"`, "assignees expression should remain a runtime string in handler config")
}
