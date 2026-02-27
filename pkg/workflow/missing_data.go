package workflow

import (
	"errors"

	"github.com/github/gh-aw/pkg/logger"
)

var missingDataLog = logger.New("workflow:missing_data")

// buildCreateOutputMissingDataJob creates the missing_data job
func (c *Compiler) buildCreateOutputMissingDataJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.MissingData == nil {
		return nil, errors.New("safe-outputs.missing-data configuration is required")
	}

	return c.buildIssueReportingJob(data, mainJobName, issueReportingJobParams{
		kind:         "missing_data",
		envPrefix:    envVarPrefix("missing_data"),
		defaultTitle: "[missing data]",
		outputKey:    "data_reported",
		stepName:     "Record Missing Data",
		config:       data.SafeOutputs.MissingData,
		log:          missingDataLog,
	})
}

// parseMissingDataConfig handles missing-data configuration
func (c *Compiler) parseMissingDataConfig(outputMap map[string]any) *MissingDataConfig {
	return c.parseIssueReportingConfig(outputMap, "missing-data", "[missing data]", missingDataLog)
}
