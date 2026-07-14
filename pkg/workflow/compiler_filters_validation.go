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

// ValidatePushBranchScope ensures that any push event in the on: section specifies a
// branch or tag ref filter. An unscoped push trigger fires on every push to every
// branch and tag, which causes unintended workflow fan-out on feature branches (the workflows
// activate immediately after new lock files are first pushed to the branch, producing
// zero-turn failures for every agentic workflow in the repository).
func ValidatePushBranchScope(frontmatter map[string]any) error {
	filterValidationLog.Print("Validating push event branch/tag scope")

	on, exists := frontmatter["on"]
	if !exists {
		return nil
	}

	onMap, ok := on.(map[string]any)
	if !ok {
		return nil
	}

	pushVal, hasPush := onMap["push"]
	if !hasPush {
		return nil
	}

	// A nil push value (bare `push:` key with no sub-keys) is unscoped.
	if pushVal == nil {
		filterValidationLog.Print("ERROR: push event has no branch/tag scope (nil push value)")
		return newUnScopedPushError()
	}

	pushMap, ok := pushVal.(map[string]any)
	if !ok {
		// Non-map push value (unexpected type); skip.
		return nil
	}

	_, hasBranches := pushMap["branches"]
	_, hasBranchesIgnore := pushMap["branches-ignore"]
	_, hasTags := pushMap["tags"]
	_, hasTagsIgnore := pushMap["tags-ignore"]

	if !hasBranches && !hasBranchesIgnore && !hasTags && !hasTagsIgnore {
		filterValidationLog.Print("ERROR: push event has no branches or tags scope")
		return newUnScopedPushError()
	}

	filterValidationLog.Print("Push event branches or tags scope is valid")
	return nil
}

func newUnScopedPushError() *WorkflowValidationError {
	return NewValidationError(
		"on.push",
		"push (no branch or tag filter)",
		"push event must specify a 'branches', 'branches-ignore', 'tags', or 'tags-ignore' filter; an unscoped push trigger fires on every push to every branch and tag and causes unintended workflow fan-out on feature branches",
		"Add a branch or tag filter to the push trigger:\n\non:\n  push:\n    branches:\n      - main\n\n# or for tag-based releases:\n\non:\n  push:\n    tags:\n      - 'v*.*.*'",
	)
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
		return NewValidationError(
			"on."+eventName,
			"branches + branches-ignore",
			eventName+" event cannot specify both 'branches' and 'branches-ignore'; expected exactly one branch filter because they are mutually exclusive in GitHub Actions",
			fmt.Sprintf("Use one branch filter for on.%s:\n\non:\n  %s:\n    branches:\n      - main\n    # OR\n    # branches-ignore:\n    #   - release/**", eventName, eventName),
		)
	}

	// Check paths/paths-ignore
	_, hasPaths := eventMap["paths"]
	_, hasPathsIgnore := eventMap["paths-ignore"]

	if hasPaths && hasPathsIgnore {
		filterValidationLog.Printf("ERROR: Event '%s' has both 'paths' and 'paths-ignore' filters", eventName)
		return NewValidationError(
			"on."+eventName,
			"paths + paths-ignore",
			eventName+" event cannot specify both 'paths' and 'paths-ignore'; expected exactly one path filter because they are mutually exclusive in GitHub Actions",
			fmt.Sprintf("Use one path filter for on.%s:\n\non:\n  %s:\n    paths:\n      - src/**\n    # OR\n    # paths-ignore:\n    #   - docs/**", eventName, eventName),
		)
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

	patterns := parseStringSliceAny(val, nil)
	if len(patterns) == 0 {
		// Skip when the value is absent, empty, or a non-list type.
		// parseStringSliceAny returns nil for unrecognised types (e.g., int, bool);
		// those type errors are handled separately by schema validation.
		return nil
	}

	validate := validateRefGlob
	if isPath {
		validate = validatePathGlob
	}

	return validateGlobPatternList(patterns, validate, func(_ int, pat string, msgs []string) error {
		filterValidationLog.Printf("ERROR: invalid glob pattern %q in %s.%s: %s", pat, eventName, filterKey, strings.Join(msgs, "; "))
		return NewValidationError(
			fmt.Sprintf("on.%s.%s", eventName, filterKey),
			pat,
			"expected a valid GitHub Actions glob pattern: "+strings.Join(msgs, "; "),
			fmt.Sprintf("Use valid glob syntax for on.%s.%s:\n\non:\n  %s:\n    %s:\n      - src/**", eventName, filterKey, eventName, filterKey),
		)
	})
}
