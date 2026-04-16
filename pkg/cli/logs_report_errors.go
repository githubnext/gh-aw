package cli

import (
	"cmp"
	"slices"
	"strings"
)

// ErrorSummary contains aggregated error/warning statistics
type ErrorSummary struct {
	Type         string `json:"type" console:"header:Type"`
	Message      string `json:"message" console:"header:Message,maxlen:80"`
	Count        int    `json:"count" console:"header:Occurrences"`
	Engine       string `json:"engine,omitempty" console:"header:Engine,omitempty"`
	RunID        int64  `json:"run_id" console:"header:Sample Run"`
	RunURL       string `json:"run_url" console:"-"`
	WorkflowName string `json:"workflow_name,omitempty" console:"-"`
	PatternID    string `json:"pattern_id,omitempty" console:"-"`
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

// buildCombinedErrorsSummary is an intentional no-op compatibility stub retained
// after error-pattern support was removed.
func buildCombinedErrorsSummary(_ []ProcessedRun) []ErrorSummary {
	// Preserve the previous call shape while reporting no combined errors.
	return []ErrorSummary{}
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
