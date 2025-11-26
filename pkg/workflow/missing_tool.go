package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var missingToolLog = logger.New("workflow:missing_tool")

// MissingToolConfig holds configuration for reporting missing tools or functionality
type MissingToolConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// NOTE: buildCreateOutputMissingToolJob has been removed.
// Missing tool processing is now integrated into the conclusion job (buildConclusionJob in notify_comment.go)
// to reduce the number of workflow jobs and simplify the workflow structure.

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
