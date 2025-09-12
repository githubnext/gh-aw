package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// PerformanceMetrics represents detailed performance analysis of a workflow run
type PerformanceMetrics struct {
	RunID           int64                   `json:"run_id"`
	WorkflowName    string                  `json:"workflow_name"`
	Duration        time.Duration           `json:"duration"`
	TokenUsage      int                     `json:"token_usage"`
	EstimatedCost   float64                 `json:"estimated_cost"`
	ToolCallCount   int                     `json:"tool_call_count"`
	ToolCallStats   []workflow.ToolCallInfo `json:"tool_call_stats"`
	EfficiencyScore float64                 `json:"efficiency_score"`
	Bottlenecks     []Bottleneck            `json:"bottlenecks"`
	Optimizations   []Optimization          `json:"optimizations"`
}

// Bottleneck represents a performance issue identified in the workflow
type Bottleneck struct {
	Type        string        `json:"type"` // "tool_call", "processing", "api_rate_limit"
	Description string        `json:"description"`
	Impact      string        `json:"impact"` // "high", "medium", "low"
	Duration    time.Duration `json:"duration"`
	Tool        string        `json:"tool,omitempty"`
}

// Optimization represents a suggested performance improvement
type Optimization struct {
	Type           string  `json:"type"` // "caching", "batching", "early_exit"
	Description    string  `json:"description"`
	ExpectedSaving float64 `json:"expected_saving"` // Percentage improvement expected
	Priority       string  `json:"priority"`        // "high", "medium", "low"
}

// CIDoctorPerformanceReport represents a comprehensive performance analysis
type CIDoctorPerformanceReport struct {
	TotalRuns                   int                  `json:"total_runs"`
	AnalyzedAt                  time.Time            `json:"analyzed_at"`
	AverageMetrics              PerformanceMetrics   `json:"average_metrics"`
	PerformanceMetrics          []PerformanceMetrics `json:"performance_metrics"`
	OptimizationRecommendations []Optimization       `json:"optimization_recommendations"`
	PerformanceInsights         []PerformanceInsight `json:"performance_insights"`
	ToolUsagePatterns           []ToolUsagePattern   `json:"tool_usage_patterns"`
}

// ToolUsagePattern represents patterns in tool usage
type ToolUsagePattern struct {
	ToolName        string        `json:"tool_name"`
	TotalCalls      int           `json:"total_calls"`
	AverageLatency  time.Duration `json:"average_latency"`
	SuccessRate     float64       `json:"success_rate"`
	CommonSequences []string      `json:"common_sequences"`
	Inefficiencies  []string      `json:"inefficiencies"`
}

// PerformanceIssue represents a specific performance problem
type PerformanceIssue struct {
	RunID       int64         `json:"run_id"`
	Type        string        `json:"type"`
	Severity    string        `json:"severity"`
	Description string        `json:"description"`
	Impact      time.Duration `json:"impact"`
	Tool        string        `json:"tool,omitempty"`
}

// PerformanceInsight represents an analysis insight
type PerformanceInsight struct {
	Type        string `json:"type"` // "efficiency", "cost", "bottleneck"
	Title       string `json:"title"`
	Description string `json:"description"`
	Metric      string `json:"metric"`
	Value       string `json:"value"`
	Trend       string `json:"trend"` // "improving", "stable", "degrading"
}

// AnalyzeCIDoctorPerformance analyzes the performance characteristics of CI Doctor workflow runs
func AnalyzeCIDoctorPerformance(runs []ProcessedRun, verbose bool) *CIDoctorPerformanceReport {
	report := &CIDoctorPerformanceReport{
		TotalRuns:          len(runs),
		AnalyzedAt:         time.Now(),
		PerformanceMetrics: make([]PerformanceMetrics, 0, len(runs)),
	}

	var totalDuration time.Duration
	var totalTokens int
	var totalCost float64
	var totalToolCalls int

	toolUsagePattern := make(map[string]*ToolUsagePattern)
	performanceIssues := make([]PerformanceIssue, 0)

	for _, run := range runs {
		metrics := extractPerformanceMetrics(run, verbose)
		report.PerformanceMetrics = append(report.PerformanceMetrics, metrics)

		// Aggregate stats
		totalDuration += metrics.Duration
		totalTokens += metrics.TokenUsage
		totalCost += metrics.EstimatedCost
		totalToolCalls += metrics.ToolCallCount

		// Analyze tool usage patterns
		analyzeToolUsagePatterns(metrics.ToolCallStats, toolUsagePattern)

		// Identify performance issues
		issues := identifyPerformanceIssues(metrics)
		performanceIssues = append(performanceIssues, issues...)
	}

	// Calculate averages and summary statistics
	if len(runs) > 0 {
		report.AverageMetrics = PerformanceMetrics{
			Duration:        totalDuration / time.Duration(len(runs)),
			TokenUsage:      totalTokens / len(runs),
			EstimatedCost:   totalCost / float64(len(runs)),
			ToolCallCount:   totalToolCalls / len(runs),
			EfficiencyScore: calculateAverageEfficiency(report.PerformanceMetrics),
		}
	}

	// Generate optimization recommendations
	report.OptimizationRecommendations = generateOptimizationRecommendations(toolUsagePattern, performanceIssues)

	// Generate performance insights
	report.PerformanceInsights = generatePerformanceInsights(report.PerformanceMetrics, toolUsagePattern)

	// Convert map to slice for ToolUsagePatterns
	report.ToolUsagePatterns = make([]ToolUsagePattern, 0, len(toolUsagePattern))
	for _, pattern := range toolUsagePattern {
		report.ToolUsagePatterns = append(report.ToolUsagePatterns, *pattern)
	}

	return report
}

