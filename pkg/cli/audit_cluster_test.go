//go:build !integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditClusterOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		err  string
	}{
		{"explicit", []string{"--cluster"}, ""},
		{"range enables clustering", []string{"--start-date=-1d"}, ""},
		{"end date", []string{"--end-date", "2026-10-15"}, ""},
		{"bad date", []string{"--start-date", "nope"}, "date"},
		{"reversed range", []string{"--start-date", "2026-10-15", "--end-date", "2026-10-14"}, "earlier"},
		{"group conflict", []string{"--cluster", "--group"}, "--group"},
		{"parse conflict", []string{"--cluster", "--parse"}, "--parse"},
		{"artifacts conflict", []string{"--cluster", "--artifacts", "agent"}, "--artifacts"},
		{"runtime conflict", []string{"--cluster", "--runtime", "cloud-hypervisor"}, "--runtime"},
		{"format conflict", []string{"--cluster", "--format", "markdown"}, "--format"},
		{"evals conflict", []string{"--cluster", "--evals"}, "--evals"},
		{"experiment conflict", []string{"--cluster", "--experiment", "x"}, "--experiment"},
		{"baseline conflict", []string{"--cluster", "--no-baseline"}, "--no-baseline"},
		{"stdin range conflict", []string{"--stdin", "--start-date=-1d"}, "--stdin"},
		{"count requires cluster", []string{"--count", "3", "123"}, "requires"},
		{"failed selector requires cluster", []string{"--failed-no-jobs", "123"}, "requires"},
		{"zero count", []string{"--cluster", "--count", "0"}, "1 and 1000"},
		{"large count", []string{"--cluster", "--count", "1001"}, "1 and 1000"},
		{"negative offset", []string{"--cluster", "--cluster-offset=-1"}, "negative"},
		{"search cap", []string{"--cluster", "--cluster-offset=1000"}, "narrow"},
		{"zero stalled threshold", []string{"--cluster", "--stalled-after=0"}, "greater than zero"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := NewAuditCommand()
			require.NoError(t, cmd.Flags().Parse(tt.args))
			opts, err := getAuditCommandOptions(cmd)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
			} else {
				require.NoError(t, err)
				assert.True(t, opts.cluster)
			}
		})
	}
}

func TestAuditClusterDefaultSelection(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		args    []string
		cluster bool
	}{
		{name: "single run", args: []string{"123"}},
		{name: "multiple runs", args: []string{"123", "456"}, cluster: true},
		{name: "discovery", cluster: true},
		{name: "explicit diff", args: []string{"--diff", "123", "456"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := NewAuditCommand()
			require.NoError(t, cmd.Flags().Parse(tt.args))
			opts, err := getAuditCommandOptions(cmd)
			require.NoError(t, err)
			assert.Equal(t, tt.cluster, opts.cluster)
		})
	}
}

func TestAuditClusterArgumentConflicts(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"--start-date=-1d", "123"},
		{"--cluster", "--cluster-offset=1", "123"},
		{"--cluster", "--diff", "123", "456"},
	} {
		cmd := NewAuditCommand()
		cmd.SetArgs(args)
		cmd.SilenceUsage, cmd.SilenceErrors = true, true
		if slices.Contains(args, "--diff") {
			require.ErrorContains(t, cmd.Execute(), "cannot be combined")
		} else {
			require.ErrorContains(t, cmd.Execute(), "run IDs cannot be combined")
		}
	}
}

func TestAuditClusterDateRange(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 15, 12, 30, 0, 0, time.UTC)
	start, end, err := auditClusterDateRange(auditCommandOptions{}, now)
	require.NoError(t, err)
	assert.Equal(t, now, end)
	assert.Equal(t, 24*time.Hour, end.Sub(start))
	start, end, err = auditClusterDateRange(auditCommandOptions{startDate: "-1d", endDate: "-1h"}, now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(-24*time.Hour), start)
	assert.Equal(t, now.Add(-time.Hour), end)
}

