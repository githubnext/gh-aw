package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var createPRReviewCommentLog = logger.New("workflow:create_pr_review_comment")

// CreatePullRequestReviewCommentsConfig holds configuration for creating GitHub pull request review comments from agent output
type CreatePullRequestReviewCommentsConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Side                 string   `yaml:"side,omitempty"`          // Side of the diff: "LEFT" or "RIGHT" (default: "RIGHT")
	Target               string   `yaml:"target,omitempty"`        // Target for comments: "triggering" (default), "*" (any PR), or explicit PR number
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`   // Target repository in format "owner/repo" for cross-repository PR review comments
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"` // List of additional repositories that PR review comments can be added to (additionally to the target-repo)
}

func (c *Compiler) parsePullRequestReviewCommentsConfig(outputMap map[string]any) *CreatePullRequestReviewCommentsConfig {
	if _, exists := outputMap["create-pull-request-review-comment"]; !exists {
		createPRReviewCommentLog.Printf("Configuration not found")
		return nil
	}

	createPRReviewCommentLog.Printf("Parsing PR review comment configuration")

	configData := outputMap["create-pull-request-review-comment"]
	prReviewCommentsConfig := &CreatePullRequestReviewCommentsConfig{Side: "RIGHT"} // Default side is RIGHT

	if configMap, ok := configData.(map[string]any); ok {
		// Parse side
		if side, exists := configMap["side"]; exists {
			if sideStr, ok := side.(string); ok {
				// Validate side value
				if sideStr == "LEFT" || sideStr == "RIGHT" {
					prReviewCommentsConfig.Side = sideStr
				}
			}
		}

		// Parse target
		if target, exists := configMap["target"]; exists {
			if targetStr, ok := target.(string); ok {
				prReviewCommentsConfig.Target = targetStr
			}
		}

		// Parse target-repo using shared helper with validation
		targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
		if isInvalid {
			return nil // Invalid configuration, return nil to cause validation error
		}
		prReviewCommentsConfig.TargetRepoSlug = targetRepoSlug

		// Parse common base fields with default max of 10
		c.parseBaseSafeOutputConfig(configMap, &prReviewCommentsConfig.BaseSafeOutputConfig, 10)
	} else {
		// If configData is nil or not a map (e.g., "create-pull-request-review-comment:" with no value),
		// still set the default max
		prReviewCommentsConfig.Max = defaultIntStr(10)
	}

	return prReviewCommentsConfig
}
