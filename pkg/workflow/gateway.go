package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var gatewayLog = logger.New("workflow:gateway")

// ValidateGatewayVersion validates that a gateway version is properly pinned
// Returns an error if the version is "latest" or empty
func ValidateGatewayVersion(version string) error {
	if version == "latest" {
		return fmt.Errorf("gh-aw-mcpg version must be pinned (e.g., v0.1.0), 'latest' is not allowed for reproducibility")
	}
	if version == "" {
		return fmt.Errorf("gh-aw-mcpg version must be specified")
	}
	if !strings.HasPrefix(version, "v") {
		return fmt.Errorf("gh-aw-mcpg version must start with 'v' (e.g., v0.1.0), got: %s", version)
	}
	return nil
}

// GetGatewayVersion returns the gateway version to use, defaulting to DefaultMCPGatewayVersion
func GetGatewayVersion(config *MCPGatewayRuntimeConfig) string {
	if config != nil && config.Version != "" {
		return config.Version
	}
	return DefaultMCPGatewayVersion
}

// GetGatewayPort returns the gateway port to use, defaulting to DefaultMCPGatewayPort
func GetGatewayPort(config *MCPGatewayRuntimeConfig) int {
	if config != nil && config.Port > 0 {
		return config.Port
	}
	return DefaultMCPGatewayPort
}

// GetGatewaySessionToken returns the session token to use, defaulting to DefaultGatewaySessionToken
func GetGatewaySessionToken(config *MCPGatewayRuntimeConfig) string {
	if config != nil && config.SessionToken != "" {
		return config.SessionToken
	}
	return DefaultGatewaySessionToken
}

// GenerateMCPGatewayDockerCommands generates the commands to start gh-aw-mcpg as a Docker container
// The gateway runs on the host and AWF containers connect to it via host.docker.internal
func GenerateMCPGatewayDockerCommands(config *MCPGatewayRuntimeConfig, mcpConfigPath string) []string {
	version := GetGatewayVersion(config)
	port := GetGatewayPort(config)

	// Validate version at generation time
	if err := ValidateGatewayVersion(version); err != nil {
		gatewayLog.Printf("Warning: %v, using default version %s", err, DefaultMCPGatewayVersion)
		version = DefaultMCPGatewayVersion
	}

	image := fmt.Sprintf("%s:%s", DefaultMCPGatewayImage, version)

	gatewayLog.Printf("Generating gh-aw-mcpg Docker commands: image=%s, port=%d", image, port)

	var commands []string

	// Create logs directory
	commands = append(commands, fmt.Sprintf("mkdir -p %s", MCPGatewayLogsFolder))

	// Build the Docker run command
	// The config is piped via stdin to avoid file system issues
	dockerCmd := fmt.Sprintf(`cat %s | docker run \
  --rm -i \
  --name %s \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -p %d:%d \
  --add-host host.docker.internal:host-gateway \
  -e GITHUB_PERSONAL_ACCESS_TOKEN \
  %s \
  --routed --listen 0.0.0.0:%d --config-stdin \
  > %s/gateway.log 2>&1 &`,
		mcpConfigPath,
		MCPGatewayContainerName,
		port,
		MCPGatewayContainerPort,
		image,
		MCPGatewayContainerPort,
		MCPGatewayLogsFolder,
	)

	commands = append(commands, dockerCmd)

	// Wait for gateway to be healthy
	commands = append(commands, fmt.Sprintf(`echo "Waiting for gh-aw-mcpg to be ready..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:%d/health > /dev/null 2>&1; then
    echo "gh-aw-mcpg is ready"
    break
  fi
  if [ $i -eq 30 ]; then
    echo "ERROR: gh-aw-mcpg failed to start"
    cat %s/gateway.log
    exit 1
  fi
  sleep 1
done`, port, MCPGatewayLogsFolder))

	return commands
}

// GenerateMCPGatewayStartStep generates the GitHub Actions step to start the MCP gateway
// This step starts gh-aw-mcpg as a Docker container
func GenerateMCPGatewayStartStep(config *MCPGatewayRuntimeConfig, mcpEnvVars map[string]string) GitHubActionStep {
	gatewayLog.Print("Generating MCP gateway start step")

	var stepLines []string

	stepLines = append(stepLines, "      - name: Start MCP Gateway")
	stepLines = append(stepLines, "        id: mcp-gateway-start")

	// Add environment variables
	if len(mcpEnvVars) > 0 {
		stepLines = append(stepLines, "        env:")
		for key, value := range mcpEnvVars {
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, value))
		}
	}

	stepLines = append(stepLines, "        run: |")

	// Generate the Docker commands
	mcpConfigPath := "/tmp/gh-aw/mcpg-config.json"
	commands := GenerateMCPGatewayDockerCommands(config, mcpConfigPath)
	for _, cmd := range commands {
		// Indent each line of multi-line commands
		for _, line := range strings.Split(cmd, "\n") {
			stepLines = append(stepLines, fmt.Sprintf("          %s", line))
		}
	}

	return GitHubActionStep(stepLines)
}

