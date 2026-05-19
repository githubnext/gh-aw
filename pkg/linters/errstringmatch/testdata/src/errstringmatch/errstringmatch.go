package errstringmatch

import (
	"errors"
	"strings"
)

var errNotFound = errors.New("not found")

// flagged: strings.Contains on err.Error() with a string literal
func checkError(err error) bool {
	return strings.Contains(err.Error(), "not found") // want `avoid strings\.Contains\(err\.Error\(\)`
}

// flagged: same pattern with a different variable name
func checkPermission(e error) bool {
	return strings.Contains(e.Error(), "403") // want `avoid strings\.Contains\(err\.Error\(\)`
}

// not flagged: using errors.Is
func checkErrorSafe(err error) bool {
	return errors.Is(err, errNotFound)
}

// not flagged: strings.Contains on a plain string, not err.Error()
func checkString(s string) bool {
	return strings.Contains(s, "prefix")
}

func checkIgnoredPreviousLine(err error) bool {
	//nolint:errstringmatch // gh CLI behavior is only available as text.
	return strings.Contains(err.Error(), "INSUFFICIENT_SCOPES")
}

func checkIgnoredSameLine(err error) bool {
	return strings.Contains(err.Error(), "already merged") //nolint:errstringmatch // gh CLI merge status is only available as text.
}
