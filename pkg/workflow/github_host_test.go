//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeGitHubHost(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bare GHES hostname",
			input:    "ghes.example.com",
			expected: "https://ghes.example.com",
		},
		{
			name:     "GHES hostname with https scheme",
			input:    "https://ghes.example.com",
			expected: "https://ghes.example.com",
		},
		{
			name:     "GHES hostname with trailing slash",
			input:    "ghes.example.com/",
			expected: "https://ghes.example.com",
		},
		{
			name:     "GHES hostname with https and trailing slash",
			input:    "https://ghes.example.com/",
			expected: "https://ghes.example.com",
		},
		{
			name:     "default github.com bare - ignored",
			input:    "github.com",
			expected: "",
		},
		{
			name:     "default github.com with scheme - ignored",
			input:    "https://github.com",
			expected: "",
		},
		{
			name:     "api.github.com - ignored",
			input:    "api.github.com",
			expected: "",
		},
		{
			name:     "api.github.com with scheme - ignored",
			input:    "https://api.github.com",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "hostname with http scheme",
			input:    "http://ghes.example.com",
			expected: "http://ghes.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeGitHubHost(tt.input)
			assert.Equal(t, tt.expected, result, "normalizeGitHubHost(%q)", tt.input)
		})
	}
}

func TestGetGitHubHost(t *testing.T) {
	tests := []struct {
		name       string
		githubTool any
		expected   string
	}{
		{
			name: "host field set",
			githubTool: map[string]any{
				"host": "ghes.example.com",
			},
			expected: "https://ghes.example.com",
		},
		{
			name: "host field with scheme",
			githubTool: map[string]any{
				"host": "https://ghes.example.com",
			},
			expected: "https://ghes.example.com",
		},
		{
			name: "host field set to github.com - returns empty",
			githubTool: map[string]any{
				"host": "github.com",
			},
			expected: "",
		},
		{
			name:       "no host field",
			githubTool: map[string]any{"mode": "local"},
			expected:   "",
		},
		{
			name:       "nil tool",
			githubTool: nil,
			expected:   "",
		},
		{
			name:       "string tool config",
			githubTool: "default",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGitHubHost(tt.githubTool)
			assert.Equal(t, tt.expected, result, "getGitHubHost with tool=%v", tt.githubTool)
		})
	}
}

func TestRenderGitHubMCPDockerConfig_WithHost(t *testing.T) {
	tests := []struct {
		name     string
		options  GitHubMCPDockerOptions
		expected []string
		notFound []string
	}{
		{
			name: "GHES host emits GITHUB_HOST",
			options: GitHubMCPDockerOptions{
				Toolsets:           "default",
				DockerImageVersion: "latest",
				Host:               "https://ghes.example.com",
			},
			expected: []string{
				`"GITHUB_HOST": "https://ghes.example.com"`,
				`"GITHUB_PERSONAL_ACCESS_TOKEN": "$GITHUB_MCP_SERVER_TOKEN"`,
			},
			notFound: nil,
		},
		{
			name: "no host - GITHUB_HOST absent",
			options: GitHubMCPDockerOptions{
				Toolsets:           "default",
				DockerImageVersion: "latest",
				Host:               "",
			},
			expected: []string{
				`"GITHUB_PERSONAL_ACCESS_TOKEN": "$GITHUB_MCP_SERVER_TOKEN"`,
			},
			notFound: []string{
				`"GITHUB_HOST"`,
			},
		},
		{
			name: "GHES host sorted before GITHUB_TOOLSETS",
			options: GitHubMCPDockerOptions{
				Toolsets:           "repos",
				DockerImageVersion: "latest",
				Host:               "https://ghes.example.com",
			},
			expected: []string{
				`"GITHUB_HOST": "https://ghes.example.com"`,
				`"GITHUB_TOOLSETS": "repos"`,
			},
			notFound: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			RenderGitHubMCPDockerConfig(&out, tt.options)
			output := out.String()

			for _, want := range tt.expected {
				assert.Contains(t, output, want, "expected output to contain %q\nOutput:\n%s", want, output)
			}
			for _, notWant := range tt.notFound {
				assert.NotContains(t, output, notWant, "expected output NOT to contain %q\nOutput:\n%s", notWant, output)
			}
		})
	}
}
