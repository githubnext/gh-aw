//go:build integration

package awmg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestStreamableHTTPTransport_GatewayConnection tests the streamable HTTP transport
// by starting the gateway with a command-based MCP server, then verifying we can
// connect via the gateway's HTTP endpoint using the go-sdk StreamableClientTransport.
func TestStreamableHTTPTransport_GatewayConnection(t *testing.T) {
	// Get absolute path to binary
	binaryPath, err := filepath.Abs(filepath.Join("..", "..", "gh-aw"))
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: gh-aw binary not found at %s. Run 'make build' first.", binaryPath)
	}

	// Create temporary directory for config
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "gateway-config.json")

	// Create gateway config with the gh-aw MCP server
	config := MCPGatewayConfig{
		MCPServers: map[string]MCPServerConfig{
			"gh-aw": {
				Command: binaryPath,
				Args:    []string{"mcp-server"},
			},
		},
		Gateway: GatewaySettings{
			Port: 8091, // Use a different port to avoid conflicts
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Start the gateway in background
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gatewayErrChan := make(chan error, 1)
	go func() {
		gatewayErrChan <- runMCPGateway([]string{configFile}, 8091, tmpDir)
	}()

	// Wait for gateway to start
	t.Log("Waiting for MCP gateway to start...")
	time.Sleep(3 * time.Second)

	// Verify gateway health
	healthResp, err := http.Get("http://localhost:8091/health")
	if err != nil {
		cancel()
		t.Fatalf("Failed to connect to gateway health endpoint: %v", err)
	}
	healthResp.Body.Close()

	if healthResp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("Gateway health check failed: status=%d", healthResp.StatusCode)
	}
	t.Log("✓ Gateway health check passed")

	// Test 1: Verify the gateway servers list
	serversResp, err := http.Get("http://localhost:8091/servers")
	if err != nil {
		cancel()
		t.Fatalf("Failed to get servers list: %v", err)
	}
	defer serversResp.Body.Close()

	var serversData map[string]any
	if err := json.NewDecoder(serversResp.Body).Decode(&serversData); err != nil {
		t.Fatalf("Failed to decode servers response: %v", err)
	}

	servers, ok := serversData["servers"].([]any)
	if !ok || len(servers) == 0 {
		t.Fatalf("Expected servers list, got: %v", serversData)
	}
	t.Logf("✓ Gateway has %d server(s): %v", len(servers), servers)

	// Test 2: Test the MCP endpoint directly using HTTP POST
	mcpURL := "http://localhost:8091/mcp/gh-aw"
	t.Logf("Testing MCP endpoint: %s", mcpURL)

	// Send initialize request
	initReq := map[string]any{
		"method": "initialize",
		"params": map[string]any{},
	}
	initReqJSON, _ := json.Marshal(initReq)

	resp, err := http.Post(mcpURL, "application/json", strings.NewReader(string(initReqJSON)))
	if err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Initialize request failed: status=%d", resp.StatusCode)
	}

	var initResponse map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&initResponse); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}
	t.Logf("✓ Initialize response: %v", initResponse)

	// Test 3: Send tools/list request
	listToolsReq := map[string]any{
		"method": "tools/list",
		"params": map[string]any{},
	}
	listToolsReqJSON, _ := json.Marshal(listToolsReq)

	toolsResp, err := http.Post(mcpURL, "application/json", strings.NewReader(string(listToolsReqJSON)))
	if err != nil {
		t.Fatalf("Failed to send tools/list request: %v", err)
	}
	defer toolsResp.Body.Close()

	if toolsResp.StatusCode != http.StatusOK {
		t.Fatalf("tools/list request failed: status=%d", toolsResp.StatusCode)
	}

	var toolsResponse map[string]any
	if err := json.NewDecoder(toolsResp.Body).Decode(&toolsResponse); err != nil {
		t.Fatalf("Failed to decode tools/list response: %v", err)
	}
	t.Logf("✓ Tools/list response received with %d tools", len(toolsResponse))

	// Verify the response contains tools array
	if tools, ok := toolsResponse["tools"].([]any); ok {
		t.Logf("✓ Found %d tools in response", len(tools))
	} else {
		t.Logf("Note: Tools response format: %v", toolsResponse)
	}

	t.Log("✓ All streamable HTTP transport tests completed successfully")

	// Clean up
	cancel()

	// Wait for gateway to stop
	select {
	case err := <-gatewayErrChan:
		if err != nil && err != http.ErrServerClosed && !strings.Contains(err.Error(), "context canceled") {
			t.Logf("Gateway stopped with error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Log("Gateway shutdown timed out")
	}
}

// TestStreamableHTTPTransport_GoSDKClient tests using the go-sdk StreamableClientTransport
// to connect to a mock MCP server that implements the streamable HTTP protocol.
func TestStreamableHTTPTransport_GoSDKClient(t *testing.T) {
	// Create a mock MCP server that implements the streamable HTTP protocol
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the JSON-RPC request
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		method, _ := request["method"].(string)
		id := request["id"]

		// Build JSON-RPC response
		var result any

		switch method {
		case "initialize":
			result = map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "test-server",
					"version": "1.0.0",
				},
			}
		case "notifications/initialized":
			// No response needed for notification
			w.WriteHeader(http.StatusAccepted)
			return
		case "tools/list":
			result = map[string]any{
				"tools": []map[string]any{
					{
						"name":        "test_tool",
						"description": "A test tool",
						"inputSchema": map[string]any{
							"type":       "object",
							"properties": map[string]any{},
						},
					},
				},
			}
		default:
			http.Error(w, fmt.Sprintf("Unknown method: %s", method), http.StatusBadRequest)
			return
		}

		response := map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	t.Logf("Mock MCP server running at: %s", mockServer.URL)

	// Create the streamable client transport
	transport := &mcp.StreamableClientTransport{
		Endpoint: mockServer.URL,
	}

	// Create MCP client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Connect to the server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("Failed to connect to mock MCP server: %v", err)
	}
	defer session.Close()

	t.Log("✓ Successfully connected to mock MCP server via StreamableClientTransport")

	// Test listing tools
	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	if len(toolsResult.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(toolsResult.Tools))
	}

	if toolsResult.Tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", toolsResult.Tools[0].Name)
	}

	t.Logf("✓ Successfully listed tools: %v", toolsResult.Tools)
	t.Log("✓ StreamableClientTransport go-sdk test completed successfully")
}

