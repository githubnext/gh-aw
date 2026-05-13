// This file provides helper functions for AWF (Agentic Workflow Firewall) integration.
//
// AWF is the network firewall/sandbox used by gh-aw to control network egress for
// AI agent execution. This file consolidates common AWF logic that was previously
// duplicated across multiple engine implementations (Copilot, Claude, Codex).
//
// # Key Functions
//
// AWF Command Building:
//   - BuildAWFCommand() - Builds complete AWF command with all arguments
//   - BuildAWFArgs() - Constructs common AWF arguments from configuration
//   - GetAWFCommandPrefix() - Determines AWF command (custom vs standard)
//   - WrapCommandInShell() - Wraps engine command in shell for AWF execution
//
// AWF Configuration:
//   - GetAWFDomains() - Combines allowed/blocked domains from various sources
//   - GetSSLBumpArgs() - Returns SSL bump configuration arguments
//   - GetAWFImageTag() - Returns pinned AWF image tag
//
// These functions extract shared AWF patterns from engine implementations,
// providing a consistent and maintainable approach to AWF integration.

package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var awfHelpersLog = logger.New("workflow:awf_helpers")

const (
	awfArcDindPrefixArgsVarName = "GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS"
	// Bash regex used in [[ ... =~ ... ]] to detect localhost TCP Docker hosts.
	// Keep this in bash-compatible syntax (escaped dots) because it is emitted directly
	// into generated shell scripts.
	awfArcDindDockerHostRegex    = `^tcp://(localhost|127\.0\.0\.1)(:[0-9]+)?$`
	awfArcDindHostPathPrefixFlag = "--docker-host-path-prefix /tmp/gh-aw"
)

// AWFCommandConfig contains configuration for building AWF commands.
// This struct centralizes all the parameters needed to construct an AWF-wrapped command.
type AWFCommandConfig struct {
	// EngineName is the engine ID (e.g., "copilot", "claude", "codex")
	EngineName string

	// EngineCommand is the command to execute inside AWF
	EngineCommand string

	// LogFile is the path to the log file
	LogFile string

	// WorkflowData contains all workflow configuration
	WorkflowData *WorkflowData

	// UsesTTY indicates if the engine requires a TTY (e.g., Claude)
	UsesTTY bool

	// AllowedDomains is the comma-separated list of allowed domains
	AllowedDomains string

	// PathSetup is optional shell commands to run before the engine command
	// (e.g., npm PATH setup)
	PathSetup string

	// ExcludeEnvVarNames is the list of environment variable names to exclude from
	// the agent container's visible environment via --exclude-env. These are the env
	// var keys whose step-env values contain secret references (${{ secrets.* }}).
	// Computed from the engine's GetRequiredSecretNames() so that every secret-bearing
	// variable is excluded — the agent can never read raw token values via `env`/`printenv`.
	// Requires AWF v0.25.3+ for --exclude-env support.
	ExcludeEnvVarNames []string
}

