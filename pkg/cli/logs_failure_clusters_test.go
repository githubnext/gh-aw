//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailureSignalsForRun(t *testing.T) {
	t.Parallel()
	pr := ProcessedRun{
		Run: WorkflowRun{Conclusion: "failure"},
		JobDetails: []JobInfoWithDuration{
			{JobInfo: JobInfo{Name: "Agent", Conclusion: "success"}},
			{JobInfo: JobInfo{Name: "Safe Outputs", Conclusion: "failure", Steps: []JobStep{
				{Name: "Create issue", Conclusion: "failure"},
				{Name: "Cleanup", Conclusion: "success"},
			}}},
			{JobInfo: JobInfo{Name: "Setup", Conclusion: "failure"}},
		},
		MCPFailures:  []MCPFailureReport{{ServerName: "github"}, {ServerName: "github"}},
		MissingTools: []MissingToolReport{{Tool: "shell"}},
		MissingData:  []MissingDataReport{{DataType: "issue"}},
		FirewallAnalysis: &FirewallAnalysis{RequestsByDomain: map[string]DomainRequestStats{
			"blocked.example": {Blocked: 2},
			"allowed.example": {Allowed: 1},
		}},
	}
	signals := failureSignalsForRun(pr)
	assert.Equal(t, []FailureSignal{
		{Kind: "blocked_domain", Value: "blocked.example"},
		{Kind: "failed_job", Value: "setup", Job: "setup"},
		{Kind: "failed_step", Value: `"safe_outputs"/"Create issue"`, Job: "safe_outputs", Step: "Create issue"},
		{Kind: "mcp_failure", Value: "github"},
		{Kind: "missing_data", Value: "issue"},
		{Kind: "missing_tool", Value: "shell"},
	}, signals)
	pr.Run.Conclusion = "success"
	assert.Empty(t, failureSignalsForRun(pr), "successful runs are not failure clusters")
}

func failureClusterTestRuns() []RunData {
	signal := FailureSignal{Kind: "mcp_failure", Value: "github"}
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	return []RunData{
		{RunID: 1, Conclusion: "failure", FailureKind: "agent_logic", WorkflowName: "One",
			Repository: "org/repo", EngineID: "copilot", Model: "model-a", CreatedAt: now,
			FailureSignals: []FailureSignal{signal, signal}, URL: "https://github.com/org/repo/actions/runs/1"},
		{RunID: 2, Conclusion: "failure", FailureKind: "agent_logic", WorkflowName: "Two",
			Repository: "org/repo", EngineID: "claude", Model: "model-b", CreatedAt: now.Add(time.Hour),
			FailureSignals: []FailureSignal{signal, {Kind: "missing_tool", Value: "shell"}},
			LogsPath:       "/logs/2", AuditPath: "/logs/2/audit.json"},
		{RunID: 3, Conclusion: "success", FailureSignals: []FailureSignal{signal},
			FailureClusterIDs: []string{"stale"}},
		{RunID: 4, Conclusion: "in_progress"},
	}
}

func TestAnnotateFailureClusters(t *testing.T) {
	t.Parallel()
	runs := failureClusterTestRuns()
	clusters := annotateFailureClusters(runs)
	require.Len(t, clusters, 2)
	c := clusters[0]
	assert.Equal(t, 2, c.Count)
	assert.Equal(t, []int64{1, 2}, c.RunIDs)
	assert.Equal(t, []string{"one", "two"}, c.Workflows)
	assert.Equal(t, []string{"claude", "copilot"}, c.Engines)
	assert.Equal(t, []string{"model-a", "model-b"}, c.Models)
	assert.Equal(t, runs[0].CreatedAt, c.FirstSeen)
	assert.Equal(t, runs[1].CreatedAt, c.LastSeen)
	assert.Equal(t, "observed_symptom", c.Evidence)
	assert.Equal(t, int64(2), c.RepresentativeRuns[0].RunID)
	assert.Equal(t, "/logs/2/audit.json", c.RepresentativeRuns[0].AuditPath)
	assert.Equal(t, []string{c.ID}, runs[0].FailureClusterIDs)
	assert.Len(t, runs[1].FailureClusterIDs, 2)
	assert.Empty(t, runs[2].FailureClusterIDs)
	assert.Empty(t, runs[3].FailureClusterIDs)

	reversed := slices.Clone(runs)
	slices.Reverse(reversed)
	assert.Equal(t, clusters, annotateFailureClusters(reversed), "input order must not affect clusters")
	assert.Equal(t, clusters, annotateFailureClusters(runs), "annotations are idempotent")
	subset := slices.Clone(runs[:1])
	subsetClusters := annotateFailureClusters(subset)
	require.Len(t, subsetClusters, 1)
	assert.Equal(t, c.ID, subsetClusters[0].ID, "IDs must not depend on batch membership")
	assert.Equal(t, []int64{1}, subsetClusters[0].RunIDs)
}

