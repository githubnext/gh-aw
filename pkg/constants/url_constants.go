package constants

// URL represents a URL string.
// This semantic type distinguishes URLs from arbitrary strings,
// making URL parameters explicit and enabling future validation logic.
//
// Example usage:
//
//	const DefaultMCPRegistryURL URL = "https://api.mcp.github.com/v0.1"
//	func FetchFromRegistry(url URL) error { ... }
type URL string

// DocURL represents a documentation URL for error messages and help text.
// This semantic type distinguishes documentation URLs from arbitrary URLs,
// making documentation references explicit and centralized for easier maintenance.
//
// Example usage:
//
//	const DocsEnginesURL DocURL = "https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/engines.md"
//	func formatError(msg string, docURL DocURL) string { ... }
type DocURL string

// String returns the string representation of the documentation URL
func (d DocURL) String() string {
	return string(d)
}

// IsValid returns true if the documentation URL is non-empty
func (d DocURL) IsValid() bool {
	return len(d) > 0
}

// DefaultMCPRegistryURL is the default MCP registry URL.
const DefaultMCPRegistryURL URL = "https://api.mcp.github.com/v0.1"

// PublicGitHubHost is the public GitHub host URL.
// This is used as the default GitHub host and for the gh-aw repository itself,
// which is always hosted on public GitHub regardless of enterprise host settings.
const PublicGitHubHost URL = "https://github.com"

// GitHubCopilotMCPDomain is the domain for the hosted GitHub MCP server.
// Used when github tool is configured with mode: remote.
const GitHubCopilotMCPDomain = "api.githubcopilot.com"

// Documentation URLs for validation error messages.
// These URLs point to the relevant documentation pages that help users
// understand and resolve validation errors.
const (
	// DocsEnginesURL is the documentation URL for engine configuration
	DocsEnginesURL DocURL = "https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/engines.md"

	// DocsToolsURL is the documentation URL for tools and MCP server configuration
	DocsToolsURL DocURL = "https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/tools.md"

	// DocsGitHubToolsURL is the documentation URL for GitHub tools configuration
	DocsGitHubToolsURL DocURL = "https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/tools.md#github-tools-github"

	// DocsPermissionsURL is the documentation URL for GitHub permissions configuration
	DocsPermissionsURL DocURL = "https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/permissions.md"

	// DocsNetworkURL is the documentation URL for network configuration
	DocsNetworkURL DocURL = "https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/network.md"

	// DocsSandboxURL is the documentation URL for sandbox configuration
	DocsSandboxURL DocURL = "https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/sandbox.md"
)
