package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var mcpServersLog = logger.New("workflow:mcp_servers")

// hasMCPServers checks if the workflow has any MCP servers configured
func HasMCPServers(workflowData *WorkflowData) bool {
	if workflowData == nil {
		return false
	}

	mcpServersLog.Print("Checking for MCP servers in workflow configuration")
	// Check for standard MCP tools
	for toolName, toolValue := range workflowData.Tools {
		// Skip if the tool is explicitly disabled (set to false)
		if toolValue == false {
			continue
		}
		if toolName == "github" || toolName == "playwright" || toolName == "cache-memory" || toolName == "agentic-workflows" {
			return true
		}
		// Check for custom MCP tools
		if mcpConfig, ok := toolValue.(map[string]any); ok {
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				return true
			}
		}
	}

	// Check if safe-outputs is enabled (adds safe-outputs MCP server)
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		return true
	}

	// Check if safe-inputs is configured and feature flag is enabled (adds safe-inputs MCP server)
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		return true
	}

	return false
}

// collectMCPEnvironmentVariables collects all MCP-related environment variables
// from the workflow configuration to be passed to both Start MCP gateway and MCP Gateway steps
func collectMCPEnvironmentVariables(tools map[string]any, mcpTools []string, workflowData *WorkflowData, hasAgenticWorkflows bool) map[string]string {
	envVars := make(map[string]string)

	// Check for GitHub MCP server token
	hasGitHub := false
	for _, toolName := range mcpTools {
		if toolName == "github" {
			hasGitHub = true
			break
		}
	}
	if hasGitHub {
		githubTool := tools["github"]
		customGitHubToken := getGitHubToken(githubTool)
		effectiveToken := getEffectiveGitHubToken(customGitHubToken, workflowData.GitHubToken)
		envVars["GITHUB_MCP_SERVER_TOKEN"] = effectiveToken

		// Add lockdown value if it's determined from step output
		// Security: Pass step output through environment variable to prevent template injection
		// Convert "true"/"false" to "1"/"0" at the source to avoid shell conversion in templates
		if !hasGitHubLockdownExplicitlySet(githubTool) {
			envVars["GITHUB_MCP_LOCKDOWN"] = "${{ steps.determine-automatic-lockdown.outputs.lockdown == 'true' && '1' || '0' }}"
		}
	}

	// Check for safe-outputs env vars
	hasSafeOutputs := false
	for _, toolName := range mcpTools {
		if toolName == "safe-outputs" {
			hasSafeOutputs = true
			break
		}
	}
	if hasSafeOutputs {
		envVars["GH_AW_SAFE_OUTPUTS"] = "${{ env.GH_AW_SAFE_OUTPUTS }}"
		// Only add upload-assets env vars if upload-assets is configured
		if workflowData.SafeOutputs.UploadAssets != nil {
			envVars["GH_AW_ASSETS_BRANCH"] = "${{ env.GH_AW_ASSETS_BRANCH }}"
			envVars["GH_AW_ASSETS_MAX_SIZE_KB"] = "${{ env.GH_AW_ASSETS_MAX_SIZE_KB }}"
			envVars["GH_AW_ASSETS_ALLOWED_EXTS"] = "${{ env.GH_AW_ASSETS_ALLOWED_EXTS }}"
		}
	}

	// Check for safe-inputs env vars
	hasSafeInputs := false
	for _, toolName := range mcpTools {
		if toolName == "safe-inputs" {
			hasSafeInputs = true
			break
		}
	}
	if hasSafeInputs {
		// Add server configuration env vars from step outputs
		envVars["GH_AW_SAFE_INPUTS_PORT"] = "${{ steps.safe-inputs-start.outputs.port }}"
		envVars["GH_AW_SAFE_INPUTS_API_KEY"] = "${{ steps.safe-inputs-start.outputs.api_key }}"

		// Add tool-specific env vars (secrets passthrough)
		safeInputsSecrets := collectSafeInputsSecrets(workflowData.SafeInputs)
		for envVarName, secretExpr := range safeInputsSecrets {
			envVars[envVarName] = secretExpr
		}
	}

	// Check for agentic-workflows GITHUB_TOKEN
	if hasAgenticWorkflows {
		envVars["GITHUB_TOKEN"] = "${{ secrets.GITHUB_TOKEN }}"
	}

	// Check for Playwright domain secrets
	hasPlaywright := false
	for _, toolName := range mcpTools {
		if toolName == "playwright" {
			hasPlaywright = true
			break
		}
	}
	if hasPlaywright {
		// Extract all expressions from playwright arguments using ExpressionExtractor
		if playwrightTool, ok := tools["playwright"]; ok {
			allowedDomains := generatePlaywrightAllowedDomains(playwrightTool)
			customArgs := getPlaywrightCustomArgs(playwrightTool)
			playwrightAllowedDomainsSecrets := extractExpressionsFromPlaywrightArgs(allowedDomains, customArgs)
			for envVarName, originalExpr := range playwrightAllowedDomainsSecrets {
				envVars[envVarName] = originalExpr
			}
		}
	}

	return envVars
}

