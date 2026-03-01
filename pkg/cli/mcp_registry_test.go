//go:build !integration

package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPRegistryClient_SearchServers(t *testing.T) {
	// Create a test server that mocks the MCP registry API
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers" {
			t.Errorf("Expected path /servers, got %s", r.URL.Path)
		}

		// Return mock response with v0.1 structure based on official specification
		response := `{
			"servers": [
				{
					"server": {
						"name": "io.github.makenotion/notion-mcp-server",
						"description": "MCP server for Notion integration",
						"version": "1.0.0",
						"repository": {
							"url": "https://github.com/example/notion-mcp",
							"source": "github"
						},
						"packages": [
							{
								"registryType": "npm",
								"identifier": "notion-mcp",
								"version": "1.0.0",
								"runtimeHint": "node",
								"transport": {
									"type": "stdio"
								},
								"packageArguments": [
									{
										"type": "positional",
										"value": "notion-mcp"
									}
								],
								"environmentVariables": [
									{
										"name": "NOTION_TOKEN",
										"description": "Notion API token",
										"isRequired": true,
										"isSecret": true
									}
								]
							}
						]
					},
					"_meta": {
						"io.modelcontextprotocol.registry/official": {
							"status": "active",
							"publishedAt": "2025-01-01T10:30:00Z",
							"isLatest": true
						}
					}
				}
			]
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer testServer.Close()

	// Create client with test server URL
	client := NewMCPRegistryClient(testServer.URL)

	// Test search
	servers, err := client.SearchServers("notion")
	if err != nil {
		t.Fatalf("SearchServers failed: %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("Expected 1 server, got %d", len(servers))
	}

	mcpServer := servers[0]
	if mcpServer.Name != "io.github.makenotion/notion-mcp-server" {
		t.Errorf("Expected server name 'io.github.makenotion/notion-mcp-server', got '%s'", mcpServer.Name)
	}

	if mcpServer.Transport != "stdio" {
		t.Errorf("Expected transport 'stdio', got '%s'", mcpServer.Transport)
	}

	if mcpServer.Command != "notion-mcp" {
		t.Errorf("Expected command 'notion-mcp', got '%s'", mcpServer.Command)
	}

	if len(mcpServer.Args) != 1 || mcpServer.Args[0] != "notion-mcp" {
		t.Errorf("Expected args ['notion-mcp'], got %v", mcpServer.Args)
	}
}

func TestNewMCPRegistryClient_DefaultURL(t *testing.T) {
	client := NewMCPRegistryClient("")
	expectedURL := "https://api.mcp.github.com/v0.1"
	if client.registryURL != expectedURL {
		t.Errorf("Expected default registry URL '%s', got '%s'", expectedURL, client.registryURL)
	}
}

func TestNewMCPRegistryClient_CustomURL(t *testing.T) {
	customURL := "https://custom.registry.com/v1"
	client := NewMCPRegistryClient(customURL)
	if client.registryURL != customURL {
		t.Errorf("Expected custom registry URL '%s', got '%s'", customURL, client.registryURL)
	}
}
