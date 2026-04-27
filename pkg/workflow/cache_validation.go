// This file provides validation for sandbox cache-memory configuration.
//
// # Cache Memory Validation
//
// This file validates that cache-memory entries in a workflow's sandbox
// configuration have unique IDs, preventing runtime conflicts when multiple
// cache entries are defined.
//
// # Validation Functions
//
//   - validateNoDuplicateCacheIDs() - Ensures each cache entry has a unique ID
//   - validateNoCacheKeyRunID() - Rejects cache keys that reference github.run_id
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - Adding new cache-memory configuration constraints
//   - Adding cross-cache validation rules (e.g., total size limits)

package workflow

import "regexp"

// cacheKeyRunIDPattern matches github.run_id as a complete token — not as a
// prefix of a longer identifier like "github.run_identifier".
// It matches "github.run_id" that is either:
//   - not followed by an underscore or word character (letter/digit/underscore)
//   - at the end of the string
var cacheKeyRunIDPattern = regexp.MustCompile(`github\.run_id(?:[^_\w]|$)`)

// validateNoCacheKeyRunID returns an error when a user-supplied cache key
// contains the ${{ github.run_id }} expression.
//
// Including run_id in a custom key makes every run write to a unique cache slot,
// so the primary cache entry can never be restored from a previous run.
// Custom keys are used as-is (stable) — they are NOT modified by the compiler.
// Users who want rolling-cache behaviour should use the default key (omit the key
// field entirely) which the compiler handles with its own run_id suffix.
func validateNoCacheKeyRunID(key string) error {
	if cacheKeyRunIDPattern.MatchString(key) {
		return NewValidationError(
			"tools.cache-memory.key",
			key,
			"cache key must not reference github.run_id — every run would write to a unique cache slot, preventing cross-run cache restoration",
			"Remove github.run_id from the key. Custom keys are kept stable so the same primary cache entry is reused across runs.\n\nExample:\n\ntools:\n  cache-memory:\n    key: my-data-${{ github.repository_owner }}\n    # ✓ stable key — same primary entry restored on every run",
		)
	}
	return nil
}

// validateNoDuplicateCacheIDs checks for duplicate cache IDs and returns an error if found.
// Uses the generic validateNoDuplicateIDs helper for consistent duplicate detection.
func validateNoDuplicateCacheIDs(caches []CacheMemoryEntry) error {
	cacheLog.Printf("Validating cache IDs: checking %d caches for duplicates", len(caches))
	err := validateNoDuplicateIDs(caches, func(c CacheMemoryEntry) string { return c.ID }, func(id string) error {
		cacheLog.Printf("Duplicate cache ID found: %s", id)
		return NewValidationError(
			"sandbox.cache-memory",
			id,
			"duplicate cache-memory ID found - each cache must have a unique ID",
			"Change the cache ID to a unique value. Example:\n\nsandbox:\n  cache-memory:\n    - id: cache-1\n      size: 100MB\n    - id: cache-2  # Use unique IDs\n      size: 50MB",
		)
	})
	if err != nil {
		return err
	}
	cacheLog.Print("Cache ID validation passed: no duplicates found")
	return nil
}
