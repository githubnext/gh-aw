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
	Summary           LogsSummary           `json:"summary" console:"title:Workflow Logs Summary"`
	Runs              []RunData             `json:"runs" console:"title:Workflow Logs Overview"`
	ToolUsage         []ToolUsageSummary    `json:"tool_usage,omitempty" console:"title:🛠️  Tool Usage Summary,omitempty"`
	ErrorsAndWarnings []ErrorSummary        `json:"errors_and_warnings,omitempty" console:"title:Errors and Warnings,omitempty"`
	MissingTools      []MissingToolSummary  `json:"missing_tools,omitempty" console:"title:🛠️  Missing Tools Summary,omitempty"`
	MCPFailures       []MCPFailureSummary   `json:"mcp_failures,omitempty" console:"title:⚠️  MCP Server Failures,omitempty"`
	AccessLog         *AccessLogSummary     `json:"access_log,omitempty" console:"title:Access Log Analysis,omitempty"`
	FirewallLog       *FirewallLogSummary   `json:"firewall_log,omitempty" console:"title:🔥 Firewall Log Analysis,omitempty"`
	Continuation      *ContinuationData     `json:"continuation,omitempty" console:"-"`
	LogsLocation      string                `json:"logs_location" console:"-"`
}

// ContinuationData provides parameters to continue querying when timeout is reached
type ContinuationData struct {
	Message      string `json:"message"`
	WorkflowName string `json:"workflow_name,omitempty"`
	Count        int    `json:"count,omitempty"`
	StartDate    string `json:"start_date,omitempty"`
	EndDate      string `json:"end_date,omitempty"`
	Engine       string `json:"engine,omitempty"`
	Branch       string `json:"branch,omitempty"`
	AfterRunID   int64  `json:"after_run_id,omitempty"`
	BeforeRunID  int64  `json:"before_run_id,omitempty"`
	Timeout      int    `json:"timeout,omitempty"`
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
	Agent            string    `json:"agent,omitempty" console:"header:Agent,omitempty"`
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

// ErrorSummary contains aggregated error/warning statistics
type ErrorSummary struct {
	Type         string `json:"type" console:"header:Type"`
	Message      string `json:"message" console:"header:Message,maxlen:80"`
	Count        int    `json:"count" console:"header:Occurrences"`
	Engine       string `json:"engine,omitempty" console:"header:Engine,omitempty"`
	RunID        int64  `json:"run_id" console:"header:Sample Run"`
	RunURL       string `json:"run_url" console:"-"`
	WorkflowName string `json:"workflow_name,omitempty" console:"-"`
	PatternID    string `json:"pattern_id,omitempty" console:"-"`
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

// FirewallLogSummary contains aggregated firewall log data
type FirewallLogSummary struct {
	TotalRequests    int                         `json:"total_requests" console:"header:Total Requests"`
	AllowedRequests  int                         `json:"allowed_requests" console:"header:Allowed"`
	DeniedRequests   int                         `json:"denied_requests" console:"header:Denied"`
	AllowedDomains   []string                    `json:"allowed_domains" console:"-"`
	DeniedDomains    []string                    `json:"denied_domains" console:"-"`
	RequestsByDomain map[string]DomainRequestStats `json:"requests_by_domain,omitempty" console:"-"`
	ByWorkflow       map[string]*FirewallAnalysis  `json:"by_workflow,omitempty" console:"-"`
}

// buildLogsData creates structured logs data from processed runs
func buildLogsData(processedRuns []ProcessedRun, outputDir string, continuation *ContinuationData) LogsData {
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

		// Extract agent/engine ID from aw_info.json
		agentID := ""
		awInfoPath := filepath.Join(run.LogsPath, "aw_info.json")
		if info, err := parseAwInfo(awInfoPath, false); err == nil && info != nil {
			agentID = info.EngineID
		}

		runData := RunData{
			DatabaseID:       run.DatabaseID,
			Number:           run.Number,
			WorkflowName:     run.WorkflowName,
			Agent:            agentID,
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

	// Build combined error and warning summary
	errorsAndWarnings := buildCombinedErrorsSummary(processedRuns)

	// Build missing tools summary
	missingTools := buildMissingToolsSummary(processedRuns)

	// Build MCP failures summary
	mcpFailures := buildMCPFailuresSummary(processedRuns)

	// Build access log summary
	accessLog := buildAccessLogSummary(processedRuns)

	// Build firewall log summary
	firewallLog := buildFirewallLogSummary(processedRuns)

	absOutputDir, _ := filepath.Abs(outputDir)

	return LogsData{
		Summary:           summary,
		Runs:              runs,
		ToolUsage:         toolUsage,
		ErrorsAndWarnings: errorsAndWarnings,
		MissingTools:      missingTools,
		MCPFailures:       mcpFailures,
		AccessLog:         accessLog,
		FirewallLog:       firewallLog,
		Continuation:      continuation,
		LogsLocation:      absOutputDir,
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

// addUniqueWorkflow adds a workflow to the list if it's not already present
func addUniqueWorkflow(workflows []string, workflow string) []string {
	for _, wf := range workflows {
		if wf == workflow {
			return workflows
		}
	}
	return append(workflows, workflow)
}

// buildMissingToolsSummary aggregates missing tools across all runs
func buildMissingToolsSummary(processedRuns []ProcessedRun) []MissingToolSummary {
	toolSummary := make(map[string]*MissingToolSummary)

	for _, pr := range processedRuns {
		for _, tool := range pr.MissingTools {
			if summary, exists := toolSummary[tool.Tool]; exists {
				summary.Count++
				summary.Workflows = addUniqueWorkflow(summary.Workflows, tool.WorkflowName)
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
		// Populate WorkflowsDisplay and FirstReasonDisplay fields for console rendering (truncation handled by maxlen tag)
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
				summary.Workflows = addUniqueWorkflow(summary.Workflows, failure.WorkflowName)
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
		// Populate WorkflowsDisplay field for console rendering (truncation handled by maxlen tag)
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

// buildFirewallLogSummary aggregates firewall log data across all runs
func buildFirewallLogSummary(processedRuns []ProcessedRun) *FirewallLogSummary {
	allAllowedDomains := make(map[string]bool)
	allDeniedDomains := make(map[string]bool)
	allRequestsByDomain := make(map[string]DomainRequestStats)
	byWorkflow := make(map[string]*FirewallAnalysis)
	totalRequests := 0
	allowedRequests := 0
	deniedRequests := 0

	for _, pr := range processedRuns {
		if pr.FirewallAnalysis != nil {
			totalRequests += pr.FirewallAnalysis.TotalRequests
			allowedRequests += pr.FirewallAnalysis.AllowedRequests
			deniedRequests += pr.FirewallAnalysis.DeniedRequests
			byWorkflow[pr.Run.WorkflowName] = pr.FirewallAnalysis

			for _, domain := range pr.FirewallAnalysis.AllowedDomains {
				allAllowedDomains[domain] = true
			}
			for _, domain := range pr.FirewallAnalysis.DeniedDomains {
				allDeniedDomains[domain] = true
			}

			// Aggregate request stats by domain
			for domain, stats := range pr.FirewallAnalysis.RequestsByDomain {
				existing := allRequestsByDomain[domain]
				existing.Allowed += stats.Allowed
				existing.Denied += stats.Denied
				allRequestsByDomain[domain] = existing
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

	return &FirewallLogSummary{
		TotalRequests:    totalRequests,
		AllowedRequests:  allowedRequests,
		DeniedRequests:   deniedRequests,
		AllowedDomains:   allowedDomains,
		DeniedDomains:    deniedDomains,
		RequestsByDomain: allRequestsByDomain,
		ByWorkflow:       byWorkflow,
	}
}

// buildCombinedErrorsSummary aggregates errors and warnings across all runs into a single list
func buildCombinedErrorsSummary(processedRuns []ProcessedRun) []ErrorSummary {
	// Track all errors and warnings in a single map
	combinedMap := make(map[string]*ErrorSummary)

	for _, pr := range processedRuns {
		// Extract metrics from run's logs
		metrics := ExtractLogMetricsFromRun(pr)

		// Get engine information for this run
		engineID := ""
		awInfoPath := filepath.Join(pr.Run.LogsPath, "aw_info.json")
		if info, err := parseAwInfo(awInfoPath, false); err == nil && info != nil {
			engineID = info.EngineID
		}

		// Process each error/warning
		for _, logErr := range metrics.Errors {
			// Create a combined key using type and message
			key := logErr.Type + ":" + logErr.Message

			if existing, exists := combinedMap[key]; exists {
				// Increment count for existing error/warning
				existing.Count++
			} else {
				// Create new entry
				// Capitalize the type for display
				displayType := logErr.Type
				if displayType == "error" {
					displayType = "Error"
				} else if displayType == "warning" {
					displayType = "Warning"
				}

				combinedMap[key] = &ErrorSummary{
					Type:         displayType,
					Message:      logErr.Message,
					Count:        1,
					PatternID:    logErr.PatternID,
					Engine:       engineID,
					RunID:        pr.Run.DatabaseID,
					RunURL:       pr.Run.URL,
					WorkflowName: pr.Run.WorkflowName,
				}
			}
		}
	}

	// Convert map to slice and sort by count (descending), then by type (errors first)
	var combined []ErrorSummary
	for _, summary := range combinedMap {
		combined = append(combined, *summary)
	}
	sort.Slice(combined, func(i, j int) bool {
		// First sort by type (Error before Warning)
		if combined[i].Type != combined[j].Type {
			return combined[i].Type == "Error"
		}
		// Then by count (descending)
		return combined[i].Count > combined[j].Count
	})

	return combined
}

// buildErrorsSummary aggregates errors and warnings across all runs
// Returns two slices: errorsSummary and warningsSummary
// DEPRECATED: Use buildCombinedErrorsSummary instead
func buildErrorsSummary(processedRuns []ProcessedRun) ([]ErrorSummary, []ErrorSummary) {
	// Track errors and warnings separately
	errorMap := make(map[string]*ErrorSummary)
	warningMap := make(map[string]*ErrorSummary)

	for _, pr := range processedRuns {
		// Extract metrics from run's logs
		metrics := ExtractLogMetricsFromRun(pr)

		// Get engine information for this run
		engineID := ""
		awInfoPath := filepath.Join(pr.Run.LogsPath, "aw_info.json")
		if info, err := parseAwInfo(awInfoPath, false); err == nil && info != nil {
			engineID = info.EngineID
		}

		// Process each error/warning
		for _, logErr := range metrics.Errors {
			// Use message as the key for aggregation
			key := logErr.Message

			var targetMap map[string]*ErrorSummary
			if logErr.Type == "error" {
				targetMap = errorMap
			} else {
				targetMap = warningMap
			}

			if existing, exists := targetMap[key]; exists {
				// Increment count for existing error/warning
				existing.Count++
			} else {
				// Capitalize the type for display
				displayType := logErr.Type
				if displayType == "error" {
					displayType = "Error"
				} else if displayType == "warning" {
					displayType = "Warning"
				}

				// Create new entry
				targetMap[key] = &ErrorSummary{
					Type:         displayType,
					Message:      logErr.Message,
					Count:        1,
					PatternID:    logErr.PatternID,
					Engine:       engineID,
					RunID:        pr.Run.DatabaseID,
					RunURL:       pr.Run.URL,
					WorkflowName: pr.Run.WorkflowName,
				}
			}
		}
	}

	// Convert maps to slices and sort by count (descending)
	var errorsSummary []ErrorSummary
	for _, summary := range errorMap {
		errorsSummary = append(errorsSummary, *summary)
	}
	sort.Slice(errorsSummary, func(i, j int) bool {
		return errorsSummary[i].Count > errorsSummary[j].Count
	})

	var warningsSummary []ErrorSummary
	for _, summary := range warningMap {
		warningsSummary = append(warningsSummary, *summary)
	}
	sort.Slice(warningsSummary, func(i, j int) bool {
		return warningsSummary[i].Count > warningsSummary[j].Count
	})

	return errorsSummary, warningsSummary
}

// renderLogsJSON outputs the logs data as JSON
func renderLogsJSON(data LogsData) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// renderLogsConsole outputs the logs data as formatted console output
func renderLogsConsole(data LogsData) {
	// Use unified console rendering for the entire logs data structure
	fmt.Print(console.RenderStruct(data))

	// Display logs location
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Downloaded %d logs to %s", data.Summary.TotalRuns, data.LogsLocation)))
}
