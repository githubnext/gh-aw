package workflow

import (
	"encoding/json"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

const (
	// outputSampleMaxLines is the maximum number of lines to include in a tool output preview.
	outputSampleMaxLines = 3
	// outputSampleMaxLineLen is the maximum character length of each line in a tool output preview.
	outputSampleMaxLineLen = 120
)

// truncateOutputSample returns the first outputSampleMaxLines lines of output,
// each truncated to outputSampleMaxLineLen characters.
func truncateOutputSample(output string) string {
	lines := strings.SplitN(output, "\n", outputSampleMaxLines+1)
	if len(lines) > outputSampleMaxLines {
		lines = lines[:outputSampleMaxLines]
	}
	for i, line := range lines {
		if len(line) > outputSampleMaxLineLen {
			runes := []rune(line)
			if len(runes) > outputSampleMaxLineLen {
				lines[i] = string(runes[:outputSampleMaxLineLen]) + "…"
			}
		}
	}
	return strings.Join(lines, "\n")
}

// sanitizeJSONBlock extracts a clean JSON object from a string that may contain
// trailing non-JSON content (e.g. [INFO] log lines appended after the closing brace).
// Returns an empty string if no valid JSON object boundary is found.
func sanitizeJSONBlock(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	open := strings.Index(trimmed, "{")
	if open < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := open; i < len(trimmed); i++ {
		ch := trimmed[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return trimmed[open : i+1]
			}
			if depth < 0 {
				return ""
			}
		}
	}

	return ""
}

func timestampedLogRemainder(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	ts, rest, ok := strings.Cut(trimmed, " ")
	if !ok {
		return "", false
	}
	if !strings.Contains(ts, "T") || !strings.Contains(ts, ":") {
		return "", false
	}
	return rest, true
}

func isTimestampedDebugOrInfoLine(line string) bool {
	rest, ok := timestampedLogRemainder(line)
	if !ok {
		return false
	}
	return strings.HasPrefix(rest, "[DEBUG]") || strings.HasPrefix(rest, "[INFO]")
}

func isTimestampedDebugLine(line string, marker string) bool {
	rest, ok := timestampedLogRemainder(line)
	if !ok {
		return false
	}
	return strings.HasPrefix(rest, marker)
}

var copilotLogsLog = logger.New("workflow:copilot_logs")

// SessionEntry represents a single entry in a Copilot session JSONL file
type SessionEntry struct {
	Type     string          `json:"type"`
	Subtype  string          `json:"subtype,omitempty"`
	Message  *SessionMessage `json:"message,omitempty"`
	Usage    *SessionUsage   `json:"usage,omitempty"`
	NumTurns int             `json:"num_turns,omitempty"`
	RawData  map[string]any  `json:"-"`
}

// SessionMessage represents the message field in session entries
type SessionMessage struct {
	Content []SessionContent `json:"content"`
}

// SessionContent represents content items in messages
type SessionContent struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   string         `json:"content,omitempty"`
}

