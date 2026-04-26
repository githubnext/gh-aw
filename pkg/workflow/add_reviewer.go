package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var addReviewerLog = logger.New("workflow:add_reviewer")

// AddReviewerConfig holds configuration for adding reviewers to PRs from agent output
type AddReviewerConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	Reviewers              []string `yaml:"reviewers,omitempty"`      // Optional list of allowed reviewers. If omitted, any reviewers are allowed.
	TeamReviewers          []string `yaml:"team-reviewers,omitempty"` // Optional list of allowed team reviewers. If omitted, any team reviewers are allowed.
}

// parseAddReviewerConfig handles add-reviewer configuration
func (c *Compiler) parseAddReviewerConfig(outputMap map[string]any) *AddReviewerConfig {
	// Check if the key exists
	if _, exists := outputMap["add-reviewer"]; !exists {
		return nil
	}

	// Get config data for pre-processing before YAML unmarshaling
	configData, _ := outputMap["add-reviewer"].(map[string]any)

	// Pre-process reviewers fields to convert single string to array BEFORE unmarshaling
	if configData != nil {
		if reviewers, exists := configData["reviewers"]; exists {
			if reviewerStr, ok := reviewers.(string); ok {
				configData["reviewers"] = []string{reviewerStr}
			}
		}
		if teamReviewers, exists := configData["team-reviewers"]; exists {
			if teamReviewerStr, ok := teamReviewers.(string); ok {
				configData["team-reviewers"] = []string{teamReviewerStr}
			}
		}
	}

	// Pre-process templatable int fields
	if err := preprocessIntFieldAsString(configData, "max", addReviewerLog); err != nil {
		addReviewerLog.Printf("Invalid max value: %v", err)
		return nil
	}

	config := parseConfigScaffold(outputMap, "add-reviewer", addReviewerLog, func(err error) *AddReviewerConfig {
		addReviewerLog.Printf("Failed to unmarshal config: %v", err)
		// For backward compatibility, handle nil/empty config
		return &AddReviewerConfig{}
	})
	if config == nil {
		return nil
	}

	// Set default max if not specified
	if config.Max == nil {
		config.Max = defaultIntStr(3)
	}

	addReviewerLog.Printf("Parsed add-reviewer config: allowed_reviewers=%d, target=%s", len(config.Reviewers), config.Target)

	return config
}
