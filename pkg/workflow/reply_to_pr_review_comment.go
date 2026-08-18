package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var replyToPRReviewCommentLog = logger.New("workflow:reply_to_pr_review_comment")

// ReplyToPullRequestReviewCommentConfig holds configuration for replying to PR review comments.
// Uses the GitHub REST API to create reply comments on existing review comment threads.
type ReplyToPullRequestReviewCommentConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
	Footer                 *string `yaml:"footer,omitempty"` // Whether to add AI-generated footer to replies
}

// parseReplyToPullRequestReviewCommentConfig handles reply-to-pull-request-review-comment configuration
func (c *Compiler) parseReplyToPullRequestReviewCommentConfig(outputMap map[string]any) *ReplyToPullRequestReviewCommentConfig {
	if configData, exists := outputMap["reply-to-pull-request-review-comment"]; exists {
		replyToPRReviewCommentLog.Print("Parsing reply-to-pull-request-review-comment configuration")
		config := &ReplyToPullRequestReviewCommentConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			replyToPRReviewCommentLog.Print("Found reply-to-pull-request-review-comment config map")

			// Parse common base fields with default max of 10
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 10)

			// Parse target config (target, target-repo, allowed-repos)
			targetConfig, isInvalid := parseSafeOutputTargetConfig(configMap, replyToPRReviewCommentLog, safeOutputTargetConfigOptions{
				parseTarget:       true,
				parseTargetRepo:   true,
				parseAllowedRepos: true,
			})
			if isInvalid {
				return nil // Invalid configuration, return nil to cause validation error
			}
			config.SafeOutputTargetConfig = targetConfig

			// Parse footer as templatable bool
			if err := preprocessBoolFieldAsString(configMap, "footer", replyToPRReviewCommentLog); err != nil {
				replyToPRReviewCommentLog.Printf("Invalid footer value: %v", err)
				return nil
			}
			if footer, ok := configMap["footer"].(string); ok {
				config.Footer = &footer
			}

			replyToPRReviewCommentLog.Printf("Parsed reply-to-pull-request-review-comment config: max=%d", config.Max)
		} else {
			// If configData is nil or not a map, still set the default max
			config.Max = defaultIntStr(10)
		}

		return config
	}

	return nil
}
