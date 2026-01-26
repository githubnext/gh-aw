package cli

// Local types inferred from GitHub MCP Registry API v0.1 structure
// Based on the official specification at:
// https://github.com/modelcontextprotocol/registry/blob/main/docs/reference/api/openapi.yaml

// ServerListResponse represents the response from the /v0.1/servers endpoint
type ServerListResponse struct {
	Servers  []ServerResponse `json:"servers"`
	Metadata *Metadata        `json:"metadata,omitempty"`
}

// Metadata represents pagination metadata
type Metadata struct {
	NextCursor string `json:"nextCursor,omitempty"`
	Count      int    `json:"count,omitempty"`
}

// ServerResponse represents the API response format with separated server data and registry metadata
type ServerResponse struct {
	Server ServerDetail   `json:"server"`
	Meta   map[string]any `json:"_meta,omitempty"`
}

// ServerDetail represents an MCP server in the registry
type ServerDetail struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Title        string         `json:"title,omitempty"`
	Version      string         `json:"version"`
	Repository   *Repository    `json:"repository,omitempty"`
	Packages     []MCPPackage   `json:"packages,omitempty"`
	Remotes      []Remote       `json:"remotes,omitempty"`
	WebsiteURL   string         `json:"websiteUrl,omitempty"`
	Schema       string         `json:"$schema,omitempty"`
	InternalMeta map[string]any `json:"_meta,omitempty"`
}

// Repository represents the source repository information
type Repository struct {
	URL       string `json:"url"`
	Source    string `json:"source,omitempty"`
	ID        string `json:"id,omitempty"`
	Subfolder string `json:"subfolder,omitempty"`
}

// MCPPackage represents a package configuration for an MCP server
type MCPPackage struct {
	RegistryType         string                `json:"registryType,omitempty"`
	RegistryBaseURL      string                `json:"registryBaseUrl,omitempty"`
	Identifier           string                `json:"identifier,omitempty"`
	Version              string                `json:"version,omitempty"`
	FileSHA256           string                `json:"fileSha256,omitempty"`
	RuntimeHint          string                `json:"runtimeHint,omitempty"`
	Transport            *Transport            `json:"transport,omitempty"`
	RuntimeArguments     []Argument            `json:"runtimeArguments,omitempty"`
	PackageArguments     []Argument            `json:"packageArguments,omitempty"`
	EnvironmentVariables []EnvironmentVariable `json:"environmentVariables,omitempty"`
}

// Transport represents the transport configuration
type Transport struct {
	Type      string                `json:"type"`
	URL       string                `json:"url,omitempty"`
	Headers   []EnvironmentVariable `json:"headers,omitempty"`
	Variables map[string]any        `json:"variables,omitempty"`
}

// Argument represents a command line argument
type Argument struct {
	Type        string         `json:"type"`
	Value       string         `json:"value,omitempty"`
	Name        string         `json:"name,omitempty"`      // For named arguments
	ValueHint   string         `json:"valueHint,omitempty"` // For positional arguments
	IsRepeated  bool           `json:"isRepeated,omitempty"`
	Description string         `json:"description,omitempty"`
	IsRequired  bool           `json:"isRequired,omitempty"`
	Format      string         `json:"format,omitempty"`
	IsSecret    bool           `json:"isSecret,omitempty"`
	Default     string         `json:"default,omitempty"`
	Placeholder string         `json:"placeholder,omitempty"`
	Choices     []string       `json:"choices,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
}

// EnvironmentVariable represents an environment variable configuration
type EnvironmentVariable struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	IsRequired  bool     `json:"isRequired,omitempty"`
	IsSecret    bool     `json:"isSecret,omitempty"`
	Default     string   `json:"default,omitempty"`
	Format      string   `json:"format,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Choices     []string `json:"choices,omitempty"`
}

// Remote represents a remote server configuration
type Remote struct {
	Type      string                `json:"type"`
	URL       string                `json:"url"`
	Headers   []EnvironmentVariable `json:"headers,omitempty"`
	Variables map[string]any        `json:"variables,omitempty"`
}

// Status constants for server status
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

// Argument type constants
const (
	ArgumentTypePositional = "positional"
	ArgumentTypeNamed      = "named"
)
