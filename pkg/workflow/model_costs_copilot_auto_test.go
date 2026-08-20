//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithCopilotAutoPricingAddsBuiltinEntry verifies that the builtin pricing overlay
// for "copilot/auto" is injected when the workflow supplies no provider pricing.
// Without it the AWF API proxy rejects the zero-config default model with HTTP 400
// missing_model_pricing while maxAiCredits is active.
func TestWithCopilotAutoPricingAddsBuiltinEntry(t *testing.T) {
	t.Parallel()

	providers := withCopilotAutoPricing(nil)

	providerEntry, ok := providers["github-copilot"].(map[string]any)
	require.True(t, ok, "github-copilot provider entry should be present")
	models, ok := providerEntry["models"].(map[string]any)
	require.True(t, ok, "github-copilot models map should be present")
	autoEntry, ok := models["auto"].(map[string]any)
	require.True(t, ok, "auto model entry should be present")
	cost, ok := autoEntry["cost"].(map[string]any)
	require.True(t, ok, "auto cost map should be present")

	assert.Equal(t, "3e-06", cost["input"], "input rate should be the Sonnet-class rate")
	assert.Equal(t, "1.5e-05", cost["output"], "output rate should be the Sonnet-class rate")
}

// TestWithCopilotAutoPricingPreservesFrontmatterPricing verifies that workflow-supplied
// models.providers pricing for github-copilot/auto always wins over the builtin overlay.
func TestWithCopilotAutoPricingPreservesFrontmatterPricing(t *testing.T) {
	t.Parallel()

	frontmatter := map[string]any{
		"github-copilot": map[string]any{
			"models": map[string]any{
				"auto": map[string]any{
					"cost": map[string]any{"input": "1e-09", "output": "2e-09"},
				},
			},
		},
	}

	providers := withCopilotAutoPricing(frontmatter)

	cost := providers["github-copilot"].(map[string]any)["models"].(map[string]any)["auto"].(map[string]any)["cost"].(map[string]any)
	assert.Equal(t, "1e-09", cost["input"], "frontmatter pricing must not be overwritten")
	assert.Equal(t, "2e-09", cost["output"], "frontmatter pricing must not be overwritten")
}

// TestWithCopilotAutoPricingDoesNotMutateInput verifies that the overlay leaves the
// caller's provider map untouched, since it is derived from shared frontmatter data.
func TestWithCopilotAutoPricingDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"openai": map[string]any{
			"models": map[string]any{
				"gpt-5-enterprise": map[string]any{"cost": map[string]any{"input": "3.75e-06"}},
			},
		},
	}

	providers := withCopilotAutoPricing(input)

	assert.Len(t, input, 1, "input map must not gain entries")
	assert.Contains(t, providers, "openai", "existing provider entries must be preserved")
	assert.Contains(t, providers, "github-copilot", "builtin copilot/auto pricing must be added")
}

// TestBuildAWFConfigJSONIncludesCopilotAutoPricing verifies that the compiled AWF config
// carries pricing for the Copilot "auto" selector even when the workflow configures no
// model pricing of its own.
func TestBuildAWFConfigJSONIncludesCopilotAutoPricing(t *testing.T) {
	t.Parallel()

	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		},
	}

	jsonStr, err := BuildAWFConfigJSON(config)
	require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

	assert.Contains(t, jsonStr, `"providers":{"github-copilot":{"models":{"auto":{"cost":`,
		"AWF config should publish builtin pricing for the copilot auto selector")
}
