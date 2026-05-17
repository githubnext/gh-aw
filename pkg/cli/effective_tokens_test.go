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
	assert.InDelta(t, 0.33, multipliers["claude-haiku-4.5"], 1e-9, "unmodified built-in multiplier")
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

func TestModelMultipliersInventoryUpdate20260510(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.InDelta(t, 6.0, loadedMultipliers["gpt-5.4"], 1e-9, "gpt-5.4 should use updated multiplier")
	assert.InDelta(t, 6.0, loadedMultipliers["gpt-5.4-mini"], 1e-9, "gpt-5.4-mini should use updated multiplier")
	assert.InDelta(t, 0.05, loadedMultipliers["gpt-5.4-nano"], 1e-9, "gpt-5.4-nano should be present")
	assert.InDelta(t, 2.0, loadedMultipliers["gpt-5.4-pro"], 1e-9, "gpt-5.4-pro should be present")
	assert.InDelta(t, 27.0, loadedMultipliers["claude-opus-4.6"], 1e-9, "claude-opus-4.6 should use updated multiplier")
	assert.InDelta(t, 0.1, loadedMultipliers["gemini-3.1-flash-lite"], 1e-9, "gemini-3.1-flash-lite should be present")
	assert.InDelta(t, 6.0, loadedMultipliers["gemini-3.1-pro-preview-customtools"], 1e-9, "gemini-3.1-pro-preview-customtools should be present")
	assert.InDelta(t, 0.2, loadedMultipliers["gemini-2.5-computer-use-preview-10-2025"], 1e-9, "gemini-2.5-computer-use-preview-10-2025 should be present")
	assert.InDelta(t, 0.33, loadedMultipliers["grok-code-fast-1"], 1e-9, "grok-code-fast-1 should be present")
	assert.InDelta(t, 1.0, loadedMultipliers["deep-research-max-preview-04-2026"], 1e-9, "deep-research-max-preview-04-2026 should be present")
}

func TestModelMultipliersInventoryUpdate20260517(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.InDelta(t, 1.0, loadedMultipliers["gpt-5-2025-08-07"], 1e-9, "gpt-5-2025-08-07 should be present")
	assert.InDelta(t, 3.0, loadedMultipliers["gpt-5.2-chat-latest"], 1e-9, "gpt-5.2-chat-latest should be present")
	assert.InDelta(t, 6.0, loadedMultipliers["gpt-5.3-codex-api-preview"], 1e-9, "gpt-5.3-codex-api-preview should be present")
	assert.InDelta(t, 7.5, loadedMultipliers["gpt-5.5-2026-04-23"], 1e-9, "gpt-5.5-2026-04-23 should be present")
	assert.InDelta(t, 3.0, loadedMultipliers["o3-deep-research-2025-06-26"], 1e-9, "o3-deep-research-2025-06-26 should be present")
	assert.InDelta(t, 0.5, loadedMultipliers["o4-mini-deep-research-2025-06-26"], 1e-9, "o4-mini-deep-research-2025-06-26 should be present")
	assert.InDelta(t, 0.2, loadedMultipliers["gemini-2.5-flash-native-audio-preview-12-2025"], 1e-9, "gemini-2.5-flash-native-audio-preview-12-2025 should be present")
	assert.InDelta(t, 1.0, loadedMultipliers["gemini-2.5-pro-preview-tts"], 1e-9, "gemini-2.5-pro-preview-tts should be present")
	assert.InDelta(t, 0.1, loadedMultipliers["gemini-2.0-flash-lite-001"], 1e-9, "gemini-2.0-flash-lite-001 should be present")
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

	// claude-sonnet-4.5: base = 1.0*500 + 4.0*100 = 900; ET = 6.0 * 900 = 5400
	sonnet := summary.ByModel["claude-sonnet-4.5"]
	require.NotNil(t, sonnet, "sonnet should be present")
	assert.Equal(t, 5400, sonnet.EffectiveTokens, "sonnet effective tokens at 6x")

	assert.Equal(t, 10800, summary.TotalEffectiveTokens, "total = custom + sonnet")
}

func TestPopulateEffectiveTokensWithCustomWeightsNilSummary(t *testing.T) {
	assert.NotPanics(t, func() {
		populateEffectiveTokensWithCustomWeights(nil, nil)
	})
}
