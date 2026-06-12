package compilerenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildDefaultMaxTurnsExpression(t *testing.T) {
	assert.Equal(t,
		"${{ vars.GH_AW_DEFAULT_MAX_TURNS || '' }}",
		BuildDefaultMaxTurnsExpression(),
	)
}

func TestBuildDefaultDetectionMaxAICreditsExpression(t *testing.T) {
	assert.Equal(t,
		"${{ vars.GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS || '400' }}",
		BuildDefaultDetectionMaxAICreditsExpression("400"),
	)
}

func TestBuildDefaultMaxDailyAICreditsExpression(t *testing.T) {
	assert.Equal(t,
		"${{ vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS || '5000' }}",
		BuildDefaultMaxDailyAICreditsExpression("5000"),
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
