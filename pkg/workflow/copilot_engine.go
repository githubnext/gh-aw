package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

const logsFolder = "/tmp/gh-aw/.copilot/logs/"

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
			experimental:           false,
			supportsToolsAllowlist: true,
			supportsHTTPTransport:  true,  // Copilot CLI supports HTTP transport via MCP
			supportsMaxTurns:       false, // Copilot CLI does not support max-turns feature yet
			supportsWebFetch:       false, // Copilot CLI does not have built-in web-fetch support
			supportsWebSearch:      false, // Copilot CLI does not have built-in web-search support
		},
	}
}

func (e *CopilotEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	// Add secret validation step
	secretValidation := GenerateSecretValidationStep(
		"COPILOT_CLI_TOKEN",
		"GitHub Copilot CLI",
		"https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default",
	)
	steps = append(steps, secretValidation)

	// First, get the setup Node.js step from npm steps
	npmSteps := BuildStandardNpmEngineInstallSteps(
		"@github/copilot",
		constants.DefaultCopilotVersion,
		"Install GitHub Copilot CLI",
		"copilot",
		workflowData,
	)

	// Add Node.js setup step first (before AWF)
	if len(npmSteps) > 0 {
		steps = append(steps, npmSteps[0]) // Setup Node.js step
	}

	// Add AWF installation steps only if "firewall" feature is enabled
	if IsFeatureEnabledWithData("firewall", workflowData) {
		// Install AWF after Node.js setup but before Copilot CLI installation
		var awfVersion string
		var cleanupScript string
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.Firewall != nil {
			awfVersion = workflowData.EngineConfig.Firewall.Version
			cleanupScript = workflowData.EngineConfig.Firewall.CleanupScript
		}

		// Install AWF binary
		awfInstall := generateAWFInstallationStep(awfVersion)
		steps = append(steps, awfInstall)

		// Pre-execution cleanup
		awfCleanup := generateAWFCleanupStep(cleanupScript)
		steps = append(steps, awfCleanup)
	}

	// Add Copilot CLI installation step after AWF
	if len(npmSteps) > 1 {
		steps = append(steps, npmSteps[1:]...) // Install Copilot CLI and subsequent steps
	}

	return steps
}

func (e *CopilotEngine) GetDeclaredOutputFiles() []string {
	return []string{logsFolder}
}

// GetVersionCommand returns the command to get Copilot CLI's version
func (e *CopilotEngine) GetVersionCommand() string {
	if IsFeatureEnabled("firewall") {
		// When firewall is enabled, use version pinning with npx
		return fmt.Sprintf("npx -y @github/copilot@%s --version", constants.DefaultCopilotVersion)
	}
	// When firewall is disabled, use unpinned command
	return "copilot --version"
}

// extractAddDirPaths extracts all directory paths from copilot args that follow --add-dir flags
func extractAddDirPaths(args []string) []string {
	var dirs []string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--add-dir" {
			dirs = append(dirs, args[i+1])
		}
	}
	return dirs
}

