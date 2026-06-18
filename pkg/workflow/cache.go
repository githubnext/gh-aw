package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var cacheLog = logger.New("workflow:cache")

// defaultCacheMemoryDir is the canonical runtime path for the default cache-memory.
// Backward-compatible: workflows that were compiled before multi-cache support was added
// continue to use this exact path.
const defaultCacheMemoryDir = "/tmp/gh-aw/cache-memory"

// cacheMemoryDirPrefix is the path prefix for non-default cache-memory directories.
// The full path is formed by appending the cache ID: cacheMemoryDirPrefix + cacheID.
const cacheMemoryDirPrefix = "/tmp/gh-aw/cache-memory-"

// cacheMemoryDirFor returns the canonical runtime directory for the given cache ID.
// Default cache → /tmp/gh-aw/cache-memory
// Named cache   → /tmp/gh-aw/cache-memory-{id}
//
// The returned path has no trailing slash. Callers that display the path as a directory
// (e.g. in LLM prompt context) should append "/" explicitly.
//
// An empty cacheID is treated the same as "default" as a safety net, though callers
// should always provide a non-empty ID.
//
// Non-default IDs must have already been validated by isValidCacheID before reaching
// this function. This function panics on invalid IDs as a defence-in-depth measure
// (the parser should have rejected them first).
func cacheMemoryDirFor(cacheID string) string {
	if cacheID == "default" || cacheID == "" {
		return defaultCacheMemoryDir
	}
	if !isValidCacheID(cacheID) {
		// This should never happen: parseCacheMemoryEntry validates IDs at parse time.
		// Panic here to surface a clear programming error rather than silently producing
		// a dangerous path.
		panic(fmt.Sprintf("cacheMemoryDirFor called with invalid cache ID %q; IDs must match [A-Za-z0-9_-]{1,64}", cacheID))
	}
	return cacheMemoryDirPrefix + cacheID
}

// validCacheMemoryScopes defines the allowed values for cache-memory scope
var validCacheMemoryScopes = []string{"workflow", "repo"}

// isValidCacheID reports whether id is a safe cache identifier.
// Allowed pattern: ^[A-Za-z0-9_-]{1,64}$ (1-64 characters).
// This prevents path-traversal attacks (e.g. "../../etc") when the ID is
// appended to cacheMemoryDirPrefix to form a filesystem path.
func isValidCacheID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		isLower := c >= 'a' && c <= 'z'
		isUpper := c >= 'A' && c <= 'Z'
		isDigit := c >= '0' && c <= '9'
		isAllowed := c == '_' || c == '-'
		if !isLower && !isUpper && !isDigit && !isAllowed {
			return false
		}
	}
	return true
}

// isValidFileExtension reports whether s is a valid file extension of the form ^\.[A-Za-z0-9]+$
// (e.g. ".json", ".md"). This strict pattern prevents YAML injection when extensions are
// embedded in generated workflow YAML as single-quoted scalars.
func isValidFileExtension(s string) bool {
	if len(s) < 2 || s[0] != '.' {
		return false
	}
	for _, c := range s[1:] {
		isLower := c >= 'a' && c <= 'z'
		isUpper := c >= 'A' && c <= 'Z'
		isDigit := c >= '0' && c <= '9'
		if !isLower && !isUpper && !isDigit {
			return false
		}
	}
	return true
}

// CacheMemoryConfig holds configuration for cache-memory functionality
type CacheMemoryConfig struct {
	Caches []CacheMemoryEntry `yaml:"caches,omitempty"` // cache configurations
}

// CacheMemoryEntry represents a single cache-memory configuration
type CacheMemoryEntry struct {
	ID                string   `yaml:"id"`                           // cache identifier (required for array notation)
	Key               string   `yaml:"key,omitempty"`                // custom cache key
	Description       string   `yaml:"description,omitempty"`        // optional description for this cache
	RetentionDays     *int     `yaml:"retention-days,omitempty"`     // retention days for upload-artifact action
	RestoreOnly       bool     `yaml:"restore-only,omitempty"`       // if true, only restore cache without saving
	Scope             string   `yaml:"scope,omitempty"`              // scope for restore keys: "workflow" (default) or "repo"
	AllowedExtensions []string `yaml:"allowed-extensions,omitempty"` // allowed file extensions (default: [".json", ".jsonl", ".txt", ".md", ".csv"])
}

// generateDefaultCacheKey generates a default cache key for a given cache ID.
// Uses the legacy format (without integrity prefix) for backward compatibility when
// computing keys during initial entry parsing. The final key used in generated steps
// is produced by computeIntegrityCacheKey, which includes integrity level and policy hash.
func generateDefaultCacheKey(cacheID string) string {
	if cacheID == "default" {
		return "memory-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}"
	}
	return fmt.Sprintf("memory-%s-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}", cacheID)
}

// parseCacheMemoryEntry parses a single cache-memory entry from a map
func parseCacheMemoryEntry(cacheMap map[string]any, defaultID string) (CacheMemoryEntry, error) {
	cacheLog.Printf("Parsing cache-memory entry: defaultID=%s", defaultID)
	entry := CacheMemoryEntry{
		ID:  defaultID,
		Key: generateDefaultCacheKey(defaultID),
	}
	if err := parseCacheMemoryIdentity(cacheMap, defaultID, &entry); err != nil {
		return entry, err
	}
	parseCacheMemoryDescription(cacheMap, &entry)
	if err := parseCacheMemoryRetentionDays(cacheMap, &entry); err != nil {
		return entry, err
	}
	parseCacheMemoryRestoreOnly(cacheMap, &entry)
	if err := parseCacheMemoryScope(cacheMap, &entry); err != nil {
		return entry, err
	}
	if err := parseCacheMemoryAllowedExtensions(cacheMap, &entry); err != nil {
		return entry, err
	}
	applyDefaultAllowedExtensions(&entry)
	cacheLog.Printf("Parsed cache-memory entry: id=%s, scope=%s, restore-only=%v, retention-days=%v", entry.ID, entry.Scope, entry.RestoreOnly, entry.RetentionDays)
	return entry, nil
}

