//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeModelPolicyOverlays_UnionizesAllowedAndDisallowed(t *testing.T) {
	imported := []map[string][]string{
		{
			"allowed":    {"gpt-5", "claude-sonnet"},
			"disallowed": {"gpt-5-pro"},
		},
		{
			"allowed":    {"gpt-5-mini"},
			"disallowed": {"claude-opus"},
		},
	}
	main := map[string][]string{
		"allowed":    {"gpt-5"},
		"disallowed": {"gemini-pro"},
	}

	allowed, disallowed := mergeModelPolicyOverlays(imported, main)
	assert.Equal(t, []string{"claude-sonnet", "gpt-5", "gpt-5-mini"}, allowed)
	assert.Equal(t, []string{"claude-opus", "gemini-pro", "gpt-5-pro"}, disallowed)
}

func TestMergeModelPolicyOverlays_DisallowedWinsOnConflict(t *testing.T) {
	imported := []map[string][]string{
		{
			"allowed":    {"gpt-5"},
			"disallowed": {"gpt-5"},
		},
	}
	allowed, disallowed := mergeModelPolicyOverlays(imported, nil)
	assert.Empty(t, allowed)
	assert.Equal(t, []string{"gpt-5"}, disallowed)
}

func TestMergeModelPolicyOverlays_DisallowedWildcardWinsOnConflict(t *testing.T) {
	imported := []map[string][]string{
		{
			"allowed":    {"claude-opus", "claude-sonnet"},
			"disallowed": {"*opus*"},
		},
	}
	allowed, disallowed := mergeModelPolicyOverlays(imported, nil)
	assert.Equal(t, []string{"claude-sonnet"}, allowed)
	assert.Equal(t, []string{"*opus*"}, disallowed)
}

func TestMergeModelPolicyOverlays_AllowedWildcardConflictsWithDisallowedExact(t *testing.T) {
	imported := []map[string][]string{
		{
			"allowed":    {"*opus*"},
			"disallowed": {"claude-opus"},
		},
	}
	allowed, disallowed := mergeModelPolicyOverlays(imported, nil)
	assert.Empty(t, allowed)
	assert.Equal(t, []string{"claude-opus"}, disallowed)
}

func TestExtractMainModelPolicyOverlay_UsesParsedFrontmatterWhenPresent(t *testing.T) {
	toolsResult := &toolsProcessingResult{
		parsedFrontmatter: &FrontmatterConfig{
			ModelPolicyAllowed:    []string{"gpt-5"},
			ModelPolicyDisallowed: []string{"gpt-5-pro"},
		},
	}

	policy := extractMainModelPolicyOverlay(toolsResult, map[string]any{})
	require.NotNil(t, policy)
	assert.Equal(t, []string{"gpt-5"}, policy["allowed"])
	assert.Equal(t, []string{"gpt-5-pro"}, policy["disallowed"])
}

func TestExtractMainModelPolicyOverlay_FallsBackToRawFrontmatter(t *testing.T) {
	toolsResult := &toolsProcessingResult{}
	frontmatter := map[string]any{
		"models": map[string]any{
			"allowed":    []any{"gpt-5-mini"},
			"disallowed": []any{"claude-opus"},
		},
	}

	policy := extractMainModelPolicyOverlay(toolsResult, frontmatter)
	require.NotNil(t, policy)
	assert.Equal(t, []string{"gpt-5-mini"}, policy["allowed"])
	assert.Equal(t, []string{"claude-opus"}, policy["disallowed"])
}

func TestExtractMainModelCostsOverlay_ExtractsNilWhenModelCostsHasOnlyPolicyKeys(t *testing.T) {
	toolsResult := &toolsProcessingResult{
		parsedFrontmatter: &FrontmatterConfig{
			ModelCosts: map[string]any{
				"allowed": []any{"gpt-5"},
			},
		},
	}

	costs := extractMainModelCostsOverlay(toolsResult, map[string]any{})
	assert.Nil(t, costs)
}

func TestExtractMainModelCostsOverlay_ExtractsOnlyProvidersAndExcludesPolicyKeys(t *testing.T) {
	toolsResult := &toolsProcessingResult{}
	frontmatter := map[string]any{
		"models": map[string]any{
			"allowed": []any{"gpt-5"},
			"providers": map[string]any{
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

	costs := extractMainModelCostsOverlay(toolsResult, frontmatter)
	require.NotNil(t, costs)
	assert.Contains(t, costs, "providers")
	assert.NotContains(t, costs, "allowed")
}
