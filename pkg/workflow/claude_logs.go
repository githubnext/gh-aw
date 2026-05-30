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

	logEntries := e.parseClaudeLogEntries(logContent, verbose)
	if len(logEntries) == 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "No valid JSON entries found in Claude log\n")
		}
		return metrics
	}

	// Look for the result entry with type: "result"
	toolCallMap := make(map[string]*ToolCallInfo) // Track tool calls across entries
	var currentSequence []string                  // Track tool sequence within current context

	for _, entry := range logEntries {
		if e.updateMetricsFromResultEntry(entry, &metrics, verbose) {
			break
		}
		e.parseAssistantEntryToolCalls(entry, toolCallMap, &currentSequence)
		e.parseUserEntryToolResults(entry, toolCallMap)
	}

	// Finalize tool calls and sequences using shared helper
	FinalizeToolCallsAndSequence(&metrics, toolCallMap, currentSequence)

	claudeLogsLog.Printf("Parsed %d log entries: tokens=%d, cost=$%.4f, turns=%d, tool_types=%d",
		len(logEntries), metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns, len(metrics.ToolCalls))

	if verbose && len(metrics.ToolSequences) > 0 {
		totalTools := 0
		for _, seq := range metrics.ToolSequences {
			totalTools += len(seq)
		}
		fmt.Fprintf(os.Stderr, "Claude parser extracted %d tool sequences with %d total tool calls\n",
			len(metrics.ToolSequences), totalTools)
	}

	return metrics
}

func (e *ClaudeEngine) parseClaudeLogEntries(logContent string, verbose bool) []map[string]any {
	var logEntries []map[string]any
	if err := json.Unmarshal([]byte(logContent), &logEntries); err == nil {
		return logEntries
	}

	claudeLogsLog.Print("JSON array parse failed, trying JSONL format")
	if verbose {
		fmt.Fprintf(os.Stderr, "Failed to parse Claude log as JSON array, trying JSONL format\n")
	}

	return e.parseClaudeMixedFormatLog(logContent, verbose)
}

func (e *ClaudeEngine) parseClaudeMixedFormatLog(logContent string, verbose bool) []map[string]any {
	var logEntries []map[string]any
	lines := strings.Split(logContent, "\n")
	for i := 0; i < len(lines); i++ {
		trimmedLine := strings.TrimSpace(lines[i])
		if trimmedLine == "" {
			continue
		}
		if strings.HasPrefix(trimmedLine, "[") {
			next, consumed := e.parseClaudeJSONArraysFromLine(lines, i, &logEntries)
			i = next
			if consumed {
				continue
			}
		}
		e.appendClaudeJSONLObjectLine(trimmedLine, &logEntries, verbose)
	}
	if verbose && len(logEntries) > 0 {
		fmt.Fprintf(os.Stderr, "Extracted %d JSON entries from mixed format Claude log\n", len(logEntries))
	}
	return logEntries
}

func (e *ClaudeEngine) parseClaudeJSONArraysFromLine(lines []string, idx int, logEntries *[]map[string]any) (int, bool) {
	buf, nextIdx := e.buildClaudeJSONArrayBuffer(lines, idx)
	if e.tryAppendClaudeJSONArray(buf, logEntries) {
		return nextIdx, true
	}
	return nextIdx, e.tryAppendEmbeddedClaudeJSONArray(buf, logEntries)
}

func (e *ClaudeEngine) buildClaudeJSONArrayBuffer(lines []string, idx int) (string, int) {
	buf := strings.TrimSpace(lines[idx])
	if strings.Contains(buf, "]") {
		return buf, idx
	}
	var sb strings.Builder
	for j := idx + 1; j < len(lines); j++ {
		sb.WriteString("\n" + lines[j])
		if strings.Contains(lines[j], "]") {
			return buf + sb.String(), j
		}
	}
	return buf + sb.String(), idx
}

func (e *ClaudeEngine) tryAppendClaudeJSONArray(buf string, logEntries *[]map[string]any) bool {
	var arr []map[string]any
	if err := json.Unmarshal([]byte(buf), &arr); err != nil {
		return false
	}
	*logEntries = append(*logEntries, arr...)
	return true
}

func (e *ClaudeEngine) tryAppendEmbeddedClaudeJSONArray(buf string, logEntries *[]map[string]any) bool {
	openIdx := strings.Index(buf, "[")
	closeIdx := strings.LastIndex(buf, "]")
	if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx {
		return false
	}
	return e.tryAppendClaudeJSONArray(buf[openIdx:closeIdx+1], logEntries)
}

func (e *ClaudeEngine) appendClaudeJSONLObjectLine(trimmedLine string, logEntries *[]map[string]any, verbose bool) {
	if !strings.HasPrefix(trimmedLine, "{") {
		return
	}
	var jsonEntry map[string]any
	if err := json.Unmarshal([]byte(trimmedLine), &jsonEntry); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Skipping invalid JSON line: %s\n", trimmedLine)
		}
		return
	}
	*logEntries = append(*logEntries, jsonEntry)
}