func TestAuditClusterOutcome(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	for _, outcome := range []string{"success", "failure", "cancelled", "skipped", "neutral", "timed_out", "action_required", "startup_failure", "stale"} {
		assert.Equal(t, outcome, auditClusterOutcome(WorkflowRun{Status: "completed", Conclusion: outcome}, now, time.Hour))
	}
	assert.Equal(t, "unknown", auditClusterOutcome(WorkflowRun{Status: "completed"}, now, time.Hour))
	assert.Equal(t, "unknown", auditClusterOutcome(WorkflowRun{}, now, time.Hour))
	assert.Equal(t, "active", auditClusterOutcome(WorkflowRun{Status: "queued"}, now, time.Hour))
	assert.Equal(t, "active", auditClusterOutcome(WorkflowRun{Status: "in_progress", UpdatedAt: now.Add(-59 * time.Minute)}, now, time.Hour))
	assert.Equal(t, "stalled", auditClusterOutcome(WorkflowRun{Status: "waiting", UpdatedAt: now.Add(-time.Hour)}, now, time.Hour))
	assert.Equal(t, "active", auditClusterOutcome(WorkflowRun{Status: "in_progress", UpdatedAt: now.Add(-time.Hour)}, now, 2*time.Hour))
}

func TestParseAuditClusterJobs(t *testing.T) {
	t.Parallel()
	for _, fixture := range []string{
		`[]`, `null`, `[{}]`, `[{"total_count":0,"jobs":null}]`,
		`[{"total_count":1,"jobs":[]}]`, `[{"total_count":1,"jobs":[{}]}]`,
		`[{"total_count":0,"jobs":[]},null]`, `garbage`,
	} {
		_, err := parseAuditClusterJobs([]byte(fixture))
		require.Error(t, err, fixture)
	}
	jobs, err := parseAuditClusterJobs([]byte(`[{"total_count":0,"jobs":[]}]`))
	require.NoError(t, err)
	assert.Empty(t, jobs)
	jobs, err = parseAuditClusterJobs([]byte(`[{"total_count":2,"jobs":[{"id":1,"name":"a"}]},{"total_count":2,"jobs":[{"id":2,"name":"b"}]}]`))
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
}

func TestAuditClusterSelectionDeduplicates(t *testing.T) {
	t.Parallel()
	opts := auditCommandOptions{repoFlag: "o/r", count: 2, stalledAfter: time.Hour}
	host := resolveAuditHostname("")
	requests, continuation, err := selectAuditClusterRuns(context.Background(), []string{"2", "1", "https://" + host + "/o/r/actions/runs/1", "3"}, opts, time.Now())
	require.NoError(t, err)
	require.Len(t, requests, 2)
	assert.Equal(t, int64(1), requests[0].runID)
	assert.Equal(t, int64(2), requests[1].runID)
	require.NotNil(t, continuation)
	assert.Contains(t, continuation.Arguments, "https://"+host+"/o/r/actions/runs/3")
	requests, _, err = selectAuditClusterRuns(context.Background(), []string{"https://github.com/a/r/actions/runs/1", "https://github.com/b/r/actions/runs/1"}, opts, time.Now())
	require.NoError(t, err)
	assert.Len(t, requests, 2, "run IDs are repository scoped")
	_, _, err = selectAuditClusterRuns(context.Background(), []string{"https://github.com/o/r/actions/runs/1/job/2"}, opts, time.Now())
	require.ErrorContains(t, err, "not job")
	_, _, err = selectAuditClusterRuns(context.Background(), []string{"1"}, auditCommandOptions{count: 100}, time.Now())
	require.ErrorContains(t, err, "--repo")
	_, _, err = selectAuditClusterRuns(context.Background(), []string{"1"}, auditCommandOptions{count: 100, repoFlag: "o/../r"}, time.Now())
	require.ErrorContains(t, err, "invalid repository")
	_, _, err = selectAuditClusterRuns(context.Background(), []string{"-1"}, opts, time.Now())
	require.ErrorContains(t, err, "positive")
}

