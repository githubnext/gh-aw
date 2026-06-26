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
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var copilotExecLog = logger.New("workflow:copilot_engine_execution")

const customEngineCommandScriptPath = "/tmp/gh-aw/engine-command.sh"

// copilotSettingsPath is the shell expression that resolves to the Copilot CLI settings
// file at runtime. The Copilot CLI resolves its config directory as ~/.copilot, which is
// /home/runner/.copilot on standard GitHub-hosted runners (HOME=/home/runner) but may
// differ on self-hosted or containerized runners. HOME is a standard POSIX environment
// variable inherited from the runner's parent process and passed through to shell steps;
// other generators (copilot_mcp.go, mcp_setup_generator.go) rely on it the same way.
const copilotSettingsPath = "$HOME/.copilot/settings.json"

// copilotSettingsContent is the JSON content written to the Copilot CLI settings file.
// Setting builtInAgents.rubberDuck to false disables the rubber-duck sub-agent, which
// would otherwise be invoked proactively for non-trivial tasks. In Copilot engine runs
// this adds unnecessary token overhead and latency with little benefit, since the primary
// model already has strong reasoning capabilities.
const copilotSettingsContent = `{"builtInAgents":{"rubberDuck":false}}`

// buildCopilotSettingsSetup returns shell commands that write the Copilot CLI settings
// file before the agent runs, disabling the rubber-duck sub-agent.
func buildCopilotSettingsSetup(fixOwnershipForCustomCommand bool) string {
	setup := "mkdir -p \"$HOME/.copilot\"\n"
	if fixOwnershipForCustomCommand {
		setup += "sudo chown -R \"$(id -u):$(id -g)\" \"$HOME/.copilot\"\n"
	}
	return setup + fmt.Sprintf("printf '%%s' %s > \"%s\"\n",
		shellEscapeArg(copilotSettingsContent), copilotSettingsPath)
}

// buildCopilotSettingsCleanupTrap returns a shell trap command that removes the
// temporary Copilot settings file at step exit. The trap body is single-quoted so
// $HOME is expanded by the shell at trap-fire time rather than trap-definition time.
func buildCopilotSettingsCleanupTrap() string {
	return fmt.Sprintf("trap 'rm -f \"%s\"' EXIT\n", copilotSettingsPath)
}

// buildCopilotMCPConfigExport returns shell commands that export Copilot-CLI-specific
// env vars whose values depend on the runtime $HOME (which may not be /home/runner on
// self-hosted or containerized runners). GitHub Actions does not shell-expand env:
// values, so these must be exported from the run script rather than the YAML env: block.
//
// Always exports:
//   - XDG_CONFIG_HOME=$HOME (Copilot CLI resolves its config dir from this)
//
// Exported only when the workflow has MCP servers:
//   - GH_AW_MCP_CONFIG=$HOME/.copilot/mcp-config.json
func buildCopilotMCPConfigExport(workflowData *WorkflowData) string {
	var b strings.Builder
	b.WriteString("export XDG_CONFIG_HOME=\"$HOME\"\n")
	if HasMCPServers(workflowData) {
		b.WriteString("export GH_AW_MCP_CONFIG=\"$HOME/.copilot/mcp-config.json\"\n")
	}
	return b.String()
}

const nodePathSetupCommand = `GH_AW_NPM_GLOBAL_ROOT="$(npm root -g 2>/dev/null || true)"; if [ -n "$GH_AW_NPM_GLOBAL_ROOT" ]; then export NODE_PATH="${GH_AW_NPM_GLOBAL_ROOT}${NODE_PATH:+:${NODE_PATH}}"; fi`
const nodeRuntimeResolutionCommand = `GH_AW_NODE_EXEC="${GH_AW_NODE_BIN:-}"; if [ -z "$GH_AW_NODE_EXEC" ] || [ ! -x "$GH_AW_NODE_EXEC" ]; then GH_AW_NODE_EXEC="$(command -v node 2>/dev/null || true)"; fi; if [ -z "$GH_AW_NODE_EXEC" ]; then echo "node runtime missing on this runner — check runtimes.node in workflow YAML" >&2; exit 127; fi; ` + nodePathSetupCommand + `; "$GH_AW_NODE_EXEC"`
const nodePathSetupCommandForCopilotSDK = `GH_AW_WORKSPACE_NODE_MODULES="${GITHUB_WORKSPACE:-$PWD}/node_modules"; if [ -d "$GH_AW_WORKSPACE_NODE_MODULES" ]; then export NODE_PATH="${GH_AW_WORKSPACE_NODE_MODULES}${NODE_PATH:+:${NODE_PATH}}"; fi; ` + nodePathSetupCommand
const nodeRuntimeResolutionCommandForCopilotSDK = `GH_AW_NODE_EXEC="${GH_AW_NODE_BIN:-}"; if [ -z "$GH_AW_NODE_EXEC" ] || [ ! -x "$GH_AW_NODE_EXEC" ]; then GH_AW_NODE_EXEC="$(command -v node 2>/dev/null || true)"; fi; if [ -z "$GH_AW_NODE_EXEC" ]; then echo "node runtime missing on this runner — check runtimes.node in workflow YAML" >&2; exit 127; fi; ` + nodePathSetupCommandForCopilotSDK + `; "$GH_AW_NODE_EXEC"`

