package workflow

import (
	"strings"
)

// generateCacheMemoryPromptStep generates a separate step for cache memory instructions
// when cache-memory is enabled, informing the agent about persistent storage capabilities
func (c *Compiler) generateCacheMemoryPromptStep(yaml *strings.Builder, config *CacheMemoryConfig) {
	if config == nil || len(config.Caches) == 0 {
		return
	}

	appendPromptStepWithHeredoc(yaml,
		"Append cache memory instructions to prompt",
		func(y *strings.Builder) {
			generateCacheMemoryPromptSection(y, config)
		})
}
