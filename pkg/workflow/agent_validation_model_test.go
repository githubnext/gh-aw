//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUniversalLLMConsumerModel(t *testing.T) {
	compiler := NewCompiler()

	t.Run("non universal engine skips validation", func(t *testing.T) {
		err := compiler.validateUniversalLLMConsumerModel(
			map[string]any{
				"engine": map[string]any{
					"id": "copilot",
				},
			},
			NewCopilotEngine(),
		)
		assert.NoError(t, err, "Non-universal engines should skip model validation")
	})

	t.Run("opencode requires model", func(t *testing.T) {
		err := compiler.validateUniversalLLMConsumerModel(
			map[string]any{
				"engine": map[string]any{
					"id": "opencode",
				},
			},
			NewOpenCodeEngine(),
		)
		require.Error(t, err, "Missing model should fail for opencode")
		assert.Contains(t, err.Error(), "engine.model is required for engine 'opencode'")
	})

	t.Run("crush requires provider/model format", func(t *testing.T) {
		err := compiler.validateUniversalLLMConsumerModel(
			map[string]any{
				"engine": map[string]any{
					"id":    "crush",
					"model": "gpt-4.1",
				},
			},
			NewCrushEngine(),
		)
		require.Error(t, err, "Unqualified model should fail for crush")
		assert.Contains(t, err.Error(), "provider/model format")
	})

	t.Run("unsupported provider fails", func(t *testing.T) {
		err := compiler.validateUniversalLLMConsumerModel(
			map[string]any{
				"engine": map[string]any{
					"id":    "opencode",
					"model": "groq/llama-4",
				},
			},
			NewOpenCodeEngine(),
		)
		require.Error(t, err, "Unsupported provider should fail")
		assert.Contains(t, err.Error(), "unsupported provider")
	})

	t.Run("supported provider passes", func(t *testing.T) {
		err := compiler.validateUniversalLLMConsumerModel(
			map[string]any{
				"engine": map[string]any{
					"id":    "crush",
					"model": "anthropic/claude-sonnet-4",
				},
			},
			NewCrushEngine(),
		)
		assert.NoError(t, err, "Supported provider/model should pass")
	})
}
