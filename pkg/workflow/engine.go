package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/types"
)

var engineLog = logger.New("workflow:engine")

// EngineConfig represents the parsed engine configuration
type EngineConfig struct {
	ID               string
	Version          string
	Model            string
	MaxTurns         string
	MaxContinuations int    // Maximum number of continuations for autopilot mode (copilot engine only; > 1 enables --autopilot)
	Concurrency      string // Agent job-level concurrency configuration (YAML format)
	UserAgent        string
	Command          string // Custom executable path (when set, skip installation steps)
	Env              map[string]string
	Config           string
	Args             []string
	Agent            string // Agent identifier for copilot --agent flag (copilot engine only)
	APITarget        string // Custom API endpoint hostname (e.g., "api.acme.ghe.com" or "api.enterprise.githubcopilot.com")
	Bare             bool   // When true, disables automatic loading of context/instructions (copilot: --no-custom-instructions, claude: --bare, codex: --no-system-prompt, gemini: GEMINI_SYSTEM_MD=/dev/null)
	// TokenWeights provides custom model cost data for effective token computation.
	// When set, overrides or extends the built-in model_multipliers.json values.
	TokenWeights *types.TokenWeights

	// Inline definition fields (populated when engine.runtime is specified in frontmatter)
	IsInlineDefinition bool   // true when the engine is defined inline via engine.runtime + optional engine.provider
	InlineProviderID   string // engine.provider.id  (e.g. "openai", "anthropic")
	// Deprecated: Use InlineProviderAuth instead. Kept for backwards compatibility when only
	// engine.provider.auth.secret is specified without a strategy.
	InlineProviderSecret string // engine.provider.auth.secret  (backwards compat: simple API key secret name)

	// Extended inline auth fields (engine.provider.auth.* beyond the simple secret)
	InlineProviderAuth *AuthDefinition // full auth definition parsed from engine.provider.auth

	// Extended inline request shaping fields (engine.provider.request.*)
	InlineProviderRequest *RequestShape // request shaping parsed from engine.provider.request
}

