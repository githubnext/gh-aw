package compilerenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnterpriseVariables(t *testing.T) {
	vars := EnterpriseVariables()
	assert.Len(t, vars, 4)
	assert.Equal(t, DefaultMaxEffectiveTokens, vars[0].Name)
	assert.Equal(t, DefaultModelCopilot, vars[1].Name)
	assert.Equal(t, DefaultModelClaude, vars[2].Name)
	assert.Equal(t, DefaultModelCodex, vars[3].Name)
}

func TestResolveDefaultMaxEffectiveTokens(t *testing.T) {
	t.Run("unset uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxEffectiveTokens, "")
		assert.Equal(t, int64(10), ResolveDefaultMaxEffectiveTokens(10))
	})

	t.Run("invalid uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxEffectiveTokens, "abc")
		assert.Equal(t, int64(10), ResolveDefaultMaxEffectiveTokens(10))
	})

	t.Run("valid value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxEffectiveTokens, "424242")
		assert.Equal(t, int64(424242), ResolveDefaultMaxEffectiveTokens(10))
	})
}

func TestBuildModelOverrideExpression(t *testing.T) {
	assert.Equal(
		t,
		"${{ vars.GH_AW_MODEL_AGENT_CODEX || vars.GH_AW_DEFAULT_MODEL_CODEX || 'gpt-5.4' }}",
		BuildModelOverrideExpression("GH_AW_MODEL_AGENT_CODEX", "GH_AW_DEFAULT_MODEL_CODEX", "gpt-5.4"),
	)
	assert.Equal(
		t,
		"${{ vars.GH_AW_MODEL_AGENT_CLAUDE || vars.GH_AW_DEFAULT_MODEL_CLAUDE || '' }}",
		BuildModelOverrideExpressionEmptyFallback("GH_AW_MODEL_AGENT_CLAUDE", "GH_AW_DEFAULT_MODEL_CLAUDE"),
	)
}