// TransformMCPConfigForGatewayClient transforms MCP server configs for the agent client
// Each server is converted to use HTTP transport via the gateway
func TransformMCPConfigForGatewayClient(mcpServers map[string]any, config *MCPGatewayRuntimeConfig) map[string]any {
	sessionToken := GetGatewaySessionToken(config)
	port := GetGatewayPort(config)

	transformed := make(map[string]any)
	for serverName := range mcpServers {
		transformed[serverName] = map[string]any{
			"type": "http",
			"url":  fmt.Sprintf("http://host.docker.internal:%d/mcp/%s", port, serverName),
			"headers": map[string]any{
				"Authorization": fmt.Sprintf("Bearer %s", sessionToken),
			},
			"tools": []string{"*"},
		}
	}

	gatewayLog.Printf("Transformed %d MCP servers for gateway client access", len(transformed))
	return transformed
}

// IsMCPGatewayEnabled checks if MCP gateway is enabled in the workflow configuration
func IsMCPGatewayEnabled(workflowData *WorkflowData) bool {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		return false
	}
	// Gateway is enabled when sandbox.mcp is configured
	return workflowData.SandboxConfig.MCP != nil
}

// GetMCPGatewayConfig returns the MCP gateway configuration from workflow data
func GetMCPGatewayConfig(workflowData *WorkflowData) *MCPGatewayRuntimeConfig {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		return nil
	}
	return workflowData.SandboxConfig.MCP
}

// GenerateGatewayConfigPath returns the path to the gateway config file
func GenerateGatewayConfigPath() string {
	return "/tmp/gh-aw/mcpg-config.json"
}

// GenerateClientMCPConfigForGateway generates the MCP client config that routes
// all MCP server requests through the gateway
func GenerateClientMCPConfigForGateway(yaml *strings.Builder, serverNames []string, config *MCPGatewayRuntimeConfig, format string) {
	sessionToken := GetGatewaySessionToken(config)
	port := GetGatewayPort(config)

	gatewayLog.Printf("Generating client MCP config for %d servers via gateway", len(serverNames))

	if format == "json" {
		// JSON format for Copilot/Claude
		for i, serverName := range serverNames {
			isLast := i == len(serverNames)-1
			yaml.WriteString(fmt.Sprintf("              \"%s\": {\n", serverName))
			yaml.WriteString("                \"type\": \"http\",\n")
			yaml.WriteString(fmt.Sprintf("                \"url\": \"http://host.docker.internal:%d/mcp/%s\",\n", port, serverName))
			yaml.WriteString("                \"headers\": {\n")
			yaml.WriteString(fmt.Sprintf("                  \"Authorization\": \"Bearer %s\"\n", sessionToken))
			yaml.WriteString("                },\n")
			yaml.WriteString("                \"tools\": [\"*\"]\n")
			if isLast {
				yaml.WriteString("              }\n")
			} else {
				yaml.WriteString("              },\n")
			}
		}
	} else if format == "toml" {
		// TOML format for Codex
		for _, serverName := range serverNames {
			yaml.WriteString("\n")
			yaml.WriteString(fmt.Sprintf("          [mcp_servers.%s]\n", serverName))
			yaml.WriteString("          type = \"http\"\n")
			yaml.WriteString(fmt.Sprintf("          url = \"http://host.docker.internal:%d/mcp/%s\"\n", port, serverName))
			yaml.WriteString(fmt.Sprintf("          headers = { Authorization = \"Bearer %s\" }\n", sessionToken))
		}
	}
}

// ShouldUseGatewayForMCP determines if the MCP gateway should be used
// Returns true if:
// - sandbox.mcp is configured
// - AWF (firewall) is enabled
func ShouldUseGatewayForMCP(workflowData *WorkflowData) bool {
	if !IsMCPGatewayEnabled(workflowData) {
		return false
	}
	// Gateway only makes sense when firewall is enabled
	// because the gateway runs on the host and containers connect to it
	return isFirewallEnabled(workflowData)
}
