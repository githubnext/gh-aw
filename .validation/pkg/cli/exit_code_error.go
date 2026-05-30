package cli

import "fmt"

// ExitCodeError is returned by library functions that need to propagate
// a specific process exit code to the cmd/ entry-point.
// The entry-point should check for this error type and call os.Exit with
// the contained Code instead of treating it as an ordinary failure.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("exit with code %d", e.Code)
}
