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
	AppID         string   `yaml:"app-id,omitempty"`      // GitHub App ID (e.g., "${{ vars.APP_ID }}")
	PrivateKey    string   `yaml:"private-key,omitempty"` // GitHub App private key (e.g., "${{ secrets.APP_PRIVATE_KEY }}")
	Owner         string   `yaml:"owner,omitempty"`       // Optional: owner of the GitHub App installation (defaults to current repository owner)
	Repositories  []string `yaml:"repositories,omitempty"` // Optional: comma or newline-separated list of repositories to grant access to
	GitHubAPIURL  string   `yaml:"github-api-url,omitempty"` // Optional: GitHub REST API URL (defaults to workflow's API URL)
}

// ========================================
// App Configuration Parsing
// ========================================

// parseAppConfig parses the app configuration from a map
func parseAppConfig(appMap map[string]any) *GitHubAppConfig {
	appConfig := &GitHubAppConfig{}

	// Parse app-id (required)
	if appID, exists := appMap["app-id"]; exists {
		if appIDStr, ok := appID.(string); ok {
			appConfig.AppID = appIDStr
		}
	}

	// Parse private-key (required)
	if privateKey, exists := appMap["private-key"]; exists {
		if privateKeyStr, ok := privateKey.(string); ok {
			appConfig.PrivateKey = privateKeyStr
		}
	}

	// Parse owner (optional)
	if owner, exists := appMap["owner"]; exists {
		if ownerStr, ok := owner.(string); ok {
			appConfig.Owner = ownerStr
		}
	}

	// Parse repositories (optional)
	if repos, exists := appMap["repositories"]; exists {
		if reposArray, ok := repos.([]any); ok {
			var repoStrings []string
			for _, repo := range reposArray {
				if repoStr, ok := repo.(string); ok {
					repoStrings = append(repoStrings, repoStr)
				}
			}
			appConfig.Repositories = repoStrings
		}
	}

	// Parse github-api-url (optional)
	if apiURL, exists := appMap["github-api-url"]; exists {
		if apiURLStr, ok := apiURL.(string); ok {
			appConfig.GitHubAPIURL = apiURLStr
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
				if appConfig.AppID != "" && appConfig.PrivateKey != "" {
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
// Permissions are automatically computed from the safe output job requirements
func (c *Compiler) buildGitHubAppTokenMintStep(app *GitHubAppConfig, permissions *Permissions) []string {
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token\n")
	steps = append(steps, "        id: app-token\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	// Add owner if specified
	if app.Owner != "" {
		steps = append(steps, fmt.Sprintf("          owner: %s\n", app.Owner))
	}

	// Add repositories if specified
	if len(app.Repositories) > 0 {
		reposStr := strings.Join(app.Repositories, ",")
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", reposStr))
	}

	// Add github-api-url if specified
	if app.GitHubAPIURL != "" {
		steps = append(steps, fmt.Sprintf("          github-api-url: %s\n", app.GitHubAPIURL))
	}

	// Add permission-* fields automatically computed from job permissions
	if permissions != nil {
		permissionFields := convertPermissionsToAppTokenFields(permissions)
		for key, value := range permissionFields {
			steps = append(steps, fmt.Sprintf("          %s: %s\n", key, value))
		}
	}

	return steps
}

// convertPermissionsToAppTokenFields converts job Permissions to permission-* action inputs
// This follows GitHub's recommendation for explicit permission control
func convertPermissionsToAppTokenFields(permissions *Permissions) map[string]string {
	fields := make(map[string]string)

	// Check each permission scope and add to fields if set
	if level, ok := permissions.Get(PermissionContents); ok {
		fields["permission-contents"] = string(level)
	}
	if level, ok := permissions.Get(PermissionIssues); ok {
		fields["permission-issues"] = string(level)
	}
	if level, ok := permissions.Get(PermissionPullRequests); ok {
		fields["permission-pull-requests"] = string(level)
	}
	if level, ok := permissions.Get(PermissionDiscussions); ok {
		fields["permission-discussions"] = string(level)
	}
	if level, ok := permissions.Get(PermissionChecks); ok {
		fields["permission-checks"] = string(level)
	}
	if level, ok := permissions.Get(PermissionStatuses); ok {
		fields["permission-statuses"] = string(level)
	}
	if level, ok := permissions.Get(PermissionActions); ok {
		fields["permission-actions"] = string(level)
	}
	if level, ok := permissions.Get(PermissionDeployments); ok {
		fields["permission-deployments"] = string(level)
	}
	if level, ok := permissions.Get(PermissionSecurityEvents); ok {
		fields["permission-security-events"] = string(level)
	}
	if level, ok := permissions.Get(PermissionModels); ok {
		fields["permission-models"] = string(level)
	}

	return fields
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
