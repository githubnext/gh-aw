package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/timeutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var auditReportLog = logger.New("cli:audit_report")

// AuditData represents the complete structured audit data for a workflow run
type AuditData struct {
	Overview                OverviewData             `json:"overview"`
	Metrics                 MetricsData              `json:"metrics"`
	KeyFindings             []Finding                `json:"key_findings,omitempty"`
	Recommendations         []Recommendation         `json:"recommendations,omitempty"`
	FailureAnalysis         *FailureAnalysis         `json:"failure_analysis,omitempty"`
	PerformanceMetrics      *PerformanceMetrics      `json:"performance_metrics,omitempty"`
	Jobs                    []JobData                `json:"jobs,omitempty"`
	DownloadedFiles         []FileInfo               `json:"downloaded_files"`
	MissingTools            []MissingToolReport      `json:"missing_tools,omitempty"`
	Noops                   []NoopReport             `json:"noops,omitempty"`
	MCPFailures             []MCPFailureReport       `json:"mcp_failures,omitempty"`
	FirewallAnalysis        *FirewallAnalysis        `json:"firewall_analysis,omitempty"`
	RedactedDomainsAnalysis *RedactedDomainsAnalysis `json:"redacted_domains_analysis,omitempty"`
	Errors                  []ErrorInfo              `json:"errors,omitempty"`
	Warnings                []ErrorInfo              `json:"warnings,omitempty"`
	ToolUsage               []ToolUsageInfo          `json:"tool_usage,omitempty"`
}

// Finding represents a key insight discovered during audit
type Finding struct {
	Category    string `json:"category"`         // e.g., "error", "performance", "cost", "tooling"
	Severity    string `json:"severity"`         // "critical", "high", "medium", "low", "info"
	Title       string `json:"title"`            // Brief title
	Description string `json:"description"`      // Detailed description
	Impact      string `json:"impact,omitempty"` // What impact this has
}

// Recommendation represents an actionable suggestion
type Recommendation struct {
	Priority string `json:"priority"`          // "high", "medium", "low"
	Action   string `json:"action"`            // What to do
	Reason   string `json:"reason"`            // Why to do it
	Example  string `json:"example,omitempty"` // Example of how to implement
}

// FailureAnalysis provides structured analysis for failed workflows
type FailureAnalysis struct {
	PrimaryFailure string   `json:"primary_failure"`      // Main reason for failure
	FailedJobs     []string `json:"failed_jobs"`          // List of failed job names
	ErrorSummary   string   `json:"error_summary"`        // Summary of errors
	RootCause      string   `json:"root_cause,omitempty"` // Identified root cause if determinable
}

// PerformanceMetrics provides aggregated performance statistics
type PerformanceMetrics struct {
	TokensPerMinute float64 `json:"tokens_per_minute,omitempty"`
	CostEfficiency  string  `json:"cost_efficiency,omitempty"` // e.g., "good", "poor"
	AvgToolDuration string  `json:"avg_tool_duration,omitempty"`
	MostUsedTool    string  `json:"most_used_tool,omitempty"`
	NetworkRequests int     `json:"network_requests,omitempty"`
}

// OverviewData contains basic information about the workflow run
type OverviewData struct {
	RunID        int64     `json:"run_id" console:"header:Run ID"`
	WorkflowName string    `json:"workflow_name" console:"header:Workflow"`
	Status       string    `json:"status" console:"header:Status"`
	Conclusion   string    `json:"conclusion,omitempty" console:"header:Conclusion,omitempty"`
	CreatedAt    time.Time `json:"created_at" console:"header:Created At"`
	StartedAt    time.Time `json:"started_at,omitempty" console:"header:Started At,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty" console:"header:Updated At,omitempty"`
	Duration     string    `json:"duration,omitempty" console:"header:Duration,omitempty"`
	Event        string    `json:"event" console:"header:Event"`
	Branch       string    `json:"branch" console:"header:Branch"`
	URL          string    `json:"url" console:"header:URL"`
}

// MetricsData contains execution metrics
type MetricsData struct {
	TokenUsage    int     `json:"token_usage,omitempty" console:"header:Token Usage,format:number,omitempty"`
	EstimatedCost float64 `json:"estimated_cost,omitempty" console:"header:Estimated Cost,format:cost,omitempty"`
	Turns         int     `json:"turns,omitempty" console:"header:Turns,omitempty"`
	ErrorCount    int     `json:"error_count" console:"header:Errors"`
	WarningCount  int     `json:"warning_count" console:"header:Warnings"`
}

// JobData contains information about individual jobs
type JobData struct {
	Name       string `json:"name" console:"header:Name"`
	Status     string `json:"status" console:"header:Status"`
	Conclusion string `json:"conclusion,omitempty" console:"header:Conclusion,omitempty"`
	Duration   string `json:"duration,omitempty" console:"header:Duration,omitempty"`
}

// FileInfo contains information about downloaded artifact files
type FileInfo struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	Description string `json:"description"`
}

