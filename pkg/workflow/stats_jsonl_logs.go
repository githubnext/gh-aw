package workflow

import (
	"encoding/json"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

type statsJSONResponse struct {
	Response string         `json:"response"`
	Stats    map[string]any `json:"stats"`
}

func parseStatsJSONLMetrics(logContent string, verbose bool, engineName string, log *logger.Logger) LogMetrics {
	log.Printf("Parsing %s log metrics: log_size=%d bytes, verbose=%v", engineName, len(logContent), verbose)

	metrics := LogMetrics{
		Turns:      0,
		TokenUsage: 0,
		ToolCalls:  []ToolCallInfo{},
	}
	toolCallCounts := make(map[string]int)

	for line := range strings.SplitSeq(logContent, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var response statsJSONResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			continue
		}

		applyStatsJSONResponseMetrics(response, &metrics, toolCallCounts)
		if verbose {
			log.Printf("Parsed JSON response: response_len=%d, stats_present=%v", len(response.Response), response.Stats != nil)
		}
	}

	for toolName, count := range toolCallCounts {
		metrics.ToolCalls = append(metrics.ToolCalls, ToolCallInfo{
			Name:      toolName,
			CallCount: count,
		})
	}

	log.Printf("Parsed metrics: turns=%d, token_usage=%d, tool_calls=%d",
		metrics.Turns, metrics.TokenUsage, len(metrics.ToolCalls))

	return metrics
}

func applyStatsJSONResponseMetrics(response statsJSONResponse, metrics *LogMetrics, toolCallCounts map[string]int) {
	if response.Response != "" {
		metrics.Turns = 1
	}

	if response.Stats == nil {
		return
	}

	if models, ok := response.Stats["models"].(map[string]any); ok {
		for _, modelStats := range models {
			stats, ok := modelStats.(map[string]any)
			if !ok {
				continue
			}

			if inputTokens, ok := stats["input_tokens"].(float64); ok {
				metrics.TokenUsage += int(inputTokens)
			}
			if outputTokens, ok := stats["output_tokens"].(float64); ok {
				metrics.TokenUsage += int(outputTokens)
			}
		}
	}

	if tools, ok := response.Stats["tools"].(map[string]any); ok {
		for toolName := range tools {
			toolCallCounts[toolName]++
		}
	}
}
