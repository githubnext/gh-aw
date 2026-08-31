package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackfillOperationalValueReportObservationsUsesWeeklyCache(t *testing.T) {
	originalGradeRun := operationalValueReportGradeRun
	t.Cleanup(func() { operationalValueReportGradeRun = originalGradeRun })

	gradeCalls := 0
	operationalValueReportGradeRun = func(_ context.Context, evaluator *operationalValueReportEvaluator, run operationalValueReportRun, _ time.Time, _ string) operationalValueReportObservation {
		gradeCalls++
		value := float64(gradeCalls) / 10
		return operationalValueReportObservation{
			Run: run, Value: &value, Status: "pass", Mature: true,
			EvidenceAt: "2026-08-31T00:00:00Z", EvidenceCutoff: "2026-08-28T00:00:00Z",
			MaturesAt: "2026-08-28T00:00:00Z", EvaluatorDigest: evaluator.EvaluatorDigest,
			Source: "evaluator-replay",
		}
	}

	evaluator := &operationalValueReportEvaluator{
		WorkflowID: "daily-file-diet", EvaluatorDigest: "abc123",
		Definition: operationalValueReportDefinition{Repository: "github/gh-aw"},
	}
	runs := []operationalValueReportRun{
		{ID: "1", Attempt: 1, CreatedAt: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)},
		{ID: "2", Attempt: 1, CreatedAt: time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)},
	}
	cacheRoot := t.TempDir()

	first, firstStats, err := backfillOperationalValueReportObservations(context.Background(), evaluator, runs, time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC), cacheRoot, "", false)
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, 2, firstStats.Evaluated)
	assert.Equal(t, 0, firstStats.CacheHits)
	assert.Equal(t, 2, gradeCalls)

	second, secondStats, err := backfillOperationalValueReportObservations(context.Background(), evaluator, runs, time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC), cacheRoot, "", false)
	require.NoError(t, err)
	require.Len(t, second, 2)
	assert.Equal(t, 0, secondStats.Evaluated)
	assert.Equal(t, 2, secondStats.CacheHits)
	assert.Equal(t, 2, gradeCalls)
	assert.Equal(t, "evaluator-replay", second[0].Source)
}

func TestBackfillOperationalValueReportObservationsDoesNotCacheNonFinalResults(t *testing.T) {
	originalGradeRun := operationalValueReportGradeRun
	t.Cleanup(func() { operationalValueReportGradeRun = originalGradeRun })

	for _, test := range []struct {
		name   string
		status string
		mature bool
	}{
		{name: "error", status: "error", mature: false},
		{name: "immature", status: "pass", mature: false},
		{name: "unavailable", status: "unavailable", mature: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			gradeCalls := 0
			operationalValueReportGradeRun = func(_ context.Context, evaluator *operationalValueReportEvaluator, run operationalValueReportRun, _ time.Time, _ string) operationalValueReportObservation {
				gradeCalls++
				return operationalValueReportObservation{Run: run, Status: test.status, Mature: test.mature, EvaluatorDigest: evaluator.EvaluatorDigest}
			}
			evaluator := &operationalValueReportEvaluator{WorkflowID: "daily-file-diet", EvaluatorDigest: "abc123", Definition: operationalValueReportDefinition{Repository: "github/gh-aw"}}
			runs := []operationalValueReportRun{{ID: "1", Attempt: 1, CreatedAt: time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)}}
			cacheRoot := t.TempDir()

			_, _, err := backfillOperationalValueReportObservations(context.Background(), evaluator, runs, time.Now(), cacheRoot, "", false)
			require.NoError(t, err)
			_, _, err = backfillOperationalValueReportObservations(context.Background(), evaluator, runs, time.Now(), cacheRoot, "", false)
			require.NoError(t, err)
			assert.Equal(t, 2, gradeCalls)
		})
	}
}
