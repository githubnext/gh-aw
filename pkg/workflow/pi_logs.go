package workflow

import (
	"encoding/json"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var piLogsLog = logger.New("workflow:pi_logs")

// piLogEvent represents a single streaming JSONL event emitted by the Pi CLI.
// Pi emits one JSON object per line during execution.
type piLogEvent struct {
	// Type identifies the event: "init", "assistant", "tool_use", "tool_result", "result"
	Type string `json:"type"`

	// init event fields
	Model     string `json:"model,omitempty"`
	SessionID string `json:"session_id,omitempty"`

	// assistant / message event fields
	Content string `json:"content,omitempty"`
	Delta   bool   `json:"delta,omitempty"`

	// tool_use event fields
	ToolName   string `json:"tool_name,omitempty"`
	ToolID     string `json:"tool_id,omitempty"`
	Parameters any    `json:"parameters,omitempty"`

	// tool_result event fields
	Output string `json:"output,omitempty"`
	Status string `json:"status,omitempty"`

	// result (final stats) event fields — token usage, duration, turn count
	Stats map[string]any `json:"stats,omitempty"`
}

// ParseLogMetrics parses Pi streaming JSONL log output and extracts metrics.
// Pi emits one JSON object per line (JSONL) with typed events. The final "result"
// event carries aggregate token-usage statistics.
func (e *PiEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	piLogsLog.Printf("Parsing Pi log metrics: log_size=%d bytes, verbose=%v", len(logContent), verbose)

	metrics := LogMetrics{
		Turns:      0,
		TokenUsage: 0,
		ToolCalls:  []ToolCallInfo{},
	}

	toolCallCounts := make(map[string]int)
	turns, tokenUsage := parsePiLogEvents(logContent, toolCallCounts)
	toolCallMap := piToolCallMap(toolCallCounts)

	FinalizeToolMetrics(FinalizeToolMetricsOptions{
		Metrics:     &metrics,
		ToolCallMap: toolCallMap,
		Turns:       turns,
		TokenUsage:  tokenUsage,
	})

	piLogsLog.Printf("Parsed Pi metrics: turns=%d, token_usage=%d, tool_calls=%d",
		metrics.Turns, metrics.TokenUsage, len(metrics.ToolCalls))
	return metrics
}

func parsePiLogEvents(logContent string, toolCallCounts map[string]int) (int, int) {
	var turns int
	var tokenUsage int
	for line := range strings.SplitSeq(logContent, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var event piLogEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		turnDelta, tokenDelta := applyPiLogEvent(event, toolCallCounts)
		turns += turnDelta
		tokenUsage += tokenDelta
	}
	return turns, tokenUsage
}

func applyPiLogEvent(event piLogEvent, toolCallCounts map[string]int) (int, int) {
	switch event.Type {
	case "assistant":
		if !event.Delta && strings.TrimSpace(event.Content) != "" {
			return 1, 0
		}
	case "tool_use":
		if event.ToolName != "" {
			toolCallCounts[event.ToolName]++
		}
	case "result":
		return 0, piResultTokenUsage(event.Stats)
	}
	return 0, 0
}

func piResultTokenUsage(stats map[string]any) int {
	if stats == nil {
		return 0
	}
	tokenUsage := 0
	if inputTokens, ok := stats["input_tokens"].(float64); ok {
		tokenUsage += int(inputTokens)
	}
	if outputTokens, ok := stats["output_tokens"].(float64); ok {
		tokenUsage += int(outputTokens)
	}
	return tokenUsage
}

func piToolCallMap(toolCallCounts map[string]int) map[string]*ToolCallInfo {
	toolCallMap := make(map[string]*ToolCallInfo, len(toolCallCounts))
	for toolName, count := range toolCallCounts {
		toolCallMap[toolName] = &ToolCallInfo{
			Name:      toolName,
			CallCount: count,
		}
	}
	return toolCallMap
}
