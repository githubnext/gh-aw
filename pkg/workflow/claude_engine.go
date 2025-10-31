package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var claudeLog = logger.New("workflow:claude_engine")

// ClaudeEngine represents the Claude Code agentic engine
type ClaudeEngine struct {
	BaseEngine
}

func NewClaudeEngine() *ClaudeEngine {
	return &ClaudeEngine{
		BaseEngine: BaseEngine{
			id:                     "claude",
			displayName:            "Claude Code",
			description:            "Uses Claude Code with full MCP tool support and allow-listing",
			experimental:           false,
			supportsToolsAllowlist: true,
			supportsHTTPTransport:  true, // Claude supports both stdio and HTTP transport
			supportsMaxTurns:       true, // Claude supports max-turns feature
			supportsWebFetch:       true, // Claude has built-in WebFetch support
			supportsWebSearch:      true, // Claude has built-in WebSearch support
		},
	}
}

func (e *ClaudeEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	claudeLog.Printf("Generating installation steps for Claude engine: workflow=%s", workflowData.Name)

	var steps []GitHubActionStep

	// Add secret validation step
	secretValidation := GenerateSecretValidationStep(
		"ANTHROPIC_API_KEY",
		"Claude Code",
		"https://githubnext.github.io/gh-aw/reference/engines/#anthropic-claude-code",
	)
	steps = append(steps, secretValidation)

	// Use shared helper for standard npm installation
	npmSteps := BuildStandardNpmEngineInstallSteps(
		"@anthropic-ai/claude-code",
		constants.DefaultClaudeCodeVersion,
		"Install Claude Code CLI",
		"claude",
		workflowData,
	)
	steps = append(steps, npmSteps...)

	// Check if network permissions are configured (only for Claude engine)
	if workflowData.EngineConfig != nil && ShouldEnforceNetworkPermissions(workflowData.NetworkPermissions) {
		// Generate network hook generator and settings generator
		hookGenerator := &NetworkHookGenerator{}
		settingsGenerator := &ClaudeSettingsGenerator{}

		allowedDomains := GetAllowedDomains(workflowData.NetworkPermissions)

		// Add settings generation step
		settingsStep := settingsGenerator.GenerateSettingsWorkflowStep()
		steps = append(steps, settingsStep)

		// Add hook generation step
		hookStep := hookGenerator.GenerateNetworkHookWorkflowStep(allowedDomains)
		steps = append(steps, hookStep)
	}

	return steps
}

