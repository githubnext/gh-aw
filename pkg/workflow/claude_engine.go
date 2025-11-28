package workflow

import (
	"fmt"
	"sort"
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
			experimental:           true,
			supportsToolsAllowlist: true,
			supportsHTTPTransport:  true, // Claude supports both stdio and HTTP transport
			supportsMaxTurns:       true, // Claude supports max-turns feature
			supportsWebFetch:       true, // Claude has built-in WebFetch support
			supportsWebSearch:      true, // Claude has built-in WebSearch support
			supportsFirewall:       true, // Claude supports AWF firewall
		},
	}
}

func (e *ClaudeEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	claudeLog.Printf("Generating installation steps for Claude engine: workflow=%s", workflowData.Name)

	var steps []GitHubActionStep

	// Add secret validation step - Claude supports both CLAUDE_CODE_OAUTH_TOKEN and ANTHROPIC_API_KEY as fallback
	secretValidation := GenerateMultiSecretValidationStep(
		[]string{"CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_API_KEY"},
		"Claude Code",
		"https://githubnext.github.io/gh-aw/reference/engines/#anthropic-claude-code",
	)
	steps = append(steps, secretValidation)

	// Use shared helper for standard npm installation
	npmSteps := BuildStandardNpmEngineInstallSteps(
		"@anthropic-ai/claude-code",
		string(constants.DefaultClaudeCodeVersion),
		"Install Claude Code CLI",
		"claude",
		workflowData,
	)

	// Install Node.js setup (first npm step)
	steps = append(steps, npmSteps[0])

	// Add AWF installation steps only if firewall is enabled
	if isFirewallEnabled(workflowData) {
		// Install AWF after Node.js setup but before Claude CLI installation
		firewallConfig := getFirewallConfig(workflowData)
		var awfVersion string
		if firewallConfig != nil {
			awfVersion = firewallConfig.Version
		}

		// Install AWF binary
		awfInstall := generateAWFInstallationStep(awfVersion)
		steps = append(steps, awfInstall)
	}

	// Install Claude CLI (remaining npm steps)
	steps = append(steps, npmSteps[1:]...)

	return steps
}