// NetworkPermissions represents network access permissions for workflow execution
// Controls which domains the workflow can access during execution.
//
// The Allowed field specifies which domains/ecosystems are permitted:
//   - nil/not set: Use default ecosystem domains (backwards compatibility)
//   - []: Empty list means deny all network access
//   - ["defaults"]: Use default ecosystem domains
//   - ["defaults", "github", "python"]: Expand and merge multiple ecosystems
//   - ["example.com"]: Allow specific domain only
//
// Examples:
//
//  1. String format - use default domains only:
//     network: defaults
//     Result: NetworkPermissions{Allowed: ["defaults"], ExplicitlyDefined: true}
//
//  2. Object format - specify allowed ecosystems/domains:
//     network:
//     allowed:
//     - defaults      # Expands to default ecosystem domains (certs, JSON schema, Ubuntu, etc.)
//     - github        # Expands to GitHub ecosystem domains (*.githubusercontent.com, etc.)
//     - example.com   # Literal domain
//     Result: NetworkPermissions{Allowed: ["defaults", "github", "example.com"], ExplicitlyDefined: true}
//
//  3. Empty object - deny all network access:
//     network: {}
//     Result: NetworkPermissions{Allowed: [], ExplicitlyDefined: true}
//
// Ecosystem identifiers in the Allowed list are expanded to their corresponding domain lists.
// See GetAllowedDomains() for the list of supported ecosystem identifiers.
type NetworkPermissions struct {
	Allowed           []string        `yaml:"allowed,omitempty"`  // List of allowed domains or ecosystem identifiers (e.g., "defaults", "github", "python")
	Blocked           []string        `yaml:"blocked,omitempty"`  // List of blocked domains (takes precedence over allowed)
	Firewall          *FirewallConfig `yaml:"firewall,omitempty"` // AWF firewall configuration (see firewall.go)
	ExplicitlyDefined bool            `yaml:"-"`                  // Internal flag: true if network field was explicitly set in frontmatter
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

			// Detect inline definition: engine.runtime sub-object present instead of engine.id
			if runtime, hasRuntime := engineObj["runtime"]; hasRuntime {
				engineLog.Print("Found inline engine definition (engine.runtime sub-object)")
				config.IsInlineDefinition = true

				if runtimeObj, ok := runtime.(map[string]any); ok {
					if id, ok := runtimeObj["id"].(string); ok {
						config.ID = id
						engineLog.Printf("Inline engine runtime.id: %s", config.ID)
					}
					if version, hasVersion := runtimeObj["version"]; hasVersion {
						config.Version = stringutil.ParseVersionValue(version)
					}
				}

				// Extract optional provider sub-object
				if provider, hasProvider := engineObj["provider"]; hasProvider {
					if providerObj, ok := provider.(map[string]any); ok {
						if id, ok := providerObj["id"].(string); ok {
							config.InlineProviderID = id
						}
						if model, ok := providerObj["model"].(string); ok {
							config.Model = model
						}
						if auth, hasAuth := providerObj["auth"]; hasAuth {
							if authObj, ok := auth.(map[string]any); ok {
								authDef := parseAuthDefinition(authObj)
								// Only store an AuthDefinition when the user actually provided
								// at least one recognised field.  An empty map (e.g. `auth: {}`)
								// must not be treated as an explicit auth override.
								if authDef.Strategy != "" || authDef.Secret != "" ||
									authDef.TokenURL != "" || authDef.ClientIDRef != "" ||
									authDef.ClientSecretRef != "" || authDef.HeaderName != "" ||
									authDef.TokenField != "" {
									config.InlineProviderAuth = authDef
									// Backwards compat: expose the simple secret field directly.
									config.InlineProviderSecret = authDef.Secret
								}
							}
						}
						if request, hasRequest := providerObj["request"]; hasRequest {
							if requestObj, ok := request.(map[string]any); ok {
								config.InlineProviderRequest = parseRequestShape(requestObj)
							}
						}
					}
				}

				// Extract optional 'bare' field (shared with non-inline path)
				if bare, hasBare := engineObj["bare"]; hasBare {
					if bareBool, ok := bare.(bool); ok {
						config.Bare = bareBool
						engineLog.Printf("Extracted bare mode (inline): %v", config.Bare)
					}
				}

				engineLog.Printf("Extracted inline engine definition: runtimeID=%s, providerID=%s", config.ID, config.InlineProviderID)
				return config.ID, config
			}

			// Extract required 'id' field
			if id, hasID := engineObj["id"]; hasID {
				if idStr, ok := id.(string); ok {
					config.ID = idStr
				}
			}

			// Extract optional 'version' field
			if version, hasVersion := engineObj["version"]; hasVersion {
				config.Version = stringutil.ParseVersionValue(version)
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
					config.MaxTurns = strconv.Itoa(maxTurnsInt)
				} else if maxTurnsUint64, ok := maxTurns.(uint64); ok {
					config.MaxTurns = strconv.FormatUint(maxTurnsUint64, 10)
				} else if maxTurnsStr, ok := maxTurns.(string); ok {
					config.MaxTurns = maxTurnsStr
				}
			}

			// Extract optional 'max-continuations' field
			if maxCont, hasMaxCont := engineObj["max-continuations"]; hasMaxCont {
				if maxContInt, ok := maxCont.(int); ok {
					config.MaxContinuations = maxContInt
				} else if maxContUint64, ok := maxCont.(uint64); ok {
					config.MaxContinuations = int(maxContUint64)
				} else if maxContStr, ok := maxCont.(string); ok {
					if parsed, err := strconv.Atoi(maxContStr); err == nil {
						config.MaxContinuations = parsed
					}
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

			// Extract optional 'command' field
			if command, hasCommand := engineObj["command"]; hasCommand {
				if commandStr, ok := command.(string); ok {
					config.Command = commandStr
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

			// Extract optional 'agent' field (string - copilot engine only)
			if agent, hasAgent := engineObj["agent"]; hasAgent {
				if agentStr, ok := agent.(string); ok {
					config.Agent = agentStr
					engineLog.Printf("Extracted agent identifier: %s", agentStr)
				}
			}

			// Extract optional 'api-target' field (custom API endpoint for any engine)
			if apiTarget, hasAPITarget := engineObj["api-target"]; hasAPITarget {
				if apiTargetStr, ok := apiTarget.(string); ok && apiTargetStr != "" {
					config.APITarget = apiTargetStr
					engineLog.Printf("Extracted api-target: %s", apiTargetStr)
				}
			}

			// Extract optional 'bare' field (disable automatic context/instruction loading)
			if bare, hasBare := engineObj["bare"]; hasBare {
				if bareBool, ok := bare.(bool); ok {
					config.Bare = bareBool
					engineLog.Printf("Extracted bare mode: %v", config.Bare)
				}
			}

			// Extract optional 'token-weights' field (custom model cost data)
			if tokenWeightsRaw, hasTokenWeights := engineObj["token-weights"]; hasTokenWeights {
				if tw := parseEngineTokenWeights(tokenWeightsRaw); tw != nil {
					config.TokenWeights = tw
					engineLog.Printf("Extracted token-weights: %d multipliers", len(tw.Multipliers))
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

// getAgenticEngine returns the agentic engine for the given engine setting
func (c *Compiler) getAgenticEngine(engineSetting string) (CodingAgentEngine, error) {
	if engineSetting == "" {
		defaultEngine := c.engineRegistry.GetDefaultEngine()
		engineLog.Printf("Using default engine: %s", defaultEngine.GetID())
		return defaultEngine, nil
	}

	engineLog.Printf("Getting agentic engine for setting: %s", engineSetting)

	// First try exact match
	if c.engineRegistry.IsValidEngine(engineSetting) {
		engine, err := c.engineRegistry.GetEngine(engineSetting)
		if err == nil {
			engineLog.Printf("Found engine by exact match: %s", engine.GetID())
		}
		return engine, err
	}

	// Try prefix match for backward compatibility
	engine, err := c.engineRegistry.GetEngineByPrefix(engineSetting)
	if err == nil {
		engineLog.Printf("Found engine by prefix match: %s", engine.GetID())
	} else {
		engineLog.Printf("Failed to find engine for setting %s: %v", engineSetting, err)
	}
	return engine, err
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

// parseAuthDefinition converts a raw auth config map (from engine.provider.auth) into
// an AuthDefinition. It is backward-compatible: a map with only a "secret" key produces
// an AuthDefinition with Strategy="" and Secret set (callers normalise Strategy to api-key).
func parseAuthDefinition(authObj map[string]any) *AuthDefinition {
	def := &AuthDefinition{}
	if s, ok := authObj["strategy"].(string); ok {
		def.Strategy = AuthStrategy(s)
	}
	if s, ok := authObj["secret"].(string); ok {
		def.Secret = s
	}
	if s, ok := authObj["token-url"].(string); ok {
		def.TokenURL = s
	}
	if s, ok := authObj["client-id"].(string); ok {
		def.ClientIDRef = s
	}
	if s, ok := authObj["client-secret"].(string); ok {
		def.ClientSecretRef = s
	}
	if s, ok := authObj["token-field"].(string); ok {
		def.TokenField = s
	}
	if s, ok := authObj["header-name"].(string); ok {
		def.HeaderName = s
	}
	return def
}

// parseRequestShape converts a raw request config map (from engine.provider.request) into
// a RequestShape.
func parseRequestShape(requestObj map[string]any) *RequestShape {
	shape := &RequestShape{}
	if s, ok := requestObj["path-template"].(string); ok {
		shape.PathTemplate = s
	}
	if q, ok := requestObj["query"].(map[string]any); ok {
		shape.Query = make(map[string]string, len(q))
		for k, v := range q {
			if vs, ok := v.(string); ok {
				shape.Query[k] = vs
			}
		}
	}
	if b, ok := requestObj["body-inject"].(map[string]any); ok {
		shape.BodyInject = make(map[string]string, len(b))
		for k, v := range b {
			if vs, ok := v.(string); ok {
				shape.BodyInject[k] = vs
			}
		}
	}
	return shape
}

// parseEngineTokenWeights converts a raw token-weights config value (from engine.token-weights)
// into a types.TokenWeights. Returns nil when the input is not a usable map or contains
// no recognisable data. Multiplier values of unexpected numeric types (anything other than
// float64, int, or uint64) are silently ignored — this matches the behaviour of the YAML
// parser which produces float64 for JSON-number literals and integers for integer literals.
func parseEngineTokenWeights(raw any) *types.TokenWeights {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	tw := &types.TokenWeights{}

	// Parse multipliers: map of model name → float64
	if multipliersRaw, ok := obj["multipliers"]; ok {
		if multipliersMap, ok := multipliersRaw.(map[string]any); ok && len(multipliersMap) > 0 {
			tw.Multipliers = make(map[string]float64, len(multipliersMap))
			for model, val := range multipliersMap {
				switch v := val.(type) {
				case float64:
					tw.Multipliers[model] = v
				case int:
					tw.Multipliers[model] = float64(v)
				case uint64:
					tw.Multipliers[model] = float64(v)
				}
			}
		}
	}

	// Parse token-class-weights
	if tcwRaw, ok := obj["token-class-weights"]; ok {
		if tcwMap, ok := tcwRaw.(map[string]any); ok {
			tcw := &types.TokenClassWeights{}
			setFloat := func(dst *float64, key string) {
				if v, ok := tcwMap[key]; ok {
					switch f := v.(type) {
					case float64:
						*dst = f
					case int:
						*dst = float64(f)
					case uint64:
						*dst = float64(f)
					}
				}
			}
			setFloat(&tcw.Input, "input")
			setFloat(&tcw.CachedInput, "cached-input")
			setFloat(&tcw.Output, "output")
			setFloat(&tcw.Reasoning, "reasoning")
			setFloat(&tcw.CacheWrite, "cache-write")
			// Only assign if at least one weight was set
			if tcw.Input != 0 || tcw.CachedInput != 0 || tcw.Output != 0 ||
				tcw.Reasoning != 0 || tcw.CacheWrite != 0 {
				tw.TokenClassWeights = tcw
			}
		}
	}

	// Return nil when nothing useful was parsed
	if len(tw.Multipliers) == 0 && tw.TokenClassWeights == nil {
		return nil
	}
	return tw
}
