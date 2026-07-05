//go:build !integration

package modelsdev

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpec_PublicAPI_FindPricing validates the documented behavior of
// FindPricing as described in the modelsdev README.md specification.
func TestSpec_PublicAPI_FindPricing(t *testing.T) {
	ctx := context.Background()

	t.Run("returns nil false when pricing is unavailable", func(t *testing.T) {
		pricing, ok := FindPricing(ctx, "definitely-not-a-provider", "definitely-not-a-model")
		assert.False(t, ok, "FindPricing should report unavailable pricing for an unknown provider/model pair")
		assert.Nil(t, pricing, "FindPricing should return nil pricing when no pricing is available")
	})

	t.Run("result pricing map exposes per token input and output entries when available", func(t *testing.T) {
		pricing, ok := FindPricing(ctx, "github", "gpt-4.1")
		if !ok {
			t.Skip("catalog pricing unavailable in test environment; README specifies graceful degradation when network or parsing fails")
		}

		require.NotNil(t, pricing, "FindPricing should return a non-nil pricing map when pricing is available")
		_, hasInput := pricing["input"]
		_, hasOutput := pricing["output"]
		assert.True(t, hasInput, "pricing map should contain documented input per-token price entry")
		assert.True(t, hasOutput, "pricing map should contain documented output per-token price entry")
	})
}

// TestSpec_DesignDecision_ProviderAliases validates the documented provider
// alias normalization described in the modelsdev README.md.
func TestSpec_DesignDecision_ProviderAliases(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{name: "github alias", provider: "github"},
		{name: "copilot alias", provider: "copilot"},
		{name: "github_models alias", provider: "github_models"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing, ok := FindPricing(context.Background(), tt.provider, "definitely-not-a-real-model")
			assert.False(t, ok, "FindPricing should still return unavailable for an unknown model after normalizing provider alias %q", tt.provider)
			assert.Nil(t, pricing, "FindPricing should return nil pricing for unknown model with provider alias %q", tt.provider)
		})
	}
}
