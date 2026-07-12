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
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

var sandboxValidationLog = newValidationLogger("sandbox")

const minSandboxDisableJustificationLength = 20

var githubActionsExpressionPattern = regexp.MustCompile(`\$\{\{[\s\S]*\}\}`)

// validateMountsSyntax validates that mount strings follow the correct syntax
// Expected format: "source:destination:mode" where mode is either "ro" or "rw"
func validateMountsSyntax(mounts []string) error {
	return validateMountEntries(mounts, func(i int, parts mountParts) {
		sandboxValidationLog.Printf("Validated mount %d: source=%s, dest=%s, mode=%s", i, parts.source, parts.dest, parts.mode)
	}, func(i int, mount string, parts mountParts, kind mountValidationKind) error {
		switch kind {
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
	})
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
	// This requires the "dangerously-disable-sandbox-agent" feature to include a
	// justification string. Without a valid justification, disabling the sandbox
	// is a validation error.
	if sandboxConfig.Agent != nil && sandboxConfig.Agent.Disabled {
		justification, err := getSandboxDisableJustification(workflowData)
		if err != nil {
			flag := string(constants.DangerouslyDisableSandboxAgentFeatureFlag)
			return NewValidationError(
				"sandbox.agent",
				"false",
				fmt.Sprintf("disabling the agent sandbox removes a trust boundary: '%s' must be a literal justification string (%d+ chars, no expressions): %v", flag, minSandboxDisableJustificationLength, err),
				fmt.Sprintf("Add the feature value to your workflow frontmatter:\n\nfeatures:\n  %s: \"controlled environment with no internet access\"\nsandbox:\n  agent: false\n\nSee: %s", flag, constants.DocsSandboxURL),
			)
		}
		sandboxConfig.Agent.DisableReason = justification
		sandboxValidationLog.Printf("sandbox.agent: false permitted by %s justification: %q", constants.DangerouslyDisableSandboxAgentFeatureFlag, justification)
	}

	// Validate mounts syntax if specified in agent config
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Mounts) > 0 {
		if err := validateMountsSyntax(agentConfig.Mounts); err != nil {
			return err
		}
	}

	// Validate gVisor runtime compatibility
	if agentConfig != nil && agentConfig.Runtime == AgentRuntimeGVisor {
		// gVisor is incompatible with ARC/DinD topology: the runner has no access to the
		// DinD sidecar's daemon config or systemd, so runsc install + systemctl restart
		// cannot succeed.
		if isArcDindTopology(workflowData) {
			return NewValidationError(
				"sandbox.agent.runtime",
				string(AgentRuntimeGVisor),
				"gvisor is incompatible with runner.topology: arc-dind",
				"gVisor requires registering the runsc runtime with Docker via systemctl, which "+
					"is not possible on ARC DinD runners where the Docker daemon runs in a sidecar. "+
					"Remove sandbox.agent.runtime: gvisor or change runner.topology.",
			)
		}

		sandboxValidationLog.Print("gVisor runtime configured -- topology check passed")
	}

	// Validate docker-sbx runtime compatibility
	if agentConfig != nil && agentConfig.Runtime == AgentRuntimeDockerSbx {
		// docker-sbx is incompatible with ARC/DinD topology: sbx requires KVM which is
		// not available on ARC DinD runners that typically lack nested virtualisation.
		if isArcDindTopology(workflowData) {
			return NewValidationError(
				"sandbox.agent.runtime",
				string(AgentRuntimeDockerSbx),
				"docker-sbx is incompatible with runner.topology: arc-dind",
				"docker-sbx requires KVM (nested virtualisation) which is typically unavailable "+
					"on ARC DinD runners. Remove sandbox.agent.runtime: docker-sbx or change runner.topology.",
			)
		}

		// docker-sbx install step requires root access; sudo: true is mandatory.
		if !agentConfig.SudoExplicitlyEnabled {
			return NewValidationError(
				"sandbox.agent.runtime",
				string(AgentRuntimeDockerSbx),
				"docker-sbx requires sandbox.agent.sudo: true",
				"The docker-sbx install step needs root access to install docker-sbx and fix KVM "+
					"device permissions. Add 'sudo: true' to your sandbox.agent configuration:\n\n"+
					"sandbox:\n  agent:\n    id: awf\n    runtime: docker-sbx\n    sudo: true",
			)
		}

		firewallConfig := getFirewallConfig(workflowData)
		var configuredVersion string
		if firewallConfig != nil {
			configuredVersion = firewallConfig.Version
		}
		if !versionAtLeast(configuredVersion, string(constants.DefaultFirewallVersion), string(constants.AWFContainerRuntimeMinVersion)) {
			effectiveVersion := configuredVersion
			if effectiveVersion == "" {
				effectiveVersion = string(constants.DefaultFirewallVersion)
			}
			return NewValidationError(
				"sandbox.agent.runtime",
				string(AgentRuntimeDockerSbx),
				fmt.Sprintf("docker-sbx requires AWF %s or newer", constants.AWFContainerRuntimeMinVersion),
				fmt.Sprintf("docker-sbx emits 'awf --container-runtime sbx', which is only supported in AWF %s+.\n\nThe effective AWF version is %s. Set firewall.version or sandbox.agent.version to %s or newer.", constants.AWFContainerRuntimeMinVersion, effectiveVersion, constants.AWFContainerRuntimeMinVersion),
			)
		}

		sandboxValidationLog.Print("docker-sbx runtime configured -- topology, sudo, and AWF version checks passed")
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

func getSandboxDisableJustification(workflowData *WorkflowData) (string, error) {
	if workflowData == nil || workflowData.Features == nil {
		return "", errors.New("dangerously-disable-sandbox-agent feature is missing")
	}

	flagName := string(constants.DangerouslyDisableSandboxAgentFeatureFlag)
	value, found := getFeatureValueCaseInsensitive(workflowData.Features, flagName)
	if !found {
		return "", errors.New("dangerously-disable-sandbox-agent feature is missing")
	}

	justification, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("feature must be a string, got %T", value)
	}

	trimmed := strings.TrimSpace(justification)
	if len(trimmed) < minSandboxDisableJustificationLength {
		return "", fmt.Errorf("feature must be at least %d characters", minSandboxDisableJustificationLength)
	}

	if githubActionsExpressionPattern.MatchString(trimmed) {
		return "", errors.New("feature cannot use GitHub Actions expressions")
	}

	return trimmed, nil
}

func getFeatureValueCaseInsensitive(features map[string]any, flagName string) (any, bool) {
	if value, exists := features[flagName]; exists {
		return value, true
	}
	for key, value := range features {
		if strings.EqualFold(key, flagName) {
			return value, true
		}
	}
	return nil, false
}
