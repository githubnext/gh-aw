package workflow

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var codexLogsLog = logger.New("workflow:codex_logs")

// ParseLogMetrics implements engine-specific log parsing for Codex
func (e *CodexEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	codexLogsLog.Printf("Parsing Codex log metrics: log_size=%d bytes, lines=%d", len(logContent), len(strings.Split(logContent, "\n")))

	var metrics LogMetrics
	var totalTokenUsage int

	lines := strings.Split(logContent, "\n")
	turns := 0
	inThinkingSection := false
	toolCallMap := make(map[string]*ToolCallInfo) // Track tool calls
	var currentSequence []string                  // Track tool sequence

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Detect thinking sections as indicators of turns
		// Support both old format: "] thinking" and new Rust format: "thinking" (standalone line)
		trimmedLine := strings.TrimSpace(line)
		if strings.Contains(line, "] thinking") || trimmedLine == "thinking" {
			if !inThinkingSection {
				turns++
				inThinkingSection = true
				// Start of a new thinking section, save previous sequence if any
				if len(currentSequence) > 0 {
					metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
					currentSequence = []string{}
				}
			}
		} else if strings.Contains(line, "] tool") || strings.Contains(line, "] exec") || strings.Contains(line, "] codex") ||
			strings.HasPrefix(trimmedLine, "tool ") || strings.HasPrefix(trimmedLine, "exec ") {
			inThinkingSection = false
		}

		// Extract tool calls from Codex logs and add to sequence
		if toolName := e.parseCodexToolCallsWithSequence(line, toolCallMap); toolName != "" {
			currentSequence = append(currentSequence, toolName)
		}

		// Extract Codex-specific token usage (always sum for Codex)
		if tokenUsage := e.extractCodexTokenUsage(line); tokenUsage > 0 {
			totalTokenUsage += tokenUsage
		}

		// Basic processing - error/warning counting moved to end of function
	}

	// Finalize metrics using shared helper
	FinalizeToolMetrics(FinalizeToolMetricsOptions{
		Metrics:         &metrics,
		ToolCallMap:     toolCallMap,
		CurrentSequence: currentSequence,
		Turns:           turns,
		TokenUsage:      totalTokenUsage,
	})

	codexLogsLog.Printf("Parsed Codex metrics: turns=%d, token_usage=%d, tool_calls=%d",
		metrics.Turns, metrics.TokenUsage, len(metrics.ToolCalls))

	return metrics
}

