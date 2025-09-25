package workflow

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// CopilotEngine represents the GitHub Copilot CLI agentic engine
type CopilotEngine struct {
	BaseEngine
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

	// TODO: handle logs folder
	logDir := filepath.Dir(logFile)
	// Build copilot CLI arguments based on configuration
	var copilotArgs []string = []string{"--log-level", "debug", "--log-dir", logDir}

	// Add model if specified (check if Copilot CLI supports this)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		copilotArgs = append(copilotArgs, "--model", workflowData.EngineConfig.Model)
	}

	copilotArgs = append(copilotArgs, "--prompt", "\"$INSTRUCTION\"")
	command := fmt.Sprintf(`INSTRUCTION=$(cat /tmp/aw-prompts/prompt.txt)

# Run copilot CLI with log capture
copilot %s 2>&1 | tee %s`, strings.Join(copilotArgs, " "), logFile)

	env := map[string]string{
		"GITHUB_TOKEN":        "${{ secrets.GITHUB_COPILOT_CLI_TOKEN }}",
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
	yaml.WriteString("          cat > /tmp/.copilot/mcpconfig << 'EOF'\n")
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"mcpServers\": {\n")

	// Add safe-outputs MCP server if safe-outputs are configured
	totalServers := len(mcpTools)
	serverCount := 0

	// Generate configuration for each MCP tool using shared logic
	for _, toolName := range mcpTools {
		serverCount++
		isLast := serverCount == totalServers

		switch toolName {
		case "github":
			githubTool := tools["github"]
			e.renderGitHubCopilotMCPConfig(yaml, githubTool, isLast)
		case "playwright":
			playwrightTool := tools["playwright"]
			e.renderPlaywrightCopilotMCPConfig(yaml, playwrightTool, isLast, workflowData.NetworkPermissions)
		case "cache-memory":
			e.renderCacheMemoryCopilotMCPConfig(yaml, isLast, workflowData)
		case "safe-outputs":
			e.renderSafeOutputsCopilotMCPConfig(yaml, isLast)
		default:
			// Handle custom MCP tools (those with MCP-compatible type)
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					if err := e.renderCopilotMCPConfig(yaml, toolName, toolConfig, isLast); err != nil {
						fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
					}
				}
			}
		}
	}

	yaml.WriteString("            }\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          EOF\n")
}

// renderGitHubCopilotMCPConfig generates the GitHub MCP server configuration for Copilot CLI
func (e *CopilotEngine) renderGitHubCopilotMCPConfig(yaml *strings.Builder, githubTool any, isLast bool) {
	yaml.WriteString("              \"GitHub\": {\n")
	yaml.WriteString("                \"type\": \"http\",\n")
	yaml.WriteString("                \"url\": \"https://api.githubcopilot.com/mcp\",\n")
	yaml.WriteString("                \"headers\": {},\n")
	yaml.WriteString("                \"tools\": [\n")
	yaml.WriteString("                  \"*\"\n")
	yaml.WriteString("                ]\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderCopilotMCPConfig generates custom MCP server configuration for a single tool in Copilot CLI mcpconfig
func (e *CopilotEngine) renderCopilotMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	yaml.WriteString(fmt.Sprintf("              \"%s\": {\n", toolName))

	// Use the shared MCP config renderer with JSON format
	renderer := MCPConfigRenderer{
		IndentLevel: "                ",
		Format:      "json",
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer)
	if err != nil {
		return err
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
}

// renderPlaywrightCopilotMCPConfig generates the Playwright MCP server configuration for Copilot CLI
// Uses npx to launch Playwright MCP instead of Docker for better performance and simplicity
func (e *CopilotEngine) renderPlaywrightCopilotMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool, networkPermissions *NetworkPermissions) {
	args := generatePlaywrightDockerArgs(playwrightTool, networkPermissions)

	yaml.WriteString("              \"playwright\": {\n")
	yaml.WriteString("                \"command\": \"npx\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"@playwright/mcp@latest\",\n")
	if len(args.AllowedDomains) > 0 {
		yaml.WriteString("                  \"--allowed-origins\",\n")
		yaml.WriteString("                  \"" + strings.Join(args.AllowedDomains, ",") + "\"\n")
	}
	yaml.WriteString("                ]\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderCacheMemoryCopilotMCPConfig handles cache-memory configuration without MCP server mounting
// Cache-memory is now a simple file share, not an MCP server
func (e *CopilotEngine) renderCacheMemoryCopilotMCPConfig(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
	// Cache-memory no longer uses MCP server mounting
	// The cache folder is available as a simple file share at /tmp/cache-memory/
	// The folder is created by the cache step and is accessible to all tools
	// No MCP configuration is needed for simple file access
}

// renderSafeOutputsCopilotMCPConfig generates the Safe Outputs MCP server configuration for Copilot CLI
func (e *CopilotEngine) renderSafeOutputsCopilotMCPConfig(yaml *strings.Builder, isLast bool) {
	yaml.WriteString("              \"safe_outputs\": {\n")
	yaml.WriteString("                \"command\": \"node\",\n")
	yaml.WriteString("                \"args\": [\"/tmp/safe-outputs/mcp-server.cjs\"],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS\": \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS_CONFIG\": ${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }}\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
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
