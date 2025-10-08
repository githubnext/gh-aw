package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubHTTPModeConfiguration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "github-http-test")
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
		expectedToken       string
		shouldContainDocker bool
		shouldContainHTTP   bool
	}{
		{
			name: "HTTP mode with URL and custom token",
			frontmatter: `---
engine: claude
tools:
  github:
    url: "https://api.mcp.github.com/v0/servers/github/github"
    github-token: "${{ secrets.CUSTOM_PAT }}"
    allowed: [list_issues, create_issue]
---`,
			expectedType:        "http",
			expectedURL:         "https://api.mcp.github.com/v0/servers/github/github",
			expectedToken:       "${{ secrets.CUSTOM_PAT }}",
			shouldContainDocker: false,
			shouldContainHTTP:   true,
		},
		{
			name: "HTTP mode with URL but no custom token",
			frontmatter: `---
engine: claude
tools:
  github:
    url: "https://api.mcp.github.com/v0/servers/github/github"
    allowed: [list_issues, create_issue]
---`,
			expectedType:        "http",
			expectedURL:         "https://api.mcp.github.com/v0/servers/github/github",
			expectedToken:       "",
			shouldContainDocker: false,
			shouldContainHTTP:   true,
		},
		{
			name: "Docker mode (default) with custom token",
			frontmatter: `---
engine: claude
tools:
  github:
    github-token: "${{ secrets.CUSTOM_PAT }}"
    allowed: [list_issues, create_issue]
---`,
			expectedType:        "docker",
			expectedURL:         "",
			expectedToken:       "${{ secrets.CUSTOM_PAT }}",
			shouldContainDocker: true,
			shouldContainHTTP:   false,
		},
		{
			name: "Docker mode (default) without custom token",
			frontmatter: `---
engine: claude
tools:
  github:
    allowed: [list_issues, create_issue]
---`,
			expectedType:        "docker",
			expectedURL:         "",
			expectedToken:       "",
			shouldContainDocker: true,
			shouldContainHTTP:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test GitHub HTTP Mode

This is a test workflow for GitHub HTTP mode configuration.
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
				if tt.expectedURL != "" && !strings.Contains(lockContent, tt.expectedURL) {
					t.Errorf("Expected URL %s but didn't find it in:\n%s", tt.expectedURL, lockContent)
				}
				if tt.expectedToken != "" {
					if !strings.Contains(lockContent, `"Authorization": "Bearer `+tt.expectedToken) {
						t.Errorf("Expected Authorization header with token %s but didn't find it in:\n%s", tt.expectedToken, lockContent)
					}
				}
				// Should NOT contain Docker configuration
				if strings.Contains(lockContent, `"command": "docker"`) {
					t.Errorf("Expected no Docker command but found it in:\n%s", lockContent)
				}
			case "docker":
				// Should contain Docker configuration
				if !strings.Contains(lockContent, `"command": "docker"`) {
					t.Errorf("Expected Docker command but didn't find it in:\n%s", lockContent)
				}
				if !strings.Contains(lockContent, `"ghcr.io/github/github-mcp-server:sha-09deac4"`) {
					t.Errorf("Expected Docker image but didn't find it in:\n%s", lockContent)
				}
				// Should NOT contain HTTP type
				if strings.Contains(lockContent, `"type": "http"`) {
					t.Errorf("Expected no HTTP type but found it in:\n%s", lockContent)
				}
				// Check for custom token in Docker mode
				if tt.expectedToken != "" {
					if !strings.Contains(lockContent, `"GITHUB_PERSONAL_ACCESS_TOKEN": "`+tt.expectedToken) {
						t.Errorf("Expected custom token %s in Docker mode but didn't find it in:\n%s", tt.expectedToken, lockContent)
					}
				} else {
					// Should use default token expression
					if !strings.Contains(lockContent, `"GITHUB_PERSONAL_ACCESS_TOKEN": "${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}"`) {
						t.Errorf("Expected default token expression but didn't find it in:\n%s", lockContent)
					}
				}
			}
		})
	}
}

func TestGitHubHTTPModeHelperFunctions(t *testing.T) {
	t.Run("getGitHubURL extracts URL correctly", func(t *testing.T) {
		githubTool := map[string]any{
			"url":     "https://api.mcp.github.com/v0/servers/github/github",
			"allowed": []string{"list_issues"},
		}

		url := getGitHubURL(githubTool)
		if url != "https://api.mcp.github.com/v0/servers/github/github" {
			t.Errorf("Expected URL 'https://api.mcp.github.com/v0/servers/github/github', got '%s'", url)
		}
	})

	t.Run("getGitHubURL returns empty string when no URL", func(t *testing.T) {
		githubTool := map[string]any{
			"allowed": []string{"list_issues"},
		}

		url := getGitHubURL(githubTool)
		if url != "" {
			t.Errorf("Expected empty URL, got '%s'", url)
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
