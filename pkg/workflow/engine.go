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
	LLMProvider        string // Inference provider override for this engine (engine.provider / engine.model-provider)
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
	Driver             string // Custom driver script filename or command. For the copilot engine (engine.driver), supports .js/.cjs/.mjs (Node.js), .py (Python), .ts/.mts (TypeScript), .rb (Ruby), or a bare command name. For the pi engine (engine.driver), supports .js/.cjs/.mjs or a bare basename resolved from the setup-action directory.
	Env                map[string]string
	Auth               *EngineAuthConfig // Engine-level auth config (mapped to AWF_AUTH_* env vars for API proxy sidecar auth)
	Config             string
	Args               []string
	Agent              string // Agent identifier for copilot --agent flag (copilot engine only)
	APITarget          string // Custom API endpoint hostname (e.g., "api.acme.ghe.com" or "api.enterprise.githubcopilot.com")
	Bare               bool   // When true, disables automatic loading of context/instructions (copilot: --no-custom-instructions, claude: --bare, codex: --no-system-prompt, gemini: GEMINI_SYSTEM_MD=/dev/null)
	// Inline definition fields (populated when engine.runtime is specified in frontmatter)
	IsInlineDefinition bool   // true when the engine is defined inline via engine.runtime + optional engine.provider
	InlineProviderID   string // engine.provider.id  (e.g. "openai", "anthropic")

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

	// Cwd is a templatable string that overrides the working directory for the engine's
	// spawned process. When set, it is passed as GH_AW_ENGINE_CWD to the execution
	// environment. JS harness engines read this variable in preference to GITHUB_WORKSPACE;
	// non-harness engines use it as the target of the shell-level cd prefix.
	// Defaults to the repository workspace (GITHUB_WORKSPACE) when empty.
	Cwd string

	// Harness retry policy fields — templatable integers (literal value or ${{ expr }}).
	// When set, the value is injected as the corresponding GH_AW_HARNESS_* env var so
	// that all harness scripts (copilot, claude, codex) can read it from the environment.
	// The harness falls back to its built-in default when the env var is absent.
	// These are populated from the engine.harness sub-object keys.
	HarnessMaxRetries        string // engine.harness.max-retries        → GH_AW_HARNESS_MAX_RETRIES
	HarnessInitialDelayMs    string // engine.harness.initial-delay-ms   → GH_AW_HARNESS_INITIAL_DELAY_MS
	HarnessBackoffMultiplier string // engine.harness.backoff-multiplier → GH_AW_HARNESS_BACKOFF_MULTIPLIER
	HarnessMaxDelayMs        string // engine.harness.max-delay-ms       → GH_AW_HARNESS_MAX_DELAY_MS
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
	limits := extractTopLevelEngineLimits(frontmatter)
	if engine, exists := frontmatter["engine"]; exists {
		engineLog.Print("Extracting engine configuration from frontmatter")
		switch typed := engine.(type) {
		case string:
			engineLog.Printf("Found engine in string format: %s", typed)
			return typed, limits.newConfig(typed)
		case map[string]any:
			engineLog.Print("Found engine in object format, parsing configuration")
			return extractEngineObjectConfig(typed, limits)
		}
	}
	if limits.hasAny() {
		return "", limits.newConfig("")
	}
	engineLog.Print("No engine configuration found in frontmatter")
	return "", nil
}

type topLevelEngineLimits struct {
	maxTurns           string
	maxToolDenials     string
	maxRuns            int
	maxTurnCacheMisses int
	maxAICredits       int64
}

func extractTopLevelEngineLimits(frontmatter map[string]any) topLevelEngineLimits {
	maxRuns := parseMaxRunsValue(frontmatter["max-turns"])
	if maxRuns == 0 {
		maxRuns = parseMaxRunsValue(frontmatter["max-runs"])
	}
	return topLevelEngineLimits{
		maxTurns:           parseMaxTurnsValue(frontmatter["max-turns"]),
		maxToolDenials:     parseMaxToolDenialsValue(frontmatter["max-tool-denials"]),
		maxRuns:            maxRuns,
		maxTurnCacheMisses: parseMaxTurnCacheMissesValue(frontmatter["max-turn-cache-misses"]),
		maxAICredits:       parseMaxAICreditsValue(frontmatter["max-ai-credits"]),
	}
}

func (l topLevelEngineLimits) hasAny() bool {
	return l.maxTurns != "" || l.maxToolDenials != "" || l.maxAICredits != 0 || l.maxRuns > 0 || l.maxTurnCacheMisses > 0
}

