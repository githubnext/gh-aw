package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
)

var claudeLogsLog = logger.New("workflow:claude_logs")

// ParseLogMetrics implements engine-specific log parsing for Claude
func (e *ClaudeEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	claudeLogsLog.Printf("Parsing Claude log metrics: %d bytes", len(logContent))
	var metrics LogMetrics
	var maxTokenUsage int

	// First try to parse as JSON array (Claude logs are structured as JSON arrays)
	if strings.TrimSpace(logContent) != "" {
		if resultMetrics := e.parseClaudeJSONLog(logContent, verbose); resultMetrics.TokenUsage > 0 || resultMetrics.EstimatedCost > 0 || resultMetrics.Turns > 0 || len(resultMetrics.ToolCalls) > 0 || len(resultMetrics.ToolSequences) > 0 {
			metrics.TokenUsage = resultMetrics.TokenUsage
			metrics.EstimatedCost = resultMetrics.EstimatedCost
			metrics.Turns = resultMetrics.Turns
			metrics.ToolCalls = resultMetrics.ToolCalls         // Copy tool calls
			metrics.ToolSequences = resultMetrics.ToolSequences // Copy tool sequences
		}
	}

	// Process line by line for error counting and fallback parsing
	lines := strings.SplitSeq(logContent, "\n")

	for line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// If we haven't found cost data yet from JSON parsing, try streaming JSON
		if metrics.TokenUsage == 0 || metrics.EstimatedCost == 0 || metrics.Turns == 0 {
			jsonMetrics := ExtractJSONMetrics(line, verbose)
			if jsonMetrics.TokenUsage > 0 || jsonMetrics.EstimatedCost > 0 {
				// Check if this is a Claude result payload with aggregated costs
				if e.isClaudeResultPayload(line) {
					// For Claude result payloads, use the aggregated values directly
					if resultMetrics := e.extractClaudeResultMetrics(line); resultMetrics.TokenUsage > 0 || resultMetrics.EstimatedCost > 0 || resultMetrics.Turns > 0 {
						metrics.TokenUsage = resultMetrics.TokenUsage
						metrics.EstimatedCost = resultMetrics.EstimatedCost
						metrics.Turns = resultMetrics.Turns
					}
				} else {
					// For streaming JSON, keep the maximum token usage found
					if jsonMetrics.TokenUsage > maxTokenUsage {
						maxTokenUsage = jsonMetrics.TokenUsage
					}
					if metrics.EstimatedCost == 0 && jsonMetrics.EstimatedCost > 0 {
						metrics.EstimatedCost += jsonMetrics.EstimatedCost
					}
				}
				continue
			}
		}
	}

	// If no result payload was found, use the maximum from streaming JSON
	if metrics.TokenUsage == 0 {
		metrics.TokenUsage = maxTokenUsage
	}

	claudeLogsLog.Printf("Parsed log metrics: tokens=%d, cost=$%.4f, turns=%d", metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns)
	return metrics
}

// isClaudeResultPayload checks if the JSON line is a Claude result payload with type: "result"
func (e *ClaudeEngine) isClaudeResultPayload(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return false
	}

	var jsonData map[string]any
	if err := json.Unmarshal([]byte(trimmed), &jsonData); err != nil {
		return false
	}

	typeField, exists := jsonData["type"]
	if !exists {
		return false
	}

	typeStr, ok := typeField.(string)
	return ok && typeStr == "result"
}

