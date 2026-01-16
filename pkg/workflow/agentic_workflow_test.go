package workflow

import (
	"strings"
	"testing"
)

func TestAgenticWorkflowsSyntaxVariations(t *testing.T) {
	tests := []struct {
		name        string
		toolValue   any
		shouldWork  bool
		description string
	}{
		{
			name:        "agentic-workflows with nil (no value)",
			toolValue:   nil,
			shouldWork:  true,
			description: "Should enable agentic-workflows when field is present without value",
		},
		{
			name:        "agentic-workflows with true",
			toolValue:   true,
			shouldWork:  true,
			description: "Should enable agentic-workflows with boolean true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal workflow with the agentic-workflows tool
			frontmatter := map[string]any{
				"on":    "workflow_dispatch",
				"tools": map[string]any{"agentic-workflows": tt.toolValue},
			}

			// Create compiler
			c := NewCompiler(false, "", "test")
			c.SetSkipValidation(true)

			// Extract tools from frontmatter
			tools := extractToolsFromFrontmatter(frontmatter)

			// Merge tools
			mergedTools, err := c.mergeToolsAndMCPServers(tools, make(map[string]any), "")
			if err != nil {
				if tt.shouldWork {
					t.Errorf("Expected tool to work but got error: %v", err)
				}
				return
			}

			if !tt.shouldWork {
				t.Errorf("Expected tool to fail but it succeeded")
				return
			}

			// Verify the agentic-workflows tool is present
			if _, exists := mergedTools["agentic-workflows"]; !exists {
				t.Errorf("Expected agentic-workflows tool to be present in merged tools")
			}
		})
	}
}

func TestAgenticWorkflowsMCPConfigGeneration(t *testing.T) {
	engines := []struct {
		name   string
		engine CodingAgentEngine
	}{
		{"Claude", NewClaudeEngine()},
		{"Copilot", NewCopilotEngine()},
		{"Custom", NewCustomEngine()},
		{"Codex", NewCodexEngine()},
	}

	for _, e := range engines {
		t.Run(e.name, func(t *testing.T) {
			// Create workflow data with agentic-workflows tool
			workflowData := &WorkflowData{
				Tools: map[string]any{
					"agentic-workflows": nil,
				},
			}

			// Generate MCP config
			var yaml strings.Builder
			mcpTools := []string{"agentic-workflows"}

			e.engine.RenderMCPConfig(&yaml, workflowData.Tools, mcpTools, workflowData)
			result := yaml.String()

			// Verify the MCP config contains agentic-workflows
			if !strings.Contains(result, "agentic_workflows") {
				t.Errorf("Expected MCP config to contain 'agentic_workflows', got: %s", result)
			}

			// Verify it uses HTTP transport
			if !strings.Contains(result, "http") {
				t.Errorf("Expected MCP config to use HTTP transport, got: %s", result)
			}

			// Verify it has URL and headers (HTTP mode)
			if !strings.Contains(result, "url") && !strings.Contains(result, "URL") {
				t.Errorf("Expected MCP config to contain 'url' field for HTTP transport, got: %s", result)
			}

			// Verify it has Authorization header
			if !strings.Contains(result, "Authorization") && !strings.Contains(result, "headers") {
				t.Errorf("Expected MCP config to contain 'Authorization' header, got: %s", result)
			}
		})
	}
}

func TestAgenticWorkflowsHasMCPServers(t *testing.T) {
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"agentic-workflows": nil,
		},
	}

	if !HasMCPServers(workflowData) {
		t.Error("Expected HasMCPServers to return true for agentic-workflows tool")
	}
}

