package awmg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var gatewayLog = logger.New("awmg:gateway")

// version is set by the main package.
var version = "dev"

// SetVersionInfo sets the version information for the awmg package.
func SetVersionInfo(v string) {
	version = v
}

// GetVersion returns the current version.
func GetVersion() string {
	return version
}

// MCPGatewayServiceConfig represents the configuration for the MCP gateway service.
type MCPGatewayServiceConfig struct {
	MCPServers map[string]parser.MCPServerConfig `json:"mcpServers"`
	Gateway    GatewaySettings                   `json:"gateway,omitempty"`
}

// GatewaySettings represents gateway-specific settings.
type GatewaySettings struct {
	Port   int    `json:"port,omitempty"`
	APIKey string `json:"apiKey,omitempty"`
}

// MCPGatewayServer manages multiple MCP sessions and exposes them via HTTP
type MCPGatewayServer struct {
	config   *MCPGatewayServiceConfig
	sessions map[string]*mcp.ClientSession
	servers  map[string]*mcp.Server // Proxy servers for each session
	mu       sync.RWMutex
	logDir   string
}

// NewMCPGatewayCommand creates the mcp-gateway command
func NewMCPGatewayCommand() *cobra.Command {
	var configFiles []string
	var port int
	var logDir string

	cmd := &cobra.Command{
		Use:   "mcp-gateway",
		Short: "Run an MCP gateway proxy that aggregates multiple MCP servers",
		Long: `Run an MCP gateway that acts as a proxy to multiple MCP servers.

The gateway:
- Integrates by default with the sandbox.mcp extension point
- Imports Claude/Copilot/Codex MCP server JSON configuration
- Starts each MCP server and mounts an MCP client on each
- Mounts an HTTP MCP server that acts as a gateway to the MCP clients
- Supports most MCP gestures through the go-MCP SDK
- Provides extensive logging to file in the MCP log folder

Configuration can be provided via:
1. --config flag(s) pointing to JSON config file(s) (can be specified multiple times)
2. stdin (reads JSON configuration from standard input)

Multiple config files are merged in order, with later files overriding earlier ones.

Configuration format:
{
  "mcpServers": {
    "server-name": {
      "command": "command",
      "args": ["arg1", "arg2"],
      "env": {"KEY": "value"}
    }
  },
  "gateway": {
    "port": 8080,
    "apiKey": "optional-key"
  }
}

Examples:
  awmg --config config.json                    # From single file
  awmg --config base.json --config override.json # From multiple files (merged)
  awmg --port 8080                             # From stdin
  echo '{"mcpServers":{...}}' | awmg           # Pipe config
  awmg --config config.json --log-dir /tmp/logs # Custom log dir`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPGateway(configFiles, port, logDir)
		},
	}

	cmd.Flags().StringArrayVarP(&configFiles, "config", "c", []string{}, "Path to MCP gateway configuration JSON file (can be specified multiple times)")
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to run HTTP gateway on")
	cmd.Flags().StringVar(&logDir, "log-dir", "/tmp/gh-aw/mcp-logs", "Directory for MCP gateway logs")

	return cmd
}

// runMCPGateway starts the MCP gateway server
func runMCPGateway(configFiles []string, port int, logDir string) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting MCP gateway (port: %d, logDir: %s, configFiles: %v)", port, logDir, configFiles)))
	gatewayLog.Printf("Starting MCP gateway on port %d", port)

	// Read configuration
	config, originalConfigPath, err := readGatewayConfig(configFiles)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to read configuration: %v", err)))
		return fmt.Errorf("failed to read gateway configuration: %w", err)
	}

	// Override port if specified in command line
	if port > 0 {
		config.Gateway.Port = port
	} else if config.Gateway.Port == 0 {
		config.Gateway.Port = 8080 // Default port
	}

	// Create log directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create gateway server
	gateway := &MCPGatewayServer{
		config:   config,
		sessions: make(map[string]*mcp.ClientSession),
		servers:  make(map[string]*mcp.Server),
		logDir:   logDir,
	}

	// Initialize MCP sessions for each server
	if err := gateway.initializeSessions(); err != nil {
		return fmt.Errorf("failed to initialize MCP sessions: %w", err)
	}

	// Rewrite the MCP config file to point servers to the gateway
	if originalConfigPath != "" {
		if err := rewriteMCPConfigForGateway(originalConfigPath, config); err != nil {
			gatewayLog.Printf("Warning: Failed to rewrite MCP config: %v", err)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to rewrite MCP config: %v", err)))
			// Don't fail - gateway can still run
		}
	} else {
		gatewayLog.Print("Skipping config rewrite (config was read from stdin)")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping config rewrite (config was read from stdin)"))
	}

	// Start HTTP server
	return gateway.startHTTPServer()
}

