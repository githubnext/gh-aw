package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/types"
	"github.com/github/gh-aw/pkg/typeutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var engineLog = logger.New("workflow:engine")

const WorkflowCallNetworkAllowedEnvVar = "GH_AW_WORKFLOW_CALL_NETWORK_ALLOWED"

func injectWorkflowCallNetworkAllowedEnv(env map[string]string, workflowData *WorkflowData) {
	if shouldUseWorkflowCallNetworkAllowedInput(workflowData) {
		env[WorkflowCallNetworkAllowedEnvVar] = fmt.Sprintf("${{ inputs.%s }}", NetworkAllowedInputName)
	}
}

func toEngineEnvValueString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v), true
	default:
		return "", false
	}
}

// EngineConfig represents the parsed engine configuration
type EngineConfig struct {
	ID                 string
	Version            string
	Model              string
	LLMProvider        string // Inference provider override for this engine (github|anthropic|openai)
	PermissionMode     string
	MaxTurns           string
	MaxToolDenials     string // Maximum repeated tool denials before stopping inference (copilot SDK mode only)
	MaxRuns            int    // Maximum number of LLM invocations per run (AWF apiProxy.maxRuns)
	MaxTurnCacheMisses int    // Maximum number of consecutive cache misses per run (AWF apiProxy.maxCacheMisses)
	MaxContinuations   int    // Maximum number of continuations for autopilot mode (copilot engine only; > 1 enables --autopilot)
	MaxAICredits       int64  // Maximum allowed AI credits per run for AWF apiProxy firewall enforcement
	Concurrency        string // Agent job-level concurrency configuration (YAML format)
	UserAgent          string
	Command            string // Custom executable path (when set, skip installation steps)
	HarnessScript      string // Custom Node.js harness script filename (replaces engine default harness script when supported)
	Driver             string // Custom driver script filename or command. For the copilot engine (engine.copilot-sdk-driver / engine.driver), supports .js/.cjs/.mjs (Node.js), .py (Python), .ts/.mts (TypeScript), .rb (Ruby), or a bare command name. For the pi engine (engine.driver), supports .js/.cjs/.mjs or a bare basename resolved from the setup-action directory.
	Env                map[string]string
	Auth               *EngineAuthConfig // Engine-level auth config (mapped to AWF_AUTH_* env vars for API proxy sidecar auth)
	Config             string
	Args               []string
	Agent              string // Agent identifier for copilot --agent flag (copilot engine only)
	APITarget          string // Custom API endpoint hostname (e.g., "api.acme.ghe.com" or "api.enterprise.githubcopilot.com")
	Bare               bool   // When true, disables automatic loading of context/instructions (copilot: --no-custom-instructions, claude: --bare, codex: --no-system-prompt, gemini: GEMINI_SYSTEM_MD=/dev/null)
	// TokenWeights provides custom model cost data for AI Credits cost ratios.
	// When set, overrides or extends built-in model cost defaults.
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

	// MCP gateway configuration from engine.mcp sub-object
	MCPSessionTimeout string // session-timeout: Go duration string for MCP gateway sessions (e.g. "4h", "30m")
	MCPToolTimeout    string // tool-timeout: Go duration string for individual MCP tool calls (e.g. "2m", "30s")

	// Extensions is a list of engine-specific plugin names to install before launching the engine.
	// Currently used by the Pi engine: each entry is passed to `pi install <extension>`.
	Extensions []string

	// CopilotSDK enables the GitHub Copilot SDK integration.
	// When true the compiler enables a harness-managed Copilot CLI headless sidecar
	// and sets COPILOT_SDK_URI on child processes so the SDK can connect to it.
	CopilotSDK bool
}

