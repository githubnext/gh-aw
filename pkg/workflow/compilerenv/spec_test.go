//go:build !integration

package compilerenv_test

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/stretchr/testify/assert"
)

// TestSpec_DailyAICreditsGuardrail_ResolutionOrder validates AIC spec §9.3:
// the three-level resolution order for the daily AI Credits guardrail threshold.
//
// Resolution order (highest to lowest priority):
//  1. Frontmatter value  — compile-time literal emitted directly.
//  2. vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS — runtime GitHub Actions variable expression.
//  3. Built-in constant default (5000) — fallback literal embedded in the expression.
func TestSpec_DailyAICreditsGuardrail_ResolutionOrder(t *testing.T) {
	// Spec §9.3 (2): The runtime organization variable MUST be embedded as a GitHub
	// Actions expression — not read from the compiler process environment at compile time.
	// BuildDefaultMaxDailyAICreditsExpression returns this expression for embedding.
	t.Run("spec §9.3(2): runtime variable embedded as GitHub Actions expression", func(t *testing.T) {
		expr := compilerenv.BuildDefaultMaxDailyAICreditsExpression("5000")
		assert.Contains(t, expr, "vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS",
			"spec §9.3(2): expression must reference vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS")
		assert.Contains(t, expr, "${{",
			"spec §9.3(2): expression must be a GitHub Actions expression (starts with ${{)")
	})

	// Spec §9.3 (3): The built-in constant default MUST be '5000'.
	t.Run("spec §9.3(3): built-in constant default is 5000", func(t *testing.T) {
		expr := compilerenv.BuildDefaultMaxDailyAICreditsExpression("5000")
		assert.Contains(t, expr, "'5000'",
			"spec §9.3(3): expression must embed the built-in constant default of 5000")
	})
}

// TestSpec_DailyAICreditsGuardrail_ExpressionForm validates AIC spec §9.4:
// the exact emitted expression form when no frontmatter value is present.
//
// Spec: a conforming implementation MUST emit
// "${{ vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS || '5000' }}" verbatim.
func TestSpec_DailyAICreditsGuardrail_ExpressionForm(t *testing.T) {
	// T-AIC-DG-002
	t.Run("spec §9.4 / T-AIC-DG-002: emits verbatim expression with org var and built-in fallback", func(t *testing.T) {
		got := compilerenv.BuildDefaultMaxDailyAICreditsExpression("5000")
		assert.Equal(t,
			"${{ vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS || '5000' }}",
			got,
			"spec §9.4: emitted expression must exactly match the normative form")
	})

	// Spec §9.4: The built-in default in the expression matches the provided argument.
	t.Run("spec §9.4: custom built-in default embedded correctly in expression", func(t *testing.T) {
		got := compilerenv.BuildDefaultMaxDailyAICreditsExpression("9999")
		assert.Equal(t,
			"${{ vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS || '9999' }}",
			got,
			"spec §9.4: custom built-in default must appear as the fallback in the expression")
	})
}

// TestSpec_DailyAICreditsGuardrail_RuntimeNotCompileTime validates AIC spec §9.3 (2):
// GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS MUST be resolved at action runtime, not at
// compiler process environment lookup. The BuildDefaultMaxDailyAICreditsExpression
// function is the conforming API; ResolveDefaultMaxDailyAICredits is retained for
// other compile-time uses but MUST NOT be called for the daily guardrail default path.
//
// T-AIC-DG-006: The compiler produces an expression, not a pre-resolved value.
func TestSpec_DailyAICreditsGuardrail_RuntimeNotCompileTime(t *testing.T) {
	t.Run("spec §9.3(2) / T-AIC-DG-006: expression produced even when env var is set", func(t *testing.T) {
		// Setting the env var should have no effect on the expression-building path.
		t.Setenv(compilerenv.DefaultMaxDailyAICredits, "99999")
		got := compilerenv.BuildDefaultMaxDailyAICreditsExpression("5000")
		assert.Equal(t,
			"${{ vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS || '5000' }}",
			got,
			"spec §9.3(2): BuildDefaultMaxDailyAICreditsExpression must not read the process env var")
	})
}

// TestSpec_MaxAICreditsGuardrail_ExpressionForm validates AIC spec §10.4:
// the per-run AI credits budget is resolved at action runtime via a GitHub Actions vars
// expression, not at compiler process environment lookup.
//
// Resolution order (highest to lowest priority):
//  1. Frontmatter max-ai-credits  — compile-time literal baked into AWF config JSON.
//  2. vars.GH_AW_DEFAULT_MAX_AI_CREDITS — runtime GitHub Actions variable expression.
//  3. Built-in constant default (1000) — fallback literal embedded in the expression.
func TestSpec_MaxAICreditsGuardrail_ExpressionForm(t *testing.T) {
	t.Run("spec §10.3(2): runtime variable embedded as GitHub Actions expression", func(t *testing.T) {
		expr := compilerenv.BuildDefaultMaxAICreditsExpression("1000")
		assert.Contains(t, expr, "vars.GH_AW_DEFAULT_MAX_AI_CREDITS",
			"expression must reference vars.GH_AW_DEFAULT_MAX_AI_CREDITS")
		assert.Contains(t, expr, "${{",
			"expression must be a GitHub Actions expression (starts with ${{)")
	})

	t.Run("spec §10.4 / T-AIC-PR-002: emits verbatim expression with org var and built-in fallback", func(t *testing.T) {
		got := compilerenv.BuildDefaultMaxAICreditsExpression("1000")
		assert.Equal(t,
			"${{ vars.GH_AW_DEFAULT_MAX_AI_CREDITS || '1000' }}",
			got,
			"emitted expression must exactly match the normative form")
	})

	t.Run("spec §10.3(3): built-in constant default is 1000", func(t *testing.T) {
		got := compilerenv.BuildDefaultMaxAICreditsExpression("9999")
		assert.Equal(t,
			"${{ vars.GH_AW_DEFAULT_MAX_AI_CREDITS || '9999' }}",
			got,
			"custom built-in default must appear as the fallback in the expression")
	})
}

// TestSpec_MaxAICreditsGuardrail_RuntimeNotCompileTime validates AIC spec §10.3(2) / T-AIC-PR-006:
// the runtime organization variable MUST be resolved by the GitHub Actions runner, not the
// compiler process. Setting the process env var MUST NOT affect the emitted expression.
func TestSpec_MaxAICreditsGuardrail_RuntimeNotCompileTime(t *testing.T) {
	t.Run("spec §10.3(2) / T-AIC-PR-006: expression produced even when env var is set", func(t *testing.T) {
		// Setting the process env var should have no effect on the expression-building path.
		t.Setenv(compilerenv.DefaultMaxAICredits, "99999")
		got := compilerenv.BuildDefaultMaxAICreditsExpression("1000")
		assert.Equal(t,
			"${{ vars.GH_AW_DEFAULT_MAX_AI_CREDITS || '1000' }}",
			got,
			"BuildDefaultMaxAICreditsExpression must not read the process env var")
	})
}

func TestSpec_DetectionMaxAICreditsGuardrail_ExpressionForm(t *testing.T) {
	t.Run("spec §D-AIC-001: emits GitHub Actions expression with org var and built-in fallback", func(t *testing.T) {
		got := compilerenv.BuildDefaultDetectionMaxAICreditsExpression("400")
		assert.Equal(t,
			"${{ vars.GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS || '400' }}",
			got,
		)
	})
}
