// This file provides AWF (Agentic Workflow Firewall) configuration file generation.
//
// AWF supports loading configuration from a JSON/YAML file via the --config flag.
// Generating a config file rather than a long list of CLI flags improves:
//   - Readability: structured JSON is easier to audit than a one-liner flag list
//   - Correctness: complex values (JSON objects) avoid shell escaping issues
//   - Composability: config files can be layered and merged
//   - Extensibility: new features add JSON fields, not more argv flags
//
// # Config File Schema
//
// The generated config file follows the AWF config file format:
// Cross-reference: /specs/awf-config-sources-spec.md documents the canonical
// gh-aw-firewall spec/schema sources that MUST be checked when evolving mappings.
//
//	{
//	  "$schema": "https://github.com/github/gh-aw-firewall/releases/download/vX.Y.Z/awf-config.schema.json",
//	  "network": {
//	    "allowDomains": ["github.com", "api.github.com"],
//	    "blockDomains": ["ads.example.com"]
//	  },
//	  "apiProxy": {
//	    "enabled": true,
//	    "targets": {
//	      "openai":    { "host": "api.openai.com" },
//	      "anthropic": { "host": "api.anthropic.com" },
//	      "copilot":   { "host": "api.githubcopilot.com" },
//	      "antigravity":    { "host": "generativelanguage.googleapis.com" }
//	    },
//	    "models": {
//	      "sonnet": ["mygateway/*sonnet*"],
//	      "":       ["sonnet", "gpt-5-mini"]
//	    }
//	  },
//	  "container": {
//	    "imageTag": "0.25.29,squid=sha256:..."
//	  }
//	}
//
// # Runtime Usage
//
// The config file is written to ${RUNNER_TEMP}/gh-aw/awf-config.json before the
// AWF invocation, and referenced via: awf --config "${RUNNER_TEMP}/gh-aw/awf-config.json"
//
// Flags not yet represented in the config schema (--env-all, --exclude-env, --mount,
// --container-workdir, --log-level, --proxy-logs-dir, --audit-dir, --enable-host-access,
// --allow-host-ports, --skip-pull, --tty, --difc-proxy-host, --difc-proxy-ca-cert,
// --ssl-bump, --memory-limit, --diagnostic-logs) remain as CLI flags.

package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/jsonutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/awf-config.schema.json
var awfConfigSchema string

var awfConfigLog = logger.New("workflow:awf_config")

// Cached compiled AWF config schema to avoid recompiling on every validation.
var (
	compiledAWFConfigSchemaOnce sync.Once
	compiledAWFConfigSchema     *jsonschema.Schema
	awfConfigSchemaCompileError error
)

// getCompiledAWFConfigSchema returns the compiled AWF config schema, compiling once and caching.
func getCompiledAWFConfigSchema() (*jsonschema.Schema, error) {
	compiledAWFConfigSchemaOnce.Do(func() {
		awfConfigLog.Print("Compiling AWF config schema (first time)")
		var schemaDoc any
		if err := json.Unmarshal([]byte(awfConfigSchema), &schemaDoc); err != nil {
			awfConfigSchemaCompileError = fmt.Errorf("failed to parse embedded AWF config schema: %w", err)
			return
		}
		loader := jsonschema.NewCompiler()
		schemaURL := fmt.Sprintf("https://github.com/github/gh-aw-firewall/releases/download/%s/awf-config.schema.json", constants.DefaultFirewallVersion)
		if err := loader.AddResource(schemaURL, schemaDoc); err != nil {
			awfConfigSchemaCompileError = fmt.Errorf("failed to add AWF config schema resource: %w", err)
			return
		}
		schema, err := loader.Compile(schemaURL)
		if err != nil {
			awfConfigSchemaCompileError = fmt.Errorf("failed to compile AWF config schema: %w", err)
			return
		}
		compiledAWFConfigSchema = schema
		awfConfigLog.Print("AWF config schema compiled successfully")
	})
	return compiledAWFConfigSchema, awfConfigSchemaCompileError
}

