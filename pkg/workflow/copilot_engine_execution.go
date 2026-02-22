// This file provides Copilot engine execution logic.
//
// This file contains the GetExecutionSteps function which generates the complete
// GitHub Actions workflow for executing GitHub Copilot CLI. This is the largest
// and most complex function in the Copilot engine, handling:
//
//   - Copilot CLI argument construction based on sandbox mode (AWF, SRT, or standard)
//   - Tool permission configuration (--allow-tool flags)
//   - MCP server configuration and environment setup
//   - Sandbox wrapping (AWF or SRT)
//   - Environment variable handling for model selection and secrets
//   - Log file configuration and output collection
//
// The execution strategy varies significantly based on sandbox mode:
//   - Standard mode: Direct copilot CLI execution
//   - AWF mode: Wrapped with awf binary for network firewalling
//   - SRT mode: Wrapped with Sandbox Runtime for process isolation
//
// This function is intentionally kept in a separate file due to its size (~430 lines)
// and complexity. Future refactoring may split it further if needed.

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

var copilotExecLog = logger.New("workflow:copilot_engine_execution")

// GetExecutionSteps returns the GitHub Actions steps for executing GitHub Copilot CLI
func (e *CopilotEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	copilotExecLog.Printf("Generating execution steps for Copilot: workflow=%s, firewall=%v", workflowData.Name, isFirewallEnabled(workflowData))

	var steps []GitHubActionStep

	// Build copilot CLI arguments based on configuration
	var copilotArgs []string
	sandboxEnabled := isFirewallEnabled(workflowData)
	if sandboxEnabled {
		// Simplified args for sandbox mode (AWF)
		copilotArgs = []string{"--add-dir", "/tmp/gh-aw/", "--log-level", "all", "--log-dir", logsFolder}

		// Always add workspace directory to --add-dir so Copilot CLI can access it
		// This allows Copilot CLI to discover agent files and access the workspace
		// Use double quotes to allow shell variable expansion
		copilotArgs = append(copilotArgs, "--add-dir", "\"${GITHUB_WORKSPACE}\"")
		copilotExecLog.Print("Added workspace directory to --add-dir")

		// Add Copilot config directory when plugins are declared so the CLI can discover installed plugins
		// Plugins are installed to ~/.copilot/plugins/ via copilot plugin install command
		// The CLI also reads plugin-index.json from ~/.copilot/ to discover installed plugins
		if workflowData.PluginInfo != nil && len(workflowData.PluginInfo.Plugins) > 0 {
			copilotArgs = append(copilotArgs, "--add-dir", "/home/runner/.copilot/")
			copilotExecLog.Printf("Added Copilot config directory to --add-dir for plugin discovery (%d plugins)", len(workflowData.PluginInfo.Plugins))
		}

		copilotExecLog.Print("Using firewall mode with simplified arguments")
	} else {
		// Original args for non-sandbox mode
		copilotArgs = []string{"--add-dir", "/tmp/", "--add-dir", "/tmp/gh-aw/", "--add-dir", "/tmp/gh-aw/agent/", "--log-level", "all", "--log-dir", logsFolder}
		copilotExecLog.Print("Using standard mode with full arguments")
	}

	// Add --disable-builtin-mcps to disable built-in MCP servers
	copilotArgs = append(copilotArgs, "--disable-builtin-mcps")

	// Model is always passed via the native COPILOT_MODEL environment variable when configured.
	// This avoids embedding the value directly in the shell command (which fails template injection
	// validation for GitHub Actions expressions like ${{ inputs.model }}).
	// Fallback for unconfigured model uses GH_AW_MODEL_AGENT_COPILOT with shell expansion.
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	// Add --agent flag if specified via engine.agent
	// Note: Agent imports (.github/agents/*.md) still work for importing markdown content,
	// but they do NOT automatically set the --agent flag. Only engine.agent controls the flag.
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Agent != "" {
		agentIdentifier := workflowData.EngineConfig.Agent
		copilotExecLog.Printf("Using agent from engine.agent: %s", agentIdentifier)
		copilotArgs = append(copilotArgs, "--agent", agentIdentifier)
	}

	// Add tool permission arguments based on configuration
	toolArgs := e.computeCopilotToolArguments(workflowData.Tools, workflowData.SafeOutputs, workflowData.SafeInputs, workflowData)
	if len(toolArgs) > 0 {
		copilotExecLog.Printf("Adding %d tool permission arguments", len(toolArgs))
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

	// When model is not configured, use the GH_AW_MODEL_AGENT_COPILOT fallback env var
	// via shell expansion (${VAR:+ --model "$VAR"}) so users can set it as a GitHub variable.
	// When model IS configured, COPILOT_MODEL is set in the env block (see below) and the
	// Copilot CLI reads it natively - no --model flag in the shell command needed.
	needsModelFlag := !modelConfigured
	// Check if this is a detection job (has no SafeOutputs config)
	isDetectionJob := workflowData.SafeOutputs == nil
	var modelEnvVar string
	if isDetectionJob {
		modelEnvVar = constants.EnvVarModelDetectionCopilot
	} else {
		modelEnvVar = constants.EnvVarModelAgentCopilot
	}

	// Determine which command to use (once for both sandbox and non-sandbox modes)
	var commandName string
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
		copilotExecLog.Printf("Using custom command: %s", commandName)
	} else if sandboxEnabled {
		// AWF - use the installed binary directly
		// The binary is mounted into the AWF container from /usr/local/bin/copilot
		commandName = "/usr/local/bin/copilot"
	} else {
		// Non-sandbox mode: use standard copilot command
		commandName = "copilot"
	}

	if sandboxEnabled {
		// Build base command
		baseCommand := fmt.Sprintf("%s %s", commandName, shellJoinArgs(copilotArgs))

		// Add conditional model flag if needed
		if needsModelFlag {
			copilotCommand = fmt.Sprintf(`%s${%s:+ --model "$%s"}`, baseCommand, modelEnvVar, modelEnvVar)
		} else {
			copilotCommand = baseCommand
		}
	} else {
		baseCommand := fmt.Sprintf("%s %s", commandName, shellJoinArgs(copilotArgs))

		// Add conditional model flag if needed
		if needsModelFlag {
			copilotCommand = fmt.Sprintf(`%s${%s:+ --model "$%s"}`, baseCommand, modelEnvVar, modelEnvVar)
		} else {
			copilotCommand = baseCommand
		}
	}

	// Conditionally wrap with sandbox (AWF only)
	var command string
	if isFirewallEnabled(workflowData) {
		// Build AWF-wrapped command using helper function - no mkdir needed, AWF handles it
		// Get allowed domains (copilot defaults + network permissions + HTTP MCP server URLs + runtime ecosystem domains)
		allowedDomains := GetCopilotAllowedDomainsWithToolsAndRuntimes(workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)

		// Build AWF command with all configuration
		// Enable API proxy sidecar if this engine supports LLM gateway
		llmGatewayPort := e.SupportsLLMGateway()
		usesAPIProxy := llmGatewayPort > 0

		// AWF v0.15.0+ uses chroot mode by default, providing transparent access to host binaries
		// AWF v0.15.0+ with --env-all handles PATH natively (chroot mode is default):
		// 1. Captures host PATH → AWF_HOST_PATH (already has correct ordering from actions/setup-*)
		// 2. Passes ALL host env vars including JAVA_HOME, DOTNET_ROOT, GOROOT
		// 3. entrypoint.sh exports PATH="${AWF_HOST_PATH}" and tool-specific vars
		// 4. Container inherits complete, correctly-ordered environment
		//
		// Version precedence works because actions/setup-* PREPEND to PATH, so
		// /opt/hostedtoolcache/go/1.25.6/x64/bin comes before /usr/bin in AWF_HOST_PATH.
		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "copilot",
			EngineCommand:  copilotCommand,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false, // Copilot doesn't require TTY
			UsesAPIProxy:   usesAPIProxy,
			AllowedDomains: allowedDomains,
			PathSetup:      "", // No path setup needed on host side
		})
	} else {
		// Run copilot command without AWF wrapper
		command = fmt.Sprintf(`set -o pipefail
COPILOT_CLI_INSTRUCTION="$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"
%s%s 2>&1 | tee %s`, mkdirCommands.String(), copilotCommand, logFile)
	}

	// Use COPILOT_GITHUB_TOKEN
	// #nosec G101 -- This is NOT a hardcoded credential. It's a GitHub Actions expression template
	// that GitHub Actions runtime replaces with the actual secret value. The string "${{ secrets.COPILOT_GITHUB_TOKEN }}"
	// is a placeholder, not an actual credential.
	copilotGitHubToken := "${{ secrets.COPILOT_GITHUB_TOKEN }}"

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
		// If GitHub App is configured, use the app token (overrides custom and default tokens)
		if workflowData.ParsedTools != nil && workflowData.ParsedTools.GitHub != nil && workflowData.ParsedTools.GitHub.App != nil {
			env["GITHUB_MCP_SERVER_TOKEN"] = "${{ steps.github-mcp-app-token.outputs.token }}"
		} else {
			customGitHubToken := getGitHubToken(workflowData.Tools["github"])
			// Use effective token with precedence: custom > default
			effectiveToken := getEffectiveGitHubToken(customGitHubToken)
			env["GITHUB_MCP_SERVER_TOKEN"] = effectiveToken
		}
	}

	// Add GH_AW_SAFE_OUTPUTS if output is needed
	applySafeOutputEnvToMap(env, workflowData)

	// Add GH_AW_STARTUP_TIMEOUT environment variable (in seconds) if startup-timeout is specified
	if workflowData.ToolsStartupTimeout > 0 {
		env["GH_AW_STARTUP_TIMEOUT"] = strconv.Itoa(workflowData.ToolsStartupTimeout)
	}

	// Add GH_AW_TOOL_TIMEOUT environment variable (in seconds) if timeout is specified
	if workflowData.ToolsTimeout > 0 {
		env["GH_AW_TOOL_TIMEOUT"] = strconv.Itoa(workflowData.ToolsTimeout)
	}

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		env["GH_AW_MAX_TURNS"] = workflowData.EngineConfig.MaxTurns
	}

	// Set the model environment variable.
	// When model is configured, use the native COPILOT_MODEL env var - the Copilot CLI reads it
	// directly, avoiding the need to embed the value in the shell command (which would fail
	// template injection validation for GitHub Actions expressions like ${{ inputs.model }}).
	// When model is not configured, fall back to GH_AW_MODEL_AGENT/DETECTION_COPILOT so users
	// can set a default via GitHub Actions variables.
	if modelConfigured {
		copilotExecLog.Printf("Setting %s env var for model: %s", constants.CopilotCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.CopilotCLIModelEnvVar] = workflowData.EngineConfig.Model
	} else {
		// No model configured - use fallback GitHub variable with shell expansion
		isDetectionJob := workflowData.SafeOutputs == nil
		if isDetectionJob {
			env[constants.EnvVarModelDetectionCopilot] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelDetectionCopilot)
		} else {
			env[constants.EnvVarModelAgentCopilot] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelAgentCopilot)
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
		copilotExecLog.Printf("Added %d custom env vars from agent config", len(agentConfig.Env))
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

	stepLines = append(stepLines, "      - name: "+stepName)
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

	// Format step with command and filtered environment variables using shared helper
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}

