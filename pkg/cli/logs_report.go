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
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/timeutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var reportLog = logger.New("cli:logs_report")

// LogsData represents the complete structured data for logs output
type LogsData struct {
	Summary           LogsSummary                `json:"summary" console:"-"`
	Runs              []RunData                  `json:"runs"`
	ToolUsage         []ToolUsageSummary         `json:"tool_usage,omitempty" console:"title:🛠️  Tool Usage Summary,omitempty"`
	ErrorsAndWarnings []ErrorSummary             `json:"errors_and_warnings,omitempty" console:"title:Errors and Warnings,omitempty"`
	MissingTools      []MissingToolSummary       `json:"missing_tools,omitempty" console:"title:🛠️  Missing Tools Summary,omitempty"`
	MCPFailures       []MCPFailureSummary        `json:"mcp_failures,omitempty" console:"title:⚠️  MCP Server Failures,omitempty"`
	AccessLog         *AccessLogSummary          `json:"access_log,omitempty" console:"title:Access Log Analysis,omitempty"`
	FirewallLog       *FirewallLogSummary        `json:"firewall_log,omitempty" console:"title:🔥 Firewall Log Analysis,omitempty"`
	RedactedDomains   *RedactedDomainsLogSummary `json:"redacted_domains,omitempty" console:"title:🔒 Redacted URL Domains,omitempty"`
	Continuation      *ContinuationData          `json:"continuation,omitempty" console:"-"`
	LogsLocation      string                     `json:"logs_location" console:"-"`
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
	WorkflowName     string    `json:"workflow_name" console:"header:Workflow Name"`
	WorkflowID       string    `json:"workflow_id" console:"header:Workflow"`
	WorkflowName     string    `json:"workflow_name" console:"header:Workflow"`
	WorkflowPath     string    `json:"workflow_path,omitempty" console:"-"`
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
	StartedAt        time.Time `json:"started_at,omitempty" console:"-"`
	UpdatedAt        time.Time `json:"updated_at,omitempty" console:"-"`
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
	TotalRequests    int                           `json:"total_requests" console:"header:Total Requests"`
	AllowedRequests  int                           `json:"allowed_requests" console:"header:Allowed"`
	DeniedRequests   int                           `json:"denied_requests" console:"header:Denied"`
	AllowedDomains   []string                      `json:"allowed_domains" console:"-"`
	DeniedDomains    []string                      `json:"denied_domains" console:"-"`
	RequestsByDomain map[string]DomainRequestStats `json:"requests_by_domain,omitempty" console:"-"`
	ByWorkflow       map[string]*FirewallAnalysis  `json:"by_workflow,omitempty" console:"-"`
}

