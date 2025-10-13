package workflow

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

// CacheMemoryConfig holds configuration for cache-memory functionality
type CacheMemoryConfig struct {
	Enabled       bool                 `yaml:"enabled,omitempty"`        // whether cache-memory is enabled (for single cache config)
	Key           string               `yaml:"key,omitempty"`            // custom cache key (for single cache config)
	RetentionDays *int                 `yaml:"retention-days,omitempty"` // retention days for upload-artifact action (for single cache config)
	Caches        []CacheMemoryEntry   `yaml:"caches,omitempty"`         // multiple cache configurations
}

// CacheMemoryEntry represents a single cache-memory configuration
type CacheMemoryEntry struct {
	ID            string `yaml:"id"`                       // cache identifier (required for array notation)
	Key           string `yaml:"key,omitempty"`            // custom cache key
	RetentionDays *int   `yaml:"retention-days,omitempty"` // retention days for upload-artifact action
}

// extractCacheMemoryConfig extracts cache-memory configuration from tools section
func (c *Compiler) extractCacheMemoryConfig(tools map[string]any) *CacheMemoryConfig {
	cacheMemoryValue, exists := tools["cache-memory"]
	if !exists {
		return nil
	}

	config := &CacheMemoryConfig{}

	// Handle nil value (simple enable with defaults) - same as true
	// This handles the case where cache-memory: is specified without a value
	if cacheMemoryValue == nil {
		config.Enabled = true
		config.Key = "memory-${{ github.workflow }}-${{ github.run_id }}"
		return config
	}

	// Handle boolean value (simple enable/disable)
	if boolValue, ok := cacheMemoryValue.(bool); ok {
		config.Enabled = boolValue
		if config.Enabled {
			// Set defaults
			config.Key = "memory-${{ github.workflow }}-${{ github.run_id }}"
		}
		return config
	}

	// Handle array of cache configurations
	if cacheArray, ok := cacheMemoryValue.([]any); ok {
		config.Caches = make([]CacheMemoryEntry, 0, len(cacheArray))
		for _, item := range cacheArray {
			if cacheMap, ok := item.(map[string]any); ok {
				entry := CacheMemoryEntry{}
				
				// ID is required for array notation
				if id, exists := cacheMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						entry.ID = idStr
					}
				}
				// Use "default" if no ID specified
				if entry.ID == "" {
					entry.ID = "default"
				}

				// Parse custom key
				if key, exists := cacheMap["key"]; exists {
					if keyStr, ok := key.(string); ok {
						entry.Key = keyStr
						// Automatically append -${{ github.run_id }} if the key doesn't already end with it
						runIdSuffix := "-${{ github.run_id }}"
						if !strings.HasSuffix(entry.Key, runIdSuffix) {
							entry.Key = entry.Key + runIdSuffix
						}
					}
				}
				// Set default key if not specified
				if entry.Key == "" {
					entry.Key = fmt.Sprintf("memory-%s-${{ github.workflow }}-${{ github.run_id }}", entry.ID)
				}

				// Parse retention days
				if retentionDays, exists := cacheMap["retention-days"]; exists {
					if retentionDaysInt, ok := retentionDays.(int); ok {
						entry.RetentionDays = &retentionDaysInt
					} else if retentionDaysFloat, ok := retentionDays.(float64); ok {
						retentionDaysIntValue := int(retentionDaysFloat)
						entry.RetentionDays = &retentionDaysIntValue
					} else if retentionDaysUint64, ok := retentionDays.(uint64); ok {
						retentionDaysIntValue := int(retentionDaysUint64)
						entry.RetentionDays = &retentionDaysIntValue
					}
				}

				config.Caches = append(config.Caches, entry)
			}
		}
		return config
	}

	// Handle object configuration (single cache, backward compatible)
	if configMap, ok := cacheMemoryValue.(map[string]any); ok {
		config.Enabled = true

		// Set defaults
		config.Key = "memory-${{ github.workflow }}-${{ github.run_id }}"

		// Parse custom key
		if key, exists := configMap["key"]; exists {
			if keyStr, ok := key.(string); ok {
				config.Key = keyStr
				// Automatically append -${{ github.run_id }} if the key doesn't already end with it
				runIdSuffix := "-${{ github.run_id }}"
				if !strings.HasSuffix(config.Key, runIdSuffix) {
					config.Key = config.Key + runIdSuffix
				}
			}
		}

		// Parse retention days
		if retentionDays, exists := configMap["retention-days"]; exists {
			if retentionDaysInt, ok := retentionDays.(int); ok {
				config.RetentionDays = &retentionDaysInt
			} else if retentionDaysFloat, ok := retentionDays.(float64); ok {
				retentionDaysIntValue := int(retentionDaysFloat)
				config.RetentionDays = &retentionDaysIntValue
			} else if retentionDaysUint64, ok := retentionDays.(uint64); ok {
				retentionDaysIntValue := int(retentionDaysUint64)
				config.RetentionDays = &retentionDaysIntValue
			}
		}

		return config
	}

	return nil
}

