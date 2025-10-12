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
	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
)

// LogsData represents the complete structured data for logs output
type LogsData struct {
	Summary      LogsSummary          `json:"summary"`
	Runs         []RunData            `json:"runs"`
	ToolUsage    []ToolUsageSummary   `json:"tool_usage,omitempty"`
	MissingTools []MissingToolSummary `json:"missing_tools,omitempty"`
	MCPFailures  []MCPFailureSummary  `json:"mcp_failures,omitempty"`
	AccessLog    *AccessLogSummary    `json:"access_log,omitempty"`
	LogsLocation string               `json:"logs_location"`
}

// LogsSummary contains aggregate metrics across all runs
type LogsSummary struct {
	TotalRuns         int     `json:"total_runs"`
	TotalDuration     string  `json:"total_duration"`
	TotalTokens       int     `json:"total_tokens"`
	TotalCost         float64 `json:"total_cost"`
	TotalTurns        int     `json:"total_turns"`
	TotalErrors       int     `json:"total_errors"`
	TotalWarnings     int     `json:"total_warnings"`
	TotalMissingTools int     `json:"total_missing_tools"`
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
	MaxOutputSize int    `json:"max_output_size,omitempty" console:"-"`
	MaxDuration   string `json:"max_duration,omitempty" console:"header:Max Duration,omitempty"`
}

// ToolUsageDisplay is a display-optimized version of ToolUsageSummary for console rendering
type ToolUsageDisplay struct {
	Name        string `console:"header:Tool"`
	TotalCalls  string `console:"header:Total Calls"`
	Runs        int    `console:"header:Runs"`
	MaxOutput   string `console:"header:Max Output"`
	MaxDuration string `console:"header:Max Duration,omitempty"`
}

// AccessLogSummary contains aggregated access log analysis
type AccessLogSummary struct {
	TotalRequests  int                        `json:"total_requests"`
	AllowedCount   int                        `json:"allowed_count"`
	DeniedCount    int                        `json:"denied_count"`
	AllowedDomains []string                   `json:"allowed_domains"`
	DeniedDomains  []string                   `json:"denied_domains"`
	ByWorkflow     map[string]*DomainAnalysis `json:"by_workflow,omitempty"`
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
	// Display overview table
	displayLogsOverviewFromData(data, verbose)

	// Display tool usage
	if len(data.ToolUsage) > 0 {
		displayToolUsageFromData(data.ToolUsage, verbose)
	}

	// Display MCP failures
	if len(data.MCPFailures) > 0 {
		displayMCPFailuresFromData(data.MCPFailures, verbose)
	}

	// Display missing tools
	if len(data.MissingTools) > 0 {
		displayMissingToolsFromData(data.MissingTools, verbose)
	}

	// Display access log analysis
	if data.AccessLog != nil {
		displayAccessLogFromData(data.AccessLog, verbose)
	}

	// Display logs location
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Downloaded %d logs to %s", data.Summary.TotalRuns, data.LogsLocation)))
}

// displayLogsOverviewFromData displays the overview table from LogsData
func displayLogsOverviewFromData(data LogsData, verbose bool) {
	headers := []string{"Run ID", "Workflow", "Status", "Duration", "Tokens", "Cost ($)", "Turns", "Errors", "Warnings", "Missing", "Created", "Logs Path"}
	var rows [][]string

	for _, run := range data.Runs {
		// Format cost
		costStr := ""
		if run.EstimatedCost > 0 {
			costStr = fmt.Sprintf("%.3f", run.EstimatedCost)
		}

		// Format tokens
		tokensStr := ""
		if run.TokenUsage > 0 {
			tokensStr = formatNumber(run.TokenUsage)
		}

		// Format turns
		turnsStr := ""
		if run.Turns > 0 {
			turnsStr = fmt.Sprintf("%d", run.Turns)
		}

		// Truncate workflow name if too long
		workflowName := run.WorkflowName
		if len(workflowName) > 20 {
			workflowName = workflowName[:17] + "..."
		}

		// Format relative path
		relPath, _ := filepath.Rel(".", run.LogsPath)

		// Format status - show conclusion directly for completed runs
		statusStr := run.Status
		if run.Status == "completed" && run.Conclusion != "" {
			statusStr = run.Conclusion
		}

		row := []string{
			fmt.Sprintf("%d", run.DatabaseID),
			workflowName,
			statusStr,
			run.Duration,
			tokensStr,
			costStr,
			turnsStr,
			fmt.Sprintf("%d", run.ErrorCount),
			fmt.Sprintf("%d", run.WarningCount),
			fmt.Sprintf("%d", run.MissingToolCount),
			run.CreatedAt.Format("2006-01-02"),
			relPath,
		}
		rows = append(rows, row)
	}

	// Prepare total row
	totalRow := []string{
		fmt.Sprintf("TOTAL (%d runs)", data.Summary.TotalRuns),
		"",
		"",
		data.Summary.TotalDuration,
		formatNumber(data.Summary.TotalTokens),
		fmt.Sprintf("%.3f", data.Summary.TotalCost),
		fmt.Sprintf("%d", data.Summary.TotalTurns),
		fmt.Sprintf("%d", data.Summary.TotalErrors),
		fmt.Sprintf("%d", data.Summary.TotalWarnings),
		fmt.Sprintf("%d", data.Summary.TotalMissingTools),
		"",
		"",
	}

	// Render table using console helper
	tableConfig := console.TableConfig{
		Title:     "Workflow Logs Overview",
		Headers:   headers,
		Rows:      rows,
		ShowTotal: true,
		TotalRow:  totalRow,
	}

	fmt.Print(console.RenderTable(tableConfig))
}