// generateMCPSetup generates the MCP server configuration setup
func (c *Compiler) generateMCPSetup(yaml *strings.Builder, tools map[string]any, engine CodingAgentEngine, workflowData *WorkflowData) {
	mcpServersLog.Print("Generating MCP server configuration setup")
	// Collect tools that need MCP server configuration
	var mcpTools []string

	// Check if workflowData is valid before accessing its fields
	if workflowData == nil {
		return
	}

	workflowTools := workflowData.Tools

	for toolName, toolValue := range workflowTools {
		// Skip if the tool is explicitly disabled (set to false)
		if toolValue == false {
			continue
		}
		// Standard MCP tools
		if toolName == "github" || toolName == "playwright" || toolName == "serena" || toolName == "cache-memory" || toolName == "agentic-workflows" {
			mcpTools = append(mcpTools, toolName)
		} else if mcpConfig, ok := toolValue.(map[string]any); ok {
			// Check if it's explicitly marked as MCP type in the new format
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				mcpTools = append(mcpTools, toolName)
			}
		}
	}

	// Check if safe-outputs is enabled and add to MCP tools
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		mcpTools = append(mcpTools, "safe-outputs")
	}

	// Check if safe-inputs is configured and feature flag is enabled, add to MCP tools
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		mcpTools = append(mcpTools, "safe-inputs")
	}

	// Generate safe-outputs configuration once to avoid duplicate computation
	var safeOutputConfig string
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		safeOutputConfig = generateSafeOutputsConfig(workflowData)
	}

	// Sort tools to ensure stable code generation
	sort.Strings(mcpTools)

	if mcpServersLog.Enabled() {
		mcpServersLog.Printf("Collected %d MCP tools: %v", len(mcpTools), mcpTools)
	}

	// Ensure MCP gateway config has defaults set before collecting Docker images
	ensureDefaultMCPGatewayConfig(workflowData)

	// Collect all Docker images that will be used and generate download step
	dockerImages := collectDockerImages(tools, workflowData)
	generateDownloadDockerImagesStep(yaml, dockerImages)

	// If no MCP tools, no configuration needed
	if len(mcpTools) == 0 {
		mcpServersLog.Print("No MCP tools configured, skipping MCP setup")
		return
	}

	// Install gh-aw extension if agentic-workflows tool is enabled
	hasAgenticWorkflows := false
	for _, toolName := range mcpTools {
		if toolName == "agentic-workflows" {
			hasAgenticWorkflows = true
			break
		}
	}

	// Check if shared/mcp/gh-aw.md is imported (which already installs gh-aw)
	hasGhAwImport := false
	for _, importPath := range workflowData.ImportedFiles {
		if strings.Contains(importPath, "shared/mcp/gh-aw.md") {
			hasGhAwImport = true
			break
		}
	}

	if hasAgenticWorkflows && hasGhAwImport {
		mcpServersLog.Print("Skipping gh-aw extension installation step (provided by shared/mcp/gh-aw.md import)")
	}

	// Only install gh-aw if needed and not already provided by imports
	if hasAgenticWorkflows && !hasGhAwImport {
		// Use effective token with precedence: top-level github-token > default
		effectiveToken := getEffectiveGitHubToken("", workflowData.GitHubToken)

		yaml.WriteString("      - name: Install gh-aw extension\n")
		yaml.WriteString("        env:\n")
		fmt.Fprintf(yaml, "          GH_TOKEN: %s\n", effectiveToken)
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Check if gh-aw extension is already installed\n")
		yaml.WriteString("          if gh extension list | grep -q \"githubnext/gh-aw\"; then\n")
		yaml.WriteString("            echo \"gh-aw extension already installed, upgrading...\"\n")
		yaml.WriteString("            gh extension upgrade gh-aw || true\n")
		yaml.WriteString("          else\n")
		yaml.WriteString("            echo \"Installing gh-aw extension...\"\n")
		yaml.WriteString("            gh extension install githubnext/gh-aw\n")
		yaml.WriteString("          fi\n")
		yaml.WriteString("          gh aw --version\n")
	}

	// Write safe-outputs MCP server if enabled
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		// Step 1: Write config files (config.json, tools.json, validation.json)
		yaml.WriteString("      - name: Write Safe Outputs Config\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          mkdir -p /opt/gh-aw/safeoutputs\n")
		yaml.WriteString("          mkdir -p /tmp/gh-aw/safeoutputs\n")
		yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-logs/safeoutputs\n")

		// Write the safe-outputs configuration to config.json
		if safeOutputConfig != "" {
			yaml.WriteString("          cat > /opt/gh-aw/safeoutputs/config.json << 'EOF'\n")
			yaml.WriteString("          " + safeOutputConfig + "\n")
			yaml.WriteString("          EOF\n")
		}

		// Generate and write the filtered tools.json file
		filteredToolsJSON, err := generateFilteredToolsJSON(workflowData, c.markdownPath)
		if err != nil {
			mcpServersLog.Printf("Error generating filtered tools JSON: %v", err)
			// Fall back to empty array on error
			filteredToolsJSON = "[]"
		}
		yaml.WriteString("          cat > /opt/gh-aw/safeoutputs/tools.json << 'EOF'\n")
		// Write each line of the indented JSON with proper YAML indentation
		for _, line := range strings.Split(filteredToolsJSON, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          EOF\n")

		// Generate and write the validation configuration from Go source of truth
		// Only include validation for activated safe output types to keep validation.json small
		var enabledTypes []string
		if safeOutputConfig != "" {
			var configMap map[string]any
			if err := json.Unmarshal([]byte(safeOutputConfig), &configMap); err == nil {
				for typeName := range configMap {
					enabledTypes = append(enabledTypes, typeName)
				}
			}
		}
		validationConfigJSON, err := GetValidationConfigJSON(enabledTypes)
		if err != nil {
			// Log error prominently - validation config is critical for safe output processing
			// The error will be caught at compile time if this ever fails
			mcpServersLog.Printf("CRITICAL: Error generating validation config JSON: %v - validation will not work correctly", err)
			validationConfigJSON = "{}"
		}
		yaml.WriteString("          cat > /opt/gh-aw/safeoutputs/validation.json << 'EOF'\n")
		// Write each line of the indented JSON with proper YAML indentation
		for _, line := range strings.Split(validationConfigJSON, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          EOF\n")

		// Note: The MCP server entry point (mcp-server.cjs) is now copied by actions/setup
		// from safe-outputs-mcp-server.cjs - no need to generate it here
	}

	// Write safe-inputs MCP server if configured and feature flag is enabled
	// For stdio mode, we only write the files but don't start the HTTP server
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		// Step 1: Write config files (JavaScript files are now copied by actions/setup)
		yaml.WriteString("      - name: Setup Safe Inputs Config\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          mkdir -p /opt/gh-aw/safe-inputs/logs\n")

		// Generate the tools.json configuration file
		toolsJSON := generateSafeInputsToolsConfig(workflowData.SafeInputs)
		yaml.WriteString("          cat > /opt/gh-aw/safe-inputs/tools.json << 'EOF_TOOLS_JSON'\n")
		for _, line := range strings.Split(toolsJSON, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          EOF_TOOLS_JSON\n")

		// Generate the MCP server entry point
		safeInputsMCPServer := generateSafeInputsMCPServerScript(workflowData.SafeInputs)
		yaml.WriteString("          cat > /opt/gh-aw/safe-inputs/mcp-server.cjs << 'EOFSI'\n")
		for _, line := range FormatJavaScriptForYAML(safeInputsMCPServer) {
			yaml.WriteString(line)
		}
		yaml.WriteString("          EOFSI\n")
		yaml.WriteString("          chmod +x /opt/gh-aw/safe-inputs/mcp-server.cjs\n")
		yaml.WriteString("          \n")

		// Step 2: Generate tool files (js/py/sh)
		yaml.WriteString("      - name: Setup Safe Inputs Tool Files\n")
		yaml.WriteString("        run: |\n")

		// Generate individual tool files (sorted by name for stable code generation)
		safeInputToolNames := make([]string, 0, len(workflowData.SafeInputs.Tools))
		for toolName := range workflowData.SafeInputs.Tools {
			safeInputToolNames = append(safeInputToolNames, toolName)
		}
		sort.Strings(safeInputToolNames)

		for _, toolName := range safeInputToolNames {
			toolConfig := workflowData.SafeInputs.Tools[toolName]
			if toolConfig.Script != "" {
				// JavaScript tool
				toolScript := generateSafeInputJavaScriptToolScript(toolConfig)
				fmt.Fprintf(yaml, "          cat > /opt/gh-aw/safe-inputs/%s.cjs << 'EOFJS_%s'\n", toolName, toolName)
				for _, line := range FormatJavaScriptForYAML(toolScript) {
					yaml.WriteString(line)
				}
				fmt.Fprintf(yaml, "          EOFJS_%s\n", toolName)
			} else if toolConfig.Run != "" {
				// Shell script tool
				toolScript := generateSafeInputShellToolScript(toolConfig)
				fmt.Fprintf(yaml, "          cat > /opt/gh-aw/safe-inputs/%s.sh << 'EOFSH_%s'\n", toolName, toolName)
				for _, line := range strings.Split(toolScript, "\n") {
					yaml.WriteString("          " + line + "\n")
				}
				fmt.Fprintf(yaml, "          EOFSH_%s\n", toolName)
				fmt.Fprintf(yaml, "          chmod +x /opt/gh-aw/safe-inputs/%s.sh\n", toolName)
			} else if toolConfig.Py != "" {
				// Python script tool
				toolScript := generateSafeInputPythonToolScript(toolConfig)
				fmt.Fprintf(yaml, "          cat > /opt/gh-aw/safe-inputs/%s.py << 'EOFPY_%s'\n", toolName, toolName)
				for _, line := range strings.Split(toolScript, "\n") {
					yaml.WriteString("          " + line + "\n")
				}
				fmt.Fprintf(yaml, "          EOFPY_%s\n", toolName)
				fmt.Fprintf(yaml, "          chmod +x /opt/gh-aw/safe-inputs/%s.py\n", toolName)
			} else if toolConfig.Go != "" {
				// Go script tool
				toolScript := generateSafeInputGoToolScript(toolConfig)
				fmt.Fprintf(yaml, "          cat > /opt/gh-aw/safe-inputs/%s.go << 'EOFGO_%s'\n", toolName, toolName)
				for _, line := range strings.Split(toolScript, "\n") {
					yaml.WriteString("          " + line + "\n")
				}
				fmt.Fprintf(yaml, "          EOFGO_%s\n", toolName)
			}
		}
		yaml.WriteString("          \n")

		// Step 3: Generate API key and choose port for HTTP server
		yaml.WriteString("      - name: Generate Safe Inputs MCP Server Config\n")
		yaml.WriteString("        id: safe-inputs-config\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Generate a secure random API key (360 bits of entropy, 40+ chars)\n")
		yaml.WriteString("          API_KEY=$(openssl rand -base64 45 | tr -d '/+=')\n")
		yaml.WriteString("          PORT=3000\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          # Register API key as secret to mask it from logs\n")
		yaml.WriteString("          echo \"::add-mask::${API_KEY}\"\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          # Set outputs for next steps\n")
		yaml.WriteString("          {\n")
		yaml.WriteString("            echo \"safe_inputs_api_key=${API_KEY}\"\n")
		yaml.WriteString("            echo \"safe_inputs_port=${PORT}\"\n")
		yaml.WriteString("          } >> \"$GITHUB_OUTPUT\"\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          echo \"Safe Inputs MCP server will run on port ${PORT}\"\n")
		yaml.WriteString("          \n")

		// Step 4: Start the HTTP server in the background
		yaml.WriteString("      - name: Start Safe Inputs MCP HTTP Server\n")
		yaml.WriteString("        id: safe-inputs-start\n")

		// Add env block with step outputs and tool-specific secrets
		// Security: Pass step outputs through environment variables to prevent template injection
		yaml.WriteString("        env:\n")
		yaml.WriteString("          GH_AW_SAFE_INPUTS_PORT: ${{ steps.safe-inputs-config.outputs.safe_inputs_port }}\n")
		yaml.WriteString("          GH_AW_SAFE_INPUTS_API_KEY: ${{ steps.safe-inputs-config.outputs.safe_inputs_api_key }}\n")

		safeInputsSecrets := collectSafeInputsSecrets(workflowData.SafeInputs)
		if len(safeInputsSecrets) > 0 {
			// Sort env var names for consistent output
			envVarNames := make([]string, 0, len(safeInputsSecrets))
			for envVarName := range safeInputsSecrets {
				envVarNames = append(envVarNames, envVarName)
			}
			sort.Strings(envVarNames)

			for _, envVarName := range envVarNames {
				secretExpr := safeInputsSecrets[envVarName]
				fmt.Fprintf(yaml, "          %s: %s\n", envVarName, secretExpr)
			}
		}

		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Environment variables are set above to prevent template injection\n")
		yaml.WriteString("          export GH_AW_SAFE_INPUTS_PORT\n")
		yaml.WriteString("          export GH_AW_SAFE_INPUTS_API_KEY\n")
		yaml.WriteString("          \n")

		// Call the bundled shell script to start the server
		yaml.WriteString("          bash /opt/gh-aw/actions/start_safe_inputs_server.sh\n")
		yaml.WriteString("          \n")
	}

	// Skip gateway setup if sandbox is disabled
	// When sandbox: false, MCP servers are configured without the gateway
	if !isSandboxDisabled(workflowData) {
		// Use the engine's RenderMCPConfig method
		yaml.WriteString("      - name: Start MCP gateway\n")
		yaml.WriteString("        id: start-mcp-gateway\n")

		// Collect all MCP-related environment variables using centralized helper
		mcpEnvVars := collectMCPEnvironmentVariables(tools, mcpTools, workflowData, hasAgenticWorkflows)

		// Add env block if any environment variables are needed
		if len(mcpEnvVars) > 0 {
			yaml.WriteString("        env:\n")

			// Sort environment variable names for consistent output
			envVarNames := make([]string, 0, len(mcpEnvVars))
			for envVarName := range mcpEnvVars {
				envVarNames = append(envVarNames, envVarName)
			}
			sort.Strings(envVarNames)

			// Write environment variables in sorted order
			for _, envVarName := range envVarNames {
				envVarValue := mcpEnvVars[envVarName]
				fmt.Fprintf(yaml, "          %s: %s\n", envVarName, envVarValue)
			}
		}

	yaml.WriteString("        run: |\n")
	yaml.WriteString("          set -eo pipefail\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-config\n")

	// Export gateway environment variables and build docker command BEFORE rendering MCP config
	// This allows the config to be piped directly to the gateway script
	// Per MCP Gateway Specification v1.0.0 section 4.2, variable expressions use "${VARIABLE_NAME}" syntax
	ensureDefaultMCPGatewayConfig(workflowData)
	gatewayConfig := workflowData.SandboxConfig.MCP

	port := gatewayConfig.Port
	if port == 0 {
		port = int(DefaultMCPGatewayPort)
	}

	domain := gatewayConfig.Domain
	if domain == "" {
		if workflowData.SandboxConfig.Agent != nil && workflowData.SandboxConfig.Agent.Disabled {
			domain = "localhost"
		} else {
			domain = "host.docker.internal"
		}
	}

	apiKey := gatewayConfig.APIKey
	if apiKey == "" {
		// Generate random API key at runtime
		apiKey = "$(openssl rand -base64 45 | tr -d '/+=')"
	}

	yaml.WriteString("          \n")
	yaml.WriteString("          # Export gateway environment variables for MCP config and gateway script\n")
	yaml.WriteString("          export MCP_GATEWAY_PORT=\"" + fmt.Sprintf("%d", port) + "\"\n")
	yaml.WriteString("          export MCP_GATEWAY_DOMAIN=\"" + domain + "\"\n")
	yaml.WriteString("          export MCP_GATEWAY_API_KEY=\"" + apiKey + "\"\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          # Register API key as secret to mask it from logs\n")
	yaml.WriteString("          echo \"::add-mask::${MCP_GATEWAY_API_KEY}\"\n")

	// Export engine type
	yaml.WriteString("          export GH_AW_ENGINE=\"" + engine.GetID() + "\"\n")

	// Add user-configured environment variables
	if len(gatewayConfig.Env) > 0 {
		envVarNames := make([]string, 0, len(gatewayConfig.Env))
		for envVarName := range gatewayConfig.Env {
			envVarNames = append(envVarNames, envVarName)
		}
		sort.Strings(envVarNames)

		for _, envVarName := range envVarNames {
			envVarValue := gatewayConfig.Env[envVarName]
			fmt.Fprintf(yaml, "          export %s=%s\n", envVarName, envVarValue)
		}
	}

	// Build container command
	containerImage := gatewayConfig.Container
	if gatewayConfig.Version != "" {
		containerImage += ":" + gatewayConfig.Version
	} else {
		containerImage += ":" + string(constants.DefaultMCPGatewayVersion)
	}

	containerCmd := "docker run -i --rm --network host"
	containerCmd += " -v /var/run/docker.sock:/var/run/docker.sock" // Enable docker-in-docker for MCP gateway
	// Pass required gateway environment variables
	containerCmd += " -e MCP_GATEWAY_PORT"
	containerCmd += " -e MCP_GATEWAY_DOMAIN"
	containerCmd += " -e MCP_GATEWAY_API_KEY"
	containerCmd += " -e DEBUG=\"*\""
	// Pass environment variables that MCP servers reference in their config
	// These are needed because awmg v0.0.12+ validates and resolves ${VAR} patterns at config load time
	// Environment variables used by MCP gateway
	containerCmd += " -e MCP_GATEWAY_LOG_DIR"
	// Environment variables used by safeoutputs MCP server
	containerCmd += " -e GH_AW_MCP_LOG_DIR"
	containerCmd += " -e GH_AW_SAFE_OUTPUTS"
	containerCmd += " -e GH_AW_SAFE_OUTPUTS_CONFIG_PATH"
	containerCmd += " -e GH_AW_SAFE_OUTPUTS_TOOLS_PATH"
	containerCmd += " -e GH_AW_ASSETS_BRANCH"
	containerCmd += " -e GH_AW_ASSETS_MAX_SIZE_KB"
	containerCmd += " -e GH_AW_ASSETS_ALLOWED_EXTS"
	containerCmd += " -e DEFAULT_BRANCH"
	// Environment variables used by GitHub MCP server
	containerCmd += " -e GITHUB_MCP_SERVER_TOKEN"
	containerCmd += " -e GITHUB_MCP_LOCKDOWN"
	// Standard GitHub Actions environment variables
	containerCmd += " -e GITHUB_REPOSITORY"
	containerCmd += " -e GITHUB_SERVER_URL"
	containerCmd += " -e GITHUB_SHA"
	containerCmd += " -e GITHUB_WORKSPACE"
	containerCmd += " -e GITHUB_TOKEN"
	// Environment variables used by safeinputs MCP server
	containerCmd += " -e GH_AW_SAFE_INPUTS_PORT"
	containerCmd += " -e GH_AW_SAFE_INPUTS_API_KEY"
	if len(gatewayConfig.Env) > 0 {
		envVarNames := make([]string, 0, len(gatewayConfig.Env))
		for envVarName := range gatewayConfig.Env {
			envVarNames = append(envVarNames, envVarName)
		}
		sort.Strings(envVarNames)
		for _, envVarName := range envVarNames {
			containerCmd += " -e " + envVarName
		}
	}

	// Add volume mounts
	if len(gatewayConfig.Mounts) > 0 {
		for _, mount := range gatewayConfig.Mounts {
			containerCmd += " -v " + mount
		}
	}

	containerCmd += " " + containerImage

	if len(gatewayConfig.EntrypointArgs) > 0 {
		for _, arg := range gatewayConfig.EntrypointArgs {
			containerCmd += " " + shellQuote(arg)
		}
	}

	if len(gatewayConfig.Args) > 0 {
		for _, arg := range gatewayConfig.Args {
			containerCmd += " " + shellQuote(arg)
		}
	}

	// Build the export command with proper quoting that allows variable expansion
	// We need to break out of quotes for ${GITHUB_WORKSPACE} variables
	cmdWithExpandableVars := buildDockerCommandWithExpandableVars(containerCmd)
	yaml.WriteString("          export MCP_GATEWAY_DOCKER_COMMAND=" + cmdWithExpandableVars + "\n")
	yaml.WriteString("          \n")

	// Render MCP config - this will pipe directly to the gateway script
	engine.RenderMCPConfig(yaml, tools, mcpTools, workflowData)
	}
	// Note: When sandbox is disabled, gateway config will be nil and MCP config will be generated
	// without the gateway section. The engine's RenderMCPConfig handles both cases.
}

