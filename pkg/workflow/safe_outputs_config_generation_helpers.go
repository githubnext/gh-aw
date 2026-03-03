package workflow

import (
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsConfigGenLog = logger.New("workflow:safe_outputs_config_generation_helpers")

// ========================================
// Safe Output Configuration Generation Helpers
// ========================================
//
// This file contains helper functions to reduce duplication in safe output
// configuration generation. These helpers extract common patterns for:
// - Generating max value configs with defaults
// - Generating configs with allowed fields (labels, repos, etc.)
// - Generating configs with optional target fields
//
// The goal is to make generateSafeOutputsConfig more maintainable by
// extracting repetitive code patterns into reusable functions.

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
