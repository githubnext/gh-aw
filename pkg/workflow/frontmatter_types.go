package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var frontmatterTypesLog = logger.New("workflow:frontmatter_types")

// RuntimeConfig represents the configuration for a single runtime
type RuntimeConfig struct {
	Version           string `json:"version,omitempty"`             // Version of the runtime (e.g., "20" for Node, "3.11" for Python)
	If                string `json:"if,omitempty"`                  // Optional GitHub Actions if condition (e.g., "hashFiles('go.mod') != ''")
	ActionRepo        string `json:"action-repo,omitempty"`         // Override the GitHub Actions repository (e.g., "actions/setup-node")
	ActionVersion     string `json:"action-version,omitempty"`      // Override the action version (e.g., "v4")
	RunInstallScripts *bool  `json:"run-install-scripts,omitempty"` // If true, allow pre/post install scripts for this runtime (supply chain risk; emits warning or error in strict mode)
}

// RuntimesConfig represents the configuration for all runtime environments
// This provides type-safe access to runtime version overrides
type RuntimesConfig struct {
	Node    *RuntimeConfig `json:"node,omitempty"`    // Node.js runtime
	Python  *RuntimeConfig `json:"python,omitempty"`  // Python runtime
	Go      *RuntimeConfig `json:"go,omitempty"`      // Go runtime
	UV      *RuntimeConfig `json:"uv,omitempty"`      // uv package installer
	Bun     *RuntimeConfig `json:"bun,omitempty"`     // Bun runtime
	Deno    *RuntimeConfig `json:"deno,omitempty"`    // Deno runtime
	Dotnet  *RuntimeConfig `json:"dotnet,omitempty"`  // .NET runtime
	Elixir  *RuntimeConfig `json:"elixir,omitempty"`  // Elixir runtime
	Haskell *RuntimeConfig `json:"haskell,omitempty"` // Haskell runtime
	Java    *RuntimeConfig `json:"java,omitempty"`    // Java runtime
	Ruby    *RuntimeConfig `json:"ruby,omitempty"`    // Ruby runtime
}

// GitHubActionsPermissionsConfig holds permission scopes supported by the GitHub Actions GITHUB_TOKEN.
// These scopes can be declared in the workflow's top-level permissions block and are enforced
// natively by GitHub Actions.
type GitHubActionsPermissionsConfig struct {
	Actions            string `json:"actions,omitempty"`
	Checks             string `json:"checks,omitempty"`
	Contents           string `json:"contents,omitempty"`
	Deployments        string `json:"deployments,omitempty"`
	IDToken            string `json:"id-token,omitempty"`
	Issues             string `json:"issues,omitempty"`
	Discussions        string `json:"discussions,omitempty"`
	Packages           string `json:"packages,omitempty"`
	Pages              string `json:"pages,omitempty"`
	PullRequests       string `json:"pull-requests,omitempty"`
	RepositoryProjects string `json:"repository-projects,omitempty"`
	SecurityEvents     string `json:"security-events,omitempty"`
	Statuses           string `json:"statuses,omitempty"`
}

// GitHubAppPermissionsConfig holds permission scopes that are exclusive to GitHub App
// installation access tokens (not supported by GITHUB_TOKEN). When any of these are
// specified, a GitHub App must be configured in the workflow.
type GitHubAppPermissionsConfig struct {
	// Organization-level permissions (the common use-case placed first)
	OrganizationProjects                string `json:"organization-projects,omitempty"`
	Members                             string `json:"members,omitempty"`
	OrganizationAdministration          string `json:"organization-administration,omitempty"`
	TeamDiscussions                     string `json:"team-discussions,omitempty"`
	OrganizationHooks                   string `json:"organization-hooks,omitempty"`
	OrganizationMembers                 string `json:"organization-members,omitempty"`
	OrganizationPackages                string `json:"organization-packages,omitempty"`
	OrganizationSelfHostedRunners       string `json:"organization-self-hosted-runners,omitempty"`
	OrganizationCustomOrgRoles          string `json:"organization-custom-org-roles,omitempty"`
	OrganizationCustomProperties        string `json:"organization-custom-properties,omitempty"`
	OrganizationCustomRepositoryRoles   string `json:"organization-custom-repository-roles,omitempty"`
	OrganizationAnnouncementBanners     string `json:"organization-announcement-banners,omitempty"`
	OrganizationEvents                  string `json:"organization-events,omitempty"`
	OrganizationPlan                    string `json:"organization-plan,omitempty"`
	OrganizationUserBlocking            string `json:"organization-user-blocking,omitempty"`
	OrganizationPersonalAccessTokenReqs string `json:"organization-personal-access-token-requests,omitempty"`
	OrganizationPersonalAccessTokens    string `json:"organization-personal-access-tokens,omitempty"`
	OrganizationCopilot                 string `json:"organization-copilot,omitempty"`
	OrganizationCodespaces              string `json:"organization-codespaces,omitempty"`
	// Repository-level permissions
	Administration             string `json:"administration,omitempty"`
	Environments               string `json:"environments,omitempty"`
	GitSigning                 string `json:"git-signing,omitempty"`
	VulnerabilityAlerts        string `json:"vulnerability-alerts,omitempty"`
	Workflows                  string `json:"workflows,omitempty"`
	RepositoryHooks            string `json:"repository-hooks,omitempty"`
	SingleFile                 string `json:"single-file,omitempty"`
	Codespaces                 string `json:"codespaces,omitempty"`
	RepositoryCustomProperties string `json:"repository-custom-properties,omitempty"`
	// User-level permissions
	EmailAddresses           string `json:"email-addresses,omitempty"`
	CodespacesLifecycleAdmin string `json:"codespaces-lifecycle-admin,omitempty"`
	CodespacesMetadata       string `json:"codespaces-metadata,omitempty"`
}

