package workflow

import (
	"maps"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/types"
)

var toolsTypesLog = logger.New("workflow:tools_types")

// ToolsConfig represents the unified configuration for all tools in a workflow.
// This type provides a structured alternative to the pervasive map[string]any pattern.
// It includes strongly-typed fields for built-in tools and a flexible Custom map for
// MCP server configurations.
//
// # Migration Pattern
//
// This unified type helps eliminate unnecessary type assertions and runtime validation
// by replacing map[string]any with strongly-typed configuration structs.
//
// # Usage Examples
//
// Creating a ToolsConfig from a map[string]any:
//
//	toolsMap := map[string]any{
//	    "github": map[string]any{"allowed": []any{"issue_read"}},
//	    "bash":   []any{"echo", "ls"},
//	}
//	config, err := ParseToolsConfig(toolsMap)
//	if err != nil {
//	    // handle error
//	}
//
// Converting back to map[string]any for legacy code:
//
//	toolsMap := config.ToMap()
//
// # Backward Compatibility
//
// For functions that currently accept map[string]any, create wrapper functions
// that handle conversion:
//
//	// New signature using ToolsConfig
//	func processTools(config *ToolsConfig) error {
//	    if config.GitHub != nil {
//	        // Access strongly-typed GitHub config
//	    }
//	    return nil
//	}
//
//	// Backward compatibility wrapper
//	func processToolsFromMap(tools map[string]any) error {
//	    config, err := ParseToolsConfig(tools)
//	    if err != nil {
//	        return err
//	    }
//	    return processTools(config)
//	}
//
// # Design Notes
//
//   - Built-in tool fields use pointers to distinguish between "not configured" (nil)
//     and "configured with defaults" (non-nil but empty struct)
//   - The Custom map stores MCP server configurations that aren't built-in tools
//   - The raw map is preserved for perfect round-trip conversion when needed
//   - Type alias Tools = ToolsConfig provides backward compatibility for existing code
type ToolsConfig struct {
	// Built-in tools - using pointers to distinguish between "not set" and "set to nil/empty"
	GitHub           *GitHubToolConfig           `yaml:"github,omitempty"`
	Bash             *BashToolConfig             `yaml:"bash,omitempty"`
	WebFetch         *WebFetchToolConfig         `yaml:"web-fetch,omitempty"`
	WebSearch        *WebSearchToolConfig        `yaml:"web-search,omitempty"`
	Edit             *EditToolConfig             `yaml:"edit,omitempty"`
	Playwright       *PlaywrightToolConfig       `yaml:"playwright,omitempty"`
	Qmd              *QmdToolConfig              `yaml:"qmd,omitempty"`
	AgenticWorkflows *AgenticWorkflowsToolConfig `yaml:"agentic-workflows,omitempty"`
	CacheMemory      *CacheMemoryToolConfig      `yaml:"cache-memory,omitempty"`
	RepoMemory       *RepoMemoryToolConfig       `yaml:"repo-memory,omitempty"`
	Timeout          *TemplatableInt32           `yaml:"timeout,omitempty"`
	StartupTimeout   *TemplatableInt32           `yaml:"startup-timeout,omitempty"`

	// Custom MCP tools (anything not in the above list)
	Custom map[string]MCPServerConfig `yaml:",inline"`

	// Raw map for backwards compatibility
	raw map[string]any
}

// Tools is a type alias for ToolsConfig for backward compatibility.
// New code should prefer using ToolsConfig to be explicit about the unified configuration pattern.
type Tools = ToolsConfig

// ParseToolsConfig creates a ToolsConfig from a map[string]any.
// This function provides backward compatibility for code that uses map[string]any.
// It parses all known tool types into their strongly-typed equivalents and stores
// unknown tools in the Custom map.
func ParseToolsConfig(toolsMap map[string]any) (*ToolsConfig, error) {
	toolsTypesLog.Printf("Parsing tools configuration: tool_count=%d", len(toolsMap))
	config := NewTools(toolsMap)
	toolNames := config.GetToolNames()
	toolsTypesLog.Printf("Parsed tools configuration: result_count=%d, tools=%v", len(toolNames), toolNames)
	return config, nil
}