// extractPerformanceMetrics extracts detailed performance metrics from a processed run
func extractPerformanceMetrics(run ProcessedRun, verbose bool) PerformanceMetrics {
	logMetrics := ExtractLogMetricsFromRun(run)

	// Calculate duration from run timestamps
	var duration time.Duration
	if !run.Run.StartedAt.IsZero() && !run.Run.UpdatedAt.IsZero() {
		duration = run.Run.UpdatedAt.Sub(run.Run.StartedAt)
	}

	// Calculate efficiency score based on multiple factors
	efficiencyScore := calculateEfficiencyScore(logMetrics, duration)

	// Identify bottlenecks
	bottlenecks := identifyBottlenecks(logMetrics, duration)

	// Generate optimization suggestions for this specific run
	optimizations := suggestOptimizations(logMetrics, bottlenecks)

	return PerformanceMetrics{
		RunID:           run.Run.DatabaseID,
		WorkflowName:    run.Run.WorkflowName,
		Duration:        duration,
		TokenUsage:      logMetrics.TokenUsage,
		EstimatedCost:   logMetrics.EstimatedCost,
		ToolCallCount:   len(logMetrics.ToolCalls),
		ToolCallStats:   logMetrics.ToolCalls,
		EfficiencyScore: efficiencyScore,
		Bottlenecks:     bottlenecks,
		Optimizations:   optimizations,
	}
}

// calculateEfficiencyScore calculates an efficiency score (0-100) based on various factors
func calculateEfficiencyScore(metrics workflow.LogMetrics, duration time.Duration) float64 {
	score := 100.0

	// Penalize excessive tool calls
	if len(metrics.ToolCalls) > 20 {
		score -= float64(len(metrics.ToolCalls)-20) * 2
	}

	// Penalize long duration
	if duration > 5*time.Minute {
		score -= (duration.Minutes() - 5) * 5
	}

	// Penalize high token usage
	if metrics.TokenUsage > 50000 {
		score -= float64(metrics.TokenUsage-50000) / 1000
	}

	// Bonus for successful completion with few tool calls
	if len(metrics.ToolCalls) < 10 && duration < 2*time.Minute {
		score += 10
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// identifyBottlenecks identifies performance bottlenecks in the workflow execution
func identifyBottlenecks(metrics workflow.LogMetrics, duration time.Duration) []Bottleneck {
	var bottlenecks []Bottleneck

	// Check for long overall duration
	if duration > 8*time.Minute {
		bottlenecks = append(bottlenecks, Bottleneck{
			Type:        "duration",
			Description: "Workflow execution time exceeds recommended 8 minutes",
			Impact:      "high",
			Duration:    duration,
		})
	}

	// Check for excessive API calls
	apiCalls := 0
	for _, tool := range metrics.ToolCalls {
		if strings.HasPrefix(tool.Name, "mcp__github__") {
			apiCalls += tool.CallCount
		}
	}

	if apiCalls > 30 {
		bottlenecks = append(bottlenecks, Bottleneck{
			Type:        "api_calls",
			Description: fmt.Sprintf("Excessive GitHub API calls (%d)", apiCalls),
			Impact:      "medium",
		})
	}

	// Check for high token usage
	if metrics.TokenUsage > 100000 {
		bottlenecks = append(bottlenecks, Bottleneck{
			Type:        "token_usage",
			Description: fmt.Sprintf("High token usage (%d tokens)", metrics.TokenUsage),
			Impact:      "medium",
		})
	}

	// Check for slow individual tools
	for _, tool := range metrics.ToolCalls {
		if tool.MaxDuration > 30*time.Second {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:        "tool_latency",
				Description: fmt.Sprintf("Tool %s has high latency", tool.Name),
				Impact:      "medium",
				Duration:    tool.MaxDuration,
				Tool:        tool.Name,
			})
		}
	}

	return bottlenecks
}

