package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/github/gh-aw/pkg/logger"
)

var frontmatterTypesLog = logger.New("workflow:frontmatter_types")

// RuntimeConfig represents the configuration for a single runtime
type RuntimeConfig struct {
	Version       string `json:"version,omitempty"`        // Version of the runtime (e.g., "20" for Node, "3.11" for Python)
	If            string `json:"if,omitempty"`             // Optional GitHub Actions if condition (e.g., "hashFiles('go.mod') != ''")
	ActionRepo    string `json:"action-repo,omitempty"`    // Override the GitHub Actions repository (e.g., "actions/setup-node")
	ActionVersion string `json:"action-version,omitempty"` // Override the action version (e.g., "v4")
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

// APMDependenciesInfo encapsulates APM (Agent Package Manager) dependency configuration.
// Supports simple array format and object format with packages, isolated, github-app, and version fields.
// When present, a pack step is emitted in the activation job and a restore step in the agent job.
type APMDependenciesInfo struct {
	Packages    []string          // APM package slugs to install (e.g., "org/package")
	Isolated    bool              // If true, agent restore step clears primitive dirs before unpacking
	GitHubApp   *GitHubAppConfig  // Optional GitHub App for cross-org private package access
	GitHubToken string            // Optional custom GitHub token expression (uses cascading fallback when empty)
	Version     string            // Optional APM CLI version override (e.g., "v0.8.0"); defaults to DefaultAPMVersion
	Env         map[string]string // Optional environment variables to set on the APM pack step
}

// RateLimitConfig represents rate limiting configuration for workflow triggers
// Limits how many times a user can trigger a workflow within a time window
type RateLimitConfig struct {
	Max          int      `json:"max,omitempty"`           // Maximum number of runs allowed per time window (default: 5)
	Window       int      `json:"window,omitempty"`        // Time window in minutes (default: 60)
	Events       []string `json:"events,omitempty"`        // Event types to apply rate limiting to (e.g., ["workflow_dispatch", "issue_comment"])
	IgnoredRoles []string `json:"ignored-roles,omitempty"` // Roles that are exempt from rate limiting (e.g., ["admin", "maintainer"])
}

// ObservabilityConfig represents workflow observability options.
type ObservabilityConfig struct {
	JobSummary string `json:"job-summary,omitempty"`
}

// FrontmatterConfig represents the structured configuration from workflow frontmatter
// This provides compile-time type safety and clearer error messages compared to map[string]any
type FrontmatterConfig struct {
	// Core workflow fields
	Name           string   `json:"name,omitempty"`
	Description    string   `json:"description,omitempty"`
	Engine         string   `json:"engine,omitempty"`
	Source         string   `json:"source,omitempty"`
	TrackerID      string   `json:"tracker-id,omitempty"`
	Version        string   `json:"version,omitempty"`
	TimeoutMinutes int      `json:"timeout-minutes,omitempty"`
	Strict         *bool    `json:"strict,omitempty"`  // Pointer to distinguish unset from false
	Private        *bool    `json:"private,omitempty"` // If true, workflow cannot be added to other repositories
	Labels         []string `json:"labels,omitempty"`

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
	RunsOn      string         `json:"runs-on,omitempty"`
	RunName     string         `json:"run-name,omitempty"`
	Steps       []any          `json:"steps,omitempty"`       // Custom workflow steps
	PostSteps   []any          `json:"post-steps,omitempty"`  // Post-workflow steps
	Environment map[string]any `json:"environment,omitempty"` // GitHub environment
	Container   map[string]any `json:"container,omitempty"`
	Services    map[string]any `json:"services,omitempty"`
	Cache       map[string]any `json:"cache,omitempty"`

	// Import and inclusion
	Imports        any      `json:"imports,omitempty"`         // Can be string or array
	Include        any      `json:"include,omitempty"`         // Can be string or array
	InlinedImports bool     `json:"inlined-imports,omitempty"` // If true, inline all imports at compile time instead of using runtime-import macros
	Resources      []string `json:"resources,omitempty"`       // Additional workflow .md or action .yml files to fetch alongside this workflow

	// Metadata
	Metadata      map[string]string    `json:"metadata,omitempty"` // Custom metadata key-value pairs
	SecretMasking *SecretMaskingConfig `json:"secret-masking,omitempty"`
	Observability *ObservabilityConfig `json:"observability,omitempty"`

	// Rate limiting configuration
	RateLimit *RateLimitConfig `json:"rate-limit,omitempty"`

	// Checkout configuration for the agent job.
	// Controls how actions/checkout is invoked.
	// Can be a single CheckoutConfig object or an array of CheckoutConfig objects.
	// Set to false to disable the default checkout step entirely.
	Checkout         any               `json:"checkout,omitempty"` // Raw value (object, array, or false)
	CheckoutConfigs  []*CheckoutConfig `json:"-"`                  // Parsed checkout configs (not in JSON)
	CheckoutDisabled bool              `json:"-"`                  // true when checkout: false is set in frontmatter
}

// ParseFrontmatterConfig creates a FrontmatterConfig from a raw frontmatter map
// This provides a single entry point for converting untyped frontmatter into
// a structured configuration with better error handling.
func ParseFrontmatterConfig(frontmatter map[string]any) (*FrontmatterConfig, error) {
	frontmatterTypesLog.Printf("Parsing frontmatter config with %d fields", len(frontmatter))
	var config FrontmatterConfig

	// Use JSON marshaling for the entire frontmatter conversion
	// This automatically handles all field mappings
	jsonBytes, err := json.Marshal(frontmatter)
	if err != nil {
		frontmatterTypesLog.Printf("Failed to marshal frontmatter: %v", err)
		return nil, fmt.Errorf("failed to marshal frontmatter to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		frontmatterTypesLog.Printf("Failed to unmarshal frontmatter: %v", err)
		return nil, fmt.Errorf("failed to unmarshal frontmatter into config: %w", err)
	}

	// Parse typed Runtimes field if runtimes exist
	if len(config.Runtimes) > 0 {
		runtimesTyped, err := parseRuntimesConfig(config.Runtimes)
		if err == nil {
			config.RuntimesTyped = runtimesTyped
			frontmatterTypesLog.Printf("Parsed typed runtimes config with %d runtimes", countRuntimes(runtimesTyped))
		}
	}

	// Parse typed Permissions field if permissions exist
	if len(config.Permissions) > 0 {
		permissionsTyped, err := parsePermissionsConfig(config.Permissions)
		if err == nil {
			config.PermissionsTyped = permissionsTyped
			frontmatterTypesLog.Print("Parsed typed permissions config")
		}
	}

	// Parse checkout field - supports single object, array of objects, or false to disable
	if config.Checkout != nil {
		if checkoutValue, ok := config.Checkout.(bool); ok && !checkoutValue {
			config.CheckoutDisabled = true
			frontmatterTypesLog.Print("Checkout disabled via checkout: false")
		} else {
			checkoutConfigs, err := ParseCheckoutConfigs(config.Checkout)
			if err == nil {
				config.CheckoutConfigs = checkoutConfigs
				frontmatterTypesLog.Printf("Parsed checkout config: %d entries", len(checkoutConfigs))
			}
		}
	}

	frontmatterTypesLog.Printf("Successfully parsed frontmatter config: name=%s, engine=%s", config.Name, config.Engine)
	return &config, nil
}

// parseRuntimesConfig converts a map[string]any to RuntimesConfig
func parseRuntimesConfig(runtimes map[string]any) (*RuntimesConfig, error) {
	config := &RuntimesConfig{}

	for runtimeID, configAny := range runtimes {
		configMap, ok := configAny.(map[string]any)
		if !ok {
			continue
		}

		// Extract version (optional)
		var version string
		if versionAny, hasVersion := configMap["version"]; hasVersion {
			// Convert version to string
			switch v := versionAny.(type) {
			case string:
				version = v
			case int:
				version = strconv.Itoa(v)
			case float64:
				if v == float64(int(v)) {
					version = strconv.Itoa(int(v))
				} else {
					version = fmt.Sprintf("%g", v)
				}
			default:
				continue
			}
		}

		// Extract if condition (optional)
		var ifCondition string
		if ifAny, hasIf := configMap["if"]; hasIf {
			if ifStr, ok := ifAny.(string); ok {
				ifCondition = ifStr
			}
		}

		// Extract action-repo and action-version overrides (optional)
		actionRepo, _ := configMap["action-repo"].(string)
		actionVersion, _ := configMap["action-version"].(string)

		// Create runtime config with all fields
		runtimeConfig := &RuntimeConfig{
			Version:       version,
			If:            ifCondition,
			ActionRepo:    actionRepo,
			ActionVersion: actionVersion,
		}

		// Map to specific runtime field
		switch runtimeID {
		case "node":
			config.Node = runtimeConfig
		case "python":
			config.Python = runtimeConfig
		case "go":
			config.Go = runtimeConfig
		case "uv":
			config.UV = runtimeConfig
		case "bun":
			config.Bun = runtimeConfig
		case "deno":
			config.Deno = runtimeConfig
		case "dotnet":
			config.Dotnet = runtimeConfig
		case "elixir":
			config.Elixir = runtimeConfig
		case "haskell":
			config.Haskell = runtimeConfig
		case "java":
			config.Java = runtimeConfig
		case "ruby":
			config.Ruby = runtimeConfig
		}
	}

	return config, nil
}

// parsePermissionsConfig converts a map[string]any to PermissionsConfig
func parsePermissionsConfig(permissions map[string]any) (*PermissionsConfig, error) {
	config := &PermissionsConfig{}

	// Check if it's a shorthand permission (single string value)
	if len(permissions) == 1 {
		for key, value := range permissions {
			if strValue, ok := value.(string); ok {
				shorthandPerms := []string{"read-all", "write-all", "read", "write", "none"}
				for _, shorthand := range shorthandPerms {
					if key == shorthand || strValue == shorthand {
						config.Shorthand = shorthand
						return config, nil
					}
				}
			}
		}
	}

	// Parse detailed permissions
	for scope, level := range permissions {
		if levelStr, ok := level.(string); ok {
			switch scope {
			// GitHub Actions permission scopes
			case "actions":
				config.Actions = levelStr
			case "checks":
				config.Checks = levelStr
			case "contents":
				config.Contents = levelStr
			case "deployments":
				config.Deployments = levelStr
			case "id-token":
				config.IDToken = levelStr
			case "issues":
				config.Issues = levelStr
			case "discussions":
				config.Discussions = levelStr
			case "packages":
				config.Packages = levelStr
			case "pages":
				config.Pages = levelStr
			case "pull-requests":
				config.PullRequests = levelStr
			case "repository-projects":
				config.RepositoryProjects = levelStr
			case "security-events":
				config.SecurityEvents = levelStr
			case "statuses":
				config.Statuses = levelStr
			case "organization-projects":
				config.OrganizationProjects = levelStr
			// GitHub App-only permission scopes
			case "administration":
				config.Administration = levelStr
			case "environments":
				config.Environments = levelStr
			case "git-signing":
				config.GitSigning = levelStr
			case "vulnerability-alerts":
				config.VulnerabilityAlerts = levelStr
			case "workflows":
				config.Workflows = levelStr
			case "repository-hooks":
				config.RepositoryHooks = levelStr
			case "single-file":
				config.SingleFile = levelStr
			case "codespaces":
				config.Codespaces = levelStr
			case "repository-custom-properties":
				config.RepositoryCustomProperties = levelStr
			case "members":
				config.Members = levelStr
			case "organization-administration":
				config.OrganizationAdministration = levelStr
			case "team-discussions":
				config.TeamDiscussions = levelStr
			case "organization-hooks":
				config.OrganizationHooks = levelStr
			case "organization-members":
				config.OrganizationMembers = levelStr
			case "organization-packages":
				config.OrganizationPackages = levelStr
			case "organization-self-hosted-runners":
				config.OrganizationSelfHostedRunners = levelStr
			case "organization-custom-org-roles":
				config.OrganizationCustomOrgRoles = levelStr
			case "organization-custom-properties":
				config.OrganizationCustomProperties = levelStr
			case "organization-custom-repository-roles":
				config.OrganizationCustomRepositoryRoles = levelStr
			case "organization-announcement-banners":
				config.OrganizationAnnouncementBanners = levelStr
			case "organization-events":
				config.OrganizationEvents = levelStr
			case "organization-plan":
				config.OrganizationPlan = levelStr
			case "organization-user-blocking":
				config.OrganizationUserBlocking = levelStr
			case "organization-personal-access-token-requests":
				config.OrganizationPersonalAccessTokenReqs = levelStr
			case "organization-personal-access-tokens":
				config.OrganizationPersonalAccessTokens = levelStr
			case "organization-copilot":
				config.OrganizationCopilot = levelStr
			case "organization-codespaces":
				config.OrganizationCodespaces = levelStr
			case "email-addresses":
				config.EmailAddresses = levelStr
			case "codespaces-lifecycle-admin":
				config.CodespacesLifecycleAdmin = levelStr
			case "codespaces-metadata":
				config.CodespacesMetadata = levelStr
			}
		}
	}

	return config, nil
}

// countRuntimes counts the number of non-nil runtimes in RuntimesConfig
func countRuntimes(config *RuntimesConfig) int {
	if config == nil {
		return 0
	}
	count := 0
	if config.Node != nil {
		count++
	}
	if config.Python != nil {
		count++
	}
	if config.Go != nil {
		count++
	}
	if config.UV != nil {
		count++
	}
	if config.Bun != nil {
		count++
	}
	if config.Deno != nil {
		count++
	}
	return count
}

// ExtractMapField is a convenience wrapper for extracting map[string]any fields
// from frontmatter. This maintains backward compatibility with existing extraction
// patterns while preserving original types (avoiding JSON conversion which would
// convert all numbers to float64).
//
// Returns an empty map if the key doesn't exist (for backward compatibility).
func ExtractMapField(frontmatter map[string]any, key string) map[string]any {
	// Check if key exists and value is not nil
	value, exists := frontmatter[key]
	if !exists || value == nil {
		frontmatterTypesLog.Printf("Field '%s' not found in frontmatter, returning empty map", key)
		return make(map[string]any)
	}

	// Direct type assertion to preserve original types (especially integers)
	// This avoids JSON marshaling which would convert integers to float64
	if valueMap, ok := value.(map[string]any); ok {
		frontmatterTypesLog.Printf("Extracted map field '%s' with %d entries", key, len(valueMap))
		return valueMap
	}

	// For backward compatibility, return empty map if not a map
	frontmatterTypesLog.Printf("Field '%s' is not a map type, returning empty map", key)
	return make(map[string]any)
}

// ToMap converts FrontmatterConfig back to map[string]any for backward compatibility
// This allows gradual migration from map[string]any to strongly-typed config
func (fc *FrontmatterConfig) ToMap() map[string]any {
	result := make(map[string]any)

	// Core fields
	if fc.Name != "" {
		result["name"] = fc.Name
	}
	if fc.Description != "" {
		result["description"] = fc.Description
	}
	if fc.Engine != "" {
		result["engine"] = fc.Engine
	}
	if fc.Source != "" {
		result["source"] = fc.Source
	}
	if fc.TrackerID != "" {
		result["tracker-id"] = fc.TrackerID
	}
	if fc.Version != "" {
		result["version"] = fc.Version
	}
	if fc.TimeoutMinutes != 0 {
		result["timeout-minutes"] = fc.TimeoutMinutes
	}
	if fc.Strict != nil {
		result["strict"] = *fc.Strict
	}
	if len(fc.Labels) > 0 {
		result["labels"] = fc.Labels
	}

	// Configuration sections
	if fc.Tools != nil {
		result["tools"] = fc.Tools.ToMap()
	}
	if fc.MCPServers != nil {
		result["mcp-servers"] = fc.MCPServers
	}
	// Prefer RuntimesTyped over Runtimes for conversion
	if fc.RuntimesTyped != nil {
		result["runtimes"] = runtimesConfigToMap(fc.RuntimesTyped)
	} else if fc.Runtimes != nil {
		result["runtimes"] = fc.Runtimes
	}
	if fc.Jobs != nil {
		result["jobs"] = fc.Jobs
	}
	if fc.SafeOutputs != nil {
		// Convert SafeOutputsConfig to map - would need a ToMap method
		result["safe-outputs"] = fc.SafeOutputs
	}
	if fc.MCPScripts != nil {
		// Convert MCPScriptsConfig to map - would need a ToMap method
		result["mcp-scripts"] = fc.MCPScripts
	}

	// Event and trigger configuration
	if fc.On != nil {
		result["on"] = fc.On
	}
	// Prefer PermissionsTyped over Permissions for conversion
	if fc.PermissionsTyped != nil {
		result["permissions"] = permissionsConfigToMap(fc.PermissionsTyped)
	} else if fc.Permissions != nil {
		result["permissions"] = fc.Permissions
	}
	if fc.Concurrency != nil {
		result["concurrency"] = fc.Concurrency
	}
	if fc.If != "" {
		result["if"] = fc.If
	}

	// Network and sandbox
	if fc.Network != nil {
		// Convert NetworkPermissions to map format
		// If allowed list is just ["defaults"], convert to string format "defaults"
		if len(fc.Network.Allowed) == 1 && fc.Network.Allowed[0] == "defaults" && fc.Network.Firewall == nil && len(fc.Network.Blocked) == 0 {
			result["network"] = "defaults"
		} else {
			networkMap := make(map[string]any)
			if len(fc.Network.Allowed) > 0 {
				networkMap["allowed"] = fc.Network.Allowed
			}
			if len(fc.Network.Blocked) > 0 {
				networkMap["blocked"] = fc.Network.Blocked
			}
			if fc.Network.Firewall != nil {
				networkMap["firewall"] = fc.Network.Firewall
			}
			if len(networkMap) > 0 {
				result["network"] = networkMap
			}
		}
	}
	if fc.Sandbox != nil {
		result["sandbox"] = fc.Sandbox
	}

	// Features and environment
	if fc.Features != nil {
		result["features"] = fc.Features
	}
	if fc.Env != nil {
		result["env"] = fc.Env
	}
	if fc.Secrets != nil {
		result["secrets"] = fc.Secrets
	}

	// Execution settings
	if fc.RunsOn != "" {
		result["runs-on"] = fc.RunsOn
	}
	if fc.RunName != "" {
		result["run-name"] = fc.RunName
	}
	if fc.Steps != nil {
		result["steps"] = fc.Steps
	}
	if fc.PostSteps != nil {
		result["post-steps"] = fc.PostSteps
	}
	if fc.Environment != nil {
		result["environment"] = fc.Environment
	}
	if fc.Container != nil {
		result["container"] = fc.Container
	}
	if fc.Services != nil {
		result["services"] = fc.Services
	}
	if fc.Cache != nil {
		result["cache"] = fc.Cache
	}

	// Import and inclusion
	if fc.Imports != nil {
		result["imports"] = fc.Imports
	}
	if fc.Include != nil {
		result["include"] = fc.Include
	}

	// Metadata
	if fc.Metadata != nil {
		result["metadata"] = fc.Metadata
	}
	if fc.SecretMasking != nil {
		result["secret-masking"] = fc.SecretMasking
	}

	return result
}

// runtimesConfigToMap converts RuntimesConfig back to map[string]any
func runtimesConfigToMap(config *RuntimesConfig) map[string]any {
	if config == nil {
		return nil
	}

	result := make(map[string]any)

	if config.Node != nil {
		nodeMap := map[string]any{}
		if config.Node.Version != "" {
			nodeMap["version"] = config.Node.Version
		}
		if config.Node.If != "" {
			nodeMap["if"] = config.Node.If
		}
		if len(nodeMap) > 0 {
			result["node"] = nodeMap
		}
	}
	if config.Python != nil {
		pythonMap := map[string]any{}
		if config.Python.Version != "" {
			pythonMap["version"] = config.Python.Version
		}
		if config.Python.If != "" {
			pythonMap["if"] = config.Python.If
		}
		if len(pythonMap) > 0 {
			result["python"] = pythonMap
		}
	}
	if config.Go != nil {
		goMap := map[string]any{}
		if config.Go.Version != "" {
			goMap["version"] = config.Go.Version
		}
		if config.Go.If != "" {
			goMap["if"] = config.Go.If
		}
		if len(goMap) > 0 {
			result["go"] = goMap
		}
	}
	if config.UV != nil {
		uvMap := map[string]any{}
		if config.UV.Version != "" {
			uvMap["version"] = config.UV.Version
		}
		if config.UV.If != "" {
			uvMap["if"] = config.UV.If
		}
		if len(uvMap) > 0 {
			result["uv"] = uvMap
		}
	}
	if config.Bun != nil {
		bunMap := map[string]any{}
		if config.Bun.Version != "" {
			bunMap["version"] = config.Bun.Version
		}
		if config.Bun.If != "" {
			bunMap["if"] = config.Bun.If
		}
		if len(bunMap) > 0 {
			result["bun"] = bunMap
		}
	}
	if config.Deno != nil {
		denoMap := map[string]any{}
		if config.Deno.Version != "" {
			denoMap["version"] = config.Deno.Version
		}
		if config.Deno.If != "" {
			denoMap["if"] = config.Deno.If
		}
		if len(denoMap) > 0 {
			result["deno"] = denoMap
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// permissionsConfigToMap converts PermissionsConfig back to map[string]any
func permissionsConfigToMap(config *PermissionsConfig) map[string]any {
	if config == nil {
		return nil
	}

	// If shorthand is set, return it directly
	if config.Shorthand != "" {
		return map[string]any{config.Shorthand: config.Shorthand}
	}

	result := make(map[string]any)

	// GitHub Actions permission scopes
	if config.Actions != "" {
		result["actions"] = config.Actions
	}
	if config.Checks != "" {
		result["checks"] = config.Checks
	}
	if config.Contents != "" {
		result["contents"] = config.Contents
	}
	if config.Deployments != "" {
		result["deployments"] = config.Deployments
	}
	if config.IDToken != "" {
		result["id-token"] = config.IDToken
	}
	if config.Issues != "" {
		result["issues"] = config.Issues
	}
	if config.Discussions != "" {
		result["discussions"] = config.Discussions
	}
	if config.Packages != "" {
		result["packages"] = config.Packages
	}
	if config.Pages != "" {
		result["pages"] = config.Pages
	}
	if config.PullRequests != "" {
		result["pull-requests"] = config.PullRequests
	}
	if config.RepositoryProjects != "" {
		result["repository-projects"] = config.RepositoryProjects
	}
	if config.SecurityEvents != "" {
		result["security-events"] = config.SecurityEvents
	}
	if config.Statuses != "" {
		result["statuses"] = config.Statuses
	}
	if config.OrganizationProjects != "" {
		result["organization-projects"] = config.OrganizationProjects
	}

	// GitHub App-only permission scopes - repository-level
	if config.Administration != "" {
		result["administration"] = config.Administration
	}
	if config.Environments != "" {
		result["environments"] = config.Environments
	}
	if config.GitSigning != "" {
		result["git-signing"] = config.GitSigning
	}
	if config.VulnerabilityAlerts != "" {
		result["vulnerability-alerts"] = config.VulnerabilityAlerts
	}
	if config.Workflows != "" {
		result["workflows"] = config.Workflows
	}
	if config.RepositoryHooks != "" {
		result["repository-hooks"] = config.RepositoryHooks
	}
	if config.SingleFile != "" {
		result["single-file"] = config.SingleFile
	}
	if config.Codespaces != "" {
		result["codespaces"] = config.Codespaces
	}
	if config.RepositoryCustomProperties != "" {
		result["repository-custom-properties"] = config.RepositoryCustomProperties
	}

	// GitHub App-only permission scopes - organization-level
	if config.Members != "" {
		result["members"] = config.Members
	}
	if config.OrganizationAdministration != "" {
		result["organization-administration"] = config.OrganizationAdministration
	}
	if config.TeamDiscussions != "" {
		result["team-discussions"] = config.TeamDiscussions
	}
	if config.OrganizationHooks != "" {
		result["organization-hooks"] = config.OrganizationHooks
	}
	if config.OrganizationMembers != "" {
		result["organization-members"] = config.OrganizationMembers
	}
	if config.OrganizationPackages != "" {
		result["organization-packages"] = config.OrganizationPackages
	}
	if config.OrganizationSelfHostedRunners != "" {
		result["organization-self-hosted-runners"] = config.OrganizationSelfHostedRunners
	}
	if config.OrganizationCustomOrgRoles != "" {
		result["organization-custom-org-roles"] = config.OrganizationCustomOrgRoles
	}
	if config.OrganizationCustomProperties != "" {
		result["organization-custom-properties"] = config.OrganizationCustomProperties
	}
	if config.OrganizationCustomRepositoryRoles != "" {
		result["organization-custom-repository-roles"] = config.OrganizationCustomRepositoryRoles
	}
	if config.OrganizationAnnouncementBanners != "" {
		result["organization-announcement-banners"] = config.OrganizationAnnouncementBanners
	}
	if config.OrganizationEvents != "" {
		result["organization-events"] = config.OrganizationEvents
	}
	if config.OrganizationPlan != "" {
		result["organization-plan"] = config.OrganizationPlan
	}
	if config.OrganizationUserBlocking != "" {
		result["organization-user-blocking"] = config.OrganizationUserBlocking
	}
	if config.OrganizationPersonalAccessTokenReqs != "" {
		result["organization-personal-access-token-requests"] = config.OrganizationPersonalAccessTokenReqs
	}
	if config.OrganizationPersonalAccessTokens != "" {
		result["organization-personal-access-tokens"] = config.OrganizationPersonalAccessTokens
	}
	if config.OrganizationCopilot != "" {
		result["organization-copilot"] = config.OrganizationCopilot
	}
	if config.OrganizationCodespaces != "" {
		result["organization-codespaces"] = config.OrganizationCodespaces
	}

	// GitHub App-only permission scopes - user-level
	if config.EmailAddresses != "" {
		result["email-addresses"] = config.EmailAddresses
	}
	if config.CodespacesLifecycleAdmin != "" {
		result["codespaces-lifecycle-admin"] = config.CodespacesLifecycleAdmin
	}
	if config.CodespacesMetadata != "" {
		result["codespaces-metadata"] = config.CodespacesMetadata
	}

	if len(result) == 0 {
		return nil
	}

	return result
}
