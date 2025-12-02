package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var safeInputsLog = logger.New("workflow:safe_inputs")

// sanitizeParameterName converts a parameter name to a safe JavaScript identifier
// by replacing non-alphanumeric characters with underscores
func sanitizeParameterName(name string) string {
	// Replace dashes and other non-alphanumeric chars with underscores
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '$' {
			return r
		}
		return '_'
	}, name)

	// Ensure it doesn't start with a number
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "_" + result
	}

	return result
}

// SafeInputsConfig holds the configuration for safe-inputs custom tools
type SafeInputsConfig struct {
	Tools map[string]*SafeInputToolConfig
}

// SafeInputToolConfig holds the configuration for a single safe-input tool
type SafeInputToolConfig struct {
	Name        string                     // Tool name (key from the config)
	Description string                     // Required: tool description
	Inputs      map[string]*SafeInputParam // Optional: input parameters
	Script      string                     // JavaScript implementation (mutually exclusive with Run)
	Run         string                     // Shell script implementation (mutually exclusive with Script)
	Env         map[string]string          // Environment variables (typically for secrets)
}

// SafeInputParam holds the configuration for a tool input parameter
type SafeInputParam struct {
	Type        string // JSON schema type (string, number, boolean, array, object)
	Description string // Description of the parameter
	Required    bool   // Whether the parameter is required
	Default     any    // Default value
}

// SafeInputsFeatureFlag is the name of the feature flag for safe-inputs
const SafeInputsFeatureFlag = "safe-inputs"

// HasSafeInputs checks if safe-inputs are configured
func HasSafeInputs(safeInputs *SafeInputsConfig) bool {
	return safeInputs != nil && len(safeInputs.Tools) > 0
}

// IsSafeInputsEnabled checks if safe-inputs are both configured AND the feature is enabled.
// The safe-inputs feature requires the feature flag to be enabled via:
// - Frontmatter: features: { safe-inputs: true }
// - Environment variable: GH_AW_FEATURES=safe-inputs
func IsSafeInputsEnabled(safeInputs *SafeInputsConfig, workflowData *WorkflowData) bool {
	if !HasSafeInputs(safeInputs) {
		return false
	}
	return isFeatureEnabled(SafeInputsFeatureFlag, workflowData)
}

// ParseSafeInputs parses safe-inputs configuration from frontmatter (standalone function for testing)
func ParseSafeInputs(frontmatter map[string]any) *SafeInputsConfig {
	if frontmatter == nil {
		return nil
	}

	safeInputs, exists := frontmatter["safe-inputs"]
	if !exists {
		return nil
	}

	safeInputsMap, ok := safeInputs.(map[string]any)
	if !ok {
		return nil
	}

	config := &SafeInputsConfig{
		Tools: make(map[string]*SafeInputToolConfig),
	}

	for toolName, toolValue := range safeInputsMap {
		toolMap, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}

		toolConfig := &SafeInputToolConfig{
			Name:   toolName,
			Inputs: make(map[string]*SafeInputParam),
			Env:    make(map[string]string),
		}

		// Parse description (required)
		if desc, exists := toolMap["description"]; exists {
			if descStr, ok := desc.(string); ok {
				toolConfig.Description = descStr
			}
		}

		// Parse inputs (optional)
		if inputs, exists := toolMap["inputs"]; exists {
			if inputsMap, ok := inputs.(map[string]any); ok {
				for paramName, paramValue := range inputsMap {
					if paramMap, ok := paramValue.(map[string]any); ok {
						param := &SafeInputParam{
							Type: "string", // default type
						}

						if t, exists := paramMap["type"]; exists {
							if tStr, ok := t.(string); ok {
								param.Type = tStr
							}
						}

						if desc, exists := paramMap["description"]; exists {
							if descStr, ok := desc.(string); ok {
								param.Description = descStr
							}
						}

						if def, exists := paramMap["default"]; exists {
							param.Default = def
						}

						if req, exists := paramMap["required"]; exists {
							if reqBool, ok := req.(bool); ok {
								param.Required = reqBool
							}
						}

						toolConfig.Inputs[paramName] = param
					}
				}
			}
		}

		// Parse script (for JavaScript tools)
		if script, exists := toolMap["script"]; exists {
			if scriptStr, ok := script.(string); ok {
				toolConfig.Script = scriptStr
			}
		}

		// Parse run (for shell tools)
		if run, exists := toolMap["run"]; exists {
			if runStr, ok := run.(string); ok {
				toolConfig.Run = runStr
			}
		}

		// Parse env (for secrets)
		if env, exists := toolMap["env"]; exists {
			if envMap, ok := env.(map[string]any); ok {
				for envName, envValue := range envMap {
					if envStr, ok := envValue.(string); ok {
						toolConfig.Env[envName] = envStr
					}
				}
			}
		}

		config.Tools[toolName] = toolConfig
	}

	return config
}