// readGatewayConfig reads the gateway configuration from files or stdin
// Returns the config, the path to the first config file (for rewriting), and any error
func readGatewayConfig(configFiles []string) (*MCPGatewayServiceConfig, string, error) {
	var configs []*MCPGatewayServiceConfig
	var originalConfigPath string

	if len(configFiles) > 0 {
		// Read from file(s)
		for i, configFile := range configFiles {
			gatewayLog.Printf("Reading configuration from file: %s", configFile)
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Reading configuration from file: %s", configFile)))

			// Store the first config file path for rewriting
			if i == 0 {
				originalConfigPath = configFile
			}

			// Check if file exists
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Configuration file not found: %s", configFile)))
				gatewayLog.Printf("Configuration file not found: %s", configFile)
				return nil, "", fmt.Errorf("configuration file not found: %s", configFile)
			}

			data, err := os.ReadFile(configFile)
			if err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to read config file: %v", err)))
				return nil, "", fmt.Errorf("failed to read config file: %w", err)
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Read %d bytes from file", len(data))))
			gatewayLog.Printf("Read %d bytes from file", len(data))

			// Validate we have data
			if len(data) == 0 {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: Configuration data is empty"))
				gatewayLog.Print("Configuration data is empty")
				return nil, "", fmt.Errorf("configuration data is empty")
			}

			config, err := parseGatewayConfig(data)
			if err != nil {
				return nil, "", err
			}

			configs = append(configs, config)
		}
	} else {
		// Read from stdin
		gatewayLog.Print("Reading configuration from stdin")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Reading configuration from stdin..."))
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to read from stdin: %v", err)))
			return nil, "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Read %d bytes from stdin", len(data))))
		gatewayLog.Printf("Read %d bytes from stdin", len(data))

		if len(data) == 0 {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: No configuration data received from stdin"))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please provide configuration via --config flag or pipe JSON to stdin"))
			gatewayLog.Print("No data received from stdin")
			return nil, "", fmt.Errorf("no configuration data received from stdin")
		}

		// Validate we have data
		if len(data) == 0 {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: Configuration data is empty"))
			gatewayLog.Print("Configuration data is empty")
			return nil, "", fmt.Errorf("configuration data is empty")
		}

		config, err := parseGatewayConfig(data)
		if err != nil {
			return nil, "", err
		}

		configs = append(configs, config)
		// No config file path when reading from stdin
		originalConfigPath = ""
	}

	// Merge all configs
	if len(configs) == 0 {
		return nil, "", fmt.Errorf("no configuration loaded")
	}

	mergedConfig := configs[0]
	for i := 1; i < len(configs); i++ {
		gatewayLog.Printf("Merging configuration %d of %d", i+1, len(configs))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Merging configuration %d of %d", i+1, len(configs))))
		mergedConfig = mergeConfigs(mergedConfig, configs[i])
	}

	gatewayLog.Printf("Successfully merged %d configuration(s)", len(configs))
	if len(configs) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully merged %d configurations", len(configs))))
	}

	gatewayLog.Printf("Loaded configuration with %d MCP servers", len(mergedConfig.MCPServers))
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully loaded configuration with %d MCP servers", len(mergedConfig.MCPServers))))

	// Validate we have at least one server configured
	if len(mergedConfig.MCPServers) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: No MCP servers configured in configuration"))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Configuration must include at least one MCP server in 'mcpServers' section"))
		gatewayLog.Print("No MCP servers configured")
		return nil, "", fmt.Errorf("no MCP servers configured in configuration")
	}

	// Log server names for debugging
	serverNames := make([]string, 0, len(mergedConfig.MCPServers))
	for name := range mergedConfig.MCPServers {
		serverNames = append(serverNames, name)
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("MCP servers configured: %v", serverNames)))
	gatewayLog.Printf("MCP servers configured: %v", serverNames)

	return mergedConfig, originalConfigPath, nil
}

