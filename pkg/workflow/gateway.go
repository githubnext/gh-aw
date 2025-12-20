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
	if config == nil {
		return nil
	}

	gatewayLog.Print("Generating MCP gateway steps")

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
		"          # Detect if we're in development mode (runtime detection)",
		"          IS_DEV_MODE=\"false\"",
		"          if [ \"${GITHUB_REF}\" = \"\" ] || [[ \"${GITHUB_REF}\" == refs/pull/* ]] || [[ \"${GITHUB_REF}\" == refs/heads/* && \"${GITHUB_REF}\" != refs/heads/release* ]]; then",
		"            IS_DEV_MODE=\"true\"",
		"          fi",
		"          ",
		"          # In development mode, always build from sources if possible",
		"          if [ \"$IS_DEV_MODE\" = \"true\" ] && [ -f \"cmd/awmg/main.go\" ] && [ -f \"Makefile\" ]; then",
		"            echo 'Building awmg from sources (development mode)...'",
		"            make build-awmg",
		"            if [ -f \"./awmg\" ]; then",
		"              echo 'Built awmg successfully'",
		"              AWMG_CMD=\"./awmg\"",
		"            else",
		"              echo 'ERROR: Failed to build awmg from sources'",
		"              exit 1",
		"            fi",
		"          # Check if awmg is already in PATH",
		"          elif command -v awmg &> /dev/null; then",
		"            echo 'awmg is already available in PATH'",
		"            AWMG_CMD=\"awmg\"",
		"          # Check for local awmg build",
		"          elif [ -f \"./awmg\" ]; then",
		"            echo 'Using existing local awmg build'",
		"            AWMG_CMD=\"./awmg\"",
		"          else",
	}
	
	stepLines = append(stepLines,
		"            # Download awmg from releases",
		"            echo 'Downloading awmg from GitHub releases...'",
		"            ",
		"            # Detect platform",
		"            OS=$(uname -s | tr '[:upper:]' '[:lower:]')",
		"            ARCH=$(uname -m)",
		"            if [ \"$ARCH\" = \"x86_64\" ]; then ARCH=\"amd64\"; fi",
		"            if [ \"$ARCH\" = \"aarch64\" ]; then ARCH=\"arm64\"; fi",
		"            ",
		"            AWMG_BINARY=\"awmg-${OS}-${ARCH}\"",
		"            if [ \"$OS\" = \"windows\" ]; then AWMG_BINARY=\"${AWMG_BINARY}.exe\"; fi",
		"            ",
		"            # Download from releases using curl (no gh CLI dependency)",
		"            RELEASE_URL=\"https://github.com/githubnext/gh-aw/releases/latest/download/$AWMG_BINARY\"",
		"            echo \"Downloading from $RELEASE_URL\"",
		"            if curl -L -f -o \"/tmp/$AWMG_BINARY\" \"$RELEASE_URL\"; then",
		"              chmod +x \"/tmp/$AWMG_BINARY\"",
		"              AWMG_CMD=\"/tmp/$AWMG_BINARY\"",
		"              echo 'Downloaded awmg successfully'",
		"            else",
		"              echo 'ERROR: Could not find or download awmg binary'",
		"              echo 'Please ensure awmg is available or download it from:'",
		"              echo 'https://github.com/githubnext/gh-aw/releases'",
		"              exit 1",
		"            fi",
		"          fi",
		"          ",
		"          # Start MCP gateway in background with config piped via stdin",
		fmt.Sprintf("          echo '%s' | $AWMG_CMD --port %d --log-dir %s > %s/gateway.log 2>&1 &", escapedJSON, port, MCPGatewayLogsFolder, MCPGatewayLogsFolder),
		"          GATEWAY_PID=$!",
		"          echo \"MCP Gateway started with PID $GATEWAY_PID\"",
		"          ",
		"          # Give the gateway a moment to start",
		"          sleep 2",
	)

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
