// This file contains rendering and display functions for MCP gateway metrics.
// It follows the established pattern of separating render logic into dedicated
// *_render.go files (see audit_diff_render.go, audit_report_render.go).

package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/sliceutil"
)

// renderGatewayMetricsTable renders gateway metrics as a console table
func renderGatewayMetricsTable(metrics *GatewayMetrics, verbose bool) string {
	if metrics == nil || len(metrics.Servers) == 0 {
		return ""
	}

	var output strings.Builder

	output.WriteString("\n")
	output.WriteString(console.FormatInfoMessage("MCP Gateway Metrics"))
	output.WriteString("\n\n")

	// Summary statistics
	fmt.Fprintf(&output, "Total Requests: %d\n", metrics.TotalRequests)
	fmt.Fprintf(&output, "Total Tool Calls: %d\n", metrics.TotalToolCalls)
	fmt.Fprintf(&output, "Total Errors: %d\n", metrics.TotalErrors)
	if metrics.TotalFiltered > 0 {
		fmt.Fprintf(&output, "Total DIFC Filtered: %d\n", metrics.TotalFiltered)
	}
	if metrics.TotalGuardBlocked > 0 {
		fmt.Fprintf(&output, "Total Guard Policy Blocked: %d\n", metrics.TotalGuardBlocked)
	}
	fmt.Fprintf(&output, "Servers: %d\n", len(metrics.Servers))

	if !metrics.StartTime.IsZero() && !metrics.EndTime.IsZero() {
		duration := metrics.EndTime.Sub(metrics.StartTime)
		fmt.Fprintf(&output, "Time Range: %s\n", duration.Round(time.Second))
	}

	output.WriteString("\n")

	// Server metrics table
	if len(metrics.Servers) > 0 {
		// Sort servers by request count
		serverNames := getSortedServerNames(metrics)

		hasFiltered := metrics.TotalFiltered > 0
		hasGuardPolicy := metrics.TotalGuardBlocked > 0
		serverRows := make([][]string, 0, len(serverNames))
		for _, serverName := range serverNames {
			server := metrics.Servers[serverName]
			avgTime := 0.0
			if server.RequestCount > 0 {
				avgTime = server.TotalDuration / float64(server.RequestCount)
			}
			row := []string{
				serverName,
				strconv.Itoa(server.RequestCount),
				strconv.Itoa(server.ToolCallCount),
				fmt.Sprintf("%.0fms", avgTime),
				strconv.Itoa(server.ErrorCount),
			}
			if hasFiltered {
				row = append(row, strconv.Itoa(server.FilteredCount))
			}
			if hasGuardPolicy {
				row = append(row, strconv.Itoa(server.GuardPolicyBlocked))
			}
			serverRows = append(serverRows, row)
		}

		headers := []string{"Server", "Requests", "Tool Calls", "Avg Time", "Errors"}
		if hasFiltered {
			headers = append(headers, "Filtered")
		}
		if hasGuardPolicy {
			headers = append(headers, "Guard Blocked")
		}
		output.WriteString(console.RenderTable(console.TableConfig{
			Title:   "Server Usage",
			Headers: headers,
			Rows:    serverRows,
		}))
	}

	// DIFC filtered events table
	if len(metrics.FilteredEvents) > 0 {
		output.WriteString("\n")
		filteredRows := make([][]string, 0, len(metrics.FilteredEvents))
		for _, fe := range metrics.FilteredEvents {
			reason := fe.Reason
			if len(reason) > 80 {
				reason = reason[:77] + "..."
			}
			filteredRows = append(filteredRows, []string{
				fe.ServerID,
				fe.ToolName,
				fe.AuthorLogin,
				reason,
			})
		}
		output.WriteString(console.RenderTable(console.TableConfig{
			Title:   "DIFC Filtered Events",
			Headers: []string{"Server", "Tool", "User", "Reason"},
			Rows:    filteredRows,
		}))
	}

	// Guard policy events table
	if len(metrics.GuardPolicyEvents) > 0 {
		output.WriteString("\n")
		guardRows := make([][]string, 0, len(metrics.GuardPolicyEvents))
		for _, gpe := range metrics.GuardPolicyEvents {
			message := gpe.Message
			if len(message) > 60 {
				message = message[:57] + "..."
			}
			repo := gpe.Repository
			if repo == "" {
				repo = "-"
			}
			guardRows = append(guardRows, []string{
				gpe.ServerID,
				gpe.ToolName,
				gpe.Reason,
				message,
				repo,
			})
		}
		output.WriteString(console.RenderTable(console.TableConfig{
			Title:   "Guard Policy Blocked Events",
			Headers: []string{"Server", "Tool", "Reason", "Message", "Repository"},
			Rows:    guardRows,
		}))
	}

	// Tool metrics table (if verbose)
	if verbose {
		output.WriteString("\n")
		output.WriteString("Tool Usage Details:\n")

		for _, serverName := range getSortedServerNames(metrics) {
			server := metrics.Servers[serverName]
			if len(server.Tools) == 0 {
				continue
			}

			// Sort tools by call count
			toolNames := sliceutil.MapToSlice(server.Tools)
			sort.Slice(toolNames, func(i, j int) bool {
				return server.Tools[toolNames[i]].CallCount > server.Tools[toolNames[j]].CallCount
			})

			toolRows := make([][]string, 0, len(toolNames))
			for _, toolName := range toolNames {
				tool := server.Tools[toolName]
				toolRows = append(toolRows, []string{
					toolName,
					strconv.Itoa(tool.CallCount),
					fmt.Sprintf("%.0fms", tool.AvgDuration),
					fmt.Sprintf("%.0fms", tool.MaxDuration),
					strconv.Itoa(tool.ErrorCount),
				})
			}

			output.WriteString(console.RenderTable(console.TableConfig{
				Title:   serverName,
				Headers: []string{"Tool", "Calls", "Avg Time", "Max Time", "Errors"},
				Rows:    toolRows,
			}))
		}
	}

	return output.String()
}

