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
//	  },
//	  "chroot": {
//	    "binariesSourcePath": "/tmp/gh-aw",
//	    "identity": {
//	      "user": "runner",
//	      "uid": 1001,
//	      "gid": 1001,
//	      "home": "/tmp/gh-aw/home"
//	    }
//	  }
//	}
//
// # Runtime Usage
//
// The config file is written to ${RUNNER_TEMP}/gh-aw/awf-config.json before the
// AWF invocation, and referenced via: awf --config "${RUNNER_TEMP}/gh-aw/awf-config.json"
//
// Flags not yet represented in the config schema (--env-all, --exclude-env, --mount,
// --container-workdir, --log-level, --enable-host-access,
// --allow-host-ports, --skip-pull, --tty, --difc-proxy-host, --difc-proxy-ca-cert,
// --ssl-bump, --memory-limit, --diagnostic-logs) remain as CLI flags.
//
// Flags moved to config: --proxy-logs-dir → logging.proxyLogsDir,
// --audit-dir → logging.auditDir, --docker-host-path-prefix → container.dockerHostPathPrefix.
// For ARC/DinD, --proxy-logs-dir and --audit-dir CLI flags still override config at runtime
// (they use ${RUNNER_TEMP} paths that require shell expansion).

package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/jsonutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
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
		schemaURL := fmt.Sprintf("https://github.com/github/gh-aw-firewall/releases/download/%s/awf-config.schema.json", constants.DefaultFirewallVersion)
		compiledAWFConfigSchema, awfConfigSchemaCompileError = compileSchema(awfConfigSchema, schemaURL)
		if awfConfigSchemaCompileError == nil {
			awfConfigLog.Print("AWF config schema compiled successfully")
		}
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
	normalizeTemplatableModelFallbackEnabled(doc)
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("AWF config schema validation failed: %w", err)
	}
	return nil
}

// normalizeTemplatableModelFallbackEnabled adjusts a generated AWF config document
// for compile-time schema validation by coercing modelFallback.enabled GitHub Actions
// expressions to a boolean placeholder. GitHub Actions resolves these expressions at
// runtime before AWF consumes the config.
func normalizeTemplatableModelFallbackEnabled(doc any) {
	root, ok := doc.(map[string]any)
	if !ok {
		return
	}
	apiProxy, ok := root["apiProxy"].(map[string]any)
	if !ok {
		return
	}
	modelFallback, ok := apiProxy["modelFallback"].(map[string]any)
	if !ok {
		return
	}
	enabled, ok := modelFallback["enabled"].(string)
	if !ok || !isExpression(enabled) {
		return
	}
	modelFallback["enabled"] = true
}

// AWFConfigFile represents the AWF configuration file schema.
// This is the top-level structure written to awf-config.json.
type AWFConfigFile struct {
	// Schema is the JSON schema reference for IDE auto-complete support.
	Schema string `json:"$schema,omitempty"`

	// Runner contains runner topology metadata that AWF uses to activate
	// topology-specific behaviors (split-filesystem handling, network isolation,
	// tool cache redirection, sysroot image selection).
	Runner *AWFRunnerConfig `json:"runner,omitempty"`

	// Network contains network egress control configuration.
	Network *AWFNetworkConfig `json:"network,omitempty"`

	// Platform contains GitHub deployment metadata used by AWF auth handling.
	Platform *AWFPlatformConfig `json:"platform,omitempty"`

	// APIProxy contains API proxy (LLM gateway) configuration.
	APIProxy *AWFAPIProxyConfig `json:"apiProxy,omitempty"`

	// BoundedQueries configures the AWF bounded-query subsystem for approved
	// cross-repository private data access. Omitted when not configured.
	BoundedQueries *AWFBoundedQueriesConfig `json:"boundedQueries,omitempty"`

	// Container contains container execution configuration.
	Container *AWFContainerConfig `json:"container,omitempty"`

	// Logging contains logging and diagnostics configuration.
	Logging *AWFLoggingConfig `json:"logging,omitempty"`

	// Chroot contains chroot execution overrides for split-filesystem ARC/DinD runners.
	// This field is not populated at compile time; it is injected at runtime when DinD topology is detected.
	Chroot *AWFChrootConfig `json:"chroot,omitempty"`
}