// generateCacheSteps generates cache steps for the workflow based on cache configuration
func generateCacheSteps(builder *strings.Builder, data *WorkflowData, verbose bool) {
	if data.Cache == "" {
		return
	}

	// Add comment indicating cache configuration was processed
	builder.WriteString("      # Cache configuration from frontmatter processed below\n")

	// Parse cache configuration to determine if it's a single cache or array
	var caches []map[string]any

	// Try to parse the cache YAML string back to determine structure
	var topLevel map[string]any
	if err := yaml.Unmarshal([]byte(data.Cache), &topLevel); err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to parse cache configuration: %v\n", err)
		}
		return
	}

	// Extract the cache section from the top-level map
	cacheConfig, exists := topLevel["cache"]
	if !exists {
		if verbose {
			fmt.Printf("Warning: No cache key found in parsed configuration\n")
		}
		return
	}

	// Handle both single cache object and array of caches
	if cacheArray, isArray := cacheConfig.([]any); isArray {
		// Multiple caches
		for _, cacheItem := range cacheArray {
			if cacheMap, ok := cacheItem.(map[string]any); ok {
				caches = append(caches, cacheMap)
			}
		}
	} else if cacheMap, isMap := cacheConfig.(map[string]any); isMap {
		// Single cache
		caches = append(caches, cacheMap)
	}

	// Generate cache steps
	for i, cache := range caches {
		stepName := "Cache"
		if len(caches) > 1 {
			stepName = fmt.Sprintf("Cache %d", i+1)
		}
		if key, hasKey := cache["key"]; hasKey {
			if keyStr, ok := key.(string); ok && keyStr != "" {
				stepName = fmt.Sprintf("Cache (%s)", keyStr)
			}
		}

		fmt.Fprintf(builder, "      - name: %s\n", stepName)
		builder.WriteString("        uses: actions/cache@v4\n")
		builder.WriteString("        with:\n")

		// Add required cache parameters
		if key, hasKey := cache["key"]; hasKey {
			fmt.Fprintf(builder, "          key: %v\n", key)
		}
		if path, hasPath := cache["path"]; hasPath {
			if pathArray, isArray := path.([]any); isArray {
				builder.WriteString("          path: |\n")
				for _, p := range pathArray {
					fmt.Fprintf(builder, "            %v\n", p)
				}
			} else {
				fmt.Fprintf(builder, "          path: %v\n", path)
			}
		}

		// Add optional cache parameters
		if restoreKeys, hasRestoreKeys := cache["restore-keys"]; hasRestoreKeys {
			if restoreArray, isArray := restoreKeys.([]any); isArray {
				builder.WriteString("          restore-keys: |\n")
				for _, key := range restoreArray {
					fmt.Fprintf(builder, "            %v\n", key)
				}
			} else {
				fmt.Fprintf(builder, "          restore-keys: %v\n", restoreKeys)
			}
		}
		if uploadChunkSize, hasSize := cache["upload-chunk-size"]; hasSize {
			fmt.Fprintf(builder, "          upload-chunk-size: %v\n", uploadChunkSize)
		}
		if failOnMiss, hasFail := cache["fail-on-cache-miss"]; hasFail {
			fmt.Fprintf(builder, "          fail-on-cache-miss: %v\n", failOnMiss)
		}
		if lookupOnly, hasLookup := cache["lookup-only"]; hasLookup {
			fmt.Fprintf(builder, "          lookup-only: %v\n", lookupOnly)
		}
	}
}

