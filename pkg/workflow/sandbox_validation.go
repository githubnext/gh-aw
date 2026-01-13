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
			return fmt.Errorf("invalid mount syntax at index %d: '%s'. Expected format: 'source:destination:mode' (e.g., '/host/path:/container/path:ro')", i, mount)
		}

		source := parts[0]
		dest := parts[1]
		mode := parts[2]

		// Validate that source and destination are not empty
		if source == "" {
			return fmt.Errorf("invalid mount at index %d: source path is empty in '%s'", i, mount)
		}
		if dest == "" {
			return fmt.Errorf("invalid mount at index %d: destination path is empty in '%s'", i, mount)
		}

		// Validate mode is either "ro" or "rw"
		if mode != "ro" && mode != "rw" {
			return fmt.Errorf("invalid mount at index %d: mode must be 'ro' (read-only) or 'rw' (read-write), got '%s' in '%s'", i, mode, mount)
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
			return fmt.Errorf("sandbox-runtime feature is experimental and requires the 'sandbox-runtime' feature flag to be enabled. Set 'features: { sandbox-runtime: true }' in frontmatter or set GH_AW_FEATURES=sandbox-runtime")
		}

		if workflowData.EngineConfig == nil || workflowData.EngineConfig.ID != "copilot" {
			engineID := "none"
			if workflowData.EngineConfig != nil {
				engineID = workflowData.EngineConfig.ID
			}
			return fmt.Errorf("sandbox-runtime is only supported with Copilot engine (current engine: %s)", engineID)
		}

		// Check for mutual exclusivity with AWF
		if workflowData.NetworkPermissions != nil && workflowData.NetworkPermissions.Firewall != nil && workflowData.NetworkPermissions.Firewall.Enabled {
			return fmt.Errorf("sandbox-runtime and AWF firewall cannot be used together; please use either 'sandbox: sandbox-runtime' or 'network.firewall' but not both")
		}
	}

	// Validate config structure if provided
	if sandboxConfig.Config != nil {
		if sandboxConfig.Type != SandboxTypeRuntime {
			return fmt.Errorf("custom sandbox config can only be provided when type is 'sandbox-runtime'")
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
			return fmt.Errorf("agent sandbox is enabled but MCP gateway is not enabled. The agent sandbox requires MCP servers to be configured. Add tools that use MCP (e.g., 'github', 'playwright') or disable the sandbox with 'sandbox: false'")
		}
		if hasExplicitSandboxConfig {
			sandboxValidationLog.Print("Sandbox enabled with MCP gateway - validation passed")
		}
	}

	return nil
}
