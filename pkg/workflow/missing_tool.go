package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var missingToolLog = logger.New("workflow:missing_tool")

// MissingToolConfig holds configuration for reporting missing tools or functionality
type MissingToolConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// buildCreateOutputMissingToolJob creates the missing_tool job
func (c *Compiler) buildCreateOutputMissingToolJob(data *WorkflowData, mainJobName string) (*Job, error) {
	missingToolLog.Printf("Building missing_tool job for workflow: %s", data.Name)

	if data.SafeOutputs == nil || data.SafeOutputs.MissingTool == nil {
		return nil, fmt.Errorf("safe-outputs.missing-tool configuration is required")
	}

	// Build custom environment variables specific to missing-tool
	var customEnvVars []string
	if data.SafeOutputs.MissingTool.Max > 0 {
		missingToolLog.Printf("Setting max missing tools limit: %d", data.SafeOutputs.MissingTool.Max)
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MISSING_TOOL_MAX: %d\n", data.SafeOutputs.MissingTool.Max))
	}

	// Add workflow metadata for consistency
	customEnvVars = append(customEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID)...)

	// Create outputs for the job
	outputs := map[string]string{
		"tools_reported": "${{ steps.missing_tool.outputs.tools_reported }}",
		"total_count":    "${{ steps.missing_tool.outputs.total_count }}",
	}

	// Build the job condition using BuildSafeOutputType
	jobCondition := BuildSafeOutputType("missing_tool")

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:       "missing_tool",
		StepName:      "Record Missing Tool",
		StepID:        "missing_tool",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        "const { main } = require('/tmp/gh-aw/actions/missing_tool.cjs'); await main();",
		Permissions:   NewPermissionsContentsRead(),
		Outputs:       outputs,
		Condition:     jobCondition,
		Token:         data.SafeOutputs.MissingTool.GitHubToken,
	})
}

// parseMissingToolConfig handles missing-tool configuration
func (c *Compiler) parseMissingToolConfig(outputMap map[string]any) *MissingToolConfig {
	if configData, exists := outputMap["missing-tool"]; exists {
		// Handle the case where configData is false (explicitly disabled)
		if configBool, ok := configData.(bool); ok && !configBool {
			missingToolLog.Print("Missing-tool configuration explicitly disabled")
			return nil
		}

		missingToolConfig := &MissingToolConfig{} // Default: no max limit

		// Handle the case where configData is nil (missing-tool: with no value)
		if configData == nil {
			missingToolLog.Print("Missing-tool configuration enabled with defaults")
			return missingToolConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			missingToolLog.Print("Parsing missing-tool configuration from map")
			// Parse common base fields with default max of 0 (no limit)
			c.parseBaseSafeOutputConfig(configMap, &missingToolConfig.BaseSafeOutputConfig, 0)
		}

		return missingToolConfig
	}

	return nil
}