// EngineAuthConfig represents engine.auth frontmatter settings that map to
// AWF_AUTH_* environment variables consumed by the AWF API proxy sidecar.
type EngineAuthConfig struct {
	Type     string
	Audience string
	Provider string // "azure" or "anthropic"
	// Azure WIF fields
	AzureTenantID string
	AzureClientID string
	AzureScope    string
	AzureCloud    string
	// Anthropic WIF fields
	AnthropicFederationRuleID string
	AnthropicOrganizationID   string
	AnthropicServiceAccountID string
	AnthropicWorkspaceID      string
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
	Allowed           []string        `yaml:"allowed,omitempty"` // List of allowed domains or ecosystem identifiers (e.g., "defaults", "github", "python")
	AllowedInput      bool            `yaml:"allowed-input,omitempty"`
	Blocked           []string        `yaml:"blocked,omitempty"`  // List of blocked domains (takes precedence over allowed)
	Firewall          *FirewallConfig `yaml:"firewall,omitempty"` // AWF firewall configuration (see firewall.go)
	ExplicitlyDefined bool            `yaml:"-"`                  // Internal flag: true if network field was explicitly set in frontmatter
}

// EngineNetworkConfig combines engine configuration with top-level network permissions
type EngineNetworkConfig struct {
	Engine  *EngineConfig
	Network *NetworkPermissions
}

// GetMaxAICredits returns the configured engine AI credits budget, falling back to the default.
func (e *EngineConfig) GetMaxAICredits() int64 {
	if e == nil || e.MaxAICredits == 0 {
		return constants.DefaultMaxAICredits
	}
	return e.MaxAICredits
}

// GetMaxRuns returns the configured AWF max-runs value, falling back to the default.
func (e *EngineConfig) GetMaxRuns() int {
	if e == nil || e.MaxRuns <= 0 {
		return constants.DefaultMaxRuns
	}
	return e.MaxRuns
}

// GetMaxTurnCacheMisses returns the configured AWF max-turn-cache-misses value, falling back
// to the enterprise override or built-in default.
func (e *EngineConfig) GetMaxTurnCacheMisses() int {
	if e == nil || e.MaxTurnCacheMisses <= 0 {
		return compilerenv.ResolveDefaultMaxTurnCacheMisses(constants.DefaultMaxTurnCacheMisses)
	}
	return e.MaxTurnCacheMisses
}

