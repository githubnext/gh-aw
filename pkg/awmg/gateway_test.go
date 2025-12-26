package awmg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestReadGatewayConfig_FromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "gateway-config.json")

	config := MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
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
	result, _, err := readGatewayConfig([]string{configFile})
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
	_, _, err := readGatewayConfig([]string{configFile})
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestMCPGatewayConfig_EmptyServers(t *testing.T) {
	config := &MCPGatewayServiceConfig{
		MCPServers: make(map[string]parser.MCPServerConfig),
		Gateway: GatewaySettings{
			Port: 8080,
		},
	}

	if len(config.MCPServers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(config.MCPServers))
	}
}

func TestMCPServerConfig_CommandType(t *testing.T) {
	config := parser.MCPServerConfig{
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
	config := parser.MCPServerConfig{
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
	config := parser.MCPServerConfig{
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
	_, _, err := readGatewayConfig([]string{"/tmp/nonexistent-gateway-config-12345.json"})
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

	config := MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{},
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
	_, _, err = readGatewayConfig([]string{configFile})
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
	_, _, err := readGatewayConfig([]string{configFile})
	if err == nil {
		t.Error("Expected error for empty config file, got nil")
	}
	if err != nil && err.Error() != "configuration data is empty" {
		t.Errorf("Expected 'configuration data is empty' error, got: %v", err)
	}
}

func TestReadGatewayConfig_MultipleFiles(t *testing.T) {
	// Create base config file
	tmpDir := t.TempDir()
	baseConfig := filepath.Join(tmpDir, "base-config.json")
	baseConfigData := MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"server1": {
				Command: "command1",
				Args:    []string{"arg1"},
			},
			"server2": {
				Command: "command2",
				Args:    []string{"arg2"},
			},
		},
		Gateway: GatewaySettings{
			Port: 8080,
		},
	}

	baseJSON, err := json.Marshal(baseConfigData)
	if err != nil {
		t.Fatalf("Failed to marshal base config: %v", err)
	}
	if err := os.WriteFile(baseConfig, baseJSON, 0644); err != nil {
		t.Fatalf("Failed to write base config: %v", err)
	}

	// Create override config file
	overrideConfig := filepath.Join(tmpDir, "override-config.json")
	overrideConfigData := MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"server2": {
				Command: "override-command2",
				Args:    []string{"override-arg2"},
			},
			"server3": {
				Command: "command3",
				Args:    []string{"arg3"},
			},
		},
		Gateway: GatewaySettings{
			Port:   9090,
			APIKey: "test-key",
		},
	}

	overrideJSON, err := json.Marshal(overrideConfigData)
	if err != nil {
		t.Fatalf("Failed to marshal override config: %v", err)
	}
	if err := os.WriteFile(overrideConfig, overrideJSON, 0644); err != nil {
		t.Fatalf("Failed to write override config: %v", err)
	}

	// Read and merge configs
	result, _, err := readGatewayConfig([]string{baseConfig, overrideConfig})
	if err != nil {
		t.Fatalf("Failed to read configs: %v", err)
	}

	// Verify merged config
	if len(result.MCPServers) != 3 {
		t.Errorf("Expected 3 servers, got %d", len(result.MCPServers))
	}

	// server1 should remain from base
	server1, exists := result.MCPServers["server1"]
	if !exists {
		t.Fatal("server1 not found in merged config")
	}
	if server1.Command != "command1" {
		t.Errorf("Expected server1 command 'command1', got '%s'", server1.Command)
	}

	// server2 should be overridden
	server2, exists := result.MCPServers["server2"]
	if !exists {
		t.Fatal("server2 not found in merged config")
	}
	if server2.Command != "override-command2" {
		t.Errorf("Expected server2 command 'override-command2', got '%s'", server2.Command)
	}

	// server3 should be added from override
	server3, exists := result.MCPServers["server3"]
	if !exists {
		t.Fatal("server3 not found in merged config")
	}
	if server3.Command != "command3" {
		t.Errorf("Expected server3 command 'command3', got '%s'", server3.Command)
	}

	// Gateway settings should be overridden
	if result.Gateway.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", result.Gateway.Port)
	}
	if result.Gateway.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", result.Gateway.APIKey)
	}
}