// suggestOptimizations generates optimization recommendations based on analysis
func suggestOptimizations(metrics workflow.LogMetrics, bottlenecks []Bottleneck) []Optimization {
	var optimizations []Optimization

	// Suggest caching for repeated API calls
	apiToolCounts := make(map[string]int)
	for _, tool := range metrics.ToolCalls {
		if strings.HasPrefix(tool.Name, "mcp__github__") {
			apiToolCounts[tool.Name] += tool.CallCount
		}
	}

	for tool, count := range apiToolCounts {
		if count > 3 {
			optimizations = append(optimizations, Optimization{
				Type:           "caching",
				Description:    fmt.Sprintf("Cache responses for %s (called %d times)", tool, count),
				ExpectedSaving: 20.0,
				Priority:       "high",
			})
		}
	}

	// Suggest batching for multiple similar calls
	if len(apiToolCounts) > 10 {
		optimizations = append(optimizations, Optimization{
			Type:           "batching",
			Description:    "Batch GitHub API calls to reduce network overhead",
			ExpectedSaving: 30.0,
			Priority:       "high",
		})
	}

	// Suggest early exit patterns
	if metrics.TokenUsage > 50000 {
		optimizations = append(optimizations, Optimization{
			Type:           "early_exit",
			Description:    "Implement early exit for known failure patterns",
			ExpectedSaving: 40.0,
			Priority:       "high",
		})
	}

	// Suggest parallel processing
	if len(metrics.ToolCalls) > 15 {
		optimizations = append(optimizations, Optimization{
			Type:           "parallelization",
			Description:    "Run independent investigations in parallel",
			ExpectedSaving: 25.0,
			Priority:       "medium",
		})
	}

	return optimizations
}

// analyzeToolUsagePatterns analyzes patterns in tool usage across runs
func analyzeToolUsagePatterns(toolCalls []workflow.ToolCallInfo, patterns map[string]*ToolUsagePattern) {
	for _, tool := range toolCalls {
		if pattern, exists := patterns[tool.Name]; exists {
			pattern.TotalCalls += tool.CallCount
			// Update running average for latency
			pattern.AverageLatency = (pattern.AverageLatency + tool.MaxDuration) / 2
		} else {
			patterns[tool.Name] = &ToolUsagePattern{
				ToolName:       tool.Name,
				TotalCalls:     tool.CallCount,
				AverageLatency: tool.MaxDuration,
				SuccessRate:    100.0, // Assume success unless we have error data
			}
		}
	}
}

// identifyPerformanceIssues identifies performance issues from metrics
func identifyPerformanceIssues(metrics PerformanceMetrics) []PerformanceIssue {
	var issues []PerformanceIssue

	for _, bottleneck := range metrics.Bottlenecks {
		severity := "medium"
		if bottleneck.Impact == "high" {
			severity = "high"
		}

		issues = append(issues, PerformanceIssue{
			RunID:       metrics.RunID,
			Type:        bottleneck.Type,
			Severity:    severity,
			Description: bottleneck.Description,
			Impact:      bottleneck.Duration,
			Tool:        bottleneck.Tool,
		})
	}

	return issues
}

// generateOptimizationRecommendations generates overall optimization recommendations
func generateOptimizationRecommendations(patterns map[string]*ToolUsagePattern, issues []PerformanceIssue) []Optimization {
	var recommendations []Optimization

	// Analyze tool usage patterns for optimization opportunities
	toolsByUsage := make([]ToolUsagePattern, 0, len(patterns))
	for _, pattern := range patterns {
		toolsByUsage = append(toolsByUsage, *pattern)
	}

	sort.Slice(toolsByUsage, func(i, j int) bool {
		return toolsByUsage[i].TotalCalls > toolsByUsage[j].TotalCalls
	})

	// Recommend optimizations for high-usage tools
	for i, tool := range toolsByUsage {
		if i >= 5 { // Top 5 tools only
			break
		}

		if tool.TotalCalls > 10 {
			recommendations = append(recommendations, Optimization{
				Type:           "caching",
				Description:    fmt.Sprintf("Implement aggressive caching for %s (used %d times)", tool.ToolName, tool.TotalCalls),
				ExpectedSaving: 25.0,
				Priority:       "high",
			})
		}

		if tool.AverageLatency > 20*time.Second {
			recommendations = append(recommendations, Optimization{
				Type:           "timeout_optimization",
				Description:    fmt.Sprintf("Optimize %s calls (avg latency: %v)", tool.ToolName, tool.AverageLatency),
				ExpectedSaving: 15.0,
				Priority:       "medium",
			})
		}
	}

	return recommendations
}

