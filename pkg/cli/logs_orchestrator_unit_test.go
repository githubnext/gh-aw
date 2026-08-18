//go:build !integration

package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsDeadlineExceeded verifies that the helper correctly identifies
// context.DeadlineExceeded and returns false for other cases (including nil error).
func TestIsDeadlineExceeded(t *testing.T) {
	t.Run("deadline exceeded context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		time.Sleep(time.Millisecond) // ensure deadline has fired
		assert.True(t, isDeadlineExceeded(ctx), "expected true for DeadlineExceeded context")
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.False(t, isDeadlineExceeded(ctx), "expected false for cancelled (not deadline) context")
	})

	t.Run("active context", func(t *testing.T) {
		ctx := context.Background()
		assert.False(t, isDeadlineExceeded(ctx), "expected false for active (non-cancelled) context")
	})
}

func TestBuildLogsDownloadContextPrefersSecondTimeout(t *testing.T) {
	before := time.Now()
	ctx, cancel, startTime, timeoutDuration := buildLogsDownloadContext(context.Background(), 5, 55, false)
	defer cancel()

	require.False(t, startTime.IsZero(), "timeout context should record a start time")
	assert.Equal(t, 55*time.Second, timeoutDuration)
	deadline, ok := ctx.Deadline()
	require.True(t, ok, "timeout context should have a deadline")

	wantMin := before.Add(50 * time.Second)
	wantMax := before.Add(60 * time.Second)
	assert.True(t, deadline.After(wantMin) && deadline.Before(wantMax),
		"deadline should use timeoutSeconds instead of timeoutMinutes; got %v from %v", deadline.Sub(before), before)
}

func TestBuildLogsDownloadContextRequiresPositiveMinuteTimeout(t *testing.T) {
	tests := []struct {
		name           string
		timeoutMinutes int
		timeoutSeconds int
	}{
		{name: "zero minutes without seconds", timeoutMinutes: 0, timeoutSeconds: 0},
		{name: "zero minutes ignores seconds", timeoutMinutes: 0, timeoutSeconds: 55},
		{name: "negative minutes ignores seconds", timeoutMinutes: -1, timeoutSeconds: 55},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel, startTime, timeoutDuration := buildLogsDownloadContext(context.Background(), tt.timeoutMinutes, tt.timeoutSeconds, false)

			assert.Nil(t, cancel)
			assert.True(t, startTime.IsZero(), "non-positive minute timeout should disable timeout even when seconds are set")
			assert.Zero(t, timeoutDuration)
			_, ok := ctx.Deadline()
			assert.False(t, ok, "non-positive minute timeout should not create a deadline even when seconds are set")
		})
	}
}

// TestNoRunsMessage verifies that the helper returns an informative message
// depending on the start_date filter and timeoutReached flag.
func TestNoRunsMessage(t *testing.T) {
	now := time.Now()
	futureDate := now.AddDate(0, 0, 5).Format("2006-01-02")
	oldDate := now.AddDate(0, 0, -100).Format("2006-01-02")
	recentDate := now.AddDate(0, 0, -5).Format("2006-01-02")
	futureRFC3339 := now.AddDate(1, 0, 0).Format(time.RFC3339)

	tests := []struct {
		name           string
		startDate      string
		timeoutReached bool
		wantContains   string
	}{
		{
			name:           "timeout reached",
			startDate:      "",
			timeoutReached: true,
			wantContains:   "Timeout reached",
		},
		{
			name:           "future date (YYYY-MM-DD)",
			startDate:      futureDate,
			timeoutReached: false,
			wantContains:   "is in the future",
		},
		{
			name:           "future date (RFC3339)",
			startDate:      futureRFC3339,
			timeoutReached: false,
			wantContains:   "is in the future",
		},
		{
			name:           "old date beyond retention",
			startDate:      oldDate,
			timeoutReached: false,
			wantContains:   "retention period",
		},
		{
			name:           "recent date within retention",
			startDate:      recentDate,
			timeoutReached: false,
			wantContains:   "No runs found matching",
		},
		{
			name:           "no start date",
			startDate:      "",
			timeoutReached: false,
			wantContains:   "No runs found matching",
		},
		{
			name:           "timeout takes priority over future date",
			startDate:      futureDate,
			timeoutReached: true,
			wantContains:   "Timeout reached",
		},
		{
			name:           "future date message includes the date value",
			startDate:      "2030-01-01",
			timeoutReached: false,
			wantContains:   "2030-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := noRunsMessage(tt.startDate, tt.timeoutReached)
			assert.Contains(t, got, tt.wantContains,
				"noRunsMessage(%q, %v) = %q, want to contain %q", tt.startDate, tt.timeoutReached, got, tt.wantContains)
		})
	}
}