// GetExecutionSteps returns the GitHub Actions steps for executing GitHub Copilot CLI
func (e *CopilotEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	// Handle custom steps if they exist in engine config
	steps := InjectCustomEngineSteps(workflowData, e.convertStepToYAML)

	// Build copilot CLI arguments based on configuration
	var copilotArgs []string
	if IsFeatureEnabledWithData("firewall", workflowData) {
		// Simplified args for firewall mode
		copilotArgs = []string{"--add-dir", "/tmp/gh-aw/", "--log-level", "all"}
	} else {
		// Original args for non-firewall mode
		copilotArgs = []string{"--add-dir", "/tmp/", "--add-dir", "/tmp/gh-aw/", "--add-dir", "/tmp/gh-aw/agent/", "--log-level", "all", "--log-dir", logsFolder}
	}

	// Add --disable-builtin-mcps to disable built-in MCP servers
	copilotArgs = append(copilotArgs, "--disable-builtin-mcps")

	// Add model if specified (check if Copilot CLI supports this)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		copilotArgs = append(copilotArgs, "--model", workflowData.EngineConfig.Model)
	}

	// Add tool permission arguments based on configuration
	toolArgs := e.computeCopilotToolArguments(workflowData.Tools, workflowData.SafeOutputs)
	copilotArgs = append(copilotArgs, toolArgs...)

	// if cache-memory tool is used, --add-dir for each cache
	if workflowData.CacheMemoryConfig != nil {
		for _, cache := range workflowData.CacheMemoryConfig.Caches {
			var cacheDir string
			if cache.ID == "default" {
				cacheDir = "/tmp/gh-aw/cache-memory/"
			} else {
				cacheDir = fmt.Sprintf("/tmp/gh-aw/cache-memory-%s/", cache.ID)
			}
			copilotArgs = append(copilotArgs, "--add-dir", cacheDir)
		}
	}

	// Add --allow-all-paths when edit tool is enabled to allow write on all paths
	// See: https://github.com/github/copilot-cli/issues/67#issuecomment-3411256174
	if _, hasEdit := workflowData.Tools["edit"]; hasEdit {
		copilotArgs = append(copilotArgs, "--allow-all-paths")
	}

	// Add custom args from engine configuration before the prompt
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		copilotArgs = append(copilotArgs, workflowData.EngineConfig.Args...)
	}

	// Add prompt argument - inline for firewall, variable for non-firewall
	if IsFeatureEnabledWithData("firewall", workflowData) {
		copilotArgs = append(copilotArgs, "--prompt", "\"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\"")
	} else {
		copilotArgs = append(copilotArgs, "--prompt", "\"$COPILOT_CLI_INSTRUCTION\"")
	}

	// Extract all --add-dir paths and generate mkdir commands
	addDirPaths := extractAddDirPaths(copilotArgs)

	// Also ensure the log directory exists
	addDirPaths = append(addDirPaths, logsFolder)

	var mkdirCommands strings.Builder
	for _, dir := range addDirPaths {
		mkdirCommands.WriteString(fmt.Sprintf("mkdir -p %s\n", dir))
	}

	// Build the copilot command
	var copilotCommand string
	if IsFeatureEnabledWithData("firewall", workflowData) {
		// When firewall is enabled, use version pinning with npx
		copilotVersion := constants.DefaultCopilotVersion
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
			copilotVersion = workflowData.EngineConfig.Version
		}
		copilotCommand = fmt.Sprintf("npx -y @github/copilot@%s %s", copilotVersion, shellJoinArgs(copilotArgs))
	} else {
		// When firewall is disabled, use unpinned copilot command
		copilotCommand = fmt.Sprintf("copilot %s", shellJoinArgs(copilotArgs))
	}

	// Conditionally wrap with AWF if "firewall" feature is enabled
	var command string
	if IsFeatureEnabledWithData("firewall", workflowData) {
		// Build the AWF-wrapped command - no mkdir needed, AWF handles it
		var awfLogLevel = "debug"
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.Firewall != nil && workflowData.EngineConfig.Firewall.LogLevel != "" {
			awfLogLevel = workflowData.EngineConfig.Firewall.LogLevel
		}

		// Get allowed domains (copilot defaults + network permissions) with specific ordering
		allowedDomains := GetCopilotAllowedDomains(workflowData.NetworkPermissions)

		command = fmt.Sprintf(`set -o pipefail
sudo -E awf --env-all \
  --allow-domains %s \
  --log-level %s \
  '%s' \
  2>&1 | tee %s

# Move preserved Copilot logs to expected location
COPILOT_LOGS_DIR=$(ls -td /tmp/copilot-logs-* 2>/dev/null | head -1)
if [ -n "$COPILOT_LOGS_DIR" ] && [ -d "$COPILOT_LOGS_DIR" ]; then
  echo "Moving Copilot logs from $COPILOT_LOGS_DIR to %s"
  mkdir -p %s
  mv "$COPILOT_LOGS_DIR"/* %s || true
  rmdir "$COPILOT_LOGS_DIR" || true
fi`, allowedDomains, awfLogLevel, copilotCommand, logFile, logsFolder, logsFolder, logsFolder)
	} else {
		// Run copilot command without AWF wrapper
		command = fmt.Sprintf(`set -o pipefail
COPILOT_CLI_INSTRUCTION=$(cat /tmp/gh-aw/aw-prompts/prompt.txt)
%s%s 2>&1 | tee %s`, mkdirCommands.String(), copilotCommand, logFile)
	}

	env := map[string]string{
		"XDG_CONFIG_HOME":           "/home/runner",
		"COPILOT_AGENT_RUNNER_TYPE": "STANDALONE",
		"GITHUB_TOKEN":              "${{ secrets.COPILOT_CLI_TOKEN  }}",
		"GITHUB_STEP_SUMMARY":       "${{ env.GITHUB_STEP_SUMMARY }}",
	}

	// Always add GH_AW_PROMPT for agentic workflows
	env["GH_AW_PROMPT"] = "/tmp/gh-aw/aw-prompts/prompt.txt"

	// Add GH_AW_MCP_CONFIG for MCP server configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "/home/runner/.copilot/mcp-config.json"
	}

	if hasGitHubTool(workflowData.Tools) {
		githubTool := workflowData.Tools["github"]
		customGitHubToken := getGitHubToken(githubTool)
		// Use effective token with precedence: custom > top-level > default
		effectiveToken := getEffectiveGitHubToken(customGitHubToken, workflowData.GitHubToken)
		env["GITHUB_PERSONAL_ACCESS_TOKEN"] = effectiveToken
	}

	// Add GH_AW_SAFE_OUTPUTS if output is needed
	applySafeOutputEnvToMap(env, workflowData)

	// Add GH_AW_STARTUP_TIMEOUT environment variable (in seconds) if startup-timeout is specified
	if workflowData.ToolsStartupTimeout > 0 {
		env["GH_AW_STARTUP_TIMEOUT"] = fmt.Sprintf("%d", workflowData.ToolsStartupTimeout)
	}

	// Add GH_AW_TOOL_TIMEOUT environment variable (in seconds) if timeout is specified
	if workflowData.ToolsTimeout > 0 {
		env["GH_AW_TOOL_TIMEOUT"] = fmt.Sprintf("%d", workflowData.ToolsTimeout)
	}

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GH_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	}

	// Add custom environment variables from engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			env[key] = value
		}
	}

	// Add HTTP MCP header secrets to env for passthrough
	headerSecrets := collectHTTPMCPHeaderSecrets(workflowData.Tools)
	for varName, secretExpr := range headerSecrets {
		// Only add if not already in env
		if _, exists := env[varName]; !exists {
			env[varName] = secretExpr
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
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %d", constants.DefaultAgenticWorkflowTimeoutMinutes)) // Default timeout for agentic workflows
	}

	// Format step with command and environment variables using shared helper
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, env)

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}