func TestFailureClustersSeparateConclusionsKindsAndIntentionalFailures(t *testing.T) {
	t.Parallel()
	var runs []RunData
	for _, conclusion := range []string{"failure", "timed_out", "cancelled", "startup_failure"} {
		for _, kind := range []string{"agent_logic", "driver_exit", ""} {
			for _, intentional := range []bool{false, true} {
				runs = append(runs, RunData{
					RunID: int64(len(runs) + 1), Conclusion: conclusion, FailureKind: kind,
					IntentionalFailure: intentional, FailureSignals: []FailureSignal{{Kind: "failed_job", Value: "agent"}},
				})
			}
		}
	}
	clusters := annotateFailureClusters(runs)
	assert.Len(t, clusters, 24)
	for _, c := range clusters {
		assert.Equal(t, 1, c.Count)
	}
}

func TestFailureClustersFallbackAndDuplicates(t *testing.T) {
	t.Parallel()
	runs := []RunData{
		{RunID: 1, Repository: "org/a", WorkflowName: "One", Conclusion: "failure"},
		{RunID: 2, Repository: "org/a", WorkflowName: "One", Conclusion: "failure"},
		{RunID: 3, Repository: "org/b", WorkflowName: "One", Conclusion: "failure"},
		{RunID: 4, Repository: "org/a", WorkflowName: "Two", Conclusion: "failure"},
		{RunID: 1, Repository: "org/a", WorkflowName: "One", Conclusion: "failure"},
	}
	clusters := annotateFailureClusters(runs)
	require.Len(t, clusters, 3)
	assert.Equal(t, 2, clusters[0].Count)
	assert.Equal(t, []int64{1, 2}, clusters[0].RunIDs)
	for _, c := range clusters {
		assert.Equal(t, "metadata_only", c.Evidence)
		assert.Len(t, c.Repositories, 1)
	}
	assert.Empty(t, annotateFailureClusters(nil))
}

func TestFailureClustersRepresentativeLimit(t *testing.T) {
	t.Parallel()
	var runs []RunData
	for i := int64(1); i <= 10; i++ {
		runs = append(runs, RunData{RunID: i, Conclusion: "failure", WorkflowName: "test"})
	}
	clusters := annotateFailureClusters(runs)
	require.Len(t, clusters, 1)
	assert.Equal(t, 10, clusters[0].Count)
	assert.Len(t, clusters[0].RepresentativeRuns, 3)
	assert.Equal(t, int64(10), clusters[0].RepresentativeRuns[0].RunID)
}

func TestFailureClustersCachedAndSummaryReports(t *testing.T) {
	t.Parallel()
	pr := ProcessedRun{
		Run:         WorkflowRun{DatabaseID: 42, Conclusion: "failure"},
		MCPFailures: []MCPFailureReport{{ServerName: "github"}},
	}
	data := buildLogsData([]ProcessedRun{pr}, t.TempDir(), nil)
	require.Len(t, data.FailureClusters, 1)
	require.Len(t, data.Runs, 1)
	persisted, err := json.Marshal(buildCachedLogsJSONLRunData(pr, data.Runs[0]))
	require.NoError(t, err)
	var cached cachedLogsJSONLRunData
	require.NoError(t, json.Unmarshal(persisted, &cached))
	cached.FailureClusterIDs = []string{"stale"}
	fromCache := buildLogsData([]ProcessedRun{processedRunFromCachedData(cached.RunData, nil, t.TempDir())}, "", nil)
	assert.Equal(t, data.FailureClusters, fromCache.FailureClusters)
	assert.Equal(t, data.FailureInvestigation, fromCache.FailureInvestigation)
	assert.Equal(t, data.Runs[0].FailureClusterIDs, fromCache.Runs[0].FailureClusterIDs)
	var compact bytes.Buffer
	require.NoError(t, renderLogsJSONToWriter(&compact, fromCache, false))
	assert.Contains(t, compact.String(), `"failure_clusters"`)
	assert.Contains(t, compact.String(), `"failure_cluster_ids"`)
	assert.Contains(t, compact.String(), `"failure_investigation"`)
	path := filepath.Join(t.TempDir(), "summary.json")
	require.NoError(t, writeSummaryFile(path, fromCache, false))
	summary, err := os.ReadFile(path)
	require.NoError(t, err)
	var saved LogsData
	require.NoError(t, json.Unmarshal(summary, &saved))
	expectedClusters, err := json.Marshal(fromCache.FailureClusters)
	require.NoError(t, err)
	savedClusters, err := json.Marshal(saved.FailureClusters)
	require.NoError(t, err)
	assert.JSONEq(t, string(expectedClusters), string(savedClusters))
	assert.Equal(t, fromCache.Runs[0].FailureClusterIDs, saved.Runs[0].FailureClusterIDs)
	assert.Equal(t, fromCache.FailureInvestigation, saved.FailureInvestigation)
}