// ensureDefaultMCPGatewayConfig ensures MCP gateway has default configuration if not provided
// The MCP gateway is mandatory and defaults to GitHubnext/gh-aw-mcpg
func ensureDefaultMCPGatewayConfig(workflowData *WorkflowData) {
	if workflowData == nil {
		return
	}

	// Ensure SandboxConfig exists
	if workflowData.SandboxConfig == nil {
		workflowData.SandboxConfig = &SandboxConfig{}
	}

	// Ensure MCP gateway config exists with defaults
	if workflowData.SandboxConfig.MCP == nil {
		mcpServersLog.Print("No MCP gateway configuration found, setting default configuration")
		workflowData.SandboxConfig.MCP = &MCPGatewayRuntimeConfig{
			Container: constants.DefaultMCPGatewayContainer,
			Version:   string(constants.DefaultMCPGatewayVersion),
			Port:      int(DefaultMCPGatewayPort),
		}
	} else {
		// Fill in defaults for missing fields
		if workflowData.SandboxConfig.MCP.Container == "" {
			workflowData.SandboxConfig.MCP.Container = constants.DefaultMCPGatewayContainer
		}
		// Only replace empty version with default - preserve user-specified versions including "latest"
		if workflowData.SandboxConfig.MCP.Version == "" {
			workflowData.SandboxConfig.MCP.Version = string(constants.DefaultMCPGatewayVersion)
		}
		if workflowData.SandboxConfig.MCP.Port == 0 {
			workflowData.SandboxConfig.MCP.Port = int(DefaultMCPGatewayPort)
		}
	}

	// Ensure default mounts are set if not provided
	if len(workflowData.SandboxConfig.MCP.Mounts) == 0 {
		mcpServersLog.Print("Setting default gateway mounts")
		workflowData.SandboxConfig.MCP.Mounts = []string{
			"/opt:/opt:ro",
			"/tmp:/tmp:rw",
			"${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:rw",
		}
	}
}