// getSortedServerNames returns server names sorted by request count
func getSortedServerNames(metrics *GatewayMetrics) []string {
	names := sliceutil.MapToSlice(metrics.Servers)
	sort.Slice(names, func(i, j int) bool {
		return metrics.Servers[names[i]].RequestCount > metrics.Servers[names[j]].RequestCount
	})
	return names
}

// displayAggregatedGatewayMetrics aggregates and displays gateway metrics across all processed runs
func displayAggregatedGatewayMetrics(processedRuns []ProcessedRun, outputDir string, verbose bool) {
	// Aggregate gateway metrics from all runs
	aggregated := &GatewayMetrics{
		Servers: make(map[string]*GatewayServerMetrics),
	}

	runCount := 0
	for _, pr := range processedRuns {
		runDir := pr.Run.LogsPath
		if runDir == "" {
			continue
		}

		// Try to parse gateway.jsonl from this run
		runMetrics, err := parseGatewayLogs(runDir, false)
		if err != nil {
			// Skip runs without gateway.jsonl (this is normal for runs without MCP gateway)
			continue
		}

		runCount++

		// Merge metrics from this run into aggregated metrics
		aggregated.TotalRequests += runMetrics.TotalRequests
		aggregated.TotalToolCalls += runMetrics.TotalToolCalls
		aggregated.TotalErrors += runMetrics.TotalErrors
		aggregated.TotalFiltered += runMetrics.TotalFiltered
		aggregated.TotalGuardBlocked += runMetrics.TotalGuardBlocked
		aggregated.TotalDuration += runMetrics.TotalDuration
		aggregated.FilteredEvents = append(aggregated.FilteredEvents, runMetrics.FilteredEvents...)
		aggregated.GuardPolicyEvents = append(aggregated.GuardPolicyEvents, runMetrics.GuardPolicyEvents...)

		// Merge server metrics
		for serverName, serverMetrics := range runMetrics.Servers {
			aggServer := getOrCreateServer(aggregated, serverName)
			aggServer.RequestCount += serverMetrics.RequestCount
			aggServer.ToolCallCount += serverMetrics.ToolCallCount
			aggServer.TotalDuration += serverMetrics.TotalDuration
			aggServer.ErrorCount += serverMetrics.ErrorCount
			aggServer.FilteredCount += serverMetrics.FilteredCount
			aggServer.GuardPolicyBlocked += serverMetrics.GuardPolicyBlocked

			// Merge tool metrics
			for toolName, toolMetrics := range serverMetrics.Tools {
				aggTool := getOrCreateTool(aggServer, toolName)
				aggTool.CallCount += toolMetrics.CallCount
				aggTool.TotalDuration += toolMetrics.TotalDuration
				aggTool.ErrorCount += toolMetrics.ErrorCount
				aggTool.TotalInputSize += toolMetrics.TotalInputSize
				aggTool.TotalOutputSize += toolMetrics.TotalOutputSize

				// Update max/min durations
				if toolMetrics.MaxDuration > aggTool.MaxDuration {
					aggTool.MaxDuration = toolMetrics.MaxDuration
				}
				if aggTool.MinDuration == 0 || (toolMetrics.MinDuration > 0 && toolMetrics.MinDuration < aggTool.MinDuration) {
					aggTool.MinDuration = toolMetrics.MinDuration
				}
			}
		}

		// Update time range
		if aggregated.StartTime.IsZero() || (!runMetrics.StartTime.IsZero() && runMetrics.StartTime.Before(aggregated.StartTime)) {
			aggregated.StartTime = runMetrics.StartTime
		}
		if aggregated.EndTime.IsZero() || (!runMetrics.EndTime.IsZero() && runMetrics.EndTime.After(aggregated.EndTime)) {
			aggregated.EndTime = runMetrics.EndTime
		}
	}

	// Only display if we found gateway metrics
	if runCount == 0 || len(aggregated.Servers) == 0 {
		return
	}

	// Recalculate averages for aggregated data
	calculateGatewayAggregates(aggregated)

	// Display the aggregated metrics
	if metricsOutput := renderGatewayMetricsTable(aggregated, verbose); metricsOutput != "" {
		fmt.Fprint(os.Stderr, metricsOutput)
		if runCount > 1 {
			fmt.Fprintf(os.Stderr, "\n%s\n",
				console.FormatInfoMessage(fmt.Sprintf("Gateway metrics aggregated from %d runs", runCount)))
		}
	}
}
