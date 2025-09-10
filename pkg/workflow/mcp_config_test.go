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
			name: "default HTTP server",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues, create_issue]
---`,
			// With HTTP MCP enabled by default for Claude engine
			expectedType: "http",
			expectedURL:  "https://api.github.com/mcp",
		},
		{
			name: "default HTTP server with list_issues",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues]
---`,
			expectedType: "http",
			expectedURL:  "https://api.github.com/mcp",
		},
		{
			name: "default HTTP server with multiple tools",
			frontmatter: `---
tools:
  github:
    allowed: [get_issue, list_pull_requests, search_issues]
---`,
			expectedType: "http",
			expectedURL:  "https://api.github.com/mcp",
		},
		{
			name: "default HTTP server with different allowed tools",
			frontmatter: `---
tools:
  github:
    allowed: [get_issue, list_pull_requests]
---`,
			expectedType: "http",
			expectedURL:  "https://api.github.com/mcp",
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
				if !strings.Contains(lockContent, `"GITHUB_PERSONAL_ACCESS_TOKEN": "${{ secrets.GITHUB_TOKEN }}"`) {
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
			// With new defaults, nil tool defaults to http (remote MCP server)
			expectedType: "http",
		},
		{
			name: "empty github tool config",
			githubTool: map[string]any{
				"allowed": []any{"list_issues"},
			},
			// With HTTP always enabled, empty config defaults to http (remote MCP server)
			expectedType: "http",
		},
		{
			name: "explicit github tool config",
			githubTool: map[string]any{
				"allowed": []any{"list_issues"},
			},
			// HTTP is always enabled now for Claude engine
			expectedType: "http",
		},
		{
			name:       "non-map github tool",
			githubTool: "invalid",
			// With HTTP always enabled, invalid tool config defaults to http (remote MCP server)
			expectedType: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yamlBuilder strings.Builder

			// Call the function under test using the Claude engine
			engine := NewClaudeEngine()
			engine.renderGitHubClaudeMCPConfig(&yamlBuilder, tt.githubTool, true)

			result := yamlBuilder.String()

			switch tt.expectedType {
			case "http":
				if !strings.Contains(result, `"type": "http"`) {
					t.Errorf("Expected HTTP type but got:\n%s", result)
				}
				if !strings.Contains(result, `"url": "https://api.github.com/mcp"`) {
					t.Errorf("Expected HTTP URL but got:\n%s", result)
				}
				if !strings.Contains(result, `"Authorization": "Bearer ${{ secrets.GITHUB_TOKEN }}"`) {
					t.Errorf("Expected Authorization header but got:\n%s", result)
				}
				if strings.Contains(result, `"command": "docker"`) {
					t.Errorf("Expected no Docker command but found it in:\n%s", result)
				}
			case "docker":
				if !strings.Contains(result, `"command": "docker"`) {
					t.Errorf("Expected Docker command but got:\n%s", result)
				}
				if !strings.Contains(result, `"ghcr.io/github/github-mcp-server:sha-09deac4"`) {
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
			name: "last server with http config",
			githubTool: map[string]any{
				"allowed": []any{"list_issues"},
			},
			isLast:   true,
			expected: `              }`,
		},
		{
			name: "not last server with http config",
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
			engine.renderGitHubClaudeMCPConfig(&yamlBuilder, tt.githubTool, tt.isLast)

			result := yamlBuilder.String()

			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected result to end with '%s' but got:\n%s", tt.expected, result)
			}
		})
	}
}

func TestCustomDockerMCPConfiguration(t *testing.T) {
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
			name: "custom docker MCP with default Claude HTTP settings",
			frontmatter: `---
tools:
  github:
    use_docker_mcp: true
    allowed: [list_issues, create_issue]
  custom_tool:
    mcp:
      type: "stdio"
      command: "docker"
      args: ["run", "-i", "--rm", "custom/mcp-server:latest"]
---`,
			expectedType: "http", // Claude engine now uses HTTP transport for GitHub
		},
		{
			name: "custom docker MCP with default Claude HTTP settings",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues, create_issue]
  custom_tool:
    mcp:
      type: "stdio"
      command: "docker"
      args: ["run", "-i", "--rm", "custom/mcp-server:latest"]
---`,
			expectedType: "http", // Claude engine now uses HTTP transport for GitHub
		},
		{
			name: "custom docker MCP with different settings",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues, create_issue]
  custom_tool:
    mcp:
      type: "stdio"
      command: "docker"
      args: ["run", "-i", "--rm", "custom/mcp-server:latest"]
---`,
			expectedType: "http", // Claude engine now uses HTTP transport for GitHub
		},
		{
			name: "mixed MCP configuration with defaults",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues, create_issue]
  filesystem:
    mcp:
      type: "stdio"
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
  docker_tool:
    mcp:
      type: "stdio"
      command: "docker"
      args: ["run", "-i", "--rm", "-v", "/tmp:/workspace", "custom/tool:latest"]
---`,
			expectedType: "http", // Claude engine now uses HTTP transport for GitHub
		},
		{
			name: "custom docker MCP with custom Docker image version",
			frontmatter: `---
tools:
  github:
    docker_image_version: "v2.0.0"
    allowed: [list_issues, create_issue]
  custom_tool:
    mcp:
      type: "stdio"
      command: "docker"
      args: ["run", "-i", "--rm", "custom/mcp-server:latest"]
---`,
			expectedType: "http", // Claude engine now uses HTTP transport for GitHub
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
			case "http":
				// Should contain HTTP configuration for GitHub (Claude engine)
				if !strings.Contains(lockContent, `"type": "http"`) {
					t.Errorf("Expected HTTP type but didn't find it in:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, `"url": "https://api.github.com/mcp"`) {
					t.Errorf("Expected GitHub MCP URL but didn't find it in:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, `"Authorization": "Bearer ${{ secrets.GITHUB_TOKEN }}"`) {
					t.Errorf("Expected Authorization header but didn't find it in:\n%s", lockContent)
				}
				// Should NOT contain Docker configuration in the GitHub server section
				githubSection := extractGitHubSection(lockContent)
				if githubSection != "" && strings.Contains(githubSection, `"command": "docker"`) {
					t.Errorf("Expected no Docker command in GitHub MCP section but found it in:\n%s", githubSection)
				}
				// Should NOT contain services configuration
				if strings.Contains(lockContent, `services:`) {
					t.Errorf("Expected no services configuration but found it in:\n%s", lockContent)
				}
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

// extractGitHubSection extracts just the GitHub MCP server section from the workflow content
func extractGitHubSection(content string) string {
	// Find the start of the github section
	startIdx := strings.Index(content, `"github": {`)
	if startIdx == -1 {
		return ""
	}
	
	// Find the matching closing brace
	braceCount := 0
	inBraces := false
	for i := startIdx; i < len(content); i++ {
		char := content[i]
		if char == '{' {
			braceCount++
			inBraces = true
		} else if char == '}' {
			braceCount--
			if inBraces && braceCount == 0 {
				return content[startIdx:i+1]
			}
		}
	}
	
	return ""
}
