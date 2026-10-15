//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthFailureClustersReuseCacheAndRejectStaleEvidence(t *testing.T) {
	root := t.TempDir()
	run := WorkflowRun{DatabaseID: 123, Status: "completed", Conclusion: "failure",
		WorkflowName: "example", WorkflowPath: "example.lock.yml",
		URL: "https://github.com/org/repo/actions/runs/123", Attempt: 1,
		UpdatedAt: time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)}
	summary := &RunSummary{CLIVersion: GetVersion(), RunID: 123, RunAnalysis: RunAnalysis{
		Run: run, JobDetails: []JobInfoWithDuration{{JobInfo: JobInfo{
			Name: "agent", Conclusion: "failure",
			Steps: []JobStep{{Name: "Start MCP Gateway", Conclusion: "failure"}},
		}}},
	}}
	require.NoError(t, os.MkdirAll(filepath.Join(root, "run-123"), 0755))
	require.NoError(t, saveRunSummary(filepath.Join(root, "run-123"), summary, false))
	config := HealthConfig{RepoOverride: "org/repo", LogsDir: root}
	clusters, plan := buildHealthFailureClusters([]WorkflowRun{run}, config)
	require.Len(t, clusters, 1)
	assert.Equal(t, "failure_boundary", clusters[0].SignalRole)
	assert.Equal(t, "Start MCP Gateway", clusters[0].Signal.Step)
	require.Len(t, plan.SelectedRuns, 1)
	assert.Equal(t, filepath.Join(root, "run-123"), plan.SelectedRuns[0].LogsPath)
	run.Attempt++
	clusters, plan = buildHealthFailureClusters([]WorkflowRun{run}, config)
	require.Len(t, clusters, 1)
	assert.Equal(t, "metadata_only", clusters[0].Evidence)
	assert.Equal(t, 1, plan.MetadataOnlyRuns)
}

func TestHealthFailureClusterOutput(t *testing.T) {
	runs := []WorkflowRun{{DatabaseID: 12, WorkflowName: "missing-cache", Conclusion: "failure"},
		{DatabaseID: 13, WorkflowName: "missing-cache", Conclusion: "success"}}
	config := HealthConfig{Days: 7, Threshold: 80, JSONOutput: true, LogsDir: t.TempDir(),
		RepoOverride: "org/repo", WorkflowName: "missing-cache"}
	for _, detailed := range []bool{false, true} {
		output := captureAuditGroupStdout(t, func() {
			if detailed {
				require.NoError(t, displayDetailedHealth(runs, config))
			} else {
				require.NoError(t, displayHealthSummary(runs, config))
			}
		})
		var data struct {
			FailureClusters      []FailureCluster      `json:"failure_clusters"`
			FailureInvestigation *FailureInvestigation `json:"failure_investigation"`
		}
		require.NoError(t, json.Unmarshal([]byte(output), &data))
		require.Len(t, data.FailureClusters, 1)
		assert.Equal(t, 1, data.FailureInvestigation.UniqueFailedRuns)
		assert.Equal(t, 1, data.FailureInvestigation.MetadataOnlyRuns)
	}
	config.JSONOutput = false
	output := captureAuditGroupStderr(t, func() {
		require.NoError(t, displayHealthSummary(runs, config))
		require.NoError(t, displayDetailedHealth(runs, config))
	})
	assert.Contains(t, output, "[failure_clusters]")
	assert.Contains(t, output, "[failure_investigation]")
}
