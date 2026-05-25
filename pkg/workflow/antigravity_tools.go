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

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
)

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
	// Always include essential read-only file system tools
	toolsCore := []string{
		"glob",
		"grep_search",
		"list_directory",
		"read_file",
		"read_many_files",
	}

	if tools == nil {
		return toolsCore
	}

	// Map bash neutral tool to run_shell_command
	if bashConfig, hasBash := tools["bash"]; hasBash {
		bashCommands, ok := bashConfig.([]any)
		if !ok || len(bashCommands) == 0 {
			// bash with no specific commands - allow all shell commands
			antigravityToolsLog.Print("bash (no specific commands) → run_shell_command")
			toolsCore = append(toolsCore, "run_shell_command")
		} else {
			// Check for wildcard (* or :*)
			hasWildcard := false
			for _, cmd := range bashCommands {
				if cmdStr, ok := cmd.(string); ok && (cmdStr == "*" || cmdStr == ":*") {
					hasWildcard = true
					break
				}
			}
			if hasWildcard {
				antigravityToolsLog.Print("bash wildcard → run_shell_command")
				toolsCore = append(toolsCore, "run_shell_command")
			} else {
				// Add an entry for each specific command: run_shell_command(cmd)
				for _, cmd := range bashCommands {
					if cmdStr, ok := cmd.(string); ok {
						// Normalize trailing " *" wildcard (e.g. "jq *" → "jq") so that
						// all engines emit the canonical prefix form (run_shell_command(jq))
						// regardless of whether the command was written with or without the wildcard.
						normalized, _ := normalizeBashCommand(cmdStr)
						entry := fmt.Sprintf("run_shell_command(%s)", normalized)
						antigravityToolsLog.Printf("bash %q → %s", cmdStr, entry)
						toolsCore = append(toolsCore, entry)
					}
				}
			}
		}
	}

	// Map edit neutral tool to write_file and replace (Antigravity's file write tools)
	if _, hasEdit := tools["edit"]; hasEdit {
		antigravityToolsLog.Print("edit → replace, write_file")
		toolsCore = append(toolsCore, "replace")
		toolsCore = append(toolsCore, "write_file")
	}

	// Map web-fetch neutral tool to web_fetch (Antigravity's native HTTP fetch tool)
	// See: https://antigravity.google/docs/cli-overview
	if _, hasWebFetch := tools["web-fetch"]; hasWebFetch {
		antigravityToolsLog.Print("web-fetch → web_fetch")
		toolsCore = append(toolsCore, "web_fetch")
	}

	sort.Strings(toolsCore)
	return toolsCore
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
	antigravityToolsLog.Printf("Generating Antigravity settings step for: %s", workflowData.Name)

	tools := workflowData.Tools
	if tools == nil {
		tools = make(map[string]any)
	}
	workflowDataWithEffectiveTools := *workflowData
	workflowDataWithEffectiveTools.Tools = tools
	tools = withMountedCLIShellCommandsInRestrictedBash(&workflowDataWithEffectiveTools)

	// Compute tools.core from neutral tool configuration
	toolsCore := computeAntigravityToolsCore(tools)
	antigravityToolsLog.Printf("tools.core entries: %d", len(toolsCore))

	// Build the settings JSON object
	config := map[string]any{
		"context": map[string]any{
			"includeDirectories": []string{"/tmp/"},
		},
		"tools": map[string]any{
			"core": toolsCore,
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		antigravityToolsLog.Printf("ERROR: Failed to marshal Antigravity settings: %v", err)
		configJSON = []byte(`{"context":{"includeDirectories":["/tmp/"]},"tools":{"core":[]}}`)
	}

	// Generate a shell script that:
	// - Creates the .antigravity directory if needed
	// - Merges settings into an existing settings.json (from MCP gateway setup), or
	// - Creates a new settings.json when no MCP servers are configured
	//
	// The JSON config is passed via the GH_AW_ANTIGRAVITY_BASE_CONFIG environment variable
	// to avoid any shell quoting issues with special characters in the JSON.
	//
	// jq merge: '$existing * $base' means the RIGHT operand ($base) overrides the LEFT
	// operand ($existing) for conflicting keys. Non-conflicting keys from $existing
	// (e.g. mcpServers written by convert_gateway_config_antigravity.sh) are preserved.
	command := `mkdir -p "$GITHUB_WORKSPACE/.antigravity"
SETTINGS="$GITHUB_WORKSPACE/.antigravity/settings.json"
BASE_CONFIG="$GH_AW_ANTIGRAVITY_BASE_CONFIG"
if [ -f "$SETTINGS" ]; then
  MERGED=$(jq -n --argjson base "$BASE_CONFIG" --argjson existing "$(cat "$SETTINGS")" '$existing * $base')
  echo "$MERGED" > "$SETTINGS"
else
  echo "$BASE_CONFIG" > "$SETTINGS"
fi`

	stepLines := []string{
		"      - name: Write Antigravity Config",
	}
	env := map[string]string{
		"GH_AW_ANTIGRAVITY_BASE_CONFIG": string(configJSON),
	}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, env)
	return GitHubActionStep(stepLines)
}