func parseCacheMemoryIdentity(cacheMap map[string]any, defaultID string, entry *CacheMemoryEntry) error {
	if idStr, ok := cacheMap["id"].(string); ok {
		if idStr != "default" && !isValidCacheID(idStr) {
			return fmt.Errorf("invalid cache-memory id %q: must contain only letters, digits, underscores, or hyphens (1-64 characters)", idStr)
		}
		entry.ID = idStr
	}
	if entry.ID != defaultID {
		entry.Key = generateDefaultCacheKey(entry.ID)
	}
	keyStr, ok := cacheMap["key"].(string)
	if !ok {
		return nil
	}
	if err := validateNoCacheKeyRunID(keyStr); err != nil {
		return err
	}
	entry.Key = ensureCacheRunIDSuffix(keyStr)
	return nil
}

func ensureCacheRunIDSuffix(key string) string {
	runIdSuffix := "-${{ github.run_id }}"
	if strings.HasSuffix(key, runIdSuffix) {
		return key
	}
	return key + runIdSuffix
}

func parseCacheMemoryDescription(cacheMap map[string]any, entry *CacheMemoryEntry) {
	if descStr, ok := cacheMap["description"].(string); ok {
		entry.Description = descStr
	}
}

func parseCacheMemoryRetentionDays(cacheMap map[string]any, entry *CacheMemoryEntry) error {
	retentionDays, exists := cacheMap["retention-days"]
	if !exists {
		return nil
	}
	entry.RetentionDays = parseOptionalInt(retentionDays)
	if entry.RetentionDays == nil {
		return nil
	}
	return validateIntRange(*entry.RetentionDays, 1, 90, "retention-days")
}

// parseOptionalInt safely converts YAML numeric values (int, float64, uint64) to *int.
//
// It returns nil when the input cannot be represented as an integer for the current
// architecture, including:
//   - NaN/Inf float64 values
//   - fractional float64 values
//   - float64 values outside the exact-integer range [-2^53, 2^53]
//   - float64 values outside the current architecture int range
//   - uint64 values larger than math.MaxInt
//   - unsupported types
func parseOptionalInt(value any) *int {
	// YAML unmarshaling can yield int, float64, or uint64 depending on parser/input.
	if intValue, ok := value.(int); ok {
		return &intValue
	}
	if floatValue, ok := value.(float64); ok {
		if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
			return nil
		}
		if floatValue != math.Trunc(floatValue) {
			return nil
		}
		if floatValue < float64(math.MinInt) || floatValue > float64(math.MaxInt) {
			return nil
		}
		// float64 can exactly represent integers only in [-2^53, 2^53].
		const maxExactFloatInt = float64(1 << 53)
		if floatValue < -maxExactFloatInt || floatValue > maxExactFloatInt {
			return nil
		}
		intValue := int(floatValue)
		return &intValue
	}
	if uintValue, ok := value.(uint64); ok {
		// Guard int conversion on 32-bit/64-bit architectures.
		if uintValue > uint64(math.MaxInt) {
			return nil
		}
		intValue := int(uintValue)
		return &intValue
	}
	return nil
}

func parseCacheMemoryRestoreOnly(cacheMap map[string]any, entry *CacheMemoryEntry) {
	if restoreOnlyBool, ok := cacheMap["restore-only"].(bool); ok {
		entry.RestoreOnly = restoreOnlyBool
	}
}

func parseCacheMemoryScope(cacheMap map[string]any, entry *CacheMemoryEntry) error {
	if scopeStr, ok := cacheMap["scope"].(string); ok {
		entry.Scope = scopeStr
	}
	if entry.Scope == "" {
		entry.Scope = "workflow"
	}
	if slices.Contains(validCacheMemoryScopes, entry.Scope) {
		return nil
	}
	return fmt.Errorf("invalid cache-memory scope %q: must be one of %v", entry.Scope, validCacheMemoryScopes)
}

func parseCacheMemoryAllowedExtensions(cacheMap map[string]any, entry *CacheMemoryEntry) error {
	allowedExts, exists := cacheMap["allowed-extensions"]
	if !exists {
		return nil
	}
	extArray, ok := allowedExts.([]any)
	if !ok {
		return nil
	}
	entry.AllowedExtensions = make([]string, 0, len(extArray))
	for _, ext := range extArray {
		extStr, ok := ext.(string)
		if !ok {
			continue
		}
		if !isValidFileExtension(extStr) {
			return fmt.Errorf("invalid allowed-extension %q: must start with '.' followed by alphanumeric characters only (e.g. .json)", extStr)
		}
		entry.AllowedExtensions = append(entry.AllowedExtensions, extStr)
	}
	return nil
}

func applyDefaultAllowedExtensions(entry *CacheMemoryEntry) {
	if len(entry.AllowedExtensions) == 0 {
		entry.AllowedExtensions = constants.DefaultAllowedMemoryExtensions
	}
}

