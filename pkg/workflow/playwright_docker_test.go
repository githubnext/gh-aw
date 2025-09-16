package workflow

import (
	"strings"
	"testing"
)

func TestCodexEnginePlaywrightDockerGeneration(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name        string
		tool        any
		expected    map[string]string // Expected parts of the config
		expectError bool
	}{
		{
			name: "playwright with default version",
			tool: map[string]any{},
			expected: map[string]string{
				"command":   "command = \"docker\"",
				"args_run":  "\"run\",",
				"args_rm":   "\"--rm\",",
				"args_shm":  "\"--shm-size=2gb\",",
				"args_cap":  "\"--cap-add=SYS_ADMIN\",",
				"image":     "\"mcr.microsoft.com/playwright:latest\"",
				"env_var":   "\"PLAYWRIGHT_ALLOWED_DOMAINS\"",
				"env_value": "\"localhost,127.0.0.1\"",
			},
		},
		{
			name: "playwright with custom version",
			tool: map[string]any{
				"docker_image_version": "v1.41.0",
			},
			expected: map[string]string{
				"command": "command = \"docker\"",
				"image":   "\"mcr.microsoft.com/playwright:v1.41.0\"",
			},
		},
		{
			name: "playwright with custom domains",
			tool: map[string]any{
				"allowed_domains": []any{"github.com", "*.example.com"},
			},
			expected: map[string]string{
				"command":   "command = \"docker\"",
				"env_value": "\"github.com,*.example.com\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			networkPermissions := &NetworkPermissions{}

			engine.renderPlaywrightCodexMCPConfig(&yaml, tt.tool, networkPermissions)
			result := yaml.String()

			// Verify we're not using npx anymore
			if strings.Contains(result, "npx") {
				t.Error("Expected Docker configuration, but found npx")
			}

			if strings.Contains(result, "@playwright/mcp") {
				t.Error("Expected Docker configuration, but found @playwright/mcp")
			}

			// Verify expected parts are present
			for key, expectedPart := range tt.expected {
				if !strings.Contains(result, expectedPart) {
					t.Errorf("Expected %s to contain '%s', but it was missing. Full output:\n%s", key, expectedPart, result)
				}
			}

			// Verify Docker-specific arguments are present
			dockerArgs := []string{
				"\"run\"",
				"\"-i\"",
				"\"--rm\"",
				"\"--shm-size=2gb\"",
				"\"--cap-add=SYS_ADMIN\"",
			}

			for _, arg := range dockerArgs {
				if !strings.Contains(result, arg) {
					t.Errorf("Expected Docker arg '%s' to be present. Full output:\n%s", arg, result)
				}
			}
		})
	}
}

func TestClaudeEnginePlaywrightDockerGeneration(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name        string
		tool        any
		expected    map[string]string // Expected parts of the config
		expectError bool
	}{
		{
			name: "playwright with default version",
			tool: map[string]any{},
			expected: map[string]string{
				"command":  "\"command\": \"docker\",",
				"args_run": "\"run\",",
				"args_rm":  "\"--rm\",",
				"args_shm": "\"--shm-size=2gb\",",
				"args_cap": "\"--cap-add=SYS_ADMIN\",",
				"image":    "\"mcr.microsoft.com/playwright:latest\"",
				"env_var":  "\"PLAYWRIGHT_ALLOWED_DOMAINS\": \"localhost,127.0.0.1\"",
			},
		},
		{
			name: "playwright with custom version",
			tool: map[string]any{
				"docker_image_version": "v1.41.0",
			},
			expected: map[string]string{
				"command": "\"command\": \"docker\",",
				"image":   "\"mcr.microsoft.com/playwright:v1.41.0\"",
			},
		},
		{
			name: "playwright with custom domains",
			tool: map[string]any{
				"allowed_domains": []any{"github.com", "*.example.com"},
			},
			expected: map[string]string{
				"command":   "\"command\": \"docker\",",
				"env_value": "\"PLAYWRIGHT_ALLOWED_DOMAINS\": \"github.com,*.example.com\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			networkPermissions := &NetworkPermissions{}

			engine.renderPlaywrightMCPConfig(&yaml, tt.tool, false, networkPermissions)
			result := yaml.String()

			// Verify we're not using npx anymore
			if strings.Contains(result, "npx") {
				t.Error("Expected Docker configuration, but found npx")
			}

			if strings.Contains(result, "@playwright/mcp") {
				t.Error("Expected Docker configuration, but found @playwright/mcp")
			}

			// Verify expected parts are present
			for key, expectedPart := range tt.expected {
				if !strings.Contains(result, expectedPart) {
					t.Errorf("Expected %s to contain '%s', but it was missing. Full output:\n%s", key, expectedPart, result)
				}
			}

			// Verify Docker-specific arguments are present
			dockerArgs := []string{
				"\"run\"",
				"\"-i\"",
				"\"--rm\"",
				"\"--shm-size=2gb\"",
				"\"--cap-add=SYS_ADMIN\"",
			}

			for _, arg := range dockerArgs {
				if !strings.Contains(result, arg) {
					t.Errorf("Expected Docker arg '%s' to be present. Full output:\n%s", arg, result)
				}
			}
		})
	}
}