// mcpServerConfigToMap converts an MCPServerConfig to map[string]any for backward compatibility
func mcpServerConfigToMap(config MCPServerConfig) map[string]any {
	result := make(map[string]any)

	// Add common fields if they're set
	if config.Command != "" {
		result["command"] = config.Command
	}
	if len(config.Args) > 0 {
		result["args"] = config.Args
	}
	if len(config.Env) > 0 {
		result["env"] = config.Env
	}
	if config.Mode != "" {
		result["mode"] = config.Mode
	}
	if config.Type != "" {
		result["type"] = config.Type
	}
	if config.Version != "" {
		result["version"] = config.Version
	}
	if len(config.Toolsets) > 0 {
		result["toolsets"] = config.Toolsets
	}

	// Add HTTP-specific fields
	if config.URL != "" {
		result["url"] = config.URL
	}
	if len(config.Headers) > 0 {
		result["headers"] = config.Headers
	}
	if config.Auth != nil {
		result["auth"] = config.Auth
	}

	// Add container-specific fields
	if config.Container != "" {
		result["container"] = config.Container
	}
	if config.Entrypoint != "" {
		result["entrypoint"] = config.Entrypoint
	}
	if len(config.EntrypointArgs) > 0 {
		result["entrypointArgs"] = config.EntrypointArgs
	}
	if len(config.Mounts) > 0 {
		result["mounts"] = config.Mounts
	}

	// Add guard policies if set
	if len(config.GuardPolicies) > 0 {
		result["guard-policies"] = config.GuardPolicies
	}

	// Add custom fields (these override standard fields if there are conflicts)
	maps.Copy(result, config.CustomFields)

	return result
}

// ToMap converts the ToolsConfig back to a map[string]any for backward compatibility.
// This is useful when interfacing with legacy code that expects map[string]any.
func (t *ToolsConfig) ToMap() map[string]any {
	if t == nil {
		toolsTypesLog.Print("Converting nil ToolsConfig to empty map")
		return make(map[string]any)
	}

	// Return the raw map if it exists
	if t.raw != nil {
		toolsTypesLog.Printf("Returning cached raw map with %d entries", len(t.raw))
		return t.raw
	}

	// Otherwise construct a new map from the fields
	toolsTypesLog.Print("Constructing map from ToolsConfig fields")
	result := make(map[string]any)

	if t.GitHub != nil {
		result["github"] = t.GitHub
	}
	if t.Bash != nil {
		result["bash"] = t.Bash.AllowedCommands
	}
	if t.WebFetch != nil {
		result["web-fetch"] = t.WebFetch
	}
	if t.WebSearch != nil {
		result["web-search"] = t.WebSearch
	}
	if t.Edit != nil {
		result["edit"] = t.Edit
	}
	if t.Playwright != nil {
		result["playwright"] = t.Playwright
	}
	if t.Qmd != nil {
		result["qmd"] = t.Qmd
	}
	if t.AgenticWorkflows != nil {
		result["agentic-workflows"] = t.AgenticWorkflows.Enabled
	}
	if t.CacheMemory != nil {
		result["cache-memory"] = t.CacheMemory.Raw
	}
	if t.RepoMemory != nil {
		result["repo-memory"] = t.RepoMemory.Raw
	}
	if t.Timeout != nil {
		result["timeout"] = t.Timeout.ToValue()
	}
	if t.StartupTimeout != nil {
		result["startup-timeout"] = t.StartupTimeout.ToValue()
	}

	// Add custom tools - convert MCPServerConfig to map[string]any
	for name, config := range t.Custom {
		result[name] = mcpServerConfigToMap(config)
	}

	toolsTypesLog.Printf("Constructed map with %d entries from ToolsConfig", len(result))
	return result
}

// GitHubToolName represents a GitHub tool name (e.g., "issue_read", "create_issue")
type GitHubToolName string

// GitHubAllowedTools is a slice of GitHub tool names
type GitHubAllowedTools []GitHubToolName

// ToStringSlice converts GitHubAllowedTools to []string
func (g GitHubAllowedTools) ToStringSlice() []string {
	result := make([]string, len(g))
	for i, tool := range g {
		result[i] = string(tool)
	}
	return result
}

