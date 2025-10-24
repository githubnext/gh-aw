package workflow

import "fmt"

// OIDCConfig represents OpenID Connect authentication configuration for agentic engines
type OIDCConfig struct {
	// Enabled indicates whether OIDC authentication is enabled
	Enabled bool `yaml:"enabled,omitempty"`

	// Audience is the OIDC audience identifier (e.g., "claude-code-github-action")
	Audience string `yaml:"audience,omitempty"`

	// TokenExchangeURL is the URL to exchange OIDC token for an app token
	TokenExchangeURL string `yaml:"token_exchange_url,omitempty"`

	// TokenRevokeURL is the URL to revoke the app token (optional)
	TokenRevokeURL string `yaml:"token_revoke_url,omitempty"`

	// EnvVarName is the environment variable name to store the token (defaults to token-specific env var)
	EnvVarName string `yaml:"env_var_name,omitempty"`

	// FallbackEnvVar is the fallback environment variable to use if OIDC fails
	FallbackEnvVar string `yaml:"fallback_env_var,omitempty"`
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

	// Extract enabled field (defaults to false)
	if enabled, hasEnabled := oidcObj["enabled"]; hasEnabled {
		if enabledBool, ok := enabled.(bool); ok {
			oidcConfig.Enabled = enabledBool
		}
	}

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

	// Extract env_var_name field (optional)
	if envVarName, hasEnvVarName := oidcObj["env_var_name"]; hasEnvVarName {
		if envVarNameStr, ok := envVarName.(string); ok {
			oidcConfig.EnvVarName = envVarNameStr
		}
	}

	// Extract fallback_env_var field (optional)
	if fallbackEnvVar, hasFallbackEnvVar := oidcObj["fallback_env_var"]; hasFallbackEnvVar {
		if fallbackEnvVarStr, ok := fallbackEnvVar.(string); ok {
			oidcConfig.FallbackEnvVar = fallbackEnvVarStr
		}
	}

	return oidcConfig
}

// HasOIDCConfig checks if the engine has OIDC configuration
func HasOIDCConfig(config *EngineConfig) bool {
	return config != nil && config.OIDC != nil && config.OIDC.Enabled
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

// GenerateOIDCSetupStep generates a GitHub Actions step to setup OIDC token
func GenerateOIDCSetupStep(oidcConfig *OIDCConfig, engine CodingAgentEngine) GitHubActionStep {
	var stepLines []string

	stepLines = append(stepLines, "      - name: Setup OIDC token")
	stepLines = append(stepLines, "        id: setup_oidc_token")
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

	// Use engine's token environment variable name
	envVarName := engine.GetTokenEnvVarName()
	stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_ENV_VAR_NAME: %s", envVarName))

	// Use the same env var as fallback
	stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_FALLBACK_ENV_VAR: %s", envVarName))
	// Add the actual fallback secret if it exists
	stepLines = append(stepLines, fmt.Sprintf("          %s: ${{ secrets.%s }}", envVarName, envVarName))

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
	stepLines = append(stepLines, "        if: always() && steps.setup_oidc_token.outputs.token != ''")
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
