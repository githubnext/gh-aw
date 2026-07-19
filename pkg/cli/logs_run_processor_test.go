//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHasEvals(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		expected bool
	}{
		{
			name: "root-level evals.jsonl (flattenSingleFileArtifacts output)",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(dir, constants.EvalsResultFilename), []byte("{}"), 0600))
			},
			expected: true,
		},
		{
			name: "evals/evals.jsonl (un-flattened artifact directory)",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				evalsDir := filepath.Join(dir, constants.EvalsArtifactName)
				require.NoError(t, os.Mkdir(evalsDir, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(evalsDir, constants.EvalsResultFilename), []byte("{}"), 0600))
			},
			expected: true,
		},
		{
			name: "hash-prefixed {hash}-evals/evals.jsonl (workflow_call variant)",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				evalsDir := filepath.Join(dir, "abc123-"+constants.EvalsArtifactName)
				require.NoError(t, os.Mkdir(evalsDir, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(evalsDir, constants.EvalsResultFilename), []byte("{}"), 0600))
			},
			expected: true,
		},
		{
			name: "evals/ directory exists but contains no evals.jsonl",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				evalsDir := filepath.Join(dir, constants.EvalsArtifactName)
				require.NoError(t, os.Mkdir(evalsDir, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(evalsDir, "other.txt"), []byte("data"), 0600))
			},
			expected: false,
		},
		{
			name: "usage/evals.jsonl (compact usage artifact)",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				usageDir := filepath.Join(dir, constants.UsageArtifactName)
				require.NoError(t, os.Mkdir(usageDir, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(usageDir, constants.EvalsResultFilename), []byte("{}"), 0600))
			},
			expected: true,
		},
		{
			name: "hash-prefixed {hash}-usage/evals.jsonl (workflow_call compact usage artifact)",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				usageDir := filepath.Join(dir, "abc123-"+constants.UsageArtifactName)
				require.NoError(t, os.Mkdir(usageDir, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(usageDir, constants.EvalsResultFilename), []byte("{}"), 0600))
			},
			expected: true,
		},
		{
			name:     "empty directory",
			setup:    func(t *testing.T, dir string) {},
			expected: false,
		},
		{
			name: "non-existent directory",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				require.NoError(t, os.RemoveAll(dir))
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			assert.Equal(t, tc.expected, runHasEvals(dir, false))
		})
	}
}

func TestBackfillRunTokenUsageFromFirewall(t *testing.T) {
	t.Run("backfills run and metrics token usage from firewall summary", func(t *testing.T) {
		metrics := LogMetrics{}
		result := DownloadResult{}
		tokenUsage := &TokenUsageSummary{
			TotalInputTokens:  2000,
			TotalOutputTokens: 1000,
		}

		backfillRunTokenUsageFromFirewall(&metrics, &result, tokenUsage)

		assert.Equal(t, 3000, metrics.TokenUsage)
		assert.Equal(t, 3000, result.Metrics.TokenUsage)
		assert.Equal(t, 3000, result.Run.TokenUsage)
	})

	t.Run("does not overwrite non-zero event token usage", func(t *testing.T) {
		metrics := LogMetrics{TokenUsage: 123}
		result := DownloadResult{
			Run:     WorkflowRun{TokenUsage: 123},
			Metrics: LogMetrics{TokenUsage: 123},
		}
		tokenUsage := &TokenUsageSummary{
			TotalInputTokens:  2000,
			TotalOutputTokens: 1000,
		}

		backfillRunTokenUsageFromFirewall(&metrics, &result, tokenUsage)

		assert.Equal(t, 123, metrics.TokenUsage)
		assert.Equal(t, 123, result.Metrics.TokenUsage)
		assert.Equal(t, 123, result.Run.TokenUsage)
	})
}

func TestTryLoadCachedRunResultBypassesForExplicitEvalsArtifactRequest(t *testing.T) {
	runOutputDir := t.TempDir()
	summary := &RunSummary{
		CLIVersion:  GetVersion(),
		RunID:       123,
		ProcessedAt: time.Now(),
		Run: WorkflowRun{
			DatabaseID: 123,
		},
	}
	require.NoError(t, saveRunSummary(runOutputDir, summary, false))

	result, ok := tryLoadCachedRunResult(context.Background(), WorkflowRun{DatabaseID: 123}, runOutputDir, concurrentRunDownloadParams{
		evalsOnly:              false,
		evalsArtifactRequested: true,
		verbose:                false,
	})
	assert.False(t, ok, "cache should be bypassed so fallback download can run for explicit --artifacts evals")
	assert.Nil(t, result)
}

func TestTryLoadCachedRunResultUsesCacheWhenEvalsNotRequested(t *testing.T) {
	runOutputDir := t.TempDir()
	summary := &RunSummary{
		CLIVersion:  GetVersion(),
		RunID:       124,
		ProcessedAt: time.Now(),
		Run: WorkflowRun{
			DatabaseID: 124,
		},
	}
	require.NoError(t, saveRunSummary(runOutputDir, summary, false))

	result, ok := tryLoadCachedRunResult(context.Background(), WorkflowRun{DatabaseID: 124}, runOutputDir, concurrentRunDownloadParams{
		evalsOnly:              false,
		evalsArtifactRequested: false,
		verbose:                false,
	})
	require.True(t, ok)
	require.NotNil(t, result)
	assert.True(t, result.Cached)
}

// TestTryLoadCachedRunResultPersistsSafeItemsCountAfterBackfill verifies that when
// tryLoadCachedRunResult heals a stale SafeItemsCount (0 → N) via backfillCacheHitIfNeeded,
// the healed value is written back to run_summary.json on disk so downstream readers
// (e.g. api-consumption-report) see the correct count without falling back to the
// activity summary.
func TestTryLoadCachedRunResultPersistsSafeItemsCountAfterBackfill(t *testing.T) {
	runOutputDir := t.TempDir()

	// Write a run_summary.json with SafeItemsCount=0 (stale cache).
	summary := &RunSummary{
		CLIVersion:  GetVersion(),
		RunID:       200,
		ProcessedAt: time.Now(),
		Run: WorkflowRun{
			DatabaseID:     200,
			SafeItemsCount: 0,
		},
	}
	require.NoError(t, saveRunSummary(runOutputDir, summary, false))

	// Write a usage/activity/summary.json so backfill has something to pull from.
	activityPath := filepath.Join(runOutputDir, "usage", "activity", "summary.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(activityPath), 0o755))
	require.NoError(t, os.WriteFile(activityPath, []byte(`{
		"schema":"usage-activity-summary/v1",
		"safe_outputs":{"total_items":5,"items_by_type":{"create_issue":5}}
	}`), 0o644))

	result, ok := tryLoadCachedRunResult(context.Background(), WorkflowRun{DatabaseID: 200}, runOutputDir, concurrentRunDownloadParams{})
	require.True(t, ok)
	require.NotNil(t, result)

	// In-memory value should be healed.
	assert.Equal(t, 5, result.Run.SafeItemsCount, "in-memory SafeItemsCount should be backfilled")

	// The on-disk run_summary.json must also reflect the healed value.
	reloaded, ok := loadRunSummary(runOutputDir, false)
	require.True(t, ok)
	assert.Equal(t, 5, reloaded.Run.SafeItemsCount, "on-disk run_summary.json SafeItemsCount should be persisted after backfill")
}
