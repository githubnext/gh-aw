package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
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
	// Check if the key exists
	if _, exists := outputMap["add-reviewer"]; !exists {
		return nil
	}

	addReviewerLog.Print("Parsing add-reviewer configuration")

	// Get config data for pre-processing before YAML unmarshaling
	configData, _ := outputMap["add-reviewer"].(map[string]any)

	// Pre-process templatable int fields
	if err := preprocessIntFieldAsString(configData, "max", addReviewerLog); err != nil {
		addReviewerLog.Printf("Invalid max value: %v", err)
		return nil
	}

	// Unmarshal into typed config struct
	var config AddReviewerConfig
	if err := unmarshalConfig(outputMap, "add-reviewer", &config, addReviewerLog); err != nil {
		addReviewerLog.Printf("Failed to unmarshal config: %v", err)
		// For backward compatibility, handle nil/empty config
		config = AddReviewerConfig{}
	}

	// Set default max if not specified
	if config.Max == nil {
		config.Max = defaultIntStr(3)
	}

	// Validate target-repo (wildcard "*" is not allowed for safe outputs)
	if validateTargetRepoSlug(config.TargetRepoSlug, addReviewerLog) {
		return nil
	}

	addReviewerLog.Printf("Parsed add-reviewer config: allowed_reviewers=%d, target=%s", len(config.Reviewers), config.Target)

	return &config
}
