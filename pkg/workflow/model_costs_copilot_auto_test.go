//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithCopilotAutoPricingAddsBuiltinEntry verifies that the builtin pricing overlay
// for "copilot/auto" is injected when the workflow supplies no provider pricing.
// Without it the AWF API proxy rejects the zero-config default model with HTTP 400
// unknown_model_ai_credits while maxAiCredits is active.
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

	assert.Equal(t, "1e-05", cost["input"], "input rate should be the conservative catalog maximum")
	assert.Equal(t, "5e-05", cost["output"], "output rate should be the conservative catalog maximum")
}

func TestCopilotAutoPricingCoversBundledCatalog(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../actions/setup/js/models.json")
	require.NoError(t, err, "bundled model catalog should be readable")

	var catalog struct {
		Providers map[string]struct {
			Models map[string]struct {
				Cost map[string]string `json:"cost"`
			} `json:"models"`
		} `json:"providers"`
	}
	require.NoError(t, json.Unmarshal(data, &catalog), "bundled model catalog should be valid JSON")

	provider, ok := catalog.Providers[copilotAutoPricingProvider]
	require.True(t, ok, "bundled model catalog should contain the Copilot pricing provider")
	require.NotEmpty(t, provider.Models, "bundled Copilot provider should contain priced models")

	for model, entry := range provider.Models {
		for costType, rawCost := range entry.Cost {
			fallbackCost, ok := copilotAutoCost[costType]
			require.True(t, ok, "copilot/auto fallback must define catalog cost type %s", costType)
			modelRate, err := strconv.ParseFloat(rawCost, 64)
			require.NoError(t, err, "catalog rate for %s/%s should be numeric", model, costType)
			fallbackRate, err := strconv.ParseFloat(fallbackCost, 64)
			require.NoError(t, err, "fallback rate for %s should be numeric", costType)
			assert.LessOrEqual(t, modelRate, fallbackRate,
				"copilot/auto fallback %s rate must cover bundled model %s", costType, model)
		}
	}
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

func TestBuildAWFConfigJSONScopesCopilotAutoPricing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		engineName     string
		model          string
		firewallConfig *FirewallConfig
		wantPricing    bool
	}{
		{
			name:           "current Copilot workflow",
			engineName:     "copilot",
			firewallConfig: &FirewallConfig{Enabled: true},
			wantPricing:    true,
		},
		{
			name:           "non-Copilot workflow",
			engineName:     "claude",
			firewallConfig: &FirewallConfig{Enabled: true},
		},
		{
			name:           "Copilot workflow with concrete model",
			engineName:     "copilot",
			model:          "claude-sonnet-5",
			firewallConfig: &FirewallConfig{Enabled: true},
		},
		{
			name:           "Copilot workflow with dynamic model expression",
			engineName:     "copilot",
			model:          "${{ inputs.model }}",
			firewallConfig: &FirewallConfig{Enabled: true},
			wantPricing:    true,
		},
		{
			name:           "Copilot workflow with provider-prefixed model expression",
			engineName:     "copilot",
			model:          "copilot/${{ inputs.model }}",
			firewallConfig: &FirewallConfig{Enabled: true},
			wantPricing:    true,
		},
		{
			name:           "Copilot workflow with expression model options",
			engineName:     "copilot",
			model:          "${{ inputs.model }}?effort=high",
			firewallConfig: &FirewallConfig{Enabled: true},
			wantPricing:    true,
		},
		{
			name:           "Copilot workflow with auto model options",
			engineName:     "copilot",
			model:          "auto?effort=high",
			firewallConfig: &FirewallConfig{Enabled: true},
			wantPricing:    true,
		},
		{
			name:           "Copilot workflow with provider-prefixed auto model options",
			engineName:     "copilot",
			model:          "copilot/auto?effort=high",
			firewallConfig: &FirewallConfig{Enabled: true},
			wantPricing:    true,
		},
		{
			name:           "Copilot workflow pinned below provider support",
			engineName:     "copilot",
			firewallConfig: &FirewallConfig{Enabled: true, Version: "v0.27.42"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := AWFCommandConfig{
				EngineName:     tt.engineName,
				AllowedDomains: "github.com",
				WorkflowData: &WorkflowData{
					Model:              tt.model,
					EngineConfig:       &EngineConfig{ID: tt.engineName},
					NetworkPermissions: &NetworkPermissions{Firewall: tt.firewallConfig},
				},
			}

			jsonStr, err := BuildAWFConfigJSON(config)
			require.NoError(t, err, "BuildAWFConfigJSON should not return an error")
			if tt.wantPricing {
				assert.Contains(t, jsonStr, `"providers":{"github-copilot"`)
			} else {
				assert.NotContains(t, jsonStr, `"providers":{"github-copilot"`)
			}
		})
	}
}
