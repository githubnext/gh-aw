//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractConfigFields_FirstWinsAndAccumulates(t *testing.T) {
	acc := newImportAccumulator()

	first := map[string]any{
		"max-turns":             10,
		"max-tool-denials":      5,
		"max-runs":              3,
		"max-turn-cache-misses": 4,
		"max-ai-credits":        1234,
		"max-daily-ai-credits":  4096,
		"mcp-servers":           map[string]any{"server-a": map[string]any{"url": "https://a.example.com"}},
		"safe-outputs":          map[string]any{"enabled": true},
		"mcp-scripts":           map[string]any{"setup": "echo first"},
		"runtimes":              map[string]any{"node": map[string]any{"version": "20"}},
		"network":               map[string]any{"allow": []any{"github.com"}},
		"permissions":           map[string]any{"contents": "read"},
		"secret-masking":        map[string]any{"enabled": true},
	}
	second := map[string]any{
		"max-turns":             99,
		"max-tool-denials":      11,
		"max-runs":              88,
		"max-turn-cache-misses": 99,
		"max-ai-credits":        55,
		"max-daily-ai-credits":  66,
		"mcp-servers":           map[string]any{"server-b": map[string]any{"url": "https://b.example.com"}},
		"safe-outputs":          map[string]any{"mode": "strict"},
		"mcp-scripts":           map[string]any{"teardown": "echo second"},
		"runtimes":              map[string]any{"python": map[string]any{"version": "3.12"}},
		"network":               map[string]any{"allow": []any{"api.github.com"}},
		"permissions":           map[string]any{"issues": "write"},
		"secret-masking":        map[string]any{"log-mask": true},
	}

	acc.extractConfigFields(first, "first.md")
	acc.extractConfigFields(second, "second.md")

	assert.Equal(t, "10", acc.mergedMaxTurns, "max-turns should be first-wins")
	assert.Equal(t, "5", acc.mergedMaxToolDenials, "max-tool-denials should be first-wins")
	assert.Equal(t, "3", acc.mergedMaxRuns, "max-runs should be first-wins")
	assert.Equal(t, "4", acc.mergedMaxTurnCacheMisses, "max-turn-cache-misses should be first-wins")
	assert.Equal(t, "1234", acc.mergedMaxAICredits, "max-ai-credits should be first-wins")
	assert.Equal(t, "4096", acc.mergedMaxDailyAICredits, "max-daily-ai-credits should be first-wins")

	assert.Len(t, acc.safeOutputs, 2, "safe-outputs should accumulate across imports")
	assert.Len(t, acc.mcpScripts, 2, "mcp-scripts should accumulate across imports")
	assert.Contains(t, acc.mcpServersBuilder.String(), "server-a")
	assert.Contains(t, acc.mcpServersBuilder.String(), "server-b")
	assert.Contains(t, acc.runtimesBuilder.String(), "node")
	assert.Contains(t, acc.runtimesBuilder.String(), "python")
	assert.Contains(t, acc.networkBuilder.String(), "github.com")
	assert.Contains(t, acc.networkBuilder.String(), "api.github.com")
	assert.Contains(t, acc.permissionsBuilder.String(), "contents")
	assert.Contains(t, acc.permissionsBuilder.String(), "issues")
	assert.Contains(t, acc.secretMaskingBuilder.String(), "enabled")
	assert.Contains(t, acc.secretMaskingBuilder.String(), "log-mask")
}
