package workflow

import (
	"fmt"
	"strings"
)

// InjectCustomEngineSteps processes custom steps from engine config and converts them to GitHubActionSteps.
// This shared function extracts the common pattern used by Copilot, Codex, and Claude engines.
//
// Parameters:
//   - workflowData: The workflow data containing engine configuration
//   - convertStepFunc: A function that converts a step map to YAML string (engine-specific)
//
// Returns:
//   - []GitHubActionStep: Array of custom steps ready to be included in the execution pipeline
func InjectCustomEngineSteps(
	workflowData *WorkflowData,
	convertStepFunc func(map[string]any) (string, error),
) []GitHubActionStep {
	var steps []GitHubActionStep

	// Handle custom steps if they exist in engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		for _, step := range workflowData.EngineConfig.Steps {
			stepYAML, err := convertStepFunc(step)
			if err != nil {
				// Log error but continue with other steps
				continue
			}
			steps = append(steps, GitHubActionStep{stepYAML})
		}
	}

	return steps
}

// RenderCustomMCPToolConfigHandler is a function type that engines must provide to render their specific MCP config
type RenderCustomMCPToolConfigHandler func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error

// HandleCustomMCPToolInSwitch processes custom MCP tools in the default case of a switch statement.
// This shared function extracts the common pattern used across all workflow engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - toolName: The name of the tool being processed
//   - tools: The tools map containing tool configurations (supports both expanded and non-expanded tools)
//   - isLast: Whether this is the last tool in the list
//   - renderFunc: Engine-specific function to render the MCP configuration
//
// Returns:
//   - bool: true if a custom MCP tool was handled, false otherwise
func HandleCustomMCPToolInSwitch(
	yaml *strings.Builder,
	toolName string,
	tools map[string]any,
	isLast bool,
	renderFunc RenderCustomMCPToolConfigHandler,
) bool {
	// Handle custom MCP tools (those with MCP-compatible type)
	if toolConfig, ok := tools[toolName].(map[string]any); ok {
		if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
			if err := renderFunc(yaml, toolName, toolConfig, isLast); err != nil {
				fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
			}
			return true
		}
	}
	return false
}
