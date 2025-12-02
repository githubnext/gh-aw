package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpGatewayLog = logger.New("workflow:mcp_gateway")

const (
	// MCPGatewayFeatureFlag is the feature flag name for enabling MCP gateway
	MCPGatewayFeatureFlag = "mcp-gateway"
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

	// First check if mcp-gateway is configured in tools
	if workflowData.Tools == nil {
		return false
	}
	if _, ok := workflowData.Tools["mcp-gateway"]; !ok {
		return false
	}

	// Then check if the feature flag is enabled
	return isFeatureEnabled(MCPGatewayFeatureFlag, workflowData)
}

// getMCPGatewayConfig extracts the MCPGatewayConfig from workflow tools
func getMCPGatewayConfig(workflowData *WorkflowData) *MCPGatewayConfig {
	if workflowData == nil || workflowData.Tools == nil {
		return nil
	}

	gatewayRaw, ok := workflowData.Tools["mcp-gateway"]
	if !ok {
		return nil
	}

	configMap, ok := gatewayRaw.(map[string]any)
	if !ok {
		return nil
	}

	return parseMCPGatewayTool(configMap)
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

	mcpGatewayLog.Printf("Generating MCP gateway steps for container: %s", config.Container)

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
	mcpGatewayLog.Print("Generating MCP gateway start step")

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
		mcpGatewayLog.Printf("Failed to marshal gateway config: %v", err)
		configJSON = []byte("{}")
	}

	// Build docker run command
	var dockerArgs []string
	dockerArgs = append(dockerArgs, "run", "-d", "--rm", "--init")
	dockerArgs = append(dockerArgs, "--name", "mcp-gateway")
	dockerArgs = append(dockerArgs, "-p", fmt.Sprintf("%d:%d", port, port))

	// Add environment variables
	dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("MCP_GATEWAY_LOG_DIR=%s", MCPGatewayLogsFolder))
	for k, v := range config.Env {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Mount logs folder
	dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:%s", MCPGatewayLogsFolder, MCPGatewayLogsFolder))

	// Container image with optional version
	containerImage := config.Container
	if config.Version != "" {
		containerImage = fmt.Sprintf("%s:%s", config.Container, config.Version)
	}
	dockerArgs = append(dockerArgs, containerImage)

	// Add container args
	dockerArgs = append(dockerArgs, config.Args...)
	dockerArgs = append(dockerArgs, config.EntrypointArgs...)

	// Escape single quotes in JSON for shell
	escapedJSON := strings.ReplaceAll(string(configJSON), "'", "'\\''")

	stepLines := []string{
		"      - name: Start MCP Gateway",
		"        run: |",
		"          mkdir -p " + MCPGatewayLogsFolder,
		"          echo 'Starting MCP Gateway...'",
		"          # Start MCP gateway in background with config piped via stdin",
		fmt.Sprintf("          echo '%s' | docker %s", escapedJSON, strings.Join(dockerArgs, " ")),
		"          echo 'MCP Gateway started'",
	}

	return GitHubActionStep(stepLines)
}

// generateMCPGatewayHealthCheckStep generates the step that pings the gateway to verify it's running
func generateMCPGatewayHealthCheckStep(config *MCPGatewayConfig) GitHubActionStep {
	mcpGatewayLog.Print("Generating MCP gateway health check step")

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
		"              exit 0",
		"            fi",
		"            retry_count=$((retry_count + 1))",
		"            echo \"Waiting for gateway... (attempt $retry_count/$max_retries)\"",
		"            sleep 1",
		"          done",
		"          echo \"Error: MCP Gateway failed to start after $max_retries attempts\"",
		"          docker logs mcp-gateway || true",
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

	mcpGatewayLog.Print("Transforming MCP config for gateway")

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
