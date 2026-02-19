package workflow

import (
	"encoding/json"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var geminiLogsLog = logger.New("workflow:gemini_logs")

// GeminiResponse represents the JSON structure returned by Gemini CLI
type GeminiResponse struct {
	Response string                 `json:"response"`
	Stats    map[string]interface{} `json:"stats"`
}

// ParseLogMetrics parses Gemini CLI log output and extracts metrics.
// Gemini CLI outputs a single JSON response when using --output-format json.
// We parse the last valid JSON line (most complete response) and aggregate stats.
func (e *GeminiEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	geminiLogsLog.Printf("Parsing Gemini log metrics: log_size=%d bytes, verbose=%v", len(logContent), verbose)

	metrics := LogMetrics{
		Turns:      0,
		TokenUsage: 0,
		ToolCalls:  []ToolCallInfo{},
	}

	// Aggregate tool calls in a map to deduplicate across multiple JSON lines
	toolCallCounts := make(map[string]int)

	// Try to parse the JSON response from Gemini
	lines := strings.Split(logContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse as JSON
		var response GeminiResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			continue
		}

		// Successfully parsed JSON response - use the last valid response for turn count
		if response.Response != "" {
			metrics.Turns = 1 // At least one turn if we got a response
		}

		// Extract token usage from stats if available
		if response.Stats != nil {
			if models, ok := response.Stats["models"].(map[string]interface{}); ok {
				for _, modelStats := range models {
					if stats, ok := modelStats.(map[string]interface{}); ok {
						if inputTokens, ok := stats["input_tokens"].(float64); ok {
							metrics.TokenUsage += int(inputTokens)
						}
						if outputTokens, ok := stats["output_tokens"].(float64); ok {
							metrics.TokenUsage += int(outputTokens)
						}
					}
				}
			}

			// Aggregate tool calls using a map to avoid duplicates
			if tools, ok := response.Stats["tools"].(map[string]interface{}); ok {
				for toolName := range tools {
					toolCallCounts[toolName]++
				}
			}
		}

		geminiLogsLog.Printf("Parsed JSON response: response_len=%d, stats_present=%v", len(response.Response), response.Stats != nil)
	}

	// Convert tool call map to slice
	for toolName, count := range toolCallCounts {
		metrics.ToolCalls = append(metrics.ToolCalls, ToolCallInfo{
			Name:      toolName,
			CallCount: count,
		})
	}

	geminiLogsLog.Printf("Parsed metrics: turns=%d, token_usage=%d, tool_calls=%d",
		metrics.Turns, metrics.TokenUsage, len(metrics.ToolCalls))

	return metrics
}

// GetLogParserScriptId returns the script ID for parsing Gemini logs
func (e *GeminiEngine) GetLogParserScriptId() string {
	return "parse_gemini_log"
}

// GetLogFileForParsing returns the log file path for parsing
func (e *GeminiEngine) GetLogFileForParsing() string {
	return "/tmp/gh-aw/agent-stdio.log"
}

// GetDefaultDetectionModel returns the default model for threat detection
// Gemini does not specify a default detection model yet
func (e *GeminiEngine) GetDefaultDetectionModel() string {
	return ""
}
