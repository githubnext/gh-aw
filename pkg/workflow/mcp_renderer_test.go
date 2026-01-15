package workflow

import (
	"strings"
	"testing"
)

func TestNewMCPConfigRenderer(t *testing.T) {
	tests := []struct {
		name    string
		options MCPRendererOptions
	}{
		{
			name: "copilot options",
			options: MCPRendererOptions{
				IncludeCopilotFields: true,
				InlineArgs:           true,
				Format:               "json",
				IsLast:               false,
			},
		},
		{
			name: "claude options",
			options: MCPRendererOptions{
				IncludeCopilotFields: false,
				InlineArgs:           false,
				Format:               "json",
				IsLast:               true,
			},
		},
		{
			name: "codex options",
			options: MCPRendererOptions{
				IncludeCopilotFields: false,
				InlineArgs:           false,
				Format:               "toml",
				IsLast:               false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewMCPConfigRenderer(tt.options)
			if renderer == nil {
				t.Fatal("Expected non-nil renderer")
			}
			if renderer.options.Format != tt.options.Format {
				t.Errorf("Expected format %s, got %s", tt.options.Format, renderer.options.Format)
			}
			if renderer.options.IncludeCopilotFields != tt.options.IncludeCopilotFields {
				t.Errorf("Expected IncludeCopilotFields %t, got %t", tt.options.IncludeCopilotFields, renderer.options.IncludeCopilotFields)
			}
			if renderer.options.InlineArgs != tt.options.InlineArgs {
				t.Errorf("Expected InlineArgs %t, got %t", tt.options.InlineArgs, renderer.options.InlineArgs)
			}
			if renderer.options.IsLast != tt.options.IsLast {
				t.Errorf("Expected IsLast %t, got %t", tt.options.IsLast, renderer.options.IsLast)
			}
		})
	}
}

func TestRenderPlaywrightMCP_JSON_Copilot(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: true,
		InlineArgs:           true,
		Format:               "json",
		IsLast:               false,
	})

	playwrightTool := map[string]any{
		"allowed-domains": []string{"example.com"},
	}

	var yaml strings.Builder
	renderer.RenderPlaywrightMCP(&yaml, playwrightTool)

	output := yaml.String()

	// Verify Copilot-specific fields
	if !strings.Contains(output, `"type": "local"`) {
		t.Error("Expected 'type': 'local' field for Copilot")
	}
	if !strings.Contains(output, `"tools": ["*"]`) {
		t.Error("Expected 'tools' field for Copilot")
	}
	if !strings.Contains(output, `"playwright": {`) {
		t.Error("Expected playwright server ID")
	}
	if !strings.Contains(output, `"command": "docker"`) {
		t.Error("Expected docker command")
	}
	// Check for trailing comma (not last)
	if !strings.Contains(output, "},\n") {
		t.Error("Expected trailing comma for non-last server")
	}
}

func TestRenderPlaywrightMCP_JSON_Claude(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: false,
		InlineArgs:           false,
		Format:               "json",
		IsLast:               true,
	})

	playwrightTool := map[string]any{
		"allowed-domains": []string{"example.com"},
	}

	var yaml strings.Builder
	renderer.RenderPlaywrightMCP(&yaml, playwrightTool)

	output := yaml.String()

	// Verify Claude format (no Copilot-specific fields)
	if strings.Contains(output, `"type"`) {
		t.Error("Should not contain 'type' field for Claude")
	}
	if strings.Contains(output, `"tools"`) {
		t.Error("Should not contain 'tools' field for Claude")
	}
	if !strings.Contains(output, `"playwright": {`) {
		t.Error("Expected playwright server ID")
	}
	// Check for no trailing comma (last)
	if !strings.Contains(output, "}\n") || strings.Contains(output, "},\n") {
		t.Error("Expected no trailing comma for last server")
	}
}

