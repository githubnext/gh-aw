package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const tempFolder = "/tmp/.copilot/"
const logsFolder = tempFolder + "logs/"

// CopilotEngine represents the GitHub Copilot CLI agentic engine
type CopilotEngine struct {
	BaseEngine
}

// CopilotMCPConfig represents the top-level MCP configuration for Copilot CLI
type CopilotMCPConfig struct {
	MCPServers map[string]CopilotMCPServer `json:"mcpServers"`
}

// CopilotMCPServer represents a single MCP server configuration for Copilot CLI
type CopilotMCPServer struct {
	Type    string                 `json:"type"`
	Command string                 `json:"command,omitempty"`
	Args    []string               `json:"args,omitempty"`
	Env     map[string]interface{} `json:"env,omitempty"`
	URL     string                 `json:"url,omitempty"`
	Headers map[string]string      `json:"headers,omitempty"`
	Tools   []string               `json:"tools,omitempty"`
}

func NewCopilotEngine() *CopilotEngine {
	return &CopilotEngine{
		BaseEngine: BaseEngine{
			id:                     "copilot",
			displayName:            "GitHub Copilot CLI",
			description:            "Uses GitHub Copilot CLI with MCP server support",
			experimental:           true,
			supportsToolsAllowlist: true,
			supportsHTTPTransport:  true,  // Copilot CLI supports HTTP transport via MCP
			supportsMaxTurns:       false, // Copilot CLI does not support max-turns feature yet
		},
	}
}

func (e *CopilotEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	// Build the npm install command, optionally with version
	installCmd := "npm install -g @github/copilot"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		installCmd = fmt.Sprintf("npm install -g @github/copilot@%s", workflowData.EngineConfig.Version)
	}

	var steps []GitHubActionStep

	installationSteps := []GitHubActionStep{
		{
			"      - name: Setup Node.js",
			"        uses: actions/setup-node@v4",
			"        with:",
			"          node-version: '22'",
		},
		{
			"      - name: Install GitHub Copilot CLI",
			fmt.Sprintf("        run: %s", installCmd),
		},
		{
			"      - name: Setup Copilot CLI MCP Configuration",
			"        run: |",
			"          mkdir -p /tmp/.copilot",
		},
	}

	steps = append(steps, installationSteps...)
	return steps
}

func (e *CopilotEngine) GetDeclaredOutputFiles() []string {
	return []string{logsFolder}
}

