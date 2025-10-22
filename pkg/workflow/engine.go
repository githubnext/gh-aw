package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var engineLog = logger.New("workflow:engine")

// EngineConfig represents the parsed engine configuration
type EngineConfig struct {
	ID            string
	Version       string
	Model         string
	MaxTurns      string
	Concurrency   string // Agent job-level concurrency configuration (YAML format)
	UserAgent     string
	Env           map[string]string
	Steps         []map[string]any
	ErrorPatterns []ErrorPattern
	Config        string
	Args          []string
	Firewall      *FirewallConfig // AWF firewall configuration
}

// NetworkPermissions represents network access permissions
type NetworkPermissions struct {
	Mode    string   `yaml:"mode,omitempty"`    // "defaults" for default access
	Allowed []string `yaml:"allowed,omitempty"` // List of allowed domains
}

// FirewallConfig represents AWF (gh-aw-firewall) configuration for network egress control
type FirewallConfig struct {
	Enabled       bool   `yaml:"enabled,omitempty"`        // Always true for copilot by default
	Version       string `yaml:"version,omitempty"`        // AWF version (empty = latest)
	LogLevel      string `yaml:"log_level,omitempty"`      // AWF log level (default: "debug")
	CleanupScript string `yaml:"cleanup_script,omitempty"` // Cleanup script path (default: "./scripts/ci/cleanup.sh")
}

// EngineNetworkConfig combines engine configuration with top-level network permissions
type EngineNetworkConfig struct {
	Engine  *EngineConfig
	Network *NetworkPermissions
}