// BuildAWFCommand builds a complete AWF command with all arguments.
// This consolidates the AWF command building logic that was duplicated across
// Copilot, Claude, and Codex engines.
//
// Parameters:
//   - config: AWF command configuration
//
// Returns:
//   - string: Complete AWF command with arguments and wrapped engine command
func BuildAWFCommand(config AWFCommandConfig) string {
	awfHelpersLog.Printf("Building AWF command for engine: %s", config.EngineName)

	// Get AWF command prefix (custom or standard)
	awfCommand := GetAWFCommandPrefix(config.WorkflowData)

	// Build AWF arguments. The returned list contains only args that are safe to pass
	// through shellJoinArgs. Expandable-var args (--container-workdir "${GITHUB_WORKSPACE}"
	// and --mount "${RUNNER_TEMP}/...") are appended raw below so that shell variable
	// expansion is not suppressed by single-quoting.
	awfArgs := BuildAWFArgs(config)
	firewallConfig := getFirewallConfig(config.WorkflowData)

	// Auto-detect ARC/DinD split daemon topology at runtime and emit
	// --docker-host-path-prefix when supported by the selected AWF version.
	// This avoids requiring workflow-authored sandbox.agent.args for standard ARC DinD setups.
	arcDindPrefixProbe := ""
	arcDindPrefixArgsRef := ""
	if awfSupportsDockerHostPathPrefix(firewallConfig) {
		arcDindPrefixProbe = fmt.Sprintf(`%s=""
if [[ "${DOCKER_HOST:-}" =~ %s ]]; then
  %s="%s"
fi`,
			awfArcDindPrefixArgsVarName,
			awfArcDindDockerHostRegex,
			awfArcDindPrefixArgsVarName,
			awfArcDindHostPathPrefixFlag)
		arcDindPrefixArgsRef = fmt.Sprintf("${%s}", awfArcDindPrefixArgsVarName)
	}

	// Build the expandable args string for args that need shell variable expansion.
	// These MUST be appended as raw (unescaped) strings because single-quoting would
	// prevent the runner's shell from expanding ${GITHUB_WORKSPACE} and ${RUNNER_TEMP}.
	ghAwDir := "${RUNNER_TEMP}/gh-aw"
	expandableArgs := fmt.Sprintf(
		`--container-workdir "${GITHUB_WORKSPACE}" --mount "%s:%s:ro" --mount "%s:/host%s:ro"`,
		ghAwDir, ghAwDir, ghAwDir, ghAwDir,
	)

	// Generate a JSON config file and reference it via --config "${RUNNER_TEMP}/gh-aw/awf-config.json".
	// This replaces several verbose CLI flags (--allow-domains, --enable-api-proxy, --image-tag,
	// API targets) with a structured JSON file that is easier to audit and extend.
	//
	// The config file is written at runtime (inside the run: step) immediately before the AWF
	// invocation, using printf to a fixed path inside the pre-existing ${RUNNER_TEMP}/gh-aw/
	// directory that is already set up by actions/setup.
	var configFileSetup string
	awfConfigJSON, err := BuildAWFConfigJSON(config)
	if err != nil {
		awfHelpersLog.Printf("Warning: failed to build AWF config JSON: %v", err)
	} else {
		// Write the config JSON to ${RUNNER_TEMP}/gh-aw/awf-config.json before AWF runs.
		// printf '%s\n' '...' is safe here because JSON uses only double quotes (never
		// single quotes), so single-quoting the JSON string requires no further escaping
		// in practice. shellEscapeArg handles the edge case where a domain value might
		// somehow contain a single quote.
		// Write the config to ${RUNNER_TEMP}/gh-aw/awf-config.json (host path read by AWF at
		// startup) and also copy it to /tmp/gh-aw/awf-config.json so the unified agent artifact
		// upload can include it alongside the other /tmp/gh-aw/ files.
		configFileSetup = fmt.Sprintf(
			"printf '%%s\\n' %s > \"${RUNNER_TEMP}/gh-aw/awf-config.json\" && cp \"${RUNNER_TEMP}/gh-aw/awf-config.json\" %s",
			shellEscapeArg(awfConfigJSON),
			constants.AWFConfigFilePath,
		)
		// Add --config as the first expandable arg so it appears before --container-workdir.
		expandableArgs = `--config "${RUNNER_TEMP}/gh-aw/awf-config.json" ` + expandableArgs
		awfHelpersLog.Print("Using AWF config file (--config flag)")
	}

	// When upload_artifact is configured, add a read-write mount for the staging directory
	// so the model can copy files there from inside the container. The parent ${RUNNER_TEMP}/gh-aw
	// is mounted :ro above; this child mount overrides access for the staging subdirectory only.
	// The staging directory must already exist on the host (created in Generate Safe Outputs Config step).
	if config.WorkflowData != nil && config.WorkflowData.SafeOutputs != nil && config.WorkflowData.SafeOutputs.UploadArtifact != nil {
		stagingDir := "${RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts"
		expandableArgs += fmt.Sprintf(` --mount "%s:%s:rw"`, stagingDir, stagingDir)
		awfHelpersLog.Print("Added read-write mount for upload_artifact staging directory")
	}

	// Add --allow-host-service-ports for services with port mappings.
	// This is appended as a raw (expandable) arg because the value contains
	// ${{ job.services.<id>.ports['<port>'] }} expressions that include single quotes.
	// These expressions are resolved by the GitHub Actions runner before shell execution,
	// so they must not be shell-escaped.
	if config.WorkflowData != nil && config.WorkflowData.ServicePortExpressions != "" {
		expandableArgs += fmt.Sprintf(` --allow-host-service-ports "%s"`, config.WorkflowData.ServicePortExpressions)
		awfHelpersLog.Printf("Added --allow-host-service-ports with %s", config.WorkflowData.ServicePortExpressions)
	}

	// Wrap engine command in shell (command already includes any internal setup like npm PATH)
	shellWrappedCommand := WrapCommandInShell(config.EngineCommand)

	// Pre-create the agent stdio log file with restrictive permissions (0600) before
	// starting the AWF container.  tee would otherwise create it with the default
	// umask (0644), leaving secrets (e.g. MCP gateway tokens) world-readable on the
	// runner host until the secret-redaction step runs.
	preCreateLog := fmt.Sprintf("(umask 177 && touch %s)", shellEscapeArg(config.LogFile))

	// Capture the epoch-millisecond timestamp at the very start of the Execute Agent CLI
	// step on the host, before the AWF container launches.  sendJobConclusionSpan reads
	// this file to set the dedicated gh-aw.<job>.agent span start time, which excludes
	// pre-agent overhead such as workspace audit and CLI proxy startup.
	writeAgentCLIStartMs := fmt.Sprintf("printf '%%s' \"$(date +%%s%%3N)\" > %s", shellEscapeArg(AgentCLIStartMsPath))

	// Build the complete command with proper formatting.
	// configFileSetup (if non-empty) writes the AWF config JSON immediately before the
	// AWF invocation so the file is present when AWF parses --config.
	var command string
	if config.PathSetup != "" && configFileSetup != "" {
		command = fmt.Sprintf(`set -o pipefail
%s
%s
%s
%s
%s
# shellcheck disable=SC1003
%s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
			writeAgentCLIStartMs,
			config.PathSetup,
			preCreateLog,
			configFileSetup,
			arcDindPrefixProbe,
			awfCommand,
			expandableArgs,
			arcDindPrefixArgsRef,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	} else if config.PathSetup != "" {
		// Include path setup before AWF command (runs on host before AWF)
		command = fmt.Sprintf(`set -o pipefail
%s
%s
%s
%s
# shellcheck disable=SC1003
%s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
			writeAgentCLIStartMs,
			config.PathSetup,
			preCreateLog,
			arcDindPrefixProbe,
			awfCommand,
			expandableArgs,
			arcDindPrefixArgsRef,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	} else if configFileSetup != "" {
		command = fmt.Sprintf(`set -o pipefail
%s
%s
%s
%s
# shellcheck disable=SC1003
%s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
			writeAgentCLIStartMs,
			preCreateLog,
			configFileSetup,
			arcDindPrefixProbe,
			awfCommand,
			expandableArgs,
			arcDindPrefixArgsRef,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	} else {
		command = fmt.Sprintf(`set -o pipefail
%s
%s
%s
# shellcheck disable=SC1003
%s %s %s %s \
  -- %s 2>&1 | tee -a %s`,
			writeAgentCLIStartMs,
			preCreateLog,
			arcDindPrefixProbe,
			awfCommand,
			expandableArgs,
			arcDindPrefixArgsRef,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	}

	awfHelpersLog.Print("Successfully built AWF command")
	return command
}

// BuildAWFArgs constructs common AWF arguments from configuration.
// This extracts the shared AWF argument building logic from engine implementations.
//
// The following flags are expressed in the generated JSON config file written by
// BuildAWFCommand and are therefore not emitted here:
//   - --allow-domains / --block-domains   → network.allowDomains / network.blockDomains
//   - --enable-api-proxy                  → apiProxy.enabled
//   - --image-tag                         → container.imageTag
//   - --openai-api-target                 → apiProxy.targets.openai.host
//   - --anthropic-api-target              → apiProxy.targets.anthropic.host
//   - --copilot-api-target                → apiProxy.targets.copilot.host
//   - --gemini-api-target                 → apiProxy.targets.gemini.host
//
// Parameters:
//   - config: AWF command configuration
//
// Returns:
//   - []string: List of AWF arguments (safe args only; expandable-var args like
//     --container-workdir and --mount are handled by BuildAWFCommand)
func BuildAWFArgs(config AWFCommandConfig) []string {
	awfHelpersLog.Printf("Building AWF args for engine: %s", config.EngineName)

	firewallConfig := getFirewallConfig(config.WorkflowData)
	agentConfig := getAgentConfig(config.WorkflowData)

	var awfArgs []string

	// Add TTY flag if needed (Claude requires this)
	if config.UsesTTY {
		awfArgs = append(awfArgs, "--tty")
	}

	// Pass all environment variables to the container, but exclude every variable whose
	// step-env value comes from a GitHub Actions secret. AWF's API proxy (--enable-api-proxy)
	// handles authentication for these tokens transparently, so the container does not need
	// the raw values. Excluding them via --exclude-env prevents a prompt-injected agent from
	// exfiltrating tokens through bash tools such as `env` or `printenv`.
	// The caller computes ExcludeEnvVarNames from ComputeAWFExcludeEnvVarNames() so that every
	// secret-bearing variable is covered — not just a hardcoded subset.
	// --exclude-env requires AWF v0.25.3+; skip the flags for workflows that pin an older version.
	awfArgs = append(awfArgs, "--env-all")
	if awfSupportsExcludeEnv(firewallConfig) {
		// Sort for deterministic output in compiled lock files.
		sortedExclude := make([]string, len(config.ExcludeEnvVarNames))
		copy(sortedExclude, config.ExcludeEnvVarNames)
		sort.Strings(sortedExclude)
		for _, excludedVar := range sortedExclude {
			awfArgs = append(awfArgs, "--exclude-env", excludedVar)
		}
	} else {
		awfHelpersLog.Printf("Skipping --exclude-env: AWF version %q is older than minimum %s", getAWFImageTag(firewallConfig), constants.AWFExcludeEnvMinVersion)
	}

	// Note: --container-workdir "${GITHUB_WORKSPACE}" and --mount "${RUNNER_TEMP}/gh-aw:..."
	// are intentionally NOT added here. They contain shell variable references that require
	// double-quote expansion. These args are appended raw in BuildAWFCommand to ensure
	// ${GITHUB_WORKSPACE} and ${RUNNER_TEMP} are expanded by the runner's shell.

	// Add custom mounts from agent config if specified
	if agentConfig != nil && len(agentConfig.Mounts) > 0 {
		// Sort mounts for consistent output
		sortedMounts := make([]string, len(agentConfig.Mounts))
		copy(sortedMounts, agentConfig.Mounts)
		sort.Strings(sortedMounts)

		for _, mount := range sortedMounts {
			awfArgs = append(awfArgs, "--mount", mount)
		}
		awfHelpersLog.Printf("Added %d custom mounts from agent config", len(sortedMounts))
	}

	// Set log level
	awfLogLevel := string(constants.AWFDefaultLogLevel)
	if firewallConfig != nil && firewallConfig.LogLevel != "" {
		awfLogLevel = firewallConfig.LogLevel
	}
	awfArgs = append(awfArgs, "--log-level", awfLogLevel)
	awfArgs = append(awfArgs, "--proxy-logs-dir", string(constants.AWFProxyLogsDir))
	awfArgs = append(awfArgs, "--audit-dir", string(constants.AWFAuditDir))
	if isFeatureEnabled(constants.AwfDiagnosticLogsFeatureFlag, config.WorkflowData) {
		awfArgs = append(awfArgs, "--diagnostic-logs")
		awfHelpersLog.Print("Added --diagnostic-logs because awf-diagnostic-logs feature flag is enabled")
	}

	// Always add --enable-host-access: needed for the API proxy sidecar
	// (to reach host.docker.internal:<port>) and for MCP gateway communication
	awfArgs = append(awfArgs, "--enable-host-access")
	awfHelpersLog.Print("Added --enable-host-access for API proxy and MCP gateway")

	// AWF's --enable-host-access defaults to ports 80,443. The MCP gateway now
	// listens on port 8080 (non-privileged), so we must explicitly allow it
	// when AWF supports --allow-host-ports.
	if awfSupportsAllowHostPorts(firewallConfig) {
		mcpGatewayPort := int(DefaultMCPGatewayPort)
		if config.WorkflowData != nil && config.WorkflowData.SandboxConfig != nil &&
			config.WorkflowData.SandboxConfig.MCP != nil && config.WorkflowData.SandboxConfig.MCP.Port > 0 {
			mcpGatewayPort = config.WorkflowData.SandboxConfig.MCP.Port
		}
		hostPorts := fmt.Sprintf("80,443,%d", mcpGatewayPort)
		awfArgs = append(awfArgs, "--allow-host-ports", hostPorts)
		awfHelpersLog.Printf("Added --allow-host-ports %s for MCP gateway access", hostPorts)
	} else {
		awfHelpersLog.Printf("Skipping --allow-host-ports: AWF version %q requires at least %s", getAWFImageTag(firewallConfig), constants.AWFAllowHostPortsMinVersion)
	}

	// Skip pulling images since they are pre-downloaded
	awfArgs = append(awfArgs, "--skip-pull")
	awfHelpersLog.Print("Using --skip-pull since images are pre-downloaded")

	// Enable CLI proxy sidecar when GitHub mode is gh-proxy.
	// Start the difc-proxy on the host and tell AWF where to connect
	// (firewall v0.25.17+).
	if isGitHubCLIModeEnabled(config.WorkflowData) {
		if awfSupportsCliProxy(firewallConfig) {
			awfArgs = append(awfArgs, "--difc-proxy-host", "host.docker.internal:18443")
			awfArgs = append(awfArgs, "--difc-proxy-ca-cert", "/tmp/gh-aw/difc-proxy-tls/ca.crt")
			awfHelpersLog.Print("Added --difc-proxy-host and --difc-proxy-ca-cert for CLI proxy sidecar")
		} else {
			awfHelpersLog.Printf("Skipping CLI proxy flags: AWF version %q is older than minimum %s", getAWFImageTag(firewallConfig), constants.AWFCliProxyMinVersion)
		}
	}

	// Pass base path if URL contains a path component
	// This is required for endpoints with path prefixes (e.g., Databricks /serving-endpoints,
	// Azure OpenAI /openai/deployments/<name>, corporate LLM routers with path-based routing)
	// Base paths remain as CLI flags — they are not yet represented in the config file schema.
	openaiBasePath := extractAPIBasePath(config.WorkflowData, "OPENAI_BASE_URL")
	if openaiBasePath != "" {
		awfArgs = append(awfArgs, "--openai-api-base-path", openaiBasePath)
		awfHelpersLog.Printf("Added --openai-api-base-path=%s", openaiBasePath)
	}

	anthropicBasePath := extractAPIBasePath(config.WorkflowData, "ANTHROPIC_BASE_URL")
	if anthropicBasePath != "" {
		awfArgs = append(awfArgs, "--anthropic-api-base-path", anthropicBasePath)
		awfHelpersLog.Printf("Added --anthropic-api-base-path=%s", anthropicBasePath)
	}

	geminiBasePath := extractAPIBasePath(config.WorkflowData, "GEMINI_API_BASE_URL")
	if geminiBasePath != "" {
		awfArgs = append(awfArgs, "--gemini-api-base-path", geminiBasePath)
		awfHelpersLog.Printf("Added --gemini-api-base-path=%s", geminiBasePath)
	}

	// Add SSL Bump support for HTTPS content inspection (v0.9.0+)
	sslBumpArgs := getSSLBumpArgs(firewallConfig)
	awfArgs = append(awfArgs, sslBumpArgs...)

	// Add custom args if specified in firewall config
	if firewallConfig != nil && len(firewallConfig.Args) > 0 {
		awfArgs = append(awfArgs, firewallConfig.Args...)
	}

	// Add custom args from agent config if specified
	if agentConfig != nil && len(agentConfig.Args) > 0 {
		awfArgs = append(awfArgs, agentConfig.Args...)
		awfHelpersLog.Printf("Added %d custom args from agent config", len(agentConfig.Args))
	}

	// Pass memory limit to AWF container if specified in agent config
	if agentConfig != nil && agentConfig.Memory != "" {
		awfArgs = append(awfArgs, "--memory-limit", agentConfig.Memory)
		awfHelpersLog.Printf("Set AWF memory limit to %s", agentConfig.Memory)
	}

	awfHelpersLog.Printf("Built %d AWF arguments", len(awfArgs))
	return awfArgs
}

// GetAWFCommandPrefix determines the AWF command to use (custom or standard).
// This extracts the common pattern for determining AWF command from agent config.
//
// Parameters:
//   - workflowData: The workflow data containing agent configuration
//
// Returns:
//   - string: The AWF command to use (e.g., "sudo -E awf" or custom command)
func GetAWFCommandPrefix(workflowData *WorkflowData) string {
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && agentConfig.Command != "" {
		awfHelpersLog.Printf("Using custom AWF command: %s", agentConfig.Command)
		return agentConfig.Command
	}

	awfHelpersLog.Print("Using standard AWF command")
	return string(constants.AWFDefaultCommand)
}

// buildAWFImageTagWithDigests returns an image tag value for AWF's --image-tag flag.
// When known firewall container digests are available, it appends AWF's digest
// metadata format:
//
//	<tag>,squid=sha256:...,agent=sha256:...,api-proxy=sha256:...,cli-proxy=sha256:...
//
// This keeps AWF sidecar configuration aligned with digest-pinned pre-download images.
func buildAWFImageTagWithDigests(imageTag string, workflowData *WorkflowData) string {
	if imageTag == "" {
		return imageTag
	}

	type digestSpec struct {
		name  string
		image string
	}
	specs := []digestSpec{
		{name: "squid", image: constants.DefaultFirewallRegistry + "/squid:" + imageTag},
		{name: "agent", image: constants.DefaultFirewallRegistry + "/agent:" + imageTag},
		{name: "agent-act", image: constants.DefaultFirewallRegistry + "/agent-act:" + imageTag},
		{name: "api-proxy", image: constants.DefaultFirewallRegistry + "/api-proxy:" + imageTag},
		{name: "cli-proxy", image: constants.DefaultFirewallRegistry + "/cli-proxy:" + imageTag},
	}

	parts := []string{imageTag}
	for _, spec := range specs {
		digest := resolveContainerDigest(spec.image, workflowData)
		if digest == "" {
			continue
		}
		parts = append(parts, spec.name+"="+digest)
	}

	if len(parts) == 1 {
		return imageTag
	}
	return strings.Join(parts, ",")
}

// resolveContainerDigest resolves a container image digest from cache first, then
// falls back to embedded container pins.
func resolveContainerDigest(image string, workflowData *WorkflowData) string {
	var cache *ActionCache
	if workflowData != nil {
		cache = workflowData.ActionCache
	}
	if pin, ok := lookupContainerPin(image, cache); ok && pin.Digest != "" {
		return pin.Digest
	}
	return ""
}

// WrapCommandInShell wraps an engine command in a shell invocation for AWF execution.
// This is needed because AWF requires commands to be wrapped in shell for proper execution.
//
// Parameters:
//   - command: The engine command to wrap (may include PATH setup and other initialization)
//
// Returns:
//   - string: Shell-wrapped command suitable for AWF execution
func WrapCommandInShell(command string) string {
	awfHelpersLog.Print("Wrapping command in shell for AWF execution")

	// Escape single quotes in the command by replacing ' with '\''
	escapedCommand := strings.ReplaceAll(command, "'", "'\\''")

	// Wrap in shell invocation
	return fmt.Sprintf("/bin/bash -c '%s'", escapedCommand)
}

// ComputeAWFExcludeEnvVarNames returns the list of environment variable names that must be
// excluded from the agent container's visible environment via AWF's --exclude-env flag.
//
// Only env var names whose step-env values WILL contain a ${{ secrets.* }} reference are
// included, so non-secret vars (e.g. GH_DEBUG: "1" in mcp-scripts) are never excluded.
//
// Parameters:
//   - workflowData: the workflow being compiled
//   - coreSecretVarNames: engine-specific fixed secret env var names (e.g. ["COPILOT_GITHUB_TOKEN"])
//
// The function augments coreSecretVarNames with:
//   - MCP_GATEWAY_API_KEY when MCP servers are present
//   - GITHUB_MCP_SERVER_TOKEN when the GitHub tool is present
//   - HTTP MCP header secret var names (values always contain ${{ secrets.* }})
//   - mcp-scripts env var names whose values contain ${{ secrets.* }}
//   - engine.env var names whose values contain ${{ secrets.* }}
//   - agent.env var names whose values contain ${{ secrets.* }}
func ComputeAWFExcludeEnvVarNames(workflowData *WorkflowData, coreSecretVarNames []string) []string {
	seen := make(map[string]bool)
	var names []string

	addUnique := func(name string) {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	// Core secret vars for this engine (always contain secret references).
	for _, name := range coreSecretVarNames {
		addUnique(name)
	}

	// MCP gateway API key is always a secret when MCP servers are present.
	if HasMCPServers(workflowData) {
		addUnique("MCP_GATEWAY_API_KEY")
	}

	// GitHub MCP server token is always a secret when the GitHub tool is present.
	if hasGitHubTool(workflowData.ParsedTools) {
		addUnique("GITHUB_MCP_SERVER_TOKEN")
	}

	// HTTP MCP header secrets: values are always ${{ secrets.* }} references.
	for varName := range collectHTTPMCPHeaderSecrets(workflowData.Tools) {
		addUnique(varName)
	}

	// mcp-scripts env vars: only add those whose configured values contain a secret reference.
	// (Non-secret vars like GH_DEBUG: "1" must NOT be excluded.)
	if workflowData.MCPScripts != nil {
		for _, toolConfig := range workflowData.MCPScripts.Tools {
			for envName, envValue := range toolConfig.Env {
				if strings.Contains(envValue, "${{ secrets.") {
					addUnique(envName)
				}
			}
		}
	}

	// engine.env vars that contain a secret reference.
	if workflowData.EngineConfig != nil {
		for varName, varValue := range workflowData.EngineConfig.Env {
			if strings.Contains(varValue, "${{ secrets.") {
				addUnique(varName)
			}
		}
	}

	// agent.env vars that contain a secret reference.
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil {
		for varName, varValue := range agentConfig.Env {
			if strings.Contains(varValue, "${{ secrets.") {
				addUnique(varName)
			}
		}
	}

	// GH_TOKEN when GitHub mode is gh-proxy: the token is passed in the AWF step env for the
	// host difc-proxy but must be excluded from the agent container.
	if isGitHubCLIModeEnabled(workflowData) {
		addUnique("GH_TOKEN")
	}

	awfHelpersLog.Printf("Computed %d AWF env vars to exclude", len(names))
	return names
}

// addCliProxyGHTokenToEnv adds GH_TOKEN to the AWF step environment when GitHub
// mode is gh-proxy. The token is NOT used by AWF or its cli-proxy
// sidecar directly — the host difc-proxy (started by start_cli_proxy.sh) already
// has it. However, --env-all passes all step env vars into the agent container,
// so we explicitly set GH_TOKEN here to ensure --exclude-env GH_TOKEN can
// reliably strip it regardless of how the token enters the environment.
// The token is excluded from the agent container via --exclude-env GH_TOKEN, so only
// inject it when the effective AWF version supports both cli-proxy flags and
// --exclude-env.
//
// #nosec G101 -- This is NOT a hardcoded credential. It is a GitHub Actions expression
// template that is resolved at runtime by the GitHub Actions runner.
func addCliProxyGHTokenToEnv(env map[string]string, workflowData *WorkflowData) {
	firewallConfig := getFirewallConfig(workflowData)
	if isGitHubCLIModeEnabled(workflowData) &&
		isFirewallEnabled(workflowData) &&
		awfSupportsCliProxy(firewallConfig) &&
		awfSupportsExcludeEnv(firewallConfig) {
		env["GH_TOKEN"] = "${{ secrets.GH_AW_GITHUB_TOKEN || github.token }}"
		awfHelpersLog.Print("Added GH_TOKEN to env for CLI proxy (excluded from agent container)")
	}
}

// awfSupportsExcludeEnv returns true when the effective AWF version supports --exclude-env
// (introduced in AWF v0.25.3).
func awfSupportsExcludeEnv(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFExcludeEnvMinVersion)
}

// awfVersionAtLeast returns true when the effective AWF version is at or above minVersion.
//
// If firewallConfig has no version set, DefaultFirewallVersion is used. "latest" always
// returns true. Non-semver strings (e.g. branch names) return false (conservative).
func awfVersionAtLeast(firewallConfig *FirewallConfig, minVersion constants.Version) bool {
	var versionStr string
	if firewallConfig != nil && firewallConfig.Version != "" {
		versionStr = firewallConfig.Version
	}
	return versionAtLeast(versionStr, string(constants.DefaultFirewallVersion), string(minVersion))
}

// awfSupportsCliProxy returns true when the effective AWF version supports --difc-proxy-host
// and --difc-proxy-ca-cert (introduced in AWF v0.26.0).
func awfSupportsCliProxy(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFCliProxyMinVersion)
}

// awfSupportsAllowHostPorts returns true when the effective AWF version supports
// --allow-host-ports.
func awfSupportsAllowHostPorts(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFAllowHostPortsMinVersion)
}

// awfSupportsDockerHostPathPrefix returns true when the effective AWF version supports
// --docker-host-path-prefix.
func awfSupportsDockerHostPathPrefix(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFDockerHostPathPrefixMinVersion)
}

// awfSupportsTokenSteering returns true when the effective AWF version supports
// apiProxy.enableTokenSteering.
func awfSupportsTokenSteering(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFTokenSteeringMinVersion)
}