// extractSafeInputsConfig extracts safe-inputs configuration from frontmatter
func (c *Compiler) extractSafeInputsConfig(frontmatter map[string]any) *SafeInputsConfig {
	safeInputsLog.Print("Extracting safe-inputs configuration from frontmatter")

	safeInputs, exists := frontmatter["safe-inputs"]
	if !exists {
		return nil
	}

	safeInputsMap, ok := safeInputs.(map[string]any)
	if !ok {
		return nil
	}

	config := &SafeInputsConfig{
		Tools: make(map[string]*SafeInputToolConfig),
	}

	for toolName, toolValue := range safeInputsMap {
		toolMap, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}

		toolConfig := &SafeInputToolConfig{
			Name:   toolName,
			Inputs: make(map[string]*SafeInputParam),
			Env:    make(map[string]string),
		}

		// Parse description (required)
		if desc, exists := toolMap["description"]; exists {
			if descStr, ok := desc.(string); ok {
				toolConfig.Description = descStr
			}
		}

		// Parse inputs (optional)
		if inputs, exists := toolMap["inputs"]; exists {
			if inputsMap, ok := inputs.(map[string]any); ok {
				for paramName, paramValue := range inputsMap {
					if paramMap, ok := paramValue.(map[string]any); ok {
						param := &SafeInputParam{
							Type: "string", // default type
						}

						if t, exists := paramMap["type"]; exists {
							if tStr, ok := t.(string); ok {
								param.Type = tStr
							}
						}

						if desc, exists := paramMap["description"]; exists {
							if descStr, ok := desc.(string); ok {
								param.Description = descStr
							}
						}

						if req, exists := paramMap["required"]; exists {
							if reqBool, ok := req.(bool); ok {
								param.Required = reqBool
							}
						}

						if def, exists := paramMap["default"]; exists {
							param.Default = def
						}

						toolConfig.Inputs[paramName] = param
					}
				}
			}
		}

		// Parse script (JavaScript implementation)
		if script, exists := toolMap["script"]; exists {
			if scriptStr, ok := script.(string); ok {
				toolConfig.Script = scriptStr
			}
		}

		// Parse run (shell script implementation)
		if run, exists := toolMap["run"]; exists {
			if runStr, ok := run.(string); ok {
				toolConfig.Run = runStr
			}
		}

		// Parse env (environment variables)
		if env, exists := toolMap["env"]; exists {
			if envMap, ok := env.(map[string]any); ok {
				for envName, envValue := range envMap {
					if envStr, ok := envValue.(string); ok {
						toolConfig.Env[envName] = envStr
					}
				}
			}
		}

		config.Tools[toolName] = toolConfig
	}

	if len(config.Tools) == 0 {
		return nil
	}

	safeInputsLog.Printf("Extracted %d safe-input tools", len(config.Tools))
	return config
}

// SafeInputsDirectory is the directory where safe-inputs files are generated
const SafeInputsDirectory = "/tmp/gh-aw/safe-inputs"

// SafeInputsToolJSON represents a tool configuration for the tools.json file
type SafeInputsToolJSON struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Handler     string         `json:"handler,omitempty"`
}

// SafeInputsConfigJSON represents the tools.json configuration file structure
type SafeInputsConfigJSON struct {
	ServerName string               `json:"serverName"`
	Version    string               `json:"version"`
	LogDir     string               `json:"logDir,omitempty"`
	Tools      []SafeInputsToolJSON `json:"tools"`
}

