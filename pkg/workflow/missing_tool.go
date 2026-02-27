package workflow

import (
	"errors"

	"github.com/github/gh-aw/pkg/logger"
)

var missingToolLog = logger.New("workflow:missing_tool")

// buildCreateOutputMissingToolJob creates the missing_tool job
func (c *Compiler) buildCreateOutputMissingToolJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.MissingTool == nil {
		return nil, errors.New("safe-outputs.missing-tool configuration is required")
	}

	return c.buildIssueReportingJob(data, mainJobName, issueReportingJobParams{
		kind:         "missing_tool",
		envPrefix:    envVarPrefix("missing_tool"),
		defaultTitle: "[missing tool]",
		outputKey:    "tools_reported",
		stepName:     "Record Missing Tool",
		config:       data.SafeOutputs.MissingTool,
		log:          missingToolLog,
	})
}

// parseMissingToolConfig handles missing-tool configuration
func (c *Compiler) parseMissingToolConfig(outputMap map[string]any) *MissingToolConfig {
	return c.parseIssueReportingConfig(outputMap, "missing-tool", "[missing tool]", missingToolLog)
}
