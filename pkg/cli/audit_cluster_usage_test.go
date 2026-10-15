//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditClusterRerunRejectsStaleUsageAndRefreshes(t *testing.T) {
	opts, calls := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_METADATA", `{"databaseId":1,"attempt":2,"status":"completed","conclusion":"failure"}`)
	cache := resolveAuditOutputDir(filepath.Join(opts.outputDir, "clusters", resolveAuditHostname(""), "o", "r"), 1)
	cache = filepath.Join(cache, "attempt-2")
	writeAuditClusterUsageFile(t, cache, "usage/activity/summary.json", os.Getenv("CLUSTER_USAGE"))
	writeAuditClusterUsageFile(t, cache, "usage/aw_info.json", `{"run_id":1,"run_attempt":"1"}`)
	require.NoError(t, markArtifactDownloaded(cache, "usage"))
	t.Setenv("CLUSTER_AW_INFO", `{"run_id":1,"run_attempt":"1"}`)
	t.Setenv("CLUSTER_TOKEN_USAGE", `{"request_id":"one","provider":"anthropic","model":"claude-sonnet-4-6","input_tokens":100,"output_tokens":20,"ai_credits_this_response":0.5}`)
	report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	require.Len(t, report.Runs, 1)
	run := report.Runs[0]
	assert.False(t, report.Complete)
	assert.Equal(t, "unknown", run.UsageState)
	assert.Nil(t, run.Turns)
	assert.Nil(t, run.FailedToolExecutions)
	assert.Nil(t, run.Tokens)
	assert.Nil(t, run.AIC)
	assert.NotContains(t, run.Signatures, "usage:failed-tool-executions")
	assert.Contains(t, strings.Join(run.Evidence, " "), "does not match run 1 attempt 2")
	t.Setenv("CLUSTER_AW_INFO", `{"run_id":1,"run_attempt":"2"}`)
	t.Setenv("CLUSTER_USAGE", `{"schema":"usage-activity-summary/v1","session":{"turns":9,"failed_tool_executions":0}}`)
	report, err = collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	assert.True(t, report.Complete, "%v", report.Errors)
	require.NotNil(t, report.Runs[0].Turns)
	assert.Equal(t, 9, *report.Runs[0].Turns)
	require.NotNil(t, report.Runs[0].Tokens)
	assert.Equal(t, 120, *report.Runs[0].Tokens)
	log, err := os.ReadFile(calls)
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(log), "run download 1 --name usage"))
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(cache), ".audit-cluster-usage-*"))
	require.NoError(t, err)
	assert.Empty(t, leftovers, "rerun downloads must always be removed")
}

func TestAuditClusterRerunRequiresArtifactLocalProvenance(t *testing.T) {
	for _, info := range []string{`{}`, `{"run_id":1}`, `{"run_id":2,"run_attempt":"2"}`, `{"run_id":1,"run_attempt":2}`, `broken`} {
		t.Run(info, func(t *testing.T) {
			opts, _ := setupAuditClusterGH(t)
			t.Setenv("CLUSTER_METADATA", `{"databaseId":1,"attempt":2,"status":"completed"}`)
			t.Setenv("CLUSTER_AW_INFO", info)
			report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
			require.NoError(t, err)
			assert.False(t, report.Complete)
			assert.Equal(t, "unknown", report.Runs[0].UsageState)
			assert.Nil(t, report.Runs[0].Turns)
		})
	}
}

func TestAuditClusterFirstAttemptRefreshesArtifactSet(t *testing.T) {
	opts, calls := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_METADATA", `{"databaseId":1,"attempt":1,"status":"in_progress"}`)
	t.Setenv("CLUSTER_ARTIFACTS", "aaa-usage")
	report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	assert.True(t, report.Complete, "%v", report.Errors)
	require.NotNil(t, report.Runs[0].Turns)
	assert.Equal(t, 7, *report.Runs[0].Turns)
	t.Setenv("CLUSTER_ARTIFACTS", "aaa-usage\nbbb-usage")
	t.Setenv("CLUSTER_USAGE_SECOND", `{"schema":"usage-activity-summary/v1","session":{"turns":5}}`)
	report, err = collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	assert.True(t, report.Complete, "%v", report.Errors)
	require.NotNil(t, report.Runs[0].Turns)
	assert.Equal(t, 12, *report.Runs[0].Turns)
	log, err := os.ReadFile(calls)
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(log), "run download 1 --name aaa-usage"))
	assert.Equal(t, 1, strings.Count(string(log), "run download 1 --name bbb-usage"))
	root := resolveAuditOutputDir(filepath.Join(opts.outputDir, "clusters", resolveAuditHostname(""), "o", "r"), 1)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	assert.Empty(t, entries, "all first-attempt download staging directories must be removed")
}

func TestAuditClusterUnsupportedPrefixMakesMixedEvidenceIncomplete(t *testing.T) {
	opts, _ := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_ARTIFACTS", "aaa-usage\nbbb-usage")
	t.Setenv("CLUSTER_USAGE_SECOND", "absent")
	report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	assert.False(t, report.Complete)
	assert.Equal(t, "available", report.Runs[0].UsageState)
	require.NotNil(t, report.Runs[0].Turns)
	assert.Equal(t, 7, *report.Runs[0].Turns)
	assert.Contains(t, strings.Join(report.Errors, " "), "bbb-usage: no supported usage summary or token records found")
}