func TestRenderPlaywrightMCP_TOML(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: false,
		InlineArgs:           false,
		Format:               "toml",
		IsLast:               false,
	})

	playwrightTool := map[string]any{
		"allowed-domains": []string{"example.com"},
	}

	var yaml strings.Builder
	renderer.RenderPlaywrightMCP(&yaml, playwrightTool)

	output := yaml.String()

	// Verify TOML format
	if !strings.Contains(output, "[mcp_servers.playwright]") {
		t.Error("Expected TOML section header")
	}
	if !strings.Contains(output, `command = "docker"`) {
		t.Error("Expected TOML command format")
	}
	if !strings.Contains(output, "args = [") {
		t.Error("Expected TOML args array")
	}
}

func TestRenderSafeOutputsMCP_JSON_Copilot(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: true,
		InlineArgs:           true,
		Format:               "json",
		IsLast:               false,
	})

	var yaml strings.Builder
	renderer.RenderSafeOutputsMCP(&yaml)

	output := yaml.String()

	// Verify Copilot-specific fields
	if !strings.Contains(output, `"type": "local"`) {
		t.Error("Expected 'type': 'local' field for Copilot")
	}
	if !strings.Contains(output, `"tools": ["*"]`) {
		t.Error("Expected 'tools' field for Copilot")
	}
	if !strings.Contains(output, `"safeoutputs": {`) {
		t.Error("Expected safeoutputs server ID")
	}
	// Verify container-based approach
	if !strings.Contains(output, `"container": "node:lts-alpine"`) {
		t.Error("Expected container field")
	}
	if !strings.Contains(output, `"entrypoint": "node"`) {
		t.Error("Expected entrypoint field")
	}
	if !strings.Contains(output, `"entrypointArgs": ["/opt/gh-aw/safeoutputs/mcp-server.cjs"]`) {
		t.Error("Expected entrypointArgs field")
	}
	// Check for env var with backslash escaping (Copilot format)
	if !strings.Contains(output, `\${`) {
		t.Error("Expected backslash-escaped env vars for Copilot")
	}
}

func TestRenderSafeOutputsMCP_JSON_Claude(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: false,
		InlineArgs:           false,
		Format:               "json",
		IsLast:               true,
	})

	var yaml strings.Builder
	renderer.RenderSafeOutputsMCP(&yaml)

	output := yaml.String()

	// Verify Claude format (no Copilot-specific fields)
	if strings.Contains(output, `"type"`) {
		t.Error("Should not contain 'type' field for Claude")
	}
	if strings.Contains(output, `"tools"`) {
		t.Error("Should not contain 'tools' field for Claude")
	}
	// Check for env var without backslash escaping (Claude format)
	if strings.Contains(output, `\${`) {
		t.Error("Should not have backslash-escaped env vars for Claude")
	}
	if !strings.Contains(output, `"$GH_AW_SAFE_OUTPUTS"`) {
		t.Error("Expected direct shell variable reference for Claude")
	}
}

func TestRenderSafeOutputsMCP_TOML(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: false,
		InlineArgs:           false,
		Format:               "toml",
		IsLast:               false,
	})

	var yaml strings.Builder
	renderer.RenderSafeOutputsMCP(&yaml)

	output := yaml.String()

	// Verify TOML format with container-based approach
	if !strings.Contains(output, "[mcp_servers.safeoutputs]") {
		t.Error("Expected TOML section header")
	}
	if !strings.Contains(output, `container = "node:lts-alpine"`) {
		t.Error("Expected TOML container format")
	}
	if !strings.Contains(output, `entrypoint = "node"`) {
		t.Error("Expected TOML entrypoint format")
	}
	if !strings.Contains(output, `entrypointArgs = ["/opt/gh-aw/safeoutputs/mcp-server.cjs"]`) {
		t.Error("Expected TOML entrypointArgs format")
	}
	if !strings.Contains(output, "env_vars = [") {
		t.Error("Expected TOML env_vars array")
	}
}

