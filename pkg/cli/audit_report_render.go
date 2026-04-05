package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/timeutil"
)

// renderJSON outputs the audit data as JSON
func renderJSON(data AuditData) error {
	auditReportLog.Print("Rendering audit report as JSON")
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// renderConsole outputs the audit data as formatted console tables
func renderConsole(data AuditData, logsPath string) {
	auditReportLog.Print("Rendering audit report to console")
	fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Workflow Run Audit Report"))
	fmt.Fprintln(os.Stderr)

	// Overview Section - use new rendering system
	fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Overview"))
	fmt.Fprintln(os.Stderr)
	renderOverview(data.Overview)

	if data.Comparison != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Comparison To Similar Successful Run"))
		fmt.Fprintln(os.Stderr)
		renderAuditComparison(data.Comparison)
	}

	if data.TaskDomain != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Detected Task Domain"))
		fmt.Fprintln(os.Stderr)
		renderTaskDomain(data.TaskDomain)
	}

	if data.BehaviorFingerprint != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Behavioral Fingerprint"))
		fmt.Fprintln(os.Stderr)
		renderBehaviorFingerprint(data.BehaviorFingerprint)
	}

	if len(data.AgenticAssessments) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Agentic Assessment"))
		fmt.Fprintln(os.Stderr)
		renderAgenticAssessments(data.AgenticAssessments)
	}

	// Key Findings Section - NEW
	if len(data.KeyFindings) > 0 {
		auditReportLog.Printf("Rendering %d key findings", len(data.KeyFindings))
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Key Findings"))
		fmt.Fprintln(os.Stderr)
		renderKeyFindings(data.KeyFindings)
	}

	// Recommendations Section - NEW
	if len(data.Recommendations) > 0 {
		auditReportLog.Printf("Rendering %d recommendations", len(data.Recommendations))
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Recommendations"))
		fmt.Fprintln(os.Stderr)
		renderRecommendations(data.Recommendations)
	}

	if len(data.ObservabilityInsights) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Observability Insights"))
		fmt.Fprintln(os.Stderr)
		renderObservabilityInsights(data.ObservabilityInsights)
	}

	// Performance Metrics Section - NEW
	if data.PerformanceMetrics != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Performance Metrics"))
		fmt.Fprintln(os.Stderr)
		renderPerformanceMetrics(data.PerformanceMetrics)
	}

	// Token Usage Section (from firewall proxy)
	if data.FirewallTokenUsage != nil && data.FirewallTokenUsage.TotalRequests > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("📊 Token Usage (Firewall Proxy)"))
		fmt.Fprintln(os.Stderr)
		renderTokenUsage(data.FirewallTokenUsage)
	}

	// GitHub API Rate Limit Usage Section
	if data.GitHubRateLimitUsage != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("🐙 GitHub API Usage"))
		fmt.Fprintln(os.Stderr)
		renderGitHubRateLimitUsage(data.GitHubRateLimitUsage)
	}

	// Engine Configuration Section
	if data.EngineConfig != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Engine Configuration"))
		fmt.Fprintln(os.Stderr)
		renderEngineConfig(data.EngineConfig)
	}

	// Prompt Analysis Section
	if data.PromptAnalysis != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Prompt Analysis"))
		fmt.Fprintln(os.Stderr)
		renderPromptAnalysis(data.PromptAnalysis)
	}

	// Session Analysis Section
	if data.SessionAnalysis != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Session & Agent Performance"))
		fmt.Fprintln(os.Stderr)
		renderSessionAnalysis(data.SessionAnalysis)
	}

	// MCP Server Health Section
	if data.MCPServerHealth != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("MCP Server Health"))
		fmt.Fprintln(os.Stderr)
		renderMCPServerHealth(data.MCPServerHealth)
	}

	// Safe Output Summary Section
	if data.SafeOutputSummary != nil {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Safe Output Summary"))
		fmt.Fprintln(os.Stderr)
		renderSafeOutputSummary(data.SafeOutputSummary)
	}

	// Metrics Section - use new rendering system
	fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Metrics"))
	fmt.Fprintln(os.Stderr)
	renderMetrics(data.Metrics)

	// Jobs Section - use new table rendering
	if len(data.Jobs) > 0 {
		auditReportLog.Printf("Rendering jobs table with %d jobs", len(data.Jobs))
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Jobs"))
		fmt.Fprintln(os.Stderr)
		renderJobsTable(data.Jobs)
	}

	// Downloaded Files Section
	if len(data.DownloadedFiles) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Downloaded Files"))
		fmt.Fprintln(os.Stderr)
		for _, file := range data.DownloadedFiles {
			formattedSize := console.FormatFileSize(file.Size)
			fmt.Fprintf(os.Stderr, "  • %s (%s)", file.Path, formattedSize)
			if file.Description != "" {
				fmt.Fprintf(os.Stderr, " - %s", file.Description)
			}
			fmt.Fprintln(os.Stderr)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Missing Tools Section
	if len(data.MissingTools) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Missing Tools"))
		fmt.Fprintln(os.Stderr)
		for _, tool := range data.MissingTools {
			fmt.Fprintf(os.Stderr, "  • %s\n", tool.Tool)
			fmt.Fprintf(os.Stderr, "    Reason: %s\n", tool.Reason)
			if tool.Alternatives != "" {
				fmt.Fprintf(os.Stderr, "    Alternatives: %s\n", tool.Alternatives)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	// Created Items Section - items created in GitHub by safe output handlers
	if len(data.CreatedItems) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Created Items"))
		fmt.Fprintln(os.Stderr)
		renderCreatedItemsTable(data.CreatedItems)
	}

	// MCP Failures Section
	if len(data.MCPFailures) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("MCP Server Failures"))
		fmt.Fprintln(os.Stderr)
		for _, failure := range data.MCPFailures {
			fmt.Fprintf(os.Stderr, "  • %s: %s\n", failure.ServerName, failure.Status)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Firewall Analysis Section
	if data.FirewallAnalysis != nil && data.FirewallAnalysis.TotalRequests > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Firewall Analysis"))
		fmt.Fprintln(os.Stderr)
		renderFirewallAnalysis(data.FirewallAnalysis)
	}

	// Firewall Policy Analysis Section (enriched with rule attribution)
	if data.PolicyAnalysis != nil && (len(data.PolicyAnalysis.RuleHits) > 0 || data.PolicyAnalysis.PolicySummary != "") {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Firewall Policy Analysis"))
		fmt.Fprintln(os.Stderr)
		renderPolicyAnalysis(data.PolicyAnalysis)
	}

	// Redacted Domains Section
	if data.RedactedDomainsAnalysis != nil && data.RedactedDomainsAnalysis.TotalDomains > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("🔒 Redacted URL Domains"))
		fmt.Fprintln(os.Stderr)
		renderRedactedDomainsAnalysis(data.RedactedDomainsAnalysis)
	}

	// Tool Usage Section - use new table rendering
	if len(data.ToolUsage) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Tool Usage"))
		fmt.Fprintln(os.Stderr)
		renderToolUsageTable(data.ToolUsage)
	}

	// MCP Tool Usage Section - detailed MCP statistics
	if data.MCPToolUsage != nil && len(data.MCPToolUsage.Summary) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("MCP Tool Usage"))
		fmt.Fprintln(os.Stderr)
		renderMCPToolUsageTable(data.MCPToolUsage)
	}

	// Errors and Warnings Section
	if len(data.Errors) > 0 || len(data.Warnings) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Errors and Warnings"))
		fmt.Fprintln(os.Stderr)

		if len(data.Errors) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Errors (%d):", len(data.Errors))))
			for _, err := range data.Errors {
				if err.File != "" && err.Line > 0 {
					fmt.Fprintf(os.Stderr, "    %s:%d: %s\n", filepath.Base(err.File), err.Line, err.Message)
				} else {
					fmt.Fprintf(os.Stderr, "    %s\n", err.Message)
				}
			}
			fmt.Fprintln(os.Stderr)
		}

		if len(data.Warnings) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warnings (%d):", len(data.Warnings))))
			for _, warn := range data.Warnings {
				if warn.File != "" && warn.Line > 0 {
					fmt.Fprintf(os.Stderr, "    %s:%d: %s\n", filepath.Base(warn.File), warn.Line, warn.Message)
				} else {
					fmt.Fprintf(os.Stderr, "    %s\n", warn.Message)
				}
			}
			fmt.Fprintln(os.Stderr)
		}
	}

	// Location
	fmt.Fprintln(os.Stderr, console.FormatSectionHeader("Logs Location"))
	fmt.Fprintln(os.Stderr)
	absPath, _ := filepath.Abs(logsPath)
	fmt.Fprintf(os.Stderr, "  %s\n", absPath)
	fmt.Fprintln(os.Stderr)
}

