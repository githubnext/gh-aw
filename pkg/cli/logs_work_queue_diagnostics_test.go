//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workqueue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkQueueCapturedDiagnostics(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(currentWorkQueueFixture(t))
	require.NoError(t, err)
	envelope, state, _, err := replayCurrentWorkQueueSnapshot(data)
	require.NoError(t, err)
	before, err := json.Marshal(state)
	require.NoError(t, err)
	snapshot, err := parseWorkQueueSnapshot(data)
	require.NoError(t, err)
	diagnostics := snapshot.Current.Diagnostics
	require.NotNil(t, diagnostics)
	assert.Equal(t, "captured_snapshot", diagnostics.Status)
	assert.Equal(t, int64(8), diagnostics.EvaluatedAt)
	assert.Equal(t, "owner/repo", diagnostics.Repository)
	assert.Empty(t, diagnostics.Branch, "artifacts do not establish a queue branch name")
	assert.Equal(t, 1, diagnostics.Available)
	assert.Equal(t, 1, diagnostics.Completed)
	assert.Equal(t, 1, diagnostics.DeliveryByState["pending"])
	assert.Zero(t, diagnostics.DeliveryByState["verified"])
	assert.Equal(t, 1, diagnostics.NativeReservations)
	require.Len(t, diagnostics.Pools, 1)
	pool := diagnostics.Pools[0]
	assert.Equal(t, 1, pool.IneligibleReasons["retry_delayed"])
	require.NotNil(t, pool.OldestAvailable)
	assert.Equal(t, int64(2), pool.OldestAvailable.AdmittedAt)
	assert.Equal(t, int64(6), pool.OldestAvailable.AgeMS)
	assert.Equal(t, int64(8), diagnostics.Activity.Until)
	require.Len(t, diagnostics.AssignedClaims, 2)
	first, second := diagnostics.AssignedClaims[0], diagnostics.AssignedClaims[1]
	assert.Equal(t, envelope.Worker.Claims[0].ClaimID, first.Claim.ClaimID)
	assert.Equal(t, int64(3), *first.Claim.GrantedAt)
	assert.Equal(t, int64(6), *first.Claim.TerminalAt)
	assert.Equal(t, 1, first.Work.Attempts)
	assert.Equal(t, "pending", first.Claim.Delivery)
	assert.Equal(t, "cancelled", second.Claim.State)
	assert.Equal(t, int64(30007), second.Claim.RetryNotBefore)
	assert.False(t, second.Claim.Current)
	assertWorkDiagnosticsPrivate(t, diagnostics)
	serialized, err := json.Marshal(snapshot)
	require.NoError(t, err)
	assert.NotContains(t, string(serialized), "private-task-content")
	assert.NotContains(t, string(serialized), `"principal"`)
	_, err = capturedWorkQueueDiagnostics(state, envelope)
	require.NoError(t, err)
	after, err := json.Marshal(state)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "predictions must not modify captured authority")
}