// GetExecutionSteps returns the GitHub Actions steps for executing GitHub Copilot CLI
func (e *CopilotEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	var steps []GitHubActionStep

	// Handle custom steps if they exist in engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		for _, step := range workflowData.EngineConfig.Steps {
			stepYAML, err := e.convertStepToYAML(step)
			if err != nil {
				// Log error but continue with other steps
				continue
			}
			steps = append(steps, GitHubActionStep{stepYAML})
		}
	}

	// Build copilot CLI arguments based on configuration
	var copilotArgs = []string{"--add-dir", "/tmp/", "--log-level", "debug", "--log-dir", logsFolder}

	// Add model if specified (check if Copilot CLI supports this)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		copilotArgs = append(copilotArgs, "--model", workflowData.EngineConfig.Model)
	}

	// Add tool permission arguments based on configuration
	toolArgs := e.computeCopilotToolArguments(workflowData.Tools, workflowData.SafeOutputs)
	copilotArgs = append(copilotArgs, toolArgs...)

	// if cache-memory tool is used, --add-dir
	if workflowData.CacheMemoryConfig != nil {
		copilotArgs = append(copilotArgs, "--add-dir", "/tmp/cache-memory/")
	}

	copilotArgs = append(copilotArgs, "--prompt", "\"$INSTRUCTION\"")
	command := fmt.Sprintf(`set -o pipefail

INSTRUCTION=$(cat /tmp/aw-prompts/prompt.txt)

# Run copilot CLI with log capture
copilot %s 2>&1 | tee %s`, strings.Join(copilotArgs, " "), logFile)

	env := map[string]string{
		"XDG_CONFIG_HOME":     tempFolder, // copilot help environment
		"XDG_STATE_HOME":      tempFolder, // copilot cache environment
		"GITHUB_TOKEN":        "${{ secrets.COPILOT_CLI_TOKEN  }}",
		"GITHUB_STEP_SUMMARY": "${{ env.GITHUB_STEP_SUMMARY }}",
	}

	// Add GITHUB_AW_SAFE_OUTPUTS if output is needed
	hasOutput := workflowData.SafeOutputs != nil
	if hasOutput {
		env["GITHUB_AW_SAFE_OUTPUTS"] = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}"
	}

	// Add custom environment variables from engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			env[key] = value
		}
	}

	// Add upload config step before running copilot CLI
	uploadConfigStep := e.generateUploadConfigStep()
	steps = append(steps, uploadConfigStep)

	// Generate the step for Copilot CLI execution
	stepName := "Execute GitHub Copilot CLI"
	var stepLines []string

	stepLines = append(stepLines, fmt.Sprintf("      - name: %s", stepName))
	stepLines = append(stepLines, "        id: agentic_execution")

	// Add tool arguments comment before the run section
	toolArgsComment := e.generateCopilotToolArgumentsComment(workflowData.Tools, workflowData.SafeOutputs, "        ")
	if toolArgsComment != "" {
		// Split the comment into lines and add each line
		commentLines := strings.Split(strings.TrimSuffix(toolArgsComment, "\n"), "\n")
		stepLines = append(stepLines, commentLines...)
	}

	// Add timeout at step level (GitHub Actions standard)
	if workflowData.TimeoutMinutes != "" {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %s", strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout_minutes: ")))
	} else {
		stepLines = append(stepLines, "        timeout-minutes: 5") // Default timeout
	}

	stepLines = append(stepLines, "        run: |")

	// Split command into lines and indent them properly
	commandLines := strings.Split(command, "\n")
	for _, line := range commandLines {
		stepLines = append(stepLines, "          "+line)
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

	steps = append(steps, GitHubActionStep(stepLines))

	// Add the log capture step using shared helper function
	steps = append(steps, generateLogCaptureStep("Copilot", logFile))

	return steps
}

// convertStepToYAML converts a step map to YAML string - uses proper YAML serialization
func (e *CopilotEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}

// generateUploadConfigStep generates a step to upload the XDG_CONFIG_HOME folder content
func (e *CopilotEngine) generateUploadConfigStep() GitHubActionStep {
	var stepLines []string

	stepLines = append(stepLines, "      - name: Upload config")
	stepLines = append(stepLines, "        if: always()")
	stepLines = append(stepLines, "        uses: actions/upload-artifact@v4")
	stepLines = append(stepLines, "        with:")
	stepLines = append(stepLines, "          name: config")
	stepLines = append(stepLines, "          path: /tmp/.copilot/")
	stepLines = append(stepLines, "          if-no-files-found: ignore")

	return GitHubActionStep(stepLines)
}

func (e *CopilotEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	// Build the MCP configuration structure
	config := CopilotMCPConfig{
		MCPServers: make(map[string]CopilotMCPServer),
	}

	// Generate configuration for each MCP tool
	for _, toolName := range mcpTools {
		var server CopilotMCPServer
		var serverName string
		var err error

		switch toolName {
		case "github":
			// GitHub MCP is built-in to Copilot CLI, so skip adding it to configuration
			continue
		case "playwright":
			playwrightTool := tools["playwright"]
			server = e.buildPlaywrightCopilotMCPServer(playwrightTool, workflowData.NetworkPermissions)
			serverName = toolName
		case "cache-memory":
			// Cache-memory is handled as a simple file share, not an MCP server
			// Skip adding it to the MCP configuration since no server is needed
			continue
		case "safe-outputs":
			server = e.buildSafeOutputsCopilotMCPServer()
			serverName = "safe_outputs"
		default:
			// Handle custom MCP tools (those with MCP-compatible type)
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					server, err = e.buildCopilotMCPServer(toolName, toolConfig)
					if err != nil {
						fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
						continue
					}
					serverName = toolName
				}
			}
		}

		config.MCPServers[serverName] = server
	}

	// Marshal to JSON
	configJSON, err := json.MarshalIndent(config, "          ", "  ")
	if err != nil {
		// Fallback to empty config if marshaling fails
		configJSON = []byte("{\n            \"mcpServers\": {}\n          }")
	}

	yaml.WriteString("          cat > /tmp/.copilot/mcp-config.json << 'EOF'\n")
	yaml.WriteString("          ")
	yaml.WriteString(string(configJSON))
	yaml.WriteString("\n          EOF\n")
}

