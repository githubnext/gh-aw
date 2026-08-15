//go:build integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApproveWorkflowRunIntegration(t *testing.T) {
	tmpDir := testutil.TempDir(t, "approve-workflow-run-integration")

	t.Run("defaults to the triggering pull request", func(t *testing.T) {
		compiled := compileSafeOutputsAppWorkflow(t, tmpDir, "approve-current-pr.md", `---
name: Approve Current Pull Request Run
on: pull_request
safe-outputs:
  approve-workflow-run:
    github-app:
      app-id: ${{ vars.APPROVE_WORKFLOW_RUN_APP_ID }}
      private-key: ${{ secrets.APPROVE_WORKFLOW_RUN_APP_PRIVATE_KEY }}
engine: copilot
---

Approve the pending workflow run for this pull request.
`)

		assert.Contains(t, compiled, `"approve_workflow_run"`)
		assert.NotContains(t, extractApproveWorkflowRunHandlerConfig(t, compiled), "allowed_pull_requests")
		step := compiledStepBlock(compiled, "approve-workflow-run-app-token")
		require.NotEmpty(t, step)
		assert.Contains(t, step, "permission-actions: write")
		assert.Contains(t, compiled, "steps.approve-workflow-run-app-token.outputs.token")
	})

	t.Run("emits configured pull request list and expression", func(t *testing.T) {
		compiled := compileSafeOutputsAppWorkflow(t, tmpDir, "approve-allowed-prs.md", `---
name: Approve Allowed Pull Request Runs
on: pull_request
safe-outputs:
  approve-workflow-run:
    github-token: ${{ secrets.APPROVE_WORKFLOW_RUN_TOKEN }}
    allowed-pull-requests:
      - "42"
      - "43"
engine: copilot
---

Approve pending workflow runs for explicitly allowed pull requests.
`)

		handlerConfig := extractApproveWorkflowRunHandlerConfig(t, compiled)
		assert.Equal(t, []any{"42", "43"}, handlerConfig["allowed_pull_requests"])

		expressionCompiled := compileSafeOutputsAppWorkflow(t, tmpDir, "approve-allowed-prs-expression.md", `---
name: Approve Expression Allowed Pull Request Runs
on: pull_request
safe-outputs:
  approve-workflow-run:
    github-token: ${{ secrets.APPROVE_WORKFLOW_RUN_TOKEN }}
    allowed-pull-requests: ${{ inputs.allowed-pull-requests }}
engine: copilot
---

Approve pending workflow runs for pull requests resolved from the workflow input.
`)

		expressionHandlerConfig := extractApproveWorkflowRunHandlerConfig(t, expressionCompiled)
		assert.Equal(t, "${{ inputs.allowed-pull-requests }}", expressionHandlerConfig["allowed_pull_requests"])
	})
}

func extractApproveWorkflowRunHandlerConfig(t *testing.T, compiled string) map[string]any {
	t.Helper()

	for line := range strings.SplitSeq(compiled, "\n") {
		if !strings.Contains(line, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			continue
		}
		parts := strings.SplitN(line, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ", 2)
		if len(parts) != 2 {
			continue
		}
		configJSON := strings.ReplaceAll(strings.Trim(strings.TrimSpace(parts[1]), "\""), "\\\"", "\"")
		var config map[string]map[string]any
		require.NoError(t, json.Unmarshal([]byte(configJSON), &config))
		require.Contains(t, config, "approve_workflow_run")
		return config["approve_workflow_run"]
	}

	t.Fatal("approve_workflow_run handler config not found")
	return nil
}
