package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var markPullRequestAsReadyForReviewLog = logger.New("workflow:mark_pull_request_as_ready_for_review")

// MarkPullRequestAsReadyForReviewConfig holds configuration for marking pull requests as ready for review
type MarkPullRequestAsReadyForReviewConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
}

// parseMarkPullRequestAsReadyForReviewConfig handles mark-pull-request-as-ready-for-review configuration
func (c *Compiler) parseMarkPullRequestAsReadyForReviewConfig(outputMap map[string]any) *MarkPullRequestAsReadyForReviewConfig {
	markPullRequestAsReadyForReviewLog.Print("Parsing mark-pull-request-as-ready-for-review configuration")

	if configData, exists := outputMap["mark-pull-request-as-ready-for-review"]; exists {
		config := &MarkPullRequestAsReadyForReviewConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base safe output config (max, github-token) with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 1)

			// Parse target config using shared helper (handles target and target-repo)
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				// target-repo validation error (wildcard not allowed)
				markPullRequestAsReadyForReviewLog.Print("Target repo validation failed")
				return nil
			}
			config.SafeOutputTargetConfig = targetConfig

			// Parse filter config (required-labels, required-title-prefix)
			config.SafeOutputFilterConfig = ParseFilterConfig(configMap)
		}

		markPullRequestAsReadyForReviewLog.Printf("Parsed mark-pull-request-as-ready-for-review config: max=%d", config.Max)
		return config
	}

	markPullRequestAsReadyForReviewLog.Print("No mark-pull-request-as-ready-for-review configuration found")
	return nil
}
