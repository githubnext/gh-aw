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
)

var sandboxValidationLog = newValidationLogger("sandbox")

// validateMountsSyntax validates that mount strings follow the correct syntax
// Expected format: "source:destination:mode" where mode is either "ro" or "rw"
func validateMountsSyntax(mounts []string) error {
	for i, mount := range mounts {
		parts, kind := parseMountEntry(mount)
		switch kind {
		case mountValidationOK:
			sandboxValidationLog.Printf("Validated mount %d: source=%s, dest=%s, mode=%s", i, parts.source, parts.dest, parts.mode)
		case mountValidationFormatError:
			return NewValidationError(
				fmt.Sprintf("sandbox.mounts[%d]", i),
				mount,
				"mount syntax must follow 'source:destination:mode' format with exactly 3 colon-separated parts",
				fmt.Sprintf("Use the format 'source:destination:mode'.\n\nExample:\nsandbox:\n  mounts:\n    - \"/host/path:/container/path:ro\"\n\nSee: %s", constants.DocsSandboxURL),
			)
		case mountValidationModeError:
			return NewValidationError(
				fmt.Sprintf("sandbox.mounts[%d].mode", i),
				parts.mode,
				"mount mode must be 'ro' (read-only) or 'rw' (read-write)",
				fmt.Sprintf("Change the mount mode to either 'ro' or 'rw'.\n\nExample:\nsandbox:\n  mounts:\n    - \"/host/path:/container/path:ro\"  # read-only\n    - \"/host/path:/container/path:rw\"  # read-write\n\nSee: %s", constants.DocsSandboxURL),
			)
		case mountValidationEmptySource:
			return NewValidationError(
				fmt.Sprintf("sandbox.mounts[%d].source", i),
				mount,
				"source path cannot be empty",
				fmt.Sprintf("Provide a valid source path.\n\nExample:\nsandbox:\n  mounts:\n    - \"/host/path:/container/path:ro\"\n\nSee: %s", constants.DocsSandboxURL),
			)
		case mountValidationEmptyDestination:
			return NewValidationError(
				fmt.Sprintf("sandbox.mounts[%d].destination", i),
				mount,
				"destination path cannot be empty",
				fmt.Sprintf("Provide a valid destination path.\n\nExample:\nsandbox:\n  mounts:\n    - \"/host/path:/container/path:ro\"\n\nSee: %s", constants.DocsSandboxURL),
			)
		default:
			return fmt.Errorf("internal error: unsupported mount validation kind %d for sandbox mount %q", kind, mount)
		}
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
	// This requires the "dangerously-disable-sandbox-agent" feature flag to be enabled.
	// Without the feature flag, setting sandbox.agent: false is a validation error.
	if sandboxConfig.Agent != nil && sandboxConfig.Agent.Disabled {
		if !isFeatureEnabled(constants.DangerouslyDisableSandboxAgentFeatureFlag, workflowData) {
			flag := string(constants.DangerouslyDisableSandboxAgentFeatureFlag)
			return NewValidationError(
				"sandbox.agent",
				"false",
				fmt.Sprintf("disabling the agent sandbox requires the '%s' feature flag", flag),
				fmt.Sprintf("Add the feature flag to your workflow frontmatter:\n\nfeatures:\n  %s: true\nsandbox:\n  agent: false\n\nSee: %s", flag, constants.DocsSandboxURL),
			)
		}
		sandboxValidationLog.Printf("sandbox.agent: false permitted by %s feature flag", constants.DangerouslyDisableSandboxAgentFeatureFlag)
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

	// Validate that if agent sandbox is enabled, MCP gateway is always enabled.
	// The MCP gateway is enabled when MCP servers are configured (tools that use MCP).
	// Note: Even if agent sandbox is disabled (sandbox.agent: false), the MCP gateway
	// must still be enabled. Agent sandbox and MCP gateway are now independent.
	if sandboxConfig.Agent != nil && !sandboxConfig.Agent.Disabled {
		if !HasMCPServers(workflowData) {
			return NewConfigurationError(
				"sandbox",
				"enabled without MCP servers",
				"agent sandbox requires MCP servers to be configured",
				"Add MCP tools to your workflow:\n\ntools:\n  github:\n    mode: remote\n  playwright: null\n\nOr disable the agent sandbox:\nsandbox:\n  agent: false",
			)
		}
		sandboxValidationLog.Print("Agent sandbox enabled with MCP gateway - validation passed")
	}

	return nil
}
