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

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
)

var geminiToolsLog = logger.New("workflow:gemini_tools")

// computeGeminiToolsCore maps neutral tool names to Gemini CLI built-in tool names
// for use in the tools.core allowlist in .gemini/settings.json.
func computeGeminiToolsCore(tools map[string]any) []string {
	toolsCore := defaultGeminiToolsCore()
	if tools == nil {
		return toolsCore
	}
	toolsCore = append(toolsCore, geminiBashToolsCore(tools)...)
	if _, hasEdit := tools["edit"]; hasEdit {
		geminiToolsLog.Print("edit → replace, write_file")
		toolsCore = append(toolsCore, "replace", "write_file")
	}
	if _, hasWebFetch := tools["web-fetch"]; hasWebFetch {
		geminiToolsLog.Print("web-fetch → web_fetch")
		toolsCore = append(toolsCore, "web_fetch")
	}
	sort.Strings(toolsCore)
	return toolsCore
}

func defaultGeminiToolsCore() []string {
	return []string{"glob", "grep_search", "list_directory", "read_file", "read_many_files"}
}

func geminiBashToolsCore(tools map[string]any) []string {
	bashConfig, hasBash := tools["bash"]
	if !hasBash {
		return nil
	}
	bashCommands, ok := bashConfig.([]any)
	if !ok || len(bashCommands) == 0 {
		geminiToolsLog.Print("bash (no specific commands) → run_shell_command")
		return []string{"run_shell_command"}
	}
	if geminiBashHasWildcard(bashCommands) {
		geminiToolsLog.Print("bash wildcard → run_shell_command")
		return []string{"run_shell_command"}
	}
	return geminiSpecificBashTools(bashCommands)
}

func geminiBashHasWildcard(bashCommands []any) bool {
	for _, cmd := range bashCommands {
		if cmdStr, ok := cmd.(string); ok && (cmdStr == "*" || cmdStr == ":*") {
			return true
		}
	}
	return false
}

func geminiSpecificBashTools(bashCommands []any) []string {
	var toolsCore []string
	for _, cmd := range bashCommands {
		cmdStr, ok := cmd.(string)
		if !ok {
			continue
		}
		normalized, _ := normalizeBashCommand(cmdStr)
		entry := fmt.Sprintf("run_shell_command(%s)", normalized)
		geminiToolsLog.Printf("bash %q → %s", cmdStr, entry)
		toolsCore = append(toolsCore, entry)
	}
	return toolsCore
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
	geminiToolsLog.Printf("Generating Gemini settings step for: %s", workflowData.Name)

	tools := workflowData.Tools
	if tools == nil {
		tools = make(map[string]any)
	}
	workflowDataWithEffectiveTools := *workflowData
	workflowDataWithEffectiveTools.Tools = tools
	tools = withMountedCLIShellCommandsInRestrictedBash(&workflowDataWithEffectiveTools)

	toolsCore := computeGeminiToolsCore(tools)
	geminiToolsLog.Printf("tools.core entries: %d", len(toolsCore))

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
		geminiToolsLog.Printf("ERROR: Failed to marshal Gemini settings: %v", err)
		configJSON = []byte(`{"context":{"includeDirectories":["/tmp/"]},"tools":{"core":[]}}`)
	}

	// Generate a shell script that:
	// - Creates the .gemini directory if needed
	// - Merges settings into an existing settings.json (from MCP gateway setup), or
	// - Creates a new settings.json when no MCP servers are configured
	//
	// The JSON config is passed via the GH_AW_GEMINI_BASE_CONFIG environment variable
	// to avoid any shell quoting issues with special characters in the JSON.
	//
	// jq merge: '$existing * $base' means the RIGHT operand ($base) overrides the LEFT
	// operand ($existing) for conflicting keys. Non-conflicting keys from $existing
	// (e.g. mcpServers written by convert_gateway_config_gemini.sh) are preserved.
	command := `mkdir -p "$GITHUB_WORKSPACE/.gemini"
SETTINGS="$GITHUB_WORKSPACE/.gemini/settings.json"
BASE_CONFIG="$GH_AW_GEMINI_BASE_CONFIG"
if [ -f "$SETTINGS" ]; then
  MERGED=$(jq -n --argjson base "$BASE_CONFIG" --argjson existing "$(cat "$SETTINGS")" '$existing * $base')
  echo "$MERGED" > "$SETTINGS"
else
  echo "$BASE_CONFIG" > "$SETTINGS"
fi`

	stepLines := []string{
		"      - name: Write Gemini Config",
	}
	env := map[string]string{
		"GH_AW_GEMINI_BASE_CONFIG": string(configJSON),
	}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, env)
	return GitHubActionStep(stepLines)
}
