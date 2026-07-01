//go:build integration

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	goyaml "github.com/goccy/go-yaml"
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

	var compiledWorkflow map[string]any
	require.NoError(t, goyaml.Unmarshal(lockContent, &compiledWorkflow), "lock file should be valid YAML")

	handlerConfigJSON := extractHandlerConfigJSON(t, compiledWorkflow)

	var handlerConfig map[string]map[string]any
	require.NoError(t, json.Unmarshal([]byte(handlerConfigJSON), &handlerConfig), "handler config should be valid JSON")

	createPullRequestConfig, ok := handlerConfig["create_pull_request"]
	require.True(t, ok, "handler config should include create_pull_request")
	assert.Equal(t, "${{ github.actor }}", createPullRequestConfig["reviewers"], "reviewers expression should remain a runtime string in handler config")
	// Handler config normalizes frontmatter keys like team-reviewers to team_reviewers
	// to match JSON property naming conventions.
	assert.Equal(t, "${{ inputs.review-team }}", createPullRequestConfig["team_reviewers"], "team_reviewers expression should remain a runtime string in handler config")
	assert.Equal(t, "${{ github.actor }}", createPullRequestConfig["assignees"], "assignees expression should remain a runtime string in handler config")
}

func extractHandlerConfigJSON(t *testing.T, compiledWorkflow map[string]any) string {
	t.Helper()

	jobs, ok := compiledWorkflow["jobs"].(map[string]any)
	require.True(t, ok, "compiled workflow should include jobs map")

	safeOutputsJob, ok := jobs["safe_outputs"].(map[string]any)
	require.True(t, ok, "compiled workflow should include safe_outputs job")

	steps, ok := safeOutputsJob["steps"].([]any)
	require.True(t, ok, "safe_outputs job should include steps")

	for _, step := range steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		env, ok := stepMap["env"].(map[string]any)
		if !ok {
			continue
		}
		handlerConfig, ok := env["GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG"].(string)
		if ok {
			return handlerConfig
		}
	}

	t.Fatal("failed to locate GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG in safe_outputs job steps")
	return ""
}
