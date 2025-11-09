package workflow

import (
	"fmt"
)

// ValidateToolsConfiguration validates tool configurations for bash and cache-memory
func ValidateToolsConfiguration(tools map[string]any) error {
	// Validate bash tool configuration
	if bashTool, hasBash := tools["bash"]; hasBash {
		if _, err := validateBashToolValue(bashTool); err != nil {
			return fmt.Errorf("bash: %w", err)
		}
	}

	// Validate cache-memory tool configuration
	if cacheMemoryTool, hasCacheMemory := tools["cache-memory"]; hasCacheMemory {
		if _, err := validateCacheMemoryToolValue(cacheMemoryTool); err != nil {
			return fmt.Errorf("cache-memory: %w", err)
		}
	}

	return nil
}
