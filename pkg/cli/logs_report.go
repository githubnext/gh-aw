package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// LogsData represents the complete structured data for logs output
type LogsData struct {
	Summary      LogsSummary          `json:"summary" console:"title:Workflow Logs Summary"`
	Runs         []RunData            `json:"runs" console:"title:Workflow Logs Overview"`
	ToolUsage    []ToolUsageSummary   `json:"tool_usage,omitempty" console:"title:🛠️  Tool Usage Summary,omitempty"`
	MissingTools []MissingToolSummary `json:"missing_tools,omitempty" console:"title:🛠️  Missing Tools Summary,omitempty"`
	MCPFailures  []MCPFailureSummary  `json:"mcp_failures,omitempty" console:"title:⚠️  MCP Server Failures,omitempty"`
	AccessLog    *AccessLogSummary    `json:"access_log,omitempty" console:"title:Access Log Analysis,omitempty"`
	LogsLocation string               `json:"logs_location" console:"-"`
}

// LogsSummary contains aggregate metrics across all runs
type LogsSummary struct {
	TotalRuns         int     `json:"total_runs" console:"header:Total Runs"`
	TotalDuration     string  `json:"total_duration" console:"header:Total Duration"`
	TotalTokens       int     `json:"total_tokens" console:"header:Total Tokens,format:number"`
	TotalCost         float64 `json:"total_cost" console:"header:Total Cost,format:cost"`
	TotalTurns        int     `json:"total_turns" console:"header:Total Turns"`
	TotalErrors       int     `json:"total_errors" console:"header:Total Errors"`
	TotalWarnings     int     `json:"total_warnings" console:"header:Total Warnings"`
	TotalMissingTools int     `json:"total_missing_tools" console:"header:Total Missing Tools"`
}

// RunData contains information about a single workflow run
type RunData struct {
	DatabaseID       int64     `json:"database_id" console:"header:Run ID"`
	Number           int       `json:"number" console:"-"`
	WorkflowName     string    `json:"workflow_name" console:"header:Workflow"`
	Status           string    `json:"status" console:"header:Status"`
	Conclusion       string    `json:"conclusion,omitempty" console:"-"`
	Duration         string    `json:"duration,omitempty" console:"header:Duration,omitempty"`
	TokenUsage       int       `json:"token_usage,omitempty" console:"header:Tokens,format:number,omitempty"`
	EstimatedCost    float64   `json:"estimated_cost,omitempty" console:"header:Cost ($),format:cost,omitempty"`
	Turns            int       `json:"turns,omitempty" console:"header:Turns,omitempty"`
	ErrorCount       int       `json:"error_count" console:"header:Errors"`
	WarningCount     int       `json:"warning_count" console:"header:Warnings"`
	MissingToolCount int       `json:"missing_tool_count" console:"header:Missing"`
	CreatedAt        time.Time `json:"created_at" console:"header:Created"`
	URL              string    `json:"url" console:"-"`
	LogsPath         string    `json:"logs_path" console:"header:Logs Path"`
	Event            string    `json:"event" console:"-"`
	Branch           string    `json:"branch" console:"-"`
}

// ToolUsageSummary contains aggregated tool usage statistics
type ToolUsageSummary struct {
	Name          string `json:"name" console:"header:Tool"`
	TotalCalls    int    `json:"total_calls" console:"header:Total Calls,format:number"`
	Runs          int    `json:"runs" console:"header:Runs"` // Number of runs that used this tool
	MaxOutputSize int    `json:"max_output_size,omitempty" console:"header:Max Output,format:filesize,default:N/A,omitempty"`
	MaxDuration   string `json:"max_duration,omitempty" console:"header:Max Duration,default:N/A,omitempty"`
}

// AccessLogSummary contains aggregated access log analysis
type AccessLogSummary struct {
	TotalRequests  int                        `json:"total_requests" console:"header:Total Requests"`
	AllowedCount   int                        `json:"allowed_count" console:"header:Allowed"`
	DeniedCount    int                        `json:"denied_count" console:"header:Denied"`
	AllowedDomains []string                   `json:"allowed_domains" console:"-"`
	DeniedDomains  []string                   `json:"denied_domains" console:"-"`
	ByWorkflow     map[string]*DomainAnalysis `json:"by_workflow,omitempty" console:"-"`
}