// extractCacheMemoryConfig extracts cache-memory configuration from tools section
// Updated to use ToolsConfig instead of map[string]any
func (c *Compiler) extractCacheMemoryConfig(toolsConfig *ToolsConfig) (*CacheMemoryConfig, error) {
	if toolsConfig == nil || toolsConfig.CacheMemory == nil {
		return nil, nil
	}
	cacheLog.Print("Extracting cache-memory configuration from ToolsConfig")
	config := &CacheMemoryConfig{}
	cacheMemoryValue := toolsConfig.CacheMemory.Raw
	if cacheMemoryValue == nil {
		config.Caches = defaultCacheMemoryEntries()
		return config, nil
	}
	if boolValue, ok := cacheMemoryValue.(bool); ok {
		if boolValue {
			config.Caches = defaultCacheMemoryEntries()
		}
		return config, nil
	}
	if cacheArray, ok := cacheMemoryValue.([]any); ok {
		entries, err := parseCacheMemoryEntries(cacheArray)
		if err != nil {
			return nil, err
		}
		config.Caches = entries
		return config, nil
	}
	if configMap, ok := cacheMemoryValue.(map[string]any); ok {
		entry, err := parseCacheMemoryEntry(configMap, "default")
		if err != nil {
			return nil, err
		}
		config.Caches = []CacheMemoryEntry{entry}
		return config, nil
	}

	return nil, nil
}

func defaultCacheMemoryEntries() []CacheMemoryEntry {
	return []CacheMemoryEntry{
		{
			ID:                "default",
			Key:               generateDefaultCacheKey("default"),
			AllowedExtensions: constants.DefaultAllowedMemoryExtensions,
		},
	}
}