// PermissionsConfig represents GitHub Actions permissions configuration.
// Supports both shorthand (read-all, write-all) and detailed scope-based permissions.
// Embeds GitHubActionsPermissionsConfig for standard GITHUB_TOKEN scopes and
// GitHubAppPermissionsConfig for GitHub App-only scopes.
type PermissionsConfig struct {
	// Shorthand permission (read-all, write-all, read, write, none)
	Shorthand string `json:"-"` // Not in JSON, set when parsing shorthand format

	// GitHub Actions GITHUB_TOKEN permission scopes
	GitHubActionsPermissionsConfig

	// GitHub App-only permission scopes (require a GitHub App to be configured)
	GitHubAppPermissionsConfig
}

// RateLimitConfig represents rate limiting configuration for workflow triggers
// Limits how many times a user can trigger a workflow within a time window
type RateLimitConfig struct {
	Max          int      `json:"max,omitempty"`           // Maximum number of runs allowed per time window (default: 5)
	Window       int      `json:"window,omitempty"`        // Time window in minutes (default: 60)
	Events       []string `json:"events,omitempty"`        // Event types to apply rate limiting to (e.g., ["workflow_dispatch", "issue_comment"])
	IgnoredRoles []string `json:"ignored-roles,omitempty"` // Roles that are exempt from rate limiting (e.g., ["admin", "maintainer"])
}

// OTLPConfig holds configuration for OTLP (OpenTelemetry Protocol) trace export.
type OTLPConfig struct {
	// Endpoint is the OTLP collector endpoint URL (e.g. "https://traces.example.com:4317").
	// Supports GitHub Actions expressions such as ${{ secrets.OTLP_ENDPOINT }}.
	// When a static URL is provided, its hostname is automatically added to the
	// network firewall allowlist.
	Endpoint string `json:"endpoint,omitempty"`

	// Headers is a comma-separated list of key=value HTTP headers to include with
	// every OTLP export request (e.g. "Authorization=Bearer <token>").
	// Supports GitHub Actions expressions such as ${{ secrets.OTLP_HEADERS }}.
	// Injected as the standard OTEL_EXPORTER_OTLP_HEADERS environment variable.
	Headers string `json:"headers,omitempty"`
}

// ObservabilityConfig represents workflow observability options.
type ObservabilityConfig struct {
	OTLP *OTLPConfig `json:"otlp,omitempty"`
}

