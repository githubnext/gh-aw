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

func (e *CopilotEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	copilotLog.Printf("Generating installation steps for Copilot engine: workflow=%s", workflowData.Name)

	var steps []GitHubActionStep

	// Define engine configuration for shared validation
	config := EngineInstallConfig{
		Secrets:         []string{"COPILOT_GITHUB_TOKEN"},
		DocsURL:         "https://githubnext.github.io/gh-aw/reference/engines/#github-copilot-default",
		NpmPackage:      "@github/copilot",
		Version:         string(constants.DefaultCopilotVersion),
		Name:            "GitHub Copilot CLI",
		CliName:         "copilot",
		InstallStepName: "Install GitHub Copilot CLI",
	}

	// Add secret validation step
	secretValidation := GenerateMultiSecretValidationStep(
		config.Secrets,
		config.Name,
		config.DocsURL,
	)
	steps = append(steps, secretValidation)

	// Determine Copilot version
	copilotVersion := config.Version
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		copilotVersion = workflowData.EngineConfig.Version
	}

	// Determine if Copilot should be installed globally or locally
	// For SRT, install locally so npx can find it without network access
	installGlobally := !isSRTEnabled(workflowData)

	// Generate install steps based on installation scope
	var npmSteps []GitHubActionStep
	if installGlobally {
		// Use the new installer script for global installation
		copilotLog.Print("Using new installer script for Copilot installation")
		npmSteps = GenerateCopilotInstallerSteps(copilotVersion, config.InstallStepName)
	} else {
		// For SRT: install locally with npm without -g flag
		copilotLog.Print("Using local Copilot installation for SRT compatibility")
		npmSteps = GenerateNpmInstallStepsWithScope(
			config.NpmPackage,
			copilotVersion,
			config.InstallStepName,
			config.CliName,
			true,  // Include Node.js setup
			false, // Install locally, not globally
		)
	}

	// Add Node.js setup step first (before sandbox installation)
	if len(npmSteps) > 0 {
		steps = append(steps, npmSteps[0]) // Setup Node.js step
	}

	// Add sandbox installation steps
	// SRT and AWF are mutually exclusive (validated earlier)
	if isSRTEnabled(workflowData) {
		// Install Sandbox Runtime (SRT)
		agentConfig := getAgentConfig(workflowData)

		// Skip standard installation if custom command is specified
		if agentConfig == nil || agentConfig.Command == "" {
			copilotLog.Print("Adding Sandbox Runtime (SRT) system dependencies step")
			srtSystemDeps := generateSRTSystemDepsStep()
			steps = append(steps, srtSystemDeps)

			copilotLog.Print("Adding Sandbox Runtime (SRT) system configuration step")
			srtSystemConfig := generateSRTSystemConfigStep()
			steps = append(steps, srtSystemConfig)

			copilotLog.Print("Adding Sandbox Runtime (SRT) installation step")
			srtInstall := generateSRTInstallationStep()
			steps = append(steps, srtInstall)
		} else {
			copilotLog.Print("Skipping SRT installation (custom command specified)")
		}
	} else if isFirewallEnabled(workflowData) {
		// Install AWF after Node.js setup but before Copilot CLI installation
		firewallConfig := getFirewallConfig(workflowData)
		agentConfig := getAgentConfig(workflowData)
		var awfVersion string
		if firewallConfig != nil {
			awfVersion = firewallConfig.Version
		}

		// Install AWF binary (or skip if custom command is specified)
		awfInstall := generateAWFInstallationStep(awfVersion, agentConfig)
		if len(awfInstall) > 0 {
			steps = append(steps, awfInstall)
		}
	}

	// Add Copilot CLI installation step after sandbox installation
	if len(npmSteps) > 1 {
		steps = append(steps, npmSteps[1:]...) // Install Copilot CLI and subsequent steps
	}

	return steps
}

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
		copilotLog.Print("Added gh CLI and copilot binary mounts to AWF container")

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
func (e *CopilotEngine) GetFirewallLogsCollectionStep(workflowData *WorkflowData) []GitHubActionStep {
	// Collection step removed - firewall logs are now at a known location
	return []GitHubActionStep{}
}

// GetSquidLogsSteps returns the steps for uploading and parsing Squid logs (after secret redaction)
func (e *CopilotEngine) GetSquidLogsSteps(workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	// Only add upload and parsing steps if firewall is enabled
	if isFirewallEnabled(workflowData) {
		copilotLog.Printf("Adding Squid logs upload and parsing steps for workflow: %s", workflowData.Name)

		squidLogsUpload := generateSquidLogsUploadStep(workflowData.Name)
		steps = append(steps, squidLogsUpload)

		// Add firewall log parsing step to create step summary
		firewallLogParsing := generateFirewallLogParsingStep(workflowData.Name)
		steps = append(steps, firewallLogParsing)
	} else {
		copilotLog.Print("Firewall disabled, skipping Squid logs upload")
	}

	return steps
}

