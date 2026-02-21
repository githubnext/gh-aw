// This file provides firewall validation functions for agentic workflow compilation.
//
// This file contains domain-specific validation functions for firewall configuration:
//   - validateFirewallConfig() - Validates the overall firewall configuration
//   - ValidateLogLevel() - Validates firewall log-level values
//
// These validation functions are organized in a dedicated file following the validation
// architecture pattern where domain-specific validation belongs in domain validation files.
// See validation.go for the complete validation architecture documentation.

package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var firewallValidationLog = logger.New("workflow:firewall_validation")

// validateFirewallConfig validates firewall configuration including log-level
func (c *Compiler) validateFirewallConfig(workflowData *WorkflowData) error {
	if workflowData.NetworkPermissions == nil || workflowData.NetworkPermissions.Firewall == nil {
		return nil
	}

	config := workflowData.NetworkPermissions.Firewall
	firewallValidationLog.Printf("Validating firewall config: enabled=%v, logLevel=%s", config.Enabled, config.LogLevel)
	if config.LogLevel != "" {
		if err := ValidateLogLevel(config.LogLevel); err != nil {
			firewallValidationLog.Printf("Invalid firewall log level: %s", config.LogLevel)
			return err
		}
	}

	firewallValidationLog.Print("Firewall config validation passed")
	return nil
}

// ValidateLogLevel validates that a firewall log-level value is one of the allowed enum values.
// Valid values are: "debug", "info", "warn", "error".
// Empty string is allowed as it defaults to "info" at runtime.
// Returns an error if the log-level is invalid.
func ValidateLogLevel(level string) error {
	firewallValidationLog.Printf("Validating firewall log-level: %s", level)

	// Empty string is allowed (defaults to "info")
	if level == "" {
		firewallValidationLog.Print("Empty log-level, using default")
		return nil
	}

	valid := []string{"debug", "info", "warn", "error"}
	for _, v := range valid {
		if level == v {
			firewallValidationLog.Printf("Valid log-level: %s", level)
			return nil
		}
	}
	firewallValidationLog.Printf("Invalid log-level: %s", level)
	return fmt.Errorf("invalid log-level '%s', must be one of: %v", level, valid)
}
