// Package workflow provides Copilot engine tool permission and error pattern logic.
//
// This file handles three key responsibilities:
//
//  1. Tool Permission Arguments (computeCopilotToolArguments):
//     Converts workflow tool configurations into --allow-tool flags for Copilot CLI.
//     Handles bash/shell tools, edit tools, safe outputs, safe inputs, and MCP servers.
//     Supports granular permissions (e.g., "github(get_file)") and server-level wildcards.
//
//  2. Tool Argument Comments (generateCopilotToolArgumentsComment):
//     Generates human-readable comments documenting which tool permissions are granted.
//     Used in compiled workflows for transparency and debugging.
//
//  3. Error Patterns (GetErrorPatterns):
//     Defines regex patterns for extracting error messages from Copilot CLI logs.
//     Includes timestamped log formats, command failures, module errors, and permission issues.
//     Used by log parsers to detect and categorize errors.
//
// These functions are grouped together because they all relate to tool configuration
// and error handling in the Copilot engine.
package workflow

import (
	"strings"
)

// computeCopilotToolArguments computes the --allow-tool arguments for Copilot CLI based on tool configurations.
// Returns a list of arguments ready to be passed to the Copilot CLI.
//
// NOTE: Bash and edit tools are deprecated. Yolo mode (--allow-all-tools) is now always enabled
// since agents run in containerized environments, making all tool restrictions unnecessary.
func (e *CopilotEngine) computeCopilotToolArguments(tools map[string]any, safeOutputs *SafeOutputsConfig, safeInputs *SafeInputsConfig, workflowData *WorkflowData) []string {
	// YOLO MODE: Always enable all tools since agents run in containerized environments.
	// Bash and edit tool configurations are deprecated and ignored.
	// --allow-all-tools covers all built-in tools (shell, write, web_fetch, etc.) and
	// all MCP servers (github, safe_outputs, safe_inputs, custom servers, etc.).
	return []string{"--allow-all-tools"}
}

// generateCopilotToolArgumentsComment generates a multi-line comment showing each tool argument.
// This is used to document which tool permissions are being granted in the compiled workflow.
func (e *CopilotEngine) generateCopilotToolArgumentsComment(tools map[string]any, safeOutputs *SafeOutputsConfig, safeInputs *SafeInputsConfig, workflowData *WorkflowData, indent string) string {
	toolArgs := e.computeCopilotToolArguments(tools, safeOutputs, safeInputs, workflowData)
	if len(toolArgs) == 0 {
		return ""
	}

	var comment strings.Builder
	comment.WriteString(indent + "# Copilot CLI tool arguments:\n")

	// Handle both single flags (--allow-all-tools) and flag-value pairs (--allow-tool value)
	for i := 0; i < len(toolArgs); i++ {
		if toolArgs[i] == "--allow-all-tools" {
			comment.WriteString(indent + "# --allow-all-tools (yolo mode: all tools enabled)\n")
		} else if toolArgs[i] == "--allow-tool" && i+1 < len(toolArgs) {
			comment.WriteString(indent + "# --allow-tool " + toolArgs[i+1] + "\n")
			i++ // Skip the next arg since we processed it
		} else {
			comment.WriteString(indent + "# " + toolArgs[i] + "\n")
		}
	}

	return comment.String()
}
