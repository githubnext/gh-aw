//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindModelPricing(t *testing.T) {
	pricing, ok := findModelPricing("anthropic", "claude-sonnet-4.6")
	require.True(t, ok)
	assert.InDelta(t, 0.000003, pricing["input"], 1e-12)
}

func TestComputeModelInferenceAIC(t *testing.T) {
	aic := computeModelInferenceAIC("anthropic", "claude-sonnet-4.6", 1000, 200, 400, 50, 25)
	assert.InDelta(t, 0.54825, aic, 1e-9)
}

func TestNormalizeCatalogProvider(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github", "github-copilot"},
		{"copilot", "github-copilot"},
		{"github_models", "github-copilot"},
		{"GITHUB_MODELS", "github-copilot"},
		{"anthropic", "anthropic"},
		{"openai", "openai"},
		{"", ""},
	}
	for _, tt := range tests {
		name := tt.input
		if name == "" {
			name = "<empty>"
		}
		t.Run(name, func(t *testing.T) {
			got := normalizeCatalogProvider(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeModelInferenceAICGitHubModels(t *testing.T) {
	// provider="github_models" is written by the AWF proxy for Copilot engine runs;
	// it must normalize to "github-copilot" so pricing is found and AIC is non-zero.
	aicViaGitHubModels := computeModelInferenceAIC("github_models", "claude-sonnet-4.6", 1000, 200, 0, 0, 0)
	aicViaGitHubCopilot := computeModelInferenceAIC("github-copilot", "claude-sonnet-4.6", 1000, 200, 0, 0, 0)
	assert.Greater(t, aicViaGitHubModels, 0.0, "github_models provider should produce non-zero AIC")
	assert.InDelta(t, aicViaGitHubCopilot, aicViaGitHubModels, 1e-9, "github_models and github-copilot should yield identical AIC")
}

func TestComputeModelInferenceAICCopilotAlias(t *testing.T) {
	// provider="copilot" is another accepted alias for "github-copilot".
	aicViaCopilot := computeModelInferenceAIC("copilot", "claude-sonnet-4.6", 1000, 200, 0, 0, 0)
	aicViaGitHubCopilot := computeModelInferenceAIC("github-copilot", "claude-sonnet-4.6", 1000, 200, 0, 0, 0)
	assert.Greater(t, aicViaCopilot, 0.0, "copilot provider alias should produce non-zero AIC")
	assert.InDelta(t, aicViaGitHubCopilot, aicViaCopilot, 1e-9, "copilot and github-copilot should yield identical AIC")
}
