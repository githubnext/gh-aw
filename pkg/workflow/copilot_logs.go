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

type sessionJSONLParseState struct {
	metrics               LogMetrics
	totalTokenUsage       int
	toolCallMap           map[string]*ToolCallInfo
	currentSequence       []string
	turns                 int
	assistantMessageCount int
	foundSessionEntry     bool
	verbose               bool
}

// parseSessionJSONL attempts to parse the log content as JSONL session format
// Returns true if successful, false if the format is not recognized
func (e *CopilotEngine) parseSessionJSONL(logContent string, verbose bool) (LogMetrics, bool) {
	state := sessionJSONLParseState{
		toolCallMap: make(map[string]*ToolCallInfo),
		verbose:     verbose,
	}
	for _, line := range strings.Split(logContent, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" || !strings.HasPrefix(trimmedLine, "{") {
			continue
		}
		var entry SessionEntry
		if err := json.Unmarshal([]byte(trimmedLine), &entry); err != nil {
			continue
		}
		state.foundSessionEntry = true
		state.processSessionEntry(entry)
	}
	return state.finalize()
}

func (s *sessionJSONLParseState) processSessionEntry(entry SessionEntry) {
	switch entry.Type {
	case "system":
		if s.verbose {
			copilotLogsLog.Printf("Found system init entry")
		}
	case "assistant":
		s.assistantMessageCount++
		s.processAssistantSessionMessage(entry.Message)
	case "user":
		s.processUserSessionMessage(entry.Message)
	case "result":
		s.processSessionResult(entry)
	}
}

func (s *sessionJSONLParseState) processAssistantSessionMessage(message *SessionMessage) {
	if message == nil {
		return
	}
	for _, content := range message.Content {
		if content.Type != "tool_use" {
			continue
		}
		toolName := content.Name
		s.currentSequence = append(s.currentSequence, toolName)
		inputSize := 0
		if content.Input != nil {
			inputJSON, _ := json.Marshal(content.Input) //nolint:jsonmarshalignoredeerror // used only for len() size metric; failure yields len(nil)==0 which is acceptable
			inputSize = len(inputJSON)
		}
		upsertToolInputSize(s.toolCallMap, toolName, inputSize)
		if s.verbose {
			copilotLogsLog.Printf("Found tool call: %s with input size %d", toolName, inputSize)
		}
	}
}

func upsertToolInputSize(toolCallMap map[string]*ToolCallInfo, toolName string, inputSize int) {
	if toolInfo, exists := toolCallMap[toolName]; exists {
		toolInfo.CallCount++
		if inputSize > toolInfo.MaxInputSize {
			toolInfo.MaxInputSize = inputSize
		}
		return
	}
	toolCallMap[toolName] = &ToolCallInfo{
		Name:          toolName,
		CallCount:     1,
		MaxInputSize:  inputSize,
		MaxOutputSize: 0,
	}
}

func (s *sessionJSONLParseState) processUserSessionMessage(message *SessionMessage) {
	if message == nil {
		return
	}
	for _, content := range message.Content {
		if content.Type == "tool_result" && content.ToolUseID != "" {
			s.updateFirstToolOutputSize(len(content.Content))
		}
	}
}

func (s *sessionJSONLParseState) updateFirstToolOutputSize(outputSize int) {
	for toolName, toolInfo := range s.toolCallMap {
		if outputSize > toolInfo.MaxOutputSize {
			toolInfo.MaxOutputSize = outputSize
			if s.verbose {
				copilotLogsLog.Printf("Updated %s MaxOutputSize to %d bytes", toolName, outputSize)
			}
			break
		}
	}
}

func (s *sessionJSONLParseState) processSessionResult(entry SessionEntry) {
	if entry.Usage == nil {
		return
	}
	s.totalTokenUsage = entry.Usage.InputTokens + entry.Usage.OutputTokens
	s.turns = entry.NumTurns
	if s.verbose {
		copilotLogsLog.Printf("Found result entry: input_tokens=%d, output_tokens=%d, num_turns=%d",
			entry.Usage.InputTokens, entry.Usage.OutputTokens, s.turns)
	}
}

