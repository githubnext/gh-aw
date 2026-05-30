package workflow

import (
	"encoding/json"
	"strings"
)

// extractSecretMaskingConfig extracts secret-masking configuration from frontmatter
func (c *Compiler) extractSecretMaskingConfig(frontmatter map[string]any) *SecretMaskingConfig {
	secretMaskingLog.Print("Extracting secret-masking configuration from frontmatter")

	if secretMasking, exists := frontmatter["secret-masking"]; exists {
		if secretMaskingMap, ok := secretMasking.(map[string]any); ok {
			config := &SecretMaskingConfig{}

			// Extract steps array
			if steps, exists := secretMaskingMap["steps"]; exists {
				if stepsArray, ok := steps.([]any); ok {
					var stepsConfig []map[string]any
					for _, step := range stepsArray {
						if stepMap, ok := step.(map[string]any); ok {
							stepsConfig = append(stepsConfig, stepMap)
						}
					}
					config.Steps = stepsConfig
					secretMaskingLog.Printf("Extracted %d secret-masking steps from frontmatter", len(stepsConfig))
				}
			}

			// Return nil if no steps were found
			if len(config.Steps) == 0 {
				secretMaskingLog.Print("No secret-masking steps found in frontmatter")
				return nil
			}

			return config
		}
	}

	secretMaskingLog.Print("No secret-masking configuration found in frontmatter")
	return nil
}

// MergeSecretMasking merges secret-masking configurations from imports with top-level config
func (c *Compiler) MergeSecretMasking(topConfig *SecretMaskingConfig, importedSecretMaskingJSON string) (*SecretMaskingConfig, error) {
	secretMaskingLog.Print("Merging secret-masking from imports")

	if importedSecretMaskingJSON == "" || importedSecretMaskingJSON == "{}" {
		secretMaskingLog.Print("No imported secret-masking to merge")
		return topConfig, nil
	}

	// Start with top-level config or create a new one
	result := &SecretMaskingConfig{}
	if topConfig != nil {
		result.Steps = make([]map[string]any, len(topConfig.Steps))
		copy(result.Steps, topConfig.Steps)
		secretMaskingLog.Printf("Starting with %d top-level steps", len(topConfig.Steps))
	}

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(importedSecretMaskingJSON, "\n")
	secretMaskingLog.Printf("Processing %d secret-masking definition lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		// Parse JSON line to secret-masking config
		var importedConfig SecretMaskingConfig
		if err := json.Unmarshal([]byte(line), &importedConfig); err != nil {
			secretMaskingLog.Printf("Failed to parse secret-masking: %v", err)
			continue // Skip invalid lines
		}

		// Append steps from imported config
		if len(importedConfig.Steps) > 0 {
			result.Steps = append(result.Steps, importedConfig.Steps...)
			secretMaskingLog.Printf("Merged %d steps from import", len(importedConfig.Steps))
		}
	}

	if len(result.Steps) == 0 {
		secretMaskingLog.Print("No secret-masking steps after merging")
		return nil, nil
	}

	secretMaskingLog.Printf("Successfully merged secret-masking with %d total steps", len(result.Steps))
	return result, nil
}
