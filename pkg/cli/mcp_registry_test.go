package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPRegistryClient_SearchServers(t *testing.T) {
	// Create a test server that mocks the MCP registry API
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers/search" {
			t.Errorf("Expected path /servers/search, got %s", r.URL.Path)
		}

		query := r.URL.Query().Get("q")
		if query != "notion" {
			t.Errorf("Expected query 'notion', got '%s'", query)
		}

		// Return mock response
		response := `{
			"servers": [
				{
					"id": "notion-mcp",
					"name": "Notion MCP Server",
					"description": "MCP server for Notion integration",
					"repository": "https://github.com/example/notion-mcp",
					"command": "npx",
					"args": ["notion-mcp"],
					"transport": "stdio",
					"config": {
						"env": {
							"NOTION_TOKEN": "${NOTION_TOKEN}"
						}
					}
				}
			],
			"total": 1
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
	if mcpServer.ID != "notion-mcp" {
		t.Errorf("Expected server ID 'notion-mcp', got '%s'", mcpServer.ID)
	}

	if mcpServer.Name != "Notion MCP Server" {
		t.Errorf("Expected server name 'Notion MCP Server', got '%s'", mcpServer.Name)
	}

	if mcpServer.Transport != "stdio" {
		t.Errorf("Expected transport 'stdio', got '%s'", mcpServer.Transport)
	}

	if mcpServer.Command != "npx" {
		t.Errorf("Expected command 'npx', got '%s'", mcpServer.Command)
	}

	if len(mcpServer.Args) != 1 || mcpServer.Args[0] != "notion-mcp" {
		t.Errorf("Expected args ['notion-mcp'], got %v", mcpServer.Args)
	}
}

func TestMCPRegistryClient_GetServer(t *testing.T) {
	// Create a test server that mocks the MCP registry API
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers/notion-mcp" {
			t.Errorf("Expected path /servers/notion-mcp, got %s", r.URL.Path)
		}

		// Return mock response
		response := `{
			"id": "notion-mcp",
			"name": "Notion MCP Server",
			"description": "MCP server for Notion integration",
			"repository": "https://github.com/example/notion-mcp",
			"command": "npx",
			"args": ["notion-mcp"],
			"transport": "stdio",
			"config": {
				"env": {
					"NOTION_TOKEN": "${NOTION_TOKEN}"
				}
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer testServer.Close()

	// Create client with test server URL
	client := NewMCPRegistryClient(testServer.URL)

	// Test get server
	serverInfo, err := client.GetServer("notion-mcp")
	if err != nil {
		t.Fatalf("GetServer failed: %v", err)
	}

	if serverInfo.ID != "notion-mcp" {
		t.Errorf("Expected server ID 'notion-mcp', got '%s'", serverInfo.ID)
	}

	if serverInfo.Name != "Notion MCP Server" {
		t.Errorf("Expected server name 'Notion MCP Server', got '%s'", serverInfo.Name)
	}
}

func TestMCPRegistryClient_GetServerNotFound(t *testing.T) {
	// Create a test server that returns 404
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	// Create client with test server URL
	client := NewMCPRegistryClient(testServer.URL)

	// Test get server that doesn't exist
	_, err := client.GetServer("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent server, got nil")
	}

	expectedError := "MCP server 'nonexistent' not found in registry"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestNewMCPRegistryClient_DefaultURL(t *testing.T) {
	client := NewMCPRegistryClient("")
	expectedURL := "https://api.mcp.github.com/v0"
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
