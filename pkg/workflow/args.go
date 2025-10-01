package workflow

import (
	"encoding/json"
	"strings"
)

// extractCustomArgs extracts custom args from tool configuration
// Handles both []any and []string formats
func extractCustomArgs(toolConfig map[string]any) []string {
	if argsValue, exists := toolConfig["args"]; exists {
		// Handle []any format
		if argsSlice, ok := argsValue.([]any); ok {
			customArgs := make([]string, 0, len(argsSlice))
			for _, arg := range argsSlice {
				if argStr, ok := arg.(string); ok {
					customArgs = append(customArgs, argStr)
				}
			}
			return customArgs
		}
		// Handle []string format
		if argsSlice, ok := argsValue.([]string); ok {
			return argsSlice
		}
	}
	return nil
}

// getGitHubCustomArgs extracts custom args from GitHub tool configuration
func getGitHubCustomArgs(githubTool any) []string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		return extractCustomArgs(toolConfig)
	}
	return nil
}

// getPlaywrightCustomArgs extracts custom args from Playwright tool configuration
func getPlaywrightCustomArgs(playwrightTool any) []string {
	if toolConfig, ok := playwrightTool.(map[string]any); ok {
		return extractCustomArgs(toolConfig)
	}
	return nil
}

// writeArgsToYAML writes custom args to YAML with proper JSON quoting and escaping
// indent specifies the indentation string for each argument line
func writeArgsToYAML(yaml *strings.Builder, args []string, indent string) {
	for _, arg := range args {
		yaml.WriteString(",\n")
		// Use json.Marshal to properly quote and escape the argument
		quotedArg, _ := json.Marshal(arg)
		yaml.WriteString(indent + string(quotedArg))
	}
}

// writeArgsToYAMLInline writes custom args to YAML inline with proper JSON quoting and escaping
// Used when args are written on the same line with comma-space separators
func writeArgsToYAMLInline(yaml *strings.Builder, args []string) {
	for _, arg := range args {
		// Use json.Marshal to properly quote and escape the argument
		quotedArg, _ := json.Marshal(arg)
		yaml.WriteString(", " + string(quotedArg))
	}
}
