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
	Overview        OverviewData        `json:"overview"`
	Metrics         MetricsData         `json:"metrics"`
	Jobs            []JobData           `json:"jobs,omitempty"`
	DownloadedFiles []FileInfo          `json:"downloaded_files"`
	MissingTools    []MissingToolReport `json:"missing_tools,omitempty"`
	MCPFailures     []MCPFailureReport  `json:"mcp_failures,omitempty"`
	Errors          []ErrorInfo         `json:"errors,omitempty"`
	Warnings        []ErrorInfo         `json:"warnings,omitempty"`
	ToolUsage       []ToolUsageInfo     `json:"tool_usage,omitempty"`
}

// OverviewData contains basic information about the workflow run
type OverviewData struct {
	RunID        int64     `json:"run_id"`
	WorkflowName string    `json:"workflow_name"`
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	Duration     string    `json:"duration,omitempty"`
	Event        string    `json:"event"`
	Branch       string    `json:"branch"`
	URL          string    `json:"url"`
}

// MetricsData contains execution metrics
type MetricsData struct {
	TokenUsage    int     `json:"token_usage,omitempty"`
	EstimatedCost float64 `json:"estimated_cost,omitempty"`
	Turns         int     `json:"turns,omitempty"`
	ErrorCount    int     `json:"error_count"`
	WarningCount  int     `json:"warning_count"`
}

// JobData contains information about individual jobs
type JobData struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion,omitempty"`
	Duration   string `json:"duration,omitempty"`
}

// FileInfo contains information about downloaded artifact files
type FileInfo struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	SizeFormatted string `json:"size_formatted"`
	Description string `json:"description"`
	IsDirectory bool   `json:"is_directory"`
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
	Name          string `json:"name"`
	CallCount     int    `json:"call_count"`
	MaxOutputSize int    `json:"max_output_size,omitempty"`
	MaxDuration   string `json:"max_duration,omitempty"`
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
		Overview:        overview,
		Metrics:         metricsData,
		DownloadedFiles: downloadedFiles,
		MissingTools:    processedRun.MissingTools,
		MCPFailures:     processedRun.MCPFailures,
		Errors:          errors,
		Warnings:        warnings,
		ToolUsage:       toolUsage,
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
		"aw_info.json":       "Engine configuration and workflow metadata",
		"safe_output.jsonl":  "Safe outputs from workflow execution",
		"agent_output.json":  "Validated safe outputs",
		"aw.patch":           "Git patch of changes made during execution",
		"agent-stdio.log":    "Agent standard output/error logs",
		"log.md":             "Human-readable agent session summary",
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

	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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

	// Overview Section
	fmt.Println(console.FormatInfoMessage("## Overview"))
	fmt.Println()
	fmt.Printf("  Run ID:        %d\n", data.Overview.RunID)
	fmt.Printf("  Workflow:      %s\n", data.Overview.WorkflowName)
	fmt.Printf("  Status:        %s", data.Overview.Status)
	if data.Overview.Conclusion != "" && data.Overview.Status == "completed" {
		fmt.Printf(" (%s)", data.Overview.Conclusion)
	}
	fmt.Println()
	if data.Overview.Duration != "" {
		fmt.Printf("  Duration:      %s\n", data.Overview.Duration)
	}
	fmt.Printf("  Event:         %s\n", data.Overview.Event)
	fmt.Printf("  Branch:        %s\n", data.Overview.Branch)
	fmt.Printf("  URL:           %s\n", data.Overview.URL)
	fmt.Println()

	// Metrics Section
	fmt.Println(console.FormatInfoMessage("## Metrics"))
	fmt.Println()
	if data.Metrics.TokenUsage > 0 {
		fmt.Printf("  Token Usage:      %s\n", formatNumber(data.Metrics.TokenUsage))
	}
	if data.Metrics.EstimatedCost > 0 {
		fmt.Printf("  Estimated Cost:   $%.3f\n", data.Metrics.EstimatedCost)
	}
	if data.Metrics.Turns > 0 {
		fmt.Printf("  Turns:            %d\n", data.Metrics.Turns)
	}
	fmt.Printf("  Errors:           %d\n", data.Metrics.ErrorCount)
	fmt.Printf("  Warnings:         %d\n", data.Metrics.WarningCount)
	fmt.Println()

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

	// Tool Usage Section
	if len(data.ToolUsage) > 0 {
		fmt.Println(console.FormatInfoMessage("## Tool Usage"))
		fmt.Println()
		fmt.Printf("  %-40s %10s %15s %15s\n", "Tool", "Calls", "Max Output", "Max Duration")
		fmt.Printf("  %s\n", strings.Repeat("-", 82))
		for _, tool := range data.ToolUsage {
			outputStr := "N/A"
			if tool.MaxOutputSize > 0 {
				outputStr = formatNumber(tool.MaxOutputSize)
			}
			durationStr := "N/A"
			if tool.MaxDuration != "" {
				durationStr = tool.MaxDuration
			}
			fmt.Printf("  %-40s %10d %15s %15s\n",
				truncateString(tool.Name, 40), tool.CallCount, outputStr, durationStr)
		}
		fmt.Println()
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
