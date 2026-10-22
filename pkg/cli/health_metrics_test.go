//go:build !integration

package cli

import (
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/stretchr/testify/assert"
)

func TestCalculateWorkflowHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		workflowName  string
		runs          []WorkflowRun
		threshold     float64
		expectedRate  float64
		expectedTrend string
	}{
		{
			name:         "all successful runs",
			workflowName: "test-workflow",
			runs: []WorkflowRun{
				{Conclusion: "success", Duration: 2 * time.Minute},
				{Conclusion: "success", Duration: 3 * time.Minute},
				{Conclusion: "success", Duration: 2 * time.Minute},
			},
			threshold:     80.0,
			expectedRate:  100.0,
			expectedTrend: "→",
		},
		{
			name:         "mixed success and failure",
			workflowName: "test-workflow",
			runs: []WorkflowRun{
				{Conclusion: "success", Duration: 2 * time.Minute},
				{Conclusion: "failure", Duration: 1 * time.Minute},
				{Conclusion: "success", Duration: 3 * time.Minute},
				{Conclusion: "success", Duration: 2 * time.Minute},
			},
			threshold:    80.0,
			expectedRate: 75.0,
			// Don't check trend for small dataset
		},
		{
			name:         "all failed runs",
			workflowName: "test-workflow",
			runs: []WorkflowRun{
				{Conclusion: "failure", Duration: 1 * time.Minute},
				{Conclusion: "failure", Duration: 2 * time.Minute},
			},
			threshold:     80.0,
			expectedRate:  0.0,
			expectedTrend: "→",
		},
		{
			name:         "empty runs",
			workflowName: "test-workflow",
			runs:         []WorkflowRun{},
			threshold:    80.0,
			expectedRate: 0.0,
			// Empty runs should not be checked for below threshold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := CalculateWorkflowHealth(tt.workflowName, tt.runs, tt.threshold)

			assert.Equal(t, tt.workflowName, health.WorkflowName, "Workflow name should match")

			// Use InDelta for all float comparisons to satisfy testifylint
			assert.InDelta(t, tt.expectedRate, health.SuccessRate, 0.01, "Success rate should match")

			if tt.expectedTrend != "" {
				assert.Equal(t, tt.expectedTrend, health.Trend, "Trend should match")
			}

			if len(tt.runs) > 0 {
				assert.Len(t, tt.runs, health.TotalRuns, "Total runs should match")
			}

			// Check below threshold flag
			if len(tt.runs) > 0 && tt.expectedRate < tt.threshold {
				assert.True(t, health.BelowThresh, "Should be marked as below threshold")
			} else if len(tt.runs) > 0 {
				assert.False(t, health.BelowThresh, "Should not be marked as below threshold")
			}
		})
	}
}

func TestCalculateTrend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		runs     []WorkflowRun
		expected TrendDirection
	}{
		{
			name: "improving trend",
			runs: []WorkflowRun{
				{Conclusion: "success"},
				{Conclusion: "success"},
				{Conclusion: "success"},
				{Conclusion: "success"},
				{Conclusion: "failure"},
				{Conclusion: "failure"},
				{Conclusion: "failure"},
				{Conclusion: "success"},
			},
			expected: TrendImproving,
		},
		{
			name: "degrading trend",
			runs: []WorkflowRun{
				{Conclusion: "failure"},
				{Conclusion: "failure"},
				{Conclusion: "failure"},
				{Conclusion: "failure"},
				{Conclusion: "success"},
				{Conclusion: "success"},
				{Conclusion: "success"},
				{Conclusion: "success"},
			},
			expected: TrendDegrading,
		},
		{
			name: "stable trend",
			runs: []WorkflowRun{
				{Conclusion: "success"},
				{Conclusion: "success"},
				{Conclusion: "failure"},
				{Conclusion: "failure"},
				{Conclusion: "success"},
				{Conclusion: "success"},
				{Conclusion: "failure"},
				{Conclusion: "failure"},
			},
			expected: TrendStable,
		},
		{
			name: "not enough data",
			runs: []WorkflowRun{
				{Conclusion: "success"},
				{Conclusion: "success"},
			},
			expected: TrendStable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trend := calculateTrend(tt.runs)
			assert.Equal(t, tt.expected, trend, "Trend direction should match expected")
		})
	}
}

