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
			WorkflowID:         "triage-issue",
			Command:            []string{"triage"},
			CommandEvents:      []string{"issue_comment", "issues"},
			CommandCentralized: true,
		},
		{
			WorkflowID:         "triage-pr",
			Command:            []string{"triage"},
			CommandEvents:      []string{"pull_request", "pull_request_comment"},
			CommandCentralized: true,
		},
		{
			WorkflowID:         "cloclo",
			Command:            []string{"cloclo"},
			CommandEvents:      []string{"discussion_comment"},
			CommandCentralized: true,
		},
	}

	require.NoError(t, GenerateCentralSlashCommandWorkflow(data, tmpDir))

	generatedPath := filepath.Join(tmpDir, centralSlashCommandWorkflowFilename)
	content, err := os.ReadFile(generatedPath)
	require.NoError(t, err)
	text := string(content)

	require.Contains(t, text, "name: \"Agentic Slash Command Trigger\"")
	require.Contains(t, text, "permissions: {}")
	require.Contains(t, text, "    permissions:\n      actions: write\n      contents: read")
	require.Contains(t, text, "issues:")
	require.Contains(t, text, "issue_comment:")
	require.Contains(t, text, "pull_request:")
	require.Contains(t, text, "discussion_comment:")
	require.Contains(t, text, `"triage":[{"workflow":"triage-issue","events":["issue_comment","issues"]},{"workflow":"triage-pr","events":["pull_request","pull_request_comment"]}]`)
	require.Contains(t, text, `"cloclo":[{"workflow":"cloclo","events":["discussion_comment"]}]`)
	require.Contains(t, text, `const routes = (routeMap[commandName] ?? []).filter(route => Array.isArray(route.events) && route.events.includes(identifier));`)
	require.Contains(t, text, `const trustedAuthorAssociations = new Set(["MEMBER", "OWNER", "COLLABORATOR"]);`)
	require.Contains(t, text, `if (!trustedAuthorAssociations.has(authorAssociation)) {`)
	require.Contains(t, text, `async function isForkBasedPullRequestEvent() {`)
	require.Contains(t, text, `if (await isForkBasedPullRequestEvent()) {`)
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

func TestRemoveIfExists(t *testing.T) {
	tmpDir := testutil.TempDir(t, "remove-if-exists-test")
	existingPath := filepath.Join(tmpDir, "existing.txt")
	missingPath := filepath.Join(tmpDir, "missing.txt")

	require.NoError(t, os.WriteFile(existingPath, []byte("content"), 0644))
	require.NoError(t, removeIfExists(existingPath))
	_, err := os.Stat(existingPath)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	require.NoError(t, removeIfExists(missingPath))
}

func TestCollectCentralSlashCommandRoutes_UnionizesMergedEvents(t *testing.T) {
	data := []*WorkflowData{
		{
			WorkflowID:         "triage-issue",
			Command:            []string{"triage"},
			CommandEvents:      []string{"issues", "issue_comment"},
			CommandCentralized: true,
		},
		{
			WorkflowID:         "triage-pr",
			Command:            []string{"triage"},
			CommandEvents:      []string{"pull_request", "pull_request_comment"},
			CommandCentralized: true,
		},
		{
			WorkflowID:         "non-centralized",
			Command:            []string{"triage"},
			CommandEvents:      []string{"discussion"},
			CommandCentralized: false,
		},
	}

	routesByCommand, mergedEvents := collectCentralSlashCommandRoutes(data)

	require.Equal(t, []slashCommandRoute{
		{Workflow: "triage-issue", Events: []string{"issue_comment", "issues"}},
		{Workflow: "triage-pr", Events: []string{"pull_request", "pull_request_comment"}},
	}, routesByCommand["triage"])

	require.ElementsMatch(t, []string{"opened", "edited", "reopened"}, typeSetKeys(mergedEvents["issues"]))
	require.ElementsMatch(t, []string{"created", "edited"}, typeSetKeys(mergedEvents["issue_comment"]))
	require.ElementsMatch(t, []string{"opened", "edited", "reopened"}, typeSetKeys(mergedEvents["pull_request"]))
	require.NotContains(t, mergedEvents, "discussion")
}

func typeSetKeys(typeSet map[string]bool) []string {
	out := make([]string, 0, len(typeSet))
	for key := range typeSet {
		out = append(out, key)
	}
	return out
}
