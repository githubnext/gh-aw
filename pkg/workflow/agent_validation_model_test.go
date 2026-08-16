//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUniversalLLMConsumerModel(t *testing.T) {
	compiler := NewCompiler()
	opencodeEngine, err := NewBehaviorDefinedEngine(&EngineDefinition{
		ID:          "opencode",
		DisplayName: "OpenCode",
		Behaviors: &EngineBehaviorDefinition{
			SecretStrategy: behaviorSecretStrategyUniversalLLMConsumer,
		},
	})
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
		require.ErrorContains(t, err, "engine.model is required for engine 'opencode'")
	})

	t.Run("opencode requires provider/model format", func(t *testing.T) {
		err := compiler.validateUniversalLLMConsumerModel(
			map[string]any{
				"engine": map[string]any{
					"id":    "opencode",
					"model": "gpt-4.1",
				},
			},
			opencodeEngine,
		)
		require.Error(t, err, "Unqualified model should fail for opencode")
		require.ErrorContains(t, err, "provider/model format")
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
		require.ErrorContains(t, err, "unsupported provider")
	})

	t.Run("supported provider passes", func(t *testing.T) {
		err := compiler.validateUniversalLLMConsumerModel(
			map[string]any{
				"engine": map[string]any{
					"id":    "opencode",
					"model": "anthropic/claude-sonnet-4",
				},
			},
			opencodeEngine,
		)
		assert.NoError(t, err, "Supported provider/model should pass")
	})
}

func TestPiEngineAutoDerivesCLIAccess(t *testing.T) {
	piEngine := NewPiEngine()

	t.Run("pi with bare github derives cli mode and mcp-mode cli", func(t *testing.T) {
		tools, err := enforceMCPProxyTools(piEngine, map[string]any{"github": true})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"mode": "cli"}, tools["github"])
		assert.Equal(t, "cli", tools["mcp-mode"])
		assert.NotContains(t, tools, "cli-proxy")
	})

	t.Run("pi with no github tool derives cli github and mcp-mode cli", func(t *testing.T) {
		tools, err := enforceMCPProxyTools(piEngine, map[string]any{})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"mode": "cli"}, tools["github"])
		assert.Equal(t, "cli", tools["mcp-mode"])
	})

	t.Run("pi rejects explicit mcp mode", func(t *testing.T) {
		_, err := enforceMCPProxyTools(piEngine, map[string]any{
			"github": map[string]any{"mode": "mcp-local"},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "tools.github.mode must be cli")
	})

	t.Run("pi ignores MCP-only fields (auto-derives cli)", func(t *testing.T) {
		tools, err := enforceMCPProxyTools(piEngine, map[string]any{
			"github": map[string]any{"toolsets": []any{"default"}},
		})
		require.NoError(t, err)
		github := tools["github"].(map[string]any)
		assert.Equal(t, "cli", github["mode"])
		assert.Equal(t, "cli", tools["mcp-mode"])
	})

	t.Run("pi rejects disabling cli-proxy", func(t *testing.T) {
		_, err := enforceMCPProxyTools(piEngine, map[string]any{"cli-proxy": false})
		require.Error(t, err)
		require.ErrorContains(t, err, "tools.cli-proxy cannot be disabled")
	})

	t.Run("copilot (MCP engine) is unchanged", func(t *testing.T) {
		in := map[string]any{"github": map[string]any{"toolsets": []any{"default"}}}
		tools, err := enforceMCPProxyTools(NewCopilotEngine(), in)
		require.NoError(t, err)
		assert.Equal(t, in, tools)
	})
}

func TestValidateGitHubAccessConfig(t *testing.T) {
	compiler := NewCompiler()

	t.Run("explicit cli with toolsets warns but does not error", func(t *testing.T) {
		err := compiler.validateGitHubAccessConfig(map[string]any{}, map[string]any{
			"github": map[string]any{"mode": "cli", "toolsets": []any{"default"}},
		}, nil)
		assert.NoError(t, err)
	})

	t.Run("explicit cli without MCP-only fields is allowed", func(t *testing.T) {
		err := compiler.validateGitHubAccessConfig(map[string]any{}, map[string]any{
			"github": map[string]any{"mode": "cli", "min-integrity": "approved"},
		}, nil)
		assert.NoError(t, err)
	})

	t.Run("integrity-reactions with explicit mcp mode is rejected", func(t *testing.T) {
		err := compiler.validateGitHubAccessConfig(
			map[string]any{"features": map[string]any{"integrity-reactions": true}},
			map[string]any{"github": map[string]any{"mode": "mcp-remote"}},
			nil,
		)
		require.Error(t, err)
		require.ErrorContains(t, err, "features.integrity-reactions requires CLI")
	})

	t.Run("both cli-proxy and mcp-mode is rejected", func(t *testing.T) {
		err := compiler.validateGitHubAccessConfig(map[string]any{}, map[string]any{
			"cli-proxy": true,
			"mcp-mode":  "cli",
		}, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, "cannot both be set")
	})

	t.Run("mcp-local with toolsets is allowed", func(t *testing.T) {
		err := compiler.validateGitHubAccessConfig(map[string]any{}, map[string]any{
			"github": map[string]any{"mode": "mcp-local", "toolsets": []any{"default"}},
		}, nil)
		assert.NoError(t, err)
	})

	t.Run("derived cli for non-MCP engine does not warn about MCP-only fields", func(t *testing.T) {
		before := compiler.GetWarningCount()
		err := compiler.validateGitHubAccessConfig(map[string]any{}, map[string]any{
			"github": map[string]any{"mode": "cli", "toolsets": []any{"default"}},
		}, NewPiEngine())
		require.NoError(t, err)
		assert.Equal(t, before, compiler.GetWarningCount())
	})
}
