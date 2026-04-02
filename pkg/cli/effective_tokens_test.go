//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveTokenMultiplierExactMatch(t *testing.T) {
	// Reset cached multipliers so tests start fresh
	loadedMultipliers = nil

	tests := []struct {
		model    string
		expected float64
	}{
		{"claude-sonnet-4.5", 1.0},
		{"claude-sonnet-4.6", 1.0},
		{"claude-haiku-4.5", 0.1},
		{"claude-opus-4.5", 5.0},
		{"gpt-4o", 1.0},
		{"gpt-4o-mini", 0.1},
		{"o1", 3.0},
		{"o3-mini", 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := effectiveTokenMultiplier(tt.model)
			assert.InDelta(t, tt.expected, got, 1e-9, "multiplier for %q", tt.model)
		})
	}
}

func TestEffectiveTokenMultiplierCaseInsensitive(t *testing.T) {
	loadedMultipliers = nil

	assert.InDelta(t, 1.0, effectiveTokenMultiplier("Claude-Sonnet-4.5"), 1e-9, "case-insensitive lookup should work")
	assert.InDelta(t, 0.1, effectiveTokenMultiplier("CLAUDE-HAIKU-4.5"), 1e-9, "uppercase should match")
}

func TestEffectiveTokenMultiplierPrefixMatch(t *testing.T) {
	loadedMultipliers = nil

	// A model variant not in the table should match via longest prefix
	got := effectiveTokenMultiplier("claude-sonnet-4.6-preview-20250101")
	assert.InDelta(t, 1.0, got, 1e-9, "should match claude-sonnet-4.6 via prefix")

	got = effectiveTokenMultiplier("gpt-4o-mini-2024-07-18")
	assert.InDelta(t, 0.1, got, 1e-9, "should match gpt-4o-mini via prefix")
}

func TestEffectiveTokenMultiplierUnknownModel(t *testing.T) {
	loadedMultipliers = nil

	// Completely unknown models should default to 1.0
	assert.InDelta(t, 1.0, effectiveTokenMultiplier("my-custom-model-v1"), 1e-9)
	assert.InDelta(t, 1.0, effectiveTokenMultiplier(""), 1e-9)
}

func TestComputeBaseWeightedTokens(t *testing.T) {
	loadedMultipliers = nil

	tests := []struct {
		name             string
		inputTokens      int
		outputTokens     int
		cacheReadTokens  int
		cacheWriteTokens int
		expected         float64
	}{
		{
			name:         "input and output only",
			inputTokens:  1000,
			outputTokens: 200,
			// base = 1.0*1000 + 0.1*0 + 4.0*200 + 1.0*0 = 1000 + 800 = 1800
			expected: 1800,
		},
		{
			name:            "with cache read",
			inputTokens:     1000,
			outputTokens:    200,
			cacheReadTokens: 400,
			// base = 1.0*1000 + 0.1*400 + 4.0*200 = 1000 + 40 + 800 = 1840
			expected: 1840,
		},
		{
			name:             "with cache write",
			inputTokens:      500,
			outputTokens:     100,
			cacheReadTokens:  200,
			cacheWriteTokens: 100,
			// base = 1.0*500 + 0.1*200 + 4.0*100 + 1.0*100 = 500 + 20 + 400 + 100 = 1020
			expected: 1020,
		},
		{
			name:     "all zeros",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeBaseWeightedTokens(tt.inputTokens, tt.outputTokens, tt.cacheReadTokens, tt.cacheWriteTokens)
			assert.InDelta(t, tt.expected, got, 1e-9)
		})
	}
}

func TestComputeModelEffectiveTokens(t *testing.T) {
	loadedMultipliers = nil

	tests := []struct {
		name             string
		model            string
		inputTokens      int
		outputTokens     int
		cacheReadTokens  int
		cacheWriteTokens int
		expected         int
	}{
		{
			name:         "sonnet at 1x",
			model:        "claude-sonnet-4.5",
			inputTokens:  1000,
			outputTokens: 200,
			// base = 1.0*1000 + 4.0*200 = 1800; ET = 1.0 * 1800 = 1800
			expected: 1800,
		},
		{
			name:         "haiku at 0.1x",
			model:        "claude-haiku-4.5",
			inputTokens:  1000,
			outputTokens: 200,
			// base = 1800; ET = 0.1 * 1800 = 180
			expected: 180,
		},
		{
			name:         "opus at 5x",
			model:        "claude-opus-4.5",
			inputTokens:  1000,
			outputTokens: 200,
			// base = 1800; ET = 5.0 * 1800 = 9000
			expected: 9000,
		},
		{
			name:             "includes cache tokens",
			model:            "claude-sonnet-4.5",
			inputTokens:      500,
			outputTokens:     100,
			cacheReadTokens:  400,
			cacheWriteTokens: 100,
			// base = 1.0*500 + 0.1*400 + 4.0*100 + 1.0*100 = 500+40+400+100 = 1040
			// ET = 1.0 * 1040 = 1040
			expected: 1040,
		},
		{
			name:     "zero tokens",
			model:    "claude-sonnet-4.5",
			expected: 0,
		},
		{
			name:        "unknown model defaults to 1x",
			model:       "unknown-model",
			inputTokens: 500,
			// base = 1.0*500 = 500; ET = 1.0 * 500 = 500
			expected: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeModelEffectiveTokens(tt.model, tt.inputTokens, tt.outputTokens, tt.cacheReadTokens, tt.cacheWriteTokens)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestPopulateEffectiveTokens(t *testing.T) {
	loadedMultipliers = nil

	summary := &TokenUsageSummary{
		ByModel: map[string]*ModelTokenUsage{
			"claude-sonnet-4.5": {
				InputTokens:  1000,
				OutputTokens: 200,
				// base = 1.0*1000 + 4.0*200 = 1800; ET = 1.0 * 1800 = 1800
			},
			"claude-haiku-4.5": {
				InputTokens:  2000,
				OutputTokens: 400,
				// base = 1.0*2000 + 4.0*400 = 3600; ET = 0.1 * 3600 = 360
			},
		},
	}

	populateEffectiveTokens(summary)

	sonnet := summary.ByModel["claude-sonnet-4.5"]
	require.NotNil(t, sonnet)
	assert.Equal(t, 1800, sonnet.EffectiveTokens, "sonnet effective tokens at 1x")

	haiku := summary.ByModel["claude-haiku-4.5"]
	require.NotNil(t, haiku)
	assert.Equal(t, 360, haiku.EffectiveTokens, "haiku effective tokens at 0.1x")

	assert.Equal(t, 2160, summary.TotalEffectiveTokens, "total = sonnet + haiku effective")
}

func TestPopulateEffectiveTokensNilSummary(t *testing.T) {
	// Should not panic on nil input
	assert.NotPanics(t, func() {
		populateEffectiveTokens(nil)
	})
}

func TestModelMultipliersJSONEmbedded(t *testing.T) {
	// Verify the embedded JSON parses without error
	loadedMultipliers = nil
	initMultipliers()
	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.NotEmpty(t, loadedMultipliers, "should have at least one multiplier entry")
}
