package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var actionCacheLog = logger.New("workflow:action_cache")

const (
	// CacheFileName is the name of the cache file in .github/aw/.
	CacheFileName = "actions-lock.json"
)

// ActionCacheEntry represents a cached action pin resolution.
type ActionCacheEntry struct {
	Repo              string                      `json:"repo"`
	Version           string                      `json:"version"`
	SHA               string                      `json:"sha"`
	Inputs            map[string]*ActionYAMLInput `json:"inputs,omitempty"`             // cached inputs from action.yml
	ActionDescription string                      `json:"action_description,omitempty"` // cached description from action.yml
}

// ContainerPin holds a pinned Docker container image reference.
// The pin maps a mutable image tag to its immutable SHA-256 digest,
// ensuring supply-chain integrity by making the pull operation deterministic.
type ContainerPin struct {
	Image       string `json:"image"`        // Original tag, e.g. "node:lts-alpine"
	Digest      string `json:"digest"`       // Bare digest, e.g. "sha256:abc123..."
	PinnedImage string `json:"pinned_image"` // Resolved reference, e.g. "node:lts-alpine@sha256:abc123..."
}

// ActionCache manages cached action pin resolutions.
type ActionCache struct {
	Entries       map[string]ActionCacheEntry `json:"entries"`              // key: "repo@version"
	ContainerPins map[string]ContainerPin     `json:"containers,omitempty"` // key: image tag
	path          string
	dirty         bool // tracks if cache has unsaved changes
}

// NewActionCache creates a new action cache instance
func NewActionCache(repoRoot string) *ActionCache {
	cachePath := filepath.Join(repoRoot, ".github", "aw", CacheFileName)
	actionCacheLog.Printf("Creating action cache with path: %s", cachePath)
	return &ActionCache{
		Entries:       make(map[string]ActionCacheEntry),
		ContainerPins: make(map[string]ContainerPin),
		path:          cachePath,
		// dirty is initialized to false (zero value)
	}
}

// GetContainerPin returns the cached pin for the given image tag.
// Returns the pin and true if a digest pin is present, otherwise empty pin and false.
func (c *ActionCache) GetContainerPin(image string) (ContainerPin, bool) {
	if c.ContainerPins == nil {
		return ContainerPin{}, false
	}
	pin, ok := c.ContainerPins[image]
	if !ok {
		actionCacheLog.Printf("Container pin cache miss for image=%s", image)
		return ContainerPin{}, false
	}
	actionCacheLog.Printf("Container pin cache hit for image=%s, pinned=%s", image, pin.PinnedImage)
	return pin, true
}

// SetContainerPin stores a digest pin for the given image tag.
// digest must be in the form "sha256:<hex>" and pinnedImage must be the full
// reference including the digest (e.g., "node:lts-alpine@sha256:<hex>").
func (c *ActionCache) SetContainerPin(image, digest, pinnedImage string) {
	if c.ContainerPins == nil {
		c.ContainerPins = make(map[string]ContainerPin)
	}
	c.ContainerPins[image] = ContainerPin{
		Image:       image,
		Digest:      digest,
		PinnedImage: pinnedImage,
	}
	c.dirty = true
	actionCacheLog.Printf("Set container pin: image=%s, digest=%s", image, digest)
}

// DeleteContainerPin removes the pin for the given image tag.
// It is a no-op if the image has no cached pin.
func (c *ActionCache) DeleteContainerPin(image string) {
	if c.ContainerPins == nil {
		return
	}
	if _, exists := c.ContainerPins[image]; exists {
		delete(c.ContainerPins, image)
		c.dirty = true
		actionCacheLog.Printf("Deleted container pin for image=%s", image)
	}
}

// Load loads the cache from disk
func (c *ActionCache) Load() error {
	actionCacheLog.Printf("Loading action cache from: %s", c.path)
	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache file doesn't exist yet, that's OK
			actionCacheLog.Print("Cache file does not exist, starting with empty cache")
			return nil
		}
		actionCacheLog.Printf("Failed to read cache file: %v", err)
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		actionCacheLog.Printf("Failed to unmarshal cache data: %v", err)
		return err
	}

	// Ensure maps are initialized even when absent from the JSON (backward compatibility).
	if c.Entries == nil {
		c.Entries = make(map[string]ActionCacheEntry)
	}
	if c.ContainerPins == nil {
		c.ContainerPins = make(map[string]ContainerPin)
	}

	// Mark cache as clean after successful load (it matches disk state)
	c.dirty = false

	actionCacheLog.Printf("Successfully loaded cache with %d entries, %d container pins", len(c.Entries), len(c.ContainerPins))
	return nil
}

