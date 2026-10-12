//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncompleteAuditTimeout(t *testing.T) {
	for _, stage := range []string{"metadata", "artifacts", "analysis", "job_logs"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			cfg := auditRunConfig{runID: 123, outputDir: dir, jsonOutput: true}
			run := WorkflowRun{DatabaseID: 123, WorkflowName: "example", Status: "completed"}
			results := auditAnalysisResults{missingData: []MissingDataReport{{Reason: "unavailable"}}}
			stdout, _ := captureOutput(t, func() error {
				err := renderIncompleteAuditIfTimedOut(cfg, run, results, stage, context.DeadlineExceeded)
				require.ErrorIs(t, err, context.DeadlineExceeded)
				return nil
			})
			var data map[string]any
			require.NoError(t, json.Unmarshal([]byte(stdout), &data))
			assert.Equal(t, true, data["partial"])
			assert.Equal(t, false, data["complete"])
			assert.Equal(t, stage, data["stage"])
			overview, ok := data["overview"].(map[string]any)
			require.True(t, ok)
			assert.InDelta(t, 123, overview["run_id"], 0)
			assert.Equal(t, "example", overview["workflow_name"])
			if stage == "analysis" {
				assert.NotEmpty(t, data["missing_data"])
			} else {
				assert.NotContains(t, data, "missing_data")
			}
			assert.DirExists(t, dir)
			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			assert.Empty(t, entries, "incomplete data must not populate caches")
		})
	}
}

func TestAuditPreparationTimeoutStage(t *testing.T) {
	assert.Equal(t, "metadata", auditPreparationTimeoutStage(WorkflowRun{}))
	assert.Equal(t, "artifacts", auditPreparationTimeoutStage(WorkflowRun{DatabaseID: 42}))
}

func TestRenderAuditReportReturnsDeadlineBeforeWritingCaches(t *testing.T) {
	runDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stdout, _ := captureOutput(t, func() error {
		err := renderAuditReport(ctx, ProcessedRun{Run: WorkflowRun{
			DatabaseID:   42,
			WorkflowPath: ".github/workflows/test.lock.yml",
			LogsPath:     runDir,
		}}, LogMetrics{}, nil, AuditOptions{
			OutputDir:  runDir,
			JSONOutput: true,
		})
		require.ErrorIs(t, err, context.Canceled)
		return nil
	})
	assert.Empty(t, stdout)
	entries, err := os.ReadDir(runDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRenderAuditReportDoesNotCacheWhenBaselineLookupTimesOut(t *testing.T) {
	withFakeBlockingGH(t)
	runDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	stdout, _ := captureOutput(t, func() error {
		err := renderAuditReport(ctx, ProcessedRun{Run: WorkflowRun{
			DatabaseID:   42,
			WorkflowPath: ".github/workflows/test.lock.yml",
			LogsPath:     runDir,
		}}, LogMetrics{}, nil, AuditOptions{
			OutputDir:  runDir,
			JSONOutput: true,
		})
		require.ErrorIs(t, err, context.DeadlineExceeded)
		return nil
	})
	assert.Empty(t, stdout)
	entries, err := os.ReadDir(runDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestFetchAuditJobLogHonorsContextDeadline(t *testing.T) {
	withFakeBlockingGH(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, _, err := fetchAuditJobLog(ctx, auditJobRunOptions{
		jobID:     123,
		outputDir: t.TempDir(),
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(started), time.Second)
}

func TestIncompleteAuditPreservesOtherErrors(t *testing.T) {
	for _, err := range []error{context.Canceled, errors.New("download failed"), context.DeadlineExceeded} {
		stdout, _ := captureOutput(t, func() error {
			got := renderIncompleteAuditIfTimedOut(auditRunConfig{}, WorkflowRun{}, auditAnalysisResults{}, "artifacts", err)
			assert.ErrorIs(t, got, err)
			return nil
		})
		assert.Empty(t, stdout, "non-JSON audit failures must not emit JSON")
	}
}

func TestAuditTimeoutFlag(t *testing.T) {
	cmd := NewAuditCommand()
	require.NoError(t, cmd.Flags().Set("timeout", "50s"))
	opts, err := getAuditCommandOptions(cmd)
	require.NoError(t, err)
	assert.Equal(t, "50s", opts.timeout.String())
	require.NoError(t, cmd.Flags().Set("timeout", "-1s"))
	_, err = getAuditCommandOptions(cmd)
	require.ErrorContains(t, err, "--timeout must not be negative")
}

func TestAuditExpiredDeadlineReturnsIncompleteJSON(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	stdout, _ := captureOutput(t, func() error {
		skipped, err := AuditWorkflowRun(ctx, 123, AuditOptions{
			Hostname: "github.com", OutputDir: t.TempDir(), JSONOutput: true,
		})
		assert.False(t, skipped)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		return nil
	})
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	assert.Equal(t, true, envelope["partial"])
	assert.Equal(t, false, envelope["complete"])
	assert.Equal(t, "metadata", envelope["stage"])
}