// TestParseFilterDate verifies that date strings accepted by the logs flags are
// correctly parsed into time.Time values.
func TestParseFilterDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"YYYY-MM-DD", "2024-01-15", false},
		{"RFC3339", "2024-01-15T10:30:00Z", false},
		{"RFC3339 with offset", "2024-01-15T10:30:00+05:00", false},
		{"invalid", "not-a-date", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFilterDate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.False(t, got.IsZero(), "expected non-zero time")
			}
		})
	}
}

// TestBuildContinuationIfNeeded exercises the helper that DownloadWorkflowLogs uses
// to emit a pagination cursor when a date-range fetch hits the count limit or times out.
func TestBuildContinuationIfNeeded(t *testing.T) {
	runs := []ProcessedRun{
		{Run: WorkflowRun{DatabaseID: 3000}},
		{Run: WorkflowRun{DatabaseID: 2999}}, // oldest – used as BeforeRunID cursor
	}

	t.Run("count limit reached emits cursor with correct message and BeforeRunID", func(t *testing.T) {
		c := buildContinuationIfNeeded(runs, false, true, continuationOptions{
			workflowName:   "my-workflow",
			startDate:      "2026-06-01",
			endDate:        "2026-06-30",
			engine:         "claude",
			branch:         "main",
			afterRunID:     0,
			count:          100,
			timeoutMinutes: 3,
		})
		require.NotNil(t, c, "expected continuation when countLimitReached=true")
		assert.Equal(t, int64(2999), c.BeforeRunID, "BeforeRunID should be oldest processed run")
		assert.Equal(t, "2026-06-01", c.StartDate)
		assert.Equal(t, "2026-06-30", c.EndDate)
		assert.Equal(t, 100, c.Count)
		assert.Contains(t, c.Message, "Count limit reached")
	})

	t.Run("timeout reached emits cursor with timeout message", func(t *testing.T) {
		c := buildContinuationIfNeeded(runs, true, false, continuationOptions{
			workflowName:   "my-workflow",
			startDate:      "2026-06-01",
			endDate:        "",
			engine:         "claude",
			branch:         "",
			afterRunID:     0,
			count:          50,
			timeoutMinutes: 10,
		})
		require.NotNil(t, c, "expected continuation when timeoutReached=true")
		assert.Equal(t, int64(2999), c.BeforeRunID)
		assert.Contains(t, c.Message, "Timeout reached")
	})

	t.Run("neither flag set returns nil", func(t *testing.T) {
		c := buildContinuationIfNeeded(runs, false, false, continuationOptions{
			workflowName:   "my-workflow",
			startDate:      "2026-06-01",
			endDate:        "",
			engine:         "claude",
			branch:         "",
			afterRunID:     0,
			count:          100,
			timeoutMinutes: 3,
		})
		assert.Nil(t, c, "expected nil when neither timeout nor count limit was reached")
	})

	t.Run("empty processedRuns returns nil even when count limit reached", func(t *testing.T) {
		c := buildContinuationIfNeeded(nil, false, true, continuationOptions{
			workflowName:   "my-workflow",
			startDate:      "2026-06-01",
			endDate:        "",
			engine:         "claude",
			branch:         "",
			afterRunID:     0,
			count:          100,
			timeoutMinutes: 3,
		})
		assert.Nil(t, c, "expected nil when no runs were processed")
	})
}

func TestComputeLogsBatchSize(t *testing.T) {
	tests := []struct {
		name            string
		workflowName    string
		count           int
		processedCount  int
		fetchAllInRange bool
		want            int
	}{
		{
			name:           "default batch size for named workflow",
			workflowName:   "logs.yml",
			count:          100,
			processedCount: 0,
			want:           BatchSize,
		},
		{
			name:           "larger default for all workflows",
			count:          100,
			processedCount: 0,
			want:           BatchSizeForAllWorkflows,
		},
		{
			name:           "small remaining count uses buffered batch size",
			workflowName:   "logs.yml",
			count:          10,
			processedCount: 8,
			want:           6,
		},
		{
			name:            "date range keeps default batch size",
			workflowName:    "logs.yml",
			count:           10,
			processedCount:  8,
			fetchAllInRange: true,
			want:            BatchSize,
		},
		{
			name:           "all workflows keep minimum scan size",
			count:          10,
			processedCount: 8,
			want:           BatchSizeForAllWorkflows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, computeLogsBatchSize(tt.workflowName, tt.count, tt.processedCount, tt.fetchAllInRange))
		})
	}
}

func TestHandleEmptyWorkflowRunBatch(t *testing.T) {
	t.Run("stop when pagination exhausted", func(t *testing.T) {
		cursor, shouldContinue, shouldStop := handleEmptyWorkflowRunBatch(workflowRunBatch{
			totalFetched: 5,
			batchSize:    10,
		}, false)
		assert.Empty(t, cursor)
		assert.False(t, shouldContinue)
		assert.True(t, shouldStop)
	})

	t.Run("advance cursor when more pages may exist", func(t *testing.T) {
		cursorTime := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
		cursor, shouldContinue, shouldStop := handleEmptyWorkflowRunBatch(workflowRunBatch{
			totalFetched:           BatchSize,
			batchSize:              BatchSize,
			oldestFetchedCreatedAt: cursorTime,
		}, false)
		assert.Equal(t, cursorTime.Format(time.RFC3339), cursor)
		assert.True(t, shouldContinue)
		assert.False(t, shouldStop)
	})
}

