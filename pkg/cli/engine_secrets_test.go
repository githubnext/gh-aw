//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequiredSecretsForEngine(t *testing.T) {
	tests := []struct {
		name                 string
		engine               string
		includeSystemSecrets bool
		includeOptional      bool
		wantSecretNames      []string
		wantMinCount         int
		wantMaxCount         int
	}{
		{
			name:                 "copilot engine without system secrets",
			engine:               string(constants.CopilotEngine),
			includeSystemSecrets: false,
			includeOptional:      false,
			wantSecretNames:      []string{"COPILOT_GITHUB_TOKEN"},
			wantMinCount:         1,
			wantMaxCount:         1,
		},
		{
			name:                 "copilot engine with system secrets",
			engine:               string(constants.CopilotEngine),
			includeSystemSecrets: true,
			includeOptional:      false,
			wantSecretNames:      []string{"COPILOT_GITHUB_TOKEN", "GH_AW_GITHUB_TOKEN"},
			wantMinCount:         2,
			wantMaxCount:         2,
		},
		{
			name:                 "copilot engine with optional secrets",
			engine:               string(constants.CopilotEngine),
			includeSystemSecrets: true,
			includeOptional:      true,
			wantSecretNames:      []string{"COPILOT_GITHUB_TOKEN", "GH_AW_GITHUB_TOKEN"},
			wantMinCount:         3, // At least 3 (required system + optional system + engine)
			wantMaxCount:         10,
		},
		{
			name:                 "claude engine",
			engine:               string(constants.ClaudeEngine),
			includeSystemSecrets: false,
			includeOptional:      false,
			wantSecretNames:      []string{"ANTHROPIC_API_KEY"},
			wantMinCount:         1,
			wantMaxCount:         1,
		},
		{
			name:                 "codex engine",
			engine:               string(constants.CodexEngine),
			includeSystemSecrets: false,
			includeOptional:      false,
			wantSecretNames:      []string{"OPENAI_API_KEY"},
			wantMinCount:         1,
			wantMaxCount:         1,
		},
		{
			name:                 "empty engine returns only system secrets when requested",
			engine:               "",
			includeSystemSecrets: true,
			includeOptional:      false,
			wantSecretNames:      []string{"GH_AW_GITHUB_TOKEN"},
			wantMinCount:         1,
			wantMaxCount:         5,
		},
		{
			name:                 "unknown engine returns no engine secrets",
			engine:               "unknown-engine",
			includeSystemSecrets: false,
			includeOptional:      false,
			wantSecretNames:      []string{},
			wantMinCount:         0,
			wantMaxCount:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requirements := GetRequiredSecretsForEngine(tt.engine, tt.includeSystemSecrets, tt.includeOptional)

			assert.GreaterOrEqual(t, len(requirements), tt.wantMinCount,
				"Should have at least %d requirements", tt.wantMinCount)
			assert.LessOrEqual(t, len(requirements), tt.wantMaxCount,
				"Should have at most %d requirements", tt.wantMaxCount)

			// Check that expected secrets are present
			secretNames := make(map[string]bool)
			for _, req := range requirements {
				secretNames[req.Name] = true
			}
			for _, wantName := range tt.wantSecretNames {
				assert.True(t, secretNames[wantName],
					"Should include secret %s", wantName)
			}
		})
	}
}

func TestGetRequiredSecretsForEngineAttributes(t *testing.T) {
	t.Run("copilot secret has correct attributes", func(t *testing.T) {
		requirements := GetRequiredSecretsForEngine(string(constants.CopilotEngine), false, false)
		require.Len(t, requirements, 1, "Should have exactly one requirement")

		req := requirements[0]
		assert.Equal(t, "COPILOT_GITHUB_TOKEN", req.Name, "Secret name should match")
		assert.True(t, req.IsEngineSecret, "Should be marked as engine secret")
		assert.Equal(t, string(constants.CopilotEngine), req.EngineName, "Engine name should match")
		assert.False(t, req.Optional, "Copilot token should be required")
		assert.NotEmpty(t, req.KeyURL, "Should have a key URL")
		assert.NotEmpty(t, req.Description, "Should have a description")
	})

	t.Run("claude secret has alternative env vars", func(t *testing.T) {
		requirements := GetRequiredSecretsForEngine(string(constants.ClaudeEngine), false, false)
		require.Len(t, requirements, 1, "Should have exactly one requirement")

		req := requirements[0]
		assert.Equal(t, "ANTHROPIC_API_KEY", req.Name, "Secret name should match")
		assert.Contains(t, req.AlternativeEnvVars, "CLAUDE_CODE_OAUTH_TOKEN",
			"Should include alternative env var")
	})

	t.Run("system secrets are not engine secrets", func(t *testing.T) {
		requirements := GetRequiredSecretsForEngine("", true, true)

		for _, req := range requirements {
			if req.Name == "GH_AW_GITHUB_TOKEN" {
				assert.False(t, req.IsEngineSecret, "System secret should not be marked as engine secret")
				assert.Empty(t, req.EngineName, "System secret should have empty engine name")
			}
		}
	})
}

