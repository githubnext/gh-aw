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

// isFirewallEnabled checks if AWF firewall is enabled for the workflow
// Firewall is enabled if network.firewall is explicitly set to true or an object
func isFirewallEnabled(workflowData *WorkflowData) bool {
	// Check network.firewall configuration
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

// enableFirewallByDefault enables firewall by default for engines that support it
// when network restrictions are present but no explicit firewall configuration exists
func enableFirewallByDefault(engine CodingAgentEngine, networkPermissions *NetworkPermissions) {
	// Only apply to engines that support firewall
	if engine == nil || !engine.SupportsFirewall() {
		return
	}

	// Check if network permissions exist
	if networkPermissions == nil {
		return
	}

	// Check if firewall is already configured
	if networkPermissions.Firewall != nil {
		firewallLog.Print("Firewall already configured, skipping default enablement")
		return
	}

	// Check if network restrictions are present (allowed domains specified)
	if len(networkPermissions.Allowed) > 0 {
		// Enable firewall by default for engines with network restrictions
		networkPermissions.Firewall = &FirewallConfig{
			Enabled: true,
		}
		firewallLog.Printf("Enabled firewall by default for %s engine with network restrictions", engine.GetID())
	}
}
