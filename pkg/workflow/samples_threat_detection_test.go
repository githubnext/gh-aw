package workflow

import "testing"

// TestExtractSafeOutputsConfig_UseSamplesDisablesThreatDetection verifies
// that --use-samples force-disables threat detection so the deterministic
// replay isn't perturbed by an LLM-backed detection job.
func TestExtractSafeOutputsConfig_UseSamplesDisablesThreatDetection(t *testing.T) {
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"samples": []any{
					map[string]any{"title": "x", "body": "y"},
				},
			},
		},
	}

	t.Run("default mode applies threat-detection", func(t *testing.T) {
		c := NewCompiler()
		cfg := c.extractSafeOutputsConfig(frontmatter)
		if cfg == nil {
			t.Fatal("expected non-nil SafeOutputsConfig")
		}
		if cfg.ThreatDetection == nil {
			t.Fatal("expected default threat-detection to be applied in default mode")
		}
	})

	t.Run("use-samples disables threat-detection (default)", func(t *testing.T) {
		c := NewCompiler()
		c.SetUseSamples(true)
		cfg := c.extractSafeOutputsConfig(frontmatter)
		if cfg == nil {
			t.Fatal("expected non-nil SafeOutputsConfig")
		}
		if cfg.ThreatDetection != nil {
			t.Fatal("expected threat-detection to be force-disabled under --use-samples")
		}
	})

	t.Run("use-samples disables threat-detection (explicit true)", func(t *testing.T) {
		fm := map[string]any{
			"safe-outputs": map[string]any{
				"threat-detection": true,
				"create-issue": map[string]any{
					"samples": []any{
						map[string]any{"title": "x", "body": "y"},
					},
				},
			},
		}
		c := NewCompiler()
		c.SetUseSamples(true)
		cfg := c.extractSafeOutputsConfig(fm)
		if cfg == nil {
			t.Fatal("expected non-nil SafeOutputsConfig")
		}
		if cfg.ThreatDetection != nil {
			t.Fatal("expected explicit threat-detection: true to be force-disabled under --use-samples")
		}
	})
}
