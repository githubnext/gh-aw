package workflow

import "github.com/github/gh-aw/pkg/logger"

var mergePullRequestLog = logger.New("workflow:merge_pull_request")

// MergePullRequestConfig holds configuration for merging pull requests with policy checks.
type MergePullRequestConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	RequiredLabels         []string `yaml:"required-labels,omitempty"`       // Labels that must ALL be present on the PR
	AllowedBranches        []string `yaml:"allowed-branches,omitempty"`      // Glob patterns for source branch names; PR branch must match one
	RequiredTitlePrefix    string   `yaml:"required-title-prefix,omitempty"` // Title prefix the PR must have
	AllowedLabels          []string `yaml:"allowed-labels,omitempty"`        // Deprecated: use required-labels
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
		if len(cfg.RequiredLabels) == 0 {
			// Deprecated: allowed-labels is migrated to required-labels by the codemod
			cfg.RequiredLabels = ParseStringArrayFromConfig(configMap, "allowed-labels", mergePullRequestLog)
		}
		targetConfig, _ := ParseTargetConfig(configMap)
		cfg.SafeOutputTargetConfig = targetConfig
		cfg.AllowedBranches = ParseStringArrayFromConfig(configMap, "allowed-branches", mergePullRequestLog)
		cfg.RequiredTitlePrefix = extractStringFromMap(configMap, "required-title-prefix", mergePullRequestLog)
		c.parseBaseSafeOutputConfig(configMap, &cfg.BaseSafeOutputConfig, 1)
		mergePullRequestLog.Printf("Parsed merge-pull-request config: requiredLabels=%v, allowedBranches=%v, requiredTitlePrefix=%q", cfg.RequiredLabels, cfg.AllowedBranches, cfg.RequiredTitlePrefix)
		return cfg
	}

	// merge-pull-request: null enables defaults
	cfg.Max = defaultIntStr(1)
	return cfg
}
