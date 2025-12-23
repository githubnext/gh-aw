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
	Mode  string // Transport mode: "http" (default) or "stdio"
	Tools map[string]*SafeInputToolConfig
}

// SafeInputToolConfig holds the configuration for a single safe-input tool
type SafeInputToolConfig struct {
	Name        string                     // Tool name (key from the config)
	Description string                     // Required: tool description
	Inputs      map[string]*SafeInputParam // Optional: input parameters
	Script      string                     // JavaScript implementation (mutually exclusive with Run and Py)
	Run         string                     // Shell script implementation (mutually exclusive with Script and Py)
	Py          string                     // Python script implementation (mutually exclusive with Script and Run)
	Env         map[string]string          // Environment variables (typically for secrets)
	Timeout     int                        // Timeout in seconds for tool execution (default: 60)
}

// SafeInputParam holds the configuration for a tool input parameter
type SafeInputParam struct {
	Type        string // JSON schema type (string, number, boolean, array, object)
	Description string // Description of the parameter
	Required    bool   // Whether the parameter is required
	Default     any    // Default value
}

// SafeInputsMode constants define the available transport modes
const (
	SafeInputsModeHTTP = "http"
)

// HasSafeInputs checks if safe-inputs are configured
func HasSafeInputs(safeInputs *SafeInputsConfig) bool {
	return safeInputs != nil && len(safeInputs.Tools) > 0
}

// IsSafeInputsHTTPMode checks if safe-inputs is configured to use HTTP mode
// Note: All safe-inputs configurations now use HTTP mode exclusively
func IsSafeInputsHTTPMode(safeInputs *SafeInputsConfig) bool {
	return safeInputs != nil
}

// IsSafeInputsEnabled checks if safe-inputs are configured.
// Safe-inputs are enabled by default when configured in the workflow.
// The workflowData parameter is kept for backward compatibility but is not used.
func IsSafeInputsEnabled(safeInputs *SafeInputsConfig, workflowData *WorkflowData) bool {
	return HasSafeInputs(safeInputs)
}

