package workflow

import "fmt"

// OIDCConfig represents OpenID Connect authentication configuration for agentic engines
type OIDCConfig struct {
	// Audience is the OIDC audience identifier (e.g., "claude-code-github-action")
	Audience string `yaml:"audience,omitempty"`

	// TokenExchangeURL is the URL to exchange OIDC token for an app token
	TokenExchangeURL string `yaml:"token_exchange_url,omitempty"`

	// TokenRevokeURL is the URL to revoke the app token (optional)
	TokenRevokeURL string `yaml:"token_revoke_url,omitempty"`

	// OauthTokenEnvVar is the environment variable name for the OAuth token obtained via OIDC
	// For Claude: CLAUDE_CODE_OAUTH_TOKEN
	OauthTokenEnvVar string `yaml:"oauth-token-env-var,omitempty"`

	// ApiTokenEnvVar is the environment variable name for the API token used as fallback
	// For Claude: ANTHROPIC_API_KEY
	ApiTokenEnvVar string `yaml:"api-token-env-var,omitempty"`
}

// ParseOIDCConfig parses OIDC configuration from engine object
func ParseOIDCConfig(engineObj map[string]any) *OIDCConfig {
	oidc, hasOIDC := engineObj["oidc"]
	if !hasOIDC {
		return nil
	}

	oidcObj, ok := oidc.(map[string]any)
	if !ok {
		return nil
	}

	oidcConfig := &OIDCConfig{}

	// Extract audience field
	if audience, hasAudience := oidcObj["audience"]; hasAudience {
		if audienceStr, ok := audience.(string); ok {
			oidcConfig.Audience = audienceStr
		}
	}

	// Extract token_exchange_url field
	if tokenExchangeURL, hasTokenExchangeURL := oidcObj["token_exchange_url"]; hasTokenExchangeURL {
		if tokenExchangeURLStr, ok := tokenExchangeURL.(string); ok {
			oidcConfig.TokenExchangeURL = tokenExchangeURLStr
		}
	}

	// Extract token_revoke_url field (optional)
	if tokenRevokeURL, hasTokenRevokeURL := oidcObj["token_revoke_url"]; hasTokenRevokeURL {
		if tokenRevokeURLStr, ok := tokenRevokeURL.(string); ok {
			oidcConfig.TokenRevokeURL = tokenRevokeURLStr
		}
	}

	// Extract oauth-token-env-var field (optional)
	if oauthTokenEnvVar, hasOauthTokenEnvVar := oidcObj["oauth-token-env-var"]; hasOauthTokenEnvVar {
		if oauthTokenEnvVarStr, ok := oauthTokenEnvVar.(string); ok {
			oidcConfig.OauthTokenEnvVar = oauthTokenEnvVarStr
		}
	}

	// Extract api-token-env-var field (optional)
	if apiTokenEnvVar, hasApiTokenEnvVar := oidcObj["api-token-env-var"]; hasApiTokenEnvVar {
		if apiTokenEnvVarStr, ok := apiTokenEnvVar.(string); ok {
			oidcConfig.ApiTokenEnvVar = apiTokenEnvVarStr
		}
	}

	return oidcConfig
}

// HasOIDCConfig checks if the engine has OIDC configuration
// OIDC is considered enabled if token_exchange_url is present
func HasOIDCConfig(config *EngineConfig) bool {
	return config != nil && config.OIDC != nil && config.OIDC.TokenExchangeURL != ""
}

// GetOIDCAudience returns the OIDC audience identifier, with a default based on engine
func (config *OIDCConfig) GetOIDCAudience(engineID string) string {
	if config.Audience != "" {
		return config.Audience
	}

	// Default audiences based on engine
	switch engineID {
	case "claude":
		return "claude-code-github-action"
	default:
		return ""
	}
}

// GetOAuthTokenEnvVar returns the OAuth token environment variable, falling back to engine default
func (config *OIDCConfig) GetOAuthTokenEnvVar(engine CodingAgentEngine) string {
	if config.OauthTokenEnvVar != "" {
		return config.OauthTokenEnvVar
	}
	return engine.GetOAuthTokenEnvVarName()
}

