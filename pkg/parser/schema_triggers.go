package parser

import (
	"fmt"
	"strings"
)

// validateEngineSpecificRules validates engine-specific rules that are not easily expressed in JSON schema
func validateEngineSpecificRules(frontmatter map[string]any) error {
	// Custom validation rules are now handled separately
	// Command trigger conflicts are validated before schema validation
	// This function is kept as a placeholder for potential future validation rules
	_ = frontmatter
	return nil
}

// validateCommandTriggerConflicts checks that command triggers are not used with conflicting events
func validateCommandTriggerConflicts(frontmatter map[string]any) error {
	// Check if 'on' field exists and is a map
	onValue, hasOn := frontmatter["on"]
	if !hasOn {
		return nil
	}

	onMap, isMap := onValue.(map[string]any)
	if !isMap {
		return nil
	}

	// Check if command trigger is present
	commandValue, hasCommand := onMap["command"]
	if !hasCommand || commandValue == nil {
		return nil
	}

	// List of conflicting events - but we'll check if issues/pull_request are label-only
	conflictingEvents := []string{"issues", "issue_comment", "pull_request", "pull_request_review_comment"}

	// Check for conflicts
	var foundConflicts []string
	for _, eventName := range conflictingEvents {
		if eventValue, hasEvent := onMap[eventName]; hasEvent && eventValue != nil {
			// Special case: allow issues/pull_request events if they only have labeled/unlabeled types
			if eventName == "issues" || eventName == "pull_request" {
				if IsLabelOnlyEvent(eventValue) {
					continue // Allow this - it doesn't conflict with command triggers
				}
			}
			foundConflicts = append(foundConflicts, eventName)
		}
	}

	if len(foundConflicts) > 0 {
		if len(foundConflicts) == 1 {
			return fmt.Errorf("command trigger cannot be used with '%s' event in the same workflow. Command triggers are designed to respond to slash commands in comments and should not be combined with event-based triggers for issues or pull requests", foundConflicts[0])
		}
		return fmt.Errorf("command trigger cannot be used with these events in the same workflow: %s. Command triggers are designed to respond to slash commands in comments and should not be combined with event-based triggers for issues or pull requests", strings.Join(foundConflicts, ", "))
	}

	return nil
}

// IsLabelOnlyEvent checks if an event configuration only contains labeled/unlabeled types
// This is exported for use in the compiler to validate command trigger combinations
func IsLabelOnlyEvent(eventValue any) bool {
	// Event can be a map with types field
	eventMap, isMap := eventValue.(map[string]any)
	if !isMap {
		return false
	}

	// Get the types field
	typesValue, hasTypes := eventMap["types"]
	if !hasTypes {
		return false
	}

	// Types should be an array
	typesArray, isArray := typesValue.([]any)
	if !isArray {
		return false
	}

	// Check if all types are labeled or unlabeled
	if len(typesArray) == 0 {
		return false
	}

	for _, typeValue := range typesArray {
		typeStr, isString := typeValue.(string)
		if !isString {
			return false
		}
		if typeStr != "labeled" && typeStr != "unlabeled" {
			return false
		}
	}

	return true
}
