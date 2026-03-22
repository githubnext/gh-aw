// This file contains strict mode sandbox customization validation.
//
// It enforces that internal-only sandbox fields (AWF agent customization and
// MCP gateway customization) cannot be configured when strict mode is enabled.

package workflow

import "fmt"

// internalSandboxFieldError returns a standardised strict-mode error for an
// internal sandbox field that must not be configured by end users.
func internalSandboxFieldError(fieldPath string) error {
	return fmt.Errorf(
		"strict mode: '%s' is not allowed because it is an internal implementation detail. "+
			"Remove '%s' or set 'strict: false' to disable strict mode. "+
			"See: https://github.github.com/gh-aw/reference/sandbox/",
		fieldPath, fieldPath,
	)
}

// validateStrictSandboxCustomization refuses internal sandbox customization fields in strict mode.
//
// The following fields are considered internal implementation/debugging details and
// are not allowed in strict mode:
//   - sandbox.agent.command, sandbox.agent.args, sandbox.agent.env  (AWF customization)
//   - sandbox.mcp.container, sandbox.mcp.version, sandbox.mcp.entrypoint,
//     sandbox.mcp.args, sandbox.mcp.entrypointArgs  (MCP gateway customization)
func (c *Compiler) validateStrictSandboxCustomization(sandboxConfig *SandboxConfig) error {
	if !c.strictMode {
		strictModeValidationLog.Printf("Strict mode disabled, skipping sandbox customization validation")
		return nil
	}

	if sandboxConfig == nil {
		return nil
	}

	// Check agent sandbox internal fields
	if agent := sandboxConfig.Agent; agent != nil {
		if agent.Command != "" {
			return internalSandboxFieldError("sandbox.agent.command")
		}
		if len(agent.Args) > 0 {
			return internalSandboxFieldError("sandbox.agent.args")
		}
		if len(agent.Env) > 0 {
			return internalSandboxFieldError("sandbox.agent.env")
		}
	}

	// Check MCP gateway internal fields
	if mcp := sandboxConfig.MCP; mcp != nil {
		if mcp.Container != "" {
			return internalSandboxFieldError("sandbox.mcp.container")
		}
		if mcp.Version != "" {
			return internalSandboxFieldError("sandbox.mcp.version")
		}
		if mcp.Entrypoint != "" {
			return internalSandboxFieldError("sandbox.mcp.entrypoint")
		}
		if len(mcp.Args) > 0 {
			return internalSandboxFieldError("sandbox.mcp.args")
		}
		if len(mcp.EntrypointArgs) > 0 {
			return internalSandboxFieldError("sandbox.mcp.entrypointArgs")
		}
	}

	strictModeValidationLog.Printf("Sandbox customization validation passed")
	return nil
}