// SessionUsage represents token usage in a session result entry
type SessionUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// parseSessionJSONL attempts to parse the log content as JSONL session format
// Returns true if successful, false if the format is not recognized
func (e *CopilotEngine) parseSessionJSONL(logContent string, verbose bool) (LogMetrics, bool) {
	var metrics LogMetrics
	var totalTokenUsage int
	toolCallMap := make(map[string]*ToolCallInfo)
	var currentSequence []string
	turns := 0
	assistantMessageCount := 0 // fallback: count assistant messages when num_turns is absent

	lines := strings.Split(logContent, "\n")
	foundSessionEntry := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines and debug log lines
		if trimmedLine == "" || !strings.HasPrefix(trimmedLine, "{") {
			continue
		}

		// Try to parse as session entry
		var entry SessionEntry
		if err := json.Unmarshal([]byte(trimmedLine), &entry); err != nil {
			continue
		}

		foundSessionEntry = true

		// Handle different entry types
		switch entry.Type {
		case "system":
			// System init entry - no action needed for metrics
			if verbose {
				copilotLogsLog.Printf("Found system init entry")
			}

		case "assistant":
			// Each assistant message represents one LLM turn
			assistantMessageCount++

			// Assistant message with potential tool calls
			if entry.Message != nil {
				for _, content := range entry.Message.Content {
					if content.Type == "tool_use" {
						toolName := content.Name

						// Track in sequence
						currentSequence = append(currentSequence, toolName)

						// Calculate input size
						inputSize := 0
						if content.Input != nil {
							inputJSON, _ := json.Marshal(content.Input) //nolint:jsonmarshalignoredeerror // used only for len() size metric; failure yields len(nil)==0 which is acceptable
							inputSize = len(inputJSON)
						}

						// Update or create tool call info
						if toolInfo, exists := toolCallMap[toolName]; exists {
							toolInfo.CallCount++
							if inputSize > toolInfo.MaxInputSize {
								toolInfo.MaxInputSize = inputSize
							}
						} else {
							toolCallMap[toolName] = &ToolCallInfo{
								Name:          toolName,
								CallCount:     1,
								MaxInputSize:  inputSize,
								MaxOutputSize: 0,
							}
						}

						if verbose {
							copilotLogsLog.Printf("Found tool call: %s with input size %d", toolName, inputSize)
						}
					}
				}
			}

		case "user":
			// User message with tool results
			if entry.Message != nil {
				for _, content := range entry.Message.Content {
					if content.Type == "tool_result" && content.ToolUseID != "" {
						// Track output size
						outputSize := len(content.Content)

						// Try to find the tool by matching recent tools in sequence
						// Since we don't have the tool ID mapping, we'll update the most recent matching tool
						for toolName, toolInfo := range toolCallMap {
							if outputSize > toolInfo.MaxOutputSize {
								toolInfo.MaxOutputSize = outputSize
								if verbose {
									copilotLogsLog.Printf("Updated %s MaxOutputSize to %d bytes", toolName, outputSize)
								}
								break // Update first matching tool
							}
						}
					}
				}
			}

		case "result":
			// Result entry with usage statistics
			if entry.Usage != nil {
				totalTokenUsage = entry.Usage.InputTokens + entry.Usage.OutputTokens
				turns = entry.NumTurns

				if verbose {
					copilotLogsLog.Printf("Found result entry: input_tokens=%d, output_tokens=%d, num_turns=%d",
						entry.Usage.InputTokens, entry.Usage.OutputTokens, turns)
				}
			}
		}
	}

	// If turns was not set from num_turns (0 or absent), fall back to counting assistant messages.
	// The Copilot CLI may omit num_turns from the result entry; each assistant message represents
	// one LLM conversation turn.
	if turns == 0 && assistantMessageCount > 0 {
		turns = assistantMessageCount
		copilotLogsLog.Printf("num_turns not available in result entry, using assistant message count as turns: %d", turns)
	}

	// If we found no session entries, return false to indicate fallback needed
	if !foundSessionEntry {
		return metrics, false
	}

	// Save current sequence before finalizing
	if len(currentSequence) > 0 {
		metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
	}

	// Finalize metrics
	copilotLogsLog.Printf("Session JSONL parsing complete: totalTokenUsage=%d, turns=%d, toolCalls=%d",
		totalTokenUsage, turns, len(toolCallMap))

	FinalizeToolMetrics(FinalizeToolMetricsOptions{
		Metrics:         &metrics,
		ToolCallMap:     toolCallMap,
		CurrentSequence: currentSequence,
		Turns:           turns,
		TokenUsage:      totalTokenUsage,
	})

	return metrics, true
}

