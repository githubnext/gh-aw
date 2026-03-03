package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var missingDataLog = logger.New("workflow:missing_data")

func (c *Compiler) parseMissingDataConfig(outputMap map[string]any) *MissingDataConfig {
	return c.parseIssueReportingConfig(outputMap, "missing-data", "[missing data]", missingDataLog)
}