// GetDeclaredOutputFiles returns the output files that Claude may produce
func (e *ClaudeEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetVersionCommand returns the command to get Claude's version
func (e *ClaudeEngine) GetVersionCommand() string {
	return "claude --version"
}

// GetExecutionSteps returns the GitHub Actions steps for executing Claude
func (e *ClaudeEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	claudeLog.Printf("Generating execution steps for Claude engine: workflow=%s", workflowData.Name)

	// Handle custom steps if they exist in engine config
	steps := InjectCustomEngineSteps(workflowData, e.convertStepToYAML)

	// Build claude CLI arguments based on configuration
	var claudeArgs []string

	// Add print flag for non-interactive mode
	claudeArgs = append(claudeArgs, "--print")

	// Add model if specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		claudeArgs = append(claudeArgs, "--model", workflowData.EngineConfig.Model)
	}

	// Add max_turns if specified (in CLI it's max-turns)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		claudeArgs = append(claudeArgs, "--max-turns", workflowData.EngineConfig.MaxTurns)
	}

	// Add MCP configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		claudeArgs = append(claudeArgs, "--mcp-config", "/tmp/gh-aw/mcp-config/mcp-servers.json")
	}

	// Add allowed tools configuration
	allowedTools := e.computeAllowedClaudeToolsString(workflowData.Tools, workflowData.SafeOutputs, workflowData.CacheMemoryConfig)
	if allowedTools != "" {
		claudeArgs = append(claudeArgs, "--allowed-tools", allowedTools)
	}

	// Add debug flag
	claudeArgs = append(claudeArgs, "--debug")

	// Always add verbose flag for enhanced debugging output
	claudeArgs = append(claudeArgs, "--verbose")

	// Add permission mode for non-interactive execution (bypass permissions)
	claudeArgs = append(claudeArgs, "--permission-mode", "bypassPermissions")

	// Add output format for structured output
	claudeArgs = append(claudeArgs, "--output-format", "stream-json")

	// Add network settings if configured
	if workflowData.EngineConfig != nil && ShouldEnforceNetworkPermissions(workflowData.NetworkPermissions) {
		claudeArgs = append(claudeArgs, "--settings", "/tmp/gh-aw/.claude/settings.json")
	}

	// Add custom args from engine configuration before the prompt
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		claudeArgs = append(claudeArgs, workflowData.EngineConfig.Args...)
	}

	var stepLines []string

	stepName := "Execute Claude Code CLI"

	stepLines = append(stepLines, fmt.Sprintf("      - name: %s", stepName))
	stepLines = append(stepLines, "        id: agentic_execution")

	// Add allowed tools comment before the run section
	allowedToolsComment := e.generateAllowedToolsComment(e.computeAllowedClaudeToolsString(workflowData.Tools, workflowData.SafeOutputs, workflowData.CacheMemoryConfig), "        ")
	if allowedToolsComment != "" {
		// Split the comment into lines and add each line
		commentLines := strings.Split(strings.TrimSuffix(allowedToolsComment, "\n"), "\n")
		stepLines = append(stepLines, commentLines...)
	}

	// Add timeout at step level (GitHub Actions standard)
	if workflowData.TimeoutMinutes != "" {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %s", strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout_minutes: ")))
	} else {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %d", constants.DefaultAgenticWorkflowTimeoutMinutes)) // Default timeout for agentic workflows
	}

	// Build the run command
	stepLines = append(stepLines, "        run: |")
	stepLines = append(stepLines, "          set -o pipefail")
	stepLines = append(stepLines, "          # Execute Claude Code CLI with prompt from file")

	// Build the prompt command - prepend custom agent file content if specified
	var promptCommand string
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.CustomAgent != "" {
		// Extract markdown body from custom agent file and prepend to prompt
		stepLines = append(stepLines, "          # Extract markdown body from custom agent file (skip frontmatter)")
		stepLines = append(stepLines, fmt.Sprintf("          AGENT_CONTENT=$(awk 'BEGIN{skip=1} /^---$/{if(skip){skip=0;next}else{skip=1;next}} !skip' %s)", workflowData.EngineConfig.CustomAgent))
		stepLines = append(stepLines, "          # Combine agent content with prompt")
		stepLines = append(stepLines, "          PROMPT_TEXT=$(printf '%s\\n\\n%s' \"$AGENT_CONTENT\" \"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\")")
		promptCommand = "\"$PROMPT_TEXT\""
	} else {
		promptCommand = "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"
	}

	// Build the command string with proper argument formatting
	// Use claude command directly (installed via npm install -g)
	commandParts := []string{"claude"}
	commandParts = append(commandParts, claudeArgs...)
	commandParts = append(commandParts, promptCommand)

	// Join command parts with proper escaping for complex arguments
	command := ""
	for i, part := range commandParts {
		if i > 0 {
			command += " "
		}
		// For complex arguments that contain spaces or special characters, quote them
		if strings.Contains(part, " ") || strings.Contains(part, ",") {
			command += "\"" + part + "\""
		} else {
			command += part
		}
	}

	// Add the command with proper indentation and tee output (preserves exit code with pipefail)
	stepLines = append(stepLines, fmt.Sprintf("          %s 2>&1 | tee %s", command, logFile))

	// Add environment section - always include environment section for GH_AW_PROMPT
	stepLines = append(stepLines, "        env:")

	// Add Anthropic API key
	stepLines = append(stepLines, "          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}")

	// Disable telemetry, error reporting, and bug command for privacy and security
	stepLines = append(stepLines, "          DISABLE_TELEMETRY: \"1\"")
	stepLines = append(stepLines, "          DISABLE_ERROR_REPORTING: \"1\"")
	stepLines = append(stepLines, "          DISABLE_BUG_COMMAND: \"1\"")

	// Always add GH_AW_PROMPT for agentic workflows
	stepLines = append(stepLines, "          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt")

	// Add GH_AW_MCP_CONFIG for MCP server configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		stepLines = append(stepLines, "          GH_AW_MCP_CONFIG: /tmp/gh-aw/mcp-config/mcp-servers.json")
	}

	// Set timeout environment variables for Claude Code
	// Use tools.startup-timeout if specified, otherwise default to DefaultMCPStartupTimeoutSeconds
	startupTimeoutMs := constants.DefaultMCPStartupTimeoutSeconds * 1000 // convert seconds to milliseconds
	if workflowData.ToolsStartupTimeout > 0 {
		startupTimeoutMs = workflowData.ToolsStartupTimeout * 1000 // convert seconds to milliseconds
	}

	// Use tools.timeout if specified, otherwise default to DefaultToolTimeoutSeconds
	timeoutMs := constants.DefaultToolTimeoutSeconds * 1000 // convert seconds to milliseconds
	if workflowData.ToolsTimeout > 0 {
		timeoutMs = workflowData.ToolsTimeout * 1000 // convert seconds to milliseconds
	}

	// MCP_TIMEOUT: Timeout for MCP server startup
	stepLines = append(stepLines, fmt.Sprintf("          MCP_TIMEOUT: \"%d\"", startupTimeoutMs))

	// MCP_TOOL_TIMEOUT: Timeout for MCP tool execution
	stepLines = append(stepLines, fmt.Sprintf("          MCP_TOOL_TIMEOUT: \"%d\"", timeoutMs))

	// BASH_DEFAULT_TIMEOUT_MS: Default timeout for Bash commands
	stepLines = append(stepLines, fmt.Sprintf("          BASH_DEFAULT_TIMEOUT_MS: \"%d\"", timeoutMs))

	// BASH_MAX_TIMEOUT_MS: Maximum timeout for Bash commands
	stepLines = append(stepLines, fmt.Sprintf("          BASH_MAX_TIMEOUT_MS: \"%d\"", timeoutMs))

	applySafeOutputEnvToSlice(&stepLines, workflowData)

	// Add GH_AW_STARTUP_TIMEOUT environment variable (in seconds) if startup-timeout is specified
	if workflowData.ToolsStartupTimeout > 0 {
		stepLines = append(stepLines, fmt.Sprintf("          GH_AW_STARTUP_TIMEOUT: \"%d\"", workflowData.ToolsStartupTimeout))
	}

	// Add GH_AW_TOOL_TIMEOUT environment variable (in seconds) if timeout is specified
	if workflowData.ToolsTimeout > 0 {
		stepLines = append(stepLines, fmt.Sprintf("          GH_AW_TOOL_TIMEOUT: \"%d\"", workflowData.ToolsTimeout))
	}

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		stepLines = append(stepLines, fmt.Sprintf("          GH_AW_MAX_TURNS: %s", workflowData.EngineConfig.MaxTurns))
	}

	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, value))
		}
	}

	steps = append(steps, GitHubActionStep(stepLines))

	// Add cleanup step for network proxy hook files (if proxy was enabled)
	if workflowData.EngineConfig != nil && ShouldEnforceNetworkPermissions(workflowData.NetworkPermissions) {
		cleanupStep := GitHubActionStep{
			"      - name: Clean up network proxy hook files",
			"        if: always()",
			"        run: |",
			"          rm -rf .claude/hooks/network_permissions.py || true",
			"          rm -rf .claude/hooks || true",
			"          rm -rf .claude || true",
		}
		steps = append(steps, cleanupStep)
	}

	return steps
}

// GetLogParserScriptId returns the JavaScript script name for parsing Claude logs
func (e *ClaudeEngine) GetLogParserScriptId() string {
	return "parse_claude_log"
}

// GetErrorPatterns returns regex patterns for extracting error messages from Claude logs
func (e *ClaudeEngine) GetErrorPatterns() []ErrorPattern {
	// Claude uses common GitHub Actions workflow commands for error reporting
	// No engine-specific log formats to parse
	return GetCommonErrorPatterns()
}