func TestCalculateHealthSummary(t *testing.T) {
	t.Parallel()
	workflowHealths := []WorkflowHealth{
		{WorkflowName: "workflow-a", SuccessRate: 90.0, BelowThresh: false},
		{WorkflowName: "workflow-b", SuccessRate: 75.0, BelowThresh: true},
		{WorkflowName: "workflow-c", SuccessRate: 85.0, BelowThresh: false},
	}

	summary := CalculateHealthSummary(workflowHealths, "Last 7 Days", 80.0)

	assert.Equal(t, "Last 7 Days", summary.Period, "Period should match")
	assert.Equal(t, 3, summary.TotalWorkflows, "Total workflows should be 3")
	assert.Equal(t, 2, summary.HealthyWorkflows, "Healthy workflows should be 2")
	assert.Equal(t, 1, summary.BelowThreshold, "Below threshold count should be 1")
	assert.Len(t, summary.Workflows, 3, "Workflows array should have 3 entries")
}

func TestCalculateHealthSummaryExcludesIntentionalFailure(t *testing.T) {
	t.Parallel()
	// Intentional-failure workflows (e.g. credit-guardrail stress tests) must be excluded
	// from fleet-health rollup counts so they do not depress the real-regression baseline.
	workflowHealths := []WorkflowHealth{
		{WorkflowName: "normal-a", SuccessRate: 90.0, BelowThresh: false},
		{WorkflowName: "normal-b", SuccessRate: 75.0, BelowThresh: true},
		// These are tagged intentional-failure: should not affect HealthyWorkflows / BelowThreshold.
		{WorkflowName: "daily-credit-limit-test", SuccessRate: 0.0, BelowThresh: true, IntentionalFailure: true},
		{WorkflowName: "daily-max-ai-credits-test", SuccessRate: 0.0, BelowThresh: true, IntentionalFailure: true},
	}

	summary := CalculateHealthSummary(workflowHealths, "Last 7 Days", 80.0)

	// TotalWorkflows still counts all four (the table shows them).
	assert.Equal(t, 4, summary.TotalWorkflows, "TotalWorkflows should include intentional-failure entries")

	// Rollup counts must exclude the two intentional-failure workflows.
	assert.Equal(t, 1, summary.HealthyWorkflows, "HealthyWorkflows should exclude intentional-failure workflows")
	assert.Equal(t, 1, summary.BelowThreshold, "BelowThreshold should exclude intentional-failure workflows")

	// The full workflows slice is preserved so callers can render the table.
	assert.Len(t, summary.Workflows, 4, "Workflows slice should contain all entries")
}

func TestTrendDirectionString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		trend    TrendDirection
		expected string
	}{
		{
			name:     "improving",
			trend:    TrendImproving,
			expected: "↑",
		},
		{
			name:     "stable",
			trend:    TrendStable,
			expected: "→",
		},
		{
			name:     "degrading",
			trend:    TrendDegrading,
			expected: "↓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.trend.String()
			assert.Equal(t, tt.expected, result, "Trend string representation should match")
		})
	}
}

func TestFormatTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		tokens   int
		expected string
	}{
		{
			name:     "zero tokens",
			tokens:   0,
			expected: "-",
		},
		{
			name:     "small tokens",
			tokens:   500,
			expected: "500",
		},
		{
			name:     "thousands",
			tokens:   5000,
			expected: "5.0K",
		},
		{
			name:     "millions",
			tokens:   2500000,
			expected: "2.5M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := console.FormatTokens(tt.tokens)
			assert.Equal(t, tt.expected, result, "Formatted tokens should match")
		})
	}
}

func TestCalculateWorkflowHealthDriverExitClassification(t *testing.T) {
	t.Parallel()
	runs := []WorkflowRun{
		{Conclusion: "success", Duration: 2 * time.Minute, Turns: 3, TurnsAvailable: true},
		// driver-exit: failed with zero turns (TurnsAvailable confirms the count is real)
		{Conclusion: "failure", Duration: 1 * time.Minute, Turns: 0, TurnsAvailable: true},
		{Conclusion: "failure", Duration: 1 * time.Minute, Turns: 0, TurnsAvailable: true},
		// agent-logic: failed but agent ran (turns > 0)
		{Conclusion: "failure", Duration: 2 * time.Minute, Turns: 5, TurnsAvailable: true},
	}

	health := CalculateWorkflowHealth("test-workflow", runs, 80.0)

	assert.Equal(t, 4, health.TotalRuns, "total runs should be 4")
	assert.Equal(t, 1, health.SuccessCount, "success count should be 1")
	assert.Equal(t, 3, health.FailureCount, "failure count should be 3")
	assert.Equal(t, 2, health.DriverExitCount, "driver-exit count should be 2 (zero-turn failures)")
	assert.Equal(t, 1, health.AgentLogicFailureCount, "agent-logic count should be 1 (non-zero-turn failure)")
}

