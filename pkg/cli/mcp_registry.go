package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
)

// MCPRegistryServer represents a server from the MCP registry API
type MCPRegistryServer struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	Version     string                 `json:"version"`
	Repository  *MCPRegistryRepository `json:"repository,omitempty"`
	Packages    []MCPRegistryPackage   `json:"packages,omitempty"`
	Remotes     []MCPRegistryRemote    `json:"remotes,omitempty"`
	WebsiteURL  string                 `json:"website_url,omitempty"`
}

// MCPRegistryRemote represents a remote server configuration
type MCPRegistryRemote struct {
	Type    string                    `json:"type"`
	URL     string                    `json:"url"`
	Headers []MCPRegistryRemoteHeader `json:"headers,omitempty"`
}

// MCPRegistryRemoteHeader represents a header for remote server
type MCPRegistryRemoteHeader struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	IsRequired  bool     `json:"is_required,omitempty"`
	IsSecret    bool     `json:"is_secret,omitempty"`
	Default     string   `json:"default,omitempty"`
	Choices     []string `json:"choices,omitempty"`
}

// MCPRegistryTransport represents transport configuration
type MCPRegistryTransport struct {
	Type string `json:"type"`
}

// MCPRegistryEnvironmentVariable represents an environment variable
type MCPRegistryEnvironmentVariable struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	IsRequired  bool     `json:"is_required,omitempty"`
	IsSecret    bool     `json:"is_secret,omitempty"`
	Default     string   `json:"default,omitempty"`
	Choices     []string `json:"choices,omitempty"`
}

// MCPRegistryArgument represents a package argument
type MCPRegistryArgument struct {
	Type        string                         `json:"type"`
	Name        string                         `json:"name,omitempty"`
	Value       string                         `json:"value,omitempty"`
	ValueHint   string                         `json:"value_hint,omitempty"`
	Description string                         `json:"description,omitempty"`
	IsRequired  bool                           `json:"is_required,omitempty"`
	IsRepeated  bool                           `json:"is_repeated,omitempty"`
	Default     string                         `json:"default,omitempty"`
	Format      string                         `json:"format,omitempty"`
	Choices     []string                       `json:"choices,omitempty"`
	Variables   map[string]MCPRegistryVariable `json:"variables,omitempty"`
}

// MCPRegistryVariable represents a variable in runtime arguments
type MCPRegistryVariable struct {
	Description string   `json:"description,omitempty"`
	Format      string   `json:"format,omitempty"`
	IsRequired  bool     `json:"is_required,omitempty"`
	Default     string   `json:"default,omitempty"`
	Choices     []string `json:"choices,omitempty"`
}

// MCPRegistryRepository represents repository information
type MCPRegistryRepository struct {
	URL       string `json:"url"`
	Source    string `json:"source"`
	ID        string `json:"id,omitempty"`
	Subfolder string `json:"subfolder,omitempty"`
}

// MCPRegistryPackage represents a package within a server
type MCPRegistryPackage struct {
	RegistryType         string                           `json:"registry_type"`
	RegistryBaseURL      string                           `json:"registry_base_url,omitempty"`
	Identifier           string                           `json:"identifier"`
	Version              string                           `json:"version"`
	RuntimeHint          string                           `json:"runtime_hint,omitempty"`
	Transport            MCPRegistryTransport             `json:"transport"`
	PackageArguments     []MCPRegistryArgument            `json:"package_arguments,omitempty"`
	RuntimeArguments     []MCPRegistryArgument            `json:"runtime_arguments,omitempty"`
	EnvironmentVariables []MCPRegistryEnvironmentVariable `json:"environment_variables,omitempty"`
	FileSha256           string                           `json:"file_sha256,omitempty"`
}

// MCPRegistryResponse represents the response from the MCP registry API
type MCPRegistryResponse struct {
	Servers  []MCPRegistryServer    `json:"servers"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MCPRegistryServerForProcessing represents a flattened server for internal use
type MCPRegistryServerForProcessing struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Repository  string                 `json:"repository"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args"`
	Transport   string                 `json:"transport"`
	Config      map[string]interface{} `json:"config"`
}

