package workflow

import (
	"os"
	"strings"
	"testing"
)

// TestMCPServersCompilation verifies that mcp-servers configuration is properly compiled into workflows
func TestMCPServersCompilation(t *testing.T) {
	// Create a temporary markdown file with mcp-servers configuration
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
network: defaults
mcp-servers:
  markitdown:
    registry: https://api.mcp.github.com/v0/servers/microsoft/markitdown
    command: npx
    args: ["-y", "@microsoft/markitdown"]
---

# Test MCP Servers Configuration

This workflow tests that mcp-servers are properly compiled into the lock file.

Please use the markitdown MCP server to convert HTML to markdown.
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-mcp-servers-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write content to file
	if _, err := tmpFile.WriteString(workflowContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Create compiler and compile workflow
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true) // Skip validation for test

	// Parse the workflow file to get WorkflowData
	workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse workflow file: %v", err)
	}

	// Verify that both github (default) and markitdown tools are recognized
	if len(workflowData.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(workflowData.Tools))
	}

	// Verify markitdown tool exists in tools
	markitdownTool, hasMarkitdown := workflowData.Tools["markitdown"]
	if !hasMarkitdown {
		t.Errorf("Expected markitdown tool to be present in tools")
	}

	// Verify markitdown tool configuration
	if markitdownConfig, ok := markitdownTool.(map[string]any); ok {
		// Check command
		if command, ok := markitdownConfig["command"].(string); !ok || command != "npx" {
			t.Errorf("Expected command 'npx', got %v", markitdownConfig["command"])
		}

		// Check args
		if args, ok := markitdownConfig["args"].([]any); ok {
			if len(args) != 2 {
				t.Errorf("Expected 2 args, got %d", len(args))
			}
			if args[0] != "-y" || args[1] != "@microsoft/markitdown" {
				t.Errorf("Expected args ['-y', '@microsoft/markitdown'], got %v", args)
			}
		} else {
			t.Errorf("Expected args to be array, got %T", markitdownConfig["args"])
		}

		// Check registry
		if registry, ok := markitdownConfig["registry"].(string); !ok || registry != "https://api.mcp.github.com/v0/servers/microsoft/markitdown" {
			t.Errorf("Expected registry URL, got %v", markitdownConfig["registry"])
		}
	} else {
		t.Errorf("Expected markitdown tool to be a map, got %T", markitdownTool)
	}

	// Generate YAML and verify MCP configuration is included
	yamlContent, err := compiler.generateYAML(workflowData, tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Verify the generated YAML contains markitdown MCP server
	if !strings.Contains(yamlContent, `"markitdown": {`) {
		t.Errorf("Generated YAML does not contain markitdown MCP server configuration")
	}

	// Verify the MCP server has correct configuration
	if !strings.Contains(yamlContent, `"command": "npx"`) {
		t.Errorf("Generated YAML does not contain correct npx command")
	}

	if !strings.Contains(yamlContent, `"@microsoft/markitdown"`) {
		t.Errorf("Generated YAML does not contain correct markitdown package")
	}

	if !strings.Contains(yamlContent, `"registry": "https://api.mcp.github.com/v0/servers/microsoft/markitdown"`) {
		t.Errorf("Generated YAML does not contain correct registry URL")
	}
}

// TestHasMCPConfigDetection verifies that hasMCPConfig properly detects MCP configurations
func TestHasMCPConfigDetection(t *testing.T) {
	testCases := []struct {
		name     string
		config   map[string]any
		expected bool
		mcpType  string
	}{
		{
			name: "explicit stdio type",
			config: map[string]any{
				"type":    "stdio",
				"command": "npx",
			},
			expected: true,
			mcpType:  "stdio",
		},
		{
			name: "explicit http type",
			config: map[string]any{
				"type": "http",
				"url":  "https://example.com",
			},
			expected: true,
			mcpType:  "http",
		},
		{
			name: "inferred stdio from command",
			config: map[string]any{
				"command": "npx",
				"args":    []any{"-y", "@microsoft/markitdown"},
			},
			expected: true,
			mcpType:  "stdio",
		},
		{
			name: "inferred http from url",
			config: map[string]any{
				"url": "https://example.com/mcp",
			},
			expected: true,
			mcpType:  "http",
		},
		{
			name: "inferred stdio from container",
			config: map[string]any{
				"container": "example/mcp:latest",
			},
			expected: true,
			mcpType:  "stdio",
		},
		{
			name: "not MCP config",
			config: map[string]any{
				"allowed": []any{"some_tool"},
			},
			expected: false,
			mcpType:  "",
		},
		{
			name: "markitdown-like config",
			config: map[string]any{
				"registry": "https://api.mcp.github.com/v0/servers/microsoft/markitdown",
				"command":  "npx",
				"args":     []any{"-y", "@microsoft/markitdown"},
			},
			expected: true,
			mcpType:  "stdio",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasMcp, mcpType := hasMCPConfig(tc.config)
			if hasMcp != tc.expected {
				t.Errorf("Expected hasMCPConfig to return %v, got %v", tc.expected, hasMcp)
			}
			if mcpType != tc.mcpType {
				t.Errorf("Expected MCP type %q, got %q", tc.mcpType, mcpType)
			}
		})
	}
}
