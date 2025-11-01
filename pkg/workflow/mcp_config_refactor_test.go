package workflow

import (
	"strings"
	"testing"
)

// TestRenderPlaywrightMCPConfigWithOptions verifies the shared Playwright config helper
// works correctly with both Copilot and non-Copilot engines
func TestRenderPlaywrightMCPConfigWithOptions(t *testing.T) {
	tests := []struct {
		name                 string
		playwrightTool       any
		isLast               bool
		includeCopilotFields bool
		inlineArgs           bool
		expectedContent      []string
		unexpectedContent    []string
	}{
		{
			name: "Copilot with inline args and type/tools fields",
			playwrightTool: map[string]any{
				"version": "v1.45.0",
			},
			isLast:               true,
			includeCopilotFields: true,
			inlineArgs:           true,
			expectedContent: []string{
				`"playwright": {`,
				`"type": "local"`,
				`"command": "npx"`,
				`"args": ["@playwright/mcp@v1.45.0"`,
				`"--output-dir", "/tmp/gh-aw/mcp-logs/playwright"`,
				`"tools": ["*"]`,
				`              }`,
			},
			unexpectedContent: []string{},
		},
		{
			name: "Claude/Custom with multi-line args, no type/tools fields",
			playwrightTool: map[string]any{
				"allowed": []string{"browser_click", "browser_navigate"},
			},
			isLast:               false,
			includeCopilotFields: false,
			inlineArgs:           false,
			expectedContent: []string{
				`"playwright": {`,
				`"command": "npx"`,
				`"args": [`,
				`"@playwright/mcp@latest"`,
				`"--output-dir"`,
				`"/tmp/gh-aw/mcp-logs/playwright"`,
				`              },`,
			},
			unexpectedContent: []string{
				`"type"`,
				`"tools"`,
			},
		},
		{
			name: "Copilot with allowed domains",
			playwrightTool: map[string]any{
				"network": map[string]any{
					"allowed": []string{"example.com", "test.com"},
				},
			},
			isLast:               false,
			includeCopilotFields: true,
			inlineArgs:           true,
			expectedContent: []string{
				`"--allowed-origins"`,
				`"localhost;localhost:*;127.0.0.1;127.0.0.1:*"`, // Default localhost is always added
			},
			unexpectedContent: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			renderPlaywrightMCPConfigWithOptions(&output, tt.playwrightTool, tt.isLast, tt.includeCopilotFields, tt.inlineArgs)

			result := output.String()

			// Check expected content
			for _, expected := range tt.expectedContent {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected content not found: %q\nActual output:\n%s", expected, result)
				}
			}

			// Check unexpected content
			for _, unexpected := range tt.unexpectedContent {
				if strings.Contains(result, unexpected) {
					t.Errorf("Unexpected content found: %q\nActual output:\n%s", unexpected, result)
				}
			}
		})
	}
}

// TestRenderSafeOutputsMCPConfigWithOptions verifies the shared Safe Outputs config helper
// works correctly with both Copilot and non-Copilot engines
func TestRenderSafeOutputsMCPConfigWithOptions(t *testing.T) {
	tests := []struct {
		name                 string
		isLast               bool
		includeCopilotFields bool
		expectedContent      []string
		unexpectedContent    []string
	}{
		{
			name:                 "Copilot with type/tools and escaped env vars",
			isLast:               true,
			includeCopilotFields: true,
			expectedContent: []string{
				`"safeoutputs": {`,
				`"type": "local"`,
				`"command": "node"`,
				`"args": ["/tmp/gh-aw/safeoutputs/mcp-server.cjs"]`,
				`"tools": ["*"]`,
				`"env": {`,
				`"GH_AW_SAFE_OUTPUTS": "\${GH_AW_SAFE_OUTPUTS}"`,
				`"GH_AW_SAFE_OUTPUTS_CONFIG": "\${GH_AW_SAFE_OUTPUTS_CONFIG}"`,
				`              }`,
			},
			unexpectedContent: []string{
				`${{ env.`,
			},
		},
		{
			name:                 "Claude/Custom without type/tools, with shell env vars",
			isLast:               false,
			includeCopilotFields: false,
			expectedContent: []string{
				`"safeoutputs": {`,
				`"command": "node"`,
				`"args": ["/tmp/gh-aw/safeoutputs/mcp-server.cjs"]`,
				// Security fix: Now uses shell variables instead of GitHub expressions
				`"GH_AW_SAFE_OUTPUTS": "$GH_AW_SAFE_OUTPUTS"`,
				`"GH_AW_SAFE_OUTPUTS_CONFIG": $GH_AW_SAFE_OUTPUTS_CONFIG`,
				`              },`,
			},
			unexpectedContent: []string{
				`"type"`,
				`"tools"`,
				`\\${`,
				// Verify GitHub expressions are NOT in the output (security fix)
				`${{ env.`,
				`${{ toJSON(`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			renderSafeOutputsMCPConfigWithOptions(&output, tt.isLast, tt.includeCopilotFields)

			result := output.String()

			// Check expected content
			for _, expected := range tt.expectedContent {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected content not found: %q\nActual output:\n%s", expected, result)
				}
			}

			// Check unexpected content
			for _, unexpected := range tt.unexpectedContent {
				if strings.Contains(result, unexpected) {
					t.Errorf("Unexpected content found: %q\nActual output:\n%s", unexpected, result)
				}
			}
		})
	}
}

