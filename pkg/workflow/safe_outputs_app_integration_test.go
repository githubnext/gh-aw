//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeOutputsGitHubAppIntegrationMatrix(t *testing.T) {
	tmpDir := testutil.TempDir(t, "safe-outputs-app-integration")

	t.Run("global app excludes handler override permissions", func(t *testing.T) {
		compiled := compileSafeOutputsAppWorkflow(t, tmpDir, "global-handler-split.md", `---
name: Global And Handler Apps
on:
  issues:
    types: [opened]
safe-outputs:
  github-app:
    app-id: ${{ vars.GLOBAL_APP_ID }}
    private-key: ${{ secrets.GLOBAL_APP_PRIVATE_KEY }}
  add-comment:
    github-app:
      app-id: ${{ vars.ISSUE_APP_ID }}
      private-key: ${{ secrets.ISSUE_APP_PRIVATE_KEY }}
    pull-requests: false
  report-incomplete:
    github-app:
      app-id: ${{ vars.INCOMPLETE_APP_ID }}
      private-key: ${{ secrets.INCOMPLETE_APP_PRIVATE_KEY }}
  dispatch-repository:
    trigger-ci:
      workflow: ci.yml
      event_type: ci_trigger
      repository: github/gh-aw
engine: copilot
---

Test workflow.
`)

		globalStep := compiledLastStepBlock(compiled, "safe-outputs-app-token")
		require.NotEmpty(t, globalStep)
		assert.Contains(t, globalStep, "permission-contents: write")
		assert.NotContains(t, globalStep, "permission-issues: write")
		assert.NotContains(t, globalStep, "permission-pull-requests: write")

		addCommentStep := compiledStepBlock(compiled, "add-comment-app-token")
		require.NotEmpty(t, addCommentStep)
		assert.Contains(t, addCommentStep, "permission-issues: write")
		assert.NotContains(t, addCommentStep, "permission-contents: write")
	})

	t.Run("report-incomplete app compiles dedicated issue token", func(t *testing.T) {
		compiled := compileSafeOutputsAppWorkflow(t, tmpDir, "report-incomplete-app.md", `---
name: Report Incomplete App
on:
  issues:
    types: [opened]
safe-outputs:
  report-incomplete:
    github-app:
      app-id: ${{ vars.INCOMPLETE_APP_ID }}
      private-key: ${{ secrets.INCOMPLETE_APP_PRIVATE_KEY }}
engine: copilot
---

Test workflow.
`)

		step := compiledStepBlock(compiled, "report-incomplete-app-token")
		require.NotEmpty(t, step)
		assert.Contains(t, step, "permission-issues: write")
		assert.NotContains(t, step, "permission-contents: write")
		assert.Contains(t, compiled, "steps.report-incomplete-app-token.outputs.token")
	})

	t.Run("dispatch-workflow app compiles dedicated actions token", func(t *testing.T) {
		ensureWorkflowFixture(t, tmpDir, "downstream.yml", `name: Downstream
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok
`)

		compiled := compileSafeOutputsAppWorkflow(t, tmpDir, "dispatch-workflow-app.md", `---
name: Dispatch Workflow App
on:
  issues:
    types: [opened]
safe-outputs:
  dispatch-workflow:
    github-app:
      app-id: ${{ vars.DISPATCH_WORKFLOW_APP_ID }}
      private-key: ${{ secrets.DISPATCH_WORKFLOW_APP_PRIVATE_KEY }}
    workflows:
      - downstream
engine: copilot
---

Test workflow.
`)

		step := compiledStepBlock(compiled, "dispatch-workflow-app-token")
		require.NotEmpty(t, step)
		assert.Contains(t, step, "permission-actions: write")
		assert.NotContains(t, step, "permission-contents: write")
		assert.NotContains(t, step, "permission-issues: write")
		assert.Contains(t, compiled, "steps.dispatch-workflow-app-token.outputs.token")
	})

	t.Run("dispatch-repository tool app compiles dedicated contents token", func(t *testing.T) {
		compiled := compileSafeOutputsAppWorkflow(t, tmpDir, "dispatch-repository-app.md", `---
name: Dispatch Repository App
on:
  issues:
    types: [opened]
safe-outputs:
  dispatch-repository:
    trigger-ci:
      workflow: ci.yml
      event_type: ci_trigger
      repository: github/gh-aw
      github-app:
        app-id: ${{ vars.DISPATCH_APP_ID }}
        private-key: ${{ secrets.DISPATCH_APP_PRIVATE_KEY }}
engine: copilot
---

Test workflow.
`)

		step := compiledStepBlock(compiled, "dispatch-repository-trigger_ci-app-token")
		require.NotEmpty(t, step)
		assert.Contains(t, step, "permission-contents: write")
		assert.NotContains(t, step, "permission-actions: write")
		assert.NotContains(t, step, "permission-issues: write")
		assert.Contains(t, compiled, "steps.dispatch-repository-trigger_ci-app-token.outputs.token")
	})

	t.Run("close handlers wire dedicated tokens into handler config with scoped permissions", func(t *testing.T) {
		compiled := compileSafeOutputsAppWorkflow(t, tmpDir, "close-handlers-app.md", `---
name: Close Handlers App
on:
  issues:
    types: [opened]
safe-outputs:
  close-issue:
    github-app:
      app-id: ${{ vars.CLOSE_ISSUE_APP_ID }}
      private-key: ${{ secrets.CLOSE_ISSUE_APP_PRIVATE_KEY }}
  close-discussion:
    github-app:
      app-id: ${{ vars.CLOSE_DISCUSSION_APP_ID }}
      private-key: ${{ secrets.CLOSE_DISCUSSION_APP_PRIVATE_KEY }}
engine: copilot
---

Test workflow.
`)

		closeIssueStep := compiledStepBlock(compiled, "close-issue-app-token")
		require.NotEmpty(t, closeIssueStep)
		assert.Contains(t, closeIssueStep, "permission-issues: write")
		assert.NotContains(t, closeIssueStep, "permission-discussions: write")
		assert.Contains(t, compiled, "steps.close-issue-app-token.outputs.token")

		closeDiscussionStep := compiledStepBlock(compiled, "close-discussion-app-token")
		require.NotEmpty(t, closeDiscussionStep)
		assert.Contains(t, closeDiscussionStep, "permission-discussions: write")
		assert.NotContains(t, closeDiscussionStep, "permission-issues: write")
		assert.Contains(t, compiled, "steps.close-discussion-app-token.outputs.token")
	})
}

