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

// buildCacheRestoreKeys derives the ordered list of restore-keys for a cache entry.
// The primary key (without the run_id suffix) is always included.
// For "repo" scope, a second key that also strips the workflow ID is appended to allow
// cross-workflow cache sharing.
//
// cacheKey must be the fully-formed primary key (e.g. as returned by
// computeIntegrityCacheKey) and scope is the cache entry's scope field
// ("workflow" or "repo"; empty is treated as "workflow").
func buildCacheRestoreKeys(cacheKey, scope string) []string {
	if scope == "" {
		scope = "workflow"
	}
	const runIDSuffix = "-${{ github.run_id }}"

	var keys []string
	if strings.HasSuffix(cacheKey, runIDSuffix) {
		keys = append(keys, strings.TrimSuffix(cacheKey, "${{ github.run_id }}"))
	} else {
		parts := strings.Split(cacheKey, "-")
		if len(parts) >= 2 {
			keys = append(keys, strings.Join(parts[:len(parts)-1], "-")+"-")
		}
	}

	if scope == "repo" {
		repoKey := strings.TrimSuffix(cacheKey, "${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}")
		if repoKey != cacheKey && repoKey != "" {
			keys = append(keys, repoKey)
		}
	}
	return keys
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
	useBackwardCompatiblePaths := len(data.CacheMemoryConfig.Caches) == 1 && data.CacheMemoryConfig.Caches[0].ID == "default"
	githubConfig := cacheMemoryGitHubConfig(data)
	integrityLevel := cacheIntegrityLevel(githubConfig)
	for i, cache := range data.CacheMemoryConfig.Caches {
		generateCacheMemorySetupForCache(builder, data, cache, i, githubConfig, integrityLevel, useBackwardCompatiblePaths)
	}
}

func cacheMemoryGitHubConfig(data *WorkflowData) *GitHubToolConfig {
	if data.ParsedTools != nil {
		return data.ParsedTools.GitHub
	}
	return nil
}

func generateCacheMemorySetupForCache(builder *strings.Builder, data *WorkflowData, cache CacheMemoryEntry, index int, githubConfig *GitHubToolConfig, integrityLevel string, useBackwardCompatiblePaths bool) {
	cacheDir := cacheMemoryDirFor(cache.ID)
	generateCacheMemoryDirectoryStep(builder, cache, cacheDir, useBackwardCompatiblePaths)
	cacheKey := ensureCacheKeyRunID(computeIntegrityCacheKey(cache, githubConfig))
	restoreKeys := buildCacheRestoreKeys(cacheKey, cache.Scope)
	restoreStepID := fmt.Sprintf("restore_cache_memory_%d", index)
	generateCacheMemoryRestoreStep(builder, data, cache, cacheDir, restoreStepID, cacheKey, restoreKeys, useBackwardCompatiblePaths)
	generateCacheMemoryGitSetupStep(builder, cache, cacheDir, integrityLevel, useBackwardCompatiblePaths)
}

func generateCacheMemoryDirectoryStep(builder *strings.Builder, cache CacheMemoryEntry, cacheDir string, useBackwardCompatiblePaths bool) {
	if useBackwardCompatiblePaths {
		builder.WriteString("      - name: Create cache-memory directory\n")
		builder.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/create_cache_memory_dir.sh\"\n")
		return
	}
	fmt.Fprintf(builder, "      - name: Create cache-memory directory (%s)\n", cache.ID)
	builder.WriteString("        run: |\n")
	fmt.Fprintf(builder, "          mkdir -p %s\n", cacheDir)
}

