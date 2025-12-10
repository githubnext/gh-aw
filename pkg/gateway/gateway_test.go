package gateway

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestNewGateway(t *testing.T) {
	tests := []struct {
		name    string
		config  GatewayConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: GatewayConfig{
				Port: 8088,
				MCPServers: map[string]parser.MCPServerConfig{
					"test": {
						Command: "echo",
						Args:    []string{"hello"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing port",
			config: GatewayConfig{
				Port: 0,
				MCPServers: map[string]parser.MCPServerConfig{
					"test": {
						Command: "echo",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no servers",
			config: GatewayConfig{
				Port:       8088,
				MCPServers: map[string]parser.MCPServerConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGateway(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGateway() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigFromJSON(t *testing.T) {
	// Create a temporary config file
	configJSON := `{
		"mcpServers": {
			"server1": {
				"command": "node",
				"args": ["server.js"]
			},
			"server2": {
				"url": "http://localhost:3000"
			}
		},
		"port": 8088
	}`

	// Test loading from reader
	reader := strings.NewReader(configJSON)
	config, err := LoadConfigFromReader(reader)
	if err != nil {
		t.Fatalf("LoadConfigFromReader() error = %v", err)
	}

	if config.Port != 8088 {
		t.Errorf("Expected port 8088, got %d", config.Port)
	}

	if len(config.MCPServers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(config.MCPServers))
	}

	// Check server1
	if server1, ok := config.MCPServers["server1"]; ok {
		if server1.Command != "node" {
			t.Errorf("Expected server1 command 'node', got '%s'", server1.Command)
		}
		if len(server1.Args) != 1 || server1.Args[0] != "server.js" {
			t.Errorf("Expected server1 args ['server.js'], got %v", server1.Args)
		}
	} else {
		t.Error("server1 not found in config")
	}

	// Check server2
	if server2, ok := config.MCPServers["server2"]; ok {
		if server2.URL != "http://localhost:3000" {
			t.Errorf("Expected server2 URL 'http://localhost:3000', got '%s'", server2.URL)
		}
	} else {
		t.Error("server2 not found in config")
	}
}

func TestLoadConfigFromReader_InvalidJSON(t *testing.T) {
	reader := strings.NewReader(`{invalid json}`)
	_, err := LoadConfigFromReader(reader)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestCreateTransports(t *testing.T) {
	gw := &Gateway{
		config: GatewayConfig{
			Port:       8088,
			MCPServers: map[string]parser.MCPServerConfig{},
		},
	}

	t.Run("stdio transport", func(t *testing.T) {
		config := parser.MCPServerConfig{
			Command: "echo",
			Args:    []string{"hello"},
			Env: map[string]string{
				"TEST_VAR": "value",
			},
		}

		transport, err := gw.createStdioTransport(config)
		if err != nil {
			t.Fatalf("createStdioTransport() error = %v", err)
		}

		if transport == nil {
			t.Error("Expected non-nil transport")
		}
	})

	t.Run("http transport", func(t *testing.T) {
		config := parser.MCPServerConfig{
			URL: "http://localhost:3000",
		}

		transport, err := gw.createHTTPTransport(config)
		if err != nil {
			t.Fatalf("createHTTPTransport() error = %v", err)
		}

		if transport == nil {
			t.Error("Expected non-nil transport")
		}
	})

	t.Run("docker transport", func(t *testing.T) {
		config := parser.MCPServerConfig{
			Container: "my-server:latest",
			Args:      []string{"--port", "3000"},
			Env: map[string]string{
				"TEST_VAR": "value",
			},
		}

		transport, err := gw.createDockerTransport(config)
		if err != nil {
			t.Fatalf("createDockerTransport() error = %v", err)
		}

		if transport == nil {
			t.Error("Expected non-nil transport")
		}
	})
}

func TestGatewayLifecycle(t *testing.T) {
	// This is a basic lifecycle test that doesn't actually start servers
	config := GatewayConfig{
		Port: 8088,
		MCPServers: map[string]parser.MCPServerConfig{
			"test": {
				Command: "echo",
				Args:    []string{"test"},
			},
		},
	}

	gw, err := NewGateway(config)
	if err != nil {
		t.Fatalf("NewGateway() error = %v", err)
	}

	// Test close on unconnected gateway
	err = gw.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestGatewayContext tests that the gateway respects context cancellation
func TestGatewayContext(t *testing.T) {
	t.Skip("Skipping integration test - requires actual MCP servers")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := GatewayConfig{
		Port: 8081, // Use a different port for test
		MCPServers: map[string]parser.MCPServerConfig{
			"test": {
				Command: "sleep",
				Args:    []string{"60"}, // Long-running command
			},
		},
	}

	gw, err := NewGateway(config)
	if err != nil {
		t.Fatalf("NewGateway() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start gateway in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- gw.Start(ctx)
	}()

	// Wait for either timeout or completion
	select {
	case err := <-done:
		// Expected to get an error or context cancellation
		if err != nil && !strings.Contains(err.Error(), "context") {
			t.Logf("Gateway returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Gateway did not stop after context timeout")
	}

	// Clean up
	if err := gw.Close(); err != nil {
		t.Logf("Close() error = %v", err)
	}
}