// displayToolUsageFromData displays tool usage statistics
func displayToolUsageFromData(toolUsage []ToolUsageSummary, verbose bool) {
	fmt.Printf("\n%s\n", console.FormatListHeader("🛠️  Tool Usage Summary"))

	// Convert to display-optimized struct
	displayData := make([]ToolUsageDisplay, 0, len(toolUsage))
	for _, tool := range toolUsage {
		outputStr := "N/A"
		if tool.MaxOutputSize > 0 {
			outputStr = pretty.FormatFileSize(int64(tool.MaxOutputSize))
		}
		durationStr := tool.MaxDuration
		if durationStr == "" {
			durationStr = "N/A"
		}

		displayData = append(displayData, ToolUsageDisplay{
			Name:        tool.Name,
			TotalCalls:  formatNumber(tool.TotalCalls),
			Runs:        tool.Runs,
			MaxOutput:   outputStr,
			MaxDuration: durationStr,
		})
	}

	fmt.Print(console.RenderStruct(displayData))
}

// displayMCPFailuresFromData displays MCP failures
func displayMCPFailuresFromData(mcpFailures []MCPFailureSummary, verbose bool) {
	fmt.Printf("\n%s\n", console.FormatListHeader("⚠️  MCP Server Failures"))

	// Convert to display-optimized struct
	displayData := make([]MCPFailureDisplay, 0, len(mcpFailures))
	for _, failure := range mcpFailures {
		workflowList := strings.Join(failure.Workflows, ", ")
		if len(workflowList) > 60 {
			workflowList = workflowList[:57] + "..."
		}

		displayData = append(displayData, MCPFailureDisplay{
			ServerName: failure.ServerName,
			Count:      failure.Count,
			Workflows:  workflowList,
		})
	}

	fmt.Print(console.RenderStruct(displayData))
}

// displayMissingToolsFromData displays missing tools
func displayMissingToolsFromData(missingTools []MissingToolSummary, verbose bool) {
	fmt.Printf("\n%s\n", console.FormatListHeader("🛠️  Missing Tools Summary"))

	// Convert to display-optimized struct
	displayData := make([]MissingToolDisplay, 0, len(missingTools))
	for _, summary := range missingTools {
		workflowList := strings.Join(summary.Workflows, ", ")
		if len(workflowList) > 40 {
			workflowList = workflowList[:37] + "..."
		}

		reason := summary.FirstReason
		if len(reason) > 50 {
			reason = reason[:47] + "..."
		}

		displayData = append(displayData, MissingToolDisplay{
			Tool:        summary.Tool,
			Count:       summary.Count,
			Workflows:   workflowList,
			FirstReason: reason,
		})
	}

	fmt.Print(console.RenderStruct(displayData))

	// Display total summary
	totalReports := 0
	for _, summary := range missingTools {
		totalReports += summary.Count
	}
	fmt.Printf("\n📊 %s: %d unique missing tools reported %d times across workflows\n",
		console.FormatCountMessage("Total"),
		len(missingTools),
		totalReports)
}

// displayAccessLogFromData displays access log analysis
func displayAccessLogFromData(accessLog *AccessLogSummary, verbose bool) {
	fmt.Printf("\n%s\n", console.FormatListHeader("🌐 Network Access Analysis"))

	fmt.Printf("\nTotal Requests: %d (%d allowed, %d denied)\n",
		accessLog.TotalRequests, accessLog.AllowedCount, accessLog.DeniedCount)

	// Display allowed domains
	if len(accessLog.AllowedDomains) > 0 {
		fmt.Printf("\n✅ Allowed Domains (%d):\n", len(accessLog.AllowedDomains))
		for _, domain := range accessLog.AllowedDomains {
			fmt.Printf("   • %s\n", domain)
		}
	}

	// Display denied domains
	if len(accessLog.DeniedDomains) > 0 {
		fmt.Printf("\n❌ Denied Domains (%d):\n", len(accessLog.DeniedDomains))
		for _, domain := range accessLog.DeniedDomains {
			fmt.Printf("   • %s\n", domain)
		}
	}
	fmt.Println()
}
