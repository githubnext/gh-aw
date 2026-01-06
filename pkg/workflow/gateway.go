package workflow

import (
	"fmt"
	"sort"
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

// getMCPGatewayConfig extracts the MCPGatewayRuntimeConfig from sandbox configuration
func getMCPGatewayConfig(workflowData *WorkflowData) *MCPGatewayRuntimeConfig {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		return nil
	}

	return workflowData.SandboxConfig.MCP
}

// generateMCPGatewaySteps generates the steps to start and verify the MCP gateway
func generateMCPGatewaySteps(workflowData *WorkflowData, mcpEnvVars map[string]string) []GitHubActionStep {
	if !isMCPGatewayEnabled(workflowData) {
		return nil
	}

	config := getMCPGatewayConfig(workflowData)
	if config == nil {
		return nil
	}

	gatewayLog.Printf("Generating MCP gateway steps: port=%d, container=%s, command=%s, env_vars=%d",
		config.Port, config.Container, config.Command, len(mcpEnvVars))

	var steps []GitHubActionStep

	// Step 1: Start MCP Gateway (background process)
	startStep := generateMCPGatewayStartStep(config, mcpEnvVars)
	steps = append(steps, startStep)

	// Step 2: Health check to verify gateway is running
	healthCheckStep := generateMCPGatewayHealthCheckStep(config)
	steps = append(steps, healthCheckStep)

	return steps
}

// generateMCPGatewayDownloadStep generates the step that downloads the awmg binary

// generateMCPGatewayStartStep generates the step that starts the MCP gateway
func generateMCPGatewayStartStep(config *MCPGatewayRuntimeConfig, mcpEnvVars map[string]string) GitHubActionStep {
	gatewayLog.Print("Generating MCP gateway start step")

	port, err := validateAndNormalizePort(config.Port)
	if err != nil {
		// In case of validation error, log and use default port
		// This shouldn't happen in practice as validation should catch it earlier
		gatewayLog.Printf("Warning: %v, using default port %d", err, DefaultMCPGatewayPort)
		port = DefaultMCPGatewayPort
	}

	// MCP config file path (created by RenderMCPConfig)
	mcpConfigPath := "/home/runner/.copilot/mcp-config.json"

	stepLines := []string{
		"      - name: Start MCP Gateway",
	}

	// Add env block if there are environment variables to pass through
	if len(mcpEnvVars) > 0 {
		stepLines = append(stepLines, "        env:")

		// Sort environment variable names for consistent output
		envVarNames := make([]string, 0, len(mcpEnvVars))
		for envVarName := range mcpEnvVars {
			envVarNames = append(envVarNames, envVarName)
		}
		sort.Strings(envVarNames)

		// Write environment variables in sorted order
		for _, envVarName := range envVarNames {
			envVarValue := mcpEnvVars[envVarName]
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", envVarName, envVarValue))
		}
	}

	stepLines = append(stepLines,
		"        run: |",
		"          mkdir -p "+MCPGatewayLogsFolder,
		"          echo 'Starting MCP Gateway...'",
		"          ",
	)

	// Check which mode to use: container or command (both are required)
	if config.Container != "" {
		// Container mode
		gatewayLog.Printf("Using container mode: %s", config.Container)
		stepLines = append(stepLines, generateContainerStartCommands(config, mcpConfigPath, port)...)
	} else if config.Command != "" {
		// Custom command mode
		gatewayLog.Printf("Using custom command mode: %s", config.Command)
		stepLines = append(stepLines, generateCommandStartCommands(config, mcpConfigPath, port)...)
	} else {
		// Error: neither container nor command specified
		gatewayLog.Print("ERROR: Neither container nor command specified for MCP gateway")
		stepLines = append(stepLines,
			"          echo 'ERROR: sandbox.mcp must specify either container or command'",
			"          echo 'Example container mode: sandbox.mcp.container: \"ghcr.io/githubnext/gh-aw-mcpg:latest\"'",
			"          echo 'Example command mode: sandbox.mcp.command: \"./custom-gateway\"'",
			"          exit 1",
		)
	}

	return GitHubActionStep(stepLines)
}

