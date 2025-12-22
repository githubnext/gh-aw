package cli

import (
	"errors"
	"regexp"
)

// workflowNameRegex validates workflow names contain only alphanumeric characters, hyphens, and underscores
var workflowNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateWorkflowName checks if the provided workflow name is valid.
// It ensures the name is not empty and contains only alphanumeric characters, hyphens, and underscores.
func ValidateWorkflowName(s string) error {
	if s == "" {
		return errors.New("workflow name cannot be empty")
	}
	if !workflowNameRegex.MatchString(s) {
		return errors.New("workflow name must contain only alphanumeric characters, hyphens, and underscores")
	}
	return nil
}
