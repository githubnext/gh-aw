package workflow

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var mcpServersLog = logger.New("workflow:mcp_servers")

// getSafeOutputsMCPServerEntryScript generates the entry point script for safe-outputs MCP server
// This script requires the individual module files (not bundled)
func generateSafeOutputsMCPServerEntryScript() string {
	return `// @ts-check
// Auto-generated safe-outputs MCP server entry point
// This script uses individual module files (not bundled)

const { startSafeOutputsServer } = require("./safe_outputs_mcp_server.cjs");

// Start the server
// The server reads configuration from /tmp/gh-aw/safeoutputs/config.json
// Log directory is configured via GH_AW_MCP_LOG_DIR environment variable
if (require.main === module) {
  try {
    startSafeOutputsServer();
  } catch (error) {
    console.error(` + "`Error starting safe-outputs server: ${error instanceof Error ? error.message : String(error)}`" + `);
    process.exit(1);
  }
}

module.exports = { startSafeOutputsServer };
`
}

// getSafeOutputsDependencies returns the list of JavaScript files required for safe-outputs MCP server
// by analyzing the dependency tree starting from safe_outputs_mcp_server.cjs source
func getSafeOutputsDependencies() ([]string, error) {
	// Get all JavaScript sources
	sources := GetJavaScriptSources()

	// Get the main safe-outputs MCP server script SOURCE (not bundled)
	mainScript, ok := sources["safe_outputs_mcp_server.cjs"]
	if !ok {
		return nil, fmt.Errorf("safe_outputs_mcp_server.cjs not found in sources")
	}

	// Find all dependencies starting from the main script
	dependencies, err := FindJavaScriptDependencies(mainScript, sources, "")
	if err != nil {
		return nil, fmt.Errorf("failed to analyze safe-outputs dependencies: %w", err)
	}

	// Add the main script itself to the list (dependency tracker only returns required modules)
	dependencies["safe_outputs_mcp_server.cjs"] = true

	// Convert map to sorted slice for stable generation
	deps := make([]string, 0, len(dependencies))
	for dep := range dependencies {
		// Strip any leading path components (we just want the filename)
		filename := filepath.Base(dep)
		deps = append(deps, filename)
	}
	sort.Strings(deps)

	mcpServersLog.Printf("Safe-outputs MCP server requires %d dependencies (including main script)", len(deps))
	return deps, nil
}

// getJavaScriptFileContent returns the content for a JavaScript file by name.
// It returns the content as-is without transforming requires.
// All files are written to the same directory (/tmp/gh-aw/safeoutputs/) so they can
// use relative requires (./file.cjs) to reference each other at runtime.
// Files are NOT inlined/bundled - they are written separately and require each other at runtime.
// Top-level await patterns (like `await main();`) are wrapped in an async IIFE to work in CommonJS.
func getJavaScriptFileContent(filename string) (string, error) {
	// Get all sources
	sources := GetJavaScriptSources()

	// Look up the file
	content, ok := sources[filename]
	if !ok {
		return "", fmt.Errorf("JavaScript file not found: %s", filename)
	}

	// Patch top-level await patterns to work in CommonJS
	// This wraps `await main();` calls in an async IIFE
	content = patchTopLevelAwait(content)

	// Return content - files use relative requires (./file.cjs)
	// which work because all files are written to the same directory
	return content, nil
}