// ExtractEngineConfig extracts engine configuration from frontmatter, supporting both string and object formats
func (c *Compiler) ExtractEngineConfig(frontmatter map[string]any) (string, *EngineConfig) {
	topLevelMaxTurns := parseMaxTurnsValue(frontmatter["max-turns"])
	topLevelMaxToolDenials := parseMaxToolDenialsValue(frontmatter["max-tool-denials"])
	topLevelMaxAICredits := parseMaxAICreditsValue(frontmatter["max-ai-credits"])
	topLevelMaxTurnCacheMisses := parseMaxTurnCacheMissesValue(frontmatter["max-turn-cache-misses"])
	topLevelMaxRuns := parseMaxRunsValue(frontmatter["max-turns"])
	if topLevelMaxRuns == 0 {
		topLevelMaxRuns = parseMaxRunsValue(frontmatter["max-runs"])
	}

	if engine, exists := frontmatter["engine"]; exists {
		engineLog.Print("Extracting engine configuration from frontmatter")

		// Handle string format (backwards compatibility)
		if engineStr, ok := engine.(string); ok {
			engineLog.Printf("Found engine in string format: %s", engineStr)
			return engineStr, &EngineConfig{
				ID:                 engineStr,
				MaxTurns:           topLevelMaxTurns,
				MaxToolDenials:     topLevelMaxToolDenials,
				MaxRuns:            topLevelMaxRuns,
				MaxTurnCacheMisses: topLevelMaxTurnCacheMisses,
				MaxAICredits:       topLevelMaxAICredits,
			}
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
				// Extract optional 'permission-mode' field (shared with non-inline path)
				if permissionMode, hasPermissionMode := engineObj["permission-mode"]; hasPermissionMode {
					if permissionModeStr, ok := permissionMode.(string); ok {
						config.PermissionMode = permissionModeStr
					}
				}
				if topLevelMaxTurns != "" {
					config.MaxTurns = topLevelMaxTurns
				}
				if topLevelMaxToolDenials != "" {
					config.MaxToolDenials = topLevelMaxToolDenials
				}
				config.MaxRuns = topLevelMaxRuns
				config.MaxTurnCacheMisses = topLevelMaxTurnCacheMisses
				config.MaxAICredits = topLevelMaxAICredits

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

			// Extract optional 'model-provider' field.
			providerValue, hasProvider := engineObj["model-provider"]
			if hasProvider {
				if providerStr, ok := providerValue.(string); ok {
					config.LLMProvider = strings.ToLower(strings.TrimSpace(providerStr))
				}
			}

			// Extract optional 'permission-mode' field
			if permissionMode, hasPermissionMode := engineObj["permission-mode"]; hasPermissionMode {
				if permissionModeStr, ok := permissionMode.(string); ok {
					config.PermissionMode = permissionModeStr
				}
			}

			// Extract optional 'max-turns' field (deprecated alias for top-level max-turns).
			// Use parseMaxTurnsValue for consistent validation: rejects negative values and
			// arbitrary strings while preserving valid integers and GitHub Actions expressions.
			if maxTurns, hasMaxTurns := engineObj["max-turns"]; hasMaxTurns {
				config.MaxTurns = parseMaxTurnsValue(maxTurns)
			}
			if topLevelMaxTurns != "" {
				config.MaxTurns = topLevelMaxTurns
			}
			if topLevelMaxToolDenials != "" {
				config.MaxToolDenials = topLevelMaxToolDenials
			}

			// Extract optional 'max-continuations' field
			if maxCont, hasMaxCont := engineObj["max-continuations"]; hasMaxCont {
				if val, ok := typeutil.ParseIntValue(maxCont); ok {
					config.MaxContinuations = val
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
					if queue, hasQueue := concurrencyObj["queue"]; hasQueue {
						if queueStr, ok := queue.(string); ok && queueStr != "" {
							if len(parts) > 0 {
								parts[0] += "\n  queue: " + queueStr
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

			// Extract optional 'harness' field (string - validated separately)
			if harness, hasHarness := engineObj["harness"]; hasHarness {
				if harnessStr, ok := harness.(string); ok {
					config.HarnessScript = harnessStr
				}
			}

			// Extract optional 'driver' / 'copilot-sdk-driver' field (string - validated separately).
			// Both keys map to the shared Driver field. 'copilot-sdk-driver' is accepted for
			// backward compatibility; 'driver' is the canonical name going forward.
			if driver, hasDriver := engineObj["driver"]; hasDriver {
				if driverStr, ok := driver.(string); ok {
					config.Driver = driverStr
					engineLog.Printf("Extracted engine.driver: %s", driverStr)
				}
			} else if sdkDriver, hasSDKDriver := engineObj["copilot-sdk-driver"]; hasSDKDriver {
				if sdkDriverStr, ok := sdkDriver.(string); ok {
					config.Driver = sdkDriverStr
					engineLog.Printf("Extracted engine.copilot-sdk-driver (→ driver): %s", sdkDriverStr)
				}
			}

			// Extract optional 'env' field (object/map of scalar values)
			if env, hasEnv := engineObj["env"]; hasEnv {
				if envMap, ok := env.(map[string]any); ok {
					config.Env = make(map[string]string)
					for key, value := range envMap {
						if valueStr, ok := toEngineEnvValueString(value); ok {
							config.Env[key] = valueStr
						}
					}
				}
			}

			// Extract optional 'auth' field (object)
			if auth, hasAuth := engineObj["auth"]; hasAuth {
				if authObj, ok := auth.(map[string]any); ok {
					config.Auth = parseEngineAuthConfig(authObj)
					applyEngineAuthEnv(config)
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

			// Extract optional 'mcp' sub-object (engine-level MCP gateway configuration)
			if mcpVal, hasMCP := engineObj["mcp"]; hasMCP {
				if mcpObj, ok := mcpVal.(map[string]any); ok {
					// Extract session-timeout (kebab-case only; camelCase is not supported)
					if stVal, hasSessionTimeout := mcpObj["session-timeout"]; hasSessionTimeout {
						if stStr, ok := stVal.(string); ok && stStr != "" {
							config.MCPSessionTimeout = stStr
							engineLog.Printf("Extracted engine.mcp.session-timeout: %s", config.MCPSessionTimeout)
						}
					}
					// Extract tool-timeout (kebab-case only; camelCase is not supported)
					if ttVal, hasToolTimeout := mcpObj["tool-timeout"]; hasToolTimeout {
						if ttStr, ok := ttVal.(string); ok && ttStr != "" {
							config.MCPToolTimeout = ttStr
							engineLog.Printf("Extracted engine.mcp.tool-timeout: %s", config.MCPToolTimeout)
						}
					}
				}
			}

			// Extract optional 'extensions' field (array of strings; used by the Pi engine)
			if extVal, hasExt := engineObj["extensions"]; hasExt {
				switch v := extVal.(type) {
				case []any:
					config.Extensions = make([]string, 0, len(v))
					for _, ext := range v {
						if extStr, ok := ext.(string); ok && extStr != "" {
							config.Extensions = append(config.Extensions, extStr)
						}
					}
					engineLog.Printf("Extracted engine.extensions: %v", config.Extensions)
				case []string:
					config.Extensions = make([]string, 0, len(v))
					for _, ext := range v {
						if ext != "" {
							config.Extensions = append(config.Extensions, ext)
						}
					}
					engineLog.Printf("Extracted engine.extensions ([]string): %v", config.Extensions)
				default:
					engineLog.Printf("Unexpected type for engine.extensions: %T, ignoring", extVal)
				}
			}

			// Return the ID as the engineSetting for backwards compatibility
			if topLevelMaxTurns != "" {
				config.MaxTurns = topLevelMaxTurns
			}
			config.MaxRuns = topLevelMaxRuns
			config.MaxTurnCacheMisses = topLevelMaxTurnCacheMisses
			config.MaxAICredits = topLevelMaxAICredits

			// Extract optional 'copilot-sdk' field (bool; copilot engine only)
			if sdkVal, hasSDK := engineObj["copilot-sdk"]; hasSDK {
				if sdkBool, ok := sdkVal.(bool); ok {
					config.CopilotSDK = sdkBool
					engineLog.Printf("Extracted copilot-sdk: %v", config.CopilotSDK)
				}
			}
			if config.Driver != "" && config.ID == "copilot" && !config.CopilotSDK {
				config.CopilotSDK = true
				engineLog.Print("Enabled copilot-sdk because driver is configured for copilot engine")
			}

			engineLog.Printf("Extracted engine configuration: ID=%s", config.ID)
			return config.ID, config
		}
	}

	if topLevelMaxTurns != "" || topLevelMaxToolDenials != "" || topLevelMaxAICredits != 0 || topLevelMaxRuns > 0 || topLevelMaxTurnCacheMisses > 0 {
		return "", &EngineConfig{
			MaxTurns:           topLevelMaxTurns,
			MaxToolDenials:     topLevelMaxToolDenials,
			MaxRuns:            topLevelMaxRuns,
			MaxTurnCacheMisses: topLevelMaxTurnCacheMisses,
			MaxAICredits:       topLevelMaxAICredits,
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
		return engine, nil
	}

	engineLog.Printf("Failed to find engine for setting %s: %v", engineSetting, err)

	validEngines := c.engineRegistry.GetSupportedEngines()
	suggestions := parser.FindClosestMatches(engineSetting, validEngines, 1)
	enginesStr := strings.Join(validEngines, ", ")

	errMsg := fmt.Sprintf("invalid engine: %s. Valid engines are: %s.\n\nExample:\nengine: copilot\n\nSee: %s",
		engineSetting, enginesStr, constants.DocsEnginesURL)
	if len(suggestions) > 0 {
		errMsg = fmt.Sprintf("invalid engine: %s. Valid engines are: %s.\n\nDid you mean: %s?\n\nExample:\nengine: copilot\n\nSee: %s",
			engineSetting, enginesStr, suggestions[0], constants.DocsEnginesURL)
	}

	return nil, errors.New(errMsg)
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

// applyEngineAuthEnv populates config.Env with AWF_AUTH_* environment variables
// derived from config.Auth. Existing config.Env values take precedence so users
// can explicitly override auth-derived values via engine.env.
func applyEngineAuthEnv(config *EngineConfig) {
	if config == nil || config.Auth == nil {
		return
	}
	if config.Env == nil {
		config.Env = make(map[string]string)
	}

	if config.Auth.Type != "" {
		if _, exists := config.Env["AWF_AUTH_TYPE"]; !exists {
			config.Env["AWF_AUTH_TYPE"] = config.Auth.Type
		}
	}
	if config.Auth.Audience != "" {
		if _, exists := config.Env["AWF_AUTH_OIDC_AUDIENCE"]; !exists {
			config.Env["AWF_AUTH_OIDC_AUDIENCE"] = config.Auth.Audience
		}
	}
	if config.Auth.AzureTenantID != "" {
		if _, exists := config.Env["AWF_AUTH_AZURE_TENANT_ID"]; !exists {
			config.Env["AWF_AUTH_AZURE_TENANT_ID"] = config.Auth.AzureTenantID
		}
	}
	if config.Auth.AzureClientID != "" {
		if _, exists := config.Env["AWF_AUTH_AZURE_CLIENT_ID"]; !exists {
			config.Env["AWF_AUTH_AZURE_CLIENT_ID"] = config.Auth.AzureClientID
		}
	}
	if config.Auth.AzureScope != "" {
		if _, exists := config.Env["AWF_AUTH_AZURE_SCOPE"]; !exists {
			config.Env["AWF_AUTH_AZURE_SCOPE"] = config.Auth.AzureScope
		}
	}
	if config.Auth.AzureCloud != "" {
		if _, exists := config.Env["AWF_AUTH_AZURE_CLOUD"]; !exists {
			config.Env["AWF_AUTH_AZURE_CLOUD"] = config.Auth.AzureCloud
		}
	}
	if config.Auth.Provider != "" {
		if _, exists := config.Env["AWF_AUTH_PROVIDER"]; !exists {
			config.Env["AWF_AUTH_PROVIDER"] = config.Auth.Provider
		}
	}
	if config.Auth.AnthropicFederationRuleID != "" {
		if _, exists := config.Env["AWF_AUTH_ANTHROPIC_FEDERATION_RULE_ID"]; !exists {
			config.Env["AWF_AUTH_ANTHROPIC_FEDERATION_RULE_ID"] = config.Auth.AnthropicFederationRuleID
		}
	}
	if config.Auth.AnthropicOrganizationID != "" {
		if _, exists := config.Env["AWF_AUTH_ANTHROPIC_ORGANIZATION_ID"]; !exists {
			config.Env["AWF_AUTH_ANTHROPIC_ORGANIZATION_ID"] = config.Auth.AnthropicOrganizationID
		}
	}
	if config.Auth.AnthropicServiceAccountID != "" {
		if _, exists := config.Env["AWF_AUTH_ANTHROPIC_SERVICE_ACCOUNT_ID"]; !exists {
			config.Env["AWF_AUTH_ANTHROPIC_SERVICE_ACCOUNT_ID"] = config.Auth.AnthropicServiceAccountID
		}
	}
	if config.Auth.AnthropicWorkspaceID != "" {
		if _, exists := config.Env["AWF_AUTH_ANTHROPIC_WORKSPACE_ID"]; !exists {
			config.Env["AWF_AUTH_ANTHROPIC_WORKSPACE_ID"] = config.Auth.AnthropicWorkspaceID
		}
	}
}
