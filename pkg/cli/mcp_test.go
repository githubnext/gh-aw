package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestMCPCommandIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Change working directory to temp for tests
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	_ = os.Chdir(tempDir)

	// Create test workflow file
	testWorkflow := `---
on:
  workflow_dispatch:

permissions: read-all
---

# Test Workflow

This is a test workflow for MCP management.`

	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(testWorkflow), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	t.Run("AddBuiltinMCPTool", func(t *testing.T) {
		// Test adding GitHub tool
		err := AddMCPTool("test-workflow", "github", []string{}, []string{"create_issue", "list_repos"}, false, false)
		if err != nil {
			t.Fatalf("Failed to add GitHub MCP tool: %v", err)
		}

		// Verify the tool was added
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		if !strings.Contains(string(content), "github:") {
			t.Error("GitHub tool not found in workflow file")
		}

		if !strings.Contains(string(content), "create_issue") {
			t.Error("Allowed tools not found in workflow file")
		}
	})

	t.Run("ListMCPTools", func(t *testing.T) {
		// Test listing MCP tools
		err := ListMCPTools("test-workflow", false, "table", false)
		if err != nil {
			t.Fatalf("Failed to list MCP tools: %v", err)
		}
	})

	t.Run("AddDuplicateToolShouldFail", func(t *testing.T) {
		// Test adding duplicate tool without force
		err := AddMCPTool("test-workflow", "github", []string{}, []string{}, false, false)
		if err == nil {
			t.Error("Expected error when adding duplicate tool, got nil")
		}

		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("AddDuplicateToolWithForce", func(t *testing.T) {
		// Test adding duplicate tool with force
		err := AddMCPTool("test-workflow", "github", []string{}, []string{"new_tool"}, true, false)
		if err != nil {
			t.Fatalf("Failed to add duplicate tool with force: %v", err)
		}

		// Verify the tool was updated
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("Failed to read workflow file: %v", err)
		}

		if !strings.Contains(string(content), "new_tool") {
			t.Error("Updated allowed tools not found in workflow file")
		}
	})

	t.Run("NonExistentWorkflow", func(t *testing.T) {
		// Test adding tool to non-existent workflow
		err := AddMCPTool("non-existent", "github", []string{}, []string{}, false, false)
		if err == nil {
			t.Error("Expected error for non-existent workflow, got nil")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})
}

func TestMCPToolConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		allowedTools []string
		expectError  bool
	}{
		{
			name:         "GitHub tool with allowed tools",
			toolName:     "github",
			allowedTools: []string{"create_issue", "list_repos"},
			expectError:  false,
		},
		{
			name:         "Playwright tool with allowed tools",
			toolName:     "playwright",
			allowedTools: []string{"navigate", "click"},
			expectError:  false,
		},
		{
			name:         "GitHub tool without allowed tools",
			toolName:     "github",
			allowedTools: []string{},
			expectError:  false,
		},
		{
			name:         "Custom tool without type",
			toolName:     "custom-tool",
			allowedTools: []string{},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := createToolConfig(tt.toolName, []string{}, tt.allowedTools)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for custom tool without type, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check configuration structure
			if tt.toolName == "github" || tt.toolName == "playwright" {
				if len(tt.allowedTools) > 0 {
					allowed, exists := config["allowed"]
					if !exists {
						t.Error("Expected 'allowed' field in config")
					}

					allowedSlice, ok := allowed.([]string)
					if !ok {
						t.Error("Expected 'allowed' to be []string")
					}

					if len(allowedSlice) != len(tt.allowedTools) {
						t.Errorf("Expected %d allowed tools, got %d", len(tt.allowedTools), len(allowedSlice))
					}
				}
			}
		})
	}
}

func TestCustomMCPToolConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		mcpType     string
		command     string
		args        []string
		url         string
		container   string
		env         map[string]string
		headers     map[string]string
		allowed     []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "stdio tool with command",
			toolName:    "custom-stdio",
			mcpType:     "stdio",
			command:     "python",
			args:        []string{"-m", "server"},
			expectError: false,
		},
		{
			name:        "stdio tool with container",
			toolName:    "custom-docker",
			mcpType:     "stdio",
			container:   "my/mcp-server:latest",
			env:         map[string]string{"API_KEY": "test"},
			expectError: false,
		},
		{
			name:        "http tool with url",
			toolName:    "custom-http",
			mcpType:     "http",
			url:         "https://api.example.com/mcp",
			headers:     map[string]string{"Authorization": "Bearer token"},
			expectError: false,
		},
		{
			name:        "stdio tool without command or container",
			toolName:    "invalid-stdio",
			mcpType:     "stdio",
			expectError: true,
			errorMsg:    "requires --command or --container",
		},
		{
			name:        "http tool without url",
			toolName:    "invalid-http",
			mcpType:     "http",
			expectError: true,
			errorMsg:    "requires --url",
		},
		{
			name:        "unsupported type",
			toolName:    "invalid-type",
			mcpType:     "websocket",
			expectError: true,
			errorMsg:    "unsupported MCP type",
		},
		{
			name:        "custom tool without type",
			toolName:    "no-type",
			expectError: true,
			errorMsg:    "requires --type flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := createToolConfigWithFlags(
				tt.toolName, tt.mcpType, tt.command, tt.args,
				tt.url, tt.container, tt.env, tt.headers, tt.allowed,
			)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify configuration structure
			mcpConfig, exists := config["mcp"]
			if !exists {
				t.Error("Expected 'mcp' section in config")
				return
			}

			mcpMap, ok := mcpConfig.(map[string]any)
			if !ok {
				t.Error("Expected 'mcp' to be a map")
				return
			}

			// Check type
			if mcpMap["type"] != tt.mcpType {
				t.Errorf("Expected type '%s', got '%v'", tt.mcpType, mcpMap["type"])
			}

			// Check type-specific fields
			switch tt.mcpType {
			case "stdio":
				if tt.container != "" {
					if mcpMap["container"] != tt.container {
						t.Errorf("Expected container '%s', got '%v'", tt.container, mcpMap["container"])
					}
				} else {
					if mcpMap["command"] != tt.command {
						t.Errorf("Expected command '%s', got '%v'", tt.command, mcpMap["command"])
					}
				}
			case "http":
				if mcpMap["url"] != tt.url {
					t.Errorf("Expected url '%s', got '%v'", tt.url, mcpMap["url"])
				}
			}
		})
	}
}

func TestGetWorkflowPath(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Change working directory to temp for tests
	oldCwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldCwd) }()
	_ = os.Chdir(tempDir)

	// Create test workflow file
	workflowPath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		workflowID  string
		expectError bool
	}{
		{
			name:        "existing workflow without extension",
			workflowID:  "test",
			expectError: false,
		},
		{
			name:        "existing workflow with extension",
			workflowID:  "test.md",
			expectError: false,
		},
		{
			name:        "non-existent workflow",
			workflowID:  "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := getWorkflowPath(tt.workflowID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for non-existent workflow, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			expectedPath := filepath.Join(".github", "workflows", "test.md")
			if path != expectedPath {
				t.Errorf("Expected path '%s', got '%s'", expectedPath, path)
			}
		})
	}
}

func TestWriteFrontmatterToFile(t *testing.T) {
	// Create temporary file
	tempFile, err := os.CreateTemp("", "test-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Create test frontmatter data
	frontmatter := map[string]any{
		"on": map[string]any{
			"workflow_dispatch": nil,
		},
		"permissions": "read-all",
		"tools": map[string]any{
			"github": map[string]any{
				"allowed": []string{"create_issue"},
			},
		},
	}

	workflowData := &parser.FrontmatterResult{
		Frontmatter: frontmatter,
		Markdown:    "# Test Workflow\n\nThis is a test.",
	}

	// Write frontmatter to file
	err = writeFrontmatterToFile(tempFile.Name(), workflowData)
	if err != nil {
		t.Fatalf("Failed to write frontmatter: %v", err)
	}

	// Read and verify file content
	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)

	// Check that frontmatter is properly formatted
	if !strings.HasPrefix(contentStr, "---\n") {
		t.Error("File should start with '---'")
	}

	if !strings.Contains(contentStr, "# Test Workflow") {
		t.Error("Markdown content not found in file")
	}

	if !strings.Contains(contentStr, "github:") {
		t.Error("Tools section not found in file")
	}

	// Parse the written file to ensure it's valid
	_, err = parser.ExtractFrontmatterFromContent(contentStr)
	if err != nil {
		t.Fatalf("Written file contains invalid frontmatter: %v", err)
	}
}
