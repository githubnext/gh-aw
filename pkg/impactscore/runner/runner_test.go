//go:build !integration

package runner

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/github/gh-aw/pkg/impactscore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareConfigDefaultsRepoFromCurrentCheckout(t *testing.T) {
	withCurrentRepository(t, func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "repo"}, nil
	})
	withOutputParentDir(t, func(path string, perm os.FileMode) error {
		assert.Equal(t, filepath.Join(os.TempDir(), "gh-aw", "impact-score"), path)
		assert.Equal(t, os.FileMode(0o755), perm)
		return nil
	})
	withTempOutputDir(t, func(dir, pattern string) (string, error) {
		assert.Equal(t, filepath.Join(os.TempDir(), "gh-aw", "impact-score"), dir)
		assert.Equal(t, "owner-repo-*", pattern)
		return filepath.Join(os.TempDir(), "gh-aw", "impact-score", "owner-repo-test"), nil
	})
	cfg := config{}

	require.NoError(t, prepareConfig(&cfg))

	assert.Equal(t, "owner/repo", cfg.Repo)
	assert.Equal(t, filepath.Join(os.TempDir(), "gh-aw", "impact-score", "owner-repo-test"), cfg.OutDir)
	assert.Equal(t, "text", cfg.ReportFormat)
}

func TestPrepareConfigAllowsAWJSONImpactPolicyPath(t *testing.T) {
	withCurrentRepository(t, func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "repo"}, nil
	})
	withOutputParentDir(t, func(string, os.FileMode) error { return nil })
	withTempOutputDir(t, func(string, string) (string, error) {
		return filepath.Join(os.TempDir(), "gh-aw", "impact-score", "owner-repo-test"), nil
	})
	t.Chdir(t.TempDir())
	require.NoError(t, os.MkdirAll(filepath.Dir(impactPolicyPath), 0o755))
	require.NoError(t, os.WriteFile(impactPolicyPath, []byte(`{"impact":{"version":1,"rules":[{"name":"security","when":{"any_label":["security"]},"min":7}]}}`), 0o644))
	cfg := config{}

	require.NoError(t, prepareConfig(&cfg))

	hasPolicy, err := hasImpactPolicy(impactPolicyPath)
	require.NoError(t, err)
	assert.True(t, hasPolicy)
}

func TestPrepareConfigErrorsWhenCurrentRepoUnavailable(t *testing.T) {
	withCurrentRepository(t, func() (repository.Repository, error) {
		return repository.Repository{}, errors.New("not a repo")
	})
	cfg := config{}

	err := prepareConfig(&cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "current repository could not be determined")
}

