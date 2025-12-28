//go:build integration

package awmg

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestMCPGateway_BasicStartup(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Create temporary config
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "gateway-config.json")

	config := MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"gh-aw": {
				Command: binaryPath,
				Args:    []string{"mcp-server"},
			},
		},
		Gateway: GatewaySettings{
			Port: 8088,
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Start gateway in background
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use the runMCPGateway function directly in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- runMCPGateway([]string{configFile}, 8088, tmpDir)
	}()

	// Wait for server to start
	select {
	case <-ctx.Done():
		t.Fatal("Context canceled before server could start")
	case <-time.After(2 * time.Second):
		// Server should be ready
	}

	// Test health endpoint
	resp, err := http.Get("http://localhost:8088/health")
	if err != nil {
		cancel()
		t.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test servers list endpoint
	resp, err = http.Get("http://localhost:8088/servers")
	if err != nil {
		cancel()
		t.Fatalf("Failed to get servers list: %v", err)
	}
	defer resp.Body.Close()

	var serversResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&serversResp); err != nil {
		t.Fatalf("Failed to decode servers response: %v", err)
	}

	servers, ok := serversResp["servers"].([]any)
	if !ok {
		t.Fatal("Expected servers array in response")
	}

	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}

	// Check if gh-aw server is present
	foundGhAw := false
	for _, server := range servers {
		if serverName, ok := server.(string); ok && serverName == "gh-aw" {
			foundGhAw = true
			break
		}
	}

	if !foundGhAw {
		t.Error("Expected gh-aw server in servers list")
	}

	// Cancel context to stop the server
	cancel()

	// Wait for server to stop or timeout
	select {
	case err := <-errChan:
		// Server stopped, check if it was a clean shutdown
		if err != nil && err != http.ErrServerClosed && err.Error() != "context canceled" {
			t.Logf("Server stopped with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Log("Server shutdown timed out")
	}
}

func TestMCPGateway_ConfigFromStdin(t *testing.T) {
	// This test would require piping config to stdin
	// which is more complex in Go tests, so we'll skip for now
	t.Skip("Stdin config test requires more complex setup")
}
