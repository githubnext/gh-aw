package workflow

import (
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

// validateAndNormalizePort validates the port value and returns the normalized port or an error
func validateAndNormalizePort(port int) (int, error) {
	// If port is 0, use the default
	if port == 0 {
		return DefaultMCPGatewayPort, nil
	}
	
	// Validate port is in valid range (1-65535)
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	
	return port, nil
}

// generateMCPGatewayStartStep generates the step that starts the MCP gateway
func generateMCPGatewayStartStep(config *MCPGatewayConfig, mcpServersConfig map[string]any) GitHubActionStep {
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
		"        run: |",
		"          mkdir -p " + MCPGatewayLogsFolder,
		"          echo 'Starting MCP Gateway...'",
		"          ",
	}

	// Check which mode to use: container, command, or default (awmg binary)
	if config.Container != "" {
		// Container mode
		gatewayLog.Printf("Using container mode: %s", config.Container)
		stepLines = append(stepLines, generateContainerStartCommands(config, mcpConfigPath, port)...)
	} else if config.Command != "" {
		// Custom command mode
		gatewayLog.Printf("Using custom command mode: %s", config.Command)
		stepLines = append(stepLines, generateCommandStartCommands(config, mcpConfigPath, port)...)
	} else {
		// Default mode: use awmg binary
		gatewayLog.Print("Using default mode: awmg binary")
		stepLines = append(stepLines, generateDefaultAWMGCommands(config, mcpConfigPath, port)...)
	}

	return GitHubActionStep(stepLines)
}

