package actions

import (
	"fmt"
	"regexp"
)

type JobID string
type StepID string
type OutputName string

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

func validateIdentifier(kind, value string) error {
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("invalid %s %q: use a letter or underscore followed by letters, digits, underscores, or hyphens", kind, value)
	}
	return nil
}
