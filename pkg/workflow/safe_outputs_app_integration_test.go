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
		assert.Contains(t, compiled, "steps.report-incomplete-app-token.outputs.token")
	})

	t.Run("dispatch-repository tool app compiles dedicated token", func(t *testing.T) {
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
		assert.Contains(t, compiled, "steps.dispatch-repository-trigger_ci-app-token.outputs.token")
	})

	t.Run("close handlers wire dedicated tokens into handler config", func(t *testing.T) {
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

		assert.Contains(t, compiled, "steps.close-issue-app-token.outputs.token")
		assert.Contains(t, compiled, "steps.close-discussion-app-token.outputs.token")
	})
}

func compileSafeOutputsAppWorkflow(t *testing.T, dir, fileName, content string) string {
	t.Helper()

	mdPath := filepath.Join(dir, fileName)
	require.NoError(t, os.WriteFile(mdPath, []byte(content), 0600))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(mdPath))

	lockPath := filepath.Join(dir, strings.TrimSuffix(fileName, ".md")+".lock.yml")
	compiledBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	return string(compiledBytes)
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