func TestWorkQueueCapturedDeliveryAndCheckpoint(t *testing.T) {
	t.Parallel()
	state, commits, ids := workDiagnosticsFixture(t)
	var dispatch *workqueue.DispatchState
	for _, candidate := range state.Dispatches {
		dispatch = candidate
	}
	origin := workqueue.Actor{Role: "worker", Principal: "1001", Repository: "owner/repo",
		RunID: "202", RunAttempt: 1, Workflow: dispatch.Profile.Workflow, DispatchID: dispatch.DispatchID}
	checkpoint, err := workqueue.CompactCheckpoint(commits, strings.Repeat("a", 40),
		workqueue.Actor{Role: "administrator", Principal: origin.Principal, Repository: origin.Repository}, 450000)
	require.NoError(t, err)
	for name, ledger := range map[string][]workqueue.QueueCommit{"original": commits, "checkpoint": checkpoint} {
		t.Run(name, func(t *testing.T) {
			log, err := workqueue.Serialize(ledger)
			require.NoError(t, err)
			data, err := json.Marshal(map[string]any{
				"version": 3, "sha": "snapshot-sha", "worker": dispatch.Assignment,
				"transactionLog": string(log), "captured_at": 500000, "origin": origin,
			})
			require.NoError(t, err)
			snapshot, err := parseWorkQueueSnapshot(data)
			require.NoError(t, err)
			diagnostics := snapshot.Current.Diagnostics
			assert.Equal(t, 1, diagnostics.DeliveryByState["verified"])
			assert.Equal(t, 1, diagnostics.DeliveryByState["failed"])
			assert.Equal(t, 1, diagnostics.DeliveryFailureReasons["verification_exhausted"])
			require.Len(t, diagnostics.AssignedClaims, 3)
			failed := diagnostics.AssignedClaims[1]
			assert.Equal(t, ids[1], failed.Work.WorkID)
			assert.Equal(t, int64(2000), *failed.Work.AdmittedAt)
			assert.Equal(t, "verification_exhausted", failed.Claim.DeliveryFailureReason)
			assert.Equal(t, "unknown", failed.Claim.Disposition)
			assert.Equal(t, int64(3000), *failed.Claim.GrantedAt)
			assert.Equal(t, 1, diagnostics.Activity.Operations["DeliveryFailure"])
			assertWorkDiagnosticsPrivate(t, snapshot)
			var output bytes.Buffer
			require.NoError(t, renderWorkQueueToWriter(&output, &WorkQueueReport{Snapshot: snapshot}))
			assert.Contains(t, output.String(), "failed=1")
			assert.Contains(t, output.String(), "verification_exhausted=1")
			assert.Contains(t, output.String(), "unknown effects must not be replayed")
			assert.Contains(t, output.String(), "failure=verification_exhausted disposition=unknown")
		})
	}
}

func TestWorkQueueCapturedDiagnosticsCacheAndLogs(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(currentWorkQueueFixture(t))
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, workQueueSnapshotFile), data, 0o600))
	report, err := extractWorkQueueReport(dir)
	require.NoError(t, err)
	current := *report.Snapshot.Current
	current.Diagnostics = nil
	cached := &WorkQueueReport{Snapshot: &WorkQueueSnapshot{Version: 3, SHA: report.Snapshot.SHA, Current: &current}}
	require.True(t, backfillWorkQueueReport(&cached, dir), "old queue summaries need diagnostic backfill")
	require.NotNil(t, cached.Snapshot.Current.Diagnostics)
	assert.False(t, backfillWorkQueueReport(&cached, dir), "identical diagnostic values must not repeatedly invalidate caches")
	run := WorkflowRun{DatabaseID: 42, Status: "completed", Conclusion: "success", LogsPath: dir}
	summary := &RunSummary{CLIVersion: GetVersion(), RunID: 42, RunAnalysis: RunAnalysis{Run: run, WorkQueue: cached}}
	require.NoError(t, saveRunSummary(dir, summary, false))
	loaded, ok := loadRunSummary(dir, false)
	require.True(t, ok)
	processed := processedRunFromSummary(loaded, dir)
	audit, _ := buildLocalAuditData(processed, LogMetrics{}, nil)
	audit.CacheSource = auditCacheSourceFull
	require.NoError(t, writeAuditData(dir, audit))
	restored, ok := loadCachedAuditData(dir, run, auditCacheSourceFull)
	require.True(t, ok)
	assert.True(t, equalWorkQueueSnapshots(cached.Snapshot, restored.WorkQueue.Snapshot))
	runData := newRunData(processed, runEngineInfo{}, SafeOutputChainMetrics{}, nil, "", 0)
	assert.True(t, equalWorkQueueSnapshots(cached.Snapshot, runData.WorkQueue.Snapshot))
	assert.Nil(t, workQueueReportRuns([]RunData{runData}, false)[0].WorkQueue, "usage-only logs must still omit queue evidence")
	var output bytes.Buffer
	renderLogsCompactToWriter(&output, LogsData{Runs: []RunData{runData}})
	assert.Contains(t, output.String(), "delivery: pending=1 verified=0 failed=0")
	assert.Contains(t, output.String(), "current=false work_attempts=1 claim_retry_not_before=30007")
	assert.Contains(t, output.String(), "activation snapshot; no live reads")
	assert.Contains(t, output.String(), "finish intents and later operation logs do not update this snapshot")
}

