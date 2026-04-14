// This file provides command-line interface functionality for gh-aw.
// This file (logs_report_tools.go) contains tool usage, missing tool, MCP failure,
// and combined error summary building functions for the logs report.

package cli

import (
	"cmp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/timeutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// isValidToolName checks if a tool name appears to be valid
// Filters out single words, common words, and other garbage that shouldn't be tools
func isValidToolName(toolName string) bool {
	name := strings.TrimSpace(toolName)

	// Filter out empty names
	if name == "" || name == "-" {
		return false
	}

	// Filter out single character names
	if len(name) == 1 {
		return false
	}

	// Filter out common English words that are likely from error messages
	commonWords := map[string]bool{
		"calls": true, "to": true, "for": true, "the": true, "a": true, "an": true,
		"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true, "may": true, "might": true,
		"Testing": true, "multiple": true, "launches": true, "command": true, "invocation": true,
		"with": true, "from": true, "by": true, "at": true, "in": true, "on": true,
	}

	if commonWords[name] {
		return false
	}

	// Tool names should typically contain underscores, hyphens, or be camelCase
	// or be all lowercase. Single words without these patterns are suspect.
	hasUnderscore := strings.Contains(name, "_")
	hasHyphen := strings.Contains(name, "-")
	hasCapital := strings.ToLower(name) != name

	// If it's a single word with no underscores/hyphens and is lowercase and short,
	// it's likely a fragment
	words := strings.Fields(name)
	if len(words) == 1 && !hasUnderscore && !hasHyphen && len(name) < 10 && !hasCapital {
		// Could be a fragment - be conservative and reject if it's a common word
		return false
	}

	return true
}

func buildToolUsageSummary(processedRuns []ProcessedRun) []ToolUsageSummary {
	reportLog.Printf("Building tool usage summary from %d processed runs", len(processedRuns))
	toolStats := make(map[string]*ToolUsageSummary)

	for _, pr := range processedRuns {
		// Extract metrics from run's logs
		metrics := ExtractLogMetricsFromRun(pr)

		// Track which runs use each tool
		toolRunTracker := make(map[string]bool)

		for _, toolCall := range metrics.ToolCalls {
			displayKey := workflow.PrettifyToolName(toolCall.Name)

			// Filter out invalid tool names
			if !isValidToolName(displayKey) {
				continue
			}

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
	slices.SortFunc(result, func(a, b ToolUsageSummary) int {
		return cmp.Compare(b.TotalCalls, a.TotalCalls)
	})

	return result
}

// addUniqueWorkflow adds a workflow to the list if it's not already present
func addUniqueWorkflow(workflows []string, workflow string) []string {
	if slices.Contains(workflows, workflow) {
		return workflows
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
	reportLog.Printf("Building missing tools summary from %d processed runs", len(processedRuns))
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
	slices.SortFunc(result, func(a, b MissingToolSummary) int {
		return cmp.Compare(b.Count, a.Count)
	})

	return result
}

// buildMissingDataSummary aggregates missing data across all runs
func buildMissingDataSummary(processedRuns []ProcessedRun) []MissingDataSummary {
	reportLog.Printf("Building missing data summary from %d processed runs", len(processedRuns))
	result := aggregateSummaryItems(
		processedRuns,
		// getItems: extract missing data from each run
		func(pr ProcessedRun) []MissingDataReport {
			return pr.MissingData
		},
		// getKey: use data type as the aggregation key
		func(data MissingDataReport) string {
			return data.DataType
		},
		// createSummary: create new summary for first occurrence
		func(data MissingDataReport) *MissingDataSummary {
			return &MissingDataSummary{
				DataType:    data.DataType,
				Count:       1,
				Workflows:   []string{data.WorkflowName},
				FirstReason: data.Reason,
				RunIDs:      []int64{data.RunID},
			}
		},
		// updateSummary: update existing summary with new occurrence
		func(summary *MissingDataSummary, data MissingDataReport) {
			summary.Count++
			summary.Workflows = addUniqueWorkflow(summary.Workflows, data.WorkflowName)
			summary.RunIDs = append(summary.RunIDs, data.RunID)
		},
		// finalizeSummary: populate display fields for console rendering
		func(summary *MissingDataSummary) {
			summary.WorkflowsDisplay = strings.Join(summary.Workflows, ", ")
			summary.FirstReasonDisplay = summary.FirstReason
		},
	)

	// Sort by count descending
	slices.SortFunc(result, func(a, b MissingDataSummary) int {
		return cmp.Compare(b.Count, a.Count)
	})

	return result
}

// buildMCPFailuresSummary aggregates MCP failures across all runs
func buildMCPFailuresSummary(processedRuns []ProcessedRun) []MCPFailureSummary {
	reportLog.Printf("Building MCP failures summary from %d processed runs", len(processedRuns))
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
	slices.SortFunc(result, func(a, b MCPFailureSummary) int {
		return cmp.Compare(b.Count, a.Count)
	})

	return result
}

func buildCombinedErrorsSummary(processedRuns []ProcessedRun) []ErrorSummary {
	// Return empty slice since error patterns have been removed
	return []ErrorSummary{}
}
