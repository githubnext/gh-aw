package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// BuildAWFArgs constructs common AWF arguments from configuration.
// This extracts the shared AWF argument building logic from engine implementations.
//
// The following flags are expressed in the generated JSON config file written by
// BuildAWFCommand and are therefore not emitted here:
//   - --allow-domains / --block-domains   → network.allowDomains / network.blockDomains
//   - --image-tag                         → container.imageTag
//   - --openai-api-target                 → apiProxy.targets.openai.host
//   - --anthropic-api-target              → apiProxy.targets.anthropic.host
//   - --copilot-api-target                → apiProxy.targets.copilot.host
//   - --gemini-api-target                 → apiProxy.targets.gemini.host
//
// Note: --enable-api-proxy is deprecated in AWF v0.27.32+ (API proxy is always on).
// The apiProxy.enabled field is still emitted in the config file for backward compat.
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

	// Add TTY flag if needed (Claude requires this), except for docker-sbx where
	// sbx exec --tty can terminate long-running Claude sessions prematurely.
	if config.UsesTTY && !isDockerSbxRuntime(config.WorkflowData) {
		awfArgs = append(awfArgs, "--tty")
	}

	// docker-sbx: tell AWF to launch the agent inside a Docker sbx microVM instead
	// of as a standard Docker Compose service. Guard on the effective AWF version so
	// older binaries do not receive an unknown flag.
	if isDockerSbxRuntime(config.WorkflowData) && awfSupportsContainerRuntime(firewallConfig) {
		awfArgs = append(awfArgs, "--container-runtime", "sbx")
		awfHelpersLog.Print("Added --container-runtime sbx for docker-sbx microVM runtime")
	} else if isDockerSbxRuntime(config.WorkflowData) {
		awfHelpersLog.Printf("Skipping --container-runtime sbx: AWF version %q is older than required minimum %s", getAWFImageTag(firewallConfig), constants.AWFContainerRuntimeMinVersion)
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
	if isFeatureEnabled(constants.AwfDiagnosticLogsFeatureFlag, config.WorkflowData) {
		awfArgs = append(awfArgs, "--diagnostic-logs")
		awfHelpersLog.Print("Added --diagnostic-logs because awf-diagnostic-logs feature flag is enabled")
	}

	// Legacy security mode: emit --legacy-security, --enable-host-access, and --allow-host-ports
	isLegacy := agentConfig != nil && agentConfig.LegacySecurity
	if isLegacy {
		if awfSupportsLegacySecurity(firewallConfig) {
			awfArgs = append(awfArgs, "--legacy-security")
			awfHelpersLog.Print("Added --legacy-security (legacy-security: enable in frontmatter)")
		} else {
			// AWF versions older than v0.27.32 don't support --legacy-security;
			// they run in legacy mode by default so the flag is unnecessary.
			awfHelpersLog.Printf("Skipping --legacy-security: AWF version %q is older than minimum %s (legacy mode is the default for older versions)", getAWFImageTag(firewallConfig), constants.AWFLegacySecurityMinVersion)
		}

		awfArgs = append(awfArgs, "--enable-host-access")
		awfHelpersLog.Print("Added --enable-host-access for legacy security mode")

		if awfSupportsAllowHostPorts(firewallConfig) {
			mcpGatewayPort := int(DefaultMCPGatewayPort)
			if config.WorkflowData != nil && config.WorkflowData.SandboxConfig != nil &&
				config.WorkflowData.SandboxConfig.MCP != nil && config.WorkflowData.SandboxConfig.MCP.Port > 0 {
				mcpGatewayPort = config.WorkflowData.SandboxConfig.MCP.Port
			}
			hostPorts := fmt.Sprintf("80,443,%d", mcpGatewayPort)
			awfArgs = append(awfArgs, "--allow-host-ports", hostPorts)
			awfHelpersLog.Printf("Added --allow-host-ports %s for legacy security mode", hostPorts)
		}
	} else {
		awfHelpersLog.Print("Strict security: skipping host-access flags (default)")
	}

	// Skip pulling images since they are pre-downloaded
	awfArgs = append(awfArgs, "--skip-pull")
	awfHelpersLog.Print("Using --skip-pull since images are pre-downloaded")

	// Enable CLI proxy sidecar when GitHub mode is gh-proxy.
	// Start the difc-proxy on the host and tell AWF where to connect
	// (firewall v0.25.17+).
	if isGitHubCLIModeEnabled(config.WorkflowData) {
		if awfSupportsCliProxy(firewallConfig) {
			difcProxyHost := "host.docker.internal:18443"
			if isAWFNetworkIsolationEnabled(config.WorkflowData) {
				difcProxyHost = "awmg-cli-proxy:18443"
			}
			awfArgs = append(awfArgs, "--difc-proxy-host", difcProxyHost)
			awfArgs = append(awfArgs, "--difc-proxy-ca-cert", constants.TmpDIFCProxyTLSCACert)
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
//   - string: The AWF command to use (e.g., "sudo -E awf", "awf", or custom command)
func GetAWFCommandPrefix(workflowData *WorkflowData) string {
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && agentConfig.Command != "" {
		awfHelpersLog.Printf("Using custom AWF command: %s", agentConfig.Command)
		return agentConfig.Command
	}

	// Legacy security mode: use sudo for backward compatibility
	if agentConfig != nil && agentConfig.LegacySecurity {
		awfHelpersLog.Print("Using legacy AWF command (legacy-security: enable)")
		return string(constants.AWFLegacySecurityCommand)
	}

	// Default strict security: AWF runs rootless (no sudo)
	awfHelpersLog.Print("Using standard AWF command (strict security, no sudo)")
	return string(constants.AWFDefaultCommand)
}

// WrapCommandInShell wraps an engine command in a shell invocation for AWF execution.
// This is needed because AWF requires commands to be wrapped in shell for proper execution.
//
// set +o histexpand disables bash history expansion so that agent-authored strings
// containing '!' characters (e.g. "!**") cannot be silently misinterpreted or dropped.
// History expansion is meaningless for non-interactive execution and has no other effect.
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

	// Wrap in shell invocation.
	// set +o histexpand is first to prevent bash from expanding !-patterns in any
	// double-quoted strings that appear in the engine command or its arguments.
	return fmt.Sprintf("/bin/bash -c 'set +o histexpand; %s'", escapedCommand)
}
