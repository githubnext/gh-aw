package cli

import (
	"cmp"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/timeutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// ToolUsageSummary contains aggregated tool usage statistics
type ToolUsageSummary struct {
	Name          string `json:"name" console:"header:Tool"`
	TotalCalls    int    `json:"total_calls" console:"header:Total Calls,format:number"`
	Runs          int    `json:"runs" console:"header:Runs"` // Number of runs that used this tool
	MaxOutputSize int    `json:"max_output_size,omitempty" console:"header:Max Output,format:filesize,default:N/A,omitempty"`
	MaxDuration   string `json:"max_duration,omitempty" console:"header:Max Duration,default:N/A,omitempty"`
}

// toolNameStopWords is a set of common English words that should never be treated as tool names.
// Built once at package init and reused across all isValidToolName calls.
var toolNameStopWords = map[string]bool{
	"calls": true, "to": true, "for": true, "the": true, "a": true, "an": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true, "may": true, "might": true,
	"Testing": true, "multiple": true, "launches": true, "command": true, "invocation": true,
	"with": true, "from": true, "by": true, "at": true, "in": true, "on": true,
}

// isValidToolName checks if a tool name appears to be valid.
// Filters out single words, common words, and other garbage that shouldn't be tools.
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
	if toolNameStopWords[name] {
		return false
	}

	// Tool names should typically contain underscores, hyphens, or be camelCase
	// or be all lowercase. Single words without these patterns are suspect.
	hasUnderscore := strings.Contains(name, "_")
	hasHyphen := strings.Contains(name, "-")
	hasCapital := strings.ToLower(name) != name

	// Reject short, all-lowercase, single-word names with no separators — these
	// are almost certainly log-message fragments rather than real tool names.
	words := strings.Fields(name)
	if len(words) == 1 && !hasUnderscore && !hasHyphen && len(name) < 10 && !hasCapital {
		return false
	}

	return true
}

// buildToolUsageSummary aggregates tool usage across all runs
// Filters out invalid tool names that appear to be fragments or garbage
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
