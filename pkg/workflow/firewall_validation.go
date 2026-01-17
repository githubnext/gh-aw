// Package workflow provides firewall validation functions for agentic workflow compilation.
//
// This file contains domain-specific validation functions for firewall configuration:
//   - ValidateLogLevel() - Validates firewall log-level values
//
// These validation functions are organized in a dedicated file following the validation
// architecture pattern where domain-specific validation belongs in domain validation files.
// See validation.go for the complete validation architecture documentation.
package workflow

import "fmt"

// ValidateLogLevel validates that a firewall log-level value is one of the allowed enum values.
// Valid values are: "debug", "info", "warn", "error".
// Empty string is allowed as it defaults to "info" at runtime.
// Returns an error if the log-level is invalid.
func ValidateLogLevel(level string) error {
	// Empty string is allowed (defaults to "info")
	if level == "" {
		return nil
	}

	valid := []string{"debug", "info", "warn", "error"}
	for _, v := range valid {
		if level == v {
			return nil
		}
	}
	return fmt.Errorf("💡 The log-level '%s' isn't recognized.\n\nValid log levels:\n  • debug - Most detailed, useful for troubleshooting\n  • info - Standard logging (default)\n  • warn - Warnings only\n  • error - Errors only\n\nExample:\n  network:\n    firewall:\n      log-level: info", level)
}