// ParseLogMetrics implements engine-specific log parsing for Copilot CLI.
//
// Parsing Strategy:
// 1. First attempts to parse as JSONL session format (from ~/.copilot/session-state/*.jsonl)
// 2. Falls back to debug log format if JSONL parsing fails or finds no entries
//
// Token Counting Behavior:
// Copilot CLI makes multiple API calls during a workflow run (one per turn).
// Each API call returns a response with usage statistics including token counts.
// This function accumulates token counts from ALL API responses to get the total
// token usage for the entire workflow run.
//
// Example: If a run has 3 turns with token counts [1000, 1500, 800],
// the total token usage will be 3300 (sum of all turns).
//
// This matches the behavior of the JavaScript parser in parse_copilot_log.cjs.
//
// Wire request block parsing (wireApi=responses format):
// When the Copilot CLI uses the OpenAI Responses API wire format, each turn is
// preceded by a [DEBUG] Wire request: block containing the full conversation
// history, including function_call_output items for completed tool calls.
// These blocks are parsed to extract tool output sizes and a response preview.
func (e *CopilotEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	// Try parsing as JSONL session format first
	if metrics, success := e.parseSessionJSONL(logContent, verbose); success {
		copilotLogsLog.Printf("Successfully parsed session JSONL format")
		return metrics
	}

	// Fall back to debug log format parsing
	copilotLogsLog.Printf("JSONL parsing failed or no entries found, falling back to debug log format")

	var metrics LogMetrics
	var totalTokenUsage int

	lines := strings.Split(logContent, "\n")
	toolCallMap := make(map[string]*ToolCallInfo) // Track tool calls
	var currentSequence []string                  // Track tool sequence
	turns := 0

	// Track multi-line JSON blocks for token extraction
	var inDataBlock bool
	var currentJSONLines []string

	// Track Wire request blocks for tool output extraction
	var inWireBlock bool
	var currentWireLines []string

	// flushDataBlock processes and clears the accumulated data block.
	flushDataBlock := func() {
		if len(currentJSONLines) == 0 {
			return
		}
		jsonStr := strings.Join(currentJSONLines, "\n")
		copilotLogsLog.Printf("Parsing JSON block with %d lines (%d bytes)", len(currentJSONLines), len(jsonStr))
		jsonMetrics := ExtractJSONMetrics(jsonStr, verbose)
		// Accumulate token usage from all responses (not just max)
		// This matches the JavaScript parser behavior in parse_copilot_log.cjs
		if jsonMetrics.TokenUsage > 0 {
			copilotLogsLog.Printf("Extracted %d tokens from JSON block", jsonMetrics.TokenUsage)
			totalTokenUsage += jsonMetrics.TokenUsage
		} else {
			copilotLogsLog.Printf("No tokens extracted from JSON block (possible format issue)")
		}
		if jsonMetrics.EstimatedCost > 0 {
			metrics.EstimatedCost += jsonMetrics.EstimatedCost
		}
		e.extractToolCallSizes(jsonStr, toolCallMap, verbose)
		inDataBlock = false
		currentJSONLines = []string{}
	}

	// flushWireBlock processes and clears the accumulated wire request block.
	flushWireBlock := func() {
		if len(currentWireLines) > 0 {
			wireStr := strings.Join(currentWireLines, "\n")
			e.extractWireRequestOutputs(wireStr, toolCallMap, verbose)
		}
		inWireBlock = false
		currentWireLines = []string{}
	}

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Detect start of a JSON data block from Copilot debug logs.
		// Format: "YYYY-MM-DDTHH:MM:SS.sssZ [DEBUG] data:"
		if isTimestampedDebugLine(line, "[DEBUG] data:") {
			// End any open wire block before starting a data block.
			if inWireBlock {
				flushWireBlock()
			}
			inDataBlock = true
			currentJSONLines = []string{}
			// Each API response data block represents one LLM conversation turn.
			// Copilot CLI debug logs don't have "User:"/"Human:" patterns, so we
			// count turns based on the number of API responses (data blocks).
			turns++
			// Save previous sequence before starting new turn
			if len(currentSequence) > 0 {
				metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
				currentSequence = []string{}
			}
			continue
		}

		// Detect start of a Wire request block (wireApi=responses format).
		// Format: "YYYY-MM-DDTHH:MM:SS.sssZ [DEBUG] Wire request: {"
		if isTimestampedDebugLine(line, "[DEBUG] Wire request:") {
			// End any open data block before starting a wire block.
			if inDataBlock {
				flushDataBlock()
			}
			// End any open wire block before starting a new wire block.
			if inWireBlock {
				flushWireBlock()
			}
			inWireBlock = true
			currentWireLines = []string{}
			// Extract the opening { from the same line (Wire request: {)
			if idx := strings.Index(line, "{"); idx >= 0 {
				currentWireLines = append(currentWireLines, line[idx:])
			}
			continue
		}

		// While in a data block, accumulate lines
		if inDataBlock {
			// Check if this line has a [DEBUG] prefix (indicates it's a log line, not raw JSON)
			hasDebug := strings.Contains(line, "[DEBUG]")

			if hasDebug {
				// Strip the timestamp and [DEBUG] prefix to see what remains
				// Format: "YYYY-MM-DDTHH:MM:SS.sssZ [DEBUG] {json content}"
				_, after, ok := strings.Cut(line, "[DEBUG]")
				if ok {
					cleanLine := strings.TrimSpace(after) // Skip "[DEBUG]"

					// If after stripping, the line starts with JSON characters, it's part of JSON
					// Otherwise, it's a new log entry and we should end the block
					if strings.HasPrefix(cleanLine, "{") || strings.HasPrefix(cleanLine, "}") ||
						strings.HasPrefix(cleanLine, "[") || strings.HasPrefix(cleanLine, "]") ||
						strings.HasPrefix(cleanLine, "\"") {
						// This is JSON content - add it
						currentJSONLines = append(currentJSONLines, cleanLine)
					} else {
						// This is a new log line (not JSON content) - end of JSON block
						flushDataBlock()
					}
				}
			} else {
				// Line has no [DEBUG] prefix — treat as raw JSON content.
				// Note: [INFO] lines (e.g. "--- End of group ---") also land here but
				// are harmless: sanitizeJSONBlock in extractToolCallSizes/ExtractJSONMetrics
				// trims everything after the last closing brace.
				currentJSONLines = append(currentJSONLines, line)
			}
		}

		// While in a wire request block, accumulate raw JSON lines.
		// Wire request JSON is not prefixed per-line; the block ends when any
		// timestamped log line ([DEBUG] or [INFO]) appears.
		if inWireBlock {
			if isTimestampedDebugOrInfoLine(line) {
				flushWireBlock()
				// The block-start checks above already evaluated this line (no match);
				// fall through to the tool-call extraction below.
			} else {
				currentWireLines = append(currentWireLines, line)
			}
		}

		// Extract tool calls and add to sequence and toolCallMap
		// "Executing tool: <name>" lines confirm tool execution and are used to populate
		// both the tool sequence and tool call statistics. This handles the common case where
		// Copilot CLI JSON blocks have empty tool_calls arrays but emit execution log lines.
		if toolName := e.parseCopilotToolCallsWithSequence(line, toolCallMap); toolName != "" {
			currentSequence = append(currentSequence, toolName)
		}
	}

	// Process any remaining blocks at the end of file
	if inDataBlock {
		flushDataBlock()
	}
	if inWireBlock {
		flushWireBlock()
	}

	// Finalize metrics using shared helper
	copilotLogsLog.Printf("Finalized metrics: totalTokenUsage=%d, turns=%d, toolCalls=%d", totalTokenUsage, turns, len(toolCallMap))
	FinalizeToolMetrics(FinalizeToolMetricsOptions{
		Metrics:         &metrics,
		ToolCallMap:     toolCallMap,
		CurrentSequence: currentSequence,
		Turns:           turns,
		TokenUsage:      totalTokenUsage,
	})

	return metrics
}

