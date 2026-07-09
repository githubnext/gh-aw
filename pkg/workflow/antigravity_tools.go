package workflow

// This file provides Antigravity engine tool configuration logic.
//
// It handles two key responsibilities:
//
//  1. Tool Core Mapping (computeAntigravityToolsCore):
//     Converts neutral tool names from the workflow configuration into
//     Antigravity CLI built-in tool names for the tools.core allowlist in
//     .antigravity/settings.json. This restricts the agent to only the tools
//     explicitly requested by the workflow.
//
//  2. Settings Step Generation (generateAntigravitySettingsStep):
//     Generates a GitHub Actions step that writes or merges .antigravity/settings.json
//     before the Antigravity CLI execution. This step always sets:
//     - context.includeDirectories: ["/tmp/"] so file tools can access /tmp/
//     - tools.core: derived from neutral tool configuration
//     The merge approach ensures MCP server config (written by convert_gateway_config_antigravity.sh)
//     is preserved while adding the context and tool settings.

import "github.com/github/gh-aw/pkg/logger"

var antigravityToolsLog = logger.New("workflow:antigravity_tools")

// computeAntigravityToolsCore maps neutral tool names to Antigravity CLI built-in tool names
// for use in the tools.core allowlist in .antigravity/settings.json.
//
// Neutral tool → Antigravity CLI tool mapping:
//   - bash: [cmd, ...]     → run_shell_command(cmd), ... (one entry per command)
//   - bash: * or bash: nil → run_shell_command           (allow all shell commands)
//   - edit: {}             → replace, write_file          (file write tools)
//
// Read-only file system tools are always included as they are essential for
// agentic workflows: glob, grep_search, list_directory, read_file, read_many_files.
//
// See: https://antigravity.google/docs/cli-overview
func computeAntigravityToolsCore(tools map[string]any) []string {
	return computeEngineToolsCore(tools, antigravityToolsLog, "antigravity")
}

// generateAntigravitySettingsStep creates a GitHub Actions step that writes the
// Antigravity CLI project settings file (.antigravity/settings.json) before execution.
//
// This step:
//  1. Sets context.includeDirectories to ["/tmp/"] so that Antigravity CLI file system
//     tools (write_file, replace) can access files in /tmp/ including
//     /tmp/gh-aw/cache-memory/ and other agent working directories.
//  2. Sets tools.core to the list of built-in tools derived from the workflow's
//     neutral tool configuration (bash → run_shell_command, edit → write_file/replace).
//  3. Merges the above settings with any existing .antigravity/settings.json, which
//     may have been written by convert_gateway_config_antigravity.sh with MCP server
//     configuration. The merge preserves the MCP server config while adding
//     the context and tools settings.
func (e *AntigravityEngine) generateAntigravitySettingsStep(workflowData *WorkflowData) GitHubActionStep {
	return generateEngineSettingsStep(
		workflowData,
		antigravityToolsLog,
		"Antigravity",
		".antigravity",
		"GH_AW_ANTIGRAVITY_BASE_CONFIG",
		"Write Antigravity Config",
	)
}