// AWFBoundedQueriesConfig is the "boundedQueries" section of the AWF config file.
// It controls the bounded-query subsystem that allows finite, pre-approved questions
// about private repositories. All optional fields are omitted when unset so that
// AWF remains the source of truth for default values.
type AWFBoundedQueriesConfig struct {
	// Enabled must be true when boundedQueries is present in the config.
	// gh-aw always sets this to true when the section is generated.
	Enabled bool `json:"enabled"`

	// PrivateRepos is the list of private repositories approved for bounded-query access.
	PrivateRepos []*AWFBoundedQueryPrivateRepo `json:"privateRepos,omitempty"`

	// Runtime is the isolated backend for each bounded-query invocation.
	// Optional; when omitted AWF uses its default. The value is emitted verbatim and
	// remains independent from the primary agent container runtime.
	Runtime BoundedQueryRuntime `json:"runtime,omitempty"`

	// Timeout is the maximum execution time in seconds for a single invocation.
	// Optional; when omitted AWF uses its default.
	Timeout int `json:"timeout,omitempty"`

	// MemoryLimit is the memory limit for bounded-query container execution (e.g. "512m").
	// Optional; when omitted AWF uses its default.
	MemoryLimit string `json:"memoryLimit,omitempty"`

	// Interpreter is the script interpreter for bounded-query execution (e.g. "python3").
	// Optional; when omitted AWF uses its default.
	Interpreter string `json:"interpreter,omitempty"`

	// MaxInvocations is the maximum number of bounded-query invocations per run.
	// Optional; when omitted AWF uses its default.
	MaxInvocations int `json:"maxInvocations,omitempty"`
}

// AWFBoundedQueryPrivateRepo describes a single private repository approved for
// bounded-query access, with its confidentiality classification.
type AWFBoundedQueryPrivateRepo struct {
	// Repo is the "owner/repo" slug of the approved private repository.
	Repo string `json:"repo"`

	// Sensitivity is the confidentiality classification.
	// Accepted values: "public", "internal", "confidential", "sealed".
	Sensitivity string `json:"sensitivity"`
}

// AWFRunnerConfig is the "runner" section of the AWF config file.
// It provides a single stable contract between gh-aw and AWF for runner topology
// detection, letting AWF resolve all internal details (network isolation, sysroot
// image, path-prefix probes, tool cache validation) from this signal.
type AWFRunnerConfig struct {
	// Topology identifies the runner execution topology.
	// Currently supported values: "arc-dind" (ARC with Docker-in-Docker sidecar).
	// When set to "arc-dind", AWF activates split-filesystem handling, network
	// isolation, sysroot image staging, and DinD pre-staging automatically.
	Topology string `json:"topology,omitempty"`
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

	// Isolation enables topology-based egress isolation mode.
	// Maps to: --network-isolation
	Isolation bool `json:"isolation,omitempty"`

	// TopologyAttach lists container names AWF should attach to awf-net.
	// Maps to: --topology-attach <name> (repeatable)
	TopologyAttach []string `json:"topologyAttach,omitempty"`
}

// AWFPlatformConfig is the "platform" section of the AWF config file.
type AWFPlatformConfig struct {
	// Type is the GitHub deployment type consumed by AWF for auth behavior.
	Type string `json:"type,omitempty"`
}

// AWFAPIProxyConfig is the "apiProxy" section of the AWF config file.
// It maps to the apiProxy.* fields in the AWF config schema.
// Note: --enable-api-proxy is deprecated since AWF v0.27.32 (API proxy is always on).
type AWFAPIProxyConfig struct {
	// Enabled enables the API proxy sidecar for LLM gateway credential isolation.
	// Since AWF v0.27.32, the API proxy is always enabled; this field is kept
	// for backward compatibility with older AWF versions.
	Enabled bool `json:"enabled"`

	// EnableTokenSteering enables budget-warning system message injection near ET budget exhaustion.
	EnableTokenSteering bool `json:"enableTokenSteering,omitempty"`

	// MaxRuns is the maximum number of LLM invocations allowed for a run.
	MaxRuns int `json:"maxRuns,omitempty"`

	// MaxTurnCacheMisses is the maximum number of consecutive cache misses allowed for a run.
	MaxTurnCacheMisses int `json:"maxCacheMisses,omitempty"`

	// MaxAICredits is the explicit per-run AI credits budget enforced by the API proxy.
	MaxAICredits int64 `json:"maxAiCredits,omitempty"`

	// ModelFallback configures the model fallback policy for unresolved model selections.
	// When nil, the AWF default (enabled=true, strategy=middle_power) is used.
	// Set enabled=false to prevent AWF from silently rewriting deployment names, which
	// is needed for BYOK Azure OpenAI deployments where rewriting causes HTTP 404.
	ModelFallback *AWFModelFallbackConfig `json:"modelFallback,omitempty"`

	// ModelMultipliers configures per-model ET accounting multipliers in AWF.
	ModelMultipliers map[string]float64 `json:"modelMultipliers,omitempty"`

	// DefaultAiCreditsPricing is the fallback per-token pricing ($/1M tokens) for
	// models not in the AWF built-in pricing table. When maxAiCredits is active and
	// a model is unrecognized, this rate is used instead of rejecting with HTTP 400.
	DefaultAiCreditsPricing *AiCreditsPricingConfig `json:"defaultAiCreditsPricing,omitempty"`

	// Targets holds per-provider API target overrides.
	// Supported keys: "openai", "anthropic", "copilot", "gemini"
	// The "gemini" target is also used for Antigravity engine routing.
	Targets map[string]*AWFAPITargetConfig `json:"targets,omitempty"`

	// Providers holds per-provider model pricing overlays used by the API proxy
	// AI-credits guardrails for models not present in the built-in pricing table.
	// Structure matches models.json provider format:
	//   providers.<provider>.models.<model>.cost.{input,output,cache_read,cache_write,reasoning}
	Providers map[string]any `json:"providers,omitempty"`

	// Models contains model alias and fallback policy definitions.
	// Keys are alias names (empty string "" = default policy); values are ordered
	// lists of vendor/modelid patterns or other alias names to try in sequence.
	// AWF resolves aliases recursively; loops are not permitted.
	// Per the AWF config schema, this lives under apiProxy.models.
	Models map[string][]string `json:"models,omitempty"`

	// AllowedModels is the explicit allowlist policy for model names/patterns.
	AllowedModels []string `json:"allowedModels,omitempty"`
	// DisallowedModels is the explicit denylist policy for model names/patterns.
	DisallowedModels []string `json:"disallowedModels,omitempty"`
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

	// ExtraHeaders holds additional non-sensitive headers injected on Copilot BYOK upstream
	// requests. Only valid for the "copilot" provider target (copilotTarget in the AWF schema).
	// Maps to AWF_BYOK_EXTRA_HEADERS in the sidecar.
	ExtraHeaders map[string]string `json:"extraHeaders,omitempty"`

	// ExtraBodyFields holds additional non-sensitive JSON body fields injected on Copilot BYOK
	// upstream requests. Only valid for the "copilot" provider target.
	// Maps to AWF_BYOK_EXTRA_BODY_FIELDS in the sidecar.
	ExtraBodyFields map[string]string `json:"extraBodyFields,omitempty"`

	// SessionId is an opt-in session identifier injected as the x-session-id request header
	// and session_id body field on Copilot BYOK upstream requests. Only valid for the
	// "copilot" provider target. Must be set explicitly; never auto-derived from GITHUB_RUN_ID.
	// Maps to AWF_PROVIDER_SESSION_ID in the sidecar.
	SessionId string `json:"sessionId,omitempty"`
}