// extractToolCallSizes extracts tool call input sizes from Copilot JSON responses.
// It sanitizes the JSON block first to handle trailing non-JSON log lines (e.g.
// [INFO] lines that are appended after the closing brace in the wireApi=responses format).
func (e *CopilotEngine) extractToolCallSizes(jsonStr string, toolCallMap map[string]*ToolCallInfo, verbose bool) {
	clean := sanitizeJSONBlock(jsonStr)
	if clean == "" {
		if verbose {
			copilotLogsLog.Printf("No valid JSON object found for tool size extraction")
		}
		return
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(clean), &data); err != nil {
		if verbose {
			copilotLogsLog.Printf("Failed to parse JSON for tool size extraction: %v", err)
		}
		return
	}

	// Look for tool_calls in the choices array (Copilot/OpenAI format)
	if choices, ok := data["choices"].([]any); ok {
		for _, choice := range choices {
			if choiceMap, ok := choice.(map[string]any); ok {
				if message, ok := choiceMap["message"].(map[string]any); ok {
					if toolCalls, ok := message["tool_calls"].([]any); ok {
						e.processToolCalls(toolCalls, toolCallMap, verbose)
					}
				}
			}
		}
	}

	// Also check for tool_calls directly in the message (alternative format)
	if message, ok := data["message"].(map[string]any); ok {
		if toolCalls, ok := message["tool_calls"].([]any); ok {
			e.processToolCalls(toolCalls, toolCallMap, verbose)
		}
	}
}