func (s *sessionJSONLParseState) finalize() (LogMetrics, bool) {
	if s.turns == 0 && s.assistantMessageCount > 0 {
		s.turns = s.assistantMessageCount
		copilotLogsLog.Printf("num_turns not available in result entry, using assistant message count as turns: %d", s.turns)
	}
	if !s.foundSessionEntry {
		return s.metrics, false
	}
	if len(s.currentSequence) > 0 {
		s.metrics.ToolSequences = append(s.metrics.ToolSequences, s.currentSequence)
	}
	copilotLogsLog.Printf("Session JSONL parsing complete: totalTokenUsage=%d, turns=%d, toolCalls=%d",
		s.totalTokenUsage, s.turns, len(s.toolCallMap))
	FinalizeToolMetrics(FinalizeToolMetricsOptions{
		Metrics:         &s.metrics,
		ToolCallMap:     s.toolCallMap,
		CurrentSequence: s.currentSequence,
		Turns:           s.turns,
		TokenUsage:      s.totalTokenUsage,
	})
	return s.metrics, true
}

type copilotDebugLogParseState struct {
	engine           *CopilotEngine
	metrics          LogMetrics
	totalTokenUsage  int
	toolCallMap      map[string]*ToolCallInfo
	currentSequence  []string
	turns            int
	inDataBlock      bool
	currentJSONLines []string
	inWireBlock      bool
	currentWireLines []string
	verbose          bool
}

// ParseLogMetrics implements engine-specific log parsing for Copilot CLI.
//
// Parsing Strategy:
// 1. First attempts to parse as JSONL session format (from ~/.copilot/session-state/*.jsonl)
// 2. Falls back to debug log format if JSONL parsing fails or finds no entries
func (e *CopilotEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	if metrics, success := e.parseSessionJSONL(logContent, verbose); success {
		copilotLogsLog.Printf("Successfully parsed session JSONL format")
		return metrics
	}
	copilotLogsLog.Printf("JSONL parsing failed or no entries found, falling back to debug log format")

	state := copilotDebugLogParseState{
		engine:      e,
		toolCallMap: make(map[string]*ToolCallInfo),
		verbose:     verbose,
	}
	for _, line := range strings.Split(logContent, "\n") {
		state.processDebugLogLine(line)
	}
	return state.finalize()
}

func (s *copilotDebugLogParseState) processDebugLogLine(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}
	if s.startDataBlock(line) || s.startWireBlock(line) {
		return
	}
	if s.inDataBlock {
		s.accumulateDataBlockLine(line)
	}
	if s.inWireBlock {
		s.accumulateWireBlockLine(line)
	}
	if toolName := s.engine.parseCopilotToolCallsWithSequence(line, s.toolCallMap); toolName != "" {
		s.currentSequence = append(s.currentSequence, toolName)
	}
}

func (s *copilotDebugLogParseState) startDataBlock(line string) bool {
	if !isTimestampedDebugLine(line, "[DEBUG] data:") {
		return false
	}
	if s.inWireBlock {
		s.flushWireBlock()
	}
	s.inDataBlock = true
	s.currentJSONLines = []string{}
	s.turns++
	if len(s.currentSequence) > 0 {
		s.metrics.ToolSequences = append(s.metrics.ToolSequences, s.currentSequence)
		s.currentSequence = []string{}
	}
	return true
}

func (s *copilotDebugLogParseState) startWireBlock(line string) bool {
	if !isTimestampedDebugLine(line, "[DEBUG] Wire request:") {
		return false
	}
	if s.inDataBlock {
		s.flushDataBlock()
	}
	if s.inWireBlock {
		s.flushWireBlock()
	}
	s.inWireBlock = true
	s.currentWireLines = []string{}
	if idx := strings.Index(line, "{"); idx >= 0 {
		s.currentWireLines = append(s.currentWireLines, line[idx:])
	}
	return true
}

func (s *copilotDebugLogParseState) accumulateDataBlockLine(line string) {
	if !strings.Contains(line, "[DEBUG]") {
		s.currentJSONLines = append(s.currentJSONLines, line)
		return
	}
	_, after, ok := strings.Cut(line, "[DEBUG]")
	if !ok {
		return
	}
	cleanLine := strings.TrimSpace(after)
	if startsJSONLine(cleanLine) {
		s.currentJSONLines = append(s.currentJSONLines, cleanLine)
		return
	}
	s.flushDataBlock()
}

func startsJSONLine(line string) bool {
	return strings.HasPrefix(line, "{") || strings.HasPrefix(line, "}") ||
		strings.HasPrefix(line, "[") || strings.HasPrefix(line, "]") || strings.HasPrefix(line, "\"")
}

func (s *copilotDebugLogParseState) accumulateWireBlockLine(line string) {
	if isTimestampedDebugOrInfoLine(line) {
		s.flushWireBlock()
		return
	}
	s.currentWireLines = append(s.currentWireLines, line)
}