func renderAuditComparison(comparison *AuditComparisonData) {
	if comparison == nil {
		return
	}

	if !comparison.BaselineFound || comparison.Baseline == nil || comparison.Delta == nil || comparison.Classification == nil {
		fmt.Fprintln(os.Stderr, "  No suitable successful run was available for baseline comparison.")
		fmt.Fprintln(os.Stderr)
		return
	}

	fmt.Fprintf(os.Stderr, "  Baseline: run %d", comparison.Baseline.RunID)
	if comparison.Baseline.Conclusion != "" {
		fmt.Fprintf(os.Stderr, " (%s)", comparison.Baseline.Conclusion)
	}
	fmt.Fprintln(os.Stderr)
	if comparison.Baseline.Selection != "" {
		fmt.Fprintf(os.Stderr, "  Selection: %s\n", strings.ReplaceAll(comparison.Baseline.Selection, "_", " "))
	}
	if len(comparison.Baseline.MatchedOn) > 0 {
		fmt.Fprintf(os.Stderr, "  Matched on: %s\n", strings.Join(comparison.Baseline.MatchedOn, ", "))
	}
	fmt.Fprintf(os.Stderr, "  Classification: %s\n", comparison.Classification.Label)
	fmt.Fprintln(os.Stderr, "  Changes:")

	if comparison.Delta.Turns.Changed {
		fmt.Fprintf(os.Stderr, "    - Turns: %d -> %d\n", comparison.Delta.Turns.Before, comparison.Delta.Turns.After)
	}
	if comparison.Delta.Posture.Changed {
		fmt.Fprintf(os.Stderr, "    - Posture: %s -> %s\n", comparison.Delta.Posture.Before, comparison.Delta.Posture.After)
	}
	if comparison.Delta.BlockedRequests.Changed {
		fmt.Fprintf(os.Stderr, "    - Blocked requests: %d -> %d\n", comparison.Delta.BlockedRequests.Before, comparison.Delta.BlockedRequests.After)
	}
	if comparison.Delta.MCPFailure != nil && comparison.Delta.MCPFailure.NewlyPresent {
		fmt.Fprintf(os.Stderr, "    - New MCP failure: %s\n", strings.Join(comparison.Delta.MCPFailure.After, ", "))
	}
	if len(comparison.Classification.ReasonCodes) == 0 {
		fmt.Fprintln(os.Stderr, "    - No meaningful behavior change from the selected successful baseline")
	}
	if comparison.Recommendation != nil && comparison.Recommendation.Action != "" {
		fmt.Fprintf(os.Stderr, "  Recommended action: %s\n", comparison.Recommendation.Action)
	}
	fmt.Fprintln(os.Stderr)
}