// shellQuote adds shell quoting to a string if needed
func shellQuote(s string) string {
	if strings.ContainsAny(s, " \t\n'\"\\$`") {
		// Escape single quotes and wrap in single quotes
		s = strings.ReplaceAll(s, "'", "'\\''")
		return "'" + s + "'"
	}
	return s
}

// buildDockerCommandWithExpandableVars builds a properly quoted docker command
// that allows ${GITHUB_WORKSPACE} and $GITHUB_WORKSPACE to be expanded at runtime
func buildDockerCommandWithExpandableVars(cmd string) string {
	// Replace ${GITHUB_WORKSPACE} with a placeholder that we'll handle specially
	// We want: 'docker run ... -v '"${GITHUB_WORKSPACE}"':'"${GITHUB_WORKSPACE}"':rw ...'
	// This closes the single quote, adds the variable in double quotes, then reopens single quote

	// Split on ${GITHUB_WORKSPACE} to handle it specially
	if strings.Contains(cmd, "${GITHUB_WORKSPACE}") {
		parts := strings.Split(cmd, "${GITHUB_WORKSPACE}")
		var result strings.Builder
		result.WriteString("'")
		for i, part := range parts {
			if i > 0 {
				// Add the variable expansion outside of single quotes
				result.WriteString("'\"${GITHUB_WORKSPACE}\"'")
			}
			// Escape single quotes in the part
			escapedPart := strings.ReplaceAll(part, "'", "'\\''")
			result.WriteString(escapedPart)
		}
		result.WriteString("'")
		return result.String()
	}

	// No GITHUB_WORKSPACE variable, use normal quoting
	return shellQuote(cmd)
}