func parseCacheMemoryEntries(cacheArray []any) ([]CacheMemoryEntry, error) {
	cacheLog.Printf("Processing cache array with %d entries", len(cacheArray))
	entries := make([]CacheMemoryEntry, 0, len(cacheArray))
	for _, item := range cacheArray {
		cacheMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry, err := parseCacheMemoryEntry(cacheMap, "default")
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := validateNoDuplicateCacheIDs(entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// extractCacheMemoryConfigFromMap is a backward compatibility wrapper for extractCacheMemoryConfig
// extractCacheMemoryConfigFromMap is a backward compatibility wrapper for extractCacheMemoryConfig
// that accepts map[string]any instead of *ToolsConfig. This allows gradual migration of calling code.
func (c *Compiler) extractCacheMemoryConfigFromMap(tools map[string]any) (*CacheMemoryConfig, error) {
	toolsConfig, err := ParseToolsConfig(tools)
	if err != nil {
		return nil, err
	}
	return c.extractCacheMemoryConfig(toolsConfig)
}

// generateCacheSteps generates cache steps for the workflow based on cache configuration
func generateCacheSteps(builder *strings.Builder, data *WorkflowData, verbose bool) {
	if data.Cache == "" {
		return
	}

	cacheLog.Print("Generating cache steps from frontmatter cache configuration")
	builder.WriteString("      # Cache configuration from frontmatter processed below\n")
	caches, err := parseCacheStepConfigs(data.Cache)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse cache configuration: %v\n", err)
		}
		return
	}
	for i, cache := range caches {
		writeCacheStep(builder, cache, i, len(caches))
	}
}

func parseCacheStepConfigs(cacheYAML string) ([]map[string]any, error) {
	var topLevel map[string]any
	if err := yaml.Unmarshal([]byte(cacheYAML), &topLevel); err != nil {
		return nil, err
	}
	cacheConfig, exists := topLevel["cache"]
	if !exists {
		return nil, errors.New("no cache key found in parsed configuration")
	}
	if cacheArray, isArray := cacheConfig.([]any); isArray {
		cacheLog.Printf("Processing %d cache entries (array format)", len(cacheArray))
		return normalizeCacheStepArray(cacheArray), nil
	}
	if cacheMap, isMap := cacheConfig.(map[string]any); isMap {
		cacheLog.Print("Processing single cache entry (object format)")
		return []map[string]any{cacheMap}, nil
	}
	return nil, nil
}

func normalizeCacheStepArray(cacheArray []any) []map[string]any {
	caches := make([]map[string]any, 0, len(cacheArray))
	for _, cacheItem := range cacheArray {
		if cacheMap, ok := cacheItem.(map[string]any); ok {
			caches = append(caches, cacheMap)
		}
	}
	return caches
}

func writeCacheStep(builder *strings.Builder, cache map[string]any, idx int, total int) {
	stepName := resolveCacheStepName(cache, idx, total)
	fmt.Fprintf(builder, "      - name: %s\n", stepName)
	fmt.Fprintf(builder, "        uses: %s\n", getActionPin("actions/cache"))
	builder.WriteString("        with:\n")
	writeCacheStepValue(builder, "key", cache["key"])
	writeCachePath(builder, cache["path"])
	writeCacheRestoreKeys(builder, cache["restore-keys"])
	writeCacheStepValue(builder, "upload-chunk-size", cache["upload-chunk-size"])
	writeCacheStepValue(builder, "fail-on-cache-miss", cache["fail-on-cache-miss"])
	writeCacheStepValue(builder, "lookup-only", cache["lookup-only"])
}

func resolveCacheStepName(cache map[string]any, idx int, total int) string {
	stepName := "Cache"
	if total > 1 {
		stepName = fmt.Sprintf("Cache %d", idx+1)
	}
	if nameStr, ok := cache["name"].(string); ok && nameStr != "" {
		return nameStr
	}
	if keyStr, ok := cache["key"].(string); ok && keyStr != "" {
		return fmt.Sprintf("Cache (%s)", keyStr)
	}
	return stepName
}

func writeCachePath(builder *strings.Builder, path any) {
	if path == nil {
		return
	}
	if pathArray, isArray := path.([]any); isArray {
		builder.WriteString("          path: |\n")
		for _, p := range pathArray {
			fmt.Fprintf(builder, "            %v\n", p)
		}
		return
	}
	fmt.Fprintf(builder, "          path: %v\n", path)
}

func writeCacheRestoreKeys(builder *strings.Builder, restoreKeys any) {
	if restoreKeys == nil {
		return
	}
	if restoreArray, isArray := restoreKeys.([]any); isArray {
		builder.WriteString("          restore-keys: |\n")
		for _, key := range restoreArray {
			fmt.Fprintf(builder, "            %v\n", key)
		}
		return
	}
	fmt.Fprintf(builder, "          restore-keys: %v\n", restoreKeys)
}

func writeCacheStepValue(builder *strings.Builder, key string, value any) {
	if value != nil {
		fmt.Fprintf(builder, "          %s: %v\n", key, value)
	}
}

// generateCacheMemorySteps generates cache setup steps (directory creation, restore, and git init) for the cache-memory configuration.
// Cache-memory provides a simple file share that LLMs can read/write freely.
// Artifact upload is handled separately by generateCacheMemoryArtifactUpload after agent execution.
func generateCacheMemorySteps(builder *strings.Builder, data *WorkflowData) {
	if data.CacheMemoryConfig == nil || len(data.CacheMemoryConfig.Caches) == 0 {
		return
	}

	cacheLog.Printf("Generating cache-memory setup steps for %d caches", len(data.CacheMemoryConfig.Caches))

	builder.WriteString("      # Cache memory file share configuration from frontmatter processed below\n")

	// Use backward-compatible paths only when there's a single cache with ID "default"
	// This maintains compatibility with existing workflows
	useBackwardCompatiblePaths := len(data.CacheMemoryConfig.Caches) == 1 && data.CacheMemoryConfig.Caches[0].ID == "default"

	// Extract GitHub guard policy for integrity-aware cache key generation.
	var githubConfig *GitHubToolConfig
	if data.ParsedTools != nil {
		githubConfig = data.ParsedTools.GitHub
	}
	integrityLevel := cacheIntegrityLevel(githubConfig)
	for i, cache := range data.CacheMemoryConfig.Caches {
		cacheDir := cacheMemoryDirFor(cache.ID)
		restoreStepID := fmt.Sprintf("restore_cache_memory_%d", i)

		// Add step to create cache-memory directory for this cache
		if useBackwardCompatiblePaths {
			// For single default cache, use the original directory for backward compatibility
			builder.WriteString("      - name: Create cache-memory directory\n")
			builder.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/create_cache_memory_dir.sh\"\n")
		} else {
			fmt.Fprintf(builder, "      - name: Create cache-memory directory (%s)\n", cache.ID)
			builder.WriteString("        run: |\n")
			fmt.Fprintf(builder, "          mkdir -p %s\n", cacheDir)
		}

		// Use integrity-aware cache key (includes integrity level + policy hash prefix).
		cacheKey := computeIntegrityCacheKey(cache, githubConfig)

		// Ensure run_id suffix is present (computeIntegrityCacheKey guarantees this,
		// but we check again for clarity and safety).
		runIdSuffix := "-${{ github.run_id }}"
		if !strings.HasSuffix(cacheKey, runIdSuffix) {
			cacheKey = cacheKey + runIdSuffix
		}

		// Generate restore keys based on scope
		// - "workflow" (default): Single restore key with workflow ID (secure)
		// - "repo": Two restore keys - with and without workflow ID (allows cross-workflow sharing)
		var restoreKeys []string

		// Determine scope (default to "workflow" for safety)
		scope := cache.Scope
		if scope == "" {
			scope = "workflow"
		}

		// First restore key: remove the run_id suffix as a single unit (don't split the key)
		// The cacheKey always ends with "-${{ github.run_id }}" (ensured by code above)
		if strings.HasSuffix(cacheKey, runIdSuffix) {
			// Remove the run_id suffix to create the restore key
			restoreKey := strings.TrimSuffix(cacheKey, "${{ github.run_id }}") // Keep the trailing "-"
			restoreKeys = append(restoreKeys, restoreKey)
		} else {
			// Fallback: split on last dash if run_id suffix not found
			// This handles edge cases where the key format might be different
			keyParts := strings.Split(cacheKey, "-")
			if len(keyParts) >= 2 {
				workflowLevelKey := strings.Join(keyParts[:len(keyParts)-1], "-") + "-"
				restoreKeys = append(restoreKeys, workflowLevelKey)
			}
		}

		// For repo scope, add an additional restore key without the workflow ID
		// This allows cache sharing across all workflows in the repository
		if scope == "repo" {
			// Remove both workflow and run_id to create a repo-wide restore key
			// For example: "memory-none-nopolicy-chroma-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}" -> "memory-none-nopolicy-chroma-"
			repoKey := strings.TrimSuffix(cacheKey, "${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}")
			if repoKey != cacheKey && repoKey != "" {
				restoreKeys = append(restoreKeys, repoKey)
			}
		}

		// Step name and action
		// Use actions/cache/restore for restore-only caches or when threat detection is enabled
		// When threat detection is enabled, we only restore the cache and defer saving to a separate job after detection
		// Use actions/cache for normal caches (which auto-saves via post-action)
		threatDetectionEnabled := IsDetectionJobEnabled(data.SafeOutputs)
		useRestoreOnly := cache.RestoreOnly || threatDetectionEnabled

		actionName := "Restore cache-memory file share data"

		if useBackwardCompatiblePaths {
			fmt.Fprintf(builder, "      - name: %s\n", actionName)
		} else {
			fmt.Fprintf(builder, "      - name: %s (%s)\n", actionName, cache.ID)
		}
		fmt.Fprintf(builder, "        id: %s\n", restoreStepID)

		// Use actions/cache/restore@v4 when restore-only or threat detection enabled
		// Use actions/cache@v4 for normal caches
		if useRestoreOnly {
			fmt.Fprintf(builder, "        uses: %s\n", getActionPin("actions/cache/restore"))
		} else {
			fmt.Fprintf(builder, "        uses: %s\n", getActionPin("actions/cache"))
		}
		builder.WriteString("        with:\n")
		fmt.Fprintf(builder, "          key: %s\n", cacheKey)

		// Path - always use the new cache directory format
		fmt.Fprintf(builder, "          path: %s\n", cacheDir)

		builder.WriteString("          restore-keys: |\n")
		for _, key := range restoreKeys {
			fmt.Fprintf(builder, "            %s\n", key)
		}

		// Add git setup step after cache restore.
		// This initialises (or migrates) the git repository used for integrity branching,
		// checks out the current integrity branch, and merges down from higher-integrity branches.
		generateCacheMemoryGitSetupStep(builder, cache, cacheDir, integrityLevel, useBackwardCompatiblePaths)
	}

}

