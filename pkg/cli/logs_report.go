package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/timeutil"
)

var reportLog = logger.New("cli:logs_report")

// LogsData represents the complete structured data for logs output
type LogsData struct {
	Summary           LogsSummary                `json:"summary" console:"title:Workflow Logs Summary"`
	Runs              []RunData                  `json:"runs" console:"title:Workflow Logs Overview"`
	Episodes          []EpisodeData              `json:"episodes" console:"-"`
	Edges             []EpisodeEdge              `json:"edges" console:"-"`
	ToolUsage         []ToolUsageSummary         `json:"tool_usage,omitempty" console:"title:🛠️  Tool Usage Summary,omitempty"`
	MCPToolUsage      *MCPToolUsageSummary       `json:"mcp_tool_usage,omitempty" console:"title:🔧 MCP Tool Usage,omitempty"`
	Observability     []ObservabilityInsight     `json:"observability_insights,omitempty" console:"-"`
	ErrorsAndWarnings []ErrorSummary             `json:"errors_and_warnings,omitempty" console:"title:Errors and Warnings,omitempty"`
	MissingTools      []MissingToolSummary       `json:"missing_tools,omitempty" console:"title:🛠️  Missing Tools Summary,omitempty"`
	MissingData       []MissingDataSummary       `json:"missing_data,omitempty" console:"title:📊 Missing Data Summary,omitempty"`
	MCPFailures       []MCPFailureSummary        `json:"mcp_failures,omitempty" console:"title:⚠️  MCP Server Failures,omitempty"`
	AccessLog         *AccessLogSummary          `json:"access_log,omitempty" console:"title:Access Log Analysis,omitempty"`
	FirewallLog       *FirewallLogSummary        `json:"firewall_log,omitempty" console:"title:🔥 Firewall Log Analysis,omitempty"`
	RedactedDomains   *RedactedDomainsLogSummary `json:"redacted_domains,omitempty" console:"title:🔒 Redacted URL Domains,omitempty"`
	Continuation      *ContinuationData          `json:"continuation,omitempty" console:"-"`
	LogsLocation      string                     `json:"logs_location" console:"-"`
	Message           string                     `json:"message,omitempty" console:"-"`
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
	TotalRuns              int     `json:"total_runs" console:"header:Total Runs"`
	TotalDuration          string  `json:"total_duration" console:"header:Total Duration"`
	TotalTokens            int     `json:"total_tokens" console:"header:Total Tokens,format:number"`
	TotalEffectiveTokens   int     `json:"total_effective_tokens" console:"header:Total Effective Tokens,format:number"`
	TotalCost              float64 `json:"total_cost" console:"header:Total Cost,format:cost"`
	TotalActionMinutes     float64 `json:"total_action_minutes" console:"header:Total Action Minutes"`
	TotalTurns             int     `json:"total_turns" console:"header:Total Turns"`
	TotalErrors            int     `json:"total_errors" console:"header:Total Errors"`
	TotalWarnings          int     `json:"total_warnings" console:"header:Total Warnings"`
	TotalMissingTools      int     `json:"total_missing_tools" console:"header:Total Missing Tools"`
	TotalMissingData       int     `json:"total_missing_data" console:"header:Total Missing Data"`
	TotalSafeItems         int     `json:"total_safe_items" console:"header:Total Safe Items"`
	TotalEpisodes          int     `json:"total_episodes" console:"header:Total Episodes"`
	HighConfidenceEpisodes int     `json:"high_confidence_episodes" console:"header:High Confidence Episodes"`
	TotalGitHubAPICalls    int     `json:"total_github_api_calls,omitempty" console:"header:Total GitHub API Calls,format:number,omitempty"`
}