func (l topLevelEngineLimits) newConfig(id string) *EngineConfig {
	return &EngineConfig{
		ID:                 id,
		MaxTurns:           l.maxTurns,
		MaxToolDenials:     l.maxToolDenials,
		MaxRuns:            l.maxRuns,
		MaxTurnCacheMisses: l.maxTurnCacheMisses,
		MaxAICredits:       l.maxAICredits,
	}
}

func extractEngineObjectConfig(engineObj map[string]any, limits topLevelEngineLimits) (string, *EngineConfig) {
	config := &EngineConfig{}
	if runtime, hasRuntime := engineObj["runtime"]; hasRuntime {
		parseInlineEngineConfig(config, engineObj, runtime, limits)
		engineLog.Printf("Extracted inline engine definition: runtimeID=%s, providerID=%s", config.ID, config.InlineProviderID)
		return config.ID, config
	}
	parseEngineIdentityFields(config, engineObj)
	parseEngineProviderFields(config, engineObj)
	parseEngineLimitFields(config, engineObj, limits)
	parseEngineConcurrency(config, engineObj)
	parseEngineCommandFields(config, engineObj)
	parseEngineEnvAndAuth(config, engineObj)
	parseEngineArgsAndAgent(config, engineObj)
	parseEngineMCPConfig(config, engineObj)
	parseEngineExtensions(config, engineObj)
	applyEngineFinalFields(config, engineObj, limits)
	engineLog.Printf("Extracted engine configuration: ID=%s", config.ID)
	return config.ID, config
}

func parseInlineEngineConfig(config *EngineConfig, engineObj map[string]any, runtime any, limits topLevelEngineLimits) {
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
	parseInlineProviderConfig(config, engineObj)
	parseInlineSharedFields(config, engineObj)
	config.MaxTurns = limits.maxTurns
	config.MaxToolDenials = limits.maxToolDenials
	config.MaxRuns = limits.maxRuns
	config.MaxTurnCacheMisses = limits.maxTurnCacheMisses
	config.MaxAICredits = limits.maxAICredits
}

func parseInlineProviderConfig(config *EngineConfig, engineObj map[string]any) {
	provider, hasProvider := engineObj["provider"]
	if !hasProvider {
		return
	}
	switch providerTyped := provider.(type) {
	case string:
		config.InlineProviderID = strings.ToLower(strings.TrimSpace(providerTyped))
	case map[string]any:
		if id, ok := providerTyped["id"].(string); ok {
			config.InlineProviderID = id
		}
		if model, ok := providerTyped["model"].(string); ok {
			config.Model = model
		}
		parseInlineProviderAuth(config, providerTyped)
		if request, hasRequest := providerTyped["request"]; hasRequest {
			if requestObj, ok := request.(map[string]any); ok {
				config.InlineProviderRequest = parseRequestShape(requestObj)
			}
		}
	}
}

func parseInlineProviderAuth(config *EngineConfig, providerTyped map[string]any) {
	auth, hasAuth := providerTyped["auth"]
	if !hasAuth {
		return
	}
	authObj, ok := auth.(map[string]any)
	if !ok {
		return
	}
	authDef := parseAuthDefinition(authObj)
	if authDef.Strategy != "" || authDef.Secret != "" || authDef.TokenURL != "" ||
		authDef.ClientIDRef != "" || authDef.ClientSecretRef != "" ||
		authDef.HeaderName != "" || authDef.TokenField != "" {
		config.InlineProviderAuth = authDef
	}
}

func parseInlineSharedFields(config *EngineConfig, engineObj map[string]any) {
	if bare, hasBare := engineObj["bare"]; hasBare {
		if bareBool, ok := bare.(bool); ok {
			config.Bare = bareBool
			engineLog.Printf("Extracted bare mode (inline): %v", config.Bare)
		}
	}
	if permissionMode, hasPermissionMode := engineObj["permission-mode"]; hasPermissionMode {
		if permissionModeStr, ok := permissionMode.(string); ok {
			config.PermissionMode = permissionModeStr
		}
	}
}

func parseEngineIdentityFields(config *EngineConfig, engineObj map[string]any) {
	if id, hasID := engineObj["id"]; hasID {
		if idStr, ok := id.(string); ok {
			config.ID = idStr
		}
	}
	if version, hasVersion := engineObj["version"]; hasVersion {
		config.Version = stringutil.ParseVersionValue(version)
	}
	if model, hasModel := engineObj["model"]; hasModel {
		if modelStr, ok := model.(string); ok {
			config.Model = modelStr
		}
	}
}