// parseCodexToolCallsWithSequence extracts tool call information from Codex log lines and returns tool name
func (e *CodexEngine) parseCodexToolCallsWithSequence(line string, toolCallMap map[string]*ToolCallInfo) string {
	trimmedLine := strings.TrimSpace(line)

	// Parse tool calls: "] tool provider.method(...)" (old format)
	// or "tool provider.method(...)" (new Rust format)
	var toolName string

	// Try old format first: "] tool provider.method(...)"
	if strings.Contains(line, "] tool ") && strings.Contains(line, "(") {
		if match := codexToolCallOldFormat.FindStringSubmatch(line); len(match) > 1 {
			toolName = strings.TrimSpace(match[1])
		}
	}

	// Try new Rust format: "tool provider.method(...)"
	if toolName == "" && strings.HasPrefix(trimmedLine, "tool ") && strings.Contains(trimmedLine, "(") {
		if match := codexToolCallNewFormat.FindStringSubmatch(trimmedLine); len(match) > 1 {
			toolName = strings.TrimSpace(match[1])
		}
	}

	if toolName != "" {
		prettifiedName := PrettifyToolName(toolName)

		// For Codex, format provider.method as provider_method (avoiding colons)
		if strings.Contains(toolName, ".") {
			parts := strings.Split(toolName, ".")
			if len(parts) >= 2 {
				provider := parts[0]
				method := strings.Join(parts[1:], "_")
				prettifiedName = fmt.Sprintf("%s_%s", provider, method)
			}
		}

		// Initialize or update tool call info
		if toolInfo, exists := toolCallMap[prettifiedName]; exists {
			toolInfo.CallCount++
		} else {
			toolCallMap[prettifiedName] = &ToolCallInfo{
				Name:          prettifiedName,
				CallCount:     1,
				MaxOutputSize: 0, // TODO: Extract output size from results if available
				MaxDuration:   0, // Will be updated when duration is found
			}
		}

		return prettifiedName
	}

	// Parse exec commands: "] exec command" (old format)
	// or "exec command in" (new Rust format) - treat as bash calls
	var execCommand string

	// Try old format: "] exec command in"
	if strings.Contains(line, "] exec ") {
		if match := codexExecCommandOldFormat.FindStringSubmatch(line); len(match) > 1 {
			execCommand = strings.TrimSpace(match[1])
		}
	}

	// Try new Rust format: "exec command in"
	if execCommand == "" && strings.HasPrefix(trimmedLine, "exec ") {
		if match := codexExecCommandNewFormat.FindStringSubmatch(trimmedLine); len(match) > 1 {
			execCommand = strings.TrimSpace(match[1])
		}
	}

	if execCommand != "" {
		// Create unique bash entry with command info, avoiding colons
		uniqueBashName := fmt.Sprintf("bash_%s", ShortenCommand(execCommand))

		// Initialize or update tool call info
		if toolInfo, exists := toolCallMap[uniqueBashName]; exists {
			toolInfo.CallCount++
		} else {
			toolCallMap[uniqueBashName] = &ToolCallInfo{
				Name:          uniqueBashName,
				CallCount:     1,
				MaxOutputSize: 0,
				MaxDuration:   0, // Will be updated when duration is found
			}
		}

		return uniqueBashName
	}

	// Parse duration from success/failure lines: "] success in 0.2s" or "] failure in 1.5s"
	if strings.Contains(line, "success in") || strings.Contains(line, "failure in") || strings.Contains(line, "failed in") {
		// Extract duration pattern like "in 0.2s", "in 1.5s"
		if match := codexDurationPattern.FindStringSubmatch(line); len(match) > 1 {
			if durationSeconds, err := strconv.ParseFloat(match[1], 64); err == nil {
				duration := time.Duration(durationSeconds * float64(time.Second))

				// Find the most recent tool call to associate with this duration
				// Since we don't have direct association, we'll update the most recent entry
				// This is a limitation of the log format, but it's the best we can do
				e.updateMostRecentToolWithDuration(toolCallMap, duration)
			}
		}
	}

	return "" // No tool call found
}

// updateMostRecentToolWithDuration updates the tool with maximum duration
// Since we can't perfectly correlate duration lines with specific tool calls in Codex logs,
// we approximate by updating any tool that doesn't have a duration yet, or updating the max
func (e *CodexEngine) updateMostRecentToolWithDuration(toolCallMap map[string]*ToolCallInfo, duration time.Duration) {
	// Find a tool that either has no duration yet or can be updated with a larger duration
	for _, toolInfo := range toolCallMap {
		if toolInfo.MaxDuration == 0 || duration > toolInfo.MaxDuration {
			toolInfo.MaxDuration = duration
			// Only update one tool per duration line to avoid over-attribution
			break
		}
	}
}

// extractCodexTokenUsage extracts token usage from Codex-specific log lines
func (e *CodexEngine) extractCodexTokenUsage(line string) int {
	// Codex format 1: "tokens used: 13934"
	// Use pre-compiled pattern for performance
	if match := codexTokenUsagePattern.FindStringSubmatch(line); len(match) > 1 {
		if count, err := strconv.Atoi(match[1]); err == nil {
			return count
		}
	}

	// Codex format 2: "TokenCount(TokenCountEvent { ... total_tokens: 13281 ..."
	// This pattern appears in newer Codex logs
	if match := codexTotalTokensPattern.FindStringSubmatch(line); len(match) > 1 {
		if count, err := strconv.Atoi(match[1]); err == nil {
			return count
		}
	}

	return 0
}

// GetLogParserScriptId returns the JavaScript script name for parsing Codex logs
func (e *CodexEngine) GetLogParserScriptId() string {
	return "parse_codex_log"
}