// AWFContainerConfig is the "container" section of the AWF config file.
// It maps to container execution CLI flags.
type AWFContainerConfig struct {
	// ImageTag is the pinned AWF Docker image tag, with optional digest metadata.
	// Format: "<tag>" or "<tag>,squid=sha256:...,agent=sha256:..."
	// Maps to: --image-tag <value>
	ImageTag string `json:"imageTag,omitempty"`

	// AgentTimeout is the maximum time (in minutes) the agent command may run.
	// docker-sbx requires this so AWF passes a concrete timeout to sbx exec.
	AgentTimeout int `json:"agentTimeout,omitempty"`

	// DockerHostPathPrefix prefixes bind-mount source paths so the Docker daemon can
	// resolve runner filesystem paths. Required for ARC DinD sidecar runners where the
	// runner and daemon have separate filesystems.
	// Maps to: --docker-host-path-prefix <value>
	DockerHostPathPrefix string `json:"dockerHostPathPrefix,omitempty"`

	// ContainerRuntime specifies the OCI runtime for the agent container.
	// "gvisor" enables gVisor's runsc runtime for additional kernel-level isolation.
	// AWF translates "gvisor" → "runsc" internally.
	ContainerRuntime string `json:"containerRuntime,omitempty"`
}

// AWFLoggingConfig is the "logging" section of the AWF config file.
// It maps to logging and diagnostics CLI flags.
type AWFLoggingConfig struct {
	// ProxyLogsDir is the directory path for Squid proxy access logs.
	// Maps to: --proxy-logs-dir <path>
	ProxyLogsDir string `json:"proxyLogsDir,omitempty"`

	// AuditDir is the directory path for audit logs (policy-manifest.json, squid.conf, etc).
	// Maps to: --audit-dir <path>
	AuditDir string `json:"auditDir,omitempty"`
}

// AWFChrootConfig is the "chroot" section of the AWF config file.
// It configures chroot execution overrides for split-filesystem ARC/DinD runners.
// These fields let AWF handle binary staging and identity resolution natively,
// eliminating the need for bootstrap actions on ARC/DinD topologies.
type AWFChrootConfig struct {
	// BinariesSourcePath is the runner-side directory to overlay at /usr/local/bin
	// inside chroot mode for split-filesystem ARC/DinD runners.
	BinariesSourcePath string `json:"binariesSourcePath,omitempty"`

	// Identity configures identity values applied after chroot pivot to override
	// HOME/USER/LOGNAME defaults inside chroot mode.
	Identity *AWFChrootIdentityConfig `json:"identity,omitempty"`
}