// buildCopilotMCPServer builds custom MCP server configuration for a single tool in Copilot CLI
func (e *CopilotEngine) buildCopilotMCPServer(toolName string, toolConfig map[string]any) (CopilotMCPServer, error) {
	// Get MCP configuration using the shared logic
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		return CopilotMCPServer{}, fmt.Errorf("failed to parse MCP config for tool '%s': %w", toolName, err)
	}

	// Copilot CLI expects "local" instead of "stdio"
	serverType := mcpConfig.Type
	if serverType == "stdio" {
		serverType = "local"
	}

	server := CopilotMCPServer{
		Type: serverType,
	}

	// Set fields based on type
	switch mcpConfig.Type {
	case "stdio", "local":
		server.Command = mcpConfig.Command
		if len(mcpConfig.Args) > 0 {
			server.Args = mcpConfig.Args
		}
		if len(mcpConfig.Env) > 0 {
			server.Env = make(map[string]interface{})
			for k, v := range mcpConfig.Env {
				server.Env[k] = v
			}
		}
	case "http":
		server.URL = mcpConfig.URL
		if len(mcpConfig.Headers) > 0 {
			server.Headers = mcpConfig.Headers
		}
	}

	return server, nil
}

// buildPlaywrightCopilotMCPServer builds the Playwright MCP server configuration for Copilot CLI
// Uses npx to launch Playwright MCP instead of Docker for better performance and simplicity
func (e *CopilotEngine) buildPlaywrightCopilotMCPServer(playwrightTool any, networkPermissions *NetworkPermissions) CopilotMCPServer {
	args := generatePlaywrightDockerArgs(playwrightTool, networkPermissions)

	// Use the version from docker args (which handles docker_image_version configuration)
	playwrightPackage := "@playwright/mcp@" + args.ImageVersion

	server := CopilotMCPServer{
		Type:    "local",
		Command: "npx",
		Args:    []string{playwrightPackage},
	}

	if len(args.AllowedDomains) > 0 {
		server.Args = append(server.Args, "--allowed-origins", strings.Join(args.AllowedDomains, ";"))
	}

	return server
}

// buildSafeOutputsCopilotMCPServer builds the Safe Outputs MCP server configuration for Copilot CLI
func (e *CopilotEngine) buildSafeOutputsCopilotMCPServer() CopilotMCPServer {
	return CopilotMCPServer{
		Type:    "local",
		Command: "node",
		Args:    []string{"/tmp/safe-outputs/mcp-server.cjs"},
		Env: map[string]interface{}{
			"GITHUB_AW_SAFE_OUTPUTS":        "${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
			"GITHUB_AW_SAFE_OUTPUTS_CONFIG": "${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }}",
		},
	}
}