// GitHubToolset represents a GitHub toolset name (e.g., "default", "repos", "issues")
type GitHubToolset string

// GitHubToolsets is a slice of GitHub toolset names
type GitHubToolsets []GitHubToolset

// ToStringSlice converts GitHubToolsets to []string
func (g GitHubToolsets) ToStringSlice() []string {
	result := make([]string, len(g))
	for i, toolset := range g {
		result[i] = string(toolset)
	}
	return result
}

// GitHubIntegrityLevel represents the minimum integrity level required for repository access
type GitHubIntegrityLevel string

const (
	// GitHubIntegrityNone allows access with no integrity requirements
	GitHubIntegrityNone GitHubIntegrityLevel = "none"
	// GitHubIntegrityUnapproved requires unapproved-level integrity
	GitHubIntegrityUnapproved GitHubIntegrityLevel = "unapproved"
	// GitHubIntegrityApproved requires approved-level integrity
	GitHubIntegrityApproved GitHubIntegrityLevel = "approved"
	// GitHubIntegrityMerged requires merged-level integrity
	GitHubIntegrityMerged GitHubIntegrityLevel = "merged"
)

// GitHubReposScope represents the repository scope for guard policy enforcement
// Can be one of: "all", "public", or an array of repository patterns
type GitHubReposScope any // string or []any (YAML-parsed arrays are []any)

// GitHubToolConfig represents the configuration for the GitHub tool
// Can be nil (enabled with defaults), string, or an object with specific settings
type GitHubToolConfig struct {
	Allowed     GitHubAllowedTools `yaml:"allowed,omitempty"`
	Mode        string             `yaml:"mode,omitempty"`
	Version     string             `yaml:"version,omitempty"`
	Args        []string           `yaml:"args,omitempty"`
	ReadOnly    bool               `yaml:"read-only,omitempty"`
	GitHubToken string             `yaml:"github-token,omitempty"`
	Toolset     GitHubToolsets     `yaml:"toolsets,omitempty"`
	Lockdown    bool               `yaml:"lockdown,omitempty"`
	GitHubApp   *GitHubAppConfig   `yaml:"github-app,omitempty"` // GitHub App configuration for token minting

	// Guard policy fields (flat syntax under github:)
	// AllowedRepos defines the access scope for policy enforcement.
	// Supports: "all", "public", or an array of patterns ["owner/repo", "owner/*"] (lowercase)
	AllowedRepos GitHubReposScope `yaml:"allowed-repos,omitempty"`
	// Repos is deprecated. Use AllowedRepos (yaml:"allowed-repos") instead.
	Repos GitHubReposScope `yaml:"repos,omitempty"`
	// MinIntegrity defines the minimum integrity level required: "none", "unapproved", "approved", "merged"
	MinIntegrity GitHubIntegrityLevel `yaml:"min-integrity,omitempty"`
	// BlockedUsers is an optional list of GitHub usernames whose content is unconditionally blocked.
	// Items from these users receive "blocked" integrity (below "none") and are always denied.
	BlockedUsers []string `yaml:"blocked-users,omitempty"`
	// BlockedUsersExpr holds a GitHub Actions expression (e.g. "${{ vars.BLOCKED_USERS }}") that
	// resolves at runtime to a comma- or newline-separated list of blocked usernames.
	// Set when the blocked-users field is a string expression rather than a literal array.
	BlockedUsersExpr string `yaml:"-"`
	// TrustedUsers is an optional list of GitHub usernames whose content is elevated to "approved"
	// integrity regardless of author_association. Takes precedence over min-integrity checks but
	// not over blocked-users. Requires min-integrity to be set.
	TrustedUsers []string `yaml:"trusted-users,omitempty"`
	// TrustedUsersExpr holds a GitHub Actions expression (e.g. "${{ vars.TRUSTED_USERS }}") that
	// resolves at runtime to a comma- or newline-separated list of trusted usernames.
	// Set when the trusted-users field is a string expression rather than a literal array.
	TrustedUsersExpr string `yaml:"-"`
	// ApprovalLabels is an optional list of GitHub label names that promote a content item's
	// effective integrity to "approved" when present. Does not override BlockedUsers.
	ApprovalLabels []string `yaml:"approval-labels,omitempty"`
	// ApprovalLabelsExpr holds a GitHub Actions expression (e.g. "${{ vars.APPROVAL_LABELS }}") that
	// resolves at runtime to a comma- or newline-separated list of approval label names.
	// Set when the approval-labels field is a string expression rather than a literal array.
	ApprovalLabelsExpr string `yaml:"-"`
}