func parseEngineProviderFields(config *EngineConfig, engineObj map[string]any) {
	if providerValue, hasProvider := engineObj["model-provider"]; hasProvider {
		if providerStr, ok := providerValue.(string); ok {
			config.LLMProvider = strings.ToLower(strings.TrimSpace(providerStr))
		}
	}
	if providerValue, hasProvider := engineObj["provider"]; hasProvider && !config.IsInlineDefinition {
		if providerStr, ok := providerValue.(string); ok {
			config.LLMProvider = strings.ToLower(strings.TrimSpace(providerStr))
		}
	}
	if permissionMode, hasPermissionMode := engineObj["permission-mode"]; hasPermissionMode {
		if permissionModeStr, ok := permissionMode.(string); ok {
			config.PermissionMode = permissionModeStr
		}
	}
}

func parseEngineLimitFields(config *EngineConfig, engineObj map[string]any, limits topLevelEngineLimits) {
	if maxTurns, hasMaxTurns := engineObj["max-turns"]; hasMaxTurns {
		config.MaxTurns = parseMaxTurnsValue(maxTurns)
	}
	if limits.maxTurns != "" {
		config.MaxTurns = limits.maxTurns
	}
	if limits.maxToolDenials != "" {
		config.MaxToolDenials = limits.maxToolDenials
	}
	if maxCont, hasMaxCont := engineObj["max-continuations"]; hasMaxCont {
		if val, ok := typeutil.ParseIntValue(maxCont); ok {
			config.MaxContinuations = val
		} else if maxContStr, ok := maxCont.(string); ok {
			if parsed, err := strconv.Atoi(maxContStr); err == nil {
				config.MaxContinuations = parsed
			}
		}
	}
}

func parseEngineConcurrency(config *EngineConfig, engineObj map[string]any) {
	concurrency, hasConcurrency := engineObj["concurrency"]
	if !hasConcurrency {
		return
	}
	if concurrencyStr, ok := concurrency.(string); ok {
		config.Concurrency = fmt.Sprintf("concurrency:\n  group: \"%s\"", concurrencyStr)
		return
	}
	if concurrencyObj, ok := concurrency.(map[string]any); ok {
		config.Concurrency = renderEngineConcurrencyObject(concurrencyObj)
	}
}