// generateCacheMemoryGitSetupStep emits a pre-agent step that sets up the git-backed integrity
// repository inside the given cache directory. It must run after the cache is restored so that
// any previous git history is available for the merge-down step.
// The step also performs pre-agent security sanitization: it strips execute bits from all
// working-tree files and, when allowed extensions are configured, removes files with
// disallowed extensions before the agent can access them.
func generateCacheMemoryGitSetupStep(builder *strings.Builder, cache CacheMemoryEntry, cacheDir, integrityLevel string, useBackwardCompatiblePaths bool) {
	if useBackwardCompatiblePaths {
		builder.WriteString("      - name: Setup cache-memory git repository\n")
	} else {
		fmt.Fprintf(builder, "      - name: Setup cache-memory git repository (%s)\n", cache.ID)
	}
	builder.WriteString("        env:\n")
	fmt.Fprintf(builder, "          GH_AW_CACHE_DIR: %s\n", cacheDir)
	fmt.Fprintf(builder, "          GH_AW_MIN_INTEGRITY: %s\n", integrityLevel)
	// Pass colon-separated allowed extensions so the setup script can remove disallowed files
	// before the agent runs (pre-agent sanitization). Skip when the list is empty (allow all).
	// Single quotes in the value are escaped ('' in YAML single-quoted scalars) as defense-in-depth,
	// even though isValidFileExtension already rejects values containing single quotes at parse time.
	if len(cache.AllowedExtensions) > 0 {
		escaped := strings.ReplaceAll(strings.Join(cache.AllowedExtensions, ":"), "'", "''")
		fmt.Fprintf(builder, "          GH_AW_ALLOWED_EXTENSIONS: '%s'\n", escaped)
	}
	builder.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/setup_cache_memory_git.sh\"\n")
}

// generateCacheMemoryGitCommitSteps emits post-agent steps that commit agent-written changes
// to the current integrity branch. These steps run after agent execution and before artifact
// upload so that the saved tarball always includes up-to-date git history.
func generateCacheMemoryGitCommitSteps(builder *strings.Builder, data *WorkflowData) {
	if data.CacheMemoryConfig == nil || len(data.CacheMemoryConfig.Caches) == 0 {
		return
	}

	cacheLog.Printf("Generating cache-memory git commit steps for %d caches", len(data.CacheMemoryConfig.Caches))

	useBackwardCompatiblePaths := len(data.CacheMemoryConfig.Caches) == 1 && data.CacheMemoryConfig.Caches[0].ID == "default"

	for _, cache := range data.CacheMemoryConfig.Caches {
		// Skip restore-only caches (nothing to commit)
		if cache.RestoreOnly {
			continue
		}

		cacheDir := cacheMemoryDirFor(cache.ID)

		if useBackwardCompatiblePaths {
			builder.WriteString("      - name: Commit cache-memory changes\n")
		} else {
			fmt.Fprintf(builder, "      - name: Commit cache-memory changes (%s)\n", cache.ID)
		}
		// Run even when agent fails so that partial work is still recorded.
		builder.WriteString("        if: always()\n")
		builder.WriteString("        env:\n")
		fmt.Fprintf(builder, "          GH_AW_CACHE_DIR: %s\n", cacheDir)
		builder.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/commit_cache_memory_git.sh\"\n")
	}
}

// generateCacheMemoryValidation generates validation steps for cache-memory file types
// This should be called after agent execution to validate files before upload/save
func generateCacheMemoryValidation(builder *strings.Builder, data *WorkflowData) {
	if data.CacheMemoryConfig == nil || len(data.CacheMemoryConfig.Caches) == 0 {
		return
	}

	cacheLog.Printf("Generating cache-memory validation steps for %d caches", len(data.CacheMemoryConfig.Caches))

	// Use backward-compatible paths only when there's a single cache with ID "default"
	useBackwardCompatiblePaths := len(data.CacheMemoryConfig.Caches) == 1 && data.CacheMemoryConfig.Caches[0].ID == "default"

	for _, cache := range data.CacheMemoryConfig.Caches {
		// Skip restore-only caches
		if cache.RestoreOnly {
			continue
		}

		// Skip validation step if allowed extensions is empty (means all files are allowed)
		if len(cache.AllowedExtensions) == 0 {
			cacheLog.Printf("Skipping validation step for cache %s (empty allowed-extensions means all files are allowed)", cache.ID)
			continue
		}

		cacheDir := cacheMemoryDirFor(cache.ID)

		// Prepare allowed extensions array for JavaScript
		allowedExtsJSON, _ := json.Marshal(cache.AllowedExtensions) //nolint:jsonmarshalignoredeerror // marshaling a string slice cannot fail

		// Build validation script
		var validationScript strings.Builder
		validationScript.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
		validationScript.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
		validationScript.WriteString("            const { validateMemoryFiles } = require('${{ runner.temp }}/gh-aw/actions/validate_memory_files.cjs');\n")
		fmt.Fprintf(&validationScript, "            const allowedExtensions = %s;\n", allowedExtsJSON)
		fmt.Fprintf(&validationScript, "            const result = validateMemoryFiles('%s', 'cache', allowedExtensions);\n", cacheDir)
		validationScript.WriteString("            if (!result.valid) {\n")
		fmt.Fprintf(&validationScript, "              core.setFailed(`File type validation failed: Found $${result.invalidFiles.length} file(s) with invalid extensions. Only %s are allowed.`);\n", strings.Join(cache.AllowedExtensions, ", "))
		validationScript.WriteString("            }\n")

		// Generate validation step using helper
		stepName := "Validate cache-memory file types"
		if !useBackwardCompatiblePaths {
			stepName = fmt.Sprintf("Validate cache-memory file types (%s)", cache.ID)
		}
		builder.WriteString(generateInlineGitHubScriptStep(stepName, validationScript.String(), "always()", data))
	}
}