// TestRenderAgenticWorkflowsMCPConfigWithOptions verifies the shared Agentic Workflows config helper
// works correctly with both Copilot and non-Copilot engines
func TestRenderAgenticWorkflowsMCPConfigWithOptions(t *testing.T) {
	tests := []struct {
		name                 string
		isLast               bool
		includeCopilotFields bool
		expectedContent      []string
		unexpectedContent    []string
	}{
		{
			name:                 "Copilot with type/tools and escaped env vars",
			isLast:               false,
			includeCopilotFields: true,
			expectedContent: []string{
				`"agentic_workflows": {`,
				`"type": "local"`,
				`"command": "gh"`,
				`"args": ["aw", "mcp-server"]`,
				`"tools": ["*"]`,
				`"GITHUB_TOKEN": "\${GITHUB_TOKEN}"`,
				`              },`,
			},
			unexpectedContent: []string{
				`${{ secrets.`,
			},
		},
		{
			name:                 "Claude/Custom without type/tools, with shell env vars",
			isLast:               true,
			includeCopilotFields: false,
			expectedContent: []string{
				`"agentic_workflows": {`,
				`"command": "gh"`,
				`"args": ["aw", "mcp-server"]`,
				// Security fix: Now uses shell variable instead of GitHub secret expression
				`"GITHUB_TOKEN": "$GITHUB_TOKEN"`,
				`              }`,
			},
			unexpectedContent: []string{
				`"type"`,
				`"tools"`,
				`\\${`,
				// Verify GitHub expressions are NOT in the output (security fix)
				`${{ secrets.`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			renderAgenticWorkflowsMCPConfigWithOptions(&output, tt.isLast, tt.includeCopilotFields)

			result := output.String()

			// Check expected content
			for _, expected := range tt.expectedContent {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected content not found: %q\nActual output:\n%s", expected, result)
				}
			}

			// Check unexpected content
			for _, unexpected := range tt.unexpectedContent {
				if strings.Contains(result, unexpected) {
					t.Errorf("Unexpected content found: %q\nActual output:\n%s", unexpected, result)
				}
			}
		})
	}
}

// TestRenderPlaywrightMCPConfigTOML verifies the TOML format helper for Codex engine
func TestRenderPlaywrightMCPConfigTOML(t *testing.T) {
	tests := []struct {
		name            string
		playwrightTool  any
		expectedContent []string
	}{
		{
			name: "Basic Playwright TOML config",
			playwrightTool: map[string]any{
				"allowed": []string{"browser_click"},
			},
			expectedContent: []string{
				`[mcp_servers.playwright]`,
				`command = "npx"`,
				`args = [`,
				`"@playwright/mcp@latest"`,
				`"--output-dir"`,
				`"/tmp/gh-aw/mcp-logs/playwright"`,
			},
		},
		{
			name: "Playwright TOML with allowed domains",
			playwrightTool: map[string]any{
				"network": map[string]any{
					"allowed": []string{"example.com"},
				},
			},
			expectedContent: []string{
				`"--allowed-origins"`,
				`"localhost;localhost:*;127.0.0.1;127.0.0.1:*"`, // Default localhost is added
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			renderPlaywrightMCPConfigTOML(&output, tt.playwrightTool)

			result := output.String()

			// Check expected content
			for _, expected := range tt.expectedContent {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected content not found: %q\nActual output:\n%s", expected, result)
				}
			}
		})
	}
}

// TestRenderSafeOutputsMCPConfigTOML verifies the Safe Outputs TOML format helper
func TestRenderSafeOutputsMCPConfigTOML(t *testing.T) {
	var output strings.Builder

	renderSafeOutputsMCPConfigTOML(&output)

	result := output.String()

	expectedContent := []string{
		`[mcp_servers.safeoutputs]`,
		`command = "node"`,
		`args = [`,
		`"/tmp/gh-aw/safeoutputs/mcp-server.cjs"`,
		`env = {`,
		`"GH_AW_SAFE_OUTPUTS" = "${{ env.GH_AW_SAFE_OUTPUTS }}"`,
		`"GH_AW_SAFE_OUTPUTS_CONFIG" = "${{ toJSON(env.GH_AW_SAFE_OUTPUTS_CONFIG) }}"`,
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected content not found: %q\nActual output:\n%s", expected, result)
		}
	}
}

// TestRenderAgenticWorkflowsMCPConfigTOML verifies the Agentic Workflows TOML format helper
func TestRenderAgenticWorkflowsMCPConfigTOML(t *testing.T) {
	var output strings.Builder

	renderAgenticWorkflowsMCPConfigTOML(&output)

	result := output.String()

	expectedContent := []string{
		`[mcp_servers.agentic_workflows]`,
		`command = "gh"`,
		`args = [`,
		`"aw"`,
		`"mcp-server"`,
		`env = { "GITHUB_TOKEN" = "${{ secrets.GITHUB_TOKEN }}" }`,
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected content not found: %q\nActual output:\n%s", expected, result)
		}
	}
}
