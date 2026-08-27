//go:build !integration

package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildClusterAnalysis_TooFewInputs(t *testing.T) {
	t.Parallel()
	assert.Nil(t, buildClusterAnalysis(nil))
	assert.Nil(t, buildClusterAnalysis([]crossRunInput{{RunID: 1}}))
}

func TestBuildClusterAnalysis_ClustersConclusion(t *testing.T) {
	t.Parallel()
	inputs := []crossRunInput{
		{RunID: 1, Conclusion: "success", Metrics: LogMetrics{TokenUsage: 1000, Turns: 5}},
		{RunID: 2, Conclusion: "success", Metrics: LogMetrics{TokenUsage: 1200, Turns: 6}},
		{RunID: 3, Conclusion: "failure", Metrics: LogMetrics{TokenUsage: 3000, Turns: 10}},
	}

	ca := buildClusterAnalysis(inputs)
	require.NotNil(t, ca)
	assert.NotEmpty(t, ca.Clusters)

	// Find conclusion clusters
	var successCluster, failureCluster *RunCluster
	for i := range ca.Clusters {
		if ca.Clusters[i].Dimension == "conclusion" {
			switch ca.Clusters[i].Value {
			case "success":
				successCluster = &ca.Clusters[i]
			case "failure":
				failureCluster = &ca.Clusters[i]
			}
		}
	}
	require.NotNil(t, successCluster)
	require.NotNil(t, failureCluster)
	assert.Equal(t, 2, successCluster.Count)
	assert.Equal(t, 1, failureCluster.Count)
	assert.Contains(t, successCluster.RunIDs, int64(1))
	assert.Contains(t, successCluster.RunIDs, int64(2))
	assert.Contains(t, failureCluster.RunIDs, int64(3))
}

func TestBuildClusterAnalysis_ClustersGradersAndEvals(t *testing.T) {
	t.Parallel()
	inputs := []crossRunInput{
		{RunID: 1, Conclusion: "success", Metrics: LogMetrics{TokenUsage: 1000, Turns: 5}, GradersCluster: "pass", EvalsCluster: "present"},
		{RunID: 2, Conclusion: "success", Metrics: LogMetrics{TokenUsage: 1200, Turns: 6}, GradersCluster: "pass", EvalsCluster: "absent"},
		{RunID: 3, Conclusion: "failure", Metrics: LogMetrics{TokenUsage: 3000, Turns: 10}, GradersCluster: "fail", EvalsCluster: "present"},
	}

	ca := buildClusterAnalysis(inputs)
	require.NotNil(t, ca)

	var gradersPass, gradersFail, evalsPresent, evalsAbsent *RunCluster
	for i := range ca.Clusters {
		c := &ca.Clusters[i]
		if c.Dimension == "graders" {
			switch c.Value {
			case "pass":
				gradersPass = c
			case "fail":
				gradersFail = c
			}
		}
		if c.Dimension == "evals" {
			switch c.Value {
			case "present":
				evalsPresent = c
			case "absent":
				evalsAbsent = c
			}
		}
	}

	require.NotNil(t, gradersPass)
	require.NotNil(t, gradersFail)
	require.NotNil(t, evalsPresent)
	require.NotNil(t, evalsAbsent)
	assert.Equal(t, 2, gradersPass.Count)
	assert.Equal(t, 1, gradersFail.Count)
	assert.Equal(t, 2, evalsPresent.Count)
	assert.Equal(t, 1, evalsAbsent.Count)
}

func TestBuildClusterAnalysis_ResourceDivergencePattern(t *testing.T) {
	t.Parallel()
	inputs := []crossRunInput{
		{RunID: 1, Conclusion: "success", Metrics: LogMetrics{TokenUsage: 1000, Turns: 5}},
		{RunID: 2, Conclusion: "success", Metrics: LogMetrics{TokenUsage: 1000, Turns: 5}},
		{RunID: 3, Conclusion: "failure", Metrics: LogMetrics{TokenUsage: 5000, Turns: 15}},
		{RunID: 4, Conclusion: "failure", Metrics: LogMetrics{TokenUsage: 5000, Turns: 15}},
	}

	ca := buildClusterAnalysis(inputs)
	require.NotNil(t, ca)

	// Should detect resource divergence (5000/1000 = 5x > 1.5x threshold)
	var found bool
	for _, p := range ca.Patterns {
		if p.Kind == "resource_divergence" {
			found = true
			assert.Equal(t, "high", p.Severity) // 5x >= 3.0
			break
		}
	}
	assert.True(t, found, "Expected resource_divergence pattern")
}