// ParseLogMetrics implements engine-specific log parsing for Copilot CLI
func (e *CopilotEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics
	var maxTokenUsage int

	lines := strings.Split(logContent, "\n")
	toolCallMap := make(map[string]*ToolCallInfo) // Track tool calls
	var currentSequence []string                  // Track tool sequence
	turns := 0
	executionStarted := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Skip empty lines
		if trimmedLine == "" {
			continue
		}

		// Detect execution start and count turns
		if strings.Contains(line, "Processing user prompt") || strings.Contains(line, "Starting GitHub Copilot CLI") {
			if executionStarted {
				turns++
			} else {
				executionStarted = true
				turns = 1
			}
			// Start of a new turn, save previous sequence if any
			if len(currentSequence) > 0 {
				metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
				currentSequence = []string{}
			}
		}

		// Count additional turns based on suggestion/response patterns
		if strings.Contains(line, "Suggestion:") || strings.Contains(line, "Response:") {
			turns++
			if len(currentSequence) > 0 {
				metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
				currentSequence = []string{}
			}
		}

		// Extract tool calls and add to sequence
		if toolName := e.parseCopilotToolCallsWithSequence(line, toolCallMap); toolName != "" {
			currentSequence = append(currentSequence, toolName)
		}

		// Extract execution time
		if strings.Contains(line, "Total execution time:") {
			// Parse execution time for potential cost estimation
			if timeMatch := regexp.MustCompile(`Total execution time:\s*([\d.]+)\s*seconds`).FindStringSubmatch(line); len(timeMatch) > 1 {
				if executionSeconds, err := strconv.ParseFloat(timeMatch[1], 64); err == nil {
					// Simple cost estimation based on execution time (placeholder)
					// This would need to be refined based on actual Copilot CLI pricing
					metrics.EstimatedCost += executionSeconds * 0.001 // $0.001 per second as placeholder
				}
			}
		}

		// Try to extract token usage from JSON format if available
		jsonMetrics := ExtractJSONMetrics(line, verbose)
		if jsonMetrics.TokenUsage > 0 || jsonMetrics.EstimatedCost > 0 {
			if jsonMetrics.TokenUsage > maxTokenUsage {
				maxTokenUsage = jsonMetrics.TokenUsage
			}
			if jsonMetrics.EstimatedCost > 0 {
				metrics.EstimatedCost += jsonMetrics.EstimatedCost
			}
		}

		// Count errors with more specific patterns
		if strings.Contains(line, "[ERROR]") || 
		   strings.Contains(line, "copilot: error:") ||
		   strings.Contains(line, "Fatal error:") ||
		   strings.Contains(line, "npm ERR!") ||
		   strings.Contains(line, "Shell command failed:") {
			metrics.ErrorCount++
		}

		// Count warnings with more specific patterns
		if strings.Contains(line, "[WARNING]") || 
		   strings.Contains(line, "[WARN]") ||
		   strings.Contains(line, "Warning:") ||
		   strings.Contains(line, "MCP server connection timeout") {
			metrics.WarningCount++
		}
	}

	// Add final sequence if any
	if len(currentSequence) > 0 {
		metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
	}

	// Ensure we have at least 1 turn if we detected execution
	if executionStarted && turns == 0 {
		turns = 1
	}

	metrics.TokenUsage = maxTokenUsage
	metrics.Turns = turns

	// Convert tool call map to slice
	for _, toolInfo := range toolCallMap {
		metrics.ToolCalls = append(metrics.ToolCalls, *toolInfo)
	}

	// Sort tool calls by name for consistent output
	sort.Slice(metrics.ToolCalls, func(i, j int) bool {
		return metrics.ToolCalls[i].Name < metrics.ToolCalls[j].Name
	})

	return metrics
}

// parseCopilotToolCallsWithSequence extracts tool call information from Copilot CLI log lines and returns tool name
func (e *CopilotEngine) parseCopilotToolCallsWithSequence(line string, toolCallMap map[string]*ToolCallInfo) string {
	// This method needs to be adjusted based on actual Copilot CLI output format
	// For now, using a generic approach that can be refined once we see actual logs

	// Look for common tool call patterns (adjust based on actual Copilot CLI output)
	if strings.Contains(line, "calling") || strings.Contains(line, "tool:") || strings.Contains(line, "function:") {
		// Extract tool name from various possible formats
		toolName := ""
		if strings.Contains(line, "github") {
			toolName = "github"
		} else if strings.Contains(line, "playwright") {
			toolName = "playwright"
		} else if strings.Contains(line, "safe") && strings.Contains(line, "output") {
			toolName = "safe_outputs"
		}

		if toolName != "" {
			// Initialize or update tool call info
			if toolInfo, exists := toolCallMap[toolName]; exists {
				toolInfo.CallCount++
			} else {
				toolCallMap[toolName] = &ToolCallInfo{
					Name:          toolName,
					CallCount:     1,
					MaxOutputSize: 0, // TODO: Extract output size from results if available
				}
			}
			return toolName
		}
	}

	return ""
}

