package workflow

import (
	"encoding/json"
	"maps"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

// ========================================
// Safe Output Configuration Generation Helpers
// ========================================
//
// This file contains helper functions that reduce duplication in
// generateSafeOutputsConfig. They extract common patterns for:
// - Generating max value configs with defaults
// - Generating configs with allowed fields (labels, repos, etc.)
// - Generating configs with optional target fields
// - Building MCP tool definitions for custom safe-output jobs

var safeOutputsConfigGenLog = logger.New("workflow:safe_outputs_config_generation_helpers")

// resolveMaxForConfig resolves a templatable max *string to a config value.
// For expression strings (e.g. "${{ inputs.max }}"), the expression is stored
// as-is so GitHub Actions can resolve it at runtime.
// For literal numeric strings, the parsed integer is used.
// Falls back to defaultMax if max is nil or zero.
func resolveMaxForConfig(max *string, defaultMax int) any {
	if max != nil {
		v := *max
		if strings.HasPrefix(v, "${{") {
			return v // expression: evaluated at runtime by GitHub Actions
		}
		if n := templatableIntValue(max); n > 0 {
			return n
		}
	}
	return defaultMax
}

// generateMaxConfig creates a simple config map with just a max value
func generateMaxConfig(max *string, defaultMax int) map[string]any {
	config := make(map[string]any)
	config["max"] = resolveMaxForConfig(max, defaultMax)
	return config
}

// generateMaxWithAllowedLabelsConfig creates a config with max and optional allowed_labels
func generateMaxWithAllowedLabelsConfig(max *string, defaultMax int, allowedLabels []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(allowedLabels) > 0 {
		config["allowed_labels"] = allowedLabels
	}
	return config
}

// generateMaxWithTargetConfig creates a config with max and optional target field
func generateMaxWithTargetConfig(max *string, defaultMax int, target string) map[string]any {
	config := make(map[string]any)
	if target != "" {
		config["target"] = target
	}
	config["max"] = resolveMaxForConfig(max, defaultMax)
	return config
}

// generateMaxWithAllowedConfig creates a config with max and optional allowed list
func generateMaxWithAllowedConfig(max *string, defaultMax int, allowed []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(allowed) > 0 {
		config["allowed"] = allowed
	}
	return config
}

// generateMaxWithAllowedAndBlockedConfig creates a config with max, optional allowed list, and optional blocked list
func generateMaxWithAllowedAndBlockedConfig(max *string, defaultMax int, allowed []string, blocked []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(allowed) > 0 {
		config["allowed"] = allowed
	}
	if len(blocked) > 0 {
		config["blocked"] = blocked
	}
	return config
}

// generateMaxWithDiscussionFieldsConfig creates a config with discussion-specific filter fields
func generateMaxWithDiscussionFieldsConfig(max *string, defaultMax int, requiredCategory string, requiredLabels []string, requiredTitlePrefix string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if requiredCategory != "" {
		config["required_category"] = requiredCategory
	}
	if len(requiredLabels) > 0 {
		config["required_labels"] = requiredLabels
	}
	if requiredTitlePrefix != "" {
		config["required_title_prefix"] = requiredTitlePrefix
	}
	return config
}

// generateMaxWithReviewersConfig creates a config with max and optional reviewers list
func generateMaxWithReviewersConfig(max *string, defaultMax int, reviewers []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(reviewers) > 0 {
		config["reviewers"] = reviewers
	}
	return config
}

// generateAssignToAgentConfig creates a config with optional max, default_agent, target, and allowed
func generateAssignToAgentConfig(max *string, defaultMax int, defaultAgent string, target string, allowed []string) map[string]any {
	if safeOutputsConfigGenLog.Enabled() {
		safeOutputsConfigGenLog.Printf("Generating assign-to-agent config: max=%v, defaultMax=%d, defaultAgent=%s, target=%s, allowed_count=%d",
			max, defaultMax, defaultAgent, target, len(allowed))
	}
	config := make(map[string]any)
	config["max"] = resolveMaxForConfig(max, defaultMax)
	if defaultAgent != "" {
		config["default_agent"] = defaultAgent
	}
	if target != "" {
		config["target"] = target
	}
	if len(allowed) > 0 {
		config["allowed"] = allowed
	}
	return config
}

// generatePullRequestConfig creates a config with all pull request fields including target-repo,
// allowed_repos, base_branch, draft, reviewers, title_prefix, fallback_as_issue, and more.
func generatePullRequestConfig(prConfig *CreatePullRequestsConfig, defaultMax int) map[string]any {
	safeOutputsConfigGenLog.Printf("Generating pull request config: max=%v, allowEmpty=%v, autoMerge=%v, expires=%d, labels_count=%d, targetRepo=%s",
		prConfig.Max, prConfig.AllowEmpty, prConfig.AutoMerge, prConfig.Expires, len(prConfig.AllowedLabels), prConfig.TargetRepoSlug)

	additionalFields := make(map[string]any)
	if len(prConfig.AllowedLabels) > 0 {
		additionalFields["allowed_labels"] = prConfig.AllowedLabels
	}
	// Pass allow_empty flag to MCP server so it can skip patch generation
	if prConfig.AllowEmpty != nil && *prConfig.AllowEmpty == "true" {
		additionalFields["allow_empty"] = true
	}
	// Pass auto_merge flag to enable auto-merge for the pull request
	if prConfig.AutoMerge != nil && *prConfig.AutoMerge == "true" {
		additionalFields["auto_merge"] = true
	}
	// Pass expires to configure pull request expiration
	if prConfig.Expires > 0 {
		additionalFields["expires"] = prConfig.Expires
	}
	// Pass base_branch to configure the base branch for the pull request
	if prConfig.BaseBranch != "" {
		additionalFields["base_branch"] = prConfig.BaseBranch
	}
	// Pass draft flag to create the pull request as a draft
	if prConfig.Draft != nil && *prConfig.Draft == "true" {
		additionalFields["draft"] = true
	}
	// Pass reviewers to assign reviewers to the pull request
	if len(prConfig.Reviewers) > 0 {
		additionalFields["reviewers"] = prConfig.Reviewers
	}
	// Pass title_prefix to prepend to pull request titles
	if prConfig.TitlePrefix != "" {
		additionalFields["title_prefix"] = prConfig.TitlePrefix
	}
	// Pass fallback_as_issue if explicitly configured
	if prConfig.FallbackAsIssue != nil {
		additionalFields["fallback_as_issue"] = *prConfig.FallbackAsIssue
	}
	// Pass preserve_branch_name to skip the random salt suffix
	if prConfig.PreserveBranchName {
		additionalFields["preserve_branch_name"] = true
	}

	// Use generateTargetConfigWithRepos to include target-repo and allowed_repos
	targetConfig := SafeOutputTargetConfig{
		TargetRepoSlug: prConfig.TargetRepoSlug,
		AllowedRepos:   prConfig.AllowedRepos,
	}
	return generateTargetConfigWithRepos(targetConfig, prConfig.Max, defaultMax, additionalFields)
}

// generateHideCommentConfig creates a config with max and optional allowed_reasons
func generateHideCommentConfig(max *string, defaultMax int, allowedReasons []string) map[string]any {
	config := generateMaxConfig(max, defaultMax)
	if len(allowedReasons) > 0 {
		config["allowed_reasons"] = allowedReasons
	}
	return config
}

// generateTargetConfigWithRepos creates a config with target, target-repo, allowed_repos, and optional fields.
// Note on naming conventions:
// - "target-repo" uses hyphen to match frontmatter YAML format (key in config.json)
// - "allowed_repos" uses underscore to match JavaScript handler expectations (see repo_helpers.cjs)
// This inconsistency is intentional to maintain compatibility with existing handler code.
func generateTargetConfigWithRepos(targetConfig SafeOutputTargetConfig, max *string, defaultMax int, additionalFields map[string]any) map[string]any {
	safeOutputsConfigGenLog.Printf("Generating target config: target=%s, targetRepo=%s, allowedReposCount=%d, additionalFieldsCount=%d",
		targetConfig.Target, targetConfig.TargetRepoSlug, len(targetConfig.AllowedRepos), len(additionalFields))

	config := generateMaxConfig(max, defaultMax)

	// Add target if specified
	if targetConfig.Target != "" {
		config["target"] = targetConfig.Target
	}

	// Add target-repo if specified (use hyphenated key for consistency with frontmatter)
	if targetConfig.TargetRepoSlug != "" {
		config["target-repo"] = targetConfig.TargetRepoSlug
	}

	// Add allowed_repos if specified (use underscore for consistency with handler code)
	if len(targetConfig.AllowedRepos) > 0 {
		config["allowed_repos"] = targetConfig.AllowedRepos
	}

	// Add any additional fields
	maps.Copy(config, additionalFields)

	return config
}

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