func TestAuditClusterGroupsDeterministic(t *testing.T) {
	t.Parallel()
	runs := []AuditClusterRun{
		{RunID: 3, Hostname: "github.com", Repository: "o/r", Outcome: "failure", Signatures: []string{"jobs:none", "usage:unknown"}},
		{RunID: 1, Hostname: "github.com", Repository: "o/r", Outcome: "failure", Signatures: []string{"jobs:none"}},
		{RunID: 2, Hostname: "github.com", Repository: "o/r", Outcome: "success", Signatures: []string{"outcome:success"}},
	}
	first := groupAuditClusterRuns(slices.Clone(runs))
	slices.Reverse(runs)
	assert.Equal(t, first, groupAuditClusterRuns(runs))
	assert.Equal(t, int64(1), first[0].RepresentativeRun.RunID)
	assert.Equal(t, []int64{1, 3}, first[0].RunIDs)
	assert.Equal(t, 2, first[0].Count)
}

func setupAuditClusterGH(t *testing.T) (auditCommandOptions, string) {
	t.Helper()
	dir := testutil.TempDir(t, "audit-cluster-*")
	calls := filepath.Join(dir, "calls.log")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$CLUSTER_CALLS"
case "$*" in
  *"/jobs?"*)
    if [ "$CLUSTER_JOBS" = "error" ]; then echo denied >&2; exit 1; fi
    if [ "$CLUSTER_JOBS" = "slow" ]; then exec sleep 10; fi
    printf '%s\n' "$CLUSTER_JOBS"; exit 0 ;;
  *"/artifacts"*)
    if [ "$CLUSTER_USAGE" != "missing" ]; then printf '%s\n' "$CLUSTER_ARTIFACTS"; fi
    exit 0 ;;
  *"actions/runs?"*)
    case "$*" in *"page=2&"*) if [ -n "$CLUSTER_LIST_PAGE2" ]; then printf '%s\n' "$CLUSTER_LIST_PAGE2"; exit 0; fi ;; esac
    printf '%s\n' "$CLUSTER_LIST"; exit 0 ;;
  "api "*)
    if [ "$CLUSTER_METADATA" = "error" ]; then echo denied >&2; exit 1; fi
    printf '%s\n' "$CLUSTER_METADATA"; exit 0 ;;
  "run download "*)
    while [ $# -gt 0 ]; do
      case "$1" in --dir) dir="$2"; shift 2 ;; --name) name="$2"; shift 2 ;; *) shift ;; esac
    done
    mkdir -p "$dir/activity"
    if [ "$name" = "bbb-usage" ] && [ -n "$CLUSTER_USAGE_SECOND" ]; then
      if [ "$CLUSTER_USAGE_SECOND" != "absent" ]; then
        printf '%s\n' "$CLUSTER_USAGE_SECOND" > "$dir/activity/summary.json"
      fi
    else
      printf '%s\n' "$CLUSTER_USAGE" > "$dir/activity/summary.json"
    fi
    printf '%s\n' "$CLUSTER_AW_INFO" > "$dir/aw_info.json"
    if [ -n "$CLUSTER_TOKEN_USAGE" ]; then printf '%s\n' "$CLUSTER_TOKEN_USAGE" > "$dir/token-usage.jsonl"; fi
    exit 0 ;;