// generateContainerStartCommands generates shell commands to start the MCP gateway using a Docker container
func generateContainerStartCommands(config *MCPGatewayRuntimeConfig, mcpConfigPath string, port int) []string {
	var lines []string

	// Build environment variables
	var envFlags []string
	if len(config.Env) > 0 {
		for key, value := range config.Env {
			envFlags = append(envFlags, fmt.Sprintf("-e %s=\"%s\"", key, value))
		}
	}
	envFlagsStr := strings.Join(envFlags, " ")

	// Build docker run command with args
	dockerCmd := "docker run"

	// Add args (e.g., --rm, -i, -v, -p)
	if len(config.Args) > 0 {
		for _, arg := range config.Args {
			dockerCmd += " " + arg
		}
	}

	// Add environment variables
	if envFlagsStr != "" {
		dockerCmd += " " + envFlagsStr
	}

	// Add container image
	containerImage := config.Container
	if config.Version != "" {
		containerImage += ":" + config.Version
	}
	dockerCmd += " " + containerImage

	// Add entrypoint args
	if len(config.EntrypointArgs) > 0 {
		for _, arg := range config.EntrypointArgs {
			dockerCmd += " " + arg
		}
	}

	lines = append(lines,
		"          # Start MCP gateway using Docker container",
		fmt.Sprintf("          echo 'Starting MCP Gateway container: %s'", config.Container),
		"          ",
		"          # Pipe MCP config to container via stdin",
		fmt.Sprintf("          cat %s | %s > %s/gateway.log 2>&1 &", mcpConfigPath, dockerCmd, MCPGatewayLogsFolder),
		"          GATEWAY_PID=$!",
		"          echo \"MCP Gateway container started with PID $GATEWAY_PID\"",
		"          ",
		"          # Give the gateway a moment to start",
		"          sleep 2",
	)

	return lines
}

// generateCommandStartCommands generates shell commands to start the MCP gateway using a custom command
func generateCommandStartCommands(config *MCPGatewayRuntimeConfig, mcpConfigPath string, port int) []string {
	var lines []string

	// Build the command with args
	command := config.Command
	if len(config.Args) > 0 {
		command += " " + strings.Join(config.Args, " ")
	}

	// Build environment variables
	var envVars []string
	if len(config.Env) > 0 {
		for key, value := range config.Env {
			envVars = append(envVars, fmt.Sprintf("export %s=\"%s\"", key, value))
		}
	}

	lines = append(lines,
		"          # Start MCP gateway using custom command",
		fmt.Sprintf("          echo 'Starting MCP Gateway with command: %s'", config.Command),
		"          ",
	)

	// Add environment variables if any
	if len(envVars) > 0 {
		lines = append(lines, "          # Set environment variables")
		for _, envVar := range envVars {
			lines = append(lines, "          "+envVar)
		}
		lines = append(lines, "          ")
	}

	lines = append(lines,
		"          # Start the command in background",
		fmt.Sprintf("          cat %s | %s > %s/gateway.log 2>&1 &", mcpConfigPath, command, MCPGatewayLogsFolder),
		"          GATEWAY_PID=$!",
		"          echo \"MCP Gateway started with PID $GATEWAY_PID\"",
		"          ",
		"          # Give the gateway a moment to start",
		"          sleep 2",
	)

	return lines
}

// generateMCPGatewayHealthCheckStep generates the step that pings the gateway to verify it's running
func generateMCPGatewayHealthCheckStep(config *MCPGatewayRuntimeConfig) GitHubActionStep {
	gatewayLog.Print("Generating MCP gateway health check step")

	port, err := validateAndNormalizePort(config.Port)
	if err != nil {
		// In case of validation error, log and use default port
		// This shouldn't happen in practice as validation should catch it earlier
		gatewayLog.Printf("Warning: %v, using default port %d", err, DefaultMCPGatewayPort)
		port = DefaultMCPGatewayPort
	}

	gatewayURL := fmt.Sprintf("http://localhost:%d", port)

	// MCP config file path (created by RenderMCPConfig)
	mcpConfigPath := "/home/runner/.copilot/mcp-config.json"

	// Call the bundled shell script to verify gateway health
	stepLines := []string{
		"      - name: Verify MCP Gateway Health",
		fmt.Sprintf("        run: bash /tmp/gh-aw/actions/verify_mcp_gateway_health.sh \"%s\" \"%s\" \"%s\"", gatewayURL, mcpConfigPath, MCPGatewayLogsFolder),
	}

	return GitHubActionStep(stepLines)
}

// getMCPGatewayURL returns the HTTP URL for the MCP gateway
func getMCPGatewayURL(config *MCPGatewayRuntimeConfig) string {
	port, err := validateAndNormalizePort(config.Port)
	if err != nil {
		// In case of validation error, log and use default port
		// This shouldn't happen in practice as validation should catch it earlier
		gatewayLog.Printf("Warning: %v, using default port %d", err, DefaultMCPGatewayPort)
		port = DefaultMCPGatewayPort
	}
	return fmt.Sprintf("http://localhost:%d", port)
}

// transformMCPConfigForGateway transforms the MCP server configuration to use the gateway URL
// instead of individual server configurations
func transformMCPConfigForGateway(mcpServers map[string]any, gatewayConfig *MCPGatewayRuntimeConfig) map[string]any {
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
