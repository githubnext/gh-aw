// Package workflow provides GitHub Actions setup step generation for MCP servers.
//
// # MCP Setup Generator
//
// This file generates the complete setup sequence for MCP servers in GitHub Actions
// workflows. It orchestrates the initialization of all MCP tools including built-in
// servers (GitHub, Playwright, safe-outputs, mcp-scripts) and custom HTTP/stdio
// MCP servers.
//
// Key responsibilities:
//   - Identifying and collecting MCP tools from workflow configuration
//   - Generating Docker image download steps
//   - Installing gh-aw extension for agentic-workflows tool
//   - Setting up safe-outputs MCP server (config, API key, HTTP server)
//   - Setting up mcp-scripts MCP server (config, tool files, HTTP server)
//   - Starting the MCP gateway with proper environment variables
//   - Rendering MCP configuration for the selected AI engine
//
// Setup sequence:
//  1. Download required Docker images
//  2. Install gh-aw extension (if agentic-workflows enabled)
//  3. Write safe-outputs config.json (may contain template expressions; kept small)
//  4. Write safe-outputs tools.json and validation.json (large, no template expressions)
//  5. Generate and start safe-outputs HTTP server
//  6. Setup mcp-scripts config and tool files (JavaScript, Python, Shell, Go)
//  7. Generate and start mcp-scripts HTTP server
//  8. Start MCP Gateway with all environment variables

// 10. Render engine-specific MCP configuration
//
// MCP tools supported:
//   - github: GitHub API access via MCP (local Docker or remote hosted)
//   - playwright: Browser automation with Playwright
//   - safe-outputs: Controlled output storage for AI agents
//   - mcp-scripts: Custom tool execution with secret passthrough
//   - cache-memory: Memory/knowledge base management
//   - agentic-workflows: Workflow execution via gh-aw
//   - Custom HTTP/stdio MCP servers
//
// Gateway modes:
//   - Enabled (default): MCP servers run through gateway proxy
//   - Disabled (sandbox: false): Direct MCP server communication
//
// Related files:
//   - mcp_gateway_config.go: Gateway configuration management
//   - mcp_environment.go: Environment variable collection
//   - mcp_renderer.go: MCP configuration YAML rendering
//   - safe_outputs.go: Safe outputs server configuration
//   - mcp_scripts.go: MCP Scripts server configuration
//
// Example workflow setup:
//   - Download Docker images
//   - Write safe-outputs config to ${RUNNER_TEMP}/gh-aw/safeoutputs/
//   - Start safe-outputs HTTP server on port 3001
//   - Write mcp-scripts config to ${RUNNER_TEMP}/gh-aw/mcp-scripts/
//   - Start mcp-scripts HTTP server on port 3000
//   - Start MCP Gateway on port 80
//   - Render MCP config based on engine (copilot/claude/codex/custom)
package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var mcpSetupGeneratorLog = logger.New("workflow:mcp_setup_generator")