// TestCollectProcessedWorkflowRunsAccumulatesBatches is a regression test for a bug where
// the batch results were assigned to a loop-scoped copy of processedRuns, so every
// processed run was discarded and `gh aw logs` reported "No workflow runs with artifacts
// found matching the specified criteria" even though artifacts had been downloaded.
func TestCollectProcessedWorkflowRunsAccumulatesBatches(t *testing.T) {
	batches := [][]WorkflowRun{
		{{DatabaseID: 1}, {DatabaseID: 2}},
		{{DatabaseID: 3}},
	}
	fetchCalls := 0

	originalFetch := logsFetchWorkflowRunBatch
	originalProcess := logsProcessWorkflowRunBatch
	t.Cleanup(func() {
		logsFetchWorkflowRunBatch = originalFetch
		logsProcessWorkflowRunBatch = originalProcess
	})

	logsFetchWorkflowRunBatch = func(_ context.Context, _ LogsDownloadOptions, _ string, _ int, _ bool) (workflowRunBatch, error) {
		if fetchCalls >= len(batches) {
			return workflowRunBatch{runs: nil, totalFetched: 0, batchSize: 2}, nil
		}
		runs := batches[fetchCalls]
		fetchCalls++
		// totalFetched == batchSize keeps pagination going after the first batch.
		return workflowRunBatch{runs: runs, totalFetched: len(runs), batchSize: 2}, nil
	}
	logsProcessWorkflowRunBatch = func(_ context.Context, batch workflowRunBatch, processedRuns []ProcessedRun, _ processWorkflowRunBatchOptions) ([]ProcessedRun, int, bool, bool) {
		for _, run := range batch.runs {
			processedRuns = append(processedRuns, ProcessedRun{Run: run})
		}
		return processedRuns, len(batch.runs), true, false
	}

	runs, timeoutReached, countLimitReached, err := collectProcessedWorkflowRuns(
		logsDownloadRuntime{activeCtx: context.Background(), fetchAllInRange: true},
		LogsDownloadOptions{Count: 100, StartDate: "-1d"},
	)
	require.NoError(t, err)
	assert.False(t, timeoutReached)
	assert.False(t, countLimitReached)
	require.Len(t, runs, 3, "runs from every batch should accumulate across iterations")
	assert.Equal(t, int64(1), runs[0].Run.DatabaseID)
	assert.Equal(t, int64(3), runs[2].Run.DatabaseID)
}

// TestStaleLogsWarning verifies that a warning is only emitted when no explicit
// start_date/end_date was requested and the newest run in the result set is older
// than the staleness threshold. This guards against the "logs" tool silently
// serving stale data without any indication when called with only a count.
func TestStaleLogsWarning(t *testing.T) {
	t.Run("no warning when start date explicitly provided", func(t *testing.T) {
		runs := []ProcessedRun{{Run: WorkflowRun{CreatedAt: time.Now().Add(-30 * 24 * time.Hour)}}}
		assert.Empty(t, staleLogsWarning(runs, "-1d", ""))
	})

	t.Run("no warning when end date explicitly provided", func(t *testing.T) {
		runs := []ProcessedRun{{Run: WorkflowRun{CreatedAt: time.Now().Add(-30 * 24 * time.Hour)}}}
		assert.Empty(t, staleLogsWarning(runs, "", "2024-01-01"))
	})

	t.Run("no warning when no runs", func(t *testing.T) {
		assert.Empty(t, staleLogsWarning(nil, "", ""))
	})

	t.Run("no warning when newest run is recent", func(t *testing.T) {
		runs := []ProcessedRun{
			{Run: WorkflowRun{CreatedAt: time.Now().Add(-1 * time.Hour)}},
			{Run: WorkflowRun{CreatedAt: time.Now().Add(-40 * 24 * time.Hour)}},
		}
		assert.Empty(t, staleLogsWarning(runs, "", ""))
	})

	t.Run("warns when no dates given and newest run is old", func(t *testing.T) {
		oldest := time.Now().Add(-11 * 24 * time.Hour)
		runs := []ProcessedRun{
			{Run: WorkflowRun{CreatedAt: oldest}},
			{Run: WorkflowRun{CreatedAt: oldest.Add(-time.Hour)}},
		}
		warning := staleLogsWarning(runs, "", "")
		require.NotEmpty(t, warning)
		assert.Contains(t, warning, "No start_date/end_date was specified")
		assert.Contains(t, warning, "start_date")
	})
}
