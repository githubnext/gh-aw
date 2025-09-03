package workflow

import (
	"fmt"
)

// EngineConfig represents the parsed engine configuration
type EngineConfig struct {
	ID          string
	Version     string
	Model       string
	MaxTurns    string
	Permissions *EnginePermissions `yaml:"permissions,omitempty"`
}

// EnginePermissions represents the permissions configuration for an engine
type EnginePermissions struct {
	Network *NetworkPermissions `yaml:"network,omitempty"`
}

// NetworkPermissions represents network access permissions
type NetworkPermissions struct {
	Allowed []string `yaml:"allowed,omitempty"`
}

// extractEngineConfig extracts engine configuration from frontmatter, supporting both string and object formats
func (c *Compiler) extractEngineConfig(frontmatter map[string]any) (string, *EngineConfig) {
	if engine, exists := frontmatter["engine"]; exists {
		// Handle string format (backwards compatibility)
		if engineStr, ok := engine.(string); ok {
			return engineStr, &EngineConfig{ID: engineStr}
		}

		// Handle object format
		if engineObj, ok := engine.(map[string]any); ok {
			config := &EngineConfig{}

			// Extract required 'id' field
			if id, hasID := engineObj["id"]; hasID {
				if idStr, ok := id.(string); ok {
					config.ID = idStr
				}
			}

			// Extract optional 'version' field
			if version, hasVersion := engineObj["version"]; hasVersion {
				if versionStr, ok := version.(string); ok {
					config.Version = versionStr
				}
			}

			// Extract optional 'model' field
			if model, hasModel := engineObj["model"]; hasModel {
				if modelStr, ok := model.(string); ok {
					config.Model = modelStr
				}
			}

			// Extract optional 'max-turns' field
			if maxTurns, hasMaxTurns := engineObj["max-turns"]; hasMaxTurns {
				if maxTurnsInt, ok := maxTurns.(int); ok {
					config.MaxTurns = fmt.Sprintf("%d", maxTurnsInt)
				} else if maxTurnsUint64, ok := maxTurns.(uint64); ok {
					config.MaxTurns = fmt.Sprintf("%d", maxTurnsUint64)
				} else if maxTurnsStr, ok := maxTurns.(string); ok {
					config.MaxTurns = maxTurnsStr
				}
			}

			// Extract optional 'permissions' field
			if permissions, hasPermissions := engineObj["permissions"]; hasPermissions {
				if permissionsObj, ok := permissions.(map[string]any); ok {
					config.Permissions = &EnginePermissions{}

					// Extract network permissions
					if network, hasNetwork := permissionsObj["network"]; hasNetwork {
						if networkObj, ok := network.(map[string]any); ok {
							config.Permissions.Network = &NetworkPermissions{}

							// Extract allowed domains
							if allowed, hasAllowed := networkObj["allowed"]; hasAllowed {
								if allowedSlice, ok := allowed.([]any); ok {
									for _, domain := range allowedSlice {
										if domainStr, ok := domain.(string); ok {
											config.Permissions.Network.Allowed = append(config.Permissions.Network.Allowed, domainStr)
										}
									}
								}
							}
						}
					}
				}
			}

			// Return the ID as the engineSetting for backwards compatibility
			return config.ID, config
		}
	}

	// No engine specified
	return "", nil
}

// validateEngine validates that the given engine ID is supported
func (c *Compiler) validateEngine(engineID string) error {
	if engineID == "" {
		return nil // Empty engine is valid (will use default)
	}

	// First try exact match
	if c.engineRegistry.IsValidEngine(engineID) {
		return nil
	}

	// Try prefix match for backward compatibility (e.g., "codex-experimental")
	_, err := c.engineRegistry.GetEngineByPrefix(engineID)
	return err
}

// getAgenticEngine returns the agentic engine for the given engine setting
func (c *Compiler) getAgenticEngine(engineSetting string) (AgenticEngine, error) {
	if engineSetting == "" {
		return c.engineRegistry.GetDefaultEngine(), nil
	}

	// First try exact match
	if c.engineRegistry.IsValidEngine(engineSetting) {
		return c.engineRegistry.GetEngine(engineSetting)
	}

	// Try prefix match for backward compatibility
	return c.engineRegistry.GetEngineByPrefix(engineSetting)
}
