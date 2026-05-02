package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var resolvePRReviewThreadLog = logger.New("workflow:resolve_pr_review_thread")

// ResolvePullRequestReviewThreadConfig holds configuration for resolving PR review threads.
// By default, resolution is scoped to the triggering PR only. When target, target-repo,
// or allowed-repos are specified, cross-repository thread resolution is supported.
type ResolvePullRequestReviewThreadConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	WarnOnError            bool `yaml:"warn-on-error,omitempty"` // If true, errors are treated as warnings instead of failures
}

// parseResolvePullRequestReviewThreadConfig handles resolve-pull-request-review-thread configuration
func (c *Compiler) parseResolvePullRequestReviewThreadConfig(outputMap map[string]any) *ResolvePullRequestReviewThreadConfig {
	if configData, exists := outputMap["resolve-pull-request-review-thread"]; exists {
		resolvePRReviewThreadLog.Print("Parsing resolve-pull-request-review-thread configuration")
		config := &ResolvePullRequestReviewThreadConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			resolvePRReviewThreadLog.Print("Found resolve-pull-request-review-thread config map")

			// Parse common base fields with default max of 10
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 10)

			// Parse target config (target, target-repo, allowed-repos)
			targetConfig, _ := ParseTargetConfig(configMap)
			config.SafeOutputTargetConfig = targetConfig

			// Parse warn-on-error: when true, API errors are treated as non-fatal warnings
			if warnOnError, exists := configMap["warn-on-error"]; exists {
				if warnOnErrorBool, ok := warnOnError.(bool); ok {
					config.WarnOnError = warnOnErrorBool
				}
			}

			resolvePRReviewThreadLog.Printf("Parsed resolve-pull-request-review-thread config: max=%d, target_repo=%s, warn-on-error=%t", templatableIntValue(config.Max), config.TargetRepoSlug, config.WarnOnError)
		} else {
			// If configData is nil or not a map, still set the default max
			config.Max = defaultIntStr(10)
		}

		return config
	}

	return nil
}