func TestBuildClusterAnalysis_ResourceDivergencePattern_WithTimedOutAndCancelled(t *testing.T) {
	t.Parallel()
	inputs := []crossRunInput{
		{RunID: 1, Conclusion: "success", Metrics: LogMetrics{TokenUsage: 1000, Turns: 5}},
		{RunID: 2, Conclusion: "success", Metrics: LogMetrics{TokenUsage: 1200, Turns: 6}},
		{RunID: 3, Conclusion: "timed_out", Metrics: LogMetrics{TokenUsage: 4000, Turns: 12}},
		{RunID: 4, Conclusion: "cancelled", Metrics: LogMetrics{TokenUsage: 5000, Turns: 8}},
	}

	ca := buildClusterAnalysis(inputs)
	require.NotNil(t, ca)

	var found bool
	for _, p := range ca.Patterns {
		if p.Kind == "resource_divergence" {
			found = true
			assert.Equal(t, "high", p.Severity) // (4000+5000)/2 vs (1000+1200)/2 => ~4.1x
			break
		}
	}
	assert.True(t, found, "Expected resource_divergence pattern with timed_out/cancelled runs")
}

func TestBuildClusterAnalysis_FailureCorrelation(t *testing.T) {
	t.Parallel()
	inputs := []crossRunInput{
		{RunID: 1, Conclusion: "success", TaskDomain: &TaskDomainInfo{Name: "code_editing"}},
		{RunID: 2, Conclusion: "failure", TaskDomain: &TaskDomainInfo{Name: "testing"}},
		{RunID: 3, Conclusion: "failure", TaskDomain: &TaskDomainInfo{Name: "testing"}},
		{RunID: 4, Conclusion: "success", TaskDomain: &TaskDomainInfo{Name: "code_editing"}},
		{RunID: 5, Conclusion: "failure", TaskDomain: &TaskDomainInfo{Name: "testing"}},
	}

	ca := buildClusterAnalysis(inputs)
	require.NotNil(t, ca)

	var found bool
	for _, p := range ca.Patterns {
		if p.Kind == "failure_correlation" {
			found = true
			assert.Equal(t, "high", p.Severity)
			assert.Contains(t, p.Title, "testing")
			break
		}
	}
	assert.True(t, found, "Expected failure_correlation pattern for testing domain")
}

func TestBuildClusterAnalysis_StyleSkew(t *testing.T) {
	t.Parallel()
	inputs := make([]crossRunInput, 10)
	for i := range inputs {
		inputs[i] = crossRunInput{
			RunID:               int64(i + 1),
			Conclusion:          "success",
			BehaviorFingerprint: &BehaviorFingerprint{ExecutionStyle: "sequential", ResourceProfile: "light"},
		}
	}
	// One outlier
	inputs[9].BehaviorFingerprint = &BehaviorFingerprint{ExecutionStyle: "iterative", ResourceProfile: "heavy"}

	ca := buildClusterAnalysis(inputs)
	require.NotNil(t, ca)

	var found bool
	for _, p := range ca.Patterns {
		if p.Kind == "style_skew" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected style_skew pattern when 90% of runs share same value")
}

func TestBuildClusterAnalysis_WithDuration(t *testing.T) {
	t.Parallel()
	inputs := []crossRunInput{
		{RunID: 1, Conclusion: "success", Duration: 30 * time.Second, Metrics: LogMetrics{TokenUsage: 100}},
		{RunID: 2, Conclusion: "failure", Duration: 90 * time.Second, Metrics: LogMetrics{TokenUsage: 200}},
	}

	ca := buildClusterAnalysis(inputs)
	require.NotNil(t, ca)

	for _, c := range ca.Clusters {
		if c.Dimension == "conclusion" && c.Value == "success" {
			assert.Equal(t, int64(30*time.Second), c.Metrics.AvgDurationNs)
		}
	}
}

func TestBuildClusterAnalysis_HomogeneousInputs(t *testing.T) {
	t.Parallel()
	// All runs identical conclusion — conclusion clusters are trivial (1 value)
	// so they should be filtered out
	inputs := []crossRunInput{
		{RunID: 1, Conclusion: "success"},
		{RunID: 2, Conclusion: "success"},
		{RunID: 3, Conclusion: "success"},
	}

	ca := buildClusterAnalysis(inputs)
	// Might be nil if all dimensions are homogeneous
	if ca != nil {
		for _, c := range ca.Clusters {
			// No single-value dimension should remain
			assert.NotEqual(t, 3, c.Count, "Trivial clusters should be filtered")
		}
	}
}

func TestComputeClusterMetrics(t *testing.T) {
	t.Parallel()
	members := []crossRunInput{
		{Conclusion: "success", Metrics: LogMetrics{TokenUsage: 100, Turns: 3}, Duration: 10 * time.Second, ErrorCount: 0},
		{Conclusion: "success", Metrics: LogMetrics{TokenUsage: 200, Turns: 5}, Duration: 20 * time.Second, ErrorCount: 1},
		{Conclusion: "failure", Metrics: LogMetrics{TokenUsage: 300, Turns: 7}, Duration: 0, ErrorCount: 2},
	}

	m := computeClusterMetrics(members)
	assert.Equal(t, 200, m.AvgTokens) // (100+200+300)/3
	assert.InDelta(t, 5.0, m.AvgTurns, 0.01)
	assert.InDelta(t, 1.0, m.AvgErrorsPerRun, 0.01) // 3 errors / 3 runs
	assert.InDelta(t, 2.0/3.0, m.SuccessRate, 0.01)
	// Duration only counts non-zero: (10+20)/2 = 15s
	assert.Equal(t, int64(15*time.Second), m.AvgDurationNs)
}
