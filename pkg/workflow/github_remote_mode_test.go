package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubRemoteModeConfiguration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "github-remote-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name          string
		frontmatter   string
		expectedType  string // "remote" or "local"
		expectedURL   string
		expectedToken string
		engineType    string
	}{
		{
			name: "Remote mode with default token",
			frontmatter: `---
engine: claude
tools:
  github:
    mode: remote
    allowed: [list_issues, create_issue]
---`,
			expectedType:  "remote",
			expectedURL:   "https://api.githubcopilot.com/mcp/",
			expectedToken: "${{ secrets.GITHUB_MCP_TOKEN }}",
			engineType:    "claude",
		},
		{
			name: "Remote mode with custom token",
			frontmatter: `---
engine: claude
tools:
  github:
    mode: remote
    github-token: "${{ secrets.CUSTOM_PAT }}"
    allowed: [list_issues, create_issue]
---`,
			expectedType:  "remote",
			expectedURL:   "https://api.githubcopilot.com/mcp/",
			expectedToken: "${{ secrets.CUSTOM_PAT }}",
			engineType:    "claude",
		},
		{
			name: "Remote mode with read-only",
			frontmatter: `---
engine: claude
tools:
  github:
    mode: remote
    read-only: true
    allowed: [list_issues]
---`,
			expectedType:  "remote",
			expectedURL:   "https://api.githubcopilot.com/mcp/",
			expectedToken: "${{ secrets.GITHUB_MCP_TOKEN }}",
			engineType:    "claude",
		},
		{
			name: "Local mode (default)",
			frontmatter: `---
engine: claude
tools:
  github:
    allowed: [list_issues, create_issue]
---`,
			expectedType:  "local",
			expectedURL:   "",
			expectedToken: "",
			engineType:    "claude",
		},
		{
			name: "Copilot remote mode with default token",
			frontmatter: `---
engine: copilot
tools:
  github:
    mode: remote
    allowed: [list_issues, create_issue]
---`,
			expectedType:  "remote",
			expectedURL:   "https://api.githubcopilot.com/mcp/",
			expectedToken: "${{ secrets.GITHUB_MCP_TOKEN }}",
			engineType:    "copilot",
		},
		{
			name: "Copilot remote mode with read-only",
			frontmatter: `---
engine: copilot
tools:
  github:
    mode: remote
    read-only: true
    allowed: [list_issues]
---`,
			expectedType:  "remote",
			expectedURL:   "https://api.githubcopilot.com/mcp/",
			expectedToken: "${{ secrets.GITHUB_MCP_TOKEN }}",
			engineType:    "copilot",
		},
		{
			name: "Codex remote mode with default token",
			frontmatter: `---
engine: codex
tools:
  github:
    mode: remote
    allowed: [list_issues, create_issue]
---`,
			expectedType:  "remote",
			expectedURL:   "https://api.githubcopilot.com/mcp/",
			expectedToken: "${{ secrets.GITHUB_MCP_TOKEN }}",
			engineType:    "codex",
		},
		{
			name: "Codex remote mode with custom token",
			frontmatter: `---
engine: codex
tools:
  github:
    mode: remote
    github-token: "${{ secrets.CUSTOM_PAT }}"
    allowed: [list_issues, create_issue]
---`,
			expectedType:  "remote",
			expectedURL:   "https://api.githubcopilot.com/mcp/",
			expectedToken: "${{ secrets.CUSTOM_PAT }}",
			engineType:    "codex",
		},
		{
			name: "Codex remote mode with read-only",
			frontmatter: `---
engine: codex
tools:
  github:
    mode: remote
    read-only: true
    allowed: [list_issues]
---`,
			expectedType:  "remote",
			expectedURL:   "https://api.githubcopilot.com/mcp/",
			expectedToken: "${{ secrets.GITHUB_MCP_TOKEN }}",
			engineType:    "codex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test GitHub Remote Mode

This is a test workflow for GitHub remote mode configuration.
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
			case "remote":
				// Codex uses TOML format, others use JSON
				if tt.engineType == "codex" {
					// Check for TOML format
					if !strings.Contains(lockContent, `type = "http"`) {
						t.Errorf("Expected HTTP configuration but didn't find 'type = \"http\"' in:\n%s", lockContent)
					}
					if tt.expectedURL != "" && !strings.Contains(lockContent, `url = "`+tt.expectedURL+`"`) {
						t.Errorf("Expected URL %s but didn't find it in:\n%s", tt.expectedURL, lockContent)
					}
					if tt.expectedToken != "" {
						if !strings.Contains(lockContent, `"Authorization" = "Bearer `+tt.expectedToken+`"`) {
							t.Errorf("Expected Authorization header with token %s but didn't find it in:\n%s", tt.expectedToken, lockContent)
						}
					}
					// Check for X-MCP-Readonly header if this is a read-only test
					if strings.Contains(tt.name, "read-only") {
						if !strings.Contains(lockContent, `"X-MCP-Readonly" = "true"`) {
							t.Errorf("Expected X-MCP-Readonly header but didn't find it in:\n%s", lockContent)
						}
					}
					// Should NOT contain Docker configuration
					if strings.Contains(lockContent, `command = "docker"`) {
						t.Errorf("Expected no Docker command but found it in:\n%s", lockContent)
					}
				} else {
					// Check for JSON format
					if !strings.Contains(lockContent, `"type": "http"`) {
						t.Errorf("Expected HTTP configuration but didn't find 'type: http' in:\n%s", lockContent)
					}
					if tt.expectedURL != "" && !strings.Contains(lockContent, tt.expectedURL) {
						t.Errorf("Expected URL %s but didn't find it in:\n%s", tt.expectedURL, lockContent)
					}
					if tt.expectedToken != "" {
						if !strings.Contains(lockContent, `"Authorization": "Bearer `+tt.expectedToken) {
							t.Errorf("Expected Authorization header with token %s but didn't find it in:\n%s", tt.expectedToken, lockContent)
						}
					}
					// Check for X-MCP-Readonly header if this is a read-only test
					if strings.Contains(tt.name, "read-only") {
						if !strings.Contains(lockContent, `"X-MCP-Readonly": "true"`) {
							t.Errorf("Expected X-MCP-Readonly header but didn't find it in:\n%s", lockContent)
						}
					}
					// Should NOT contain Docker configuration
					if strings.Contains(lockContent, `"command": "docker"`) {
						t.Errorf("Expected no Docker command but found it in:\n%s", lockContent)
					}
				}
			case "local":
				// Should contain Docker or local configuration
				switch tt.engineType {
				case "copilot":
					if !strings.Contains(lockContent, `"type": "local"`) {
						t.Errorf("Expected Copilot local type but didn't find it in:\n%s", lockContent)
					}
				case "codex":
					// Codex uses TOML format for Docker
					if !strings.Contains(lockContent, `command = "docker"`) {
						t.Errorf("Expected Docker command but didn't find it in:\n%s", lockContent)
					}
				default:
					// For Claude, check for Docker command
					if !strings.Contains(lockContent, `"command": "docker"`) {
						t.Errorf("Expected Docker command but didn't find it in:\n%s", lockContent)
					}
				}
				if !strings.Contains(lockContent, `ghcr.io/github/github-mcp-server:sha-09deac4`) {
					t.Errorf("Expected Docker image but didn't find it in:\n%s", lockContent)
				}
				// Should NOT contain HTTP type
				if tt.engineType == "codex" {
					if strings.Contains(lockContent, `type = "http"`) {
						t.Errorf("Expected no HTTP type but found it in:\n%s", lockContent)
					}
				} else {
					if strings.Contains(lockContent, `"type": "http"`) {
						t.Errorf("Expected no HTTP type but found it in:\n%s", lockContent)
					}
				}
			}
		})
	}
}