func (s *copilotDebugLogParseState) flushDataBlock() {
	if len(s.currentJSONLines) == 0 {
		return
	}
	jsonStr := strings.Join(s.currentJSONLines, "\n")
	copilotLogsLog.Printf("Parsing JSON block with %d lines (%d bytes)", len(s.currentJSONLines), len(jsonStr))
	jsonMetrics := ExtractJSONMetrics(jsonStr, s.verbose)
	if jsonMetrics.TokenUsage > 0 {
		copilotLogsLog.Printf("Extracted %d tokens from JSON block", jsonMetrics.TokenUsage)
		s.totalTokenUsage += jsonMetrics.TokenUsage
	} else {
		copilotLogsLog.Printf("No tokens extracted from JSON block (possible format issue)")
	}
	if jsonMetrics.EstimatedCost > 0 {
		s.metrics.EstimatedCost += jsonMetrics.EstimatedCost
	}
	s.engine.extractToolCallSizes(jsonStr, s.toolCallMap, s.verbose)
	s.inDataBlock = false
	s.currentJSONLines = []string{}
}

func (s *copilotDebugLogParseState) flushWireBlock() {
	if len(s.currentWireLines) > 0 {
		wireStr := strings.Join(s.currentWireLines, "\n")
		s.engine.extractWireRequestOutputs(wireStr, s.toolCallMap, s.verbose)
	}
	s.inWireBlock = false
	s.currentWireLines = []string{}
}

func (s *copilotDebugLogParseState) finalize() LogMetrics {
	if s.inDataBlock {
		s.flushDataBlock()
	}
	if s.inWireBlock {
		s.flushWireBlock()
	}
	copilotLogsLog.Printf("Finalized metrics: totalTokenUsage=%d, turns=%d, toolCalls=%d", s.totalTokenUsage, s.turns, len(s.toolCallMap))
	FinalizeToolMetrics(FinalizeToolMetricsOptions{
		Metrics:         &s.metrics,
		ToolCallMap:     s.toolCallMap,
		CurrentSequence: s.currentSequence,
		Turns:           s.turns,
		TokenUsage:      s.totalTokenUsage,
	})
	return s.metrics
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
func (e *CopilotEngine) extractWireRequestOutputs(jsonStr string, toolCallMap map[string]*ToolCallInfo, verbose bool) {
	inputs, ok := parseWireRequestInputs(jsonStr, verbose)
	if !ok {
		return
	}
	callIDToTool := wireRequestCallIDToTool(inputs)
	for _, item := range inputs {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		toolName, output, ok := wireRequestOutput(itemMap, callIDToTool)
		if ok {
			updateWireRequestToolOutput(toolCallMap, toolName, output, verbose)
		}
	}
}

func parseWireRequestInputs(jsonStr string, verbose bool) ([]any, bool) {
	clean := sanitizeJSONBlock(jsonStr)
	if clean == "" {
		return nil, false
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(clean), &data); err != nil {
		if verbose {
			copilotLogsLog.Printf("Failed to parse Wire request JSON: %v", err)
		}
		return nil, false
	}
	inputs, ok := data["input"].([]any)
	return inputs, ok
}

func wireRequestCallIDToTool(inputs []any) map[string]string {
	callIDToTool := make(map[string]string, len(inputs))
	for _, item := range inputs {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := itemMap["type"].(string); typ != "function_call" {
			continue
		}
		callID, _ := itemMap["call_id"].(string)
		name, _ := itemMap["name"].(string)
		if callID != "" && name != "" {
			callIDToTool[callID] = name
		}
	}
	return callIDToTool
}

func wireRequestOutput(itemMap map[string]any, callIDToTool map[string]string) (string, string, bool) {
	if typ, _ := itemMap["type"].(string); typ != "function_call_output" {
		return "", "", false
	}
	callID, _ := itemMap["call_id"].(string)
	output, _ := itemMap["output"].(string)
	if callID == "" || output == "" {
		return "", "", false
	}
	toolName := callIDToTool[callID]
	return toolName, output, toolName != ""
}

func updateWireRequestToolOutput(toolCallMap map[string]*ToolCallInfo, toolName, output string, verbose bool) {
	outputSize := len(output)
	if toolInfo, exists := toolCallMap[toolName]; exists {
		if outputSize > toolInfo.MaxOutputSize {
			toolInfo.MaxOutputSize = outputSize
			toolInfo.OutputSample = truncateOutputSample(output)
			if verbose {
				copilotLogsLog.Printf("Updated %s MaxOutputSize to %d bytes with sample", toolName, outputSize)
			}
		}
		return
	}
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
