package types

// MCPServerAuth holds authentication configuration for an HTTP MCP server.
// It is used to configure token-based authentication that the MCP gateway
// handles transparently on behalf of the agent.
type MCPServerAuth struct {
	// Type is the authentication mechanism. Currently supported values:
	//   "github-oidc" - GitHub Actions OIDC token (short-lived JWT signed by GitHub).
	//     The gateway requests a fresh token from the GitHub OIDC endpoint on each
	//     proxied request (or caches it with TTL-based refresh) and injects it as
	//     "Authorization: Bearer <token>".  Requires id-token: write permission on
	//     the agent job.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Audience is the "aud" claim for the OIDC token.  Set this to the URL of the
	// MCP server so the server can reject tokens that were not minted for it.
	// Required when Type is "github-oidc".
	Audience string `json:"audience,omitempty" yaml:"audience,omitempty"`
}

// BaseMCPServerConfig contains the shared fields common to all MCP server configurations.
// This base type is embedded by both parser.MCPServerConfig and workflow.MCPServerConfig
// to eliminate duplication while allowing each to have domain-specific fields and struct tags.
type BaseMCPServerConfig struct {
	// Common execution fields
	Command string            `json:"command,omitempty" yaml:"command,omitempty"` // Command to execute (for stdio mode)
	Args    []string          `json:"args,omitempty" yaml:"args,omitempty"`       // Arguments for the command
	Env     map[string]string `json:"env,omitempty" yaml:"env,omitempty"`         // Environment variables

	// Type and version
	Type    string `json:"type,omitempty" yaml:"type,omitempty"`       // MCP server type (stdio, http, local, remote)
	Version string `json:"version,omitempty" yaml:"version,omitempty"` // Optional version/tag

	// HTTP-specific fields
	URL     string            `json:"url,omitempty" yaml:"url,omitempty"`         // URL for HTTP mode MCP servers
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"` // HTTP headers for HTTP mode

	// Container-specific fields
	Container      string   `json:"container,omitempty" yaml:"container,omitempty"`           // Container image for the MCP server
	Entrypoint     string   `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`         // Optional entrypoint override for container
	EntrypointArgs []string `json:"entrypointArgs,omitempty" yaml:"entrypointArgs,omitempty"` // Arguments passed to container entrypoint
	Mounts         []string `json:"mounts,omitempty" yaml:"mounts,omitempty"`                 // Volume mounts for container (format: "source:dest:mode")

	// Authentication for HTTP MCP servers (mutually exclusive with a github-app credential)
	Auth *MCPServerAuth `json:"auth,omitempty" yaml:"auth,omitempty"` // Token-based auth (e.g. github-oidc)
}
