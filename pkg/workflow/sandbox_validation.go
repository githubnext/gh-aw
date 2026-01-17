// Package workflow provides sandbox validation functions for agentic workflow compilation.
//
// This file contains domain-specific validation functions for sandbox configuration:
//   - validateMountsSyntax() - Validates container mount syntax
//   - validateSandboxConfig() - Validates complete sandbox configuration
//
// These validation functions are organized in a dedicated file following the validation
// architecture pattern where domain-specific validation belongs in domain validation files.
// See validation.go for the complete validation architecture documentation.
package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var sandboxValidationLog = logger.New("workflow:sandbox_validation")

// validateMountsSyntax validates that mount strings follow the correct syntax
// Expected format: "source:destination:mode" where mode is either "ro" or "rw"
func validateMountsSyntax(mounts []string) error {
	for i, mount := range mounts {
		// Split the mount string by colons
		parts := strings.Split(mount, ":")

		// Must have exactly 3 parts: source, destination, mode
		if len(parts) != 3 {
			return fmt.Errorf("💡 The mount syntax at index %d needs adjustment: '%s'\n\nExpected format: 'source:destination:mode'\n\nWhere:\n  • source: Path on host machine\n  • destination: Path in container\n  • mode: Either 'ro' (read-only) or 'rw' (read-write)\n\nExample:\n  /host/path:/container/path:ro", i, mount)
		}

		source := parts[0]
		dest := parts[1]
		mode := parts[2]

		// Validate that source and destination are not empty
		if source == "" {
			return fmt.Errorf("💡 Mount at index %d is missing the source path in '%s'.\n\nThe source path tells us what directory to mount from the host.\n\nFormat: 'source:destination:mode'\nExample: /host/data:/container/data:ro", i, mount)
		}
		if dest == "" {
			return fmt.Errorf("💡 Mount at index %d is missing the destination path in '%s'.\n\nThe destination path tells us where to mount it in the container.\n\nFormat: 'source:destination:mode'\nExample: /host/data:/container/data:ro", i, mount)
		}

		// Validate mode is either "ro" or "rw"
		if mode != "ro" && mode != "rw" {
			return fmt.Errorf("💡 Mount at index %d has an invalid mode '%s' in '%s'.\n\nMode must be either:\n  • 'ro' - read-only (recommended for security)\n  • 'rw' - read-write (use with caution)\n\nExample: /host/data:/container/data:ro", i, mode, mount)
		}

		sandboxValidationLog.Printf("Validated mount %d: source=%s, dest=%s, mode=%s", i, source, dest, mode)
	}

	return nil
}

// validateSandboxConfig validates the sandbox configuration
// Returns an error if the configuration is invalid
func validateSandboxConfig(workflowData *WorkflowData) error {
	if workflowData == nil {
		return nil
	}

	if workflowData.SandboxConfig == nil {
		return nil // No sandbox config is valid
	}

	sandboxConfig := workflowData.SandboxConfig

	// Check if sandbox: false or sandbox.agent: false was specified
	// In non-strict mode, this is allowed (with a warning shown at compile time)
	// The strict mode check happens in validateStrictFirewall()
	if sandboxConfig.Agent != nil && sandboxConfig.Agent.Disabled {
		// sandbox: false is allowed in non-strict mode, so we don't error here
		// The warning is emitted in compiler.go
		sandboxValidationLog.Print("sandbox: false detected, will be validated by strict mode check")
	}

	// Validate mounts syntax if specified in agent config
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Mounts) > 0 {
		if err := validateMountsSyntax(agentConfig.Mounts); err != nil {
			return err
		}
	}

	// Validate that SRT is only used with Copilot engine
	if isSRTEnabled(workflowData) {
		// Check if the sandbox-runtime feature flag is enabled
		if !isFeatureEnabled(constants.SandboxRuntimeFeatureFlag, workflowData) {
			return fmt.Errorf("🏗️  Sandbox-runtime is experimental and needs a feature flag.\n\nWhy? This feature is still in active development.\n\nTo enable:\n  features:\n    sandbox-runtime: true\n\nOr set environment variable:\n  GH_AW_FEATURES=sandbox-runtime")
		}

		if workflowData.EngineConfig == nil || workflowData.EngineConfig.ID != "copilot" {
			engineID := "none"
			if workflowData.EngineConfig != nil {
				engineID = workflowData.EngineConfig.ID
			}
			return fmt.Errorf("⚠️  Sandbox-runtime requires the Copilot engine (currently using: %s).\n\nWhy? This feature is specifically designed for GitHub Copilot.\n\nTo fix:\n  engine: copilot\n  sandbox: sandbox-runtime", engineID)
		}

		// Check for mutual exclusivity with AWF
		if workflowData.NetworkPermissions != nil && workflowData.NetworkPermissions.Firewall != nil && workflowData.NetworkPermissions.Firewall.Enabled {
			return fmt.Errorf("⚠️  Both sandbox-runtime and AWF firewall are enabled.\n\nWhy this matters: These two security features can't be used together - choose one approach.\n\nOptions:\n  1. Use sandbox-runtime:\n     sandbox: sandbox-runtime\n\n  2. Use AWF firewall:\n     network:\n       firewall:\n         enabled: true\n\nBoth provide network security, but use different approaches")
		}
	}

	// Validate config structure if provided
	if sandboxConfig.Config != nil {
		if sandboxConfig.Type != SandboxTypeRuntime {
			return fmt.Errorf("💡 Custom sandbox configuration detected.\n\nCustom sandbox configs are only supported when type is 'sandbox-runtime'.\n\nExample:\n  sandbox:\n    type: sandbox-runtime\n    config:\n      # your custom config here")
		}
	}

	// Validate that if agent sandbox is enabled, MCP gateway must be enabled
	// The MCP gateway is enabled when MCP servers are configured (tools that use MCP)
	// Only validate this when sandbox is explicitly configured (not nil)
	// If SandboxConfig is nil, defaults will be applied later and MCP check doesn't apply yet
	if !isSandboxDisabled(workflowData) {
		// Sandbox is enabled - check if MCP gateway is enabled
		// Only enforce this if sandbox was explicitly configured (has agent or type set)
		// This prevents false positives for workflows where sandbox defaults haven't been applied yet
		hasExplicitSandboxConfig := (sandboxConfig.Agent != nil && !sandboxConfig.Agent.Disabled) ||
			sandboxConfig.Type != ""

		if hasExplicitSandboxConfig && !HasMCPServers(workflowData) {
			return fmt.Errorf("🏗️  Agent sandbox is enabled, but the MCP gateway isn't configured.\n\nWhy this matters: The agent sandbox requires MCP servers to work properly.\n\nTo fix, either:\n  1. Add MCP tools (recommended):\n     tools:\n       github:\n         mode: remote\n\n  2. Disable the sandbox:\n     sandbox: false\n\nLearn more: https://githubnext.github.io/gh-aw/guides/mcp-servers/")
		}
		if hasExplicitSandboxConfig {
			sandboxValidationLog.Print("Sandbox enabled with MCP gateway - validation passed")
		}
	}

	return nil
}