// RunData contains information about a single workflow run
type RunData struct {
	DatabaseID          int64                `json:"database_id" console:"header:Run ID"`
	Number              int                  `json:"number" console:"-"`
	WorkflowName        string               `json:"workflow_name" console:"header:Workflow"`
	WorkflowPath        string               `json:"workflow_path" console:"-"`
	Agent               string               `json:"agent,omitempty" console:"header:Agent,omitempty"`
	Status              string               `json:"status" console:"header:Status"`
	Conclusion          string               `json:"conclusion,omitempty" console:"-"`
	Classification      string               `json:"classification" console:"-"`
	Duration            string               `json:"duration,omitempty" console:"header:Duration,omitempty"`
	ActionMinutes       float64              `json:"action_minutes,omitempty" console:"header:Action Minutes,omitempty"`
	TokenUsage          int                  `json:"token_usage,omitempty" console:"header:Tokens,format:number,omitempty"`
	EffectiveTokens     int                  `json:"effective_tokens,omitempty" console:"header:Effective Tokens,format:number,omitempty"`
	EstimatedCost       float64              `json:"estimated_cost,omitempty" console:"header:Cost ($),format:cost,omitempty"`
	Turns               int                  `json:"turns,omitempty" console:"header:Turns,omitempty"`
	ErrorCount          int                  `json:"error_count" console:"header:Errors"`
	WarningCount        int                  `json:"warning_count" console:"header:Warnings"`
	MissingToolCount    int                  `json:"missing_tool_count" console:"header:Missing Tools"`
	MissingDataCount    int                  `json:"missing_data_count" console:"header:Missing Data"`
	SafeItemsCount      int                  `json:"safe_items_count,omitempty" console:"header:Safe Items,omitempty"`
	CreatedAt           time.Time            `json:"created_at" console:"header:Created"`
	StartedAt           time.Time            `json:"started_at,omitzero" console:"-"`
	UpdatedAt           time.Time            `json:"updated_at,omitzero" console:"-"`
	URL                 string               `json:"url" console:"-"`
	LogsPath            string               `json:"logs_path" console:"header:Logs Path"`
	Event               string               `json:"event" console:"-"`
	Branch              string               `json:"branch" console:"-"`
	HeadSHA             string               `json:"head_sha,omitempty" console:"-"`
	DisplayTitle        string               `json:"display_title,omitempty" console:"-"`
	Repository          string               `json:"repository,omitempty" console:"-"`
	Organization        string               `json:"organization,omitempty" console:"-"`
	Ref                 string               `json:"ref,omitempty" console:"-"`
	SHA                 string               `json:"sha,omitempty" console:"-"`
	Actor               string               `json:"actor,omitempty" console:"-"`
	RunAttempt          string               `json:"run_attempt,omitempty" console:"-"`
	TargetRepo          string               `json:"target_repo,omitempty" console:"-"`
	EventName           string               `json:"event_name,omitempty" console:"-"`
	Comparison          *AuditComparisonData `json:"comparison,omitempty" console:"-"`
	TaskDomain          *TaskDomainInfo      `json:"task_domain,omitempty" console:"-"`
	BehaviorFingerprint *BehaviorFingerprint `json:"behavior_fingerprint,omitempty" console:"-"`
	AgenticAssessments  []AgenticAssessment  `json:"agentic_assessments,omitempty" console:"-"`
	AwContext           *AwContext           `json:"context,omitempty" console:"-"`                                                        // aw_context data from aw_info.json
	TokenUsageSummary   *TokenUsageSummary   `json:"token_usage_summary,omitempty" console:"-"`                                            // Token usage from firewall proxy
	GitHubAPICalls      int                  `json:"github_api_calls,omitempty" console:"header:GitHub API Calls,format:number,omitempty"` // GitHub API calls made during the run
}

