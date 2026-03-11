//go:build !integration

// This file contains tests for AuthDefinition and RequestShape validation.
// It covers:
//   - OAuth client-credentials definition validates correctly
//   - Missing tokenUrl/clientId/clientSecret produce helpful errors
//   - Unknown auth strategy produces a clear error
//   - Strict mode includes auth-binding secrets in required secret list
//   - Existing built-in auth flows are unchanged (regression)

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthDefinition_RequiredSecretNames verifies that RequiredSecretNames returns the
// correct secret names depending on the auth strategy.
func TestAuthDefinition_RequiredSecretNames(t *testing.T) {
	tests := []struct {
		name     string
		auth     *AuthDefinition
		expected []string
	}{
		{
			name:     "nil auth returns empty",
			auth:     nil,
			expected: []string{},
		},
		{
			name: "api-key strategy returns Secret",
			auth: &AuthDefinition{
				Strategy: AuthStrategyAPIKey,
				Secret:   "MY_API_KEY",
			},
			expected: []string{"MY_API_KEY"},
		},
		{
			name: "bearer strategy returns Secret",
			auth: &AuthDefinition{
				Strategy: AuthStrategyBearer,
				Secret:   "MY_BEARER_TOKEN",
			},
			expected: []string{"MY_BEARER_TOKEN"},
		},
		{
			name: "oauth-client-credentials returns ClientIDRef and ClientSecretRef",
			auth: &AuthDefinition{
				Strategy:        AuthStrategyOAuthClientCreds,
				TokenURL:        "https://auth.example.com/oauth/token",
				ClientIDRef:     "MY_CLIENT_ID",
				ClientSecretRef: "MY_CLIENT_SECRET",
				HeaderName:      "api-key",
			},
			expected: []string{"MY_CLIENT_ID", "MY_CLIENT_SECRET"},
		},
		{
			name: "oauth with only ClientIDRef returns just that",
			auth: &AuthDefinition{
				Strategy:    AuthStrategyOAuthClientCreds,
				ClientIDRef: "MY_CLIENT_ID",
			},
			expected: []string{"MY_CLIENT_ID"},
		},
		{
			name: "empty strategy with Secret treated like api-key/bearer",
			auth: &AuthDefinition{
				Secret: "SOME_SECRET",
			},
			expected: []string{"SOME_SECRET"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.auth.RequiredSecretNames()
			assert.Equal(t, tt.expected, result, "RequiredSecretNames() should return expected secrets")
		})
	}
}

// TestValidateEngineAuthDefinition_OAuthClientCredentials checks that a valid
// oauth-client-credentials definition passes validation.
func TestValidateEngineAuthDefinition_OAuthClientCredentials(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy:        AuthStrategyOAuthClientCreds,
			TokenURL:        "https://auth.example.com/oauth/token",
			ClientIDRef:     "MY_CLIENT_ID",
			ClientSecretRef: "MY_CLIENT_SECRET",
			HeaderName:      "api-key",
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	require.NoError(t, err, "valid oauth-client-credentials definition should pass validation")
}

// TestValidateEngineAuthDefinition_MissingTokenURL verifies that omitting token-url
// in an oauth-client-credentials definition produces a clear error.
func TestValidateEngineAuthDefinition_MissingTokenURL(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy:        AuthStrategyOAuthClientCreds,
			ClientIDRef:     "MY_CLIENT_ID",
			ClientSecretRef: "MY_CLIENT_SECRET",
			HeaderName:      "api-key",
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	require.Error(t, err, "missing token-url should produce an error")
	assert.Contains(t, err.Error(), "token-url", "error should mention missing token-url field")
	assert.Contains(t, err.Error(), "oauth-client-credentials", "error should mention the strategy")
}

// TestValidateEngineAuthDefinition_MissingClientID verifies that omitting client-id
// produces a clear error.
func TestValidateEngineAuthDefinition_MissingClientID(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy:        AuthStrategyOAuthClientCreds,
			TokenURL:        "https://auth.example.com/oauth/token",
			ClientSecretRef: "MY_CLIENT_SECRET",
			HeaderName:      "api-key",
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	require.Error(t, err, "missing client-id should produce an error")
	assert.Contains(t, err.Error(), "client-id", "error should mention missing client-id field")
}

