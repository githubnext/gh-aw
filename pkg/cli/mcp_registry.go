package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/httputil"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpRegistryLog = logger.New("cli:mcp_registry")

// MCPRegistryServerForProcessing represents a flattened server for internal use
type MCPRegistryServerForProcessing struct {
	Name                 string                `json:"name"`
	Description          string                `json:"description"`
	Repository           string                `json:"repository"`
	Command              string                `json:"command"`
	Args                 []string              `json:"args"`
	RuntimeHint          string                `json:"runtime_hint"`
	RuntimeArguments     []string              `json:"runtime_arguments"`
	Transport            string                `json:"transport"`
	Config               map[string]any        `json:"config"`
	EnvironmentVariables []EnvironmentVariable `json:"environment_variables"`
}

// MCPRegistryClient handles communication with MCP registries
type MCPRegistryClient struct {
	registryURL string
	httpClient  *httputil.Client
}

// NewMCPRegistryClient creates a new MCP registry client
func NewMCPRegistryClient(registryURL string) *MCPRegistryClient {
	if registryURL == "" {
		registryURL = constants.DefaultMCPRegistryURL
	}

	mcpRegistryLog.Printf("Creating MCP registry client: url=%s", registryURL)

	return &MCPRegistryClient{
		registryURL: registryURL,
		httpClient:  httputil.NewClient(nil),
	}
}