// patchTopLevelAwait wraps top-level `await main();` calls in an async IIFE.
// CommonJS modules don't support top-level await, so we need to wrap it.
//
// This transforms:
//
//	await main();
//
// Into:
//
//	(async () => { await main(); })();
func patchTopLevelAwait(content string) string {
	// Match `await main();` at the end of the file (with optional whitespace/newlines)
	// This pattern is used in safe output scripts as the entry point
	awaitMainRegex := regexp.MustCompile(`(?m)^await\s+main\s*\(\s*\)\s*;?\s*$`)

	return awaitMainRegex.ReplaceAllString(content, "(async () => { await main(); })();")
}

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

	// Collect all Docker images that will be used and generate download step
	dockerImages := collectDockerImages(tools)
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
		yaml.WriteString("          mkdir -p /tmp/gh-aw/safeoutputs\n")
		yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-logs/safeoutputs\n")

		// Write the safe-outputs configuration to config.json
		if safeOutputConfig != "" {
			yaml.WriteString("          cat > /tmp/gh-aw/safeoutputs/config.json << 'EOF'\n")
			yaml.WriteString("          " + safeOutputConfig + "\n")
			yaml.WriteString("          EOF\n")
		}

		// Generate and write the filtered tools.json file
		filteredToolsJSON, err := generateFilteredToolsJSON(workflowData)
		if err != nil {
			mcpServersLog.Printf("Error generating filtered tools JSON: %v", err)
			// Fall back to empty array on error
			filteredToolsJSON = "[]"
		}
		yaml.WriteString("          cat > /tmp/gh-aw/safeoutputs/tools.json << 'EOF'\n")
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
		yaml.WriteString("          cat > /tmp/gh-aw/safeoutputs/validation.json << 'EOF'\n")
		// Write each line of the indented JSON with proper YAML indentation
		for _, line := range strings.Split(validationConfigJSON, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          EOF\n")

		// Step 2: Copy JavaScript files using the setup-safe-outputs action
		setupSafeOutputsActionRef := c.resolveActionReference("./actions/setup-safe-outputs", workflowData)
		if setupSafeOutputsActionRef != "" {
			// For dev mode (local action path), checkout the actions folder first
			if c.actionMode.IsDev() {
				yaml.WriteString("      - name: Checkout actions folder for safe-outputs\n")
				fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/checkout"))
				yaml.WriteString("        with:\n")
				yaml.WriteString("          sparse-checkout: |\n")
				yaml.WriteString("            actions\n")
			}

			yaml.WriteString("      - name: Setup Safe Outputs JavaScript Files\n")
			fmt.Fprintf(yaml, "        uses: %s\n", setupSafeOutputsActionRef)
			yaml.WriteString("        with:\n")
			yaml.WriteString("          destination: /tmp/gh-aw/safeoutputs\n")
		} else {
			// Fallback: Write JavaScript files directly if action reference cannot be resolved
			yaml.WriteString("      - name: Write Safe Outputs JavaScript Files\n")
			yaml.WriteString("        run: |\n")

			// Get the list of required JavaScript dependencies dynamically
			dependencies, err := getSafeOutputsDependencies()
			if err != nil {
				mcpServersLog.Printf("CRITICAL: Error getting safe-outputs dependencies: %v", err)
				// Fallback to empty list if there's an error
				dependencies = []string{}
			}

			// Write each required JavaScript file
			for _, filename := range dependencies {
				// Get the content for this file
				content, err := getJavaScriptFileContent(filename)
				if err != nil {
					mcpServersLog.Printf("Error getting content for %s: %v", filename, err)
					continue
				}

				// Generate a unique EOF marker based on filename
				// Remove extension and convert to uppercase for marker
				markerName := strings.ToUpper(strings.TrimSuffix(filename, filepath.Ext(filename)))
				markerName = strings.ReplaceAll(markerName, ".", "_")
				markerName = strings.ReplaceAll(markerName, "-", "_")

				fmt.Fprintf(yaml, "          cat > /tmp/gh-aw/safeoutputs/%s << 'EOF_%s'\n", filename, markerName)
				for _, line := range FormatJavaScriptForYAML(content) {
					yaml.WriteString(line)
				}
				fmt.Fprintf(yaml, "          EOF_%s\n", markerName)
			}
		}

		// Step 3: Write the main MCP server entry point (simple script that requires modules)
		yaml.WriteString("      - name: Write Safe Outputs MCP Server Entry Point\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          cat > /tmp/gh-aw/safeoutputs/mcp-server.cjs << 'EOF'\n")
		// Use the simple entry point script instead of bundled version
		for _, line := range FormatJavaScriptForYAML(generateSafeOutputsMCPServerEntryScript()) {
			yaml.WriteString(line)
		}
		yaml.WriteString("          EOF\n")
		yaml.WriteString("          chmod +x /tmp/gh-aw/safeoutputs/mcp-server.cjs\n")
		yaml.WriteString("          \n")
	}

	// Write safe-inputs MCP server if configured and feature flag is enabled
	// For stdio mode, we only write the files but don't start the HTTP server
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		// Step 1: Create logs directory and copy JavaScript files using the setup-safe-inputs action
		yaml.WriteString("      - name: Setup Safe Inputs Directory\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          mkdir -p /tmp/gh-aw/safe-inputs/logs\n")

		// Step 2: Copy JavaScript files using the setup-safe-inputs action
		setupSafeInputsActionRef := c.resolveActionReference("./actions/setup-safe-inputs", workflowData)
		if setupSafeInputsActionRef != "" {
			// For dev mode (local action path), checkout the actions folder first
			if c.actionMode.IsDev() {
				yaml.WriteString("      - name: Checkout actions folder for safe-inputs\n")
				fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/checkout"))
				yaml.WriteString("        with:\n")
				yaml.WriteString("          sparse-checkout: |\n")
				yaml.WriteString("            actions\n")
			}

			yaml.WriteString("      - name: Setup Safe Inputs JavaScript Files\n")
			fmt.Fprintf(yaml, "        uses: %s\n", setupSafeInputsActionRef)
			yaml.WriteString("        with:\n")
			yaml.WriteString("          destination: /tmp/gh-aw/safe-inputs\n")
		} else {
			// Fallback: Write JavaScript files directly if action reference cannot be resolved
			yaml.WriteString("      - name: Write Safe Inputs JavaScript Files\n")
			yaml.WriteString("        run: |\n")

			// Write the reusable MCP server core modules
			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/read_buffer.cjs << 'EOF_READ_BUFFER'\n")
			for _, line := range FormatJavaScriptForYAML(GetReadBufferScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_READ_BUFFER\n")

			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/mcp_server_core.cjs << 'EOF_MCP_CORE'\n")
			for _, line := range FormatJavaScriptForYAML(GetMCPServerCoreScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_MCP_CORE\n")

			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/mcp_http_transport.cjs << 'EOF_MCP_HTTP_TRANSPORT'\n")
			for _, line := range FormatJavaScriptForYAML(GetMCPHTTPTransportScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_MCP_HTTP_TRANSPORT\n")

			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/mcp_logger.cjs << 'EOF_MCP_LOGGER'\n")
			for _, line := range FormatJavaScriptForYAML(GetMCPLoggerScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_MCP_LOGGER\n")

			// Write handler modules (only loaded when needed)
			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/mcp_handler_shell.cjs << 'EOF_HANDLER_SHELL'\n")
			for _, line := range FormatJavaScriptForYAML(GetMCPHandlerShellScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_HANDLER_SHELL\n")

			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/mcp_handler_python.cjs << 'EOF_HANDLER_PYTHON'\n")
			for _, line := range FormatJavaScriptForYAML(GetMCPHandlerPythonScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_HANDLER_PYTHON\n")

			// Write safe-inputs helper modules
			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/safe_inputs_config_loader.cjs << 'EOF_CONFIG_LOADER'\n")
			for _, line := range FormatJavaScriptForYAML(GetSafeInputsConfigLoaderScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_CONFIG_LOADER\n")

			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/safe_inputs_tool_factory.cjs << 'EOF_TOOL_FACTORY'\n")
			for _, line := range FormatJavaScriptForYAML(GetSafeInputsToolFactoryScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_TOOL_FACTORY\n")

			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/safe_inputs_validation.cjs << 'EOF_VALIDATION'\n")
			for _, line := range FormatJavaScriptForYAML(GetSafeInputsValidationScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_VALIDATION\n")

			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/safe_inputs_bootstrap.cjs << 'EOF_BOOTSTRAP'\n")
			for _, line := range FormatJavaScriptForYAML(GetSafeInputsBootstrapScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_BOOTSTRAP\n")

			// Write safe-inputs MCP server main module
			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/safe_inputs_mcp_server.cjs << 'EOF_SAFE_INPUTS_SERVER'\n")
			for _, line := range FormatJavaScriptForYAML(GetSafeInputsMCPServerScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_SAFE_INPUTS_SERVER\n")

			// Write safe-inputs MCP server HTTP transport module
			yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/safe_inputs_mcp_server_http.cjs << 'EOF_SAFE_INPUTS_SERVER_HTTP'\n")
			for _, line := range FormatJavaScriptForYAML(GetSafeInputsMCPServerHTTPScript()) {
				yaml.WriteString(line)
			}
			yaml.WriteString("          EOF_SAFE_INPUTS_SERVER_HTTP\n")
		}

		// Step 3: Write configuration files (tools.json and mcp-server.cjs entry point)
		yaml.WriteString("      - name: Setup Safe Inputs Config Files\n")
		yaml.WriteString("        run: |\n")

		// Generate the tools.json configuration file
		toolsJSON := generateSafeInputsToolsConfig(workflowData.SafeInputs)
		yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/tools.json << 'EOF_TOOLS_JSON'\n")
		for _, line := range strings.Split(toolsJSON, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          EOF_TOOLS_JSON\n")

		// Generate the MCP server entry point
		safeInputsMCPServer := generateSafeInputsMCPServerScript(workflowData.SafeInputs)
		yaml.WriteString("          cat > /tmp/gh-aw/safe-inputs/mcp-server.cjs << 'EOFSI'\n")
		for _, line := range FormatJavaScriptForYAML(safeInputsMCPServer) {
			yaml.WriteString(line)
		}
		yaml.WriteString("          EOFSI\n")
		yaml.WriteString("          chmod +x /tmp/gh-aw/safe-inputs/mcp-server.cjs\n")
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
				fmt.Fprintf(yaml, "          cat > /tmp/gh-aw/safe-inputs/%s.cjs << 'EOFJS_%s'\n", toolName, toolName)
				for _, line := range FormatJavaScriptForYAML(toolScript) {
					yaml.WriteString(line)
				}
				fmt.Fprintf(yaml, "          EOFJS_%s\n", toolName)
			} else if toolConfig.Run != "" {
				// Shell script tool
				toolScript := generateSafeInputShellToolScript(toolConfig)
				fmt.Fprintf(yaml, "          cat > /tmp/gh-aw/safe-inputs/%s.sh << 'EOFSH_%s'\n", toolName, toolName)
				for _, line := range strings.Split(toolScript, "\n") {
					yaml.WriteString("          " + line + "\n")
				}
				fmt.Fprintf(yaml, "          EOFSH_%s\n", toolName)
				fmt.Fprintf(yaml, "          chmod +x /tmp/gh-aw/safe-inputs/%s.sh\n", toolName)
			} else if toolConfig.Py != "" {
				// Python script tool
				toolScript := generateSafeInputPythonToolScript(toolConfig)
				fmt.Fprintf(yaml, "          cat > /tmp/gh-aw/safe-inputs/%s.py << 'EOFPY_%s'\n", toolName, toolName)
				for _, line := range strings.Split(toolScript, "\n") {
					yaml.WriteString("          " + line + "\n")
				}
				fmt.Fprintf(yaml, "          EOFPY_%s\n", toolName)
				fmt.Fprintf(yaml, "          chmod +x /tmp/gh-aw/safe-inputs/%s.py\n", toolName)
			}
		}
		yaml.WriteString("          \n")

		// Step 3: Generate API key and choose port for HTTP server using JavaScript
		yaml.WriteString("      - name: Generate Safe Inputs MCP Server Config\n")
		yaml.WriteString("        id: safe-inputs-config\n")
		fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
		yaml.WriteString("        with:\n")
		yaml.WriteString("          script: |\n")

		// Get the bundled script
		configScript := getGenerateSafeInputsConfigScript()
		for _, line := range FormatJavaScriptForYAML(configScript) {
			yaml.WriteString(line)
		}
		yaml.WriteString("            \n")
		yaml.WriteString("            // Execute the function\n")
		yaml.WriteString("            const crypto = require('crypto');\n")
		yaml.WriteString("            generateSafeInputsConfig({ core, crypto });\n")
		yaml.WriteString("          \n")

		// Step 4: Start the HTTP server in the background
		yaml.WriteString("      - name: Start Safe Inputs MCP HTTP Server\n")
		yaml.WriteString("        id: safe-inputs-start\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Set environment variables for the server\n")
		yaml.WriteString("          export GH_AW_SAFE_INPUTS_PORT=${{ steps.safe-inputs-config.outputs.safe_inputs_port }}\n")
		yaml.WriteString("          export GH_AW_SAFE_INPUTS_API_KEY=${{ steps.safe-inputs-config.outputs.safe_inputs_api_key }}\n")
		yaml.WriteString("          \n")

		// Pass through environment variables from safe-inputs config
		envVars := getSafeInputsEnvVars(workflowData.SafeInputs)
		for _, envVar := range envVars {
			fmt.Fprintf(yaml, "          export %s=\"${%s}\"\n", envVar, envVar)
		}
		yaml.WriteString("          \n")

		// Call the bundled shell script to start the server
		yaml.WriteString("          bash /tmp/gh-aw/actions/start_safe_inputs_server.sh\n")
		yaml.WriteString("          \n")
	}

	// Use the engine's RenderMCPConfig method
	yaml.WriteString("      - name: Setup MCPs\n")

	// Add env block for environment variables to prevent template injection
	needsEnvBlock := false
	hasGitHub := false
	hasSafeOutputs := false
	hasSafeInputs := false
	hasPlaywright := false
	var playwrightAllowedDomainsSecrets map[string]string
	// Note: hasAgenticWorkflows is already declared earlier in this function

	for _, toolName := range mcpTools {
		if toolName == "github" {
			hasGitHub = true
			needsEnvBlock = true
		}
		if toolName == "safe-outputs" {
			hasSafeOutputs = true
			needsEnvBlock = true
		}
		if toolName == "safe-inputs" {
			hasSafeInputs = true
			// Safe-inputs now always needs env block for port and API key
			needsEnvBlock = true
		}
		if toolName == "agentic-workflows" {
			needsEnvBlock = true
		}
		if toolName == "playwright" {
			hasPlaywright = true
			// Extract all expressions from playwright arguments using ExpressionExtractor
			if playwrightTool, ok := tools["playwright"]; ok {
				allowedDomains := generatePlaywrightAllowedDomains(playwrightTool)
				customArgs := getPlaywrightCustomArgs(playwrightTool)
				playwrightAllowedDomainsSecrets = extractExpressionsFromPlaywrightArgs(allowedDomains, customArgs)
				if len(playwrightAllowedDomainsSecrets) > 0 {
					needsEnvBlock = true
				}
			}
		}
	}

	if needsEnvBlock {
		yaml.WriteString("        env:\n")

		// Add GitHub token env var if GitHub tool is present
		if hasGitHub {
			githubTool := tools["github"]
			customGitHubToken := getGitHubToken(githubTool)
			effectiveToken := getEffectiveGitHubToken(customGitHubToken, workflowData.GitHubToken)
			yaml.WriteString("          GITHUB_MCP_SERVER_TOKEN: " + effectiveToken + "\n")
		}

		// Add safe-outputs env vars if present
		if hasSafeOutputs {
			yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}\n")
			// Only add upload-assets env vars if upload-assets is configured
			if workflowData.SafeOutputs.UploadAssets != nil {
				yaml.WriteString("          GH_AW_ASSETS_BRANCH: ${{ env.GH_AW_ASSETS_BRANCH }}\n")
				yaml.WriteString("          GH_AW_ASSETS_MAX_SIZE_KB: ${{ env.GH_AW_ASSETS_MAX_SIZE_KB }}\n")
				yaml.WriteString("          GH_AW_ASSETS_ALLOWED_EXTS: ${{ env.GH_AW_ASSETS_ALLOWED_EXTS }}\n")
			}
		}

		// Add safe-inputs env vars if present
		if hasSafeInputs {
			// Add server configuration env vars from step outputs
			yaml.WriteString("          GH_AW_SAFE_INPUTS_PORT: ${{ steps.safe-inputs-start.outputs.port }}\n")
			yaml.WriteString("          GH_AW_SAFE_INPUTS_API_KEY: ${{ steps.safe-inputs-start.outputs.api_key }}\n")

			// Add tool-specific env vars (secrets passthrough)
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
		}

		// Add GITHUB_TOKEN for agentic-workflows if present
		if hasAgenticWorkflows {
			yaml.WriteString("          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")
		}

		// Add Playwright expression environment variables if present
		if hasPlaywright && len(playwrightAllowedDomainsSecrets) > 0 {
			// Sort env var names for consistent output
			envVarNames := make([]string, 0, len(playwrightAllowedDomainsSecrets))
			for envVarName := range playwrightAllowedDomainsSecrets {
				envVarNames = append(envVarNames, envVarName)
			}
			sort.Strings(envVarNames)

			for _, envVarName := range envVarNames {
				originalExpr := playwrightAllowedDomainsSecrets[envVarName]
				fmt.Fprintf(yaml, "          %s: %s\n", envVarName, originalExpr)
			}
		}
	}

	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-config\n")
	engine.RenderMCPConfig(yaml, tools, mcpTools, workflowData)

	// Generate MCP gateway steps if configured (after Setup MCPs completes)
	// Note: Currently passing nil for mcpServersConfig as the gateway is configured via sandbox.mcp
	gatewaySteps := generateMCPGatewaySteps(workflowData, nil)
	for _, step := range gatewaySteps {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}
}

func getGitHubDockerImageVersion(githubTool any) string {
	githubDockerImageVersion := string(constants.DefaultGitHubMCPServerVersion) // Default Docker image version
	// Extract version setting from tool properties
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if versionSetting, exists := toolConfig["version"]; exists {
			if stringValue, ok := versionSetting.(string); ok {
				githubDockerImageVersion = stringValue
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
			if stringValue, ok := versionSetting.(string); ok {
				playwrightDockerImageVersion = stringValue
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