// parseGatewayConfig parses raw JSON data into a gateway config
func parseGatewayConfig(data []byte) (*MCPGatewayServiceConfig, error) {
	gatewayLog.Printf("Parsing %d bytes of configuration data", len(data))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Parsing %d bytes of configuration data", len(data))))

	var config MCPGatewayServiceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to parse JSON: %v", err)))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Data received (first 500 chars): %s", string(data[:min(500, len(data))]))))
		gatewayLog.Printf("Failed to parse JSON: %v", err)
		return nil, fmt.Errorf("failed to parse configuration JSON: %w", err)
	}

	gatewayLog.Printf("Successfully parsed JSON configuration")

	// Apply environment variable expansion to all server configurations
	// This supports ${VAR} or $VAR patterns in URLs, headers, and env values
	expandedServers := make(map[string]parser.MCPServerConfig)
	for name, serverConfig := range config.MCPServers {
		// Expand URL field
		if serverConfig.URL != "" {
			serverConfig.URL = os.ExpandEnv(serverConfig.URL)
			gatewayLog.Printf("Expanded URL for server %s: %s", name, serverConfig.URL)
		}

		// Expand headers
		if len(serverConfig.Headers) > 0 {
			expandedHeaders := make(map[string]string)
			for key, value := range serverConfig.Headers {
				expandedHeaders[key] = os.ExpandEnv(value)
			}
			serverConfig.Headers = expandedHeaders
			gatewayLog.Printf("Expanded %d headers for server %s", len(expandedHeaders), name)
		}

		// Expand environment variables
		if len(serverConfig.Env) > 0 {
			expandedEnv := make(map[string]string)
			for key, value := range serverConfig.Env {
				expandedEnv[key] = os.ExpandEnv(value)
			}
			serverConfig.Env = expandedEnv
			gatewayLog.Printf("Expanded %d env vars for server %s", len(expandedEnv), name)
		}

		expandedServers[name] = serverConfig
	}
	config.MCPServers = expandedServers

	return &config, nil
}

// mergeConfigs merges two gateway configurations, with the second overriding the first
func mergeConfigs(base, override *MCPGatewayServiceConfig) *MCPGatewayServiceConfig {
	result := &MCPGatewayServiceConfig{
		MCPServers: make(map[string]parser.MCPServerConfig),
		Gateway:    base.Gateway,
	}

	// Copy all servers from base
	for name, config := range base.MCPServers {
		result.MCPServers[name] = config
	}

	// Override/add servers from override config
	for name, config := range override.MCPServers {
		gatewayLog.Printf("Merging server config for: %s", name)
		result.MCPServers[name] = config
	}

	// Override gateway settings if provided
	if override.Gateway.Port != 0 {
		result.Gateway.Port = override.Gateway.Port
		gatewayLog.Printf("Override gateway port: %d", override.Gateway.Port)
	}
	if override.Gateway.APIKey != "" {
		result.Gateway.APIKey = override.Gateway.APIKey
		gatewayLog.Printf("Override gateway API key (length: %d)", len(override.Gateway.APIKey))
	}

	return result
}

