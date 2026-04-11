//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasGitHubOIDCAuthInTools(t *testing.T) {
	tests := []struct {
		name     string
		tools    map[string]any
		expected bool
	}{
		{
			name:     "empty tools",
			tools:    map[string]any{},
			expected: false,
		},
		{
			name: "only standard tools (github, playwright)",
			tools: map[string]any{
				"github":     map[string]any{},
				"playwright": map[string]any{},
			},
			expected: false,
		},
		{
			name: "http server with headers but no auth",
			tools: map[string]any{
				"tavily": map[string]any{
					"type": "http",
					"url":  "https://mcp.tavily.com/mcp/",
					"headers": map[string]any{
						"Authorization": "Bearer ${{ secrets.TAVILY_API_KEY }}",
					},
				},
			},
			expected: false,
		},
		{
			name: "http server with github-oidc auth",
			tools: map[string]any{
				"oidc-server": map[string]any{
					"type": "http",
					"url":  "https://my-server.example.com/mcp",
					"auth": map[string]any{
						"type":     "github-oidc",
						"audience": "https://my-server.example.com",
					},
				},
			},
			expected: true,
		},
		{
			name: "http server with github-oidc auth no audience",
			tools: map[string]any{
				"oidc-server": map[string]any{
					"type": "http",
					"url":  "https://my-server.example.com/mcp",
					"auth": map[string]any{
						"type": "github-oidc",
					},
				},
			},
			expected: true,
		},
		{
			name: "mixed servers with one oidc",
			tools: map[string]any{
				"github": map[string]any{},
				"tavily": map[string]any{
					"type": "http",
					"url":  "https://mcp.tavily.com/mcp/",
					"headers": map[string]any{
						"Authorization": "Bearer ${{ secrets.TAVILY_API_KEY }}",
					},
				},
				"oidc-server": map[string]any{
					"type": "http",
					"url":  "https://my-server.example.com/mcp",
					"auth": map[string]any{
						"type": "github-oidc",
					},
				},
			},
			expected: true,
		},
		{
			name: "stdio server is not treated as oidc",
			tools: map[string]any{
				"my-stdio": map[string]any{
					"type":      "stdio",
					"container": "mcp/server:latest",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasGitHubOIDCAuthInTools(tt.tools)
			assert.Equal(t, tt.expected, result, "hasGitHubOIDCAuthInTools should return %v", tt.expected)
		})
	}
}