// processToolCalls processes tool_calls array and updates tool call map with sizes
func (e *CopilotEngine) processToolCalls(toolCalls []any, toolCallMap map[string]*ToolCallInfo, verbose bool) {
	for _, toolCall := range toolCalls {
		if tcMap, ok := toolCall.(map[string]any); ok {
			// Extract function information
			if function, ok := tcMap["function"].(map[string]any); ok {
				if toolName, ok := function["name"].(string); ok {
					// Calculate input size from arguments (if present)
					inputSize := 0
					if arguments, ok := function["arguments"].(string); ok {
						inputSize = len(arguments)
					}

					// Initialize or update tool call info
					if toolInfo, exists := toolCallMap[toolName]; exists {
						// If a stub entry was first created from function_call_output in a
						// Wire request, it already carries evidence of one invocation.
						// Avoid double-counting when the corresponding tool_call arrives later.
						if !isWireOutputStub(toolInfo) {
							toolInfo.CallCount++
						}
						// Update max input size if this call is larger
						if inputSize > toolInfo.MaxInputSize {
							toolInfo.MaxInputSize = inputSize
							if verbose {
								copilotLogsLog.Printf("Updated %s MaxInputSize to %d bytes", toolName, inputSize)
							}
						}
					} else {
						toolCallMap[toolName] = &ToolCallInfo{
							Name:         toolName,
							CallCount:    1,
							MaxInputSize: inputSize,
						}
						if verbose {
							copilotLogsLog.Printf("Created tool info for %s with MaxInputSize=%d bytes", toolName, inputSize)
						}
					}
				}
			}
		}
	}
}

// isWireOutputStub returns true when a ToolCallInfo entry was inferred from a
// function_call_output item before we observed the corresponding tool_call input.
// In this state, CallCount is already seeded to 1 based on output evidence.
func isWireOutputStub(toolInfo *ToolCallInfo) bool {
	return toolInfo.CallCount == 1 && toolInfo.MaxInputSize == 0 && toolInfo.MaxOutputSize > 0
}

