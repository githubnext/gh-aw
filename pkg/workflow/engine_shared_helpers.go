package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// InjectCustomEngineSteps processes custom steps from engine config and converts them to GitHubActionSteps.
// This shared function extracts the common pattern used by Copilot, Codex, and Claude engines.
//
// Parameters:
//   - workflowData: The workflow data containing engine configuration
//   - convertStepFunc: A function that converts a step map to YAML string (engine-specific)
//
// Returns:
//   - []GitHubActionStep: Array of custom steps ready to be included in the execution pipeline
func InjectCustomEngineSteps(
	workflowData *WorkflowData,
	convertStepFunc func(map[string]any) (string, error),
) []GitHubActionStep {
	var steps []GitHubActionStep

	// Handle custom steps if they exist in engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		for _, step := range workflowData.EngineConfig.Steps {
			stepYAML, err := convertStepFunc(step)
			if err != nil {
				// Log error but continue with other steps
				continue
			}
			steps = append(steps, GitHubActionStep{stepYAML})
		}
	}

	return steps
}

// RenderCustomMCPToolConfigHandler is a function type that engines must provide to render their specific MCP config
type RenderCustomMCPToolConfigHandler func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error

// HandleCustomMCPToolInSwitch processes custom MCP tools in the default case of a switch statement.
// This shared function extracts the common pattern used across all workflow engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - toolName: The name of the tool being processed
//   - tools: The tools map containing tool configurations (supports both expanded and non-expanded tools)
//   - isLast: Whether this is the last tool in the list
//   - renderFunc: Engine-specific function to render the MCP configuration
//
// Returns:
//   - bool: true if a custom MCP tool was handled, false otherwise
func HandleCustomMCPToolInSwitch(
	yaml *strings.Builder,
	toolName string,
	tools map[string]any,
	isLast bool,
	renderFunc RenderCustomMCPToolConfigHandler,
) bool {
	// Handle custom MCP tools (those with MCP-compatible type)
	if toolConfig, ok := tools[toolName].(map[string]any); ok {
		if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
			if err := renderFunc(yaml, toolName, toolConfig, isLast); err != nil {
				fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
			}
			return true
		}
	}
	return false
}

// FormatStepWithCommandAndEnv formats a GitHub Actions step with command and environment variables.
// This shared function extracts the common pattern used by Copilot and Codex engines.
//
// Parameters:
//   - stepLines: Existing step lines to append to (e.g., name, id, comments, timeout)
//   - command: The command to execute (may contain multiple lines)
//   - env: Map of environment variables to include in the step
//
// Returns:
//   - []string: Complete step lines including run command and env section
func FormatStepWithCommandAndEnv(stepLines []string, command string, env map[string]string) []string {
	// Add the run section
	stepLines = append(stepLines, "        run: |")

	// Split command into lines and indent them properly
	commandLines := strings.Split(command, "\n")
	for _, line := range commandLines {
		// Don't add indentation to empty lines
		if line == "" {
			stepLines = append(stepLines, "")
		} else {
			stepLines = append(stepLines, "          "+line)
		}
	}

	// Add environment variables
	if len(env) > 0 {
		stepLines = append(stepLines, "        env:")
		// Sort environment keys for consistent output
		envKeys := make([]string, 0, len(env))
		for key := range env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)

		for _, key := range envKeys {
			value := env[key]
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, value))
		}
	}

	return stepLines
}

// MCPToolRenderers holds engine-specific rendering functions for each MCP tool type
type MCPToolRenderers struct {
	RenderGitHub           func(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData)
	RenderPlaywright       func(yaml *strings.Builder, playwrightTool any, isLast bool)
	RenderCacheMemory      func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData)
	RenderAgenticWorkflows func(yaml *strings.Builder, isLast bool)
	RenderSafeOutputs      func(yaml *strings.Builder, isLast bool)
	RenderWebFetch         func(yaml *strings.Builder, isLast bool)
	RenderCustomMCPConfig  RenderCustomMCPToolConfigHandler
}

// JSONMCPConfigOptions defines configuration for JSON-based MCP config rendering
type JSONMCPConfigOptions struct {
	// ConfigPath is the file path for the MCP config (e.g., "/tmp/gh-aw/mcp-config/mcp-servers.json")
	ConfigPath string
	// Renderers contains engine-specific rendering functions for each tool
	Renderers MCPToolRenderers
	// FilterTool is an optional function to filter out tools before processing
	// Returns true if the tool should be included, false to skip it
	FilterTool func(toolName string) bool
	// PostEOFCommands is an optional function to add commands after the EOF (e.g., debug output)
	PostEOFCommands func(yaml *strings.Builder)
}