// rewriteMCPConfigForGateway rewrites the MCP config file to point all servers to the gateway
func rewriteMCPConfigForGateway(configPath string, config *MCPGatewayServiceConfig) error {
	// Sanitize the path to prevent path traversal attacks
	cleanPath := filepath.Clean(configPath)
	if !filepath.IsAbs(cleanPath) {
		gatewayLog.Printf("Invalid config file path (not absolute): %s", configPath)
		return fmt.Errorf("config path must be absolute: %s", configPath)
	}

	gatewayLog.Printf("Rewriting MCP config file: %s", cleanPath)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Rewriting MCP config file: %s", cleanPath)))

	// Read the original config file to preserve non-proxied servers
	gatewayLog.Printf("Reading original config from %s", cleanPath)
	// #nosec G304 - cleanPath is validated: sanitized with filepath.Clean() and verified to be absolute path (lines 377-381)
	originalConfigData, err := os.ReadFile(cleanPath)
	if err != nil {
		gatewayLog.Printf("Failed to read original config: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to read original config: %v", err)))
		return fmt.Errorf("failed to read original config: %w", err)
	}

	var originalConfig map[string]any
	if err := json.Unmarshal(originalConfigData, &originalConfig); err != nil {
		gatewayLog.Printf("Failed to parse original config: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to parse original config: %v", err)))
		return fmt.Errorf("failed to parse original config: %w", err)
	}

	port := config.Gateway.Port
	if port == 0 {
		port = 8080
	}
	// Use localhost since the rewritten config is consumed by Copilot CLI running on the host
	gatewayURL := fmt.Sprintf("http://localhost:%d", port)

	gatewayLog.Printf("Gateway URL: %s", gatewayURL)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Gateway URL: %s", gatewayURL)))

	// Get original mcpServers to preserve non-proxied servers
	var originalMCPServers map[string]any
	if servers, ok := originalConfig["mcpServers"].(map[string]any); ok {
		originalMCPServers = servers
		gatewayLog.Printf("Found %d servers in original config", len(originalMCPServers))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d servers in original config", len(originalMCPServers))))
	} else {
		originalMCPServers = make(map[string]any)
		gatewayLog.Print("No mcpServers found in original config, starting with empty map")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No mcpServers found in original config"))
	}

	// Create merged config with rewritten proxied servers and preserved non-proxied servers
	rewrittenConfig := make(map[string]any)
	mcpServers := make(map[string]any)

	// Track which servers are rewritten vs ignored for summary logging
	var rewrittenServers []string
	var ignoredServers []string

	// First, copy all servers from original (preserves non-proxied servers like safeinputs/safeoutputs)
	gatewayLog.Printf("Copying %d servers from original config to preserve non-proxied servers", len(originalMCPServers))
	for serverName, serverConfig := range originalMCPServers {
		mcpServers[serverName] = serverConfig
		gatewayLog.Printf("  Preserved server: %s", serverName)

		// Track if this server will be ignored (not rewritten)
		if _, willBeRewritten := config.MCPServers[serverName]; !willBeRewritten {
			ignoredServers = append(ignoredServers, serverName)
		}
	}

	gatewayLog.Printf("Transforming %d proxied servers to point to gateway", len(config.MCPServers))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Transforming %d proxied servers to point to gateway", len(config.MCPServers))))

	// Then, overwrite with gateway URLs for proxied servers only
	for serverName := range config.MCPServers {
		serverURL := fmt.Sprintf("%s/mcp/%s", gatewayURL, serverName)

		gatewayLog.Printf("Rewriting server '%s' to use gateway URL: %s", serverName, serverURL)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  %s -> %s", serverName, serverURL)))

		serverConfig := map[string]any{
			"type":  "http",
			"url":   serverURL,
			"tools": []string{"*"},
		}

		// Add authentication header if API key is configured
		if config.Gateway.APIKey != "" {
			gatewayLog.Printf("Adding authorization header for server '%s'", serverName)
			serverConfig["headers"] = map[string]any{
				"Authorization": fmt.Sprintf("Bearer %s", config.Gateway.APIKey),
			}
		}

		mcpServers[serverName] = serverConfig
		rewrittenServers = append(rewrittenServers, serverName)
	}

	rewrittenConfig["mcpServers"] = mcpServers

	// Do NOT include gateway section in rewritten config (per requirement)
	gatewayLog.Print("Gateway section removed from rewritten config")

	// Log summary of servers rewritten vs ignored
	gatewayLog.Printf("Server summary: %d rewritten, %d ignored, %d total", len(rewrittenServers), len(ignoredServers), len(mcpServers))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Server summary: %d rewritten, %d ignored", len(rewrittenServers), len(ignoredServers))))

	if len(rewrittenServers) > 0 {
		gatewayLog.Printf("Servers rewritten (proxied through gateway):")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Servers rewritten (proxied through gateway):"))
		for _, serverName := range rewrittenServers {
			gatewayLog.Printf("  - %s", serverName)
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  - %s", serverName)))
		}
	}

	if len(ignoredServers) > 0 {
		gatewayLog.Printf("Servers ignored (preserved as-is):")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Servers ignored (preserved as-is):"))
		for _, serverName := range ignoredServers {
			gatewayLog.Printf("  - %s", serverName)
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  - %s", serverName)))
		}
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(rewrittenConfig, "", "  ")
	if err != nil {
		gatewayLog.Printf("Failed to marshal rewritten config: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to marshal rewritten config: %v", err)))
		return fmt.Errorf("failed to marshal rewritten config: %w", err)
	}

	gatewayLog.Printf("Marshaled config to JSON: %d bytes", len(data))
	gatewayLog.Printf("Writing to file: %s", cleanPath)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Writing %d bytes to config file: %s", len(data), cleanPath)))

	// Log a preview of the config being written (first 500 chars, redacting sensitive data)
	preview := string(data)
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	// Redact any Bearer tokens in the preview
	preview = strings.ReplaceAll(preview, config.Gateway.APIKey, "******")
	gatewayLog.Printf("Config preview (redacted): %s", preview)

	// Write back to file with restricted permissions (0600) since it contains sensitive API keys
	gatewayLog.Printf("Writing file with permissions 0600 (owner read/write only)")
	// #nosec G304 - cleanPath is validated: sanitized with filepath.Clean() and verified to be absolute path (lines 377-381)
	if err := os.WriteFile(cleanPath, data, 0600); err != nil {
		gatewayLog.Printf("Failed to write rewritten config to %s: %v", cleanPath, err)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to write rewritten config: %v", err)))
		return fmt.Errorf("failed to write rewritten config: %w", err)
	}

	gatewayLog.Printf("Successfully wrote config file: %s", cleanPath)

	// Self-check: Read back the file and verify it was written correctly
	gatewayLog.Print("Performing self-check: verifying config was written correctly")
	// #nosec G304 - cleanPath is validated: sanitized with filepath.Clean() and verified to be absolute path (lines 377-381)
	verifyData, err := os.ReadFile(cleanPath)
	if err != nil {
		gatewayLog.Printf("Self-check failed: could not read back config file: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Could not verify config was written: %v", err)))
	} else {
		var verifyConfig map[string]any
		if err := json.Unmarshal(verifyData, &verifyConfig); err != nil {
			gatewayLog.Printf("Self-check failed: could not parse config: %v", err)
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Could not parse rewritten config: %v", err)))
		} else {
			// Verify mcpServers section exists
			verifyServers, ok := verifyConfig["mcpServers"].(map[string]any)
			if !ok {
				gatewayLog.Print("Self-check failed: mcpServers section missing or invalid")
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: Self-check failed - mcpServers section missing"))
				return fmt.Errorf("self-check failed: mcpServers section missing after rewrite")
			}

			// Verify all proxied servers were rewritten correctly
			verificationErrors := []string{}
			for serverName := range config.MCPServers {
				serverConfig, ok := verifyServers[serverName].(map[string]any)
				if !ok {
					verificationErrors = append(verificationErrors, fmt.Sprintf("Server '%s' missing from rewritten config", serverName))
					continue
				}

				// Check that server has correct type and URL
				serverType, hasType := serverConfig["type"].(string)
				serverURL, hasURL := serverConfig["url"].(string)

				if !hasType || serverType != "http" {
					verificationErrors = append(verificationErrors, fmt.Sprintf("Server '%s' missing 'type: http' field", serverName))
				}

				if !hasURL || !strings.Contains(serverURL, gatewayURL) {
					verificationErrors = append(verificationErrors, fmt.Sprintf("Server '%s' URL does not point to gateway", serverName))
				}
			}

			if len(verificationErrors) > 0 {
				gatewayLog.Printf("Self-check found %d verification errors", len(verificationErrors))
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("ERROR: Self-check found %d verification errors:", len(verificationErrors))))
				for _, errMsg := range verificationErrors {
					gatewayLog.Printf("  - %s", errMsg)
					fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("  - %s", errMsg)))
				}
				return fmt.Errorf("self-check failed: config rewrite verification errors")
			}

			gatewayLog.Printf("Self-check passed: all %d proxied servers correctly rewritten", len(config.MCPServers))
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Self-check passed: all %d proxied servers correctly rewritten", len(config.MCPServers))))
		}
	}

	gatewayLog.Printf("Successfully rewrote MCP config file")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully rewrote MCP config: %s", configPath)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  %d proxied servers now point to gateway at %s", len(config.MCPServers), gatewayURL)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  %d total servers in config", len(mcpServers))))

	return nil
}

