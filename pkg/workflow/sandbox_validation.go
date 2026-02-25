// This file provides sandbox validation functions for agentic workflow compilation.
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

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var sandboxValidationLog = logger.New("workflow:sandbox_validation")

// validateMountsSyntax validates that mount strings follow the correct syntax
// Expected format: "source:destination:mode" where mode is either "ro" or "rw"
func validateMountsSyntax(mounts []string) error {
	for i, mount := range mounts {
		source, dest, mode, err := validateMountStringFormat(mount)
		if err != nil {
			// Distinguish format error (3-parts) from mode error
			if source == "" && dest == "" && mode == "" {
				return NewValidationError(
					fmt.Sprintf("sandbox.mounts[%d]", i),
					mount,
					"mount syntax must follow 'source:destination:mode' format with exactly 3 colon-separated parts",
					fmt.Sprintf("Use the format 'source:destination:mode'.\n\nExample:\nsandbox:\n  mounts:\n    - \"/host/path:/container/path:ro\"\n\nSee: %s", constants.DocsSandboxURL),
				)
			}
			return NewValidationError(
				fmt.Sprintf("sandbox.mounts[%d].mode", i),
				mode,
				"mount mode must be 'ro' (read-only) or 'rw' (read-write)",
				fmt.Sprintf("Change the mount mode to either 'ro' or 'rw'.\n\nExample:\nsandbox:\n  mounts:\n    - \"/host/path:/container/path:ro\"  # read-only\n    - \"/host/path:/container/path:rw\"  # read-write\n\nSee: %s", constants.DocsSandboxURL),
			)
		}

		// Validate that source and destination are not empty
		if source == "" {
			return NewValidationError(
				fmt.Sprintf("sandbox.mounts[%d].source", i),
				mount,
				"source path cannot be empty",
				fmt.Sprintf("Provide a valid source path.\n\nExample:\nsandbox:\n  mounts:\n    - \"/host/path:/container/path:ro\"\n\nSee: %s", constants.DocsSandboxURL),
			)
		}
		if dest == "" {
			return NewValidationError(
				fmt.Sprintf("sandbox.mounts[%d].destination", i),
				mount,
				"destination path cannot be empty",
				fmt.Sprintf("Provide a valid destination path.\n\nExample:\nsandbox:\n  mounts:\n    - \"/host/path:/container/path:ro\"\n\nSee: %s", constants.DocsSandboxURL),
			)
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

	// Check if sandbox.agent: false was specified
	// In non-strict mode, this is allowed (with a warning shown at compile time)
	// The strict mode check happens in validateStrictFirewall()
	if sandboxConfig.Agent != nil && sandboxConfig.Agent.Disabled {
		// sandbox.agent: false is allowed in non-strict mode, so we don't error here
		// The warning is emitted in compiler.go
		sandboxValidationLog.Print("sandbox.agent: false detected, will be validated by strict mode check")
	}

	// Validate mounts syntax if specified in agent config
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Mounts) > 0 {
		if err := validateMountsSyntax(agentConfig.Mounts); err != nil {
			return err
		}
	}

	// Validate config structure if provided (deprecated - was only for SRT)
	if sandboxConfig.Config != nil {
		// Config is no longer used - SRT removed
		return NewConfigurationError(
			"sandbox.config",
			"deprecated",
			"custom sandbox config is deprecated (was only for Sandbox Runtime which has been removed)",
			"Remove sandbox.config from your workflow. AWF (Agent Workflow Firewall) is the only supported sandbox and does not use this configuration.",
		)
	}

	// Validate MCP gateway port if configured
	if sandboxConfig.MCP != nil && sandboxConfig.MCP.Port != 0 {
		if err := validateIntRange(sandboxConfig.MCP.Port, constants.MinNetworkPort, constants.MaxNetworkPort, "sandbox.mcp.port"); err != nil {
			return err
		}
		sandboxValidationLog.Printf("Validated MCP gateway port: %d", sandboxConfig.MCP.Port)
	}

	// Validate that if agent sandbox is enabled, MCP gateway is always enabled
	// The MCP gateway is enabled when MCP servers are configured (tools that use MCP)
	// Only validate this when sandbox is explicitly configured (not nil)
	// If SandboxConfig is nil, defaults will be applied later and MCP check doesn't apply yet
	//
	// Note: Even if agent sandbox is disabled (sandbox.agent: false), the MCP gateway
	// must still be enabled. Agent sandbox and MCP gateway are now independent.
	if sandboxConfig.Agent != nil && !sandboxConfig.Agent.Disabled {
		// Agent sandbox is enabled - check if MCP gateway is enabled
		// Only enforce this if sandbox was explicitly configured (has agent or type set)
		// This prevents false positives for workflows where sandbox defaults haven't been applied yet
		hasExplicitSandboxConfig := (sandboxConfig.Agent != nil && !sandboxConfig.Agent.Disabled) ||
			sandboxConfig.Type != ""

		if hasExplicitSandboxConfig && !HasMCPServers(workflowData) {
			return NewConfigurationError(
				"sandbox",
				"enabled without MCP servers",
				"agent sandbox requires MCP servers to be configured",
				"Add MCP tools to your workflow:\n\ntools:\n  github:\n    mode: remote\n  playwright: null\n\nOr disable the agent sandbox:\nsandbox:\n  agent: false",
			)
		}
		if hasExplicitSandboxConfig {
			sandboxValidationLog.Print("Agent sandbox enabled with MCP gateway - validation passed")
		}
	}

	return nil
}
