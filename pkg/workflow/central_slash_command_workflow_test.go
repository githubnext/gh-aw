//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestGenerateCentralSlashCommandWorkflow_GeneratesWorkflow(t *testing.T) {
	tmpDir := testutil.TempDir(t, "central-slash-workflow-test")
	t.Setenv("GH_AW_ACTION_MODE", "dev")
	originalVersion := compilerVersion
	originalIsRelease := isReleaseBuild
	t.Cleanup(func() {
		compilerVersion = originalVersion
		isReleaseBuild = originalIsRelease
	})
	SetVersion("c610c2a")
	SetIsRelease(false)

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
	lines := strings.Split(text, "\n")
	require.NotEmpty(t, lines)
	require.Contains(t, lines[0], "# gh-aw-commands: ")
	metadataJSON := strings.TrimPrefix(lines[0], "# gh-aw-commands: ")
	var metadata commandsHeaderMetadata
	require.NoError(t, json.Unmarshal([]byte(metadataJSON), &metadata))
	require.Equal(t, "v1", metadata.PayloadVersion)
	require.Equal(t, "v1", metadata.SchemaVersion)
	require.Equal(t, "dev", metadata.Compiler)
	require.Equal(t, []string{"cloclo", "triage"}, metadata.Commands)
	require.Equal(t, []string{"cloclo", "triage-issue", "triage-pr"}, metadata.Workflows)

	require.Contains(t, text, "name: \"Agentic Commands\"")
	require.NotContains(t, text, "Compiler version:")
	require.Contains(t, text, "permissions: {}")
	require.Contains(t, text, "runs-on: ubuntu-slim")
	require.Contains(t, text, "    permissions:\n      actions: write\n      contents: read")
	require.Contains(t, text, "      - name: Setup Scripts")
	require.Contains(t, text, "        uses: ./actions/setup")
	require.Contains(t, text, "          destination: ${{ runner.temp }}/gh-aw/actions")
	require.Contains(t, text, "issues:")
	require.Contains(t, text, "issue_comment:")
	require.Contains(t, text, "pull_request:")
	require.Contains(t, text, "discussion_comment:")
	require.Contains(t, text, `"triage":[{"workflow":"triage-issue","events":["issue_comment","issues"]},{"workflow":"triage-pr","events":["pull_request","pull_request_comment"]}]`)
	require.Contains(t, text, `"cloclo":[{"workflow":"cloclo","events":["discussion_comment"]}]`)
	require.Contains(t, text, `require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs')`)
	require.Contains(t, text, `setupGlobals(core, github, context, exec, io, getOctokit);`)
	require.Contains(t, text, `require('${{ runner.temp }}/gh-aw/actions/route_slash_command.cjs')`)
	require.NotContains(t, text, `const routeMap = JSON.parse(process.env.GH_AW_SLASH_ROUTING || "{}");`)
	require.NotContains(t, text, `trustedAuthorAssociations`)
	require.NotContains(t, text, `isForkBasedPullRequestEvent`)
	require.NotContains(t, text, `workflow_id: route.workflow + ".lock.yml"`)
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

func TestGenerateCentralSlashCommandWorkflow_UsesCentralizedRunsOnResolution(t *testing.T) {
	tmpDir := testutil.TempDir(t, "central-slash-workflow-runs-on-test")
	data := []*WorkflowData{
		{
			WorkflowID:         "one",
			Command:            []string{"one"},
			CommandEvents:      []string{"issue_comment"},
			CommandCentralized: true,
			RunsOnSlim:         "ubuntu-latest",
		},
		{
			WorkflowID:         "two",
			Command:            []string{"two"},
			CommandEvents:      []string{"issue_comment"},
			CommandCentralized: true,
			SafeOutputs: &SafeOutputsConfig{
				RunsOn: "self-hosted",
			},
		},
		{
			WorkflowID:         "three",
			Command:            []string{"three"},
			CommandEvents:      []string{"issue_comment"},
			CommandCentralized: true,
			SafeOutputs: &SafeOutputsConfig{
				RunsOn: "self-hosted",
			},
		},
	}

	require.NoError(t, GenerateCentralSlashCommandWorkflow(data, tmpDir))
	content, err := os.ReadFile(filepath.Join(tmpDir, centralSlashCommandWorkflowFilename))
	require.NoError(t, err)
	require.Contains(t, string(content), "runs-on: self-hosted")
}

func TestBuildCommandsHeaderMetadata_UsesReleaseVersionOnlyForReleaseBuilds(t *testing.T) {
	originalVersion := compilerVersion
	originalIsRelease := isReleaseBuild
	t.Cleanup(func() {
		compilerVersion = originalVersion
		isReleaseBuild = originalIsRelease
	})

	routesByCommand := map[string][]slashCommandRoute{
		"triage": {
			{Workflow: "triage-issue", Events: []string{"issues"}},
		},
	}

	SetVersion("abc1234")
	SetIsRelease(false)
	metadata := buildCommandsHeaderMetadata(routesByCommand)
	require.Equal(t, "dev", metadata.Compiler)

	SetVersion("v1.2.3")
	SetIsRelease(true)
	metadata = buildCommandsHeaderMetadata(routesByCommand)
	require.Equal(t, "v1.2.3", metadata.Compiler)
}

func typeSetKeys(typeSet map[string]bool) []string {
	out := make([]string, 0, len(typeSet))
	for key := range typeSet {
		out = append(out, key)
	}
	return out
}
