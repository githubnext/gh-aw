package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const logsFolder = "/tmp/.copilot/logs/"

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
	}

	steps = append(steps, installationSteps...)
	return steps
}

func (e *CopilotEngine) GetDeclaredOutputFiles() []string {
	return []string{logsFolder}
}

// GetVersionCommand returns the command to get Copilot CLI's version
func (e *CopilotEngine) GetVersionCommand() string {
	return "copilot --version"
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
	var copilotArgs = []string{"--add-dir", "/tmp/", "--log-level", "all", "--log-dir", logsFolder}

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
copilot %s 2>&1 | tee %s`, shellJoinArgs(copilotArgs), logFile)

	env := map[string]string{
		"XDG_CONFIG_HOME":           "/home/runner",
		"COPILOT_AGENT_RUNNER_TYPE": "STANDALONE",
		"GITHUB_TOKEN":              "${{ secrets.COPILOT_CLI_TOKEN  }}",
		"GITHUB_STEP_SUMMARY":       "${{ env.GITHUB_STEP_SUMMARY }}",
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

func (e *CopilotEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	yaml.WriteString("          mkdir -p /home/runner/.copilot\n")
	yaml.WriteString("          cat > /home/runner/.copilot/mcp-config.json << 'EOF'\n")
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"mcpServers\": {\n")

	// Filter out tools that don't need MCP configuration
	var actualMCPTools []string
	for _, toolName := range mcpTools {
		switch toolName {
		case "cache-memory":
			// Cache-memory is handled as a simple file share, not an MCP server
			// Skip adding it to the MCP configuration since no server is needed
			continue
		default:
			// Include all other tools (github, playwright, safe-outputs, and custom MCP tools)
			actualMCPTools = append(actualMCPTools, toolName)
		}
	}

	// Generate configuration for each MCP tool
	totalServers := len(actualMCPTools)
	serverCount := 0

	for _, toolName := range actualMCPTools {
		serverCount++
		isLast := serverCount == totalServers

		switch toolName {
		case "github":
			githubTool := tools["github"]
			e.renderGitHubCopilotMCPConfig(yaml, githubTool, isLast, workflowData)
		case "playwright":
			playwrightTool := tools["playwright"]
			e.renderPlaywrightCopilotMCPConfig(yaml, playwrightTool, isLast, workflowData.NetworkPermissions)
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
	yaml.WriteString("          echo \"-------START MCP CONFIG-----------\"\n")
	yaml.WriteString("          cat /home/runner/.copilot/mcp-config.json\n")
	yaml.WriteString("          echo \"-------END MCP CONFIG-----------\"\n")
	yaml.WriteString("          echo \"-------/home/runner/.copilot-----------\"\n")
	yaml.WriteString("          find /home/runner/.copilot\n")
	//GITHUB_COPILOT_CLI_MODE
	yaml.WriteString("          echo \"HOME: $HOME\"\n")
	yaml.WriteString("          echo \"GITHUB_COPILOT_CLI_MODE: $GITHUB_COPILOT_CLI_MODE\"\n")
	//yaml.WriteString("          echo \"GITHUB_AW_SAFE_OUTPUTS_CONFIG: ${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }}\"")

}

// renderGitHubCopilotMCPConfig generates the GitHub MCP server configuration for Copilot CLI
// Uses Docker-based GitHub MCP server (similar to Claude engine)
func (e *CopilotEngine) renderGitHubCopilotMCPConfig(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData) {
	githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)

	yaml.WriteString("              \"github\": {\n")
	yaml.WriteString("                \"type\": \"local\",\n")

	// Use Docker-based GitHub MCP server (same as Claude engine)
	yaml.WriteString("                \"command\": \"docker\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"run\",\n")
	yaml.WriteString("                  \"-i\",\n")
	yaml.WriteString("                  \"--rm\",\n")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN=${{ secrets.GITHUB_TOKEN }}\",\n")
	yaml.WriteString("                  \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"\n")
	yaml.WriteString("                ],\n")
	yaml.WriteString("                \"tools\": [\"*\"]\n")
	// copilot does not support env

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderPlaywrightCopilotMCPConfig generates the Playwright MCP server configuration for Copilot CLI
func (e *CopilotEngine) renderPlaywrightCopilotMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool, networkPermissions *NetworkPermissions) {
	args := generatePlaywrightDockerArgs(playwrightTool, networkPermissions)

	// Use the version from docker args (which handles version configuration)
	playwrightPackage := "@playwright/mcp@" + args.ImageVersion

	yaml.WriteString("              \"playwright\": {\n")
	yaml.WriteString("                \"type\": \"local\",\n")
	yaml.WriteString("                \"command\": \"npx\",\n")
	yaml.WriteString("                \"args\": [\"" + playwrightPackage + "\", \"--output-dir\", \"/tmp/mcp-logs/playwright\"")

	if len(args.AllowedDomains) > 0 {
		yaml.WriteString(", \"--allowed-origins\", \"" + strings.Join(args.AllowedDomains, ";") + "\"")
	}

	yaml.WriteString("],\n")
	yaml.WriteString("                \"tools\": [\"*\"]\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderSafeOutputsCopilotMCPConfig generates the Safe Outputs MCP server configuration for Copilot CLI
func (e *CopilotEngine) renderSafeOutputsCopilotMCPConfig(yaml *strings.Builder, isLast bool) {
	yaml.WriteString("              \"safe_outputs\": {\n")
	yaml.WriteString("                \"type\": \"local\",\n")
	yaml.WriteString("                \"command\": \"node\",\n")
	yaml.WriteString("                \"args\": [\"/tmp/safe-outputs/mcp-server.cjs\"],\n")
	yaml.WriteString("                \"tools\": [\"*\"],\n")
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

// renderCopilotMCPConfig generates custom MCP server configuration for Copilot CLI
func (e *CopilotEngine) renderCopilotMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	// Use the shared renderer with copilot-specific requirements
	renderer := MCPConfigRenderer{
		Format:                "json",
		IndentLevel:           "                ",
		RequiresCopilotFields: true,
	}

	yaml.WriteString("              \"" + toolName + "\": {\n")

	// Use shared renderer for the server configuration
	if err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer); err != nil {
		return err
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
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

		// Basic processing - error/warning counting moved to end of function
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

	// Count errors and warnings using pattern matching for better accuracy
	errorPatterns := e.GetErrorPatterns()
	if len(errorPatterns) > 0 {
		counts := CountErrorsAndWarningsWithPatterns(logContent, errorPatterns)
		metrics.ErrorCount = counts.ErrorCount
		metrics.WarningCount = counts.WarningCount
	}

	// Detect permission errors and create missing-tool entries
	e.detectPermissionErrorsAndCreateMissingTools(logContent, verbose)

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

	// Handle GitHub MCP tools - GitHub is built-in to Copilot CLI but still needs tool allowlist
	if githubConfig, hasGithub := tools["github"]; hasGithub {
		// Check if GitHub is explicitly disabled (false value)
		if githubBool, isBool := githubConfig.(bool); isBool && !githubBool {
			// GitHub is explicitly disabled - deny it
			args = append(args, "--deny-tool", "github")
		} else if githubConfigMap, ok := githubConfig.(map[string]any); ok {
			// GitHub is built-in, so we don't add --allow-tool github itself
			// But we do need to add --allow-tool github(tool_name) for each allowed tool
			if allowed, hasAllowed := githubConfigMap["allowed"]; hasAllowed {
				if allowedList, ok := allowed.([]any); ok {
					if len(allowedList) == 0 {
						// Empty allowed list means GitHub is disabled - deny it
						args = append(args, "--deny-tool", "github")
					} else {
						for _, allowedTool := range allowedList {
							if toolStr, ok := allowedTool.(string); ok {
								args = append(args, "--allow-tool", fmt.Sprintf("github(%s)", toolStr))
							}
						}
					}
				}
			} else {
				// No allowed field means GitHub is disabled - deny it
				args = append(args, "--deny-tool", "github")
			}
		} else if githubConfig == nil {
			// GitHub is set to nil - deny it
			args = append(args, "--deny-tool", "github")
		}
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

	// Sort arguments while preserving their flag types (--allow-tool or --deny-tool)
	if len(args) > 0 {
		// Create pairs of (flag, value) for sorting
		type argPair struct {
			flag  string
			value string
		}
		var pairs []argPair
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				pairs = append(pairs, argPair{flag: args[i], value: args[i+1]})
			}
		}

		// Sort by value
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].value < pairs[j].value
		})

		// Rebuild args with sorted pairs
		newArgs := make([]string, 0, len(args))
		for _, pair := range pairs {
			newArgs = append(newArgs, pair.flag, pair.value)
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
		// Permission error patterns for missing tools
		{
			Pattern:      `(?i)access denied.*only authorized.*can trigger.*workflow`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Permission denied - workflow access restriction",
		},
		{
			Pattern:      `(?i)access denied.*user.*not authorized`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Permission denied - user not authorized",
		},
		{
			Pattern:      `(?i)repository permission check failed`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Repository permission check failure",
		},
		{
			Pattern:      `(?i)configuration error.*required permissions not specified`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Configuration error - missing permissions",
		},
		{
			Pattern:      `(?i)permission.*denied`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Generic permission denied error",
		},
		{
			Pattern:      `(?i)unauthorized`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Unauthorized access error",
		},
		{
			Pattern:      `(?i)forbidden`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Forbidden access error",
		},
		{
			Pattern:      `(?i)access.*restricted`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Access restricted error",
		},
		{
			Pattern:      `(?i)insufficient.*permission`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Insufficient permissions error",
		},
		{
			Pattern:      `(?i)authentication failed`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Authentication failure with Copilot CLI",
		},
		{
			Pattern:      `(?i)token.*invalid`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Invalid token error with Copilot CLI",
		},
		{
			Pattern:      `(?i)not authorized.*copilot`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Not authorized for Copilot CLI access",
		},
	}
}

// detectPermissionErrorsAndCreateMissingTools scans Copilot CLI log content for permission errors
// and creates missing-tool entries in the safe outputs file
func (e *CopilotEngine) detectPermissionErrorsAndCreateMissingTools(logContent string, verbose bool) {
	patterns := e.getPermissionErrorPatterns()
	lines := strings.Split(logContent, "\n")

	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for _, line := range lines {
			if regex.MatchString(line) {
				// Found a permission error - for Copilot CLI, the tool is generally the CLI itself
				toolName := "github-copilot-cli"
				e.createCopilotMissingToolEntry(toolName, line, verbose)
			}
		}
	}
}

// getPermissionErrorPatterns returns only the permission-related error patterns
func (e *CopilotEngine) getPermissionErrorPatterns() []ErrorPattern {
	allPatterns := e.GetErrorPatterns()
	var permissionPatterns []ErrorPattern

	for _, pattern := range allPatterns {
		if strings.Contains(strings.ToLower(pattern.Description), "permission") ||
			strings.Contains(strings.ToLower(pattern.Description), "unauthorized") ||
			strings.Contains(strings.ToLower(pattern.Description), "forbidden") ||
			strings.Contains(strings.ToLower(pattern.Description), "access") ||
			strings.Contains(strings.ToLower(pattern.Description), "authentication") ||
			strings.Contains(strings.ToLower(pattern.Description), "token") {
			permissionPatterns = append(permissionPatterns, pattern)
		}
	}

	return permissionPatterns
}

// createCopilotMissingToolEntry creates a missing-tool entry in the safe outputs file
func (e *CopilotEngine) createCopilotMissingToolEntry(toolName, reason string, verbose bool) {
	// Get the safe outputs file path from environment
	safeOutputsFile := os.Getenv("GITHUB_AW_SAFE_OUTPUTS")
	if safeOutputsFile == "" {
		if verbose {
			fmt.Printf("GITHUB_AW_SAFE_OUTPUTS not set, cannot write permission error missing-tool entry\n")
		}
		return
	}

	// Create missing-tool entry
	missingToolEntry := map[string]any{
		"type":         "missing-tool",
		"tool":         toolName,
		"reason":       fmt.Sprintf("Permission denied: %s", reason),
		"alternatives": "Check repository permissions and access controls",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	// Convert to JSON and append to safe outputs file
	entryJSON, err := json.Marshal(missingToolEntry)
	if err != nil {
		if verbose {
			fmt.Printf("Failed to marshal missing-tool entry: %v\n", err)
		}
		return
	}

	// Append to the safe outputs file
	file, err := os.OpenFile(safeOutputsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if verbose {
			fmt.Printf("Failed to open safe outputs file: %v\n", err)
		}
		return
	}
	defer file.Close()

	if _, err := file.WriteString(string(entryJSON) + "\n"); err != nil {
		if verbose {
			fmt.Printf("Failed to write missing-tool entry: %v\n", err)
		}
		return
	}

	if verbose {
		fmt.Printf("Recorded permission error as missing tool: %s\n", toolName)
	}
}