// AWFChrootIdentityConfig is the "chroot.identity" section of the AWF config file.
// It provides identity values applied after chroot pivot to override HOME/USER
// defaults inside chroot mode.
type AWFChrootIdentityConfig struct {
	// User is the USER/LOGNAME string to export inside chroot mode.
	User string `json:"user,omitempty"`

	// UID is the UID hint used for chroot identity synthesis and user switching.
	// Must be >= 1 (root is not supported).
	UID int `json:"uid,omitempty"`

	// GID is the GID hint used for chroot identity synthesis and user switching.
	// Must be >= 1.
	GID int `json:"gid,omitempty"`

	// Home is the home directory path to export inside chroot mode.
	Home string `json:"home,omitempty"`
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
	firewallConfig := getFirewallConfig(config.WorkflowData)
	awfConfig := AWFConfigFile{Schema: buildAWFConfigSchemaURL(firewallConfig)}
	applyAWFRunnerConfig(&awfConfig, config.WorkflowData)
	awfConfig.Network = buildAWFNetworkConfig(config)
	applyAWFPlatformConfig(&awfConfig, config.WorkflowData)
	awfConfig.APIProxy = buildAWFAPIProxyConfig(config, firewallConfig)
	applyAWFContainerConfig(&awfConfig, config, firewallConfig)
	applyAWFLoggingConfig(&awfConfig, config.WorkflowData)
	applyAWFBoundedQueriesConfig(&awfConfig, config.WorkflowData, firewallConfig)
	return marshalAndValidateAWFConfigJSON(awfConfig, config.WorkflowData)
}

type awfAPIProxyRuntimeOptions struct {
	maxAICredits        int64
	maxRuns             int
	maxTurnCacheMisses  int
	enableTokenSteering bool
}

func applyAWFRunnerConfig(awfConfig *AWFConfigFile, workflowData *WorkflowData) {
	if topology := getRunnerTopology(workflowData); topology != "" {
		awfConfig.Runner = &AWFRunnerConfig{Topology: topology}
		awfConfigLog.Printf("Runner section: topology=%s", topology)
	}
}

func buildAWFNetworkConfig(config AWFCommandConfig) *AWFNetworkConfig {
	network := buildAWFAllowedAndBlockedDomains(config)
	if isAWFNetworkIsolationEnabled(config.WorkflowData) {
		network = ensureAWFNetworkConfig(network)
		network.Isolation = true
		network.TopologyAttach = buildAWFTopologyAttachList(config.WorkflowData)
		awfConfigLog.Printf("Network section: isolation enabled with %d topology attachments", len(network.TopologyAttach))
	}
	if isDockerSbxRuntime(config.WorkflowData) {
		network = ensureAWFNetworkConfig(network)
		const hostDockerInternal = "host.docker.internal"
		if !slices.Contains(network.AllowDomains, hostDockerInternal) {
			network.AllowDomains = append(network.AllowDomains, hostDockerInternal)
			awfConfigLog.Printf("Network section: added %s for docker-sbx microVM routing", hostDockerInternal)
		}
	}
	return network
}

func buildAWFAllowedAndBlockedDomains(config AWFCommandConfig) *AWFNetworkConfig {
	if config.AllowedDomains == "" {
		return nil
	}
	network := &AWFNetworkConfig{AllowDomains: splitDomainList(config.AllowedDomains)}
	awfConfigLog.Printf("Network section: %d allowed domains", len(network.AllowDomains))
	if config.WorkflowData == nil {
		return network
	}
	blockedDomainsStr := formatBlockedDomains(config.WorkflowData.NetworkPermissions)
	if blockedDomainsStr != "" {
		network.BlockDomains = splitDomainList(blockedDomainsStr)
		awfConfigLog.Printf("Network section: %d blocked domains", len(network.BlockDomains))
	}
	return network
}

func ensureAWFNetworkConfig(network *AWFNetworkConfig) *AWFNetworkConfig {
	if network != nil {
		return network
	}
	return &AWFNetworkConfig{}
}

func applyAWFPlatformConfig(awfConfig *AWFConfigFile, workflowData *WorkflowData) {
	if platformType := extractPlatformType(workflowData); platformType != "" {
		awfConfig.Platform = &AWFPlatformConfig{Type: platformType}
		awfConfigLog.Printf("Platform section: type=%s", platformType)
	}
}

func buildAWFAPIProxyConfig(config AWFCommandConfig, firewallConfig *FirewallConfig) *AWFAPIProxyConfig {
	options := resolveAWFAPIProxyRuntimeOptions(config.WorkflowData)
	apiProxy := &AWFAPIProxyConfig{
		Enabled:             true,
		MaxRuns:             options.maxRuns,
		MaxTurnCacheMisses:  options.maxTurnCacheMisses,
		MaxAICredits:        options.maxAICredits,
		EnableTokenSteering: options.enableTokenSteering && awfSupportsTokenSteering(firewallConfig),
	}
	logAWFTokenSteeringDecision(options.enableTokenSteering, firewallConfig)
	applyAWFAPIProxyModelFallback(apiProxy, config.WorkflowData)
	applyAWFAPIProxyPricing(apiProxy, config.WorkflowData)
	applyAWFAPIProxyTargets(apiProxy, config)
	applyAWFAPIProxyProviders(apiProxy, config.WorkflowData, firewallConfig)
	applyAWFAPIProxyModels(apiProxy, config.WorkflowData)
	return apiProxy
}