// generateContainerStartCommands generates shell commands to start the MCP gateway using a Docker container
func generateContainerStartCommands(config *MCPGatewayConfig, mcpConfigPath string, port int) []string {
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
func generateCommandStartCommands(config *MCPGatewayConfig, mcpConfigPath string, port int) []string {
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

// generateDefaultAWMGCommands generates shell commands to start the MCP gateway using the default awmg binary
func generateDefaultAWMGCommands(config *MCPGatewayConfig, mcpConfigPath string, port int) []string {
	var lines []string

	// Detect action mode at compile time
	// Gateway steps use dev mode by default since they're generated at compile time
	actionMode := DetectActionMode("dev")
	gatewayLog.Printf("Generating gateway step for action mode: %s", actionMode)

	// Generate different installation code based on compile-time mode
	if actionMode.IsDev() {
		// Development mode: build from sources
		gatewayLog.Print("Using development mode - will build awmg from sources")
		lines = append(lines,
			"          # Development mode: Build awmg from sources",
			"          if [ -f \"cmd/awmg/main.go\" ] && [ -f \"Makefile\" ]; then",
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
			"            echo 'ERROR: Could not find awmg binary or source files'",
			"            echo 'Please build awmg with: make build-awmg'",
			"            exit 1",
			"          fi",
		)
	} else {
		// Release mode: download from GitHub releases
		gatewayLog.Print("Using release mode - will download awmg from releases")
		lines = append(lines,
			"          # Release mode: Download awmg from releases",
			"          # Check if awmg is already in PATH",
			"          if command -v awmg &> /dev/null; then",
			"            echo 'awmg is already available in PATH'",
			"            AWMG_CMD=\"awmg\"",
			"          # Check for local awmg build",
			"          elif [ -f \"./awmg\" ]; then",
			"            echo 'Using existing local awmg build'",
			"            AWMG_CMD=\"./awmg\"",
			"          else",
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
			"              echo 'ERROR: Could not download awmg binary'",
			"              echo 'Please ensure awmg is available or download it from:'",
			"              echo 'https://github.com/githubnext/gh-aw/releases'",
			"              exit 1",
			"            fi",
			"          fi",
		)
	}

	lines = append(lines,
		"          ",
		"          # Start MCP gateway in background with config file",
		fmt.Sprintf("          $AWMG_CMD --config %s --port %d --log-dir %s > %s/gateway.log 2>&1 &", mcpConfigPath, port, MCPGatewayLogsFolder, MCPGatewayLogsFolder),
		"          GATEWAY_PID=$!",
		"          echo \"MCP Gateway started with PID $GATEWAY_PID\"",
		"          ",
		"          # Give the gateway a moment to start",
		"          sleep 2",
	)

	return lines
}

// generateMCPGatewayHealthCheckStep generates the step that pings the gateway to verify it's running
func generateMCPGatewayHealthCheckStep(config *MCPGatewayConfig) GitHubActionStep {
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

	stepLines := []string{
		"      - name: Verify MCP Gateway Health",
		"        run: |",
		"          echo 'Waiting for MCP Gateway to be ready...'",
		"          ",
		"          # Show MCP config file content",
		"          echo 'MCP Configuration:'",
		fmt.Sprintf("          cat %s || echo 'No MCP config file found'", mcpConfigPath),
		"          echo ''",
		"          ",
		"          # Verify safeinputs and safeoutputs are present in config",
		fmt.Sprintf("          if ! grep -q '\"safeinputs\"' %s; then", mcpConfigPath),
		"            echo 'ERROR: safeinputs server not found in MCP configuration'",
		"            exit 1",
		"          fi",
		fmt.Sprintf("          if ! grep -q '\"safeoutputs\"' %s; then", mcpConfigPath),
		"            echo 'ERROR: safeoutputs server not found in MCP configuration'",
		"            exit 1",
		"          fi",
		"          echo 'Verified: safeinputs and safeoutputs are present in configuration'",
		"          ",
		"          max_retries=30",
		"          retry_count=0",
		fmt.Sprintf("          gateway_url=\"%s\"", gatewayURL),
		"          while [ $retry_count -lt $max_retries ]; do",
		"            if curl -s -o /dev/null -w \"%{http_code}\" \"${gateway_url}/health\" | grep -q \"200\\|204\"; then",
		"              echo \"MCP Gateway is ready!\"",
		"              curl -s \"${gateway_url}/servers\" || echo \"Could not fetch servers list\"",
		"              ",
		"              # Test MCP server connectivity through gateway",
		"              echo ''",
		"              echo 'Testing MCP server connectivity...'",
		"              ",
		"              # Extract first external MCP server name from config (excluding safeinputs/safeoutputs)",
		fmt.Sprintf("              mcp_server=$(jq -r '.mcpServers | to_entries[] | select(.key != \"safeinputs\" and .key != \"safeoutputs\") | .key' %s | head -n 1)", mcpConfigPath),
		"              if [ -n \"$mcp_server\" ]; then",
		"                echo \"Testing connectivity to MCP server: $mcp_server\"",
		"                mcp_url=\"${gateway_url}/mcp/${mcp_server}\"",
		"                echo \"MCP URL: $mcp_url\"",
		"                ",
		"                # Test with MCP initialize call",
		"                response=$(curl -s -w \"\\n%{http_code}\" -X POST \"$mcp_url\" \\",
		"                  -H \"Content-Type: application/json\" \\",
		"                  -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2024-11-05\",\"capabilities\":{},\"clientInfo\":{\"name\":\"test\",\"version\":\"1.0.0\"}}}')",
		"                ",
		"                http_code=$(echo \"$response\" | tail -n 1)",
		"                body=$(echo \"$response\" | head -n -1)",
		"                ",
		"                echo \"HTTP Status: $http_code\"",
		"                echo \"Response: $body\"",
		"                ",
		"                if [ \"$http_code\" = \"200\" ]; then",
		"                  echo \"✓ MCP server connectivity test passed\"",
		"                else",
		"                  echo \"⚠ MCP server returned HTTP $http_code (may need authentication or different request)\"",
		"                fi",
		"              else",
		"                echo \"No external MCP servers configured for testing\"",
		"              fi",
		"              ",
		"              exit 0",
		"            fi",
		"            retry_count=$((retry_count + 1))",
		"            echo \"Waiting for gateway... (attempt $retry_count/$max_retries)\"",
		"            sleep 1",
		"          done",
		"          echo \"Error: MCP Gateway failed to start after $max_retries attempts\"",
		"          ",
		"          # Show gateway logs for debugging",
		"          echo 'Gateway logs:'",
		fmt.Sprintf("          cat %s/gateway.log || echo 'No gateway logs found'", MCPGatewayLogsFolder),
		"          exit 1",
	}

	return GitHubActionStep(stepLines)
}

// getMCPGatewayURL returns the HTTP URL for the MCP gateway
func getMCPGatewayURL(config *MCPGatewayConfig) string {
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
