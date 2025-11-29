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

// HasSafeInputs checks if safe-inputs are configured
func HasSafeInputs(safeInputs *SafeInputsConfig) bool {
	return safeInputs != nil && len(safeInputs.Tools) > 0
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

// generateSafeInputsMCPServerScript generates a self-contained MCP server for safe-inputs
func generateSafeInputsMCPServerScript(safeInputs *SafeInputsConfig) string {
	var sb strings.Builder

	// Write the MCP server core inline (simplified version for safe-inputs)
	sb.WriteString(`// @ts-check
// Auto-generated safe-inputs MCP server

const fs = require("fs");
const path = require("path");
const { execFile } = require("child_process");
const { promisify } = require("util");

const execFileAsync = promisify(execFile);

// Simple ReadBuffer implementation for JSON-RPC parsing
class ReadBuffer {
  constructor() {
    this.buffer = Buffer.alloc(0);
  }
  append(chunk) {
    this.buffer = Buffer.concat([this.buffer, chunk]);
  }
  readMessage() {
    const headerEndIndex = this.buffer.indexOf("\r\n\r\n");
    if (headerEndIndex === -1) return null;
    const header = this.buffer.slice(0, headerEndIndex).toString();
    const match = header.match(/Content-Length:\s*(\d+)/i);
    if (!match) return null;
    const contentLength = parseInt(match[1], 10);
    const messageStart = headerEndIndex + 4;
    if (this.buffer.length < messageStart + contentLength) return null;
    const content = this.buffer.slice(messageStart, messageStart + contentLength).toString();
    this.buffer = this.buffer.slice(messageStart + contentLength);
    return JSON.parse(content);
  }
}

// Create MCP server
const serverInfo = { name: "safeinputs", version: "1.0.0" };
const tools = {};
const readBuffer = new ReadBuffer();

function debug(msg) {
  const timestamp = new Date().toISOString();
  process.stderr.write("[" + timestamp + "] [safeinputs] " + msg + "\n");
}

function writeMessage(message) {
  const json = JSON.stringify(message);
  const header = "Content-Length: " + Buffer.byteLength(json) + "\r\n\r\n";
  process.stdout.write(header + json);
}

function replyResult(id, result) {
  writeMessage({ jsonrpc: "2.0", id, result });
}

function replyError(id, code, message) {
  writeMessage({ jsonrpc: "2.0", id, error: { code, message } });
}

function registerTool(name, description, inputSchema, handler) {
  tools[name] = { name, description, inputSchema, handler };
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
			sb.WriteString(fmt.Sprintf(`registerTool(%q, %q, %s, async (args) => {
  try {
    const toolModule = require("./%s.cjs");
    const result = await toolModule.execute(args || {});
    return { content: [{ type: "text", text: typeof result === "string" ? result : JSON.stringify(result, null, 2) }] };
  } catch (error) {
    return { content: [{ type: "text", text: "Error: " + (error instanceof Error ? error.message : String(error)) }], isError: true };
  }
});

`, toolName, toolConfig.Description, string(inputSchemaJSON), toolName))
		} else {
			sb.WriteString(fmt.Sprintf(`registerTool(%q, %q, %s, async (args) => {
  try {
    // Set input parameters as environment variables
    const env = { ...process.env };
`, toolName, toolConfig.Description, string(inputSchemaJSON)))

			for paramName := range toolConfig.Inputs {
				// Use bracket notation for safer property access
				safeEnvName := strings.ToUpper(sanitizeParameterName(paramName))
				sb.WriteString(fmt.Sprintf(`    if (args && args[%q] !== undefined) {
      env["INPUT_%s"] = typeof args[%q] === "object" ? JSON.stringify(args[%q]) : String(args[%q]);
    }
`, paramName, safeEnvName, paramName, paramName, paramName))
			}

			sb.WriteString(fmt.Sprintf(`
    const scriptPath = path.join(__dirname, "%s.sh");
    const { stdout, stderr } = await execFileAsync("bash", [scriptPath], { env });
    const output = stdout + (stderr ? "\nStderr: " + stderr : "");
    return { content: [{ type: "text", text: output }] };
  } catch (error) {
    return { content: [{ type: "text", text: "Error: " + (error instanceof Error ? error.message : String(error)) }], isError: true };
  }
});

`, toolName))
		}
	}

	// Add message handler and start
	sb.WriteString(`// Handle incoming messages
async function handleMessage(message) {
  if (message.method === "initialize") {
    debug("Received initialize request");
    replyResult(message.id, {
      protocolVersion: "2024-11-05",
      capabilities: { tools: {} },
      serverInfo
    });
  } else if (message.method === "notifications/initialized") {
    debug("Client initialized");
  } else if (message.method === "tools/list") {
    debug("Received tools/list request");
    const toolList = Object.values(tools).map(t => ({
      name: t.name,
      description: t.description,
      inputSchema: t.inputSchema
    }));
    replyResult(message.id, { tools: toolList });
  } else if (message.method === "tools/call") {
    const toolName = message.params?.name;
    const toolArgs = message.params?.arguments || {};
    debug("Received tools/call for: " + toolName);
    const tool = tools[toolName];
    if (!tool) {
      replyError(message.id, -32601, "Unknown tool: " + toolName);
      return;
    }
    try {
      const result = await tool.handler(toolArgs);
      replyResult(message.id, result);
    } catch (error) {
      replyError(message.id, -32603, error instanceof Error ? error.message : String(error));
    }
  } else {
    debug("Unknown method: " + message.method);
    if (message.id !== undefined) {
      replyError(message.id, -32601, "Method not found");
    }
  }
}

// Start server
debug("Starting safe-inputs MCP server");
process.stdin.on("data", async (chunk) => {
  readBuffer.append(chunk);
  let message;
  while ((message = readBuffer.readMessage()) !== null) {
    await handleMessage(message);
  }
});
`)

	return sb.String()
}

// generateSafeInputJavaScriptToolScript generates the JavaScript tool file for a safe-input tool
func generateSafeInputJavaScriptToolScript(toolConfig *SafeInputToolConfig) string {
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

	for _, toolConfig := range safeInputs.Tools {
		for envName, envValue := range toolConfig.Env {
			secrets[envName] = envValue
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