// PlaywrightToolConfig represents the configuration for the Playwright tool
type PlaywrightToolConfig struct {
	Version string   `yaml:"version,omitempty"`
	Args    []string `yaml:"args,omitempty"`
}

// QmdDocCollection represents a named documentation collection for the qmd tool.
// Each collection indexes its own set of files and can optionally target a different
// repository via its own checkout configuration.
type QmdDocCollection struct {
	// Name is the collection identifier used in the qmd index.
	// Defaults to "docs-<index>" when not provided (e.g. "docs-0", "docs-1").
	Name string `yaml:"name,omitempty"`

	// Pattern is the glob pattern for files to include in this collection.
	// Example: "docs/**/*.md"
	Pattern string `yaml:"pattern,omitempty"`

	// Ignore is an optional list of glob patterns for files to exclude from the collection.
	// Example: ["**/node_modules/**", "**/*.test.md"]
	Ignore []string `yaml:"ignore,omitempty"`

	// Context is optional extra context injected into the qmd collection,
	// providing the agent with additional hints about the content (e.g. "GitHub Actions documentation").
	Context string `yaml:"context,omitempty"`

	// Checkout configures which repository to check out for this collection.
	// Uses the same syntax as the top-level checkout configuration.
	// Defaults to the current repository if not set.
	Checkout *CheckoutConfig `yaml:"checkout,omitempty"`
}

// QmdSearchEntry represents a single GitHub search entry whose results are
// downloaded and added to the qmd index as individual files.
type QmdSearchEntry struct {
	// Name is an optional name for the resulting qmd collection.
	// When empty, the collection is named "search-{index}".
	Name string `yaml:"name,omitempty"`

	// Type controls the search backend. Supported values:
	//   "code"   (default) – uses `gh search code` to find repository files
	//   "issues"           – uses `gh issue list` to fetch open issues from
	//                        a repository and save each as a markdown file
	// When type is "issues", Query is the repository slug ("owner/repo").
	// If Query is empty for an issue search, ${{ github.repository }} is used.
	Type string `yaml:"type,omitempty"`

	// Query is the GitHub code search query string (type "code") or the
	// repository slug "owner/repo" (type "issues").
	// Example (code):   "repo:owner/repo language:Markdown path:docs/"
	// Example (issues): "owner/repo"  (or empty to use current repository)
	Query string `yaml:"query,omitempty"`

	// Min is the minimum number of results required. If fewer are found
	// the activation step fails.
	Min int `yaml:"min,omitempty"`

	// Max is the maximum number of results to download.
	// Defaults to 30 (type "code") or 500 (type "issues") when not set.
	Max int `yaml:"max,omitempty"`

	// GitHubToken overrides the default GITHUB_TOKEN used to authenticate
	// the GitHub API request.
	// Mutually exclusive with GitHubApp.
	GitHubToken string `yaml:"github-token,omitempty"`

	// GitHubApp configures GitHub App-based authentication for the API request.
	// Mutually exclusive with GitHubToken.
	GitHubApp *GitHubAppConfig `yaml:"github-app,omitempty"`
}

