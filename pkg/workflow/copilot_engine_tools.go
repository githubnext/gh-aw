// This file provides Copilot engine tool permission and error pattern logic.
//
// This file handles three key responsibilities:
//
//  1. Tool Permission Arguments (computeCopilotToolArguments):
//     Converts workflow tool configurations into --allow-tool flags for Copilot CLI.
//     Handles bash/shell tools, edit tools, safe outputs, mcp-scripts, and MCP servers.
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
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
)

var copilotEngineToolsLog = logger.New("workflow:copilot_engine_tools")

var copilotBuiltInTools = map[string]struct{}{
	"bash":       {},
	"edit":       {},
	"web-search": {},
	"playwright": {},
}

// sanitizeCopilotShellCommand truncates a bash tool command at the first single
// quote to produce a safe prefix for the Copilot CLI --allow-tool shell() argument.
//
// Copilot CLI uses prefix matching for shell() arguments, so shell(jq) matches any
// jq invocation including "jq '.filter' ...". Single quotes in the allow-tool argument
// cause the Copilot CLI to crash at startup because of quoting conflicts in the
// multi-level shell escaping required by the AWF entrypoint.
//
// Returns the sanitized command and whether sanitization was needed.
func sanitizeCopilotShellCommand(cmdStr string) (string, bool) {
	prefix, _, found := strings.Cut(cmdStr, "'")
	if !found {
		return cmdStr, false
	}
	// Trim trailing whitespace from the prefix.
	// shell(jq) prefix-matches any jq invocation, preserving full tool access.
	return strings.TrimRight(prefix, " "), true
}

// computeCopilotToolArguments computes the --allow-tool arguments for Copilot CLI based on tool configurations.
// It handles bash/shell tools, edit tools, safe outputs, mcp-scripts, and MCP server tools.
// Returns a sorted list of arguments ready to be passed to the Copilot CLI.
func (e *CopilotEngine) computeCopilotToolArguments(tools map[string]any, safeOutputs *SafeOutputsConfig, mcpScripts *MCPScriptsConfig, workflowData *WorkflowData) []string {
	copilotEngineToolsLog.Printf("Computing tool arguments: tools=%d", len(tools))
	tools = ensureToolMap(tools)
	if hasCopilotBashWildcard(tools) {
		copilotEngineToolsLog.Print("Bash wildcard detected, using --allow-all-tools")
		return []string{"--allow-all-tools"}
	}

	args, hasRestrictedBashAllowlist := collectCopilotBashToolArguments(tools)
	args = e.addRestrictedBashMountArguments(args, hasRestrictedBashAllowlist, tools, safeOutputs, mcpScripts, workflowData)
	args = appendCopilotBuiltinToolArguments(args, tools, safeOutputs, mcpScripts)
	args = appendCopilotMCPServerArguments(args, tools)
	args = sortAndDeduplicateAllowToolArgs(args)

	copilotEngineToolsLog.Printf("Computed %d tool arguments", len(args)/2)
	return args
}

func ensureToolMap(tools map[string]any) map[string]any {
	if tools != nil {
		return tools
	}
	return make(map[string]any)
}

func hasCopilotBashWildcard(tools map[string]any) bool {
	bashCommands, ok := tools["bash"].([]any)
	if !ok {
		return false
	}
	for _, cmd := range bashCommands {
		cmdStr, ok := cmd.(string)
		if ok && (cmdStr == ":*" || cmdStr == "*") {
			return true
		}
	}
	return false
}

func collectCopilotBashToolArguments(tools map[string]any) ([]string, bool) {
	bashConfig, hasBash := tools["bash"]
	if !hasBash {
		return nil, false
	}
	bashCommands, ok := bashConfig.([]any)
	if !ok {
		return []string{"--allow-tool", "shell"}, false
	}

	args := make([]string, 0, len(bashCommands)*2)
	for _, cmd := range bashCommands {
		cmdStr, ok := cmd.(string)
		if !ok {
			continue
		}
		args = append(args, "--allow-tool", formatCopilotShellAllowTool(cmdStr))
	}
	return args, true
}

func formatCopilotShellAllowTool(cmdStr string) string {
	cmdStr, _ = normalizeBashCommand(cmdStr)
	if !strings.Contains(cmdStr, ":") && !strings.Contains(cmdStr, " ") && constants.CopilotStemCommands[cmdStr] {
		return fmt.Sprintf("shell(%s:*)", cmdStr)
	}

	sanitized, wasSanitized := sanitizeCopilotShellCommand(cmdStr)
	if wasSanitized {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			fmt.Sprintf("bash tool %q contains single quotes that crash Copilot CLI; truncated to safe prefix %q for shell() prefix-matching. Use %q in your workflow to silence this warning.",
				cmdStr, sanitized, sanitized)))
	}
	return fmt.Sprintf("shell(%s)", sanitized)
}