// ExtractEngineConfig extracts engine configuration from frontmatter, supporting both string and object formats
func (c *Compiler) ExtractEngineConfig(frontmatter map[string]any) (string, *EngineConfig) {
	if engine, exists := frontmatter["engine"]; exists {
		engineLog.Print("Extracting engine configuration from frontmatter")

		// Handle string format (backwards compatibility)
		if engineStr, ok := engine.(string); ok {
			engineLog.Printf("Found engine in string format: %s", engineStr)
			return engineStr, &EngineConfig{ID: engineStr}
		}

		// Handle object format
		if engineObj, ok := engine.(map[string]any); ok {
			engineLog.Print("Found engine in object format, parsing configuration")
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

			// Extract optional 'concurrency' field (string or object format)
			if concurrency, hasConcurrency := engineObj["concurrency"]; hasConcurrency {
				if concurrencyStr, ok := concurrency.(string); ok {
					// Simple string format (group name)
					config.Concurrency = fmt.Sprintf("concurrency:\n  group: \"%s\"", concurrencyStr)
				} else if concurrencyObj, ok := concurrency.(map[string]any); ok {
					// Object format with group and optional cancel-in-progress
					var parts []string
					if group, hasGroup := concurrencyObj["group"]; hasGroup {
						if groupStr, ok := group.(string); ok {
							parts = append(parts, fmt.Sprintf("concurrency:\n  group: \"%s\"", groupStr))
						}
					}
					if cancel, hasCancel := concurrencyObj["cancel-in-progress"]; hasCancel {
						if cancelBool, ok := cancel.(bool); ok && cancelBool {
							if len(parts) > 0 {
								parts[0] += "\n  cancel-in-progress: true"
							}
						}
					}
					if len(parts) > 0 {
						config.Concurrency = parts[0]
					}
				}
			}

			// Extract optional 'user-agent' field
			if userAgent, hasUserAgent := engineObj["user-agent"]; hasUserAgent {
				if userAgentStr, ok := userAgent.(string); ok {
					config.UserAgent = userAgentStr
				}
			}

			// Extract optional 'env' field (object/map of strings)
			if env, hasEnv := engineObj["env"]; hasEnv {
				if envMap, ok := env.(map[string]any); ok {
					config.Env = make(map[string]string)
					for key, value := range envMap {
						if valueStr, ok := value.(string); ok {
							config.Env[key] = valueStr
						}
					}
				}
			}

			// Extract optional 'steps' field (array of step objects)
			if steps, hasSteps := engineObj["steps"]; hasSteps {
				if stepsArray, ok := steps.([]any); ok {
					config.Steps = make([]map[string]any, 0, len(stepsArray))
					for _, step := range stepsArray {
						if stepMap, ok := step.(map[string]any); ok {
							config.Steps = append(config.Steps, stepMap)
						}
					}
				}
			}

			// Extract optional 'error_patterns' field (array of error pattern objects)
			if errorPatterns, hasErrorPatterns := engineObj["error_patterns"]; hasErrorPatterns {
				if patternsArray, ok := errorPatterns.([]any); ok {
					config.ErrorPatterns = make([]ErrorPattern, 0, len(patternsArray))
					for _, patternRaw := range patternsArray {
						if patternMap, ok := patternRaw.(map[string]any); ok {
							pattern := ErrorPattern{}

							// Extract pattern field (required)
							if patternStr, ok := patternMap["pattern"].(string); ok {
								pattern.Pattern = patternStr
							} else {
								continue // Skip invalid patterns without pattern field
							}

							// Extract level_group field (optional, defaults to 0)
							if levelGroup, ok := patternMap["level_group"].(int); ok {
								pattern.LevelGroup = levelGroup
							} else if levelGroupFloat, ok := patternMap["level_group"].(float64); ok {
								pattern.LevelGroup = int(levelGroupFloat)
							} else if levelGroupUint64, ok := patternMap["level_group"].(uint64); ok {
								pattern.LevelGroup = int(levelGroupUint64)
							}

							// Extract message_group field (optional, defaults to 0)
							if messageGroup, ok := patternMap["message_group"].(int); ok {
								pattern.MessageGroup = messageGroup
							} else if messageGroupFloat, ok := patternMap["message_group"].(float64); ok {
								pattern.MessageGroup = int(messageGroupFloat)
							} else if messageGroupUint64, ok := patternMap["message_group"].(uint64); ok {
								pattern.MessageGroup = int(messageGroupUint64)
							}

							// Extract description field (optional)
							if description, ok := patternMap["description"].(string); ok {
								pattern.Description = description
							}

							config.ErrorPatterns = append(config.ErrorPatterns, pattern)
						}
					}
				}
			}

			// Extract optional 'config' field (additional TOML configuration)
			if config_field, hasConfig := engineObj["config"]; hasConfig {
				if configStr, ok := config_field.(string); ok {
					config.Config = configStr
				}
			}

			// Extract optional 'args' field (array of strings)
			if args, hasArgs := engineObj["args"]; hasArgs {
				if argsArray, ok := args.([]any); ok {
					config.Args = make([]string, 0, len(argsArray))
					for _, arg := range argsArray {
						if argStr, ok := arg.(string); ok {
							config.Args = append(config.Args, argStr)
						}
					}
				} else if argsStrArray, ok := args.([]string); ok {
					config.Args = argsStrArray
				}
			}

			// Extract optional 'firewall' field (object format)
			if firewall, hasFirewall := engineObj["firewall"]; hasFirewall {
				if firewallObj, ok := firewall.(map[string]any); ok {
					firewallConfig := &FirewallConfig{}

					// Extract enabled field (defaults to true for copilot)
					if enabled, hasEnabled := firewallObj["enabled"]; hasEnabled {
						if enabledBool, ok := enabled.(bool); ok {
							firewallConfig.Enabled = enabledBool
						}
					}

					// Extract version field (empty = latest)
					if version, hasVersion := firewallObj["version"]; hasVersion {
						if versionStr, ok := version.(string); ok {
							firewallConfig.Version = versionStr
						}
					}

					// Extract log_level field (default: "debug")
					if logLevel, hasLogLevel := firewallObj["log_level"]; hasLogLevel {
						if logLevelStr, ok := logLevel.(string); ok {
							firewallConfig.LogLevel = logLevelStr
						}
					}

					// Extract cleanup_script field (default: "./scripts/ci/cleanup.sh")
					if cleanupScript, hasCleanupScript := firewallObj["cleanup_script"]; hasCleanupScript {
						if cleanupScriptStr, ok := cleanupScript.(string); ok {
							firewallConfig.CleanupScript = cleanupScriptStr
						}
					}

					config.Firewall = firewallConfig
					engineLog.Print("Extracted firewall configuration")
				}
			}

			// Return the ID as the engineSetting for backwards compatibility
			engineLog.Printf("Extracted engine configuration: ID=%s", config.ID)
			return config.ID, config
		}
	}

	// No engine specified
	engineLog.Print("No engine configuration found in frontmatter")
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
func (c *Compiler) getAgenticEngine(engineSetting string) (CodingAgentEngine, error) {
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

// validateSingleEngineSpecification validates that only one engine field exists across all files
func (c *Compiler) validateSingleEngineSpecification(mainEngineSetting string, includedEnginesJSON []string) (string, error) {
	var allEngines []string

	// Add main engine if specified
	if mainEngineSetting != "" {
		allEngines = append(allEngines, mainEngineSetting)
	}

	// Add included engines
	for _, engineJSON := range includedEnginesJSON {
		if engineJSON != "" {
			allEngines = append(allEngines, engineJSON)
		}
	}

	// Check count
	if len(allEngines) == 0 {
		return "", nil // No engine specified anywhere, will use default
	}

	if len(allEngines) > 1 {
		return "", fmt.Errorf("multiple engine fields found. Only one engine field is allowed across the main workflow and all included files. Remove engine specifications to have only one")
	}

	// Exactly one engine found - parse and return it
	if mainEngineSetting != "" {
		return mainEngineSetting, nil
	}

	// Must be from included file
	var firstEngine any
	if err := json.Unmarshal([]byte(includedEnginesJSON[0]), &firstEngine); err != nil {
		return "", fmt.Errorf("failed to parse included engine configuration: %w", err)
	}

	// Handle string format
	if engineStr, ok := firstEngine.(string); ok {
		return engineStr, nil
	} else if engineObj, ok := firstEngine.(map[string]any); ok {
		// Handle object format - return the ID
		if id, hasID := engineObj["id"]; hasID {
			if idStr, ok := id.(string); ok {
				return idStr, nil
			}
		}
	}

	return "", fmt.Errorf("invalid engine configuration in included file")
}

// extractEngineConfigFromJSON parses engine configuration from JSON string (from included files)
func (c *Compiler) extractEngineConfigFromJSON(engineJSON string) (*EngineConfig, error) {
	if engineJSON == "" {
		return nil, nil
	}

	var engineData any
	if err := json.Unmarshal([]byte(engineJSON), &engineData); err != nil {
		return nil, fmt.Errorf("failed to parse engine JSON: %w", err)
	}

	// Use the existing ExtractEngineConfig function by creating a temporary frontmatter map
	tempFrontmatter := map[string]any{
		"engine": engineData,
	}

	_, config := c.ExtractEngineConfig(tempFrontmatter)
	return config, nil
}