// generateCacheMemorySteps generates cache steps for the cache-memory configuration
// Cache-memory provides a simple file share that LLMs can read/write freely
func generateCacheMemorySteps(builder *strings.Builder, data *WorkflowData) {
	if data.CacheMemoryConfig == nil {
		return
	}

	// Handle multiple caches (array notation)
	if len(data.CacheMemoryConfig.Caches) > 0 {
		builder.WriteString("      # Cache memory file share configuration from frontmatter processed below\n")
		
		for _, cache := range data.CacheMemoryConfig.Caches {
			cacheDir := fmt.Sprintf("/tmp/gh-aw/cache-memory/%s", cache.ID)
			
			// Add step to create cache-memory directory for this cache
			builder.WriteString(fmt.Sprintf("      - name: Create cache-memory directory (%s)\n", cache.ID))
			builder.WriteString("        run: |\n")
			builder.WriteString(fmt.Sprintf("          mkdir -p %s\n", cacheDir))

			cacheKey := cache.Key
			if cacheKey == "" {
				cacheKey = fmt.Sprintf("memory-%s-${{ github.workflow }}-${{ github.run_id }}", cache.ID)
			}

			// Automatically append -${{ github.run_id }} if the key doesn't already end with it
			runIdSuffix := "-${{ github.run_id }}"
			if !strings.HasSuffix(cacheKey, runIdSuffix) {
				cacheKey = cacheKey + runIdSuffix
			}

			// Generate restore keys automatically by splitting the cache key on '-'
			var restoreKeys []string
			keyParts := strings.Split(cacheKey, "-")
			for i := len(keyParts) - 1; i > 0; i-- {
				restoreKey := strings.Join(keyParts[:i], "-") + "-"
				restoreKeys = append(restoreKeys, restoreKey)
			}

			builder.WriteString(fmt.Sprintf("      - name: Cache memory file share data (%s)\n", cache.ID))
			builder.WriteString("        uses: actions/cache@v4\n")
			builder.WriteString("        with:\n")
			fmt.Fprintf(builder, "          key: %s\n", cacheKey)
			fmt.Fprintf(builder, "          path: %s\n", cacheDir)
			builder.WriteString("          restore-keys: |\n")
			for _, key := range restoreKeys {
				fmt.Fprintf(builder, "            %s\n", key)
			}

			// Add upload-artifact step for each cache (runs always)
			builder.WriteString(fmt.Sprintf("      - name: Upload cache-memory data as artifact (%s)\n", cache.ID))
			builder.WriteString("        uses: actions/upload-artifact@v4\n")
			builder.WriteString("        with:\n")
			fmt.Fprintf(builder, "          name: cache-memory-%s\n", cache.ID)
			fmt.Fprintf(builder, "          path: %s\n", cacheDir)
			// Add retention-days if configured
			if cache.RetentionDays != nil {
				fmt.Fprintf(builder, "          retention-days: %d\n", *cache.RetentionDays)
			}
		}
		return
	}

	// Handle single cache (backward compatible with enabled field)
	if !data.CacheMemoryConfig.Enabled {
		return
	}

	// Add comment indicating cache-memory configuration was processed
	builder.WriteString("      # Cache memory file share configuration from frontmatter processed below\n")

	// Add step to create cache-memory directory
	builder.WriteString("      - name: Create cache-memory directory\n")
	builder.WriteString("        run: |\n")
	WriteShellScriptToYAML(builder, createCacheMemoryDirScript, "          ")

	// Use the parsed configuration
	cacheKey := data.CacheMemoryConfig.Key
	if cacheKey == "" {
		cacheKey = "memory-${{ github.workflow }}-${{ github.run_id }}"
	}

	// Automatically append -${{ github.run_id }} if the key doesn't already end with it
	runIdSuffix := "-${{ github.run_id }}"
	if !strings.HasSuffix(cacheKey, runIdSuffix) {
		cacheKey = cacheKey + runIdSuffix
	}

	// Generate restore keys automatically by splitting the cache key on '-'
	// This creates a progressive fallback hierarchy
	var restoreKeys []string
	keyParts := strings.Split(cacheKey, "-")
	for i := len(keyParts) - 1; i > 0; i-- {
		restoreKey := strings.Join(keyParts[:i], "-") + "-"
		restoreKeys = append(restoreKeys, restoreKey)
	}

	builder.WriteString("      - name: Cache memory file share data\n")
	builder.WriteString("        uses: actions/cache@v4\n")
	builder.WriteString("        with:\n")
	fmt.Fprintf(builder, "          key: %s\n", cacheKey)
	builder.WriteString("          path: /tmp/gh-aw/cache-memory\n")
	builder.WriteString("          restore-keys: |\n")
	for _, key := range restoreKeys {
		fmt.Fprintf(builder, "            %s\n", key)
	}

	// Always add upload-artifact step for cache-memory (runs always)
	builder.WriteString("      - name: Upload cache-memory data as artifact\n")
	builder.WriteString("        uses: actions/upload-artifact@v4\n")
	builder.WriteString("        with:\n")
	builder.WriteString("          name: cache-memory\n")
	builder.WriteString("          path: /tmp/gh-aw/cache-memory\n")
	// Add retention-days if configured
	if data.CacheMemoryConfig.RetentionDays != nil {
		fmt.Fprintf(builder, "          retention-days: %d\n", *data.CacheMemoryConfig.RetentionDays)
	}
}