// renderOverview renders the overview section using the new rendering system
func renderOverview(overview OverviewData) {
	// Format Status with optional Conclusion
	statusLine := overview.Status
	if overview.Conclusion != "" && overview.Status == "completed" {
		statusLine = fmt.Sprintf("%s (%s)", overview.Status, overview.Conclusion)
	}

	display := OverviewDisplay{
		RunID:    overview.RunID,
		Workflow: overview.WorkflowName,
		Status:   statusLine,
		Duration: overview.Duration,
		Event:    overview.Event,
		Branch:   overview.Branch,
		URL:      overview.URL,
		Files:    overview.LogsPath,
	}

	fmt.Fprint(os.Stderr, console.RenderStruct(display))
}

// renderMetrics renders the metrics section using the new rendering system
func renderMetrics(metrics MetricsData) {
	fmt.Fprint(os.Stderr, console.RenderStruct(metrics))
}

type taskDomainDisplay struct {
	Domain string `console:"header:Domain"`
	Reason string `console:"header:Reason"`
}

type behaviorFingerprintDisplay struct {
	Execution string `console:"header:Execution"`
	Tools     string `console:"header:Tools"`
	Actuation string `console:"header:Actuation"`
	Resource  string `console:"header:Resources"`
	Dispatch  string `console:"header:Dispatch"`
}

func renderTaskDomain(domain *TaskDomainInfo) {
	if domain == nil {
		return
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(taskDomainDisplay{
		Domain: domain.Label,
		Reason: domain.Reason,
	}))
}

func renderBehaviorFingerprint(fingerprint *BehaviorFingerprint) {
	if fingerprint == nil {
		return
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(behaviorFingerprintDisplay{
		Execution: fingerprint.ExecutionStyle,
		Tools:     fingerprint.ToolBreadth,
		Actuation: fingerprint.ActuationStyle,
		Resource:  fingerprint.ResourceProfile,
		Dispatch:  fingerprint.DispatchMode,
	}))
}

