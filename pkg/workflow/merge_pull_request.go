package workflow

import "github.com/github/gh-aw/pkg/logger"

var mergePullRequestLog = logger.New("workflow:merge_pull_request")

// MergePullRequestConfig holds configuration for merging pull requests with policy checks.
type MergePullRequestConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	RequiredLabels       []string `yaml:"required-labels,omitempty"`  // Labels that must ALL be present on the PR
	RequiredLabel        []string `yaml:"required-label,omitempty"`   // Disjunctive: at least one PR label must match
	AllowedLabels        []string `yaml:"allowed-labels,omitempty"`   // Deprecated: use required-label
	RequiredBranch       []string `yaml:"required-branch,omitempty"`  // Glob patterns: source branch must match one
	AllowedBranches      []string `yaml:"allowed-branches,omitempty"` // Deprecated: use required-branch
}

// parseMergePullRequestConfig handles merge-pull-request configuration.
func (c *Compiler) parseMergePullRequestConfig(outputMap map[string]any) *MergePullRequestConfig {
	configData, exists := outputMap["merge-pull-request"]
	if !exists {
		return nil
	}

	mergePullRequestLog.Print("Parsing merge-pull-request config")
	cfg := &MergePullRequestConfig{}
	if configMap, ok := configData.(map[string]any); ok {
		cfg.RequiredLabels = ParseStringArrayFromConfig(configMap, "required-labels", mergePullRequestLog)
		// Parse required-label (preferred) with fallback to deprecated allowed-labels
		cfg.RequiredLabel = ParseStringArrayFromConfig(configMap, "required-label", mergePullRequestLog)
		if len(cfg.RequiredLabel) == 0 {
			cfg.RequiredLabel = ParseStringArrayFromConfig(configMap, "allowed-labels", mergePullRequestLog)
		}
		// Parse required-branch (preferred) with fallback to deprecated allowed-branches
		cfg.RequiredBranch = ParseStringArrayFromConfig(configMap, "required-branch", mergePullRequestLog)
		if len(cfg.RequiredBranch) == 0 {
			cfg.RequiredBranch = ParseStringArrayFromConfig(configMap, "allowed-branches", mergePullRequestLog)
		}
		c.parseBaseSafeOutputConfig(configMap, &cfg.BaseSafeOutputConfig, 1)
		mergePullRequestLog.Printf("Parsed merge-pull-request config: requiredLabels=%v, requiredLabel=%v, requiredBranch=%v", cfg.RequiredLabels, cfg.RequiredLabel, cfg.RequiredBranch)
		return cfg
	}

	// merge-pull-request: null enables defaults
	cfg.Max = defaultIntStr(1)
	return cfg
}