func failureClusterTestLogsData() LogsData {
	runs := failureClusterTestRuns()
	clusters := annotateFailureClusters(runs)
	return LogsData{Runs: runs, FailureClusters: clusters, FailureInvestigation: buildFailureInvestigation(runs, clusters)}
}

func TestFailureClusterRenderers(t *testing.T) {
	t.Parallel()
	data := failureClusterTestLogsData()
	for name, render := range map[string]func(io.Writer, LogsData){
		"compact": func(w io.Writer, d LogsData) { require.NoError(t, renderLogsCompactToWriter(w, d)) },
		"verbose": func(w io.Writer, d LogsData) { require.NoError(t, renderLogsCompactVerboseToWriter(w, d)) },
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			render(&buf, data)
			assert.Equal(t, 1, strings.Count(buf.String(), "[failure_clusters]"))
			assert.Equal(t, 1, strings.Count(buf.String(), "[failure_investigation]"))
			assert.Contains(t, buf.String(), "gh aw audit 2 --repo 'org/repo'")
			assert.Contains(t, buf.String(), "investigate=2")
			if name == "verbose" {
				assert.Contains(t, buf.String(), "runs=1,2")
			} else {
				assert.NotContains(t, buf.String(), "runs=1,2")
			}
		})
	}
	var md bytes.Buffer
	renderCrossRunReportMarkdownToWriter(&md, &CrossRunAuditReport{
		FailureClusters: data.FailureClusters, FailureInvestigation: data.FailureInvestigation,
	})
	assert.Contains(t, md.String(), "## Failure Investigation Clusters")
	assert.Contains(t, md.String(), "runs=1,2")
	assert.Contains(t, md.String(), "/logs/2/audit.json")
	require.ErrorIs(t, renderFailureClustersToWriter(failingClusterWriter{}, data.FailureClusters, ""), io.ErrClosedPipe)
	require.ErrorIs(t, renderFailureReportToWriter(failingClusterWriter{}, data.FailureClusters, data.FailureInvestigation, "", false), io.ErrClosedPipe)
}

type failingClusterWriter struct{}

func (failingClusterWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestFailureClusterRenderingEscapesSignalNewlines(t *testing.T) {
	t.Parallel()
	runs := []RunData{{RunID: 1, Conclusion: "failure", FailureSignals: []FailureSignal{
		{Kind: "missing_tool", Value: "shell\n[fake]"},
	}}}
	clusters := annotateFailureClusters(runs)
	var buf bytes.Buffer
	require.NoError(t, renderFailureClustersToWriter(&buf, clusters, "# "))
	assert.NotContains(t, buf.String(), "\n[fake]")
	require.NoError(t, renderFailureClustersToWriter(io.Discard, nil, ""))
}

func TestFailureClusterTSVOutput(t *testing.T) {
	data := failureClusterTestLogsData()
	for _, verbose := range []bool{false, true} {
		output := captureAuditGroupStdout(t, func() {
			require.NoError(t, renderLogsOutputTSV(data, verbose))
		})
		assert.Equal(t, 1, strings.Count(output, "[failure_clusters]"))
		assert.Contains(t, output, "# [failure_clusters]")
		assert.Contains(t, output, "# [failure_investigation]")
		if verbose {
			assert.Contains(t, output, "runs=1,2")
		}
	}

}

func TestFailureClusterAuditCommand(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "gh aw audit 42 --repo 'org/repo'", failureClusterAuditCommand(FailureClusterTarget{
		RunID: 42, Repository: "org/repo",
	}))
	assert.Equal(t, "gh aw audit 'https://example.test/it'\\''s;not-a-command'", failureClusterAuditCommand(FailureClusterTarget{
		URL: "https://example.test/it's;not-a-command",
	}))
}

func TestFailureClusterCrossRunOutput(t *testing.T) {
	data := failureClusterTestLogsData()
	output := captureAuditGroupStdout(t, func() {
		require.NoError(t, renderLogsOutputCrossRun(nil, data, renderLogsOutputOptions{jsonOutput: true}))
	})
	var report CrossRunAuditReport
	require.NoError(t, json.Unmarshal([]byte(output), &report))
	assert.Equal(t, data.FailureClusters, report.FailureClusters)
	assert.Equal(t, data.FailureInvestigation, report.FailureInvestigation)
	path := filepath.Join(t.TempDir(), "report.md")
	require.NoError(t, renderLogsOutputCrossRun(nil, data, renderLogsOutputOptions{
		format: "markdown", reportFile: path,
	}))
	markdown, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(markdown), "## Failure Investigation Clusters")
	assert.Contains(t, string(markdown), "runs=1,2")
}