func TestGitHubRemoteModeHelperFunctions(t *testing.T) {
	t.Run("getGitHubType extracts mode correctly", func(t *testing.T) {
		githubTool := map[string]any{
			"mode":    "remote",
			"allowed": []string{"list_issues"},
		}

		githubType := getGitHubType(githubTool)
		if githubType != "remote" {
			t.Errorf("Expected mode 'remote', got '%s'", githubType)
		}
	})

	t.Run("getGitHubType returns default local when no mode", func(t *testing.T) {
		githubTool := map[string]any{
			"allowed": []string{"list_issues"},
		}

		githubType := getGitHubType(githubTool)
		if githubType != "local" {
			t.Errorf("Expected default mode 'local', got '%s'", githubType)
		}
	})

	t.Run("getGitHubToken extracts custom token correctly", func(t *testing.T) {
		githubTool := map[string]any{
			"github-token": "${{ secrets.CUSTOM_PAT }}",
			"allowed":      []string{"list_issues"},
		}

		token := getGitHubToken(githubTool)
		if token != "${{ secrets.CUSTOM_PAT }}" {
			t.Errorf("Expected token '${{ secrets.CUSTOM_PAT }}', got '%s'", token)
		}
	})

	t.Run("getGitHubToken returns empty string when no token", func(t *testing.T) {
		githubTool := map[string]any{
			"allowed": []string{"list_issues"},
		}

		token := getGitHubToken(githubTool)
		if token != "" {
			t.Errorf("Expected empty token, got '%s'", token)
		}
	})
}