// GitHubMCPDockerOptions defines configuration for GitHub MCP Docker rendering
type GitHubMCPDockerOptions struct {
	// ReadOnly enables read-only mode for GitHub API operations
	ReadOnly bool
	// Toolsets specifies the GitHub toolsets to enable
	Toolsets string
	// DockerImageVersion specifies the GitHub MCP server Docker image version
	DockerImageVersion string
	// CustomArgs are additional arguments to append to the Docker command
	CustomArgs []string
	// IncludeTypeField indicates whether to include the "type": "local" field (Copilot needs it, Claude doesn't)
	IncludeTypeField bool
	// AllowedTools specifies the list of allowed tools (Copilot uses this, Claude doesn't)
	AllowedTools []string
	// EffectiveToken is the GitHub token to use (Claude uses this, Copilot uses env passthrough)
	EffectiveToken string
}

// RenderGitHubMCPDockerConfig renders the GitHub MCP server configuration for Docker (local mode).
// This shared function extracts the duplicate pattern from Claude and Copilot engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - options: GitHub MCP Docker rendering options
func RenderGitHubMCPDockerConfig(yaml *strings.Builder, options GitHubMCPDockerOptions) {
	// Add type field if needed (Copilot requires this, Claude doesn't)
	if options.IncludeTypeField {
		yaml.WriteString("                \"type\": \"local\",\n")
	}

	yaml.WriteString("                \"command\": \"docker\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"run\",\n")
	yaml.WriteString("                  \"-i\",\n")
	yaml.WriteString("                  \"--rm\",\n")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")

	if options.ReadOnly {
		yaml.WriteString("                  \"-e\",\n")
		yaml.WriteString("                  \"GITHUB_READ_ONLY=1\",\n")
	}

	// Add GITHUB_TOOLSETS environment variable (always configured, defaults to "default")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString(fmt.Sprintf("                  \"GITHUB_TOOLSETS=%s\",\n", options.Toolsets))

	yaml.WriteString("                  \"ghcr.io/github/github-mcp-server:" + options.DockerImageVersion + "\"")

	// Append custom args if present
	writeArgsToYAML(yaml, options.CustomArgs, "                  ")

	yaml.WriteString("\n")
	yaml.WriteString("                ],\n")

	// Add tools field if provided (Copilot uses this, Claude doesn't)
	if len(options.AllowedTools) > 0 {
		yaml.WriteString("                \"tools\": [\n")
		for i, tool := range options.AllowedTools {
			comma := ","
			if i == len(options.AllowedTools)-1 {
				comma = ""
			}
			fmt.Fprintf(yaml, "                  \"%s\"%s\n", tool, comma)
		}
		yaml.WriteString("                ],\n")
	} else if options.IncludeTypeField {
		// Copilot always includes tools field, even if empty (uses wildcard)
		yaml.WriteString("                \"tools\": [\"*\"],\n")
	}

	// Add env section
	yaml.WriteString("                \"env\": {\n")
	if options.EffectiveToken != "" {
		// Claude uses effective token directly
		yaml.WriteString(fmt.Sprintf("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"%s\"", options.EffectiveToken))
	} else {
		// Copilot uses env passthrough
		yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"\\${GITHUB_MCP_SERVER_TOKEN}\"")
	}
	yaml.WriteString("\n")
	yaml.WriteString("                }\n")
}

// GitHubMCPRemoteOptions defines configuration for GitHub MCP remote mode rendering
type GitHubMCPRemoteOptions struct {
	// ReadOnly enables read-only mode for GitHub API operations
	ReadOnly bool
	// Toolsets specifies the GitHub toolsets to enable
	Toolsets string
	// AuthorizationValue is the value for the Authorization header
	// For Claude: "Bearer {effectiveToken}"
	// For Copilot: "Bearer \\${GITHUB_PERSONAL_ACCESS_TOKEN}"
	AuthorizationValue string
	// IncludeToolsField indicates whether to include the "tools" field (Copilot needs it, Claude doesn't)
	IncludeToolsField bool
	// AllowedTools specifies the list of allowed tools (Copilot uses this, Claude doesn't)
	AllowedTools []string
	// IncludeEnvSection indicates whether to include the env section (Copilot needs it, Claude doesn't)
	IncludeEnvSection bool
}