func TestMergeConfigs(t *testing.T) {
	base := &MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"server1": {
				Command: "cmd1",
			},
			"server2": {
				Command: "cmd2",
			},
		},
		Gateway: GatewaySettings{
			Port:   8080,
			APIKey: "base-key",
		},
	}

	override := &MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"server2": {
				Command: "override-cmd2",
			},
			"server3": {
				Command: "cmd3",
			},
		},
		Gateway: GatewaySettings{
			Port: 9090,
			// APIKey not set, should keep base
		},
	}

	merged := mergeConfigs(base, override)

	// Check servers
	if len(merged.MCPServers) != 3 {
		t.Errorf("Expected 3 servers, got %d", len(merged.MCPServers))
	}

	if merged.MCPServers["server1"].Command != "cmd1" {
		t.Error("server1 should remain from base")
	}

	if merged.MCPServers["server2"].Command != "override-cmd2" {
		t.Error("server2 should be overridden")
	}

	if merged.MCPServers["server3"].Command != "cmd3" {
		t.Error("server3 should be added from override")
	}

	// Check gateway settings
	if merged.Gateway.Port != 9090 {
		t.Error("Port should be overridden")
	}

	if merged.Gateway.APIKey != "base-key" {
		t.Error("APIKey should be kept from base when not set in override")
	}
}

func TestMergeConfigs_EmptyOverride(t *testing.T) {
	base := &MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"server1": {
				Command: "cmd1",
			},
		},
		Gateway: GatewaySettings{
			Port: 8080,
		},
	}

	override := &MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{},
		Gateway:    GatewaySettings{},
	}

	merged := mergeConfigs(base, override)

	// Should keep base config
	if len(merged.MCPServers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(merged.MCPServers))
	}

	if merged.Gateway.Port != 8080 {
		t.Error("Port should be kept from base")
	}
}

func TestParseGatewayConfig_FiltersInternalServers(t *testing.T) {
	// Create a config with safeinputs, safeoutputs, and other servers
	configJSON := `{
		"mcpServers": {
			"safeinputs": {
				"command": "node",
				"args": ["/tmp/gh-aw/safeinputs/mcp-server.cjs"]
			},
			"safeoutputs": {
				"command": "node",
				"args": ["/tmp/gh-aw/safeoutputs/mcp-server.cjs"]
			},
			"github": {
				"command": "gh",
				"args": ["aw", "mcp-server", "--toolsets", "default"]
			},
			"custom-server": {
				"command": "custom-command",
				"args": ["arg1"]
			}
		},
		"gateway": {
			"port": 8080
		}
	}`

	config, err := parseGatewayConfig([]byte(configJSON))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify that safeinputs and safeoutputs are filtered out
	if _, exists := config.MCPServers["safeinputs"]; exists {
		t.Error("safeinputs should be filtered out")
	}

	if _, exists := config.MCPServers["safeoutputs"]; exists {
		t.Error("safeoutputs should be filtered out")
	}

	// Verify that other servers are kept
	if _, exists := config.MCPServers["github"]; !exists {
		t.Error("github server should be kept")
	}

	if _, exists := config.MCPServers["custom-server"]; !exists {
		t.Error("custom-server should be kept")
	}

	// Verify server count
	if len(config.MCPServers) != 2 {
		t.Errorf("Expected 2 servers after filtering, got %d", len(config.MCPServers))
	}
}

func TestParseGatewayConfig_OnlyInternalServers(t *testing.T) {
	// Create a config with only safeinputs and safeoutputs
	configJSON := `{
		"mcpServers": {
			"safeinputs": {
				"command": "node",
				"args": ["/tmp/gh-aw/safeinputs/mcp-server.cjs"]
			},
			"safeoutputs": {
				"command": "node",
				"args": ["/tmp/gh-aw/safeoutputs/mcp-server.cjs"]
			}
		}
	}`

	config, err := parseGatewayConfig([]byte(configJSON))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify that all internal servers are filtered out, resulting in 0 servers
	if len(config.MCPServers) != 0 {
		t.Errorf("Expected 0 servers after filtering internal servers, got %d", len(config.MCPServers))
	}
}