func TestWorkQueueCapturedDiagnosticsEquality(t *testing.T) {
	t.Parallel()
	fixture := currentWorkQueueFixture(t)
	data, err := json.Marshal(fixture)
	require.NoError(t, err)
	snapshot, err := parseWorkQueueSnapshot(data)
	require.NoError(t, err)
	encoded, err := json.Marshal(snapshot.Current.Diagnostics)
	require.NoError(t, err)
	for name, change := range map[string]func(*WorkQueueDiagnostics){
		"counts":       func(d *WorkQueueDiagnostics) { d.Completed++ },
		"evaluation":   func(d *WorkQueueDiagnostics) { d.EvaluatedAt++ },
		"delivery":     func(d *WorkQueueDiagnostics) { d.DeliveryByState["pending"]++ },
		"failure":      func(d *WorkQueueDiagnostics) { d.DeliveryFailureReasons["verification_exhausted"]++ },
		"nil reasons":  func(d *WorkQueueDiagnostics) { d.DeliveryFailureReasons = nil },
		"activity":     func(d *WorkQueueDiagnostics) { d.Activity.Operations["Completion"]++ },
		"last time":    func(d *WorkQueueDiagnostics) { *d.Activity.LastAt++ },
		"pool":         func(d *WorkQueueDiagnostics) { d.Pools[0].IneligibleReasons["retry_delayed"]++ },
		"oldest":       func(d *WorkQueueDiagnostics) { d.Pools[0].OldestAvailable.AgeMS++ },
		"prediction":   func(d *WorkQueueDiagnostics) { d.Pools[0].Prediction.Reason = "other" },
		"assignments":  func(d *WorkQueueDiagnostics) { d.Pools[0].Prediction.Assignments = nil },
		"claim":        func(d *WorkQueueDiagnostics) { d.AssignedClaims[0].Claim.Delivery = "verified" },
		"grant time":   func(d *WorkQueueDiagnostics) { *d.AssignedClaims[0].Claim.GrantedAt++ },
		"retry limit":  func(d *WorkQueueDiagnostics) { *d.AssignedClaims[0].Work.MaxAttempts++ },
		"admission":    func(d *WorkQueueDiagnostics) { *d.AssignedClaims[0].Work.AdmittedAt++ },
		"dependencies": func(d *WorkQueueDiagnostics) { d.AssignedClaims[0].Work.DependsOn = nil },
	} {
		t.Run(name, func(t *testing.T) {
			var copy WorkQueueDiagnostics
			require.NoError(t, json.Unmarshal(encoded, &copy))
			require.True(t, equalWorkQueueDiagnostics(snapshot.Current.Diagnostics, &copy))
			change(&copy)
			require.False(t, equalWorkQueueDiagnostics(snapshot.Current.Diagnostics, &copy))
			current := *snapshot.Current
			current.Diagnostics = &copy
			cached := &WorkQueueReport{Snapshot: snapshot}
			extracted := &WorkQueueReport{Snapshot: &WorkQueueSnapshot{Version: 3, SHA: snapshot.SHA, Current: &current}}
			assert.True(t, mergeWorkQueueReport(&cached, extracted))
			assert.False(t, mergeWorkQueueReport(&cached, extracted))
		})
	}
}

func TestWorkQueueCapturedDiagnosticsRenderingBounds(t *testing.T) {
	t.Parallel()
	fixture := currentWorkQueueFixture(t)
	data, err := json.Marshal(fixture)
	require.NoError(t, err)
	snapshot, err := parseWorkQueueSnapshot(data)
	require.NoError(t, err)
	diagnostics := snapshot.Current.Diagnostics
	pool, claim := diagnostics.Pools[0], diagnostics.AssignedClaims[0]
	pool.Pool = "unsafe\n\x1b[31m"
	pool.IneligibleReasons = map[string]int{"z": 2, "a": 1}
	diagnostics.Pools, diagnostics.AssignedClaims = nil, nil
	for range 17 {
		diagnostics.Pools = append(diagnostics.Pools, pool)
		diagnostics.AssignedClaims = append(diagnostics.AssignedClaims, claim)
	}
	text := workQueueDiagnosticsText(diagnostics)
	assert.NotContains(t, text, "unsafe\n")
	assert.NotContains(t, text, "\x1b")
	assert.Contains(t, text, "blocked=a=1,z=2")
	assert.Equal(t, 16, strings.Count(text, "    pool: "))
	assert.Equal(t, 16, strings.Count(text, "attempts=1"))
	assert.Contains(t, text, "additional pools available in JSON")
	assert.Contains(t, text, "additional original Claims available in JSON")
	assert.Empty(t, workQueueDiagnosticsText(nil))
}