// ErrorInfo contains detailed error information
type ErrorInfo struct {
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ToolUsageInfo contains aggregated tool usage statistics
type ToolUsageInfo struct {
	Name          string `json:"name" console:"header:Tool"`
	CallCount     int    `json:"call_count" console:"header:Calls"`
	MaxInputSize  int    `json:"max_input_size,omitempty" console:"header:Max Input,format:number,omitempty"`
	MaxOutputSize int    `json:"max_output_size,omitempty" console:"header:Max Output,format:number,omitempty"`
	MaxDuration   string `json:"max_duration,omitempty" console:"header:Max Duration,omitempty"`
}

// OverviewDisplay is a display-optimized version of OverviewData for console rendering
type OverviewDisplay struct {
	RunID    int64  `console:"header:Run ID"`
	Workflow string `console:"header:Workflow"`
	Status   string `console:"header:Status"`
	Duration string `console:"header:Duration,omitempty"`
	Event    string `console:"header:Event"`
	Branch   string `console:"header:Branch"`
	URL      string `console:"header:URL"`
}

// buildAuditData creates structured audit data from workflow run information
func buildAuditData(processedRun ProcessedRun, metrics LogMetrics) AuditData {
	run := processedRun.Run
	auditReportLog.Printf("Building audit data for run ID %d", run.DatabaseID)

	// Build overview
	overview := OverviewData{
		RunID:        run.DatabaseID,
		WorkflowName: run.WorkflowName,
		Status:       run.Status,
		Conclusion:   run.Conclusion,
		CreatedAt:    run.CreatedAt,
		StartedAt:    run.StartedAt,
		UpdatedAt:    run.UpdatedAt,
		Event:        run.Event,
		Branch:       run.HeadBranch,
		URL:          run.URL,
	}
	if run.Duration > 0 {
		overview.Duration = timeutil.FormatDuration(run.Duration)
	}

	// Build metrics
	metricsData := MetricsData{
		TokenUsage:    run.TokenUsage,
		EstimatedCost: run.EstimatedCost,
		Turns:         run.Turns,
		ErrorCount:    run.ErrorCount,
		WarningCount:  run.WarningCount,
	}

	// Build job data
	var jobs []JobData
	for _, jobDetail := range processedRun.JobDetails {
		job := JobData{
			Name:       jobDetail.Name,
			Status:     jobDetail.Status,
			Conclusion: jobDetail.Conclusion,
		}
		if jobDetail.Duration > 0 {
			job.Duration = timeutil.FormatDuration(jobDetail.Duration)
		}
		jobs = append(jobs, job)
	}

	// Build downloaded files list
	downloadedFiles := extractDownloadedFiles(run.LogsPath)

	// Build errors and warnings lists
	var errors []ErrorInfo
	var warnings []ErrorInfo
	for _, logErr := range metrics.Errors {
		errInfo := ErrorInfo{
			File:    logErr.File,
			Line:    logErr.Line,
			Type:    logErr.Type,
			Message: logErr.Message,
		}
		if strings.ToLower(logErr.Type) == "error" {
			errors = append(errors, errInfo)
		} else {
			warnings = append(warnings, errInfo)
		}
	}

	// Build tool usage
	var toolUsage []ToolUsageInfo
	toolStats := make(map[string]*ToolUsageInfo)
	for _, toolCall := range metrics.ToolCalls {
		displayKey := workflow.PrettifyToolName(toolCall.Name)
		if existing, exists := toolStats[displayKey]; exists {
			existing.CallCount += toolCall.CallCount
			if toolCall.MaxInputSize > existing.MaxInputSize {
				existing.MaxInputSize = toolCall.MaxInputSize
			}
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
			info := &ToolUsageInfo{
				Name:          displayKey,
				CallCount:     toolCall.CallCount,
				MaxInputSize:  toolCall.MaxInputSize,
				MaxOutputSize: toolCall.MaxOutputSize,
			}
			if toolCall.MaxDuration > 0 {
				info.MaxDuration = timeutil.FormatDuration(toolCall.MaxDuration)
			}
			toolStats[displayKey] = info
		}
	}
	for _, info := range toolStats {
		toolUsage = append(toolUsage, *info)
	}

	// Generate key findings
	findings := generateFindings(processedRun, metricsData, errors, warnings)

	// Generate recommendations
	recommendations := generateRecommendations(processedRun, metricsData, findings)

	// Generate failure analysis if workflow failed
	var failureAnalysis *FailureAnalysis
	if run.Conclusion == "failure" || run.Conclusion == "timed_out" || run.Conclusion == "cancelled" {
		failureAnalysis = generateFailureAnalysis(processedRun, errors)
	}

	// Generate performance metrics
	performanceMetrics := generatePerformanceMetrics(processedRun, metricsData, toolUsage)

	if auditReportLog.Enabled() {
		auditReportLog.Printf("Built audit data: %d jobs, %d errors, %d warnings, %d tool types, %d findings, %d recommendations",
			len(jobs), len(errors), len(warnings), len(toolUsage), len(findings), len(recommendations))
	}

	return AuditData{
		Overview:                overview,
		Metrics:                 metricsData,
		KeyFindings:             findings,
		Recommendations:         recommendations,
		FailureAnalysis:         failureAnalysis,
		PerformanceMetrics:      performanceMetrics,
		Jobs:                    jobs,
		DownloadedFiles:         downloadedFiles,
		MissingTools:            processedRun.MissingTools,
		Noops:                   processedRun.Noops,
		MCPFailures:             processedRun.MCPFailures,
		FirewallAnalysis:        processedRun.FirewallAnalysis,
		RedactedDomainsAnalysis: processedRun.RedactedDomainsAnalysis,
		Errors:                  errors,
		Warnings:                warnings,
		ToolUsage:               toolUsage,
	}
}

// extractDownloadedFiles scans the logs directory and returns file information
func extractDownloadedFiles(logsPath string) []FileInfo {
	auditReportLog.Printf("Extracting downloaded files from: %s", logsPath)
	var files []FileInfo

	entries, err := os.ReadDir(logsPath)
	if err != nil {
		auditReportLog.Printf("Failed to read logs directory: %v", err)
		return files
	}

	// Get current working directory to calculate relative paths
	cwd, err := os.Getwd()
	if err != nil {
		auditReportLog.Printf("Failed to get current directory: %v", err)
		cwd = ""
	}

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		fullPath := filepath.Join(logsPath, name)

		// Calculate relative path from workspace root (current working directory)
		relativePath := fullPath
		if cwd != "" {
			if relPath, err := filepath.Rel(cwd, fullPath); err == nil {
				relativePath = relPath
			}
		}

		fileInfo := FileInfo{
			Path:        relativePath,
			Description: describeFile(name),
		}

		if info, err := os.Stat(fullPath); err == nil {
			fileInfo.Size = info.Size()
		}

		files = append(files, fileInfo)
	}

	auditReportLog.Printf("Extracted %d files from logs directory", len(files))
	return files
}