func (e *ClaudeEngine) updateMetricsFromResultEntry(entry map[string]any, metrics *LogMetrics, verbose bool) bool {
	typeStr, ok := typeutil.LookupString(entry, "type")
	if !ok || typeStr != "result" {
		return false
	}
	e.applyClaudeResultEntryMetrics(entry, metrics)
	if verbose {
		fmt.Fprintf(os.Stderr, "Extracted from Claude result payload: tokens=%d, cost=%.4f, turns=%d\n",
			metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns)
	}
	return true
}

func (e *ClaudeEngine) applyClaudeResultEntryMetrics(entry map[string]any, metrics *LogMetrics) {
	if totalCost, exists := entry["total_cost_usd"]; exists {
		if cost := typeutil.ConvertToFloat(totalCost); cost > 0 {
			metrics.EstimatedCost = cost
		}
	}
	if usage, exists := entry["usage"]; exists {
		if usageMap, ok := usage.(map[string]any); ok {
			totalTokens := typeutil.ConvertToInt(usageMap["input_tokens"]) +
				typeutil.ConvertToInt(usageMap["output_tokens"]) +
				typeutil.ConvertToInt(usageMap["cache_creation_input_tokens"]) +
				typeutil.ConvertToInt(usageMap["cache_read_input_tokens"])
			if totalTokens > 0 {
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

func (e *ClaudeEngine) parseAssistantEntryToolCalls(entry map[string]any, toolCallMap map[string]*ToolCallInfo, currentSequence *[]string) {
	typeStr, ok := typeutil.LookupString(entry, "type")
	if !ok || typeStr != "assistant" {
		return
	}
	contentArray, ok := e.lookupClaudeEntryContentArray(entry)
	if !ok {
		return
	}
	sequenceInMessage := e.parseToolCallsWithSequence(contentArray, toolCallMap)
	if len(sequenceInMessage) > 0 {
		*currentSequence = append(*currentSequence, sequenceInMessage...)
	}
}

func (e *ClaudeEngine) parseUserEntryToolResults(entry map[string]any, toolCallMap map[string]*ToolCallInfo) {
	typeStr, ok := typeutil.LookupString(entry, "type")
	if !ok || typeStr != "user" {
		return
	}
	contentArray, ok := e.lookupClaudeEntryContentArray(entry)
	if !ok {
		return
	}
	// Sequence return value intentionally discarded; only toolCallMap is needed here.
	e.parseToolCallsWithSequence(contentArray, toolCallMap)
}

func (e *ClaudeEngine) lookupClaudeEntryContentArray(entry map[string]any) ([]any, bool) {
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
			prettifiedName, inputSize, ok := e.extractToolUseInfo(contentMap)
			if !ok {
				continue
			}
			sequence = append(sequence, prettifiedName)
			e.updateToolCallMap(toolCallMap, prettifiedName, inputSize)
		case "tool_result":
			e.updateToolResultsOutputSize(contentMap, toolCallMap)
		}
	}

	return sequence
}

func (e *ClaudeEngine) extractToolUseInfo(contentMap map[string]any) (string, int, bool) {
	nameStr, ok := typeutil.LookupString(contentMap, "name")
	if !ok {
		return "", 0, false
	}
	prettifiedName := PrettifyToolName(nameStr)
	if nameStr == "Bash" {
		prettifiedName = e.resolveClaudeBashToolName(contentMap, prettifiedName)
	}
	return prettifiedName, e.lookupToolInputSize(contentMap), true
}

func (e *ClaudeEngine) resolveClaudeBashToolName(contentMap map[string]any, fallback string) string {
	commandStr, ok := typeutil.LookupStringPath(contentMap, "input", "command")
	if !ok {
		return fallback
	}
	return "bash_" + ShortenCommand(commandStr)
}

func (e *ClaudeEngine) lookupToolInputSize(contentMap map[string]any) int {
	input, exists := contentMap["input"]
	if !exists {
		return 0
	}
	return e.estimateInputSize(input)
}

func (e *ClaudeEngine) updateToolCallMap(toolCallMap map[string]*ToolCallInfo, prettifiedName string, inputSize int) {
	if toolInfo, exists := toolCallMap[prettifiedName]; exists {
		toolInfo.CallCount++
		if inputSize > toolInfo.MaxInputSize {
			toolInfo.MaxInputSize = inputSize
		}
		return
	}
	toolCallMap[prettifiedName] = &ToolCallInfo{
		Name:          prettifiedName,
		CallCount:     1,
		MaxInputSize:  inputSize,
		MaxOutputSize: 0, // Will be updated when we find tool results
		MaxDuration:   0, // Will be updated when we find execution timing
	}
}

func (e *ClaudeEngine) updateToolResultsOutputSize(contentMap map[string]any, toolCallMap map[string]*ToolCallInfo) {
	contentStr, ok := typeutil.LookupString(contentMap, "content")
	if !ok {
		return
	}
	if _, ok := typeutil.LookupString(contentMap, "tool_use_id"); !ok {
		return
	}
	outputSize := len(contentStr) / 4
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