// initializeSessions creates MCP sessions for all configured servers
func (g *MCPGatewayServer) initializeSessions() error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Initializing %d MCP sessions", len(g.config.MCPServers))))
	gatewayLog.Printf("Initializing %d MCP sessions", len(g.config.MCPServers))

	// This should never happen as we validate in readGatewayConfig, but double-check
	if len(g.config.MCPServers) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("ERROR: No MCP servers to initialize"))
		gatewayLog.Print("No MCP servers to initialize")
		return fmt.Errorf("no MCP servers configured")
	}

	successCount := 0
	for serverName, serverConfig := range g.config.MCPServers {
		gatewayLog.Printf("Initializing session for server: %s", serverName)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Initializing session for server: %s (command: %s, args: %v)", serverName, serverConfig.Command, serverConfig.Args)))

		session, err := g.createMCPSession(serverName, serverConfig)
		if err != nil {
			gatewayLog.Printf("Failed to initialize session for %s: %v", serverName, err)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to initialize session for %s: %v", serverName, err)))
			return fmt.Errorf("failed to create session for server %s: %w", serverName, err)
		}

		g.mu.Lock()
		g.sessions[serverName] = session
		g.mu.Unlock()

		// Create a proxy MCP server that forwards calls to this session
		proxyServer := g.createProxyServer(serverName, session)
		g.mu.Lock()
		g.servers[serverName] = proxyServer
		g.mu.Unlock()

		successCount++
		gatewayLog.Printf("Successfully initialized session for %s (%d/%d)", serverName, successCount, len(g.config.MCPServers))
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully initialized session for %s (%d/%d)", serverName, successCount, len(g.config.MCPServers))))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("All %d MCP sessions initialized successfully", len(g.config.MCPServers))))
	gatewayLog.Printf("All %d MCP sessions initialized successfully", len(g.config.MCPServers))
	return nil
}