// validateAWFConfigJSON validates the provided AWF config JSON string against the
// embedded AWF config schema. Returns nil if validation passes.
func validateAWFConfigJSON(configJSON string) error {
	schema, err := getCompiledAWFConfigSchema()
	if err != nil {
		return err
	}
	var doc any
	if err := json.Unmarshal([]byte(configJSON), &doc); err != nil {
		return fmt.Errorf("failed to parse AWF config JSON: %w", err)
	}
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("AWF config schema validation failed: %w", err)
	}
	return nil
}

// AWFConfigFile represents the AWF configuration file schema.
// This is the top-level structure written to awf-config.json.
type AWFConfigFile struct {
	// Schema is the JSON schema reference for IDE auto-complete support.
	Schema string `json:"$schema,omitempty"`

	// Network contains network egress control configuration.
	Network *AWFNetworkConfig `json:"network,omitempty"`

	// APIProxy contains API proxy (LLM gateway) configuration.
	APIProxy *AWFAPIProxyConfig `json:"apiProxy,omitempty"`

	// Container contains container execution configuration.
	Container *AWFContainerConfig `json:"container,omitempty"`
}

// AWFNetworkConfig is the "network" section of the AWF config file.
// It maps to the --allow-domains and --block-domains CLI flags.
type AWFNetworkConfig struct {
	// AllowDomains is the list of allowed egress domains.
	// Supports wildcards (e.g. "*.github.com") and exact matches.
	// Maps to: --allow-domains <comma-separated>
	AllowDomains []string `json:"allowDomains,omitempty"`

	// BlockDomains is the list of explicitly blocked egress domains.
	// Maps to: --block-domains <comma-separated>
	BlockDomains []string `json:"blockDomains,omitempty"`
}

// AWFAPIProxyConfig is the "apiProxy" section of the AWF config file.
// It maps to the --enable-api-proxy and --*-api-target CLI flags.
type AWFAPIProxyConfig struct {
	// Enabled enables the API proxy sidecar for LLM gateway credential isolation.
	// Maps to: --enable-api-proxy
	Enabled bool `json:"enabled"`

	// EnableTokenSteering enables budget-warning system message injection near ET budget exhaustion.
	EnableTokenSteering bool `json:"enableTokenSteering,omitempty"`

	// MaxRuns is the maximum number of LLM invocations allowed for a run.
	MaxRuns int `json:"maxRuns,omitempty"`

	// MaxEffectiveTokens is the explicit ET budget enforced by the API proxy.
	MaxEffectiveTokens int64 `json:"maxEffectiveTokens,omitempty"`

	// ModelFallback configures the model fallback policy for unresolved model selections.
	// When nil, the AWF default (enabled=true, strategy=middle_power) is used.
	// Set enabled=false to prevent AWF from silently rewriting deployment names, which
	// is needed for BYOK Azure OpenAI deployments where rewriting causes HTTP 404.
	ModelFallback *AWFModelFallbackConfig `json:"modelFallback,omitempty"`

	// ModelMultipliers configures per-model ET accounting multipliers in AWF.
	ModelMultipliers map[string]float64 `json:"modelMultipliers,omitempty"`

	// Targets holds per-provider API target overrides.
	// Supported keys: "openai", "anthropic", "copilot", "gemini"
	// The "gemini" target is also used for Antigravity engine routing.
	Targets map[string]*AWFAPITargetConfig `json:"targets,omitempty"`

	// Models contains model alias and fallback policy definitions.
	// Keys are alias names (empty string "" = default policy); values are ordered
	// lists of vendor/modelid patterns or other alias names to try in sequence.
	// AWF resolves aliases recursively; loops are not permitted.
	// Per the AWF config schema, this lives under apiProxy.models.
	Models map[string][]string `json:"models,omitempty"`
}

// AWFModelFallbackConfig is the "apiProxy.modelFallback" section of the AWF config file.
// It controls whether model fallback is enabled for unresolved model selections.
type AWFModelFallbackConfig struct {
	// Enabled controls whether middle-power fallback is applied when model resolution fails.
	// It accepts literal booleans and GitHub Actions expressions. A nil value omits the field,
	// letting AWF use its default.
	Enabled *TemplatableBool `json:"enabled,omitempty"`
}

