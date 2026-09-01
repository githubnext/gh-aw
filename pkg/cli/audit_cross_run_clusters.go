package cli

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stats"
)

var auditCrossRunClustersLog = logger.New("cli:audit_cross_run_clusters")

// RunCluster groups runs that share a common behavioral dimension (e.g., same
// conclusion, same task domain, or similar execution style).
type RunCluster struct {
	// Dimension is the clustering axis (e.g., "conclusion", "task_domain", "execution_style").
	Dimension string `json:"dimension"`
	// Value is the cluster key within the dimension (e.g., "success", "code_editing", "sequential").
	Value string `json:"value"`
	// RunIDs lists the workflow run IDs in this cluster.
	RunIDs []int64 `json:"run_ids"`
	// Count is len(RunIDs), duplicated for convenience in JSON consumers.
	Count int `json:"count"`
	// Metrics summarizes token/turn/duration statistics within the cluster.
	Metrics ClusterMetrics `json:"metrics"`
}

// ClusterMetrics holds aggregate metrics for a cluster of runs.
type ClusterMetrics struct {
	AvgTokens       int     `json:"avg_tokens"`
	MedianTokens    float64 `json:"median_tokens"`
	StdDevTokens    float64 `json:"stddev_tokens"`
	AvgTurns        float64 `json:"avg_turns"`
	AvgDurationNs   int64   `json:"avg_duration_ns"`
	AvgErrorsPerRun float64 `json:"avg_errors_per_run"`
	SuccessRate     float64 `json:"success_rate"` // Fraction of runs with conclusion=="success"
}

// ClusterPattern describes a detected pattern across clusters (e.g., "failed runs
// use 2.5x more tokens than successful runs").
type ClusterPattern struct {
	Kind        string `json:"kind"`        // Pattern type: "resource_divergence", "failure_correlation", "style_skew"
	Severity    string `json:"severity"`    // "high", "medium", "low"
	Title       string `json:"title"`       // Human-readable title
	Description string `json:"description"` // Full explanation
	Evidence    string `json:"evidence,omitempty"`
}

// ClusterAnalysis is the top-level cluster/pattern output added to CrossRunAuditReport.
type ClusterAnalysis struct {
	Clusters []RunCluster     `json:"clusters"`
	Patterns []ClusterPattern `json:"patterns"`
}

// buildClusterAnalysis derives cluster groupings and cross-cluster patterns from
// per-run inputs. It clusters along four dimensions: conclusion, task domain,
// execution style, and resource profile.
func buildClusterAnalysis(inputs []crossRunInput) *ClusterAnalysis {
	if len(inputs) < 2 {
		return nil
	}
	auditCrossRunClustersLog.Printf("Building cluster analysis from %d inputs", len(inputs))

	clusters := buildRunClusters(inputs)

	// Filter out trivial clusters (single-value dimensions where all runs are in one cluster)
	filtered := make([]RunCluster, 0, len(clusters))
	dimCounts := make(map[string]int)
	for i := range clusters {
		dimCounts[clusters[i].Dimension]++
	}
	for i := range clusters {
		// Keep cluster if its dimension has more than 1 distinct value
		if dimCounts[clusters[i].Dimension] > 1 {
			filtered = append(filtered, clusters[i])
		}
	}

	patterns := detectClusterPatterns(filtered, inputs)

	auditCrossRunClustersLog.Printf("Cluster analysis complete: clusters=%d, patterns=%d", len(filtered), len(patterns))

	if len(filtered) == 0 && len(patterns) == 0 {
		return nil
	}
	return &ClusterAnalysis{
		Clusters: filtered,
		Patterns: patterns,
	}
}