// QmdToolConfig represents the configuration for the qmd documentation search tool.
// qmd (https://github.com/tobi/qmd) provides local vector search over documentation files.
// The index is built in a dedicated indexing job and shared via GitHub Actions cache, so no
// contents:read permission is needed in the agent job.
//
// Two sources can contribute to the index:
//
//  1. checkouts – glob-based collections from checked-out repositories
//  2. searches  – GitHub search queries whose results are downloaded as files
//
// Optionally, the index can be cached in GitHub Actions cache using the cache-key field.
// When cache-key is set without any sources (checkouts/searches), qmd operates in
// read-only mode: it restores the index from cache and skips all indexing steps.
type QmdToolConfig struct {
	// Checkouts is the list of named documentation collections.
	// Each collection can specify its own checkout to target a different repository.
	Checkouts []*QmdDocCollection `yaml:"checkouts,omitempty"`

	// Searches is the list of GitHub search queries whose results are downloaded
	// and added to the qmd index.
	Searches []*QmdSearchEntry `yaml:"searches,omitempty"`

	// CacheKey is an optional GitHub Actions cache key used to persist the qmd index
	// across workflow runs. When set:
	//   - If sources (checkouts/searches) are also configured: the index is built
	//     normally and then saved to the cache. On subsequent runs, the cached index is
	//     restored and the build steps are skipped if the cache hit is exact.
	//   - If no sources are configured (read-only mode): the index is restored directly
	//     from cache without any indexing steps.
	// Example: "qmd-index-${{ hashFiles('docs/**') }}"
	CacheKey string `yaml:"cache-key,omitempty"`

	// GPU controls whether node-llama-cpp (used by @tobilu/qmd internally) may use
	// GPU acceleration. Defaults to false: NODE_LLAMA_CPP_GPU=false is injected into
	// the indexing step so that GPU auto-detection is skipped on CPU-only runners.
	// Set to true only when the indexing runner has a GPU.
	GPU bool `yaml:"gpu,omitempty"`

	// RunsOn overrides the runner image for the indexing job.
	// Defaults to the same runner as the agent job (ubuntu-latest or as configured).
	// Example: "ubuntu-latest-gpu" or ["self-hosted", "gpu"]
	RunsOn string `yaml:"runs-on,omitempty"`
}

// BashToolConfig represents the configuration for the Bash tool
// Can be nil (all commands allowed) or an array of allowed commands
type BashToolConfig struct {
	AllowedCommands []string `yaml:"-"` // List of allowed bash commands
}

// WebFetchToolConfig represents the configuration for the web-fetch tool
type WebFetchToolConfig struct {
	// Currently an empty object or nil
}

// WebSearchToolConfig represents the configuration for the web-search tool
type WebSearchToolConfig struct {
	// Currently an empty object or nil
}

// EditToolConfig represents the configuration for the edit tool
type EditToolConfig struct {
	// Currently an empty object or nil
}

// AgenticWorkflowsToolConfig represents the configuration for the agentic-workflows tool
type AgenticWorkflowsToolConfig struct {
	// Can be boolean or nil
	Enabled bool `yaml:"-"`
}

// CacheMemoryToolConfig represents the configuration for cache-memory
// This is handled separately by the existing CacheMemoryConfig in cache.go
type CacheMemoryToolConfig struct {
	// Can be boolean, object, or array - handled by cache.go
	Raw any `yaml:"-"`
}

// MCPServerConfig represents the configuration for a custom MCP server.
// It embeds BaseMCPServerConfig for common fields and adds workflow-specific fields.
// This provides partial type safety for common MCP configuration fields
// while maintaining flexibility for truly dynamic configurations.
type MCPServerConfig struct {
	types.BaseMCPServerConfig

	// Workflow-specific fields
	Mode     string   `yaml:"mode,omitempty"`     // MCP server mode (stdio, http, remote, local)
	Toolsets []string `yaml:"toolsets,omitempty"` // Toolsets to enable

	// Guard policies for access control at the MCP gateway level
	// This is a general field that can hold server-specific policy configurations
	// For GitHub: policies are represented via GitHubAllowOnlyPolicy on GitHubToolConfig
	// For Jira/WorkIQ: define similar server-specific policy types
	GuardPolicies map[string]any `yaml:"guard-policies,omitempty"`

	// For truly dynamic configuration (server-specific fields not covered above)
	CustomFields map[string]any `yaml:",inline"`
}

