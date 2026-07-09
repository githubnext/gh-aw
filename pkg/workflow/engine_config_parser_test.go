package workflow

import "testing"

func TestParseIntOrExpressionWrappers(t *testing.T) {
	t.Run("parseMaxTurnsValue keeps expressions and rejects zero", func(t *testing.T) {
		if got := parseMaxTurnsValue("${{ inputs.max_turns }}"); got != "${{ inputs.max_turns }}" {
			t.Fatalf("expected expression passthrough, got %q", got)
		}
		if got := parseMaxTurnsValue(0); got != "" {
			t.Fatalf("expected zero to be rejected, got %q", got)
		}
	})

	t.Run("parseNonNegativeIntOrExpressionValue accepts zero", func(t *testing.T) {
		if got := parseNonNegativeIntOrExpressionValue(0); got != "0" {
			t.Fatalf("expected zero to be accepted, got %q", got)
		}
	})

	t.Run("parseMaxToolDenialsValue keeps positive integers", func(t *testing.T) {
		if got := parseMaxToolDenialsValue("2"); got != "2" {
			t.Fatalf("expected positive numeric string to pass, got %q", got)
		}
		if got := parseMaxToolDenialsValue("${{ inputs.denials }}"); got != "${{ inputs.denials }}" {
			t.Fatalf("expected expression passthrough, got %q", got)
		}
	})
}
