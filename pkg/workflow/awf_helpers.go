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

	// Build AWF arguments
	awfArgs := BuildAWFArgs(config)

	// Wrap engine command in shell (command already includes any internal setup like npm PATH)
	shellWrappedCommand := WrapCommandInShell(config.EngineCommand)

	// Build the complete command with proper formatting
	var command string
	if config.PathSetup != "" {
		// Include path setup before AWF command (runs on host before AWF)
		command = fmt.Sprintf(`set -o pipefail
%s
# shellcheck disable=SC1003
%s %s \
  -- %s 2>&1 | tee -a %s`,
			config.PathSetup,
			awfCommand,
			shellJoinArgs(awfArgs),
			shellWrappedCommand,
			shellEscapeArg(config.LogFile))
	} else {
		command = fmt.Sprintf(`set -o pipefail
# shellcheck disable=SC1003
%s %s \
  -- %s 2>&1 | tee -a %s`,
			awfCommand,
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
//   - []string: List of AWF arguments
func BuildAWFArgs(config AWFCommandConfig) []string {
	awfHelpersLog.Printf("Building AWF args for engine: %s", config.EngineName)

	firewallConfig := getFirewallConfig(config.WorkflowData)
	agentConfig := getAgentConfig(config.WorkflowData)

	var awfArgs []string

	// Add TTY flag if needed (Claude requires this)
	if config.UsesTTY {
		awfArgs = append(awfArgs, "--tty")
	}

	// Pass all environment variables to the container
	awfArgs = append(awfArgs, "--env-all")

	// Set container working directory to match GITHUB_WORKSPACE
	awfArgs = append(awfArgs, "--container-workdir", "\"${GITHUB_WORKSPACE}\"")
	awfHelpersLog.Print("Set container working directory to GITHUB_WORKSPACE")

	// Mount ${RUNNER_TEMP}/gh-aw read-only inside the container.
	// This overrides the broader $HOME:rw mount that AWF sets up in chroot mode,
	// preventing the agent from directly writing to gh-aw infrastructure files
	// (e.g. safeoutputs/outputs.jsonl) and bypassing the safe-outputs MCP server.
	// The MCP server runs on the host (outside AWF) and retains full write access.
	// Two entries are needed: one for the direct path (non-chroot) and one for the
	// /host-prefixed path (chroot mode, where the agent sees the host FS under /host).
	// Pre-wrap in double quotes so shellEscapeArg leaves them alone and ${RUNNER_TEMP}
	// is expanded by the shell at runtime (single quotes would suppress expansion).
	ghAwDir := "${RUNNER_TEMP}/gh-aw"
	awfArgs = append(awfArgs,
		"--mount", "\""+ghAwDir+":"+ghAwDir+":ro\"",
		"--mount", "\""+ghAwDir+":/host"+ghAwDir+":ro\"",
	)
	awfHelpersLog.Print("Mounted ${RUNNER_TEMP}/gh-aw read-only in AWF container")

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

	// Add allowed domains
	// Use double-quoted form (via shellDoubleQuoteArg) so wildcards like *.domain.com are
	// treated as plain arguments rather than shell globs, fixing ShellCheck SC1003, while
	// still escaping $, `, \, and " to prevent unintended shell expansion.
	awfArgs = append(awfArgs, "--allow-domains", shellDoubleQuoteArg(config.AllowedDomains))

	// Add blocked domains if specified
	blockedDomains := formatBlockedDomains(config.WorkflowData.NetworkPermissions)
	if blockedDomains != "" {
		// Same double-quoting rationale as --allow-domains above
		awfArgs = append(awfArgs, "--block-domains", shellDoubleQuoteArg(blockedDomains))
		awfHelpersLog.Printf("Added blocked domains: %s", blockedDomains)
	}

	// Set log level
	awfLogLevel := string(constants.AWFDefaultLogLevel)
	if firewallConfig != nil && firewallConfig.LogLevel != "" {
		awfLogLevel = firewallConfig.LogLevel
	}
	awfArgs = append(awfArgs, "--log-level", awfLogLevel)
	awfArgs = append(awfArgs, "--proxy-logs-dir", string(constants.AWFProxyLogsDir))

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