// describeFile provides a short description for known artifact files
func describeFile(filename string) string {
	descriptions := map[string]string{
		"aw_info.json":      "Engine configuration and workflow metadata",
		"safe_output.jsonl": "Safe outputs from workflow execution",
		"agent_output.json": "Validated safe outputs",
		"aw.patch":          "Git patch of changes made during execution",
		"agent-stdio.log":   "Agent standard output/error logs",
		"log.md":            "Human-readable agent session summary",
		"firewall.md":       "Firewall log analysis report",
		"run_summary.json":  "Cached summary of workflow run analysis",
		"prompt.txt":        "Input prompt for AI agent",
	}

	if desc, ok := descriptions[filename]; ok {
		return desc
	}

	// Handle directories
	if strings.HasSuffix(filename, "/") {
		return "Directory"
	}

	// Common directory names
	if filename == "agent_output" || filename == "firewall-logs" || filename == "squid-logs" {
		return "Directory containing log files"
	}
	if filename == "aw-prompts" {
		return "Directory containing AI prompts"
	}

	// Handle file patterns by extension
	if strings.HasSuffix(filename, ".log") {
		return "Log file"
	}
	if strings.HasSuffix(filename, ".md") {
		return "Markdown documentation"
	}
	if strings.HasSuffix(filename, ".json") {
		return "JSON data file"
	}
	if strings.HasSuffix(filename, ".jsonl") {
		return "JSON Lines data file"
	}
	if strings.HasSuffix(filename, ".patch") {
		return "Git patch file"
	}
	if strings.HasSuffix(filename, ".txt") {
		return "Text file"
	}

	return ""
}