// parseSafeInputsMap parses safe-inputs configuration from a map.
// This is the shared implementation used by both ParseSafeInputs and extractSafeInputsConfig.
// Returns the config and a boolean indicating whether any tools were found.
func parseSafeInputsMap(safeInputsMap map[string]any) (*SafeInputsConfig, bool) {
	config := &SafeInputsConfig{
		Mode:  "http", // Only HTTP mode is supported
		Tools: make(map[string]*SafeInputToolConfig),
	}

	// Mode field is ignored - only HTTP mode is supported
	// All safe-inputs configurations use HTTP transport

	for toolName, toolValue := range safeInputsMap {
		// Skip the "mode" field as it's not a tool definition
		if toolName == "mode" {
			continue
		}

		toolMap, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}

		toolConfig := &SafeInputToolConfig{
			Name:    toolName,
			Inputs:  make(map[string]*SafeInputParam),
			Env:     make(map[string]string),
			Timeout: 60, // Default timeout: 60 seconds
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

		// Parse py (Python script implementation)
		if py, exists := toolMap["py"]; exists {
			if pyStr, ok := py.(string); ok {
				toolConfig.Py = pyStr
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

		// Parse timeout (optional, default is 60 seconds)
		if timeout, exists := toolMap["timeout"]; exists {
			switch t := timeout.(type) {
			case int:
				toolConfig.Timeout = t
			case uint64:
				toolConfig.Timeout = int(t)
			case float64:
				toolConfig.Timeout = int(t)
			case string:
				// Try to parse string as integer
				_, _ = fmt.Sscanf(t, "%d", &toolConfig.Timeout)
			}
		}

		config.Tools[toolName] = toolConfig
	}

	return config, len(config.Tools) > 0
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

	config, _ := parseSafeInputsMap(safeInputsMap)
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

	config, hasTools := parseSafeInputsMap(safeInputsMap)
	if !hasTools {
		return nil
	}

	safeInputsLog.Printf("Extracted %d safe-input tools", len(config.Tools))
	return config
}

// SafeInputsDirectory is the directory where safe-inputs files are generated
const SafeInputsDirectory = "/tmp/gh-aw/safe-inputs"

// SafeInputsToolJSON represents a tool configuration for the tools.json file
type SafeInputsToolJSON struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	InputSchema map[string]any    `json:"inputSchema"`
	Handler     string            `json:"handler,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
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
		} else if toolConfig.Py != "" {
			handler = toolName + ".py"
		}

		// Build env list of required environment variables (not actual secrets)
		// This documents which env vars the tool needs, but doesn't store secret values
		// The actual values are passed as environment variables and accessed via process.env
		var envRefs map[string]string
		if len(toolConfig.Env) > 0 {
			envRefs = make(map[string]string)
			// Sort env var names for stable output
			envVarNames := make([]string, 0, len(toolConfig.Env))
			for envVarName := range toolConfig.Env {
				envVarNames = append(envVarNames, envVarName)
			}
			sort.Strings(envVarNames)

			for _, envVarName := range envVarNames {
				// Store just the environment variable name without $ prefix or secret value
				// Handlers access the actual value via process.env[envVarName] at runtime
				envRefs[envVarName] = envVarName
			}
		}

		config.Tools = append(config.Tools, SafeInputsToolJSON{
			Name:        toolName,
			Description: toolConfig.Description,
			InputSchema: inputSchema,
			Handler:     handler,
			Env:         envRefs,
			Timeout:     toolConfig.Timeout,
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
// This script uses HTTP transport exclusively
func generateSafeInputsMCPServerScript(safeInputs *SafeInputsConfig) string {
	var sb strings.Builder

	// HTTP transport - server started in separate step
	sb.WriteString(`// @ts-check
// Auto-generated safe-inputs MCP server entry point (HTTP transport)
// This script uses the reusable safe_inputs_mcp_server_http module

const path = require("path");
const { startHttpServer } = require("./safe_inputs_mcp_server_http.cjs");

// Configuration file path (generated alongside this script)
const configPath = path.join(__dirname, "tools.json");

// Get port and API key from environment variables
const port = parseInt(process.env.GH_AW_SAFE_INPUTS_PORT || "3000", 10);
const apiKey = process.env.GH_AW_SAFE_INPUTS_API_KEY || "";

// Start the HTTP server
startHttpServer(configPath, {
  port: port,
  stateless: false,
  logDir: "/tmp/gh-aw/safe-inputs/logs"
}).catch(error => {
  console.error("Failed to start safe-inputs HTTP server:", error);
  process.exit(1);
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
		fmt.Fprintf(&sb, " * @param {%s} inputs.%s - %s\n", param.Type, paramName, param.Description)
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
		fmt.Fprintf(&sb, "  const { %s } = inputs || {};\n\n", strings.Join(paramNames, ", "))
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

// generateSafeInputPythonToolScript generates the Python script for a safe-input tool
// Python scripts receive inputs as a dictionary (parsed from JSON stdin):
// - Input parameters are available as a pre-parsed 'inputs' dictionary
// - Individual parameters can be destructured: param = inputs.get('param', default)
// - Outputs are printed to stdout as JSON
// - Environment variables from env: field are available via os.environ
func generateSafeInputPythonToolScript(toolConfig *SafeInputToolConfig) string {
	var sb strings.Builder

	sb.WriteString("#!/usr/bin/env python3\n")
	sb.WriteString("# Auto-generated safe-input tool: " + toolConfig.Name + "\n")
	sb.WriteString("# " + toolConfig.Description + "\n\n")
	sb.WriteString("import json\n")
	sb.WriteString("import os\n")
	sb.WriteString("import sys\n\n")

	// Add wrapper code to read inputs from stdin
	sb.WriteString("# Read inputs from stdin (JSON format)\n")
	sb.WriteString("try:\n")
	sb.WriteString("    inputs = json.loads(sys.stdin.read()) if not sys.stdin.isatty() else {}\n")
	sb.WriteString("except (json.JSONDecodeError, Exception):\n")
	sb.WriteString("    inputs = {}\n\n")

	// Add helper comment about input parameters
	if len(toolConfig.Inputs) > 0 {
		sb.WriteString("# Input parameters available in 'inputs' dictionary:\n")
		// Sort input names for stable code generation
		inputNames := make([]string, 0, len(toolConfig.Inputs))
		for paramName := range toolConfig.Inputs {
			inputNames = append(inputNames, paramName)
		}
		sort.Strings(inputNames)
		for _, paramName := range inputNames {
			param := toolConfig.Inputs[paramName]
			defaultValue := ""
			if param.Default != nil {
				defaultValue = fmt.Sprintf(", default=%v", param.Default)
			}
			fmt.Fprintf(&sb, "# %s = inputs.get('%s'%s)  # %s\n",
				sanitizePythonVariableName(paramName), paramName, defaultValue, param.Description)
		}
		sb.WriteString("\n")
	}

	// Add user's Python code
	sb.WriteString("# User code:\n")
	sb.WriteString(toolConfig.Py + "\n")

	return sb.String()
}

// sanitizePythonVariableName converts a parameter name to a valid Python identifier
func sanitizePythonVariableName(name string) string {
	// Replace dashes and other non-alphanumeric chars with underscores
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
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
	envVars := getSafeInputsEnvVars(safeInputs)

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

	// Add env block for environment variable passthrough
	envVarsWithServerConfig := append([]string{"GH_AW_SAFE_INPUTS_PORT", "GH_AW_SAFE_INPUTS_API_KEY"}, envVars...)
	yaml.WriteString("                \"env\": {\n")

	// Write environment variables with appropriate escaping
	for i, envVar := range envVarsWithServerConfig {
		isLastEnvVar := i == len(envVarsWithServerConfig)-1
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

// mergeSafeInputs merges safe-inputs configuration from imports into the main configuration
func (c *Compiler) mergeSafeInputs(main *SafeInputsConfig, importedConfigs []string) *SafeInputsConfig {
	if main == nil {
		main = &SafeInputsConfig{
			Mode:  "http", // Default to HTTP mode
			Tools: make(map[string]*SafeInputToolConfig),
		}
	}

	for _, configJSON := range importedConfigs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}

		// Merge the imported JSON config
		var importedMap map[string]any
		if err := json.Unmarshal([]byte(configJSON), &importedMap); err != nil {
			safeInputsLog.Printf("Warning: failed to parse imported safe-inputs config: %v", err)
			continue
		}

		// Mode field is ignored - only HTTP mode is supported
		// All safe-inputs configurations use HTTP transport

		// Merge each tool from the imported config
		for toolName, toolValue := range importedMap {
			// Skip mode field as it's already handled
			if toolName == "mode" {
				continue
			}

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
				Name:    toolName,
				Inputs:  make(map[string]*SafeInputParam),
				Env:     make(map[string]string),
				Timeout: 60, // Default timeout: 60 seconds
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

			// Parse py
			if py, exists := toolMap["py"]; exists {
				if pyStr, ok := py.(string); ok {
					toolConfig.Py = pyStr
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

			// Parse timeout (optional, default is 60 seconds)
			if timeout, exists := toolMap["timeout"]; exists {
				switch t := timeout.(type) {
				case int:
					toolConfig.Timeout = t
				case uint64:
					toolConfig.Timeout = int(t)
				case float64:
					toolConfig.Timeout = int(t)
				case string:
					// Try to parse string as integer
					_, _ = fmt.Sscanf(t, "%d", &toolConfig.Timeout)
				}
			}

			main.Tools[toolName] = toolConfig
			safeInputsLog.Printf("Merged imported safe-input tool: %s", toolName)
		}
	}

	return main
}

// Public wrapper functions for CLI use

// GenerateSafeInputsToolsConfigForInspector generates the tools.json configuration for the safe-inputs MCP server
// This is a public wrapper for use by the CLI inspector command
func GenerateSafeInputsToolsConfigForInspector(safeInputs *SafeInputsConfig) string {
	return generateSafeInputsToolsConfig(safeInputs)
}

// GenerateSafeInputsMCPServerScriptForInspector generates the MCP server entry point script
// This is a public wrapper for use by the CLI inspector command
func GenerateSafeInputsMCPServerScriptForInspector(safeInputs *SafeInputsConfig) string {
	return generateSafeInputsMCPServerScript(safeInputs)
}

// GenerateSafeInputJavaScriptToolScriptForInspector generates a JavaScript tool handler script
// This is a public wrapper for use by the CLI inspector command
func GenerateSafeInputJavaScriptToolScriptForInspector(toolConfig *SafeInputToolConfig) string {
	return generateSafeInputJavaScriptToolScript(toolConfig)
}

// GenerateSafeInputShellToolScriptForInspector generates a shell script tool handler
// This is a public wrapper for use by the CLI inspector command
func GenerateSafeInputShellToolScriptForInspector(toolConfig *SafeInputToolConfig) string {
	return generateSafeInputShellToolScript(toolConfig)
}

// GenerateSafeInputPythonToolScriptForInspector generates a Python script tool handler
// This is a public wrapper for use by the CLI inspector command
func GenerateSafeInputPythonToolScriptForInspector(toolConfig *SafeInputToolConfig) string {
	return generateSafeInputPythonToolScript(toolConfig)
}
