package workflow

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var claudeLog = logger.New("workflow:claude_engine")

// ClaudeEngine represents the Claude Code agentic engine
type ClaudeEngine struct {
	BaseEngine
}

func NewClaudeEngine() *ClaudeEngine {
	return &ClaudeEngine{
		BaseEngine: BaseEngine{
			id:                       "claude",
			displayName:              "Claude Code",
			description:              "Uses Claude Code with full MCP tool support and allow-listing",
			experimental:             false,
			supportsToolsAllowlist:   true,
			supportsMaxTurns:         true,  // Claude supports max-turns feature
			supportsMaxContinuations: false, // Claude Code does not support --max-autopilot-continues-style continuation
			supportsWebSearch:        true,  // Claude has built-in WebSearch support
			supportsNativeAgentFile:  false, // Claude does not support agent file natively; the compiler prepends the agent file content to prompt.txt
			supportsBareMode:         true,  // Claude CLI supports --bare
			dedicatedLLMGatewayPort:  constants.ClaudeLLMGatewayPort,
		},
	}
}

// GetModelEnvVarName returns the native environment variable name that the Claude Code CLI uses
// for model selection. Setting ANTHROPIC_MODEL is equivalent to passing --model to the CLI.
func (e *ClaudeEngine) GetModelEnvVarName() string {
	return constants.ClaudeCLIModelEnvVar
}

// GetAPMTarget returns "claude" so that apm-action packs Claude-specific primitives.
func (e *ClaudeEngine) GetAPMTarget() string {
	return "claude"
}

// GetRequiredSecretNames returns the list of secrets required by the Claude engine
// This includes ANTHROPIC_API_KEY and optionally MCP_GATEWAY_API_KEY and mcp-scripts secrets
func (e *ClaudeEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	return append([]string{"ANTHROPIC_API_KEY"}, collectCommonMCPSecrets(workflowData)...)
}

// GetSecretValidationStep returns the secret validation step for the Claude engine.
// Returns an empty step if custom command is specified.
func (e *ClaudeEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return BuildDefaultSecretValidationStep(
		workflowData,
		[]string{"ANTHROPIC_API_KEY"},
		"Claude Code",
		"https://github.github.com/gh-aw/reference/engines/#anthropic-claude-code",
	)
}