esac
echo "unexpected gh call" >&2
exit 1
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CLUSTER_CALLS", calls)
	t.Setenv("CLUSTER_METADATA", `{"databaseId":1,"workflowName":"test","workflowPath":".github/workflows/test.lock.yml","headSha":"abc123","event":"push","createdAt":"2026-10-15T01:00:00Z","headBranch":"main","status":"completed","conclusion":"failure","url":"https://github.com/o/r/actions/runs/1"}`)
	t.Setenv("CLUSTER_JOBS", `[{"total_count":0,"jobs":[]}]`)
	t.Setenv("CLUSTER_USAGE", `{"schema":"usage-activity-summary/v1","session":{"turns":7,"failed_tool_executions":2}}`)
	t.Setenv("CLUSTER_TOKEN_USAGE", "")
	t.Setenv("CLUSTER_ARTIFACTS", "usage")
	t.Setenv("CLUSTER_USAGE_SECOND", "")
	t.Setenv("CLUSTER_AW_INFO", "{}")
	t.Setenv("CLUSTER_LIST_PAGE2", "")
	return auditCommandOptions{cluster: true, repoFlag: "o/r", outputDir: filepath.Join(dir, "output"), count: 100, stalledAfter: time.Hour}, calls
}

func TestAuditClusterCollectZeroJobsUsage(t *testing.T) {
	opts, calls := setupAuditClusterGH(t)
	opts.failedNoJobs = true
	report, err := collectAuditClusters(context.Background(), []string{"1", "1"}, opts, time.Now())
	require.NoError(t, err)
	assert.True(t, report.Complete)
	assert.Equal(t, 1, report.RunsExamined)
	require.Len(t, report.Runs, 1)
	run := report.Runs[0]
	assert.Equal(t, ".github/workflows/test.lock.yml", run.WorkflowPath)
	assert.Equal(t, "abc123", run.HeadSHA)
	assert.Equal(t, "push", run.Event)
	assert.Equal(t, "2026-10-15T01:00:00Z", run.CreatedAt.Format(time.RFC3339))
	assert.Equal(t, "main", run.HeadBranch)
	require.NotNil(t, run.JobCount)
	assert.Zero(t, *run.JobCount)
	assert.Equal(t, "available", run.JobsState)
	assert.Equal(t, "available", run.UsageState)
	require.NotNil(t, run.Turns)
	assert.Equal(t, 7, *run.Turns)
	assert.Nil(t, run.Tokens, "absent token evidence must not become zero")
	assert.Contains(t, run.Signatures, "jobs:none")
	assert.Contains(t, run.Signatures, "usage:failed-tool-executions")
	log, err := os.ReadFile(calls)
	require.NoError(t, err)
	assert.Contains(t, string(log), "run download 1 --name usage")
	assert.NotContains(t, string(log), "/logs")
	assert.NotContains(t, string(log), "--name agent")
}

func TestAuditClusterCollectUnknownJobsStillReadsUsage(t *testing.T) {
	for _, jobs := range []string{"error", `[{}]`, `[{"total_count":1,"jobs":[]}]`} {
		t.Run(jobs, func(t *testing.T) {
			opts, calls := setupAuditClusterGH(t)
			t.Setenv("CLUSTER_JOBS", jobs)
			report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
			require.NoError(t, err)
			require.Len(t, report.Runs, 1)
			assert.Nil(t, report.Runs[0].JobCount)
			assert.Equal(t, "unknown", report.Runs[0].JobsState)
			assert.Equal(t, "available", report.Runs[0].UsageState)
			assert.NotEmpty(t, report.Errors)
			assert.False(t, report.Complete)
			opts.failedNoJobs = true
			report, err = collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
			require.NoError(t, err)
			assert.Zero(t, report.RunsSelected, "unavailable jobs are not confirmed zero")
			assert.False(t, report.Complete, "filtered populations with unavailable jobs are incomplete")
			log, err := os.ReadFile(calls)
			require.NoError(t, err)
			assert.Contains(t, string(log), "run download 1 --name usage")
		})
	}

}