func TestRenderAgenticWorkflowsMCP_JSON_Copilot(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: true,
		InlineArgs:           true,
		Format:               "json",
		IsLast:               true,
	})

	var yaml strings.Builder
	renderer.RenderAgenticWorkflowsMCP(&yaml)

	output := yaml.String()

	// Verify Copilot-specific fields for containerized format
	if !strings.Contains(output, `"type": "stdio"`) {
		t.Errorf("Expected 'type': 'stdio' field for containerized Copilot server\nGot: %s", output)
	}
	if !strings.Contains(output, `"agentic_workflows": {`) {
		t.Error("Expected agentic_workflows server ID")
	}
	// Verify containerized format (not command-based)
	if !strings.Contains(output, `"container": "alpine:3.21"`) {
		t.Errorf("Expected container field with Alpine image\nGot: %s", output)
	}
	if !strings.Contains(output, `"entrypoint": "/bin/sh"`) {
		t.Errorf("Expected entrypoint field\nGot: %s", output)
	}
	if !strings.Contains(output, `"entrypointArgs": ["-c"`) {
		t.Errorf("Expected entrypointArgs field\nGot: %s", output)
	}
	// Verify the old command format is NOT present
	if strings.Contains(output, `"command": "gh"`) {
		t.Error("Should not contain deprecated 'command' field")
	}
}

func TestRenderAgenticWorkflowsMCP_JSON_Claude(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: false,
		InlineArgs:           false,
		Format:               "json",
		IsLast:               false,
	})

	var yaml strings.Builder
	renderer.RenderAgenticWorkflowsMCP(&yaml)

	output := yaml.String()

	// Verify Claude format (no Copilot-specific fields but still containerized)
	if strings.Contains(output, `"type"`) {
		t.Error("Should not contain 'type' field for Claude")
	}
	if strings.Contains(output, `"tools"`) {
		t.Error("Should not contain 'tools' field for Claude")
	}
	// Verify containerized format is present
	if !strings.Contains(output, `"container": "alpine:3.21"`) {
		t.Errorf("Expected container field with Alpine image\nGot: %s", output)
	}
	// Verify deprecated command format is not present
	if strings.Contains(output, `"command": "gh"`) {
		t.Error("Should not contain deprecated 'command' field")
	}
}

func TestRenderAgenticWorkflowsMCP_TOML(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: false,
		InlineArgs:           false,
		Format:               "toml",
		IsLast:               false,
	})

	var yaml strings.Builder
	renderer.RenderAgenticWorkflowsMCP(&yaml)

	output := yaml.String()

	// Verify TOML format
	if !strings.Contains(output, "[mcp_servers.agentic_workflows]") {
		t.Error("Expected TOML section header")
	}
	if !strings.Contains(output, `command = "gh"`) {
		t.Error("Expected TOML command format")
	}
	if !strings.Contains(output, "args = [") {
		t.Error("Expected TOML args array")
	}
}

func TestRenderGitHubMCP_JSON_Copilot_Local(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: true,
		InlineArgs:           true,
		Format:               "json",
		IsLast:               false,
	})

	githubTool := map[string]any{
		"mode":     "local",
		"toolsets": "default",
	}

	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	var yaml strings.Builder
	renderer.RenderGitHubMCP(&yaml, githubTool, workflowData)

	output := yaml.String()

	// Verify GitHub MCP config
	if !strings.Contains(output, `"github": {`) {
		t.Error("Expected github server ID")
	}
	if !strings.Contains(output, `"type": "local"`) {
		t.Error("Expected 'type': 'local' field for Copilot")
	}
	if !strings.Contains(output, `"command": "docker"`) {
		t.Error("Expected docker command for local mode")
	}
}