// copilotSDKDriverExecArgs returns the runtime command and driver path argument for the
// given SDK driver filename.
//
// For language scripts with a recognized extension, the runtime command is the appropriate
// language executor and driverArg is the driver filename (which the caller will prefix with
// SetupActionDestinationShell). For bare command names (no extension), the driver is treated
// as an arbitrary executable in PATH: runtimeCmd is the command itself and driverArg is empty.
//
//   - .js/.cjs/.mjs → ("$GH_AW_NODE_EXEC", "driver.cjs")
//   - .py           → ("python3",           "driver.py")
//   - .ts/.mts      → ("ts-node",           "driver.ts")
//   - .rb           → ("ruby",              "driver.rb")
//   - (no ext)      → ("my-driver",         "")
func copilotSDKDriverExecArgs(driverName string) (runtimeCmd, driverArg string) {
	ext := strings.ToLower(filepath.Ext(driverName))
	switch ext {
	case ".js", ".cjs", ".mjs":
		return `"$GH_AW_NODE_EXEC"`, driverName
	case ".py":
		return "python3", driverName
	case ".ts", ".mts":
		return "ts-node", driverName
	case ".rb":
		return "ruby", driverName
	default:
		// No extension — arbitrary command in PATH; use name directly as command.
		return driverName, ""
	}
}

