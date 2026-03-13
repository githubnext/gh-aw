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
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - Adding new cache-memory configuration constraints
//   - Adding cross-cache validation rules (e.g., total size limits)

package workflow

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