// generateSafeInputsToolsConfig generates the tools.json configuration for the safe-inputs MCP server
func generateSafeInputsToolsConfig(safeInputs *SafeInputsConfig) string {
	config := SafeInputsConfigJSON{
		ServerName: "safeinputs",
		Version:    constants.SafeInputsMCPVersion,
		LogDir:     SafeInputsDirectory + "/logs",
		Tools:      []SafeInputsToolJSON{},
	}

	// Sort tool names for stable output
	toolNames := make([]string, 0, len(safeInputs.Tools))
	for toolName := range safeInputs.Tools {
		toolNames = append(toolNames, toolName)
	}
	sort.Strings(toolNames)

	for _, toolName := range toolNames {
		toolConfig := safeInputs.Tools[toolName]

		// Build input schema
		inputSchema := map[string]any{
			"type":       "object",
			"properties": make(map[string]any),
		}

		props := inputSchema["properties"].(map[string]any)
		var required []string

		// Sort input names for stable output
		inputNames := make([]string, 0, len(toolConfig.Inputs))
		for paramName := range toolConfig.Inputs {
			inputNames = append(inputNames, paramName)
		}
		sort.Strings(inputNames)

		for _, paramName := range inputNames {
			param := toolConfig.Inputs[paramName]
			propDef := map[string]any{
				"type":        param.Type,
				"description": param.Description,
			}
			if param.Default != nil {
				propDef["default"] = param.Default
			}
			props[paramName] = propDef
			if param.Required {
				required = append(required, paramName)
			}
		}

		sort.Strings(required)
		if len(required) > 0 {
			inputSchema["required"] = required
		}

		// Determine handler path based on script type
		var handler string
		if toolConfig.Script != "" {
			handler = toolName + ".cjs"
		} else if toolConfig.Run != "" {
			handler = toolName + ".sh"
		}

		config.Tools = append(config.Tools, SafeInputsToolJSON{
			Name:        toolName,
			Description: toolConfig.Description,
			InputSchema: inputSchema,
			Handler:     handler,
		})
	}

	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		safeInputsLog.Printf("Error marshaling tools config: %v", err)
		return "{}"
	}
	return string(jsonBytes)
}

// generateSafeInputsMCPServerScript generates the entry point script for the safe-inputs MCP server
// This uses the reusable safe_inputs_mcp_server.cjs module and reads tool configuration from tools.json
func generateSafeInputsMCPServerScript(safeInputs *SafeInputsConfig) string {
	var sb strings.Builder

	// Write a simple entry point that uses the modular MCP server
	sb.WriteString(`// @ts-check
// Auto-generated safe-inputs MCP server entry point
// This script uses the reusable safe_inputs_mcp_server module

const path = require("path");
const { startSafeInputsServer } = require("./safe_inputs_mcp_server.cjs");

// Configuration file path (generated alongside this script)
const configPath = path.join(__dirname, "tools.json");

// Start the server
startSafeInputsServer(configPath, {
  logDir: "/tmp/gh-aw/safe-inputs/logs"
});
`)

	return sb.String()
}

// generateSafeInputJavaScriptToolScript generates the JavaScript tool file for a safe-input tool
// The user's script code is automatically wrapped in a function with module.exports,
// so users can write simple code without worrying about exports.
// Input parameters are destructured and available as local variables.
func generateSafeInputJavaScriptToolScript(toolConfig *SafeInputToolConfig) string {
	var sb strings.Builder

	sb.WriteString("// @ts-check\n")
	sb.WriteString("// Auto-generated safe-input tool: " + toolConfig.Name + "\n\n")
	sb.WriteString("/**\n")
	sb.WriteString(" * " + toolConfig.Description + "\n")
	sb.WriteString(" * @param {Object} inputs - Input parameters\n")
	// Sort input names for stable code generation in JSDoc
	inputNamesForDoc := make([]string, 0, len(toolConfig.Inputs))
	for paramName := range toolConfig.Inputs {
		inputNamesForDoc = append(inputNamesForDoc, paramName)
	}
	sort.Strings(inputNamesForDoc)
	for _, paramName := range inputNamesForDoc {
		param := toolConfig.Inputs[paramName]
		sb.WriteString(fmt.Sprintf(" * @param {%s} inputs.%s - %s\n", param.Type, paramName, param.Description))
	}
	sb.WriteString(" * @returns {Promise<any>} Tool result\n")
	sb.WriteString(" */\n")
	sb.WriteString("async function execute(inputs) {\n")

	// Destructure inputs to make parameters available as local variables
	if len(toolConfig.Inputs) > 0 {
		var paramNames []string
		for paramName := range toolConfig.Inputs {
			safeName := sanitizeParameterName(paramName)
			if safeName != paramName {
				// If sanitized, use alias
				paramNames = append(paramNames, fmt.Sprintf("%s: %s", paramName, safeName))
			} else {
				paramNames = append(paramNames, paramName)
			}
		}
		sort.Strings(paramNames)
		sb.WriteString(fmt.Sprintf("  const { %s } = inputs || {};\n\n", strings.Join(paramNames, ", ")))
	}

	// Indent the user's script code
	sb.WriteString("  " + strings.ReplaceAll(toolConfig.Script, "\n", "\n  ") + "\n")
	sb.WriteString("}\n\n")
	sb.WriteString("module.exports = { execute };\n")

	return sb.String()
}

