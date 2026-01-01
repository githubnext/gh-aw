package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var copilotLog = logger.New("workflow:copilot_engine")

const logsFolder = "/tmp/gh-aw/sandbox/agent/logs/"

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
			supportsFirewall:       true,  // Copilot supports network firewalling via AWF
		},
	}
}

// GetDefaultDetectionModel returns the default model for threat detection
// Uses gpt-5-mini as a cost-effective model for detection tasks
func (e *CopilotEngine) GetDefaultDetectionModel() string {
	return string(constants.DefaultCopilotDetectionModel)
}

// GetInstallationSteps is implemented in copilot_engine_installation.go

func (e *CopilotEngine) GetDeclaredOutputFiles() []string {
	return []string{logsFolder}
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
	copilotLog.Printf("Generating execution steps for Copilot: workflow=%s, firewall=%v", workflowData.Name, isFirewallEnabled(workflowData))

	// Handle custom steps if they exist in engine config
	steps := InjectCustomEngineSteps(workflowData, e.convertStepToYAML)

	// Build copilot CLI arguments based on configuration
	var copilotArgs []string
	sandboxEnabled := isFirewallEnabled(workflowData) || isSRTEnabled(workflowData)
	if sandboxEnabled {
		// Simplified args for sandbox mode (AWF or SRT)
		copilotArgs = []string{"--add-dir", "/tmp/gh-aw/", "--log-level", "all", "--log-dir", logsFolder}

		// Always add workspace directory to --add-dir so Copilot CLI can access it
		// This allows Copilot CLI to discover agent files and access the workspace
		// Use double quotes to allow shell variable expansion
		copilotArgs = append(copilotArgs, "--add-dir", "\"${GITHUB_WORKSPACE}\"")
		copilotLog.Print("Added workspace directory to --add-dir")

		copilotLog.Print("Using firewall mode with simplified arguments")
	} else {
		// Original args for non-sandbox mode
		copilotArgs = []string{"--add-dir", "/tmp/", "--add-dir", "/tmp/gh-aw/", "--add-dir", "/tmp/gh-aw/agent/", "--log-level", "all", "--log-dir", logsFolder}
		copilotLog.Print("Using standard mode with full arguments")
	}

	// Add --disable-builtin-mcps to disable built-in MCP servers
	copilotArgs = append(copilotArgs, "--disable-builtin-mcps")

	// Add model if specified
	// Model can be configured via:
	// 1. Explicit model in workflow config (highest priority)
	// 2. GH_AW_MODEL_AGENT_COPILOT environment variable (set via GitHub Actions variables)
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""
	if modelConfigured {
		copilotLog.Printf("Using custom model: %s", workflowData.EngineConfig.Model)
		copilotArgs = append(copilotArgs, "--model", workflowData.EngineConfig.Model)
	}

	// Add --agent flag if custom agent file is specified (via imports)
	// Copilot CLI expects agent identifier (filename without extension), not full path
	if workflowData.AgentFile != "" {
		agentIdentifier := ExtractAgentIdentifier(workflowData.AgentFile)
		copilotLog.Printf("Using custom agent: %s (identifier: %s)", workflowData.AgentFile, agentIdentifier)
		copilotArgs = append(copilotArgs, "--agent", agentIdentifier)
	}

	// Add tool permission arguments based on configuration
	toolArgs := e.computeCopilotToolArguments(workflowData.Tools, workflowData.SafeOutputs, workflowData.SafeInputs, workflowData)
	if len(toolArgs) > 0 {
		copilotLog.Printf("Adding %d tool permission arguments", len(toolArgs))
	}
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
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.Edit != nil {
		copilotArgs = append(copilotArgs, "--allow-all-paths")
	}

	// Add custom args from engine configuration before the prompt
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		copilotArgs = append(copilotArgs, workflowData.EngineConfig.Args...)
	}

	// Add prompt argument - inline for sandbox modes, variable for non-sandbox
	if sandboxEnabled {
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
		fmt.Fprintf(&mkdirCommands, "mkdir -p %s\n", dir)
	}

	// Build the copilot command
	var copilotCommand string

	// Determine if we need to conditionally add --model flag based on environment variable
	needsModelFlag := !modelConfigured
	// Check if this is a detection job (has no SafeOutputs config)
	isDetectionJob := workflowData.SafeOutputs == nil
	var modelEnvVar string
	if isDetectionJob {
		modelEnvVar = constants.EnvVarModelDetectionCopilot
	} else {
		modelEnvVar = constants.EnvVarModelAgentCopilot
	}

	if sandboxEnabled {
		// Build base command
		var baseCommand string
		// For SRT: use locally installed package without -y flag to avoid internet fetch
		// For AWF: use the installed binary directly
		if isSRTEnabled(workflowData) {
			// Use node explicitly to invoke copilot CLI to ensure env vars propagate correctly through sandbox
			// The .bin/copilot shell wrapper doesn't properly pass environment variables through bubblewrap
			// Environment variables are explicitly exported in the SRT wrapper to propagate through sandbox
			baseCommand = fmt.Sprintf("node ./node_modules/.bin/copilot %s", shellJoinArgs(copilotArgs))
		} else {
			// AWF - use the copilot binary installed by the installer script
			// The binary is mounted into the AWF container from /usr/local/bin/copilot
			baseCommand = fmt.Sprintf("/usr/local/bin/copilot %s", shellJoinArgs(copilotArgs))
		}

		// Add conditional model flag if needed
		if needsModelFlag {
			copilotCommand = fmt.Sprintf(`%s${%s:+ --model "$%s"}`, baseCommand, modelEnvVar, modelEnvVar)
		} else {
			copilotCommand = baseCommand
		}
	} else {
		// When sandbox is disabled, use unpinned copilot command
		baseCommand := fmt.Sprintf("copilot %s", shellJoinArgs(copilotArgs))

		// Add conditional model flag if needed
		if needsModelFlag {
			copilotCommand = fmt.Sprintf(`%s${%s:+ --model "$%s"}`, baseCommand, modelEnvVar, modelEnvVar)
		} else {
			copilotCommand = baseCommand
		}
	}

	// Conditionally wrap with sandbox (AWF or SRT)
	var command string
	if isSRTEnabled(workflowData) {
		// Build the SRT-wrapped command
		copilotLog.Print("Using Sandbox Runtime (SRT) for execution")

		agentConfig := getAgentConfig(workflowData)

		// Generate SRT config JSON
		srtConfigJSON, err := generateSRTConfigJSON(workflowData)
		if err != nil {
			copilotLog.Printf("Error generating SRT config: %v", err)
			// Fallback to empty config
			srtConfigJSON = "{}"
		}

		// Check if custom command is specified
		if agentConfig != nil && agentConfig.Command != "" {
			// Use custom command for SRT
			copilotLog.Printf("Using custom SRT command: %s", agentConfig.Command)

			// Build args list with custom args appended
			var srtArgs []string
			if len(agentConfig.Args) > 0 {
				srtArgs = append(srtArgs, agentConfig.Args...)
				copilotLog.Printf("Added %d custom args from agent config", len(agentConfig.Args))
			}

			// Build the command with custom SRT command
			// The custom command should handle wrapping copilot with SRT
			command = fmt.Sprintf(`set -o pipefail
%s %s -- %s 2>&1 | tee %s`, agentConfig.Command, shellJoinArgs(srtArgs), copilotCommand, shellEscapeArg(logFile))
		} else {
			// Create the Node.js wrapper script for SRT (standard installation)
			srtWrapperScript := generateSRTWrapperScript(copilotCommand, srtConfigJSON, logFile, logsFolder)
			command = srtWrapperScript
		}
	} else if isFirewallEnabled(workflowData) {
		// Build the AWF-wrapped command - no mkdir needed, AWF handles it
		firewallConfig := getFirewallConfig(workflowData)
		agentConfig := getAgentConfig(workflowData)
		var awfLogLevel = "info"
		if firewallConfig != nil && firewallConfig.LogLevel != "" {
			awfLogLevel = firewallConfig.LogLevel
		}

		// Check if safe-inputs is enabled to include host.docker.internal in allowed domains
		hasSafeInputs := IsSafeInputsEnabled(workflowData.SafeInputs, workflowData)

		// Get allowed domains (copilot defaults + network permissions + host.docker.internal if safe-inputs enabled)
		allowedDomains := GetCopilotAllowedDomainsWithSafeInputs(workflowData.NetworkPermissions, hasSafeInputs)

		// Build AWF arguments: mount points + standard flags + custom args from config
		var awfArgs []string
		awfArgs = append(awfArgs, "--env-all")

		// Set container working directory to match GITHUB_WORKSPACE
		// This ensures pwd inside the container matches what the prompt tells the AI
		awfArgs = append(awfArgs, "--container-workdir", "\"${GITHUB_WORKSPACE}\"")
		copilotLog.Print("Set container working directory to GITHUB_WORKSPACE")

		// Add mount arguments for required paths
		// Always mount /tmp for temporary files and cache
		awfArgs = append(awfArgs, "--mount", "/tmp:/tmp:rw")

		// Always mount the workspace directory so Copilot CLI can access it
		// Use double quotes to allow shell variable expansion
		awfArgs = append(awfArgs, "--mount", "\"${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:rw\"")
		copilotLog.Print("Added workspace mount to AWF")

		// Mount gh CLI binary from host so it's available inside the container
		// This allows workflows to use gh CLI commands within the sandboxed environment
		awfArgs = append(awfArgs, "--mount", "/usr/bin/date:/usr/bin/date:ro")
		awfArgs = append(awfArgs, "--mount", "/usr/bin/gh:/usr/bin/gh:ro")
		awfArgs = append(awfArgs, "--mount", "/usr/bin/yq:/usr/bin/yq:ro")

		// Mount copilot CLI binary from /usr/local/bin (where the installer script places it)
		awfArgs = append(awfArgs, "--mount", "/usr/local/bin/copilot:/usr/local/bin/copilot:ro")

		// Mount .copilot directory for MCP configuration
		// XDG_CONFIG_HOME is set to /home/runner, so Copilot CLI looks for config at /home/runner/.copilot/mcp-config.json
		// Mount host /home/runner/.copilot to container /home/runner/.copilot with read-write access for CLI state/logs
		awfArgs = append(awfArgs, "--mount", "/home/runner/.copilot:/home/runner/.copilot:rw")
		copilotLog.Print("Added gh CLI, copilot binary, and .copilot config directory mounts to AWF container")

		// Add custom mounts from agent config if specified
		if agentConfig != nil && len(agentConfig.Mounts) > 0 {
			// Sort mounts for consistent output
			sortedMounts := make([]string, len(agentConfig.Mounts))
			copy(sortedMounts, agentConfig.Mounts)
			sort.Strings(sortedMounts)

			for _, mount := range sortedMounts {
				awfArgs = append(awfArgs, "--mount", mount)
			}
			copilotLog.Printf("Added %d custom mounts from agent config", len(sortedMounts))
		}

		awfArgs = append(awfArgs, "--allow-domains", allowedDomains)
		awfArgs = append(awfArgs, "--log-level", awfLogLevel)
		awfArgs = append(awfArgs, "--proxy-logs-dir", "/tmp/gh-aw/sandbox/firewall/logs")

		// Pin AWF Docker image version to match the installed binary version
		awfImageTag := getAWFImageTag(firewallConfig)
		awfArgs = append(awfArgs, "--image-tag", awfImageTag)
		copilotLog.Printf("Pinned AWF image tag to %s", awfImageTag)

		// Add custom args if specified in firewall config
		if firewallConfig != nil && len(firewallConfig.Args) > 0 {
			awfArgs = append(awfArgs, firewallConfig.Args...)
		}

		// Add custom args from agent config if specified
		if agentConfig != nil && len(agentConfig.Args) > 0 {
			awfArgs = append(awfArgs, agentConfig.Args...)
			copilotLog.Printf("Added %d custom args from agent config", len(agentConfig.Args))
		}

		// Determine the AWF command to use (custom or standard)
		var awfCommand string
		if agentConfig != nil && agentConfig.Command != "" {
			awfCommand = agentConfig.Command
			copilotLog.Printf("Using custom AWF command: %s", awfCommand)
		} else {
			awfCommand = "sudo -E awf"
			copilotLog.Print("Using standard AWF command")
		}

		// Build the full AWF command with proper argument separation
		// AWF v0.2.0 uses -- to separate AWF args from the actual command
		// The command arguments should be passed as individual shell arguments, not as a single string
		command = fmt.Sprintf(`set -o pipefail
%s %s \
  -- %s \
  2>&1 | tee %s`, awfCommand, shellJoinArgs(awfArgs), copilotCommand, shellEscapeArg(logFile))
	} else {
		// Run copilot command without AWF wrapper
		command = fmt.Sprintf(`set -o pipefail
COPILOT_CLI_INSTRUCTION="$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"
%s%s 2>&1 | tee %s`, mkdirCommands.String(), copilotCommand, logFile)
	}

	// Use COPILOT_GITHUB_TOKEN
	// If github-token is specified at workflow level, use that instead
	var copilotGitHubToken string
	if workflowData.GitHubToken != "" {
		copilotGitHubToken = workflowData.GitHubToken
	} else {
		copilotGitHubToken = "${{ secrets.COPILOT_GITHUB_TOKEN }}"
	}

	env := map[string]string{
		"XDG_CONFIG_HOME":           "/home/runner",
		"COPILOT_AGENT_RUNNER_TYPE": "STANDALONE",
		"COPILOT_GITHUB_TOKEN":      copilotGitHubToken,
		"GITHUB_STEP_SUMMARY":       "${{ env.GITHUB_STEP_SUMMARY }}",
		"GITHUB_HEAD_REF":           "${{ github.head_ref }}",
		"GITHUB_REF_NAME":           "${{ github.ref_name }}",
		"GITHUB_WORKSPACE":          "${{ github.workspace }}",
	}

	// Always add GH_AW_PROMPT for agentic workflows
	env["GH_AW_PROMPT"] = "/tmp/gh-aw/aw-prompts/prompt.txt"

	// Add GH_AW_MCP_CONFIG for MCP server configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "/home/runner/.copilot/mcp-config.json"
	}

	if hasGitHubTool(workflowData.ParsedTools) {
		customGitHubToken := getGitHubToken(workflowData.Tools["github"])
		// Use effective token with precedence: custom > top-level > default
		effectiveToken := getEffectiveGitHubToken(customGitHubToken, workflowData.GitHubToken)
		env["GITHUB_MCP_SERVER_TOKEN"] = effectiveToken
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

	// Add model environment variable if model is not explicitly configured
	// This allows users to configure the default model via GitHub Actions variables
	// Use different env vars for agent vs detection jobs
	if workflowData.EngineConfig == nil || workflowData.EngineConfig.Model == "" {
		// Check if this is a detection job (has no SafeOutputs config)
		isDetectionJob := workflowData.SafeOutputs == nil
		if isDetectionJob {
			// For detection, use detection-specific env var (no builtin default, CLI will use its own)
			env[constants.EnvVarModelDetectionCopilot] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelDetectionCopilot)
		} else {
			// For agent execution, use agent-specific env var
			env[constants.EnvVarModelAgentCopilot] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelAgentCopilot)
		}
	}

	// Add custom environment variables from engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			env[key] = value
		}
	}

	// Add custom environment variables from agent config
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		for key, value := range agentConfig.Env {
			env[key] = value
		}
		copilotLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
	}

	// Add HTTP MCP header secrets to env for passthrough
	headerSecrets := collectHTTPMCPHeaderSecrets(workflowData.Tools)
	for varName, secretExpr := range headerSecrets {
		// Only add if not already in env
		if _, exists := env[varName]; !exists {
			env[varName] = secretExpr
		}
	}

	// Add safe-inputs secrets to env for passthrough to MCP servers
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		safeInputsSecrets := collectSafeInputsSecrets(workflowData.SafeInputs)
		for varName, secretExpr := range safeInputsSecrets {
			// Only add if not already in env
			if _, exists := env[varName]; !exists {
				env[varName] = secretExpr
			}
		}
	}

	// Generate the step for Copilot CLI execution
	stepName := "Execute GitHub Copilot CLI"
	var stepLines []string

	stepLines = append(stepLines, fmt.Sprintf("      - name: %s", stepName))
	stepLines = append(stepLines, "        id: agentic_execution")

	// Add tool arguments comment before the run section
	toolArgsComment := e.generateCopilotToolArgumentsComment(workflowData.Tools, workflowData.SafeOutputs, workflowData.SafeInputs, workflowData, "        ")
	if toolArgsComment != "" {
		// Split the comment into lines and add each line
		commentLines := strings.Split(strings.TrimSuffix(toolArgsComment, "\n"), "\n")
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

	// Format step with command and environment variables using shared helper
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, env)

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}