// GetLogParserScript returns the JavaScript script name for parsing Copilot logs
func (e *CopilotEngine) GetLogParserScriptId() string {
	return "parse_copilot_log"
}

// computeCopilotToolArguments generates Copilot CLI tool permission arguments from workflow tools configuration
func (e *CopilotEngine) computeCopilotToolArguments(tools map[string]any, safeOutputs *SafeOutputsConfig) []string {
	if tools == nil {
		tools = make(map[string]any)
	}

	var args []string

	// Handle bash/shell tools
	if bashConfig, hasBash := tools["bash"]; hasBash {
		hasWildcard := false
		if bashCommands, ok := bashConfig.([]any); ok {
			// Check for :* wildcard first - if present, allow all shell commands
			for _, cmd := range bashCommands {
				if cmdStr, ok := cmd.(string); ok {
					if cmdStr == ":*" || cmdStr == "*" {
						// Allow all shell commands
						args = append(args, "--allow-tool", "shell")
						hasWildcard = true
						break
					}
				}
			}
			// Add specific shell commands only if no wildcard found
			if !hasWildcard {
				for _, cmd := range bashCommands {
					if cmdStr, ok := cmd.(string); ok {
						args = append(args, "--allow-tool", fmt.Sprintf("shell(%s)", cmdStr))
					}
				}
			}
		} else {
			// Bash with no specific commands or null value - allow all shell
			args = append(args, "--allow-tool", "shell")
		}
	}

	// Handle edit tools requirement for file write access
	// Note: safe-outputs do not need write permission as they use MCP
	if _, hasEdit := tools["edit"]; hasEdit {
		args = append(args, "--allow-tool", "write")
	}

	// Handle safe_outputs MCP server - allow all tools if safe outputs are enabled
	// This includes both safeOutputs config and safeOutputs.Jobs
	if HasSafeOutputsEnabled(safeOutputs) {
		args = append(args, "--allow-tool", "safe_outputs")
	}

	// Built-in tool names that should be skipped when processing MCP servers
	builtInTools := map[string]bool{
		"bash":       true,
		"edit":       true,
		"web-fetch":  true,
		"web-search": true,
		"playwright": true,
		"github":     true,
	}

	// Handle MCP server tools
	for toolName, toolConfig := range tools {
		// Skip built-in tools we've already handled
		if builtInTools[toolName] {
			continue
		}

		// Check if this is an MCP server configuration
		if toolConfigMap, ok := toolConfig.(map[string]any); ok {
			if hasMcp, _ := hasMCPConfig(toolConfigMap); hasMcp {
				// Allow the entire MCP server
				args = append(args, "--allow-tool", toolName)

				// If it has specific allowed tools, add them individually
				if allowed, hasAllowed := toolConfigMap["allowed"]; hasAllowed {
					if allowedList, ok := allowed.([]any); ok {
						for _, allowedTool := range allowedList {
							if toolStr, ok := allowedTool.(string); ok {
								args = append(args, "--allow-tool", fmt.Sprintf("%s(%s)", toolName, toolStr))
							}
						}
					}
				}
			}
		}
	}

	// Simple sort - extract values, sort them, and rebuild args
	if len(args) > 0 {
		var values []string
		for i := 1; i < len(args); i += 2 {
			values = append(values, args[i])
		}
		sort.Strings(values)

		// Rebuild args with sorted values
		newArgs := make([]string, 0, len(args))
		for _, value := range values {
			newArgs = append(newArgs, "--allow-tool", value)
		}
		args = newArgs
	}

	return args
}

