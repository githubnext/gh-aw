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
	"github.com/github/gh-aw/pkg/semverutil"
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

		// Note: --add-dir "${GITHUB_WORKSPACE}" is appended raw after shellJoinArgs below
		// to allow shell variable expansion (cannot go through shellEscapeArg).
		copilotExecLog.Print("Added workspace directory to --add-dir")

		copilotExecLog.Print("Using firewall mode with simplified arguments")
	} else {
		// Original args for non-sandbox mode
		copilotArgs = []string{"--add-dir", "/tmp/", "--add-dir", "/tmp/gh-aw/", "--add-dir", "/tmp/gh-aw/agent/", "--log-level", "all", "--log-dir", logsFolder}
		copilotExecLog.Print("Using standard mode with full arguments")
	}

	// Add --disable-builtin-mcps to disable built-in MCP servers
	copilotArgs = append(copilotArgs, "--disable-builtin-mcps")

	// Add --no-ask-user to enable fully autonomous runs (suppresses interactive prompts).
	// Emitted for both agent and detection jobs when the Copilot CLI version supports it
	// (v1.0.19+). Latest and unspecified versions always include the flag.
	isDetectionJob := workflowData.SafeOutputs == nil
	if copilotSupportsNoAskUser(workflowData.EngineConfig) {
		copilotExecLog.Print("Adding --no-ask-user for fully autonomous run")
		copilotArgs = append(copilotArgs, "--no-ask-user")
	}

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

	// Add --autopilot and --max-autopilot-continues when max-continuations > 1
	// Never apply autopilot flags to detection jobs; they are only meaningful for the agent run.
	if !isDetectionJob && workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxContinuations > 1 {
		maxCont := workflowData.EngineConfig.MaxContinuations
		copilotExecLog.Printf("Enabling autopilot mode with max-autopilot-continues=%d", maxCont)
		copilotArgs = append(copilotArgs, "--autopilot", "--max-autopilot-continues", strconv.Itoa(maxCont))
	}

	// Add tool permission arguments based on configuration
	toolArgs := e.computeCopilotToolArguments(workflowData.Tools, workflowData.SafeOutputs, workflowData.MCPScripts, workflowData)
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

	// Add --no-custom-instructions when bare mode is enabled to suppress automatic
	// loading of custom instructions from .github/AGENTS.md and user-level configs.
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Bare {
		copilotExecLog.Print("Bare mode enabled: adding --no-custom-instructions")
		copilotArgs = append(copilotArgs, "--no-custom-instructions")
	}

	// Add custom args from engine configuration before the prompt
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		copilotArgs = append(copilotArgs, workflowData.EngineConfig.Args...)
	}

	// Note: the --prompt argument and (in sandbox mode) --add-dir "${GITHUB_WORKSPACE}"
	// are appended raw after shellJoinArgs in the command building step below.
	// These contain shell variable references that must NOT go through shellEscapeArg
	// because single-quoting them would prevent shell expansion at runtime.

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

	// Determine model org variable name based on job type (used in env block below).
	// The model is always passed via the native COPILOT_MODEL env var - no --model flag needed.
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

	// Build the command - model is always passed via COPILOT_MODEL env var (see env block below).
	// The --add-dir "${GITHUB_WORKSPACE}" arg is appended raw (not through shellJoinArgs)
	// because it contains a shell variable reference that must expand at runtime.
	//
	// When a driver script is provided (GetDriverScriptName), wrap the copilot invocation with
	// `node <driver> <commandName> <args>` to enable retry logic for transient CAPIError 400 errors.
	//
	// Use ${GH_AW_NODE_BIN:-node} instead of plain `node` so the absolute node path
	// (exported by install_awf_binary.sh as GH_AW_NODE_BIN when the bundle is installed)
	// is used inside the AWF container. In AWF's chroot mode the host filesystem is
	// accessible, so the absolute path works even when sudo resets PATH on GPU runners
	// (e.g. aw-gpu-runner-T4) and the actions/setup-node directory is not in PATH.
	driverScriptName := e.GetDriverScriptName()
	var execPrefix string
	if driverScriptName != "" {
		// Driver wraps the copilot subprocess; ${RUNNER_TEMP} and ${GH_AW_NODE_BIN} expand in the shell context.
		execPrefix = fmt.Sprintf(`${GH_AW_NODE_BIN:-node} %s/%s %s`, SetupActionDestinationShell, driverScriptName, commandName)
	} else {
		execPrefix = commandName
	}

	if sandboxEnabled {
		// Sandbox mode: add workspace dir and pass prompt file path directly
		copilotCommand = fmt.Sprintf(`%s %s --add-dir "${GITHUB_WORKSPACE}" --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt`, execPrefix, shellJoinArgs(copilotArgs))
	} else {
		// Non-sandbox mode: pass prompt file path directly
		copilotCommand = fmt.Sprintf(`%s %s --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt`, execPrefix, shellJoinArgs(copilotArgs))
	}

	// Conditionally wrap with sandbox (AWF only)
	var command string
	if isFirewallEnabled(workflowData) {
		// Build AWF-wrapped command using helper function - no mkdir needed, AWF handles it
		// For detection runs use the minimal detection domain list (excludes registry.npmjs.org
		// and raw.githubusercontent.com — not needed when MCP servers are disabled and the
		// Copilot CLI binary is already installed on the runner).
		// For normal agent runs use the full domain set (defaults + ecosystem + user-specified).
		var allowedDomains string
		if workflowData.IsDetectionRun {
			allowedDomains = GetThreatDetectionAllowedDomains(workflowData.NetworkPermissions)
		} else {
			allowedDomains = GetCopilotAllowedDomainsWithToolsAndRuntimes(workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
		}
		// Add Copilot API target domains to the firewall allow-list.
		// Resolved from engine.api-target or GITHUB_COPILOT_BASE_URL in engine.env.
		if copilotAPITarget := GetCopilotAPITarget(workflowData); copilotAPITarget != "" {
			allowedDomains = mergeAPITargetDomains(allowedDomains, copilotAPITarget)
		}

		// AWF v0.15.0+ uses chroot mode by default, providing transparent access to host binaries
		// AWF v0.15.0+ with --env-all handles PATH natively (chroot mode is default):
		// 1. Captures host PATH → AWF_HOST_PATH (already has correct ordering from actions/setup-*)
		// 2. Passes ALL host env vars including JAVA_HOME, DOTNET_ROOT, GOROOT
		// 3. entrypoint.sh exports PATH="${AWF_HOST_PATH}" and tool-specific vars
		// 4. Container inherits complete, correctly-ordered environment
		//
		// Version precedence works because actions/setup-* PREPEND to PATH, so
		// /opt/hostedtoolcache/go/1.25.6/x64/bin comes before /usr/bin in AWF_HOST_PATH.
		//
		// MCP CLI bin directory: when mount-as-clis is enabled, the CLI wrapper scripts
		// live under ${RUNNER_TEMP}/gh-aw/mcp-cli/bin. core.addPath() adds this to
		// $GITHUB_PATH for subsequent steps, but sudo's secure_path may strip it.
		// Prepending it to the engine command ensures the agent can find them.
		engineCommand := copilotCommand
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			engineCommand = fmt.Sprintf("%s && %s", mcpCLIPath, copilotCommand)
		}
		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "copilot",
			EngineCommand:  engineCommand,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false, // Copilot doesn't require TTY
			AllowedDomains: allowedDomains,
			// Create the agent step summary file before AWF starts so it is accessible
			// inside the sandbox. The agent writes its step summary content here, and the
			// file is appended to $GITHUB_STEP_SUMMARY after secret redaction.
			//
			// Resolve the absolute node binary path before `sudo -E awf` runs.
			// On GPU runners (e.g. aw-gpu-runner-T4) sudo resets PATH via sudoers
			// secure_path, stripping the actions/setup-node directory.  By capturing
			// the path here (where PATH is still intact) and exporting it, sudo -E
			// preserves the variable and AWF's --env-all forwards it into the container,
			// where ${GH_AW_NODE_BIN:-node} resolves to the correct binary.
			PathSetup: "touch " + AgentStepSummaryPath + "\n" +
				"GH_AW_NODE_BIN=$(command -v node 2>/dev/null || true)\n" +
				"export GH_AW_NODE_BIN",
			// Exclude every env var whose step-env value is a secret so the agent
			// cannot read raw token values via bash tools (env / printenv).
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, []string{"COPILOT_GITHUB_TOKEN"}),
		})
	} else {
		// Run copilot command without AWF wrapper.
		// Prepend a touch command to create the agent step summary file before copilot runs.
		command = fmt.Sprintf(`set -o pipefail
touch %s
(umask 177 && touch %s)
%s%s 2>&1 | tee %s`, AgentStepSummaryPath, logFile, mkdirCommands.String(), copilotCommand, logFile)
	}

	// Use COPILOT_GITHUB_TOKEN: when the copilot-requests feature is enabled, use the GitHub
	// Actions token directly (${{ github.token }}). Otherwise use the COPILOT_GITHUB_TOKEN secret.
	// #nosec G101 -- These are NOT hardcoded credentials. They are GitHub Actions expression templates
	// that the runtime replaces with actual values. The strings "${{ secrets.COPILOT_GITHUB_TOKEN }}"
	// and "${{ github.token }}" are placeholders, not actual credentials.
	var copilotGitHubToken string
	useCopilotRequests := isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData)
	if useCopilotRequests {
		copilotGitHubToken = "${{ github.token }}"
		copilotExecLog.Print("Using GitHub Actions token as COPILOT_GITHUB_TOKEN (copilot-requests feature enabled)")
	} else {
		copilotGitHubToken = "${{ secrets.COPILOT_GITHUB_TOKEN }}"
	}

	env := map[string]string{
		"XDG_CONFIG_HOME":           "/home/runner",
		"COPILOT_AGENT_RUNNER_TYPE": "STANDALONE",
		"COPILOT_GITHUB_TOKEN":      copilotGitHubToken,
		// Override GITHUB_STEP_SUMMARY with a path that exists inside the sandbox.
		// The runner's original path is unreachable within the AWF isolated filesystem;
		// we create this file before the agent starts and append it to the real
		// $GITHUB_STEP_SUMMARY after secret redaction.
		"GITHUB_STEP_SUMMARY": AgentStepSummaryPath,
		"GITHUB_HEAD_REF":     "${{ github.head_ref }}",
		"GITHUB_REF_NAME":     "${{ github.ref_name }}",
		"GITHUB_WORKSPACE":    "${{ github.workspace }}",
		// Pass GitHub server URL and API URL for GitHub Enterprise compatibility.
		// In standard GitHub.com environments these resolve to https://github.com and
		// https://api.github.com. In GitHub Enterprise they resolve to the enterprise
		// server URL (e.g. https://COMPANY.ghe.com and https://COMPANY.ghe.com/api/v3).
		"GITHUB_SERVER_URL": "${{ github.server_url }}",
		"GITHUB_API_URL":    "${{ github.api_url }}",
	}

	// When copilot-requests feature is enabled, set S2STOKENS=true to allow the Copilot CLI
	// to accept GitHub App installation tokens (ghs_*) such as ${{ github.token }}.
	if useCopilotRequests {
		env["S2STOKENS"] = "true"
	}

	// In sandbox (AWF) mode, set git identity environment variables so the first git commit
	// succeeds inside the container. AWF's --env-all forwards these to the container, ensuring
	// git does not rely on the host-side ~/.gitconfig which is not visible in the sandbox.
	if sandboxEnabled {
		maps.Copy(env, getGitIdentityEnvVars())
	}

	// Always add GH_AW_PROMPT for agentic workflows
	env["GH_AW_PROMPT"] = "/tmp/gh-aw/aw-prompts/prompt.txt"

	// Tag the step as a GitHub AW agentic execution for discoverability by agents
	env["GITHUB_AW"] = "true"
	// Inject the integration ID only when the feature flag is explicitly enabled.
	// Default off — the env var may cause Copilot CLI failures.
	// See https://github.com/github/gh-aw/issues/25516
	if isFeatureEnabled(constants.CopilotIntegrationIDFeatureFlag, workflowData) {
		env[constants.CopilotCLIIntegrationIDEnvVar] = constants.CopilotCLIIntegrationIDValue
	}
	// Indicate the phase: "agent" for the main run, "detection" for threat detection
	if workflowData.IsDetectionRun {
		env["GH_AW_PHASE"] = "detection"
	} else {
		env["GH_AW_PHASE"] = "agent"
	}
	// Include the compiler version so agents can identify which gh-aw version generated the workflow.
	// Only emit the real version in release builds; otherwise use "dev".
	if IsRelease() {
		env["GH_AW_VERSION"] = GetVersion()
	} else {
		env["GH_AW_VERSION"] = "dev"
	}

	// Add GH_AW_MCP_CONFIG for MCP server configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "/home/runner/.copilot/mcp-config.json"
	}

	if hasGitHubTool(workflowData.ParsedTools) {
		// If GitHub App is configured, use the app token minted directly in the agent job.
		// The token cannot be passed via job outputs from the activation job because
		// actions/create-github-app-token calls ::add-mask:: on the token, and the
		// GitHub Actions runner silently drops masked values in job outputs (runner v2.308+).
		if workflowData.ParsedTools != nil && workflowData.ParsedTools.GitHub != nil && workflowData.ParsedTools.GitHub.GitHubApp != nil {
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
	// The model is always passed via the native COPILOT_MODEL env var, which the Copilot CLI reads
	// directly. This avoids embedding the value in the shell command (which would fail template
	// injection validation for GitHub Actions expressions like ${{ inputs.model }}).
	// When model is explicitly configured, use its value directly.
	// When model is not configured, map the GitHub org variable to COPILOT_MODEL so users can set
	// a default via GitHub Actions variables without requiring per-workflow frontmatter changes.
	if modelConfigured {
		copilotExecLog.Printf("Setting %s env var for model: %s", constants.CopilotCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.CopilotCLIModelEnvVar] = workflowData.EngineConfig.Model
	} else {
		// No model configured - map org variable to native COPILOT_MODEL env var
		env[constants.CopilotCLIModelEnvVar] = fmt.Sprintf("${{ vars.%s || '' }}", modelEnvVar)
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

	// Inject the dummy COPILOT_API_KEY AFTER all env merges so that legacy/manual
	// wiring in engine.env or agent.env cannot accidentally overwrite the sentinel
	// value that triggers AWF's runtime BYOK detection path.
	if isFeatureEnabled(constants.ByokCopilotFeatureFlag, workflowData) {
		env["COPILOT_API_KEY"] = constants.CopilotBYOKDummyAPIKey
	}

	// Add HTTP MCP header secrets to env for passthrough
	headerSecrets := collectHTTPMCPHeaderSecrets(workflowData.Tools)
	for varName, secretExpr := range headerSecrets {
		// Only add if not already in env
		if _, exists := env[varName]; !exists {
			env[varName] = secretExpr
		}
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

	// Generate the step for Copilot CLI execution
	stepName := "Execute GitHub Copilot CLI"
	var stepLines []string

	stepLines = append(stepLines, "      - name: "+stepName)
	stepLines = append(stepLines, "        id: agentic_execution")

	// Add tool arguments comment before the run section
	toolArgsComment := e.generateCopilotToolArgumentsComment(workflowData.Tools, workflowData.SafeOutputs, workflowData.MCPScripts, workflowData, "        ")
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

	// Inject GH_TOKEN for CLI proxy (added after filtering since it uses a special
	// fallback expression that is always allowed when cli-proxy is enabled)
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)

	// Format step with command and filtered environment variables using shared helper
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}

// generateCopilotErrorDetectionStep generates a single step that detects known Copilot CLI
// errors by scanning the agent stdio log. It sets four outputs:
//   - inference_access_error: token lacks inference access (policy access denied)
//   - mcp_policy_error: MCP servers blocked by enterprise/organization policy
//   - agentic_engine_timeout: process killed by signal (SIGTERM/SIGKILL/SIGINT), typically step timeout
//   - model_not_supported_error: requested model unavailable for the subscription tier
func generateCopilotErrorDetectionStep() GitHubActionStep {
	var step []string

	step = append(step, "      - name: Detect Copilot errors")
	step = append(step, "        id: detect-copilot-errors")
	step = append(step, "        if: always()")
	step = append(step, "        continue-on-error: true")
	step = append(step, "        run: node \"${RUNNER_TEMP}/gh-aw/actions/detect_copilot_errors.cjs\"")

	return GitHubActionStep(step)
}

// copilotSupportsNoAskUser returns true when the effective Copilot CLI version supports the
// --no-ask-user flag, which enables fully autonomous agentic runs by suppressing interactive prompts.
//
// The --no-ask-user flag was introduced in Copilot CLI v1.0.19. Any workflow that pins an
// explicit version older than v1.0.19 must not emit --no-ask-user or the run will fail at startup.
//
// Special cases:
//   - No version override (engineConfig is nil or has no Version): use DefaultCopilotVersion
//     which is always ≥ CopilotNoAskUserMinVersion → returns true.
//   - "latest": always returns true (latest is always a new release).
//   - Any semver string ≥ CopilotNoAskUserMinVersion: returns true.
//   - Any semver string < CopilotNoAskUserMinVersion: returns false.
//   - Non-semver string (e.g. a branch name): returns false (conservative).
func copilotSupportsNoAskUser(engineConfig *EngineConfig) bool {
	var versionStr string
	if engineConfig != nil && engineConfig.Version != "" {
		versionStr = engineConfig.Version
	} else {
		// No override → use the default, which is always ≥ the minimum.
		return true
	}

	// "latest" means the newest release — always supports the flag.
	if strings.EqualFold(versionStr, "latest") {
		return true
	}

	minVersion := string(constants.CopilotNoAskUserMinVersion)
	return semverutil.Compare(versionStr, minVersion) >= 0
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

// generateCopilotSessionFileCopyStep generates a step to copy the entire Copilot
// session-state directory from ~/.copilot/session-state/ to /tmp/gh-aw/sandbox/agent/logs/
// This ensures all session files (events.jsonl, session.db, plan.md, checkpoints, etc.)
// are in /tmp/gh-aw/ where secret redaction can scan them and they get uploaded as artifacts.
// The logic is in actions/setup/sh/copy_copilot_session_state.sh.
func generateCopilotSessionFileCopyStep() GitHubActionStep {
	var step []string

	step = append(step, "      - name: Copy Copilot session state files to logs")
	step = append(step, "        if: always()")
	step = append(step, "        continue-on-error: true")
	step = append(step, "        run: bash \"${RUNNER_TEMP}/gh-aw/actions/copy_copilot_session_state.sh\"")

	return GitHubActionStep(step)
}