// Save saves the cache to disk with sorted entries
// If the cache is empty, the file is not created or is deleted if it exists
// Deduplicates entries by keeping only the most precise version reference for each repo+SHA combination
// Only saves if the cache has been modified (dirty flag is true)
func (c *ActionCache) Save() error {
	// Skip saving if cache hasn't been modified
	if !c.dirty {
		actionCacheLog.Printf("Cache is clean (no changes), skipping save")
		return nil
	}

	actionCacheLog.Printf("Saving action cache to: %s with %d entries", c.path, len(c.Entries))

	// If cache is empty (no entries and no container pins), skip saving and delete the file if it exists
	if len(c.Entries) == 0 && len(c.ContainerPins) == 0 {
		actionCacheLog.Print("Cache is empty, skipping file creation")
		// Remove the file if it exists
		if _, err := os.Stat(c.path); err == nil {
			actionCacheLog.Printf("Removing existing empty cache file: %s", c.path)
			if err := os.Remove(c.path); err != nil {
				actionCacheLog.Printf("Failed to remove empty cache file: %v", err)
				return err
			}
		}
		c.dirty = false
		return nil
	}

	// Deduplicate entries before saving
	c.deduplicateEntries()

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		actionCacheLog.Printf("Failed to create cache directory: %v", err)
		return err
	}

	// Marshal with sorted entries
	data, err := c.marshalSorted()
	if err != nil {
		actionCacheLog.Printf("Failed to marshal cache data: %v", err)
		return err
	}

	// Add trailing newline for prettier compliance
	data = append(data, '\n')

	if err := os.WriteFile(c.path, data, 0644); err != nil {
		actionCacheLog.Printf("Failed to write cache file: %v", err)
		return err
	}

	actionCacheLog.Print("Successfully saved action cache")
	c.dirty = false
	return nil
}

// marshalSorted marshals the cache with entries sorted by key
func (c *ActionCache) marshalSorted() ([]byte, error) {
	// Extract and sort the entry keys
	keys := make([]string, 0, len(c.Entries))
	for key := range c.Entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Manually construct JSON with sorted keys
	var result []byte
	result = append(result, []byte("{\n  \"entries\": {\n")...)

	for i, key := range keys {
		entry := c.Entries[key]

		// Marshal the entry
		entryJSON, err := json.MarshalIndent(entry, "    ", "  ")
		if err != nil {
			return nil, err
		}

		// Add the key and entry
		result = append(result, []byte("    \""+key+"\": ")...)
		result = append(result, entryJSON...)

		// Add comma if not the last entry
		if i < len(keys)-1 {
			result = append(result, ',')
		}
		result = append(result, '\n')
	}

	result = append(result, []byte("  }")...)

	// Add containers section if non-empty
	if len(c.ContainerPins) > 0 {
		pinKeys := make([]string, 0, len(c.ContainerPins))
		for k := range c.ContainerPins {
			pinKeys = append(pinKeys, k)
		}
		sort.Strings(pinKeys)

		result = append(result, []byte(",\n  \"containers\": {\n")...)
		for i, k := range pinKeys {
			pin := c.ContainerPins[k]
			pinJSON, err := json.MarshalIndent(pin, "    ", "  ")
			if err != nil {
				return nil, err
			}
			result = append(result, []byte("    \""+k+"\": ")...)
			result = append(result, pinJSON...)
			if i < len(pinKeys)-1 {
				result = append(result, ',')
			}
			result = append(result, '\n')
		}
		result = append(result, []byte("  }")...)
	}

	result = append(result, '\n', '}')
	return result, nil
}

