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

	// Build the expandable args string for args that need shell variable expansion.
	// These MUST be appended as raw (unescaped) strings because single-quoting would
	// prevent the runner's shell from expanding ${GITHUB_WORKSPACE} and ${RUNNER_TEMP}.
	ghAwDir := "${RUNNER_TEMP}/gh-aw"
	expandableArgs := fmt.Sprintf(
		`--container-workdir "${GITHUB_WORKSPACE}" --mount "%s:%s:ro" --mount "%s:/host%s:ro"`,
		ghAwDir, ghAwDir, ghAwDir, ghAwDir,
	)

	// When upload_artifact is configured, add a read-write mount for the staging directory
	// so the model can copy files there from inside the container. The parent ${RUNNER_TEMP}/gh-aw
	// is mounted :ro above; this child mount overrides access for the staging subdirectory only.
	// The staging directory must already exist on the host (created in Write Safe Outputs Config step).
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

	// Build the complete command with proper formatting
	var command string
	if config.PathSetup != "" {
		// Include path setup before AWF command (runs on host before AWF)
		command = fmt.Sprintf(`set -o pipefail
%s
# shellcheck disable=SC1003
%s %s %s \
  -- %s 2>&1 | tee -a %s`,
			config.PathSetup,
			awfCommand,
			expandableArgs,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	} else {
		command = fmt.Sprintf(`set -o pipefail
# shellcheck disable=SC1003
%s %s %s \
  -- %s 2>&1 | tee -a %s`,
			awfCommand,
			expandableArgs,
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

	// Add allowed domains. Pass the raw value so shellEscapeArg (via shellJoinArgs)
	// single-quotes it, which safely handles wildcards like *.domain.com without
	// shell glob expansion and without adding literal double-quote characters.
	awfArgs = append(awfArgs, "--allow-domains", config.AllowedDomains)

	// Add blocked domains if specified
	blockedDomains := formatBlockedDomains(config.WorkflowData.NetworkPermissions)
	if blockedDomains != "" {
		// Same single-quoting rationale as --allow-domains above
		awfArgs = append(awfArgs, "--block-domains", blockedDomains)
		awfHelpersLog.Printf("Added blocked domains: %s", blockedDomains)
	}

	// Set log level
	awfLogLevel := string(constants.AWFDefaultLogLevel)
	if firewallConfig != nil && firewallConfig.LogLevel != "" {
		awfLogLevel = firewallConfig.LogLevel
	}
	awfArgs = append(awfArgs, "--log-level", awfLogLevel)
	awfArgs = append(awfArgs, "--proxy-logs-dir", string(constants.AWFProxyLogsDir))
	awfArgs = append(awfArgs, "--audit-dir", string(constants.AWFAuditDir))

	// Always add --enable-host-access: needed for the API proxy sidecar
	// (to reach host.docker.internal:<port>) and for MCP gateway communication
	awfArgs = append(awfArgs, "--enable-host-access")
	awfHelpersLog.Print("Added --enable-host-access for API proxy and MCP gateway")

	// Pin AWF Docker image version to match the installed binary version
	awfImageTag := getAWFImageTag(firewallConfig)
	awfArgs = append(awfArgs, "--image-tag", awfImageTag)
	awfHelpersLog.Printf("Pinned AWF image tag to %s", awfImageTag)

	// Skip pulling images since they are pre-downloaded
	awfArgs = append(awfArgs, "--skip-pull")
	awfHelpersLog.Print("Using --skip-pull since images are pre-downloaded")

	// Enable API proxy sidecar (always required for LLM gateway)
	awfArgs = append(awfArgs, "--enable-api-proxy")
	awfHelpersLog.Print("Added --enable-api-proxy for LLM API proxying")

	// Enable CLI proxy sidecar when the cli-proxy feature flag is set.
	// This gives the agent secure gh CLI access without exposing GITHUB_TOKEN
	// in the agent container (firewall v0.25.14+).
	if isFeatureEnabled(constants.CliProxyFeatureFlag, config.WorkflowData) {
		if awfSupportsCliProxy(firewallConfig) {
			awfArgs = append(awfArgs, "--enable-cli-proxy")
			awfHelpersLog.Print("Added --enable-cli-proxy for gh CLI proxy sidecar")

			// Generate and pass the guard policy JSON for the cli-proxy.
			// Reuses getDIFCProxyPolicyJSON() to build the static policy from tools.github config
			// (min-integrity and repos fields), matching the DIFC proxy guard policy semantics.
			if config.WorkflowData != nil {
				githubTool := config.WorkflowData.Tools["github"]
				policyJSON := getDIFCProxyPolicyJSON(githubTool)
				if policyJSON != "" {
					awfArgs = append(awfArgs, "--cli-proxy-policy", policyJSON)
					awfHelpersLog.Print("Added --cli-proxy-policy with guard policy")
				}
			}
		} else {
			awfHelpersLog.Printf("Skipping --enable-cli-proxy: AWF version %q is older than minimum %s", getAWFImageTag(firewallConfig), constants.AWFCliProxyMinVersion)
		}
	}

	// Add custom API targets if configured in engine.env
	// This allows AWF's credential isolation and firewall to work with custom endpoints
	// (e.g., corporate LLM routers, Azure OpenAI, self-hosted APIs)
	openaiTarget := extractAPITargetHost(config.WorkflowData, "OPENAI_BASE_URL")
	if openaiTarget != "" {
		awfArgs = append(awfArgs, "--openai-api-target", openaiTarget)
		awfHelpersLog.Printf("Added --openai-api-target=%s", openaiTarget)
	}

	anthropicTarget := extractAPITargetHost(config.WorkflowData, "ANTHROPIC_BASE_URL")
	if anthropicTarget != "" {
		awfArgs = append(awfArgs, "--anthropic-api-target", anthropicTarget)
		awfHelpersLog.Printf("Added --anthropic-api-target=%s", anthropicTarget)
	}

	// Pass base path if URL contains a path component
	// This is required for endpoints with path prefixes (e.g., Databricks /serving-endpoints,
	// Azure OpenAI /openai/deployments/<name>, corporate LLM routers with path-based routing)
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

	// Add Copilot API target for custom Copilot endpoints (GHEC, GHES, or custom).
	// Resolved from engine.api-target (explicit) or GITHUB_COPILOT_BASE_URL in engine.env (implicit).
	if copilotTarget := GetCopilotAPITarget(config.WorkflowData); copilotTarget != "" {
		awfArgs = append(awfArgs, "--copilot-api-target", copilotTarget)
		awfHelpersLog.Printf("Added --copilot-api-target=%s", copilotTarget)
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

// extractAPITargetHost extracts the hostname from a custom API base URL in engine.env.
// This supports custom OpenAI-compatible or Anthropic-compatible endpoints (e.g., internal
// LLM routers, Azure OpenAI) while preserving AWF's credential isolation and firewall features.
//
// The function:
// 1. Checks if the specified env var (e.g., "OPENAI_BASE_URL") exists in engine.env
// 2. Extracts the hostname from the URL (e.g., "https://llm-router.internal.example.com/v1" → "llm-router.internal.example.com")
// 3. Returns empty string if no custom URL is configured or if the URL is invalid
//
// Parameters:
//   - workflowData: The workflow data containing engine configuration
//   - envVar: The environment variable name (e.g., "OPENAI_BASE_URL", "ANTHROPIC_BASE_URL")
//
// Returns:
//   - string: The hostname to use as --openai-api-target or --anthropic-api-target, or empty string if not configured
//
// Example:
//
//	engine:
//	  id: codex
//	  env:
//	    OPENAI_BASE_URL: "https://llm-router.internal.example.com/v1"
//	    OPENAI_API_KEY: ${{ secrets.LLM_ROUTER_KEY }}
//
//	extractAPITargetHost(workflowData, "OPENAI_BASE_URL")
//	// Returns: "llm-router.internal.example.com"
func extractAPITargetHost(workflowData *WorkflowData, envVar string) string {
	// Check if engine config and env are available
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Env == nil {
		return ""
	}

	// Get the custom base URL from engine.env
	baseURL, exists := workflowData.EngineConfig.Env[envVar]
	if !exists || baseURL == "" {
		return ""
	}

	// Extract hostname from URL
	// URLs can be:
	// - "https://llm-router.internal.example.com/v1" → "llm-router.internal.example.com"
	// - "http://localhost:8080/v1" → "localhost:8080"
	// - "api.openai.com" → "api.openai.com" (treated as hostname)

	// Remove protocol prefix if present
	host := baseURL
	if idx := strings.Index(host, "://"); idx != -1 {
		host = host[idx+3:]
	}

	// Remove path suffix if present (everything after first /)
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}

	// Validate that we have a non-empty hostname
	if host == "" {
		awfHelpersLog.Printf("Invalid %s URL (no hostname): %s", envVar, baseURL)
		return ""
	}

	awfHelpersLog.Printf("Extracted API target host from %s: %s", envVar, host)
	return host
}

// extractAPIBasePath extracts the path component from a custom API base URL in engine.env.
// Returns the path prefix (e.g., "/serving-endpoints") or empty string if no path is present.
// Root-only paths ("/") and empty paths return empty string.
//
// This is used to pass --openai-api-base-path and --anthropic-api-base-path to AWF when
// the configured base URL contains a path (e.g., Databricks serving endpoints, Azure OpenAI
// deployments, or corporate LLM routers with path-based routing).
func extractAPIBasePath(workflowData *WorkflowData, envVar string) string {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Env == nil {
		return ""
	}

	baseURL, exists := workflowData.EngineConfig.Env[envVar]
	if !exists || baseURL == "" {
		return ""
	}

	// Remove protocol prefix if present
	host := baseURL
	if idx := strings.Index(host, "://"); idx != -1 {
		host = host[idx+3:]
	}

	// Extract path (everything after the first /)
	if idx := strings.Index(host, "/"); idx != -1 {
		path := host[idx:] // e.g., "/serving-endpoints"
		// Strip query string or fragment if present
		if qi := strings.IndexAny(path, "?#"); qi != -1 {
			path = path[:qi]
		}
		// Remove trailing slashes; a root-only path "/" becomes "" and returns empty
		path = strings.TrimRight(path, "/")
		if path != "" {
			awfHelpersLog.Printf("Extracted API base path from %s: %s", envVar, path)
			return path
		}
	}

	return ""
}

// GetCopilotAPITarget returns the effective Copilot API target hostname, checking in order:
//  1. engine.api-target (explicit, takes precedence)
//  2. GITHUB_COPILOT_BASE_URL in engine.env (implicit, derived from the configured Copilot base URL)
//
// This mirrors the pattern used by other engines:
//   - Codex:    OPENAI_BASE_URL     → --openai-api-target
//   - Claude:   ANTHROPIC_BASE_URL  → --anthropic-api-target
//   - Copilot:  GITHUB_COPILOT_BASE_URL → --copilot-api-target (fallback when api-target not set)
//
// Returns empty string if neither source is configured.
func GetCopilotAPITarget(workflowData *WorkflowData) string {
	// Explicit engine.api-target takes precedence.
	if workflowData != nil && workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
		return workflowData.EngineConfig.APITarget
	}

	// Fallback: derive from the well-known GITHUB_COPILOT_BASE_URL env var.
	return extractAPITargetHost(workflowData, "GITHUB_COPILOT_BASE_URL")
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

	awfHelpersLog.Printf("Computed %d AWF env vars to exclude", len(names))
	return names
}

// awfSupportsExcludeEnv returns true when the effective AWF version supports --exclude-env.
//
// The --exclude-env flag was introduced in AWF v0.25.3. Any workflow that pins an explicit
// version older than v0.25.3 must not emit --exclude-env or the run will fail at startup.
//
// Special cases:
//   - No version override (firewallConfig is nil or has no Version): use DefaultFirewallVersion
//     which is always ≥ AWFExcludeEnvMinVersion → returns true.
//   - "latest": always returns true (latest is always a new release).
//   - Any semver string ≥ AWFExcludeEnvMinVersion: returns true.
//   - Any semver string < AWFExcludeEnvMinVersion: returns false.
//   - Non-semver string (e.g. a branch name): returns false (conservative).
func awfSupportsExcludeEnv(firewallConfig *FirewallConfig) bool {
	var versionStr string
	if firewallConfig != nil && firewallConfig.Version != "" {
		versionStr = firewallConfig.Version
	} else {
		// No override → use the default, which is always ≥ the minimum.
		return true
	}

	// "latest" means the newest release — always supports the flag.
	if strings.EqualFold(versionStr, "latest") {
		return true
	}

	// Normalise the v-prefix for compareVersions.
	minVersion := string(constants.AWFExcludeEnvMinVersion)
	return compareVersions(versionStr, minVersion) >= 0
}

// awfSupportsCliProxy returns true when the effective AWF version supports --enable-cli-proxy.
//
// The --enable-cli-proxy flag was introduced in AWF v0.25.14. Any workflow that pins an explicit
// version older than v0.25.14 must not emit --enable-cli-proxy or the run will fail at startup.
//
// Special cases:
//   - No version override (firewallConfig is nil or has no Version): use DefaultFirewallVersion
//     which is always ≥ AWFCliProxyMinVersion → returns true.
//   - "latest": always returns true (latest is always a new release).
//   - Any semver string ≥ AWFCliProxyMinVersion: returns true.
//   - Any semver string < AWFCliProxyMinVersion: returns false.
//   - Non-semver string (e.g. a branch name): returns false (conservative).
func awfSupportsCliProxy(firewallConfig *FirewallConfig) bool {
	var versionStr string
	if firewallConfig != nil && firewallConfig.Version != "" {
		versionStr = firewallConfig.Version
	} else {
		// No override → use the default, which is always ≥ the minimum.
		return true
	}

	// "latest" means the newest release — always supports the flag.
	if strings.EqualFold(versionStr, "latest") {
		return true
	}

	// Normalise the v-prefix for compareVersions.
	minVersion := string(constants.AWFCliProxyMinVersion)
	return compareVersions(versionStr, minVersion) >= 0
}