func renderEngineConcurrencyObject(concurrencyObj map[string]any) string {
	var parts []string
	if group, hasGroup := concurrencyObj["group"]; hasGroup {
		if groupStr, ok := group.(string); ok {
			parts = append(parts, fmt.Sprintf("concurrency:\n  group: \"%s\"", groupStr))
		}
	}
	if cancel, hasCancel := concurrencyObj["cancel-in-progress"]; hasCancel {
		if cancelBool, ok := cancel.(bool); ok && cancelBool && len(parts) > 0 {
			parts[0] += "\n  cancel-in-progress: true"
		}
	}
	if queue, hasQueue := concurrencyObj["queue"]; hasQueue {
		if queueStr, ok := queue.(string); ok && queueStr != "" && len(parts) > 0 {
			parts[0] += "\n  queue: " + queueStr
		}
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func parseEngineCommandFields(config *EngineConfig, engineObj map[string]any) {
	if userAgent, hasUserAgent := engineObj["user-agent"]; hasUserAgent {
		if userAgentStr, ok := userAgent.(string); ok {
			config.UserAgent = userAgentStr
		}
	}
	if command, hasCommand := engineObj["command"]; hasCommand {
		if commandStr, ok := command.(string); ok {
			config.Command = commandStr
		}
	}
	parseEngineHarness(config, engineObj)
	if driver, hasDriver := engineObj["driver"]; hasDriver {
		if driverStr, ok := driver.(string); ok {
			config.Driver = driverStr
			engineLog.Printf("Extracted engine.driver: %s", driverStr)
		}
	}
}

func parseEngineHarness(config *EngineConfig, engineObj map[string]any) {
	harness, hasHarness := engineObj["harness"]
	if !hasHarness {
		return
	}
	switch h := harness.(type) {
	case string:
		config.HarnessScript = h
	case map[string]any:
		if use, ok := h["use"].(string); ok {
			config.HarnessScript = use
		}
		if v, ok := h["max-retries"]; ok {
			config.HarnessMaxRetries = parseHarnessMaxRetriesValue(v)
		}
		if v, ok := h["initial-delay-ms"]; ok {
			config.HarnessInitialDelayMs = parseMaxTurnsValue(v)
		}
		if v, ok := h["backoff-multiplier"]; ok {
			config.HarnessBackoffMultiplier = parseMaxTurnsValue(v)
		}
		if v, ok := h["max-delay-ms"]; ok {
			config.HarnessMaxDelayMs = parseMaxTurnsValue(v)
		}
	}
}

func parseEngineEnvAndAuth(config *EngineConfig, engineObj map[string]any) {
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
	if auth, hasAuth := engineObj["auth"]; hasAuth {
		if authObj, ok := auth.(map[string]any); ok {
			config.Auth = parseEngineAuthConfig(authObj)
			applyEngineAuthEnv(config)
		}
	}
	if configField, hasConfig := engineObj["config"]; hasConfig {
		if configStr, ok := configField.(string); ok {
			config.Config = configStr
		}
	}
}

func parseEngineArgsAndAgent(config *EngineConfig, engineObj map[string]any) {
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
	if agent, hasAgent := engineObj["agent"]; hasAgent {
		if agentStr, ok := agent.(string); ok {
			config.Agent = agentStr
			engineLog.Printf("Extracted agent identifier: %s", agentStr)
		}
	}
	parseEngineAPITargetAndBare(config, engineObj)
}

func parseEngineAPITargetAndBare(config *EngineConfig, engineObj map[string]any) {
	if apiTarget, hasAPITarget := engineObj["api-target"]; hasAPITarget {
		if apiTargetStr, ok := apiTarget.(string); ok && apiTargetStr != "" {
			config.APITarget = apiTargetStr
			engineLog.Printf("Extracted api-target: %s", apiTargetStr)
		}
	}
	if bare, hasBare := engineObj["bare"]; hasBare {
		if bareBool, ok := bare.(bool); ok {
			config.Bare = bareBool
			engineLog.Printf("Extracted bare mode: %v", config.Bare)
		}
	}
}

func parseEngineMCPConfig(config *EngineConfig, engineObj map[string]any) {
	mcpVal, hasMCP := engineObj["mcp"]
	if !hasMCP {
		return
	}
	mcpObj, ok := mcpVal.(map[string]any)
	if !ok {
		return
	}
	if stVal, hasSessionTimeout := mcpObj["session-timeout"]; hasSessionTimeout {
		if stStr, ok := stVal.(string); ok && stStr != "" {
			config.MCPSessionTimeout = stStr
			engineLog.Printf("Extracted engine.mcp.session-timeout: %s", config.MCPSessionTimeout)
		}
	}
	if ttVal, hasToolTimeout := mcpObj["tool-timeout"]; hasToolTimeout {
		if ttStr, ok := ttVal.(string); ok && ttStr != "" {
			config.MCPToolTimeout = ttStr
			engineLog.Printf("Extracted engine.mcp.tool-timeout: %s", config.MCPToolTimeout)
		}
	}
}

func parseEngineExtensions(config *EngineConfig, engineObj map[string]any) {
	extVal, hasExt := engineObj["extensions"]
	if !hasExt {
		return
	}
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

func applyEngineFinalFields(config *EngineConfig, engineObj map[string]any, limits topLevelEngineLimits) {
	if limits.maxTurns != "" {
		config.MaxTurns = limits.maxTurns
	}
	config.MaxRuns = limits.maxRuns
	config.MaxTurnCacheMisses = limits.maxTurnCacheMisses
	config.MaxAICredits = limits.maxAICredits
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
	if cwdVal, hasCwd := engineObj["cwd"]; hasCwd {
		if cwdStr, ok := cwdVal.(string); ok && cwdStr != "" {
			config.Cwd = cwdStr
			engineLog.Printf("Extracted engine.cwd: %s", config.Cwd)
		}
	}
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

	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_TYPE", config.Auth.Type)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_OIDC_AUDIENCE", config.Auth.Audience)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_AZURE_TENANT_ID", config.Auth.AzureTenantID)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_AZURE_CLIENT_ID", config.Auth.AzureClientID)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_AZURE_SCOPE", config.Auth.AzureScope)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_AZURE_CLOUD", config.Auth.AzureCloud)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_PROVIDER", config.Auth.Provider)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_ANTHROPIC_FEDERATION_RULE_ID", config.Auth.AnthropicFederationRuleID)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_ANTHROPIC_ORGANIZATION_ID", config.Auth.AnthropicOrganizationID)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_ANTHROPIC_SERVICE_ACCOUNT_ID", config.Auth.AnthropicServiceAccountID)
	setEngineAuthEnvIfMissing(config.Env, "AWF_AUTH_ANTHROPIC_WORKSPACE_ID", config.Auth.AnthropicWorkspaceID)
}

func setEngineAuthEnvIfMissing(env map[string]string, key, value string) {
	if value == "" {
		return
	}
	if _, exists := env[key]; !exists {
		env[key] = value
	}
}