// parseDurationString parses a duration string back to time.Duration (best effort)
func parseDurationString(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}

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
	fmt.Println(console.FormatInfoMessage("# Workflow Run Audit Report"))
	fmt.Println()

	// Overview Section - use new rendering system
	fmt.Println(console.FormatInfoMessage("## Overview"))
	fmt.Println()
	renderOverview(data.Overview)

	// Key Findings Section - NEW
	if len(data.KeyFindings) > 0 {
		fmt.Println(console.FormatInfoMessage("## Key Findings"))
		fmt.Println()
		renderKeyFindings(data.KeyFindings)
	}

	// Recommendations Section - NEW
	if len(data.Recommendations) > 0 {
		fmt.Println(console.FormatInfoMessage("## Recommendations"))
		fmt.Println()
		renderRecommendations(data.Recommendations)
	}

	// Failure Analysis Section - NEW
	if data.FailureAnalysis != nil {
		fmt.Println(console.FormatInfoMessage("## Failure Analysis"))
		fmt.Println()
		renderFailureAnalysis(data.FailureAnalysis)
	}

	// Performance Metrics Section - NEW
	if data.PerformanceMetrics != nil {
		fmt.Println(console.FormatInfoMessage("## Performance Metrics"))
		fmt.Println()
		renderPerformanceMetrics(data.PerformanceMetrics)
	}

	// Metrics Section - use new rendering system
	fmt.Println(console.FormatInfoMessage("## Metrics"))
	fmt.Println()
	renderMetrics(data.Metrics)

	// Jobs Section - use new table rendering
	if len(data.Jobs) > 0 {
		fmt.Println(console.FormatInfoMessage("## Jobs"))
		fmt.Println()
		renderJobsTable(data.Jobs)
	}

	// Downloaded Files Section
	if len(data.DownloadedFiles) > 0 {
		fmt.Println(console.FormatInfoMessage("## Downloaded Files"))
		fmt.Println()
		for _, file := range data.DownloadedFiles {
			formattedSize := console.FormatFileSize(file.Size)
			fmt.Printf("  • %s (%s)", file.Path, formattedSize)
			if file.Description != "" {
				fmt.Printf(" - %s", file.Description)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Missing Tools Section
	if len(data.MissingTools) > 0 {
		fmt.Println(console.FormatInfoMessage("## Missing Tools"))
		fmt.Println()
		for _, tool := range data.MissingTools {
			fmt.Printf("  • %s\n", tool.Tool)
			fmt.Printf("    Reason: %s\n", tool.Reason)
			if tool.Alternatives != "" {
				fmt.Printf("    Alternatives: %s\n", tool.Alternatives)
			}
		}
		fmt.Println()
	}

	// MCP Failures Section
	if len(data.MCPFailures) > 0 {
		fmt.Println(console.FormatInfoMessage("## MCP Server Failures"))
		fmt.Println()
		for _, failure := range data.MCPFailures {
			fmt.Printf("  • %s: %s\n", failure.ServerName, failure.Status)
		}
		fmt.Println()
	}

	// Firewall Analysis Section
	if data.FirewallAnalysis != nil && data.FirewallAnalysis.TotalRequests > 0 {
		fmt.Println(console.FormatInfoMessage("## Firewall Analysis"))
		fmt.Println()
		renderFirewallAnalysis(data.FirewallAnalysis)
	}

	// Redacted Domains Section
	if data.RedactedDomainsAnalysis != nil && data.RedactedDomainsAnalysis.TotalDomains > 0 {
		fmt.Println(console.FormatInfoMessage("## 🔒 Redacted URL Domains"))
		fmt.Println()
		renderRedactedDomainsAnalysis(data.RedactedDomainsAnalysis)
	}

	// Tool Usage Section - use new table rendering
	if len(data.ToolUsage) > 0 {
		fmt.Println(console.FormatInfoMessage("## Tool Usage"))
		fmt.Println()
		renderToolUsageTable(data.ToolUsage)
	}

	// Errors and Warnings Section
	if len(data.Errors) > 0 || len(data.Warnings) > 0 {
		fmt.Println(console.FormatInfoMessage("## Errors and Warnings"))
		fmt.Println()

		if len(data.Errors) > 0 {
			fmt.Println(console.FormatErrorMessage(fmt.Sprintf("  Errors (%d):", len(data.Errors))))
			for _, err := range data.Errors {
				if err.File != "" && err.Line > 0 {
					fmt.Printf("    %s:%d: %s\n", filepath.Base(err.File), err.Line, err.Message)
				} else {
					fmt.Printf("    %s\n", err.Message)
				}
			}
			fmt.Println()
		}

		if len(data.Warnings) > 0 {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("  Warnings (%d):", len(data.Warnings))))
			for _, warn := range data.Warnings {
				if warn.File != "" && warn.Line > 0 {
					fmt.Printf("    %s:%d: %s\n", filepath.Base(warn.File), warn.Line, warn.Message)
				} else {
					fmt.Printf("    %s\n", warn.Message)
				}
			}
			fmt.Println()
		}
	}

	// Location
	fmt.Println(console.FormatInfoMessage("## Logs Location"))
	fmt.Println()
	absPath, _ := filepath.Abs(logsPath)
	fmt.Printf("  %s\n", absPath)
	fmt.Println()
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
	}

	fmt.Print(console.RenderStruct(display))
}

// renderMetrics renders the metrics section using the new rendering system
func renderMetrics(metrics MetricsData) {
	fmt.Print(console.RenderStruct(metrics))
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
			truncateString(job.Name, 40),
			job.Status,
			conclusion,
			duration,
		}
		config.Rows = append(config.Rows, row)
	}

	fmt.Print(console.RenderTable(config))
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
			truncateString(tool.Name, 40),
			fmt.Sprintf("%d", tool.CallCount),
			inputStr,
			outputStr,
			durationStr,
		}
		config.Rows = append(config.Rows, row)
	}

	fmt.Print(console.RenderTable(config))
}