// MCPGatewayRuntimeConfig represents the configuration for the MCP gateway runtime execution
// The gateway routes MCP server calls through a unified HTTP endpoint
// Per MCP Gateway Specification v1.0.0: All stdio-based MCP servers MUST be containerized.
// Direct command execution is not supported.
type MCPGatewayRuntimeConfig struct {
	Container            string            `yaml:"container,omitempty"`              // Container image for the gateway (required)
	Version              string            `yaml:"version,omitempty"`                // Optional version/tag for the container
	Entrypoint           string            `yaml:"entrypoint,omitempty"`             // Optional entrypoint override for the container
	Args                 []string          `yaml:"args,omitempty"`                   // Arguments for docker run
	EntrypointArgs       []string          `yaml:"entrypointArgs,omitempty"`         // Arguments passed to container entrypoint
	Env                  map[string]string `yaml:"env,omitempty"`                    // Environment variables for the gateway
	Port                 int               `yaml:"port,omitempty"`                   // Port for the gateway HTTP server (default: 8080)
	APIKey               string            `yaml:"api-key,omitempty"`                // API key for gateway authentication
	Domain               string            `yaml:"domain,omitempty"`                 // Domain for gateway URL (localhost or host.docker.internal)
	Mounts               []string          `yaml:"mounts,omitempty"`                 // Volume mounts for the gateway container (format: "source:dest:mode")
	PayloadDir           string            `yaml:"payload-dir,omitempty"`            // Directory path for storing large payload JSON files (must be absolute path)
	PayloadPathPrefix    string            `yaml:"payload-path-prefix,omitempty"`    // Path prefix to remap payload paths for agent containers (e.g., /workspace/payloads)
	PayloadSizeThreshold int               `yaml:"payload-size-threshold,omitempty"` // Size threshold in bytes for storing payloads to disk (default: 524288 = 512KB)
	TrustedBots          []string          `yaml:"trusted-bots,omitempty"`           // Additional bot identity strings to pass to the gateway, merged with its built-in list
	KeepaliveInterval    int               `yaml:"keepalive-interval,omitempty"`     // Keepalive ping interval in seconds for HTTP MCP backends (0=default 1500s, -1=disabled, >0=custom)
	OTLPEndpoint         string            `yaml:"-"`                                // OTLP collector endpoint (derived from observability.otlp, not user-settable)
	OTLPHeaders          string            `yaml:"-"`                                // Raw OTLP HTTP headers string (derived from observability.otlp, not user-settable)
}

// HasTool checks if a tool is present in the configuration
func (t *Tools) HasTool(name string) bool {
	if t == nil {
		return false
	}

	toolsTypesLog.Printf("Checking if tool exists: name=%s", name)

	switch name {
	case "github":
		return t.GitHub != nil
	case "bash":
		return t.Bash != nil
	case "web-fetch":
		return t.WebFetch != nil
	case "web-search":
		return t.WebSearch != nil
	case "edit":
		return t.Edit != nil
	case "playwright":
		return t.Playwright != nil
	case "qmd":
		return t.Qmd != nil
	case "agentic-workflows":
		return t.AgenticWorkflows != nil
	case "cache-memory":
		return t.CacheMemory != nil
	case "repo-memory":
		return t.RepoMemory != nil
	case "timeout":
		return t.Timeout != nil
	case "startup-timeout":
		return t.StartupTimeout != nil
	default:
		_, exists := t.Custom[name]
		return exists
	}
}

// GetToolNames returns a list of all tool names configured
func (t *Tools) GetToolNames() []string {
	if t == nil {
		return []string{}
	}

	toolsTypesLog.Print("Collecting configured tool names")
	names := []string{}

	if t.GitHub != nil {
		names = append(names, "github")
	}
	if t.Bash != nil {
		names = append(names, "bash")
	}
	if t.WebFetch != nil {
		names = append(names, "web-fetch")
	}
	if t.WebSearch != nil {
		names = append(names, "web-search")
	}
	if t.Edit != nil {
		names = append(names, "edit")
	}
	if t.Playwright != nil {
		names = append(names, "playwright")
	}
	if t.Qmd != nil {
		names = append(names, "qmd")
	}
	if t.AgenticWorkflows != nil {
		names = append(names, "agentic-workflows")
	}
	if t.CacheMemory != nil {
		names = append(names, "cache-memory")
	}
	if t.RepoMemory != nil {
		names = append(names, "repo-memory")
	}
	if t.Timeout != nil {
		names = append(names, "timeout")
	}
	if t.StartupTimeout != nil {
		names = append(names, "startup-timeout")
	}

	// Add custom tools
	for name := range t.Custom {
		names = append(names, name)
	}

	toolsTypesLog.Printf("Found %d configured tools: %v", len(names), names)
	return names
}
