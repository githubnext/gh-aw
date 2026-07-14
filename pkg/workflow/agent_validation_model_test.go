//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUniversalLLMConsumerModel(t *testing.T) {
	compiler := NewCompiler()
	opencodeEngine, err := newBuiltinBehaviorDefinedEngine("opencode")
	require.NoError(t, err)
	crushEngine, err := newBuiltinBehaviorDefinedEngine("crush")
	require.NoError(t, err)

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
			opencodeEngine,
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
			crushEngine,
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
			opencodeEngine,
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
			crushEngine,
		)
		assert.NoError(t, err, "Supported provider/model should pass")
	})
}

func TestValidatePiEngineRequirements(t *testing.T) {
	compiler := NewCompiler()

	t.Run("non pi engine skips validation", func(t *testing.T) {
		err := compiler.validatePiEngineRequirements(NewTools(map[string]any{}), NewCopilotEngine())
		assert.NoError(t, err)
	})

	t.Run("pi requires github gh-proxy mode", func(t *testing.T) {
		err := compiler.validatePiEngineRequirements(NewTools(map[string]any{
			"github": true,
		}), NewPiEngine())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tools.github.mode: gh-proxy")
	})

	t.Run("pi requires cli-proxy", func(t *testing.T) {
		err := compiler.validatePiEngineRequirements(NewTools(map[string]any{
			"github": map[string]any{"mode": "gh-proxy"},
		}), NewPiEngine())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tools.cli-proxy: true")
	})

	t.Run("valid pi tool config passes", func(t *testing.T) {
		err := compiler.validatePiEngineRequirements(NewTools(map[string]any{
			"github":    map[string]any{"mode": "gh-proxy"},
			"cli-proxy": true,
		}), NewPiEngine())
		assert.NoError(t, err)
	})
}