// renderFirewallAnalysis renders firewall analysis with summary and domain breakdown
func renderFirewallAnalysis(analysis *FirewallAnalysis) {
	// Summary statistics
	fmt.Printf("  Total Requests : %d\n", analysis.TotalRequests)
	fmt.Printf("  Allowed        : %d\n", analysis.AllowedRequests)
	fmt.Printf("  Denied         : %d\n", analysis.DeniedRequests)
	fmt.Println()

	// Allowed domains
	if len(analysis.AllowedDomains) > 0 {
		fmt.Println("  Allowed Domains:")
		for _, domain := range analysis.AllowedDomains {
			if stats, ok := analysis.RequestsByDomain[domain]; ok {
				fmt.Printf("    ✓ %s (%d requests)\n", domain, stats.Allowed)
			}
		}
		fmt.Println()
	}

	// Denied domains
	if len(analysis.DeniedDomains) > 0 {
		fmt.Println("  Denied Domains:")
		for _, domain := range analysis.DeniedDomains {
			if stats, ok := analysis.RequestsByDomain[domain]; ok {
				fmt.Printf("    ✗ %s (%d requests)\n", domain, stats.Denied)
			}
		}
		fmt.Println()
	}
}

// renderRedactedDomainsAnalysis renders redacted domains analysis
func renderRedactedDomainsAnalysis(analysis *RedactedDomainsAnalysis) {
	// Summary statistics
	fmt.Printf("  Total Domains Redacted: %d\n", analysis.TotalDomains)
	fmt.Println()

	// List domains
	if len(analysis.Domains) > 0 {
		fmt.Println("  Redacted Domains:")
		for _, domain := range analysis.Domains {
			fmt.Printf("    🔒 %s\n", domain)
		}
		fmt.Println()
	}
}

// truncateString truncates a string to maxLen, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// generateFindings analyzes the workflow run and generates key findings
func generateFindings(processedRun ProcessedRun, metrics MetricsData, errors []ErrorInfo, warnings []ErrorInfo) []Finding {
	auditReportLog.Printf("Generating findings: errors=%d, warnings=%d, conclusion=%s", len(errors), len(warnings), processedRun.Run.Conclusion)
	var findings []Finding
	run := processedRun.Run

	// Failure findings
	if run.Conclusion == "failure" {
		findings = append(findings, Finding{
			Category:    "error",
			Severity:    "critical",
			Title:       "Workflow Failed",
			Description: fmt.Sprintf("Workflow '%s' failed with %d error(s)", run.WorkflowName, metrics.ErrorCount),
			Impact:      "Workflow did not complete successfully and may need intervention",
		})
	}

	if run.Conclusion == "timed_out" {
		findings = append(findings, Finding{
			Category:    "performance",
			Severity:    "high",
			Title:       "Workflow Timeout",
			Description: "Workflow exceeded time limit and was terminated",
			Impact:      "Tasks may be incomplete, consider optimizing workflow or increasing timeout",
		})
	}

	// Cost findings
	if metrics.EstimatedCost > 1.0 {
		findings = append(findings, Finding{
			Category:    "cost",
			Severity:    "high",
			Title:       "High Cost Detected",
			Description: fmt.Sprintf("Estimated cost of $%.2f exceeds typical threshold", metrics.EstimatedCost),
			Impact:      "Review token usage and consider optimization opportunities",
		})
	} else if metrics.EstimatedCost > 0.5 {
		findings = append(findings, Finding{
			Category:    "cost",
			Severity:    "medium",
			Title:       "Moderate Cost",
			Description: fmt.Sprintf("Estimated cost of $%.2f is moderate", metrics.EstimatedCost),
			Impact:      "Monitor costs if this workflow runs frequently",
		})
	}

	// Token usage findings
	if metrics.TokenUsage > 50000 {
		findings = append(findings, Finding{
			Category:    "performance",
			Severity:    "medium",
			Title:       "High Token Usage",
			Description: fmt.Sprintf("Used %s tokens", console.FormatNumber(metrics.TokenUsage)),
			Impact:      "High token usage may indicate verbose outputs or inefficient prompts",
		})
	}

	// Turn count findings
	if metrics.Turns > 10 {
		findings = append(findings, Finding{
			Category:    "performance",
			Severity:    "medium",
			Title:       "Many Iterations",
			Description: fmt.Sprintf("Workflow took %d turns to complete", metrics.Turns),
			Impact:      "Many turns may indicate task complexity or unclear instructions",
		})
	}

	// Error findings
	if len(errors) > 5 {
		findings = append(findings, Finding{
			Category:    "error",
			Severity:    "high",
			Title:       "Multiple Errors",
			Description: fmt.Sprintf("Encountered %d errors during execution", len(errors)),
			Impact:      "Multiple errors may indicate systemic issues requiring attention",
		})
	}

	// MCP failure findings
	if len(processedRun.MCPFailures) > 0 {
		serverNames := make([]string, len(processedRun.MCPFailures))
		for i, failure := range processedRun.MCPFailures {
			serverNames[i] = failure.ServerName
		}
		findings = append(findings, Finding{
			Category:    "tooling",
			Severity:    "high",
			Title:       "MCP Server Failures",
			Description: fmt.Sprintf("Failed MCP servers: %s", strings.Join(serverNames, ", ")),
			Impact:      "Missing tools may limit workflow capabilities",
		})
	}

	// Missing tool findings
	if len(processedRun.MissingTools) > 0 {
		toolNames := make([]string, 0, min(3, len(processedRun.MissingTools)))
		for i := 0; i < len(processedRun.MissingTools) && i < 3; i++ {
			toolNames = append(toolNames, processedRun.MissingTools[i].Tool)
		}
		desc := fmt.Sprintf("Missing tools: %s", strings.Join(toolNames, ", "))
		if len(processedRun.MissingTools) > 3 {
			desc += fmt.Sprintf(" (and %d more)", len(processedRun.MissingTools)-3)
		}
		findings = append(findings, Finding{
			Category:    "tooling",
			Severity:    "medium",
			Title:       "Tools Not Available",
			Description: desc,
			Impact:      "Agent requested tools that were not configured or available",
		})
	}

	// Firewall findings
	if processedRun.FirewallAnalysis != nil && processedRun.FirewallAnalysis.DeniedRequests > 0 {
		findings = append(findings, Finding{
			Category:    "network",
			Severity:    "medium",
			Title:       "Blocked Network Requests",
			Description: fmt.Sprintf("%d network requests were blocked by firewall", processedRun.FirewallAnalysis.DeniedRequests),
			Impact:      "Blocked requests may indicate missing network permissions or unexpected behavior",
		})
	}

	// Success findings
	if run.Conclusion == "success" && len(errors) == 0 {
		findings = append(findings, Finding{
			Category:    "success",
			Severity:    "info",
			Title:       "Workflow Completed Successfully",
			Description: fmt.Sprintf("Completed in %d turns with no errors", metrics.Turns),
			Impact:      "No action needed",
		})
	}

	return findings
}