func resolveAWFAPIProxyRuntimeOptions(workflowData *WorkflowData) awfAPIProxyRuntimeOptions {
	options := awfAPIProxyRuntimeOptions{
		maxRuns:             constants.DefaultMaxRuns,
		maxTurnCacheMisses:  (*EngineConfig)(nil).GetMaxTurnCacheMisses(),
		enableTokenSteering: true,
	}
	if workflowData != nil && workflowData.EngineConfig != nil {
		options.maxAICredits = workflowData.EngineConfig.MaxAICredits
		options.maxRuns = workflowData.EngineConfig.GetMaxRuns()
		options.maxTurnCacheMisses = workflowData.EngineConfig.GetMaxTurnCacheMisses()
	}
	if options.maxAICredits < 0 {
		options.maxAICredits = 0
		options.enableTokenSteering = false
	}
	return options
}

func logAWFTokenSteeringDecision(enableTokenSteering bool, firewallConfig *FirewallConfig) {
	if !enableTokenSteering {
		awfConfigLog.Printf("Skipping apiProxy.enableTokenSteering: max-ai-credits is negative (disabled)")
		return
	}
	if !awfSupportsTokenSteering(firewallConfig) {
		awfConfigLog.Printf("Skipping apiProxy.enableTokenSteering: AWF version %q requires at least %s", getAWFImageTag(firewallConfig), constants.AWFTokenSteeringMinVersion)
	}
}

func applyAWFAPIProxyModelFallback(apiProxy *AWFAPIProxyConfig, workflowData *WorkflowData) {
	if mf := extractModelFallback(workflowData); mf != nil {
		apiProxy.ModelFallback = mf
		enabledDisplay := "<unset>"
		if mf.Enabled != nil {
			enabledDisplay = mf.Enabled.String()
		}
		awfConfigLog.Printf("API proxy: modelFallback configured: enabled=%s", enabledDisplay)
	}
}

func applyAWFAPIProxyPricing(apiProxy *AWFAPIProxyConfig, workflowData *WorkflowData) {
	if pricing := extractDefaultAiCreditsPricing(workflowData); pricing != nil {
		apiProxy.DefaultAiCreditsPricing = pricing
		awfConfigLog.Printf("API proxy: defaultAiCreditsPricing configured: input=%g, output=%g", pricing.Input, pricing.Output)
	}
}

func applyAWFAPIProxyTargets(apiProxy *AWFAPIProxyConfig, config AWFCommandConfig) {
	targets := buildAWFAPITargets(config.WorkflowData, config.EngineName)
	if len(targets) == 0 {
		return
	}
	apiProxy.Targets = targets
	awfConfigLog.Printf("API proxy: %d custom targets configured", len(targets))
}

func buildAWFAPITargets(workflowData *WorkflowData, engineName string) map[string]*AWFAPITargetConfig {
	targets := map[string]*AWFAPITargetConfig{}
	addAWFTargetHost(targets, "openai", extractAPITargetHost(workflowData, "OPENAI_BASE_URL"))
	addAWFTargetHost(targets, "anthropic", extractAPITargetHost(workflowData, "ANTHROPIC_BASE_URL"))
	applyAWFTargetAuthHeaders(targets, workflowData)
	applyAWFCopilotTarget(targets, workflowData)
	applyAWFGeminiTarget(targets, workflowData, engineName)
	return targets
}

func addAWFTargetHost(targets map[string]*AWFAPITargetConfig, provider string, host string) {
	if host == "" {
		return
	}
	targets[provider] = &AWFAPITargetConfig{Host: host}
	awfConfigLog.Printf("API proxy: custom %s target=%s", provider, host)
}

