//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractInputsFromParsedWorkflow_TriggerAware(t *testing.T) {
	workflow := map[string]any{
		"on": map[string]any{
			"workflow_dispatch": map[string]any{
				"inputs": map[string]any{
					"environment": map[string]any{"type": "string"},
				},
			},
			"workflow_call": map[string]any{
				"inputs": map[string]any{
					"payload": map[string]any{"type": "string"},
				},
			},
		},
	}

	dispatchInputs := extractInputsFromParsedWorkflow(workflow, "workflow_dispatch")
	assert.Contains(t, dispatchInputs, "environment")
	assert.NotContains(t, dispatchInputs, "payload")

	callInputs := extractInputsFromParsedWorkflow(workflow, "workflow_call")
	assert.Contains(t, callInputs, "payload")
	assert.NotContains(t, callInputs, "environment")
}

func TestExtractInputsFromMarkdown_TriggerAware(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "worker.md")
	content := `---
on:
  workflow_dispatch:
    inputs:
      environment:
        type: string
  workflow_call:
    inputs:
      payload:
        type: string
engine: copilot
---

# Worker
`
	require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

	dispatchInputs, err := extractInputsFromMarkdown(mdPath, "workflow_dispatch")
	require.NoError(t, err)
	assert.Contains(t, dispatchInputs, "environment")
	assert.NotContains(t, dispatchInputs, "payload")

	callInputs, err := extractInputsFromMarkdown(mdPath, "workflow_call")
	require.NoError(t, err)
	assert.Contains(t, callInputs, "payload")
	assert.NotContains(t, callInputs, "environment")
}