func TestCalculateWorkflowHealthDriverExitCountZeroWhenNoFailures(t *testing.T) {
	t.Parallel()
	runs := []WorkflowRun{
		{Conclusion: "success", Duration: 2 * time.Minute, Turns: 3},
		{Conclusion: "success", Duration: 3 * time.Minute, Turns: 4},
	}

	health := CalculateWorkflowHealth("test-workflow", runs, 80.0)

	assert.Equal(t, 0, health.DriverExitCount, "driver-exit count should be zero when all runs succeed")
	assert.Equal(t, 0, health.AgentLogicFailureCount, "agent-logic count should be zero when all runs succeed")
}

func TestCalculateWorkflowHealthTurnsUnavailableSkipsClassification(t *testing.T) {
	t.Parallel()
	// Simulate the gh aw health path: GitHub API metadata only, no artifact logs,
	// so TurnsAvailable is false for every run. Classification should be skipped and
	// both DriverExitCount and AgentLogicFailureCount must stay at zero.
	runs := []WorkflowRun{
		{Conclusion: "success", Duration: 2 * time.Minute},
		{Conclusion: "failure", Duration: 1 * time.Minute, Turns: 0, TurnsAvailable: false},
		{Conclusion: "failure", Duration: 1 * time.Minute, Turns: 0, TurnsAvailable: false},
	}

	health := CalculateWorkflowHealth("test-workflow", runs, 80.0)

	assert.Equal(t, 2, health.FailureCount, "failure count should still be 2")
	assert.Equal(t, 0, health.DriverExitCount, "driver-exit count should be zero when TurnsAvailable=false")
	assert.Equal(t, 0, health.AgentLogicFailureCount, "agent-logic count should be zero when TurnsAvailable=false")
}

func TestIsDriverExitFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		run      WorkflowRun
		expected bool
	}{
		{
			name:     "failure with zero turns is driver-exit",
			run:      WorkflowRun{Conclusion: "failure", Turns: 0, TurnsAvailable: true},
			expected: true,
		},
		{
			name:     "timed_out with zero turns is driver-exit",
			run:      WorkflowRun{Conclusion: "timed_out", Turns: 0, TurnsAvailable: true},
			expected: true,
		},
		{
			name:     "cancelled with zero turns is driver-exit",
			run:      WorkflowRun{Conclusion: "cancelled", Turns: 0, TurnsAvailable: true},
			expected: true,
		},
		{
			name:     "failure with non-zero turns is agent-logic (not driver-exit)",
			run:      WorkflowRun{Conclusion: "failure", Turns: 3, TurnsAvailable: true},
			expected: false,
		},
		{
			name:     "failure with zero turns but TurnsAvailable=false is unclassified",
			run:      WorkflowRun{Conclusion: "failure", Turns: 0, TurnsAvailable: false},
			expected: false,
		},
		{
			name:     "success with zero turns is not driver-exit",
			run:      WorkflowRun{Conclusion: "success", Turns: 0},
			expected: false,
		},
		{
			name:     "success with non-zero turns is not driver-exit",
			run:      WorkflowRun{Conclusion: "success", Turns: 5},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDriverExitFailure(tt.run)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthSummaryStatusCountsAndAboveThresholdFailures(t *testing.T) {
	workflows := []WorkflowHealth{
		{Status: "healthy", SuccessRate: 100},
		{Status: "recovery_observed", SuccessRate: 100},
		{Status: "failures_observed", SuccessRate: 90, FailureCount: 1},
		{Status: "insufficient_evidence", SuccessRate: 50, FailureCount: 1},
		{Status: "subagent_failures_observed", SuccessRate: 100, SubagentFailureCount: 2},
		{Status: "failures_observed", IntentionalFailure: true, FailureCount: 1},
		{Status: "no_recent_runs"},
	}
	summary := CalculateHealthSummary(workflows, "bounded", 80)
	assert.Equal(t, 2, summary.HealthyWorkflows, "an above-threshold failure status is not healthy")
	assert.Equal(t, 2, summary.ObservedFailureWorkflows)
	assert.Equal(t, 1, summary.DegradedWorkflows, "legacy degradation is sub-agent-only")
	assert.Equal(t, 2, summary.InsufficientWorkflows)
	assert.Equal(t, map[string]int{
		"healthy": 1, "recovery_observed": 1, "failures_observed": 2,
		"insufficient_evidence": 1, "subagent_failures_observed": 1, "no_recent_runs": 1,
	}, summary.StatusCounts)
	total := 0
	for _, count := range summary.StatusCounts {
		total += count
	}
	assert.Equal(t, summary.TotalWorkflows, total)
}