func applyAWFTargetAuthHeaders(targets map[string]*AWFAPITargetConfig, workflowData *WorkflowData) {
	for _, provider := range []string{"openai", "anthropic"} {
		authHeader := extractAPITargetAuthHeader(workflowData, provider)
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
}

func applyAWFCopilotTarget(targets map[string]*AWFAPITargetConfig, workflowData *WorkflowData) {
	if copilotTarget := GetCopilotAPITarget(workflowData); copilotTarget != "" {
		targets["copilot"] = &AWFAPITargetConfig{Host: copilotTarget}
		awfConfigLog.Printf("API proxy: custom copilot target=%s", copilotTarget)
	}
	copilotFrontmatter := extractCopilotTargetConfig(workflowData)
	if copilotFrontmatter == nil {
		return
	}
	target := ensureAWFAPITarget(targets, "copilot")
	if copilotFrontmatter.AuthHeader != "" {
		target.AuthHeader = copilotFrontmatter.AuthHeader
		awfConfigLog.Printf("API proxy: copilot authHeader=%s", copilotFrontmatter.AuthHeader)
	}
	if len(copilotFrontmatter.ExtraHeaders) > 0 {
		target.ExtraHeaders = copilotFrontmatter.ExtraHeaders
		awfConfigLog.Printf("API proxy: copilot extraHeaders configured (%d header(s))", len(copilotFrontmatter.ExtraHeaders))
	}
	if len(copilotFrontmatter.ExtraBodyFields) > 0 {
		target.ExtraBodyFields = copilotFrontmatter.ExtraBodyFields
		awfConfigLog.Printf("API proxy: copilot extraBodyFields configured (%d field(s))", len(copilotFrontmatter.ExtraBodyFields))
	}
	if copilotFrontmatter.SessionId != "" {
		target.SessionId = copilotFrontmatter.SessionId
		awfConfigLog.Printf("API proxy: copilot sessionId configured")
	}
}

func ensureAWFAPITarget(targets map[string]*AWFAPITargetConfig, provider string) *AWFAPITargetConfig {
	if target, ok := targets[provider]; ok {
		return target
	}
	targets[provider] = &AWFAPITargetConfig{}
	return targets[provider]
}

func applyAWFGeminiTarget(targets map[string]*AWFAPITargetConfig, workflowData *WorkflowData, engineName string) {
	if antigravityTarget := GetAntigravityAPITarget(workflowData, engineName); antigravityTarget != "" {
		awfConfigLog.Printf("API proxy: mapped antigravity target to gemini provider target=%s", antigravityTarget)
		targets["gemini"] = &AWFAPITargetConfig{Host: antigravityTarget}
		return
	}
	if geminiTarget := GetGeminiAPITarget(workflowData, engineName); geminiTarget != "" {
		awfConfigLog.Printf("API proxy: custom gemini target=%s", geminiTarget)
		targets["gemini"] = &AWFAPITargetConfig{Host: geminiTarget}
	}
}

func applyAWFAPIProxyProviders(apiProxy *AWFAPIProxyConfig, workflowData *WorkflowData, firewallConfig *FirewallConfig) {
	if providers := extractModelCostProviders(workflowData); len(providers) > 0 {
		if awfSupportsAPIProxyProviders(firewallConfig) {
			apiProxy.Providers = providers
			awfConfigLog.Printf("API proxy: %d model-cost provider override(s) configured", len(providers))
		} else {
			awfConfigLog.Printf("Skipping apiProxy.providers: AWF version %q requires at least %s", getAWFImageTag(firewallConfig), constants.AWFAPIProxyProvidersMinVersion)
		}
	}
}

func applyAWFAPIProxyModels(apiProxy *AWFAPIProxyConfig, workflowData *WorkflowData) {
	if workflowData != nil && len(workflowData.ModelMappings) > 0 {
		apiProxy.Models = workflowData.ModelMappings
		awfConfigLog.Printf("Models section: %d alias entries", len(workflowData.ModelMappings))
	}
	allowedModels, disallowedModels := resolveModelPolicyForAWFConfig(workflowData)
	if len(allowedModels) > 0 {
		apiProxy.AllowedModels = allowedModels
		awfConfigLog.Printf("Models policy: %d allowed model pattern(s)", len(allowedModels))
	}
	if len(disallowedModels) > 0 {
		apiProxy.DisallowedModels = disallowedModels
		awfConfigLog.Printf("Models policy: %d disallowed model pattern(s)", len(disallowedModels))
	}
}

func applyAWFContainerConfig(awfConfig *AWFConfigFile, config AWFCommandConfig, firewallConfig *FirewallConfig) {
	awfImageTag := buildAWFImageTagWithDigests(getAWFImageTag(firewallConfig), config.WorkflowData)
	agentRuntime, agentTimeout := resolveAWFContainerRuntime(config.WorkflowData, firewallConfig)
	if awfImageTag == "" && !isArcDindTopology(config.WorkflowData) && agentRuntime == "" && agentTimeout == 0 {
		return
	}
	awfConfig.Container = &AWFContainerConfig{ImageTag: awfImageTag, AgentTimeout: agentTimeout, ContainerRuntime: agentRuntime}
	if awfImageTag != "" {
		awfConfigLog.Printf("Container section: image_tag=%s", awfImageTag)
	}
	if agentRuntime != "" {
		awfConfigLog.Printf("Container section: containerRuntime=%s", agentRuntime)
	}
	if agentTimeout > 0 {
		awfConfigLog.Printf("Container section: agentTimeout=%d", agentTimeout)
	}
}

func resolveAWFContainerRuntime(workflowData *WorkflowData, firewallConfig *FirewallConfig) (string, int) {
	agentRuntime := getAgentContainerRuntime(workflowData)
	agentTimeout := 0
	if isDockerSbxRuntime(workflowData) {
		agentTimeout = resolveAWFContainerAgentTimeoutMinutes(workflowData)
	}
	if awfSupportsContainerRuntime(firewallConfig) {
		return agentRuntime, agentTimeout
	}
	if agentRuntime != "" {
		awfConfigLog.Printf("Skipping containerRuntime: AWF version %q requires at least %s (gh-aw-firewall#6093)", getAWFImageTag(firewallConfig), constants.AWFContainerRuntimeMinVersion)
	}
	return "", agentTimeout
}

func applyAWFLoggingConfig(awfConfig *AWFConfigFile, workflowData *WorkflowData) {
	awfConfig.Logging = &AWFLoggingConfig{ProxyLogsDir: string(constants.AWFProxyLogsDir), AuditDir: string(constants.AWFAuditDir)}
	if isArcDindTopology(workflowData) {
		awfConfig.Logging.ProxyLogsDir = awfArcDindProxyLogsDirExpr
		awfConfig.Logging.AuditDir = awfArcDindAuditDirExpr
	}
	awfConfigLog.Printf("Logging section: proxyLogsDir=%s, auditDir=%s", awfConfig.Logging.ProxyLogsDir, awfConfig.Logging.AuditDir)
}

func applyAWFBoundedQueriesConfig(awfConfig *AWFConfigFile, workflowData *WorkflowData, firewallConfig *FirewallConfig) {
	if bq := extractBoundedQueriesConfig(workflowData); bq != nil {
		if awfSupportsBoundedQueries(firewallConfig) {
			awfConfig.BoundedQueries = bq
			awfConfigLog.Printf("Bounded queries section: %d private repo(s)", len(bq.PrivateRepos))
		} else {
			awfConfigLog.Printf("Skipping boundedQueries: AWF version %q requires at least %s", getAWFImageTag(firewallConfig), constants.AWFBoundedQueriesMinVersion)
		}
	}
}

func marshalAndValidateAWFConfigJSON(awfConfig AWFConfigFile, workflowData *WorkflowData) (string, error) {
	jsonStr, err := jsonutil.MarshalCompactNoHTMLEscape(awfConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AWF config to JSON: %w", err)
	}
	awfConfigLog.Printf("AWF config JSON generated: %d bytes", len(jsonStr))
	if workflowData != nil && workflowData.ValidateAWFConfig {
		if err := validateAWFConfigJSON(jsonStr); err != nil {
			return "", fmt.Errorf("generated AWF config failed schema validation: %w", err)
		}
	}
	return jsonStr, nil
}

func resolveAWFContainerAgentTimeoutMinutes(workflowData *WorkflowData) int {
	// Reuse the workflow-level default timeout so docker-sbx inherits the same
	// runtime ceiling when top-level timeout-minutes is omitted or non-numeric.
	defaultTimeout := compilerenv.ResolveDefaultTimeoutMinutes(int(constants.DefaultAgenticWorkflowTimeout / time.Minute))
	if workflowData == nil || workflowData.TimeoutMinutes == "" {
		return defaultTimeout
	}

	rawTimeout := strings.TrimSpace(workflowData.TimeoutMinutes)
	if after, ok := strings.CutPrefix(rawTimeout, "timeout-minutes:"); ok {
		rawTimeout = strings.TrimSpace(after)
	}

	timeoutMinutes, err := strconv.Atoi(rawTimeout)
	if err == nil && timeoutMinutes > 0 {
		return timeoutMinutes
	}

	if rawTimeout != "" {
		awfConfigLog.Printf("Container section: non-numeric timeout-minutes %q (e.g. a GitHub Actions expression) cannot be emitted in integer-only agentTimeout; using default %d", rawTimeout, defaultTimeout)
	}
	return defaultTimeout
}

// buildAWFTopologyAttachList returns container names that AWF should attach to
// the internal awf-net network when network isolation mode is enabled.
// The list always includes the MCP gateway and conditionally includes the
// host-started CLI proxy sidecar when gh-proxy mode is active.
func buildAWFTopologyAttachList(workflowData *WorkflowData) []string {
	targets := []string{"awmg-mcpg"}
	if isCliProxyNeeded(workflowData) {
		targets = append(targets, "awmg-cli-proxy")
	}
	return targets
}

// splitDomainList splits a comma-separated domain string into a deduplicated
// slice. Empty entries are ignored. The order of the original list is preserved for
// non-duplicate entries; this keeps the allow-list deterministic.
func splitDomainList(domains string) []string {
	var result []string
	seen := make(map[string]struct {
	})
	for d := range strings.SplitSeq(domains, ",") {
		d = strings.TrimSpace(d)
		if d != "" && !setutil.Contains(seen, d) {
			seen[d] = struct {
			}{}
			result = append(result, d)
		}
	}
	return result
}

// resolveModelPolicyForAWFConfig applies policy precedence independently per list:
// allowed rules are narrowed using intersection with env policy, while blocked
// rules are widened using union with env policy.
func resolveModelPolicyForAWFConfig(workflowData *WorkflowData) ([]string, []string) {
	envAllowed, hasAllowedOverride := compilerenv.ResolvePolicyModelsAllowed()
	envBlocked, hasBlockedOverride := compilerenv.ResolvePolicyModelsBlocked()
	var allowed []string
	var blocked []string
	if workflowData != nil {
		allowed = workflowData.ModelPolicyAllowed
		blocked = workflowData.ModelPolicyBlocked
	}
	if hasAllowedOverride {
		allowed = intersectModelPolicyRules(allowed, envAllowed)
	}
	if hasBlockedOverride {
		blocked = unionModelPolicyRules(blocked, envBlocked)
	}
	blockedSet := make(map[string]struct{}, len(blocked))
	for _, model := range blocked {
		blockedSet[model] = struct{}{}
	}
	allowed = filterAllowedModelConflictsWithSet(allowed, blockedSet)
	return allowed, blocked
}

func intersectModelPolicyRules(local, override []string) []string {
	if len(override) == 0 {
		return append([]string(nil), local...)
	}
	// No local allow-list means no workflow restriction; keep the env allow-list.
	if len(local) == 0 {
		return append([]string(nil), override...)
	}
	localSet := make(map[string]struct{}, len(local))
	for _, model := range local {
		localSet[model] = struct{}{}
	}
	result := make([]string, 0, len(override))
	for _, model := range override {
		if _, ok := localSet[model]; ok {
			result = append(result, model)
		}
	}
	return result
}

func unionModelPolicyRules(local, override []string) []string {
	result := make([]string, 0, len(local)+len(override))
	seen := make(map[string]struct{}, len(local)+len(override))
	for _, model := range local {
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		result = append(result, model)
	}
	for _, model := range override {
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		result = append(result, model)
	}
	return result
}

// extractPlatformType returns sandbox.agent.platform only for enabled AWF sandbox
// agents, or an empty string to let AWF fall back to its default platform logic.
func extractPlatformType(workflowData *WorkflowData) string {
	if workflowData == nil || workflowData.SandboxConfig == nil || workflowData.SandboxConfig.Agent == nil {
		return ""
	}
	if workflowData.SandboxConfig.Agent.Disabled {
		return ""
	}
	if !isSupportedSandboxType(getAgentType(workflowData.SandboxConfig.Agent)) {
		return ""
	}
	return workflowData.SandboxConfig.Agent.Platform
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

// extractDefaultAiCreditsPricing returns an AiCreditsPricingConfig if the workflow has
// configured models.default-ai-credits-pricing, or nil if the field is absent.
// This fallback pricing is used when maxAiCredits is active and the requested model is not in
// the built-in pricing table, preventing HTTP 400 unknown_model_ai_credits for BYOK/self-hosted models.
func extractDefaultAiCreditsPricing(workflowData *WorkflowData) *AiCreditsPricingConfig {
	if workflowData == nil {
		return nil
	}
	p := workflowData.DefaultAiCreditsPricing
	if p == nil {
		return nil
	}
	return &AiCreditsPricingConfig{
		Input:       p.Input,
		Output:      p.Output,
		CachedInput: p.CachedInput,
		CacheWrite:  p.CacheWrite,
	}
}

func extractModelCostProviders(workflowData *WorkflowData) map[string]any {
	if workflowData == nil || len(workflowData.ModelCosts) == 0 {
		return nil
	}
	providers, ok := workflowData.ModelCosts["providers"].(map[string]any)
	if !ok {
		awfConfigLog.Printf("API proxy: models.providers has unexpected type %T; skipping provider overlay", workflowData.ModelCosts["providers"])
		return nil
	}
	if len(providers) == 0 {
		return nil
	}
	clone := make(map[string]any, len(providers))
	maps.Copy(clone, providers)
	return clone
}

// extractBoundedQueriesConfig returns an AWFBoundedQueriesConfig populated from
// tools.github.bounded-queries, or nil when the field is absent.
// Only fields explicitly set in frontmatter are included; optional fields that
// were not specified are omitted so that AWF remains the source of truth for defaults.
func extractBoundedQueriesConfig(workflowData *WorkflowData) *AWFBoundedQueriesConfig {
	if workflowData == nil {
		return nil
	}
	if workflowData.ParsedTools == nil || workflowData.ParsedTools.GitHub == nil {
		return nil
	}
	bq := workflowData.ParsedTools.GitHub.BoundedQueries
	if bq == nil {
		return nil
	}

	awfBQ := &AWFBoundedQueriesConfig{
		Enabled:     true,
		Runtime:     bq.Runtime,
		MemoryLimit: bq.MemoryLimit,
		Interpreter: bq.Interpreter,
	}
	if bq.Timeout != nil {
		awfBQ.Timeout = *bq.Timeout
	}
	if bq.MaxInvocations != nil {
		awfBQ.MaxInvocations = *bq.MaxInvocations
	}

	for _, r := range bq.PrivateRepos {
		awfBQ.PrivateRepos = append(awfBQ.PrivateRepos, &AWFBoundedQueryPrivateRepo{
			Repo:        r.Repo,
			Sensitivity: r.Sensitivity,
		})
	}

	return awfBQ
}

// getRunnerTopology extracts the runner topology string from WorkflowData.
// Returns an empty string when no topology is configured.
func getRunnerTopology(workflowData *WorkflowData) string {
	if workflowData == nil || workflowData.RunnerConfig == nil {
		return ""
	}
	return workflowData.RunnerConfig.Topology
}

// isArcDindTopology returns true when the workflow targets ARC/DinD runners.
func isArcDindTopology(workflowData *WorkflowData) bool {
	return getRunnerTopology(workflowData) == RunnerTopologyArcDind
}
