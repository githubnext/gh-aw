package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/stringutil"
)

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