// generateMCPSetup generates the MCP server configuration setup
func (c *Compiler) generateMCPSetup(yaml *strings.Builder, tools map[string]any, engine CodingAgentEngine, workflowData *WorkflowData) error {
	mcpSetupGeneratorLog.Print("Generating MCP server configuration setup")
	// Collect tools that need MCP server configuration
	var mcpTools []string

	// Check if workflowData is valid before accessing its fields
	if workflowData == nil {
		return nil
	}

	workflowTools := workflowData.Tools

	for toolName, toolValue := range workflowTools {
		// Skip if the tool is explicitly disabled (set to false)
		if toolValue == false {
			continue
		}
		// Standard MCP tools
		if toolName == "github" || toolName == "playwright" || toolName == "qmd" || toolName == "cache-memory" || toolName == "agentic-workflows" {
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

	// Check if mcp-scripts is configured and feature flag is enabled, add to MCP tools
	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		mcpTools = append(mcpTools, "mcp-scripts")
	}

	// Populate dispatch-workflow file mappings before generating config
	// This ensures workflow_files is available in the config.json
	populateDispatchWorkflowFiles(workflowData, c.markdownPath)

	// Populate call-workflow file mappings before generating config
	// This ensures workflow_files is available in the config.json
	populateCallWorkflowFiles(workflowData, c.markdownPath)

	// Generate safe-outputs configuration once to avoid duplicate computation
	var safeOutputConfig string
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		safeOutputConfig = generateSafeOutputsConfig(workflowData)
	}

	// Sort tools to ensure stable code generation
	sort.Strings(mcpTools)

	if mcpSetupGeneratorLog.Enabled() {
		mcpSetupGeneratorLog.Printf("Collected %d MCP tools: %v", len(mcpTools), mcpTools)
	}

	// Ensure MCP gateway config has defaults set before collecting Docker images
	ensureDefaultMCPGatewayConfig(workflowData)

	// Collect all Docker images that will be used and generate download step
	dockerImages := collectDockerImages(tools, workflowData, c.actionMode)
	generateDownloadDockerImagesStep(yaml, dockerImages)

	// If no MCP tools, no configuration needed
	if len(mcpTools) == 0 {
		mcpSetupGeneratorLog.Print("No MCP tools configured, skipping MCP setup")
		return nil
	}

	// Install gh-aw extension if agentic-workflows tool is enabled
	hasAgenticWorkflows := slices.Contains(mcpTools, "agentic-workflows")

	// Check if shared/mcp/gh-aw.md is imported (which already installs gh-aw)
	hasGhAwImport := false
	for _, importPath := range workflowData.ImportedFiles {
		if strings.Contains(importPath, "shared/mcp/gh-aw.md") {
			hasGhAwImport = true
			break
		}
	}

	if hasAgenticWorkflows && hasGhAwImport {
		mcpSetupGeneratorLog.Print("Skipping gh-aw extension installation step (provided by shared/mcp/gh-aw.md import)")
	}

	// Only install gh-aw if needed and not already provided by imports
	if hasAgenticWorkflows && !hasGhAwImport {
		// Use effective token with precedence: custom > default
		effectiveToken := getEffectiveGitHubToken("")

		yaml.WriteString("      - name: Install gh-aw extension\n")
		yaml.WriteString("        env:\n")
		fmt.Fprintf(yaml, "          GH_TOKEN: %s\n", effectiveToken)
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Check if gh-aw extension is already installed\n")
		yaml.WriteString("          if gh extension list | grep -q \"github/gh-aw\"; then\n")
		yaml.WriteString("            echo \"gh-aw extension already installed, upgrading...\"\n")
		yaml.WriteString("            gh extension upgrade gh-aw || true\n")
		yaml.WriteString("          else\n")
		yaml.WriteString("            echo \"Installing gh-aw extension...\"\n")
		yaml.WriteString("            gh extension install github/gh-aw\n")
		yaml.WriteString("          fi\n")
		yaml.WriteString("          gh aw --version\n")
		yaml.WriteString("          # Copy the gh-aw binary to ${RUNNER_TEMP}/gh-aw for MCP server containerization\n")
		yaml.WriteString("          mkdir -p ${RUNNER_TEMP}/gh-aw\n")
		yaml.WriteString("          GH_AW_BIN=$(which gh-aw 2>/dev/null || find ~/.local/share/gh/extensions/gh-aw -name 'gh-aw' -type f 2>/dev/null | head -1)\n")
		yaml.WriteString("          if [ -n \"$GH_AW_BIN\" ] && [ -f \"$GH_AW_BIN\" ]; then\n")
		yaml.WriteString("            cp \"$GH_AW_BIN\" ${RUNNER_TEMP}/gh-aw/gh-aw\n")
		yaml.WriteString("            chmod +x ${RUNNER_TEMP}/gh-aw/gh-aw\n")
		yaml.WriteString("            echo \"Copied gh-aw binary to ${RUNNER_TEMP}/gh-aw/gh-aw\"\n")
		yaml.WriteString("          else\n")
		yaml.WriteString("            echo \"::error::Failed to find gh-aw binary for MCP server\"\n")
		yaml.WriteString("            exit 1\n")
		yaml.WriteString("          fi\n")
	}

	// Write safe-outputs MCP server if enabled
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		// Step 1a: Write config.json (small, may contain GitHub Actions template expressions
		// such as ${{ github.ref_name }} from create-pull-request base-branch config).
		// This MUST be its own run: block, separate from the large tools.json step below,
		// to avoid "Exceeded max expression length 21000" errors in GitHub Actions.
		// GitHub Actions rejects any YAML scalar value that contains ${{ }} expressions
		// AND exceeds 21,000 characters total.
		yaml.WriteString("      - name: Write Safe Outputs Config\n")

		// SECURITY: extract any ${{ secrets.* }} from config.json content and pass them
		// as env vars so the shell treats the values as data, not syntax.
		configSecrets := ExtractSecretsFromValue(safeOutputConfig)
		if len(configSecrets) > 0 {
			yaml.WriteString("        env:\n")
			secretKeys := make([]string, 0, len(configSecrets))
			for k := range configSecrets {
				secretKeys = append(secretKeys, k)
			}
			sort.Strings(secretKeys)
			for _, varName := range secretKeys {
				yaml.WriteString("          " + varName + ": " + configSecrets[varName] + "\n")
			}
		}

		yaml.WriteString("        run: |\n")
		yaml.WriteString("          mkdir -p ${RUNNER_TEMP}/gh-aw/safeoutputs\n")
		yaml.WriteString("          mkdir -p /tmp/gh-aw/safeoutputs\n")
		yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-logs/safeoutputs\n")

		// Write the safe-outputs configuration to config.json
		delimiter := GenerateHeredocDelimiterFromSeed("SAFE_OUTPUTS_CONFIG", workflowData.FrontmatterHash)
		if safeOutputConfig != "" {
			if len(configSecrets) > 0 {
				// Replace ${{ secrets.X }} with ${X} and use unquoted heredoc so the
				// shell expands the env var references we set above.
				sanitizedConfig := safeOutputConfig
				for varName, secretExpr := range configSecrets {
					sanitizedConfig = strings.ReplaceAll(sanitizedConfig, secretExpr, "${"+varName+"}")
				}
				yaml.WriteString("          cat > ${RUNNER_TEMP}/gh-aw/safeoutputs/config.json << " + delimiter + "\n")
				yaml.WriteString("          " + sanitizedConfig + "\n")
				yaml.WriteString("          " + delimiter + "\n")
			} else {
				yaml.WriteString("          cat > ${RUNNER_TEMP}/gh-aw/safeoutputs/config.json << '" + delimiter + "'\n")
				yaml.WriteString("          " + safeOutputConfig + "\n")
				yaml.WriteString("          " + delimiter + "\n")
			}
		}

		// Step 1b: Write tools_meta.json and validation.json in a SEPARATE step.
		// tools_meta.json replaces the large inlined tools.json heredoc: it contains
		// only the workflow-specific customisations (description suffixes, repo params,
		// dynamic tools). At runtime, generate_safe_outputs_tools.cjs combines this
		// with the source safe_outputs_tools.json from the actions folder to produce
		// the final ${RUNNER_TEMP}/gh-aw/safeoutputs/tools.json.
		//
		// Keeping this in a separate run: block ensures it never combines with
		// expression-containing content to exceed GitHub Actions' 21,000-character limit.

		// Generate tools_meta.json: small file with description suffixes, repo params,
		// and dynamic tools. The large static tool definitions are loaded at runtime
		// from the actions folder by generate_safe_outputs_tools.cjs.
		toolsMetaJSON, err := generateToolsMetaJSON(workflowData, c.markdownPath)
		if err != nil {
			mcpSetupGeneratorLog.Printf("Error generating tools meta JSON: %v", err)
			// Fall back to empty meta on error
			toolsMetaJSON = `{"description_suffixes":{},"repo_params":{},"dynamic_tools":[]}`
		}

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
			mcpSetupGeneratorLog.Printf("CRITICAL: Error generating validation config JSON: %v - validation will not work correctly", err)
			validationConfigJSON = "{}"
		}

		yaml.WriteString("      - name: Write Safe Outputs Tools\n")
		yaml.WriteString("        run: |\n")

		toolsMetaDelimiter := GenerateHeredocDelimiterFromSeed("SAFE_OUTPUTS_TOOLS_META", workflowData.FrontmatterHash)
		yaml.WriteString("          cat > ${RUNNER_TEMP}/gh-aw/safeoutputs/tools_meta.json << '" + toolsMetaDelimiter + "'\n")
		// Write each line of the compact meta JSON with proper YAML indentation
		for line := range strings.SplitSeq(toolsMetaJSON, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          " + toolsMetaDelimiter + "\n")

		validationDelimiter := GenerateHeredocDelimiterFromSeed("SAFE_OUTPUTS_VALIDATION", workflowData.FrontmatterHash)
		yaml.WriteString("          cat > ${RUNNER_TEMP}/gh-aw/safeoutputs/validation.json << '" + validationDelimiter + "'\n")
		// Write each line of the indented JSON with proper YAML indentation
		for line := range strings.SplitSeq(validationConfigJSON, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          " + validationDelimiter + "\n")

		// Generate the final tools.json at runtime from the source file in the actions folder.
		// generate_safe_outputs_tools.cjs reads safe_outputs_tools.json (deployed by actions/setup),
		// applies the meta overrides from tools_meta.json, and writes tools.json.
		yaml.WriteString("          node ${RUNNER_TEMP}/gh-aw/actions/generate_safe_outputs_tools.cjs\n")

		// Note: The MCP server entry point (mcp-server.cjs) is now copied by actions/setup
		// from safe-outputs-mcp-server.cjs - no need to generate it here

		// Step 2: Generate API key and choose port for HTTP server
		yaml.WriteString("      - name: Generate Safe Outputs MCP Server Config\n")
		yaml.WriteString("        id: safe-outputs-config\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Generate a secure random API key (360 bits of entropy, 40+ chars)\n")
		yaml.WriteString("          # Mask immediately to prevent timing vulnerabilities\n")
		yaml.WriteString("          API_KEY=$(openssl rand -base64 45 | tr -d '/+=')\n")
		yaml.WriteString("          echo \"::add-mask::${API_KEY}\"\n")
		yaml.WriteString("          \n")
		fmt.Fprintf(yaml, "          PORT=%d\n", constants.DefaultMCPInspectorPort)
		yaml.WriteString("          \n")
		yaml.WriteString("          # Set outputs for next steps\n")
		yaml.WriteString("          {\n")
		yaml.WriteString("            echo \"safe_outputs_api_key=${API_KEY}\"\n")
		yaml.WriteString("            echo \"safe_outputs_port=${PORT}\"\n")
		yaml.WriteString("          } >> \"$GITHUB_OUTPUT\"\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          echo \"Safe Outputs MCP server will run on port ${PORT}\"\n")
		yaml.WriteString("          \n")

		// Step 3: Start the HTTP server in the background
		yaml.WriteString("      - name: Start Safe Outputs MCP HTTP Server\n")
		yaml.WriteString("        id: safe-outputs-start\n")

		// Add env block with step outputs
		yaml.WriteString("        env:\n")
		yaml.WriteString("          DEBUG: '*'\n")
		yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}\n")
		yaml.WriteString("          GH_AW_SAFE_OUTPUTS_PORT: ${{ steps.safe-outputs-config.outputs.safe_outputs_port }}\n")
		yaml.WriteString("          GH_AW_SAFE_OUTPUTS_API_KEY: ${{ steps.safe-outputs-config.outputs.safe_outputs_api_key }}\n")
		yaml.WriteString("          GH_AW_SAFE_OUTPUTS_TOOLS_PATH: ${{ runner.temp }}/gh-aw/safeoutputs/tools.json\n")
		yaml.WriteString("          GH_AW_SAFE_OUTPUTS_CONFIG_PATH: ${{ runner.temp }}/gh-aw/safeoutputs/config.json\n")
		yaml.WriteString("          GH_AW_MCP_LOG_DIR: /tmp/gh-aw/mcp-logs/safeoutputs\n")

		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Environment variables are set above to prevent template injection\n")
		yaml.WriteString("          export DEBUG\n")
		yaml.WriteString("          export GH_AW_SAFE_OUTPUTS\n")
		yaml.WriteString("          export GH_AW_SAFE_OUTPUTS_PORT\n")
		yaml.WriteString("          export GH_AW_SAFE_OUTPUTS_API_KEY\n")
		yaml.WriteString("          export GH_AW_SAFE_OUTPUTS_TOOLS_PATH\n")
		yaml.WriteString("          export GH_AW_SAFE_OUTPUTS_CONFIG_PATH\n")
		yaml.WriteString("          export GH_AW_MCP_LOG_DIR\n")
		yaml.WriteString("          \n")

		// Call the bundled shell script to start the server
		yaml.WriteString("          bash ${RUNNER_TEMP}/gh-aw/actions/start_safe_outputs_server.sh\n")
		yaml.WriteString("          \n")
	}

	// Write mcp-scripts MCP server if configured and feature flag is enabled
	// For stdio mode, we only write the files but don't start the HTTP server
	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		// Step 1: Write config files (JavaScript files are now copied by actions/setup)
		yaml.WriteString("      - name: Setup MCP Scripts Config\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          mkdir -p ${RUNNER_TEMP}/gh-aw/mcp-scripts/logs\n")

		// Generate the tools.json configuration file
		toolsJSON := GenerateMCPScriptsToolsConfig(workflowData.MCPScripts)
		toolsDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_TOOLS", workflowData.FrontmatterHash)
		yaml.WriteString("          cat > ${RUNNER_TEMP}/gh-aw/mcp-scripts/tools.json << '" + toolsDelimiter + "'\n")
		for line := range strings.SplitSeq(toolsJSON, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          " + toolsDelimiter + "\n")

		// Generate the MCP server entry point
		mcpScriptsMCPServer := GenerateMCPScriptsMCPServerScript(workflowData.MCPScripts)
		serverDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_SERVER", workflowData.FrontmatterHash)
		yaml.WriteString("          cat > ${RUNNER_TEMP}/gh-aw/mcp-scripts/mcp-server.cjs << '" + serverDelimiter + "'\n")
		for _, line := range FormatJavaScriptForYAML(mcpScriptsMCPServer) {
			yaml.WriteString(line)
		}
		yaml.WriteString("          " + serverDelimiter + "\n")
		yaml.WriteString("          chmod +x ${RUNNER_TEMP}/gh-aw/mcp-scripts/mcp-server.cjs\n")
		yaml.WriteString("          \n")

		// Step 2: Generate tool files (js/py/sh)
		yaml.WriteString("      - name: Setup MCP Scripts Tool Files\n")
		yaml.WriteString("        run: |\n")

		// Generate individual tool files (sorted by name for stable code generation)
		mcpScriptToolNames := sliceutil.MapToSlice(workflowData.MCPScripts.Tools)
		sort.Strings(mcpScriptToolNames)

		for _, toolName := range mcpScriptToolNames {
			toolConfig := workflowData.MCPScripts.Tools[toolName]
			if toolConfig.Script != "" {
				// JavaScript tool
				toolScript := GenerateMCPScriptJavaScriptToolScript(toolConfig)
				jsDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_JS_"+strings.ToUpper(toolName), workflowData.FrontmatterHash)
				fmt.Fprintf(yaml, "          cat > ${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.cjs << '%s'\n", toolName, jsDelimiter)
				for _, line := range FormatJavaScriptForYAML(toolScript) {
					yaml.WriteString(line)
				}
				fmt.Fprintf(yaml, "          %s\n", jsDelimiter)
			} else if toolConfig.Run != "" {
				// Shell script tool
				toolScript := GenerateMCPScriptShellToolScript(toolConfig)
				shDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_SH_"+strings.ToUpper(toolName), workflowData.FrontmatterHash)
				fmt.Fprintf(yaml, "          cat > ${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.sh << '%s'\n", toolName, shDelimiter)
				for line := range strings.SplitSeq(toolScript, "\n") {
					yaml.WriteString("          " + line + "\n")
				}
				fmt.Fprintf(yaml, "          %s\n", shDelimiter)
				fmt.Fprintf(yaml, "          chmod +x ${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.sh\n", toolName)
			} else if toolConfig.Py != "" {
				// Python script tool
				toolScript := GenerateMCPScriptPythonToolScript(toolConfig)
				pyDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_PY_"+strings.ToUpper(toolName), workflowData.FrontmatterHash)
				fmt.Fprintf(yaml, "          cat > ${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.py << '%s'\n", toolName, pyDelimiter)
				for line := range strings.SplitSeq(toolScript, "\n") {
					yaml.WriteString("          " + line + "\n")
				}
				fmt.Fprintf(yaml, "          %s\n", pyDelimiter)
				fmt.Fprintf(yaml, "          chmod +x ${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.py\n", toolName)
			} else if toolConfig.Go != "" {
				// Go script tool
				toolScript := GenerateMCPScriptGoToolScript(toolConfig)
				goDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_GO_"+strings.ToUpper(toolName), workflowData.FrontmatterHash)
				fmt.Fprintf(yaml, "          cat > ${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.go << '%s'\n", toolName, goDelimiter)
				for line := range strings.SplitSeq(toolScript, "\n") {
					yaml.WriteString("          " + line + "\n")
				}
				fmt.Fprintf(yaml, "          %s\n", goDelimiter)
			}
		}
		yaml.WriteString("          \n")

		// Step 3: Generate API key and choose port for HTTP server
		yaml.WriteString("      - name: Generate MCP Scripts Server Config\n")
		yaml.WriteString("        id: mcp-scripts-config\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Generate a secure random API key (360 bits of entropy, 40+ chars)\n")
		yaml.WriteString("          # Mask immediately to prevent timing vulnerabilities\n")
		yaml.WriteString("          API_KEY=$(openssl rand -base64 45 | tr -d '/+=')\n")
		yaml.WriteString("          echo \"::add-mask::${API_KEY}\"\n")
		yaml.WriteString("          \n")
		fmt.Fprintf(yaml, "          PORT=%d\n", constants.DefaultMCPServerPort)
		yaml.WriteString("          \n")
		yaml.WriteString("          # Set outputs for next steps\n")
		yaml.WriteString("          {\n")
		yaml.WriteString("            echo \"mcp_scripts_api_key=${API_KEY}\"\n")
		yaml.WriteString("            echo \"mcp_scripts_port=${PORT}\"\n")
		yaml.WriteString("          } >> \"$GITHUB_OUTPUT\"\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          echo \"MCP Scripts server will run on port ${PORT}\"\n")
		yaml.WriteString("          \n")

		// Step 4: Start the HTTP server in the background
		yaml.WriteString("      - name: Start MCP Scripts HTTP Server\n")
		yaml.WriteString("        id: mcp-scripts-start\n")

		// Add env block with step outputs and tool-specific secrets
		// Security: Pass step outputs through environment variables to prevent template injection
		yaml.WriteString("        env:\n")
		yaml.WriteString("          DEBUG: '*'\n")
		yaml.WriteString("          GH_AW_MCP_SCRIPTS_PORT: ${{ steps.mcp-scripts-config.outputs.mcp_scripts_port }}\n")
		yaml.WriteString("          GH_AW_MCP_SCRIPTS_API_KEY: ${{ steps.mcp-scripts-config.outputs.mcp_scripts_api_key }}\n")

		mcpScriptsSecrets := collectMCPScriptsSecrets(workflowData.MCPScripts)
		if len(mcpScriptsSecrets) > 0 {
			// Sort env var names for consistent output - using functional helper
			envVarNames := sliceutil.MapToSlice(mcpScriptsSecrets)
			sort.Strings(envVarNames)

			for _, envVarName := range envVarNames {
				secretExpr := mcpScriptsSecrets[envVarName]
				fmt.Fprintf(yaml, "          %s: %s\n", envVarName, secretExpr)
			}
		}

		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Environment variables are set above to prevent template injection\n")
		yaml.WriteString("          export DEBUG\n")
		yaml.WriteString("          export GH_AW_MCP_SCRIPTS_PORT\n")
		yaml.WriteString("          export GH_AW_MCP_SCRIPTS_API_KEY\n")
		yaml.WriteString("          \n")

		// Call the bundled shell script to start the server
		yaml.WriteString("          bash ${RUNNER_TEMP}/gh-aw/actions/start_mcp_scripts_server.sh\n")
		yaml.WriteString("          \n")
	}

	// Start the qmd MCP HTTP server natively if qmd is configured.
	// qmd must run on the host VM (not in Docker) because node-llama-cpp compiles
	// platform-native binaries. HTTP mode keeps MCP traffic on TCP, separate from stdout.
	if workflowData != nil && workflowData.QmdConfig != nil {
		yaml.WriteString(generateQmdStartStep(workflowData.QmdConfig))
	}

	// The MCP gateway is always enabled, even when agent sandbox is disabled
	// Use the engine's RenderMCPConfig method
	yaml.WriteString("      - name: Start MCP Gateway\n")
	yaml.WriteString("        id: start-mcp-gateway\n")

	// Collect all MCP-related environment variables using centralized helper
	mcpEnvVars := collectMCPEnvironmentVariables(tools, mcpTools, workflowData, hasAgenticWorkflows)

	// Add env block if any environment variables are needed
	if len(mcpEnvVars) > 0 {
		yaml.WriteString("        env:\n")

		// Sort environment variable names for consistent output
		// Using functional helper to extract map keys
		envVarNames := sliceutil.MapToSlice(mcpEnvVars)
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
	// Pre-create the playwright output directory on the host so the Docker container
	// can write screenshots to the mounted volume path without ENOENT errors
	if slices.Contains(mcpTools, "playwright") {
		yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-logs/playwright\n")
	}

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

	yaml.WriteString("          \n")
	yaml.WriteString("          # Export gateway environment variables for MCP config and gateway script\n")
	yaml.WriteString("          export MCP_GATEWAY_PORT=\"" + strconv.Itoa(port) + "\"\n")
	yaml.WriteString("          export MCP_GATEWAY_DOMAIN=\"" + domain + "\"\n")

	// Generate API key with proper error handling (avoid SC2155)
	// Mask immediately after generation to prevent timing vulnerabilities
	if apiKey == "" {
		yaml.WriteString("          MCP_GATEWAY_API_KEY=$(openssl rand -base64 45 | tr -d '/+=')\n")
		yaml.WriteString("          echo \"::add-mask::${MCP_GATEWAY_API_KEY}\"\n")
		yaml.WriteString("          export MCP_GATEWAY_API_KEY\n")
	} else {
		yaml.WriteString("          export MCP_GATEWAY_API_KEY=\"" + apiKey + "\"\n")
		yaml.WriteString("          echo \"::add-mask::${MCP_GATEWAY_API_KEY}\"\n")
	}

	// Export payload directory and ensure it exists
	payloadDir := gatewayConfig.PayloadDir
	if payloadDir == "" {
		payloadDir = constants.DefaultMCPGatewayPayloadDir
	}
	yaml.WriteString("          export MCP_GATEWAY_PAYLOAD_DIR=\"" + payloadDir + "\"\n")
	yaml.WriteString("          mkdir -p \"${MCP_GATEWAY_PAYLOAD_DIR}\"\n")

	// Export payload path prefix if configured
	payloadPathPrefix := gatewayConfig.PayloadPathPrefix
	if payloadPathPrefix != "" {
		yaml.WriteString("          export MCP_GATEWAY_PAYLOAD_PATH_PREFIX=\"" + payloadPathPrefix + "\"\n")
	}

	// Export payload size threshold (use default if not configured)
	payloadSizeThreshold := gatewayConfig.PayloadSizeThreshold
	if payloadSizeThreshold == 0 {
		payloadSizeThreshold = constants.DefaultMCPGatewayPayloadSizeThreshold
	}
	yaml.WriteString("          export MCP_GATEWAY_PAYLOAD_SIZE_THRESHOLD=\"" + strconv.Itoa(payloadSizeThreshold) + "\"\n")

	yaml.WriteString("          export DEBUG=\"*\"\n")
	yaml.WriteString("          \n")

	// Export engine type
	yaml.WriteString("          export GH_AW_ENGINE=\"" + engine.GetID() + "\"\n")

	// For Copilot engine with GitHub remote MCP, export GITHUB_PERSONAL_ACCESS_TOKEN
	// This is needed because the MCP gateway validates ${VAR} references in headers at config load time
	// and the Copilot MCP config uses ${GITHUB_PERSONAL_ACCESS_TOKEN} in the Authorization header
	githubTool, hasGitHub := tools["github"]
	if hasGitHub && getGitHubType(githubTool) == "remote" && engine.GetID() == "copilot" {
		yaml.WriteString("          export GITHUB_PERSONAL_ACCESS_TOKEN=\"$GITHUB_MCP_SERVER_TOKEN\"\n")
	}

	// Add user-configured environment variables
	if len(gatewayConfig.Env) > 0 {
		// Using functional helper to extract map keys
		envVarNames := sliceutil.MapToSlice(gatewayConfig.Env)
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

	var containerCmd strings.Builder
	containerCmd.WriteString("docker run -i --rm --network host")
	containerCmd.WriteString(" -v /var/run/docker.sock:/var/run/docker.sock") // Enable docker-in-docker for MCP gateway
	// Pass required gateway environment variables
	containerCmd.WriteString(" -e MCP_GATEWAY_PORT")
	containerCmd.WriteString(" -e MCP_GATEWAY_DOMAIN")
	containerCmd.WriteString(" -e MCP_GATEWAY_API_KEY")
	containerCmd.WriteString(" -e MCP_GATEWAY_PAYLOAD_DIR")
	if payloadPathPrefix != "" {
		containerCmd.WriteString(" -e MCP_GATEWAY_PAYLOAD_PATH_PREFIX")
	}
	containerCmd.WriteString(" -e MCP_GATEWAY_PAYLOAD_SIZE_THRESHOLD")
	containerCmd.WriteString(" -e DEBUG")
	// Pass environment variables that MCP servers reference in their config
	// These are needed because awmg v0.0.12+ validates and resolves ${VAR} patterns at config load time
	// Environment variables used by MCP gateway
	containerCmd.WriteString(" -e MCP_GATEWAY_LOG_DIR")
	// Environment variables used by safeoutputs MCP server
	containerCmd.WriteString(" -e GH_AW_MCP_LOG_DIR")
	containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS")
	containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_CONFIG_PATH")
	containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_TOOLS_PATH")
	containerCmd.WriteString(" -e GH_AW_ASSETS_BRANCH")
	containerCmd.WriteString(" -e GH_AW_ASSETS_MAX_SIZE_KB")
	containerCmd.WriteString(" -e GH_AW_ASSETS_ALLOWED_EXTS")
	containerCmd.WriteString(" -e DEFAULT_BRANCH")
	// Environment variables used by GitHub MCP server
	containerCmd.WriteString(" -e GITHUB_MCP_SERVER_TOKEN")
	// For Copilot engine with GitHub remote MCP, also pass GITHUB_PERSONAL_ACCESS_TOKEN
	// This allows the gateway to expand ${GITHUB_PERSONAL_ACCESS_TOKEN} references in headers
	if hasGitHub && getGitHubType(githubTool) == "remote" && engine.GetID() == "copilot" {
		containerCmd.WriteString(" -e GITHUB_PERSONAL_ACCESS_TOKEN")
	}
	// Automatic guard policy env vars (set from determine-automatic-lockdown step outputs)
	containerCmd.WriteString(" -e GITHUB_MCP_GUARD_MIN_INTEGRITY")
	containerCmd.WriteString(" -e GITHUB_MCP_GUARD_REPOS")
	// Standard GitHub Actions environment variables (repository context)
	containerCmd.WriteString(" -e GITHUB_REPOSITORY")
	containerCmd.WriteString(" -e GITHUB_SERVER_URL")
	containerCmd.WriteString(" -e GITHUB_SHA")
	containerCmd.WriteString(" -e GITHUB_WORKSPACE")
	containerCmd.WriteString(" -e GITHUB_TOKEN")
	// GitHub Actions run context
	containerCmd.WriteString(" -e GITHUB_RUN_ID")
	containerCmd.WriteString(" -e GITHUB_RUN_NUMBER")
	containerCmd.WriteString(" -e GITHUB_RUN_ATTEMPT")
	containerCmd.WriteString(" -e GITHUB_JOB")
	containerCmd.WriteString(" -e GITHUB_ACTION")
	// GitHub Actions event context
	containerCmd.WriteString(" -e GITHUB_EVENT_NAME")
	containerCmd.WriteString(" -e GITHUB_EVENT_PATH")
	// GitHub Actions actor context
	containerCmd.WriteString(" -e GITHUB_ACTOR")
	containerCmd.WriteString(" -e GITHUB_ACTOR_ID")
	containerCmd.WriteString(" -e GITHUB_TRIGGERING_ACTOR")
	// GitHub Actions workflow context
	containerCmd.WriteString(" -e GITHUB_WORKFLOW")
	containerCmd.WriteString(" -e GITHUB_WORKFLOW_REF")
	containerCmd.WriteString(" -e GITHUB_WORKFLOW_SHA")
	// GitHub Actions ref context
	containerCmd.WriteString(" -e GITHUB_REF")
	containerCmd.WriteString(" -e GITHUB_REF_NAME")
	containerCmd.WriteString(" -e GITHUB_REF_TYPE")
	containerCmd.WriteString(" -e GITHUB_HEAD_REF")
	containerCmd.WriteString(" -e GITHUB_BASE_REF")
	// Environment variables used by safeinputs MCP server
	// Only add if mcp-scripts is actually enabled (has tools configured)
	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		containerCmd.WriteString(" -e GH_AW_MCP_SCRIPTS_PORT")
		containerCmd.WriteString(" -e GH_AW_MCP_SCRIPTS_API_KEY")
	}
	// Environment variables used by safeoutputs MCP server
	// Only add if safe-outputs is actually enabled (has tools configured)
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_PORT")
		containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_API_KEY")
	}
	// OpenTelemetry trace correlation env vars - pass to gateway so it can expand the
	// ${GITHUB_AW_OTEL_TRACE_ID} and ${GITHUB_AW_OTEL_PARENT_SPAN_ID} references written
	// directly in the opentelemetry config block (spec §4.1.3.6). These are set at
	// runtime via GITHUB_ENV by actions/setup and cannot be known at compile time.
	// The endpoint and headers are written as literal values in the config, so their
	// corresponding env vars (OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_HEADERS)
	// are not passed to the gateway container.
	if workflowData.OTLPEndpoint != "" {
		containerCmd.WriteString(" -e GITHUB_AW_OTEL_TRACE_ID")
		containerCmd.WriteString(" -e GITHUB_AW_OTEL_PARENT_SPAN_ID")
	}
	if len(gatewayConfig.Env) > 0 {
		// Using functional helper to extract map keys
		envVarNames := sliceutil.MapToSlice(gatewayConfig.Env)
		sort.Strings(envVarNames)
		for _, envVarName := range envVarNames {
			containerCmd.WriteString(" -e " + envVarName)
		}
	}

	// Add environment variables collected from HTTP MCP servers (e.g., TAVILY_API_KEY)
	// These are needed for the gateway to resolve ${VAR} references in MCP server configs
	if len(mcpEnvVars) > 0 {
		// Get list of environment variable names already added to avoid duplicates
		addedEnvVars := make(map[string]bool)

		// Mark standard environment variables as already added
		standardEnvVars := []string{
			"MCP_GATEWAY_PORT", "MCP_GATEWAY_DOMAIN", "MCP_GATEWAY_API_KEY", "MCP_GATEWAY_PAYLOAD_DIR", "DEBUG",
			"MCP_GATEWAY_LOG_DIR", "GH_AW_MCP_LOG_DIR", "GH_AW_SAFE_OUTPUTS",
			"GH_AW_SAFE_OUTPUTS_CONFIG_PATH", "GH_AW_SAFE_OUTPUTS_TOOLS_PATH",
			"GH_AW_ASSETS_BRANCH", "GH_AW_ASSETS_MAX_SIZE_KB", "GH_AW_ASSETS_ALLOWED_EXTS",
			"DEFAULT_BRANCH", "GITHUB_MCP_SERVER_TOKEN", "GITHUB_MCP_GUARD_MIN_INTEGRITY", "GITHUB_MCP_GUARD_REPOS",
			"GITHUB_REPOSITORY", "GITHUB_SERVER_URL", "GITHUB_SHA", "GITHUB_WORKSPACE",
			"GITHUB_TOKEN", "GITHUB_RUN_ID", "GITHUB_RUN_NUMBER", "GITHUB_RUN_ATTEMPT",
			"GITHUB_JOB", "GITHUB_ACTION", "GITHUB_EVENT_NAME", "GITHUB_EVENT_PATH",
			"GITHUB_ACTOR", "GITHUB_ACTOR_ID", "GITHUB_TRIGGERING_ACTOR",
			"GITHUB_WORKFLOW", "GITHUB_WORKFLOW_REF", "GITHUB_WORKFLOW_SHA",
			"GITHUB_REF", "GITHUB_REF_NAME", "GITHUB_REF_TYPE", "GITHUB_HEAD_REF", "GITHUB_BASE_REF",
		}
		for _, envVar := range standardEnvVars {
			addedEnvVars[envVar] = true
		}

		// Mark conditionally added environment variables
		if hasGitHub && getGitHubType(githubTool) == "remote" && engine.GetID() == "copilot" {
			addedEnvVars["GITHUB_PERSONAL_ACCESS_TOKEN"] = true
		}
		if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
			addedEnvVars["GH_AW_MCP_SCRIPTS_PORT"] = true
			addedEnvVars["GH_AW_MCP_SCRIPTS_API_KEY"] = true
		}
		if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
			addedEnvVars["GH_AW_SAFE_OUTPUTS_PORT"] = true
			addedEnvVars["GH_AW_SAFE_OUTPUTS_API_KEY"] = true
		}
		if workflowData.OTLPEndpoint != "" {
			addedEnvVars["GITHUB_AW_OTEL_TRACE_ID"] = true
			addedEnvVars["GITHUB_AW_OTEL_PARENT_SPAN_ID"] = true
		}

		// Mark gateway config environment variables as added
		if len(gatewayConfig.Env) > 0 {
			for envVarName := range gatewayConfig.Env {
				addedEnvVars[envVarName] = true
			}
		}

		// Add remaining environment variables from mcpEnvVars
		var envVarNames []string
		for envVarName := range mcpEnvVars {
			if !addedEnvVars[envVarName] {
				envVarNames = append(envVarNames, envVarName)
			}
		}
		sort.Strings(envVarNames)

		for _, envVarName := range envVarNames {
			containerCmd.WriteString(" -e " + envVarName)
		}

		if mcpSetupGeneratorLog.Enabled() && len(envVarNames) > 0 {
			mcpSetupGeneratorLog.Printf("Added %d HTTP MCP environment variables to gateway container: %v", len(envVarNames), envVarNames)
		}
	}

	// Add volume mounts
	// First, add the payload directory mount (rw for both agent and gateway)
	if payloadDir != "" {
		containerCmd.WriteString(" -v " + payloadDir + ":" + payloadDir + ":rw")
	}

	// Then add user-configured mounts
	if len(gatewayConfig.Mounts) > 0 {
		for _, mount := range gatewayConfig.Mounts {
			containerCmd.WriteString(" -v " + mount)
		}
	}

	// Add entrypoint override if specified
	if gatewayConfig.Entrypoint != "" {
		containerCmd.WriteString(" --entrypoint " + shellEscapeArg(gatewayConfig.Entrypoint))
	}

	containerCmd.WriteString(" " + containerImage)

	if len(gatewayConfig.EntrypointArgs) > 0 {
		for _, arg := range gatewayConfig.EntrypointArgs {
			containerCmd.WriteString(" " + shellEscapeArg(arg))
		}
	}

	if len(gatewayConfig.Args) > 0 {
		for _, arg := range gatewayConfig.Args {
			containerCmd.WriteString(" " + shellEscapeArg(arg))
		}
	}

	// Build the export command with proper quoting that allows variable expansion
	// We need to break out of quotes for ${GITHUB_WORKSPACE} variables
	cmdWithExpandableVars := buildDockerCommandWithExpandableVars(containerCmd.String())
	yaml.WriteString("          export MCP_GATEWAY_DOCKER_COMMAND=" + cmdWithExpandableVars + "\n")
	yaml.WriteString("          \n")

	// Render MCP config - this will pipe directly to the gateway script
	// The MCP gateway is always enabled, even when agent sandbox is disabled
	return engine.RenderMCPConfig(yaml, tools, mcpTools, workflowData)
}
