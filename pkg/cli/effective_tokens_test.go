//go:build !integration

package cli

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
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
	assert.InDelta(t, 6.0, loadedMultipliers["gpt-5.4-pro"], 1e-9, "gpt-5.4-pro should use updated multiplier")
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
	assert.InDelta(t, 57.0, loadedMultipliers["gpt-5.5-2026-04-23"], 1e-9, "gpt-5.5-2026-04-23 should match documented multiplier")
	assert.InDelta(t, 3.0, loadedMultipliers["o3-deep-research-2025-06-26"], 1e-9, "o3-deep-research-2025-06-26 should be present")
	assert.InDelta(t, 0.5, loadedMultipliers["o4-mini-deep-research-2025-06-26"], 1e-9, "o4-mini-deep-research-2025-06-26 should be present")
	assert.InDelta(t, 0.2, loadedMultipliers["gemini-2.5-flash-native-audio-preview-12-2025"], 1e-9, "gemini-2.5-flash-native-audio-preview-12-2025 should be present")
	assert.InDelta(t, 1.0, loadedMultipliers["gemini-2.5-pro-preview-tts"], 1e-9, "gemini-2.5-pro-preview-tts should be present")
	assert.InDelta(t, 0.1, loadedMultipliers["gemini-2.0-flash-lite-001"], 1e-9, "gemini-2.0-flash-lite-001 should be present")
}

func TestModelMultipliersInventoryUpdate20260519(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.InDelta(t, 1.0, loadedMultipliers["gpt-5-search-api"], 1e-9, "gpt-5-search-api should be present")
	assert.InDelta(t, 1.0, loadedMultipliers["gpt-5-search-api-2025-10-14"], 1e-9, "gpt-5-search-api-2025-10-14 should be present")
	assert.InDelta(t, 1.0, loadedMultipliers["gpt-5-chat-latest"], 1e-9, "gpt-5-chat-latest should be present")
}

func TestModelMultipliersInventoryUpdate20260520(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.InDelta(t, 14.0, loadedMultipliers["gemini-3.5-flash"], 1e-9, "gemini-3.5-flash should use the documented premium multiplier")
}

func TestModelMultipliersInventoryUpdate20260521(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.InDelta(t, 1.0, loadedMultipliers["gpt-4.1-mini"], 1e-9, "gpt-4.1-mini should match documented multiplier")
	assert.InDelta(t, 1.0, loadedMultipliers["gpt-4.1-nano"], 1e-9, "gpt-4.1-nano should match documented multiplier")
	assert.InDelta(t, 0.33, loadedMultipliers["gpt-5.1-codex-mini"], 1e-9, "gpt-5.1-codex-mini should match documented multiplier")
	assert.InDelta(t, 3.0, loadedMultipliers["gpt-5.2-pro"], 1e-9, "gpt-5.2-pro should match documented multiplier")
	assert.InDelta(t, 3.0, loadedMultipliers["gpt-5.2-pro-2025-12-11"], 1e-9, "gpt-5.2-pro-2025-12-11 should match documented multiplier")
	assert.InDelta(t, 6.0, loadedMultipliers["gpt-5.4-nano-2026-03-17"], 1e-9, "gpt-5.4-nano-2026-03-17 should match documented multiplier")
	assert.InDelta(t, 6.0, loadedMultipliers["gpt-5.4-pro-2026-03-05"], 1e-9, "gpt-5.4-pro-2026-03-05 should match documented multiplier")
	assert.InDelta(t, 0.33, loadedMultipliers["gemini-3-flash-preview"], 1e-9, "gemini-3-flash-preview should be present with official billing multiplier")
	assert.InDelta(t, 6.0, loadedMultipliers["gemini-3-pro-preview"], 1e-9, "gemini-3-pro-preview should be present with official billing multiplier")
	assert.InDelta(t, 6.0, loadedMultipliers["gemini-3.1-pro-preview"], 1e-9, "gemini-3.1-pro-preview should be present with official billing multiplier")
	assert.NotContains(t, loadedMultipliers, "gemini-3-flash", "gemini-3-flash should not be present when only preview variant is defined")
	assert.NotContains(t, loadedMultipliers, "gemini-3-pro", "gemini-3-pro should not be present when only preview variant is defined")
	assert.NotContains(t, loadedMultipliers, "gemini-3.1-pro", "gemini-3.1-pro should not be present when only preview variant is defined")
}