// generateRecommendations creates actionable recommendations based on findings
func generateRecommendations(processedRun ProcessedRun, metrics MetricsData, findings []Finding) []Recommendation {
	auditReportLog.Printf("Generating recommendations: findings_count=%d, workflow_conclusion=%s", len(findings), processedRun.Run.Conclusion)
	var recommendations []Recommendation
	run := processedRun.Run

	// Check for high-severity findings
	hasCriticalFindings := false
	hasHighCostFindings := false
	hasManyTurns := false
	for _, finding := range findings {
		if finding.Severity == "critical" {
			hasCriticalFindings = true
		}
		if finding.Category == "cost" && (finding.Severity == "high" || finding.Severity == "medium") {
			hasHighCostFindings = true
		}
		if finding.Category == "performance" && strings.Contains(finding.Title, "Iterations") {
			hasManyTurns = true
		}
	}

	// Recommendations for failures
	if run.Conclusion == "failure" || hasCriticalFindings {
		recommendations = append(recommendations, Recommendation{
			Priority: "high",
			Action:   "Review error logs to identify root cause of failure",
			Reason:   "Understanding failure causes helps prevent recurrence",
			Example:  "Check the Errors section below for specific error messages and file locations",
		})
	}

	// Recommendations for cost optimization
	if hasHighCostFindings {
		recommendations = append(recommendations, Recommendation{
			Priority: "medium",
			Action:   "Optimize prompt size and reduce verbose outputs",
			Reason:   "High token usage increases costs and may slow execution",
			Example:  "Use concise prompts, limit output verbosity, and consider caching repeated data",
		})
	}

	// Recommendations for many turns
	if hasManyTurns {
		recommendations = append(recommendations, Recommendation{
			Priority: "medium",
			Action:   "Clarify workflow instructions or break into smaller tasks",
			Reason:   "Many iterations may indicate unclear objectives or overly complex tasks",
			Example:  "Split complex workflows into discrete steps with clear success criteria",
		})
	}

	// Recommendations for missing tools
	if len(processedRun.MissingTools) > 0 {
		recommendations = append(recommendations, Recommendation{
			Priority: "medium",
			Action:   "Add missing tools to workflow configuration",
			Reason:   "Missing tools limit agent capabilities and may cause failures",
			Example:  fmt.Sprintf("Add tools configuration for: %s", processedRun.MissingTools[0].Tool),
		})
	}

	// Recommendations for MCP failures
	if len(processedRun.MCPFailures) > 0 {
		recommendations = append(recommendations, Recommendation{
			Priority: "high",
			Action:   "Fix MCP server configuration or dependencies",
			Reason:   "MCP server failures prevent agent from accessing required tools",
			Example:  "Check server logs and verify MCP server is properly configured and accessible",
		})
	}

	// Recommendations for firewall blocks
	if processedRun.FirewallAnalysis != nil && processedRun.FirewallAnalysis.DeniedRequests > 10 {
		recommendations = append(recommendations, Recommendation{
			Priority: "medium",
			Action:   "Review network access configuration",
			Reason:   "Many blocked requests suggest missing network permissions",
			Example:  "Add allowed domains to network configuration or review firewall rules",
		})
	}

	// General best practices
	if len(recommendations) == 0 && run.Conclusion == "success" {
		recommendations = append(recommendations, Recommendation{
			Priority: "low",
			Action:   "Monitor workflow performance over time",
			Reason:   "Tracking metrics helps identify trends and optimization opportunities",
			Example:  "Run 'gh aw logs' periodically to review cost and performance trends",
		})
	}

	return recommendations
}