func renderAgenticAssessments(assessments []AgenticAssessment) {
	for _, assessment := range assessments {
		severity := strings.ToUpper(assessment.Severity)
		fmt.Fprintf(os.Stderr, "  [%s] %s\n", severity, assessment.Summary)
		if assessment.Evidence != "" {
			fmt.Fprintf(os.Stderr, "     Evidence: %s\n", assessment.Evidence)
		}
		if assessment.Recommendation != "" {
			fmt.Fprintf(os.Stderr, "     Recommendation: %s\n", assessment.Recommendation)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderJobsTable renders the jobs as a table using console.RenderTable
func renderJobsTable(jobs []JobData) {
	auditReportLog.Printf("Rendering jobs table with %d jobs", len(jobs))
	config := console.TableConfig{
		Headers: []string{"Name", "Status", "Conclusion", "Duration"},
		Rows:    make([][]string, 0, len(jobs)),
	}

	for _, job := range jobs {
		conclusion := job.Conclusion
		if conclusion == "" {
			conclusion = "-"
		}
		duration := job.Duration
		if duration == "" {
			duration = "-"
		}

		row := []string{
			stringutil.Truncate(job.Name, 40),
			job.Status,
			conclusion,
			duration,
		}
		config.Rows = append(config.Rows, row)
	}

	fmt.Fprint(os.Stderr, console.RenderTable(config))
}

// renderToolUsageTable renders tool usage as a table with custom formatting
func renderToolUsageTable(toolUsage []ToolUsageInfo) {
	auditReportLog.Printf("Rendering tool usage table with %d tools", len(toolUsage))
	config := console.TableConfig{
		Headers: []string{"Tool", "Calls", "Max Input", "Max Output", "Max Duration"},
		Rows:    make([][]string, 0, len(toolUsage)),
	}

	for _, tool := range toolUsage {
		inputStr := "N/A"
		if tool.MaxInputSize > 0 {
			inputStr = console.FormatNumber(tool.MaxInputSize)
		}
		outputStr := "N/A"
		if tool.MaxOutputSize > 0 {
			outputStr = console.FormatNumber(tool.MaxOutputSize)
		}
		durationStr := "N/A"
		if tool.MaxDuration != "" {
			durationStr = tool.MaxDuration
		}

		row := []string{
			stringutil.Truncate(tool.Name, 40),
			strconv.Itoa(tool.CallCount),
			inputStr,
			outputStr,
			durationStr,
		}
		config.Rows = append(config.Rows, row)
	}

	fmt.Fprint(os.Stderr, console.RenderTable(config))
}

// renderMCPToolUsageTable renders MCP tool usage with detailed statistics
func renderMCPToolUsageTable(mcpData *MCPToolUsageData) {
	auditReportLog.Printf("Rendering MCP tool usage table with %d tools", len(mcpData.Summary))

	// Render server-level statistics first
	if len(mcpData.Servers) > 0 {
		fmt.Fprintln(os.Stderr, "  Server Statistics:")
		fmt.Fprintln(os.Stderr)

		serverConfig := console.TableConfig{
			Headers: []string{"Server", "Requests", "Tool Calls", "Total Input", "Total Output", "Avg Duration", "Errors"},
			Rows:    make([][]string, 0, len(mcpData.Servers)),
		}

		for _, server := range mcpData.Servers {
			inputStr := console.FormatFileSize(int64(server.TotalInputSize))
			outputStr := console.FormatFileSize(int64(server.TotalOutputSize))
			durationStr := server.AvgDuration
			if durationStr == "" {
				durationStr = "N/A"
			}
			errorStr := strconv.Itoa(server.ErrorCount)
			if server.ErrorCount == 0 {
				errorStr = "-"
			}

			row := []string{
				stringutil.Truncate(server.ServerName, 25),
				strconv.Itoa(server.RequestCount),
				strconv.Itoa(server.ToolCallCount),
				inputStr,
				outputStr,
				durationStr,
				errorStr,
			}
			serverConfig.Rows = append(serverConfig.Rows, row)
		}

		fmt.Fprint(os.Stderr, console.RenderTable(serverConfig))
		fmt.Fprintln(os.Stderr)
	}

	// Render tool-level statistics
	if len(mcpData.Summary) > 0 {
		fmt.Fprintln(os.Stderr, "  Tool Statistics:")
		fmt.Fprintln(os.Stderr)

		toolConfig := console.TableConfig{
			Headers: []string{"Server", "Tool", "Calls", "Total In", "Total Out", "Max In", "Max Out"},
			Rows:    make([][]string, 0, len(mcpData.Summary)),
		}

		for _, tool := range mcpData.Summary {
			totalInStr := console.FormatFileSize(int64(tool.TotalInputSize))
			totalOutStr := console.FormatFileSize(int64(tool.TotalOutputSize))
			maxInStr := console.FormatFileSize(int64(tool.MaxInputSize))
			maxOutStr := console.FormatFileSize(int64(tool.MaxOutputSize))

			row := []string{
				stringutil.Truncate(tool.ServerName, 20),
				stringutil.Truncate(tool.ToolName, 30),
				strconv.Itoa(tool.CallCount),
				totalInStr,
				totalOutStr,
				maxInStr,
				maxOutStr,
			}
			toolConfig.Rows = append(toolConfig.Rows, row)
		}

		fmt.Fprint(os.Stderr, console.RenderTable(toolConfig))
	}

	// Render guard policy summary
	if mcpData.GuardPolicySummary != nil && mcpData.GuardPolicySummary.TotalBlocked > 0 {
		renderGuardPolicySummary(mcpData.GuardPolicySummary)
	}
}

// renderGuardPolicySummary renders the guard policy enforcement summary
func renderGuardPolicySummary(summary *GuardPolicySummary) {
	auditReportLog.Printf("Rendering guard policy summary: %d total blocked", summary.TotalBlocked)

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
		fmt.Sprintf("Guard Policy: %d tool call(s) blocked", summary.TotalBlocked)))
	fmt.Fprintln(os.Stderr)

	// Breakdown by reason
	fmt.Fprintln(os.Stderr, "  Block Reasons:")
	if summary.IntegrityBlocked > 0 {
		fmt.Fprintf(os.Stderr, "    Integrity below minimum : %d\n", summary.IntegrityBlocked)
	}
	if summary.RepoScopeBlocked > 0 {
		fmt.Fprintf(os.Stderr, "    Repository not allowed  : %d\n", summary.RepoScopeBlocked)
	}
	if summary.AccessDenied > 0 {
		fmt.Fprintf(os.Stderr, "    Access denied           : %d\n", summary.AccessDenied)
	}
	if summary.BlockedUserDenied > 0 {
		fmt.Fprintf(os.Stderr, "    Blocked user            : %d\n", summary.BlockedUserDenied)
	}
	if summary.PermissionDenied > 0 {
		fmt.Fprintf(os.Stderr, "    Insufficient permissions: %d\n", summary.PermissionDenied)
	}
	if summary.PrivateRepoDenied > 0 {
		fmt.Fprintf(os.Stderr, "    Private repo denied     : %d\n", summary.PrivateRepoDenied)
	}
	fmt.Fprintln(os.Stderr)

	// Most frequently blocked tools
	if len(summary.BlockedToolCounts) > 0 {
		toolNames := sliceutil.MapToSlice(summary.BlockedToolCounts)
		sort.Slice(toolNames, func(i, j int) bool {
			return summary.BlockedToolCounts[toolNames[i]] > summary.BlockedToolCounts[toolNames[j]]
		})

		toolRows := make([][]string, 0, len(toolNames))
		for _, name := range toolNames {
			toolRows = append(toolRows, []string{name, strconv.Itoa(summary.BlockedToolCounts[name])})
		}
		fmt.Fprint(os.Stderr, console.RenderTable(console.TableConfig{
			Title:   "Most Blocked Tools",
			Headers: []string{"Tool", "Blocked"},
			Rows:    toolRows,
		}))
	}

	// Guard policy event details
	if len(summary.Events) > 0 {
		fmt.Fprintln(os.Stderr)
		eventRows := make([][]string, 0, len(summary.Events))
		for _, evt := range summary.Events {
			message := evt.Message
			if len(message) > 60 {
				message = message[:57] + "..."
			}
			repo := evt.Repository
			if repo == "" {
				repo = "-"
			}
			eventRows = append(eventRows, []string{
				stringutil.Truncate(evt.ServerID, 20),
				stringutil.Truncate(evt.ToolName, 25),
				evt.Reason,
				message,
				repo,
			})
		}
		fmt.Fprint(os.Stderr, console.RenderTable(console.TableConfig{
			Title:   "Guard Policy Events",
			Headers: []string{"Server", "Tool", "Reason", "Message", "Repository"},
			Rows:    eventRows,
		}))
	}
}

// renderFirewallAnalysis renders firewall analysis with summary and domain breakdown
func renderFirewallAnalysis(analysis *FirewallAnalysis) {
	auditReportLog.Printf("Rendering firewall analysis: total=%d, allowed=%d, blocked=%d, allowed_domains=%d, blocked_domains=%d",
		analysis.TotalRequests, analysis.AllowedRequests, analysis.BlockedRequests, len(analysis.AllowedDomains), len(analysis.BlockedDomains))
	// Summary statistics
	fmt.Fprintf(os.Stderr, "  Total Requests : %d\n", analysis.TotalRequests)
	fmt.Fprintf(os.Stderr, "  Allowed        : %d\n", analysis.AllowedRequests)
	fmt.Fprintf(os.Stderr, "  Blocked        : %d\n", analysis.BlockedRequests)
	fmt.Fprintln(os.Stderr)

	// Allowed domains
	if len(analysis.AllowedDomains) > 0 {
		fmt.Fprintln(os.Stderr, "  Allowed Domains:")
		for _, domain := range analysis.AllowedDomains {
			if stats, ok := analysis.RequestsByDomain[domain]; ok {
				fmt.Fprintf(os.Stderr, "    ✓ %s (%d requests)\n", domain, stats.Allowed)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	// Blocked domains
	if len(analysis.BlockedDomains) > 0 {
		fmt.Fprintln(os.Stderr, "  Blocked Domains:")
		for _, domain := range analysis.BlockedDomains {
			if stats, ok := analysis.RequestsByDomain[domain]; ok {
				fmt.Fprintf(os.Stderr, "    ✗ %s (%d requests)\n", domain, stats.Blocked)
			}
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderRedactedDomainsAnalysis renders redacted domains analysis
func renderRedactedDomainsAnalysis(analysis *RedactedDomainsAnalysis) {
	auditReportLog.Printf("Rendering redacted domains analysis: total_domains=%d", analysis.TotalDomains)
	// Summary statistics
	fmt.Fprintf(os.Stderr, "  Total Domains Redacted: %d\n", analysis.TotalDomains)
	fmt.Fprintln(os.Stderr)

	// List domains
	if len(analysis.Domains) > 0 {
		fmt.Fprintln(os.Stderr, "  Redacted Domains:")
		for _, domain := range analysis.Domains {
			fmt.Fprintf(os.Stderr, "    🔒 %s\n", domain)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderCreatedItemsTable renders the list of items created in GitHub by safe output handlers
// as a table with clickable URLs for easy auditing.
func renderCreatedItemsTable(items []CreatedItemReport) {
	auditReportLog.Printf("Rendering created items table with %d item(s)", len(items))
	config := console.TableConfig{
		Headers: []string{"Type", "Repo", "Number", "Temp ID", "URL"},
		Rows:    make([][]string, 0, len(items)),
	}

	for _, item := range items {
		numberStr := ""
		if item.Number > 0 {
			numberStr = strconv.Itoa(item.Number)
		}

		row := []string{
			item.Type,
			item.Repo,
			numberStr,
			item.TemporaryID,
			item.URL,
		}
		config.Rows = append(config.Rows, row)
	}

	fmt.Fprint(os.Stderr, console.RenderTable(config))
	fmt.Fprintln(os.Stderr)
}

// renderKeyFindings renders key findings with colored severity indicators
func renderKeyFindings(findings []Finding) {
	auditReportLog.Printf("Rendering key findings: total=%d", len(findings))
	// Group findings by severity for better presentation
	critical := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "critical" })
	high := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "high" })
	medium := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "medium" })
	low := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "low" })
	info := sliceutil.Filter(findings, func(f Finding) bool {
		return f.Severity != "critical" && f.Severity != "high" && f.Severity != "medium" && f.Severity != "low"
	})

	// Render critical findings first
	for _, finding := range critical {
		fmt.Fprintf(os.Stderr, "  🔴 %s [%s]\n", console.FormatErrorMessage(finding.Title), finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Then high severity
	for _, finding := range high {
		fmt.Fprintf(os.Stderr, "  🟠 %s [%s]\n", console.FormatWarningMessage(finding.Title), finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Medium severity
	for _, finding := range medium {
		fmt.Fprintf(os.Stderr, "  🟡 %s [%s]\n", finding.Title, finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Low severity
	for _, finding := range low {
		fmt.Fprintf(os.Stderr, "  ℹ️  %s [%s]\n", finding.Title, finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Info findings
	for _, finding := range info {
		fmt.Fprintf(os.Stderr, "  ✅ %s [%s]\n", console.FormatSuccessMessage(finding.Title), finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderRecommendations renders actionable recommendations
func renderRecommendations(recommendations []Recommendation) {
	auditReportLog.Printf("Rendering recommendations: total=%d", len(recommendations))
	// Group by priority
	high := sliceutil.Filter(recommendations, func(r Recommendation) bool { return r.Priority == "high" })
	medium := sliceutil.Filter(recommendations, func(r Recommendation) bool { return r.Priority == "medium" })
	low := sliceutil.Filter(recommendations, func(r Recommendation) bool { return r.Priority != "high" && r.Priority != "medium" })

	// Render high priority first
	for i, rec := range high {
		fmt.Fprintf(os.Stderr, "  %d. [HIGH] %s\n", i+1, console.FormatWarningMessage(rec.Action))
		fmt.Fprintf(os.Stderr, "     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Fprintf(os.Stderr, "     Example: %s\n", rec.Example)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Medium priority
	startIdx := len(high) + 1
	for i, rec := range medium {
		fmt.Fprintf(os.Stderr, "  %d. [MEDIUM] %s\n", startIdx+i, rec.Action)
		fmt.Fprintf(os.Stderr, "     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Fprintf(os.Stderr, "     Example: %s\n", rec.Example)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Low priority
	startIdx += len(medium)
	for i, rec := range low {
		fmt.Fprintf(os.Stderr, "  %d. [LOW] %s\n", startIdx+i, rec.Action)
		fmt.Fprintf(os.Stderr, "     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Fprintf(os.Stderr, "     Example: %s\n", rec.Example)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderPerformanceMetrics renders performance metrics
func renderPerformanceMetrics(metrics *PerformanceMetrics) {
	auditReportLog.Printf("Rendering performance metrics: tokens_per_min=%.1f, cost_efficiency=%s, most_used_tool=%s",
		metrics.TokensPerMinute, metrics.CostEfficiency, metrics.MostUsedTool)
	if metrics.TokensPerMinute > 0 {
		fmt.Fprintf(os.Stderr, "  Tokens per Minute: %.1f\n", metrics.TokensPerMinute)
	}

	if metrics.CostEfficiency != "" {
		efficiencyDisplay := metrics.CostEfficiency
		switch metrics.CostEfficiency {
		case "excellent", "good":
			efficiencyDisplay = console.FormatSuccessMessage(metrics.CostEfficiency)
		case "moderate":
			efficiencyDisplay = console.FormatWarningMessage(metrics.CostEfficiency)
		case "poor":
			efficiencyDisplay = console.FormatErrorMessage(metrics.CostEfficiency)
		}
		fmt.Fprintf(os.Stderr, "  Cost Efficiency: %s\n", efficiencyDisplay)
	}

	if metrics.AvgToolDuration != "" {
		fmt.Fprintf(os.Stderr, "  Average Tool Duration: %s\n", metrics.AvgToolDuration)
	}

	if metrics.MostUsedTool != "" {
		fmt.Fprintf(os.Stderr, "  Most Used Tool: %s\n", metrics.MostUsedTool)
	}

	if metrics.NetworkRequests > 0 {
		fmt.Fprintf(os.Stderr, "  Network Requests: %d\n", metrics.NetworkRequests)
	}

	fmt.Fprintln(os.Stderr)
}

// renderPolicyAnalysis renders the enriched firewall policy analysis with rule attribution
func renderPolicyAnalysis(analysis *PolicyAnalysis) {
	auditReportLog.Printf("Rendering policy analysis: rules=%d, denied=%d", len(analysis.RuleHits), analysis.DeniedCount)

	// Policy summary using RenderStruct
	display := PolicySummaryDisplay{
		Policy:        analysis.PolicySummary,
		TotalRequests: analysis.TotalRequests,
		Allowed:       analysis.AllowedCount,
		Denied:        analysis.DeniedCount,
		UniqueDomains: analysis.UniqueDomains,
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(display))
	fmt.Fprintln(os.Stderr)

	// Rule hit table
	if len(analysis.RuleHits) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Policy Rules:"))
		fmt.Fprintln(os.Stderr)

		ruleConfig := console.TableConfig{
			Headers: []string{"Rule", "Action", "Description", "Hits"},
			Rows:    make([][]string, 0, len(analysis.RuleHits)),
		}

		for _, rh := range analysis.RuleHits {
			row := []string{
				stringutil.Truncate(rh.Rule.ID, 30),
				rh.Rule.Action,
				stringutil.Truncate(rh.Rule.Description, 50),
				strconv.Itoa(rh.Hits),
			}
			ruleConfig.Rows = append(ruleConfig.Rows, row)
		}

		fmt.Fprint(os.Stderr, console.RenderTable(ruleConfig))
		fmt.Fprintln(os.Stderr)
	}

	// Denied requests detail
	if len(analysis.DeniedRequests) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Denied Requests (%d):", len(analysis.DeniedRequests))))
		fmt.Fprintln(os.Stderr)

		deniedConfig := console.TableConfig{
			Headers: []string{"Time", "Domain", "Rule", "Reason"},
			Rows:    make([][]string, 0, len(analysis.DeniedRequests)),
		}

		for _, req := range analysis.DeniedRequests {
			timeStr := formatUnixTimestamp(req.Timestamp)
			row := []string{
				timeStr,
				stringutil.Truncate(req.Host, 40),
				stringutil.Truncate(req.RuleID, 25),
				stringutil.Truncate(req.Reason, 40),
			}
			deniedConfig.Rows = append(deniedConfig.Rows, row)
		}

		fmt.Fprint(os.Stderr, console.RenderTable(deniedConfig))
		fmt.Fprintln(os.Stderr)
	}
}

// formatUnixTimestamp converts a Unix timestamp (float64) to a human-readable time string (HH:MM:SS).
func formatUnixTimestamp(ts float64) string {
	if ts <= 0 {
		return "-"
	}
	sec := int64(math.Floor(ts))
	nsec := int64((ts - float64(sec)) * 1e9)
	t := time.Unix(sec, nsec).UTC()
	return t.Format("15:04:05")
}

// renderEngineConfig renders engine configuration details
func renderEngineConfig(config *EngineConfig) {
	if config == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "  Engine ID:         %s\n", config.EngineID)
	if config.EngineName != "" {
		fmt.Fprintf(os.Stderr, "  Engine Name:       %s\n", config.EngineName)
	}
	if config.Model != "" {
		fmt.Fprintf(os.Stderr, "  Model:             %s\n", config.Model)
	}
	if config.Version != "" {
		fmt.Fprintf(os.Stderr, "  Version:           %s\n", config.Version)
	}
	if config.CLIVersion != "" {
		fmt.Fprintf(os.Stderr, "  CLI Version:       %s\n", config.CLIVersion)
	}
	if config.FirewallVersion != "" {
		fmt.Fprintf(os.Stderr, "  Firewall Version:  %s\n", config.FirewallVersion)
	}
	if config.TriggerEvent != "" {
		fmt.Fprintf(os.Stderr, "  Trigger Event:     %s\n", config.TriggerEvent)
	}
	if config.Repository != "" {
		fmt.Fprintf(os.Stderr, "  Repository:        %s\n", config.Repository)
	}
	if len(config.MCPServers) > 0 {
		fmt.Fprintf(os.Stderr, "  MCP Servers:       %s\n", strings.Join(config.MCPServers, ", "))
	}
	fmt.Fprintln(os.Stderr)
}

// renderPromptAnalysis renders prompt analysis metrics
func renderPromptAnalysis(analysis *PromptAnalysis) {
	if analysis == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "  Prompt Size:       %s chars\n", console.FormatNumber(analysis.PromptSize))
	if analysis.PromptFile != "" {
		fmt.Fprintf(os.Stderr, "  Prompt File:       %s\n", analysis.PromptFile)
	}
	fmt.Fprintln(os.Stderr)
}

// renderSessionAnalysis renders session and agent performance metrics
func renderSessionAnalysis(session *SessionAnalysis) {
	if session == nil {
		return
	}
	if session.WallTime != "" {
		fmt.Fprintf(os.Stderr, "  Wall Time:         %s\n", session.WallTime)
	}
	if session.TurnCount > 0 {
		fmt.Fprintf(os.Stderr, "  Turn Count:        %d\n", session.TurnCount)
	}
	if session.AvgTurnDuration != "" {
		fmt.Fprintf(os.Stderr, "  Avg Turn Duration: %s\n", session.AvgTurnDuration)
	}
	if session.TokensPerMinute > 0 {
		fmt.Fprintf(os.Stderr, "  Tokens/Minute:     %.1f\n", session.TokensPerMinute)
	}
	if session.NoopCount > 0 {
		fmt.Fprintf(os.Stderr, "  Noop Count:        %d\n", session.NoopCount)
	}
	if session.TimeoutDetected {
		fmt.Fprintf(os.Stderr, "  Timeout Detected:  %s\n", console.FormatWarningMessage("Yes"))
	} else {
		fmt.Fprintf(os.Stderr, "  Timeout Detected:  %s\n", console.FormatSuccessMessage("No"))
	}
	fmt.Fprintln(os.Stderr)
}

// renderMCPServerHealth renders MCP server health summary
func renderMCPServerHealth(health *MCPServerHealth) {
	if health == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "  %s\n", health.Summary)
	if health.TotalRequests > 0 {
		fmt.Fprintf(os.Stderr, "  Total Requests:    %d\n", health.TotalRequests)
		fmt.Fprintf(os.Stderr, "  Total Errors:      %d\n", health.TotalErrors)
		fmt.Fprintf(os.Stderr, "  Error Rate:        %.1f%%\n", health.ErrorRate)
	}
	fmt.Fprintln(os.Stderr)

	// Server health table
	if len(health.Servers) > 0 {
		config := console.TableConfig{
			Headers: []string{"Server", "Requests", "Tool Calls", "Errors", "Error Rate", "Avg Latency", "Status"},
			Rows:    make([][]string, 0, len(health.Servers)),
		}
		for _, server := range health.Servers {
			row := []string{
				server.ServerName,
				strconv.Itoa(server.RequestCount),
				strconv.Itoa(server.ToolCalls),
				strconv.Itoa(server.ErrorCount),
				server.ErrorRateStr,
				server.AvgLatency,
				server.Status,
			}
			config.Rows = append(config.Rows, row)
		}
		fmt.Fprint(os.Stderr, console.RenderTable(config))
	}

	// Slowest tool calls
	if len(health.SlowestCalls) > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  Slowest Tool Calls:")
		config := console.TableConfig{
			Headers: []string{"Server", "Tool", "Duration"},
			Rows:    make([][]string, 0, len(health.SlowestCalls)),
		}
		for _, call := range health.SlowestCalls {
			row := []string{call.ServerName, call.ToolName, call.Duration}
			config.Rows = append(config.Rows, row)
		}
		fmt.Fprint(os.Stderr, console.RenderTable(config))
	}

	fmt.Fprintln(os.Stderr)
}

// renderSafeOutputSummary renders safe output summary with type breakdown
func renderSafeOutputSummary(summary *SafeOutputSummary) {
	if summary == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "  Total Items:       %d\n", summary.TotalItems)
	fmt.Fprintf(os.Stderr, "  Summary:           %s\n", summary.Summary)
	fmt.Fprintln(os.Stderr)

	// Type breakdown table
	if len(summary.TypeDetails) > 0 {
		config := console.TableConfig{
			Headers: []string{"Type", "Count"},
			Rows:    make([][]string, 0, len(summary.TypeDetails)),
		}
		for _, detail := range summary.TypeDetails {
			row := []string{detail.Type, strconv.Itoa(detail.Count)}
			config.Rows = append(config.Rows, row)
		}
		fmt.Fprint(os.Stderr, console.RenderTable(config))
		fmt.Fprintln(os.Stderr)
	}
}

// renderTokenUsage displays token usage data from the firewall proxy
func renderTokenUsage(summary *TokenUsageSummary) {
	totalTokens := summary.TotalTokens()
	cacheTokens := summary.TotalCacheReadTokens + summary.TotalCacheWriteTokens

	fmt.Fprintf(os.Stderr, "  Total:      %s tokens (%s input, %s output, %s cache)\n",
		console.FormatNumber(totalTokens),
		console.FormatNumber(summary.TotalInputTokens),
		console.FormatNumber(summary.TotalOutputTokens),
		console.FormatNumber(cacheTokens))
	fmt.Fprintf(os.Stderr, "  Requests:   %d (avg %s)\n",
		summary.TotalRequests, timeutil.FormatDurationMs(summary.AvgDurationMs()))
	if summary.CacheEfficiency > 0 {
		fmt.Fprintf(os.Stderr, "  Cache hit:  %.1f%%\n", summary.CacheEfficiency*100)
	}
	fmt.Fprintln(os.Stderr)

	rows := summary.ModelRows()
	if len(rows) > 0 {
		config := console.TableConfig{
			Headers: []string{"Model", "Provider", "Input", "Output", "Cache Read", "Cache Write", "Requests", "Avg Duration"},
			Rows:    make([][]string, 0, len(rows)),
		}
		for _, row := range rows {
			config.Rows = append(config.Rows, []string{
				row.Model,
				row.Provider,
				console.FormatNumber(row.InputTokens),
				console.FormatNumber(row.OutputTokens),
				console.FormatNumber(row.CacheReadTokens),
				console.FormatNumber(row.CacheWriteTokens),
				strconv.Itoa(row.Requests),
				row.AvgDuration,
			})
		}
		fmt.Fprint(os.Stderr, console.RenderTable(config))
		fmt.Fprintln(os.Stderr)
	}
}

// renderGitHubRateLimitUsage displays GitHub API quota consumption for the run.
func renderGitHubRateLimitUsage(usage *GitHubRateLimitUsage) {
	if usage == nil {
		return
	}

	// Summary line
	summary := "Total GitHub API calls: " + console.FormatNumber(usage.TotalRequestsMade)
	if usage.CoreLimit > 0 {
		summary += fmt.Sprintf("  |  Core quota consumed: %s / %s  (remaining: %s)",
			console.FormatNumber(usage.CoreConsumed),
			console.FormatNumber(usage.CoreLimit),
			console.FormatNumber(usage.CoreRemaining),
		)
	}
	fmt.Fprintf(os.Stderr, "  %s\n\n", summary)

	// Per-resource breakdown table (only when there are multiple resources or non-core resources)
	rows := usage.ResourceRows()
	if len(rows) == 0 {
		return
	}
	cfg := console.TableConfig{
		Headers: []string{"Resource", "API Calls", "Quota Consumed", "Remaining", "Limit"},
		Rows:    make([][]string, 0, len(rows)),
	}
	for _, row := range rows {
		cfg.Rows = append(cfg.Rows, []string{
			row.Resource,
			console.FormatNumber(row.RequestsMade),
			console.FormatNumber(row.QuotaConsumed),
			console.FormatNumber(row.FinalRemaining),
			console.FormatNumber(row.Limit),
		})
	}
	fmt.Fprint(os.Stderr, console.RenderTable(cfg))
	fmt.Fprintln(os.Stderr)
}