func TestModelMultipliersInventoryUpdate20260525(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.InDelta(t, 27.0, loadedMultipliers["claude-opus-4-7"], 1e-9, "claude-opus-4-7 should match claude-opus-4.7 multiplier")
	assert.InDelta(t, 0.33, loadedMultipliers["gpt-4o-2024-05-13"], 1e-9, "gpt-4o-2024-05-13 should match gpt-4o multiplier")
	assert.InDelta(t, 0.33, loadedMultipliers["gpt-4o-2024-08-06"], 1e-9, "gpt-4o-2024-08-06 should match gpt-4o multiplier")
	assert.InDelta(t, 0.33, loadedMultipliers["gpt-4o-2024-11-20"], 1e-9, "gpt-4o-2024-11-20 should match gpt-4o multiplier")
	assert.InDelta(t, 0.33, loadedMultipliers["gpt-4o-mini-2024-07-18"], 1e-9, "gpt-4o-mini-2024-07-18 should match gpt-4o-mini multiplier")
	assert.InDelta(t, 1.0, loadedMultipliers["gpt-4-o-preview"], 1e-9, "gpt-4-o-preview should be present with legacy gpt-4 tier multiplier")
	assert.InDelta(t, 1.0, loadedMultipliers["gpt-4-0613"], 1e-9, "gpt-4-0613 should be present with legacy gpt-4 tier multiplier")
	assert.InDelta(t, 0.0, loadedMultipliers["gpt-3.5-turbo"], 1e-9, "gpt-3.5-turbo should be present with zero multiplier")
	assert.InDelta(t, 0.0, loadedMultipliers["gpt-3.5-turbo-0613"], 1e-9, "gpt-3.5-turbo-0613 should be present with zero multiplier")
}

func TestModelMultipliersInventoryUpdate20260530(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.InDelta(t, 27.0, loadedMultipliers["claude-opus-4-7"], 1e-9, "claude-opus-4-7 should match documented multiplier")
	assert.InDelta(t, 57.0, loadedMultipliers["gpt-5.5"], 1e-9, "gpt-5.5 should match the documented multiplier")
	assert.InDelta(t, 57.0, loadedMultipliers["gpt-5.5-2026-04-23"], 1e-9, "gpt-5.5-2026-04-23 should match the documented multiplier")
	assert.InDelta(t, 27.0, loadedMultipliers["claude-opus-4-8"], 1e-9, "claude-opus-4-8 should be present in current registry")
	assert.InDelta(t, 27.0, loadedMultipliers["claude-opus-4.7"], 1e-9, "claude-opus-4.7 alias should be present in current registry")
	assert.InDelta(t, 27.0, loadedMultipliers["claude-opus-4.8"], 1e-9, "claude-opus-4.8 alias should be present in current registry")
}

func TestModelMultipliersInventoryUpdate20260603(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.InDelta(t, 27.0, loadedMultipliers["claude-opus-4.6-fast"], 1e-9, "claude-opus-4.6-fast should match inferred opus 4.6 multiplier")
	assert.InDelta(t, 0.33, loadedMultipliers["mai-code-1-flash"], 1e-9, "MAI-Code-1-Flash should match documented billing multiplier")
}

func TestModelMultipliersInventoryUpdate20260602(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.InDelta(t, 1.0, loadedMultipliers["antigravity-preview-05-2026"], 1e-9, "antigravity-preview-05-2026 should match inferred pro-tier multiplier")
	assert.InDelta(t, 0.2, loadedMultipliers["nano-banana-pro-preview"], 1e-9, "nano-banana-pro-preview should match inferred lightweight multiplier")
}