func TestPlaywrightDockerEnvironmentVariables(t *testing.T) {
	codexEngine := NewCodexEngine()
	claudeEngine := NewClaudeEngine()

	tests := []struct {
		name                string
		tool                any
		expectedDomains     string
		expectedImageFormat string
	}{
		{
			name:                "default localhost configuration",
			tool:                map[string]any{},
			expectedDomains:     "localhost,127.0.0.1",
			expectedImageFormat: "mcr.microsoft.com/playwright:latest",
		},
		{
			name: "custom domains configuration",
			tool: map[string]any{
				"allowed_domains": []any{"github.com", "api.github.com"},
			},
			expectedDomains:     "github.com,api.github.com",
			expectedImageFormat: "mcr.microsoft.com/playwright:latest",
		},
		{
			name: "custom version and domains",
			tool: map[string]any{
				"docker_image_version": "v1.41.0",
				"allowed_domains":      []any{"example.com"},
			},
			expectedDomains:     "example.com",
			expectedImageFormat: "mcr.microsoft.com/playwright:v1.41.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (codex)", func(t *testing.T) {
			var yaml strings.Builder
			networkPermissions := &NetworkPermissions{}

			codexEngine.renderPlaywrightCodexMCPConfig(&yaml, tt.tool, networkPermissions)
			result := yaml.String()

			// Check environment variable configuration
			expectedEnvConfig := "env = { \"PLAYWRIGHT_ALLOWED_DOMAINS\" = \"" + tt.expectedDomains + "\" }"
			if !strings.Contains(result, expectedEnvConfig) {
				t.Errorf("Expected environment config '%s' not found. Full output:\n%s", expectedEnvConfig, result)
			}

			// Check image version
			if !strings.Contains(result, tt.expectedImageFormat) {
				t.Errorf("Expected image format '%s' not found. Full output:\n%s", tt.expectedImageFormat, result)
			}
		})

		t.Run(tt.name+" (claude)", func(t *testing.T) {
			var yaml strings.Builder
			networkPermissions := &NetworkPermissions{}

			claudeEngine.renderPlaywrightMCPConfig(&yaml, tt.tool, false, networkPermissions)
			result := yaml.String()

			// Check environment variable configuration
			expectedEnvConfig := "\"PLAYWRIGHT_ALLOWED_DOMAINS\": \"" + tt.expectedDomains + "\""
			if !strings.Contains(result, expectedEnvConfig) {
				t.Errorf("Expected environment config '%s' not found. Full output:\n%s", expectedEnvConfig, result)
			}

			// Check image version
			if !strings.Contains(result, tt.expectedImageFormat) {
				t.Errorf("Expected image format '%s' not found. Full output:\n%s", tt.expectedImageFormat, result)
			}
		})
	}
}
