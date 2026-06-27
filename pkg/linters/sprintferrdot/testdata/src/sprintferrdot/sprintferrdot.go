// Package sprintferrdot is the test fixture for the sprintferrdot analyzer.
package sprintferrdot

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var sentinel = errors.New("sentinel")

// badSprintfS calls fmt.Sprintf with err.Error() and a %s verb — should be flagged.
func badSprintfS(err error) string {
	return fmt.Sprintf("operation failed: %s", err.Error()) // want `redundant \.Error\(\) call: pass the error value directly with %s`
}

// badSprintfV calls fmt.Sprintf with err.Error() and a %v verb — should be flagged.
func badSprintfV(err error) string {
	return fmt.Sprintf("operation failed: %v", err.Error()) // want `redundant \.Error\(\) call: pass the error value directly with %v`
}

// badErrorf calls fmt.Errorf with err.Error() and a %s verb — should be flagged.
func badErrorf(err error) error {
	return fmt.Errorf("wrapped: %s", err.Error()) // want `redundant \.Error\(\) call: pass the error value directly with %s`
}

// badFprintf calls fmt.Fprintf with err.Error() and a %s verb — should be flagged.
func badFprintf(w io.Writer, err error) {
	fmt.Fprintf(w, "error: %s\n", err.Error()) // want `redundant \.Error\(\) call: pass the error value directly with %s`
}

// goodSprintfS passes the error value directly — no diagnostic expected.
func goodSprintfS(err error) string {
	return fmt.Sprintf("operation failed: %s", err)
}

// goodSprintfW wraps with %w — no diagnostic expected.
func goodSprintfW(err error) error {
	return fmt.Errorf("wrapped: %w", err)
}

// goodStandaloneError calls .Error() outside of a format function — no diagnostic expected.
func goodStandaloneError(err error) string {
	return err.Error()
}

// goodMultiVerb has a %d verb for the error position — no diagnostic expected.
func goodMultiVerb(n int, err error) string {
	return fmt.Sprintf("code %d: %T", n, err.Error())
}

// goodNonErrorDot calls .SomeMethod() that is not Error() — no diagnostic expected.
func goodNonErrorDot(f *os.File) string {
	name := f.Name()
	return fmt.Sprintf("file: %s", name)
}

// goodNilErr explicitly avoids the pattern.
func goodNilErr() string {
	return fmt.Sprintf("sentinel: %s", sentinel)
}

// suppressedSprintfS intentionally keeps err.Error() with suppression.
func suppressedSprintfS(err error) string {
	return fmt.Sprintf("operation failed: %s", err.Error()) //nolint:sprintferrdot
}
