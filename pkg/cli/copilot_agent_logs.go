package cli

import (
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

var (
	agentTurnPattern = regexp.MustCompile(`(?i)task.*iteration|agent.*turn|step.*\d+`)
	// agentToolCallPattern matches log lines that indicate an actual tool invocation.
	// It requires either a bullet point prefix (● toolname) or a colon separator after
	// the keyword (e.g. "Tool call:", "Calling:", "Executing tool:"). This prevents
	// false positives from natural language prose that contains words like "tool calls",
	// "calling a function", etc.
	agentToolCallPattern = regexp.MustCompile(`(?i)\btool\s+call\s*:|\btool\s*:|executing\s+tool\s*:|executing\s*:|calling\s*:|using\s+tool\s*:|^\s*●\s*\S`)
	toolNamePatterns     = []*regexp.Regexp{
		// "● toolname" - bullet point format used by Copilot coding agent STDIO logs
		regexp.MustCompile(`^\s*●\s*([a-zA-Z0-9_][a-zA-Z0-9_-]*)`),
		// "Tool call: toolname" - colon separator required
		regexp.MustCompile(`(?i)\btool\s+call\s*:\s*([a-zA-Z0-9_][a-zA-Z0-9_-]*)`),
		// "Tool: toolname" - colon immediately after "tool" required
		regexp.MustCompile(`(?i)\btool\s*:\s*([a-zA-Z0-9_][a-zA-Z0-9_-]*)`),
		// "Calling: toolname" - colon separator required
		regexp.MustCompile(`(?i)\bcalling\s*:\s*([a-zA-Z0-9_][a-zA-Z0-9_-]*)`),
		// "Executing tool: toolname" - colon separator required
		regexp.MustCompile(`(?i)\bexecuting\s+tool\s*:\s*([a-zA-Z0-9_][a-zA-Z0-9_-]*)`),
		// "Executing: toolname" - colon separator required
		regexp.MustCompile(`(?i)\bexecuting\s*:\s*([a-zA-Z0-9_][a-zA-Z0-9_-]*)`),
		// "Using tool: toolname" - colon separator required
		regexp.MustCompile(`(?i)\busing\s+tool\s*:\s*([a-zA-Z0-9_][a-zA-Z0-9_-]*)`),
	}
)

// ParseCopilotCodingAgentLogMetrics extracts metrics from GitHub Copilot coding agent logs
// This is different from Copilot CLI logs and requires specialized parsing
func ParseCopilotCodingAgentLogMetrics(logContent string, verbose bool) workflow.LogMetrics {
	copilotCodingAgentLog.Printf("Parsing GitHub Copilot coding agent log metrics: %d bytes", len(logContent))

	var metrics workflow.LogMetrics
	lines := strings.Split(logContent, "\n")
	toolCallMap := make(map[string]*workflow.ToolCallInfo)
	var currentSequence []string
	turns := 0
	maxTokenUsage := 0

	for _, line := range lines {
		parseCopilotCodingAgentLogMetricsLine(
			line,
			verbose,
			&metrics,
			toolCallMap,
			&currentSequence,
			&turns,
			&maxTokenUsage,
		)
	}

	parseCopilotCodingAgentLogMetricsFinalize(&metrics, toolCallMap, currentSequence, maxTokenUsage, turns)

	copilotCodingAgentLog.Printf("Parsed metrics: tokens=%d, cost=$%.4f, turns=%d",
		metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns)

	return metrics
}

func parseCopilotCodingAgentLogMetricsLine(
	line string,
	verbose bool,
	metrics *workflow.LogMetrics,
	toolCallMap map[string]*workflow.ToolCallInfo,
	currentSequence *[]string,
	turns *int,
	maxTokenUsage *int,
) {
	if strings.TrimSpace(line) == "" {
		return
	}
	parseCopilotCodingAgentLogMetricsTurn(line, metrics, currentSequence, turns)
	parseCopilotCodingAgentLogMetricsTool(line, verbose, toolCallMap, currentSequence)
	parseCopilotCodingAgentLogMetricsJSON(line, verbose, metrics, maxTokenUsage)
}

func parseCopilotCodingAgentLogMetricsTurn(line string, metrics *workflow.LogMetrics, currentSequence *[]string, turns *int) {
	if agentTurnPattern.MatchString(line) {
		(*turns)++
		// Start of a new turn, save previous sequence if any
		if len(*currentSequence) > 0 {
			metrics.ToolSequences = append(metrics.ToolSequences, *currentSequence)
			*currentSequence = []string{}
		}
	}
}

func parseCopilotCodingAgentLogMetricsTool(line string, verbose bool, toolCallMap map[string]*workflow.ToolCallInfo, currentSequence *[]string) {
	if !agentToolCallPattern.MatchString(line) {
		return
	}
	toolName := extractToolName(line)
	if toolName == "" {
		return
	}
	if _, exists := toolCallMap[toolName]; !exists {
		toolCallMap[toolName] = &workflow.ToolCallInfo{
			Name:      toolName,
			CallCount: 0,
		}
	}
	toolCallMap[toolName].CallCount++
	*currentSequence = append(*currentSequence, toolName)
	if verbose {
		copilotCodingAgentLog.Printf("Found tool call: %s", toolName)
	}
}

func parseCopilotCodingAgentLogMetricsJSON(line string, verbose bool, metrics *workflow.LogMetrics, maxTokenUsage *int) {
	jsonMetrics := workflow.ExtractJSONMetrics(line, verbose)
	if jsonMetrics.TokenUsage > 0 || jsonMetrics.EstimatedCost > 0 {
		if jsonMetrics.TokenUsage > *maxTokenUsage {
			*maxTokenUsage = jsonMetrics.TokenUsage
		}
		if jsonMetrics.EstimatedCost > 0 {
			metrics.EstimatedCost += jsonMetrics.EstimatedCost
		}
	}
}

func parseCopilotCodingAgentLogMetricsFinalize(
	metrics *workflow.LogMetrics,
	toolCallMap map[string]*workflow.ToolCallInfo,
	currentSequence []string,
	maxTokenUsage int,
	turns int,
) {
	if len(currentSequence) > 0 {
		metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
	}
	for _, toolInfo := range toolCallMap {
		metrics.ToolCalls = append(metrics.ToolCalls, *toolInfo)
	}
	metrics.TokenUsage = maxTokenUsage
	metrics.Turns = turns
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