func TestAuditClusterMixedAttemptUsageArtifacts(t *testing.T) {
	dir := t.TempDir()
	for _, artifact := range []string{"aaa-usage", "bbb-usage"} {
		writeAuditClusterUsageFile(t, dir, artifact+"/activity/summary.json", `{"schema":"usage-activity-summary/v1","session":{"turns":4,"failed_tool_executions":1}}`)
	}
	writeAuditClusterUsageFile(t, dir, "aaa-usage/aw_info.json", `{"run_id":1,"run_attempt":"2"}`)
	// A run-root copy must not provide provenance for a different artifact.
	writeAuditClusterUsageFile(t, dir, "aw_info.json", `{"run_id":1,"run_attempt":"2"}`)
	request := auditClusterRequest{
		groupedAuditRunRequest: groupedAuditRunRequest{runID: 1},
		metadata:               &WorkflowRun{Attempt: 2},
	}
	result := AuditClusterRun{UsageState: "unknown"}
	collectAuditClusterUsageArtifacts(dir, request, false, &result)
	assert.Equal(t, "available", result.UsageState)
	require.NotNil(t, result.Turns)
	assert.Equal(t, 4, *result.Turns)
	assert.Contains(t, strings.Join(result.Evidence, " "), "bbb-usage: usage attempt provenance unavailable")
	writeAuditClusterUsageFile(t, dir, "bbb-usage/aw_info.json", `{"run_id":1,"run_attempt":"1"}`)
	result = AuditClusterRun{UsageState: "unknown"}
	collectAuditClusterUsageArtifacts(dir, request, false, &result)
	require.NotNil(t, result.Turns)
	assert.Equal(t, 4, *result.Turns)
	assert.Contains(t, strings.Join(result.Evidence, " "), "bbb-usage: usage provenance does not match")
}

func TestAuditClusterAggregatesPrefixedUsage(t *testing.T) {
	opts, _ := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_ARTIFACTS", "bbb-usage\naaa-usage")
	t.Setenv("CLUSTER_USAGE", `{"schema":"usage-activity-summary/v1","session":{"turns":3,"failed_tool_executions":1},"gateway":{"tools":[{"server_name":"github","tool_name":"issue_read","failed_calls":1}]}}`)
	t.Setenv("CLUSTER_USAGE_SECOND", `{"schema":"usage-activity-summary/v1","session":{"turns":5,"failed_tool_executions":2},"gateway":{"tools":[{"server_name":"github","tool_name":"issue_write","failed_calls":2}]}}`)
	t.Setenv("CLUSTER_TOKEN_USAGE", `{"request_id":"one","provider":"anthropic","model":"claude-sonnet-4-6","input_tokens":100,"output_tokens":20,"ai_credits_this_response":0.5}`)
	report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	assert.True(t, report.Complete, "%v", report.Errors)
	run := report.Runs[0]
	require.NotNil(t, run.Turns)
	assert.Equal(t, 8, *run.Turns)
	require.NotNil(t, run.FailedToolExecutions)
	assert.Equal(t, 3, *run.FailedToolExecutions)
	require.NotNil(t, run.Tokens)
	assert.Equal(t, 240, *run.Tokens)
	require.NotNil(t, run.AIC)
	assert.InDelta(t, 1.0, *run.AIC, 1e-9)
	assert.Contains(t, run.Signatures, "usage:tool:github/issue_read:failure")
	assert.Contains(t, run.Signatures, "usage:tool:github/issue_write:failure")
	assert.Contains(t, run.Signatures, "usage:failed-tool-executions")
}

func TestAuditClusterMalformedPrefixedSummaryWithTokens(t *testing.T) {
	for _, summary := range []string{`bad`, `{"schema":"other"}`, `{"schema":"usage-activity-summary/v1","session":{"turns":"bad"}}`} {
		t.Run(summary, func(t *testing.T) {
			opts, _ := setupAuditClusterGH(t)
			t.Setenv("CLUSTER_ARTIFACTS", "aaa-usage\nbbb-usage")
			t.Setenv("CLUSTER_USAGE_SECOND", summary)
			t.Setenv("CLUSTER_TOKEN_USAGE", `{"request_id":"one","provider":"anthropic","model":"claude-sonnet-4-6","input_tokens":100,"output_tokens":20}`)
			report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
			require.NoError(t, err)
			assert.False(t, report.Complete)
			assert.Equal(t, "available", report.Runs[0].UsageState)
			assert.NotNil(t, report.Runs[0].Tokens)
			assert.Contains(t, strings.Join(report.Errors, " "), "bbb-usage usage activity:")
		})
	}
}

func TestAuditClusterPrefixedMeasurementsAndSupportedDirectories(t *testing.T) {
	for _, session := range []string{`{}`, `{"turns":0,"failed_tool_executions":0}`} {
		t.Run(session, func(t *testing.T) {
			dir := t.TempDir()
			writeAuditClusterUsageFile(t, dir, "aaa-usage/activity/summary.json", `{"schema":"usage-activity-summary/v1","session":`+session+`}`)
			writeAuditClusterUsageFile(t, dir, "not-usage-summary/activity/summary.json", `{"schema":"usage-activity-summary/v1","session":{"turns":99}}`)
			writeAuditClusterUsageFile(t, dir, "other/nested-usage/activity/summary.json", `{"schema":"usage-activity-summary/v1","session":{"turns":99}}`)
			result := AuditClusterRun{UsageState: "unknown"}
			collectAuditClusterUsageArtifacts(dir, auditClusterRequest{}, false, &result)
			assert.Empty(t, result.Evidence)
			if session == `{}` {
				assert.Nil(t, result.Turns)
				assert.Nil(t, result.FailedToolExecutions)
			} else {
				require.NotNil(t, result.Turns)
				require.NotNil(t, result.FailedToolExecutions)
				assert.Zero(t, *result.Turns)
				assert.Zero(t, *result.FailedToolExecutions)
			}
		})
	}
}

func writeAuditClusterUsageFile(t *testing.T, dir, name, contents string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
}