// GetApiTokenEnvVar returns the API token environment variable, falling back to engine default
func (config *OIDCConfig) GetApiTokenEnvVar(engine CodingAgentEngine) string {
	if config.ApiTokenEnvVar != "" {
		return config.ApiTokenEnvVar
	}
	return engine.GetTokenEnvVarName()
}

// GenerateOIDCSetupStep generates a GitHub Actions step to setup OIDC token
func GenerateOIDCSetupStep(oidcConfig *OIDCConfig, engine CodingAgentEngine) GitHubActionStep {
	var stepLines []string

	stepLines = append(stepLines, "      - name: Setup OIDC token")
	stepLines = append(stepLines, "        id: setup_oidc_token")
	// Only run if the fallback API token secret exists (check for non-empty secret)
	apiTokenEnvVar := oidcConfig.GetApiTokenEnvVar(engine)
	stepLines = append(stepLines, fmt.Sprintf("        if: secrets.%s != ''", apiTokenEnvVar))
	stepLines = append(stepLines, "        uses: actions/github-script@v8")
	stepLines = append(stepLines, "        env:")

	// Add OIDC configuration as environment variables
	audience := oidcConfig.GetOIDCAudience(engine.GetID())
	if audience != "" {
		stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_AUDIENCE: %s", audience))
	}

	if oidcConfig.TokenExchangeURL != "" {
		stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_EXCHANGE_URL: %s", oidcConfig.TokenExchangeURL))
	}

	// OAuth token environment variable (where OIDC token will be stored)
	oauthTokenEnvVar := oidcConfig.GetOAuthTokenEnvVar(engine)
	stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_OAUTH_TOKEN: %s", oauthTokenEnvVar))

	// API token environment variable (fallback) - already declared above
	stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_API_KEY: %s", apiTokenEnvVar))
	// Add the actual fallback secret if it exists
	stepLines = append(stepLines, fmt.Sprintf("          %s: ${{ secrets.%s }}", apiTokenEnvVar, apiTokenEnvVar))

	stepLines = append(stepLines, "        with:")
	stepLines = append(stepLines, "          script: |")

	// Add the JavaScript script with proper indentation
	formattedScript := FormatJavaScriptForYAML(setupOIDCTokenScript)
	stepLines = append(stepLines, formattedScript...)

	return GitHubActionStep(stepLines)
}

// GenerateOIDCRevokeStep generates a GitHub Actions step to revoke OIDC token
func GenerateOIDCRevokeStep(oidcConfig *OIDCConfig) GitHubActionStep {
	var stepLines []string

	stepLines = append(stepLines, "      - name: Revoke OIDC token")
	stepLines = append(stepLines, "        id: revoke_oidc_token")
	stepLines = append(stepLines, "        if: always() && steps.setup_oidc_token.outputs.token_source == 'oauth'")
	stepLines = append(stepLines, "        uses: actions/github-script@v8")
	stepLines = append(stepLines, "        env:")

	// Add revoke URL if configured
	if oidcConfig.TokenRevokeURL != "" {
		stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_REVOKE_URL: %s", oidcConfig.TokenRevokeURL))
	}

	// Pass token obtained status from setup step
	stepLines = append(stepLines, "          GH_AW_OIDC_TOKEN_OBTAINED: ${{ steps.setup_oidc_token.outputs.oidc_token_obtained }}")

	// Pass the token for revocation
	stepLines = append(stepLines, "          GH_AW_OIDC_TOKEN: ${{ steps.setup_oidc_token.outputs.token }}")

	stepLines = append(stepLines, "        with:")
	stepLines = append(stepLines, "          script: |")

	// Add the JavaScript script with proper indentation
	formattedScript := FormatJavaScriptForYAML(revokeOIDCTokenScript)
	stepLines = append(stepLines, formattedScript...)

	return GitHubActionStep(stepLines)
}
