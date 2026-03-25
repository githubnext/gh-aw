package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var markPullRequestAsReadyForReviewLog = logger.New("workflow:mark_pull_request_as_ready_for_review")

// MarkPullRequestAsReadyForReviewConfig holds configuration for marking draft PRs as ready for review
type MarkPullRequestAsReadyForReviewConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
}

// parseMarkPullRequestAsReadyForReviewConfig handles mark-pull-request-as-ready-for-review configuration
func (c *Compiler) parseMarkPullRequestAsReadyForReviewConfig(outputMap map[string]any) *MarkPullRequestAsReadyForReviewConfig {
	config := parseConfigScaffold(outputMap, "mark-pull-request-as-ready-for-review", markPullRequestAsReadyForReviewLog,
		func(err error) *MarkPullRequestAsReadyForReviewConfig {
			return nil
		})
	if config == nil {
		return nil
	}

	// Postprocess: parse common target configuration (target, target-repo) and
	// filter configuration (required-labels, required-title-prefix) from the raw map,
	// as these fields require additional extraction beyond YAML unmarshaling.
	var configMap map[string]any
	if configVal, exists := outputMap["mark-pull-request-as-ready-for-review"]; exists {
		if cfgMap, ok := configVal.(map[string]any); ok {
			configMap = cfgMap
		} else {
			configMap = make(map[string]any)
		}
	}

	targetConfig, _ := ParseTargetConfig(configMap)
	config.SafeOutputTargetConfig = targetConfig

	filterConfig := ParseFilterConfig(configMap)
	config.SafeOutputFilterConfig = filterConfig

	return config
}