// Delete removes the cache entry for the given repo and version.
// It first tries the canonical formatted key, then falls back to scanning all
// entries for a matching repo+version pair to handle key/version mismatches.
// It is a no-op if no matching entry is found.
func (c *ActionCache) Delete(repo, version string) {
	key := formatActionCacheKey(repo, version)

	deleted := false

	// First, try deleting by the canonical formatted key.
	if _, exists := c.Entries[key]; exists {
		delete(c.Entries, key)
		deleted = true
		actionCacheLog.Printf("Deleted cache entry: key=%s", key)
	}

	// Also delete any entries whose stored fields match repo and version,
	// in case the map key does not exactly match formatActionCacheKey
	// (key/version mismatch in the cache file).
	for k, entry := range c.Entries {
		if entry.Repo == repo && entry.Version == version {
			delete(c.Entries, k)
			deleted = true
			actionCacheLog.Printf("Deleted cache entry with mismatched key: key=%s, repo=%s, version=%s", k, repo, version)
		}
	}

	if deleted {
		c.dirty = true
	}
}

// DeleteByKey removes the cache entry with the given raw map key.
// This is useful when the caller already holds the exact key from iterating
// the Entries map, avoiding recomputation and handling key/version mismatches.
// It is a no-op if the key does not exist.
func (c *ActionCache) DeleteByKey(key string) {
	if _, exists := c.Entries[key]; exists {
		delete(c.Entries, key)
		c.dirty = true
		actionCacheLog.Printf("Deleted cache entry by key: key=%s", key)
	}
}

// Get retrieves a cached entry if it exists
func (c *ActionCache) Get(repo, version string) (string, bool) {
	key := formatActionCacheKey(repo, version)
	entry, exists := c.Entries[key]
	if !exists {
		actionCacheLog.Printf("Cache miss for key=%s", key)
		return "", false
	}

	actionCacheLog.Printf("Cache hit for key=%s, sha=%s", key, entry.SHA)
	return entry.SHA, true
}

// FindEntryBySHA finds a cache entry with the given repo and SHA
// Returns the entry and true if found, or empty entry and false if not found
func (c *ActionCache) FindEntryBySHA(repo, sha string) (ActionCacheEntry, bool) {
	for key, entry := range c.Entries {
		if entry.Repo == repo && entry.SHA == sha {
			actionCacheLog.Printf("Found cache entry for %s with SHA %s: %s", repo, sha[:8], key)
			return entry, true
		}
	}
	return ActionCacheEntry{}, false
}

// Set stores a new cache entry, preserving any already-cached inputs when the SHA
// is unchanged. If the SHA changes (e.g. a moving tag points to a new commit),
// cached inputs are cleared to stay consistent with the newly-pinned commit.
func (c *ActionCache) Set(repo, version, sha string) {
	key := formatActionCacheKey(repo, version)

	// Check if there are existing entries with the same repo+SHA but different version
	for existingKey, entry := range c.Entries {
		if entry.Repo == repo && entry.SHA == sha && entry.Version != version {
			// Truncate SHA for logging (handle short SHAs in tests)
			shortSHA := sha
			if len(sha) > 8 {
				shortSHA = sha[:8]
			}
			actionCacheLog.Printf("WARNING: Adding cache entry %s with SHA %s that already exists as %s",
				key, shortSHA, existingKey)
			actionCacheLog.Printf("This may cause version comment flipping in lock files. Consider using consistent version tags.")
		}
	}

	actionCacheLog.Printf("Setting cache entry: key=%s, sha=%s", key, sha)

	// Preserve previously-cached inputs only when the SHA is unchanged. If the SHA
	// changes (e.g. for a moving tag that now points to a new commit), drop any
	// existing inputs so they stay consistent with the pinned commit.
	existing := c.Entries[key]
	var inputs map[string]*ActionYAMLInput
	var description string
	if existing.SHA == sha {
		inputs = existing.Inputs
		description = existing.ActionDescription
	} else if existing.SHA != "" {
		// Log when an existing entry's SHA is being changed (covers both the case
		// where cached inputs exist and where they don't, for consistent observability).
		actionCacheLog.Printf("Clearing cached inputs for key=%s due to SHA change (%s -> %s)", key, existing.SHA, sha)
	}
	c.Entries[key] = ActionCacheEntry{
		Repo:              repo,
		Version:           version,
		SHA:               sha,
		Inputs:            inputs,
		ActionDescription: description,
	}
	c.dirty = true // Mark cache as modified
}

