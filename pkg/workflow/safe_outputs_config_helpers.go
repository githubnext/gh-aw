package workflow

import (
	"encoding/json"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Configuration Helpers
// ========================================
//
// This file contains helper utilities used by the safe-outputs compiler:
// - Token resolution for PR checkout and project operations
// - JSON serialisation of custom job/script names for the handler manager

var safeOutputsConfigGenLog = logger.New("workflow:safe_outputs_config_generation_helpers")

// computeEffectivePRCheckoutToken returns the token to use for PR checkout and git operations.
// Applies the following precedence (highest to lowest):
//  1. Per-config PAT: create-pull-request.github-token
//  2. Per-config PAT: push-to-pull-request-branch.github-token
//  3. GitHub App minted token (if a github-app is configured)
//  4. safe-outputs level PAT: safe-outputs.github-token
//  5. Default fallback via getEffectiveSafeOutputGitHubToken()
//
// Per-config tokens take precedence over the GitHub App so that individual operations
// can override the app-wide authentication with a dedicated PAT when needed.
//
// This is used by buildSharedPRCheckoutSteps and buildHandlerManagerStep to ensure consistent token handling.
//
// Returns:
//   - token: the effective GitHub Actions token expression to use for git operations
//   - isCustom: true when a custom non-default token was explicitly configured (per-config PAT, app, or safe-outputs PAT)
func computeEffectivePRCheckoutToken(safeOutputs *SafeOutputsConfig) (token string, isCustom bool) {
	if safeOutputs == nil {
		return getEffectiveSafeOutputGitHubToken(""), false
	}

	// Per-config PAT tokens take highest precedence (overrides GitHub App)
	var createPRToken string
	if safeOutputs.CreatePullRequests != nil {
		createPRToken = safeOutputs.CreatePullRequests.GitHubToken
	}
	var pushToPRBranchToken string
	if safeOutputs.PushToPullRequestBranch != nil {
		pushToPRBranchToken = safeOutputs.PushToPullRequestBranch.GitHubToken
	}
	perConfigToken := createPRToken
	if perConfigToken == "" {
		perConfigToken = pushToPRBranchToken
	}
	if perConfigToken != "" {
		return getEffectiveSafeOutputGitHubToken(perConfigToken), true
	}

	// GitHub App token takes precedence over the safe-outputs level PAT
	if safeOutputs.GitHubApp != nil {
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template placeholder, not a hardcoded credential
		return "${{ steps.safe-outputs-app-token.outputs.token }}", true
	}

	// safe-outputs level PAT as final custom option
	if safeOutputs.GitHubToken != "" {
		return getEffectiveSafeOutputGitHubToken(safeOutputs.GitHubToken), true
	}

	// No custom token - fall back to default
	return getEffectiveSafeOutputGitHubToken(""), false
}

// computeEffectiveProjectToken computes the effective project token using the precedence:
//  1. Per-config token (e.g., from update-project, create-project-status-update)
//  2. Safe-outputs level token
//  3. Magic secret fallback via getEffectiveProjectGitHubToken()
func computeEffectiveProjectToken(perConfigToken string, safeOutputsToken string) string {
	token := perConfigToken
	if token == "" {
		token = safeOutputsToken
	}
	return getEffectiveProjectGitHubToken(token)
}

// computeProjectURLAndToken computes the project URL and token from the various project-related
// safe-output configurations. Priority order: update-project > create-project-status-update > create-project.
// Returns the project URL (may be empty for create-project) and the effective token.
func computeProjectURLAndToken(safeOutputs *SafeOutputsConfig) (projectURL, projectToken string) {
	if safeOutputs == nil {
		return "", ""
	}

	safeOutputsToken := safeOutputs.GitHubToken

	// Check update-project first (highest priority)
	if safeOutputs.UpdateProjects != nil && safeOutputs.UpdateProjects.Project != "" {
		projectURL = safeOutputs.UpdateProjects.Project
		projectToken = computeEffectiveProjectToken(safeOutputs.UpdateProjects.GitHubToken, safeOutputsToken)
		safeOutputsConfigGenLog.Printf("Setting GH_AW_PROJECT_URL from update-project config: %s", projectURL)
		safeOutputsConfigGenLog.Printf("Setting GH_AW_PROJECT_GITHUB_TOKEN from update-project config")
		return
	}

	// Check create-project-status-update second
	if safeOutputs.CreateProjectStatusUpdates != nil && safeOutputs.CreateProjectStatusUpdates.Project != "" {
		projectURL = safeOutputs.CreateProjectStatusUpdates.Project
		projectToken = computeEffectiveProjectToken(safeOutputs.CreateProjectStatusUpdates.GitHubToken, safeOutputsToken)
		safeOutputsConfigGenLog.Printf("Setting GH_AW_PROJECT_URL from create-project-status-update config: %s", projectURL)
		safeOutputsConfigGenLog.Printf("Setting GH_AW_PROJECT_GITHUB_TOKEN from create-project-status-update config")
		return
	}

	// Check create-project for token even if no URL is set (create-project doesn't have a project URL field)
	// This ensures GH_AW_PROJECT_GITHUB_TOKEN is set when create-project is configured
	if safeOutputs.CreateProjects != nil {
		projectToken = computeEffectiveProjectToken(safeOutputs.CreateProjects.GitHubToken, safeOutputsToken)
		safeOutputsConfigGenLog.Printf("Setting GH_AW_PROJECT_GITHUB_TOKEN from create-project config")
	}

	return
}

// buildCustomSafeOutputJobsJSON builds a JSON mapping of custom safe output job names to empty
// strings, for use in the GH_AW_SAFE_OUTPUT_JOBS env var of the handler manager step.
// This allows the handler manager to silently skip messages handled by custom safe-output job
// steps rather than reporting them as "No handler loaded for message type '...'".
func buildCustomSafeOutputJobsJSON(data *WorkflowData) string {
	if data.SafeOutputs == nil || len(data.SafeOutputs.Jobs) == 0 {
		return ""
	}

	// Build mapping of normalized job names to empty strings (no URL output for custom jobs)
	jobMapping := make(map[string]string, len(data.SafeOutputs.Jobs))
	for jobName := range data.SafeOutputs.Jobs {
		normalizedName := stringutil.NormalizeSafeOutputIdentifier(jobName)
		jobMapping[normalizedName] = ""
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(jobMapping))
	for k := range jobMapping {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make(map[string]string, len(keys))
	for _, k := range keys {
		ordered[k] = jobMapping[k]
	}

	jsonBytes, err := json.Marshal(ordered)
	if err != nil {
		safeOutputsConfigGenLog.Printf("Warning: failed to marshal custom safe output jobs: %v", err)
		return ""
	}
	return string(jsonBytes)
}