// generateCacheMemoryArtifactUpload generates artifact upload steps for cache-memory.
// This should be called after agent execution steps to ensure cache is uploaded after the agent has finished.
// pinAction resolves the upload-artifact action reference; pass c.getActionPin from Compiler methods.
func generateCacheMemoryArtifactUpload(builder *strings.Builder, data *WorkflowData, pinAction func(string) string) {
	if data.CacheMemoryConfig == nil || len(data.CacheMemoryConfig.Caches) == 0 {
		return
	}

	// Only upload artifacts when threat detection is enabled (needed for update_cache_memory job)
	// When threat detection is disabled, cache is saved automatically by actions/cache post-action
	threatDetectionEnabled := IsDetectionJobEnabled(data.SafeOutputs)
	if !threatDetectionEnabled {
		cacheLog.Print("Skipping cache-memory artifact upload (threat detection disabled)")
		return
	}

	cacheLog.Printf("Generating cache-memory artifact upload steps for %d caches", len(data.CacheMemoryConfig.Caches))

	// Use backward-compatible paths only when there's a single cache with ID "default"
	useBackwardCompatiblePaths := len(data.CacheMemoryConfig.Caches) == 1 && data.CacheMemoryConfig.Caches[0].ID == "default"

	// In workflow_call context, apply the per-invocation prefix to avoid artifact name clashes.
	prefix := artifactPrefixExprForDownstreamJob(data)

	for _, cache := range data.CacheMemoryConfig.Caches {
		// Skip restore-only caches
		if cache.RestoreOnly {
			continue
		}

		cacheDir := cacheMemoryDirFor(cache.ID)

		// Add a best-effort git integrity check and reseed step before upload.
		// This prevents upload-artifact from failing on torn/corrupt .git object stores.
		if useBackwardCompatiblePaths {
			builder.WriteString("      - name: Check cache-memory git integrity\n")
		} else {
			fmt.Fprintf(builder, "      - name: Check cache-memory git integrity (%s)\n", cache.ID)
		}
		builder.WriteString("        if: always()\n")
		builder.WriteString("        continue-on-error: true\n")
		builder.WriteString("        env:\n")
		fmt.Fprintf(builder, "          GH_AW_CACHE_DIR: %s\n", cacheDir)
		builder.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/check_cache_memory_git_integrity.sh\"\n")

		// Add upload-artifact step for each cache (runs always)
		if useBackwardCompatiblePaths {
			builder.WriteString("      - name: Upload cache-memory data as artifact\n")
		} else {
			fmt.Fprintf(builder, "      - name: Upload cache-memory data as artifact (%s)\n", cache.ID)
		}
		fmt.Fprintf(builder, "        uses: %s\n", pinAction("actions/upload-artifact"))
		builder.WriteString("        if: always()\n")
		builder.WriteString("        with:\n")
		// Always use the new artifact name and path format, with prefix in workflow_call context
		if useBackwardCompatiblePaths {
			fmt.Fprintf(builder, "          name: %scache-memory\n", prefix)
		} else {
			fmt.Fprintf(builder, "          name: %scache-memory-%s\n", prefix, cache.ID)
		}
		builder.WriteString("          include-hidden-files: true\n")
		fmt.Fprintf(builder, "          path: %s\n", cacheDir)
		// Add retention-days if configured
		if cache.RetentionDays != nil {
			fmt.Fprintf(builder, "          retention-days: %d\n", *cache.RetentionDays)
		}
	}
}

