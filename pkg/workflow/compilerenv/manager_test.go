package compilerenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnterpriseVariables(t *testing.T) {
	vars := EnterpriseVariables()
	names := make([]string, 0, len(vars))
	for _, v := range vars {
		names = append(names, v.Name)
	}
	assert.Contains(t, names, DefaultMaxEffectiveTokens)
	assert.Contains(t, names, DefaultMaxTurns)
	assert.Contains(t, names, DefaultTimeoutMinutes)
	assert.Contains(t, names, DefaultDetectionModel)
	assert.Contains(t, names, DefaultModelCopilot)
	assert.Contains(t, names, DefaultModelClaude)
	assert.Contains(t, names, DefaultModelCodex)
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

	t.Run("negative value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxEffectiveTokens, "-1")
		assert.Equal(t, int64(-1), ResolveDefaultMaxEffectiveTokens(10))
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

func TestResolveDefaultMaxTurns(t *testing.T) {
	t.Run("unset uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxTurns, "")
		assert.Equal(t, "7", ResolveDefaultMaxTurns("7"))
	})

	t.Run("invalid uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxTurns, "abc")
		assert.Equal(t, "7", ResolveDefaultMaxTurns("7"))
	})

	t.Run("zero uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxTurns, "0")
		assert.Equal(t, "7", ResolveDefaultMaxTurns("7"))
	})

	t.Run("valid value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxTurns, "15")
		assert.Equal(t, "15", ResolveDefaultMaxTurns("7"))
	})
}

func TestResolveDefaultTimeoutMinutes(t *testing.T) {
	t.Run("unset uses fallback", func(t *testing.T) {
		t.Setenv(DefaultTimeoutMinutes, "")
		assert.Equal(t, 20, ResolveDefaultTimeoutMinutes(20))
	})

	t.Run("invalid uses fallback", func(t *testing.T) {
		t.Setenv(DefaultTimeoutMinutes, "abc")
		assert.Equal(t, 20, ResolveDefaultTimeoutMinutes(20))
	})

	t.Run("zero uses fallback", func(t *testing.T) {
		t.Setenv(DefaultTimeoutMinutes, "0")
		assert.Equal(t, 20, ResolveDefaultTimeoutMinutes(20))
	})

	t.Run("valid value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultTimeoutMinutes, "45")
		assert.Equal(t, 45, ResolveDefaultTimeoutMinutes(20))
	})
}

func TestResolveDefaultDetectionModel(t *testing.T) {
	t.Run("unset uses fallback", func(t *testing.T) {
		t.Setenv(DefaultDetectionModel, "")
		assert.Empty(t, ResolveDefaultDetectionModel(""))
	})

	t.Run("unset keeps non-empty fallback", func(t *testing.T) {
		t.Setenv(DefaultDetectionModel, "")
		assert.Equal(t, "gpt-5.5-mini", ResolveDefaultDetectionModel("gpt-5.5-mini"))
	})

	t.Run("set value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultDetectionModel, "gpt-5.5-mini")
		assert.Equal(t, "gpt-5.5-mini", ResolveDefaultDetectionModel(""))
	})
}
