//go:build !integration

package cli

import (
	"os"
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
			wantSecretNames:      []string{"COPILOT_GITHUB_TOKEN"}, // No system secrets since all are optional
			wantMinCount:         1,
			wantMaxCount:         1,
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
			includeOptional:      true, // Changed to true to include optional system secrets
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
			requirements := getSecretRequirementsForEngine(tt.engine, tt.includeSystemSecrets, tt.includeOptional)

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
		requirements := getSecretRequirementsForEngine(string(constants.CopilotEngine), false, false)
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
		requirements := getSecretRequirementsForEngine(string(constants.ClaudeEngine), false, false)
		require.Len(t, requirements, 1, "Should have exactly one requirement")

		req := requirements[0]
		assert.Equal(t, "ANTHROPIC_API_KEY", req.Name, "Secret name should match")
		assert.Contains(t, req.AlternativeEnvVars, "CLAUDE_CODE_OAUTH_TOKEN",
			"Should include alternative env var")
	})

	t.Run("system secrets are not engine secrets", func(t *testing.T) {
		requirements := getSecretRequirementsForEngine("", true, true)

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

func TestGetEngineSecretNameAndValue(t *testing.T) {
	// Save current env and restore after test
	oldCopilotToken := os.Getenv("COPILOT_GITHUB_TOKEN")
	oldAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	oldOpenAIKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if oldCopilotToken != "" {
			os.Setenv("COPILOT_GITHUB_TOKEN", oldCopilotToken)
		} else {
			os.Unsetenv("COPILOT_GITHUB_TOKEN")
		}
		if oldAnthropicKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", oldAnthropicKey)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
		if oldOpenAIKey != "" {
			os.Setenv("OPENAI_API_KEY", oldOpenAIKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	t.Run("secret exists in repository", func(t *testing.T) {
		existingSecrets := map[string]bool{
			"COPILOT_GITHUB_TOKEN": true,
		}

		name, value, existsInRepo, err := GetEngineSecretNameAndValue("copilot", existingSecrets)

		require.NoError(t, err, "Should not error when secret exists in repo")
		assert.Equal(t, "COPILOT_GITHUB_TOKEN", name)
		assert.Empty(t, value, "Value should be empty when secret exists in repo")
		assert.True(t, existsInRepo, "Should indicate secret exists in repo")
	})

	t.Run("secret found in environment", func(t *testing.T) {
		os.Setenv("ANTHROPIC_API_KEY", "test-api-key-12345")
		defer os.Unsetenv("ANTHROPIC_API_KEY")

		existingSecrets := map[string]bool{}

		name, value, existsInRepo, err := GetEngineSecretNameAndValue("claude", existingSecrets)

		require.NoError(t, err, "Should not error when secret in environment")
		assert.Equal(t, "ANTHROPIC_API_KEY", name)
		assert.Equal(t, "test-api-key-12345", value)
		assert.False(t, existsInRepo, "Should indicate secret does not exist in repo")
	})

	t.Run("secret not in repo or environment", func(t *testing.T) {
		os.Unsetenv("OPENAI_API_KEY")

		existingSecrets := map[string]bool{}

		name, value, existsInRepo, err := GetEngineSecretNameAndValue("codex", existingSecrets)

		require.NoError(t, err, "Should not error even when secret not found")
		assert.Equal(t, "OPENAI_API_KEY", name)
		assert.Empty(t, value, "Value should be empty when not found in env")
		assert.False(t, existsInRepo, "Should indicate secret does not exist in repo")
	})

	t.Run("unknown engine returns error", func(t *testing.T) {
		existingSecrets := map[string]bool{}

		_, _, _, err := GetEngineSecretNameAndValue("unknown-engine", existingSecrets)

		require.Error(t, err, "Should error for unknown engine")
		assert.Contains(t, err.Error(), "unknown engine", "Error should mention unknown engine")
	})

	t.Run("alternative secret exists in repo", func(t *testing.T) {
		// Claude has CLAUDE_CODE_OAUTH_TOKEN as alternative
		existingSecrets := map[string]bool{
			"CLAUDE_CODE_OAUTH_TOKEN": true, // Alternative for ANTHROPIC_API_KEY
		}

		name, value, existsInRepo, err := GetEngineSecretNameAndValue("claude", existingSecrets)

		require.NoError(t, err, "Should not error when alternative exists")
		assert.Equal(t, "ANTHROPIC_API_KEY", name)
		assert.Empty(t, value, "Value should be empty when alternative exists in repo")
		assert.True(t, existsInRepo, "Should indicate secret exists via alternative")
	})

	t.Run("prefers primary secret over environment", func(t *testing.T) {
		os.Setenv("COPILOT_GITHUB_TOKEN", "test-token-from-env")
		defer os.Unsetenv("COPILOT_GITHUB_TOKEN")

		existingSecrets := map[string]bool{
			"COPILOT_GITHUB_TOKEN": true,
		}

		name, value, existsInRepo, err := GetEngineSecretNameAndValue("copilot", existingSecrets)

		require.NoError(t, err, "Should not error")
		assert.Equal(t, "COPILOT_GITHUB_TOKEN", name)
		assert.Empty(t, value, "Should prefer existing repo secret over environment")
		assert.True(t, existsInRepo, "Should indicate secret exists in repo")
	})
}

func TestGetMissingRequiredSecrets(t *testing.T) {
	t.Run("all secrets missing", func(t *testing.T) {
		requirements := []SecretRequirement{
			{Name: "SECRET1", Optional: false},
			{Name: "SECRET2", Optional: false},
			{Name: "SECRET3", Optional: false},
		}
		existingSecrets := map[string]bool{}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Len(t, missing, 3, "Should have 3 missing secrets")
		assert.Equal(t, "SECRET1", missing[0].Name)
		assert.Equal(t, "SECRET2", missing[1].Name)
		assert.Equal(t, "SECRET3", missing[2].Name)
	})

	t.Run("all secrets exist", func(t *testing.T) {
		requirements := []SecretRequirement{
			{Name: "SECRET1", Optional: false},
			{Name: "SECRET2", Optional: false},
		}
		existingSecrets := map[string]bool{
			"SECRET1": true,
			"SECRET2": true,
		}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Empty(t, missing, "Should have no missing secrets")
	})

	t.Run("some secrets missing", func(t *testing.T) {
		requirements := []SecretRequirement{
			{Name: "SECRET1", Optional: false},
			{Name: "SECRET2", Optional: false},
			{Name: "SECRET3", Optional: false},
		}
		existingSecrets := map[string]bool{
			"SECRET1": true,
		}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Len(t, missing, 2, "Should have 2 missing secrets")
		assert.Equal(t, "SECRET2", missing[0].Name)
		assert.Equal(t, "SECRET3", missing[1].Name)
	})

	t.Run("optional secrets are skipped", func(t *testing.T) {
		requirements := []SecretRequirement{
			{Name: "REQUIRED1", Optional: false},
			{Name: "OPTIONAL1", Optional: true},
			{Name: "REQUIRED2", Optional: false},
			{Name: "OPTIONAL2", Optional: true},
		}
		existingSecrets := map[string]bool{}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Len(t, missing, 2, "Should only include required secrets")
		assert.Equal(t, "REQUIRED1", missing[0].Name)
		assert.Equal(t, "REQUIRED2", missing[1].Name)
	})

	t.Run("alternative secret names work", func(t *testing.T) {
		requirements := []SecretRequirement{
			{
				Name:               "PRIMARY_SECRET",
				Optional:           false,
				AlternativeEnvVars: []string{"ALT_SECRET1", "ALT_SECRET2"},
			},
			{Name: "OTHER_SECRET", Optional: false},
		}
		existingSecrets := map[string]bool{
			"ALT_SECRET1": true, // Alternative exists
		}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Len(t, missing, 1, "Should have 1 missing secret")
		assert.Equal(t, "OTHER_SECRET", missing[0].Name, "Should not include PRIMARY_SECRET since alternative exists")
	})

	t.Run("alternative secret names - second alternative", func(t *testing.T) {
		requirements := []SecretRequirement{
			{
				Name:               "PRIMARY_SECRET",
				Optional:           false,
				AlternativeEnvVars: []string{"ALT_SECRET1", "ALT_SECRET2"},
			},
		}
		existingSecrets := map[string]bool{
			"ALT_SECRET2": true, // Second alternative exists
		}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Empty(t, missing, "Should find second alternative")
	})

	t.Run("primary secret takes precedence over alternatives", func(t *testing.T) {
		requirements := []SecretRequirement{
			{
				Name:               "PRIMARY_SECRET",
				Optional:           false,
				AlternativeEnvVars: []string{"ALT_SECRET"},
			},
		}
		existingSecrets := map[string]bool{
			"PRIMARY_SECRET": true,
			"ALT_SECRET":     true,
		}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Empty(t, missing, "Should not include secret when primary exists")
	})

	t.Run("empty requirements list", func(t *testing.T) {
		requirements := []SecretRequirement{}
		existingSecrets := map[string]bool{
			"SECRET1": true,
		}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Empty(t, missing, "Should return empty list for empty requirements")
	})

	t.Run("empty existing secrets map", func(t *testing.T) {
		requirements := []SecretRequirement{
			{Name: "SECRET1", Optional: false},
			{Name: "SECRET2", Optional: false},
		}
		existingSecrets := map[string]bool{}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Len(t, missing, 2, "Should return all required secrets as missing")
	})

	t.Run("nil existing secrets map", func(t *testing.T) {
		requirements := []SecretRequirement{
			{Name: "SECRET1", Optional: false},
		}
		var existingSecrets map[string]bool // nil map

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Len(t, missing, 1, "Should handle nil map and return all as missing")
	})

	t.Run("mixed required and optional with alternatives", func(t *testing.T) {
		requirements := []SecretRequirement{
			{
				Name:               "COPILOT_GITHUB_TOKEN",
				Optional:           false,
				IsEngineSecret:     true,
				AlternativeEnvVars: []string{"GITHUB_TOKEN"},
			},
			{
				Name:     "GH_AW_GITHUB_TOKEN",
				Optional: true,
			},
			{
				Name:               "ANTHROPIC_API_KEY",
				Optional:           false,
				IsEngineSecret:     true,
				AlternativeEnvVars: []string{"CLAUDE_API_KEY"},
			},
		}
		existingSecrets := map[string]bool{
			"GITHUB_TOKEN": true, // Alternative for COPILOT_GITHUB_TOKEN
		}

		missing := getMissingRequiredSecrets(requirements, existingSecrets)

		assert.Len(t, missing, 1, "Should have 1 missing required secret")
		assert.Equal(t, "ANTHROPIC_API_KEY", missing[0].Name, "Should only include ANTHROPIC_API_KEY")
	})
}