func TestModelMultipliersRemovedCopilotAliases(t *testing.T) {
	loadedMultipliers = nil
	initMultipliers()

	require.NotNil(t, loadedMultipliers, "multipliers should be loaded from embedded JSON")
	assert.NotContains(t, loadedMultipliers, "gpt-4o", "gpt-4o should remain removed from the alias list")
	assert.NotContains(t, loadedMultipliers, "gpt-4o-mini", "gpt-4o-mini should remain removed from the alias list")
	assert.NotContains(t, loadedMultipliers, "gpt-4.1", "gpt-4.1 should remain removed from the alias list")
	assert.NotContains(t, loadedMultipliers, "gpt-5.4-nano", "gpt-5.4-nano should remain removed from the alias list")
}

// TestModelMultipliersNoPlaceholders verifies R-REG-007: the registry MUST NOT contain
// placeholder values such as null, "TBD", or empty strings for any model multiplier entry.
// Each declared model key MUST map to a finite numeric multiplier value.
// See effective-tokens-specification.md §Model Multiplier Registry.
func TestModelMultipliersNoPlaceholders(t *testing.T) {
	// Parse the raw JSON to detect null or non-numeric values that map[string]float64
	// would silently discard or coerce.
	var raw struct {
		Multipliers map[string]json.RawMessage `json:"multipliers"`
	}
	require.NoError(t, json.Unmarshal(modelMultipliersJSON, &raw),
		"model_multipliers.json must be valid JSON")
	require.NotEmpty(t, raw.Multipliers, "multipliers map must not be empty")

	for model, rawVal := range raw.Multipliers {
		assert.NotEmpty(t, model, "multiplier key must not be an empty string")

		// Reject null values.
		assert.NotEqual(t, "null", string(rawVal),
			"R-REG-007: multiplier for %q must not be null", model)

		// Reject string values such as "TBD".
		var asString string
		if json.Unmarshal(rawVal, &asString) == nil {
			t.Errorf("R-REG-007: multiplier for %q must be a number, got string %q", model, asString)
			continue
		}

		// Value must be a valid finite float64.
		var asFloat float64
		require.NoError(t, json.Unmarshal(rawVal, &asFloat),
			"R-REG-007: multiplier for %q must be a numeric value", model)
		assert.False(t, math.IsNaN(asFloat),
			"R-REG-007: multiplier for %q must not be NaN", model)
		assert.False(t, math.IsInf(asFloat, 0),
			"R-REG-007: multiplier for %q must not be Inf", model)
	}
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

func TestModelMultiplierSourcePrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	mergedPath := filepath.Join(tmpDir, "merged_model_multipliers.json")
	envJSON := `{"token_class_weights":{"input":1,"cached_input":0.1,"output":4,"reasoning":4,"cache_write":1},"multipliers":{"model-x":2}}`
	mergedJSON := `{"token_class_weights":{"input":1,"cached_input":0.1,"output":4,"reasoning":4,"cache_write":1},"multipliers":{"model-x":7}}`
	require.NoError(t, os.WriteFile(mergedPath, []byte(mergedJSON), 0o644))

	t.Setenv(mergedModelMultipliersPathEnvVar, mergedPath)
	t.Setenv(modelMultipliersEnvVar, envJSON)
	loadedMultipliers = nil

	multipliers, _ := resolveEffectiveWeights(nil)
	assert.InDelta(t, 7.0, multipliers["model-x"], 1e-9, "merged multipliers file should take precedence over env var")

	require.NoError(t, os.Remove(mergedPath))
	loadedMultipliers = nil
	multipliers, _ = resolveEffectiveWeights(nil)
	assert.InDelta(t, 2.0, multipliers["model-x"], 1e-9, "env var should take precedence when merged file is unavailable")

	t.Setenv(modelMultipliersEnvVar, "{bad json")
	loadedMultipliers = nil
	multipliers, _ = resolveEffectiveWeights(nil)
	_, hasModelX := multipliers["model-x"]
	assert.False(t, hasModelX, "built-in defaults should be used when env var JSON is malformed")
	_, hasBuiltin := multipliers["claude-sonnet-4.5"]
	assert.True(t, hasBuiltin, "built-in defaults should still load when env var JSON is malformed")
}

