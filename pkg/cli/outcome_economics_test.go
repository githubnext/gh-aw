//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutcomeSummaryAICPerAcceptedOutcome(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		accepted int
		aic      float64
		expected float64
	}{
		{"accepted share of full run cost", 2, 12, 6},
		{"no accepted outcomes", 0, 12, 0},
		{"missing cost", 2, 0, 0},
		{"invalid cost", 2, -1, 0},
		{"not a number", 2, math.NaN(), 0},
		{"infinite cost", 2, math.Inf(1), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := OutcomeSummary{Accepted: tc.accepted}
			summary.addAIC(tc.aic)
			assert.InDelta(t, tc.expected, summary.CostPerAcceptedOutcome, 1e-9)
			data, err := json.Marshal(summary)
			require.NoError(t, err)
			if tc.expected == 0 {
				assert.NotContains(t, string(data), "cost_per_accepted_outcome")
			}
		})
	}
}

func TestLoadOutcomeAIC(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	aic, err := loadOutcomeAIC(dir, &RunSummary{RunAnalysis: RunAnalysis{
		TokenUsage: &TokenUsageSummary{TotalAIC: 12},
	}})
	require.NoError(t, err)
	assert.InDelta(t, 12, aic, 1e-9)

	aic, err = loadOutcomeAIC(dir, nil)
	require.NoError(t, err)
	assert.Zero(t, aic, "absent usage must not invent a cost")

	require.NoError(t, os.Mkdir(filepath.Join(dir, "usage"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "usage", "agent_usage.jsonl"), []byte("{\"aic\":4.5}\n"), 0o600))
	aic, err = loadOutcomeAIC(dir, nil)
	require.NoError(t, err)
	assert.InDelta(t, 4.5, aic, 1e-9)
}

func TestAuditOutcomeSummaryIncludesAICPerAcceptedOutcome(t *testing.T) {
	original := workflowOutcomeGHAPIGet
	t.Cleanup(func() { workflowOutcomeGHAPIGet = original })
	workflowOutcomeGHAPIGet = func(_ context.Context, endpoint, repo string) (map[string]any, error) {
		assert.Equal(t, "acme/repo", repo)
		return map[string]any{"status": "completed", "conclusion": "success"}, nil
	}
	data := &AuditData{Metrics: MetricsData{AIC: 12}}
	items := []CreatedItemReport{
		{Type: "dispatch_workflow", Repo: "acme/repo", Metadata: map[string]any{"run_id": 1}},
		{Type: "dispatch_workflow", Repo: "acme/repo", Metadata: map[string]any{"run_id": 2}},
	}
	addAuditOutcomeSummary(context.Background(), data, items)
	require.NotNil(t, data.OutcomeSummary)
	assert.Equal(t, 2, data.OutcomeSummary.Accepted)
	assert.InDelta(t, 6, data.OutcomeSummary.CostPerAcceptedOutcome, 1e-9)
}
