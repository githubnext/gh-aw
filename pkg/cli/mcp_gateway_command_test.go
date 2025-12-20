package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadGatewayConfig_FromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "gateway-config.json")

	config := MCPGatewayConfig{
		MCPServers: map[string]MCPServerConfig{
			"test-server": {
				Command: "test-command",
				Args:    []string{"arg1", "arg2"},
				Env: map[string]string{
					"KEY": "value",
				},
			},
		},
		Gateway: GatewaySettings{
			Port: 8080,
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Read config
	result, err := readGatewayConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Verify config
	if len(result.MCPServers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(result.MCPServers))
	}

	testServer, exists := result.MCPServers["test-server"]
	if !exists {
		t.Fatal("test-server not found in config")
	}

	if testServer.Command != "test-command" {
		t.Errorf("Expected command 'test-command', got '%s'", testServer.Command)
	}

	if len(testServer.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(testServer.Args))
	}

	if result.Gateway.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", result.Gateway.Port)
	}
}

func TestReadGatewayConfig_InvalidJSON(t *testing.T) {
	// Create a temporary config file with invalid JSON
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid-config.json")

	if err := os.WriteFile(configFile, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Read config - should fail
	_, err := readGatewayConfig(configFile)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestMCPGatewayConfig_EmptyServers(t *testing.T) {
	config := &MCPGatewayConfig{
		MCPServers: make(map[string]MCPServerConfig),
		Gateway: GatewaySettings{
			Port: 8080,
		},
	}

	if len(config.MCPServers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(config.MCPServers))
	}
}

func TestMCPServerConfig_CommandType(t *testing.T) {
	config := MCPServerConfig{
		Command: "gh",
		Args:    []string{"aw", "mcp-server"},
		Env: map[string]string{
			"DEBUG": "cli:*",
		},
	}

	if config.Command != "gh" {
		t.Errorf("Expected command 'gh', got '%s'", config.Command)
	}

	if config.URL != "" {
		t.Error("Expected empty URL for command-based server")
	}

	if config.Container != "" {
		t.Error("Expected empty container for command-based server")
	}
}

func TestMCPServerConfig_URLType(t *testing.T) {
	config := MCPServerConfig{
		URL: "http://localhost:3000",
	}

	if config.URL != "http://localhost:3000" {
		t.Errorf("Expected URL 'http://localhost:3000', got '%s'", config.URL)
	}

	if config.Command != "" {
		t.Error("Expected empty command for URL-based server")
	}
}

func TestMCPServerConfig_ContainerType(t *testing.T) {
	config := MCPServerConfig{
		Container: "mcp-server:latest",
		Args:      []string{"--verbose"},
		Env: map[string]string{
			"LOG_LEVEL": "debug",
		},
	}

	if config.Container != "mcp-server:latest" {
		t.Errorf("Expected container 'mcp-server:latest', got '%s'", config.Container)
	}

	if config.Command != "" {
		t.Error("Expected empty command for container-based server")
	}

	if config.URL != "" {
		t.Error("Expected empty URL for container-based server")
	}
}

func TestGatewaySettings_DefaultPort(t *testing.T) {
	settings := GatewaySettings{}

	if settings.Port != 0 {
		t.Errorf("Expected default port 0, got %d", settings.Port)
	}
}

func TestGatewaySettings_WithAPIKey(t *testing.T) {
	settings := GatewaySettings{
		Port:   8080,
		APIKey: "test-api-key",
	}

	if settings.APIKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", settings.APIKey)
	}
}

func TestReadGatewayConfig_FileNotFound(t *testing.T) {
	// Try to read a non-existent file
	_, err := readGatewayConfig("/tmp/nonexistent-gateway-config-12345.json")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
	if err != nil && err.Error() != "configuration file not found: /tmp/nonexistent-gateway-config-12345.json" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestReadGatewayConfig_EmptyServers(t *testing.T) {
	// Create a config file with no servers
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "empty-servers.json")

	config := MCPGatewayConfig{
		MCPServers: map[string]MCPServerConfig{},
		Gateway: GatewaySettings{
			Port: 8080,
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Try to read config - should fail with no servers
	_, err = readGatewayConfig(configFile)
	if err == nil {
		t.Error("Expected error for config with no servers, got nil")
	}
	if err != nil && err.Error() != "no MCP servers configured in configuration" {
		t.Errorf("Expected 'no MCP servers configured' error, got: %v", err)
	}
}

func TestReadGatewayConfig_EmptyData(t *testing.T) {
	// Create an empty config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "empty.json")

	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty config file: %v", err)
	}

	// Try to read config - should fail with empty data
	_, err := readGatewayConfig(configFile)
	if err == nil {
		t.Error("Expected error for empty config file, got nil")
	}
	if err != nil && err.Error() != "configuration data is empty" {
		t.Errorf("Expected 'configuration data is empty' error, got: %v", err)
	}
}