// buildMCPGatewayConfig builds the gateway configuration for inclusion in MCP config files
// Per MCP Gateway Specification v1.0.0 section 4.1.3, the gateway section is required with port and domain
// Returns nil if sandbox is disabled (sandbox: false) to skip gateway completely
func buildMCPGatewayConfig(workflowData *WorkflowData) *MCPGatewayRuntimeConfig {
	if workflowData == nil {
		return nil
	}

	// If sandbox is disabled, skip gateway configuration entirely
	if isSandboxDisabled(workflowData) {
		return nil
	}

	// Ensure default configuration is set
	ensureDefaultMCPGatewayConfig(workflowData)

	// Return gateway config with required fields populated
	// Use ${...} syntax for environment variable references that will be resolved by the gateway at runtime
	// Per MCP Gateway Specification v1.0.0 section 4.2, variable expressions use "${VARIABLE_NAME}" syntax
	return &MCPGatewayRuntimeConfig{
		Port:   int(DefaultMCPGatewayPort), // Will be formatted as "${MCP_GATEWAY_PORT}" in renderer
		Domain: "${MCP_GATEWAY_DOMAIN}",    // Gateway variable expression
		APIKey: "${MCP_GATEWAY_API_KEY}",   // Gateway variable expression
	}
}

func getGitHubDockerImageVersion(githubTool any) string {
	githubDockerImageVersion := string(constants.DefaultGitHubMCPServerVersion) // Default Docker image version
	// Extract version setting from tool properties
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if versionSetting, exists := toolConfig["version"]; exists {
			// Handle different version types
			switch v := versionSetting.(type) {
			case string:
				githubDockerImageVersion = v
			case int:
				githubDockerImageVersion = fmt.Sprintf("%d", v)
			case int64:
				githubDockerImageVersion = fmt.Sprintf("%d", v)
			case uint64:
				githubDockerImageVersion = fmt.Sprintf("%d", v)
			case float64:
				// Use %g to avoid trailing zeros and scientific notation for simple numbers
				githubDockerImageVersion = fmt.Sprintf("%g", v)
			}
		}
	}
	return githubDockerImageVersion
}

