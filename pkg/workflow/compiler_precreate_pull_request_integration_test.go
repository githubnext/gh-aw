//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

func TestCompilePreCreatePullRequest(t *testing.T) {
	tmpDir := testutil.TempDir(t, "pre-create-pull-request")
	workflowPath := filepath.Join(tmpDir, "pre-create.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(`---
on: workflow_dispatch
engine: copilot
strict: false
safe-outputs:
  create-pull-request:
    pre-create: true
---

# Pre-create test

Create a change and open a pull request.
`), 0o644))

	compiler := NewCompiler(WithVersion("dev"))
	compiler.SetActionMode(ActionModeDev)
	require.NoError(t, compiler.CompileWorkflow(workflowPath))

	compiled, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
	require.NoError(t, err)
	yaml := string(compiled)

	activation := extractJobSection(yaml, "activation")
	agent := extractJobSection(yaml, "agent")
	safeOutputs := extractJobSection(yaml, "safe_outputs")
	conclusion := extractJobSection(yaml, "conclusion")

	assert.Contains(t, activation, "id: pre-create-pull-request")
	assert.Contains(t, activation, "contents: write")
	assert.Contains(t, activation, "pull-requests: write")
	assert.Contains(t, activation, "checks: write")
	assert.Contains(t, agent, "ref: ${{ needs.activation.outputs.pre_created_pull_request_branch }}")
	assert.Contains(t, safeOutputs, "ref: ${{ needs.activation.outputs.pre_created_pull_request_branch }}")
	assert.Contains(t, safeOutputs, "pre_created_pull_request_number")
	assert.Contains(t, conclusion, "complete_pre_created_check_run.cjs")
}
