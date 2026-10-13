//go:build !integration

package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateWorkQueueEventWorkflowWhenIssueMirroringIsEnabled(t *testing.T) {
	t.Setenv("GH_AW_ACTION_MODE", "dev")
	dir := t.TempDir()
	data := []*WorkflowData{{
		Tools: map[string]any{"work-queue": map[string]any{"issues": true}},
	}}

	require.NoError(t, GenerateWorkQueueEventWorkflow(context.Background(), data, dir, nil))
	content, err := os.ReadFile(filepath.Join(dir, workQueueEventWorkflowFilename))
	require.NoError(t, err)
	require.Contains(t, string(content), "name: Agentic Workqueue")
	require.Contains(t, string(content), "  issues:\n    types: [closed]")
	require.Contains(t, string(content), "  issue_comment:\n    types: [created]")
	require.Contains(t, string(content), "cancel_work_queue_closed_issue.cjs")
}

func TestGenerateWorkQueueEventWorkflowRemovesWhenIssueMirroringIsUnused(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, workQueueEventWorkflowFilename)
	require.NoError(t, os.WriteFile(path, []byte("stale"), 0o600))

	require.NoError(t, GenerateWorkQueueEventWorkflow(context.Background(), []*WorkflowData{{
		Tools: map[string]any{"work-queue": true},
	}}, dir, nil))
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}
