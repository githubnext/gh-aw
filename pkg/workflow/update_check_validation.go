// Package workflow provides the update-check validation for strict mode.
//
// The update-check flag controls whether the version update check step runs in the
// activation job. Setting update-check: false disables the step, which is useful when
// running in air-gapped environments or when the update check is not desired.
//
// Security policy:
//   - In strict mode: setting update-check: false raises a compilation error.
//   - In non-strict mode: setting update-check: false emits a warning.
//
// See: https://github.github.com/gh-aw/reference/update-check/
package workflow

import (
	"errors"
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
)

var updateCheckValidationLog = newValidationLogger("update_check")

// validateUpdateCheck enforces the policy for the update-check: false flag.
// In strict mode it returns an error; in non-strict mode it emits a warning.
func (c *Compiler) validateUpdateCheck(frontmatter map[string]any) error {
	// Determine whether update-check: false is set
	updateCheckDisabled := false
	if rawVal, ok := frontmatter["update-check"]; ok {
		if boolVal, ok := rawVal.(bool); ok && !boolVal {
			updateCheckDisabled = true
		}
	}

	if !updateCheckDisabled {
		updateCheckValidationLog.Printf("update-check is enabled (default), skipping validation")
		return nil
	}

	updateCheckValidationLog.Printf("update-check: false detected")

	if c.strictMode {
		return errors.New("strict mode: 'update-check: false' is not allowed. The version update check must remain enabled in strict mode to ensure the workflow uses a supported compile-agentic version")
	}

	// Non-strict mode: emit a warning and continue
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
		"'update-check: false' disables the compile-agentic version check. "+
			"The workflow will not verify that it was compiled with a supported version of gh-aw. "+
			"It is strongly recommended to keep update-check enabled.",
	))
	c.IncrementWarningCount()

	return nil
}