func buildRunClusters(inputs []crossRunInput) []RunCluster {
	clusters := make([]RunCluster, 0)
	clusters = append(clusters, buildDimensionClusters("conclusion", inputs, func(in crossRunInput) string {
		return in.Conclusion
	})...)
	clusters = append(clusters, buildDimensionClusters("task_domain", inputs, func(in crossRunInput) string {
		if in.TaskDomain == nil {
			return "unknown"
		}
		return in.TaskDomain.Name
	})...)
	clusters = append(clusters, buildDimensionClusters("execution_style", inputs, func(in crossRunInput) string {
		if in.BehaviorFingerprint == nil {
			return "unknown"
		}
		return in.BehaviorFingerprint.ExecutionStyle
	})...)
	clusters = append(clusters, buildDimensionClusters("resource_profile", inputs, func(in crossRunInput) string {
		if in.BehaviorFingerprint == nil {
			return "unknown"
		}
		return in.BehaviorFingerprint.ResourceProfile
	})...)
	clusters = append(clusters, buildDimensionClusters("graders", inputs, func(in crossRunInput) string {
		if in.GradersCluster == "" {
			return "absent"
		}
		return in.GradersCluster
	})...)
	clusters = append(clusters, buildDimensionClusters("evals", inputs, func(in crossRunInput) string {
		if in.EvalsCluster == "" {
			return "absent"
		}
		return in.EvalsCluster
	})...)
	return clusters
}

// buildDimensionClusters groups inputs by a keying function and returns clusters.
func buildDimensionClusters(dimension string, inputs []crossRunInput, keyFn func(crossRunInput) string) []RunCluster {
	groups := make(map[string][]crossRunInput)
	for _, in := range inputs {
		key := keyFn(in)
		if key == "" {
			key = "unknown"
		}
		groups[key] = append(groups[key], in)
	}

	keys := sliceutil.SortedKeys(groups)
	clusters := make([]RunCluster, 0, len(keys))
	for _, key := range keys {
		members := groups[key]
		cluster := RunCluster{
			Dimension: dimension,
			Value:     key,
			RunIDs:    make([]int64, 0, len(members)),
			Count:     len(members),
			Metrics:   computeClusterMetrics(members),
		}
		for _, m := range members {
			cluster.RunIDs = append(cluster.RunIDs, m.RunID)
		}
		clusters = append(clusters, cluster)
	}
	return clusters
}

// computeClusterMetrics computes aggregate statistics for a group of runs.
func computeClusterMetrics(members []crossRunInput) ClusterMetrics {
	if len(members) == 0 {
		return ClusterMetrics{}
	}
	var tokenStat, turnStat stats.StatVar
	var totalErrors int
	var totalDurationNs int64
	var durCount int
	var successCount int

	for _, m := range members {
		tokenStat.Add(float64(m.Metrics.TokenUsage))
		turnStat.Add(float64(m.Metrics.Turns))
		totalErrors += m.ErrorCount
		if m.Duration > 0 {
			totalDurationNs += int64(m.Duration)
			durCount++
		}
		if m.Conclusion == "success" {
			successCount++
		}
	}

	cm := ClusterMetrics{
		AvgTokens:       int(tokenStat.Mean()),
		MedianTokens:    tokenStat.Median(),
		StdDevTokens:    tokenStat.SampleStdDev(),
		AvgTurns:        turnStat.Mean(),
		AvgErrorsPerRun: float64(totalErrors) / float64(len(members)),
		SuccessRate:     float64(successCount) / float64(len(members)),
	}
	if durCount > 0 {
		cm.AvgDurationNs = totalDurationNs / int64(durCount)
	}
	return cm
}

// detectClusterPatterns looks for cross-cluster patterns: resource divergence between
// success/failure, execution style skew, etc.
func detectClusterPatterns(clusters []RunCluster, inputs []crossRunInput) []ClusterPattern {
	patterns := make([]ClusterPattern, 0)
	patterns = append(patterns, detectResourceDivergence(clusters)...)
	patterns = append(patterns, detectFailureCorrelation(clusters)...)
	patterns = append(patterns, detectStyleSkew(clusters, inputs)...)

	// Sort by severity (high first)
	slices.SortStableFunc(patterns, func(a, b ClusterPattern) int {
		return clusterSeverityRank(b.Severity) - clusterSeverityRank(a.Severity)
	})
	return patterns
}