// buildLogsData creates structured logs data from processed runs
func buildLogsData(processedRuns []ProcessedRun, outputDir string, continuation *ContinuationData) LogsData {
	reportLog.Printf("Building logs data from %d processed runs", len(processedRuns))

	// Build summary
	var totalDuration time.Duration
	var totalTokens int
	var totalEffectiveTokens int
	var totalCost float64
	var totalActionMinutes float64
	var totalTurns int
	var totalErrors int
	var totalWarnings int
	var totalMissingTools int
	var totalMissingData int
	var totalSafeItems int
	var totalGitHubAPICalls int

	// Build runs data
	// Initialize as empty slice to ensure JSON marshals to [] instead of null
	runs := make([]RunData, 0, len(processedRuns))
	for _, pr := range processedRuns {
		run := pr.Run

		if run.Duration > 0 {
			totalDuration += run.Duration
		}
		totalTokens += run.TokenUsage
		totalEffectiveTokens += run.EffectiveTokens
		totalCost += run.EstimatedCost
		totalActionMinutes += run.ActionMinutes
		totalTurns += run.Turns
		totalErrors += run.ErrorCount
		totalWarnings += run.WarningCount
		totalMissingTools += run.MissingToolCount
		totalMissingData += run.MissingDataCount
		totalSafeItems += run.SafeItemsCount

		// Accumulate GitHub API call counts
		var gitHubAPICalls int
		if pr.GitHubRateLimitUsage != nil {
			gitHubAPICalls = pr.GitHubRateLimitUsage.TotalRequestsMade
		}
		totalGitHubAPICalls += gitHubAPICalls

		// Extract agent/engine ID and aw_context from aw_info.json.
		agentID := ""
		var awContext *AwContext
		var awInfo *AwInfo
		awInfoPath := filepath.Join(run.LogsPath, "aw_info.json")
		if info, err := parseAwInfo(awInfoPath, false); err == nil && info != nil {
			awInfo = info
			agentID = info.EngineID
			awContext = info.Context
		}
		if awContext == nil {
			awContext = pr.AwContext
		}

		comparison := buildAuditComparisonForProcessedRuns(pr, processedRuns)

		runData := RunData{
			DatabaseID:          run.DatabaseID,
			Number:              run.Number,
			WorkflowName:        run.WorkflowName,
			WorkflowPath:        run.WorkflowPath,
			Agent:               agentID,
			Status:              run.Status,
			Conclusion:          run.Conclusion,
			Classification:      deriveRunClassification(comparison),
			TokenUsage:          run.TokenUsage,
			EffectiveTokens:     run.EffectiveTokens,
			EstimatedCost:       run.EstimatedCost,
			ActionMinutes:       run.ActionMinutes,
			Turns:               run.Turns,
			ErrorCount:          run.ErrorCount,
			WarningCount:        run.WarningCount,
			MissingToolCount:    run.MissingToolCount,
			MissingDataCount:    run.MissingDataCount,
			SafeItemsCount:      run.SafeItemsCount,
			CreatedAt:           run.CreatedAt,
			StartedAt:           run.StartedAt,
			UpdatedAt:           run.UpdatedAt,
			URL:                 run.URL,
			LogsPath:            run.LogsPath,
			Event:               run.Event,
			Branch:              run.HeadBranch,
			HeadSHA:             run.HeadSha,
			DisplayTitle:        run.DisplayTitle,
			Comparison:          comparison,
			TaskDomain:          pr.TaskDomain,
			BehaviorFingerprint: pr.BehaviorFingerprint,
			AgenticAssessments:  pr.AgenticAssessments,
			AwContext:           awContext,
			TokenUsageSummary:   pr.TokenUsage,
			GitHubAPICalls:      gitHubAPICalls,
		}
		if awInfo != nil {
			runData.Repository = awInfo.Repository
			if awInfo.Repository != "" {
				if parts := strings.SplitN(awInfo.Repository, "/", 2); len(parts) == 2 {
					runData.Organization = parts[0]
				}
			}
			runData.Ref = awInfo.Ref
			runData.SHA = awInfo.SHA
			runData.Actor = awInfo.Actor
			runData.RunAttempt = awInfo.RunAttempt
			runData.TargetRepo = awInfo.TargetRepo
			runData.EventName = awInfo.EventName
		}
		if run.Duration > 0 {
			runData.Duration = timeutil.FormatDuration(run.Duration)
		}
		runs = append(runs, runData)
	}

	summary := LogsSummary{
		TotalRuns:            len(processedRuns),
		TotalDuration:        timeutil.FormatDuration(totalDuration),
		TotalTokens:          totalTokens,
		TotalEffectiveTokens: totalEffectiveTokens,
		TotalCost:            totalCost,
		TotalActionMinutes:   totalActionMinutes,
		TotalTurns:           totalTurns,
		TotalErrors:          totalErrors,
		TotalWarnings:        totalWarnings,
		TotalMissingTools:    totalMissingTools,
		TotalMissingData:     totalMissingData,
		TotalSafeItems:       totalSafeItems,
		TotalGitHubAPICalls:  totalGitHubAPICalls,
	}

	episodes, edges := buildEpisodeData(runs, processedRuns)
	for _, episode := range episodes {
		summary.TotalEpisodes++
		if episode.Confidence == "high" {
			summary.HighConfidenceEpisodes++
		}
	}

	// Build tool usage summary
	toolUsage := buildToolUsageSummary(processedRuns)

	// Build combined error and warning summary
	errorsAndWarnings := buildCombinedErrorsSummary(processedRuns)

	// Build missing tools summary
	missingTools := buildMissingToolsSummary(processedRuns)

	// Build missing data summary
	missingData := buildMissingDataSummary(processedRuns)

	// Build MCP failures summary
	mcpFailures := buildMCPFailuresSummary(processedRuns)

	// Build MCP tool usage summary
	mcpToolUsage := buildMCPToolUsageSummary(processedRuns)

	// Build access log summary
	accessLog := buildAccessLogSummary(processedRuns)

	// Build firewall log summary
	firewallLog := buildFirewallLogSummary(processedRuns)

	// Build redacted domains summary
	redactedDomains := buildRedactedDomainsSummary(processedRuns)

	observability := buildLogsObservabilityInsights(processedRuns, toolUsage)
	observability = append(observability, buildDrain3InsightsMultiRun(processedRuns)...)

	absOutputDir, _ := filepath.Abs(outputDir)

	return LogsData{
		Summary:           summary,
		Runs:              runs,
		Episodes:          episodes,
		Edges:             edges,
		ToolUsage:         toolUsage,
		MCPToolUsage:      mcpToolUsage,
		Observability:     observability,
		ErrorsAndWarnings: errorsAndWarnings,
		MissingTools:      missingTools,
		MissingData:       missingData,
		MCPFailures:       mcpFailures,
		AccessLog:         accessLog,
		FirewallLog:       firewallLog,
		RedactedDomains:   redactedDomains,
		Continuation:      continuation,
		LogsLocation:      absOutputDir,
	}
}

