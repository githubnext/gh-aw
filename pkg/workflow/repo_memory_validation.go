// This file provides validation for repo-memory configuration.
//
// # Repo Memory Validation
//
// This file validates that repo-memory entries have unique IDs and that
// branch prefix configuration meets naming requirements.
//
// # Validation Functions
//
//   - validateBranchPrefix() - Validates branch prefix length, format, and reserved names
//   - validateNoDuplicateMemoryIDs() - Ensures each memory entry has a unique ID
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - Adding new repo-memory configuration constraints
//   - Adding new branch naming rules

package workflow

import (
	"errors"
	"fmt"
	"strings"
)

var repoMemValidationLog = newValidationLogger("repo_memory")

// validateBranchPrefix validates that the branch prefix meets requirements
func validateBranchPrefix(prefix string) error {
	if prefix == "" {
		return nil // Empty means use default
	}

	repoMemValidationLog.Printf("Validating branch prefix: %q", prefix)

	// Check length (4-32 characters)
	if len(prefix) < 4 {
		return fmt.Errorf("branch-prefix must be at least 4 characters long, got %d", len(prefix))
	}
	if len(prefix) > 32 {
		return fmt.Errorf("branch-prefix must be at most 32 characters long, got %d", len(prefix))
	}

	// Check for alphanumeric and branch-friendly characters (alphanumeric, hyphens, underscores)
	// Use pre-compiled regex from package level for performance
	if !branchPrefixValidPattern.MatchString(prefix) {
		return fmt.Errorf("branch-prefix must contain only alphanumeric characters, hyphens, and underscores, got '%s'", prefix)
	}

	// Cannot be "copilot"
	if strings.ToLower(prefix) == "copilot" {
		return errors.New("branch-prefix cannot be 'copilot' (reserved)")
	}

	repoMemValidationLog.Printf("Branch prefix %q passed validation", prefix)
	return nil
}

// validateNoDuplicateMemoryIDs checks for duplicate memory IDs and returns an error if found.
// Uses the generic validateNoDuplicateIDs helper for consistent duplicate detection.
func validateNoDuplicateMemoryIDs(memories []RepoMemoryEntry) error {
	repoMemValidationLog.Printf("Validating %d memory entries for duplicate IDs", len(memories))
	return validateNoDuplicateIDs(memories, func(m RepoMemoryEntry) string { return m.ID }, func(id string) error {
		return fmt.Errorf("duplicate memory ID found: '%s'. Each memory must have a unique ID", id)
	})
}
