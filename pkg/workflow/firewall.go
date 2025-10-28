package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
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

// generateAWFInstallationStep creates a GitHub Actions step to install the AWF binary
// This is a shared helper function used by all engines that support firewall
func generateAWFInstallationStep(version string) GitHubActionStep {
	stepLines := []string{
		"      - name: Install awf binary",
		"        run: |",
	}

	// Use default version if not specified to ensure reproducible builds
	if version == "" {
		version = constants.DefaultFirewallVersion
	}

	stepLines = append(stepLines, fmt.Sprintf("          echo \"Installing awf from release: %s\"", version))
	stepLines = append(stepLines, fmt.Sprintf("          curl -L https://github.com/githubnext/gh-aw-firewall/releases/download/%s/awf-linux-x64 -o awf", version))

	stepLines = append(stepLines,
		"          chmod +x awf",
		"          sudo mv awf /usr/local/bin/",
		"          which awf",
		"          awf --version",
	)

	return GitHubActionStep(stepLines)
}

// generateAWFCleanupStep creates a GitHub Actions step to cleanup AWF resources
// This is a shared helper function used by all engines that support firewall
func generateAWFCleanupStep(scriptPath string) GitHubActionStep {
	if scriptPath == "" {
		scriptPath = "./scripts/ci/cleanup.sh"
	}

	stepLines := []string{
		"      - name: Cleanup any existing awf resources",
		fmt.Sprintf("        run: %s || true", scriptPath),
	}

	return GitHubActionStep(stepLines)
}
