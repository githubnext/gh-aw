package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
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
	// Check if the key exists
	if _, exists := outputMap["mark-pull-request-as-ready-for-review"]; !exists {
		markPullRequestAsReadyForReviewLog.Print("No configuration found for mark-pull-request-as-ready-for-review")
		return nil
	}

	markPullRequestAsReadyForReviewLog.Print("Parsing mark-pull-request-as-ready-for-review configuration")

	// Unmarshal into typed config struct
	var config MarkPullRequestAsReadyForReviewConfig
	if err := unmarshalConfig(outputMap, "mark-pull-request-as-ready-for-review", &config, markPullRequestAsReadyForReviewLog); err != nil {
		return nil
	}

	// Parse common target configuration (target, target-repo)
	ParseTargetConfig(&config.SafeOutputTargetConfig, outputMap, "mark-pull-request-as-ready-for-review", markPullRequestAsReadyForReviewLog)

	// Parse filter configuration (required-labels, required-title-prefix)
	ParseFilterConfig(&config.SafeOutputFilterConfig, outputMap, "mark-pull-request-as-ready-for-review", markPullRequestAsReadyForReviewLog)

	return &config
}
