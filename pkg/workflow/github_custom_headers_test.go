package workflow

import (
	"strings"
	"testing"
)

func TestGetGitHubHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected map[string]string
	}{
		{
			name: "Extract headers from map[string]any",
			input: map[string]any{
				"mode": "remote",
				"headers": map[string]any{
					"Authorization":  "Bearer ${{ secrets.MCP_PAT_GITHUB }}",
					"X-Custom-Value": "test-value",
				},
			},
			expected: map[string]string{
				"Authorization":  "Bearer ${{ secrets.MCP_PAT_GITHUB }}",
				"X-Custom-Value": "test-value",
			},
		},
		{
			name: "Extract headers from map[string]string",
			input: map[string]any{
				"mode": "remote",
				"headers": map[string]string{
					"Authorization": "Bearer ${{ secrets.TOKEN }}",
				},
			},
			expected: map[string]string{
				"Authorization": "Bearer ${{ secrets.TOKEN }}",
			},
		},
		{
			name:     "No headers field",
			input:    map[string]any{"mode": "remote"},
			expected: nil,
		},
		{
			name:     "Non-map input",
			input:    "github",
			expected: nil,
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGitHubHeaders(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d headers, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected header %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("For header %s, expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestRenderGitHubMCPRemoteConfigWithCustomHeaders(t *testing.T) {
	tests := []struct {
		name          string
		options       GitHubMCPRemoteOptions
		expectedParts []string
		notExpected   []string
	}{
		{
			name: "Custom headers merged with built-in headers",
			options: GitHubMCPRemoteOptions{
				ReadOnly:           false,
				Lockdown:           false,
				Toolsets:           "all",
				AuthorizationValue: "Bearer \\${GITHUB_PERSONAL_ACCESS_TOKEN}",
				IncludeToolsField:  true,
				AllowedTools:       []string{"*"},
				IncludeEnvSection:  true,
				CustomHeaders: map[string]string{
					"X-Custom-Header": "custom-value",
					"X-Another":       "${{ secrets.MY_SECRET }}",
				},
			},
			expectedParts: []string{
				"\"type\": \"http\"",
				"\"url\": \"https://api.githubcopilot.com/mcp/\"",
				"\"headers\": {",
				"\"Authorization\": \"Bearer \\${GITHUB_PERSONAL_ACCESS_TOKEN}\"",
				"\"X-MCP-Toolsets\": \"all\"",
				"\"X-Custom-Header\": \"custom-value\"",
				"\"X-Another\": \"${{ secrets.MY_SECRET }}\"",
				"\"tools\": [",
				"\"*\"",
				"\"env\": {",
				"\"GITHUB_PERSONAL_ACCESS_TOKEN\": \"\\${GITHUB_MCP_SERVER_TOKEN}\"",
			},
		},
		{
			name: "Custom Authorization header overridden by built-in",
			options: GitHubMCPRemoteOptions{
				AuthorizationValue: "Bearer \\${GITHUB_PERSONAL_ACCESS_TOKEN}",
				CustomHeaders: map[string]string{
					"Authorization": "Bearer custom-token",
				},
			},
			expectedParts: []string{
				"\"Authorization\": \"Bearer \\${GITHUB_PERSONAL_ACCESS_TOKEN}\"",
			},
			notExpected: []string{
				"\"Authorization\": \"Bearer custom-token\"",
			},
		},
		{
			name: "No custom headers",
			options: GitHubMCPRemoteOptions{
				ReadOnly:           true,
				AuthorizationValue: "Bearer token",
				CustomHeaders:      nil,
			},
			expectedParts: []string{
				"\"Authorization\": \"Bearer token\"",
				"\"X-MCP-Readonly\": \"true\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			RenderGitHubMCPRemoteConfig(&yaml, tt.options)
			result := yaml.String()

			for _, expected := range tt.expectedParts {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nGot:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(result, notExpected) {
					t.Errorf("Expected output NOT to contain %q, but it did.\nGot:\n%s", notExpected, result)
				}
			}
		})
	}
}

func TestCollectHTTPMCPHeaderSecretsWithGitHubHeaders(t *testing.T) {
	tools := map[string]any{
		"github": map[string]any{
			"mode": "remote",
			"headers": map[string]string{
				"Authorization":  "Bearer ${{ secrets.MCP_PAT_GITHUB }}",
				"X-Custom-Token": "${{ secrets.CUSTOM_TOKEN }}",
				"X-Static":       "static-value",
			},
		},
		"other-tool": map[string]any{
			"type": "http",
			"url":  "https://example.com",
			"headers": map[string]string{
				"API-Key": "${{ secrets.API_KEY }}",
			},
		},
	}

	result := collectHTTPMCPHeaderSecrets(tools)

	expected := map[string]string{
		"MCP_PAT_GITHUB": "${{ secrets.MCP_PAT_GITHUB }}",
		"CUSTOM_TOKEN":   "${{ secrets.CUSTOM_TOKEN }}",
		"API_KEY":        "${{ secrets.API_KEY }}",
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d secrets, got %d", len(expected), len(result))
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists {
			t.Errorf("Expected secret %s not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("For secret %s, expected %q, got %q", key, expectedValue, actualValue)
		}
	}
}
