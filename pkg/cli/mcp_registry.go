package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
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
	httpClient  *http.Client
}

// NewMCPRegistryClient creates a new MCP registry client
func NewMCPRegistryClient(registryURL string) *MCPRegistryClient {
	if registryURL == "" {
		registryURL = string(constants.DefaultMCPRegistryURL)
	}

	mcpRegistryLog.Printf("Creating MCP registry client: url=%s", registryURL)

	return &MCPRegistryClient{
		registryURL: registryURL,
		httpClient: &http.Client{
			Timeout: constants.DefaultHTTPClientTimeout,
		},
	}
}

// createRegistryRequest creates an HTTP request with appropriate headers for the MCP registry
func (c *MCPRegistryClient) createRegistryRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	// Set standard headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gh-aw-cli")

	return req, nil
}

// SearchServers searches for MCP servers in the registry by fetching all servers and filtering locally
func (c *MCPRegistryClient) SearchServers(ctx context.Context, query string) ([]MCPRegistryServerForProcessing, error) {
	mcpRegistryLog.Printf("Searching MCP servers: query=%q", query)

	searchURL := c.registryURL + "/servers"
	response, err := c.fetchServerList(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	servers := flattenRegistryServers(response.Servers)
	if query != "" {
		filteredServers := filterRegistryServersByQuery(servers, query)
		mcpRegistryLog.Printf("Filtered to %d servers matching query", len(filteredServers))
		return filteredServers, nil
	}
	if err := validateProductionRegistryServerCount(c.registryURL, len(servers)); err != nil {
		return nil, err
	}
	return servers, nil
}

func (c *MCPRegistryClient) fetchServerList(ctx context.Context, searchURL string) (*ServerListResponse, error) {
	req, err := c.createRegistryRequest(ctx, "GET", searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry request: %w", err)
	}

	spinner := console.NewSpinner(fmt.Sprintf("Fetching servers from %s...", searchURL))
	spinner.Start()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return nil, fmt.Errorf("failed to search MCP registry: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		spinner.Stop()
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		spinner.Stop()
		return nil, formatRegistryStatusError(resp.StatusCode, body)
	}

	var response ServerListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		spinner.Stop()
		return nil, fmt.Errorf("failed to parse registry response: %w", err)
	}

	spinner.StopWithMessage(fmt.Sprintf("✓ Fetched %d servers from registry", len(response.Servers)))
	return &response, nil
}

func formatRegistryStatusError(statusCode int, body []byte) error {
	message := string(body)
	switch statusCode {
	case http.StatusForbidden:
		return fmt.Errorf("MCP registry access forbidden (403): %s\nThis may be due to network or firewall restrictions", message)
	case http.StatusUnauthorized:
		return fmt.Errorf("MCP registry access unauthorized (401): %s\nAuthentication may be required", message)
	case http.StatusNotFound:
		return fmt.Errorf("MCP registry endpoint not found (404): %s\nPlease verify the registry URL is correct", message)
	case http.StatusTooManyRequests:
		return fmt.Errorf("MCP registry rate limit exceeded (429): %s\nPlease try again later", message)
	default:
		return fmt.Errorf("MCP registry returned status %d: %s", statusCode, message)
	}
}

func flattenRegistryServers(serverResponses []ServerResponse) []MCPRegistryServerForProcessing {
	mcpRegistryLog.Printf("Processing %d servers from registry", len(serverResponses))
	servers := make([]MCPRegistryServerForProcessing, 0, len(serverResponses))
	for _, serverResp := range serverResponses {
		if !isActiveRegistryServer(serverResp) {
			continue
		}
		servers = append(servers, flattenRegistryServer(serverResp.Server))
	}
	return servers
}

func isActiveRegistryServer(serverResp ServerResponse) bool {
	meta, ok := serverResp.Meta["io.modelcontextprotocol.registry/official"].(map[string]any)
	if !ok {
		return true
	}
	status, ok := meta["status"].(string)
	return !ok || status == StatusActive
}

