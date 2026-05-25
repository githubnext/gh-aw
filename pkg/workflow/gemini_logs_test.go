//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeminiEngine_ParseLogMetrics_UsesFinalizeToolMetrics(t *testing.T) {
	engine := NewGeminiEngine()

	lines := []string{
		toJSON(map[string]any{
			"response": "done",
			"stats": map[string]any{
				"models": map[string]any{
					"gemini-2.5": map[string]any{
						"input_tokens":  float64(3),
						"output_tokens": float64(7),
					},
				},
				"tools": map[string]any{
					"bash":      map[string]any{},
					"read_file": map[string]any{},
				},
			},
		}),
		toJSON(map[string]any{
			"stats": map[string]any{
				"models": map[string]any{
					"gemini-2.5": map[string]any{
						"input_tokens":  float64(10),
						"output_tokens": float64(20),
					},
				},
				"tools": map[string]any{
					"bash":   map[string]any{},
					"github": map[string]any{},
				},
			},
		}),
	}

	metrics := engine.ParseLogMetrics(strings.Join(lines, "\n"), false)

	assert.Equal(t, 1, metrics.Turns, "expected a single turn once any response is present")
	assert.Equal(t, 40, metrics.TokenUsage, "expected aggregated model token usage")
	assert.Len(t, metrics.ToolCalls, 3, "expected tool map finalization into deduplicated slice")

	assert.Equal(t, "bash", metrics.ToolCalls[0].Name)
	assert.Equal(t, 2, metrics.ToolCalls[0].CallCount)
	assert.Equal(t, "github", metrics.ToolCalls[1].Name)
	assert.Equal(t, 1, metrics.ToolCalls[1].CallCount)
	assert.Equal(t, "read_file", metrics.ToolCalls[2].Name)
	assert.Equal(t, 1, metrics.ToolCalls[2].CallCount)
}

func TestGeminiEngine_ParseLogMetrics_IgnoresInvalidLines(t *testing.T) {
	engine := NewGeminiEngine()

	logContent := strings.Join([]string{
		"not-json",
		"",
		toJSON(map[string]any{
			"response": "ok",
			"stats": map[string]any{
				"models": map[string]any{
					"gemini-2.5": map[string]any{
						"input_tokens":  float64(2),
						"output_tokens": float64(3),
					},
				},
			},
		}),
	}, "\n")

	metrics := engine.ParseLogMetrics(logContent, false)
	assert.Equal(t, 1, metrics.Turns)
	assert.Equal(t, 5, metrics.TokenUsage)
	assert.Empty(t, metrics.ToolCalls)
}
