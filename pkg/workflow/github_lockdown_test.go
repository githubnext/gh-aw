//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEffectiveGitHubGuardPolicies(t *testing.T) {
	tests := []struct {
		name       string
		githubTool any
		wantRepos  string
		wantInteg  string
	}{
		{
			name: "default policy when no guard policy configured",
			githubTool: map[string]any{
				"mode": "local",
			},
			wantRepos: "all",
			wantInteg: "approved",
		},
		{
			name:       "default policy when github tool is empty",
			githubTool: map[string]any{},
			wantRepos:  "all",
			wantInteg:  "approved",
		},
		{
			name: "explicit repos and min-integrity override default",
			githubTool: map[string]any{
				"repos":         "public",
				"min-integrity": "merged",
			},
			wantRepos: "public",
			wantInteg: "merged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEffectiveGitHubGuardPolicies(tt.githubTool)
			assert.NotNil(t, result, "guard policies should not be nil")

			allowOnly, ok := result["allow-only"].(map[string]any)
			assert.True(t, ok, "allow-only should be a map")

			assert.Equal(t, tt.wantRepos, allowOnly["repos"], "repos should match")
			assert.Equal(t, tt.wantInteg, allowOnly["min-integrity"], "min-integrity should match")
		})
	}
}

func TestRenderGitHubMCPDockerConfigNoLockdown(t *testing.T) {
	tests := []struct {
		name     string
		options  GitHubMCPDockerOptions
		expected []string
		notFound []string
	}{
		{
			name: "Docker mode does not render GITHUB_LOCKDOWN_MODE",
			options: GitHubMCPDockerOptions{
				ReadOnly:           false,
				Toolsets:           "default",
				DockerImageVersion: "latest",
				IncludeTypeField:   true,
			},
			expected: []string{
				`"type": "stdio"`,
				`"GITHUB_TOOLSETS": "default"`,
				`"container": "ghcr.io/github/github-mcp-server:latest"`,
			},
			notFound: []string{
				`"GITHUB_LOCKDOWN_MODE"`,
			},
		},
		{
			name: "Docker mode with read-only does not render lockdown",
			options: GitHubMCPDockerOptions{
				ReadOnly:           true,
				Toolsets:           "default",
				DockerImageVersion: "v1.0.0",
				IncludeTypeField:   false,
			},
			expected: []string{
				`"GITHUB_READ_ONLY": "1"`,
				`"container": "ghcr.io/github/github-mcp-server:v1.0.0"`,
			},
			notFound: []string{
				`"GITHUB_LOCKDOWN_MODE"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			RenderGitHubMCPDockerConfig(&yaml, tt.options)
			output := yaml.String()

			for _, expected := range tt.expected {
				assert.Contains(t, output, expected, "should contain %q", expected)
			}

			for _, notFound := range tt.notFound {
				assert.NotContains(t, output, notFound, "should not contain %q", notFound)
			}
		})
	}
}

func TestRenderGitHubMCPRemoteConfigNoLockdown(t *testing.T) {
	tests := []struct {
		name     string
		options  GitHubMCPRemoteOptions
		expected []string
		notFound []string
	}{
		{
			name: "Remote mode does not render X-MCP-Lockdown",
			options: GitHubMCPRemoteOptions{
				ReadOnly:           false,
				Toolsets:           "default",
				AuthorizationValue: "Bearer test-token",
				IncludeToolsField:  true,
				AllowedTools:       []string{"*"},
				IncludeEnvSection:  false,
			},
			expected: []string{
				`"type": "http"`,
				`"X-MCP-Toolsets": "default"`,
				`"Authorization": "Bearer test-token"`,
			},
			notFound: []string{
				`"X-MCP-Lockdown"`,
				`"X-MCP-Readonly"`,
			},
		},
		{
			name: "Remote mode with read-only does not render X-MCP-Lockdown",
			options: GitHubMCPRemoteOptions{
				ReadOnly:           true,
				Toolsets:           "repos,issues",
				AuthorizationValue: "Bearer test-token",
				IncludeToolsField:  false,
				AllowedTools:       nil,
				IncludeEnvSection:  false,
			},
			expected: []string{
				`"type": "http"`,
				`"X-MCP-Readonly": "true"`,
				`"X-MCP-Toolsets": "repos,issues"`,
			},
			notFound: []string{
				`"X-MCP-Lockdown"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			RenderGitHubMCPRemoteConfig(&yaml, tt.options)
			output := yaml.String()

			for _, expected := range tt.expected {
				assert.Contains(t, output, expected, "should contain %q", expected)
			}

			for _, notFound := range tt.notFound {
				assert.NotContains(t, output, notFound, "should not contain %q", notFound)
			}
		})
	}
}