// extractAddDirPaths extracts all directory paths from copilot args that follow --add-dir flags
func extractAddDirPaths(args []string) []string {
	var dirs []string
	for i := range len(args) - 1 {
		if args[i] == "--add-dir" {
			dirs = append(dirs, args[i+1])
		}
	}
	return dirs
}

// generateCopilotSessionFileCopyStep generates a step to copy Copilot session state files
// from ~/.copilot/session-state/ to /tmp/gh-aw/sandbox/agent/logs/
// This ensures session files are in /tmp/gh-aw/ where secret redaction can scan them
func generateCopilotSessionFileCopyStep() GitHubActionStep {
	var step []string

	step = append(step, "      - name: Copy Copilot session state files to logs")
	step = append(step, "        if: always()")
	step = append(step, "        continue-on-error: true")
	step = append(step, "        run: |")
	step = append(step, "          # Copy Copilot session state files to logs folder for artifact collection")
	step = append(step, "          # This ensures they are in /tmp/gh-aw/ where secret redaction can scan them")
	step = append(step, "          SESSION_STATE_DIR=\"$HOME/.copilot/session-state\"")
	step = append(step, "          LOGS_DIR=\"/tmp/gh-aw/sandbox/agent/logs\"")
	step = append(step, "          ")
	step = append(step, "          if [ -d \"$SESSION_STATE_DIR\" ]; then")
	step = append(step, "            echo \"Copying Copilot session state files from $SESSION_STATE_DIR to $LOGS_DIR\"")
	step = append(step, "            mkdir -p \"$LOGS_DIR\"")
	step = append(step, "            cp -v \"$SESSION_STATE_DIR\"/*.jsonl \"$LOGS_DIR/\" 2>/dev/null || true")
	step = append(step, "            echo \"Session state files copied successfully\"")
	step = append(step, "          else")
	step = append(step, "            echo \"No session-state directory found at $SESSION_STATE_DIR\"")
	step = append(step, "          fi")

	return GitHubActionStep(step)
}
