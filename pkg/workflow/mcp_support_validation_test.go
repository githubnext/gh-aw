//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMCPLessEngine(t *testing.T) *BehaviorDefinedEngine {
	t.Helper()
	engine, err := NewBehaviorDefinedEngine(&EngineDefinition{
		ID:          "aider",
		DisplayName: "Aider",
		Behaviors: &EngineBehaviorDefinition{
			MCP: &EngineMCPDefinition{Unsupported: true},
		},
	})
	require.NoError(t, err)
	return engine
}

func TestValidateMCPSupport(t *testing.T) {
	compiler := NewCompiler()
	mcpLessEngine := newMCPLessEngine(t)

	t.Run("engine with MCP support is not restricted", func(t *testing.T) {
		err := compiler.validateMCPSupport(map[string]any{"github": nil}, NewCopilotEngine())
		assert.NoError(t, err)
	})

	t.Run("MCP-less engine allows non-MCP tools", func(t *testing.T) {
		err := compiler.validateMCPSupport(map[string]any{
			"bash": []any{"*"},
			"edit": nil,
		}, mcpLessEngine)
		assert.NoError(t, err)
	})

	t.Run("MCP-less engine rejects github tool", func(t *testing.T) {
		err := compiler.validateMCPSupport(map[string]any{"github": nil}, mcpLessEngine)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "engine 'aider' does not support MCP servers")
		assert.Contains(t, err.Error(), "github")
	})

	t.Run("MCP-less engine rejects custom MCP server", func(t *testing.T) {
		err := compiler.validateMCPSupport(map[string]any{
			"custom": map[string]any{
				"url": "https://example.com/mcp",
			},
		}, mcpLessEngine)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "custom")
	})

	t.Run("MCP-less engine ignores disabled tools", func(t *testing.T) {
		err := compiler.validateMCPSupport(map[string]any{"github": false}, mcpLessEngine)
		assert.NoError(t, err)
	})
}

func TestIsMCPUnsupported(t *testing.T) {
	assert.True(t, engineDisallowsMCP(newMCPLessEngine(t)))
	assert.False(t, engineDisallowsMCP(NewCopilotEngine()))
}