// AWFAPITargetConfig is a single API proxy target entry.
// Maps to: --<provider>-api-target <host>
type AWFAPITargetConfig struct {
	// Host is the hostname (and optional port) of the API endpoint.
	Host string `json:"host,omitempty"`

	// AuthHeader is the custom authentication header name sent with API requests.
	// When set, the raw API key is sent as "<authHeader>: <key>" instead of the
	// provider default (e.g. "Authorization: ******" for OpenAI, or
	// "x-api-key: <key>" for Anthropic). This supports gateways like Azure OpenAI
	// that require "api-key: <rawkey>" in place of the standard provider scheme.
	// Maps to: --openai-api-auth-header / --anthropic-api-auth-header
	AuthHeader string `json:"authHeader,omitempty"`
}

// AWFContainerConfig is the "container" section of the AWF config file.
// It maps to the --image-tag CLI flag.
type AWFContainerConfig struct {
	// ImageTag is the pinned AWF Docker image tag, with optional digest metadata.
	// Format: "<tag>" or "<tag>,squid=sha256:...,agent=sha256:..."
	// Maps to: --image-tag <value>
	ImageTag string `json:"imageTag,omitempty"`
}

// buildAWFConfigSchemaURL returns the release-pinned JSON schema URL for the AWF config file.
// The URL is versioned so that schema validation tools always reference the exact schema
// that matches the AWF binary being used. When DefaultFirewallVersion is bumped the URL
// automatically tracks the new release.
//
// If firewallConfig carries an explicit version (e.g. sandbox.agent.version) that version
// is used; otherwise DefaultFirewallVersion is used.
func buildAWFConfigSchemaURL(firewallConfig *FirewallConfig) string {
	version := string(constants.DefaultFirewallVersion)
	if firewallConfig != nil && firewallConfig.Version != "" {
		version = firewallConfig.Version
	}
	// Special-case "latest": the GitHub Releases /latest/download/ shortcut serves
	// assets from the most recent release without requiring a tag in the path.
	if strings.EqualFold(version, "latest") {
		return "https://github.com/github/gh-aw-firewall/releases/latest/download/awf-config.schema.json"
	}
	// Ensure version has the 'v' prefix required by GitHub release tag URLs.
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return fmt.Sprintf("https://github.com/github/gh-aw-firewall/releases/download/%s/awf-config.schema.json", version)
}