// extractWireRequestOutputs parses a [DEBUG] Wire request: JSON block and updates
// MaxOutputSize and OutputSample for each tool that has a function_call_output entry.
//
// The wireApi=responses format includes the full conversation history in each request's
// "input" array. Completed tool calls appear as consecutive function_call / function_call_output
// pairs, letting us extract both the tool name (from function_call.name) and the tool
// response (from function_call_output.output) in a single pass.
func (e *CopilotEngine) extractWireRequestOutputs(jsonStr string, toolCallMap map[string]*ToolCallInfo, verbose bool) {
	clean := sanitizeJSONBlock(jsonStr)
	if clean == "" {
		return
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(clean), &data); err != nil {
		if verbose {
			copilotLogsLog.Printf("Failed to parse Wire request JSON: %v", err)
		}
		return
	}

	inputs, ok := data["input"].([]any)
	if !ok {
		return
	}

	// Build a local call_id → tool name map from function_call items in this request.
	// The Wire request contains the full conversation history, so all historical
	// function_call / function_call_output pairs are present in a single block.
	callIDToTool := make(map[string]string, len(inputs))
	for _, item := range inputs {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := itemMap["type"].(string); typ == "function_call" {
			callID, _ := itemMap["call_id"].(string)
			name, _ := itemMap["name"].(string)
			if callID != "" && name != "" {
				callIDToTool[callID] = name
			}
		}
	}

	// Extract output sizes and samples from function_call_output items.
	for _, item := range inputs {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := itemMap["type"].(string); typ != "function_call_output" {
			continue
		}

		callID, _ := itemMap["call_id"].(string)
		output, _ := itemMap["output"].(string)
		if callID == "" || output == "" {
			continue
		}

		toolName := callIDToTool[callID]
		if toolName == "" {
			continue
		}

		outputSize := len(output)
		if toolInfo, exists := toolCallMap[toolName]; exists {
			if outputSize > toolInfo.MaxOutputSize {
				toolInfo.MaxOutputSize = outputSize
				toolInfo.OutputSample = truncateOutputSample(output)
				if verbose {
					copilotLogsLog.Printf("Updated %s MaxOutputSize to %d bytes with sample", toolName, outputSize)
				}
			}
		} else {
			// Tool entry not yet created by extractToolCallSizes — create a stub so the
			// output sample is not lost (e.g. when wire-request ordering differs from data blocks).
			toolCallMap[toolName] = &ToolCallInfo{
				Name:          toolName,
				CallCount:     1,
				MaxOutputSize: outputSize,
				OutputSample:  truncateOutputSample(output),
			}
			if verbose {
				copilotLogsLog.Printf("Created stub entry for %s from wire request output (%d bytes)", toolName, outputSize)
			}
		}
	}
}

// parseCopilotToolCallsWithSequence extracts tool call information from Copilot CLI log lines and returns tool name.
// It also updates toolCallMap with the tool execution count for statistics tracking.
func (e *CopilotEngine) parseCopilotToolCallsWithSequence(line string, toolCallMap map[string]*ToolCallInfo) string {
	// Look for "Executing tool:" pattern in Copilot logs
	if strings.Contains(line, "Executing tool:") {
		// Extract tool name from "Executing tool: <name>" format
		parts := strings.Split(line, "Executing tool:")
		if len(parts) > 1 {
			toolName := strings.TrimSpace(parts[1])
			if toolName == "" {
				return ""
			}
			// Update toolCallMap: this captures tool calls from execution log lines.
			// This is the primary source of tool call data in the Copilot CLI debug log
			// format, since JSON response blocks often have empty tool_calls arrays.
			if toolInfo, exists := toolCallMap[toolName]; exists {
				toolInfo.CallCount++
			} else {
				toolCallMap[toolName] = &ToolCallInfo{
					Name:      toolName,
					CallCount: 1,
				}
			}
			return toolName
		}
	}

	return ""
}

// GetLogParserScriptId returns the JavaScript script name for parsing Copilot logs
func (e *CopilotEngine) GetLogParserScriptId() string {
	return "parse_copilot_log"
}

// GetErrorDetectionScriptId returns the JavaScript script name for detecting agent errors
// from the agent stdio log. The script runs on the host runner after the AWF container exits,
// allowing it to write GITHUB_OUTPUT values that are not accessible inside the container.
func (e *CopilotEngine) GetErrorDetectionScriptId() string {
	return "detect_agent_errors"
}

// GetLogFileForParsing returns the log directory for Copilot CLI logs
// Copilot writes detailed debug logs to /tmp/gh-aw/sandbox/agent/logs/
func (e *CopilotEngine) GetLogFileForParsing() string {
	return constants.TmpSandboxAgentLogsDir
}
