package workflow

import (
	"fmt"
	"sort"
	"strings"
)

var githubAppPermissionsLog = newValidationLogger("github_app_permissions")

// validateGitHubAppOnlyPermissions validates that when GitHub App-only permissions
// are specified in the workflow, a GitHub App is configured somewhere in the workflow.
//
// GitHub App-only permissions (e.g., members, administration, secrets) cannot be exercised
// through the GITHUB_TOKEN — they require a GitHub App installation access token. When such
// permissions are declared, a GitHub App must be configured via one of:
//   - tools.github.github-app
//   - safe-outputs.github-app
//   - the top-level github-app field (for the activation/pre-activation jobs)
//
// Returns an error if GitHub App-only permissions are used without any GitHub App configured.
func validateGitHubAppOnlyPermissions(workflowData *WorkflowData) error {
	githubAppPermissionsLog.Print("Starting GitHub App-only permissions validation")

	if workflowData.Permissions == "" {
		githubAppPermissionsLog.Print("No permissions defined, validation passed")
		return nil
	}

	permissions := NewPermissionsParser(workflowData.Permissions).ToPermissions()
	if permissions == nil {
		githubAppPermissionsLog.Print("Could not parse permissions, validation passed")
		return nil
	}

	// Find any GitHub App-only permission scopes that are set
	var appOnlyScopes []PermissionScope
	for _, scope := range GetAllGitHubAppOnlyScopes() {
		if _, exists := permissions.Get(scope); exists {
			appOnlyScopes = append(appOnlyScopes, scope)
		}
	}

	if len(appOnlyScopes) == 0 {
		githubAppPermissionsLog.Print("No GitHub App-only permissions found, validation passed")
		return nil
	}

	githubAppPermissionsLog.Printf("Found %d GitHub App-only permissions, checking for GitHub App configuration", len(appOnlyScopes))

	// Check if any GitHub App is configured
	if hasGitHubAppConfigured(workflowData) {
		githubAppPermissionsLog.Print("GitHub App is configured, validation passed")
		return nil
	}

	// Format the error message
	return formatGitHubAppRequiredError(appOnlyScopes)
}

// hasGitHubAppConfigured returns true if a GitHub App is configured anywhere in the workflow
func hasGitHubAppConfigured(workflowData *WorkflowData) bool {
	// Check tools.github.github-app
	if workflowData.ParsedTools != nil &&
		workflowData.ParsedTools.GitHub != nil &&
		workflowData.ParsedTools.GitHub.GitHubApp != nil {
		githubAppPermissionsLog.Print("Found GitHub App in tools.github")
		return true
	}

	// Check safe-outputs.github-app
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.GitHubApp != nil {
		githubAppPermissionsLog.Print("Found GitHub App in safe-outputs")
		return true
	}

	// Check the activation job github-app
	if workflowData.ActivationGitHubApp != nil {
		githubAppPermissionsLog.Print("Found GitHub App in activation config")
		return true
	}

	return false
}

// formatGitHubAppRequiredError formats an error message when GitHub App-only permissions
// are used without a GitHub App configured.
func formatGitHubAppRequiredError(appOnlyScopes []PermissionScope) error {
	// Sort for deterministic output
	scopeStrs := make([]string, len(appOnlyScopes))
	for i, s := range appOnlyScopes {
		scopeStrs[i] = string(s)
	}
	sort.Strings(scopeStrs)

	var lines []string
	lines = append(lines, "GitHub App-only permissions require a GitHub App to be configured.")
	lines = append(lines, "The following permissions are not supported by the GITHUB_TOKEN and")
	lines = append(lines, "can only be exercised through a GitHub App installation access token:")
	lines = append(lines, "")
	for _, s := range scopeStrs {
		lines = append(lines, "  - "+s)
	}
	lines = append(lines, "")
	lines = append(lines, "To fix this, configure a GitHub App in your workflow. For example:")
	lines = append(lines, ""+"tools:")
	lines = append(lines, "  github:")
	lines = append(lines, "    github-app:")
	lines = append(lines, "      app-id: ${{ vars.APP_ID }}")
	lines = append(lines, "      private-key: ${{ secrets.APP_PRIVATE_KEY }}")
	lines = append(lines, "")
	lines = append(lines, "Or in the safe-outputs section:")
	lines = append(lines, "safe-outputs:")
	lines = append(lines, "  github-app:")
	lines = append(lines, "    app-id: ${{ vars.APP_ID }}")
	lines = append(lines, "    private-key: ${{ secrets.APP_PRIVATE_KEY }}")

	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}
