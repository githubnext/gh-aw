//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeminiEngineParseLogMetrics(t *testing.T) {
	engine := NewGeminiEngine()

	t.Run("parses metrics from json lines", func(t *testing.T) {
		logContent := `not-json
{"response":"first response","stats":{"models":{"model-a":{"input_tokens":10,"output_tokens":5}},"tools":{"bash":{},"github.search":{}}}}
{"response":"","stats":{"models":{"model-a":{"input_tokens":1,"output_tokens":1}},"tools":{"bash":{}}}}`

		metrics := engine.ParseLogMetrics(logContent, false)

		assert.Equal(t, 1, metrics.Turns)
		assert.Equal(t, 17, metrics.TokenUsage)
		assert.Equal(t, map[string]int{
			"bash":          2,
			"github.search": 1,
		}, toolCallCountsByName(metrics.ToolCalls))
	})

	t.Run("returns zero metrics for empty input", func(t *testing.T) {
		metrics := engine.ParseLogMetrics("", false)
		assert.Equal(t, LogMetrics{
			Turns:      0,
			TokenUsage: 0,
			ToolCalls:  []ToolCallInfo{},
		}, metrics)
	})
}