// hasGitHubTool checks if the GitHub tool is configured (using ParsedTools)
func hasGitHubTool(parsedTools *Tools) bool {
	if parsedTools == nil {
		return false
	}
	return parsedTools.GitHub != nil
}

// getGitHubType extracts the mode from GitHub tool configuration (local or remote)
func getGitHubType(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if modeSetting, exists := toolConfig["mode"]; exists {
			if stringValue, ok := modeSetting.(string); ok {
				return stringValue
			}
		}
	}
	return "local" // default to local (Docker)
}

// getGitHubToken extracts the custom github-token from GitHub tool configuration
func getGitHubToken(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if tokenSetting, exists := toolConfig["github-token"]; exists {
			if stringValue, ok := tokenSetting.(string); ok {
				return stringValue
			}
		}
	}
	return ""
}

// getGitHubReadOnly checks if read-only mode is enabled for GitHub tool
// Defaults to true for security
func getGitHubReadOnly(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if readOnlySetting, exists := toolConfig["read-only"]; exists {
			if boolValue, ok := readOnlySetting.(bool); ok {
				return boolValue
			}
		}
	}
	return true // default to read-only for security
}

// getGitHubLockdown checks if lockdown mode is enabled for GitHub tool
// Defaults to false (lockdown disabled)
func getGitHubLockdown(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if lockdownSetting, exists := toolConfig["lockdown"]; exists {
			if boolValue, ok := lockdownSetting.(bool); ok {
				return boolValue
			}
		}
	}
	return false // default to lockdown disabled
}

