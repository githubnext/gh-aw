package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestVersionField(t *testing.T) {
	// Test GitHub tool version extraction
	t.Run("GitHub version field extraction", func(t *testing.T) {
		// Test "version" field
		githubTool := map[string]any{
			"allowed": []any{"create_issue"},
			"version": "v2.0.0",
		}
		result := getGitHubDockerImageVersion(githubTool)
		if result != "v2.0.0" {
			t.Errorf("Expected v2.0.0, got %s", result)
		}

		// Test default value when version field is not present
		githubToolDefault := map[string]any{
			"allowed": []any{"create_issue"},
		}
		result = getGitHubDockerImageVersion(githubToolDefault)
		if result != constants.DefaultGitHubMCPServerVersion {
			t.Errorf("Expected default %s, got %s", constants.DefaultGitHubMCPServerVersion, result)
		}
	})

	// Test Playwright tool version extraction
	t.Run("Playwright version field extraction", func(t *testing.T) {
		// Test "version" field
		playwrightTool := map[string]any{
			"allowed_domains": []any{"example.com"},
			"version":         "v1.41.0",
		}
		result := getPlaywrightDockerImageVersion(playwrightTool)
		if result != "v1.41.0" {
			t.Errorf("Expected v1.41.0, got %s", result)
		}

		// Test default value when version field is not present
		playwrightToolDefault := map[string]any{
			"allowed_domains": []any{"example.com"},
		}
		result = getPlaywrightDockerImageVersion(playwrightToolDefault)
		if result != "latest" {
			t.Errorf("Expected default latest, got %s", result)
		}
	})

	// Test MCP parser integration
	t.Run("MCP parser version field integration", func(t *testing.T) {
		// Test GitHub tool with "version" field
		frontmatter := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{
					"allowed": []any{"create_issue"},
					"version": "v2.0.0",
				},
			},
		}

		configs, err := parser.ExtractMCPConfigurations(frontmatter, "")
		if err != nil {
			t.Fatalf("Error parsing with version field: %v", err)
		}

		if len(configs) == 0 {
			t.Fatal("No configs returned")
		}

		found := false
		for _, arg := range configs[0].Args {
			if strings.Contains(arg, "ghcr.io/github/github-mcp-server:v2.0.0") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find v2.0.0 in args, got: %v", configs[0].Args)
		}

		// Test Playwright tool with "version" field
		frontmatterPlaywright := map[string]any{
			"tools": map[string]any{
				"playwright": map[string]any{
					"allowed_domains": []any{"example.com"},
					"version":         "v1.41.0",
				},
			},
		}

		configs, err = parser.ExtractMCPConfigurations(frontmatterPlaywright, "")
		if err != nil {
			t.Fatalf("Error parsing Playwright with version field: %v", err)
		}

		if len(configs) == 0 {
			t.Fatal("No configs returned")
		}

		found = false
		for _, arg := range configs[0].Args {
			if strings.Contains(arg, "mcr.microsoft.com/playwright:v1.41.0") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find v1.41.0 in args, got: %v", configs[0].Args)
		}
	})
}