// buildCacheMemoryPromptSection builds a PromptSection for cache memory instructions
// Returns a PromptSection that references a template file with substitutions, or nil if no cache is configured
func buildCacheMemoryPromptSection(config *CacheMemoryConfig) *PromptSection {
	if config == nil || len(config.Caches) == 0 {
		return nil
	}

	// Check if there's only one cache with ID "default" to use singular template
	if len(config.Caches) == 1 && config.Caches[0].ID == "default" {
		cache := config.Caches[0]
		// Trailing slash makes the path look like a directory in prompt context.
		cacheDir := cacheMemoryDirFor(cache.ID) + "/"

		// Build description text
		descriptionText := ""
		if cache.Description != "" {
			descriptionText = cache.Description
		}

		// Build allowed extensions text.
		// When non-empty, add a compact plain-text restriction line.
		// When empty (all extensions allowed), the placeholder is replaced with nothing.
		var allowedExtsText string
		if len(cache.AllowedExtensions) > 0 {
			allowedExtsText = "\nAllowed file extensions: " + strings.Join(cache.AllowedExtensions, ", ") + "."
		}

		cacheLog.Printf("Building cache memory prompt section with env vars: cache_dir=%s, description=%s, allowed_extensions=%v", cacheDir, descriptionText, cache.AllowedExtensions)

		// Return prompt section with template file and environment variables for substitution
		return &PromptSection{
			Content: cacheMemoryPromptFile,
			IsFile:  true,
			EnvVars: map[string]string{
				"GH_AW_CACHE_DIR":          cacheDir,
				"GH_AW_CACHE_DESCRIPTION":  descriptionText,
				"GH_AW_ALLOWED_EXTENSIONS": allowedExtsText,
			},
		}
	}

	// Multiple caches or non-default single cache - use template file with substitutions
	cacheLog.Print("Building cache memory prompt section for multiple caches using template")

	// Build cache list
	var cacheList strings.Builder
	for _, cache := range config.Caches {
		// Trailing slash makes the path look like a directory in prompt context.
		cacheDir := cacheMemoryDirFor(cache.ID) + "/"
		if cache.Description != "" {
			fmt.Fprintf(&cacheList, "- **%s**: `%s` - %s\n", cache.ID, cacheDir, cache.Description)
		} else {
			fmt.Fprintf(&cacheList, "- **%s**: `%s`\n", cache.ID, cacheDir)
		}
	}

	// Build allowed extensions text.
	// Compute the union of all allowed extensions across all caches.
	// When non-empty, add a compact plain-text restriction line.
	// When empty (all extensions allowed for all caches), the placeholder is replaced with nothing.
	allSame := true
	for i := 1; i < len(config.Caches); i++ {
		if len(config.Caches[i].AllowedExtensions) != len(config.Caches[0].AllowedExtensions) {
			allSame = false
			break
		}
		for j, ext := range config.Caches[i].AllowedExtensions {
			if ext != config.Caches[0].AllowedExtensions[j] {
				allSame = false
				break
			}
		}
		if !allSame {
			break
		}
	}

	var extsUnion []string
	if allSame {
		extsUnion = config.Caches[0].AllowedExtensions
	} else {
		extensionSet := make(map[string]struct {
		})
		for _, cache := range config.Caches {
			for _, ext := range cache.AllowedExtensions {
				extensionSet[ext] = struct {
				}{}
			}
		}
		for ext := range extensionSet {
			extsUnion = append(extsUnion, ext)
		}
		sort.Strings(extsUnion)
	}

	var allowedExtsText string
	if len(extsUnion) > 0 {
		allowedExtsText = "\nAllowed file extensions: " + strings.Join(extsUnion, ", ") + "."
	}

	// Build cache examples
	var cacheExamples strings.Builder
	for _, cache := range config.Caches {
		cacheDir := cacheMemoryDirFor(cache.ID)
		fmt.Fprintf(&cacheExamples, "- `%s/notes.txt` - general notes and observations\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/notes.md` - markdown formatted notes\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/preferences.json` - user preferences and settings\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/history.jsonl` - activity history in JSON Lines format\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/data.csv` - tabular data\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/state/` - organized state files in subdirectories (with allowed file types)\n", cacheDir)
	}

	return &PromptSection{
		Content: cacheMemoryPromptMultiFile,
		IsFile:  true,
		EnvVars: map[string]string{
			"GH_AW_CACHE_LIST":         cacheList.String(),
			"GH_AW_ALLOWED_EXTENSIONS": allowedExtsText,
			"GH_AW_CACHE_EXAMPLES":     cacheExamples.String(),
		},
	}
}