// GetInputs retrieves the cached action inputs for the given repo and version.
// Returns the inputs map and true if cached inputs exist, otherwise nil and false.
func (c *ActionCache) GetInputs(repo, version string) (map[string]*ActionYAMLInput, bool) {
	key := formatActionCacheKey(repo, version)
	entry, exists := c.Entries[key]
	if !exists || entry.Inputs == nil {
		actionCacheLog.Printf("No cached inputs for key=%s", key)
		return nil, false
	}
	actionCacheLog.Printf("Cache hit for inputs: key=%s, inputs=%d", key, len(entry.Inputs))
	return entry.Inputs, true
}

// SetInputs stores the action inputs in the cache entry for the given repo and version.
// If no cache entry exists for the key, a new entry is created with an empty SHA so that
// inputs fetched from the network are persisted even before the SHA is resolved.
func (c *ActionCache) SetInputs(repo, version string, inputs map[string]*ActionYAMLInput) {
	key := formatActionCacheKey(repo, version)
	entry, exists := c.Entries[key]
	if !exists {
		actionCacheLog.Printf("No cache entry for key=%s, creating new entry to store inputs", key)
		entry = ActionCacheEntry{
			Repo:    repo,
			Version: version,
		}
	}
	entry.Inputs = inputs
	c.Entries[key] = entry
	c.dirty = true
	actionCacheLog.Printf("Cached inputs for key=%s, inputs=%d", key, len(inputs))
}

// GetActionDescription retrieves the cached action description for the given repo and version.
// Returns the description and true if a non-empty description is cached, otherwise "" and false.
func (c *ActionCache) GetActionDescription(repo, version string) (string, bool) {
	key := formatActionCacheKey(repo, version)
	entry, exists := c.Entries[key]
	if !exists || entry.ActionDescription == "" {
		return "", false
	}
	return entry.ActionDescription, true
}

// SetActionDescription stores the action description in the cache entry for the given repo and version.
// If no cache entry exists for the key, a new entry is created.
// Empty descriptions are not stored; actions without a description string are treated the same as
// actions whose description has not yet been fetched, so we avoid caching an empty string that
// would prevent a later fetch from populating the field.
func (c *ActionCache) SetActionDescription(repo, version, description string) {
	if description == "" {
		// Skip persisting empty descriptions; callers that want to distinguish
		// "no description fetched" from "action has no description" should use
		// a sentinel value. For our use case (action.yml display text), omitting
		// empty values is intentional to keep the cache file tidy.
		return
	}
	key := formatActionCacheKey(repo, version)
	entry, exists := c.Entries[key]
	if !exists {
		entry = ActionCacheEntry{
			Repo:    repo,
			Version: version,
		}
	}
	entry.ActionDescription = description
	c.Entries[key] = entry
	c.dirty = true
	actionCacheLog.Printf("Cached description for key=%s", key)
}

// GetCachePath returns the path to the cache file
func (c *ActionCache) GetCachePath() string {
	return c.path
}

