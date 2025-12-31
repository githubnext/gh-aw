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
// Only supports HTTP transport mode
func renderSafeInputsMCPConfigWithOptions(yaml *strings.Builder, safeInputs *SafeInputsConfig, isLast bool, includeCopilotFields bool) {
	yaml.WriteString("              \"" + constants.SafeInputsMCPServerID + "\": {\n")

	// HTTP transport configuration - server started in separate step
	// Add type field for HTTP (required by MCP specification for HTTP transport)
	yaml.WriteString("                \"type\": \"http\",\n")

	// HTTP URL using environment variable
	// Use host.docker.internal to allow access from firewall container
	if includeCopilotFields {
		// Copilot format: backslash-escaped shell variable reference
		yaml.WriteString("                \"url\": \"http://host.docker.internal:\\${GH_AW_SAFE_INPUTS_PORT}\",\n")
	} else {
		// Claude/Custom format: direct shell variable reference
		yaml.WriteString("                \"url\": \"http://host.docker.internal:$GH_AW_SAFE_INPUTS_PORT\",\n")
	}

	// Add Authorization header with API key
	yaml.WriteString("                \"headers\": {\n")
	if includeCopilotFields {
		// Copilot format: backslash-escaped shell variable reference
		yaml.WriteString("                  \"Authorization\": \"Bearer \\${GH_AW_SAFE_INPUTS_API_KEY}\"\n")
	} else {
		// Claude/Custom format: direct shell variable reference
		yaml.WriteString("                  \"Authorization\": \"Bearer $GH_AW_SAFE_INPUTS_API_KEY\"\n")
	}
	yaml.WriteString("                },\n")

	// Add tools field for Copilot
	if includeCopilotFields {
		yaml.WriteString("                \"tools\": [\"*\"],\n")
	}

	// Add env block for server configuration environment variables only
	// Note: Tool-specific env vars (like GH_AW_GH_TOKEN) are already set in the step's env block
	// and don't need to be passed through the MCP config since the server uses HTTP transport
	yaml.WriteString("                \"env\": {\n")

	// Only include server configuration variables
	serverConfigVars := []string{"GH_AW_SAFE_INPUTS_PORT", "GH_AW_SAFE_INPUTS_API_KEY"}

	// Write environment variables with appropriate escaping
	for i, envVar := range serverConfigVars {
		isLastEnvVar := i == len(serverConfigVars)-1
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