// generateFailureAnalysis creates structured analysis for failed workflows
func generateFailureAnalysis(processedRun ProcessedRun, errors []ErrorInfo) *FailureAnalysis {
	run := processedRun.Run
	auditReportLog.Printf("Generating failure analysis: conclusion=%s, error_count=%d", run.Conclusion, len(errors))

	// Determine primary failure reason
	primaryFailure := run.Conclusion
	if primaryFailure == "" {
		primaryFailure = "unknown"
	}

	// Collect failed job names
	var failedJobs []string
	for _, job := range processedRun.JobDetails {
		if job.Conclusion == "failure" || job.Conclusion == "timed_out" || job.Conclusion == "cancelled" {
			failedJobs = append(failedJobs, job.Name)
		}
	}

	// Generate error summary
	errorSummary := "No specific errors identified"
	if len(errors) > 0 {
		if len(errors) == 1 {
			errorSummary = errors[0].Message
		} else {
			errorSummary = fmt.Sprintf("%d errors: %s (and %d more)", len(errors), errors[0].Message, len(errors)-1)
		}
	}

	// Attempt to identify root cause
	rootCause := ""
	if len(processedRun.MCPFailures) > 0 {
		rootCause = fmt.Sprintf("MCP server failure: %s", processedRun.MCPFailures[0].ServerName)
	} else if len(errors) > 0 {
		// Look for common error patterns
		firstError := errors[0].Message
		if strings.Contains(strings.ToLower(firstError), "timeout") {
			rootCause = "Operation timeout"
		} else if strings.Contains(strings.ToLower(firstError), "permission") {
			rootCause = "Permission denied"
		} else if strings.Contains(strings.ToLower(firstError), "not found") {
			rootCause = "Resource not found"
		} else if strings.Contains(strings.ToLower(firstError), "authentication") {
			rootCause = "Authentication failure"
		}
	}

	return &FailureAnalysis{
		PrimaryFailure: primaryFailure,
		FailedJobs:     failedJobs,
		ErrorSummary:   errorSummary,
		RootCause:      rootCause,
	}
}

// generatePerformanceMetrics calculates aggregated performance statistics
func generatePerformanceMetrics(processedRun ProcessedRun, metrics MetricsData, toolUsage []ToolUsageInfo) *PerformanceMetrics {
	run := processedRun.Run
	auditReportLog.Printf("Generating performance metrics: token_usage=%d, tool_count=%d, duration=%v", metrics.TokenUsage, len(toolUsage), run.Duration)
	pm := &PerformanceMetrics{}

	// Calculate tokens per minute
	if run.Duration > 0 && metrics.TokenUsage > 0 {
		minutes := run.Duration.Minutes()
		if minutes > 0 {
			pm.TokensPerMinute = float64(metrics.TokenUsage) / minutes
		}
	}

	// Determine cost efficiency
	if metrics.EstimatedCost > 0 && run.Duration > 0 {
		costPerMinute := metrics.EstimatedCost / run.Duration.Minutes()
		if costPerMinute < 0.01 {
			pm.CostEfficiency = "excellent"
		} else if costPerMinute < 0.05 {
			pm.CostEfficiency = "good"
		} else if costPerMinute < 0.10 {
			pm.CostEfficiency = "moderate"
		} else {
			pm.CostEfficiency = "poor"
		}
	}

	// Find most used tool
	if len(toolUsage) > 0 {
		mostUsed := toolUsage[0]
		for i := 1; i < len(toolUsage); i++ {
			if toolUsage[i].CallCount > mostUsed.CallCount {
				mostUsed = toolUsage[i]
			}
		}
		pm.MostUsedTool = fmt.Sprintf("%s (%d calls)", mostUsed.Name, mostUsed.CallCount)
	}

	// Calculate average tool duration
	if len(toolUsage) > 0 {
		totalDuration := time.Duration(0)
		count := 0
		for _, tool := range toolUsage {
			if tool.MaxDuration != "" {
				// Try to parse duration string using time.ParseDuration
				if d, err := time.ParseDuration(tool.MaxDuration); err == nil {
					totalDuration += d
					count++
				}
			}
		}
		if count > 0 {
			avgDuration := totalDuration / time.Duration(count)
			pm.AvgToolDuration = timeutil.FormatDuration(avgDuration)
		}
	}

	// Network request count from firewall
	if processedRun.FirewallAnalysis != nil {
		pm.NetworkRequests = processedRun.FirewallAnalysis.TotalRequests
	}

	return pm
}

