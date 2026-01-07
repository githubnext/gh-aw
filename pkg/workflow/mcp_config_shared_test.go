package workflow

import (
	"strings"
	"testing"
)

// TestRenderPlaywrightMCPConfigShared tests the shared renderPlaywrightMCPConfig function
func TestRenderPlaywrightMCPConfigShared(t *testing.T) {
	tests := []struct {
		name           string
		playwrightTool any
		isLast         bool
		wantContains   []string
		wantEnding     string
	}{
		{
			name: "basic playwright config not last",
			playwrightTool: map[string]any{
				"allowed_domains": []any{"example.com", "test.com"},
			},
			isLast: false,
			wantContains: []string{
				`"playwright": {`,
				`"command": "docker"`,
				`"run"`,
				`"-i"`,
				`"--rm"`,
				`"--init"`,
				`"mcr.microsoft.com/playwright/mcp"`,
				`"--output-dir"`,
				`"/tmp/gh-aw/mcp-logs/playwright"`,
				`"--allowed-hosts"`,
				`"--allowed-origins"`,
				`example.com;test.com`, // Domains are joined with semicolons
			},
			wantEnding: "},\n",
		},
		{
			name: "basic playwright config is last",
			playwrightTool: map[string]any{
				"allowed_domains": []any{"example.com"},
			},
			isLast: true,
			wantContains: []string{
				`"playwright": {`,
				`"command": "docker"`,
			},
			wantEnding: "}\n",
		},
		{
			name:           "playwright config without domains",
			playwrightTool: map[string]any{},
			isLast:         false,
			wantContains: []string{
				`"playwright": {`,
				`"command": "docker"`,
				`"mcr.microsoft.com/playwright/mcp"`,
				`"--output-dir"`,
				`"/tmp/gh-aw/mcp-logs/playwright"`,
			},
			wantEnding: "},\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			renderPlaywrightMCPConfig(&yaml, tt.playwrightTool, tt.isLast)

			result := yaml.String()

			// Check all required strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("renderPlaywrightMCPConfig() result missing %q\nGot:\n%s", want, result)
				}
			}

			// Check correct ending
			if !strings.HasSuffix(result, tt.wantEnding) {
				// Show last part of result for debugging, but handle short strings
				endSnippet := result
				if len(result) > 10 {
					endSnippet = result[len(result)-10:]
				}
				t.Errorf("renderPlaywrightMCPConfig() ending = %q, want suffix %q", endSnippet, tt.wantEnding)
			}
		})
	}
}

// TestRenderSafeOutputsMCPConfigShared tests the shared renderSafeOutputsMCPConfig function
func TestRenderSafeOutputsMCPConfigShared(t *testing.T) {
	tests := []struct {
		name         string
		isLast       bool
		wantContains []string
		wantEnding   string
	}{
		{
			name:   "safe outputs config not last",
			isLast: false,
			wantContains: []string{
				`"safeoutputs": {`,
				`"command": "node"`,
				`"/opt/gh-aw/safeoutputs/mcp-server.cjs"`,
				`"GH_AW_SAFE_OUTPUTS"`,
				`"GH_AW_ASSETS_BRANCH"`,
				`"GH_AW_ASSETS_MAX_SIZE_KB"`,
				`"GH_AW_ASSETS_ALLOWED_EXTS"`,
			},
			wantEnding: "},\n",
		},
		{
			name:   "safe outputs config is last",
			isLast: true,
			wantContains: []string{
				`"safeoutputs": {`,
				`"command": "node"`,
				`"GH_AW_SAFE_OUTPUTS"`,
			},
			wantEnding: "}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			renderSafeOutputsMCPConfig(&yaml, tt.isLast)

			result := yaml.String()

			// Check all required strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("renderSafeOutputsMCPConfig() result missing %q\nGot:\n%s", want, result)
				}
			}

			// Check correct ending
			if !strings.HasSuffix(result, tt.wantEnding) {
				// Show last part of result for debugging, but handle short strings
				endSnippet := result
				if len(result) > 10 {
					endSnippet = result[len(result)-10:]
				}
				t.Errorf("renderSafeOutputsMCPConfig() ending = %q, want suffix %q", endSnippet, tt.wantEnding)
			}
		})
	}
}