// GetCleanupStep returns the post-execution cleanup step
func (e *CopilotEngine) GetCleanupStep(workflowData *WorkflowData) GitHubActionStep {
	// Return empty step - cleanup steps have been removed
	return GitHubActionStep([]string{})
}

func (e *CopilotEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	copilotLog.Printf("Rendering MCP config for Copilot engine: mcpTools=%d", len(mcpTools))

	// Create the directory first
	yaml.WriteString("          mkdir -p /home/runner/.copilot\n")

	// Create unified renderer with Copilot-specific options
	// Copilot uses JSON format with type and tools fields, and inline args
	createRenderer := func(isLast bool) *MCPConfigRendererUnified {
		return NewMCPConfigRenderer(MCPRendererOptions{
			IncludeCopilotFields: true, // Copilot uses "type" and "tools" fields
			InlineArgs:           true, // Copilot uses inline args format
			Format:               "json",
			IsLast:               isLast,
		})
	}

	// Use shared JSON MCP config renderer with unified renderer methods
	RenderJSONMCPConfig(yaml, tools, mcpTools, workflowData, JSONMCPConfigOptions{
		ConfigPath: "/home/runner/.copilot/mcp-config.json",
		Renderers: MCPToolRenderers{
			RenderGitHub: func(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData) {
				renderer := createRenderer(isLast)
				renderer.RenderGitHubMCP(yaml, githubTool, workflowData)
			},
			RenderPlaywright: func(yaml *strings.Builder, playwrightTool any, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderPlaywrightMCP(yaml, playwrightTool)
			},
			RenderSerena: func(yaml *strings.Builder, serenaTool any, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderSerenaMCP(yaml, serenaTool)
			},
			RenderCacheMemory: func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
				// Cache-memory is not used for Copilot (filtered out)
			},
			RenderAgenticWorkflows: func(yaml *strings.Builder, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderAgenticWorkflowsMCP(yaml)
			},
			RenderSafeOutputs: func(yaml *strings.Builder, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderSafeOutputsMCP(yaml)
			},
			RenderSafeInputs: func(yaml *strings.Builder, safeInputs *SafeInputsConfig, isLast bool) {
				renderer := createRenderer(isLast)
				renderer.RenderSafeInputsMCP(yaml, safeInputs)
			},
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

// renderCopilotMCPConfig generates custom MCP server configuration for Copilot CLI
func (e *CopilotEngine) renderCopilotMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	copilotLog.Printf("Rendering custom MCP config for tool: %s", toolName)
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
	var totalTokenUsage int

	lines := strings.Split(logContent, "\n")
	toolCallMap := make(map[string]*ToolCallInfo) // Track tool calls
	var currentSequence []string                  // Track tool sequence
	turns := 0

	// Track multi-line JSON blocks for token extraction
	var inDataBlock bool
	var currentJSONLines []string

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Detect start of a JSON data block from Copilot debug logs
		// Format: "YYYY-MM-DDTHH:MM:SS.sssZ [DEBUG] data:"
		if strings.Contains(line, "[DEBUG] data:") {
			inDataBlock = true
			currentJSONLines = []string{}
			continue
		}

		// While in a data block, accumulate lines
		if inDataBlock {
			// Check if this line has a timestamp (indicates it's a log line, not raw JSON)
			hasTimestamp := strings.Contains(line, "[DEBUG]")

			if hasTimestamp {
				// Strip the timestamp and [DEBUG] prefix to see what remains
				// Format: "YYYY-MM-DDTHH:MM:SS.sssZ [DEBUG] {json content}"
				debugIndex := strings.Index(line, "[DEBUG]")
				if debugIndex != -1 {
					cleanLine := strings.TrimSpace(line[debugIndex+7:]) // Skip "[DEBUG]"

					// If after stripping, the line starts with JSON characters, it's part of JSON
					// Otherwise, it's a new log entry and we should end the block
					if strings.HasPrefix(cleanLine, "{") || strings.HasPrefix(cleanLine, "}") ||
						strings.HasPrefix(cleanLine, "[") || strings.HasPrefix(cleanLine, "]") ||
						strings.HasPrefix(cleanLine, "\"") {
						// This is JSON content - add it
						currentJSONLines = append(currentJSONLines, cleanLine)
					} else {
						// This is a new log line (not JSON content) - end of JSON block
						// Try to parse the accumulated JSON
						if len(currentJSONLines) > 0 {
							jsonStr := strings.Join(currentJSONLines, "\n")
							jsonMetrics := ExtractJSONMetrics(jsonStr, verbose)
							// Accumulate token usage from all responses (not just max)
							// This matches the JavaScript parser behavior in parse_copilot_log.cjs
							if jsonMetrics.TokenUsage > 0 {
								totalTokenUsage += jsonMetrics.TokenUsage
							}
							if jsonMetrics.EstimatedCost > 0 {
								metrics.EstimatedCost += jsonMetrics.EstimatedCost
							}
						}

						inDataBlock = false
						currentJSONLines = []string{}
					}
				}
			} else {
				// Line has no timestamp - it's raw JSON, add it
				currentJSONLines = append(currentJSONLines, line)
			}
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
	}

	// Process any remaining JSON block at the end of file
	if inDataBlock && len(currentJSONLines) > 0 {
		jsonStr := strings.Join(currentJSONLines, "\n")
		jsonMetrics := ExtractJSONMetrics(jsonStr, verbose)
		// Accumulate token usage from all responses (not just max)
		if jsonMetrics.TokenUsage > 0 {
			totalTokenUsage += jsonMetrics.TokenUsage
		}
		if jsonMetrics.EstimatedCost > 0 {
			metrics.EstimatedCost += jsonMetrics.EstimatedCost
		}
	}

	// Finalize metrics using shared helper
	FinalizeToolMetrics(&metrics, toolCallMap, currentSequence, turns, totalTokenUsage, logContent, e.GetErrorPatterns())

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
			toolName = constants.SafeOutputsMCPServerID
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
func (e *CopilotEngine) computeCopilotToolArguments(tools map[string]any, safeOutputs *SafeOutputsConfig, safeInputs *SafeInputsConfig, workflowData *WorkflowData) []string {
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
		args = append(args, "--allow-tool", constants.SafeOutputsMCPServerID)
	}

	// Handle safe_inputs MCP server - allow the server if safe inputs are configured and feature flag is enabled
	if IsSafeInputsEnabled(safeInputs, workflowData) {
		args = append(args, "--allow-tool", constants.SafeInputsMCPServerID)
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
func (e *CopilotEngine) generateCopilotToolArgumentsComment(tools map[string]any, safeOutputs *SafeOutputsConfig, safeInputs *SafeInputsConfig, workflowData *WorkflowData, indent string) string {
	toolArgs := e.computeCopilotToolArguments(tools, safeOutputs, safeInputs, workflowData)
	if len(toolArgs) == 0 {
		return ""
	}

	var comment strings.Builder
	comment.WriteString(indent + "# Copilot CLI tool arguments (sorted):\n")

	// Group flag-value pairs for better readability
	for i := 0; i < len(toolArgs); i += 2 {
		if i+1 < len(toolArgs) {
			fmt.Fprintf(&comment, "%s# %s %s\n", indent, toolArgs[i], toolArgs[i+1])
		}
	}

	return comment.String()
}

// GetErrorPatterns returns regex patterns for extracting error messages from Copilot CLI logs
func (e *CopilotEngine) GetErrorPatterns() []ErrorPattern {
	patterns := GetCommonErrorPatterns()

	// Add benign error patterns first (so they can be explicitly filtered)
	patterns = append(patterns, GetBenignErrorPatterns()...)

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
		// Permission-related patterns (classified as warnings, not errors)
		{
			ID:           "copilot-permission-denied",
			Pattern:      `(?i)\berror\b.*permission.*denied`,
			LevelGroup:   0,
			MessageGroup: 0,
			Severity:     "warning",
			Description:  "Permission denied error (requires error context)",
		},
		{
			ID:           "copilot-unauthorized",
			Pattern:      `(?i)\berror\b.*unauthorized`,
			LevelGroup:   0,
			MessageGroup: 0,
			Severity:     "warning",
			Description:  "Unauthorized access error (requires error context)",
		},
		{
			ID:           "copilot-forbidden",
			Pattern:      `(?i)\berror\b.*forbidden`,
			LevelGroup:   0,
			MessageGroup: 0,
			Severity:     "warning",
			Description:  "Forbidden access error (requires error context)",
		},
	}...)

	return patterns
}

// generateAWFInstallationStep creates a GitHub Actions step to install the AWF binary
func generateAWFInstallationStep(version string, agentConfig *AgentSandboxConfig) GitHubActionStep {
	// If custom command is specified, skip installation (command replaces binary)
	if agentConfig != nil && agentConfig.Command != "" {
		copilotLog.Print("Skipping AWF binary installation (custom command specified)")
		// Return empty step - custom command will be used in execution
		return GitHubActionStep([]string{})
	}

	// Use default version for logging when not specified
	if version == "" {
		version = string(constants.DefaultFirewallVersion)
	}

	stepLines := []string{
		"      - name: Install awf binary",
		"        run: |",
		fmt.Sprintf("          echo \"Installing awf via installer script (requested version: %s)\"", version),
		fmt.Sprintf("          curl -sSL https://raw.githubusercontent.com/githubnext/gh-aw-firewall/main/install.sh | sudo AWF_VERSION=%s bash", version),
		"          which awf",
		"          awf --version",
	}

	return GitHubActionStep(stepLines)
}

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
