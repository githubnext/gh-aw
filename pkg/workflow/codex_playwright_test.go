//go:build !integration

package workflow

import "testing"

func TestCodexEnginePlaywrightUsesCLI(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name  string
		input map[string]any
	}{
		{
			name:  "playwright null does not create MCP config",
			input: map[string]any{"playwright": nil},
		},
		{
			name: "playwright CLI config does not create MCP config",
			input: map[string]any{
				"playwright": map[string]any{
					"mode":    "cli",
					"version": "0.1.18",
				},
			},
		},
		{
			name: "legacy MCP config does not create MCP config",
			input: map[string]any{
				"playwright": map[string]any{"mode": "mcp"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.expandNeutralToolsToCodexToolsFromMap(tt.input)
			if playwrightRaw, hasPlaywright := result["playwright"]; hasPlaywright {
				t.Errorf("expected playwright to be absent from MCP config, got: %v", playwrightRaw)
			}
		})
	}
}
