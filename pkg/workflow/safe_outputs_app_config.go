package workflow

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsAppLog = logger.New("workflow:safe_outputs_app")

// ========================================
// GitHub App Configuration
// ========================================

// GitHubAppConfig holds configuration for GitHub App-based token minting
type GitHubAppConfig struct {
	AppID        string            `yaml:"app-id,omitempty"`       // GitHub App ID (e.g., "${{ vars.APP_ID }}")
	PrivateKey   string            `yaml:"private-key,omitempty"`  // GitHub App private key (e.g., "${{ secrets.APP_PRIVATE_KEY }}")
	Owner        string            `yaml:"owner,omitempty"`        // Optional: owner of the GitHub App installation (defaults to current repository owner)
	Repositories []string          `yaml:"repositories,omitempty"` // Optional: comma or newline-separated list of repositories to grant access to
	Permissions  map[string]string `yaml:"permissions,omitempty"`  // Optional: extra permission-* fields to merge into the minted token (nested wins over job-level)
}

// ========================================
// App Configuration Parsing
// ========================================

// parseAppConfig parses the app configuration from a map
func parseAppConfig(appMap map[string]any) *GitHubAppConfig {
	safeOutputsAppLog.Print("Parsing GitHub App configuration")
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

	// Parse permissions (optional) - extra permission-* fields to merge into the minted token
	if perms, exists := appMap["permissions"]; exists {
		if permsMap, ok := perms.(map[string]any); ok {
			appConfig.Permissions = make(map[string]string, len(permsMap))
			for key, val := range permsMap {
				if valStr, ok := val.(string); ok {
					appConfig.Permissions[key] = valStr
				} else {
					safeOutputsAppLog.Printf("Ignoring github-app.permissions[%q]: expected string value, got %T", key, val)
				}
			}
		} else {
			safeOutputsAppLog.Printf("Ignoring github-app.permissions: expected object, got %T", perms)
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
	safeOutputsAppLog.Printf("Merging app configuration: included_configs=%d", len(includedConfigs))
	// If top-level workflow already has app configured, use it (no merge needed)
	if topSafeOutputs != nil && topSafeOutputs.GitHubApp != nil {
		safeOutputsAppLog.Print("Using top-level app configuration")
		return topSafeOutputs.GitHubApp, nil
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

		// Extract app from the safe-outputs.github-app field
		if appData, exists := safeOutputsConfig["github-app"]; exists {
			if appMap, ok := appData.(map[string]any); ok {
				appConfig := parseAppConfig(appMap)

				// Return first valid app configuration found
				if appConfig.AppID != "" && appConfig.PrivateKey != "" {
					safeOutputsAppLog.Print("Found valid app configuration in included config")
					return appConfig, nil
				}
			}
		}
	}

	safeOutputsAppLog.Print("No app configuration found in included configs")
	return nil, nil
}

// ========================================
// GitHub App Token Steps Generation
// ========================================

// buildGitHubAppTokenMintStep generates the step to mint a GitHub App installation access token
// Permissions are automatically computed from the safe output job requirements.
// fallbackRepoExpr overrides the default ${{ github.event.repository.name }} fallback when
// no explicit repositories are configured (e.g. pass needs.activation.outputs.target_repo_name for
// workflow_call relay workflows so the token is scoped to the platform repo's NAME, not the full
// owner/repo slug — actions/create-github-app-token expects repo names only when owner is also set).
func (c *Compiler) buildGitHubAppTokenMintStep(app *GitHubAppConfig, permissions *Permissions, fallbackRepoExpr string) []string {
	safeOutputsAppLog.Printf("Building GitHub App token mint step: owner=%s, repos=%d", app.Owner, len(app.Repositories))
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token\n")
	steps = append(steps, "        id: safe-outputs-app-token\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	// Add owner - default to current repository owner if not specified
	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))

	// Add repositories - behavior depends on configuration:
	// - If repositories is ["*"], omit the field to allow org-wide access
	// - If repositories is a single value, use inline format
	// - If repositories has multiple values, use block scalar format (newline-separated)
	//   to ensure clarity and proper parsing by actions/create-github-app-token
	// - If repositories is empty/not specified, default to fallbackRepoExpr or the current repository
	if len(app.Repositories) == 1 && app.Repositories[0] == "*" {
		// Org-wide access: omit repositories field entirely
		safeOutputsAppLog.Print("Using org-wide GitHub App token (repositories: *)")
	} else if len(app.Repositories) == 1 {
		// Single repository: use inline format for clarity
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", app.Repositories[0]))
	} else if len(app.Repositories) > 1 {
		// Multiple repositories: use block scalar format (newline-separated)
		// This format is more readable and avoids potential issues with comma-separated parsing
		steps = append(steps, "          repositories: |-\n")
		for _, repo := range app.Repositories {
			steps = append(steps, fmt.Sprintf("            %s\n", repo))
		}
	} else {
		// No explicit repositories: use fallback expression, or default to the triggering repo's name.
		// For workflow_call relay scenarios the caller passes needs.activation.outputs.target_repo_name so
		// the token is scoped to the platform (host) repo name rather than the full owner/repo slug.
		repoExpr := fallbackRepoExpr
		if repoExpr == "" {
			repoExpr = "${{ github.event.repository.name }}"
		}
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", repoExpr))
	}

	// Always add github-api-url from environment variable
	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")

	// Add permission-* fields automatically computed from job permissions
	// Sort keys to ensure deterministic compilation order
	if permissions != nil {
		permissionFields := convertPermissionsToAppTokenFields(permissions)

		// Extract and sort keys for deterministic ordering
		keys := make([]string, 0, len(permissionFields))
		for key := range permissionFields {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Add permissions in sorted order
		for _, key := range keys {
			steps = append(steps, fmt.Sprintf("          %s: %s\n", key, permissionFields[key]))
		}
	}

	return steps
}

// convertPermissionsToAppTokenFields converts job Permissions to permission-* action inputs
// This follows GitHub's recommendation for explicit permission control
// Note: This maps all permissions (both GitHub Actions and GitHub App-only) to their
// corresponding permission-* fields in actions/create-github-app-token.
// Some GitHub Actions permissions (like 'models', 'id-token', 'attestations', 'copilot-requests')
// don't have corresponding GitHub App permissions and are skipped.
//
// For GitHub Actions permissions (actions, checks, contents, …) we use Get() so that shorthand
// permissions like "read-all" are correctly expanded.
// For GitHub App-only permissions (administration, members, organization-secrets, …) we use
// GetExplicit() so that only scopes the user actually declared are forwarded — a "read-all"
// shorthand must never accidentally grant broad GitHub App-only permissions.
func convertPermissionsToAppTokenFields(permissions *Permissions) map[string]string {
	fields := make(map[string]string)

	// Map GitHub Actions permissions to GitHub App permissions
	// See: https://github.com/actions/create-github-app-token#permissions

	// GitHub Actions permissions that also exist in GitHub App
	if level, ok := permissions.Get(PermissionActions); ok {
		fields["permission-actions"] = string(level)
	}
	if level, ok := permissions.Get(PermissionChecks); ok {
		fields["permission-checks"] = string(level)
	}
	if level, ok := permissions.Get(PermissionContents); ok {
		fields["permission-contents"] = string(level)
	}
	if level, ok := permissions.Get(PermissionDeployments); ok {
		fields["permission-deployments"] = string(level)
	}
	if level, ok := permissions.Get(PermissionIssues); ok {
		fields["permission-issues"] = string(level)
	}
	if level, ok := permissions.Get(PermissionPackages); ok {
		fields["permission-packages"] = string(level)
	}
	if level, ok := permissions.Get(PermissionPages); ok {
		fields["permission-pages"] = string(level)
	}
	if level, ok := permissions.Get(PermissionPullRequests); ok {
		fields["permission-pull-requests"] = string(level)
	}
	if level, ok := permissions.Get(PermissionSecurityEvents); ok {
		fields["permission-security-events"] = string(level)
	}
	if level, ok := permissions.Get(PermissionStatuses); ok {
		fields["permission-statuses"] = string(level)
	}
	// "permission-discussions" is a declared input in actions/create-github-app-token v3+.
	// Crucially, when ANY permission-* input is specified the action scopes the token to ONLY those
	// permissions (returning undefined → inherit-all only when zero permission-* inputs are present).
	// Since the compiler always emits other permission-* fields, omitting permission-discussions causes
	// the minted token to lack discussions access even when the GitHub App installation has that permission.
	if level, ok := permissions.Get(PermissionDiscussions); ok {
		fields["permission-discussions"] = string(level)
	}

	// GitHub App-only permissions (not available in GitHub Actions GITHUB_TOKEN).
	// Use GetExplicit() so that shorthand permissions like "read-all" do not accidentally
	// expand into broad GitHub App-only grants that the user never declared.
	// Repository-level
	if level, ok := permissions.GetExplicit(PermissionAdministration); ok {
		fields["permission-administration"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionEnvironments); ok {
		fields["permission-environments"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionGitSigning); ok {
		fields["permission-git-signing"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionVulnerabilityAlerts); ok {
		fields["permission-vulnerability-alerts"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionWorkflows); ok {
		fields["permission-workflows"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionRepositoryHooks); ok {
		fields["permission-repository-hooks"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionSingleFile); ok {
		fields["permission-single-file"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionCodespaces); ok {
		fields["permission-codespaces"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionRepositoryCustomProperties); ok {
		fields["permission-repository-custom-properties"] = string(level)
	}
	// Organization-level
	if level, ok := permissions.GetExplicit(PermissionOrganizationProj); ok {
		fields["permission-organization-projects"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionMembers); ok {
		fields["permission-members"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationAdministration); ok {
		fields["permission-organization-administration"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionTeamDiscussions); ok {
		fields["permission-team-discussions"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationHooks); ok {
		fields["permission-organization-hooks"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationMembers); ok {
		fields["permission-organization-members"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationPackages); ok {
		fields["permission-organization-packages"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationSelfHostedRunners); ok {
		fields["permission-organization-self-hosted-runners"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationCustomOrgRoles); ok {
		fields["permission-organization-custom-org-roles"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationCustomProperties); ok {
		fields["permission-organization-custom-properties"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationCustomRepositoryRoles); ok {
		fields["permission-organization-custom-repository-roles"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationAnnouncementBanners); ok {
		fields["permission-organization-announcement-banners"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationEvents); ok {
		fields["permission-organization-events"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationPlan); ok {
		fields["permission-organization-plan"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationUserBlocking); ok {
		fields["permission-organization-user-blocking"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationPersonalAccessTokenReqs); ok {
		fields["permission-organization-personal-access-token-requests"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationPersonalAccessTokens); ok {
		fields["permission-organization-personal-access-tokens"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationCopilot); ok {
		fields["permission-organization-copilot"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionOrganizationCodespaces); ok {
		fields["permission-organization-codespaces"] = string(level)
	}
	// User-level
	if level, ok := permissions.GetExplicit(PermissionEmailAddresses); ok {
		fields["permission-email-addresses"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionCodespacesLifecycleAdmin); ok {
		fields["permission-codespaces-lifecycle-admin"] = string(level)
	}
	if level, ok := permissions.GetExplicit(PermissionCodespacesMetadata); ok {
		fields["permission-codespaces-metadata"] = string(level)
	}

	// Note: The following GitHub Actions permissions do NOT have GitHub App equivalents
	// and are therefore not mapped to permission-* fields:
	// - models: no GitHub App permission for AI model access
	// - id-token: not applicable to GitHub Apps (OIDC-specific)
	// - attestations: no GitHub App permission for this
	// - copilot-requests: GitHub Actions-specific Copilot authentication token
	// - metadata: GitHub App metadata permission is automatically included (read-only)

	return fields
}

// buildGitHubAppTokenInvalidationStep generates the step to invalidate the GitHub App token
// This step always runs (even on failure) to ensure tokens are properly cleaned up
// Only runs if a token was successfully minted
func (c *Compiler) buildGitHubAppTokenInvalidationStep() []string {
	var steps []string

	steps = append(steps, "      - name: Invalidate GitHub App token\n")
	steps = append(steps, "        if: always() && steps.safe-outputs-app-token.outputs.token != ''\n")
	steps = append(steps, "        env:\n")
	steps = append(steps, "          TOKEN: ${{ steps.safe-outputs-app-token.outputs.token }}\n")
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

// ========================================
// Activation Token Steps Generation
// ========================================

// buildActivationAppTokenMintStep generates the step to mint a GitHub App installation access token
// for use in the pre-activation (reaction) and activation (status comment) jobs.
func (c *Compiler) buildActivationAppTokenMintStep(app *GitHubAppConfig, permissions *Permissions) []string {
	safeOutputsAppLog.Printf("Building activation GitHub App token mint step: owner=%s", app.Owner)
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token for activation\n")
	steps = append(steps, "        id: activation-app-token\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	// Add owner - default to current repository owner if not specified
	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))

	// Default to current repository
	steps = append(steps, "          repositories: ${{ github.event.repository.name }}\n")

	// Always add github-api-url from environment variable
	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")

	// Add permission-* fields automatically computed from job permissions
	if permissions != nil {
		permissionFields := convertPermissionsToAppTokenFields(permissions)

		keys := make([]string, 0, len(permissionFields))
		for key := range permissionFields {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			steps = append(steps, fmt.Sprintf("          %s: %s\n", key, permissionFields[key]))
		}
	}

	return steps
}

// resolveActivationToken returns the GitHub token to use for activation steps (reactions, status comments).
// Priority: GitHub App minted token > custom github-token > GITHUB_TOKEN (default)
//
// When returning the app token reference, callers MUST ensure that buildActivationAppTokenMintStep
// has already been called to generate the 'activation-app-token' step, since this function returns
// a reference to that step's output (${{ steps.activation-app-token.outputs.token }}).
func (c *Compiler) resolveActivationToken(data *WorkflowData) string {
	if data.ActivationGitHubApp != nil {
		return "${{ steps.activation-app-token.outputs.token }}"
	}
	if data.ActivationGitHubToken != "" {
		return data.ActivationGitHubToken
	}
	return "${{ secrets.GITHUB_TOKEN }}"
}