// deriveRunClassification maps a run's AuditComparisonData to one of four
// human-readable classification labels:
//
//   - "risky"       – comparison detected a risk signal (e.g. posture change, new MCP failure).
//   - "normal"      – comparison found no risk signals (stable or minor changes).
//   - "baseline"    – no prior successful run was available to compare against;
//     this run acts as its own baseline.
//   - "unclassified" – comparison data is absent or incomplete.
func deriveRunClassification(comparison *AuditComparisonData) string {
	if comparison == nil {
		return "unclassified"
	}
	if !comparison.BaselineFound {
		return "baseline"
	}
	if comparison.Classification == nil {
		return "unclassified"
	}
	if comparison.Classification.Label == "risky" {
		return "risky"
	}
	return "normal"
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
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Wrote summary to "+path))
	}

	reportLog.Printf("Successfully wrote summary file: %s", path)
	return nil
}

// renderLogsConsole outputs the logs data as formatted console output
func renderLogsConsole(data LogsData) {
	reportLog.Printf("Rendering logs data to console: %d runs, %d errors, %d warnings",
		data.Summary.TotalRuns, data.Summary.TotalErrors, data.Summary.TotalWarnings)

	// Use unified console rendering for the entire logs data structure
	fmt.Print(console.RenderStruct(data))

	// Display concise summary at the end
	fmt.Fprintln(os.Stderr, "") // Blank line for spacing
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Downloaded %d workflow logs to %s", data.Summary.TotalRuns, data.LogsLocation)))

	// Show key metrics in a concise format
	if data.Summary.TotalErrors > 0 || data.Summary.TotalWarnings > 0 {
		fmt.Fprintf(os.Stderr, "  %s %d errors, %d warnings across %d runs\n",
			console.FormatInfoMessage("•"),
			data.Summary.TotalErrors,
			data.Summary.TotalWarnings,
			data.Summary.TotalRuns)
	}

	if len(data.ToolUsage) > 0 {
		fmt.Fprintf(os.Stderr, "  %s %d unique tools used\n",
			console.FormatInfoMessage("•"),
			len(data.ToolUsage))
	}

	if len(data.Observability) > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Observability Insights"))
		fmt.Fprintln(os.Stderr)
		renderObservabilityInsights(data.Observability)
	}
}