func TestStringContainsSecretName(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		secretName string
		want       bool
	}{
		{
			name:       "exact match single line",
			output:     "GH_AW_GITHUB_TOKEN",
			secretName: "GH_AW_GITHUB_TOKEN",
			want:       true,
		},
		{
			name:       "match with tab separator",
			output:     "GH_AW_GITHUB_TOKEN\t2024-01-01",
			secretName: "GH_AW_GITHUB_TOKEN",
			want:       true,
		},
		{
			name:       "match with space separator",
			output:     "GH_AW_GITHUB_TOKEN 2024-01-01",
			secretName: "GH_AW_GITHUB_TOKEN",
			want:       true,
		},
		{
			name:       "match in multiline output",
			output:     "SOME_SECRET\t2024-01-01\nGH_AW_GITHUB_TOKEN\t2024-02-01\nOTHER_SECRET\t2024-03-01",
			secretName: "GH_AW_GITHUB_TOKEN",
			want:       true,
		},
		{
			name:       "no match - different secret",
			output:     "SOME_OTHER_TOKEN\t2024-01-01",
			secretName: "GH_AW_GITHUB_TOKEN",
			want:       false,
		},
		{
			name:       "no match - prefix only",
			output:     "GH_AW_GITHUB_TOKEN_EXTENDED\t2024-01-01",
			secretName: "GH_AW_GITHUB_TOKEN",
			want:       false,
		},
		{
			name:       "no match - empty output",
			output:     "",
			secretName: "GH_AW_GITHUB_TOKEN",
			want:       false,
		},
		{
			name:       "no match - secret longer than line",
			output:     "SHORT",
			secretName: "GH_AW_GITHUB_TOKEN",
			want:       false,
		},
		{
			name:       "match copilot token",
			output:     "COPILOT_GITHUB_TOKEN\t2024-01-15\nANTHROPIC_API_KEY\t2024-01-20",
			secretName: "COPILOT_GITHUB_TOKEN",
			want:       true,
		},
		{
			name:       "match anthropic key in mixed output",
			output:     "COPILOT_GITHUB_TOKEN\t2024-01-15\nANTHROPIC_API_KEY\t2024-01-20",
			secretName: "ANTHROPIC_API_KEY",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringContainsSecretName(tt.output, tt.secretName)
			assert.Equal(t, tt.want, got,
				"stringContainsSecretName(%q, %q) = %v, want %v",
				tt.output, tt.secretName, got, tt.want)
		})
	}
}

func TestGetEngineSecretDescription(t *testing.T) {
	tests := []struct {
		name         string
		engineValue  string
		wantContains string
	}{
		{
			name:         "copilot engine description",
			engineValue:  string(constants.CopilotEngine),
			wantContains: "Fine-grained PAT",
		},
		{
			name:         "copilot-sdk engine description",
			engineValue:  string(constants.CopilotSDKEngine),
			wantContains: "Fine-grained PAT",
		},
		{
			name:         "claude engine description",
			engineValue:  string(constants.ClaudeEngine),
			wantContains: "Anthropic Console",
		},
		{
			name:         "codex engine description",
			engineValue:  string(constants.CodexEngine),
			wantContains: "OpenAI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := constants.GetEngineOption(tt.engineValue)
			require.NotNil(t, opt, "Engine option should exist for %s", tt.engineValue)

			desc := getEngineSecretDescription(opt)
			assert.Contains(t, desc, tt.wantContains,
				"Description should contain %q", tt.wantContains)
		})
	}
}

func TestSecretRequirementStructure(t *testing.T) {
	t.Run("SecretRequirement has all required fields", func(t *testing.T) {
		req := SecretRequirement{
			Name:               "TEST_SECRET",
			WhenNeeded:         "When testing",
			Description:        "Test description",
			Optional:           false,
			AlternativeEnvVars: []string{"ALT_SECRET"},
			KeyURL:             "https://example.com/keys",
			IsEngineSecret:     true,
			EngineName:         "test-engine",
		}

		assert.Equal(t, "TEST_SECRET", req.Name)
		assert.Equal(t, "When testing", req.WhenNeeded)
		assert.Equal(t, "Test description", req.Description)
		assert.False(t, req.Optional)
		assert.Contains(t, req.AlternativeEnvVars, "ALT_SECRET")
		assert.Equal(t, "https://example.com/keys", req.KeyURL)
		assert.True(t, req.IsEngineSecret)
		assert.Equal(t, "test-engine", req.EngineName)
	})
}

func TestEngineSecretConfigStructure(t *testing.T) {
	t.Run("EngineSecretConfig has all required fields", func(t *testing.T) {
		config := EngineSecretConfig{
			RepoSlug:             "owner/repo",
			Engine:               "copilot",
			Verbose:              true,
			ExistingSecrets:      map[string]bool{"SECRET1": true},
			IncludeSystemSecrets: true,
			IncludeOptional:      false,
		}

		assert.Equal(t, "owner/repo", config.RepoSlug)
		assert.Equal(t, "copilot", config.Engine)
		assert.True(t, config.Verbose)
		assert.True(t, config.ExistingSecrets["SECRET1"])
		assert.True(t, config.IncludeSystemSecrets)
		assert.False(t, config.IncludeOptional)
	})
}