func TestRewriteMCPConfigForGateway(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.json")

	// Initial config with multiple servers
	initialConfig := map[string]any{
		"mcpServers": map[string]any{
			"github": map[string]any{
				"command": "gh",
				"args":    []string{"aw", "mcp-server"},
			},
			"custom": map[string]any{
				"command": "node",
				"args":    []string{"server.js"},
			},
		},
		"gateway": map[string]any{
			"port": 8080,
		},
	}

	initialJSON, _ := json.Marshal(initialConfig)
	if err := os.WriteFile(configFile, initialJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create a gateway config (after filtering)
	gatewayConfig := &MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"github": {
				Command: "gh",
				Args:    []string{"aw", "mcp-server"},
			},
			"custom": {
				Command: "node",
				Args:    []string{"server.js"},
			},
		},
		Gateway: GatewaySettings{
			Port: 8080,
		},
	}

	// Rewrite the config
	if err := rewriteMCPConfigForGateway(configFile, gatewayConfig); err != nil {
		t.Fatalf("rewriteMCPConfigForGateway failed: %v", err)
	}

	// Read back the rewritten config
	rewrittenData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read rewritten config: %v", err)
	}

	var rewrittenConfig map[string]any
	if err := json.Unmarshal(rewrittenData, &rewrittenConfig); err != nil {
		t.Fatalf("Failed to parse rewritten config: %v", err)
	}

	// Verify structure
	mcpServers, ok := rewrittenConfig["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers not found or wrong type")
	}

	if len(mcpServers) != 2 {
		t.Errorf("Expected 2 servers in rewritten config, got %d", len(mcpServers))
	}

	// Verify github server points to gateway
	github, ok := mcpServers["github"].(map[string]any)
	if !ok {
		t.Fatal("github server not found")
	}

	githubURL, ok := github["url"].(string)
	if !ok {
		t.Fatal("github server missing url")
	}

	expectedURL := "http://host.docker.internal:8080/mcp/github"
	if githubURL != expectedURL {
		t.Errorf("Expected github URL %s, got %s", expectedURL, githubURL)
	}

	// Verify custom server points to gateway
	custom, ok := mcpServers["custom"].(map[string]any)
	if !ok {
		t.Fatal("custom server not found")
	}

	customURL, ok := custom["url"].(string)
	if !ok {
		t.Fatal("custom server missing url")
	}

	expectedCustomURL := "http://host.docker.internal:8080/mcp/custom"
	if customURL != expectedCustomURL {
		t.Errorf("Expected custom URL %s, got %s", expectedCustomURL, customURL)
	}

	// Verify gateway settings are NOT included in rewritten config
	_, hasGateway := rewrittenConfig["gateway"]
	if hasGateway {
		t.Error("Gateway section should not be included in rewritten config")
	}
}

func TestRewriteMCPConfigForGateway_WithAPIKey(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.json")

	initialConfig := map[string]any{
		"mcpServers": map[string]any{
			"github": map[string]any{
				"command": "gh",
				"args":    []string{"aw", "mcp-server"},
			},
		},
	}

	initialJSON, _ := json.Marshal(initialConfig)
	if err := os.WriteFile(configFile, initialJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create a gateway config with API key
	gatewayConfig := &MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"github": {
				Command: "gh",
				Args:    []string{"aw", "mcp-server"},
			},
		},
		Gateway: GatewaySettings{
			Port:   8080,
			APIKey: "test-api-key",
		},
	}

	// Rewrite the config
	if err := rewriteMCPConfigForGateway(configFile, gatewayConfig); err != nil {
		t.Fatalf("rewriteMCPConfigForGateway failed: %v", err)
	}

	// Read back the rewritten config
	rewrittenData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read rewritten config: %v", err)
	}

	var rewrittenConfig map[string]any
	if err := json.Unmarshal(rewrittenData, &rewrittenConfig); err != nil {
		t.Fatalf("Failed to parse rewritten config: %v", err)
	}

	// Verify server has authorization header
	mcpServers := rewrittenConfig["mcpServers"].(map[string]any)
	github := mcpServers["github"].(map[string]any)

	headers, ok := github["headers"].(map[string]any)
	if !ok {
		t.Fatal("Expected headers in server config")
	}

	auth, ok := headers["Authorization"].(string)
	if !ok {
		t.Fatal("Expected Authorization header")
	}

	expectedAuth := "Bearer test-api-key"
	if auth != expectedAuth {
		t.Errorf("Expected auth '%s', got '%s'", expectedAuth, auth)
	}
}
