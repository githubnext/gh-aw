package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMCPGatewayCommand(t *testing.T) {
	cmd := NewMCPGatewayCommand()

	if cmd.Use != "mcp-gateway" {
		t.Errorf("Expected Use to be 'mcp-gateway', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected Short description to be non-empty")
	}

	if cmd.Long == "" {
		t.Error("Expected Long description to be non-empty")
	}

	// Check flags
	portFlag := cmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Error("Expected --port flag to be defined")
	}

	apiKeyFlag := cmd.Flags().Lookup("api-key")
	if apiKeyFlag == nil {
		t.Error("Expected --api-key flag to be defined")
	}

	logsDirFlag := cmd.Flags().Lookup("logs-dir")
	if logsDirFlag == nil {
		t.Error("Expected --logs-dir flag to be defined")
	}

	mcpsFlag := cmd.Flags().Lookup("mcps")
	if mcpsFlag == nil {
		t.Error("Expected --mcps flag to be defined")
	}

	scriptsFlag := cmd.Flags().Lookup("scripts")
	if scriptsFlag == nil {
		t.Error("Expected --scripts flag to be defined")
	}
}

func TestMCPGatewayInvalidConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(configPath, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Try to run with invalid config (as mcps config)
	err := runMCPGateway(configPath, "", 0, "", "")
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestMCPGatewayMissingFile(t *testing.T) {
	err := runMCPGateway("/nonexistent/config.json", "", 0, "", "")
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

func TestMCPGatewayMissingPort(t *testing.T) {
	// This test is no longer valid since we have a default port
	// Leaving it here to test that default port (8088) is used
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "no-port.json")

	// Write config without port
	configJSON := `{
		"mcpServers": {
			"test": {
				"command": "echo",
				"args": ["hello"]
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Should now use default port 8088 instead of failing
	// (will fail on connection, not config)
	t.Skip("Skipping - requires actual MCP server connection")
}

func TestMCPGatewayValidConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "valid.json")

	// Write valid config
	configJSON := `{
		"mcpServers": {
			"test": {
				"command": "echo",
				"args": ["hello"]
			}
		},
		"port": 8088
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// This test just validates that the configuration is loaded correctly
	// We can't actually start the gateway in a unit test without MCP servers
	// So we'll just test that it doesn't fail during config validation

	// Note: We would need to start the gateway in a goroutine and then stop it
	// but that requires actual MCP servers to be available
	t.Skip("Skipping actual gateway startup - requires MCP servers")
}

func TestMCPGatewayWithAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"mcpServers": {
			"test": {
				"command": "echo"
			}
		},
		"port": 8088
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test that API key is accepted (will fail on connection, not auth setup)
	t.Skip("Skipping - requires actual MCP server connection")
}

func TestMCPGatewayWithLogsDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"mcpServers": {
			"test": {
				"command": "echo"
			}
		},
		"port": 8088
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test that logs directory is created
	// This will fail at connection, but should create the logs directory
	t.Skip("Skipping - requires actual MCP server connection")
}