// createMCPSession creates an MCP session for a single server configuration
func (g *MCPGatewayServer) createMCPSession(serverName string, config parser.MCPServerConfig) (*mcp.ClientSession, error) {
	// Create log file for this server (flat directory structure)
	logFile := filepath.Join(g.logDir, fmt.Sprintf("%s.log", serverName))
	gatewayLog.Printf("Creating log file for %s: %s", serverName, logFile)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Creating log file for %s: %s", serverName, logFile)))

	logFd, err := os.Create(logFile)
	if err != nil {
		gatewayLog.Printf("Failed to create log file for %s: %v", serverName, err)
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFd.Close()

	gatewayLog.Printf("Log file created successfully for %s", serverName)

	// Handle different server types
	if config.URL != "" {
		// Streamable HTTP transport using the go-sdk StreamableClientTransport
		gatewayLog.Printf("Creating streamable HTTP client for %s at %s", serverName, config.URL)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using streamable HTTP transport: %s", config.URL)))

		// Create streamable client transport
		transport := &mcp.StreamableClientTransport{
			Endpoint: config.URL,
		}

		gatewayLog.Printf("Creating MCP client for %s", serverName)
		client := mcp.NewClient(&mcp.Implementation{
			Name:    fmt.Sprintf("gateway-client-%s", serverName),
			Version: GetVersion(),
		}, nil)

		gatewayLog.Printf("Connecting to MCP server %s with 30s timeout", serverName)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Connecting to %s...", serverName)))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			gatewayLog.Printf("Failed to connect to HTTP server %s: %v", serverName, err)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Connection failed for %s: %v", serverName, err)))
			return nil, fmt.Errorf("failed to connect to HTTP server: %w", err)
		}

		gatewayLog.Printf("Successfully connected to MCP server %s via streamable HTTP", serverName)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Connected to %s successfully via streamable HTTP", serverName)))
		return session, nil
	} else if config.Command != "" {
		// Command transport (subprocess with stdio)
		gatewayLog.Printf("Creating command client for %s with command: %s %v", serverName, config.Command, config.Args)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using command transport: %s %v", config.Command, config.Args)))

		// Create command with environment variables
		cmd := exec.Command(config.Command, config.Args...)
		if len(config.Env) > 0 {
			gatewayLog.Printf("Setting %d environment variables for %s", len(config.Env), serverName)
			cmd.Env = os.Environ()
			for k, v := range config.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
				gatewayLog.Printf("Env var for %s: %s=%s", serverName, k, v)
			}
		}

		// Create command transport
		gatewayLog.Printf("Creating CommandTransport for %s", serverName)
		transport := &mcp.CommandTransport{
			Command: cmd,
		}

		gatewayLog.Printf("Creating MCP client for %s", serverName)
		client := mcp.NewClient(&mcp.Implementation{
			Name:    fmt.Sprintf("gateway-client-%s", serverName),
			Version: GetVersion(),
		}, nil)

		gatewayLog.Printf("Connecting to MCP server %s with 30s timeout", serverName)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Connecting to %s...", serverName)))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			gatewayLog.Printf("Failed to connect to command server %s: %v", serverName, err)
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Connection failed for %s: %v", serverName, err)))
			return nil, fmt.Errorf("failed to connect to command server: %w", err)
		}

		gatewayLog.Printf("Successfully connected to MCP server %s", serverName)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Connected to %s successfully", serverName)))
		return session, nil
	} else if config.Container != "" {
		// Docker container (not yet implemented)
		gatewayLog.Printf("Docker container requested for %s but not yet implemented", serverName)
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Docker container support not available for %s", serverName)))
		return nil, fmt.Errorf("docker container support not yet implemented")
	}

	gatewayLog.Printf("Invalid server configuration for %s: no command, url, or container specified", serverName)
	fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Invalid configuration for %s: must specify command, url, or container", serverName)))
	return nil, fmt.Errorf("invalid server configuration: must specify command, url, or container")
}