// SearchServers searches for MCP servers in the registry by fetching all servers and filtering locally
func (c *MCPRegistryClient) SearchServers(query string) ([]MCPRegistryServerForProcessing, error) {
	mcpRegistryLog.Printf("Searching MCP servers: query=%q", query)

	// Always use servers endpoint for listing all servers
	searchURL := fmt.Sprintf("%s/servers", c.registryURL)

	// Create HTTP request with proper headers
	req, err := c.httpClient.NewRequest("GET", searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry request: %w", err)
	}

	// Make HTTP request with spinner
	spinnerMessage := fmt.Sprintf("Fetching servers from %s...", searchURL)
	spinner := console.NewSpinner(spinnerMessage)
	spinner.Start()
	resp, err := c.httpClient.Do(req)
	spinner.Stop()

	if err != nil {
		return nil, fmt.Errorf("failed to search MCP registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := httputil.ReadResponseBody(resp)
		return nil, httputil.FormatHTTPError(resp.StatusCode, body, "MCP registry")
	}

	// Parse response
	body, err := httputil.ReadResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	var response ServerListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse registry response: %w", err)
	}

	// Convert servers to flattened format and filter by status
	mcpRegistryLog.Printf("Processing %d servers from registry", len(response.Servers))
	servers := make([]MCPRegistryServerForProcessing, 0, len(response.Servers))
	for _, server := range response.Servers {
		// Only include active servers
		if server.Status != StatusActive {
			continue
		}

		processedServer := MCPRegistryServerForProcessing{
			Name:        server.Name,
			Description: server.Description,
		}

		// Set repository URL if available
		if server.Repository.URL != "" {
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

			// Set runtime hint (used for the actual command execution)
			processedServer.RuntimeHint = pkg.RuntimeHint

			// Extract runtime arguments
			runtimeArgs := make([]string, 0)
			for _, arg := range pkg.RuntimeArguments {
				if arg.Type == ArgumentTypePositional && arg.Value != "" {
					runtimeArgs = append(runtimeArgs, arg.Value)
				}
			}
			processedServer.RuntimeArguments = runtimeArgs

			// Extract string values from package arguments as command args
			args := make([]string, 0)
			for _, arg := range pkg.PackageArguments {
				if arg.Type == ArgumentTypePositional && arg.Value != "" {
					args = append(args, arg.Value)
				}
			}
			processedServer.Args = args

			// Convert environment variables to config
			if len(pkg.EnvironmentVariables) > 0 {
				processedServer.Config = make(map[string]any)
				envVars := make(map[string]any)

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

				// Preserve environment variable metadata for proper GitHub Actions conversion
				processedServer.EnvironmentVariables = pkg.EnvironmentVariables
			}
		} else if len(server.Remotes) > 0 {
			// Handle remote servers
			remote := server.Remotes[0]
			processedServer.Transport = remote.Type
			processedServer.Config = map[string]any{
				"url": remote.URL,
			}

			// Add headers if present
			if len(remote.Headers) > 0 {
				headers := make(map[string]any)
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

		mcpRegistryLog.Printf("Filtered to %d servers matching query", len(filteredServers))
		return filteredServers, nil
	}

	// Validate minimum server count for production registry
	// Note: This validation helps detect issues with the registry API, but we make it more lenient
	// to accommodate potential changes in the registry size
	if strings.Contains(c.registryURL, "api.mcp.github.com") && len(servers) < 10 {
		return nil, fmt.Errorf("registry validation failed: expected at least 10 servers from production registry, got %d\nThis may indicate an issue with the registry API or access restrictions", len(servers))
	}

	return servers, nil
}

// GetServer gets a specific server by name from the registry
func (c *MCPRegistryClient) GetServer(serverName string) (*MCPRegistryServerForProcessing, error) {
	mcpRegistryLog.Printf("Getting MCP server: name=%s", serverName)

	// Use the servers endpoint and filter locally, just like SearchServers
	serversURL := fmt.Sprintf("%s/servers", c.registryURL)

	// Create HTTP request with proper headers
	req, err := c.httpClient.NewRequest("GET", serversURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry request: %w", err)
	}

	// Make HTTP request with spinner
	spinner := console.NewSpinner(fmt.Sprintf("Fetching MCP server '%s'...", serverName))
	spinner.Start()
	resp, err := c.httpClient.Do(req)
	spinner.Stop()

	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := httputil.ReadResponseBody(resp)
		return nil, httputil.FormatHTTPError(resp.StatusCode, body, "MCP registry")
	}

	// Parse response
	body, err := httputil.ReadResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	var response ServerListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}

	// Find exact match by name, filtering locally
	for _, server := range response.Servers {
		if server.Name == serverName && server.Status == StatusActive {
			// Convert to flattened format similar to SearchServers
			processedServer := MCPRegistryServerForProcessing{
				Name:        server.Name,
				Description: server.Description,
			}

			// Set repository URL if available
			if server.Repository.URL != "" {
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

				// Set runtime hint (used for the actual command execution)
				processedServer.RuntimeHint = pkg.RuntimeHint

				// Extract runtime arguments
				runtimeArgs := make([]string, 0)
				for _, arg := range pkg.RuntimeArguments {
					if arg.Type == ArgumentTypePositional && arg.Value != "" {
						runtimeArgs = append(runtimeArgs, arg.Value)
					}
				}
				processedServer.RuntimeArguments = runtimeArgs

				// Extract string values from package arguments as command args
				args := make([]string, 0)
				for _, arg := range pkg.PackageArguments {
					if arg.Type == ArgumentTypePositional && arg.Value != "" {
						args = append(args, arg.Value)
					}
				}
				processedServer.Args = args

				// Convert environment variables to config
				if len(pkg.EnvironmentVariables) > 0 {
					processedServer.Config = make(map[string]any)
					envVars := make(map[string]any)

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

					// Preserve environment variable metadata for proper GitHub Actions conversion
					processedServer.EnvironmentVariables = pkg.EnvironmentVariables
				}
			} else if len(server.Remotes) > 0 {
				// Handle remote servers
				remote := server.Remotes[0]
				processedServer.Transport = remote.Type
				processedServer.Config = map[string]any{
					"url": remote.URL,
				}

				// Add headers if present
				if len(remote.Headers) > 0 {
					headers := make(map[string]any)
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

			mcpRegistryLog.Printf("Found MCP server: name=%s, transport=%s", serverName, processedServer.Transport)
			return &processedServer, nil
		}
	}

	mcpRegistryLog.Printf("MCP server not found: name=%s", serverName)
	return nil, fmt.Errorf("MCP server '%s' not found in registry", serverName)
}

// cleanMCPToolID removes common MCP prefixes and suffixes from tool IDs
// Examples: "notion-mcp" -> "notion", "mcp-notion" -> "notion", "some-mcp-server" -> "some-server"
func cleanMCPToolID(toolID string) string {
	cleaned := toolID

	// Remove "mcp-" prefix
	cleaned = strings.TrimPrefix(cleaned, "mcp-")

	// Remove "-mcp" suffix
	cleaned = strings.TrimSuffix(cleaned, "-mcp")

	// If the result is empty, use the original
	if cleaned == "" {
		return toolID
	}

	return cleaned
}
