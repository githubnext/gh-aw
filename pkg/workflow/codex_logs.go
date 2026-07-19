package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var codexLogsLog = logger.New("workflow:codex_logs")

// ParseLogMetrics implements engine-specific log parsing for Codex
func (e *CodexEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	codexLogsLog.Printf("Parsing Codex log metrics: log_size=%d bytes, lines=%d", len(logContent), strings.Count(logContent, "\n")+1)

	lines := strings.Split(logContent, "\n")
	state := &codexLogParseState{
		toolCallMap: make(map[string]*ToolCallInfo),
	}

	for i := range lines {
		e.processCodexLogLine(lines, i, state)
	}

	// Finalize metrics using shared helper
	FinalizeToolMetrics(FinalizeToolMetricsOptions{
		Metrics:         &state.metrics,
		ToolCallMap:     state.toolCallMap,
		CurrentSequence: state.currentSequence,
		Turns:           state.turns,
		TokenUsage:      state.totalTokenUsage,
	})

	codexLogsLog.Printf("Parsed Codex metrics: turns=%d, token_usage=%d, tool_calls=%d",
		state.metrics.Turns, state.metrics.TokenUsage, len(state.metrics.ToolCalls))

	return state.metrics
}

type codexLogParseState struct {
	metrics         LogMetrics
	totalTokenUsage int
	turns           int
	inThinking      bool
	toolCallMap     map[string]*ToolCallInfo
	currentSequence []string
	lastToolName    string
}

func (e *CodexEngine) processCodexLogLine(lines []string, index int, state *codexLogParseState) {
	line := lines[index]
	if strings.TrimSpace(line) == "" {
		return
	}

	e.updateCodexThinkingState(line, state)
	if toolName := e.parseCodexToolCallsWithSequence(line, state.toolCallMap); toolName != "" {
		state.currentSequence = append(state.currentSequence, toolName)
		state.lastToolName = toolName
	}
	e.updateCodexOutputSize(line, lines, index, state)
	if tokenUsage := e.extractCodexTokenUsage(line); tokenUsage > 0 {
		state.totalTokenUsage += tokenUsage
	}
}

func (e *CodexEngine) updateCodexThinkingState(line string, state *codexLogParseState) {
	trimmedLine := strings.TrimSpace(line)
	if strings.Contains(line, "] thinking") || trimmedLine == "thinking" {
		if !state.inThinking {
			state.turns++
			state.inThinking = true
			if len(state.currentSequence) > 0 {
				state.metrics.ToolSequences = append(state.metrics.ToolSequences, state.currentSequence)
				state.currentSequence = []string{}
			}
		}
	} else if strings.Contains(line, "] tool") || strings.Contains(line, "] exec") || strings.Contains(line, "] codex") ||
		strings.HasPrefix(trimmedLine, "tool ") || strings.HasPrefix(trimmedLine, "exec ") {
		state.inThinking = false
	}
}

func (e *CodexEngine) updateCodexOutputSize(line string, lines []string, index int, state *codexLogParseState) {
	outputSize := e.extractOutputSizeFromResult(line, lines, index)
	if outputSize <= 0 || state.lastToolName == "" {
		return
	}
	if toolInfo, exists := state.toolCallMap[state.lastToolName]; exists && outputSize > toolInfo.MaxOutputSize {
		toolInfo.MaxOutputSize = outputSize
		codexLogsLog.Printf("Updated %s MaxOutputSize to %d characters", state.lastToolName, outputSize)
	}
}

// parseCodexToolCallsWithSequence extracts tool call information from Codex log lines and returns tool name
func (e *CodexEngine) parseCodexToolCallsWithSequence(line string, toolCallMap map[string]*ToolCallInfo) string {
	trimmedLine := strings.TrimSpace(line)

	if toolName := parseCodexToolName(line, trimmedLine); toolName != "" {
		return recordCodexToolCall(toolName, toolCallMap)
	}
	if execCommand := parseCodexExecCommand(line, trimmedLine); execCommand != "" {
		return recordCodexExecCommand(execCommand, toolCallMap)
	}
	e.updateCodexDuration(line, toolCallMap)
	return "" // No tool call found
}

func parseCodexToolName(line, trimmedLine string) string {
	if strings.Contains(line, "] tool ") && strings.Contains(line, "(") {
		if match := codexToolCallOldFormat.FindStringSubmatch(line); len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}
	if strings.HasPrefix(trimmedLine, "tool ") && strings.Contains(trimmedLine, "(") {
		if match := codexToolCallNewFormat.FindStringSubmatch(trimmedLine); len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}
	return ""
}

func recordCodexToolCall(toolName string, toolCallMap map[string]*ToolCallInfo) string {
	prettifiedName := PrettifyToolName(toolName)
	if strings.Contains(toolName, ".") {
		parts := strings.Split(toolName, ".")
		if len(parts) >= 2 {
			prettifiedName = fmt.Sprintf("%s_%s", parts[0], strings.Join(parts[1:], "_"))
		}
	}
	incrementCodexToolInfo(prettifiedName, toolCallMap)
	return prettifiedName
}

func parseCodexExecCommand(line, trimmedLine string) string {
	if strings.Contains(line, "] exec ") {
		if match := codexExecCommandOldFormat.FindStringSubmatch(line); len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}
	if strings.HasPrefix(trimmedLine, "exec ") {
		if match := codexExecCommandNewFormat.FindStringSubmatch(trimmedLine); len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}
	return ""
}