// generateCopilotToolArgumentsComment generates a multi-line comment showing each tool argument
func (e *CopilotEngine) generateCopilotToolArgumentsComment(tools map[string]any, safeOutputs *SafeOutputsConfig, indent string) string {
	toolArgs := e.computeCopilotToolArguments(tools, safeOutputs)
	if len(toolArgs) == 0 {
		return ""
	}

	var comment strings.Builder
	comment.WriteString(indent + "# Copilot CLI tool arguments (sorted):\n")

	// Group flag-value pairs for better readability
	for i := 0; i < len(toolArgs); i += 2 {
		if i+1 < len(toolArgs) {
			comment.WriteString(fmt.Sprintf("%s# %s %s\n", indent, toolArgs[i], toolArgs[i+1]))
		}
	}

	return comment.String()
}

// GetErrorPatterns returns regex patterns for extracting error messages from Copilot CLI logs
func (e *CopilotEngine) GetErrorPatterns() []ErrorPattern {
	return []ErrorPattern{
		{
			Pattern:      `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[(ERROR)\]\s+(.+)`,
			LevelGroup:   2, // "ERROR" is in the second capture group
			MessageGroup: 3, // error message is in the third capture group
			Description:  "Copilot CLI timestamped ERROR messages",
		},
		{
			Pattern:      `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[(WARN|WARNING)\]\s+(.+)`,
			LevelGroup:   2, // "WARN" or "WARNING" is in the second capture group
			MessageGroup: 3, // warning message is in the third capture group
			Description:  "Copilot CLI timestamped WARNING messages",
		},
		{
			Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\]\s+(CRITICAL|ERROR):\s+(.+)`,
			LevelGroup:   2, // "CRITICAL" or "ERROR" is in the second capture group
			MessageGroup: 3, // error message is in the third capture group
			Description:  "Copilot CLI bracketed critical/error messages with timestamp",
		},
		{
			Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\]\s+(WARNING):\s+(.+)`,
			LevelGroup:   2, // "WARNING" is in the second capture group
			MessageGroup: 3, // warning message is in the third capture group
			Description:  "Copilot CLI bracketed warning messages with timestamp",
		},
		{
			Pattern:      `(Error):\s+(.+)`,
			LevelGroup:   1, // "Error" is in the first capture group
			MessageGroup: 2, // error message is in the second capture group
			Description:  "Generic error messages from Copilot CLI or Node.js",
		},
		{
			Pattern:      `npm ERR!\s+(.+)`,
			LevelGroup:   0, // No level group, will be inferred as "error"
			MessageGroup: 1, // error message is in the first capture group
			Description:  "NPM error messages during Copilot CLI installation or execution",
		},
		{
			Pattern:      `(Warning):\s+(.+)`,
			LevelGroup:   1, // "Warning" is in the first capture group
			MessageGroup: 2, // warning message is in the second capture group
			Description:  "Generic warning messages from Copilot CLI",
		},
		{
			Pattern:      `(Fatal error):\s+(.+)`,
			LevelGroup:   1, // "Fatal error" is in the first capture group (will be treated as error)
			MessageGroup: 2, // error message is in the second capture group
			Description:  "Fatal error messages from Copilot CLI",
		},
		{
			Pattern:      `copilot:\s+(error):\s+(.+)`,
			LevelGroup:   1, // "error" is in the first capture group
			MessageGroup: 2, // error message is in the second capture group
			Description:  "Copilot CLI command-level error messages",
		},
		{
			Pattern:      `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[ERROR\]\s+Shell command failed:\s*(.+)`,
			LevelGroup:   0, // No level group, will be inferred as "error" 
			MessageGroup: 2, // error message is in the second capture group
			Description:  "Copilot CLI shell command execution errors",
		},
		{
			Pattern:      `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[ERROR\]\s+Failed to connect to\s+(.+)\s+MCP server`,
			LevelGroup:   0, // No level group, will be inferred as "error"
			MessageGroup: 2, // server name is in the second capture group
			Description:  "Copilot CLI MCP server connection errors",
		},
	}
}
