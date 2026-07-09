package workflow

// This file provides Gemini engine tool configuration logic.
//
// It handles two key responsibilities:
//
//  1. Tool Core Mapping (computeGeminiToolsCore):
//     Converts neutral tool names from the workflow configuration into
//     Gemini CLI built-in tool names for the tools.core allowlist in
//     .gemini/settings.json. This restricts the agent to only the tools
//     explicitly requested by the workflow.
//
//  2. Settings Step Generation (generateGeminiSettingsStep):
//     Generates a GitHub Actions step that writes or merges .gemini/settings.json
//     before the Gemini CLI execution. This step always sets:
//     - context.includeDirectories: ["/tmp/"] so file tools can access /tmp/
//     - tools.core: derived from neutral tool configuration
//     The merge approach ensures MCP server config (written by convert_gateway_config_gemini.sh)
//     is preserved while adding the context and tool settings.

import "github.com/github/gh-aw/pkg/logger"

var geminiToolsLog = logger.New("workflow:gemini_tools")

// computeGeminiToolsCore maps neutral tool names to Gemini CLI built-in tool names
// for use in the tools.core allowlist in .gemini/settings.json.
//
// Neutral tool → Gemini CLI tool mapping:
//   - bash: [cmd, ...]     → run_shell_command(cmd), ... (one entry per command)
//   - bash: * or bash: nil → run_shell_command           (allow all shell commands)
//   - edit: {}             → replace, write_file          (file write tools)
//
// Read-only file system tools are always included as they are essential for
// agentic workflows: glob, grep_search, list_directory, read_file, read_many_files.
//
// See: https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/file-system.md
// See: https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/shell.md
func computeGeminiToolsCore(tools map[string]any) []string {
	return computeEngineToolsCore(tools, geminiToolsLog, "gemini")
}

// generateGeminiSettingsStep creates a GitHub Actions step that writes the
// Gemini CLI project settings file (.gemini/settings.json) before execution.
//
// This step:
//  1. Sets context.includeDirectories to ["/tmp/"] so that Gemini CLI file system
//     tools (write_file, replace) can access files in /tmp/ including
//     /tmp/gh-aw/cache-memory/ and other agent working directories.
//  2. Sets tools.core to the list of built-in tools derived from the workflow's
//     neutral tool configuration (bash → run_shell_command, edit → write_file/replace).
//  3. Merges the above settings with any existing .gemini/settings.json, which
//     may have been written by convert_gateway_config_gemini.sh with MCP server
//     configuration. The merge preserves the MCP server config while adding
//     the context and tools settings.
func (e *GeminiEngine) generateGeminiSettingsStep(workflowData *WorkflowData) GitHubActionStep {
	return generateEngineSettingsStep(
		workflowData,
		geminiToolsLog,
		"Gemini",
		".gemini",
		"GH_AW_GEMINI_BASE_CONFIG",
		"Write Gemini Config",
	)
}
