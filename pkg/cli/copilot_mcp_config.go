package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var copilotMCPConfigLog = logger.New("cli:copilot_mcp_config")

// CopilotMCPServerConfig represents a single MCP server configuration for GitHub Copilot
type CopilotMCPServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// CopilotMCPConfig represents the structure of copilot-mcp-config.json
type CopilotMCPConfig struct {
	MCPServers map[string]CopilotMCPServerConfig `json:"mcpServers"`
}

// ensureCopilotMCPConfig creates or updates .github/workflows/copilot-mcp-config.json
// with gh-aw MCP server configuration for GitHub Copilot Agent
func ensureCopilotMCPConfig(verbose bool) error {
	copilotMCPConfigLog.Print("Creating or updating copilot-mcp-config.json")

	// Create .github/workflows directory if it doesn't exist
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}
	copilotMCPConfigLog.Printf("Ensured directory exists: %s", workflowsDir)

	copilotMCPConfigPath := filepath.Join(workflowsDir, "copilot-mcp-config.json")

	// Read existing config if it exists
	var config CopilotMCPConfig
	if data, err := os.ReadFile(copilotMCPConfigPath); err == nil {
		copilotMCPConfigLog.Printf("Reading existing config from: %s", copilotMCPConfigPath)
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing copilot-mcp-config.json: %w", err)
		}
	} else {
		copilotMCPConfigLog.Print("No existing config found, creating new one")
		config.MCPServers = make(map[string]CopilotMCPServerConfig)
	}

	// Add or update gh-aw MCP server configuration
	ghAwServerName := "github-agentic-workflows"
	ghAwConfig := CopilotMCPServerConfig{
		Command: "gh",
		Args:    []string{"aw", "mcp-server"},
	}

	// Check if the server is already configured
	if existingConfig, exists := config.MCPServers[ghAwServerName]; exists {
		copilotMCPConfigLog.Printf("Server '%s' already exists in config", ghAwServerName)
		// Check if configuration is different
		existingJSON, _ := json.Marshal(existingConfig)
		newJSON, _ := json.Marshal(ghAwConfig)
		if string(existingJSON) == string(newJSON) {
			copilotMCPConfigLog.Print("Configuration is identical, skipping update")
			if verbose {
				fmt.Fprintf(os.Stderr, "MCP server '%s' already configured in %s\n", ghAwServerName, copilotMCPConfigPath)
			}
			return nil
		}
		copilotMCPConfigLog.Print("Configuration differs, updating")
	}

	config.MCPServers[ghAwServerName] = ghAwConfig

	// Write config file with proper indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal copilot-mcp-config.json: %w", err)
	}

	if err := os.WriteFile(copilotMCPConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write copilot-mcp-config.json: %w", err)
	}
	copilotMCPConfigLog.Printf("Wrote config to: %s", copilotMCPConfigPath)

	return nil
}
