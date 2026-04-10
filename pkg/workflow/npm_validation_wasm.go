//go:build js || wasm

// This file provides WASM/JS no-op stubs for npm validation functions.
// The canonical (non-WASM) implementations live in npm_validation.go.
// If any function signatures change in npm_validation.go, this file must be updated to match.

package workflow

import "errors"

// ErrNpmNotAvailable is returned by validateNpxPackages when npm is not installed on the system.
var ErrNpmNotAvailable = errors.New("npm not available")

// isErrNpmNotAvailable reports whether err indicates that npm is not installed on the system.
func isErrNpmNotAvailable(err error) bool {
	return errors.Is(err, ErrNpmNotAvailable)
}

func (c *Compiler) validateNpxPackages(workflowData *WorkflowData) error {
	return nil
}
