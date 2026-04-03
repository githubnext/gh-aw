//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelMultipliersJSONEmbedded(t *testing.T) {
	// Verify the embedded JSON parses without error
	loadedMultipliers = nil
	initMultipliers()
	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.NotEmpty(t, loadedMultipliers, "should have at least one multiplier entry")
}

func TestResolveEffectiveWeightsNoCustom(t *testing.T) {
	loadedMultipliers = nil

	multipliers, classWeights := resolveEffectiveWeights(nil)

	assert.NotEmpty(t, multipliers, "should have built-in multipliers")
	assert.InDelta(t, 1.0, classWeights.Input, 1e-9, "default input weight")
	assert.InDelta(t, 0.1, classWeights.CachedInput, 1e-9, "default cached input weight")
	assert.InDelta(t, 4.0, classWeights.Output, 1e-9, "default output weight")
}

func TestResolveEffectiveWeightsCustomMultipliers(t *testing.T) {
	loadedMultipliers = nil

	custom := &types.TokenWeights{
		Multipliers: map[string]float64{
			"my-custom-model":   2.5,
			"claude-sonnet-4.5": 1.5, // override existing
		},
	}
	multipliers, classWeights := resolveEffectiveWeights(custom)

	assert.InDelta(t, 2.5, multipliers["my-custom-model"], 1e-9, "custom model multiplier")
	assert.InDelta(t, 1.5, multipliers["claude-sonnet-4.5"], 1e-9, "overridden model multiplier")
	// Built-in models not mentioned in custom should remain
	assert.InDelta(t, 0.1, multipliers["claude-haiku-4.5"], 1e-9, "unmodified built-in multiplier")
	// Class weights unchanged when not specified
	assert.InDelta(t, 4.0, classWeights.Output, 1e-9, "output weight unchanged")
}

func TestResolveEffectiveWeightsCustomClassWeights(t *testing.T) {
	loadedMultipliers = nil

	custom := &types.TokenWeights{
		TokenClassWeights: &types.TokenClassWeights{
			Output:      6.0,
			CachedInput: 0.05,
		},
	}
	_, classWeights := resolveEffectiveWeights(custom)

	assert.InDelta(t, 6.0, classWeights.Output, 1e-9, "custom output weight")
	assert.InDelta(t, 0.05, classWeights.CachedInput, 1e-9, "custom cached input weight")
	// Unset fields keep their defaults
	assert.InDelta(t, 1.0, classWeights.Input, 1e-9, "input weight unchanged")
	assert.InDelta(t, 4.0, classWeights.Reasoning, 1e-9, "reasoning weight unchanged")
}

func TestPopulateEffectiveTokensWithCustomWeights(t *testing.T) {
	loadedMultipliers = nil

	summary := &TokenUsageSummary{
		ByModel: map[string]*ModelTokenUsage{
			"my-custom-model": {
				InputTokens:  1000,
				OutputTokens: 200,
			},
			"claude-sonnet-4.5": {
				InputTokens:  500,
				OutputTokens: 100,
			},
		},
	}

	custom := &types.TokenWeights{
		Multipliers: map[string]float64{
			"my-custom-model": 3.0,
		},
	}

	populateEffectiveTokensWithCustomWeights(summary, custom)

	// my-custom-model: base = 1.0*1000 + 4.0*200 = 1800; ET = 3.0 * 1800 = 5400
	customModel := summary.ByModel["my-custom-model"]
	require.NotNil(t, customModel, "custom model should be present")
	assert.Equal(t, 5400, customModel.EffectiveTokens, "custom model effective tokens at 3.0x")

	// claude-sonnet-4.5: base = 1.0*500 + 4.0*100 = 900; ET = 1.0 * 900 = 900
	sonnet := summary.ByModel["claude-sonnet-4.5"]
	require.NotNil(t, sonnet, "sonnet should be present")
	assert.Equal(t, 900, sonnet.EffectiveTokens, "sonnet effective tokens at 1x")

	assert.Equal(t, 6300, summary.TotalEffectiveTokens, "total = custom + sonnet")
}

func TestPopulateEffectiveTokensWithCustomWeightsNilSummary(t *testing.T) {
	assert.NotPanics(t, func() {
		populateEffectiveTokensWithCustomWeights(nil, nil)
	})
}