func TestAuditClusterCollectTokenEvidence(t *testing.T) {
	opts, _ := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_TOKEN_USAGE", `{"timestamp":"2026-10-15T01:00:00Z","request_id":"one","provider":"anthropic","model":"claude-sonnet-4-6","status":200,"input_tokens":100,"output_tokens":20,"ai_credits_this_response":0.5}`)
	report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	require.Len(t, report.Runs, 1)
	require.NotNil(t, report.Runs[0].Tokens)
	assert.Equal(t, 120, *report.Runs[0].Tokens)
	require.NotNil(t, report.Runs[0].AIC)
	assert.InDelta(t, 0.5, *report.Runs[0].AIC, 1e-9)
	assert.Equal(t, map[string]int{"failure": 1}, report.Outcomes)
}

func TestAuditClusterUsageMeasurementsRequirePresence(t *testing.T) {
	opts, _ := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_USAGE", `{"schema":"usage-activity-summary/v1","session":{}}`)
	report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	assert.Nil(t, report.Runs[0].Turns)
	assert.Nil(t, report.Runs[0].FailedToolExecutions)
	assert.Nil(t, report.Runs[0].AIC)
	t.Setenv("CLUSTER_USAGE", `{"schema":"usage-activity-summary/v1","session":{"turns":0,"failed_tool_executions":0}}`)
	report, err = collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	require.NotNil(t, report.Runs[0].Turns)
	assert.Zero(t, *report.Runs[0].Turns)
	require.NotNil(t, report.Runs[0].FailedToolExecutions)
	assert.Zero(t, *report.Runs[0].FailedToolExecutions)
}

func TestAuditClusterCollectMissingUsage(t *testing.T) {
	for _, usage := range []string{"missing", "invalid-json"} {
		t.Run(usage, func(t *testing.T) {
			opts, _ := setupAuditClusterGH(t)
			t.Setenv("CLUSTER_USAGE", usage)
			report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
			require.NoError(t, err)
			require.Len(t, report.Runs, 1)
			assert.Equal(t, "unknown", report.Runs[0].UsageState)
			assert.False(t, report.Complete, "missing or malformed usage is an evidence coverage gap")
			assert.Nil(t, report.Runs[0].Tokens)
			assert.Nil(t, report.Runs[0].Turns)
			assert.NotEmpty(t, report.Runs[0].Evidence)
			assert.Contains(t, report.Runs[0].Signatures, "usage:unknown")
		})
	}
}

func TestAuditClusterCollectNonFailureZeroJobs(t *testing.T) {
	for _, conclusion := range []string{"success", "action_required", "cancelled", "skipped"} {
		t.Run(conclusion, func(t *testing.T) {
			opts, calls := setupAuditClusterGH(t)
			opts.failedNoJobs = true
			t.Setenv("CLUSTER_METADATA", `{"databaseId":1,"status":"completed","conclusion":"`+conclusion+`"}`)
			report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
			require.NoError(t, err)
			assert.Zero(t, report.RunsSelected)
			log, err := os.ReadFile(calls)
			require.NoError(t, err)
			assert.Contains(t, string(log), "run download 1 --name usage", "filtering must happen after evidence collection")
		})
	}
}

func TestAuditClusterDiscoveryAndContinuation(t *testing.T) {
	opts, calls := setupAuditClusterGH(t)
	opts.startDate, opts.endDate, opts.count = "2026-10-14", "2026-10-15", 1
	t.Setenv("CLUSTER_LIST", `{"total_count":2,"runs":[{"databaseId":2,"status":"completed","conclusion":"failure"},{"databaseId":1,"status":"completed","conclusion":"success"}]}`)
	requests, continuation, err := selectAuditClusterRuns(context.Background(), nil, opts, time.Now())
	require.NoError(t, err)
	require.Len(t, requests, 1)
	assert.Equal(t, int64(2), requests[0].runID)
	require.NotNil(t, requests[0].metadata)
	require.NotNil(t, continuation)
	assert.Contains(t, strings.Join(continuation.Arguments, " "), "--cluster-offset 1")
	log, err := os.ReadFile(calls)
	require.NoError(t, err)
	assert.Contains(t, string(log), "2026-10-14T00%3A00%3A00Z..2026-10-15T00%3A00%3A00Z")
	assert.NotContains(t, string(log), "status=", "action_required and runs with zero jobs must not be prefiltered")
	opts.clusterOffset = 1
	requests, continuation, err = selectAuditClusterRuns(context.Background(), nil, opts, time.Now())
	require.NoError(t, err)
	require.Len(t, requests, 1)
	assert.Equal(t, int64(1), requests[0].runID)
	assert.Nil(t, continuation)
}

