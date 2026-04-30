package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var missingDataLog = logger.New("workflow:missing_data")
var missingToolLog = logger.New("workflow:missing_tool")
var reportIncompleteLog = logger.New("workflow:report_incomplete")

// IssueReportingConfig holds configuration shared by safe-output types that create GitHub issues
// (missing-data and missing-tool). Both types have identical fields; the yaml tags on the
// parent struct fields give them their distinct YAML keys.
type IssueReportingConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	CreateIssue          *string  `yaml:"create-issue,omitempty"` // Whether to create/update issues (default: true). Supports literal bool or GitHub Actions expression.
	TitlePrefix          string   `yaml:"title-prefix,omitempty"` // Prefix for issue titles
	Labels               []string `yaml:"labels,omitempty"`       // Labels to add to created issues
}

// Type aliases so existing code (compiler_types.go, tests, etc.) continues to compile unchanged.
// Both resolve to IssueReportingConfig; the distinct names preserve semantic clarity at usage sites.
type MissingDataConfig = IssueReportingConfig
type MissingToolConfig = IssueReportingConfig

// ReportIncompleteConfig holds configuration for the report_incomplete safe output.
// report_incomplete is a structured signal that the agent could not complete its
// assigned task due to an infrastructure or tool failure (e.g., MCP server crash,
// missing authentication, inaccessible repository).
//
// When an agent emits report_incomplete, gh-aw activates failure handling even
// when the agent process exits 0 and other safe outputs were also emitted.
// This prevents semantically-empty outputs (e.g., a comment describing tool
// failures) from being classified as a successful result.
//
// ReportIncompleteConfig is a type alias for IssueReportingConfig so that it
// supports the same create-issue, title-prefix, and labels configuration fields
// as missing-tool and missing-data.
type ReportIncompleteConfig = IssueReportingConfig

func (c *Compiler) parseMissingDataConfig(outputMap map[string]any) *MissingDataConfig {
	return c.parseIssueReportingConfig(outputMap, "missing-data", "[missing data]", missingDataLog)
}

func (c *Compiler) parseMissingToolConfig(outputMap map[string]any) *MissingToolConfig {
	return c.parseIssueReportingConfig(outputMap, "missing-tool", "[missing tool]", missingToolLog)
}

// parseReportIncompleteConfig handles report_incomplete configuration.
func (c *Compiler) parseReportIncompleteConfig(outputMap map[string]any) *ReportIncompleteConfig {
	return c.parseIssueReportingConfig(outputMap, "report-incomplete", "[incomplete]", reportIncompleteLog)
}

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
		trueVal := "true"
		cfg.CreateIssue = &trueVal
		cfg.TitlePrefix = defaultTitle
		cfg.Labels = []string{}
		return cfg
	}

	if configMap, ok := configData.(map[string]any); ok {
		log.Printf("Parsing %s configuration from map", yamlKey)
		c.parseBaseSafeOutputConfig(configMap, &cfg.BaseSafeOutputConfig, 0)

		// Pre-process create-issue to support literal booleans and GitHub Actions expressions.
		if err := preprocessBoolFieldAsString(configMap, "create-issue", log); err != nil {
			log.Printf("Invalid create-issue value for %s: %v", yamlKey, err)
			return nil
		}

		if createIssueVal, exists := configMap["create-issue"]; exists {
			if createIssueStr, ok := createIssueVal.(string); ok {
				cfg.CreateIssue = &createIssueStr
				log.Printf("create-issue: %s", createIssueStr)
			}
		} else {
			trueVal := "true"
			cfg.CreateIssue = &trueVal
		}

		if titlePrefix, exists := configMap["title-prefix"]; exists {
			if titlePrefixStr, ok := titlePrefix.(string); ok {
				cfg.TitlePrefix = titlePrefixStr
				log.Printf("title-prefix: %s", titlePrefixStr)
			}
		} else {
			cfg.TitlePrefix = defaultTitle
		}

		if _, exists := configMap["labels"]; exists {
			cfg.Labels = ParseStringArrayFromConfig(configMap, "labels", log)
			log.Printf("labels: %v", cfg.Labels)
		} else {
			cfg.Labels = []string{}
		}
	}

	return cfg
}
