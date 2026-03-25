// This file provides validation for GitHub Actions event filter mutual exclusivity
// and glob pattern validity.
//
// # Filter Validation
//
// This file validates that event filters follow GitHub Actions requirements for mutual exclusivity.
// GitHub Actions rejects workflows that specify both:
//   - branches and branches-ignore in the same event
//   - paths and paths-ignore in the same event
//
// # Glob Pattern Validation
//
// This file also validates that glob patterns used in event filters are syntactically valid
// according to GitHub Actions glob syntax, using the glob validator in glob_validation.go:
//   - Branch and tag patterns use validateRefGlob
//   - Path patterns use validatePathGlob
//
// A notable check is that path patterns starting with "./" are always invalid in GitHub Actions.
//
// # Validation Functions
//
//   - ValidateEventFilters() - Main entry point for filter mutual-exclusivity validation
//   - ValidateGlobPatterns() - Main entry point for glob pattern syntax validation
//   - validateFilterExclusivity() - Validates a single event's filter configuration
//   - validateGlobList() - Validates a list of glob patterns for a given filter key
//
// # GitHub Actions Requirements
//
// From GitHub Actions documentation:
//   - You cannot use both branches and branches-ignore filters for the same event
//   - You cannot use both paths and paths-ignore filters for the same event
//
// These restrictions apply to push and pull_request event filters.
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates event filter configurations
//   - It checks for GitHub Actions filter requirements
//   - It validates mutual exclusivity of filter options
//   - It validates glob pattern syntax in event filters
//
// For general validation, see validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"errors"
	"fmt"
	"strings"
)

var filterValidationLog = newValidationLogger("filter")

// ValidateEventFilters checks for GitHub Actions filter mutual exclusivity rules
func ValidateEventFilters(frontmatter map[string]any) error {
	filterValidationLog.Print("Validating event filter mutual exclusivity")

	on, exists := frontmatter["on"]
	if !exists {
		filterValidationLog.Print("No 'on' section found, skipping filter validation")
		return nil
	}

	onMap, ok := on.(map[string]any)
	if !ok {
		filterValidationLog.Print("'on' section is not a map, skipping filter validation")
		return nil
	}

	// Check push event
	if pushVal, exists := onMap["push"]; exists {
		filterValidationLog.Print("Validating push event filters")
		if err := validateFilterExclusivity(pushVal, "push"); err != nil {
			return err
		}
	}

	// Check pull_request event
	if prVal, exists := onMap["pull_request"]; exists {
		filterValidationLog.Print("Validating pull_request event filters")
		if err := validateFilterExclusivity(prVal, "pull_request"); err != nil {
			return err
		}
	}

	filterValidationLog.Print("Event filter validation completed successfully")
	return nil
}

// validateFilterExclusivity validates that a single event doesn't use mutually exclusive filters
func validateFilterExclusivity(eventVal any, eventName string) error {
	eventMap, ok := eventVal.(map[string]any)
	if !ok {
		filterValidationLog.Printf("Event '%s' is not a map, skipping filter validation", eventName)
		return nil
	}

	// Check branches/branches-ignore
	_, hasBranches := eventMap["branches"]
	_, hasBranchesIgnore := eventMap["branches-ignore"]

	if hasBranches && hasBranchesIgnore {
		filterValidationLog.Printf("ERROR: Event '%s' has both 'branches' and 'branches-ignore' filters", eventName)
		return fmt.Errorf("%s event cannot specify both 'branches' and 'branches-ignore' - they are mutually exclusive per GitHub Actions requirements. Use either 'branches' to include specific branches, or 'branches-ignore' to exclude specific branches, but not both", eventName)
	}

	// Check paths/paths-ignore
	_, hasPaths := eventMap["paths"]
	_, hasPathsIgnore := eventMap["paths-ignore"]

	if hasPaths && hasPathsIgnore {
		filterValidationLog.Printf("ERROR: Event '%s' has both 'paths' and 'paths-ignore' filters", eventName)
		return fmt.Errorf("%s event cannot specify both 'paths' and 'paths-ignore' - they are mutually exclusive per GitHub Actions requirements. Use either 'paths' to include specific paths, or 'paths-ignore' to exclude specific paths, but not both", eventName)
	}

	filterValidationLog.Printf("Event '%s' filters are valid", eventName)
	return nil
}

// refFilterKeys are the event filter keys whose patterns must be valid Git ref globs.
var refFilterKeys = []string{"branches", "branches-ignore", "tags", "tags-ignore"}

// pathFilterKeys are the event filter keys whose patterns must be valid path globs.
var pathFilterKeys = []string{"paths", "paths-ignore"}

// globValidationEvents are the GitHub Actions event types that support branch/tag/path filters.
var globValidationEvents = []string{"push", "pull_request", "pull_request_target", "workflow_run"}

// ValidateGlobPatterns validates branch, tag, and path glob patterns in the 'on' section
// of a workflow's frontmatter. It returns the first validation error encountered, if any.
func ValidateGlobPatterns(frontmatter map[string]any) error {
	filterValidationLog.Print("Validating glob patterns in event filters")

	on, exists := frontmatter["on"]
	if !exists {
		return nil
	}

	onMap, ok := on.(map[string]any)
	if !ok {
		return nil
	}

	for _, eventName := range globValidationEvents {
		eventVal, exists := onMap[eventName]
		if !exists {
			continue
		}
		eventMap, ok := eventVal.(map[string]any)
		if !ok {
			continue
		}

		// Validate ref globs (branches, tags, branches-ignore, tags-ignore)
		for _, key := range refFilterKeys {
			if err := validateGlobList(eventMap, eventName, key, false); err != nil {
				return err
			}
		}

		// Validate path globs (paths, paths-ignore)
		for _, key := range pathFilterKeys {
			if err := validateGlobList(eventMap, eventName, key, true); err != nil {
				return err
			}
		}
	}

	filterValidationLog.Print("Glob pattern validation completed successfully")
	return nil
}

// validateGlobList validates each pattern in a filter list (e.g. branches, paths).
// When isPath is true, validatePathGlob is used; otherwise validateRefGlob.
func validateGlobList(eventMap map[string]any, eventName, filterKey string, isPath bool) error {
	val, exists := eventMap[filterKey]
	if !exists {
		return nil
	}

	patterns, err := toStringSlice(val)
	if err != nil {
		// Non-string-list values are skipped; schema validation handles type errors separately
		return nil
	}

	for _, pat := range patterns {
		var errs []invalidGlobPattern
		if isPath {
			errs = validatePathGlob(pat)
		} else {
			errs = validateRefGlob(pat)
		}
		if len(errs) > 0 {
			msgs := make([]string, 0, len(errs))
			for _, e := range errs {
				msgs = append(msgs, e.Message)
			}
			filterValidationLog.Printf("ERROR: invalid glob pattern %q in %s.%s: %s", pat, eventName, filterKey, strings.Join(msgs, "; "))
			return fmt.Errorf("invalid glob pattern %q in on.%s.%s: %s", pat, eventName, filterKey, strings.Join(msgs, "; "))
		}
	}
	return nil
}

// toStringSlice converts an any value to a []string, supporting []string, []any, and string.
func toStringSlice(val any) ([]string, error) {
	switch v := val.(type) {
	case []string:
		return v, nil
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, errors.New("non-string item in list")
			}
			result = append(result, s)
		}
		return result, nil
	case string:
		return []string{v}, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", val)
	}
}