// convertStepToYAML converts a step map to YAML string - uses proper YAML serialization
func (e *CopilotEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}

// GetSquidLogsSteps returns the steps for collecting and uploading Squid logs
func (e *CopilotEngine) GetSquidLogsSteps(workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	// Only add Squid logs collection and upload steps if "firewall" feature is enabled
	if IsFeatureEnabledWithData("firewall", workflowData) {
		squidLogsCollection := generateSquidLogsCollectionStep(workflowData.Name)
		steps = append(steps, squidLogsCollection)

		squidLogsUpload := generateSquidLogsUploadStep(workflowData.Name)
		steps = append(steps, squidLogsUpload)
	}

	return steps
}

// GetCleanupStep returns the post-execution cleanup step
func (e *CopilotEngine) GetCleanupStep(workflowData *WorkflowData) GitHubActionStep {
	// Only add cleanup step if "firewall" feature is enabled
	if IsFeatureEnabledWithData("firewall", workflowData) {
		var postCleanupScript string
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.Firewall != nil {
			postCleanupScript = workflowData.EngineConfig.Firewall.CleanupScript
		}
		return generateAWFPostExecutionCleanupStep(postCleanupScript)
	}
	// Return empty step if firewall is disabled
	return GitHubActionStep([]string{})
}

func (e *CopilotEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	// Create the directory first
	yaml.WriteString("          mkdir -p /home/runner/.copilot\n")

	// Use shared JSON MCP config renderer with Copilot-specific options
	RenderJSONMCPConfig(yaml, tools, mcpTools, workflowData, JSONMCPConfigOptions{
		ConfigPath: "/home/runner/.copilot/mcp-config.json",
		Renderers: MCPToolRenderers{
			RenderGitHub: func(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData) {
				e.renderGitHubCopilotMCPConfig(yaml, githubTool, isLast)
			},
			RenderPlaywright: e.renderPlaywrightCopilotMCPConfig,
			RenderCacheMemory: func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
				// Cache-memory is not used for Copilot (filtered out)
			},
			RenderAgenticWorkflows: e.renderAgenticWorkflowsCopilotMCPConfig,
			RenderSafeOutputs:      e.renderSafeOutputsCopilotMCPConfig,
			RenderWebFetch: func(yaml *strings.Builder, isLast bool) {
				renderMCPFetchServerConfig(yaml, "json", "              ", isLast, true)
			},
			RenderCustomMCPConfig: e.renderCopilotMCPConfig,
		},
		FilterTool: func(toolName string) bool {
			// Filter out cache-memory for Copilot
			// Cache-memory is handled as a simple file share, not an MCP server
			return toolName != "cache-memory"
		},
		PostEOFCommands: func(yaml *strings.Builder) {
			// Add debug output
			yaml.WriteString("          echo \"-------START MCP CONFIG-----------\"\n")
			yaml.WriteString("          cat /home/runner/.copilot/mcp-config.json\n")
			yaml.WriteString("          echo \"-------END MCP CONFIG-----------\"\n")
			yaml.WriteString("          echo \"-------/home/runner/.copilot-----------\"\n")
			yaml.WriteString("          find /home/runner/.copilot\n")
		},
	})
	//GITHUB_COPILOT_CLI_MODE
	yaml.WriteString("          echo \"HOME: $HOME\"\n")
	yaml.WriteString("          echo \"GITHUB_COPILOT_CLI_MODE: $GITHUB_COPILOT_CLI_MODE\"\n")
}

