package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestVersionFieldBackwardCompatibility(t *testing.T) {
	// Test GitHub tool version extraction
	t.Run("GitHub version field extraction", func(t *testing.T) {
		// Test new "version" field
		githubToolNew := map[string]any{
			"allowed": []any{"create_issue"},
			"version": "v2.0.0",
		}
		result := getGitHubDockerImageVersion(githubToolNew)
		if result != "v2.0.0" {
			t.Errorf("Expected v2.0.0, got %s", result)
		}

		// Test legacy "version" field
		githubToolLegacy := map[string]any{
			"allowed": []any{"create_issue"},
			"version": "v1.5.0",
		}
		result = getGitHubDockerImageVersion(githubToolLegacy)
		if result != "v1.5.0" {
			t.Errorf("Expected v1.5.0, got %s", result)
		}

		// Test precedence: "version" should override "docker_image_version"
		githubToolBoth := map[string]any{
			"allowed":              []any{"create_issue"},
			"version":              "v3.0.0",
			"docker_image_version": "v1.5.0",
		}
		result = getGitHubDockerImageVersion(githubToolBoth)
		if result != "v3.0.0" {
			t.Errorf("Expected v3.0.0 (version should take precedence), got %s", result)
		}

		// Test default value when neither field is present
		githubToolDefault := map[string]any{
			"allowed": []any{"create_issue"},
		}
		result = getGitHubDockerImageVersion(githubToolDefault)
		if result != "sha-09deac4" {
			t.Errorf("Expected default sha-09deac4, got %s", result)
		}
	})

	// Test Playwright tool version extraction
	t.Run("Playwright version field extraction", func(t *testing.T) {
		// Test new "version" field
		playwrightToolNew := map[string]any{
			"allowed_domains": []any{"example.com"},
			"version":         "v1.41.0",
		}
		result := getPlaywrightDockerImageVersion(playwrightToolNew)
		if result != "v1.41.0" {
			t.Errorf("Expected v1.41.0, got %s", result)
		}

		// Test legacy "version" field
		playwrightToolLegacy := map[string]any{
			"allowed_domains": []any{"example.com"},
			"version":         "v1.40.0",
		}
		result = getPlaywrightDockerImageVersion(playwrightToolLegacy)
		if result != "v1.40.0" {
			t.Errorf("Expected v1.40.0, got %s", result)
		}

		// Test precedence: "version" should override "docker_image_version"
		playwrightToolBoth := map[string]any{
			"allowed_domains":      []any{"example.com"},
			"version":              "v1.42.0",
			"docker_image_version": "v1.40.0",
		}
		result = getPlaywrightDockerImageVersion(playwrightToolBoth)
		if result != "v1.42.0" {
			t.Errorf("Expected v1.42.0 (version should take precedence), got %s", result)
		}

		// Test default value when neither field is present
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
		// Test GitHub tool with new "version" field
		frontmatterNew := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{
					"allowed": []any{"create_issue"},
					"version": "v2.0.0",
				},
			},
		}

		configs, err := parser.ExtractMCPConfigurations(frontmatterNew, "")
		if err != nil {
			t.Fatalf("Error parsing with new field: %v", err)
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

		// Test GitHub tool with legacy "version" field
		frontmatterLegacy := map[string]any{
			"tools": map[string]any{
				"github": map[string]any{
					"allowed": []any{"create_issue"},
					"version": "v1.5.0",
				},
			},
		}

		configs, err = parser.ExtractMCPConfigurations(frontmatterLegacy, "")
		if err != nil {
			t.Fatalf("Error parsing with legacy field: %v", err)
		}

		if len(configs) == 0 {
			t.Fatal("No configs returned")
		}

		found = false
		for _, arg := range configs[0].Args {
			if strings.Contains(arg, "ghcr.io/github/github-mcp-server:v1.5.0") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find v1.5.0 in args, got: %v", configs[0].Args)
		}

		// Test Playwright tool with new "version" field
		frontmatterPlaywrightNew := map[string]any{
			"tools": map[string]any{
				"playwright": map[string]any{
					"allowed_domains": []any{"example.com"},
					"version":         "v1.41.0",
				},
			},
		}

		configs, err = parser.ExtractMCPConfigurations(frontmatterPlaywrightNew, "")
		if err != nil {
			t.Fatalf("Error parsing Playwright with new field: %v", err)
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
