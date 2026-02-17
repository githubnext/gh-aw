//go:build !integration

package workflow

import (
	"testing"
)

func TestGetEffectiveGitHubToken(t *testing.T) {
	tests := []struct {
		name        string
		customToken string
		expected    string
	}{
		{
			name:        "custom token has highest precedence",
			customToken: "${{ secrets.CUSTOM_TOKEN }}",
			expected:    "${{ secrets.CUSTOM_TOKEN }}",
		},
		{
			name:        "default fallback includes GH_AW_GITHUB_MCP_SERVER_TOKEN (for MCP and tools)",
			customToken: "",
			expected:    "${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEffectiveGitHubToken(tt.customToken)
			if result != tt.expected {
				t.Errorf("getEffectiveGitHubToken() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetEffectiveSafeOutputGitHubToken(t *testing.T) {
	tests := []struct {
		name        string
		customToken string
		expected    string
	}{
		{
			name:        "custom token has highest precedence",
			customToken: "${{ secrets.CUSTOM_TOKEN }}",
			expected:    "${{ secrets.CUSTOM_TOKEN }}",
		},
		{
			name:        "default fallback includes GH_AW_GITHUB_TOKEN (safe outputs chain)",
			customToken: "",
			expected:    "${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEffectiveSafeOutputGitHubToken(tt.customToken)
			if result != tt.expected {
				t.Errorf("getEffectiveSafeOutputGitHubToken() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetEffectiveCopilotGitHubToken(t *testing.T) {
	tests := []struct {
		name        string
		customToken string
		expected    string
	}{
		{
			name:        "custom token has highest precedence",
			customToken: "${{ secrets.CUSTOM_COPILOT_TOKEN }}",
			expected:    "${{ secrets.CUSTOM_COPILOT_TOKEN }}",
		},
		{
			name:        "default fallback for Copilot",
			customToken: "",
			expected:    "${{ secrets.COPILOT_GITHUB_TOKEN }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEffectiveCopilotRequestsToken(tt.customToken)
			if result != tt.expected {
				t.Errorf("getEffectiveCopilotRequestsToken() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetEffectiveAgentGitHubToken(t *testing.T) {
	tests := []struct {
		name        string
		customToken string
		expected    string
	}{
		{
			name:        "custom token has highest precedence",
			customToken: "${{ secrets.CUSTOM_AGENT_TOKEN }}",
			expected:    "${{ secrets.CUSTOM_AGENT_TOKEN }}",
		},
		{
			name:        "default fallback chain for agent operations",
			customToken: "",
			expected:    "${{ secrets.GH_AW_AGENT_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEffectiveCopilotCodingAgentGitHubToken(tt.customToken)
			if result != tt.expected {
				t.Errorf("getEffectiveCopilotCodingAgentGitHubToken() = %q, want %q", result, tt.expected)
			}
		})
	}
}
