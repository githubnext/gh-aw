package compilerenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	t.Run("suffix value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxEffectiveTokens, "100M")
		assert.Equal(t, int64(100000000), ResolveDefaultMaxEffectiveTokens(10))
	})

	t.Run("negative value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxEffectiveTokens, "-1")
		assert.Equal(t, int64(-1), ResolveDefaultMaxEffectiveTokens(10))
	})
}

func TestResolveDefaultMaxDailyEffectiveTokens(t *testing.T) {
	t.Run("unset uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyEffectiveTokens, "")
		assert.Empty(t, ResolveDefaultMaxDailyEffectiveTokens(""))
	})

	t.Run("invalid uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyEffectiveTokens, "abc")
		assert.Equal(t, "123", ResolveDefaultMaxDailyEffectiveTokens("123"))
	})

	t.Run("zero uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyEffectiveTokens, "0")
		assert.Equal(t, "123", ResolveDefaultMaxDailyEffectiveTokens("123"))
	})

	t.Run("valid value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyEffectiveTokens, "424242")
		assert.Equal(t, "424242", ResolveDefaultMaxDailyEffectiveTokens(""))
	})

	t.Run("suffix value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyEffectiveTokens, "100M")
		assert.Equal(t, "100000000", ResolveDefaultMaxDailyEffectiveTokens(""))
	})

	t.Run("negative value disables guardrail", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyEffectiveTokens, "-1")
		assert.Equal(t, "-1", ResolveDefaultMaxDailyEffectiveTokens("123"))
	})
}

func TestResolveDefaultMaxDailyAICredits(t *testing.T) {
	t.Run("unset uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyAICredits, "")
		assert.Equal(t, "500000", ResolveDefaultMaxDailyAICredits("500000"))
	})

	t.Run("invalid uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyAICredits, "abc")
		assert.Equal(t, "500000", ResolveDefaultMaxDailyAICredits("500000"))
	})

	t.Run("zero uses fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyAICredits, "0")
		assert.Equal(t, "500000", ResolveDefaultMaxDailyAICredits("500000"))
	})

	t.Run("valid value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyAICredits, "1000000")
		assert.Equal(t, "1000000", ResolveDefaultMaxDailyAICredits("500000"))
	})

	t.Run("suffix value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyAICredits, "2M")
		assert.Equal(t, "2000000", ResolveDefaultMaxDailyAICredits("500000"))
	})

	t.Run("disables guardrail with -1", func(t *testing.T) {
		t.Setenv(DefaultMaxDailyAICredits, "-1")
		assert.Equal(t, "-1", ResolveDefaultMaxDailyAICredits("500000"))
	})
}

func TestBuildDefaultMaxTurnsExpression(t *testing.T) {
	assert.Equal(t,
		"${{ vars.GH_AW_DEFAULT_MAX_TURNS || '' }}",
		BuildDefaultMaxTurnsExpression(),
	)
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

func TestResolveDefaultUTC(t *testing.T) {
	t.Run("unset uses fallback", func(t *testing.T) {
		t.Setenv(DefaultUTC, "")
		assert.Equal(t, "+00:00", ResolveDefaultUTC("+00:00"))
	})

	t.Run("set value overrides fallback", func(t *testing.T) {
		t.Setenv(DefaultUTC, "-08:00")
		assert.Equal(t, "-08:00", ResolveDefaultUTC("+00:00"))
	})
}