// BuildAWFConfigJSON generates a compact JSON config file for AWF from the provided
// command configuration. The JSON is single-line (no indentation) for safe embedding
// in a shell printf command.
//
// The caller is responsible for writing the returned JSON to disk at the path expected
// by the AWF --config flag. See BuildAWFCommand for how this is wired together.
func BuildAWFConfigJSON(config AWFCommandConfig) (string, error) {
	awfConfigLog.Printf("Building AWF config JSON: engine=%s, allowed_domains=%q", config.EngineName, config.AllowedDomains)

	// Resolve firewall config once — used for both the schema URL and the container image tag.
	firewallConfig := getFirewallConfig(config.WorkflowData)

	awfConfig := AWFConfigFile{
		Schema: buildAWFConfigSchemaURL(firewallConfig),
	}

	// ── Network section ──────────────────────────────────────────────────────
	if config.AllowedDomains != "" {
		allowList := splitDomainList(config.AllowedDomains)
		awfConfig.Network = &AWFNetworkConfig{
			AllowDomains: allowList,
		}
		awfConfigLog.Printf("Network section: %d allowed domains", len(allowList))

		// Blocked domains (if configured in the workflow)
		if config.WorkflowData != nil {
			blockedDomainsStr := formatBlockedDomains(config.WorkflowData.NetworkPermissions)
			if blockedDomainsStr != "" {
				blockList := splitDomainList(blockedDomainsStr)
				awfConfig.Network.BlockDomains = blockList
				awfConfigLog.Printf("Network section: %d blocked domains", len(blockList))
			}
		}
	}

	// ── API proxy section ─────────────────────────────────────────────────────
	maxEffectiveTokens := compilerenv.ResolveDefaultMaxEffectiveTokens(constants.DefaultMaxEffectiveTokens)
	maxRuns := constants.DefaultMaxRuns
	if config.WorkflowData != nil && config.WorkflowData.EngineConfig != nil {
		if config.WorkflowData.EngineConfig.MaxEffectiveTokens != 0 {
			maxEffectiveTokens = config.WorkflowData.EngineConfig.MaxEffectiveTokens
		}
		maxRuns = config.WorkflowData.EngineConfig.GetMaxRuns()
	}

	// Token steering is enabled by default; a negative max-effective-tokens value
	// disables both budget enforcement and token steering.
	enableTokenSteering := maxEffectiveTokens >= 0
	if maxEffectiveTokens < 0 {
		// Negative signals "disabled" — omit the budget from the AWF config.
		maxEffectiveTokens = 0
	}

	apiProxy := &AWFAPIProxyConfig{
		Enabled:             true,
		MaxRuns:             maxRuns,
		MaxEffectiveTokens:  maxEffectiveTokens,
		EnableTokenSteering: enableTokenSteering && awfSupportsTokenSteering(firewallConfig),
	}

	if !enableTokenSteering {
		awfConfigLog.Printf("Skipping apiProxy.enableTokenSteering: max-effective-tokens is negative (disabled)")
	} else if !awfSupportsTokenSteering(firewallConfig) {
		awfConfigLog.Printf("Skipping apiProxy.enableTokenSteering: AWF version %q requires at least %s", getAWFImageTag(firewallConfig), constants.AWFTokenSteeringMinVersion)
	}

	if modelMultipliers := extractModelMultipliers(config.WorkflowData); len(modelMultipliers) > 0 {
		apiProxy.ModelMultipliers = modelMultipliers
		awfConfigLog.Printf("API proxy: %d model multipliers configured", len(apiProxy.ModelMultipliers))
	}

	if mf := extractModelFallback(config.WorkflowData); mf != nil {
		apiProxy.ModelFallback = mf
		enabledDisplay := "<unset>"
		if mf.Enabled != nil {
			enabledDisplay = mf.Enabled.String()
		}
		awfConfigLog.Printf("API proxy: modelFallback configured: enabled=%s", enabledDisplay)
	}

	targets := map[string]*AWFAPITargetConfig{}

	if openaiTarget := extractAPITargetHost(config.WorkflowData, "OPENAI_BASE_URL"); openaiTarget != "" {
		targets["openai"] = &AWFAPITargetConfig{Host: openaiTarget}
		awfConfigLog.Printf("API proxy: custom openai target=%s", openaiTarget)
	}
	if anthropicTarget := extractAPITargetHost(config.WorkflowData, "ANTHROPIC_BASE_URL"); anthropicTarget != "" {
		targets["anthropic"] = &AWFAPITargetConfig{Host: anthropicTarget}
		awfConfigLog.Printf("API proxy: custom anthropic target=%s", anthropicTarget)
	}

	// Apply authHeader overrides from sandbox.agent.targets frontmatter.
	// These are independent of the host/env-var settings: authHeader can be set
	// even when no custom host is configured.
	for _, provider := range []string{"openai", "anthropic"} {
		authHeader := extractAPITargetAuthHeader(config.WorkflowData, provider)
		if authHeader == "" {
			continue
		}
		if existing, ok := targets[provider]; ok {
			existing.AuthHeader = authHeader
		} else {
			targets[provider] = &AWFAPITargetConfig{AuthHeader: authHeader}
		}
		awfConfigLog.Printf("API proxy: custom %s authHeader=%s", provider, authHeader)
	}
	if copilotTarget := GetCopilotAPITarget(config.WorkflowData); copilotTarget != "" {
		targets["copilot"] = &AWFAPITargetConfig{Host: copilotTarget}
		awfConfigLog.Printf("API proxy: custom copilot target=%s", copilotTarget)
	}
	if antigravityTarget := GetAntigravityAPITarget(config.WorkflowData, config.EngineName); antigravityTarget != "" {
		// Route the Antigravity-resolved API target through the "gemini" provider key
		// to match AWF's supported target providers.
		geminiTarget := GetGeminiAPITarget(config.WorkflowData, config.EngineName)
		if geminiTarget != "" && geminiTarget != antigravityTarget {
			awfConfigLog.Printf("API proxy: overriding gemini target %s with antigravity target %s; configure only one of GEMINI_API_BASE_URL or ANTIGRAVITY_API_BASE_URL to avoid ambiguity", geminiTarget, antigravityTarget)
		}
		awfConfigLog.Printf("API proxy: mapped antigravity target to gemini provider target=%s", antigravityTarget)
		targets["gemini"] = &AWFAPITargetConfig{Host: antigravityTarget}
	} else if geminiTarget := GetGeminiAPITarget(config.WorkflowData, config.EngineName); geminiTarget != "" {
		awfConfigLog.Printf("API proxy: custom gemini target=%s", geminiTarget)
		targets["gemini"] = &AWFAPITargetConfig{Host: geminiTarget}
	}

	if len(targets) > 0 {
		apiProxy.Targets = targets
		awfConfigLog.Printf("API proxy: %d custom targets configured", len(targets))
	}

	// ── Models section (nested under apiProxy per AWF config schema) ──────────
	if config.WorkflowData != nil && len(config.WorkflowData.ModelMappings) > 0 {
		apiProxy.Models = config.WorkflowData.ModelMappings
		awfConfigLog.Printf("Models section: %d alias entries", len(config.WorkflowData.ModelMappings))
	}

	awfConfig.APIProxy = apiProxy

	// ── Container section ─────────────────────────────────────────────────────
	awfImageTag := buildAWFImageTagWithDigests(getAWFImageTag(firewallConfig), config.WorkflowData)
	if awfImageTag != "" {
		awfConfig.Container = &AWFContainerConfig{
			ImageTag: awfImageTag,
		}
		awfConfigLog.Printf("Container section: image_tag=%s", awfImageTag)
	}

	jsonStr, err := jsonutil.MarshalCompactNoHTMLEscape(awfConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AWF config to JSON: %w", err)
	}

	awfConfigLog.Printf("AWF config JSON generated: %d bytes", len(jsonStr))

	if config.WorkflowData != nil && config.WorkflowData.ValidateAWFConfig {
		if err := validateAWFConfigJSON(jsonStr); err != nil {
			return "", fmt.Errorf("generated AWF config failed schema validation: %w", err)
		}
	}

	return jsonStr, nil
}