// TestValidateEngineAuthDefinition_MissingClientSecret verifies that omitting client-secret
// produces a clear error.
func TestValidateEngineAuthDefinition_MissingClientSecret(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy:    AuthStrategyOAuthClientCreds,
			TokenURL:    "https://auth.example.com/oauth/token",
			ClientIDRef: "MY_CLIENT_ID",
			HeaderName:  "api-key",
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	require.Error(t, err, "missing client-secret should produce an error")
	assert.Contains(t, err.Error(), "client-secret", "error should mention missing client-secret field")
}

// TestValidateEngineAuthDefinition_MissingHeaderNameForOAuth verifies that omitting
// header-name for oauth-client-credentials produces a clear error.
func TestValidateEngineAuthDefinition_MissingHeaderNameForOAuth(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy:        AuthStrategyOAuthClientCreds,
			TokenURL:        "https://auth.example.com/oauth/token",
			ClientIDRef:     "MY_CLIENT_ID",
			ClientSecretRef: "MY_CLIENT_SECRET",
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	require.Error(t, err, "missing header-name for oauth should produce an error")
	assert.Contains(t, err.Error(), "header-name", "error should mention missing header-name field")
}

// TestValidateEngineAuthDefinition_MissingHeaderNameForAPIKey verifies that api-key
// strategy without header-name produces a clear error.
func TestValidateEngineAuthDefinition_MissingHeaderNameForAPIKey(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy: AuthStrategyAPIKey,
			Secret:   "MY_API_KEY",
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	require.Error(t, err, "api-key without header-name should produce an error")
	assert.Contains(t, err.Error(), "header-name", "error should mention missing header-name field")
}

// TestValidateEngineAuthDefinition_APIKeyRequiresSecret verifies that api-key
// strategy without a secret produces a clear error.
func TestValidateEngineAuthDefinition_APIKeyRequiresSecret(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy:   AuthStrategyAPIKey,
			HeaderName: "x-api-key",
			// Secret intentionally omitted
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	require.Error(t, err, "api-key without secret should produce an error")
	assert.Contains(t, err.Error(), "auth.secret", "error should mention missing secret field")
}

// TestValidateEngineAuthDefinition_BearerRequiresSecret verifies that bearer
// strategy without a secret produces a clear error.
func TestValidateEngineAuthDefinition_BearerRequiresSecret(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy: AuthStrategyBearer,
			// Secret intentionally omitted
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	require.Error(t, err, "bearer without secret should produce an error")
	assert.Contains(t, err.Error(), "auth.secret", "error should mention missing secret field")
}

// TestValidateEngineAuthDefinition_APIKeyValid verifies that a complete api-key definition
// (secret + header-name) passes validation.
func TestValidateEngineAuthDefinition_APIKeyValid(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy:   AuthStrategyAPIKey,
			Secret:     "MY_API_KEY",
			HeaderName: "x-api-key",
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	assert.NoError(t, err, "complete api-key definition should pass validation")
}

// does not require header-name (it uses the standard Authorization header by convention).
func TestValidateEngineAuthDefinition_BearerNoHeaderRequired(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy: AuthStrategyBearer,
			Secret:   "MY_BEARER_TOKEN",
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	assert.NoError(t, err, "bearer strategy without header-name should be valid")
}

// TestValidateEngineAuthDefinition_UnknownStrategy verifies that an unknown strategy
// produces a clear error listing the valid strategies.
func TestValidateEngineAuthDefinition_UnknownStrategy(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy: "invalid-strategy",
			Secret:   "MY_SECRET",
		},
	}

	err := compiler.validateEngineAuthDefinition(config)
	require.Error(t, err, "unknown strategy should produce an error")
	assert.Contains(t, err.Error(), "invalid-strategy", "error should mention the unknown strategy")
	assert.Contains(t, err.Error(), "api-key", "error should list valid strategies")
	assert.Contains(t, err.Error(), "oauth-client-credentials", "error should list valid strategies")
	assert.Contains(t, err.Error(), "bearer", "error should list valid strategies")
}

// TestValidateEngineAuthDefinition_NilAuth verifies that a nil InlineProviderAuth is a no-op.
func TestValidateEngineAuthDefinition_NilAuth(t *testing.T) {
	compiler := newTestCompiler(t)
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: nil,
	}

	err := compiler.validateEngineAuthDefinition(config)
	assert.NoError(t, err, "nil auth definition should pass validation")
}

