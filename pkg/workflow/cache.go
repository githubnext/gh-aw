package workflow

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

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
		builder.WriteString("        uses: actions/cache@v5\n")
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
func generateCacheMemorySteps(builder *strings.Builder, data *WorkflowData, verbose bool) {
	if data.CacheMemoryConfig == nil || !data.CacheMemoryConfig.Enabled {
		return
	}

	// Add comment indicating cache-memory configuration was processed
	builder.WriteString("      # Cache memory MCP configuration from frontmatter processed below\n")

	// Add step to create cache-memory directory
	builder.WriteString("      - name: Create cache-memory directory\n")
	builder.WriteString("        run: mkdir -p /tmp/cache-memory\n")

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

	builder.WriteString("      - name: Cache memory MCP data\n")
	builder.WriteString("        uses: actions/cache@v5\n")
	builder.WriteString("        with:\n")
	fmt.Fprintf(builder, "          key: %s\n", cacheKey)
	builder.WriteString("          path: /tmp/cache-memory\n")
	builder.WriteString("          restore-keys: |\n")
	for _, key := range restoreKeys {
		fmt.Fprintf(builder, "            %s\n", key)
	}

	// Add upload-artifact step if retention-days is configured
	if data.CacheMemoryConfig.RetentionDays != nil {
		builder.WriteString("      - name: Upload memory MCP data as artifact\n")
		builder.WriteString("        uses: actions/upload-artifact@v4\n")
		builder.WriteString("        with:\n")
		builder.WriteString("          name: cache-memory-data\n")
		builder.WriteString("          path: /tmp/cache-memory\n")
		fmt.Fprintf(builder, "          retention-days: %d\n", *data.CacheMemoryConfig.RetentionDays)
	}
}