func (e *ClaudeEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	claudeLog.Printf("Generating installation steps for Claude engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		claudeLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	// Use version from engine config if provided, otherwise default to pinned version
	version := string(constants.DefaultClaudeCodeVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	// Claude Code requires post-install scripts (native binaries) so --ignore-scripts must
	// NOT be passed. This is intentionally different from other engine installs.
	npmSteps := GenerateNpmInstallSteps(
		"@anthropic-ai/claude-code",
		version,
		"Install Claude Code CLI",
		"claude",
		true, // Include Node.js setup
		true, // Claude Code requires post-install scripts for native binaries
	)
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

// GetDeclaredOutputFiles returns the output files that Claude may produce
func (e *ClaudeEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetAgentManifestFiles returns Claude-specific instruction files that should be
// treated as security-sensitive manifests.  Modifying these files can change the
// agent's instructions, guidelines, or permissions on the next run.
// CLAUDE.md is the primary per-project instruction file; AGENTS.md is the
// cross-engine convention that Claude Code also reads.
func (e *ClaudeEngine) GetAgentManifestFiles() []string {
	return []string{"CLAUDE.md", "AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns Claude-specific config directory prefixes.
// The .claude/ directory contains settings, custom commands, hooks, and other
// engine configuration that could affect agent behaviour.
func (e *ClaudeEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".claude/"}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Claude
func (e *ClaudeEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	claudeLog.Printf("Generating execution steps for Claude engine: workflow=%s, firewall=%v", workflowData.Name, isFirewallEnabled(workflowData))

	var steps []GitHubActionStep

	// Build claude CLI arguments based on configuration
	var claudeArgs []string
	toolsWithMountedCLIs := withMountedCLIShellCommandsInRestrictedBash(workflowData)

	// Add print flag for non-interactive mode
	claudeArgs = append(claudeArgs, "--print")

	// Disable Chrome integration for security and deterministic execution
	claudeArgs = append(claudeArgs, "--no-chrome")

	// Model is always passed via the native ANTHROPIC_MODEL environment variable when configured.
	// This avoids embedding the value directly in the shell command (which fails template injection
	// validation for GitHub Actions expressions like ${{ inputs.model }}).
	// Fallback for unconfigured model uses GH_AW_MODEL_AGENT_CLAUDE with shell expansion.
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	// Add max_turns if specified (in CLI it's max-turns)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		claudeLog.Printf("Setting max turns: %s", workflowData.EngineConfig.MaxTurns)
		claudeArgs = append(claudeArgs, "--max-turns", workflowData.EngineConfig.MaxTurns)
	}

	// Add MCP configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		claudeLog.Print("Adding MCP configuration")
		claudeArgs = append(claudeArgs, "--mcp-config", "${{ runner.temp }}/gh-aw/mcp-config/mcp-servers.json")
	}

	// Add allowed tools configuration
	// Note: Claude Code CLI v2.0.31 introduced a simpler --tools flag, but we continue to use
	// --allowed-tools because it provides fine-grained control needed by gh-aw:
	// - Specific bash commands: Bash(git:*), Bash(ls)
	// - MCP tool prefixes: mcp__github__issue_read
	// - Path-specific tools: Read(/tmp/gh-aw/cache-memory/*)
	// The --tools flag only supports basic tool names (e.g., "Bash,Edit,Read") without patterns.
	allowedTools := e.computeAllowedClaudeToolsString(toolsWithMountedCLIs, workflowData.SafeOutputs, workflowData.CacheMemoryConfig, workflowData.MCPScripts)
	if allowedTools != "" {
		claudeArgs = append(claudeArgs, "--allowed-tools", allowedTools)
	}

	// Add debug-file flag to write debug logs directly to file
	// This implicitly enables debug mode and provides cleaner, more reliable log capture
	// than shell redirection with 2>&1 | tee
	claudeArgs = append(claudeArgs, "--debug-file", logFile)

	// Always add verbose flag for enhanced debugging output
	claudeArgs = append(claudeArgs, "--verbose")

	// Add permission mode for non-interactive execution.
	//
	// We use "acceptEdits" by default so that Claude Code honours --allowed-tools as
	// the effective MCP tool boundary.  In "bypassPermissions" mode the allowlist is
	// silently ignored — every tool exposed by the MCP gateway is reachable regardless
	// of the workflow's declared tool configuration.
	//
	// Exception: when the workflow grants unrestricted bash access (bash: "*"), the
	// agent can reach any tool via the shell regardless, so --allowed-tools provides
	// no meaningful security boundary.  In that case we switch back to
	// "bypassPermissions" which auto-approves all permission requests and produces a
	// smoother headless execution experience.
	permissionMode := "acceptEdits"
	if hasBashWildcardInTools(workflowData.Tools) {
		claudeLog.Print("Unrestricted bash detected: using bypassPermissions mode")
		permissionMode = "bypassPermissions"
	}
	claudeArgs = append(claudeArgs, "--permission-mode", permissionMode)

	// Add output format for structured output
	// Use "stream-json" to output JSONL format (newline-delimited JSON objects)
	// This format is compatible with the log parser which expects either JSON array or JSONL
	claudeArgs = append(claudeArgs, "--output-format", "stream-json")

	// Add --bare when bare mode is enabled to suppress automatic loading of memory
	// files (CLAUDE.md, ~/.claude/) and other context injections.
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Bare {
		claudeLog.Print("Bare mode enabled: adding --bare")
		claudeArgs = append(claudeArgs, "--bare")
	}

	// Add custom args from engine configuration before the prompt
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		claudeArgs = append(claudeArgs, workflowData.EngineConfig.Args...)
	}

	// The prompt is always read from prompt.txt, which is assembled by the compiler in the
	// activation job.  For engines that do not support native agent-file handling (including
	// Claude), the compiler prepends the agent file content to prompt.txt so no special
	// shell variable juggling is needed here.
	promptCommand := `"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"`

	// Build the command string with proper argument formatting
	// Determine which command to use
	var commandName string
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
		claudeLog.Printf("Using custom command: %s", commandName)
	} else {
		// Use regular claude command - PATH is inherited via --env-all in AWF mode
		commandName = "claude"
	}

	commandParts := []string{commandName}
	commandParts = append(commandParts, claudeArgs...)

	// Join command parts (excluding the prompt) with proper escaping.
	// The prompt command is appended raw after shellJoinArgs because it contains
	// shell variable references ("$(cat ...)") that must NOT be escaped —
	// single-quoting them would prevent shell expansion at runtime.
	claudeCommand := fmt.Sprintf("%s %s", shellJoinArgs(commandParts), promptCommand)

	// When model is not configured, use the GH_AW_MODEL_AGENT_CLAUDE fallback env var
	// via shell expansion so users can set a default via GitHub Actions variables.
	// When model IS configured, ANTHROPIC_MODEL is set in the env block (see below) and the
	// Claude CLI reads it natively - no --model flag in the shell command needed.
	if !modelConfigured {
		isDetectionJob := workflowData.SafeOutputs == nil
		var modelEnvVar string
		if isDetectionJob {
			modelEnvVar = constants.EnvVarModelDetectionClaude
		} else {
			modelEnvVar = constants.EnvVarModelAgentClaude
		}
		claudeCommand = fmt.Sprintf(`%s${%s:+ --model "$%s"}`, claudeCommand, modelEnvVar, modelEnvVar)
	}

	// Build the full command based on whether firewall is enabled
	var command string
	if isFirewallEnabled(workflowData) {
		// Build the AWF-wrapped command using helper function
		// Get allowed domains (Claude defaults + network permissions + HTTP MCP server URLs + runtime ecosystem domains)
		allowedDomains := GetClaudeAllowedDomainsWithToolsAndRuntimes(workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
		// Add GHES/custom API target domains to the firewall allow-list when engine.api-target is set
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
			allowedDomains = mergeAPITargetDomains(allowedDomains, workflowData.EngineConfig.APITarget)
		}

		// Build AWF command with all configuration
		// AWF v0.15.0+ uses chroot mode by default, providing transparent access to host binaries
		// AWF with --enable-chroot and --env-all handles most PATH setup natively:
		// - GOROOT, JAVA_HOME, etc. are handled via AWF_HOST_PATH and entrypoint.sh
		// However, npm-installed CLIs (like claude) need hostedtoolcache bin directories in PATH.
		// We prepend GetNpmBinPathSetup() to the engine command so it runs inside the AWF container.
		npmPathSetup := GetNpmBinPathSetup()
		claudeCommandWithPath := fmt.Sprintf(`%s && %s`, npmPathSetup, claudeCommand)
		// Add MCP CLI bin directory to PATH when mount-as-clis is enabled
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			claudeCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, claudeCommandWithPath)
		}

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "claude",
			EngineCommand:  claudeCommandWithPath, // Command with npm PATH setup runs inside AWF
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        true, // Claude Code CLI requires TTY
			AllowedDomains: allowedDomains,
			PathSetup:      "touch " + AgentStepSummaryPath, // Runs BEFORE AWF on the host
			// Exclude every env var whose step-env value is a secret so the agent
			// cannot read raw token values via bash tools (env / printenv).
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"ANTHROPIC_API_KEY"}),
		})
	} else {
		// Run Claude command without AWF wrapper
		// Note: Claude Code CLI writes debug logs to --debug-file and JSON output to stdout
		// Use tee to capture stdout (stream-json output) to the log file while also displaying on console
		// The combined output (debug logs + JSON) will be in the log file for parsing
		// PATH is already set correctly by actions/setup-* steps which prepend to PATH
		command = fmt.Sprintf(`set -o pipefail
          touch %s
          (umask 177 && touch %s)
          # Execute Claude Code CLI with prompt from file
          %s 2>&1 | tee -a %s`, AgentStepSummaryPath, logFile, claudeCommand, logFile)
	}

	// Build environment variables map
	env := map[string]string{
		"ANTHROPIC_API_KEY":       "${{ secrets.ANTHROPIC_API_KEY }}",
		"DISABLE_TELEMETRY":       "1",
		"DISABLE_ERROR_REPORTING": "1",
		"DISABLE_BUG_COMMAND":     "1",
		"GH_AW_PROMPT":            "/tmp/gh-aw/aw-prompts/prompt.txt",
		// Tag the step as a GitHub AW agentic execution for discoverability by agents
		"GITHUB_AW": "true",
		// Override GITHUB_STEP_SUMMARY with a path that exists inside the sandbox.
		// The runner's original path is unreachable within the AWF isolated filesystem;
		// we create this file before the agent starts and append it to the real
		// $GITHUB_STEP_SUMMARY after secret redaction.
		"GITHUB_STEP_SUMMARY": AgentStepSummaryPath,
		"GITHUB_WORKSPACE":    "${{ github.workspace }}",
	}
	// Indicate the phase: "agent" for the main run, "detection" for threat detection
	// Include the compiler version so agents can identify which gh-aw version generated the workflow
	if workflowData.IsDetectionRun {
		env["GH_AW_PHASE"] = "detection"
	} else {
		env["GH_AW_PHASE"] = "agent"
	}
	if IsRelease() {
		env["GH_AW_VERSION"] = GetVersion()
	} else {
		env["GH_AW_VERSION"] = "dev"
	}

	// Add GH_AW_MCP_CONFIG for MCP server configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "${{ runner.temp }}/gh-aw/mcp-config/mcp-servers.json"
	}

	// In sandbox (AWF) mode, set git identity environment variables so the first git commit
	// succeeds inside the container. AWF's --env-all forwards these to the container, ensuring
	// git does not rely on the host-side ~/.gitconfig which is not visible in the sandbox.
	if isFirewallEnabled(workflowData) {
		maps.Copy(env, getGitIdentityEnvVars())
	}

	// Set timeout environment variables for Claude Code
	// Use tools.startup-timeout if specified, otherwise default to DefaultMCPStartupTimeout
	// For expressions, fall back to default (can't compute ms value at compile time)
	startupTimeoutMs := int(constants.DefaultMCPStartupTimeout / time.Millisecond)
	if n := templatableIntValue(&workflowData.ToolsStartupTimeout); n > 0 {
		startupTimeoutMs = n * 1000 // convert seconds to milliseconds
	}

	// Use tools.timeout if specified, otherwise default to DefaultToolTimeout
	// For expressions, fall back to default (can't compute ms value at compile time)
	timeoutMs := int(constants.DefaultToolTimeout / time.Millisecond)
	if n := templatableIntValue(&workflowData.ToolsTimeout); n > 0 {
		timeoutMs = n * 1000 // convert seconds to milliseconds
	}

	env["MCP_TIMEOUT"] = strconv.Itoa(startupTimeoutMs)
	env["MCP_TOOL_TIMEOUT"] = strconv.Itoa(timeoutMs)
	env["BASH_DEFAULT_TIMEOUT_MS"] = strconv.Itoa(timeoutMs)
	env["BASH_MAX_TIMEOUT_MS"] = strconv.Itoa(timeoutMs)

	// Add GH_AW_SAFE_OUTPUTS if output is needed
	applySafeOutputEnvToMap(env, workflowData)

	// Add GH_AW_STARTUP_TIMEOUT environment variable (in seconds) if startup-timeout is specified
	// Supports both literal integers and GitHub Actions expressions (e.g. "${{ inputs.startup-timeout }}")
	if workflowData.ToolsStartupTimeout != "" {
		env["GH_AW_STARTUP_TIMEOUT"] = workflowData.ToolsStartupTimeout
	}

	// Add GH_AW_TOOL_TIMEOUT environment variable (in seconds) if timeout is specified
	// Supports both literal integers and GitHub Actions expressions (e.g. "${{ inputs.tool-timeout }}")
	if workflowData.ToolsTimeout != "" {
		env["GH_AW_TOOL_TIMEOUT"] = workflowData.ToolsTimeout
	}

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GH_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	}

	// Set the model environment variable.
	// When model is configured, use the native ANTHROPIC_MODEL env var - the Claude CLI reads it
	// directly, avoiding the need to embed the value in the shell command (which would fail
	// template injection validation for GitHub Actions expressions like ${{ inputs.model }}).
	// When model is not configured, fall back to GH_AW_MODEL_AGENT/DETECTION_CLAUDE so users
	// can set a default via GitHub Actions variables.
	if modelConfigured {
		claudeLog.Printf("Setting %s env var for model: %s", constants.ClaudeCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.ClaudeCLIModelEnvVar] = workflowData.EngineConfig.Model
	} else {
		// No model configured - use fallback GitHub variable with shell expansion
		isDetectionJob := workflowData.SafeOutputs == nil
		if isDetectionJob {
			env[constants.EnvVarModelDetectionClaude] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelDetectionClaude)
		} else {
			env[constants.EnvVarModelAgentClaude] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelAgentClaude)
		}
	}

	// Add custom environment variables from engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}

	// Add custom environment variables from agent config
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
		claudeLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}

	// Add mcp-scripts secrets to env for passthrough to MCP servers
	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		mcpScriptsSecrets := collectMCPScriptsSecrets(workflowData.MCPScripts)
		for varName, secretExpr := range mcpScriptsSecrets {
			// Only add if not already in env
			if _, exists := env[varName]; !exists {
				env[varName] = secretExpr
			}
		}
	}

	// Generate the step for Claude CLI execution
	stepName := "Execute Claude Code CLI"
	var stepLines []string

	stepLines = append(stepLines, "      - name: "+stepName)
	stepLines = append(stepLines, "        id: agentic_execution")

	// Add allowed tools comment before the run section
	// Reuse the already-computed allowedTools string (computed earlier for --allowed-tools flag)
	// to avoid redundant allocations from calling computeAllowedClaudeToolsString twice.
	allowedToolsComment := e.generateAllowedToolsComment(allowedTools, "        ")
	if allowedToolsComment != "" {
		// Split the comment into lines and add each line
		commentLines := strings.Split(strings.TrimSuffix(allowedToolsComment, "\n"), "\n")
		stepLines = append(stepLines, commentLines...)
	}

	// Add timeout at step level (GitHub Actions standard)
	if workflowData.TimeoutMinutes != "" {
		// Strip timeout-minutes prefix
		timeoutValue := strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout-minutes: ")
		stepLines = append(stepLines, "        timeout-minutes: "+timeoutValue)
	} else {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %d", int(constants.DefaultAgenticWorkflowTimeout/time.Minute))) // Default timeout for agentic workflows
	}

	// Filter environment variables to only include allowed secrets
	// This is a security measure to prevent exposing unnecessary secrets to the AWF container
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)

	// Inject GH_TOKEN for CLI proxy (added after filtering since it uses a special
	// fallback expression that is always allowed when cli-proxy is enabled)
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)

	// Format step with command and filtered environment variables using shared helper
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}

// GetLogParserScriptId returns the JavaScript script name for parsing Claude logs
func (e *ClaudeEngine) GetLogParserScriptId() string {
	return "parse_claude_log"
}

// GetFirewallLogsCollectionStep returns the step for collecting firewall logs (before secret redaction)
// No longer needed since we know where the logs are in the sandbox folder structure
func (e *ClaudeEngine) GetFirewallLogsCollectionStep(workflowData *WorkflowData) []GitHubActionStep {
	// Collection step removed - firewall logs are now at a known location
	return []GitHubActionStep{}
}

// GetSquidLogsSteps returns the steps for uploading and parsing Squid logs (after secret redaction)
func (e *ClaudeEngine) GetSquidLogsSteps(workflowData *WorkflowData) []GitHubActionStep {
	return defaultGetSquidLogsSteps(workflowData, claudeLog)
}
