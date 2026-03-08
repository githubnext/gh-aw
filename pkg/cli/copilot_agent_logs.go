package cli

import (
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

var (
	agentTurnPattern     = regexp.MustCompile(`(?i)task.*iteration|agent.*turn|step.*\d+`)
	agentToolCallPattern = regexp.MustCompile(`(?i)tool.*call|executing.*tool|calling.*(\w+)`)
	toolNamePatterns     = []*regexp.Regexp{
		regexp.MustCompile(`(?i)tool[:\s]+([a-zA-Z0-9_-]+)`),
		regexp.MustCompile(`(?i)calling[:\s]+([a-zA-Z0-9_-]+)`),
		regexp.MustCompile(`(?i)executing[:\s]+([a-zA-Z0-9_-]+)`),
		regexp.MustCompile(`(?i)using[:\s]+tool[:\s]+([a-zA-Z0-9_-]+)`),
	}
)

// ParseCopilotCodingAgentLogMetrics extracts metrics from GitHub Copilot coding agent logs
// This is different from Copilot CLI logs and requires specialized parsing
func ParseCopilotCodingAgentLogMetrics(logContent string, verbose bool) workflow.LogMetrics {
	copilotCodingAgentLog.Printf("Parsing GitHub Copilot coding agent log metrics: %d bytes", len(logContent))

	var metrics workflow.LogMetrics
	var maxTokenUsage int

	lines := strings.Split(logContent, "\n")
	toolCallMap := make(map[string]*workflow.ToolCallInfo)
	var currentSequence []string
	turns := 0

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Count turns based on agent iteration patterns
		if agentTurnPattern.MatchString(line) {
			turns++
			// Start of a new turn, save previous sequence if any
			if len(currentSequence) > 0 {
				metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
				currentSequence = []string{}
			}
		}

		// Extract tool calls from agent logs
		if matches := agentToolCallPattern.FindStringSubmatch(line); len(matches) > 1 {
			toolName := extractToolName(line)
			if toolName != "" {
				// Track tool call
				if _, exists := toolCallMap[toolName]; !exists {
					toolCallMap[toolName] = &workflow.ToolCallInfo{
						Name:      toolName,
						CallCount: 0,
					}
				}
				toolCallMap[toolName].CallCount++

				// Add to current sequence
				currentSequence = append(currentSequence, toolName)

				if verbose {
					copilotCodingAgentLog.Printf("Found tool call: %s", toolName)
				}
			}
		}

		// Try to extract token usage from JSON format if available
		jsonMetrics := workflow.ExtractJSONMetrics(line, verbose)
		if jsonMetrics.TokenUsage > 0 || jsonMetrics.EstimatedCost > 0 {
			if jsonMetrics.TokenUsage > maxTokenUsage {
				maxTokenUsage = jsonMetrics.TokenUsage
			}
			if jsonMetrics.EstimatedCost > 0 {
				metrics.EstimatedCost += jsonMetrics.EstimatedCost
			}
		}
	}

	// Add final sequence if any
	if len(currentSequence) > 0 {
		metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
	}

	// Convert tool call map to slice
	for _, toolInfo := range toolCallMap {
		metrics.ToolCalls = append(metrics.ToolCalls, *toolInfo)
	}

	metrics.TokenUsage = maxTokenUsage
	metrics.Turns = turns

	copilotCodingAgentLog.Printf("Parsed metrics: tokens=%d, cost=$%.4f, turns=%d",
		metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns)

	return metrics
}

// extractToolName extracts a tool name from a log line
func extractToolName(line string) string {
	// Try to extract tool name from various patterns
	for _, pattern := range toolNamePatterns {
		if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return ""
}