func (e *CopilotEngine) addRestrictedBashMountArguments(args []string, hasRestrictedBashAllowlist bool, tools map[string]any, safeOutputs *SafeOutputsConfig, mcpScripts *MCPScriptsConfig, workflowData *WorkflowData) []string {
	if !hasRestrictedBashAllowlist {
		return args
	}

	effectiveWorkflowData := buildCLIWorkflowDataForMounts(workflowData, tools, safeOutputs, mcpScripts)
	for _, serverName := range getMountedCLIServerNamesIfBashRestricted(effectiveWorkflowData, tools, safeOutputs, mcpScripts) {
		args = append(args, "--allow-tool", fmt.Sprintf("shell(%s:*)", serverName))
	}
	if workflowData != nil && isPlaywrightCLIMode(workflowData.Tools) {
		args = append(args, "--allow-tool", "shell(playwright-cli:*)")
	}
	if isGitHubCLIModeEnabled(effectiveWorkflowData) {
		args = append(args, "--allow-tool", "shell(gh:*)")
	}
	return args
}

func appendCopilotBuiltinToolArguments(args []string, tools map[string]any, safeOutputs *SafeOutputsConfig, mcpScripts *MCPScriptsConfig) []string {
	if _, hasEdit := tools["edit"]; hasEdit {
		copilotEngineToolsLog.Print("Edit tool enabled, adding write permission")
		args = append(args, "--allow-tool", "write")
	}
	if HasSafeOutputsEnabled(safeOutputs) {
		copilotEngineToolsLog.Print("Safe-outputs enabled, adding MCP server permission")
		args = append(args, "--allow-tool", constants.SafeOutputsMCPServerID.String())
	}
	if IsMCPScriptsEnabled(mcpScripts) {
		args = append(args, "--allow-tool", constants.MCPScriptsMCPServerID.String())
	}
	if _, hasWebFetch := tools["web-fetch"]; hasWebFetch {
		copilotEngineToolsLog.Print("Web-fetch tool enabled, adding web_fetch permission")
		args = append(args, "--allow-tool", "web_fetch")
	}
	return args
}

func appendCopilotMCPServerArguments(args []string, tools map[string]any) []string {
	for toolName, toolConfig := range tools {
		if setutil.Contains(copilotBuiltInTools, toolName) {
			continue
		}
		if toolName == "github" {
			args = appendGitHubToolArguments(args, toolConfig)
			continue
		}
		args = appendCustomMCPToolArguments(args, toolName, toolConfig)
	}
	return args
}

func appendGitHubToolArguments(args []string, toolConfig any) []string {
	toolConfigMap, ok := toolConfig.(map[string]any)
	if !ok {
		return append(args, "--allow-tool", "github")
	}
	allowed, hasAllowed := toolConfigMap["allowed"]
	if !hasAllowed {
		return append(args, "--allow-tool", "github")
	}
	allowedList, ok := allowed.([]any)
	if !ok {
		return args
	}

	hasWildcard := false
	for _, allowedTool := range allowedList {
		toolStr, ok := allowedTool.(string)
		if !ok {
			continue
		}
		if toolStr == "*" {
			hasWildcard = true
			continue
		}
		args = append(args, "--allow-tool", fmt.Sprintf("github(%s)", toolStr))
	}
	if hasWildcard {
		args = append(args, "--allow-tool", "github")
	}
	return args
}

func appendCustomMCPToolArguments(args []string, toolName string, toolConfig any) []string {
	toolConfigMap, ok := toolConfig.(map[string]any)
	if !ok {
		return args
	}
	hasMcp, _ := hasMCPConfig(toolConfigMap)
	if !hasMcp {
		return args
	}

	copilotEngineToolsLog.Printf("Adding custom MCP server permission: %s", toolName)
	args = append(args, "--allow-tool", toolName)
	allowedList, ok := toolConfigMap["allowed"].([]any)
	if !ok {
		return args
	}
	for _, allowedTool := range allowedList {
		toolStr, ok := allowedTool.(string)
		if ok {
			args = append(args, "--allow-tool", fmt.Sprintf("%s(%s)", toolName, toolStr))
		}
	}
	return args
}

func sortAndDeduplicateAllowToolArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	values := make([]string, 0, len(args)/2)
	for i := 1; i < len(args); i += 2 {
		values = append(values, args[i])
	}
	sort.Strings(values)

	newArgs := make([]string, 0, len(args))
	prev := ""
	for _, value := range values {
		if value == prev {
			continue
		}
		newArgs = append(newArgs, "--allow-tool", value)
		prev = value
	}
	return newArgs
}

// generateCopilotToolArgumentsComment generates a multi-line comment showing each tool argument.
// This is used to document which tool permissions are being granted in the compiled workflow.
func (e *CopilotEngine) generateCopilotToolArgumentsComment(tools map[string]any, safeOutputs *SafeOutputsConfig, mcpScripts *MCPScriptsConfig, workflowData *WorkflowData, indent string) string {
	toolArgs := e.computeCopilotToolArguments(tools, safeOutputs, mcpScripts, workflowData)
	if len(toolArgs) == 0 {
		return ""
	}

	var comment strings.Builder
	comment.WriteString(indent + "# Copilot CLI tool arguments (sorted):\n")

	// Group flag-value pairs for better readability
	for i := 0; i < len(toolArgs); i += 2 {
		if i+1 < len(toolArgs) {
			fmt.Fprintf(&comment, "%s# %s %s\n", indent, toolArgs[i], toolArgs[i+1])
		}
	}

	return comment.String()
}