// generateSafeInputShellToolScript generates the shell script for a safe-input tool
func generateSafeInputShellToolScript(toolConfig *SafeInputToolConfig) string {
	var sb strings.Builder

	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("# Auto-generated safe-input tool: " + toolConfig.Name + "\n")
	sb.WriteString("# " + toolConfig.Description + "\n\n")
	sb.WriteString("set -euo pipefail\n\n")
	sb.WriteString(toolConfig.Run + "\n")

	return sb.String()
}

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
func renderSafeInputsMCPConfigWithOptions(yaml *strings.Builder, safeInputs *SafeInputsConfig, isLast bool, includeCopilotFields bool) {
	envVars := getSafeInputsEnvVars(safeInputs)

	renderBuiltinMCPServerBlock(
		yaml,
		constants.SafeInputsMCPServerID,
		"node",
		[]string{SafeInputsDirectory + "/mcp-server.cjs"},
		envVars,
		isLast,
		includeCopilotFields,
	)
}

// mergeSafeInputs merges safe-inputs configuration from imports into the main configuration
func (c *Compiler) mergeSafeInputs(main *SafeInputsConfig, importedConfigs []string) *SafeInputsConfig {
	if main == nil {
		main = &SafeInputsConfig{
			Tools: make(map[string]*SafeInputToolConfig),
		}
	}

	for _, configJSON := range importedConfigs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}

		// Parse the imported JSON config
		var importedMap map[string]any
		if err := json.Unmarshal([]byte(configJSON), &importedMap); err != nil {
			safeInputsLog.Printf("Warning: failed to parse imported safe-inputs config: %v", err)
			continue
		}

		// Merge each tool from the imported config
		for toolName, toolValue := range importedMap {
			// Skip if tool already exists in main config (main takes precedence)
			if _, exists := main.Tools[toolName]; exists {
				safeInputsLog.Printf("Skipping imported tool '%s' - already defined in main config", toolName)
				continue
			}

			toolMap, ok := toolValue.(map[string]any)
			if !ok {
				continue
			}

			toolConfig := &SafeInputToolConfig{
				Name:   toolName,
				Inputs: make(map[string]*SafeInputParam),
				Env:    make(map[string]string),
			}

			// Parse description
			if desc, exists := toolMap["description"]; exists {
				if descStr, ok := desc.(string); ok {
					toolConfig.Description = descStr
				}
			}

			// Parse inputs
			if inputs, exists := toolMap["inputs"]; exists {
				if inputsMap, ok := inputs.(map[string]any); ok {
					for paramName, paramValue := range inputsMap {
						if paramMap, ok := paramValue.(map[string]any); ok {
							param := &SafeInputParam{
								Type: "string",
							}
							if t, exists := paramMap["type"]; exists {
								if tStr, ok := t.(string); ok {
									param.Type = tStr
								}
							}
							if desc, exists := paramMap["description"]; exists {
								if descStr, ok := desc.(string); ok {
									param.Description = descStr
								}
							}
							if req, exists := paramMap["required"]; exists {
								if reqBool, ok := req.(bool); ok {
									param.Required = reqBool
								}
							}
							if def, exists := paramMap["default"]; exists {
								param.Default = def
							}
							toolConfig.Inputs[paramName] = param
						}
					}
				}
			}

			// Parse script
			if script, exists := toolMap["script"]; exists {
				if scriptStr, ok := script.(string); ok {
					toolConfig.Script = scriptStr
				}
			}

			// Parse run
			if run, exists := toolMap["run"]; exists {
				if runStr, ok := run.(string); ok {
					toolConfig.Run = runStr
				}
			}

			// Parse env
			if env, exists := toolMap["env"]; exists {
				if envMap, ok := env.(map[string]any); ok {
					for envName, envValue := range envMap {
						if envStr, ok := envValue.(string); ok {
							toolConfig.Env[envName] = envStr
						}
					}
				}
			}

			main.Tools[toolName] = toolConfig
			safeInputsLog.Printf("Merged imported safe-input tool: %s", toolName)
		}
	}

	return main
}