// extractClaudeResultMetrics extracts metrics from Claude result payload
func (e *ClaudeEngine) extractClaudeResultMetrics(line string) LogMetrics {
	claudeLogsLog.Print("Extracting metrics from Claude result payload")
	var metrics LogMetrics

	trimmed := strings.TrimSpace(line)
	var jsonData map[string]any
	if err := json.Unmarshal([]byte(trimmed), &jsonData); err != nil {
		return metrics
	}

	// Extract total_cost_usd directly
	if totalCost, exists := jsonData["total_cost_usd"]; exists {
		if cost := typeutil.ConvertToFloat(totalCost); cost > 0 {
			metrics.EstimatedCost = cost
		}
	}

	// Extract usage information with all token types
	if usage, exists := jsonData["usage"]; exists {
		if usageMap, ok := usage.(map[string]any); ok {
			inputTokens := typeutil.ConvertToInt(usageMap["input_tokens"])
			outputTokens := typeutil.ConvertToInt(usageMap["output_tokens"])
			cacheCreationTokens := typeutil.ConvertToInt(usageMap["cache_creation_input_tokens"])
			cacheReadTokens := typeutil.ConvertToInt(usageMap["cache_read_input_tokens"])

			totalTokens := inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens
			if totalTokens > 0 {
				metrics.TokenUsage = totalTokens
			}
		}
	}

	// Extract number of turns
	if numTurns, exists := jsonData["num_turns"]; exists {
		if turns := typeutil.ConvertToInt(numTurns); turns > 0 {
			metrics.Turns = turns
		}
	}

	// Note: Duration extraction is handled in the main parsing logic where we have access to tool calls
	// This is because we need to distribute duration among tool calls

	claudeLogsLog.Printf("Extracted Claude result metrics: tokens=%d, cost=$%.4f, turns=%d", metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns)
	return metrics
}

// parseClaudeJSONLog parses Claude logs as a JSON array or mixed format (debug logs + JSONL)
func (e *ClaudeEngine) parseClaudeJSONLog(logContent string, verbose bool) LogMetrics {
	claudeLogsLog.Print("Attempting to parse Claude JSON log")
	var metrics LogMetrics

	logEntries, ok := parseClaudeLogEntries(logContent, verbose)
	if !ok {
		return metrics
	}

	toolCallMap := make(map[string]*ToolCallInfo)
	var currentSequence []string

	for _, entry := range logEntries {
		if typeStr, _ := entry["type"].(string); typeStr == "result" {
			applyClaudeResultEntryMetrics(entry, &metrics)
			if verbose {
				fmt.Fprintf(os.Stderr, "Extracted from Claude result payload: tokens=%d, cost=%.4f, turns=%d\n",
					metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns)
			}
			break
		} else if typeStr == "assistant" {
			currentSequence = e.appendClaudeAssistantToolSequence(entry, toolCallMap, currentSequence)
		}

		if entry["type"] == "user" {
			e.parseClaudeUserToolResults(entry, toolCallMap)
		}
	}

	FinalizeToolCallsAndSequence(&metrics, toolCallMap, currentSequence)
	claudeLogsLog.Printf("Parsed %d log entries: tokens=%d, cost=$%.4f, turns=%d, tool_types=%d",
		len(logEntries), metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns, len(metrics.ToolCalls))
	logClaudeToolSequenceSummary(metrics, verbose)
	return metrics
}

func parseClaudeLogEntries(logContent string, verbose bool) ([]map[string]any, bool) {
	var logEntries []map[string]any
	if err := json.Unmarshal([]byte(logContent), &logEntries); err == nil {
		return logEntries, true
	} else {
		claudeLogsLog.Print("JSON array parse failed, trying JSONL format")
		if verbose {
			fmt.Fprintf(os.Stderr, "Failed to parse Claude log as JSON array, trying JSONL format: %v\n", err)
		}
	}
	logEntries = parseClaudeMixedLogEntries(logContent, verbose)
	if len(logEntries) == 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "No valid JSON entries found in Claude log\n")
		}
		return nil, false
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Extracted %d JSON entries from mixed format Claude log\n", len(logEntries))
	}
	return logEntries, true
}