// renderGitHubCopilotMCPConfig generates the GitHub MCP server configuration for Copilot CLI
// Supports both local (Docker) and remote (hosted) modes
func (e *CopilotEngine) renderGitHubCopilotMCPConfig(yaml *strings.Builder, githubTool any, isLast bool) {
	githubType := getGitHubType(githubTool)
	readOnly := getGitHubReadOnly(githubTool)
	toolsets := getGitHubToolsets(githubTool)
	allowedTools := getGitHubAllowedTools(githubTool)

	yaml.WriteString("              \"github\": {\n")

	// Check if remote mode is enabled (type: remote)
	if githubType == "remote" {
		// Remote mode - use hosted GitHub MCP server
		yaml.WriteString("                \"type\": \"http\",\n")
		yaml.WriteString("                \"url\": \"https://api.githubcopilot.com/mcp/\",\n")
		yaml.WriteString("                \"headers\": {\n")

		// Collect headers in a map
		headers := make(map[string]string)
		headers["Authorization"] = "Bearer \\${GITHUB_PERSONAL_ACCESS_TOKEN}"

		// Add X-MCP-Readonly header if read-only mode is enabled
		if readOnly {
			headers["X-MCP-Readonly"] = "true"
		}

		// Add X-MCP-Toolsets header if toolsets are configured
		if toolsets != "" {
			headers["X-MCP-Toolsets"] = toolsets
		}

		// Write headers using helper
		writeHeadersToYAML(yaml, headers, "                  ")

		yaml.WriteString("                },\n")

		// Populate tools field with allowed tools or "*" if none specified
		if len(allowedTools) > 0 {
			yaml.WriteString("                \"tools\": [\n")
			for i, tool := range allowedTools {
				comma := ","
				if i == len(allowedTools)-1 {
					comma = ""
				}
				fmt.Fprintf(yaml, "                  \"%s\"%s\n", tool, comma)
			}
			yaml.WriteString("                ],\n")
		} else {
			yaml.WriteString("                \"tools\": [\"*\"],\n")
		}

		// Add env section for passthrough
		yaml.WriteString("                \"env\": {\n")
		yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"\\${GITHUB_PERSONAL_ACCESS_TOKEN}\"\n")
		yaml.WriteString("                }\n")
	} else {
		// Local mode - use Docker-based GitHub MCP server (default)
		githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
		customArgs := getGitHubCustomArgs(githubTool)

		RenderGitHubMCPDockerConfig(yaml, GitHubMCPDockerOptions{
			ReadOnly:           readOnly,
			Toolsets:           toolsets,
			DockerImageVersion: githubDockerImageVersion,
			CustomArgs:         customArgs,
			IncludeTypeField:   true, // Copilot includes "type": "local" field
			AllowedTools:       allowedTools,
			EffectiveToken:     "", // Copilot uses env passthrough
		})
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderPlaywrightCopilotMCPConfig generates the Playwright MCP server configuration for Copilot CLI
// Uses the shared helper with Copilot-specific options
func (e *CopilotEngine) renderPlaywrightCopilotMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool) {
	renderPlaywrightMCPConfigWithOptions(yaml, playwrightTool, isLast, true, true)
}

