package workflow

import (
	"fmt"

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
