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
	"regexp"
	"strings"
)

var safeOutputsStepsShellExpansionLog = newValidationLogger("safe_outputs_steps_shell_expansion")

// shellExpansionPattern matches dangerous bash expansion constructs inside a run: script.
//
// Captured groups by name:
//   - "paramTransform":  ${var@operator} — bash 4.4+ parameter transformation
//   - "indirectExpand":  ${!var}         — bash indirect expansion
//   - "commandSubst":    $(...)          — command substitution (any $( sequence)
//   - "backtick":        `...`           — backtick command substitution
//
// After matching, callers must exclude false-positives:
//   - "commandSubst" matches starting with "$((" (arithmetic expansion) are ignored.
//   - GitHub Actions expressions use "${{ ... }}", which starts with "{" not "(",
//     so they do not match the "commandSubst" pattern (\$\() at all.
//
// Note: Go's regexp/RE2 does not support lookaheads, so post-match filtering is used
// instead of inline negative lookaheads.
var shellExpansionPattern = regexp.MustCompile(
	// Parameter transformation: ${var@op} — the @ must follow word characters
	`(?P<paramTransform>\$\{[A-Za-z_][A-Za-z0-9_]*@[^}]*\})` +
		`|` +
		// Indirect expansion: ${!var} — '!' immediately after '{'
		`(?P<indirectExpand>\$\{![A-Za-z_])` +
		`|` +
		// Command substitution: any $( sequence
		`(?P<commandSubst>\$\()` +
		`|` +
		// Backtick command substitution: `...`
		"(?P<backtick>`[^`\n]+`)",
)

// dangerousPatternDescription maps a named capture group to a human-readable
// description used in compiler error messages.
var dangerousPatternDescription = map[string]string{
	"paramTransform": "parameter transformation (e.g. ${var@P})",
	"indirectExpand": "indirect expansion (e.g. ${!var})",
	"commandSubst":   "command substitution (e.g. $(command))",
	"backtick":       "backtick command substitution (e.g. `command`)",
}

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
	// Fast path: no '$' or backtick character means no expansion pattern can be present.
	if !strings.ContainsAny(script, "$`") {
		return nil
	}

	// Scan all matches so we can skip false positives (e.g. arithmetic $((, GitHub
	// Actions ${{ expressions) before reporting the first true violation.
	groupNames := shellExpansionPattern.SubexpNames()
	allMatches := shellExpansionPattern.FindAllStringSubmatchIndex(script, -1)

	for _, match := range allMatches {
		snippet := ""
		patternDescription := "dangerous shell expansion"
		groupName := ""
		matchStart := match[0]

		for gi, name := range groupNames {
			if name == "" {
				continue
			}
			start, end := match[gi*2], match[gi*2+1]
			if start < 0 {
				continue // group did not participate in this match
			}
			if _, known := dangerousPatternDescription[name]; known {
				raw := script[start:end]
				groupName = name
				// For short patterns like $( or ${! that don't capture the full construct,
				// extend the snippet to include context up to the end of the line or 60 chars.
				if len(raw) < 10 {
					lineEnd := strings.IndexByte(script[start:], '\n')
					if lineEnd < 0 {
						lineEnd = len(script) - start
					}
					raw = script[start : start+lineEnd]
					// Remove any trailing control characters.
					raw = strings.TrimRight(raw, "\r\t ")
				}
				// Clip the snippet to 60 characters to keep the error readable.
				if len(raw) > 60 {
					raw = raw[:57] + "..."
				}
				snippet = raw
				patternDescription = dangerousPatternDescription[name]
				break
			}
		}

		if groupName == "" {
			continue
		}

		// Skip arithmetic expansion $((: it uses $(( not $(command).
		if groupName == "commandSubst" {
			// Check the two characters after $( to detect $((
			if matchStart+3 <= len(script) && script[matchStart:matchStart+3] == "$((" {
				continue
			}
		}

		safeOutputsStepsShellExpansionLog.Printf("Dangerous pattern found in safe-outputs step %d: %s", stepIndex, patternDescription)

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
			patternDescription,
			snippet,
		)
	}

	return nil
}
