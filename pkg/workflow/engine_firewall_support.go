package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// hasNetworkRestrictions checks if the workflow has network restrictions defined
// Network restrictions exist if:
// - network.allowed has domains specified (non-empty list)
func hasNetworkRestrictions(networkPermissions *NetworkPermissions) bool {
	if networkPermissions == nil {
		return false
	}

	// If allowed domains are specified, we have restrictions
	if len(networkPermissions.Allowed) > 0 {
		return true
	}

	// Empty network object {} means deny-all, which is also a restriction
	// But mode "defaults" is not a restriction
	if networkPermissions.Mode == "" && len(networkPermissions.Allowed) == 0 {
		// Empty object {} - this is a restriction (deny-all)
		return true
	}

	return false
}

// checkNetworkSupport validates that the selected engine supports network restrictions
// when network restrictions are defined in the workflow
func (c *Compiler) checkNetworkSupport(engine CodingAgentEngine, networkPermissions *NetworkPermissions) error {
	// First, check for explicit firewall disable
	if err := c.checkFirewallDisable(engine, networkPermissions); err != nil {
		return err
	}

	// Check for wildcards with AWF (Copilot engine)
	if err := c.checkAWFWildcardSupport(engine, networkPermissions); err != nil {
		return err
	}

	// Check if network restrictions exist
	if !hasNetworkRestrictions(networkPermissions) {
		// No restrictions, no validation needed
		return nil
	}

	// Check if engine supports firewall
	if engine.SupportsFirewall() {
		// Engine supports firewall, no issue
		return nil
	}

	// Engine does not support firewall, but network restrictions are present
	message := fmt.Sprintf(
		"Selected engine '%s' does not support network firewalling; workflow specifies network restrictions (network.allowed). Network may not be sandboxed.",
		engine.GetID(),
	)

	if c.strictMode {
		// In strict mode, this is an error
		return fmt.Errorf("strict mode: engine must support firewall when network restrictions (network.allowed) are set")
	}

	// In non-strict mode, emit a warning
	fmt.Println(console.FormatWarningMessage(message))
	c.IncrementWarningCount()

	return nil
}

// checkAWFWildcardSupport validates that wildcard patterns are not used with Copilot engine
// AWF (Agent Workflow Firewall) does not support wildcard syntax like *.example.com
// Instead, AWF automatically matches subdomains when a base domain is specified
func (c *Compiler) checkAWFWildcardSupport(engine CodingAgentEngine, networkPermissions *NetworkPermissions) error {
	// Only check for Copilot engine with firewall enabled
	if engine.GetID() != "copilot" {
		return nil
	}

	// Check if firewall is enabled
	if networkPermissions == nil || networkPermissions.Firewall == nil || !networkPermissions.Firewall.Enabled {
		return nil
	}

	// Check for wildcards in allowed domains
	var wildcardDomains []string
	if networkPermissions != nil && len(networkPermissions.Allowed) > 0 {
		for _, domain := range networkPermissions.Allowed {
			if strings.HasPrefix(domain, "*.") {
				wildcardDomains = append(wildcardDomains, domain)
			}
		}
	}

	if len(wildcardDomains) == 0 {
		return nil
	}

	// Wildcards detected with Copilot/AWF
	message := fmt.Sprintf(
		"AWF does not support wildcard syntax (found: %s). AWF automatically matches subdomains - use base domain instead (e.g., 'example.com' matches 'api.example.com'). See https://github.com/githubnext/gh-aw-firewall/blob/main/docs/QUICKSTART.md#limitations",
		strings.Join(wildcardDomains, ", "),
	)

	if c.strictMode {
		// In strict mode, this is an error
		return fmt.Errorf("strict mode: %s", message)
	}

	// In non-strict mode, emit a warning
	fmt.Println(console.FormatWarningMessage(message))
	c.IncrementWarningCount()

	return nil
}

// checkFirewallDisable validates firewall: "disable" configuration
// - Warning if allowed != * (unrestricted)
// - Error in strict mode if allowed is not * or engine does not support firewall
func (c *Compiler) checkFirewallDisable(engine CodingAgentEngine, networkPermissions *NetworkPermissions) error {
	if networkPermissions == nil || networkPermissions.Firewall == nil {
		return nil
	}

	// Check if firewall is explicitly disabled
	if !networkPermissions.Firewall.Enabled {
		// Check if network has restrictions (allowed list specified with domains)
		hasRestrictions := len(networkPermissions.Allowed) > 0

		if hasRestrictions {
			message := "Firewall is disabled but network restrictions are specified (network.allowed). Network may not be properly sandboxed."

			if c.strictMode {
				// In strict mode, this is an error
				return fmt.Errorf("strict mode: cannot disable firewall when network restrictions (network.allowed) are set")
			}

			// In non-strict mode, emit a warning
			fmt.Println(console.FormatWarningMessage(message))
			c.IncrementWarningCount()
		}

		// Also check if engine doesn't support firewall in strict mode when there are no restrictions
		if c.strictMode && !hasRestrictions && !engine.SupportsFirewall() {
			return fmt.Errorf("strict mode: engine '%s' does not support firewall", engine.GetID())
		}
	}

	return nil
}
