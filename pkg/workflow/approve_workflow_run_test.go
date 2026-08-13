//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApproveWorkflowRunConfiguration(t *testing.T) {
	workflowFile := filepath.Join(t.TempDir(), "approve.md")
	err := os.WriteFile(workflowFile, []byte(`---
on: issues
engine: copilot
safe-outputs:
  approve-workflow-run:
    max: 2
    staged: true
---

Approve eligible workflow runs.
`), 0o600)
	require.NoError(t, err)

	data, err := NewCompiler(WithVersion("test")).ParseWorkflowFile(workflowFile)
	require.NoError(t, err)
	require.NotNil(t, data.SafeOutputs)
	require.NotNil(t, data.SafeOutputs.ApproveWorkflowRun)
	assert.Equal(t, strPtr("2"), data.SafeOutputs.ApproveWorkflowRun.Max)
	require.NotNil(t, data.SafeOutputs.ApproveWorkflowRun.Staged)
	assert.Equal(t, TemplatableBool("true"), *data.SafeOutputs.ApproveWorkflowRun.Staged)

	enabledTools := computeEnabledToolNames(data)
	assert.Contains(t, enabledTools, "approve_workflow_run")
	assert.True(t, hasHandlerManagerTypes(data))
}

func TestApproveWorkflowRunDefaultConfiguration(t *testing.T) {
	config := NewCompiler().parseApproveWorkflowRunConfig(map[string]any{
		"approve-workflow-run": nil,
	})

	require.NotNil(t, config)
	assert.Equal(t, strPtr("1"), config.Max)
}
