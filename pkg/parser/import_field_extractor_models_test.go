//go:build !integration

package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendModelsField_ExtractsModelPolicySets(t *testing.T) {
	acc := newImportAccumulator()
	fm := map[string]any{
		"models": map[string]any{
			"allowed": []any{"gpt-5", "claude-sonnet"},
			"blocked": []any{"gpt-5-pro"},
		},
	}

	acc.appendModelsField(fm, "import-a.md")

	require.Len(t, acc.modelPolicies, 1, "expected one model policy set")
	assert.Equal(t, []string{"gpt-5", "claude-sonnet"}, acc.modelPolicies[0]["allowed"])
	assert.Equal(t, []string{"gpt-5-pro"}, acc.modelPolicies[0]["blocked"])
	assert.Empty(t, acc.models, "policy fields should not be interpreted as model aliases")
}

func TestAppendModelsField_ExtractsModelCostsAndPolicyTogether(t *testing.T) {
	acc := newImportAccumulator()
	fm := map[string]any{
		"models": map[string]any{
			"allowed": []any{"gpt-5-mini"},
			"providers": map[string]any{
				"openai": map[string]any{
					"models": map[string]any{
						"gpt-5-mini": map[string]any{
							"cost": map[string]any{"input": "1e-6"},
						},
					},
				},
			},
		},
	}

	acc.appendModelsField(fm, "import-b.md")

	require.Len(t, acc.modelCosts, 1, "expected one model cost overlay")
	require.Len(t, acc.modelPolicies, 1, "expected one model policy set")
	assert.Equal(t, []string{"gpt-5-mini"}, acc.modelPolicies[0]["allowed"])
	assert.Contains(t, acc.modelCosts[0], "providers")
	assert.Len(t, acc.modelCosts[0], 1)
	for _, key := range []string{"allowed", "blocked"} {
		_, present := acc.modelCosts[0][key]
		assert.Falsef(t, present, "model cost overlay should not contain policy key %q", key)
	}
}

func TestAppendModelsField_InvalidPolicyAndProviders_EmitsWarningsAndSkipsCosts(t *testing.T) {
	acc := newImportAccumulator()
	fm := map[string]any{
		"models": map[string]any{
			"allowed":   "gpt-5",
			"providers": "not-an-object",
		},
	}

	acc.appendModelsField(fm, "import-c.md")

	assert.Empty(t, acc.modelPolicies)
	assert.Empty(t, acc.modelCosts)
	require.NotEmpty(t, acc.warnings)
	warningsText := strings.Join(acc.warnings, "\n")
	assert.Contains(t, warningsText, "models.allowed")
	assert.Contains(t, warningsText, "models.providers")
}

func TestAppendModelsField_InvalidModelsShape_EmitsWarning(t *testing.T) {
	acc := newImportAccumulator()
	fm := map[string]any{
		"models": []any{"not-an-object"},
	}

	acc.appendModelsField(fm, "import-d.md")

	assert.Empty(t, acc.modelPolicies)
	assert.Empty(t, acc.modelCosts)
	require.NotEmpty(t, acc.warnings)
	assert.Contains(t, strings.Join(acc.warnings, "\n"), "models field is not a valid object")
}

func TestAppendModelsField_ProvidersPolicyKeysAreExcludedFromModelCosts(t *testing.T) {
	acc := newImportAccumulator()
	fm := map[string]any{
		"models": map[string]any{
			"providers": map[string]any{
				"allowed": []any{"gpt-5"},
				"openai": map[string]any{
					"models": map[string]any{
						"gpt-5": map[string]any{
							"cost": map[string]any{"input": "1e-6"},
						},
					},
				},
			},
		},
	}

	acc.appendModelsField(fm, "import-e.md")

	require.Len(t, acc.modelCosts, 1)
	providers, ok := acc.modelCosts[0]["providers"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, providers, "openai")
	assert.NotContains(t, providers, "allowed")
	assert.NotContains(t, providers, "blocked")
	require.NotEmpty(t, acc.warnings)
	assert.Contains(t, strings.Join(acc.warnings, "\n"), "models.providers.allowed is reserved for policy")
}

func TestAppendModelsField_ProvidersAndAliasesBothExtracted(t *testing.T) {
	acc := newImportAccumulator()
	fm := map[string]any{
		"models": map[string]any{
			"providers": map[string]any{
				"openai": map[string]any{
					"models": map[string]any{
						"gpt-5": map[string]any{
							"cost": map[string]any{"input": "1e-6"},
						},
					},
				},
			},
			"agent": []any{"gpt-5"},
		},
	}

	acc.appendModelsField(fm, "import-f.md")

	require.Len(t, acc.modelCosts, 1)
	require.Len(t, acc.models, 1)
	assert.Equal(t, []string{"gpt-5"}, acc.models[0]["agent"])
}

func TestAppendModelsField_ExtractsDefaultAICreditsPricingFirstWins(t *testing.T) {
	acc := newImportAccumulator()
	first := map[string]any{
		"models": map[string]any{
			"default-ai-credits-pricing": map[string]any{
				"input":  5.0,
				"output": 25.0,
			},
		},
	}
	second := map[string]any{
		"models": map[string]any{
			"default-ai-credits-pricing": map[string]any{
				"input":  1.0,
				"output": 2.0,
			},
		},
	}

	acc.appendModelsField(first, "import-first.md")
	acc.appendModelsField(second, "import-second.md")

	require.NotNil(t, acc.defaultAiCreditsPricing)
	input, ok := acc.defaultAiCreditsPricing["input"].(float64)
	require.True(t, ok)
	output, ok := acc.defaultAiCreditsPricing["output"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 5.0, input, 1e-9)
	assert.InDelta(t, 25.0, output, 1e-9)
}

func TestAppendModelsField_InvalidDefaultAICreditsPricingWarns(t *testing.T) {
	acc := newImportAccumulator()
	fm := map[string]any{
		"models": map[string]any{
			"default-ai-credits-pricing": "not-an-object",
		},
	}

	acc.appendModelsField(fm, "import-invalid.md")

	assert.Nil(t, acc.defaultAiCreditsPricing)
	require.NotEmpty(t, acc.warnings)
	assert.Contains(t, strings.Join(acc.warnings, "\n"), "models.default-ai-credits-pricing must be an object")
}
