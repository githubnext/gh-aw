// This file validates safe-outputs.steps run scripts for dangerous shell expansion
// patterns that would be blocked at runtime by the safe-outputs security harness.
//
// # Shell Expansion Security in Safe-Outputs Steps
//
// The safe-outputs security harness blocks shell scripts that contain dangerous
// bash expansion patterns. This validator detects those patterns at compile time
// so workflow authors receive a clear error during `gh aw compile` rather than
// a confusing runtime failure.
//
// # Blocked Patterns
//
// The following bash constructs are rejected:
//   - ${var@operator}: bash 4.4+ parameter transformation (e.g. ${foo@P}, ${bar@U})
//   - ${!var}: bash indirect expansion
//   - $(command): command substitution
//   - `command`: backtick command substitution
//
// GitHub Actions template expressions (${{ ... }}) are explicitly allowed and are
// excluded from the checks.
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - A new dangerous shell expansion variant must be detected in safe-outputs run scripts
//   - The safe-outputs harness blocks a new class of shell pattern at runtime
//
// For general safe-outputs validation, see safe_outputs_validation.go.

package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow/shellpattern"
)

var safeOutputsStepsShellExpansionLog = logger.New("workflow:safe_outputs_steps_shell_expansion_validation")

// validateSafeOutputsStepsShellExpansion checks every run: script in the
// safe-outputs.steps list for dangerous bash expansion patterns that would be
// blocked by the safe-outputs security harness at runtime.
//
// Returning a non-nil error causes compilation to fail with a descriptive message
// that includes the step index, the offending snippet, and the pattern category.
func validateSafeOutputsStepsShellExpansion(config *SafeOutputsConfig) error {
	if config == nil || len(config.Steps) == 0 {
		return nil
	}

	safeOutputsStepsShellExpansionLog.Printf("Validating %d safe-outputs steps for dangerous shell expansion patterns", len(config.Steps))

	for i, step := range config.Steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		runVal, exists := stepMap["run"]
		if !exists {
			continue
		}
		runScript, ok := runVal.(string)
		if !ok {
			continue
		}

		if err := validateRunScriptForShellExpansion(i, runScript); err != nil {
			return err
		}
	}

	safeOutputsStepsShellExpansionLog.Print("Safe-outputs steps shell expansion validation passed")
	return nil
}

// validateRunScriptForShellExpansion checks a single run: script for dangerous
// bash expansion patterns. stepIndex is 0-based and is included in error messages.
func validateRunScriptForShellExpansion(stepIndex int, script string) error {
	expansion, found := shellpattern.FirstUnsafeExpansion(script)
	if !found {
		return nil
	}
	raw := script[expansion.Start:expansion.End]
	if len(raw) < 10 {
		lineEnd := strings.IndexByte(script[expansion.Start:], '\n')
		if lineEnd < 0 {
			lineEnd = len(script) - expansion.Start
		}
		raw = strings.TrimRight(script[expansion.Start:expansion.Start+lineEnd], "\r\t ")
	}
	snippet := stringutil.Truncate(raw, 60)
	safeOutputsStepsShellExpansionLog.Printf("Dangerous pattern found in safe-outputs step %d: %s", stepIndex, expansion.Description)
	return fmt.Errorf(
		"safe-outputs.steps[%d]: run script contains %s, which is blocked by the "+
			"safe-outputs security harness at runtime\n\n"+
			"  Offending snippet: %s\n\n"+
			"Avoid command substitution, backticks, indirect expansion, and parameter "+
			"transformation in safe-outputs run scripts. Write URL values and other "+
			"dynamic content to files in /tmp/gh-aw/agent/ during the agent turn, then "+
			"read the file contents in the safe-outputs step (e.g. with 'cat' or by "+
			"passing a script argument)",
		stepIndex,
		expansion.Description,
		snippet,
	)
}
