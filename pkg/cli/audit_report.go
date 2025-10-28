package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
)

// AuditData represents the complete structured audit data for a workflow run
type AuditData struct {
	Overview         OverviewData        `json:"overview"`
	Metrics          MetricsData         `json:"metrics"`
	Jobs             []JobData           `json:"jobs,omitempty"`
	DownloadedFiles  []FileInfo          `json:"downloaded_files"`
	MissingTools     []MissingToolReport `json:"missing_tools,omitempty"`
	MCPFailures      []MCPFailureReport  `json:"mcp_failures,omitempty"`
	FirewallAnalysis *FirewallAnalysis   `json:"firewall_analysis,omitempty"`
	Errors           []ErrorInfo         `json:"errors,omitempty"`
	Warnings         []ErrorInfo         `json:"warnings,omitempty"`
	ToolUsage        []ToolUsageInfo     `json:"tool_usage,omitempty"`
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
	Path          string `json:"path"`
	Size          int64  `json:"size"`
	SizeFormatted string `json:"size_formatted"`
	Description   string `json:"description"`
	IsDirectory   bool   `json:"is_directory"`
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
		overview.Duration = formatDuration(run.Duration)
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
			job.Duration = formatDuration(jobDetail.Duration)
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
				maxDur := formatDuration(toolCall.MaxDuration)
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
				info.MaxDuration = formatDuration(toolCall.MaxDuration)
			}
			toolStats[displayKey] = info
		}
	}
	for _, info := range toolStats {
		toolUsage = append(toolUsage, *info)
	}

	return AuditData{
		Overview:         overview,
		Metrics:          metricsData,
		Jobs:             jobs,
		DownloadedFiles:  downloadedFiles,
		MissingTools:     processedRun.MissingTools,
		MCPFailures:      processedRun.MCPFailures,
		FirewallAnalysis: processedRun.FirewallAnalysis,
		Errors:           errors,
		Warnings:         warnings,
		ToolUsage:        toolUsage,
	}
}

// extractDownloadedFiles scans the logs directory and returns file information
func extractDownloadedFiles(logsPath string) []FileInfo {
	var files []FileInfo

	entries, err := os.ReadDir(logsPath)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(logsPath, name)

		fileInfo := FileInfo{
			Path:        name,
			IsDirectory: entry.IsDir(),
			Description: describeFile(name),
		}

		if !entry.IsDir() {
			if info, err := os.Stat(fullPath); err == nil {
				fileInfo.Size = info.Size()
				fileInfo.SizeFormatted = pretty.FormatFileSize(info.Size())
			}
		} else {
			// For directories, sum the sizes of files inside
			totalSize := calculateDirectorySize(fullPath)
			fileInfo.Size = totalSize
			if totalSize > 0 {
				fileInfo.SizeFormatted = pretty.FormatFileSize(totalSize)
			}
		}

		files = append(files, fileInfo)
	}

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
	}

	if desc, ok := descriptions[filename]; ok {
		return desc
	}

	// Handle directories
	if strings.HasSuffix(filename, "/") || filename == "agent_output" {
		return "Directory containing agent output files"
	}

	// Generic log file
	if strings.HasSuffix(filename, ".log") {
		return "Log file"
	}

	return ""
}

// calculateDirectorySize recursively calculates the total size of files in a directory
func calculateDirectorySize(dirPath string) int64 {
	var totalSize int64

	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize
}

// parseDurationString parses a duration string back to time.Duration (best effort)
func parseDurationString(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}

// renderJSON outputs the audit data as JSON
func renderJSON(data AuditData) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// renderConsole outputs the audit data as formatted console tables
func renderConsole(data AuditData, logsPath string) {
	fmt.Println(console.FormatInfoMessage("# Workflow Run Audit Report"))
	fmt.Println()

	// Overview Section - use new rendering system
	fmt.Println(console.FormatInfoMessage("## Overview"))
	fmt.Println()
	renderOverview(data.Overview)

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
			if file.IsDirectory {
				fmt.Printf("  • %s/", file.Path)
				if file.SizeFormatted != "" {
					fmt.Printf(" (%s)", file.SizeFormatted)
				}
			} else {
				fmt.Printf("  • %s (%s)", file.Path, file.SizeFormatted)
			}
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
