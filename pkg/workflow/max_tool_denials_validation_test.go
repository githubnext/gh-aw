//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateMaxToolDenialsSupport(t *testing.T) {
	t.Parallel()

	compiler := NewCompiler()
	registry := GetGlobalEngineRegistry()

	copilotEngine, err := registry.GetEngine("copilot")
	require.NoError(t, err)

	claudeEngine, err := registry.GetEngine("claude")
	require.NoError(t, err)

	tests := []struct {
		name        string
		frontmatter map[string]any
		engine      CodingAgentEngine
		expectError string
	}{
		{
			name: "no max-tool-denials",
			frontmatter: map[string]any{
				"engine": "claude",
			},
			engine: claudeEngine,
		},
		{
			name: "copilot sdk with max-tool-denials",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id":          "copilot",
					"copilot-sdk": true,
				},
				"max-tool-denials": 6,
			},
			engine: copilotEngine,
		},
		{
			name: "copilot without sdk rejects max-tool-denials",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
				},
				"max-tool-denials": 6,
			},
			engine:      copilotEngine,
			expectError: "requires Copilot SDK mode",
		},
		{
			name: "non-copilot rejects max-tool-denials",
			frontmatter: map[string]any{
				"engine":           "claude",
				"max-tool-denials": 6,
			},
			engine:      claudeEngine,
			expectError: "does not support max-tool-denials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := compiler.validateMaxToolDenialsSupport(tt.frontmatter, tt.engine)
			if tt.expectError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.expectError)
		})
	}
}