func TestAgenticWorkflowsInstallStepIncludesGHToken(t *testing.T) {
	// Create workflow data with agentic-workflows tool
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"agentic-workflows": nil,
		},
	}

	// Create compiler
	c := NewCompiler(false, "", "test")
	c.SetSkipValidation(true)

	// Generate MCP setup
	var yaml strings.Builder
	engine := NewCopilotEngine()

	c.generateMCPSetup(&yaml, workflowData.Tools, engine, workflowData)
	result := yaml.String()

	// Verify the install step is present
	if !strings.Contains(result, "Install gh-aw extension") {
		t.Error("Expected 'Install gh-aw extension' step not found in generated YAML")
	}

	// Verify GH_TOKEN environment variable is set with the default token expression
	if !strings.Contains(result, "GH_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}") {
		t.Errorf("Expected GH_TOKEN environment variable to be set with default token expression in install step, got:\n%s", result)
	}

	// Verify the install commands are present
	if !strings.Contains(result, "gh extension install githubnext/gh-aw") {
		t.Error("Expected 'gh extension install' command not found in generated YAML")
	}

	if !strings.Contains(result, "gh aw --version") {
		t.Error("Expected 'gh aw --version' command not found in generated YAML")
	}
}

func TestAgenticWorkflowsInstallStepWithCustomToken(t *testing.T) {
	// Create workflow data with agentic-workflows tool and custom github-token
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"agentic-workflows": nil,
		},
		GitHubToken: "${{ secrets.CUSTOM_PAT }}",
	}

	// Create compiler
	c := NewCompiler(false, "", "test")
	c.SetSkipValidation(true)

	// Generate MCP setup
	var yaml strings.Builder
	engine := NewCopilotEngine()

	c.generateMCPSetup(&yaml, workflowData.Tools, engine, workflowData)
	result := yaml.String()

	// Verify the install step is present
	if !strings.Contains(result, "Install gh-aw extension") {
		t.Error("Expected 'Install gh-aw extension' step not found in generated YAML")
	}

	// Verify GH_TOKEN environment variable is set with the custom token
	if !strings.Contains(result, "GH_TOKEN: ${{ secrets.CUSTOM_PAT }}") {
		t.Errorf("Expected GH_TOKEN environment variable to use custom token in install step, got:\n%s", result)
	}

	// Verify it doesn't use the default token when custom is provided
	if strings.Contains(result, "GH_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}") {
		t.Error("Should not use default token when custom token is specified")
	}
}

func TestAgenticWorkflowsInstallStepSkippedWithImport(t *testing.T) {
	// Create workflow data with agentic-workflows tool AND shared/mcp/gh-aw.md import
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"agentic-workflows": nil,
		},
		ImportedFiles: []string{"shared/mcp/gh-aw.md"},
	}

	// Create compiler
	c := NewCompiler(false, "", "test")
	c.SetSkipValidation(true)

	// Generate MCP setup
	var yaml strings.Builder
	engine := NewCopilotEngine()

	c.generateMCPSetup(&yaml, workflowData.Tools, engine, workflowData)
	result := yaml.String()

	// Verify the install step is NOT present when import exists
	if strings.Contains(result, "Install gh-aw extension") {
		t.Error("Expected 'Install gh-aw extension' step to be skipped when shared/mcp/gh-aw.md is imported, but it was present")
	}

	// Verify the install command is also not present
	if strings.Contains(result, "gh extension install githubnext/gh-aw") {
		t.Error("Expected 'gh extension install' command to be absent when shared/mcp/gh-aw.md is imported, but it was present")
	}
}

func TestAgenticWorkflowsInstallStepPresentWithoutImport(t *testing.T) {
	// Create workflow data with agentic-workflows tool but NO import
	workflowData := &WorkflowData{
		Tools: map[string]any{
			"agentic-workflows": nil,
		},
		ImportedFiles: []string{}, // Empty imports
	}

	// Create compiler
	c := NewCompiler(false, "", "test")
	c.SetSkipValidation(true)

	// Generate MCP setup
	var yaml strings.Builder
	engine := NewCopilotEngine()

	c.generateMCPSetup(&yaml, workflowData.Tools, engine, workflowData)
	result := yaml.String()

	// Verify the install step IS present when no import exists
	if !strings.Contains(result, "Install gh-aw extension") {
		t.Error("Expected 'Install gh-aw extension' step to be present when shared/mcp/gh-aw.md is NOT imported, but it was missing")
	}

	// Verify the install command is present
	if !strings.Contains(result, "gh extension install githubnext/gh-aw") {
		t.Error("Expected 'gh extension install' command to be present when shared/mcp/gh-aw.md is NOT imported, but it was missing")
	}
}
