package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/stringutil"
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
		Files:    overview.LogsPath,
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
			stringutil.Truncate(job.Name, 40),
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
			stringutil.Truncate(tool.Name, 40),
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
