package awmg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/types"

	"github.com/githubnext/gh-aw/pkg/parser"
)

// TestRewriteMCPConfigForGateway_ProxiesSafeInputsAndSafeOutputs tests that
// safeinputs and safeoutputs servers ARE proxied through the gateway (rewritten)
func TestRewriteMCPConfigForGateway_ProxiesSafeInputsAndSafeOutputs(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.json")

	// Initial config with both proxied and non-proxied servers
	initialConfig := map[string]any{
		"mcpServers": map[string]any{
			"safeinputs": map[string]any{
				"command": "gh",
				"args":    []string{"aw", "mcp-server", "--mode", "safe-inputs"},
			},
			"safeoutputs": map[string]any{
				"command": "gh",
				"args":    []string{"aw", "mcp-server", "--mode", "safe-outputs"},
			},
			"github": map[string]any{
				"command": "docker",
				"args":    []string{"run", "-i", "--rm", "ghcr.io/github-mcp-server"},
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

	// Gateway config includes ALL servers (including safeinputs/safeoutputs)
	gatewayConfig := &MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"safeinputs": {
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Command: "gh",
					Args:    []string{"aw", "mcp-server", "--mode", "safe-inputs"},
				},
			},
			"safeoutputs": {
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Command: "gh",
					Args:    []string{"aw", "mcp-server", "--mode", "safe-outputs"},
				},
			},
			"github": {
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Command: "docker",
					Args:    []string{"run", "-i", "--rm", "ghcr.io/github-mcp-server"},
				},
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

	// Should have all 3 servers, all rewritten
	if len(mcpServers) != 3 {
		t.Errorf("Expected 3 servers in rewritten config, got %d", len(mcpServers))
	}

	// Verify safeinputs points to gateway (rewritten)
	safeinputs, ok := mcpServers["safeinputs"].(map[string]any)
	if !ok {
		t.Fatal("safeinputs server not found")
	}

	safeinputsURL, ok := safeinputs["url"].(string)
	if !ok {
		t.Fatal("safeinputs server should have url (rewritten)")
	}

	expectedURL := "http://localhost:8080/mcp/safeinputs"
	if safeinputsURL != expectedURL {
		t.Errorf("Expected safeinputs URL %s, got %s", expectedURL, safeinputsURL)
	}

	safeinputsType, ok := safeinputs["type"].(string)
	if !ok || safeinputsType != "http" {
		t.Errorf("Expected safeinputs to have type 'http', got %v", safeinputsType)
	}

	// Verify safeinputs does NOT have command/args (was rewritten)
	if _, hasCommand := safeinputs["command"]; hasCommand {
		t.Error("Rewritten safeinputs server should not have 'command' field")
	}

	// Verify safeoutputs points to gateway (rewritten)
	safeoutputs, ok := mcpServers["safeoutputs"].(map[string]any)
	if !ok {
		t.Fatal("safeoutputs server not found")
	}

	safeoutputsURL, ok := safeoutputs["url"].(string)
	if !ok {
		t.Fatal("safeoutputs server should have url (rewritten)")
	}

	expectedURL = "http://localhost:8080/mcp/safeoutputs"
	if safeoutputsURL != expectedURL {
		t.Errorf("Expected safeoutputs URL %s, got %s", expectedURL, safeoutputsURL)
	}

	safeoutputsType, ok := safeoutputs["type"].(string)
	if !ok || safeoutputsType != "http" {
		t.Errorf("Expected safeoutputs to have type 'http', got %v", safeoutputsType)
	}

	// Verify safeoutputs does NOT have command/args (was rewritten)
	if _, hasCommand := safeoutputs["command"]; hasCommand {
		t.Error("Rewritten safeoutputs server should not have 'command' field")
	}

	// Verify github server points to gateway (was rewritten)
	github, ok := mcpServers["github"].(map[string]any)
	if !ok {
		t.Fatal("github server not found")
	}

	githubURL, ok := github["url"].(string)
	if !ok {
		t.Fatal("github server should have url (rewritten)")
	}

	expectedURL = "http://localhost:8080/mcp/github"
	if githubURL != expectedURL {
		t.Errorf("Expected github URL %s, got %s", expectedURL, githubURL)
	}

	// Verify github server has type: http
	githubType, ok := github["type"].(string)
	if !ok || githubType != "http" {
		t.Errorf("Expected github server to have type 'http', got %v", githubType)
	}

	// Verify github server has tools: ["*"]
	githubTools, ok := github["tools"].([]any)
	if !ok {
		t.Fatal("github server should have tools array")
	}
	if len(githubTools) != 1 || githubTools[0].(string) != "*" {
		t.Errorf("Expected github server to have tools ['*'], got %v", githubTools)
	}

	// Verify github server does NOT have command/args (was rewritten)
	if _, hasCommand := github["command"]; hasCommand {
		t.Error("Rewritten github server should not have 'command' field")
	}

	// Verify gateway settings are NOT included in rewritten config
	_, hasGateway := rewrittenConfig["gateway"]
	if hasGateway {
		t.Error("Gateway section should not be included in rewritten config")
	}
}

// TestRewriteMCPConfigForGateway_NoGatewaySection tests that gateway section is removed
func TestRewriteMCPConfigForGateway_NoGatewaySection(t *testing.T) {
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
		"gateway": map[string]any{
			"port":   8080,
			"apiKey": "test-key",
		},
	}

	initialJSON, _ := json.Marshal(initialConfig)
	if err := os.WriteFile(configFile, initialJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	gatewayConfig := &MCPGatewayServiceConfig{
		MCPServers: map[string]parser.MCPServerConfig{
			"github": {
				BaseMCPServerConfig: types.BaseMCPServerConfig{
					Command: "gh",
					Args:    []string{"aw", "mcp-server"},
				},
			},
		},
		Gateway: GatewaySettings{
			Port:   8080,
			APIKey: "test-key",
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

	// Verify gateway settings are NOT included in rewritten config
	_, hasGateway := rewrittenConfig["gateway"]
	if hasGateway {
		t.Error("Gateway section should not be included in rewritten config")
	}

	// Verify mcpServers still exists
	_, hasMCPServers := rewrittenConfig["mcpServers"]
	if !hasMCPServers {
		t.Error("mcpServers section should be present in rewritten config")
	}

	// Verify the rewritten server has type and tools
	mcpServers, ok := rewrittenConfig["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers not found or wrong type")
	}

	github, ok := mcpServers["github"].(map[string]any)
	if !ok {
		t.Fatal("github server not found")
	}

	// Check type field
	githubType, ok := github["type"].(string)
	if !ok || githubType != "http" {
		t.Errorf("Expected github server to have type 'http', got %v", githubType)
	}

	// Check tools field
	githubTools, ok := github["tools"].([]any)
	if !ok {
		t.Fatal("github server should have tools array")
	}
	if len(githubTools) != 1 || githubTools[0].(string) != "*" {
		t.Errorf("Expected github server to have tools ['*'], got %v", githubTools)
	}

	// Check headers field (API key was configured)
	githubHeaders, ok := github["headers"].(map[string]any)
	if !ok {
		t.Fatal("github server should have headers (API key configured)")
	}

	authHeader, ok := githubHeaders["Authorization"].(string)
	if !ok || authHeader != "Bearer test-key" {
		t.Errorf("Expected Authorization header 'Bearer test-key', got %v", authHeader)
	}
}