func compileSafeOutputsAppWorkflow(t *testing.T, dir, fileName, content string) string {
	t.Helper()

	awDir := filepath.Join(dir, ".github", "aw")
	require.NoError(t, os.MkdirAll(awDir, 0o755))

	mdPath := filepath.Join(awDir, fileName)
	require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdPath))

	lockPath := filepath.Join(awDir, strings.TrimSuffix(fileName, ".md")+".lock.yml")
	compiledBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	return string(compiledBytes)
}

func ensureWorkflowFixture(t *testing.T, dir, fileName, content string) {
	t.Helper()

	workflowsDir := filepath.Join(dir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, fileName), []byte(content), 0600))
}

func compiledStepBlock(compiled, stepID string) string {
	marker := "id: " + stepID
	start := strings.Index(compiled, marker)
	if start == -1 {
		return ""
	}
	rest := compiled[start:]
	next := strings.Index(rest[len(marker):], "\n      - name: ")
	if next == -1 {
		return rest
	}
	return rest[:len(marker)+next]
}

func compiledLastStepBlock(compiled, stepID string) string {
	marker := "id: " + stepID
	start := strings.LastIndex(compiled, marker)
	if start == -1 {
		return ""
	}
	rest := compiled[start:]
	next := strings.Index(rest[len(marker):], "\n      - name: ")
	if next == -1 {
		return rest
	}
	return rest[:len(marker)+next]
}