// MCPRegistryClient handles communication with MCP registries
type MCPRegistryClient struct {
	registryURL string
	httpClient  *http.Client
}

// NewMCPRegistryClient creates a new MCP registry client
func NewMCPRegistryClient(registryURL string) *MCPRegistryClient {
	if registryURL == "" {
		registryURL = "https://registry.modelcontextprotocol.io/v0"
	}

	return &MCPRegistryClient{
		registryURL: registryURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchServers searches for MCP servers in the registry by fetching all servers and filtering locally
func (c *MCPRegistryClient) SearchServers(query string) ([]MCPRegistryServerForProcessing, error) {
	// Always use servers endpoint for listing all servers
	searchURL := fmt.Sprintf("%s/servers", c.registryURL)

	var spinnerMessage string
	if query == "" {
		spinnerMessage = "Fetching all MCP servers..."
	} else {
		spinnerMessage = fmt.Sprintf("Fetching MCP servers and filtering for '%s'...", query)
	}

	// Make HTTP request with spinner
	spinner := console.NewSpinner(spinnerMessage)
	spinner.Start()
	resp, err := c.httpClient.Get(searchURL)
	spinner.Stop()

	if err != nil {
		return nil, fmt.Errorf("failed to search MCP registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP registry returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	var response MCPRegistryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse registry response: %w", err)
	}

	// Convert servers to flattened format and filter by status
	servers := make([]MCPRegistryServerForProcessing, 0, len(response.Servers))
	for _, server := range response.Servers {
		// Only include active servers
		if server.Status != "active" {
			continue
		}

		processedServer := MCPRegistryServerForProcessing{
			Name:        server.Name,
			Description: server.Description,
		}

		// Set repository URL if available
		if server.Repository != nil {
			processedServer.Repository = server.Repository.URL
		}

		// Extract transport and config from first package if available
		if len(server.Packages) > 0 {
			pkg := server.Packages[0]

			// Use transport type from package
			processedServer.Transport = pkg.Transport.Type
			if processedServer.Transport == "" {
				processedServer.Transport = "stdio" // default fallback
			}

			// Set command from package identifier
			processedServer.Command = pkg.Identifier

			// Extract string values from package arguments as command args
			args := make([]string, 0)
			for _, arg := range pkg.PackageArguments {
				if arg.Type == "positional" && arg.Value != "" {
					args = append(args, arg.Value)
				}
			}
			processedServer.Args = args

			// Convert environment variables to config
			if len(pkg.EnvironmentVariables) > 0 {
				processedServer.Config = make(map[string]interface{})
				envVars := make(map[string]interface{})

				for _, envVar := range pkg.EnvironmentVariables {
					// Use name as key, and create a placeholder value for secrets
					if envVar.IsSecret {
						envVars[envVar.Name] = fmt.Sprintf("${%s}", envVar.Name)
					} else if envVar.Default != "" {
						envVars[envVar.Name] = envVar.Default
					} else {
						envVars[envVar.Name] = fmt.Sprintf("${%s}", envVar.Name)
					}
				}
				processedServer.Config["env"] = envVars
			}
		} else if len(server.Remotes) > 0 {
			// Handle remote servers
			remote := server.Remotes[0]
			processedServer.Transport = remote.Type
			processedServer.Config = map[string]interface{}{
				"url": remote.URL,
			}

			// Add headers if present
			if len(remote.Headers) > 0 {
				headers := make(map[string]interface{})
				for _, header := range remote.Headers {
					if header.IsSecret {
						headers[header.Name] = fmt.Sprintf("${%s}", header.Name)
					} else if header.Default != "" {
						headers[header.Name] = header.Default
					} else {
						headers[header.Name] = fmt.Sprintf("${%s}", header.Name)
					}
				}
				processedServer.Config["headers"] = headers
			}
		} else {
			processedServer.Transport = "stdio" // default fallback
		}

		servers = append(servers, processedServer)
	}

	// Apply local filtering if query is provided
	if query != "" {
		filteredServers := make([]MCPRegistryServerForProcessing, 0)
		queryLower := strings.ToLower(query)

		for _, server := range servers {
			// Check if query matches name or description (case-insensitive)
			if strings.Contains(strings.ToLower(server.Name), queryLower) ||
				strings.Contains(strings.ToLower(server.Description), queryLower) {
				filteredServers = append(filteredServers, server)
			}
		}

		return filteredServers, nil
	}

	return servers, nil
}

// GetServer gets a specific server by name from the registry
func (c *MCPRegistryClient) GetServer(serverName string) (*MCPRegistryServerForProcessing, error) {
	// Build server URL - we'll search by name since the new API doesn't use server IDs for direct access
	searchURL := fmt.Sprintf("%s/servers?search=%s", c.registryURL, url.QueryEscape(serverName))

	// Make HTTP request with spinner
	spinner := console.NewSpinner(fmt.Sprintf("Fetching MCP server '%s'...", serverName))
	spinner.Start()
	resp, err := c.httpClient.Get(searchURL)
	spinner.Stop()

	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP registry returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	var response MCPRegistryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}

	// Find exact match by name
	for _, server := range response.Servers {
		if server.Name == serverName && server.Status == "active" {
			// Convert to flattened format similar to SearchServers
			processedServer := MCPRegistryServerForProcessing{
				Name:        server.Name,
				Description: server.Description,
			}

			// Set repository URL if available
			if server.Repository != nil {
				processedServer.Repository = server.Repository.URL
			}

			// Extract transport and config from first package if available
			if len(server.Packages) > 0 {
				pkg := server.Packages[0]

				// Use transport type from package
				processedServer.Transport = pkg.Transport.Type
				if processedServer.Transport == "" {
					processedServer.Transport = "stdio" // default fallback
				}

				// Set command from package identifier
				processedServer.Command = pkg.Identifier

				// Extract string values from package arguments as command args
				args := make([]string, 0)
				for _, arg := range pkg.PackageArguments {
					if arg.Type == "positional" && arg.Value != "" {
						args = append(args, arg.Value)
					}
				}
				processedServer.Args = args

				// Convert environment variables to config
				if len(pkg.EnvironmentVariables) > 0 {
					processedServer.Config = make(map[string]interface{})
					envVars := make(map[string]interface{})

					for _, envVar := range pkg.EnvironmentVariables {
						// Use name as key, and create a placeholder value for secrets
						if envVar.IsSecret {
							envVars[envVar.Name] = fmt.Sprintf("${%s}", envVar.Name)
						} else if envVar.Default != "" {
							envVars[envVar.Name] = envVar.Default
						} else {
							envVars[envVar.Name] = fmt.Sprintf("${%s}", envVar.Name)
						}
					}
					processedServer.Config["env"] = envVars
				}
			} else if len(server.Remotes) > 0 {
				// Handle remote servers
				remote := server.Remotes[0]
				processedServer.Transport = remote.Type
				processedServer.Config = map[string]interface{}{
					"url": remote.URL,
				}

				// Add headers if present
				if len(remote.Headers) > 0 {
					headers := make(map[string]interface{})
					for _, header := range remote.Headers {
						if header.IsSecret {
							headers[header.Name] = fmt.Sprintf("${%s}", header.Name)
						} else if header.Default != "" {
							headers[header.Name] = header.Default
						} else {
							headers[header.Name] = fmt.Sprintf("${%s}", header.Name)
						}
					}
					processedServer.Config["headers"] = headers
				}
			} else {
				processedServer.Transport = "stdio" // default fallback
			}

			return &processedServer, nil
		}
	}

	return nil, fmt.Errorf("MCP server '%s' not found in registry", serverName)
}
