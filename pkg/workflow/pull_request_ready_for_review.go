package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var pullRequestReadyForReviewLog = logger.New("workflow:pull_request_ready_for_review")

// PullRequestReadyForReviewConfig holds configuration for marking pull requests as ready for review
type PullRequestReadyForReviewConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
}

// parsePullRequestReadyForReviewConfig handles pull-request-ready-for-review configuration
func (c *Compiler) parsePullRequestReadyForReviewConfig(outputMap map[string]any) *PullRequestReadyForReviewConfig {
	pullRequestReadyForReviewLog.Print("Parsing pull-request-ready-for-review configuration")

	if configData, exists := outputMap["pull-request-ready-for-review"]; exists {
		config := &PullRequestReadyForReviewConfig{}
		config.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base safe output config (max, github-token)
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig)

			// Parse target config using shared helper (handles target and target-repo)
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				// target-repo validation error (wildcard not allowed)
				pullRequestReadyForReviewLog.Print("Target repo validation failed")
				return nil
			}
			config.SafeOutputTargetConfig = targetConfig

			// Parse filter config (required-labels, required-title-prefix)
			config.SafeOutputFilterConfig = ParseFilterConfig(configMap)
		}

		pullRequestReadyForReviewLog.Printf("Parsed pull-request-ready-for-review config: max=%d", config.Max)
		return config
	}

	pullRequestReadyForReviewLog.Print("No pull-request-ready-for-review configuration found")
	return nil
}