// buildLogsData creates structured logs data from processed runs
func buildLogsData(processedRuns []ProcessedRun, outputDir string, continuation *ContinuationData) LogsData {
	reportLog.Printf("Building logs data from %d processed runs", len(processedRuns))

	// Build summary
	var totalDuration time.Duration
	var totalTokens int
	var totalCost float64
	var totalTurns int
	var totalErrors int
	var totalWarnings int
	var totalMissingTools int

	// Build runs data
	// Initialize as empty slice to ensure JSON marshals to [] instead of null
	runs := make([]RunData, 0, len(processedRuns))
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

		// Extract workflow ID from WorkflowPath
		// WorkflowPath format: .github/workflows/workflow-name.lock.yml
		workflowID := extractWorkflowID(run.WorkflowPath)

		runData := RunData{
			DatabaseID:       run.DatabaseID,
			Number:           run.Number,
			WorkflowName:     run.WorkflowName,
			WorkflowID:       workflowID,
			WorkflowPath:     run.WorkflowPath,
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
			StartedAt:        run.StartedAt,
			UpdatedAt:        run.UpdatedAt,
			URL:              run.URL,
			LogsPath:         run.LogsPath,
			Event:            run.Event,
			Branch:           run.HeadBranch,
		}
		if run.Duration > 0 {
			runData.Duration = timeutil.FormatDuration(run.Duration)
		}
		runs = append(runs, runData)
	}

	summary := LogsSummary{
		TotalRuns:         len(processedRuns),
		TotalDuration:     timeutil.FormatDuration(totalDuration),
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

	// Build redacted domains summary
	redactedDomains := buildRedactedDomainsSummary(processedRuns)

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
		RedactedDomains:   redactedDomains,
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
					maxDur := timeutil.FormatDuration(toolCall.MaxDuration)
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
					info.MaxDuration = timeutil.FormatDuration(toolCall.MaxDuration)
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

// aggregateSummaryItems is a generic helper that aggregates items from processed runs into summaries
// It handles the common pattern of grouping by key, counting occurrences, tracking unique workflows, and collecting run IDs
func aggregateSummaryItems[TItem any, TSummary any](
	processedRuns []ProcessedRun,
	getItems func(ProcessedRun) []TItem,
	getKey func(TItem) string,
	createSummary func(TItem) *TSummary,
	updateSummary func(*TSummary, TItem),
	finalizeSummary func(*TSummary),
) []TSummary {
	summaryMap := make(map[string]*TSummary)

	// Aggregate items from all runs
	for _, pr := range processedRuns {
		for _, item := range getItems(pr) {
			key := getKey(item)
			if summary, exists := summaryMap[key]; exists {
				updateSummary(summary, item)
			} else {
				summaryMap[key] = createSummary(item)
			}
		}
	}

	// Convert map to slice and finalize each summary
	var result []TSummary
	for _, summary := range summaryMap {
		finalizeSummary(summary)
		result = append(result, *summary)
	}

	return result
}

// buildMissingToolsSummary aggregates missing tools across all runs
func buildMissingToolsSummary(processedRuns []ProcessedRun) []MissingToolSummary {
	result := aggregateSummaryItems(
		processedRuns,
		// getItems: extract missing tools from each run
		func(pr ProcessedRun) []MissingToolReport {
			return pr.MissingTools
		},
		// getKey: use tool name as the aggregation key
		func(tool MissingToolReport) string {
			return tool.Tool
		},
		// createSummary: create new summary for first occurrence
		func(tool MissingToolReport) *MissingToolSummary {
			return &MissingToolSummary{
				Tool:        tool.Tool,
				Count:       1,
				Workflows:   []string{tool.WorkflowName},
				FirstReason: tool.Reason,
				RunIDs:      []int64{tool.RunID},
			}
		},
		// updateSummary: update existing summary with new occurrence
		func(summary *MissingToolSummary, tool MissingToolReport) {
			summary.Count++
			summary.Workflows = addUniqueWorkflow(summary.Workflows, tool.WorkflowName)
			summary.RunIDs = append(summary.RunIDs, tool.RunID)
		},
		// finalizeSummary: populate display fields for console rendering
		func(summary *MissingToolSummary) {
			summary.WorkflowsDisplay = strings.Join(summary.Workflows, ", ")
			summary.FirstReasonDisplay = summary.FirstReason
		},
	)

	// Sort by count descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

// buildMCPFailuresSummary aggregates MCP failures across all runs
func buildMCPFailuresSummary(processedRuns []ProcessedRun) []MCPFailureSummary {
	result := aggregateSummaryItems(
		processedRuns,
		// getItems: extract MCP failures from each run
		func(pr ProcessedRun) []MCPFailureReport {
			return pr.MCPFailures
		},
		// getKey: use server name as the aggregation key
		func(failure MCPFailureReport) string {
			return failure.ServerName
		},
		// createSummary: create new summary for first occurrence
		func(failure MCPFailureReport) *MCPFailureSummary {
			return &MCPFailureSummary{
				ServerName: failure.ServerName,
				Count:      1,
				Workflows:  []string{failure.WorkflowName},
				RunIDs:     []int64{failure.RunID},
			}
		},
		// updateSummary: update existing summary with new occurrence
		func(summary *MCPFailureSummary, failure MCPFailureReport) {
			summary.Count++
			summary.Workflows = addUniqueWorkflow(summary.Workflows, failure.WorkflowName)
			summary.RunIDs = append(summary.RunIDs, failure.RunID)
		},
		// finalizeSummary: populate display fields for console rendering
		func(summary *MCPFailureSummary) {
			summary.WorkflowsDisplay = strings.Join(summary.Workflows, ", ")
		},
	)

	// Sort by count descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

// domainAggregation holds the result of aggregating domain statistics
type domainAggregation struct {
	allAllowedDomains map[string]bool
	allDeniedDomains  map[string]bool
	totalRequests     int
	allowedCount      int
	deniedCount       int
}

// aggregateDomainStats aggregates domain statistics across runs
// This is a shared helper for both access log and firewall log summaries
func aggregateDomainStats(processedRuns []ProcessedRun, getAnalysis func(*ProcessedRun) (allowedDomains, deniedDomains []string, totalRequests, allowedCount, deniedCount int, exists bool)) *domainAggregation {
	agg := &domainAggregation{
		allAllowedDomains: make(map[string]bool),
		allDeniedDomains:  make(map[string]bool),
	}

	for _, pr := range processedRuns {
		allowedDomains, deniedDomains, totalRequests, allowedCount, deniedCount, exists := getAnalysis(&pr)
		if !exists {
			continue
		}

		agg.totalRequests += totalRequests
		agg.allowedCount += allowedCount
		agg.deniedCount += deniedCount

		for _, domain := range allowedDomains {
			agg.allAllowedDomains[domain] = true
		}
		for _, domain := range deniedDomains {
			agg.allDeniedDomains[domain] = true
		}
	}

	return agg
}

// convertDomainsToSortedSlices converts domain maps to sorted slices
func convertDomainsToSortedSlices(allowedMap, deniedMap map[string]bool) (allowed, denied []string) {
	for domain := range allowedMap {
		allowed = append(allowed, domain)
	}
	sort.Strings(allowed)

	for domain := range deniedMap {
		denied = append(denied, domain)
	}
	sort.Strings(denied)

	return allowed, denied
}

// buildAccessLogSummary aggregates access log data across all runs
func buildAccessLogSummary(processedRuns []ProcessedRun) *AccessLogSummary {
	byWorkflow := make(map[string]*DomainAnalysis)

	// Use shared aggregation helper
	agg := aggregateDomainStats(processedRuns, func(pr *ProcessedRun) ([]string, []string, int, int, int, bool) {
		if pr.AccessAnalysis == nil {
			return nil, nil, 0, 0, 0, false
		}
		byWorkflow[pr.Run.WorkflowName] = pr.AccessAnalysis
		return pr.AccessAnalysis.AllowedDomains,
			pr.AccessAnalysis.DeniedDomains,
			pr.AccessAnalysis.TotalRequests,
			pr.AccessAnalysis.AllowedCount,
			pr.AccessAnalysis.DeniedCount,
			true
	})

	if agg.totalRequests == 0 {
		return nil
	}

	allowedDomains, deniedDomains := convertDomainsToSortedSlices(agg.allAllowedDomains, agg.allDeniedDomains)

	return &AccessLogSummary{
		TotalRequests:  agg.totalRequests,
		AllowedCount:   agg.allowedCount,
		DeniedCount:    agg.deniedCount,
		AllowedDomains: allowedDomains,
		DeniedDomains:  deniedDomains,
		ByWorkflow:     byWorkflow,
	}
}

// buildFirewallLogSummary aggregates firewall log data across all runs
func buildFirewallLogSummary(processedRuns []ProcessedRun) *FirewallLogSummary {
	allRequestsByDomain := make(map[string]DomainRequestStats)
	byWorkflow := make(map[string]*FirewallAnalysis)

	// Use shared aggregation helper
	agg := aggregateDomainStats(processedRuns, func(pr *ProcessedRun) ([]string, []string, int, int, int, bool) {
		if pr.FirewallAnalysis == nil {
			return nil, nil, 0, 0, 0, false
		}
		byWorkflow[pr.Run.WorkflowName] = pr.FirewallAnalysis

		// Aggregate request stats by domain (firewall-specific)
		for domain, stats := range pr.FirewallAnalysis.RequestsByDomain {
			existing := allRequestsByDomain[domain]
			existing.Allowed += stats.Allowed
			existing.Denied += stats.Denied
			allRequestsByDomain[domain] = existing
		}

		return pr.FirewallAnalysis.AllowedDomains,
			pr.FirewallAnalysis.DeniedDomains,
			pr.FirewallAnalysis.TotalRequests,
			pr.FirewallAnalysis.AllowedRequests,
			pr.FirewallAnalysis.DeniedRequests,
			true
	})

	if agg.totalRequests == 0 {
		return nil
	}

	allowedDomains, deniedDomains := convertDomainsToSortedSlices(agg.allAllowedDomains, agg.allDeniedDomains)

	return &FirewallLogSummary{
		TotalRequests:    agg.totalRequests,
		AllowedRequests:  agg.allowedCount,
		DeniedRequests:   agg.deniedCount,
		AllowedDomains:   allowedDomains,
		DeniedDomains:    deniedDomains,
		RequestsByDomain: allRequestsByDomain,
		ByWorkflow:       byWorkflow,
	}
}

// buildRedactedDomainsSummary aggregates redacted domains data across all runs
func buildRedactedDomainsSummary(processedRuns []ProcessedRun) *RedactedDomainsLogSummary {
	allDomainsSet := make(map[string]bool)
	byWorkflow := make(map[string]*RedactedDomainsAnalysis)
	hasData := false

	for _, pr := range processedRuns {
		if pr.RedactedDomainsAnalysis == nil {
			continue
		}
		hasData = true
		byWorkflow[pr.Run.WorkflowName] = pr.RedactedDomainsAnalysis

		// Collect all unique domains
		for _, domain := range pr.RedactedDomainsAnalysis.Domains {
			allDomainsSet[domain] = true
		}
	}

	if !hasData {
		return nil
	}

	// Convert set to sorted slice
	var allDomains []string
	for domain := range allDomainsSet {
		allDomains = append(allDomains, domain)
	}
	sort.Strings(allDomains)

	return &RedactedDomainsLogSummary{
		TotalDomains: len(allDomains),
		Domains:      allDomains,
		ByWorkflow:   byWorkflow,
	}
}

// logErrorAggregator defines how to aggregate log errors
type logErrorAggregator struct {
	// generateKey creates a unique key for deduplication
	generateKey func(logErr workflow.LogError) string
	// selectMap chooses which map to store the error in (for separate error/warning maps)
	selectMap func(logErr workflow.LogError) map[string]*ErrorSummary
	// sortResults sorts the final aggregated results
	sortResults func(results []ErrorSummary)
}

// aggregateLogErrors extracts common error aggregation logic
// It iterates through processedRuns, extracts metrics, resolves engine info,
// and aggregates errors using the provided aggregator strategy
func aggregateLogErrors(processedRuns []ProcessedRun, agg logErrorAggregator) []ErrorSummary {
	reportLog.Printf("Aggregating log errors from %d runs", len(processedRuns))
	aggregatedMap := make(map[string]*ErrorSummary)

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
			// Generate key for this error
			key := agg.generateKey(logErr)

			// Determine target map (if using multiple maps)
			targetMap := aggregatedMap
			if agg.selectMap != nil {
				targetMap = agg.selectMap(logErr)
			}

			if existing, exists := targetMap[key]; exists {
				// Increment count for existing error/warning
				existing.Count++
			} else {
				// Capitalize the type for display
				displayType := logErr.Type
				switch displayType {
				case "error":
					displayType = "Error"
				case "warning":
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

	// Convert map to slice
	var results []ErrorSummary
	for _, summary := range aggregatedMap {
		results = append(results, *summary)
	}

	// Sort results using provided strategy
	if agg.sortResults != nil {
		agg.sortResults(results)
	}

	return results
}

// buildCombinedErrorsSummary aggregates errors and warnings across all runs into a single list
func buildCombinedErrorsSummary(processedRuns []ProcessedRun) []ErrorSummary {
	agg := logErrorAggregator{
		generateKey: func(logErr workflow.LogError) string {
			// Create a combined key using type and message
			return logErr.Type + ":" + logErr.Message
		},
		selectMap: nil, // Use single map
		sortResults: func(results []ErrorSummary) {
			sort.Slice(results, func(i, j int) bool {
				// First sort by type (Error before Warning)
				if results[i].Type != results[j].Type {
					return results[i].Type == "Error"
				}
				// Then by count (descending)
				return results[i].Count > results[j].Count
			})
		},
	}

	return aggregateLogErrors(processedRuns, agg)
}

// buildErrorsSummary aggregates errors and warnings across all runs
// Returns two slices: errorsSummary and warningsSummary
// DEPRECATED: Use buildCombinedErrorsSummary instead
func buildErrorsSummary(processedRuns []ProcessedRun) ([]ErrorSummary, []ErrorSummary) {
	// Get combined summary
	combined := buildCombinedErrorsSummary(processedRuns)

	// Separate into errors and warnings
	var errorsSummary []ErrorSummary
	var warningsSummary []ErrorSummary

	for _, summary := range combined {
		if summary.Type == "Error" {
			errorsSummary = append(errorsSummary, summary)
		} else {
			warningsSummary = append(warningsSummary, summary)
		}
	}

	// Sort each by count (descending)
	sort.Slice(errorsSummary, func(i, j int) bool {
		return errorsSummary[i].Count > errorsSummary[j].Count
	})
	sort.Slice(warningsSummary, func(i, j int) bool {
		return warningsSummary[i].Count > warningsSummary[j].Count
	})

	return errorsSummary, warningsSummary
}

// renderLogsJSON outputs the logs data as JSON
func renderLogsJSON(data LogsData) error {
	reportLog.Printf("Rendering logs data as JSON: %d runs", data.Summary.TotalRuns)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// writeSummaryFile writes the logs data to a JSON file
// This file contains complete metrics and run data for all downloaded workflow runs.
// It's primarily designed for campaign orchestrators to access workflow execution data
// in subsequent steps without needing GitHub CLI access.
//
// The summary file includes:
//   - Aggregate metrics (total runs, tokens, costs, errors, warnings)
//   - Individual run details with metrics and metadata
//   - Tool usage statistics
//   - Error and warning summaries
//   - Network access logs (if available)
//   - Firewall logs (if available)
func writeSummaryFile(path string, data LogsData, verbose bool) error {
	reportLog.Printf("Writing summary file: path=%s, runs=%d", path, data.Summary.TotalRuns)

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for summary file: %w", err)
	}

	// Marshal to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal logs data to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write summary file: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Wrote summary to %s", path)))
	}

	reportLog.Printf("Successfully wrote summary file: %s", path)
	return nil
}

// renderLogsConsole outputs the logs data as formatted console output
func renderLogsConsole(data LogsData) {
	reportLog.Printf("Rendering logs data to console: %d runs, %d errors, %d warnings",
		data.Summary.TotalRuns, data.Summary.TotalErrors, data.Summary.TotalWarnings)

	// Manually render the runs table with totals as footer
	renderRunsTable(data)

	// Render other sections using RenderStruct
	// Create a struct with only the sections we want to render
	otherSections := struct {
		ToolUsage         []ToolUsageSummary         `console:"title:🛠️  Tool Usage Summary,omitempty"`
		ErrorsAndWarnings []ErrorSummary             `console:"title:Errors and Warnings,omitempty"`
		MissingTools      []MissingToolSummary       `console:"title:🛠️  Missing Tools Summary,omitempty"`
		MCPFailures       []MCPFailureSummary        `console:"title:⚠️  MCP Server Failures,omitempty"`
		AccessLog         *AccessLogSummary          `console:"title:Access Log Analysis,omitempty"`
		FirewallLog       *FirewallLogSummary        `console:"title:🔥 Firewall Log Analysis,omitempty"`
		RedactedDomains   *RedactedDomainsLogSummary `console:"title:🔒 Redacted URL Domains,omitempty"`
	}{
		ToolUsage:         data.ToolUsage,
		ErrorsAndWarnings: data.ErrorsAndWarnings,
		MissingTools:      data.MissingTools,
		MCPFailures:       data.MCPFailures,
		AccessLog:         data.AccessLog,
		FirewallLog:       data.FirewallLog,
		RedactedDomains:   data.RedactedDomains,
	}
	fmt.Print(console.RenderStruct(otherSections))

	// Display logs location
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Downloaded %d logs to %s", data.Summary.TotalRuns, data.LogsLocation)))
}

// renderRunsTable renders the runs table with totals footer
func renderRunsTable(data LogsData) {
	if len(data.Runs) == 0 {
		return
	}

	// Build table headers from RunData struct tags
	headers := []string{"Run ID", "Workflow", "Agent", "Status", "Duration", "Tokens", "Cost ($)", "Turns", "Errors", "Warnings", "Missing", "Created", "Logs Path"}

	// Build table rows
	var rows [][]string
	for _, run := range data.Runs {
		row := []string{
			fmt.Sprintf("%d", run.DatabaseID),
			console.TruncateString(run.WorkflowID, 40),
			run.Agent,
			run.Status,
			run.Duration,
			console.FormatNumberOrEmpty(run.TokenUsage),
			console.FormatCostOrEmpty(run.EstimatedCost),
			console.FormatIntOrEmpty(run.Turns),
			fmt.Sprintf("%d", run.ErrorCount),
			fmt.Sprintf("%d", run.WarningCount),
			fmt.Sprintf("%d", run.MissingToolCount),
			run.CreatedAt.Format("2006-01-02 15:04:05"),
			run.LogsPath,
		}
		rows = append(rows, row)
	}

	// Build total row
	totalRow := []string{
		fmt.Sprintf("TOTAL (%d)", data.Summary.TotalRuns),
		"", // Workflow
		"", // Agent
		"", // Status
		data.Summary.TotalDuration,
		console.FormatNumberOrEmpty(data.Summary.TotalTokens),
		console.FormatCostOrEmpty(data.Summary.TotalCost),
		console.FormatIntOrEmpty(data.Summary.TotalTurns),
		fmt.Sprintf("%d", data.Summary.TotalErrors),
		fmt.Sprintf("%d", data.Summary.TotalWarnings),
		fmt.Sprintf("%d", data.Summary.TotalMissingTools),
		"", // Created
		"", // Logs Path
	}

	// Render table with totals
	config := console.TableConfig{
		Headers:   headers,
		Rows:      rows,
		ShowTotal: true,
		TotalRow:  totalRow,
	}

	fmt.Print(console.RenderTable(config))
}

// extractWorkflowID extracts the workflow ID from a workflow path
// WorkflowPath format: .github/workflows/workflow-name.lock.yml or .github/workflows/workflow-name.campaign.lock.yml
// Returns: workflow-name
func extractWorkflowID(workflowPath string) string {
	if workflowPath == "" {
		return ""
	}

	// Get the basename
	base := filepath.Base(workflowPath)

	// Remove .lock.yml extension
	base = strings.TrimSuffix(base, ".lock.yml")

	// Remove .campaign suffix if present
	base = strings.TrimSuffix(base, ".campaign")

	return base
}
