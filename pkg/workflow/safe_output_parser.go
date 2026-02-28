package workflow

import "github.com/github/gh-aw/pkg/logger"

var safeOutputParserLog = logger.New("workflow:safe_output_parser")

// SafeOutputTargetConfig contains common target-related fields for safe output configurations.
// Embed this in safe output config structs that support targeting specific items.
type SafeOutputTargetConfig struct {
	Target         string   `yaml:"target,omitempty"`        // Target for the operation: "triggering" (default), "*" (any item), or explicit number
	TargetRepoSlug string   `yaml:"target-repo,omitempty"`   // Target repository in format "owner/repo" for cross-repository operations
	AllowedRepos   []string `yaml:"allowed-repos,omitempty"` // List of additional repositories that operations can target (additionally to the target-repo)
}

// SafeOutputFilterConfig contains common filtering fields for safe output configurations.
// Embed this in safe output config structs that support filtering by labels or title prefix.
type SafeOutputFilterConfig struct {
	RequiredLabels      []string `yaml:"required-labels,omitempty"`       // Required labels for the operation
	RequiredTitlePrefix string   `yaml:"required-title-prefix,omitempty"` // Required title prefix for the operation
}

// SafeOutputDiscussionFilterConfig extends SafeOutputFilterConfig with discussion-specific fields.
type SafeOutputDiscussionFilterConfig struct {
	SafeOutputFilterConfig `yaml:",inline"`
	RequiredCategory       string `yaml:"required-category,omitempty"` // Required category for discussion operations
}

// CloseJobConfig represents common configuration for close operations (close-issue, close-discussion, close-pull-request)
type CloseJobConfig struct {
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
}

// ListJobConfig represents common configuration for list-based operations (add-labels, add-reviewer, assign-milestone)
type ListJobConfig struct {
	SafeOutputTargetConfig `yaml:",inline"`
	Allowed                []string `yaml:"allowed,omitempty"` // Optional list of allowed values
	Blocked                []string `yaml:"blocked,omitempty"` // Optional list of blocked patterns (supports glob patterns)
}

// ParseTargetConfig parses target and target-repo fields from a config map.
// Returns the parsed SafeOutputTargetConfig and a boolean indicating if there was a validation error.
// target-repo accepts "*" (wildcard) to indicate that any repository can be targeted.
func ParseTargetConfig(configMap map[string]any) (SafeOutputTargetConfig, bool) {
	safeOutputParserLog.Print("Parsing target config from map")
	config := SafeOutputTargetConfig{}

	// Parse target
	if target, exists := configMap["target"]; exists {
		if targetStr, ok := target.(string); ok {
			config.Target = targetStr
			safeOutputParserLog.Printf("Target set to: %s", targetStr)
		}
	}

	// Parse target-repo; wildcard "*" is allowed and means "any repository"
	config.TargetRepoSlug = parseTargetRepoFromConfig(configMap)

	return config, false
}

// ParseFilterConfig parses required-labels and required-title-prefix fields from a config map.
func ParseFilterConfig(configMap map[string]any) SafeOutputFilterConfig {
	safeOutputParserLog.Print("Parsing filter config from map")
	config := SafeOutputFilterConfig{}

	// Parse required-labels
	config.RequiredLabels = parseRequiredLabelsFromConfig(configMap)
	if len(config.RequiredLabels) > 0 {
		safeOutputParserLog.Printf("Parsed %d required labels", len(config.RequiredLabels))
	}

	// Parse required-title-prefix
	config.RequiredTitlePrefix = parseRequiredTitlePrefixFromConfig(configMap)

	return config
}

// parseRequiredLabelsFromConfig extracts and validates required-labels from a config map.
// Returns a slice of label strings, or nil if not present or invalid.
func parseRequiredLabelsFromConfig(configMap map[string]any) []string {
	return ParseStringArrayFromConfig(configMap, "required-labels", safeOutputParserLog)
}

// parseRequiredTitlePrefixFromConfig extracts required-title-prefix from a config map.
// Returns the prefix string, or empty string if not present or invalid.
func parseRequiredTitlePrefixFromConfig(configMap map[string]any) string {
	return extractStringFromMap(configMap, "required-title-prefix", safeOutputParserLog)
}