func generateCacheMemoryRestoreStep(builder *strings.Builder, data *WorkflowData, cache CacheMemoryEntry, cacheDir, restoreStepID, cacheKey string, restoreKeys []string, useBackwardCompatiblePaths bool) {
	useRestoreOnly := cache.RestoreOnly || IsDetectionJobEnabled(data.SafeOutputs)
	if useBackwardCompatiblePaths {
		builder.WriteString("      - name: Restore cache-memory file share data\n")
	} else {
		fmt.Fprintf(builder, "      - name: Restore cache-memory file share data (%s)\n", cache.ID)
	}
	fmt.Fprintf(builder, "        id: %s\n", restoreStepID)
	if useRestoreOnly {
		fmt.Fprintf(builder, "        uses: %s\n", getActionPin("actions/cache/restore"))
	} else {
		fmt.Fprintf(builder, "        uses: %s\n", getActionPin("actions/cache"))
	}
	builder.WriteString("        with:\n")
	fmt.Fprintf(builder, "          key: %s\n", cacheKey)
	fmt.Fprintf(builder, "          path: %s\n", cacheDir)
	builder.WriteString("          restore-keys: |\n")
	for _, key := range restoreKeys {
		fmt.Fprintf(builder, "            %s\n", key)
	}
}

func ensureCacheKeyRunID(cacheKey string) string {
	runIdSuffix := "-${{ github.run_id }}"
	if !strings.HasSuffix(cacheKey, runIdSuffix) {
		return cacheKey + runIdSuffix
	}
	return cacheKey
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
	if !IsDetectionJobEnabled(data.SafeOutputs) {
		cacheLog.Print("Skipping cache-memory artifact upload (threat detection disabled)")
		return
	}
	cacheLog.Printf("Generating cache-memory artifact upload steps for %d caches", len(data.CacheMemoryConfig.Caches))
	useBackwardCompatiblePaths := len(data.CacheMemoryConfig.Caches) == 1 && data.CacheMemoryConfig.Caches[0].ID == "default"
	prefix := artifactPrefixExprForDownstreamJob(data)
	for _, cache := range data.CacheMemoryConfig.Caches {
		if cache.RestoreOnly {
			continue
		}
		generateCacheMemoryArtifactUploadForCache(builder, cache, prefix, useBackwardCompatiblePaths, pinAction)
	}
}

func generateCacheMemoryArtifactUploadForCache(builder *strings.Builder, cache CacheMemoryEntry, prefix string, useBackwardCompatiblePaths bool, pinAction func(string) string) {
	cacheDir := cacheMemoryDirFor(cache.ID)
	generateCacheMemoryIntegrityCheckStep(builder, cache, cacheDir, useBackwardCompatiblePaths)
	if useBackwardCompatiblePaths {
		builder.WriteString("      - name: Upload cache-memory data as artifact\n")
	} else {
		fmt.Fprintf(builder, "      - name: Upload cache-memory data as artifact (%s)\n", cache.ID)
	}
	fmt.Fprintf(builder, "        uses: %s\n", pinAction("actions/upload-artifact"))
	builder.WriteString("        if: always()\n")
	builder.WriteString("        with:\n")
	if useBackwardCompatiblePaths {
		fmt.Fprintf(builder, "          name: %scache-memory\n", prefix)
	} else {
		fmt.Fprintf(builder, "          name: %scache-memory-%s\n", prefix, cache.ID)
	}
	builder.WriteString("          include-hidden-files: true\n")
	fmt.Fprintf(builder, "          path: %s\n", cacheDir)
	if cache.RetentionDays != nil {
		fmt.Fprintf(builder, "          retention-days: %d\n", *cache.RetentionDays)
	}
}

func generateCacheMemoryIntegrityCheckStep(builder *strings.Builder, cache CacheMemoryEntry, cacheDir string, useBackwardCompatiblePaths bool) {
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
}