func TestRenderGitHubMCP_JSON_Claude_Local(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: false,
		InlineArgs:           false,
		Format:               "json",
		IsLast:               true,
	})

	githubTool := map[string]any{
		"mode":     "local",
		"toolsets": "default",
	}

	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	var yaml strings.Builder
	renderer.RenderGitHubMCP(&yaml, githubTool, workflowData)

	output := yaml.String()

	// Verify GitHub MCP config for Claude (no type field)
	if !strings.Contains(output, `"github": {`) {
		t.Error("Expected github server ID")
	}
	if strings.Contains(output, `"type"`) {
		t.Error("Should not contain 'type' field for Claude")
	}
	if !strings.Contains(output, `"command": "docker"`) {
		t.Error("Expected docker command for local mode")
	}
}

func TestRenderGitHubMCP_JSON_Copilot_Remote(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: true,
		InlineArgs:           true,
		Format:               "json",
		IsLast:               false,
	})

	githubTool := map[string]any{
		"mode":     "remote",
		"toolsets": "default",
	}

	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	var yaml strings.Builder
	renderer.RenderGitHubMCP(&yaml, githubTool, workflowData)

	output := yaml.String()

	// Verify remote GitHub MCP config
	if !strings.Contains(output, `"github": {`) {
		t.Error("Expected github server ID")
	}
	if !strings.Contains(output, `"type": "http"`) {
		t.Error("Expected 'type': 'http' field for remote mode")
	}
	if !strings.Contains(output, `"url"`) {
		t.Error("Expected url field for remote mode")
	}
}

func TestRenderGitHubMCP_TOML(t *testing.T) {
	renderer := NewMCPConfigRenderer(MCPRendererOptions{
		IncludeCopilotFields: false,
		InlineArgs:           false,
		Format:               "toml",
		IsLast:               false,
	})

	githubTool := map[string]any{
		"mode":     "local",
		"toolsets": "default",
	}

	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	var yaml strings.Builder
	renderer.RenderGitHubMCP(&yaml, githubTool, workflowData)

	output := yaml.String()

	// TOML format should now be supported and generate valid output
	if output == "" {
		t.Error("Expected non-empty output for TOML format")
	}

	// Verify key TOML elements are present
	expectedElements := []string{
		"[mcp_servers.github]",
		"user_agent =",
		"startup_timeout_sec =",
		"tool_timeout_sec =",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", expected, output)
		}
	}
}

func TestOptionCombinations(t *testing.T) {
	tests := []struct {
		name    string
		options MCPRendererOptions
	}{
		{
			name: "all true",
			options: MCPRendererOptions{
				IncludeCopilotFields: true,
				InlineArgs:           true,
				Format:               "json",
				IsLast:               true,
			},
		},
		{
			name: "all false",
			options: MCPRendererOptions{
				IncludeCopilotFields: false,
				InlineArgs:           false,
				Format:               "json",
				IsLast:               false,
			},
		},
		{
			name: "mixed copilot inline",
			options: MCPRendererOptions{
				IncludeCopilotFields: true,
				InlineArgs:           false,
				Format:               "json",
				IsLast:               false,
			},
		},
		{
			name: "mixed claude inline",
			options: MCPRendererOptions{
				IncludeCopilotFields: false,
				InlineArgs:           true,
				Format:               "json",
				IsLast:               false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewMCPConfigRenderer(tt.options)

			// Test each render method doesn't panic
			var yaml strings.Builder

			playwrightTool := map[string]any{
				"allowed-domains": []string{"example.com"},
			}
			renderer.RenderPlaywrightMCP(&yaml, playwrightTool)

			yaml.Reset()
			renderer.RenderSafeOutputsMCP(&yaml)

			yaml.Reset()
			renderer.RenderAgenticWorkflowsMCP(&yaml)

			yaml.Reset()
			githubTool := map[string]any{
				"mode":     "local",
				"toolsets": "default",
			}
			workflowData := &WorkflowData{Name: "test"}
			renderer.RenderGitHubMCP(&yaml, githubTool, workflowData)
		})
	}
}
