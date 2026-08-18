package workflow

import "github.com/github/gh-aw/pkg/logger"

var safeOutputParserLog = logger.New("workflow:safe_outputs_parser")

// SafeOutputTargetConfig contains common target-related fields for safe output configurations.
// Embed this in safe output config structs that support targeting specific items.
type SafeOutputTargetConfig struct {
	Target         string   `yaml:"target,omitempty"`        // Target for the operation: "triggering" (default), "*" (any item), or explicit number
	TargetRepoSlug string   `yaml:"target-repo,omitempty"`   // Target repository in format "owner/repo" for cross-repository operations
	AllowedRepos   []string `yaml:"allowed-repos,omitempty"` // List of additional repositories that operations can target (additionally to the target-repo)
}

type safeOutputTargetConfigOptions struct {
	parseTarget                 bool
	parseTargetRepo             bool
	allowTargetRepoWildcard     bool
	parseAllowedRepos           bool
	allowAllowedReposExpression bool
}

// SafeOutputAllowedLabelsConfig contains the shared allowed-labels field for safe output configurations.
// Embed this in safe output config structs that restrict which labels may be used.
type SafeOutputAllowedLabelsConfig struct {
	AllowedLabels []string `yaml:"allowed-labels,omitempty"` // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
}

// SafeOutputFilterConfig contains common filtering fields for safe output configurations.
// Embed this in safe output config structs that support filtering by labels or title prefix.
type SafeOutputFilterConfig struct {
	SafeOutputAllowedLabelsConfig `yaml:",inline"`
	RequiredLabels                []string `yaml:"required-labels,omitempty"`       // Required labels for the operation (ALL must match)
	RequiredTitlePrefix           string   `yaml:"required-title-prefix,omitempty"` // Required title prefix for the operation
	TitlePrefix                   string   `yaml:"title-prefix,omitempty"`          // Deprecated alias for required-title-prefix
}

// SafeOutputDiscussionFilterConfig extends SafeOutputFilterConfig with discussion-specific fields.
type SafeOutputDiscussionFilterConfig struct {
	SafeOutputFilterConfig `yaml:",inline"`
	RequiredCategory       string `yaml:"required-category,omitempty"` // Required category for discussion operations
}

// SafeOutputAllowBlockConfig contains common allow/block lists for safe output configurations.
// Embed this in safe output config structs that support optional allowed/blocked value filters.
type SafeOutputAllowBlockConfig struct {
	Allowed []string `yaml:"allowed,omitempty"` // Optional list of allowed values
	Blocked []string `yaml:"blocked,omitempty"` // Optional list of blocked patterns (supports glob patterns)
}

// CloseJobConfig represents common configuration for close operations (close-issue, close-discussion, close-pull-request)
type CloseJobConfig struct {
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
}

// ListJobConfig represents common configuration for list-based operations (add-labels, add-reviewer, assign-milestone)
type ListJobConfig struct {
	SafeOutputTargetConfig     `yaml:",inline"`
	SafeOutputAllowBlockConfig `yaml:",inline"`
}

// ParseTargetConfig parses target and target-repo fields from a config map.
// Returns the parsed SafeOutputTargetConfig and a boolean indicating if there was a validation error.
// target-repo accepts "*" (wildcard) to indicate that any repository can be targeted.
func ParseTargetConfig(configMap map[string]any) (SafeOutputTargetConfig, bool) {
	return parseSafeOutputTargetConfig(configMap, safeOutputParserLog, safeOutputTargetConfigOptions{
		parseTarget:             true,
		parseTargetRepo:         true,
		allowTargetRepoWildcard: true,
		parseAllowedRepos:       true,
	})
}

func parseSafeOutputTargetConfig(configMap map[string]any, debugLog *logger.Logger, opts safeOutputTargetConfigOptions) (SafeOutputTargetConfig, bool) {
	if debugLog != nil {
		debugLog.Print("Parsing target config from map")
	}

	config := SafeOutputTargetConfig{}

	if opts.parseTarget {
		config.Target = extractStringFromMap(configMap, "target", debugLog)
		if config.Target != "" && debugLog != nil {
			debugLog.Printf("Target set to: %s", config.Target)
		}
	}

	if opts.parseTargetRepo {
		config.TargetRepoSlug = extractStringFromMap(configMap, "target-repo", debugLog)
		if config.TargetRepoSlug == "*" && !opts.allowTargetRepoWildcard {
			if debugLog != nil {
				debugLog.Print("Invalid target-repo: wildcard '*' is not allowed")
			}
			return SafeOutputTargetConfig{}, true
		}
	}

	if opts.parseAllowedRepos {
		if opts.allowAllowedReposExpression {
			config.AllowedRepos = ParseStringArrayOrExprFromConfig(configMap, "allowed-repos", debugLog)
		} else {
			config.AllowedRepos = ParseStringArrayFromConfig(configMap, "allowed-repos", debugLog)
		}
	}

	return config, false
}

// ParseFilterConfig parses required-labels and required-title-prefix fields from a config map.
func ParseFilterConfig(configMap map[string]any) SafeOutputFilterConfig {
	safeOutputParserLog.Print("Parsing filter config from map")
	config := SafeOutputFilterConfig{}

	// Parse required-labels (ALL must match)
	config.RequiredLabels = ParseStringArrayFromConfig(configMap, "required-labels", safeOutputParserLog)
	if len(config.RequiredLabels) > 0 {
		safeOutputParserLog.Printf("Parsed %d required labels", len(config.RequiredLabels))
	}

	// Parse required-title-prefix (preferred) with fallback to deprecated title-prefix
	config.RequiredTitlePrefix = extractStringFromMap(configMap, "required-title-prefix", safeOutputParserLog)
	if config.RequiredTitlePrefix == "" {
		config.RequiredTitlePrefix = extractStringFromMap(configMap, "title-prefix", safeOutputParserLog)
	}

	return config
}