// buildLogsData creates structured logs data from processed runs
func buildLogsData(processedRuns []ProcessedRun, outputDir string, verbose bool) LogsData {
	// Build summary
	var totalDuration time.Duration
	var totalTokens int
	var totalCost float64
	var totalTurns int
	var totalErrors int
	var totalWarnings int
	var totalMissingTools int

	// Build runs data
	var runs []RunData
	for _, pr := range processedRuns {
		run := pr.Run

		if run.Duration > 0 {
			totalDuration += run.Duration
		}
		totalTokens += run.TokenUsage
		totalCost += run.EstimatedCost
		totalTurns += run.Turns
		totalErrors += run.ErrorCount
		totalWarnings += run.WarningCount
		totalMissingTools += run.MissingToolCount

		runData := RunData{
			DatabaseID:       run.DatabaseID,
			Number:           run.Number,
			WorkflowName:     run.WorkflowName,
			Status:           run.Status,
			Conclusion:       run.Conclusion,
			TokenUsage:       run.TokenUsage,
			EstimatedCost:    run.EstimatedCost,
			Turns:            run.Turns,
			ErrorCount:       run.ErrorCount,
			WarningCount:     run.WarningCount,
			MissingToolCount: run.MissingToolCount,
			CreatedAt:        run.CreatedAt,
			URL:              run.URL,
			LogsPath:         run.LogsPath,
			Event:            run.Event,
			Branch:           run.HeadBranch,
		}
		if run.Duration > 0 {
			runData.Duration = formatDuration(run.Duration)
		}
		runs = append(runs, runData)
	}

	summary := LogsSummary{
		TotalRuns:         len(processedRuns),
		TotalDuration:     formatDuration(totalDuration),
		TotalTokens:       totalTokens,
		TotalCost:         totalCost,
		TotalTurns:        totalTurns,
		TotalErrors:       totalErrors,
		TotalWarnings:     totalWarnings,
		TotalMissingTools: totalMissingTools,
	}

	// Build tool usage summary
	toolUsage := buildToolUsageSummary(processedRuns)

	// Build missing tools summary
	missingTools := buildMissingToolsSummary(processedRuns)

	// Build MCP failures summary
	mcpFailures := buildMCPFailuresSummary(processedRuns)

	// Build access log summary
	accessLog := buildAccessLogSummary(processedRuns)

	absOutputDir, _ := filepath.Abs(outputDir)

	return LogsData{
		Summary:      summary,
		Runs:         runs,
		ToolUsage:    toolUsage,
		MissingTools: missingTools,
		MCPFailures:  mcpFailures,
		AccessLog:    accessLog,
		LogsLocation: absOutputDir,
	}
}

// buildToolUsageSummary aggregates tool usage across all runs
func buildToolUsageSummary(processedRuns []ProcessedRun) []ToolUsageSummary {
	toolStats := make(map[string]*ToolUsageSummary)

	for _, pr := range processedRuns {
		// Extract metrics from run's logs
		metrics := ExtractLogMetricsFromRun(pr)

		// Track which runs use each tool
		toolRunTracker := make(map[string]bool)

		for _, toolCall := range metrics.ToolCalls {
			displayKey := workflow.PrettifyToolName(toolCall.Name)
			toolRunTracker[displayKey] = true

			if existing, exists := toolStats[displayKey]; exists {
				existing.TotalCalls += toolCall.CallCount
				if toolCall.MaxOutputSize > existing.MaxOutputSize {
					existing.MaxOutputSize = toolCall.MaxOutputSize
				}
				if toolCall.MaxDuration > 0 {
					maxDur := formatDuration(toolCall.MaxDuration)
					if existing.MaxDuration == "" || toolCall.MaxDuration > parseDurationString(existing.MaxDuration) {
						existing.MaxDuration = maxDur
					}
				}
			} else {
				info := &ToolUsageSummary{
					Name:          displayKey,
					TotalCalls:    toolCall.CallCount,
					MaxOutputSize: toolCall.MaxOutputSize,
					Runs:          0, // Will be incremented below
				}
				if toolCall.MaxDuration > 0 {
					info.MaxDuration = formatDuration(toolCall.MaxDuration)
				}
				toolStats[displayKey] = info
			}
		}

		// Increment run count for tools used in this run
		for toolName := range toolRunTracker {
			if stat, exists := toolStats[toolName]; exists {
				stat.Runs++
			}
		}
	}

	var result []ToolUsageSummary
	for _, info := range toolStats {
		result = append(result, *info)
	}

	// Sort by total calls descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCalls > result[j].TotalCalls
	})

	return result
}

