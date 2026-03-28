package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

// IssueReportingConfig holds configuration shared by safe-output types that create GitHub issues
// (missing-data and missing-tool). Both types have identical fields; the yaml tags on the
// parent struct fields give them their distinct YAML keys.
type IssueReportingConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	CreateIssue          bool     `yaml:"create-issue,omitempty"` // Whether to create/update issues (default: true)
	TitlePrefix          string   `yaml:"title-prefix,omitempty"` // Prefix for issue titles
	Labels               []string `yaml:"labels,omitempty"`       // Labels to add to created issues
}

// Type aliases so existing code (compiler_types.go, tests, etc.) continues to compile unchanged.
// Both resolve to IssueReportingConfig; the distinct names preserve semantic clarity at usage sites.
type MissingDataConfig = IssueReportingConfig
type MissingToolConfig = IssueReportingConfig

func (c *Compiler) parseIssueReportingConfig(outputMap map[string]any, yamlKey, defaultTitle string, log *logger.Logger) *IssueReportingConfig {
	configData, exists := outputMap[yamlKey]
	if !exists {
		return nil
	}

	// Explicitly disabled: missing-data: false
	if configBool, ok := configData.(bool); ok && !configBool {
		log.Printf("%s configuration explicitly disabled", yamlKey)
		return nil
	}

	cfg := &IssueReportingConfig{}

	// Enabled with no value: missing-data: (nil)
	if configData == nil {
		log.Printf("%s configuration enabled with defaults", yamlKey)
		cfg.CreateIssue = true
		cfg.TitlePrefix = defaultTitle
		cfg.Labels = []string{}
		return cfg
	}

	if configMap, ok := configData.(map[string]any); ok {
		log.Printf("Parsing %s configuration from map", yamlKey)
		c.parseBaseSafeOutputConfig(configMap, &cfg.BaseSafeOutputConfig, 0)

		if createIssue, exists := configMap["create-issue"]; exists {
			if createIssueBool, ok := createIssue.(bool); ok {
				cfg.CreateIssue = createIssueBool
				log.Printf("create-issue: %v", createIssueBool)
			}
		} else {
			cfg.CreateIssue = true
		}

		if titlePrefix, exists := configMap["title-prefix"]; exists {
			if titlePrefixStr, ok := titlePrefix.(string); ok {
				cfg.TitlePrefix = titlePrefixStr
				log.Printf("title-prefix: %s", titlePrefixStr)
			}
		} else {
			cfg.TitlePrefix = defaultTitle
		}

		if labels, exists := configMap["labels"]; exists {
			if labelsArray, ok := labels.([]any); ok {
				var labelStrings []string
				for _, label := range labelsArray {
					if labelStr, ok := label.(string); ok {
						labelStrings = append(labelStrings, labelStr)
					}
				}
				cfg.Labels = labelStrings
				log.Printf("labels: %v", labelStrings)
			}
		} else {
			cfg.Labels = []string{}
		}
	}

	return cfg
}
