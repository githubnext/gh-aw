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

// GetTokenEnvVarName returns the environment variable name for the token
func (config *OIDCConfig) GetTokenEnvVarName(engineID string) string {
	if config.EnvVarName != "" {
		return config.EnvVarName
	}

	// Default env var names based on engine
	switch engineID {
	case "claude":
		return "ANTHROPIC_API_KEY"
	default:
		return "GITHUB_TOKEN"
	}
}

// GetFallbackEnvVar returns the fallback environment variable name
func (config *OIDCConfig) GetFallbackEnvVar(engineID string) string {
	if config.FallbackEnvVar != "" {
		return config.FallbackEnvVar
	}

	// Default fallback based on engine
	switch engineID {
	case "claude":
		return "ANTHROPIC_API_KEY"
	default:
		return "GITHUB_TOKEN"
	}
}

// GenerateOIDCSetupStep generates a GitHub Actions step to setup OIDC token
func GenerateOIDCSetupStep(oidcConfig *OIDCConfig, engineID string) GitHubActionStep {
	var stepLines []string

	stepLines = append(stepLines, "      - name: Setup OIDC token")
	stepLines = append(stepLines, "        id: setup_oidc_token")
	stepLines = append(stepLines, "        uses: actions/github-script@v8")
	stepLines = append(stepLines, "        env:")

	// Add OIDC configuration as environment variables
	audience := oidcConfig.GetOIDCAudience(engineID)
	if audience != "" {
		stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_AUDIENCE: %s", audience))
	}

	if oidcConfig.TokenExchangeURL != "" {
		stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_EXCHANGE_URL: %s", oidcConfig.TokenExchangeURL))
	}

	envVarName := oidcConfig.GetTokenEnvVarName(engineID)
	stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_ENV_VAR_NAME: %s", envVarName))

	fallbackEnvVar := oidcConfig.GetFallbackEnvVar(engineID)
	if fallbackEnvVar != "" {
		stepLines = append(stepLines, fmt.Sprintf("          GH_AW_OIDC_FALLBACK_ENV_VAR: %s", fallbackEnvVar))
		// Add the actual fallback secret if it exists
		stepLines = append(stepLines, fmt.Sprintf("          %s: ${{ secrets.%s }}", fallbackEnvVar, fallbackEnvVar))
	}

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
	stepLines = append(stepLines, "        if: always()")
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