// createProxyServer creates a proxy MCP server that forwards all calls to the backend session
func (g *MCPGatewayServer) createProxyServer(serverName string, session *mcp.ClientSession) *mcp.Server {
	gatewayLog.Printf("Creating proxy MCP server for %s", serverName)

	// Create a server that will proxy requests to the backend session
	server := mcp.NewServer(&mcp.Implementation{
		Name:    fmt.Sprintf("gateway-proxy-%s", serverName),
		Version: GetVersion(),
	}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{
				ListChanged: false,
			},
			Resources: &mcp.ResourceCapabilities{
				Subscribe:   false,
				ListChanged: false,
			},
			Prompts: &mcp.PromptCapabilities{
				ListChanged: false,
			},
		},
		Logger: logger.NewSlogLoggerWithHandler(gatewayLog),
	})

	// Query backend for its tools and register them on the proxy server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// List tools from backend
	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		gatewayLog.Printf("Warning: Failed to list tools from backend %s: %v", serverName, err)
	} else {
		// Register each tool on the proxy server
		for _, tool := range toolsResult.Tools {
			toolCopy := tool // Capture for closure
			gatewayLog.Printf("Registering tool %s from backend %s", tool.Name, serverName)

			server.AddTool(toolCopy, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				gatewayLog.Printf("Proxy %s: Calling tool %s on backend", serverName, req.Params.Name)
				return session.CallTool(ctx, &mcp.CallToolParams{
					Name:      req.Params.Name,
					Arguments: req.Params.Arguments,
				})
			})
		}
		gatewayLog.Printf("Registered %d tools from backend %s", len(toolsResult.Tools), serverName)
	}

	// List resources from backend
	resourcesResult, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		gatewayLog.Printf("Warning: Failed to list resources from backend %s: %v", serverName, err)
	} else {
		// Register each resource on the proxy server
		for _, resource := range resourcesResult.Resources {
			resourceCopy := resource // Capture for closure
			gatewayLog.Printf("Registering resource %s from backend %s", resource.URI, serverName)

			server.AddResource(resourceCopy, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				gatewayLog.Printf("Proxy %s: Reading resource %s from backend", serverName, req.Params.URI)
				return session.ReadResource(ctx, &mcp.ReadResourceParams{
					URI: req.Params.URI,
				})
			})
		}
		gatewayLog.Printf("Registered %d resources from backend %s", len(resourcesResult.Resources), serverName)
	}

	// List prompts from backend
	promptsResult, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		gatewayLog.Printf("Warning: Failed to list prompts from backend %s: %v", serverName, err)
	} else {
		// Register each prompt on the proxy server
		for _, prompt := range promptsResult.Prompts {
			promptCopy := prompt // Capture for closure
			gatewayLog.Printf("Registering prompt %s from backend %s", prompt.Name, serverName)

			server.AddPrompt(promptCopy, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
				gatewayLog.Printf("Proxy %s: Getting prompt %s from backend", serverName, req.Params.Name)
				return session.GetPrompt(ctx, &mcp.GetPromptParams{
					Name:      req.Params.Name,
					Arguments: req.Params.Arguments,
				})
			})
		}
		gatewayLog.Printf("Registered %d prompts from backend %s", len(promptsResult.Prompts), serverName)
	}

	gatewayLog.Printf("Proxy MCP server created for %s", serverName)
	return server
}