func flattenRegistryServer(server ServerDetail) MCPRegistryServerForProcessing {
	processedServer := MCPRegistryServerForProcessing{
		Name:        server.Name,
		Description: server.Description,
	}
	if server.Repository != nil && server.Repository.URL != "" {
		processedServer.Repository = server.Repository.URL
	}

	switch {
	case len(server.Packages) > 0:
		applyRegistryPackageDetails(&processedServer, server.Packages[0])
	case len(server.Remotes) > 0:
		applyRegistryRemoteDetails(&processedServer, server.Remotes[0])
	default:
		processedServer.Transport = "stdio"
	}
	return processedServer
}

func applyRegistryPackageDetails(processedServer *MCPRegistryServerForProcessing, pkg MCPPackage) {
	if pkg.Transport != nil {
		processedServer.Transport = pkg.Transport.Type
	}
	if processedServer.Transport == "" {
		processedServer.Transport = "stdio"
	}
	processedServer.Command = pkg.Identifier
	processedServer.RuntimeHint = pkg.RuntimeHint
	processedServer.RuntimeArguments = extractPositionalRegistryArgumentValues(pkg.RuntimeArguments)
	processedServer.Args = extractPositionalRegistryArgumentValues(pkg.PackageArguments)
	if len(pkg.EnvironmentVariables) > 0 {
		processedServer.Config = map[string]any{"env": buildRegistryEnvironmentVariableConfig(pkg.EnvironmentVariables)}
		processedServer.EnvironmentVariables = pkg.EnvironmentVariables
	}
}

func applyRegistryRemoteDetails(processedServer *MCPRegistryServerForProcessing, remote Remote) {
	processedServer.Transport = remote.Type
	processedServer.Config = map[string]any{"url": remote.URL}
	if len(remote.Headers) > 0 {
		processedServer.Config["headers"] = buildRegistryHeaderConfig(remote.Headers)
	}
}

func extractPositionalRegistryArgumentValues(arguments []Argument) []string {
	values := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if argument.Type == ArgumentTypePositional && argument.Value != "" {
			values = append(values, argument.Value)
		}
	}
	return values
}

func buildRegistryEnvironmentVariableConfig(envVars []EnvironmentVariable) map[string]any {
	values := make(map[string]any, len(envVars))
	for _, envVar := range envVars {
		values[envVar.Name] = registryVariableValue(envVar.Name, envVar.Default, envVar.IsSecret)
	}
	return values
}

func buildRegistryHeaderConfig(headers []EnvironmentVariable) map[string]any {
	values := make(map[string]any, len(headers))
	for _, header := range headers {
		values[header.Name] = registryVariableValue(header.Name, header.Default, header.IsSecret)
	}
	return values
}

func registryVariableValue(name, defaultValue string, isSecret bool) any {
	placeholder := fmt.Sprintf("${%s}", name)
	switch {
	case isSecret:
		return placeholder
	case defaultValue != "":
		return defaultValue
	default:
		return placeholder
	}
}

func filterRegistryServersByQuery(servers []MCPRegistryServerForProcessing, query string) []MCPRegistryServerForProcessing {
	filteredServers := make([]MCPRegistryServerForProcessing, 0, len(servers))
	queryLower := strings.ToLower(query)
	for _, server := range servers {
		if strings.Contains(strings.ToLower(server.Name), queryLower) ||
			strings.Contains(strings.ToLower(server.Description), queryLower) {
			filteredServers = append(filteredServers, server)
		}
	}
	return filteredServers
}

func validateProductionRegistryServerCount(registryURL string, serverCount int) error {
	if strings.Contains(registryURL, "api.mcp.github.com") && serverCount < 10 {
		return fmt.Errorf("registry validation failed: expected at least 10 servers from production registry, got %d\nThis may indicate an issue with the registry API or access restrictions", serverCount)
	}
	return nil
}