// splitDomainList splits a comma-separated domain string into a deduplicated
// slice. Empty entries are ignored. The order of the original list is preserved for
// non-duplicate entries; this keeps the allow-list deterministic.
func splitDomainList(domains string) []string {
	var result []string
	seen := make(map[string]bool)
	for d := range strings.SplitSeq(domains, ",") {
		d = strings.TrimSpace(d)
		if d != "" && !seen[d] {
			seen[d] = true
			result = append(result, d)
		}
	}
	return result
}

func extractModelMultipliers(workflowData *WorkflowData) map[string]float64 {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.TokenWeights == nil {
		return nil
	}
	if len(workflowData.EngineConfig.TokenWeights.Multipliers) == 0 {
		return nil
	}
	return workflowData.EngineConfig.TokenWeights.Multipliers
}

// extractModelFallback returns an AWFModelFallbackConfig if the workflow has configured
// sandbox.agent.model-fallback, or nil if the field is absent (letting AWF use its default).
func extractModelFallback(workflowData *WorkflowData) *AWFModelFallbackConfig {
	if workflowData == nil {
		return nil
	}
	if workflowData.SandboxConfig == nil {
		return nil
	}
	if workflowData.SandboxConfig.Agent == nil {
		return nil
	}
	mf := workflowData.SandboxConfig.Agent.ModelFallback
	if mf == nil {
		return nil
	}
	return &AWFModelFallbackConfig{
		Enabled: mf,
	}
}