// RenderGitHubMCPRemoteConfig renders the GitHub MCP server configuration for remote (hosted) mode.
// This shared function extracts the duplicate pattern from Claude and Copilot engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - options: GitHub MCP remote rendering options
func RenderGitHubMCPRemoteConfig(yaml *strings.Builder, options GitHubMCPRemoteOptions) {
	// Remote mode - use hosted GitHub MCP server
	yaml.WriteString("                \"type\": \"http\",\n")
	yaml.WriteString("                \"url\": \"https://api.githubcopilot.com/mcp/\",\n")
	yaml.WriteString("                \"headers\": {\n")

	// Collect headers in a map
	headers := make(map[string]string)
	headers["Authorization"] = options.AuthorizationValue

	// Add X-MCP-Readonly header if read-only mode is enabled
	if options.ReadOnly {
		headers["X-MCP-Readonly"] = "true"
	}

	// Add X-MCP-Toolsets header if toolsets are configured
	if options.Toolsets != "" {
		headers["X-MCP-Toolsets"] = options.Toolsets
	}

	// Write headers using helper
	writeHeadersToYAML(yaml, headers, "                  ")

	// Close headers section
	if options.IncludeToolsField || options.IncludeEnvSection {
		yaml.WriteString("                },\n")
	} else {
		yaml.WriteString("                }\n")
	}

	// Add tools field if needed (Copilot uses this, Claude doesn't)
	if options.IncludeToolsField {
		if len(options.AllowedTools) > 0 {
			yaml.WriteString("                \"tools\": [\n")
			for i, tool := range options.AllowedTools {
				comma := ","
				if i == len(options.AllowedTools)-1 {
					comma = ""
				}
				fmt.Fprintf(yaml, "                  \"%s\"%s\n", tool, comma)
			}
			yaml.WriteString("                ],\n")
		} else {
			yaml.WriteString("                \"tools\": [\"*\"],\n")
		}
	}

	// Add env section if needed (Copilot uses this, Claude doesn't)
	if options.IncludeEnvSection {
		yaml.WriteString("                \"env\": {\n")
		yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"\\${GITHUB_MCP_SERVER_TOKEN}\"\n")
		yaml.WriteString("                }\n")
	}
}



// BuildMCPConfigJSON generates MCP configuration as a compact JSON string.
// This function reuses the existing renderers but captures the JSON content
// instead of writing it to a heredoc file.
//
// Parameters:
//   - tools: Map of tool configurations
//   - mcpTools: Ordered list of MCP tool names to render
//   - workflowData: Workflow configuration data  
//   - options: JSON MCP config rendering options
//
// Returns:
//   - string: The MCP config as a compact JSON string (single line, minimized)
//   - error: Error if JSON generation or validation fails
func BuildMCPConfigJSON(
	tools map[string]any,
	mcpTools []string,
	workflowData *WorkflowData,
	options JSONMCPConfigOptions,
) (string, error) {
	var json strings.Builder
	
	// Start building JSON structure (matching the indentation used in heredoc)
	// The heredoc in RenderJSONMCPConfig uses 10 spaces base + 2 for each level
	json.WriteString("          {\n")                   // Base indent (10 spaces)
	json.WriteString("            \"mcpServers\": {\n") // +2 spaces (12 total)

	// Filter tools if needed (e.g., Copilot filters out cache-memory)
	var filteredTools []string
	for _, toolName := range mcpTools {
		if options.FilterTool != nil && !options.FilterTool(toolName) {
			continue
		}
		filteredTools = append(filteredTools, toolName)
	}

	// Process each MCP tool (reusing existing renderers)
	// Renderers expect to write at 14+ spaces indentation
	totalServers := len(filteredTools)
	serverCount := 0

	for _, toolName := range filteredTools {
		serverCount++
		isLast := serverCount == totalServers

		switch toolName {
		case "github":
			githubTool := tools["github"]
			options.Renderers.RenderGitHub(&json, githubTool, isLast, workflowData)
		case "playwright":
			playwrightTool := tools["playwright"]
			options.Renderers.RenderPlaywright(&json, playwrightTool, isLast)
		case "cache-memory":
			options.Renderers.RenderCacheMemory(&json, isLast, workflowData)
		case "agentic-workflows":
			options.Renderers.RenderAgenticWorkflows(&json, isLast)
		case "safe-outputs":
			options.Renderers.RenderSafeOutputs(&json, isLast)
		case "web-fetch":
			options.Renderers.RenderWebFetch(&json, isLast)
		default:
			// Handle custom MCP tools using shared helper
			HandleCustomMCPToolInSwitch(&json, toolName, tools, isLast, options.Renderers.RenderCustomMCPConfig)
		}
	}

	// Close JSON structure
	json.WriteString("            }\n")  // Closing mcpServers (12 spaces)
	json.WriteString("          }\n")    // Closing root (10 spaces)
	
	// Validate and compact the JSON using encoding/json
	jsonStr := json.String()
	return validateAndCompactJSON(jsonStr)
}

