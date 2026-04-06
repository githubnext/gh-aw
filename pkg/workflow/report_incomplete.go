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
//
// ReportIncompleteConfig is a type alias for IssueReportingConfig so that it
// supports the same create-issue, title-prefix, and labels configuration fields
// as missing-tool and missing-data.
type ReportIncompleteConfig = IssueReportingConfig

// parseReportIncompleteConfig handles report_incomplete configuration.
func (c *Compiler) parseReportIncompleteConfig(outputMap map[string]any) *ReportIncompleteConfig {
	return c.parseIssueReportingConfig(outputMap, "report-incomplete", "[incomplete]", reportIncompleteLog)
}