// startHTTPServer starts the HTTP server for the gateway
func (g *MCPGatewayServer) startHTTPServer() error {
	port := g.config.Gateway.Port
	gatewayLog.Printf("Starting HTTP server on port %d", port)

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// List servers endpoint
	mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		g.handleListServers(w, r)
	})

	// Create StreamableHTTPHandler for each MCP server
	for serverName := range g.config.MCPServers {
		serverNameCopy := serverName // Capture for closure
		path := fmt.Sprintf("/mcp/%s", serverName)
		gatewayLog.Printf("Registering StreamableHTTPHandler endpoint: %s", path)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Registering StreamableHTTPHandler endpoint: %s", path)))

		// Create streamable HTTP handler for this server
		handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
			// Get the proxy server for this backend
			g.mu.RLock()
			defer g.mu.RUnlock()
			server, exists := g.servers[serverNameCopy]
			if !exists {
				gatewayLog.Printf("Server not found in handler: %s", serverNameCopy)
				return nil
			}
			gatewayLog.Printf("Returning proxy server for: %s", serverNameCopy)
			return server
		}, &mcp.StreamableHTTPOptions{
			SessionTimeout: 2 * time.Hour, // Close idle sessions after 2 hours
			Logger:         logger.NewSlogLoggerWithHandler(gatewayLog),
		})

		// Add authentication middleware if API key is configured
		if g.config.Gateway.APIKey != "" {
			wrappedHandler := g.withAuth(handler, serverNameCopy)
			mux.Handle(path, wrappedHandler)
		} else {
			mux.Handle(path, handler)
		}
	}

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	fmt.Fprintf(os.Stderr, "%s\n", console.FormatSuccessMessage(fmt.Sprintf("MCP gateway listening on http://localhost:%d", port)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using StreamableHTTPHandler for MCP protocol"))
	gatewayLog.Printf("HTTP server ready on port %d with StreamableHTTPHandler", port)

	return httpServer.ListenAndServe()
}

// withAuth wraps an HTTP handler with authentication if API key is configured
func (g *MCPGatewayServer) withAuth(handler http.Handler, serverName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		expectedAuth := fmt.Sprintf("Bearer %s", g.config.Gateway.APIKey)
		if authHeader != expectedAuth {
			gatewayLog.Printf("Unauthorized request for %s", serverName)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// handleListServers handles the /servers endpoint
func (g *MCPGatewayServer) handleListServers(w http.ResponseWriter, r *http.Request) {
	gatewayLog.Print("Handling list servers request")

	g.mu.RLock()
	servers := make([]string, 0, len(g.sessions))
	for name := range g.sessions {
		servers = append(servers, name)
	}
	g.mu.RUnlock()

	response := map[string]any{
		"servers": servers,
		"count":   len(servers),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		gatewayLog.Printf("Failed to encode JSON response: %v", err)
	}
}