// validateAndCompactJSON compacts JSON string for CLI argument passing.
// Note: This does NOT validate the JSON because it may contain GitHub Actions expressions
// like ${{ env.VAR }} which are not valid JSON until evaluated by GitHub Actions.
// The JSON structure is validated by tests that check the generated workflow output.
func validateAndCompactJSON(jsonStr string) (string, error) {
	// Compact the JSON by removing unnecessary whitespace
	// We can't use json.Marshal because the content contains GitHub Actions expressions
	
	var compacted strings.Builder
	var inString bool
	var escape bool
	
	for i := 0; i < len(jsonStr); i++ {
		ch := jsonStr[i]
		
		// Track string state to preserve spaces inside strings
		if ch == '"' && !escape {
			inString = !inString
			compacted.WriteByte(ch)
			continue
		}
		
		// Track escape sequences
		if ch == '\\' && !escape {
			escape = true
			compacted.WriteByte(ch)
			continue
		}
		
		if escape {
			escape = false
			compacted.WriteByte(ch)
			continue
		}
		
		// Skip whitespace outside of strings
		if !inString && (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') {
			continue
		}
		
		compacted.WriteByte(ch)
	}
	
	return compacted.String(), nil
}

// RenderJSONMCPConfig renders MCP configuration in JSON format with the common mcpServers structure.
// This shared function extracts the duplicate pattern from Claude, Copilot, and Custom engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - tools: Map of tool configurations
//   - mcpTools: Ordered list of MCP tool names to render
//   - workflowData: Workflow configuration data
//   - options: JSON MCP config rendering options
func RenderJSONMCPConfig(
	yaml *strings.Builder,
	tools map[string]any,
	mcpTools []string,
	workflowData *WorkflowData,
	options JSONMCPConfigOptions,
) {
	// Write config file header
	yaml.WriteString(fmt.Sprintf("          cat > %s << EOF\n", options.ConfigPath))
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"mcpServers\": {\n")

	// Filter tools if needed (e.g., Copilot filters out cache-memory)
	var filteredTools []string
	for _, toolName := range mcpTools {
		if options.FilterTool != nil && !options.FilterTool(toolName) {
			continue
		}
		filteredTools = append(filteredTools, toolName)
	}

	// Process each MCP tool
	totalServers := len(filteredTools)
	serverCount := 0

	for _, toolName := range filteredTools {
		serverCount++
		isLast := serverCount == totalServers

		switch toolName {
		case "github":
			githubTool := tools["github"]
			options.Renderers.RenderGitHub(yaml, githubTool, isLast, workflowData)
		case "playwright":
			playwrightTool := tools["playwright"]
			options.Renderers.RenderPlaywright(yaml, playwrightTool, isLast)
		case "cache-memory":
			options.Renderers.RenderCacheMemory(yaml, isLast, workflowData)
		case "agentic-workflows":
			options.Renderers.RenderAgenticWorkflows(yaml, isLast)
		case "safe-outputs":
			options.Renderers.RenderSafeOutputs(yaml, isLast)
		case "web-fetch":
			options.Renderers.RenderWebFetch(yaml, isLast)
		default:
			// Handle custom MCP tools using shared helper
			HandleCustomMCPToolInSwitch(yaml, toolName, tools, isLast, options.Renderers.RenderCustomMCPConfig)
		}
	}

	// Write config file footer
	yaml.WriteString("            }\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          EOF\n")

	// Add any post-EOF commands (e.g., debug output for Copilot)
	if options.PostEOFCommands != nil {
		options.PostEOFCommands(yaml)
	}
}