// TestRegisterInlineEngineDefinition_WithAuthDefinition checks that an inline engine
// definition with an AuthDefinition is correctly stored in the catalog.
func TestRegisterInlineEngineDefinition_WithAuthDefinition(t *testing.T) {
	compiler := newTestCompiler(t)

	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderID:   "azure-openai",
		InlineProviderAuth: &AuthDefinition{
			Strategy:        AuthStrategyOAuthClientCreds,
			TokenURL:        "https://auth.example.com/oauth/token",
			ClientIDRef:     "MY_CLIENT_ID",
			ClientSecretRef: "MY_CLIENT_SECRET",
			HeaderName:      "api-key",
		},
		InlineProviderRequest: &RequestShape{
			PathTemplate: "/openai/deployments/{model}/chat/completions",
			Query:        map[string]string{"api-version": "2024-10-01-preview"},
		},
	}

	compiler.registerInlineEngineDefinition(config)

	def := compiler.engineCatalog.Get("codex")
	require.NotNil(t, def, "engine definition should be in catalog after registration")
	assert.Equal(t, "azure-openai", def.Provider.Name, "provider name should be set")
	require.NotNil(t, def.Provider.Auth, "AuthDefinition should be populated")
	assert.Equal(t, AuthStrategyOAuthClientCreds, def.Provider.Auth.Strategy, "strategy should be oauth-client-credentials")
	assert.Equal(t, "MY_CLIENT_ID", def.Provider.Auth.ClientIDRef, "ClientIDRef should be set")
	assert.Equal(t, "MY_CLIENT_SECRET", def.Provider.Auth.ClientSecretRef, "ClientSecretRef should be set")
	require.NotNil(t, def.Provider.Request, "RequestShape should be populated")
	assert.Equal(t, "/openai/deployments/{model}/chat/completions", def.Provider.Request.PathTemplate, "PathTemplate should match")
}

// TestRegisterInlineEngineDefinition_BackwardsCompatSimpleSecret checks that the legacy
// auth.secret path (no strategy) still works with backwards-compat normalization.
func TestRegisterInlineEngineDefinition_BackwardsCompatSimpleSecret(t *testing.T) {
	compiler := newTestCompiler(t)

	config := &EngineConfig{
		ID:                   "codex",
		IsInlineDefinition:   true,
		InlineProviderSecret: "MY_API_KEY",
		InlineProviderAuth: &AuthDefinition{
			Secret: "MY_API_KEY",
			// Strategy intentionally empty (backwards compat)
		},
	}

	compiler.registerInlineEngineDefinition(config)

	def := compiler.engineCatalog.Get("codex")
	require.NotNil(t, def, "engine definition should be in catalog")
	require.NotNil(t, def.Provider.Auth, "AuthDefinition should be populated")
	assert.Equal(t, AuthStrategyAPIKey, def.Provider.Auth.Strategy,
		"empty strategy with Secret should be normalised to api-key")
	assert.Equal(t, "MY_API_KEY", def.Provider.Auth.Secret, "Secret should be set")
}

// TestStrictModeGetEngineBaseEnvVarKeys_IncludesAuthSecrets verifies that
// getEngineBaseEnvVarKeys includes secrets from an inline engine's AuthDefinition.
func TestStrictModeGetEngineBaseEnvVarKeys_IncludesAuthSecrets(t *testing.T) {
	compiler := newTestCompiler(t)

	// Register an inline definition with an AuthDefinition so the catalog carries it.
	config := &EngineConfig{
		ID:                 "codex",
		IsInlineDefinition: true,
		InlineProviderAuth: &AuthDefinition{
			Strategy:        AuthStrategyOAuthClientCreds,
			TokenURL:        "https://auth.example.com/oauth/token",
			ClientIDRef:     "MY_CLIENT_ID",
			ClientSecretRef: "MY_CLIENT_SECRET",
			HeaderName:      "api-key",
		},
	}
	compiler.registerInlineEngineDefinition(config)

	keys := compiler.getEngineBaseEnvVarKeys("codex")
	assert.True(t, keys["MY_CLIENT_ID"], "client ID secret should be in allowed env-var keys")
	assert.True(t, keys["MY_CLIENT_SECRET"], "client secret should be in allowed env-var keys")
}

