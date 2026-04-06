package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var reportIncompleteLog = logger.New("workflow:report_incomplete")

// ReportIncompleteConfig holds configuration for the report_incomplete safe output.
// report_incomplete is a structured signal that the agent could not complete its
// assigned task due to an infrastructure or tool failure (e.g., MCP server crash,
// missing authentication, inaccessible repository).
//
// When an agent emits report_incomplete, gh-aw activates failure handling even
// when the agent process exits 0 and other safe outputs were also emitted.
// This prevents semantically-empty outputs (e.g., a comment describing tool
// failures) from being classified as a successful result.
type ReportIncompleteConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// parseReportIncompleteConfig handles report_incomplete configuration.
func (c *Compiler) parseReportIncompleteConfig(outputMap map[string]any) *ReportIncompleteConfig {
	configData, exists := outputMap["report-incomplete"]
	if !exists {
		reportIncompleteLog.Print("No report-incomplete configuration found")
		return nil
	}

	// Explicitly disabled: report-incomplete: false
	if configBool, ok := configData.(bool); ok && !configBool {
		reportIncompleteLog.Print("report-incomplete explicitly disabled")
		return nil
	}

	cfg := &ReportIncompleteConfig{}

	// Enabled with no value: report-incomplete: (nil)
	if configData == nil {
		cfg.Max = defaultIntStr(5)
		reportIncompleteLog.Print("report-incomplete enabled with defaults")
		return cfg
	}

	if configMap, ok := configData.(map[string]any); ok {
		c.parseBaseSafeOutputConfig(configMap, &cfg.BaseSafeOutputConfig, 5)
		reportIncompleteLog.Printf("Parsed report-incomplete configuration")
	}

	return cfg
}
