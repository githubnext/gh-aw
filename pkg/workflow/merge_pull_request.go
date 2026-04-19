package workflow

import "github.com/github/gh-aw/pkg/logger"

var mergePullRequestLog = logger.New("workflow:merge_pull_request")

// MergePullRequestConfig holds configuration for merging pull requests with policy checks.
type MergePullRequestConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	RequiredLabels       []string `yaml:"required-labels,omitempty"`  // Labels that must be present on the PR
	AllowedLabels        []string `yaml:"allowed-labels,omitempty"`   // Exact label names; at least one PR label must match when configured
	AllowedBranches      []string `yaml:"allowed-branches,omitempty"` // Glob patterns for source branch names
	AllowedFiles         []string `yaml:"allowed-files,omitempty"`    // Glob patterns; all changed files must match when configured
	ProtectedFiles       []string `yaml:"protected-files,omitempty"`  // Glob patterns; any match blocks merge
}

// parseMergePullRequestConfig handles merge-pull-request configuration.
func (c *Compiler) parseMergePullRequestConfig(outputMap map[string]any) *MergePullRequestConfig {
	configData, exists := outputMap["merge-pull-request"]
	if !exists {
		return nil
	}

	cfg := &MergePullRequestConfig{}
	if configMap, ok := configData.(map[string]any); ok {
		cfg.RequiredLabels = ParseStringArrayFromConfig(configMap, "required-labels", mergePullRequestLog)
		cfg.AllowedLabels = ParseStringArrayFromConfig(configMap, "allowed-labels", mergePullRequestLog)
		cfg.AllowedBranches = ParseStringArrayFromConfig(configMap, "allowed-branches", mergePullRequestLog)
		cfg.AllowedFiles = ParseStringArrayFromConfig(configMap, "allowed-files", mergePullRequestLog)
		cfg.ProtectedFiles = ParseStringArrayFromConfig(configMap, "protected-files", mergePullRequestLog)
		c.parseBaseSafeOutputConfig(configMap, &cfg.BaseSafeOutputConfig, 1)
		return cfg
	}

	// merge-pull-request: null enables defaults
	cfg.Max = defaultIntStr(1)
	return cfg
}
