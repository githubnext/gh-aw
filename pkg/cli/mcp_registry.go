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

// MCPRegistryRuntimeArgument represents a runtime argument
type MCPRegistryRuntimeArgument struct {
	Format     string `json:"format"`
	IsRequired bool   `json:"is_required"`
	Type       string `json:"type"`
	Value      string `json:"value"`
	ValueHint  string `json:"value_hint"`
}

// MCPRegistryRepository represents repository information
type MCPRegistryRepository struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	URL    string `json:"url"`
	Readme string `json:"readme"`
}

// MCPRegistryVersionDetail represents version information
type MCPRegistryVersionDetail struct {
	Version     string `json:"version"`
	IsLatest    bool   `json:"is_latest"`
	ReleaseDate string `json:"release_date"`
}

// MCPRegistryPackage represents a package within a server
type MCPRegistryPackage struct {
	Name                 string                        `json:"name"`
	RegistryName         string                        `json:"registry_name"`
	Version              string                        `json:"version"`
	RuntimeHint          string                        `json:"runtime_hint"`
	RuntimeArguments     []MCPRegistryRuntimeArgument  `json:"runtime_arguments"`
	EnvironmentVariables []map[string]interface{}      `json:"environment_variables"`
}

// MCPRegistryServerData represents the actual server data nested in the response
type MCPRegistryServerData struct {
	ID            string                    `json:"id"`
	Name          string                    `json:"name"`
	Description   string                    `json:"description"`
	Repository    MCPRegistryRepository     `json:"repository"`
	CreatedAt     string                    `json:"created_at"`
	UpdatedAt     string                    `json:"updated_at"`
	VersionDetail MCPRegistryVersionDetail  `json:"version_detail"`
	Packages      []MCPRegistryPackage      `json:"packages"`
}

// MCPRegistryServerWrapper represents a server entry from the MCP registry API
type MCPRegistryServerWrapper struct {
	Server MCPRegistryServerData `json:"server"`
}

// MCPRegistryServer represents a flattened server for internal use
type MCPRegistryServer struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Repository  string                 `json:"repository"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args"`
	Transport   string                 `json:"transport"`
	Config      map[string]interface{} `json:"config"`
}

// MCPRegistryResponse represents the response from the MCP registry API
type MCPRegistryResponse struct {
	Servers  []MCPRegistryServerWrapper `json:"servers"`
	Metadata map[string]interface{}     `json:"metadata"`
}

// MCPRegistryClient handles communication with MCP registries
type MCPRegistryClient struct {
	registryURL string
	httpClient  *http.Client
}

// NewMCPRegistryClient creates a new MCP registry client
func NewMCPRegistryClient(registryURL string) *MCPRegistryClient {
	if registryURL == "" {
		registryURL = "https://api.mcp.github.com/v0"
	}

	return &MCPRegistryClient{
		registryURL: registryURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchServers searches for MCP servers in the registry by fetching all servers and filtering locally
func (c *MCPRegistryClient) SearchServers(query string) ([]MCPRegistryServer, error) {
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

	// Convert wrapped servers to flattened format
	servers := make([]MCPRegistryServer, 0, len(response.Servers))
	for _, wrapper := range response.Servers {
		server := MCPRegistryServer{
			ID:          wrapper.Server.ID,
			Name:        wrapper.Server.Name,
			Description: wrapper.Server.Description,
			Repository:  wrapper.Server.Repository.URL,
		}

		// Extract transport and config from first package if available
		if len(wrapper.Server.Packages) > 0 {
			pkg := wrapper.Server.Packages[0]
			
			// Determine transport type from runtime hint
			switch pkg.RuntimeHint {
			case "node", "python", "binary":
				server.Transport = "stdio"
				server.Command = pkg.RegistryName
				
				// Extract string values from runtime arguments
				args := make([]string, 0, len(pkg.RuntimeArguments))
				for _, arg := range pkg.RuntimeArguments {
					args = append(args, arg.Value)
				}
				server.Args = args
			default:
				server.Transport = "stdio" // default fallback
			}

			// Convert environment variables to config
			if len(pkg.EnvironmentVariables) > 0 {
				server.Config = make(map[string]interface{})
				server.Config["env"] = pkg.EnvironmentVariables
			}
		} else {
			server.Transport = "stdio" // default fallback
		}

		servers = append(servers, server)
	}

	// Apply local filtering if query is provided
	if query != "" {
		filteredServers := make([]MCPRegistryServer, 0)
		queryLower := strings.ToLower(query)
		
		for _, server := range servers {
			// Check if query matches ID, name, or description (case-insensitive)
			if strings.Contains(strings.ToLower(server.ID), queryLower) ||
			   strings.Contains(strings.ToLower(server.Name), queryLower) ||
			   strings.Contains(strings.ToLower(server.Description), queryLower) {
				filteredServers = append(filteredServers, server)
			}
		}
		
		return filteredServers, nil
	}

	return servers, nil
}

// GetServer gets a specific server by ID from the registry
func (c *MCPRegistryClient) GetServer(serverID string) (*MCPRegistryServer, error) {
	// Build server URL
	serverURL := fmt.Sprintf("%s/servers/%s", c.registryURL, url.PathEscape(serverID))

	// Make HTTP request with spinner
	spinner := console.NewSpinner(fmt.Sprintf("Fetching MCP server '%s'...", serverID))
	spinner.Start()
	resp, err := c.httpClient.Get(serverURL)
	spinner.Stop()

	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("MCP server '%s' not found in registry", serverID)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP registry returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	var wrapper MCPRegistryServerWrapper
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}

	// Convert to flattened format
	server := MCPRegistryServer{
		ID:          wrapper.Server.ID,
		Name:        wrapper.Server.Name,
		Description: wrapper.Server.Description,
		Repository:  wrapper.Server.Repository.URL,
	}

	// Extract transport and config from first package if available
	if len(wrapper.Server.Packages) > 0 {
		pkg := wrapper.Server.Packages[0]
		
		// Determine transport type from runtime hint
		switch pkg.RuntimeHint {
		case "node", "python", "binary":
			server.Transport = "stdio"
			server.Command = pkg.RegistryName
			
			// Extract string values from runtime arguments
			args := make([]string, 0, len(pkg.RuntimeArguments))
			for _, arg := range pkg.RuntimeArguments {
				args = append(args, arg.Value)
			}
			server.Args = args
		default:
			server.Transport = "stdio" // default fallback
		}

		// Convert environment variables to config
		if len(pkg.EnvironmentVariables) > 0 {
			server.Config = make(map[string]interface{})
			server.Config["env"] = pkg.EnvironmentVariables
		}
	} else {
		server.Transport = "stdio" // default fallback
	}

	return &server, nil
}