// clusterSeverityRank maps severity strings to a numeric rank for sorting.
func clusterSeverityRank(s string) int {
	switch s {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// detectResourceDivergence checks if failed runs consume significantly more tokens
// than successful runs within the "conclusion" dimension.
func detectResourceDivergence(clusters []RunCluster) []ClusterPattern {
	var successCluster *RunCluster
	var failureClusters []*RunCluster
	for i := range clusters {
		if clusters[i].Dimension != "conclusion" {
			continue
		}
		if clusters[i].Value == "success" {
			successCluster = &clusters[i]
			continue
		}
		if isFailureConclusion(clusters[i].Value) {
			failureClusters = append(failureClusters, &clusters[i])
		}
	}
	if successCluster == nil || len(failureClusters) == 0 {
		return nil
	}
	if successCluster.Metrics.AvgTokens == 0 {
		return nil
	}

	var failureTokenTotal int
	var failureCount int
	for _, fc := range failureClusters {
		failureTokenTotal += fc.Metrics.AvgTokens * fc.Count
		failureCount += fc.Count
	}
	if failureCount == 0 {
		return nil
	}

	failureAvgTokens := failureTokenTotal / failureCount
	ratio := float64(failureAvgTokens) / float64(successCluster.Metrics.AvgTokens)
	if ratio < 1.5 {
		return nil
	}

	severity := "low"
	if ratio >= 3.0 {
		severity = "high"
	} else if ratio >= 2.0 {
		severity = "medium"
	}

	return []ClusterPattern{{
		Kind:     "resource_divergence",
		Severity: severity,
		Title:    "Failed runs consume disproportionate resources",
		Description: formatResourceDivergenceDesc(
			failureAvgTokens, successCluster.Metrics.AvgTokens, ratio,
			failureCount, successCluster.Count,
		),
		Evidence: formatResourceDivergenceEvidence(ratio),
	}}
}

func formatResourceDivergenceDesc(failTokens, successTokens int, ratio float64, failCount, successCount int) string {
	return "Failed runs average " + formatTokensCompact(failTokens) +
		" tokens vs " + formatTokensCompact(successTokens) +
		" for successful runs (" + formatRatio(ratio) + "x). " +
		"Cluster sizes: " + formatInt(failCount) + " failed, " + formatInt(successCount) + " successful."
}

func formatResourceDivergenceEvidence(ratio float64) string {
	return "token_ratio=" + formatRatio(ratio) + "x (threshold: 1.5x)"
}

// detectFailureCorrelation checks if a specific task domain or execution style
// correlates strongly with failures.
func detectFailureCorrelation(clusters []RunCluster) []ClusterPattern {
	patterns := make([]ClusterPattern, 0)
	for i := range clusters {
		c := &clusters[i]
		if c.Dimension == "conclusion" || c.Count < 2 {
			continue
		}
		if c.Metrics.SuccessRate > 0.0 {
			continue
		}
		// All runs in this cluster failed — that's a strong signal
		patterns = append(patterns, ClusterPattern{
			Kind:     "failure_correlation",
			Severity: "high",
			Title:    "All runs with " + c.Dimension + "=" + c.Value + " failed",
			Description: "Every run classified as " + c.Dimension + "=\"" + c.Value +
				"\" ended in failure (" + formatInt(c.Count) + " runs). " +
				"This suggests the " + c.Dimension + " dimension strongly predicts failure.",
		})
	}
	return patterns
}

// detectStyleSkew checks for dominant clustering (>80% of runs in one cluster value).
func detectStyleSkew(clusters []RunCluster, inputs []crossRunInput) []ClusterPattern {
	if len(inputs) < 5 {
		return nil
	}
	patterns := make([]ClusterPattern, 0)
	totalRuns := len(inputs)

	// Group clusters by dimension
	dimClusters := make(map[string][]RunCluster)
	for _, c := range clusters {
		dimClusters[c.Dimension] = append(dimClusters[c.Dimension], c)
	}

	for dim, dClusters := range dimClusters {
		if dim == "conclusion" {
			continue // conclusion skew is expected
		}
		for _, c := range dClusters {
			fraction := float64(c.Count) / float64(totalRuns)
			if fraction >= 0.8 && len(dClusters) > 1 {
				patterns = append(patterns, ClusterPattern{
					Kind:     "style_skew",
					Severity: "low",
					Title:    "Dominant " + dim + ": " + c.Value,
					Description: fmt.Sprintf("%s of runs share %s=\"%s\" (%d/%d). Low diversity may indicate homogeneous workload or classification bias.",
						formatPercent(fraction*100), dim, c.Value, c.Count, totalRuns),
				})
			}
		}
	}
	return patterns
}

func formatTokensCompact(tokens int) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000.0)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000.0)
	}
	return strconv.Itoa(tokens)
}

func formatRatio(r float64) string {
	return fmt.Sprintf("%.1f", r)
}

func formatInt(n int) string {
	return strconv.Itoa(n)
}
