package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ========================================
// GitHub App Configuration
// ========================================

// GitHubAppConfig holds configuration for GitHub App-based token minting
type GitHubAppConfig struct {
	ID            string   `yaml:"id,omitempty"`             // GitHub App ID (e.g., "${{ vars.APP_ID }}")
	Secret        string   `yaml:"secret,omitempty"`         // GitHub App private key (e.g., "${{ secrets.APP_PRIVATE_KEY }}")
	RepositoryIDs []string `yaml:"repository-ids,omitempty"` // Optional: specific repository IDs to scope the token
}

// ========================================
// App Configuration Parsing
// ========================================

// parseAppConfig parses the app configuration from a map
func parseAppConfig(appMap map[string]any) *GitHubAppConfig {
	appConfig := &GitHubAppConfig{}

	// Parse id (required)
	if id, exists := appMap["id"]; exists {
		if idStr, ok := id.(string); ok {
			appConfig.ID = idStr
		}
	}

	// Parse secret (required)
	if secret, exists := appMap["secret"]; exists {
		if secretStr, ok := secret.(string); ok {
			appConfig.Secret = secretStr
		}
	}

	// Parse repository-ids (optional)
	if repoIDs, exists := appMap["repository-ids"]; exists {
		if repoIDsArray, ok := repoIDs.([]any); ok {
			var repoIDStrings []string
			for _, repoID := range repoIDsArray {
				if repoIDStr, ok := repoID.(string); ok {
					repoIDStrings = append(repoIDStrings, repoIDStr)
				}
			}
			appConfig.RepositoryIDs = repoIDStrings
		}
	}

	return appConfig
}

// ========================================
// App Configuration Merging
// ========================================

// mergeAppFromIncludedConfigs merges app configuration from included safe-outputs configurations
// If the top-level workflow has an app configured, it takes precedence
// Otherwise, the first app configuration found in included configs is used
func (c *Compiler) mergeAppFromIncludedConfigs(topSafeOutputs *SafeOutputsConfig, includedConfigs []string) (*GitHubAppConfig, error) {
	// If top-level workflow already has app configured, use it (no merge needed)
	if topSafeOutputs != nil && topSafeOutputs.App != nil {
		return topSafeOutputs.App, nil
	}

	// Otherwise, find the first app configuration in included configs
	for _, configJSON := range includedConfigs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}

		// Parse the safe-outputs configuration
		var safeOutputsConfig map[string]any
		if err := json.Unmarshal([]byte(configJSON), &safeOutputsConfig); err != nil {
			continue // Skip invalid JSON
		}

		// Extract app from the safe-outputs.app field
		if appData, exists := safeOutputsConfig["app"]; exists {
			if appMap, ok := appData.(map[string]any); ok {
				appConfig := parseAppConfig(appMap)

				// Return first valid app configuration found
				if appConfig.ID != "" && appConfig.Secret != "" {
					return appConfig, nil
				}
			}
		}
	}

	return nil, nil
}

// ========================================
// GitHub App Token Steps Generation
// ========================================

// buildGitHubAppTokenMintStep generates the step to mint a GitHub App installation access token
func (c *Compiler) buildGitHubAppTokenMintStep(app *GitHubAppConfig) []string {
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token\n")
	steps = append(steps, "        id: app-token\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.ID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.Secret))

	// Add repository-ids if specified
	if len(app.RepositoryIDs) > 0 {
		repoIDsStr := strings.Join(app.RepositoryIDs, ",")
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", repoIDsStr))
	}

	return steps
}

// buildGitHubAppTokenInvalidationStep generates the step to invalidate the GitHub App token
// This step always runs (even on failure) to ensure tokens are properly cleaned up
// Only runs if a token was successfully minted
func (c *Compiler) buildGitHubAppTokenInvalidationStep() []string {
	var steps []string

	steps = append(steps, "      - name: Invalidate GitHub App token\n")
	steps = append(steps, "        if: always() && steps.app-token.outputs.token != ''\n")
	steps = append(steps, "        env:\n")
	steps = append(steps, "          TOKEN: ${{ steps.app-token.outputs.token }}\n")
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          echo \"Revoking GitHub App installation token...\"\n")
	steps = append(steps, "          # GitHub CLI will auth with the token being revoked.\n")
	steps = append(steps, "          gh api \\\n")
	steps = append(steps, "            --method DELETE \\\n")
	steps = append(steps, "            -H \"Authorization: token $TOKEN\" \\\n")
	steps = append(steps, "            /installation/token || echo \"Token revoke may already be expired.\"\n")
	steps = append(steps, "          \n")
	steps = append(steps, "          echo \"Token invalidation step complete.\"\n")

	return steps
}