func parseClaudeMixedLogEntries(logContent string, verbose bool) []map[string]any {
	var logEntries []map[string]any
	lines := strings.Split(logContent, "\n")
	for i := 0; i < len(lines); i++ {
		trimmedLine := strings.TrimSpace(lines[i])
		if trimmedLine == "" {
			continue
		}
		if strings.HasPrefix(trimmedLine, "[") {
			arr, consumedTo, ok := parseClaudeJSONArrayFromLines(lines, i, trimmedLine)
			if ok {
				logEntries = append(logEntries, arr...)
				i = consumedTo
				continue
			}
		}
		if !strings.HasPrefix(trimmedLine, "{") {
			continue
		}
		var jsonEntry map[string]any
		if err := json.Unmarshal([]byte(trimmedLine), &jsonEntry); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Skipping invalid JSON line: %s\n", trimmedLine)
			}
			continue
		}
		logEntries = append(logEntries, jsonEntry)
	}
	return logEntries
}

func parseClaudeJSONArrayFromLines(lines []string, start int, trimmedLine string) ([]map[string]any, int, bool) {
	buf := trimmedLine
	consumedTo := start
	if !strings.Contains(trimmedLine, "]") {
		var sb strings.Builder
		for j := start + 1; j < len(lines); j++ {
			sb.WriteString("\n" + lines[j])
			if strings.Contains(lines[j], "]") {
				consumedTo = j
				break
			}
		}
		buf += sb.String()
	}
	if arr, ok := unmarshalClaudeJSONArray(buf); ok {
		return arr, consumedTo, true
	}
	openIdx := strings.Index(buf, "[")
	closeIdx := strings.LastIndex(buf, "]")
	if openIdx != -1 && closeIdx != -1 && closeIdx > openIdx {
		if arr, ok := unmarshalClaudeJSONArray(buf[openIdx : closeIdx+1]); ok {
			return arr, consumedTo, true
		}
	}
	return nil, consumedTo, false
}

func unmarshalClaudeJSONArray(raw string) ([]map[string]any, bool) {
	var arr []map[string]any
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil, false
	}
	return arr, true
}

func applyClaudeResultEntryMetrics(entry map[string]any, metrics *LogMetrics) {
	if totalCost, exists := entry["total_cost_usd"]; exists {
		if cost := typeutil.ConvertToFloat(totalCost); cost > 0 {
			metrics.EstimatedCost = cost
		}
	}
	if usage, exists := entry["usage"]; exists {
		if usageMap, ok := usage.(map[string]any); ok {
			inputTokens := typeutil.ConvertToInt(usageMap["input_tokens"])
			outputTokens := typeutil.ConvertToInt(usageMap["output_tokens"])
			cacheCreationTokens := typeutil.ConvertToInt(usageMap["cache_creation_input_tokens"])
			cacheReadTokens := typeutil.ConvertToInt(usageMap["cache_read_input_tokens"])
			if totalTokens := inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens; totalTokens > 0 {
				metrics.TokenUsage = totalTokens
			}
		}
	}
	if numTurns, exists := entry["num_turns"]; exists {
		if turns := typeutil.ConvertToInt(numTurns); turns > 0 {
			metrics.Turns = turns
		}
	}
}

func (e *ClaudeEngine) appendClaudeAssistantToolSequence(entry map[string]any, toolCallMap map[string]*ToolCallInfo, currentSequence []string) []string {
	if contentArray, ok := claudeLogContentArray(entry); ok {
		if sequenceInMessage := e.parseToolCallsWithSequence(contentArray, toolCallMap); len(sequenceInMessage) > 0 {
			currentSequence = append(currentSequence, sequenceInMessage...)
		}
	}
	return currentSequence
}

func (e *ClaudeEngine) parseClaudeUserToolResults(entry map[string]any, toolCallMap map[string]*ToolCallInfo) {
	if contentArray, ok := claudeLogContentArray(entry); ok {
		e.parseToolCallsWithSequence(contentArray, toolCallMap)
	}
}

