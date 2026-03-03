package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var missingToolLog = logger.New("workflow:missing_tool")

func (c *Compiler) parseMissingToolConfig(outputMap map[string]any) *MissingToolConfig {
	return c.parseIssueReportingConfig(outputMap, "missing-tool", "[missing tool]", missingToolLog)
}