func TestGitHubClientRetriesTransientServerErrors(t *testing.T) {
	previousWait := waitBeforeGitHubAPIRetry
	waitBeforeGitHubAPIRetry = func(context.Context, time.Duration) error { return nil }
	t.Cleanup(func() { waitBeforeGitHubAPIRetry = previousWait })

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		attempts++
		if attempts == 1 {
			responseWriter.WriteHeader(http.StatusBadGateway)
			return
		}
		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{"name":"ok"}`))
	}))
	t.Cleanup(server.Close)

	client := githubClient{client: server.Client()}
	var payload map[string]string

	err := client.getJSON(context.Background(), server.URL, &payload)

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, "ok", payload["name"])
}

func withCurrentRepository(t *testing.T, fn func() (repository.Repository, error)) {
	t.Helper()
	previous := currentRepository
	currentRepository = fn
	t.Cleanup(func() { currentRepository = previous })
}

func withTempOutputDir(t *testing.T, fn func(string, string) (string, error)) {
	t.Helper()
	previous := createTempOutputDir
	createTempOutputDir = fn
	t.Cleanup(func() { createTempOutputDir = previous })
}

func withOutputParentDir(t *testing.T, fn func(string, os.FileMode) error) {
	t.Helper()
	previous := createOutputParentDir
	createOutputParentDir = fn
	t.Cleanup(func() { createOutputParentDir = previous })
}

func TestWorkflowDefinitionFromText(t *testing.T) {
	workflow := workflowDefinitionFromText("fallback-name.lock.yml", ".github/workflows/fallback-name.lock.yml", ".github/workflows/fallback-name.md", "name: Remote Workflow\ntitle-prefix: '[remote] '")

	assert.Equal(t, "Remote Workflow", workflow.Name)
	assert.Contains(t, workflow.Aliases, "fallback-name")
	assert.Equal(t, ".github/workflows/fallback-name.lock.yml", workflow.Path)
	assert.Equal(t, ".github/workflows/fallback-name.md", workflow.SourcePath)
	assert.Equal(t, "[remote]", workflow.TitlePrefix)
}

func TestWorkflowDefinitionFromTextFallsBackToLockName(t *testing.T) {
	workflow := workflowDefinitionFromText("fallback-name.lock.yml", ".github/workflows/fallback-name.lock.yml", ".github/workflows/fallback-name.md", "")

	assert.Equal(t, "fallback-name", workflow.Name)
	assert.Contains(t, workflow.Aliases, "fallback-name")
	assert.Equal(t, ".github/workflows/fallback-name.lock.yml", workflow.Path)
	assert.Equal(t, ".github/workflows/fallback-name.md", workflow.SourcePath)
}

func TestIsAgenticWorkflowLockFileOnlyAllowsGeneratedLocks(t *testing.T) {
	assert.True(t, isAgenticWorkflowLockFile("triage.lock.yml"))
	assert.True(t, isAgenticWorkflowLockFile("triage.lock.yaml"))
	assert.False(t, isAgenticWorkflowLockFile("triage.yml"))
	assert.False(t, isAgenticWorkflowLockFile("triage.yaml"))
	assert.False(t, isAgenticWorkflowLockFile("triage.md"))
}

func TestSourceWorkflowsCanonicalizesWorkflowAliases(t *testing.T) {
	workflows := []impactscore.WorkflowDefinition{{Name: "Network Isolation Test", Aliases: []string{"network-isolation-test"}}}

	sources := sourceWorkflows("[network-isolation-test] Network Isolation Test Results", "gh-aw-agentic-workflow: network-isolation-test", workflows)

	assert.Equal(t, []string{"Network Isolation Test"}, sources)
}

func TestCanonicalizeSourceDataWorkflowsUsesCostRunAliases(t *testing.T) {
	data := canonicalizeSourceDataWorkflows(sourceData{
		Items:     []impactscore.WorkItem{{Number: 1, Type: "issue", SourceWorkflows: []string{"dependency-security-monitor"}}},
		Workflows: []impactscore.WorkflowDefinition{{Name: "dependency-security-monitor", Path: ".github/workflows/dependency-security-monitor.lock.yml", SourcePath: ".github/workflows/dependency-security-monitor.md"}},
		CostRuns:  []impactscore.WorkflowCostRun{{Workflow: "Dependency Security Monitor", AICCost: 3}},
	})

	require.Len(t, data.Workflows, 1)
	assert.Equal(t, "Dependency Security Monitor", data.Workflows[0].Name)
	assert.Contains(t, data.Workflows[0].Aliases, "dependency-security-monitor")
	require.Len(t, data.Items, 1)
	assert.Equal(t, []string{"Dependency Security Monitor"}, data.Items[0].SourceWorkflows)
	require.Len(t, data.CostRuns, 1)
	assert.Equal(t, "Dependency Security Monitor", data.CostRuns[0].Workflow)
}

func TestCanonicalizeSourceDataWorkflowsInfersCooccurringAliases(t *testing.T) {
	data := canonicalizeSourceDataWorkflows(sourceData{
		Items: []impactscore.WorkItem{
			{Number: 1, Type: "issue", SourceWorkflows: []string{"Refactoring Opportunity Scanner", "refactoring-scanner"}},
			{Number: 2, Type: "issue", SourceWorkflows: []string{"Refactoring Opportunity Scanner", "refactoring-scanner"}},
		},
		Workflows: []impactscore.WorkflowDefinition{{Name: "refactoring-scanner", Path: ".github/workflows/refactoring-scanner.lock.yml", SourcePath: ".github/workflows/refactoring-scanner.md"}},
	})

	require.Len(t, data.Workflows, 1)
	assert.Equal(t, "Refactoring Opportunity Scanner", data.Workflows[0].Name)
	assert.Contains(t, data.Workflows[0].Aliases, "refactoring-scanner")
	assert.Equal(t, []string{"Refactoring Opportunity Scanner"}, data.Items[0].SourceWorkflows)
	assert.Equal(t, []string{"Refactoring Opportunity Scanner"}, data.Items[1].SourceWorkflows)
}

func TestWorkflowSourcePathMapsGeneratedLockToMarkdown(t *testing.T) {
	available := map[string]bool{".github/workflows/triage.md": true, ".github/workflows/triage.lock.yml": true, ".github/workflows/triage.yml": true}

	assert.Equal(t, ".github/workflows/triage.md", workflowSourcePath(".github/workflows/triage.lock.yml", available))
	assert.Equal(t, ".github/workflows/triage.yml", workflowSourcePath(".github/workflows/triage.yml", available))
	assert.True(t, isGeneratedWorkflowLock(".github/workflows/triage.lock.yml"))
}

func TestIsAgenticWorkflowPathOnlyCountsGeneratedLocks(t *testing.T) {
	assert.True(t, isAgenticWorkflowPath(".github/workflows/triage.lock.yml"))
	assert.True(t, isAgenticWorkflowPath(".github/workflows/triage.lock.yaml"))
	assert.False(t, isAgenticWorkflowPath(".github/workflows/triage.yml"))
	assert.False(t, isAgenticWorkflowPath(".github/workflows/triage.yaml"))
	assert.False(t, isAgenticWorkflowPath("action.yml"))
}

func TestWorkflowRunReferenceUsesSourceMarkdownPath(t *testing.T) {
	reference := workflowRunReferenceFromRun(githubWorkflowRun{
		Name:    "Triage",
		Path:    ".github/workflows/triage.lock.yml",
		HTMLURL: "https://github.com/owner/repo/actions/runs/123",
	}, "", []impactscore.WorkflowDefinition{{Name: "Triage", Path: ".github/workflows/triage.md", SourcePath: ".github/workflows/triage.md"}})

	assert.Equal(t, "Triage", reference.Name)
	assert.Equal(t, ".github/workflows/triage.md", reference.SourcePath)
	assert.Equal(t, "https://github.com/owner/repo/actions/runs/123", reference.RunURL)
}

func TestParseGHAWLogsCostRuns(t *testing.T) {
	runs, err := parseGHAWLogsCostRuns([]byte(`{
  "runs": [
    {
      "workflow_name": "triage",
      "run_id": 123,
      "url": "https://github.com/owner/repo/actions/runs/123",
      "aic": 2.5,
      "token_usage": 3000,
      "turns": 4,
      "action_minutes": 1.25,
      "error_count": 1
    },
    {"workflow_name": ""}
  ]
}`))

	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, "triage", runs[0].Workflow)
	assert.Equal(t, "123", runs[0].RunID)
	assert.Equal(t, "https://github.com/owner/repo/actions/runs/123", runs[0].RunURL)
	assert.InDelta(t, 2.5, runs[0].AICCost, 0.001)
	assert.InDelta(t, 3000.0, runs[0].TokenUsage, 0.001)
	assert.InDelta(t, 4.0, runs[0].Turns, 0.001)
	assert.InDelta(t, 1.25, runs[0].ActionMinutes, 0.001)
	assert.InDelta(t, 1.0, runs[0].Errors, 0.001)
	assert.Equal(t, "gh aw logs", runs[0].Source)
}

func TestParseIssueTextAICCostRuns(t *testing.T) {
	runs := parseIssueTextAICCostRuns(`> Generated from [Network Isolation Test](https://github.com/github/gh-aw-firewall/actions/runs/28112808921) · 4.08 AIC · details`, nil, "issue comment AIC")

	require.Len(t, runs, 1)
	assert.Equal(t, "Network Isolation Test", runs[0].Workflow)
	assert.Equal(t, "28112808921", runs[0].RunID)
	assert.Equal(t, "https://github.com/github/gh-aw-firewall/actions/runs/28112808921", runs[0].RunURL)
	assert.InDelta(t, 4.08, runs[0].AICCost, 0.001)
	assert.Equal(t, "issue comment AIC", runs[0].Source)
}

func TestParseIssueTextAICCostRunsUsesSingleSourceWorkflowFallback(t *testing.T) {
	runs := parseIssueTextAICCostRuns(`Agent job failed after spending 2.5 AIC.`, []string{"Security Guard"}, "issue body AIC")

	require.Len(t, runs, 1)
	assert.Equal(t, "Security Guard", runs[0].Workflow)
	assert.InDelta(t, 2.5, runs[0].AICCost, 0.001)
}

func TestNormalizeItemPreservesIssueStateReason(t *testing.T) {
	item, err := normalizeItem(context.Background(), githubClient{}, "owner", "repo", githubIssue{Number: 1, Title: "won't do", State: "closed", StateReason: "not_planned"}, nil, newWorkflowMatcher(nil), &issueCommentCache{values: map[int][]githubComment{}}, &workflowRunCache{values: map[string]workflowRunReference{}})

	require.NoError(t, err)
	assert.Equal(t, "closed", item.State)
	assert.Equal(t, "not_planned", item.StateReason)
	assert.Equal(t, []string{"not_planned"}, item.Dimensions[impactscore.DimensionStateReason])
}

func TestPullRequestStateReason(t *testing.T) {
	assert.Equal(t, "merged", pullRequestStateReason("closed", githubPullRequest{Merged: true}))
	assert.Equal(t, "merged", pullRequestStateReason("closed", githubPullRequest{MergedAt: "2026-06-25T00:00:00Z"}))
	assert.Equal(t, "closed_unmerged", pullRequestStateReason("closed", githubPullRequest{}))
	assert.Equal(t, "draft", pullRequestStateReason("open", githubPullRequest{Draft: true}))
	assert.Empty(t, pullRequestStateReason("open", githubPullRequest{}))
}

func TestRunOnceReusesOutArtifactsAndWritesCSV(t *testing.T) {
	outDir := t.TempDir()
	t.Chdir(outDir)
	require.NoError(t, writeJSON(filepath.Join(outDir, "items.json"), []impactscore.WorkItem{{
		Repo:                  "owner/repo",
		Number:                1,
		Type:                  "issue",
		State:                 "open",
		Title:                 "security issue",
		Labels:                []string{"security"},
		Dimensions:            map[string][]string{},
		Measures:              map[string]float64{},
		Released:              true,
		ReleaseNoteImportance: 5,
		SourceWorkflows:       []string{"triage"},
	}}))
	require.NoError(t, writeJSON(filepath.Join(outDir, "workflows.json"), []impactscore.WorkflowDefinition{{Name: "triage"}}))
	require.NoError(t, writeJSON(filepath.Join(outDir, "cost_runs.json"), []impactscore.WorkflowCostRun{{Workflow: "triage", RunID: "1", AICCost: 2, Source: "test"}}))
	require.NoError(t, os.WriteFile(filepath.Join(outDir, "impact_score_dashboard.html"), []byte("stale\n"), 0o644))

	result, err := runOnce(context.Background(), config{Repo: "owner/repo", OutDir: outDir})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Len(t, result.WorkflowRanks, 1)
	assert.Equal(t, "triage", result.WorkflowRanks[0].Workflow)
	assert.Greater(t, result.WorkflowRanks[0].AttributedImpactScore, 0.0)
	assertCSVContains(t, filepath.Join(outDir, "items.csv"), "security issue")
	assertCSVContains(t, filepath.Join(outDir, "items.csv"), "state_reason")
	assertCSVContains(t, filepath.Join(outDir, "workflow_ranks.csv"), "triage")
	assertCSVContains(t, filepath.Join(outDir, "cost_runs.csv"), "test")
	assertCSVContains(t, filepath.Join(outDir, "impact_score_report.txt"), "Impact Score Report")
	assertCSVContains(t, filepath.Join(outDir, "impact_score_report.txt"), "agentic workflow: triage")
	assert.NoFileExists(t, filepath.Join(outDir, "impact_score_dashboard.html"))
}

func TestRenderTextReportShowsUnlinkedAgenticWorkflowItems(t *testing.T) {
	report := string(renderTextReport(output{ItemRanks: []impactscore.ItemRank{{Number: 1, ItemType: "pr", State: "open", Title: "human fix", ImpactScore: 5, ScoreSource: "aw.json:bug work"}}}))

	assert.Contains(t, report, "no linked agentic workflow")
}

func TestWriteArtifactsCanRenderHTMLReport(t *testing.T) {
	outDir := t.TempDir()
	result := output{
		Repo:        "owner/repo",
		GeneratedAt: "2026-06-25T00:00:00Z",
		WorkflowRanks: []impactscore.WorkflowRank{{
			Workflow:              "triage",
			ActionZone:            "keep / scale",
			AttributedImpactScore: 5,
			LinkedItems:           1,
			TotalAICCost:          2,
		}},
	}

	require.NoError(t, writeArtifacts(outDir, result, "html"))

	assert.FileExists(t, filepath.Join(outDir, "impact_score_dashboard.html"))
	assert.NoFileExists(t, filepath.Join(outDir, "impact_score_report.txt"))
}

func assertCSVContains(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, strings.ReplaceAll(string(data), "\r\n", "\n"), expected)
}