func claudeLogContentArray(entry map[string]any) ([]any, bool) {
	message, exists := entry["message"]
	if !exists {
		return nil, false
	}
	messageMap, ok := message.(map[string]any)
	if !ok {
		return nil, false
	}
	content, exists := messageMap["content"]
	if !exists {
		return nil, false
	}
	contentArray, ok := content.([]any)
	return contentArray, ok
}

func logClaudeToolSequenceSummary(metrics LogMetrics, verbose bool) {
	if !verbose || len(metrics.ToolSequences) == 0 {
		return
	}
	totalTools := 0
	for _, seq := range metrics.ToolSequences {
		totalTools += len(seq)
	}
	fmt.Fprintf(os.Stderr, "Claude parser extracted %d tool sequences with %d total tool calls\n",
		len(metrics.ToolSequences), totalTools)
}

// parseToolCallsWithSequence extracts tool call information from Claude log content array and returns sequence
func (e *ClaudeEngine) parseToolCallsWithSequence(contentArray []any, toolCallMap map[string]*ToolCallInfo) []string {
	var sequence []string

	for _, contentItem := range contentArray {
		contentMap, ok := contentItem.(map[string]any)
		if !ok {
			continue
		}

		typeStr, ok := typeutil.LookupString(contentMap, "type")
		if !ok {
			continue
		}

		switch typeStr {
		case "tool_use":
			if prettifiedName, ok := e.parseClaudeToolUse(contentMap, toolCallMap); ok {
				sequence = append(sequence, prettifiedName)
			}
		case "tool_result":
			updateClaudeToolResultOutputSizes(contentMap, toolCallMap)
		}
	}

	return sequence
}

func (e *ClaudeEngine) parseClaudeToolUse(contentMap map[string]any, toolCallMap map[string]*ToolCallInfo) (string, bool) {
	nameStr, ok := typeutil.LookupString(contentMap, "name")
	if !ok {
		return "", false
	}
	prettifiedName := claudePrettifiedToolName(contentMap, nameStr)
	inputSize := 0
	if input, exists := contentMap["input"]; exists {
		inputSize = e.estimateInputSize(input)
	}
	if toolInfo, exists := toolCallMap[prettifiedName]; exists {
		toolInfo.CallCount++
		if inputSize > toolInfo.MaxInputSize {
			toolInfo.MaxInputSize = inputSize
		}
	} else {
		toolCallMap[prettifiedName] = &ToolCallInfo{
			Name:          prettifiedName,
			CallCount:     1,
			MaxInputSize:  inputSize,
			MaxOutputSize: 0,
			MaxDuration:   0,
		}
	}
	return prettifiedName, true
}

func claudePrettifiedToolName(contentMap map[string]any, nameStr string) string {
	prettifiedName := PrettifyToolName(nameStr)
	if nameStr == "Bash" {
		if commandStr, ok := typeutil.LookupStringPath(contentMap, "input", "command"); ok {
			prettifiedName = "bash_" + ShortenCommand(commandStr)
		}
	}
	return prettifiedName
}

func updateClaudeToolResultOutputSizes(contentMap map[string]any, toolCallMap map[string]*ToolCallInfo) {
	contentStr, ok := typeutil.LookupString(contentMap, "content")
	if !ok {
		return
	}
	outputSize := len(contentStr) / 4
	if _, ok := typeutil.LookupString(contentMap, "tool_use_id"); !ok {
		return
	}
	for _, toolInfo := range toolCallMap {
		if outputSize > toolInfo.MaxOutputSize {
			toolInfo.MaxOutputSize = outputSize
		}
	}
}

// estimateInputSize estimates the input size in tokens from a tool input object
func (e *ClaudeEngine) estimateInputSize(input any) int {
	// Convert input to JSON string to get approximate size
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return 0
	}
	// Estimate token count (rough approximation: 1 token = ~4 characters)
	return len(inputJSON) / 4
}
