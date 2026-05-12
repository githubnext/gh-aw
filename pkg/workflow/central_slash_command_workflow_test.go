//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestGenerateCentralSlashCommandWorkflow_GeneratesWorkflow(t *testing.T) {
	tmpDir := testutil.TempDir(t, "central-slash-workflow-test")

	data := []*WorkflowData{
		{
			WorkflowID:         "triage",
			Command:            []string{"triage"},
			CommandEvents:      []string{"issue_comment"},
			CommandCentralized: true,
		},
		{
			WorkflowID:         "review",
			Command:            []string{"triage"},
			CommandEvents:      []string{"pull_request_comment"},
			CommandCentralized: true,
		},
	}

	require.NoError(t, GenerateCentralSlashCommandWorkflow(data, tmpDir))

	generatedPath := filepath.Join(tmpDir, centralSlashCommandWorkflowFilename)
	content, err := os.ReadFile(generatedPath)
	require.NoError(t, err)
	text := string(content)

	require.Contains(t, text, "name: \"Agentic Slash Command Trigger\"")
	require.Contains(t, text, "issue_comment:")
	require.Contains(t, text, `"triage":[{"workflow":"review","events":["pull_request_comment"]},{"workflow":"triage","events":["issue_comment"]}]`)
	require.Contains(t, text, `workflow_id: route.workflow + ".lock.yml"`)
}

func TestGenerateCentralSlashCommandWorkflow_DeletesWhenUnused(t *testing.T) {
	tmpDir := testutil.TempDir(t, "central-slash-workflow-delete-test")
	generatedPath := filepath.Join(tmpDir, centralSlashCommandWorkflowFilename)
	require.NoError(t, os.WriteFile(generatedPath, []byte("stale"), 0644))

	data := []*WorkflowData{
		{
			WorkflowID:         "regular",
			Command:            []string{"regular"},
			CommandEvents:      []string{"issue_comment"},
			CommandCentralized: false,
		},
	}

	require.NoError(t, GenerateCentralSlashCommandWorkflow(data, tmpDir))
	_, err := os.Stat(generatedPath)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}
