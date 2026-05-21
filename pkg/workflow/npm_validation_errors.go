package workflow

import "errors"

// ErrNpmNotAvailable is returned by validateNpxPackages when npm is not installed on the system.
// Callers should treat this as a warning rather than a hard error, since the workflow may still
// compile and run successfully in environments that have npm (e.g., GitHub Actions).
var ErrNpmNotAvailable = errors.New("npm not available")

// isErrNpmNotAvailable reports whether err indicates that npm is not installed on the system.
func isErrNpmNotAvailable(err error) bool {
	return errors.Is(err, ErrNpmNotAvailable)
}