// GetExecutionSteps returns the GitHub Actions steps for executing GitHub Copilot CLI
func (e *CopilotEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	copilotExecLog.Printf("Generating execution steps for Copilot: workflow=%s, firewall=%v", workflowData.Name, isFirewallEnabled(workflowData))

	var steps []GitHubActionStep

	// Build copilot CLI arguments based on configuration
	var copilotArgs []string
	sandboxEnabled := isFirewallEnabled(workflowData)
	llmProvider := e.ResolveLLMProvider(workflowData)
	providerOverrideBYOK := llmProvider != LLMProviderGitHub && sandboxEnabled
	// isBYOKMode is true when the user has set COPILOT_PROVIDER_BASE_URL in engine.env,
	// which routes Copilot requests to a non-GitHub provider. In that mode the GitHub
	// identity token (COPILOT_GITHUB_TOKEN) must NOT be injected into the step env:
	// forwarding it to a third-party host would be a credential leak.
	isBYOKMode := providerOverrideBYOK || engineEnvHasKey(workflowData, constants.CopilotProviderBaseURL)
	if sandboxEnabled {
		// Simplified args for sandbox mode (AWF)
		copilotArgs = []string{"--add-dir", constants.TmpGhAwDirSlash, "--log-level", "all", "--log-dir", logsFolder}

		// Note: --add-dir "${GITHUB_WORKSPACE}" is appended raw after shellJoinArgs below
		// to allow shell variable expansion (cannot go through shellEscapeArg).
		copilotExecLog.Print("Added workspace directory to --add-dir")

		copilotExecLog.Print("Using firewall mode with simplified arguments")
	} else {
		// Original args for non-sandbox mode
		copilotArgs = []string{"--add-dir", "/tmp/", "--add-dir", constants.TmpGhAwDirSlash, "--add-dir", constants.TmpGhAwAgentDir, "--log-level", "all", "--log-dir", logsFolder}
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
			// Trailing slash tells copilot CLI to treat this as a directory.
			cacheDir := cacheMemoryDirFor(cache.ID) + "/"
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
	var customCommandScriptSetup string
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = customEngineCommandScriptPath
		customCommandScriptSetup = buildEngineCommandScriptSetup(workflowData.EngineConfig.Command)
		copilotExecLog.Printf("Using serialized custom command script: %s", commandName)
	} else if sandboxEnabled {
		// AWF - use the installed binary directly
		// The binary is mounted into the AWF container from /usr/local/bin/copilot
		commandName = constants.CopilotBinaryPath
	} else {
		// Non-sandbox mode: use standard copilot command
		commandName = "copilot"
	}

	// Build the command - model is always passed via COPILOT_MODEL env var (see env block below).
	// The --add-dir "${GITHUB_WORKSPACE}" arg is appended raw (not through shellJoinArgs)
	// because it contains a shell variable reference that must expand at runtime.
	//
	// When a harness script is provided (GetHarnessScriptName), wrap the copilot invocation with
	// `node <harness> <commandName> <args>` to enable retry logic for transient CAPIError 400 errors.
	//
	// Resolve node dynamically at runtime:
	// - Prefer GH_AW_NODE_BIN when set and executable.
	// - Fall back to `command -v node` if GH_AW_NODE_BIN points to a non-mounted toolcache path.
	// This prevents agent startup failures when host toolcache paths are not present in the AWF container.
	harnessScriptName := e.GetHarnessScriptName()
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.HarnessScript != "" {
		harnessScriptName = workflowData.EngineConfig.HarnessScript
	}
	isCopilotSDKMode := workflowData.EngineConfig != nil && workflowData.EngineConfig.CopilotSDK
	sdkDriverScriptName := "copilot_sdk_driver.cjs"
	customSDKDriverConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Driver != ""
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Driver != "" {
		sdkDriverScriptName = workflowData.EngineConfig.Driver
	}

	// copilotSDKServerArgsJSON holds the JSON-encoded server-args array that will be set in
	// GH_AW_COPILOT_SDK_SERVER_ARGS when copilot-sdk: true. It is declared here so that the
	// env-block section further down can reference the same value that was computed while
	// building the command, avoiding the need to re-derive it separately.
	var copilotSDKServerArgsJSON string

	var execPrefix string
	if harnessScriptName != "" {
		// Harness wraps the copilot subprocess; ${RUNNER_TEMP} and ${GH_AW_NODE_BIN} expand in the shell context.
		runtimeResolutionCommand := nodeRuntimeResolutionCommand
		if isCopilotSDKMode {
			runtimeResolutionCommand = nodeRuntimeResolutionCommandForCopilotSDK
		}
		if isCopilotSDKMode {
			// Driver mode: the harness receives the driver runtime command and the driver path (or just
			// the arbitrary command) as its argv, then calls runProcess(command, args) on the driver.
			//
			// For language scripts (.js/.cjs/.mjs, .py, .ts/.mts, .rb), the runtime command is
			// determined by extension. Built-in drivers resolve from the setup action directory;
			// custom drivers are specified as a path relative to ${GITHUB_WORKSPACE}:
			//   .js/.cjs/.mjs → "$GH_AW_NODE_EXEC" driver.cjs copilot-binary
			//   .py            → python3             driver.py  copilot-binary
			//   .ts/.mts       → ts-node             driver.ts  copilot-binary
			//   .rb            → ruby                driver.rb  copilot-binary
			//
			// For bare command names (no extension), the driver is treated as an arbitrary executable
			// in PATH — no SetupActionDestinationShell prefix, no runtime wrapper:
			//   my-driver copilot-binary
			driverRuntimeCmd, driverArg := copilotSDKDriverExecArgs(sdkDriverScriptName)
			if driverArg != "" {
				var driverPath string
				if customSDKDriverConfigured {
					// Custom driver: sdkDriverScriptName is a validated workspace-relative path.
					// Validation ensures no shell metacharacters, quotes, or path traversal,
					// so it is safe to embed directly in the double-quoted shell argument.
					driverPath = `"${GITHUB_WORKSPACE}/` + sdkDriverScriptName + `"`
				} else {
					driverPath = fmt.Sprintf(`"%s/%s"`, SetupActionDestinationShell, sdkDriverScriptName)
				}
				// Language script: harness runs <runtime> <setup-action-dir>/<harness> <runtime> <driver-path> <copilot-binary>
				execPrefix = fmt.Sprintf(`%s %s/%s %s %s %s`,
					runtimeResolutionCommand, SetupActionDestinationShell, harnessScriptName,
					driverRuntimeCmd,
					driverPath, commandName)
			} else {
				// Arbitrary command: harness runs <driver-cmd> <copilot-binary> directly
				execPrefix = fmt.Sprintf(`%s %s/%s %s %s`,
					runtimeResolutionCommand, SetupActionDestinationShell, harnessScriptName,
					driverRuntimeCmd, commandName)
			}
		} else {
			execPrefix = fmt.Sprintf(`%s %s/%s %s`, runtimeResolutionCommand, SetupActionDestinationShell, harnessScriptName, commandName)
		}
	} else {
		execPrefix = commandName
	}

	if isCopilotSDKMode {
		// SDK driver mode: configuration is passed via environment variables so that
		// copilot_sdk_driver.cjs is a self-contained program started by the harness
		// like any other command.
		//
		// GH_AW_COPILOT_SDK_SERVER_ARGS carries the JSON-encoded CLI argument list for
		// the headless Copilot CLI sidecar (--headless, --no-auto-update, --port, and all
		// configuration flags). The driver reads this at runtime and passes the args
		// directly to the spawned sidecar process without any argument parsing.
		//
		// The driver appends --add-dir $GITHUB_WORKSPACE automatically when that env var
		// is set, so addWorkspaceDir does not need to be signalled separately.
		serverArgs := append(
			[]string{"--headless", "--no-auto-update", "--port", strconv.Itoa(constants.DefaultCopilotSDKPort)},
			copilotArgs...,
		)
		serverArgsJSON, err := json.Marshal(serverArgs)
		if err != nil {
			// This should never happen with a plain string slice, but fall back to an
			// empty array so the run is not blocked.
			copilotExecLog.Printf("warning: failed to marshal SDK server args: %v; falling back to empty array", err)
			serverArgsJSON = []byte(`[]`)
		}
		copilotSDKServerArgsJSON = string(serverArgsJSON)
		// No CLI args are appended; all options are in env vars.
		copilotCommand = execPrefix
	} else if sandboxEnabled {
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
		} else if workflowData.CachedAllowedDomainsComputed {
			// Use the pre-warmed cache (populated before GetExecutionSteps is called)
			// to avoid re-running the expensive map+sort operation.
			allowedDomains = workflowData.CachedAllowedDomainsStr
		} else {
			allowedDomains = GetAllowedDomainsForEngine(constants.CopilotEngine, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
		}
		// Add Copilot BYOK/API target domains to the firewall allow-list.
		// This keeps normal and detection runs in sync while preserving detection's
		// otherwise-minimal network footprint.
		for _, copilotTarget := range GetCopilotAllowlistTargets(workflowData) {
			allowedDomains = mergeAPITargetDomains(allowedDomains, copilotTarget)
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
		// AWF v0.15.0+ uses chroot mode by default, and sudo's secure_path may strip
		// the PATH additions from actions/setup-node, so the container may not find node.
		//
		// Prepend GetNpmBinPathSetup() to the engine command so it runs inside the
		// AWF container before the node resolution command. This adds RUNNER_TOOL_CACHE
		// bin directories to PATH, ensuring that the command -v node fallback in
		// nodeRuntimeResolutionCommand succeeds regardless of the runner's tool cache
		// location. This mirrors the pattern used by the Claude and Codex engines.
		npmPathSetup := GetNpmBinPathSetup()
		engineCommand := fmt.Sprintf("%s && %s", npmPathSetup, copilotCommand)

		// MCP CLI bin directory: when cli-proxy is enabled, the CLI wrapper scripts
		// live under ${RUNNER_TEMP}/gh-aw/mcp-cli/bin. core.addPath() adds this to
		// $GITHUB_PATH for subsequent steps, but sudo's secure_path may strip it.
		// Prepending it to the engine command ensures the agent can find them.
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			engineCommand = fmt.Sprintf("%s && %s", mcpCLIPath, engineCommand)
		}
		pathSetup := "touch " + AgentStepSummaryPath + "\n" +
			"GH_AW_NODE_BIN=$(command -v node 2>/dev/null || true)\n" +
			"export GH_AW_NODE_BIN\n" +
			// Export COPILOT_API_KEY via shell variable expansion so the sentinel
			// value is never written as a literal next to a *_API_KEY key in the
			// generated YAML env: block. GitHub Actions env: values are not
			// shell-expanded, but this run: shell script is — $COPILOT_DUMMY_BYOK
			// expands to the sentinel before sudo -E awf runs, and sudo -E preserves
			// the variable for the AWF container.
			"export COPILOT_API_KEY=\"$" + constants.CopilotBYOKDummyAPIKeyEnvVar + "\""
		if customCommandScriptSetup != "" {
			pathSetup = customCommandScriptSetup + "\n" + pathSetup
		}
		// Write the Copilot settings file before AWF starts. The file is created on the
		// host and AWF mounts it into the container, where the Copilot CLI reads it to
		// disable the rubber-duck sub-agent.
		pathSetup = buildCopilotSettingsCleanupTrap() + buildCopilotSettingsSetup(customCommandScriptSetup != "") + buildCopilotMCPConfigExport(workflowData) + pathSetup
		// Build the list of core secret var names to hide from the agent shell tools.
		// In BYOK mode COPILOT_GITHUB_TOKEN is not injected into the step env at all,
		// so there is nothing to exclude. Excluding it unconditionally would produce
		// spurious --exclude-env flags when the token is absent.
		var copilotCoreSecrets []string
		if !isBYOKMode {
			copilotCoreSecrets = []string{"COPILOT_GITHUB_TOKEN"}
		}
		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "copilot",
			EngineCommand:  engineCommand,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false, // Copilot doesn't require TTY
			AllowedDomains: allowedDomains,
			// Keep max-ai-credits runtime expression in step env (not run:) to reduce
			// template-injection findings on generated Execute GitHub Copilot CLI steps.
			ResolveMaxAICreditsFromEnv: true,
			// Create the agent step summary file before AWF starts so it is accessible
			// inside the sandbox. The agent writes its step summary content here, and the
			// file is appended to $GITHUB_STEP_SUMMARY after secret redaction.
			//
			// Resolve the absolute node binary path before `sudo -E awf` runs.
			// On GPU runners (e.g. aw-gpu-runner-T4) sudo resets PATH via sudoers
			// secure_path, stripping the actions/setup-node directory.  By capturing
			// the path here (where PATH is still intact) and exporting it, sudo -E
			// preserves the variable and AWF's --env-all forwards it into the container,
			// where the execution command validates GH_AW_NODE_BIN and falls back to
			// command -v node (now reliably in PATH via GetNpmBinPathSetup above).
			PathSetup: pathSetup,
			// Exclude every env var whose step-env value is a secret so the agent
			// cannot read raw token values via bash tools (env / printenv).
			ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, copilotCoreSecrets),
		})
	} else {
		// Run copilot command without AWF wrapper.
		// Prepend a touch command to create the agent step summary file before copilot runs.
		preCommandSetup := mkdirCommands.String()
		if customCommandScriptSetup != "" {
			preCommandSetup = customCommandScriptSetup + "\n" + preCommandSetup
		}
		// Write the Copilot settings file before the agent runs to disable the rubber-duck
		// sub-agent. This reduces token overhead and latency for Copilot engine runs.
		preCommandSetup = buildCopilotSettingsCleanupTrap() + buildCopilotSettingsSetup(customCommandScriptSetup != "") + buildCopilotMCPConfigExport(workflowData) + preCommandSetup
		command = fmt.Sprintf(`set -o pipefail
printf '%%s' "$(date +%%s%%3N)" > %s
touch %s
(umask 177 && touch %s)
%s%s 2>&1 | tee %s`, AgentCLIStartMsPath, AgentStepSummaryPath, logFile, preCommandSetup, copilotCommand, logFile)
	}

	// COPILOT_GITHUB_TOKEN injection: in BYOK mode (COPILOT_PROVIDER_BASE_URL set), skip
	// this entirely. The request goes to a third-party provider; forwarding the GitHub
	// identity token would be a credential leak. The token is only needed for GitHub's
	// own Copilot backend. When not in BYOK mode, use the GitHub Actions token when
	// permissions.copilot-requests is write, otherwise use the COPILOT_GITHUB_TOKEN secret.
	// #nosec G101 -- These are NOT hardcoded credentials. They are GitHub Actions expression templates
	// that the runtime replaces with actual values. The strings "${{ secrets.COPILOT_GITHUB_TOKEN }}"
	// and "${{ github.token }}" are placeholders, not actual credentials.
	var copilotGitHubToken string
	useCopilotRequests := hasCopilotRequestsWritePermission(workflowData)
	if isBYOKMode {
		copilotExecLog.Print("Skipping COPILOT_GITHUB_TOKEN injection: BYOK mode active (COPILOT_PROVIDER_BASE_URL is set)")
	} else if useCopilotRequests {
		copilotGitHubToken = "${{ github.token }}"
		copilotExecLog.Print("Using GitHub Actions token as COPILOT_GITHUB_TOKEN (permissions.copilot-requests=write)")
	} else {
		copilotGitHubToken = "${{ secrets.COPILOT_GITHUB_TOKEN }}"
	}
	timeoutValue := strconv.Itoa(int(constants.DefaultAgenticWorkflowTimeout / time.Minute))
	if workflowData.TimeoutMinutes != "" {
		rawTimeoutValue := strings.TrimSpace(workflowData.TimeoutMinutes)
		if after, ok := strings.CutPrefix(rawTimeoutValue, "timeout-minutes:"); ok {
			rawTimeoutValue = strings.TrimSpace(after)
		}
		if rawTimeoutValue != "" {
			timeoutValue = rawTimeoutValue
		}
	}

	env := map[string]string{
		"COPILOT_AGENT_RUNNER_TYPE": "STANDALONE",
		// Note: XDG_CONFIG_HOME is exported from the run script (see buildCopilotMCPConfigExport)
		// rather than set here so $HOME is resolved at runtime; GitHub Actions does not
		// shell-expand env: values, and HOME may differ from /home/runner on self-hosted runners.
		// Override GITHUB_STEP_SUMMARY with a path that exists inside the sandbox.
		// The runner's original path is unreachable within the AWF isolated filesystem;
		// we create this file before the agent starts and append it to the real
		// $GITHUB_STEP_SUMMARY after secret redaction.
		"GITHUB_STEP_SUMMARY":   AgentStepSummaryPath,
		"GITHUB_HEAD_REF":       "${{ github.head_ref }}",
		"GITHUB_REF_NAME":       "${{ github.ref_name }}",
		"GITHUB_WORKSPACE":      "${{ github.workspace }}",
		"RUNNER_TEMP":           "${{ runner.temp }}",
		"GH_AW_TIMEOUT_MINUTES": timeoutValue,
		// Pass GitHub server URL and API URL for GitHub Enterprise compatibility.
		// In standard GitHub.com environments these resolve to https://github.com and
		// https://api.github.com. In GitHub Enterprise they resolve to the enterprise
		// server URL (e.g. https://COMPANY.ghe.com and https://COMPANY.ghe.com/api/v3).
		"GITHUB_SERVER_URL": "${{ github.server_url }}",
		"GITHUB_API_URL":    "${{ github.api_url }}",
	}
	env["GH_AW_LLM_PROVIDER"] = llmProvider

	// Auto-configure Copilot BYOK routing when engine.model-provider selects a non-GitHub provider.
	// Explicit engine.env values still win later via maps.Copy.
	if providerOverrideBYOK {
		env[constants.CopilotProviderBaseURL] = llmProviderGatewayBaseURL(llmProvider)
		env[constants.CopilotProviderAPIKey] = llmProviderSecretExpression(llmProvider, workflowData)
	}

	// Inject the GitHub token only when not in BYOK mode. The engine.env merge that
	// happens later (maps.Copy(env, workflowData.EngineConfig.Env)) can still override
	// or nullify this if the user explicitly sets COPILOT_GITHUB_TOKEN in engine.env.
	if !isBYOKMode {
		env["COPILOT_GITHUB_TOKEN"] = copilotGitHubToken
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)

	// When permissions.copilot-requests is write, set S2STOKENS=true to allow the Copilot CLI
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
	env["GH_AW_PROMPT"] = constants.AwPromptsFile

	// Tag the step as a GitHub AW agentic execution for discoverability by agents
	env["GITHUB_AW"] = "true"
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
	applyDefaultMaxAICreditsEnvToMap(env, workflowData)

	// Add GH_AW_MCP_CONFIG for MCP server configuration only if there are MCP servers.
	// The value is exported from the run script (see buildCopilotMCPConfigExport) so
	// that $HOME is resolved at runtime; GitHub Actions does not shell-expand env: values.

	if hasGitHubTool(workflowData.ParsedTools) {
		// If GitHub App is configured, use the app token minted directly in the agent job.
		// The token cannot be passed via job outputs from the activation job because
		// actions/create-github-app-token calls ::add-mask:: on the token, and the
		// GitHub Actions runner silently drops masked values in job outputs (runner v2.308+).
		if workflowData.ParsedTools != nil && workflowData.ParsedTools.GitHub != nil && workflowData.ParsedTools.GitHub.GitHubApp != nil {
			tokenExpression := "${{ steps.github-mcp-app-token.outputs.token }}"
			if workflowData.ParsedTools.GitHub.GitHubApp.shouldIgnoreMissingKey() {
				githubToolConfig, _ := workflowData.Tools["github"].(map[string]any)
				customGitHubToken := getGitHubToken(githubToolConfig)
				tokenExpression = combineTokenExpressions(tokenExpression, getEffectiveGitHubToken(customGitHubToken))
			}
			env["GITHUB_MCP_SERVER_TOKEN"] = tokenExpression
		} else {
			githubToolConfig, _ := workflowData.Tools["github"].(map[string]any)
			customGitHubToken := getGitHubToken(githubToolConfig)
			// Use effective token with precedence: custom > default
			effectiveToken := getEffectiveGitHubToken(customGitHubToken)
			env["GITHUB_MCP_SERVER_TOKEN"] = effectiveToken
		}
	}

	// Add GH_AW_SAFE_OUTPUTS if output is needed
	applySafeOutputEnvToMap(env, workflowData)

	// Propagate W3C trace context so engine spans nest under the gh-aw.agent.setup span.
	applyTraceContextEnvToMap(env)

	applyOptionalEngineToolTimeouts(env, workflowData)
	applyEngineMaxTurnsEnv(env, workflowData)

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.CopilotSDK {
		if workflowData.EngineConfig.MaxToolDenials != "" {
			env[constants.EnvVarMaxToolDenials] = workflowData.EngineConfig.MaxToolDenials
		} else {
			env[constants.EnvVarMaxToolDenials] = strconv.Itoa(constants.DefaultMaxToolDenials)
		}
	}

	// Set the model environment variable.
	// The model is always passed via the native COPILOT_MODEL env var, which the Copilot CLI reads
	// directly. This avoids embedding the value in the shell command (which would fail template
	// injection validation for GitHub Actions expressions like ${{ inputs.model }}).
	// When model is explicitly configured, use its value directly.
	// When model is not configured, map the GitHub org variable to COPILOT_MODEL so users can set
	// a default via GitHub Actions variables without requiring per-workflow frontmatter changes.
	// Copilot uses BYOK defaults by default and requires a non-empty fallback model.
	if modelConfigured {
		copilotExecLog.Printf("Setting %s env var for model: %s", constants.CopilotCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.CopilotCLIModelEnvVar] = workflowData.EngineConfig.Model
	} else {
		env[constants.CopilotCLIModelEnvVar] = compilerenv.BuildModelOverrideExpression(modelEnvVar, compilerenv.DefaultModelCopilot, constants.CopilotBYOKDefaultModel)
	}

	// Inject GH_AW_ENGINE_CWD when engine.cwd is configured.
	applyEngineCwdEnv(env, workflowData)

	applyEngineAndAgentEnv(env, workflowData, copilotExecLog)

	// Always inject the Copilot integration ID for agentic workflows after all env merges
	// so user-supplied env does not override this value.
	env[constants.CopilotCLIIntegrationIDEnvVar] = constants.CopilotCLIIntegrationIDValue

	// Inject the dummy BYOK sentinel and AWF_REFLECT_ENABLED only when the AWF sandbox
	// is active. The COPILOT_API_KEY (set to this value) triggers AWF's runtime BYOK
	// detection path, which requires the api-proxy sidecar to be running. When
	// sandbox.agent: false, no api-proxy is started, so injecting the key would break
	// Copilot CLI authentication. Similarly, AWF_REFLECT_ENABLED tells the harness to
	// skip the /reflect preflight when the api-proxy is not available.
	//
	// To avoid secret-scanner false positives on generated lock files, the sentinel
	// value is placed in COPILOT_DUMMY_BYOK (a non-*_API_KEY-shaped variable) in the
	// env: block. COPILOT_API_KEY itself is exported in PathSetup via:
	//   export COPILOT_API_KEY="$COPILOT_DUMMY_BYOK"
	// Shell variable expansion runs before sudo -E awf executes, so COPILOT_API_KEY
	// receives the correct value at runtime without ever appearing as a YAML key with
	// a token-shaped literal value (GitHub Actions env: values are not shell-expanded).
	if sandboxEnabled {
		env[constants.CopilotBYOKDummyAPIKeyEnvVar] = constants.CopilotBYOKDummyAPIKey
		env["AWF_REFLECT_ENABLED"] = "1"
	}

	// When copilot-sdk: true, provide the SDK URI that the harness uses to start a
	// separate Copilot CLI headless server and that child processes use to connect.
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.CopilotSDK {
		env[constants.CopilotSDKURIEnvVar] = fmt.Sprintf("http://127.0.0.1:%d", constants.DefaultCopilotSDKPort)
		copilotExecLog.Printf("copilot-sdk enabled: set %s=%s", constants.CopilotSDKURIEnvVar, env[constants.CopilotSDKURIEnvVar])
		// Signal the harness to start the driver as a normal subprocess rather than
		// managing the SDK session inline.
		env[constants.CopilotSDKDriverEnvVar] = "1"
		// Provide the complete CLI argument list for the headless sidecar so the
		// harness can start it in driver mode without any argument parsing.
		env[constants.CopilotSDKServerArgsEnvVar] = copilotSDKServerArgsJSON
		copilotExecLog.Printf("copilot-sdk driver mode: set %s and %s", constants.CopilotSDKDriverEnvVar, constants.CopilotSDKServerArgsEnvVar)
	}

	// Add HTTP MCP header secrets to env for passthrough
	headerSecrets := collectHTTPMCPHeaderSecrets(workflowData.Tools)
	for varName, secretExpr := range headerSecrets {
		// Only add if not already in env
		if _, exists := env[varName]; !exists {
			env[varName] = secretExpr
		}
	}

	applyMCPScriptsSecretEnv(env, workflowData)

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
	stepLines = append(stepLines, "        timeout-minutes: "+timeoutValue)

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