// GetFirewallLogsCollectionStep returns the step for collecting firewall logs (before secret redaction)
// No longer needed since we know where the logs are in the sandbox folder structure
// RenderMCPConfig is implemented in copilot_mcp.go

// ParseLogMetrics is implemented in copilot_logs.go

// extractToolCallSizes is implemented in copilot_logs.go

// processToolCalls is implemented in copilot_logs.go

// parseCopilotToolCallsWithSequence is implemented in copilot_logs.go

// GetLogParserScriptId is implemented in copilot_logs.go

// GetLogFileForParsing is implemented in copilot_logs.go

// GetFirewallLogsCollectionStep is implemented in copilot_logs.go

// GetSquidLogsSteps is implemented in copilot_logs.go

// GetCleanupStep is implemented in copilot_logs.go

// computeCopilotToolArguments is implemented in copilot_engine_tools.go

// generateCopilotToolArgumentsComment is implemented in copilot_engine_tools.go

// GetErrorPatterns is implemented in copilot_engine_tools.go

// generateAWFInstallationStep is implemented in copilot_engine_installation.go


// GenerateCopilotInstallerSteps creates GitHub Actions steps for installing Copilot CLI using the official installer script
// Parameters:
//   - version: The Copilot CLI version to install (e.g., "0.0.369" or "v0.0.369")
//   - stepName: The name to display for the install step (e.g., "Install GitHub Copilot CLI")
//
// Returns steps for installing Copilot CLI using the official install.sh script from the Copilot CLI repository.
// The script is downloaded from https://raw.githubusercontent.com/github/copilot-cli/main/install.sh
// and executed with the VERSION environment variable set.
//
// Security Implementation:
//  1. Downloads the official installer script from the Copilot CLI repository
//  2. Saves script to a temporary file before execution (not piped directly to bash)
//  3. Uses the official script which includes platform detection and error handling
//
// Version Handling:
// The VERSION environment variable is used by the install.sh script.
// The script automatically adds 'v' prefix if not present.
// Examples:
//   - VERSION=0.0.369 → downloads and installs v0.0.369
//   - VERSION=v0.0.369 → downloads and installs v0.0.369
//   - VERSION=1.2.3 → downloads and installs v1.2.3