// renderSafeOutputsCopilotMCPConfig generates the Safe Outputs MCP server configuration for Copilot CLI
// Uses the shared helper with Copilot-specific options
func (e *CopilotEngine) renderSafeOutputsCopilotMCPConfig(yaml *strings.Builder, isLast bool) {
	renderSafeOutputsMCPConfigWithOptions(yaml, isLast, true)
}

// renderAgenticWorkflowsCopilotMCPConfig generates the Agentic Workflows MCP server configuration for Copilot CLI
// Uses the shared helper with Copilot-specific options
func (e *CopilotEngine) renderAgenticWorkflowsCopilotMCPConfig(yaml *strings.Builder, isLast bool) {
	renderAgenticWorkflowsMCPConfigWithOptions(yaml, isLast, true)
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
		metrics.Errors = CountErrorsAndWarningsWithPatterns(logContent, errorPatterns)
	}

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
					MaxInputSize:  0, // TODO: Extract input size from tool call parameters if available
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

// GetLogFileForParsing returns the log directory for Copilot CLI logs
// Copilot writes detailed debug logs to /tmp/gh-aw/.copilot/logs/ which should be parsed
// instead of the agent-stdio.log file
func (e *CopilotEngine) GetLogFileForParsing() string {
	return logsFolder
}

