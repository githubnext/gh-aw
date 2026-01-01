package cli

import "github.com/githubnext/gh-aw/pkg/logger"

var mcpRegistryTypesLog = logger.New("cli:mcp_registry_types")

// Local types inferred from GitHub MCP Registry API structure
// This replaces the dependency on github.com/modelcontextprotocol/registry

// ServerListResponse represents the response from the /servers endpoint
type ServerListResponse struct {
	Servers []Server `json:"servers"`
}

// Server represents an MCP server in the registry
type Server struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	Version     string       `json:"version,omitempty"`
	Repository  Repository   `json:"repository,omitempty"`
	Packages    []MCPPackage `json:"packages,omitempty"`
	Remotes     []Remote     `json:"remotes,omitempty"`
}

// Repository represents the source repository information
type Repository struct {
	URL    string `json:"url"`
	Source string `json:"source,omitempty"`
}

// MCPPackage represents a package configuration for an MCP server
type MCPPackage struct {
	RegistryType         string                `json:"registry_type,omitempty"`
	Identifier           string                `json:"identifier,omitempty"`
	Version              string                `json:"version,omitempty"`
	RuntimeHint          string                `json:"runtime_hint,omitempty"`
	Transport            Transport             `json:"transport,omitempty"`
	RuntimeArguments     []Argument            `json:"runtime_arguments,omitempty"`
	PackageArguments     []Argument            `json:"package_arguments,omitempty"`
	EnvironmentVariables []EnvironmentVariable `json:"environment_variables,omitempty"`
}

// Transport represents the transport configuration
type Transport struct {
	Type string `json:"type"`
}

// Argument represents a command line argument
type Argument struct {
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
}

// EnvironmentVariable represents an environment variable configuration
type EnvironmentVariable struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsRequired  bool   `json:"is_required,omitempty"`
	IsSecret    bool   `json:"is_secret,omitempty"`
	Default     string `json:"default,omitempty"`
}

// Remote represents a remote server configuration
type Remote struct {
	Type    string   `json:"type"`
	URL     string   `json:"url"`
	Headers []Header `json:"headers,omitempty"`
}

// Header represents an HTTP header for remote servers
type Header struct {
	Name     string `json:"name"`
	IsSecret bool   `json:"is_secret,omitempty"`
	Default  string `json:"default,omitempty"`
}

// Status constants for server status
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

// Argument type constants
const (
	ArgumentTypePositional = "positional"
)