func TestComputeModelEffectiveTokensWithWeights_UnknownModelFallbackAndEffectiveInput(t *testing.T) {
	w := defaultTokenClassWeights()
	multipliers := map[string]float64{"known-model": 2.0}

	// effective_input=max(50-100,0)=0
	// base=(1.0*0)+(0.1*100)+(4.0*80)+(4.0*10)+(1.0*0)=370
	// unknown model fallback multiplier=1.0 -> ET=370
	et := computeModelEffectiveTokensWithWeights(effectiveTokensOptions{
		model:            "unknown-model",
		provider:         "anthropic",
		inputTokens:      50,
		outputTokens:     80,
		cacheReadTokens:  100,
		cacheWriteTokens: 0,
		reasoningTokens:  10,
		multipliers:      multipliers,
		weights:          w,
	})
	assert.Equal(t, 370, et)
}

func TestComputeModelEffectiveTokensWithWeights_NoCacheReadSubtractionForUnknownProvider(t *testing.T) {
	w := defaultTokenClassWeights()
	multipliers := map[string]float64{"known-model": 2.0}

	// Provider "test-provider" is treated as additive cache semantics by default:
	// effective_input=50 (no subtraction), base=(1.0*50)+(0.1*100)+(4.0*80)+(4.0*10)=420
	// unknown model fallback multiplier=1.0 -> ET=420
	et := computeModelEffectiveTokensWithWeights(effectiveTokensOptions{
		model:            "unknown-model",
		provider:         "test-provider",
		inputTokens:      50,
		outputTokens:     80,
		cacheReadTokens:  100,
		cacheWriteTokens: 0,
		reasoningTokens:  10,
		multipliers:      multipliers,
		weights:          w,
	})
	assert.Equal(t, 420, et)

	// Contrast with a known bundled provider where subtraction applies.
	etBundled := computeModelEffectiveTokensWithWeights(effectiveTokensOptions{
		model:            "unknown-model",
		provider:         "anthropic",
		inputTokens:      50,
		outputTokens:     80,
		cacheReadTokens:  100,
		cacheWriteTokens: 0,
		reasoningTokens:  10,
		multipliers:      multipliers,
		weights:          w,
	})
	assert.Equal(t, 370, etBundled)
}

func TestPopulateEffectiveTokensWithCustomWeightsRecomputesFromRawUsage(t *testing.T) {
	summary := &TokenUsageSummary{
		ByModel: map[string]*ModelTokenUsage{
			"unknown": {
				InputTokens:     10,
				OutputTokens:    5,
				EffectiveTokens: 9999, // stale precomputed value should be ignored
			},
		},
	}

	customA := &types.TokenWeights{
		TokenClassWeights: &types.TokenClassWeights{Output: 4.0},
	}
	customB := &types.TokenWeights{
		TokenClassWeights: &types.TokenClassWeights{Output: 8.0},
	}

	populateEffectiveTokensWithCustomWeights(summary, customA)
	assert.Equal(t, 30, summary.TotalEffectiveTokens, "ET should be recomputed from raw usage under initial weights")

	populateEffectiveTokensWithCustomWeights(summary, customB)
	assert.Equal(t, 50, summary.TotalEffectiveTokens, "ET should be recomputed from raw usage when token class weights change")
	assert.Equal(t, 50, summary.ByModel["unknown"].EffectiveTokens, "stale stored ET must not be reused")
}