// copilotSupportsNoAskUser returns true when the effective Copilot CLI version supports the
// --no-ask-user flag, which enables fully autonomous agentic runs by suppressing interactive prompts.
//
// The --no-ask-user flag was introduced in Copilot CLI v1.0.19. Any workflow that pins an
// explicit version older than v1.0.19 must not emit --no-ask-user or the run will fail at startup.
//
// Special cases:
//   - No version override (engineConfig is nil or has no Version): use
//     DefaultCopilotVersion. This preserves existing behavior while avoiding drift if
//     DefaultCopilotVersion is ever lowered below CopilotNoAskUserMinVersion.
//   - "latest": always returns true (latest is always a new release).
//   - Any semver string ≥ CopilotNoAskUserMinVersion: returns true.
//   - Any semver string < CopilotNoAskUserMinVersion: returns false.
//   - Non-semver string (e.g. a branch name): returns false (conservative).
func copilotSupportsNoAskUser(engineConfig *EngineConfig) bool {
	var versionStr string
	if engineConfig != nil && engineConfig.Version != "" {
		versionStr = engineConfig.Version
	}
	return versionAtLeast(
		versionStr,
		string(constants.DefaultCopilotVersion),
		string(constants.CopilotNoAskUserMinVersion),
	)
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

func buildEngineCommandScriptSetup(command string) string {
	// engine.command intentionally accepts shell-form commands from trusted workflow
	// configuration authored in-repo; preserve shell semantics and forward driver args.
	scriptContent := fmt.Sprintf("#!/usr/bin/env bash\nset +o histexpand\nset -eo pipefail\n%s \"$@\"\n", command)
	heredocDelimiter := "GH_AW_ENGINE_COMMAND_EOF"
	for strings.Contains(scriptContent, heredocDelimiter) {
		heredocDelimiter += "_X"
	}

	return fmt.Sprintf(`mkdir -p /tmp/gh-aw
umask 0177
cat > %s <<'%s'
%s
%s
chmod 700 %s`, customEngineCommandScriptPath, heredocDelimiter, scriptContent, heredocDelimiter, customEngineCommandScriptPath)
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