// generatePerformanceInsights generates insights from the performance analysis
func generatePerformanceInsights(metrics []PerformanceMetrics, patterns map[string]*ToolUsagePattern) []PerformanceInsight {
	var insights []PerformanceInsight

	if len(metrics) == 0 {
		return insights
	}

	// Calculate efficiency trends
	avgEfficiency := 0.0
	for _, m := range metrics {
		avgEfficiency += m.EfficiencyScore
	}
	avgEfficiency /= float64(len(metrics))

	insights = append(insights, PerformanceInsight{
		Type:        "efficiency",
		Title:       "CI Doctor Efficiency Score",
		Description: "Average efficiency score across all analyzed runs",
		Metric:      "efficiency_score",
		Value:       fmt.Sprintf("%.1f/100", avgEfficiency),
		Trend:       determineTrend(metrics, "efficiency"),
	})

	// Cost analysis
	avgCost := 0.0
	for _, m := range metrics {
		avgCost += m.EstimatedCost
	}
	avgCost /= float64(len(metrics))

	insights = append(insights, PerformanceInsight{
		Type:        "cost",
		Title:       "Average Investigation Cost",
		Description: "Average cost per CI Doctor investigation",
		Metric:      "cost_usd",
		Value:       fmt.Sprintf("$%.4f", avgCost),
		Trend:       determineTrend(metrics, "cost"),
	})

	// Duration analysis
	avgDuration := time.Duration(0)
	for _, m := range metrics {
		avgDuration += m.Duration
	}
	avgDuration /= time.Duration(len(metrics))

	insights = append(insights, PerformanceInsight{
		Type:        "performance",
		Title:       "Average Investigation Time",
		Description: "Average time to complete an investigation",
		Metric:      "duration_minutes",
		Value:       fmt.Sprintf("%.1f min", avgDuration.Minutes()),
		Trend:       determineTrend(metrics, "duration"),
	})

	return insights
}

// calculateAverageEfficiency calculates the average efficiency score
func calculateAverageEfficiency(metrics []PerformanceMetrics) float64 {
	if len(metrics) == 0 {
		return 0
	}

	total := 0.0
	for _, m := range metrics {
		total += m.EfficiencyScore
	}
	return total / float64(len(metrics))
}

// determineTrend determines if a metric is improving, stable, or degrading
func determineTrend(metrics []PerformanceMetrics, metricType string) string {
	if len(metrics) < 3 {
		return "stable"
	}

	// Simple trend analysis - compare first half vs second half
	mid := len(metrics) / 2
	firstHalf := metrics[:mid]
	secondHalf := metrics[mid:]

	var firstAvg, secondAvg float64

	switch metricType {
	case "efficiency":
		for _, m := range firstHalf {
			firstAvg += m.EfficiencyScore
		}
		firstAvg /= float64(len(firstHalf))

		for _, m := range secondHalf {
			secondAvg += m.EfficiencyScore
		}
		secondAvg /= float64(len(secondHalf))

		if secondAvg > firstAvg+5 {
			return "improving"
		} else if secondAvg < firstAvg-5 {
			return "degrading"
		}

	case "cost":
		for _, m := range firstHalf {
			firstAvg += m.EstimatedCost
		}
		firstAvg /= float64(len(firstHalf))

		for _, m := range secondHalf {
			secondAvg += m.EstimatedCost
		}
		secondAvg /= float64(len(secondHalf))

		if secondAvg < firstAvg*0.9 {
			return "improving"
		} else if secondAvg > firstAvg*1.1 {
			return "degrading"
		}

	case "duration":
		for _, m := range firstHalf {
			firstAvg += m.Duration.Minutes()
		}
		firstAvg /= float64(len(firstHalf))

		for _, m := range secondHalf {
			secondAvg += m.Duration.Minutes()
		}
		secondAvg /= float64(len(secondHalf))

		if secondAvg < firstAvg*0.9 {
			return "improving"
		} else if secondAvg > firstAvg*1.1 {
			return "degrading"
		}
	}

	return "stable"
}
