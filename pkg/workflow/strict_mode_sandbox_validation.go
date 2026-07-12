// This file contains strict mode sandbox customization validation.
//
// It enforces that internal-only sandbox fields (AWF agent customization and
// MCP gateway customization) cannot be configured when strict mode is enabled.

package workflow

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
)

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

// validateStrictSandboxCustomization refuses internal sandbox customization fields in strict mode
// and warns about deprecated sandbox.agent.sudo: true in non-strict mode.
//
// The following fields are considered internal implementation/debugging details and
// are not allowed in strict mode:
//   - sandbox.agent.command, sandbox.agent.args, sandbox.agent.env  (AWF customization)
//   - sandbox.mcp.container, sandbox.mcp.version, sandbox.mcp.entrypoint,
//     sandbox.mcp.args, sandbox.mcp.entrypointArgs  (MCP gateway customization)
//
// Additionally, sandbox.agent.sudo: true is an error in strict mode and a warning in
// non-strict mode because the global default has changed to sudo: false (network isolation).
//
// A sandbox.agent object without an explicit 'id' is explicitly set to AWF in strict mode.
func (c *Compiler) validateStrictSandboxCustomization(sandboxConfig *SandboxConfig) error {
	if sandboxConfig == nil {
		return nil
	}

	// Check agent sandbox fields
	if agent := sandboxConfig.Agent; agent != nil {
		// sandbox.agent.sudo: true is deprecated regardless of strict mode.
		// It is an error in strict mode and a warning otherwise.
		// Exception: docker-sbx fundamentally requires sudo for its install step, so
		// the deprecation message is suppressed — sudo: true is mandatory for that runtime.
		if agent.SudoExplicitlyEnabled && agent.Runtime != AgentRuntimeDockerSbx {
			const sudoTrueMsg = "sandbox.agent.sudo: true re-enables host-access (sudo) mode. " +
				"The default is now sudo: false (network isolation). " +
				"Remove 'sudo: true' to use the secure default. " +
				"See: https://github.github.com/gh-aw/reference/sandbox/"
			if c.strictMode {
				return fmt.Errorf("strict mode: %s", sudoTrueMsg)
			}
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(sudoTrueMsg))
			c.IncrementWarningCount()
		}
	}

	if !c.strictMode {
		strictModeValidationLog.Printf("Strict mode disabled, skipping sandbox customization validation")
		return nil
	}

	if agent := sandboxConfig.Agent; agent != nil {
		// In strict mode, if sandbox.agent has no id/type set, explicitly default it to AWF
		// so the sandbox configuration is always unambiguous.
		if !agent.Disabled && !isSupportedSandboxType(getAgentType(agent)) {
			strictModeValidationLog.Printf("sandbox.agent has no id/type in strict mode, defaulting to awf")
			agent.Type = SandboxTypeAWF
		}

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