func TestAuditClusterDiscoveryPagination(t *testing.T) {
	opts, calls := setupAuditClusterGH(t)
	opts.count = 101
	page := struct {
		Total int           `json:"total_count"`
		Runs  []WorkflowRun `json:"runs"`
	}{Total: 101}
	for id := int64(1); id <= 100; id++ {
		page.Runs = append(page.Runs, WorkflowRun{DatabaseID: id})
	}
	fixture, err := json.Marshal(page)
	require.NoError(t, err)
	t.Setenv("CLUSTER_LIST", string(fixture))
	t.Setenv("CLUSTER_LIST_PAGE2", `{"total_count":101,"runs":[{"databaseId":101,"status":"completed","conclusion":"action_required"}]}`)
	requests, continuation, err := selectAuditClusterRuns(context.Background(), nil, opts, time.Now())
	require.NoError(t, err)
	assert.Len(t, requests, 101)
	assert.Nil(t, continuation)
	log, err := os.ReadFile(calls)
	require.NoError(t, err)
	assert.Contains(t, string(log), "page=2")
	assert.Equal(t, "action_required", requests[100].metadata.Conclusion)
}

func TestAuditClusterDiscoveryDedupAndIncompleteResponse(t *testing.T) {
	opts, _ := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_LIST", `{"total_count":3,"runs":[{"databaseId":1},{"databaseId":1},{"databaseId":2}]}`)
	requests, continuation, err := selectAuditClusterRuns(context.Background(), nil, opts, time.Now())
	require.NoError(t, err)
	assert.Len(t, requests, 2)
	assert.Nil(t, continuation)
	t.Setenv("CLUSTER_LIST", `{"total_count":1001,"runs":[]}`)
	requests, continuation, err = selectAuditClusterRuns(context.Background(), nil, opts, time.Now())
	require.NoError(t, err)
	assert.Empty(t, requests)
	require.NotNil(t, continuation)
	assert.True(t, continuation.RequiresNarrowerRange)
	assert.Empty(t, continuation.Arguments, "do not offer a nonadvancing pagination cursor")
	t.Setenv("CLUSTER_LIST", `{}`)
	_, _, err = selectAuditClusterRuns(context.Background(), nil, opts, time.Now())
	require.ErrorContains(t, err, "incomplete run-list")
}

func TestAuditClusterGitHubSearchCapCannotResume(t *testing.T) {
	opts, _ := setupAuditClusterGH(t)
	opts.clusterOffset, opts.count = 999, 1
	page := struct {
		Total int           `json:"total_count"`
		Runs  []WorkflowRun `json:"runs"`
	}{Total: 1001}
	for id := int64(901); id <= 1000; id++ {
		page.Runs = append(page.Runs, WorkflowRun{DatabaseID: id})
	}
	fixture, err := json.Marshal(page)
	require.NoError(t, err)
	t.Setenv("CLUSTER_LIST", string(fixture))
	requests, continuation, err := selectAuditClusterRuns(context.Background(), nil, opts, time.Now())
	require.NoError(t, err)
	require.Len(t, requests, 1)
	assert.Equal(t, int64(1000), requests[0].runID)
	require.NotNil(t, continuation)
	assert.True(t, continuation.RequiresNarrowerRange)
	assert.Empty(t, continuation.Arguments)
	assert.Contains(t, continuation.Reason, "narrow the date range")
}