// generateCacheMemoryPromptSection generates the cache folder notification section for prompts
// when cache-memory is enabled, informing the agent about persistent storage capabilities
func generateCacheMemoryPromptSection(yaml *strings.Builder, config *CacheMemoryConfig) {
	if config == nil {
		return
	}

	// Handle multiple caches (array notation)
	if len(config.Caches) > 0 {
		yaml.WriteString("          \n")
		yaml.WriteString("          ---\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          ## Cache Folders Available\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          You have access to persistent cache folders where you can read and write files to create memories and store information:\n")
		yaml.WriteString("          \n")
		for _, cache := range config.Caches {
			cacheDir := fmt.Sprintf("/tmp/gh-aw/cache-memory/%s/", cache.ID)
			yaml.WriteString(fmt.Sprintf("          - **%s**: `%s`\n", cache.ID, cacheDir))
		}
		yaml.WriteString("          \n")
		yaml.WriteString("          - **Read/Write Access**: You can freely read from and write to any files in these folders\n")
		yaml.WriteString("          - **Persistence**: Files in these folders persist across workflow runs via GitHub Actions cache\n")
		yaml.WriteString("          - **Last Write Wins**: If multiple processes write to the same file, the last write will be preserved\n")
		yaml.WriteString("          - **File Share**: Use these as simple file shares - organize files as you see fit\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          Examples of what you can store:\n")
		for _, cache := range config.Caches {
			cacheDir := fmt.Sprintf("/tmp/gh-aw/cache-memory/%s", cache.ID)
			yaml.WriteString(fmt.Sprintf("          - `%s/notes.txt` - general notes and observations\n", cacheDir))
			yaml.WriteString(fmt.Sprintf("          - `%s/preferences.json` - user preferences and settings\n", cacheDir))
			yaml.WriteString(fmt.Sprintf("          - `%s/state/` - organized state files in subdirectories\n", cacheDir))
		}
		yaml.WriteString("          \n")
		yaml.WriteString("          Feel free to create, read, update, and organize files in these folders as needed for your tasks.\n")
		return
	}

	// Handle single cache (backward compatible)
	if !config.Enabled {
		return
	}

	yaml.WriteString("          \n")
	yaml.WriteString("          ---\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          ## Cache Folder Available\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          You have access to a persistent cache folder at `/tmp/gh-aw/cache-memory/` where you can read and write files to create memories and store information.\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          - **Read/Write Access**: You can freely read from and write to any files in this folder\n")
	yaml.WriteString("          - **Persistence**: Files in this folder persist across workflow runs via GitHub Actions cache\n")
	yaml.WriteString("          - **Last Write Wins**: If multiple processes write to the same file, the last write will be preserved\n")
	yaml.WriteString("          - **File Share**: Use this as a simple file share - organize files as you see fit\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          Examples of what you can store:\n")
	yaml.WriteString("          - `/tmp/gh-aw/cache-memory/notes.txt` - general notes and observations\n")
	yaml.WriteString("          - `/tmp/gh-aw/cache-memory/preferences.json` - user preferences and settings\n")
	yaml.WriteString("          - `/tmp/gh-aw/cache-memory/history.log` - activity history and logs\n")
	yaml.WriteString("          - `/tmp/gh-aw/cache-memory/state/` - organized state files in subdirectories\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          Feel free to create, read, update, and organize files in this folder as needed for your tasks.\n")
}
