package workflow

import (
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// getSafeInputsEnvVars returns the list of environment variables needed for safe-inputs
func getSafeInputsEnvVars(safeInputs *SafeInputsConfig) []string {
	envVars := []string{}
	seen := make(map[string]bool)

	if safeInputs == nil {
		return envVars
	}

	for _, toolConfig := range safeInputs.Tools {
		for envName := range toolConfig.Env {
			if !seen[envName] {
				envVars = append(envVars, envName)
				seen[envName] = true
			}
		}
	}

	sort.Strings(envVars)
	return envVars
}

// collectSafeInputsSecrets collects all secrets from safe-inputs configuration
func collectSafeInputsSecrets(safeInputs *SafeInputsConfig) map[string]string {
	secrets := make(map[string]string)

	if safeInputs == nil {
		return secrets
	}

	// Sort tool names for consistent behavior when same env var appears in multiple tools
	toolNames := make([]string, 0, len(safeInputs.Tools))
	for toolName := range safeInputs.Tools {
		toolNames = append(toolNames, toolName)
	}
	sort.Strings(toolNames)

	for _, toolName := range toolNames {
		toolConfig := safeInputs.Tools[toolName]
		// Sort env var names for consistent order within each tool
		envNames := make([]string, 0, len(toolConfig.Env))
		for envName := range toolConfig.Env {
			envNames = append(envNames, envName)
		}
		sort.Strings(envNames)

		for _, envName := range envNames {
			secrets[envName] = toolConfig.Env[envName]
		}
	}

	return secrets
}

// renderSafeInputsMCPConfigWithOptions generates the Safe Inputs MCP server configuration with engine-specific options
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypoint, entrypointArgs, and mounts fields.
func renderSafeInputsMCPConfigWithOptions(yaml *strings.Builder, safeInputs *SafeInputsConfig, isLast bool, includeCopilotFields bool, workflowData *WorkflowData) {
	// Collect environment variables needed by safe-inputs tools
	envVars := getSafeInputsEnvVars(safeInputs)
	
	// Add standard environment variables for safe-inputs
	standardEnvVars := []string{
		"GH_AW_MCP_LOG_DIR",
	}
	
	// Combine and deduplicate env vars
	allEnvVars := append(standardEnvVars, envVars...)
	
	// Use MCP Gateway spec format with container, entrypoint, entrypointArgs, and mounts
	// This will be transformed to Docker command by getMCPConfig transformation logic
	yaml.WriteString("              \"" + constants.SafeInputsMCPServerID + "\": {\n")

	// Add type field for Copilot (per MCP Gateway Specification v1.0.0, use "stdio" for containerized servers)
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"stdio\",\n")
	}

	// MCP Gateway spec fields for containerized stdio servers
	yaml.WriteString("                \"container\": \"" + constants.DefaultNodeAlpineLTSImage + "\",\n")
	yaml.WriteString("                \"entrypoint\": \"node\",\n")
	yaml.WriteString("                \"entrypointArgs\": [\"/opt/gh-aw/safe-inputs/mcp-server.cjs\"],\n")
	yaml.WriteString("                \"mounts\": [\"/opt/gh-aw:/opt/gh-aw:ro\", \"/tmp/gh-aw:/tmp/gh-aw:rw\"],\n")

	// Note: tools field is NOT included here - the converter script adds it back
	// for Copilot. This keeps the gateway config compatible with the schema.

	// Write environment variables
	yaml.WriteString("                \"env\": {\n")
	for i, envVar := range allEnvVars {
		isLastEnvVar := i == len(allEnvVars)-1
		comma := ""
		if !isLastEnvVar {
			comma = ","
		}

		if includeCopilotFields {
			// Copilot format: backslash-escaped shell variable reference
			yaml.WriteString("                  \"" + envVar + "\": \"\\${" + envVar + "}\"" + comma + "\n")
		} else {
			// Claude/Custom format: direct shell variable reference
			yaml.WriteString("                  \"" + envVar + "\": \"$" + envVar + "\"" + comma + "\n")
		}
	}
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}
