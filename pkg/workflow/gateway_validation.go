// Package workflow provides gateway validation functions for agentic workflow compilation.
//
// This file contains domain-specific validation functions for MCP gateway configuration:
//   - validateAndNormalizePort() - Validates and normalizes gateway port values
//
// These validation functions are organized in a dedicated file following the validation
// architecture pattern where domain-specific validation belongs in domain validation files.
// See validation.go for the complete validation architecture documentation.
package workflow

// validateAndNormalizePort validates the port value and returns the normalized port or an error
func validateAndNormalizePort(port int) (int, error) {
	// If port is 0, use the default
	if port == 0 {
		return DefaultMCPGatewayPort, nil
	}

	// Validate port is in valid range (1-65535)
	if err := validateIntRange(port, 1, 65535, "port"); err != nil {
		return 0, err
	}

	return port, nil
}