// computeCopilotToolArguments generates Copilot CLI tool permission arguments from workflow tools configuration
func (e *CopilotEngine) computeCopilotToolArguments(tools map[string]any, safeOutputs *SafeOutputsConfig) []string {
	if tools == nil {
		tools = make(map[string]any)
	}

	var args []string

	// Check if bash has wildcard - if so, use --allow-all-tools instead
	if bashConfig, hasBash := tools["bash"]; hasBash {
		if bashCommands, ok := bashConfig.([]any); ok {
			// Check for :* or * wildcard - if present, allow all tools
			for _, cmd := range bashCommands {
				if cmdStr, ok := cmd.(string); ok {
					if cmdStr == ":*" || cmdStr == "*" {
						// Use --allow-all-tools flag instead of individual tool permissions
						return []string{"--allow-all-tools"}
					}
				}
			}
		}
	}

	// Handle bash/shell tools (when no wildcard)
	if bashConfig, hasBash := tools["bash"]; hasBash {
		if bashCommands, ok := bashConfig.([]any); ok {
			// Add specific shell commands
			for _, cmd := range bashCommands {
				if cmdStr, ok := cmd.(string); ok {
					args = append(args, "--allow-tool", fmt.Sprintf("shell(%s)", cmdStr))
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
	// Note: GitHub is NOT included here because it needs MCP configuration in CLI mode
	// Note: web-fetch is NOT included here because it may be an MCP server for engines without native support
	builtInTools := map[string]bool{
		"bash":       true,
		"edit":       true,
		"web-search": true,
		"playwright": true,
	}

	// Handle MCP server tools
	for toolName, toolConfig := range tools {
		// Skip built-in tools we've already handled
		if builtInTools[toolName] {
			continue
		}

		// GitHub is a special case - it's an MCP server but doesn't have explicit MCP config in the workflow
		// It gets MCP configuration through the parser's processBuiltinMCPTool
		if toolName == "github" {
			if toolConfigMap, ok := toolConfig.(map[string]any); ok {
				if allowed, hasAllowed := toolConfigMap["allowed"]; hasAllowed {
					if allowedList, ok := allowed.([]any); ok {
						// Process allowed list in a single pass
						hasWildcard := false
						for _, allowedTool := range allowedList {
							if toolStr, ok := allowedTool.(string); ok {
								if toolStr == "*" {
									// Wildcard means allow entire GitHub MCP server
									hasWildcard = true
								} else {
									// Add individual tool permission
									args = append(args, "--allow-tool", fmt.Sprintf("github(%s)", toolStr))
								}
							}
						}

						// Add server-level permission only if wildcard was present
						if hasWildcard {
							args = append(args, "--allow-tool", "github")
						}
					}
				} else {
					// No allowed field specified - allow entire GitHub MCP server
					args = append(args, "--allow-tool", "github")
				}
			} else {
				// GitHub tool exists but is not a map (e.g., github: null) - allow entire server
				args = append(args, "--allow-tool", "github")
			}
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
	patterns := GetCommonErrorPatterns()

	// Add Copilot-specific error patterns for timestamp-based log formats
	patterns = append(patterns, []ErrorPattern{
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
		// Copilot CLI-specific error indicators without "ERROR:" prefix
		{
			Pattern:      `✗\s+(.+)`,
			LevelGroup:   0,
			MessageGroup: 1,
			Description:  "Copilot CLI failed command indicator",
		},
		{
			Pattern:      `(?:command not found|not found):\s*(.+)|(.+):\s*(?:command not found|not found)`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Shell command not found error",
		},
		{
			Pattern:      `Cannot find module\s+['"](.+)['"]`,
			LevelGroup:   0,
			MessageGroup: 1,
			Description:  "Node.js module not found error",
		},
		{
			Pattern:      `Permission denied and could not request permission from user`,
			LevelGroup:   0,
			MessageGroup: 0,
			Severity:     "warning",
			Description:  "Copilot CLI permission denied warning (user interaction required)",
		},
	}...)

	return patterns
}

// generateAWFInstallationStep creates a GitHub Actions step to install the AWF binary
func generateAWFInstallationStep(version string) GitHubActionStep {
	stepLines := []string{
		"      - name: Install awf binary",
		"        run: |",
	}

	if version == "" {
		stepLines = append(stepLines, "          LATEST_TAG=$(gh release view --repo githubnext/gh-aw-firewall --json tagName --jq .tagName)")
		stepLines = append(stepLines, "          echo \"Installing awf from release: $LATEST_TAG\"")
		stepLines = append(stepLines, "          curl -L https://github.com/githubnext/gh-aw-firewall/releases/download/${LATEST_TAG}/awf-linux-x64 -o awf")
	} else {
		stepLines = append(stepLines, fmt.Sprintf("          echo \"Installing awf from release: %s\"", version))
		stepLines = append(stepLines, fmt.Sprintf("          curl -L https://github.com/githubnext/gh-aw-firewall/releases/download/%s/awf-linux-x64 -o awf", version))
	}

	stepLines = append(stepLines,
		"          chmod +x awf",
		"          sudo mv awf /usr/local/bin/",
		"          which awf",
		"          awf --version",
		"        env:",
		"          GH_TOKEN: ${{ github.token }}",
	)

	return GitHubActionStep(stepLines)
}

// generateAWFCleanupStep creates a GitHub Actions step to cleanup AWF resources
func generateAWFCleanupStep(scriptPath string) GitHubActionStep {
	if scriptPath == "" {
		scriptPath = "./scripts/ci/cleanup.sh"
	}

	stepLines := []string{
		"      - name: Cleanup any existing awf resources",
		fmt.Sprintf("        run: %s || true", scriptPath),
	}

	return GitHubActionStep(stepLines)
}

// sanitizeWorkflowName sanitizes a workflow name for use in artifact names and file paths
// Removes or replaces characters that are invalid in YAML artifact names or filesystem paths
func sanitizeWorkflowName(name string) string {
	// Replace colons, slashes, and other problematic characters with hyphens
	sanitized := strings.ReplaceAll(name, ":", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	// Remove any remaining special characters that might cause issues
	sanitized = strings.Map(func(r rune) rune {
		// Allow alphanumeric, hyphens, underscores, and periods
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, sanitized)
	return sanitized
}

// generateSquidLogsCollectionStep creates a GitHub Actions step to collect Squid logs from AWF
func generateSquidLogsCollectionStep(workflowName string) GitHubActionStep {
	sanitizedName := strings.ToLower(sanitizeWorkflowName(workflowName))
	squidLogsDir := fmt.Sprintf("/tmp/gh-aw/squid-logs-%s/", sanitizedName)

	stepLines := []string{
		"      - name: Agent Firewall logs",
		"        if: always()",
		"        run: |",
		"          # Squid logs are preserved in timestamped directories",
		"          SQUID_LOGS_DIR=$(ls -td /tmp/squid-logs-* 2>/dev/null | head -1)",
		"          if [ -n \"$SQUID_LOGS_DIR\" ] && [ -d \"$SQUID_LOGS_DIR\" ]; then",
		"            echo \"Found Squid logs at: $SQUID_LOGS_DIR\"",
		fmt.Sprintf("            mkdir -p %s", squidLogsDir),
		fmt.Sprintf("            sudo cp -r \"$SQUID_LOGS_DIR\"/* %s || true", squidLogsDir),
		fmt.Sprintf("            sudo chmod -R a+r %s || true", squidLogsDir),
		"          fi",
	}

	return GitHubActionStep(stepLines)
}

// generateSquidLogsUploadStep creates a GitHub Actions step to upload Squid logs as artifact
func generateSquidLogsUploadStep(workflowName string) GitHubActionStep {
	sanitizedName := strings.ToLower(sanitizeWorkflowName(workflowName))
	artifactName := fmt.Sprintf("squid-logs-%s", sanitizedName)
	squidLogsDir := fmt.Sprintf("/tmp/gh-aw/squid-logs-%s/", sanitizedName)

	stepLines := []string{
		"      - name: Upload Squid logs",
		"        if: always()",
		"        uses: actions/upload-artifact@v4",
		"        with:",
		fmt.Sprintf("          name: %s", artifactName),
		fmt.Sprintf("          path: %s", squidLogsDir),
		"          if-no-files-found: ignore",
	}

	return GitHubActionStep(stepLines)
}

// generateAWFPostExecutionCleanupStep creates a GitHub Actions step to cleanup AWF resources after execution
func generateAWFPostExecutionCleanupStep(scriptPath string) GitHubActionStep {
	if scriptPath == "" {
		scriptPath = "./scripts/ci/cleanup.sh"
	}

	stepLines := []string{
		"      - name: Cleanup awf resources",
		"        if: always()",
		fmt.Sprintf("        run: %s || true", scriptPath),
	}

	return GitHubActionStep(stepLines)
}
