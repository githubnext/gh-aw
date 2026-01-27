package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// ========================================
// Safe Output Configuration Helpers
// ========================================

// formatSafeOutputsRunsOn formats the runs-on value from SafeOutputsConfig for job output
func (c *Compiler) formatSafeOutputsRunsOn(safeOutputs *SafeOutputsConfig) string {
	if safeOutputs == nil || safeOutputs.RunsOn == "" {
		return fmt.Sprintf("runs-on: %s", constants.DefaultActivationJobRunnerImage)
	}

	return fmt.Sprintf("runs-on: %s", safeOutputs.RunsOn)
}

// HasSafeOutputsEnabled checks if any safe-outputs are enabled
func HasSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	enabled := hasAnySafeOutputEnabled(safeOutputs)

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Safe outputs enabled check: %v", enabled)
	}

	return enabled
}

// GetEnabledSafeOutputToolNames returns a list of enabled safe output tool names
// that can be used in the prompt to inform the agent which tools are available
func GetEnabledSafeOutputToolNames(safeOutputs *SafeOutputsConfig) []string {
	tools := getEnabledSafeOutputToolNamesReflection(safeOutputs)

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Enabled safe output tools: %v", tools)
	}

	return tools
}

// usesAllowAllTools checks if the workflow will use --allow-all-tools flag
// This happens when bash tool has wildcard (*) access
func usesAllowAllTools(tools map[string]any) bool {
	if bashConfig, hasBash := tools["bash"]; hasBash {
		if bashCommands, ok := bashConfig.([]any); ok {
			for _, cmd := range bashCommands {
				if cmdStr, ok := cmd.(string); ok {
					if cmdStr == ":*" || cmdStr == "*" {
						return true
					}
				}
			}
		}
	}
	return false
}

// GetEnabledSafeOutputToolNamesWithPrefix returns a list of enabled safe output tool names
// with optional MCP server prefix. For Copilot engine using MCP gateway with --allow-all-tools,
// tools need to be prefixed with their MCP server name (safeoutputs___create_issue).
func GetEnabledSafeOutputToolNamesWithPrefix(safeOutputs *SafeOutputsConfig, engineConfig *EngineConfig, tools map[string]any) []string {
	toolNames := getEnabledSafeOutputToolNamesReflection(safeOutputs)

	// Add MCP server prefix for Copilot engine when using --allow-all-tools
	// This is needed because the MCP gateway registers tools with prefixes (safeoutputs___*)
	// but the Copilot agent needs to know the full prefixed name when calling tools
	if engineConfig != nil && engineConfig.ID == "copilot" && usesAllowAllTools(tools) {
		prefixedTools := make([]string, len(toolNames))
		for i, tool := range toolNames {
			prefixedTools[i] = constants.SafeOutputsMCPServerID + "___" + tool
		}
		if safeOutputsConfigLog.Enabled() {
			safeOutputsConfigLog.Printf("Added MCP prefix for Copilot with --allow-all-tools: %v", prefixedTools)
		}
		return prefixedTools
	}

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Enabled safe output tools (no prefix): %v", toolNames)
	}

	return toolNames
}
