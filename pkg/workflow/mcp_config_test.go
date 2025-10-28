package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubMCPConfiguration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                string
		frontmatter         string
		expectedType        string // "http" or "docker"
		expectedURL         string
		expectedCommand     string
		expectedDockerImage string
	}{
		{
			name: "default Docker server",
			frontmatter: `---
engine: claude
tools:
  github:
    allowed: [list_issues, create_issue]
---`,
			// With Docker MCP always enabled, default is docker (not services)
			expectedType:        "docker",
			expectedCommand:     "docker",
			expectedDockerImage: "ghcr.io/github/github-mcp-server:v0.20.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test MCP Configuration

This is a test workflow for MCP configuration.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Replace the file extension to .lock.yml
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			// Read the generated lock file
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check the MCP configuration based on expected type
			switch tt.expectedType {
			case "http":
				// Should contain HTTP configuration
				if !strings.Contains(lockContent, `"type": "http"`) {
					t.Errorf("Expected HTTP configuration but didn't find 'type: http' in:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, tt.expectedURL) {
					t.Errorf("Expected URL '%s' but didn't find it in:\n%s", tt.expectedURL, lockContent)
				}
				if !strings.Contains(lockContent, `"Authorization": "Bearer ${{ secrets.GITHUB_TOKEN }}"`) {
					t.Errorf("Expected Authorization header but didn't find it in:\n%s", lockContent)
				}
				// Should NOT contain Docker configuration
				if strings.Contains(lockContent, `"command": "docker"`) {
					t.Errorf("Expected no Docker configuration but found it in:\n%s", lockContent)
				}
			case "docker":
				// Should contain Docker configuration
				if !strings.Contains(lockContent, `"command": "`+tt.expectedCommand+`"`) {
					t.Errorf("Expected command '%s' but didn't find it in:\n%s", tt.expectedCommand, lockContent)
				}
				if !strings.Contains(lockContent, tt.expectedDockerImage) {
					t.Errorf("Expected Docker image '%s' but didn't find it in:\n%s", tt.expectedDockerImage, lockContent)
				}
				if !strings.Contains(lockContent, `"GITHUB_PERSONAL_ACCESS_TOKEN": "${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}"`) {
					t.Errorf("Expected GITHUB_PERSONAL_ACCESS_TOKEN env var but didn't find it in:\n%s", lockContent)
				}
				// Should NOT contain HTTP configuration
				if strings.Contains(lockContent, `"type": "http"`) {
					t.Errorf("Expected no HTTP configuration but found it in:\n%s", lockContent)
				}
				// Should NOT contain services configuration
				if strings.Contains(lockContent, `services:`) {
					t.Errorf("Expected no services configuration but found it in:\n%s", lockContent)
				}
			}

			// All configurations should contain the github server
			if !strings.Contains(lockContent, `"github": {`) {
				t.Errorf("Expected github server configuration but didn't find it in:\n%s", lockContent)
			}
		})
	}
}

func TestGenerateGitHubMCPConfig(t *testing.T) {
	tests := []struct {
		name         string
		githubTool   any
		expectedType string
	}{
		{
			name:       "nil github tool",
			githubTool: nil,
			// With new defaults, nil tool defaults to docker (not services)
			expectedType: "docker",
		},
		{
			name: "empty github tool config",
			githubTool: map[string]any{
				"allowed": []any{"list_issues"},
			},
			// With Docker always enabled, empty config defaults to docker (not services)
			expectedType: "docker",
		},
		{
			name: "explicit docker config (redundant)",
			githubTool: map[string]any{
				"allowed": []any{"list_issues"},
			},
			// Docker is always enabled now
			expectedType: "docker",
		},
		{
			name:       "non-map github tool",
			githubTool: "invalid",
			// With Docker always enabled, invalid tool config defaults to docker (not services)
			expectedType: "docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yamlBuilder strings.Builder

			// Call the function under test using the Claude engine
			engine := NewClaudeEngine()
			workflowData := &WorkflowData{}
			engine.renderGitHubClaudeMCPConfig(&yamlBuilder, tt.githubTool, true, workflowData)

			result := yamlBuilder.String()

			switch tt.expectedType {
			case "docker":
				if !strings.Contains(result, `"command": "docker"`) {
					t.Errorf("Expected Docker command but got:\n%s", result)
				}
				if !strings.Contains(result, `"ghcr.io/github/github-mcp-server:v0.20.1"`) {
					t.Errorf("Expected Docker image but got:\n%s", result)
				}
				if strings.Contains(result, `"type": "http"`) {
					t.Errorf("Expected no HTTP type but found it in:\n%s", result)
				}
			}
		})
	}
}

func TestMCPConfigurationEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		githubTool any
		isLast     bool
		expected   string
	}{
		{
			name: "last server with docker config",
			githubTool: map[string]any{
				"allowed": []any{"list_issues"},
			},
			isLast:   true,
			expected: `              }`,
		},
		{
			name: "not last server with docker config",
			githubTool: map[string]any{
				"allowed": []any{"list_issues"},
			},
			isLast:   false,
			expected: `              },`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yamlBuilder strings.Builder

			// Call the function under test using the Claude engine
			engine := NewClaudeEngine()
			workflowData := &WorkflowData{}
			engine.renderGitHubClaudeMCPConfig(&yamlBuilder, tt.githubTool, tt.isLast, workflowData)

			result := yamlBuilder.String()

			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to end with '%s' but got:\n%s", tt.expected, result)
			}
		})
	}
}

func TestCustomDockerMCPConfiguration(t *testing.T) {
	t.Skip("Skipping test for new MCP format - implementation in progress (schema requires custom tools to be strings, not objects)")

	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "custom-docker-mcp-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                string
		frontmatter         string
		expectedType        string // "docker" or "http"
		expectedDockerImage string // Expected Docker image version
	}{
		{
			name: "custom docker MCP with default settings",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues, create_issue]
  custom_tool:
    type: "stdio"
    command: "docker"
    args: ["run", "-i", "--rm", "custom/mcp-server:latest"]
    allowed: ["custom_action"]
---`,
			expectedType:        "docker",  // Services mode removed - always Docker
			expectedDockerImage: "v0.20.1", // Default version
		},
		{
			name: "custom docker MCP with different settings",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues, create_issue]
  custom_tool:
    type: "stdio"
    command: "docker"
    args: ["run", "-i", "--rm", "custom/mcp-server:latest"]
    allowed: ["custom_action"]
---`,
			expectedType:        "docker",
			expectedDockerImage: "v0.20.1", // Default version
		},
		{
			name: "mixed MCP configuration with defaults",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues, create_issue]
  filesystem:
    type: "stdio"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    allowed: ["list_files", "read_file"]
  docker_tool:
    type: "stdio"
    command: "docker"
    args: ["run", "-i", "--rm", "-v", "/tmp:/workspace", "custom/tool:latest"]
    allowed: ["docker_action"]
---`,
			expectedType:        "docker",  // GitHub should now use docker by default (not services)
			expectedDockerImage: "v0.20.1", // Default version
		},
		{
			name: "custom docker MCP with custom Docker image version",
			frontmatter: `---
tools:
  github:
    version: "v2.0.0"
    allowed: [list_issues, create_issue]
  custom_tool:
    type: "stdio"
    command: "docker"
    args: ["run", "-i", "--rm", "custom/mcp-server:latest"]
    allowed: ["custom_action"]
---`,
			expectedType:        "docker", // GitHub always uses docker now
			expectedDockerImage: "v2.0.0", // Custom version
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Custom Docker MCP Configuration

This is a test workflow for custom Docker MCP configuration with different scenarios.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check the GitHub MCP configuration based on expected type
			switch tt.expectedType {
			case "docker":
				// Should contain Docker configuration for GitHub
				if !strings.Contains(lockContent, `"command": "docker"`) {
					t.Errorf("Expected Docker command but didn't find it in:\n%s", lockContent)
				}
				if tt.expectedDockerImage != "" {
					expectedImageString := fmt.Sprintf(`"ghcr.io/github/github-mcp-server:%s"`, tt.expectedDockerImage)
					if !strings.Contains(lockContent, expectedImageString) {
						t.Errorf("Expected Docker image '%s' but didn't find it in:\n%s", expectedImageString, lockContent)
					}
				}
				// Should NOT contain services configuration
				if strings.Contains(lockContent, `services:`) {
					t.Errorf("Expected no services configuration but found it in:\n%s", lockContent)
				}
			}

			// Services mode has been removed - never expect services section
			if strings.Contains(lockContent, `services:`) {
				t.Errorf("Expected no services section (services mode removed) but found it in:\n%s", lockContent)
			}

			// All configurations should contain the github server
			if !strings.Contains(lockContent, `"github": {`) {
				t.Errorf("Expected github server configuration but didn't find it in:\n%s", lockContent)
			}

			// Should contain custom MCP tools if specified
			if strings.Contains(tt.frontmatter, "custom_tool") {
				if !strings.Contains(lockContent, `"custom_tool": {`) {
					t.Errorf("Expected custom_tool server configuration but didn't find it in:\n%s", lockContent)
				}
			}
			if strings.Contains(tt.frontmatter, "filesystem") {
				if !strings.Contains(lockContent, `"filesystem": {`) {
					t.Errorf("Expected filesystem server configuration but didn't find it in:\n%s", lockContent)
				}
			}
			if strings.Contains(tt.frontmatter, "docker_tool") {
				if !strings.Contains(lockContent, `"docker_tool": {`) {
					t.Errorf("Expected docker_tool server configuration but didn't find it in:\n%s", lockContent)
				}
			}
		})
	}
}