// GetDeclaredOutputFiles returns the output files that Claude may produce
func (e *ClaudeEngine) GetDeclaredOutputFiles() []string {
	return []string{}
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
		claudeLog.Printf("Using custom model: %s", workflowData.EngineConfig.Model)
		claudeArgs = append(claudeArgs, "--model", workflowData.EngineConfig.Model)
	}

	// Add max_turns if specified (in CLI it's max-turns)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		claudeLog.Printf("Setting max turns: %s", workflowData.EngineConfig.MaxTurns)
		claudeArgs = append(claudeArgs, "--max-turns", workflowData.EngineConfig.MaxTurns)
	}

	// Add MCP configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		claudeLog.Print("Adding MCP configuration")
		claudeArgs = append(claudeArgs, "--mcp-config", "/tmp/gh-aw/mcp-config/mcp-servers.json")
	}

	// Add allowed tools configuration
	// Note: Claude Code CLI v2.0.31 introduced a simpler --tools flag, but we continue to use
	// --allowed-tools because it provides fine-grained control needed by gh-aw:
	// - Specific bash commands: Bash(git:*), Bash(ls)
	// - MCP tool prefixes: mcp__github__issue_read
	// - Path-specific tools: Read(/tmp/gh-aw/cache-memory/*)
	// The --tools flag only supports basic tool names (e.g., "Bash,Edit,Read") without patterns.
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
		// Strip both possible prefixes (timeout_minutes or timeout-minutes)
		timeoutValue := strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout_minutes: ")
		timeoutValue = strings.TrimPrefix(timeoutValue, "timeout-minutes: ")
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %s", timeoutValue))
	} else {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %d", constants.DefaultAgenticWorkflowTimeoutMinutes)) // Default timeout for agentic workflows
	}

	// Build the run command
	stepLines = append(stepLines, "        run: |")
	stepLines = append(stepLines, "          set -o pipefail")
	stepLines = append(stepLines, "          # Execute Claude Code CLI with prompt from file")

	// Build the prompt command - prepend custom agent file content if specified (via imports)
	var promptCommand string
	if workflowData.AgentFile != "" {
		agentPath := ResolveAgentFilePath(workflowData.AgentFile)
		claudeLog.Printf("Using custom agent file: %s", workflowData.AgentFile)
		// Extract markdown body from custom agent file and prepend to prompt
		stepLines = append(stepLines, "          # Extract markdown body from custom agent file (skip frontmatter)")
		stepLines = append(stepLines, fmt.Sprintf("          AGENT_CONTENT=\"$(awk 'BEGIN{skip=1} /^---$/{if(skip){skip=0;next}else{skip=1;next}} !skip' %s)\"", agentPath))
		stepLines = append(stepLines, "          # Combine agent content with prompt")
		stepLines = append(stepLines, "          PROMPT_TEXT=\"$(printf '%s\\n\\n%s' \"$AGENT_CONTENT\" \"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\")\"")
		promptCommand = "\"$PROMPT_TEXT\""
	} else {
		promptCommand = "\"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\""
	}

	// Build the command string with proper argument formatting
	var claudeCommand string
	if isFirewallEnabled(workflowData) {
		// When firewall is enabled, use npx to run Claude CLI
		// This ensures the CLI is accessible within the AWF container
		claudeVersion := string(constants.DefaultClaudeCodeVersion)
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
			claudeVersion = workflowData.EngineConfig.Version
		}
		// Build command with npx -y for automatic download
		commandParts := []string{fmt.Sprintf("npx -y @anthropic-ai/claude-code@%s", claudeVersion)}
		commandParts = append(commandParts, claudeArgs...)
		commandParts = append(commandParts, promptCommand)
		claudeCommand = shellJoinArgs(commandParts)
		claudeLog.Printf("Using npx to run Claude CLI with firewall: %s", claudeVersion)
	} else {
		// Use claude command directly (installed via npm install -g)
		commandParts := []string{"claude"}
		commandParts = append(commandParts, claudeArgs...)
		commandParts = append(commandParts, promptCommand)
		claudeCommand = shellJoinArgs(commandParts)
	}

	// Conditionally wrap with AWF if firewall is enabled
	if isFirewallEnabled(workflowData) {
		// Build the AWF-wrapped command
		firewallConfig := getFirewallConfig(workflowData)
		var awfLogLevel = "info"
		if firewallConfig != nil && firewallConfig.LogLevel != "" {
			awfLogLevel = firewallConfig.LogLevel
		}

		// Get allowed domains (claude defaults + network permissions)
		allowedDomains := GetClaudeAllowedDomains(workflowData.NetworkPermissions)

		// Build AWF arguments
		var awfArgs []string
		awfArgs = append(awfArgs, "--env-all")
		awfArgs = append(awfArgs, "--container-workdir", "\"${GITHUB_WORKSPACE}\"")

		// Add mount arguments
		awfArgs = append(awfArgs, "--mount", "/tmp:/tmp:rw")
		awfArgs = append(awfArgs, "--mount", "\"${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:rw\"")

		awfArgs = append(awfArgs, "--allow-domains", allowedDomains)
		awfArgs = append(awfArgs, "--log-level", awfLogLevel)

		// Add custom args if specified
		if firewallConfig != nil && len(firewallConfig.Args) > 0 {
			awfArgs = append(awfArgs, firewallConfig.Args...)
		}

		// Build the full AWF command with multiline formatting
		stepLines = append(stepLines, fmt.Sprintf("          sudo -E awf %s \\", shellJoinArgs(awfArgs)))
		stepLines = append(stepLines, fmt.Sprintf("            -- %s \\", claudeCommand))
		stepLines = append(stepLines, fmt.Sprintf("            2>&1 | tee %s", shellEscapeArg(logFile)))
	} else {
		// Run claude command without AWF wrapper
		stepLines = append(stepLines, fmt.Sprintf("          %s 2>&1 | tee %s", claudeCommand, logFile))
	}

	// Add environment section - always include environment section for GH_AW_PROMPT
	stepLines = append(stepLines, "        env:")

	// Add both API keys - Claude Code CLI handles them separately and determines precedence
	stepLines = append(stepLines, "          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}")
	stepLines = append(stepLines, "          CLAUDE_CODE_OAUTH_TOKEN: ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}")

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

	// Add safe-inputs secrets to env for passthrough to MCP servers
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		safeInputsSecrets := collectSafeInputsSecrets(workflowData.SafeInputs)
		// Sort keys for consistent output
		var keys []string
		for key := range safeInputsSecrets {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, safeInputsSecrets[key]))
		}
	}

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}

// GetSquidLogsSteps returns the steps for collecting and uploading Squid logs
func (e *ClaudeEngine) GetSquidLogsSteps(workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	// Only add Squid logs collection and upload steps if firewall is enabled
	if isFirewallEnabled(workflowData) {
		claudeLog.Printf("Adding Squid logs collection steps for workflow: %s", workflowData.Name)
		squidLogsCollection := generateSquidLogsCollectionStep(workflowData.Name)
		steps = append(steps, squidLogsCollection)

		squidLogsUpload := generateSquidLogsUploadStep(workflowData.Name)
		steps = append(steps, squidLogsUpload)

		// Add firewall log parsing step to create step summary
		firewallLogParsing := generateFirewallLogParsingStep(workflowData.Name)
		steps = append(steps, firewallLogParsing)
	} else {
		claudeLog.Print("Firewall disabled, skipping Squid logs collection")
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
