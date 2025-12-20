package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var gatewayLog = logger.New("workflow:gateway")

const (
	// DefaultMCPGatewayPort is the default port for the MCP gateway
	DefaultMCPGatewayPort = 8080
	// MCPGatewayLogsFolder is the folder where MCP gateway logs are stored
	MCPGatewayLogsFolder = "/tmp/gh-aw/mcp-gateway-logs"
)

// MCPGatewayStdinConfig represents the configuration passed to the MCP gateway via stdin
type MCPGatewayStdinConfig struct {
	MCPServers map[string]any     `json:"mcpServers"`
	Gateway    MCPGatewaySettings `json:"gateway"`
}

// MCPGatewaySettings represents the gateway-specific settings
type MCPGatewaySettings struct {
	Port   int    `json:"port"`
	APIKey string `json:"apiKey,omitempty"`
}

// isMCPGatewayEnabled checks if the MCP gateway feature is enabled for the workflow
func isMCPGatewayEnabled(workflowData *WorkflowData) bool {
	if workflowData == nil {
		return false
	}

	// Check if sandbox.mcp is configured
	if workflowData.SandboxConfig == nil {
		return false
	}
	if workflowData.SandboxConfig.MCP == nil {
		return false
	}

	// MCP gateway is enabled by default when sandbox.mcp is configured
	return true
}

// getMCPGatewayConfig extracts the MCPGatewayConfig from sandbox configuration
func getMCPGatewayConfig(workflowData *WorkflowData) *MCPGatewayConfig {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		return nil
	}

	return workflowData.SandboxConfig.MCP
}

// generateMCPGatewaySteps generates the steps to start and verify the MCP gateway
func generateMCPGatewaySteps(workflowData *WorkflowData, mcpServersConfig map[string]any) []GitHubActionStep {
	if !isMCPGatewayEnabled(workflowData) {
		return nil
	}

	config := getMCPGatewayConfig(workflowData)
	if config == nil || config.Container == "" {
		return nil
	}

	gatewayLog.Printf("Generating MCP gateway steps for container: %s", config.Container)

	var steps []GitHubActionStep

	// Step 1: Start MCP Gateway (background process)
	startStep := generateMCPGatewayStartStep(config, mcpServersConfig)
	steps = append(steps, startStep)

	// Step 2: Health check to verify gateway is running
	healthCheckStep := generateMCPGatewayHealthCheckStep(config)
	steps = append(steps, healthCheckStep)

	return steps
}

// generateMCPGatewayStartStep generates the step that starts the MCP gateway
func generateMCPGatewayStartStep(config *MCPGatewayConfig, mcpServersConfig map[string]any) GitHubActionStep {
	gatewayLog.Print("Generating MCP gateway start step")

	port := config.Port
	if port == 0 {
		port = DefaultMCPGatewayPort
	}

	// Build the gateway stdin configuration
	gatewayConfig := MCPGatewayStdinConfig{
		MCPServers: mcpServersConfig,
		Gateway: MCPGatewaySettings{
			Port:   port,
			APIKey: config.APIKey,
		},
	}

	configJSON, err := json.Marshal(gatewayConfig)
	if err != nil {
		gatewayLog.Printf("Failed to marshal gateway config: %v", err)
		configJSON = []byte("{}")
	}

	// Escape single quotes in JSON for shell
	escapedJSON := strings.ReplaceAll(string(configJSON), "'", "'\\''")

	stepLines := []string{
		"      - name: Start MCP Gateway",
		"        run: |",
		"          mkdir -p " + MCPGatewayLogsFolder,
		"          echo 'Starting MCP Gateway...'",
		"          ",
		"          # Install awmg CLI if not already available",
		"          if ! command -v awmg &> /dev/null; then",
		"            # Check if this is a local build (gh-aw repo)",
		"            if [ -f \"./awmg\" ]; then",
		"              echo 'Using local awmg build'",
		"              AWMG_CMD=\"./awmg\"",
		"            # Check if gh-aw extension is installed (has awmg)",
		"            elif gh extension list 2>/dev/null | grep -q 'githubnext/gh-aw' && [ -f \"$HOME/.local/share/gh/extensions/gh-aw/awmg\" ]; then",
		"              echo 'Using awmg from gh-aw extension'",
		"              AWMG_CMD=\"$HOME/.local/share/gh/extensions/gh-aw/awmg\"",
		"            else",
		"              # Download awmg from releases",
		"              echo 'Downloading awmg from GitHub releases...'",
		"              AWMG_VERSION=$(gh release list --repo githubnext/gh-aw --limit 1 | grep -v draft | grep -v prerelease | head -n 1 | awk '{print $1}')",
		"              if [ -z \"$AWMG_VERSION\" ]; then",
		"                echo 'No release version found, using latest'",
		"                AWMG_VERSION='latest'",
		"              fi",
		"              ",
		"              # Detect platform",
		"              OS=$(uname -s | tr '[:upper:]' '[:lower:]')",
		"              ARCH=$(uname -m)",
		"              if [ \"$ARCH\" = \"x86_64\" ]; then ARCH=\"amd64\"; fi",
		"              if [ \"$ARCH\" = \"aarch64\" ]; then ARCH=\"arm64\"; fi",
		"              ",
		"              AWMG_BINARY=\"awmg-${OS}-${ARCH}\"",
		"              if [ \"$OS\" = \"windows\" ]; then AWMG_BINARY=\"${AWMG_BINARY}.exe\"; fi",
		"              ",
		"              # Download from releases",
		"              if [ \"$AWMG_VERSION\" = \"latest\" ]; then",
		"                gh release download --repo githubnext/gh-aw --pattern \"$AWMG_BINARY\" --dir /tmp || true",
		"              else",
		"                gh release download \"$AWMG_VERSION\" --repo githubnext/gh-aw --pattern \"$AWMG_BINARY\" --dir /tmp || true",
		"              fi",
		"              ",
		"              if [ -f \"/tmp/$AWMG_BINARY\" ]; then",
		"                chmod +x \"/tmp/$AWMG_BINARY\"",
		"                AWMG_CMD=\"/tmp/$AWMG_BINARY\"",
		"                echo 'Downloaded awmg successfully'",
		"              else",
		"                echo 'ERROR: Could not find or download awmg binary'",
		"                echo 'Please ensure awmg is available or download it from:'",
		"                echo 'https://github.com/githubnext/gh-aw/releases'",
		"                exit 1",
		"              fi",
		"            fi",
		"          else",
		"            echo 'awmg is already available'",
		"            AWMG_CMD=\"awmg\"",
		"          fi",
		"          ",
		"          # Start MCP gateway in background with config piped via stdin",
		fmt.Sprintf("          echo '%s' | $AWMG_CMD --port %d --log-dir %s > %s/gateway.log 2>&1 &", escapedJSON, port, MCPGatewayLogsFolder, MCPGatewayLogsFolder),
		"          GATEWAY_PID=$!",
		"          echo \"MCP Gateway started with PID $GATEWAY_PID\"",
		"          ",
		"          # Give the gateway a moment to start",
		"          sleep 2",
	}

	return GitHubActionStep(stepLines)
}