// hasGitHubLockdownExplicitlySet checks if lockdown field is explicitly set in GitHub tool config
func hasGitHubLockdownExplicitlySet(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		_, exists := toolConfig["lockdown"]
		return exists
	}
	return false
}

// isSandboxDisabled checks if sandbox features are completely disabled (sandbox: false)
func isSandboxDisabled(workflowData *WorkflowData) bool {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		return false
	}
	// Check if sandbox was explicitly disabled via sandbox: false
	return workflowData.SandboxConfig.Agent != nil && workflowData.SandboxConfig.Agent.Disabled
}

// getGitHubToolsets extracts the toolsets configuration from GitHub tool
// Expands "default" to individual toolsets for action-friendly compatibility
func getGitHubToolsets(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if toolsetsSetting, exists := toolConfig["toolsets"]; exists {
			// Handle array format only
			switch v := toolsetsSetting.(type) {
			case []any:
				// Convert array to comma-separated string
				toolsets := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						toolsets = append(toolsets, str)
					}
				}
				toolsetsStr := strings.Join(toolsets, ",")
				// Expand "default" to individual toolsets for action-friendly compatibility
				return expandDefaultToolset(toolsetsStr)
			case []string:
				toolsetsStr := strings.Join(v, ",")
				// Expand "default" to individual toolsets for action-friendly compatibility
				return expandDefaultToolset(toolsetsStr)
			}
		}
	}
	// default to action-friendly toolsets (excludes "users" which GitHub Actions tokens don't support)
	return strings.Join(ActionFriendlyGitHubToolsets, ",")
}

// expandDefaultToolset expands "default" and "action-friendly" keywords to individual toolsets.
// This ensures that "default" and "action-friendly" in the source expand to action-friendly toolsets
// (excluding "users" which GitHub Actions tokens don't support).
func expandDefaultToolset(toolsetsStr string) string {
	if toolsetsStr == "" {
		return strings.Join(ActionFriendlyGitHubToolsets, ",")
	}

	// Split by comma and check if "default" or "action-friendly" is present
	toolsets := strings.Split(toolsetsStr, ",")
	var result []string
	seenToolsets := make(map[string]bool)

	for _, toolset := range toolsets {
		toolset = strings.TrimSpace(toolset)
		if toolset == "" {
			continue
		}

		if toolset == "default" || toolset == "action-friendly" {
			// Expand "default" or "action-friendly" to action-friendly toolsets (excludes "users")
			for _, dt := range ActionFriendlyGitHubToolsets {
				if !seenToolsets[dt] {
					result = append(result, dt)
					seenToolsets[dt] = true
				}
			}
		} else {
			// Keep other toolsets as-is (including "all", individual toolsets, etc.)
			if !seenToolsets[toolset] {
				result = append(result, toolset)
				seenToolsets[toolset] = true
			}
		}
	}

	return strings.Join(result, ",")
}

// getGitHubAllowedTools extracts the allowed tools list from GitHub tool configuration
// Returns the list of allowed tools, or nil if no allowed list is specified (which means all tools are allowed)
func getGitHubAllowedTools(githubTool any) []string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if allowedSetting, exists := toolConfig["allowed"]; exists {
			// Handle array format
			switch v := allowedSetting.(type) {
			case []any:
				// Convert array to string slice
				tools := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						tools = append(tools, str)
					}
				}
				return tools
			case []string:
				return v
			}
		}
	}
	return nil
}

func getPlaywrightDockerImageVersion(playwrightTool any) string {
	playwrightDockerImageVersion := string(constants.DefaultPlaywrightBrowserVersion) // Default Playwright browser Docker image version
	// Extract version setting from tool properties
	if toolConfig, ok := playwrightTool.(map[string]any); ok {
		if versionSetting, exists := toolConfig["version"]; exists {
			switch v := versionSetting.(type) {
			case string:
				playwrightDockerImageVersion = v
			case int:
				playwrightDockerImageVersion = fmt.Sprintf("%d", v)
			case int64:
				playwrightDockerImageVersion = fmt.Sprintf("%d", v)
			case uint64:
				playwrightDockerImageVersion = fmt.Sprintf("%d", v)
			case float64:
				playwrightDockerImageVersion = fmt.Sprintf("%g", v)
			}
		}
	}
	return playwrightDockerImageVersion
}

// getPlaywrightMCPPackageVersion extracts version setting for the @playwright/mcp NPM package
// This is separate from the Docker image version because they follow different versioning schemes
func getPlaywrightMCPPackageVersion(playwrightTool any) string {
	// Always use the default @playwright/mcp package version.
	return string(constants.DefaultPlaywrightMCPVersion)
}

// generatePlaywrightAllowedDomains extracts domain list from Playwright tool configuration with bundle resolution
// Uses the same domain bundle resolution as top-level network configuration, defaulting to localhost only
func generatePlaywrightAllowedDomains(playwrightTool any) []string {
	// Default to localhost with all port variations (same as Copilot agent default)
	allowedDomains := constants.DefaultAllowedDomains

	// Extract allowed_domains from Playwright tool configuration
	if toolConfig, ok := playwrightTool.(map[string]any); ok {
		if domainsConfig, exists := toolConfig["allowed_domains"]; exists {
			// Create a mock NetworkPermissions structure to use the same domain resolution logic
			playwrightNetwork := &NetworkPermissions{}

			switch domains := domainsConfig.(type) {
			case []string:
				playwrightNetwork.Allowed = domains
			case []any:
				// Convert []any to []string
				allowedDomainsSlice := make([]string, len(domains))
				for i, domain := range domains {
					if domainStr, ok := domain.(string); ok {
						allowedDomainsSlice[i] = domainStr
					}
				}
				playwrightNetwork.Allowed = allowedDomainsSlice
			case string:
				// Single domain as string
				playwrightNetwork.Allowed = []string{domains}
			}

			// Use the same domain bundle resolution as the top-level network configuration
			resolvedDomains := GetAllowedDomains(playwrightNetwork)

			// Ensure localhost domains are always included
			allowedDomains = parser.EnsureLocalhostDomains(resolvedDomains)
		}
	}

	return allowedDomains
}

