package workflow

import "github.com/github/gh-aw/pkg/logger"

var dismissPullRequestReviewLog = logger.New("workflow:dismiss_pull_request_review")

// DismissPullRequestReviewConfig holds configuration for dismissing pull request reviews.
type DismissPullRequestReviewConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
}

// parseDismissPullRequestReviewConfig handles dismiss-pull-request-review configuration.
func (c *Compiler) parseDismissPullRequestReviewConfig(outputMap map[string]any) *DismissPullRequestReviewConfig {
	var configData any
	if value, exists := outputMap["dismiss-pull-request-review"]; exists {
		configData = value
	} else if value, exists := outputMap["dismiss-review"]; exists {
		// Backward-compatible alias.
		configData = value
	} else {
		return nil
	}

	dismissPullRequestReviewLog.Print("Parsing dismiss-pull-request-review configuration")
	config := &DismissPullRequestReviewConfig{}

	if configMap, ok := configData.(map[string]any); ok {
		// Parse common base fields with default max of 10.
		c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 10)

		// Parse target config (target, target-repo, allowed-repos).
		targetConfig, isInvalid := ParseTargetConfig(configMap)
		if isInvalid {
			return nil
		}
		config.SafeOutputTargetConfig = targetConfig

		// Parse filter config (required-labels, required-title-prefix).
		config.SafeOutputFilterConfig = ParseFilterConfig(configMap)
	} else {
		// If configData is nil or not a map, still set the default max.
		config.Max = defaultIntStr(10)
	}

	return config
}