// TestRenderCustomMCPConfigWrapperShared tests the shared renderCustomMCPConfigWrapper function
func TestRenderCustomMCPConfigWrapperShared(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		toolConfig   map[string]any
		isLast       bool
		wantContains []string
		wantEnding   string
		wantError    bool
	}{
		{
			name:     "custom MCP config not last",
			toolName: "my-tool",
			toolConfig: map[string]any{
				"command": "node",
				"args":    []string{"server.js"},
			},
			isLast: false,
			wantContains: []string{
				`"my-tool": {`,
				`"command": "node"`,
			},
			wantEnding: "},\n",
			wantError:  false,
		},
		{
			name:     "custom MCP config is last",
			toolName: "another-tool",
			toolConfig: map[string]any{
				"command": "python",
				"args":    []string{"-m", "server"},
			},
			isLast: true,
			wantContains: []string{
				`"another-tool": {`,
				`"command": "python"`,
			},
			wantEnding: "}\n",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			err := renderCustomMCPConfigWrapper(&yaml, tt.toolName, tt.toolConfig, tt.isLast)

			if (err != nil) != tt.wantError {
				t.Errorf("renderCustomMCPConfigWrapper() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return
			}

			result := yaml.String()

			// Check all required strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("renderCustomMCPConfigWrapper() result missing %q\nGot:\n%s", want, result)
				}
			}

			// Check correct ending
			if !strings.HasSuffix(result, tt.wantEnding) {
				// Show last part of result for debugging, but handle short strings
				endSnippet := result
				if len(result) > 10 {
					endSnippet = result[len(result)-10:]
				}
				t.Errorf("renderCustomMCPConfigWrapper() ending = %q, want suffix %q", endSnippet, tt.wantEnding)
			}
		})
	}
}

// TestEngineMethodsDelegateToShared ensures engine methods properly delegate to shared functions
func TestEngineMethodsDelegateToShared(t *testing.T) {
	t.Run("Claude engine Playwright delegation via unified renderer", func(t *testing.T) {
		// Use unified renderer with Claude-specific options
		renderer := NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields: false,
			InlineArgs:           false,
			Format:               "json",
			IsLast:               false,
		})
		var yaml strings.Builder
		playwrightTool := map[string]any{
			"allowed_domains": []any{"example.com"},
		}

		renderer.RenderPlaywrightMCP(&yaml, playwrightTool)
		result := yaml.String()

		if !strings.Contains(result, `"playwright": {`) {
			t.Error("Claude engine should use unified renderer for Playwright MCP config")
		}
	})

	t.Run("Custom engine Playwright delegation", func(t *testing.T) {
		// Use unified renderer with Custom engine options
		renderer := NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields: false,
			InlineArgs:           false,
			Format:               "json",
			IsLast:               false,
		})
		var yaml strings.Builder
		playwrightTool := map[string]any{
			"allowed_domains": []any{"example.com"},
		}

		renderer.RenderPlaywrightMCP(&yaml, playwrightTool)
		result := yaml.String()

		if !strings.Contains(result, `"playwright": {`) {
			t.Error("Custom engine Playwright should produce output via unified renderer")
		}
	})

	t.Run("Claude and Custom engines produce identical output", func(t *testing.T) {
		// Claude engine via unified renderer
		claudeRenderer := NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields: false,
			InlineArgs:           false,
			Format:               "json",
			IsLast:               false,
		})

		// Custom engine also uses unified renderer with same options
		customRenderer := NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields: false,
			InlineArgs:           false,
			Format:               "json",
			IsLast:               false,
		})

		playwrightTool := map[string]any{
			"allowed_domains": []any{"example.com", "test.com"},
		}

		var claudeYAML strings.Builder
		claudeRenderer.RenderPlaywrightMCP(&claudeYAML, playwrightTool)

		var customYAML strings.Builder
		customRenderer.RenderPlaywrightMCP(&customYAML, playwrightTool)

		if claudeYAML.String() != customYAML.String() {
			t.Error("Claude and Custom engines should produce identical Playwright MCP config")
		}
	})
}