func recordCodexExecCommand(execCommand string, toolCallMap map[string]*ToolCallInfo) string {
	uniqueBashName := "bash_" + ShortenCommand(execCommand)
	incrementCodexToolInfo(uniqueBashName, toolCallMap)
	return uniqueBashName
}

func incrementCodexToolInfo(name string, toolCallMap map[string]*ToolCallInfo) {
	if toolInfo, exists := toolCallMap[name]; exists {
		toolInfo.CallCount++
		return
	}
	toolCallMap[name] = &ToolCallInfo{
		Name:          name,
		CallCount:     1,
		MaxOutputSize: 0,
		MaxDuration:   0,
	}
}

func (e *CodexEngine) updateCodexDuration(line string, toolCallMap map[string]*ToolCallInfo) {
	if !strings.Contains(line, "success in") && !strings.Contains(line, "failure in") && !strings.Contains(line, "failed in") {
		return
	}
	if match := codexDurationPattern.FindStringSubmatch(line); len(match) > 1 {
		if durationSeconds, err := strconv.ParseFloat(match[1], 64); err == nil {
			duration := time.Duration(durationSeconds * float64(time.Second))
			e.updateMostRecentToolWithDuration(toolCallMap, duration)
		}
	}
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

// extractOutputSizeFromResult extracts output size from success/failure result lines
// Returns the character count of the output content if found, 0 otherwise
func (e *CodexEngine) extractOutputSizeFromResult(line string, lines []string, currentIndex int) int {
	// Check if this is a success or failure line
	if !strings.Contains(line, "success in") && !strings.Contains(line, "failure in") && !strings.Contains(line, "failed in") {
		return 0
	}

	// Parse JSON block following the result line
	// The format is typically:
	// [timestamp] tool.method(...) success in Xms:
	// {
	//   "content": [...],
	//   "isError": false
	// }

	var jsonLines []string
	inJSON := false
	braceCount := 0

	// Look ahead to collect JSON block
	for i := currentIndex + 1; i < len(lines); i++ {
		trimmedLine := strings.TrimSpace(lines[i])

		// Start of JSON block
		if !inJSON && trimmedLine == "{" {
			inJSON = true
			braceCount = 1
			jsonLines = append(jsonLines, lines[i])
			continue
		}

		if inJSON {
			jsonLines = append(jsonLines, lines[i])
			// Count braces to detect end of JSON
			braceCount += strings.Count(lines[i], "{")
			braceCount -= strings.Count(lines[i], "}")

			if braceCount == 0 {
				break
			}
		}

		// If we hit a non-empty line that's not part of JSON, stop
		if !inJSON && trimmedLine != "" {
			break
		}
	}

	if len(jsonLines) == 0 {
		return 0
	}

	// Parse the JSON to extract content
	jsonStr := strings.Join(jsonLines, "\n")
	outputSize := e.extractOutputSizeFromJSON(jsonStr)

	return outputSize
}

// extractOutputSizeFromJSON extracts the output size from a Codex result JSON block
func (e *CodexEngine) extractOutputSizeFromJSON(jsonStr string) int {
	// Try to parse as proper JSON first
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// If JSON parsing fails, fallback to simple string extraction
		codexLogsLog.Printf("Failed to parse JSON result, using fallback: %v", err)
		return e.extractOutputSizeFromJSONFallback(jsonStr)
	}

	// Extract content array
	contentInterface, exists := result["content"]
	if !exists {
		return 0
	}

	contentArray, ok := contentInterface.([]any)
	if !ok {
		return 0
	}

	// Sum up text content from all content items
	totalSize := 0
	for _, item := range contentArray {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		// Look for text field
		if text, exists := itemMap["text"]; exists {
			if textStr, ok := text.(string); ok {
				totalSize += len(textStr)
			}
		}
	}

	return totalSize
}

// extractOutputSizeFromJSONFallback is a fallback method for extracting output size
// when proper JSON parsing fails
func (e *CodexEngine) extractOutputSizeFromJSONFallback(jsonStr string) int {
	// For simple extraction without full JSON parsing, look for "text" fields in content array
	// Format: {"content": [{"text": "...", "type": "text"}], "isError": false}

	// Find all text content - use a simple approach counting characters in quoted strings
	// after "text": markers
	totalSize := 0

	// Split by "text": to find text content
	parts := strings.Split(jsonStr, "\"text\":")
	for i := 1; i < len(parts); i++ {
		// Find the quoted string value
		part := strings.TrimSpace(parts[i])
		if part == "" || part[0] != '"' {
			continue
		}

		// Find the closing quote, handling escaped quotes
		inEscape := false
		endQuote := -1
		for j := 1; j < len(part); j++ {
			if inEscape {
				inEscape = false
				continue
			}
			if part[j] == '\\' {
				inEscape = true
				continue
			}
			if part[j] == '"' {
				endQuote = j
				break
			}
		}

		if endQuote > 0 {
			textContent := part[1:endQuote]
			totalSize += len(textContent)
		}
	}

	return totalSize
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

// GetErrorDetectionScriptId returns the JavaScript script name for detecting
// post-run agent errors from the host runner (including invalid/unsupported model names).
func (e *CodexEngine) GetErrorDetectionScriptId() string {
	return "detect_agent_errors"
}
