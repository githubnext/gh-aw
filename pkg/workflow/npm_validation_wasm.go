//go:build js || wasm

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
