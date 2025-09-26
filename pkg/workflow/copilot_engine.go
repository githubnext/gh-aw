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

	// Check if network permissions are configured (only for Copilot engine)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.ID == "copilot" && ShouldEnforceNetworkPermissions(workflowData.NetworkPermissions) {
		// Generate network hook generator and settings generator
		hookGenerator := &NetworkHookGenerator{}
		settingsGenerator := &ClaudeSettingsGenerator{} // Using Claude settings generator as it's generic

		allowedDomains := GetAllowedDomains(workflowData.NetworkPermissions)

		// Add settings generation step
		settingsStep := settingsGenerator.GenerateSettingsWorkflowStep()
		steps = append(steps, settingsStep)

		// Add hook generation step
		hookStep := hookGenerator.GenerateNetworkHookWorkflowStep(allowedDomains)
		steps = append(steps, hookStep)
	}

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

	// Generate the step for Copilot CLI execution
	stepName := "Execute GitHub Copilot CLI"
	var stepLines []string

	stepLines = append(stepLines, fmt.Sprintf("      - name: %s", stepName))
	stepLines = append(stepLines, "        id: agentic_execution")

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

	return steps
}

// convertStepToYAML converts a step map to YAML string - uses proper YAML serialization
func (e *CopilotEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
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
			githubTool := tools["github"]
			server = e.buildGitHubCopilotMCPServer(githubTool)
			serverName = "GitHub" // Copilot CLI expects "GitHub" (capital G)
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

// buildGitHubCopilotMCPServer builds the GitHub MCP server configuration for Copilot CLI
func (e *CopilotEngine) buildGitHubCopilotMCPServer(githubTool any) CopilotMCPServer {
	return CopilotMCPServer{
		Type:  "http",
		URL:   "https://api.githubcopilot.com/mcp",
		Tools: []string{"*"},
	}
}

// buildCopilotMCPServer builds custom MCP server configuration for a single tool in Copilot CLI
func (e *CopilotEngine) buildCopilotMCPServer(toolName string, toolConfig map[string]any) (CopilotMCPServer, error) {
	// Get MCP configuration using the shared logic
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		return CopilotMCPServer{}, fmt.Errorf("failed to parse MCP config for tool '%s': %w", toolName, err)
	}

	server := CopilotMCPServer{
		Type: mcpConfig.Type,
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

	server := CopilotMCPServer{
		Type:    "local",
		Command: "npx",
		Args:    []string{"@playwright/mcp@latest"},
	}

	if len(args.AllowedDomains) > 0 {
		server.Args = append(server.Args, "--allowed-origins", strings.Join(args.AllowedDomains, ";"))
	}

	return server
}

// buildCacheMemoryCopilotMCPServer handles cache-memory configuration without MCP server mounting
// Cache-memory is now a simple file share, not an MCP server
func (e *CopilotEngine) buildCacheMemoryCopilotMCPServer(workflowData *WorkflowData) CopilotMCPServer {
	// Cache-memory no longer uses MCP server mounting
	// The cache folder is available as a simple file share at /tmp/cache-memory/
	// The folder is created by the cache step and is accessible to all tools
	// No MCP configuration is needed for simple file access
	return CopilotMCPServer{}
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

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Count turns based on interaction patterns (adjust based on actual Copilot CLI output)
		if strings.Contains(line, "User:") || strings.Contains(line, "Human:") || strings.Contains(line, "Query:") {
			turns++
			// Start of a new turn, save previous sequence if any
			if len(currentSequence) > 0 {
				metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
				currentSequence = []string{}
			}
		}

		// Extract tool calls and add to sequence (adjust based on actual Copilot CLI output format)
		if toolName := e.parseCopilotToolCallsWithSequence(line, toolCallMap); toolName != "" {
			currentSequence = append(currentSequence, toolName)
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

		// Count errors and warnings
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") {
			metrics.ErrorCount++
		}
		if strings.Contains(lowerLine, "warning") {
			metrics.WarningCount++
		}
	}

	// Add final sequence if any
	if len(currentSequence) > 0 {
		metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
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
