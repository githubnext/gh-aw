// This file provides command-line interface functionality for gh-aw.
// This file (logs_report_mcp.go) contains the MCP tool usage summary builder
// for the logs report, aggregating gateway metrics across multiple workflow runs.

package cli

import (
	"cmp"
	"slices"
	"time"

	"github.com/github/gh-aw/pkg/timeutil"
)

// buildMCPToolUsageSummary aggregates MCP tool usage data across all runs
func buildMCPToolUsageSummary(processedRuns []ProcessedRun) *MCPToolUsageSummary {
	reportLog.Printf("Building MCP tool usage summary from %d processed runs", len(processedRuns))

	// Maps for aggregating data
	toolSummaryMap := make(map[string]*MCPToolSummary) // Key: serverName:toolName
	serverStatsMap := make(map[string]*MCPServerStats) // Key: serverName
	var allToolCalls []MCPToolCall
	var allFilteredEvents []DifcFilteredEvent

	// Aggregate data from all runs
	for _, pr := range processedRuns {
		if pr.MCPToolUsage == nil {
			continue
		}

		// Aggregate tool calls
		allToolCalls = append(allToolCalls, pr.MCPToolUsage.ToolCalls...)

		// Aggregate DIFC filtered events
		allFilteredEvents = append(allFilteredEvents, pr.MCPToolUsage.FilteredEvents...)

		// Aggregate tool summaries
		for _, summary := range pr.MCPToolUsage.Summary {
			key := summary.ServerName + ":" + summary.ToolName

			if existing, exists := toolSummaryMap[key]; exists {
				// Store previous count before updating
				prevCallCount := existing.CallCount

				// Merge with existing summary
				existing.CallCount += summary.CallCount
				existing.TotalInputSize += summary.TotalInputSize
				existing.TotalOutputSize += summary.TotalOutputSize

				// Update max sizes
				if summary.MaxInputSize > existing.MaxInputSize {
					existing.MaxInputSize = summary.MaxInputSize
				}
				if summary.MaxOutputSize > existing.MaxOutputSize {
					existing.MaxOutputSize = summary.MaxOutputSize
				}

				// Update error count
				existing.ErrorCount += summary.ErrorCount

				// Recalculate average duration (weighted)
				if summary.AvgDuration != "" && existing.CallCount > 0 {
					existingDur := parseDurationString(existing.AvgDuration)
					newDur := parseDurationString(summary.AvgDuration)
					// Weight by call counts using previous count
					weightedDur := (existingDur*time.Duration(prevCallCount) + newDur*time.Duration(summary.CallCount)) / time.Duration(existing.CallCount)
					existing.AvgDuration = timeutil.FormatDuration(weightedDur)
				}

				// Update max duration
				if summary.MaxDuration != "" {
					maxDur := parseDurationString(summary.MaxDuration)
					existingMaxDur := parseDurationString(existing.MaxDuration)
					if maxDur > existingMaxDur {
						existing.MaxDuration = summary.MaxDuration
					}
				}
			} else {
				// Create new summary entry (copy to avoid mutation)
				newSummary := summary
				toolSummaryMap[key] = &newSummary
			}
		}

		// Aggregate server stats
		for _, serverStats := range pr.MCPToolUsage.Servers {
			if existing, exists := serverStatsMap[serverStats.ServerName]; exists {
				// Store previous count before updating
				prevRequestCount := existing.RequestCount

				// Merge with existing stats
				existing.RequestCount += serverStats.RequestCount
				existing.ToolCallCount += serverStats.ToolCallCount
				existing.TotalInputSize += serverStats.TotalInputSize
				existing.TotalOutputSize += serverStats.TotalOutputSize
				existing.ErrorCount += serverStats.ErrorCount

				// Recalculate average duration (weighted)
				if serverStats.AvgDuration != "" && existing.RequestCount > 0 {
					existingDur := parseDurationString(existing.AvgDuration)
					newDur := parseDurationString(serverStats.AvgDuration)
					// Weight by request counts using previous count
					weightedDur := (existingDur*time.Duration(prevRequestCount) + newDur*time.Duration(serverStats.RequestCount)) / time.Duration(existing.RequestCount)
					existing.AvgDuration = timeutil.FormatDuration(weightedDur)
				}
			} else {
				// Create new server stats entry (copy to avoid mutation)
				newStats := serverStats
				serverStatsMap[serverStats.ServerName] = &newStats
			}
		}
	}

	// Return nil if no MCP tool usage data was found
	if len(toolSummaryMap) == 0 && len(serverStatsMap) == 0 && len(allFilteredEvents) == 0 {
		return nil
	}

	// Convert maps to slices
	var summaries []MCPToolSummary
	for _, summary := range toolSummaryMap {
		summaries = append(summaries, *summary)
	}

	var servers []MCPServerStats
	for _, stats := range serverStatsMap {
		servers = append(servers, *stats)
	}

	// Sort summaries by server name, then tool name
	slices.SortFunc(summaries, func(a, b MCPToolSummary) int {
		if a.ServerName != b.ServerName {
			return cmp.Compare(a.ServerName, b.ServerName)
		}
		return cmp.Compare(a.ToolName, b.ToolName)
	})

	// Sort servers by name
	slices.SortFunc(servers, func(a, b MCPServerStats) int {
		return cmp.Compare(a.ServerName, b.ServerName)
	})

	reportLog.Printf("Built MCP tool usage summary: %d tool summaries, %d servers, %d total tool calls, %d DIFC filtered events",
		len(summaries), len(servers), len(allToolCalls), len(allFilteredEvents))

	return &MCPToolUsageSummary{
		Summary:        summaries,
		Servers:        servers,
		ToolCalls:      allToolCalls,
		FilteredEvents: allFilteredEvents,
	}
}
