package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var safeInputsLog = logger.New("workflow:safe_inputs")

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

// HasSafeInputs checks if safe-inputs are configured
func HasSafeInputs(safeInputs *SafeInputsConfig) bool {
	return safeInputs != nil && len(safeInputs.Tools) > 0
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

// generateSafeInputsFiles generates the MCP server and tool files for safe-inputs
func (c *Compiler) generateSafeInputsFiles(safeInputs *SafeInputsConfig) error {
	if safeInputs == nil || len(safeInputs.Tools) == 0 {
		return nil
	}

	safeInputsLog.Printf("Generating safe-inputs files for %d tools", len(safeInputs.Tools))

	// Create the safe-inputs directory
	if err := os.MkdirAll(SafeInputsDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create safe-inputs directory: %w", err)
	}

	// Generate tools configuration JSON
	toolsConfig, err := c.generateToolsConfig(safeInputs)
	if err != nil {
		return fmt.Errorf("failed to generate tools config: %w", err)
	}

	// Write tools configuration
	toolsConfigPath := filepath.Join(SafeInputsDirectory, "tools.json")
	if err := os.WriteFile(toolsConfigPath, []byte(toolsConfig), 0644); err != nil {
		return fmt.Errorf("failed to write tools config: %w", err)
	}

	// Generate individual tool files
	for toolName, toolConfig := range safeInputs.Tools {
		if toolConfig.Script != "" {
			// JavaScript tool
			toolPath := filepath.Join(SafeInputsDirectory, toolName+".cjs")
			toolScript := c.generateJavaScriptTool(toolConfig)
			if err := os.WriteFile(toolPath, []byte(toolScript), 0644); err != nil {
				return fmt.Errorf("failed to write JavaScript tool %s: %w", toolName, err)
			}
		} else if toolConfig.Run != "" {
			// Shell script tool
			toolPath := filepath.Join(SafeInputsDirectory, toolName+".sh")
			toolScript := c.generateShellTool(toolConfig)
			if err := os.WriteFile(toolPath, []byte(toolScript), 0755); err != nil {
				return fmt.Errorf("failed to write shell tool %s: %w", toolName, err)
			}
		}
	}

	// Generate MCP server
	mcpServerPath := filepath.Join(SafeInputsDirectory, "mcp-server.cjs")
	mcpServerScript := c.generateSafeInputsMCPServer(safeInputs)
	if err := os.WriteFile(mcpServerPath, []byte(mcpServerScript), 0644); err != nil {
		return fmt.Errorf("failed to write MCP server: %w", err)
	}

	safeInputsLog.Print("Safe-inputs files generated successfully")
	return nil
}

// generateToolsConfig generates the JSON configuration for all tools
func (c *Compiler) generateToolsConfig(safeInputs *SafeInputsConfig) (string, error) {
	type ToolInputSchema struct {
		Type        string `json:"type"`
		Description string `json:"description,omitempty"`
		Default     any    `json:"default,omitempty"`
	}

	type ToolConfig struct {
		Name        string                     `json:"name"`
		Description string                     `json:"description"`
		Type        string                     `json:"type"` // "javascript" or "shell"
		InputSchema map[string]ToolInputSchema `json:"inputSchema"`
		Required    []string                   `json:"required,omitempty"`
	}

	tools := make(map[string]ToolConfig)

	for toolName, toolConfig := range safeInputs.Tools {
		tc := ToolConfig{
			Name:        toolName,
			Description: toolConfig.Description,
			InputSchema: make(map[string]ToolInputSchema),
		}

		if toolConfig.Script != "" {
			tc.Type = "javascript"
		} else {
			tc.Type = "shell"
		}

		var required []string
		for paramName, param := range toolConfig.Inputs {
			tc.InputSchema[paramName] = ToolInputSchema{
				Type:        param.Type,
				Description: param.Description,
				Default:     param.Default,
			}
			if param.Required {
				required = append(required, paramName)
			}
		}
		sort.Strings(required)
		tc.Required = required

		tools[toolName] = tc
	}

	data, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// generateJavaScriptTool generates the JavaScript wrapper for a tool
func (c *Compiler) generateJavaScriptTool(toolConfig *SafeInputToolConfig) string {
	var sb strings.Builder

	sb.WriteString("// @ts-check\n")
	sb.WriteString("// Auto-generated safe-input tool: " + toolConfig.Name + "\n\n")
	sb.WriteString("/**\n")
	sb.WriteString(" * " + toolConfig.Description + "\n")
	sb.WriteString(" * @param {Object} inputs - Input parameters\n")
	for paramName, param := range toolConfig.Inputs {
		sb.WriteString(fmt.Sprintf(" * @param {%s} inputs.%s - %s\n", param.Type, paramName, param.Description))
	}
	sb.WriteString(" * @returns {Promise<any>} Tool result\n")
	sb.WriteString(" */\n")
	sb.WriteString("async function execute(inputs) {\n")
	sb.WriteString("  " + strings.ReplaceAll(toolConfig.Script, "\n", "\n  ") + "\n")
	sb.WriteString("}\n\n")
	sb.WriteString("module.exports = { execute };\n")

	return sb.String()
}

// generateShellTool generates the shell script wrapper for a tool
func (c *Compiler) generateShellTool(toolConfig *SafeInputToolConfig) string {
	var sb strings.Builder

	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("# Auto-generated safe-input tool: " + toolConfig.Name + "\n")
	sb.WriteString("# " + toolConfig.Description + "\n\n")
	sb.WriteString("set -euo pipefail\n\n")
	sb.WriteString(toolConfig.Run + "\n")

	return sb.String()
}

// generateSafeInputsMCPServer generates the MCP server JavaScript for safe-inputs
func (c *Compiler) generateSafeInputsMCPServer(safeInputs *SafeInputsConfig) string {
	var sb strings.Builder

	sb.WriteString(`// @ts-check
// Auto-generated safe-inputs MCP server

const fs = require("fs");
const path = require("path");
const { execFile } = require("child_process");
const { promisify } = require("util");

const execFileAsync = promisify(execFile);

const { createServer, registerTool, start } = require("./mcp_server_core.cjs");

// Server info for safe inputs MCP server
const SERVER_INFO = { name: "safeinputs", version: "1.0.0" };

// Create the server instance
const MCP_LOG_DIR = process.env.GH_AW_MCP_LOG_DIR;
const server = createServer(SERVER_INFO, { logDir: MCP_LOG_DIR });

// Load tools configuration
const toolsConfigPath = path.join(__dirname, "tools.json");
let toolsConfig = {};

try {
  if (fs.existsSync(toolsConfigPath)) {
    toolsConfig = JSON.parse(fs.readFileSync(toolsConfigPath, "utf8"));
    server.debug("Loaded tools config: " + JSON.stringify(Object.keys(toolsConfig)));
  }
} catch (error) {
  server.debug("Error loading tools config: " + (error instanceof Error ? error.message : String(error)));
}

`)

	// Register each tool
	for toolName, toolConfig := range safeInputs.Tools {
		sb.WriteString(fmt.Sprintf("// Register tool: %s\n", toolName))
		
		// Build input schema
		inputSchema := map[string]any{
			"type":       "object",
			"properties": make(map[string]any),
		}
		
		props := inputSchema["properties"].(map[string]any)
		var required []string
		
		for paramName, param := range toolConfig.Inputs {
			props[paramName] = map[string]any{
				"type":        param.Type,
				"description": param.Description,
			}
			if param.Default != nil {
				props[paramName].(map[string]any)["default"] = param.Default
			}
			if param.Required {
				required = append(required, paramName)
			}
		}
		
		sort.Strings(required)
		if len(required) > 0 {
			inputSchema["required"] = required
		}
		
		inputSchemaJSON, _ := json.Marshal(inputSchema)
		
		if toolConfig.Script != "" {
			sb.WriteString(fmt.Sprintf(`registerTool(server, {
  name: %q,
  description: %q,
  inputSchema: %s,
  handler: async (args) => {
    try {
      const toolModule = require("./%s.cjs");
      const result = await toolModule.execute(args || {});
      return { content: [{ type: "text", text: JSON.stringify(result, null, 2) }] };
    } catch (error) {
      return { content: [{ type: "text", text: "Error: " + (error instanceof Error ? error.message : String(error)) }], isError: true };
    }
  }
});

`, toolName, toolConfig.Description, string(inputSchemaJSON), toolName))
		} else {
			sb.WriteString(fmt.Sprintf(`registerTool(server, {
  name: %q,
  description: %q,
  inputSchema: %s,
  handler: async (args) => {
    try {
      // Set input parameters as environment variables
      const env = { ...process.env };
`, toolName, toolConfig.Description, string(inputSchemaJSON)))

			for paramName := range toolConfig.Inputs {
				sb.WriteString(fmt.Sprintf(`      if (args && args.%s !== undefined) {
        env["INPUT_%s"] = typeof args.%s === "object" ? JSON.stringify(args.%s) : String(args.%s);
      }
`, paramName, strings.ToUpper(paramName), paramName, paramName, paramName))
			}

			sb.WriteString(fmt.Sprintf(`
      const scriptPath = path.join(__dirname, "%s.sh");
      const { stdout, stderr } = await execFileAsync("bash", [scriptPath], { env });
      const output = stdout + (stderr ? "\nStderr: " + stderr : "");
      return { content: [{ type: "text", text: output }] };
    } catch (error) {
      return { content: [{ type: "text", text: "Error: " + (error instanceof Error ? error.message : String(error)) }], isError: true };
    }
  }
});

`, toolName))
		}
	}

	sb.WriteString(`// Start the MCP server
start(server);
`)

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

	for _, toolConfig := range safeInputs.Tools {
		for envName, envValue := range toolConfig.Env {
			secrets[envName] = envValue
		}
	}

	return secrets
}

// renderSafeInputsMCPConfig generates the Safe Inputs MCP server configuration
func renderSafeInputsMCPConfig(yaml *strings.Builder, safeInputs *SafeInputsConfig, isLast bool) {
	safeInputsLog.Print("Rendering Safe Inputs MCP configuration")
	renderSafeInputsMCPConfigWithOptions(yaml, safeInputs, isLast, false)
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
