package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var firewallLog = logger.New("workflow:firewall")

// FirewallConfig represents AWF (gh-aw-firewall) configuration for network egress control
type FirewallConfig struct {
	Enabled       bool     `yaml:"enabled,omitempty"`        // Enable/disable AWF (default: true for copilot when network restrictions present)
	Version       string   `yaml:"version,omitempty"`        // AWF version (empty = latest)
	Args          []string `yaml:"args,omitempty"`           // Additional arguments to pass to AWF
	LogLevel      string   `yaml:"log_level,omitempty"`      // AWF log level (default: "info")
	CleanupScript string   `yaml:"cleanup_script,omitempty"` // Cleanup script path (default: "./scripts/ci/cleanup.sh")
}

// isFirewallDisabledBySandboxAgent checks if the firewall is disabled via sandbox.agent: false
func isFirewallDisabledBySandboxAgent(workflowData *WorkflowData) bool {
	return workflowData != nil &&
		workflowData.SandboxConfig != nil &&
		workflowData.SandboxConfig.Agent != nil &&
		workflowData.SandboxConfig.Agent.Disabled
}

// isFirewallEnabled checks if AWF firewall is enabled for the workflow
// Firewall is enabled if network.firewall is explicitly set to true or an object
// Firewall is disabled if sandbox.agent is explicitly set to false
func isFirewallEnabled(workflowData *WorkflowData) bool {
	// Check if sandbox.agent: false (new way to disable firewall)
	if isFirewallDisabledBySandboxAgent(workflowData) {
		firewallLog.Print("Firewall disabled via sandbox.agent: false")
		return false
	}

	// Check network.firewall configuration (deprecated)
	if workflowData != nil && workflowData.NetworkPermissions != nil && workflowData.NetworkPermissions.Firewall != nil {
		enabled := workflowData.NetworkPermissions.Firewall.Enabled
		firewallLog.Printf("Firewall enabled check: %v", enabled)
		return enabled
	}

	firewallLog.Print("Firewall not configured, returning false")
	return false
}

// getFirewallConfig returns the firewall configuration from network permissions
func getFirewallConfig(workflowData *WorkflowData) *FirewallConfig {
	if workflowData == nil {
		return nil
	}

	// Check network.firewall configuration
	if workflowData.NetworkPermissions != nil && workflowData.NetworkPermissions.Firewall != nil {
		config := workflowData.NetworkPermissions.Firewall
		if firewallLog.Enabled() {
			firewallLog.Printf("Retrieved firewall config: enabled=%v, version=%s, logLevel=%s",
				config.Enabled, config.Version, config.LogLevel)
		}
		return config
	}

	return nil
}

// getAgentConfig returns the agent sandbox configuration from sandbox config
func getAgentConfig(workflowData *WorkflowData) *AgentSandboxConfig {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		return nil
	}

	return workflowData.SandboxConfig.Agent
}

// enableFirewallByDefaultForCopilot enables firewall by default for copilot engine
// when network restrictions are present but no explicit firewall configuration exists
// and no SRT sandbox is configured (SRT and AWF are mutually exclusive)
// and sandbox.agent is not explicitly set to false
func enableFirewallByDefaultForCopilot(engineID string, networkPermissions *NetworkPermissions, sandboxConfig *SandboxConfig) {
	// Only apply to copilot engine
	if engineID != "copilot" {
		return
	}

	// Check if network permissions exist
	if networkPermissions == nil {
		return
	}

	// Check if sandbox.agent: false is set (disables firewall)
	// Use a minimal check here since we don't have WorkflowData
	if sandboxConfig != nil && sandboxConfig.Agent != nil && sandboxConfig.Agent.Disabled {
		firewallLog.Print("sandbox.agent: false is set, skipping AWF auto-enablement")
		return
	}

	// Check if SRT is enabled - skip AWF auto-enablement if SRT is configured
	if sandboxConfig != nil && sandboxConfig.Type == SandboxTypeRuntime {
		firewallLog.Print("SRT sandbox is enabled, skipping AWF auto-enablement")
		return
	}

	// Check if firewall is already configured
	if networkPermissions.Firewall != nil {
		firewallLog.Print("Firewall already configured, skipping default enablement")
		return
	}

	// Check if network restrictions are present (allowed domains specified)
	if len(networkPermissions.Allowed) > 0 {
		// Enable firewall by default for copilot engine with network restrictions
		networkPermissions.Firewall = &FirewallConfig{
			Enabled: true,
		}
		firewallLog.Print("Enabled firewall by default for copilot engine with network restrictions")
	}
}