// buildMissingToolsSummary aggregates missing tools across all runs
func buildMissingToolsSummary(processedRuns []ProcessedRun) []MissingToolSummary {
	toolSummary := make(map[string]*MissingToolSummary)

	for _, pr := range processedRuns {
		for _, tool := range pr.MissingTools {
			if summary, exists := toolSummary[tool.Tool]; exists {
				summary.Count++
				// Add workflow if not already in the list
				found := false
				for _, wf := range summary.Workflows {
					if wf == tool.WorkflowName {
						found = true
						break
					}
				}
				if !found {
					summary.Workflows = append(summary.Workflows, tool.WorkflowName)
				}
				summary.RunIDs = append(summary.RunIDs, tool.RunID)
			} else {
				toolSummary[tool.Tool] = &MissingToolSummary{
					Tool:        tool.Tool,
					Count:       1,
					Workflows:   []string{tool.WorkflowName},
					FirstReason: tool.Reason,
					RunIDs:      []int64{tool.RunID},
				}
			}
		}
	}

	var result []MissingToolSummary
	for _, summary := range toolSummary {
		// Populate display fields (truncation handled by console rendering with maxlen tag)
		summary.WorkflowsDisplay = strings.Join(summary.Workflows, ", ")
		summary.FirstReasonDisplay = summary.FirstReason

		result = append(result, *summary)
	}

	// Sort by count descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

// buildMCPFailuresSummary aggregates MCP failures across all runs
func buildMCPFailuresSummary(processedRuns []ProcessedRun) []MCPFailureSummary {
	failureSummary := make(map[string]*MCPFailureSummary)

	for _, pr := range processedRuns {
		for _, failure := range pr.MCPFailures {
			if summary, exists := failureSummary[failure.ServerName]; exists {
				summary.Count++
				// Add workflow if not already in the list
				found := false
				for _, wf := range summary.Workflows {
					if wf == failure.WorkflowName {
						found = true
						break
					}
				}
				if !found {
					summary.Workflows = append(summary.Workflows, failure.WorkflowName)
				}
				summary.RunIDs = append(summary.RunIDs, failure.RunID)
			} else {
				failureSummary[failure.ServerName] = &MCPFailureSummary{
					ServerName: failure.ServerName,
					Count:      1,
					Workflows:  []string{failure.WorkflowName},
					RunIDs:     []int64{failure.RunID},
				}
			}
		}
	}

	var result []MCPFailureSummary
	for _, summary := range failureSummary {
		// Populate display field for workflows
		// Populate display field (truncation handled by console rendering with maxlen tag)
		summary.WorkflowsDisplay = strings.Join(summary.Workflows, ", ")

		result = append(result, *summary)
	}

	// Sort by count descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

// buildAccessLogSummary aggregates access log data across all runs
func buildAccessLogSummary(processedRuns []ProcessedRun) *AccessLogSummary {
	allAllowedDomains := make(map[string]bool)
	allDeniedDomains := make(map[string]bool)
	byWorkflow := make(map[string]*DomainAnalysis)
	totalRequests := 0
	allowedCount := 0
	deniedCount := 0

	for _, pr := range processedRuns {
		if pr.AccessAnalysis != nil {
			totalRequests += pr.AccessAnalysis.TotalRequests
			allowedCount += pr.AccessAnalysis.AllowedCount
			deniedCount += pr.AccessAnalysis.DeniedCount
			byWorkflow[pr.Run.WorkflowName] = pr.AccessAnalysis

			for _, domain := range pr.AccessAnalysis.AllowedDomains {
				allAllowedDomains[domain] = true
			}
			for _, domain := range pr.AccessAnalysis.DeniedDomains {
				allDeniedDomains[domain] = true
			}
		}
	}

	if totalRequests == 0 {
		return nil
	}

	// Convert maps to slices
	var allowedDomains []string
	for domain := range allAllowedDomains {
		allowedDomains = append(allowedDomains, domain)
	}
	sort.Strings(allowedDomains)

	var deniedDomains []string
	for domain := range allDeniedDomains {
		deniedDomains = append(deniedDomains, domain)
	}
	sort.Strings(deniedDomains)

	return &AccessLogSummary{
		TotalRequests:  totalRequests,
		AllowedCount:   allowedCount,
		DeniedCount:    deniedCount,
		AllowedDomains: allowedDomains,
		DeniedDomains:  deniedDomains,
		ByWorkflow:     byWorkflow,
	}
}

// renderLogsJSON outputs the logs data as JSON
func renderLogsJSON(data LogsData) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// renderLogsConsole outputs the logs data as formatted console output
func renderLogsConsole(data LogsData, verbose bool) {
	// Use unified console rendering for the entire logs data structure
	fmt.Print(console.RenderStruct(data))

	// Display logs location
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Downloaded %d logs to %s", data.Summary.TotalRuns, data.LogsLocation)))
}