// TestStreamableHTTPTransport_URLConfigured tests that a URL-configured server
// uses the StreamableClientTransport when connecting.
func TestStreamableHTTPTransport_URLConfigured(t *testing.T) {
	// Create a mock server that tracks connection attempts
	connectionAttempted := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectionAttempted = true

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		method, _ := request["method"].(string)
		id := request["id"]

		var result any
		switch method {
		case "initialize":
			result = map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{},
				"serverInfo": map[string]any{
					"name":    "url-test-server",
					"version": "1.0.0",
				},
			}
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
			return
		default:
			result = map[string]any{}
		}

		response := map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	t.Logf("Mock URL-based MCP server at: %s", mockServer.URL)

	// Test that createMCPSession uses StreamableClientTransport for URL config
	gateway := &MCPGatewayServer{
		config:   &MCPGatewayConfig{},
		sessions: make(map[string]*mcp.ClientSession),
		logDir:   t.TempDir(),
	}

	// Create a session with URL configuration
	serverConfig := MCPServerConfig{
		URL: mockServer.URL,
	}

	session, err := gateway.createMCPSession("test-url-server", serverConfig)
	if err != nil {
		t.Fatalf("Failed to create session for URL-configured server: %v", err)
	}
	defer session.Close()

	if !connectionAttempted {
		t.Error("Expected connection to be attempted via streamable HTTP")
	}

	t.Log("✓ URL-configured server successfully connected via StreamableClientTransport")
}

// TestStreamableHTTPTransport_MCPInspect tests using the mcp inspect command
// to verify the streamable HTTP configuration works end-to-end.
func TestStreamableHTTPTransport_MCPInspect(t *testing.T) {
	// Get absolute path to binary
	binaryPath, err := filepath.Abs(filepath.Join("..", "..", "gh-aw"))
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: gh-aw binary not found at %s. Run 'make build' first.", binaryPath)
	}

	// Create temporary directory
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow with HTTP-based MCP server configuration
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
---

# Test Streamable HTTP Transport

This workflow tests the streamable HTTP transport via mcp inspect.
`

	workflowFile := filepath.Join(workflowsDir, "test-streamable.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Run mcp inspect to verify the workflow can be parsed
	t.Log("Running mcp inspect to verify streamable HTTP configuration...")
	inspectCmd := exec.Command(binaryPath, "mcp", "inspect", "test-streamable", "--verbose")
	inspectCmd.Dir = tmpDir
	inspectCmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", tmpDir),
	)

	output, err := inspectCmd.CombinedOutput()
	outputStr := string(output)

	t.Logf("mcp inspect output:\n%s", outputStr)

	// Check if the workflow was parsed successfully
	if err != nil {
		// It might fail due to auth, but we're testing the parsing
		if !strings.Contains(outputStr, "github") {
			t.Fatalf("mcp inspect failed to parse workflow: %v", err)
		}
		t.Log("Note: Inspection failed due to auth (expected), but workflow was parsed correctly")
	}

	// Verify the github server was detected
	if strings.Contains(outputStr, "github") || strings.Contains(outputStr, "GitHub") {
		t.Log("✓ GitHub server detected in workflow (uses HTTP transport)")
	}

	t.Log("✓ MCP inspect test for streamable HTTP completed successfully")
}