// buildCacheMemoryPromptSection builds a PromptSection for cache memory instructions
// Returns a PromptSection that references a template file with substitutions, or nil if no cache is configured
func buildCacheMemoryPromptSection(config *CacheMemoryConfig) *PromptSection {
	if config == nil || len(config.Caches) == 0 {
		return nil
	}
	if len(config.Caches) == 1 && config.Caches[0].ID == "default" {
		return buildDefaultCacheMemoryPromptSection(config.Caches[0])
	}
	cacheLog.Print("Building cache memory prompt section for multiple caches using template")
	return &PromptSection{
		Content: cacheMemoryPromptMultiFile,
		IsFile:  true,
		EnvVars: map[string]string{
			"GH_AW_CACHE_LIST":         buildCacheMemoryPromptCacheList(config.Caches),
			"GH_AW_ALLOWED_EXTENSIONS": buildCacheMemoryPromptAllowedExtensions(config.Caches),
			"GH_AW_CACHE_EXAMPLES":     buildCacheMemoryPromptExamples(config.Caches),
		},
	}
}

func buildDefaultCacheMemoryPromptSection(cache CacheMemoryEntry) *PromptSection {
	cacheDir := cacheMemoryDirFor(cache.ID) + "/"
	descriptionText := cache.Description
	allowedExtsText := formatCacheMemoryAllowedExtensions(cache.AllowedExtensions)
	cacheLog.Printf("Building cache memory prompt section with env vars: cache_dir=%s, description=%s, allowed_extensions=%v", cacheDir, descriptionText, cache.AllowedExtensions)
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

func buildCacheMemoryPromptCacheList(caches []CacheMemoryEntry) string {
	var cacheList strings.Builder
	for _, cache := range caches {
		cacheDir := cacheMemoryDirFor(cache.ID) + "/"
		if cache.Description != "" {
			fmt.Fprintf(&cacheList, "- **%s**: `%s` - %s\n", cache.ID, cacheDir, cache.Description)
		} else {
			fmt.Fprintf(&cacheList, "- **%s**: `%s`\n", cache.ID, cacheDir)
		}
	}
	return cacheList.String()
}

func buildCacheMemoryPromptAllowedExtensions(caches []CacheMemoryEntry) string {
	return formatCacheMemoryAllowedExtensions(cacheMemoryAllowedExtensionsUnion(caches))
}

func cacheMemoryAllowedExtensionsUnion(caches []CacheMemoryEntry) []string {
	allSame := true
	for i := 1; i < len(caches); i++ {
		if len(caches[i].AllowedExtensions) != len(caches[0].AllowedExtensions) {
			allSame = false
			break
		}
		for j, ext := range caches[i].AllowedExtensions {
			if ext != caches[0].AllowedExtensions[j] {
				allSame = false
				break
			}
		}
		if !allSame {
			break
		}
	}
	if allSame {
		return caches[0].AllowedExtensions
	}
	extensionSet := make(map[string]struct {
	})
	for _, cache := range caches {
		for _, ext := range cache.AllowedExtensions {
			extensionSet[ext] = struct {
			}{}
		}
	}
	extsUnion := make([]string, 0, len(extensionSet))
	for ext := range extensionSet {
		extsUnion = append(extsUnion, ext)
	}
	sort.Strings(extsUnion)
	return extsUnion
}

func formatCacheMemoryAllowedExtensions(extensions []string) string {
	if len(extensions) == 0 {
		return ""
	}
	return "\nAllowed file extensions: " + strings.Join(extensions, ", ") + "."
}

func buildCacheMemoryPromptExamples(caches []CacheMemoryEntry) string {
	var cacheExamples strings.Builder
	for _, cache := range caches {
		cacheDir := cacheMemoryDirFor(cache.ID)
		fmt.Fprintf(&cacheExamples, "- `%s/notes.txt` - general notes and observations\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/notes.md` - markdown formatted notes\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/preferences.json` - user preferences and settings\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/history.jsonl` - activity history in JSON Lines format\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/data.csv` - tabular data\n", cacheDir)
		fmt.Fprintf(&cacheExamples, "- `%s/state/` - organized state files in subdirectories (with allowed file types)\n", cacheDir)
	}
	return cacheExamples.String()
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
	cacheArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	for _, cache := range data.CacheMemoryConfig.Caches {
		if cache.RestoreOnly {
			continue
		}
		steps = c.appendUpdateCacheMemorySteps(steps, data, cache, cacheArtifactPrefix)
	}
	if len(steps) == 0 {
		return nil, nil
	}
	setupActionRef, setupSteps := c.buildUpdateCacheMemorySetupSteps(data)
	steps = append(setupSteps, steps...)
	return &Job{
		Name:        updateCacheMemoryJobName,
		DisplayName: "", // No display name - job ID is sufficient
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		If:          buildUpdateCacheMemoryJobCondition(),
		Permissions: c.updateCacheMemoryPermissions(data, setupActionRef),
		Needs:       []string{string(constants.AgentJobName), string(constants.DetectionJobName), string(constants.ActivationJobName)},
		Env:         updateCacheMemoryJobEnv(data),
		Steps:       steps,
	}, nil
}

func (c *Compiler) appendUpdateCacheMemorySteps(steps []string, data *WorkflowData, cache CacheMemoryEntry, artifactPrefix string) []string {
	cacheDir := cacheMemoryDirFor(cache.ID)
	checkStepID := strings.ReplaceAll("check_cache_"+cache.ID, "-", "_")
	steps = append(steps, c.buildUpdateCacheMemoryDownloadStep(cache, cacheDir, artifactPrefix))
	steps = append(steps, buildUpdateCacheMemoryCheckStep(cache, cacheDir, checkStepID))
	steps = appendUpdateCacheMemoryValidationStep(steps, data, cache, cacheDir, checkStepID)
	githubConfig := cacheMemoryGitHubConfig(data)
	cacheKey := ensureCacheKeyRunID(computeIntegrityCacheKey(cache, githubConfig))
	return append(steps, buildUpdateCacheMemorySaveStep(cache, cacheDir, checkStepID, cacheKey))
}

func (c *Compiler) buildUpdateCacheMemoryDownloadStep(cache CacheMemoryEntry, cacheDir, artifactPrefix string) string {
	artifactName := artifactPrefix + "cache-memory"
	if cache.ID != "default" {
		artifactName = artifactPrefix + "cache-memory-" + cache.ID
	}
	downloadStepID := strings.ReplaceAll("download_cache_"+cache.ID, "-", "_")
	var downloadStep strings.Builder
	fmt.Fprintf(&downloadStep, "      - name: Download cache-memory artifact (%s)\n", cache.ID)
	fmt.Fprintf(&downloadStep, "        id: %s\n", downloadStepID)
	fmt.Fprintf(&downloadStep, "        uses: %s\n", c.getActionPin("actions/download-artifact"))
	downloadStep.WriteString("        continue-on-error: true\n")
	downloadStep.WriteString("        with:\n")
	fmt.Fprintf(&downloadStep, "          name: %s\n", artifactName)
	fmt.Fprintf(&downloadStep, "          path: %s\n", cacheDir)
	return downloadStep.String()
}

func buildUpdateCacheMemoryCheckStep(cache CacheMemoryEntry, cacheDir, checkStepID string) string {
	var checkStep strings.Builder
	fmt.Fprintf(&checkStep, "      - name: Check if cache-memory folder has content (%s)\n", cache.ID)
	fmt.Fprintf(&checkStep, "        id: %s\n", checkStepID)
	checkStep.WriteString("        shell: bash\n")
	checkStep.WriteString("        run: |\n")
	fmt.Fprintf(&checkStep, "          if [ -d \"%s\" ] && [ \"$(ls -A %s 2>/dev/null)\" ]; then\n", cacheDir, cacheDir)
	checkStep.WriteString("            echo \"has_content=true\" >> \"$GITHUB_OUTPUT\"\n")
	checkStep.WriteString("          else\n")
	checkStep.WriteString("            echo \"has_content=false\" >> \"$GITHUB_OUTPUT\"\n")
	checkStep.WriteString("          fi\n")
	return checkStep.String()
}

func appendUpdateCacheMemoryValidationStep(steps []string, data *WorkflowData, cache CacheMemoryEntry, cacheDir, checkStepID string) []string {
	if len(cache.AllowedExtensions) == 0 {
		cacheLog.Printf("Skipping validation step for cache %s in update job (empty allowed-extensions means all files are allowed)", cache.ID)
		return steps
	}
	allowedExtsJSON, _ := json.Marshal(cache.AllowedExtensions) //nolint:jsonmarshalignoredeerror // marshaling a string slice cannot fail
	var validationScript strings.Builder
	validationScript.WriteString("            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');\n")
	validationScript.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	validationScript.WriteString("            const { validateMemoryFiles } = require('${{ runner.temp }}/gh-aw/actions/validate_memory_files.cjs');\n")
	fmt.Fprintf(&validationScript, "            const allowedExtensions = %s;\n", allowedExtsJSON)
	fmt.Fprintf(&validationScript, "            const result = validateMemoryFiles('%s', 'cache', allowedExtensions);\n", cacheDir)
	validationScript.WriteString("            if (!result.valid) {\n")
	fmt.Fprintf(&validationScript, "              core.setFailed(`File type validation failed: Found ${result.invalidFiles.length} file(s) with invalid extensions. Only %s are allowed.`);\n", strings.Join(cache.AllowedExtensions, ", "))
	validationScript.WriteString("            }\n")
	stepName := fmt.Sprintf("Validate cache-memory file types (%s)", cache.ID)
	condition := fmt.Sprintf("steps.%s.outputs.has_content == 'true'", checkStepID)
	return append(steps, generateInlineGitHubScriptStep(stepName, validationScript.String(), condition, data))
}

func buildUpdateCacheMemorySaveStep(cache CacheMemoryEntry, cacheDir, checkStepID, cacheKey string) string {
	var saveStep strings.Builder
	fmt.Fprintf(&saveStep, "      - name: Save cache-memory to cache (%s)\n", cache.ID)
	fmt.Fprintf(&saveStep, "        if: steps.%s.outputs.has_content == 'true'\n", checkStepID)
	fmt.Fprintf(&saveStep, "        uses: %s\n", getActionPin("actions/cache/save"))
	saveStep.WriteString("        with:\n")
	fmt.Fprintf(&saveStep, "          key: %s\n", cacheKey)
	fmt.Fprintf(&saveStep, "          path: %s\n", cacheDir)
	return saveStep.String()
}

func (c *Compiler) buildUpdateCacheMemorySetupSteps(data *WorkflowData) (string, []string) {
	var setupSteps []string
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		setupSteps = append(setupSteps, c.generateCheckoutActionsFolder(data)...)
		cacheTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		cacheParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		setupSteps = append(setupSteps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, cacheTraceID, cacheParentSpanID)...)
	}
	return setupActionRef, setupSteps
}

func buildUpdateCacheMemoryJobCondition() string {
	agentSucceeded := BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("success"),
	)
	return RenderCondition(BuildAnd(BuildAnd(BuildFunctionCall("always"), buildDetectionSuccessCondition()), agentSucceeded))
}

func (c *Compiler) updateCacheMemoryPermissions(data *WorkflowData, setupActionRef string) string {
	if setupActionRef != "" && len(c.generateCheckoutActionsFolder(data)) > 0 {
		return NewPermissionsContentsRead().RenderToYAML()
	}
	return NewPermissionsEmpty().RenderToYAML()
}

func updateCacheMemoryJobEnv(data *WorkflowData) map[string]string {
	if data.WorkflowID == "" {
		return nil
	}
	return map[string]string{
		"GH_AW_WORKFLOW_ID_SANITIZED": SanitizeWorkflowIDForCacheKey(data.WorkflowID),
	}
}