// generateMCPGatewayHealthCheckStep generates the step that pings the gateway to verify it's running
func generateMCPGatewayHealthCheckStep(config *MCPGatewayConfig) GitHubActionStep {
	gatewayLog.Print("Generating MCP gateway health check step")

	port := config.Port
	if port == 0 {
		port = DefaultMCPGatewayPort
	}

	gatewayURL := fmt.Sprintf("http://localhost:%d", port)

	stepLines := []string{
		"      - name: Verify MCP Gateway Health",
		"        run: |",
		"          echo 'Waiting for MCP Gateway to be ready...'",
		"          max_retries=30",
		"          retry_count=0",
		fmt.Sprintf("          gateway_url=\"%s\"", gatewayURL),
		"          while [ $retry_count -lt $max_retries ]; do",
		"            if curl -s -o /dev/null -w \"%{http_code}\" \"${gateway_url}/health\" | grep -q \"200\\|204\"; then",
		"              echo \"MCP Gateway is ready!\"",
		"              curl -s \"${gateway_url}/servers\" || echo \"Could not fetch servers list\"",
		"              exit 0",
		"            fi",
		"            retry_count=$((retry_count + 1))",
		"            echo \"Waiting for gateway... (attempt $retry_count/$max_retries)\"",
		"            sleep 1",
		"          done",
		"          echo \"Error: MCP Gateway failed to start after $max_retries attempts\"",
		"          ",
		"          # Show gateway logs for debugging",
		fmt.Sprintf("          echo 'Gateway logs:'"),
		fmt.Sprintf("          cat %s/gateway.log || echo 'No gateway logs found'", MCPGatewayLogsFolder),
		"          exit 1",
	}

	return GitHubActionStep(stepLines)
}

// getMCPGatewayURL returns the HTTP URL for the MCP gateway
func getMCPGatewayURL(config *MCPGatewayConfig) string {
	port := config.Port
	if port == 0 {
		port = DefaultMCPGatewayPort
	}
	return fmt.Sprintf("http://localhost:%d", port)
}

// transformMCPConfigForGateway transforms the MCP server configuration to use the gateway URL
// instead of individual server configurations
func transformMCPConfigForGateway(mcpServers map[string]any, gatewayConfig *MCPGatewayConfig) map[string]any {
	if gatewayConfig == nil {
		return mcpServers
	}

	gatewayLog.Print("Transforming MCP config for gateway")

	gatewayURL := getMCPGatewayURL(gatewayConfig)

	// Create a new config that points all servers to the gateway
	transformed := make(map[string]any)
	for serverName := range mcpServers {
		transformed[serverName] = map[string]any{
			"type": "http",
			"url":  fmt.Sprintf("%s/mcp/%s", gatewayURL, serverName),
		}
		// Add API key header if configured
		if gatewayConfig.APIKey != "" {
			transformed[serverName].(map[string]any)["headers"] = map[string]any{
				"Authorization": "Bearer ${MCP_GATEWAY_API_KEY}",
			}
		}
	}

	return transformed
}
