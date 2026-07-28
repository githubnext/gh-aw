// This file provides engine driver and script validation for agentic workflows.
//
// # Engine Driver Validation
//
// This file validates engine.driver and engine.harness configuration fields,
// ensuring that script filenames and driver paths meet safety requirements.
//
// # Validation Functions
//
//   - validateEngineScriptFilename() - Validates that a script filename is a safe Node.js basename
//   - validateEngineHarnessScript() - Validates optional engine.harness configuration
//   - validateEngineDriver() - Validates the shared engine.driver field
//   - validateInlineEngineDriver() - Validates an inline (source-embedded) driver configuration
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates engine.driver or engine.harness field values
//   - It checks script filename or path safety (no traversal, no metacharacters)
//   - It validates file extensions for engine scripts
//   - It validates inline driver source/runtime combinations
//
// For engine version and MCP timeout validation, see engine_validation.go.
// For inline engine definition and auth validation, see engine_inline_definition_validation.go.
// For general validation, see validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

var safeHarnessScriptPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)

// safeSDKDriverSegmentPattern allows path segments that may start with a dot followed by an
// alphanumeric/underscore (e.g. ".github"), but still rejects ".." traversals, leading hyphens,
// and shell metacharacters.
var safeSDKDriverSegmentPattern = regexp.MustCompile(`^(?:\.[A-Za-z0-9_]|[A-Za-z0-9_])[A-Za-z0-9._-]*$`)

// validateEngineScriptFilename validates that a script name is a safe Node.js basename.
// The name must not contain path separators, '..', or shell metacharacters, and must
// end with one of the supported JavaScript extensions: .js, .cjs, or .mjs.
func validateEngineScriptFilename(fieldName, scriptName string) error {
	if strings.TrimSpace(scriptName) != scriptName {
		return fmt.Errorf("%s must be a safe basename without leading/trailing whitespace (found: %s).\n\nSee: %s", fieldName, scriptName, constants.DocsEnginesURL)
	}

	if filepath.IsAbs(scriptName) ||
		strings.Contains(scriptName, "/") ||
		strings.Contains(scriptName, `\`) ||
		strings.Contains(scriptName, "..") ||
		!safeHarnessScriptPattern.MatchString(scriptName) {
		return fmt.Errorf("%s must be a safe basename (no path separators, '..', or shell metacharacters) ending with .js, .cjs, or .mjs (found: %s).\n\nSee: %s", fieldName, scriptName, constants.DocsEnginesURL)
	}

	ext := strings.ToLower(filepath.Ext(scriptName))
	switch ext {
	case ".js", ".cjs", ".mjs":
		return nil
	default:
		return fmt.Errorf("%s must be a Node.js script ending with .js, .cjs, or .mjs (found: %s).\n\nSee: %s", fieldName, scriptName, constants.DocsEnginesURL)
	}
}

// validateEngineHarnessScript validates optional engine.harness configuration.
// engine.harness must point to a Node.js script.
func (c *Compiler) validateEngineHarnessScript(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.HarnessScript == "" {
		return nil
	}

	return validateEngineScriptFilename("engine.harness", workflowData.EngineConfig.HarnessScript)
}

// validateEngineDriver validates the shared engine.driver field.
// engine.driver must be either:
//   - a relative path (with safe segments separated by '/') ending with a supported
//     extension for the engine in use, or
//   - a bare command name without any extension (treated as an arbitrary executable in
//     PATH for the copilot engine, or as a built-in driver for the pi engine).
//
// The allowed extensions depend on the engine:
//   - copilot: .js, .cjs, .mjs (Node.js), .py (Python), .ts/.mts (TypeScript), .rb (Ruby)
//   - all other engines (e.g. pi): .js, .cjs, .mjs only
//
// Absolute paths, backslashes, '..' components, and shell metacharacters are rejected.
func (c *Compiler) validateEngineDriver(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Driver == "" {
		return nil
	}

	if workflowData.EngineConfig.InlineDriver != nil {
		return c.validateInlineEngineDriver(workflowData)
	}

	name := workflowData.EngineConfig.Driver
	isCopilotEngine := workflowData.EngineConfig.ID == "copilot"

	if strings.TrimSpace(name) != name {
		return fmt.Errorf("engine.driver must be a safe path without leading/trailing whitespace (found: %s).\n\nSee: %s", name, constants.DocsEnginesURL)
	}

	if filepath.IsAbs(name) ||
		strings.Contains(name, `\`) ||
		strings.Contains(name, "..") {
		return fmt.Errorf("engine.driver must be a relative path (no absolute paths, '..', or backslashes) with a supported extension (found: %s).\n\nSee: %s", name, constants.DocsEnginesURL)
	}

	// Each path segment must be safe (alphanumeric, underscore, dot, hyphen; may start with dot).
	// Empty segments (consecutive slashes, leading/trailing slashes) are rejected.
	for segment := range strings.SplitSeq(name, "/") {
		if segment == "" {
			return fmt.Errorf("engine.driver must not contain empty path segments (e.g. consecutive '/' or leading/trailing '/') (found: %s).\n\nSee: %s", name, constants.DocsEnginesURL)
		}
		if !safeSDKDriverSegmentPattern.MatchString(segment) {
			return fmt.Errorf("engine.driver must not contain shell metacharacters (found unsafe segment %q in: %s).\n\nSee: %s", segment, name, constants.DocsEnginesURL)
		}
	}

	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".js", ".cjs", ".mjs":
		return nil
	case ".py", ".ts", ".mts", ".rb":
		if isCopilotEngine {
			return nil
		}
		return fmt.Errorf("engine.driver has unsupported extension %q for this engine (found: %s). Must be a JavaScript file ending with .js, .cjs, or .mjs, or a bare name without an extension.\n\nSee: %s", ext, name, constants.DocsEnginesURL)
	case "":
		// No extension — valid as a bare built-in driver name or arbitrary command in PATH.
		return nil
	default:
		if isCopilotEngine {
			return fmt.Errorf("engine.driver has unsupported extension %q (found: %s). Must be a script ending with .js, .cjs, .mjs, .py, .ts, .mts, or .rb, or a bare command name without an extension.\n\nSee: %s", ext, name, constants.DocsEnginesURL)
		}
		return fmt.Errorf("engine.driver has unsupported extension %q (found: %s). Must be a JavaScript file ending with .js, .cjs, or .mjs, or a bare name without an extension.\n\nSee: %s", ext, name, constants.DocsEnginesURL)
	}
}

func (c *Compiler) validateInlineEngineDriver(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.InlineDriver == nil {
		return nil
	}

	inlineDriver := workflowData.EngineConfig.InlineDriver

	if inlineDriver.MultipleRuntime {
		return fmt.Errorf("engine.driver: exactly one runtime key is allowed (node, python, go, java); found multiple.\n\nSee: %s", constants.DocsEnginesURL)
	}

	if workflowData.EngineConfig.ID != "copilot" {
		return fmt.Errorf("inline engine.driver sources are only supported for the copilot engine.\n\nSee: %s", constants.DocsEnginesURL)
	}

	if strings.TrimSpace(inlineDriver.Source) == "" {
		return fmt.Errorf("engine.driver.%s must not be empty.\n\nSee: %s", inlineDriver.Runtime, constants.DocsEnginesURL)
	}

	switch inlineDriver.Runtime {
	case "node", "python", "go", "java":
		return nil
	default:
		return fmt.Errorf("engine.driver inline runtime %q is not supported. Use one of: node, python, go, java.\n\nSee: %s", inlineDriver.Runtime, constants.DocsEnginesURL)
	}
}