// deduplicateEntries removes duplicate entries by keeping only the most precise version reference
// for each repo+SHA combination. For example, if both "actions/cache@v4" and "actions/cache@v4.3.0"
// point to the same SHA and version, only "actions/cache@v4.3.0" is kept.
func (c *ActionCache) deduplicateEntries() {
	// Group entries by repo+SHA
	type entryKey struct {
		repo string
		sha  string
	}
	groups := make(map[entryKey][]string)

	for key, entry := range c.Entries {
		ek := entryKey{repo: entry.Repo, sha: entry.SHA}
		groups[ek] = append(groups[ek], key)
	}

	// For each group with multiple entries, keep only the most precise one
	var toDelete []string
	var deduplicationDetails []string // Track details for user-friendly message

	for ek, keys := range groups {
		if len(keys) <= 1 {
			continue
		}

		// Truncate SHA for logging (handle short SHAs in tests)
		shortSHA := ek.sha
		if len(ek.sha) > 8 {
			shortSHA = ek.sha[:8]
		}
		actionCacheLog.Printf("Found %d cache entries for %s with SHA %s", len(keys), ek.repo, shortSHA)

		// Find the most precise version reference
		// Extract the version reference from each key (format: "repo@versionRef")
		type keyInfo struct {
			key        string
			versionRef string
		}
		keyInfos := make([]keyInfo, len(keys))
		for i, key := range keys {
			parts := strings.SplitN(key, "@", 2)
			versionRef := ""
			if len(parts) == 2 {
				versionRef = parts[1]
			}
			keyInfos[i] = keyInfo{key: key, versionRef: versionRef}
		}

		// Sort by version precision (most precise first)
		sort.Slice(keyInfos, func(i, j int) bool {
			return isMorePreciseVersion(keyInfos[i].versionRef, keyInfos[j].versionRef)
		})

		// Keep the most precise version, mark others for deletion
		keepVersion := keyInfos[0].versionRef
		var removedVersions []string
		for i := 1; i < len(keyInfos); i++ {
			toDelete = append(toDelete, keyInfos[i].key)
			removedVersions = append(removedVersions, keyInfos[i].versionRef)
			actionCacheLog.Printf("Deduplicating: keeping %s, removing %s", keyInfos[0].key, keyInfos[i].key)
		}

		// Build user-friendly message
		detail := fmt.Sprintf("%s: kept %s, removed %s", ek.repo, keepVersion, strings.Join(removedVersions, ", "))
		deduplicationDetails = append(deduplicationDetails, detail)
	}

	// Delete the less precise entries
	for _, key := range toDelete {
		delete(c.Entries, key)
	}

	if len(toDelete) > 0 {
		actionCacheLog.Printf("Deduplicated %d entries, %d entries remaining", len(toDelete), len(c.Entries))
		// Log detailed deduplication info at verbose level
		for _, detail := range deduplicationDetails {
			actionCacheLog.Printf("Deduplication detail: %s", detail)
		}
	}
}

// PruneStaleGHAWEntries removes entries from the cache for the gh-aw-actions
// repository whose version does not match the current compiler version.
//
// When the compiler is updated (e.g., from v0.67.1 to v0.67.3), previously
// compiled workflows referenced setup@v0.67.1 but the new compiler pins to
// setup@v0.67.3. Without pruning, both entries survive in actions-lock.json,
// leaving a stale entry that is never referenced by any compiled lock file.
//
// Only prunes when the current version is a release version (starts with "v").
// Dev builds, empty versions, and other non-release versions are skipped to
// avoid accidentally removing valid entries during development.
//
// Parameters:
//   - currentVersion: the compiler version that is currently in use (e.g., "v0.67.3")
//   - actionsRepoPrefix: the org/repo prefix for gh-aw-actions (e.g., "github/gh-aw-actions")
func (c *ActionCache) PruneStaleGHAWEntries(currentVersion string, actionsRepoPrefix string) {
	if currentVersion == "" || actionsRepoPrefix == "" {
		return
	}
	// Only prune for clean release versions (e.g., "v0.67.3"), not dev/dirty builds
	if !strings.HasPrefix(currentVersion, "v") || strings.Contains(currentVersion, "-") {
		return
	}

	var toDelete []string
	for key, entry := range c.Entries {
		if !strings.HasPrefix(entry.Repo, actionsRepoPrefix+"/") {
			continue
		}
		if entry.Version != currentVersion {
			actionCacheLog.Printf("Pruning stale gh-aw-actions entry: %s (version %s != current %s)", key, entry.Version, currentVersion)
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		delete(c.Entries, key)
	}

	if len(toDelete) > 0 {
		c.dirty = true
		actionCacheLog.Printf("Pruned %d stale gh-aw-actions entries, %d entries remaining", len(toDelete), len(c.Entries))
	}
}

// isMorePreciseVersion returns true if v1 is more precise than v2
// For example: "v4.3.0" is more precise than "v4"
func isMorePreciseVersion(v1, v2 string) bool {
	// Count the number of dots in each version string
	// More dots means more precision
	dots1 := strings.Count(v1, ".")
	dots2 := strings.Count(v2, ".")

	if dots1 != dots2 {
		return dots1 > dots2
	}

	// If same number of dots, compare lexicographically
	// This handles cases like "v1.2.3" vs "v1.2.10" correctly
	return v1 > v2
}