// PlaywrightDockerArgs represents the common Docker arguments for Playwright container
type PlaywrightDockerArgs struct {
	ImageVersion      string // Version for Docker image (mcr.microsoft.com/playwright:version)
	MCPPackageVersion string // Version for NPM package (@playwright/mcp@version)
	AllowedDomains    []string
}

// generatePlaywrightDockerArgs creates the common Docker arguments for Playwright MCP server
func generatePlaywrightDockerArgs(playwrightTool any) PlaywrightDockerArgs {
	return PlaywrightDockerArgs{
		ImageVersion:      getPlaywrightDockerImageVersion(playwrightTool),
		MCPPackageVersion: getPlaywrightMCPPackageVersion(playwrightTool),
		AllowedDomains:    generatePlaywrightAllowedDomains(playwrightTool),
	}
}

// extractExpressionsFromPlaywrightArgs extracts all GitHub Actions expressions from playwright arguments
// Returns a map of environment variable names to their original expressions
// Uses the same ExpressionExtractor as used for shell script security
func extractExpressionsFromPlaywrightArgs(allowedDomains []string, customArgs []string) map[string]string {
	// Combine all arguments into a single string for extraction
	var allArgs []string
	allArgs = append(allArgs, allowedDomains...)
	allArgs = append(allArgs, customArgs...)

	if len(allArgs) == 0 {
		return make(map[string]string)
	}

	// Join all arguments with a separator that won't appear in expressions
	combined := strings.Join(allArgs, "\n")

	// Use ExpressionExtractor to find all expressions
	extractor := NewExpressionExtractor()
	mappings, err := extractor.ExtractExpressions(combined)
	if err != nil {
		return make(map[string]string)
	}

	// Convert to map of env var name -> original expression
	result := make(map[string]string)
	for _, mapping := range mappings {
		result[mapping.EnvVar] = mapping.Original
	}

	return result
}

// replaceExpressionsInPlaywrightArgs replaces all GitHub Actions expressions with environment variable references
// This prevents any expressions from being exposed in GitHub Actions logs
func replaceExpressionsInPlaywrightArgs(args []string, expressions map[string]string) []string {
	if len(expressions) == 0 {
		return args
	}

	// Create a temporary extractor with the same mappings
	combined := strings.Join(args, "\n")
	extractor := NewExpressionExtractor()
	_, _ = extractor.ExtractExpressions(combined)

	// Replace expressions in the combined string
	replaced := extractor.ReplaceExpressionsWithEnvVars(combined)

	// Split back into individual arguments
	return strings.Split(replaced, "\n")
}

// generateGitHubMCPLockdownDetectionStep generates a step to determine automatic lockdown mode
// for GitHub MCP server based on repository visibility. This step is added when:
// - GitHub tool is enabled AND
// - lockdown field is not explicitly specified in the workflow configuration
// The step includes a runtime condition that only executes if GH_AW_GITHUB_MCP_SERVER_TOKEN is defined
func (c *Compiler) generateGitHubMCPLockdownDetectionStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if GitHub tool is present
	githubTool, hasGitHub := data.Tools["github"]
	if !hasGitHub || githubTool == false {
		return
	}

	// Check if lockdown is already explicitly set
	if hasGitHubLockdownExplicitlySet(githubTool) {
		mcpServersLog.Print("Lockdown explicitly set in workflow, skipping automatic lockdown determination")
		return
	}

	mcpServersLog.Print("Generating automatic lockdown determination step for GitHub MCP server")

	// Resolve the latest version of actions/github-script
	actionRepo := "actions/github-script"
	actionVersion := string(constants.DefaultGitHubScriptVersion)
	pinnedAction, err := GetActionPinWithData(actionRepo, actionVersion, data)
	if err != nil {
		mcpServersLog.Printf("Failed to resolve %s@%s: %v", actionRepo, actionVersion, err)
		// In strict mode, this error would have been returned by GetActionPinWithData
		// In normal mode, we fall back to using the version tag without pinning
		pinnedAction = fmt.Sprintf("%s@%s", actionRepo, actionVersion)
	}

	// Generate the step using the determine_automatic_lockdown.cjs action
	// The step only runs if GH_AW_GITHUB_MCP_SERVER_TOKEN secret is defined
	yaml.WriteString("      - name: Determine automatic lockdown mode for GitHub MCP server\n")
	yaml.WriteString("        id: determine-automatic-lockdown\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          TOKEN_CHECK: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}\n")
	yaml.WriteString("        if: env.TOKEN_CHECK != ''\n")
	fmt.Fprintf(yaml, "        uses: %s\n", pinnedAction)
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const determineAutomaticLockdown = require('/opt/gh-aw/actions/determine_automatic_lockdown.cjs');\n")
	yaml.WriteString("            await determineAutomaticLockdown(github, context, core);\n")
}
