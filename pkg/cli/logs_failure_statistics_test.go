//go:build !integration

package cli

import (
	"math"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailureShareWilson95(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		k, n         int
		lower, upper float64
	}{
		{0, 0, 0, 0},
		{0, 1, 0, 0.7934506856},
		{1, 1, 0.2065493144, 1},
		{50, 100, 0.4038315304, 0.5961684696},
		{1, 100, 0.0017674321, 0.0544861962},
	} {
		lower, upper := failureShareWilson95(tc.k, tc.n)
		assert.InDelta(t, tc.lower, lower, 1e-9)
		assert.InDelta(t, tc.upper, upper, 1e-9)
	}
	for n := 1; n <= 60; n++ {
		previousLower, previousUpper := 0.0, 0.0
		for k := 0; k <= n; k++ {
			lower, upper := failureShareWilson95(k, n)
			assert.False(t, math.IsNaN(lower) || math.IsNaN(upper))
			assert.GreaterOrEqual(t, lower, 0.0)
			assert.LessOrEqual(t, upper, 1.0)
			assert.LessOrEqual(t, lower, float64(k)/float64(n)+1e-15)
			assert.GreaterOrEqual(t, upper+1e-15, float64(k)/float64(n))
			assert.GreaterOrEqual(t, lower, previousLower)
			assert.GreaterOrEqual(t, upper, previousUpper)
			previousLower, previousUpper = lower, upper
		}
	}
}

func TestFailureClusterStatisticsPopulation(t *testing.T) {
	t.Parallel()
	signal := FailureSignal{Kind: "mcp_failure", Value: "github"}
	runs := []RunData{
		{RunID: 1, Conclusion: "failure", FailureSignals: []FailureSignal{signal}},
		{RunID: 2, Conclusion: "failure"},
		{RunID: 3, Conclusion: "success"},
		{RunID: 4, Conclusion: "in_progress"},
		{RunID: 5, Conclusion: "failure", IntentionalFailure: true, FailureSignals: []FailureSignal{signal}},
		{RunID: 1, Conclusion: "failure", FailureSignals: []FailureSignal{signal}},
	}
	clusters := annotateFailureClusters(runs)
	require.Len(t, clusters, 3)
	for _, cluster := range clusters {
		if cluster.IntentionalFailure {
			assert.Equal(t, 1, cluster.Statistics.FailedPopulation)
			assert.InDelta(t, 1.0, cluster.Statistics.Share, 1e-9)
		} else {
			assert.Equal(t, 2, cluster.Statistics.FailedPopulation)
			assert.InDelta(t, 0.5, cluster.Statistics.Share, 1e-9)
		}
	}
	slices.Reverse(runs)
	assert.Equal(t, clusters, annotateFailureClusters(runs))
}

func TestFailureInvestigationMarginalReachWithinRole(t *testing.T) {
	t.Parallel()
	clusters := map[string]FailureCluster{
		"covered":   {SignalRole: "failure_boundary", RunIDs: []int64{1, 2, 3, 4, 5}},
		"overlap":   {SignalRole: "failure_boundary", RunIDs: []int64{1, 2, 3, 4, 5, 6}},
		"new":       {SignalRole: "failure_boundary", RunIDs: []int64{7, 8}},
		"aftermath": {SignalRole: "aftermath", RunIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8}},
	}
	runs := []RunData{{RunID: 6}, {RunID: 8}, {RunID: 7, AuditPath: "/cached/audit.json"}}
	candidates := []failureInvestigationCandidate{
		{run: &runs[0], clusterIDs: []string{"overlap"}, signature: "overlap"},
		{run: &runs[1], clusterIDs: []string{"new"}, signature: "new"},
		{run: &runs[2], clusterIDs: []string{"new"}, signature: "new"},
	}
	covered := map[string]struct{}{"covered": {}, "aftermath": {}}
	best := selectFailureInvestigationCandidate(candidates, clusters, covered)
	require.NotNil(t, best)
	assert.Equal(t, int64(7), best.run.RunID, "prefer two new boundary members over one; cached gains must preserve evidence tie-breaks")
	slices.Reverse(candidates)
	assert.Equal(t, int64(7), selectFailureInvestigationCandidate(candidates, clusters, covered).run.RunID)
}

func BenchmarkFailureInvestigationPopulation(b *testing.B) {
	for _, size := range []int{1000, 10000, 100000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			runs := make([]RunData, 0, size)
			for i := range size {
				runs = append(runs, RunData{RunID: int64(i + 1), Conclusion: "failure",
					WorkflowName: "population", FailureSignals: []FailureSignal{
						{Kind: "mcp_failure", Value: strconv.Itoa(i % 20)},
						{Kind: "failed_job", Value: strconv.Itoa(i % 7)},
					}})
			}
			clusters := annotateFailureClusters(runs)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				buildFailureInvestigation(runs, clusters)
			}
		})
	}
}
