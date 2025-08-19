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

		builder.WriteString(fmt.Sprintf("      - name: %s\n", stepName))
		builder.WriteString("        uses: actions/cache@v3\n")
		builder.WriteString("        with:\n")

		// Add required cache parameters
		if key, hasKey := cache["key"]; hasKey {
			builder.WriteString(fmt.Sprintf("          key: %v\n", key))
		}
		if path, hasPath := cache["path"]; hasPath {
			if pathArray, isArray := path.([]any); isArray {
				builder.WriteString("          path: |\n")
				for _, p := range pathArray {
					builder.WriteString(fmt.Sprintf("            %v\n", p))
				}
			} else {
				builder.WriteString(fmt.Sprintf("          path: %v\n", path))
			}
		}

		// Add optional cache parameters
		if restoreKeys, hasRestoreKeys := cache["restore-keys"]; hasRestoreKeys {
			if restoreArray, isArray := restoreKeys.([]any); isArray {
				builder.WriteString("          restore-keys: |\n")
				for _, key := range restoreArray {
					builder.WriteString(fmt.Sprintf("            %v\n", key))
				}
			} else {
				builder.WriteString(fmt.Sprintf("          restore-keys: %v\n", restoreKeys))
			}
		}
		if uploadChunkSize, hasSize := cache["upload-chunk-size"]; hasSize {
			builder.WriteString(fmt.Sprintf("          upload-chunk-size: %v\n", uploadChunkSize))
		}
		if failOnMiss, hasFail := cache["fail-on-cache-miss"]; hasFail {
			builder.WriteString(fmt.Sprintf("          fail-on-cache-miss: %v\n", failOnMiss))
		}
		if lookupOnly, hasLookup := cache["lookup-only"]; hasLookup {
			builder.WriteString(fmt.Sprintf("          lookup-only: %v\n", lookupOnly))
		}
	}
}