// TestBuiltInEngineAuthUnchanged is a regression test verifying that the built-in engines
// (claude, codex, copilot, gemini) retain their original auth configuration after the
// AuthDefinition changes.
func TestBuiltInEngineAuthUnchanged(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	tests := []struct {
		engineID       string
		wantAuthSecret string // expected legacy AuthBinding secret (empty = no binding)
	}{
		{"claude", "ANTHROPIC_API_KEY"},
		{"codex", "CODEX_API_KEY"},
		{"copilot", ""}, // copilot has no API-key binding
		{"gemini", ""},  // gemini has no API-key binding
	}

	for _, tt := range tests {
		t.Run(tt.engineID, func(t *testing.T) {
			def := catalog.Get(tt.engineID)
			require.NotNilf(t, def, "built-in engine %s should be in catalog", tt.engineID)

			// Provider.Auth should be nil for built-in engines (they use AuthBinding only).
			assert.Nil(t, def.Provider.Auth,
				"built-in engine %s should have no Provider.Auth (uses legacy AuthBinding)", tt.engineID)

			if tt.wantAuthSecret != "" {
				require.Lenf(t, def.Auth, 1, "engine %s should have exactly one AuthBinding", tt.engineID)
				assert.Equal(t, tt.wantAuthSecret, def.Auth[0].Secret,
					"engine %s AuthBinding.Secret should be unchanged", tt.engineID)
			} else {
				assert.Empty(t, def.Auth, "engine %s should have no AuthBinding", tt.engineID)
			}
		})
	}
}

// TestParseAuthDefinition verifies that parseAuthDefinition correctly maps YAML map keys
// to AuthDefinition fields.
func TestParseAuthDefinition(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected AuthDefinition
	}{
		{
			name:     "empty map produces zero-value AuthDefinition",
			input:    map[string]any{},
			expected: AuthDefinition{},
		},
		{
			name: "simple secret only (backwards compat)",
			input: map[string]any{
				"secret": "MY_API_KEY",
			},
			expected: AuthDefinition{Secret: "MY_API_KEY"},
		},
		{
			name: "full oauth-client-credentials config",
			input: map[string]any{
				"strategy":      "oauth-client-credentials",
				"token-url":     "https://auth.example.com/oauth/token",
				"client-id":     "MY_CLIENT_ID",
				"client-secret": "MY_CLIENT_SECRET",
				"token-field":   "access_token",
				"header-name":   "api-key",
			},
			expected: AuthDefinition{
				Strategy:        AuthStrategyOAuthClientCreds,
				TokenURL:        "https://auth.example.com/oauth/token",
				ClientIDRef:     "MY_CLIENT_ID",
				ClientSecretRef: "MY_CLIENT_SECRET",
				TokenField:      "access_token",
				HeaderName:      "api-key",
			},
		},
		{
			name: "api-key strategy with header-name",
			input: map[string]any{
				"strategy":    "api-key",
				"secret":      "CUSTOM_API_KEY",
				"header-name": "x-api-key",
			},
			expected: AuthDefinition{
				Strategy:   AuthStrategyAPIKey,
				Secret:     "CUSTOM_API_KEY",
				HeaderName: "x-api-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAuthDefinition(tt.input)
			require.NotNil(t, result, "parseAuthDefinition should never return nil")
			assert.Equal(t, tt.expected, *result, "parsed AuthDefinition should match expected")
		})
	}
}

// TestParseRequestShape verifies that parseRequestShape correctly maps YAML map keys
// to RequestShape fields.
func TestParseRequestShape(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected RequestShape
	}{
		{
			name:     "empty map produces zero-value RequestShape",
			input:    map[string]any{},
			expected: RequestShape{},
		},
		{
			name: "path-template only",
			input: map[string]any{
				"path-template": "/openai/deployments/{model}/chat/completions",
			},
			expected: RequestShape{
				PathTemplate: "/openai/deployments/{model}/chat/completions",
			},
		},
		{
			name: "full request shape",
			input: map[string]any{
				"path-template": "/openai/deployments/{model}/chat/completions",
				"query": map[string]any{
					"api-version": "2024-10-01-preview",
				},
				"body-inject": map[string]any{
					"appKey": "{APP_KEY_SECRET}",
				},
			},
			expected: RequestShape{
				PathTemplate: "/openai/deployments/{model}/chat/completions",
				Query:        map[string]string{"api-version": "2024-10-01-preview"},
				BodyInject:   map[string]string{"appKey": "{APP_KEY_SECRET}"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRequestShape(tt.input)
			require.NotNil(t, result, "parseRequestShape should never return nil")
			assert.Equal(t, tt.expected, *result, "parsed RequestShape should match expected")
		})
	}
}

// newTestCompiler creates a minimal Compiler for unit testing engine-definition logic.
func newTestCompiler(t *testing.T) *Compiler {
	t.Helper()
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)
	return &Compiler{
		engineRegistry: registry,
		engineCatalog:  catalog,
	}
}