func TestAuditClusterPrettyOutputEscapesUntrustedNames(t *testing.T) {
	opts, _ := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_JOBS", `[{"total_count":1,"jobs":[{"id":1,"name":"\u001b[31magent\u001b[0m\nnext","conclusion":"failure"}]}]`)
	cmd := NewAuditCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	require.NoError(t, runAuditCluster(context.Background(), cmd, []string{"1"}, opts))
	assert.NotContains(t, output.String(), "\x1b")
	assert.Contains(t, output.String(), `\x1b[31magent`)
	assert.Contains(t, output.String(), `\nnext`)
}

func TestAuditClusterMissingMetadataAndFailedJobSignatures(t *testing.T) {
	opts, _ := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_METADATA", "error")
	report, err := collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	require.Len(t, report.Runs, 1)
	assert.Equal(t, "unknown", report.Runs[0].Outcome)
	assert.False(t, report.Complete)
	assert.Equal(t, "available", report.Runs[0].UsageState)
	t.Setenv("CLUSTER_METADATA", `{"databaseId":1,"status":"completed","conclusion":"failure"}`)
	t.Setenv("CLUSTER_JOBS", `[{"total_count":1,"jobs":[{"id":1,"name":"agent","conclusion":"failure","steps":[{"name":"z","conclusion":"failure"},{"name":"a","conclusion":"failure"}]}]}]`)
	report, err = collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	assert.Contains(t, report.Runs[0].Signatures, "job:agent:failure/step:a:failure/step:z:failure")
	opts.failedNoJobs = true
	report, err = collectAuditClusters(context.Background(), []string{"1"}, opts, time.Now())
	require.NoError(t, err)
	assert.Zero(t, report.RunsSelected)
}

func TestAuditClusterCancelledContext(t *testing.T) {
	opts, calls := setupAuditClusterGH(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, err := collectAuditClusters(ctx, []string{"1"}, opts, time.Now())
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, report.Complete)
	assert.Zero(t, report.RunsExamined)
	require.NotNil(t, report.Continuation)
	assert.Contains(t, report.Continuation.Arguments, "1")
	_, statErr := os.Stat(calls)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestAuditClusterDeadlineDuringCollection(t *testing.T) {
	opts, _ := setupAuditClusterGH(t)
	t.Setenv("CLUSTER_JOBS", "slow")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	report, err := collectAuditClusters(ctx, []string{"1", "2"}, opts, time.Now())
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 2*time.Second)
	assert.False(t, report.Complete)
	require.NotNil(t, report.Continuation)
	assert.Contains(t, report.Continuation.Arguments, "1")
	assert.Contains(t, report.Continuation.Arguments, "2")
}

func TestAuditClusterCommandJSON(t *testing.T) {
	_, _ = setupAuditClusterGH(t)
	cmd := NewAuditCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--repo", "o/r", "--json", "--output", testutil.TempDir(t, "audit-cluster-command-*"), "1", "2"})
	require.NoError(t, cmd.Execute())
	var report AuditClusterReport
	require.NoError(t, json.Unmarshal(output.Bytes(), &report))
	assert.Equal(t, "audit-clusters/v1", report.Schema)
	assert.Equal(t, 2, report.RunsSelected)
}

func TestAuditDefaultsToDiscoveryClusteringWithoutRunIDs(t *testing.T) {
	_, _ = setupAuditClusterGH(t)
	t.Setenv("CLUSTER_LIST", `{"total_count":1,"runs":[{"databaseId":1,"status":"completed","conclusion":"failure"}]}`)
	cmd := NewAuditCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--repo", "o/r", "--json", "--output", testutil.TempDir(t, "audit-cluster-discovery-*")})
	require.NoError(t, cmd.Execute())
	var report AuditClusterReport
	require.NoError(t, json.Unmarshal(output.Bytes(), &report))
	assert.Equal(t, "audit-clusters/v1", report.Schema)
	assert.Equal(t, 1, report.RunsSelected)
}