// buildUpdateCacheMemoryJob builds a job that updates cache-memory after detection passes
// This job downloads cache-memory artifacts and saves them to GitHub Actions cache
func (c *Compiler) buildUpdateCacheMemoryJob(data *WorkflowData, threatDetectionEnabled bool) (*Job, error) {
	if data.CacheMemoryConfig == nil || len(data.CacheMemoryConfig.Caches) == 0 {
		return nil, nil
	}

	// Only create this job if threat detection is enabled
	// Otherwise, cache is updated automatically by actions/cache post-action
	if !threatDetectionEnabled {
		return nil, nil
	}

	cacheLog.Printf("Building update_cache_memory job for %d caches (threatDetectionEnabled=%v)", len(data.CacheMemoryConfig.Caches), threatDetectionEnabled)

	var steps []string

	// Build steps for each cache
	// In workflow_call context, use the per-invocation prefix from the agent job.
	cacheArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)

	for _, cache := range data.CacheMemoryConfig.Caches {
		// Skip restore-only caches
		if cache.RestoreOnly {
			continue
		}

		// Determine artifact name and cache directory.
		// Apply the workflow_call prefix to ensure we download the correct invocation's artifact.
		cacheDir := cacheMemoryDirFor(cache.ID)
		var artifactName string
		if cache.ID == "default" {
			artifactName = cacheArtifactPrefix + "cache-memory"
		} else {
			artifactName = cacheArtifactPrefix + "cache-memory-" + cache.ID
		}

		// Download artifact step
		var downloadStep strings.Builder
		// Generate a safe step ID from cache ID (replace hyphens with underscores)
		downloadStepID := strings.ReplaceAll("download_cache_"+cache.ID, "-", "_")
		fmt.Fprintf(&downloadStep, "      - name: Download cache-memory artifact (%s)\n", cache.ID)
		fmt.Fprintf(&downloadStep, "        id: %s\n", downloadStepID)
		fmt.Fprintf(&downloadStep, "        uses: %s\n", c.getActionPin("actions/download-artifact"))
		downloadStep.WriteString("        continue-on-error: true\n")
		downloadStep.WriteString("        with:\n")
		fmt.Fprintf(&downloadStep, "          name: %s\n", artifactName)
		fmt.Fprintf(&downloadStep, "          path: %s\n", cacheDir)
		steps = append(steps, downloadStep.String())

		// Check if cache folder exists and is not empty
		var checkStep strings.Builder
		checkStepID := strings.ReplaceAll("check_cache_"+cache.ID, "-", "_")
		fmt.Fprintf(&checkStep, "      - name: Check if cache-memory folder has content (%s)\n", cache.ID)
		fmt.Fprintf(&checkStep, "        id: %s\n", checkStepID)
		checkStep.WriteString("        shell: bash\n")
		checkStep.WriteString("        run: |\n")
		fmt.Fprintf(&checkStep, "          if [ -d \"%s\" ] && [ \"$(ls -A %s 2>/dev/null)\" ]; then\n", cacheDir, cacheDir)
		checkStep.WriteString("            echo \"has_content=true\" >> \"$GITHUB_OUTPUT\"\n")
		checkStep.WriteString("          else\n")
		checkStep.WriteString("            echo \"has_content=false\" >> \"$GITHUB_OUTPUT\"\n")
		checkStep.WriteString("          fi\n")
		steps = append(steps, checkStep.String())

		// Skip validation step if allowed extensions is empty (means all files are allowed)
		if len(cache.AllowedExtensions) == 0 {
			cacheLog.Printf("Skipping validation step for cache %s in update job (empty allowed-extensions means all files are allowed)", cache.ID)
		} else {
			// Prepare allowed extensions array for JavaScript
			allowedExtsJSON, _ := json.Marshal(cache.AllowedExtensions) //nolint:jsonmarshalignoredeerror // marshaling a string slice cannot fail

			// Build validation script
			var validationScript strings.Builder
			validationScript.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
			validationScript.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
			validationScript.WriteString("            const { validateMemoryFiles } = require('${{ runner.temp }}/gh-aw/actions/validate_memory_files.cjs');\n")
			fmt.Fprintf(&validationScript, "            const allowedExtensions = %s;\n", allowedExtsJSON)
			fmt.Fprintf(&validationScript, "            const result = validateMemoryFiles('%s', 'cache', allowedExtensions);\n", cacheDir)
			validationScript.WriteString("            if (!result.valid) {\n")
			fmt.Fprintf(&validationScript, "              core.setFailed(`File type validation failed: Found ${result.invalidFiles.length} file(s) with invalid extensions. Only %s are allowed.`);\n", strings.Join(cache.AllowedExtensions, ", "))
			validationScript.WriteString("            }\n")

			// Generate validation step using helper with condition to only run if cache has content
			stepName := fmt.Sprintf("Validate cache-memory file types (%s)", cache.ID)
			condition := fmt.Sprintf("steps.%s.outputs.has_content == 'true'", checkStepID)
			steps = append(steps, generateInlineGitHubScriptStep(stepName, validationScript.String(), condition, data))
		}

		// Generate cache key using integrity-aware format (matches generateCacheMemorySteps)
		var githubConfig *GitHubToolConfig
		if data.ParsedTools != nil {
			githubConfig = data.ParsedTools.GitHub
		}
		cacheKey := computeIntegrityCacheKey(cache, githubConfig)

		// Ensure run_id suffix is present
		runIdSuffix := "-${{ github.run_id }}"
		if !strings.HasSuffix(cacheKey, runIdSuffix) {
			cacheKey = cacheKey + runIdSuffix
		}

		// Save to cache step - only run if cache has content
		var saveStep strings.Builder
		fmt.Fprintf(&saveStep, "      - name: Save cache-memory to cache (%s)\n", cache.ID)
		fmt.Fprintf(&saveStep, "        if: steps.%s.outputs.has_content == 'true'\n", checkStepID)
		fmt.Fprintf(&saveStep, "        uses: %s\n", getActionPin("actions/cache/save"))
		saveStep.WriteString("        with:\n")
		fmt.Fprintf(&saveStep, "          key: %s\n", cacheKey)
		fmt.Fprintf(&saveStep, "          path: %s\n", cacheDir)
		steps = append(steps, saveStep.String())
	}

	// If no writable caches, return nil
	if len(steps) == 0 {
		return nil, nil
	}

	// Add setup step to copy scripts at the beginning
	var setupSteps []string
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		// For dev mode (local action path), checkout the actions folder first
		setupSteps = append(setupSteps, c.generateCheckoutActionsFolder(data)...)

		// Cache restore job doesn't need project support
		// Cache job depends on agent job; reuse the agent's trace ID so all jobs share one OTLP trace
		cacheTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		cacheParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		setupSteps = append(setupSteps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, cacheTraceID, cacheParentSpanID)...)
	}

	// Prepend setup steps to all cache steps
	steps = append(setupSteps, steps...)

	// Job condition: run only if detection job succeeded (no threats found),
	// AND the agent job succeeded (do not persist cache when agent failed or was skipped).
	// Using always() so this condition is evaluated even if an upstream job is skipped/failed.
	agentSucceeded := BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("success"),
	)
	jobCondition := RenderCondition(BuildAnd(BuildAnd(BuildFunctionCall("always"), buildDetectionSuccessCondition()), agentSucceeded))

	// Set up permissions for the cache update job
	// If using local actions (dev mode without action-tag), we need contents: read to checkout the actions folder
	permissions := NewPermissionsEmpty().RenderToYAML() // Default: no special permissions needed
	if setupActionRef != "" && len(c.generateCheckoutActionsFolder(data)) > 0 {
		// Need contents: read to checkout the actions folder
		perms := NewPermissionsContentsRead()
		permissions = perms.RenderToYAML()
	}

	// Set GH_AW_WORKFLOW_ID_SANITIZED so cache keys match those used in the agent job
	var jobEnv map[string]string
	if data.WorkflowID != "" {
		jobEnv = map[string]string{
			"GH_AW_WORKFLOW_ID_SANITIZED": SanitizeWorkflowIDForCacheKey(data.WorkflowID),
		}
	}

	job := &Job{
		Name:        "update_cache_memory",
		DisplayName: "", // No display name - job ID is sufficient
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		If:          jobCondition,
		Permissions: permissions,
		Needs:       []string{string(constants.AgentJobName), string(constants.DetectionJobName), string(constants.ActivationJobName)},
		Env:         jobEnv,
		Steps:       steps,
	}

	return job, nil
}
