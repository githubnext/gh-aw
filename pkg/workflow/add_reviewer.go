package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var addReviewerLog = logger.New("workflow:add_reviewer")

// AddReviewerConfig holds configuration for adding reviewers to PRs from agent output
type AddReviewerConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Reviewers              []string `yaml:"reviewers,omitempty"` // Optional list of allowed reviewers. If omitted, any reviewers are allowed.
}

// parseAddReviewerConfig handles add-reviewer configuration
func (c *Compiler) parseAddReviewerConfig(outputMap map[string]any) *AddReviewerConfig {
	if configData, exists := outputMap["add-reviewer"]; exists {
		addReviewerLog.Print("Parsing add-reviewer configuration")
		addReviewerConfig := &AddReviewerConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse reviewers (supports both string and array)
			if reviewers, exists := configMap["reviewers"]; exists {
				if reviewerStr, ok := reviewers.(string); ok {
					// Single string format
					addReviewerConfig.Reviewers = []string{reviewerStr}
				} else if reviewersArray, ok := reviewers.([]any); ok {
					// Array format
					var reviewerStrings []string
					for _, reviewer := range reviewersArray {
						if reviewerStr, ok := reviewer.(string); ok {
							reviewerStrings = append(reviewerStrings, reviewerStr)
						}
					}
					addReviewerConfig.Reviewers = reviewerStrings
				}
			}

			// Parse target config (target, target-repo)
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				addReviewerLog.Print("Invalid target configuration for add-reviewer")
				return nil // Invalid configuration, return nil to cause validation error
			}
			addReviewerConfig.SafeOutputTargetConfig = targetConfig
			addReviewerLog.Printf("Parsed add-reviewer config: allowed_reviewers=%d, target=%s", len(addReviewerConfig.Reviewers), targetConfig.Target)

			// Parse common base fields (github-token, max) with default max of 3
			c.parseBaseSafeOutputConfig(configMap, &addReviewerConfig.BaseSafeOutputConfig, 3)
		} else {
			// If configData is nil or not a map (e.g., "add-reviewer:" with no value),
			// still set the default max
			addReviewerConfig.Max = 3
		}

		return addReviewerConfig
	}

	return nil
}
