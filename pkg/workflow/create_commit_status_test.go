package workflow

import (
	"testing"
)

func TestParseCommitStatusConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected *CreateCommitStatusConfig
	}{
		{
			name:     "nil when not configured",
			input:    map[string]any{},
			expected: nil,
		},
		{
			name: "empty config with defaults",
			input: map[string]any{
				"create-commit-status": map[string]any{},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1, // Default max is 1
				},
			},
		},
		{
			name: "config with custom context",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"context": "custom-status",
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				Context: "custom-status",
			},
		},
		{
			name: "config with custom max",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"max": 3,
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1, // Max is always enforced to 1 for commit status
				},
			},
		},
		{
			name: "config with allowed-domains",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"allowed-domains": []any{"example.com", "*.trusted.org"},
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				AllowedDomains: []string{"example.com", "*.trusted.org"},
			},
		},
		{
			name: "config with github-token",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"github-token": "${{ secrets.CUSTOM_TOKEN }}",
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max:         1,
					GitHubToken: "${{ secrets.CUSTOM_TOKEN }}",
				},
			},
		},
		{
			name: "full config",
			input: map[string]any{
				"create-commit-status": map[string]any{
					"context":      "ci/build",
					"max":          2,
					"github-token": "${{ secrets.TOKEN }}",
				},
			},
			expected: &CreateCommitStatusConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max:         1, // Max is always enforced to 1 for commit status
					GitHubToken: "${{ secrets.TOKEN }}",
				},
				Context: "ci/build",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{}
			result := c.parseCommitStatusConfig(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Expected non-nil result")
			}

			if result.Max != tt.expected.Max {
				t.Errorf("Max: expected %d, got %d", tt.expected.Max, result.Max)
			}

			if result.Context != tt.expected.Context {
				t.Errorf("Context: expected %q, got %q", tt.expected.Context, result.Context)
			}

			if result.GitHubToken != tt.expected.GitHubToken {
				t.Errorf("GitHubToken: expected %q, got %q", tt.expected.GitHubToken, result.GitHubToken)
			}

			// Check AllowedDomains
			if len(result.AllowedDomains) != len(tt.expected.AllowedDomains) {
				t.Errorf("AllowedDomains length: expected %d, got %d", len(tt.expected.AllowedDomains), len(result.AllowedDomains))
			} else {
				for i, domain := range tt.expected.AllowedDomains {
					if result.AllowedDomains[i] != domain {
						t.Errorf("AllowedDomains[%d]: expected %q, got %q", i, domain, result.AllowedDomains[i])
					}
				}
			}
		})
	}
}

// Note: buildCreateCommitStatusJob function was removed as the implementation
// now uses pending/final status lifecycle instead of a separate safe-output job

func TestHasSafeOutputsEnabledWithCommitStatus(t *testing.T) {
	tests := []struct {
		name     string
		config   *SafeOutputsConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty config",
			config:   &SafeOutputsConfig{},
			expected: false,
		},
		{
			name: "only create-commit-status enabled",
			config: &SafeOutputsConfig{
				CreateCommitStatus: &CreateCommitStatusConfig{},
			},
			expected: true,
		},
		{
			name: "multiple safe outputs enabled",
			config: &SafeOutputsConfig{
				CreateIssues:       &CreateIssuesConfig{},
				CreateCommitStatus: &CreateCommitStatusConfig{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasSafeOutputsEnabled(tt.config)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