// renderKeyFindings renders key findings with colored severity indicators
func renderKeyFindings(findings []Finding) {
	// Group findings by severity for better presentation
	critical := []Finding{}
	high := []Finding{}
	medium := []Finding{}
	low := []Finding{}
	info := []Finding{}

	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			critical = append(critical, finding)
		case "high":
			high = append(high, finding)
		case "medium":
			medium = append(medium, finding)
		case "low":
			low = append(low, finding)
		default:
			info = append(info, finding)
		}
	}

	// Render critical findings first
	for _, finding := range critical {
		fmt.Printf("  🔴 %s [%s]\n", console.FormatErrorMessage(finding.Title), finding.Category)
		fmt.Printf("     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Printf("     Impact: %s\n", finding.Impact)
		}
		fmt.Println()
	}

	// Then high severity
	for _, finding := range high {
		fmt.Printf("  🟠 %s [%s]\n", console.FormatWarningMessage(finding.Title), finding.Category)
		fmt.Printf("     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Printf("     Impact: %s\n", finding.Impact)
		}
		fmt.Println()
	}

	// Medium severity
	for _, finding := range medium {
		fmt.Printf("  🟡 %s [%s]\n", finding.Title, finding.Category)
		fmt.Printf("     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Printf("     Impact: %s\n", finding.Impact)
		}
		fmt.Println()
	}

	// Low severity
	for _, finding := range low {
		fmt.Printf("  ℹ️  %s [%s]\n", finding.Title, finding.Category)
		fmt.Printf("     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Printf("     Impact: %s\n", finding.Impact)
		}
		fmt.Println()
	}

	// Info findings
	for _, finding := range info {
		fmt.Printf("  ✅ %s [%s]\n", console.FormatSuccessMessage(finding.Title), finding.Category)
		fmt.Printf("     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Printf("     Impact: %s\n", finding.Impact)
		}
		fmt.Println()
	}
}

// renderRecommendations renders actionable recommendations
func renderRecommendations(recommendations []Recommendation) {
	// Group by priority
	high := []Recommendation{}
	medium := []Recommendation{}
	low := []Recommendation{}

	for _, rec := range recommendations {
		switch rec.Priority {
		case "high":
			high = append(high, rec)
		case "medium":
			medium = append(medium, rec)
		default:
			low = append(low, rec)
		}
	}

	// Render high priority first
	for i, rec := range high {
		fmt.Printf("  %d. [HIGH] %s\n", i+1, console.FormatWarningMessage(rec.Action))
		fmt.Printf("     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Printf("     Example: %s\n", rec.Example)
		}
		fmt.Println()
	}

	// Medium priority
	startIdx := len(high) + 1
	for i, rec := range medium {
		fmt.Printf("  %d. [MEDIUM] %s\n", startIdx+i, rec.Action)
		fmt.Printf("     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Printf("     Example: %s\n", rec.Example)
		}
		fmt.Println()
	}

	// Low priority
	startIdx += len(medium)
	for i, rec := range low {
		fmt.Printf("  %d. [LOW] %s\n", startIdx+i, rec.Action)
		fmt.Printf("     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Printf("     Example: %s\n", rec.Example)
		}
		fmt.Println()
	}
}

// renderFailureAnalysis renders failure analysis information
func renderFailureAnalysis(analysis *FailureAnalysis) {
	fmt.Printf("  Primary Failure: %s\n", console.FormatErrorMessage(analysis.PrimaryFailure))
	fmt.Println()

	if len(analysis.FailedJobs) > 0 {
		fmt.Printf("  Failed Jobs:\n")
		for _, job := range analysis.FailedJobs {
			fmt.Printf("    • %s\n", job)
		}
		fmt.Println()
	}

	fmt.Printf("  Error Summary: %s\n", analysis.ErrorSummary)
	fmt.Println()

	if analysis.RootCause != "" {
		fmt.Printf("  Identified Root Cause: %s\n", console.FormatWarningMessage(analysis.RootCause))
		fmt.Println()
	}
}

// renderPerformanceMetrics renders performance metrics
func renderPerformanceMetrics(metrics *PerformanceMetrics) {
	if metrics.TokensPerMinute > 0 {
		fmt.Printf("  Tokens per Minute: %.1f\n", metrics.TokensPerMinute)
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
		fmt.Printf("  Cost Efficiency: %s\n", efficiencyDisplay)
	}

	if metrics.AvgToolDuration != "" {
		fmt.Printf("  Average Tool Duration: %s\n", metrics.AvgToolDuration)
	}

	if metrics.MostUsedTool != "" {
		fmt.Printf("  Most Used Tool: %s\n", metrics.MostUsedTool)
	}

	if metrics.NetworkRequests > 0 {
		fmt.Printf("  Network Requests: %d\n", metrics.NetworkRequests)
	}

	fmt.Println()
}
