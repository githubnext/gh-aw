package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
)

// MCPRegistryServer represents a server entry from the MCP registry
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
	Servers []MCPRegistryServer `json:"servers"`
	Total   int                 `json:"total"`
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

// SearchServers searches for MCP servers in the registry
func (c *MCPRegistryClient) SearchServers(query string) ([]MCPRegistryServer, error) {
	var searchURL string
	var spinnerMessage string

	if query == "" {
		// Use servers endpoint for listing all servers
		searchURL = fmt.Sprintf("%s/servers", c.registryURL)
		spinnerMessage = "Fetching all MCP servers..."
	} else {
		// Use search endpoint for specific queries
		searchURL = fmt.Sprintf("%s/servers/search", c.registryURL)
		params := url.Values{}
		params.Set("q", query)
		searchURL = fmt.Sprintf("%s?%s", searchURL, params.Encode())
		spinnerMessage = fmt.Sprintf("Searching MCP registry for '%s'...", query)
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

	return response.Servers, nil
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

	var server MCPRegistryServer
	if err := json.Unmarshal(body, &server); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}

	return &server, nil
}