// FrontmatterConfig represents the structured configuration from workflow frontmatter
// This provides compile-time type safety and clearer error messages compared to map[string]any
type FrontmatterConfig struct {
	// Core workflow fields
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	// Engine accepts both a plain string engine name (e.g. "copilot") and an object-style
	// configuration (e.g. {id: copilot, max-continuations: 2}).  Using any prevents
	// JSON unmarshal failures when the engine is an object, which would otherwise cause
	// ParseFrontmatterConfig to return nil and break features that depend on it (e.g. OTLP).
	Engine            any               `json:"engine,omitempty"`
	Source            string            `json:"source,omitempty"`
	Redirect          string            `json:"redirect,omitempty"`
	TrackerID         string            `json:"tracker-id,omitempty"`
	Version           string            `json:"version,omitempty"`
	TimeoutMinutes    *TemplatableInt32 `json:"timeout-minutes,omitempty"`
	Strict            *bool             `json:"strict,omitempty"`              // Pointer to distinguish unset from false
	Private           *bool             `json:"private,omitempty"`             // If true, workflow cannot be added to other repositories
	RunInstallScripts *bool             `json:"run-install-scripts,omitempty"` // If true, allow pre/post install scripts globally (supply chain risk; emits warning or error in strict mode)
	Labels            []string          `json:"labels,omitempty"`

	// Configuration sections - using strongly-typed structs
	Tools            *ToolsConfig       `json:"tools,omitempty"`
	MCPServers       map[string]any     `json:"mcp-servers,omitempty"` // Legacy field, use Tools instead
	RuntimesTyped    *RuntimesConfig    `json:"-"`                     // New typed field (not in JSON to avoid conflict)
	Runtimes         map[string]any     `json:"runtimes,omitempty"`    // Deprecated: use RuntimesTyped
	Jobs             map[string]any     `json:"jobs,omitempty"`        // Custom workflow jobs (too dynamic to type)
	SafeOutputs      *SafeOutputsConfig `json:"safe-outputs,omitempty"`
	MCPScripts       *MCPScriptsConfig  `json:"mcp-scripts,omitempty"`
	PermissionsTyped *PermissionsConfig `json:"-"` // New typed field (not in JSON to avoid conflict)

	// Event and trigger configuration
	On          map[string]any `json:"on,omitempty"`          // Complex trigger config with many variants (too dynamic to type)
	Permissions map[string]any `json:"permissions,omitempty"` // Deprecated: use PermissionsTyped (can be string or map)
	Concurrency map[string]any `json:"concurrency,omitempty"`
	If          string         `json:"if,omitempty"`

	// Network and sandbox configuration
	Network *NetworkPermissions `json:"network,omitempty"`
	Sandbox *SandboxConfig      `json:"sandbox,omitempty"`

	// Feature flags and other settings
	Features map[string]any    `json:"features,omitempty"` // Dynamic feature flags
	Env      map[string]string `json:"env,omitempty"`
	Secrets  map[string]any    `json:"secrets,omitempty"`

	// Workflow execution settings
	RunsOn        string         `json:"runs-on,omitempty"`
	RunsOnSlim    string         `json:"runs-on-slim,omitempty"` // Runner for all framework/generated jobs (activation, safe-outputs, unlock, etc.)
	RunName       string         `json:"run-name,omitempty"`
	PreSteps      []any          `json:"pre-steps,omitempty"`       // Pre-workflow steps (run before checkout)
	Steps         []any          `json:"steps,omitempty"`           // Custom workflow steps
	PreAgentSteps []any          `json:"pre-agent-steps,omitempty"` // Steps run immediately before agent execution
	PostSteps     []any          `json:"post-steps,omitempty"`      // Post-workflow steps
	Environment   map[string]any `json:"environment,omitempty"`     // GitHub environment
	Container     map[string]any `json:"container,omitempty"`
	Services      map[string]any `json:"services,omitempty"`
	Cache         map[string]any `json:"cache,omitempty"`

	// Import and inclusion
	Imports        any            `json:"imports,omitempty"`         // Can be string or array
	ImportSchema   map[string]any `json:"import-schema,omitempty"`   // Schema for validating 'with' values when this workflow is imported
	Include        any            `json:"include,omitempty"`         // Can be string or array
	InlinedImports bool           `json:"inlined-imports,omitempty"` // If true, inline all imports at compile time instead of using runtime-import macros
	Resources      []string       `json:"resources,omitempty"`       // Additional workflow .md or action .yml files to fetch alongside this workflow

	// Metadata
	Metadata      map[string]string    `json:"metadata,omitempty"` // Custom metadata key-value pairs
	SecretMasking *SecretMaskingConfig `json:"secret-masking,omitempty"`
	Observability *ObservabilityConfig `json:"observability,omitempty"`

	// Rate limiting configuration
	RateLimit *RateLimitConfig `json:"rate-limit,omitempty"`

	// Update check configuration.
	// When set to false, the version update check step is skipped in the activation job.
	// This flag is not allowed in strict mode.
	UpdateCheck *bool `json:"check-for-updates,omitempty"`

	// Checkout configuration for the agent job.
	// Controls how actions/checkout is invoked.
	// Can be a single CheckoutConfig object or an array of CheckoutConfig objects.
	// Set to false to disable the default checkout step entirely.
	Checkout         any               `json:"checkout,omitempty"` // Raw value (object, array, or false)
	CheckoutConfigs  []*CheckoutConfig `json:"-"`                  // Parsed checkout configs (not in JSON)
	CheckoutDisabled bool              `json:"-"`                  // true when checkout: false is set in frontmatter
}
