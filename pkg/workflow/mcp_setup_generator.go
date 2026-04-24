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
//   - Start MCP Gateway (default port 8080)
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
	if workflowData == nil {
		return nil
	}

	mcpTools := collectMCPTools(workflowData)

	// Populate dispatch-workflow file mappings before generating config
	// This ensures workflow_files is available in the config.json
	populateDispatchWorkflowFiles(workflowData, c.markdownPath)

	// Populate call-workflow file mappings before generating config
	// This ensures workflow_files is available in the config.json
	populateCallWorkflowFiles(workflowData, c.markdownPath)

	safeOutputConfig, err := generateSafeOutputsConfigIfEnabled(workflowData)
	if err != nil {
		return fmt.Errorf("safe outputs setup preparation failed: %w", err)
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

	// If no MCP tools, skip setup unless the engine still needs MCP gateway/config bootstrap.
	// Codex with AWF firewall enabled requires MCP config generation to set its OpenAI proxy
	// provider, even when no MCP tools are configured (e.g. threat-detection jobs).
	needsSetupWithoutMCPTools := len(mcpTools) == 0 && engine.GetID() == "codex" && isFirewallEnabled(workflowData)
	if len(mcpTools) == 0 && !needsSetupWithoutMCPTools {
		mcpSetupGeneratorLog.Print("No MCP tools configured, skipping MCP setup")
		return nil
	}

	hasAgenticWorkflows := slices.Contains(mcpTools, "agentic-workflows")
	hasGhAwImport := hasGhAwSharedImport(workflowData)
	generateAgenticWorkflowsInstallStep(yaml, hasAgenticWorkflows, hasGhAwImport)

	generateSafeOutputsSetup(c, yaml, safeOutputConfig, workflowData)
	if err := generateMCPScriptsSetup(yaml, workflowData); err != nil {
		return fmt.Errorf("failed to generate mcp-scripts setup YAML: %w", err)
	}
	return generateMCPGatewaySetup(yaml, tools, mcpTools, engine, workflowData, hasAgenticWorkflows)
}

func collectMCPTools(workflowData *WorkflowData) []string {
	var mcpTools []string
	for toolName, toolValue := range workflowData.Tools {
		if toolValue == false {
			continue
		}
		if toolName == "github" && isGitHubCLIModeEnabled(workflowData) {
			mcpSetupGeneratorLog.Print("Skipping GitHub MCP server registration: tools.github.mode is gh-proxy")
			continue
		}
		if toolName == "github" || toolName == "playwright" || toolName == "cache-memory" || toolName == "agentic-workflows" {
			mcpTools = append(mcpTools, toolName)
			continue
		}
		if mcpConfig, ok := toolValue.(map[string]any); ok {
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				mcpTools = append(mcpTools, toolName)
			}
		}
	}
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		mcpTools = append(mcpTools, "safe-outputs")
	}
	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		mcpTools = append(mcpTools, "mcp-scripts")
	}
	return mcpTools
}

func generateSafeOutputsConfigIfEnabled(workflowData *WorkflowData) (string, error) {
	if !HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		return "", nil
	}
	safeOutputConfig, err := generateSafeOutputsConfig(workflowData)
	if err != nil {
		return "", fmt.Errorf("failed to generate safe outputs config: %w", err)
	}
	return safeOutputConfig, nil
}

func hasGhAwSharedImport(workflowData *WorkflowData) bool {
	for _, importPath := range workflowData.ImportedFiles {
		if strings.Contains(importPath, "shared/mcp/gh-aw.md") {
			return true
		}
	}
	return false
}

func generateAgenticWorkflowsInstallStep(yaml *strings.Builder, hasAgenticWorkflows bool, hasGhAwImport bool) {
	if !hasAgenticWorkflows {
		return
	}
	if hasGhAwImport {
		mcpSetupGeneratorLog.Print("Skipping gh-aw extension installation step (provided by shared/mcp/gh-aw.md import)")
		return
	}
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
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw\"\n")
	yaml.WriteString("          GH_AW_BIN=\"\"\n")
	yaml.WriteString("          GH_AW_BIN=$(command -v gh-aw 2>/dev/null) || true\n")
	yaml.WriteString("          if [ -z \"$GH_AW_BIN\" ]; then\n")
	yaml.WriteString("            GH_AW_BIN=$(find \"${HOME}/.local/share/gh/extensions/gh-aw\" -name 'gh-aw' -type f 2>/dev/null | head -1) || true\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          if [ -n \"$GH_AW_BIN\" ] && [ -f \"$GH_AW_BIN\" ]; then\n")
	yaml.WriteString("            cp \"$GH_AW_BIN\" \"${RUNNER_TEMP}/gh-aw/gh-aw\"\n")
	yaml.WriteString("            chmod +x \"${RUNNER_TEMP}/gh-aw/gh-aw\"\n")
	yaml.WriteString("            echo \"Copied gh-aw binary to ${RUNNER_TEMP}/gh-aw/gh-aw\"\n")
	yaml.WriteString("          else\n")
	yaml.WriteString("            echo \"::error::Failed to find gh-aw binary for MCP server\"\n")
	yaml.WriteString("            exit 1\n")
	yaml.WriteString("          fi\n")
}

func generateSafeOutputsSetup(c *Compiler, yaml *strings.Builder, safeOutputConfig string, workflowData *WorkflowData) {
	if !HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		return
	}
	yaml.WriteString("      - name: Write Safe Outputs Config\n")
	configSecrets := ExtractSecretsFromValue(safeOutputConfig)
	configContextVars := ExtractGitHubContextExpressionsFromValue(safeOutputConfig)
	hasEnvVars := len(configSecrets) > 0 || len(configContextVars) > 0
	if hasEnvVars {
		yaml.WriteString("        env:\n")
		envKeys := make([]string, 0, len(configSecrets)+len(configContextVars))
		envValues := make(map[string]string, len(configSecrets)+len(configContextVars))
		for k, v := range configContextVars {
			envKeys = append(envKeys, k)
			envValues[k] = v
		}
		for k, v := range configSecrets {
			if _, exists := envValues[k]; !exists {
				envKeys = append(envKeys, k)
			}
			envValues[k] = v
		}
		sort.Strings(envKeys)
		for _, varName := range envKeys {
			yaml.WriteString("          " + varName + ": " + envValues[varName] + "\n")
		}
	}
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/safeoutputs\"\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/safeoutputs\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-logs/safeoutputs\n")
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.UploadArtifact != nil {
		yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts\"\n")
	}

	delimiter := GenerateHeredocDelimiterFromSeed("SAFE_OUTPUTS_CONFIG", workflowData.FrontmatterHash)
	if safeOutputConfig != "" {
		if hasEnvVars {
			sanitizedConfig := safeOutputConfig
			for varName, secretExpr := range configSecrets {
				sanitizedConfig = strings.ReplaceAll(sanitizedConfig, secretExpr, "${"+varName+"}")
			}
			for varName, ctxExpr := range configContextVars {
				sanitizedConfig = strings.ReplaceAll(sanitizedConfig, ctxExpr, "${"+varName+"}")
			}
			yaml.WriteString("          cat > \"${RUNNER_TEMP}/gh-aw/safeoutputs/config.json\" << " + delimiter + "\n")
			yaml.WriteString("          " + sanitizedConfig + "\n")
			yaml.WriteString("          " + delimiter + "\n")
		} else {
			yaml.WriteString("          cat > \"${RUNNER_TEMP}/gh-aw/safeoutputs/config.json\" << '" + delimiter + "'\n")
			yaml.WriteString("          " + safeOutputConfig + "\n")
			yaml.WriteString("          " + delimiter + "\n")
		}
	}

	toolsMetaJSON, err := generateToolsMetaJSON(workflowData, c.markdownPath)
	if err != nil {
		mcpSetupGeneratorLog.Printf("Error generating tools meta JSON: %v", err)
		toolsMetaJSON = `{"description_suffixes":{},"repo_params":{},"dynamic_tools":[]}`
	}

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
		mcpSetupGeneratorLog.Printf("CRITICAL: Error generating validation config JSON: %v - validation will not work correctly", err)
		validationConfigJSON = "{}"
	}

	yaml.WriteString("      - name: Write Safe Outputs Tools\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_TOOLS_META_JSON: |\n")
	for line := range strings.SplitSeq(toolsMetaJSON, "\n") {
		yaml.WriteString("            " + line + "\n")
	}
	yaml.WriteString("          GH_AW_VALIDATION_JSON: |\n")
	for line := range strings.SplitSeq(validationConfigJSON, "\n") {
		yaml.WriteString("            " + line + "\n")
	}
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", workflowData))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString(generateGitHubScriptWithRequire("generate_safe_outputs_tools.cjs"))

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

	yaml.WriteString("      - name: Start Safe Outputs MCP HTTP Server\n")
	yaml.WriteString("        id: safe-outputs-start\n")
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
	yaml.WriteString("          bash \"${RUNNER_TEMP}/gh-aw/actions/start_safe_outputs_server.sh\"\n")
	yaml.WriteString("          \n")
}

func generateMCPScriptsSetup(yaml *strings.Builder, workflowData *WorkflowData) error {
	if !IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		return nil
	}
	yaml.WriteString("      - name: Write MCP Scripts Config\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/mcp-scripts/logs\"\n")

	toolsJSON := GenerateMCPScriptsToolsConfig(workflowData.MCPScripts)
	toolsDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_TOOLS", workflowData.FrontmatterHash)
	if err := ValidateHeredocContent(toolsJSON, toolsDelimiter); err != nil {
		return fmt.Errorf("mcp-scripts tools.json: %w", err)
	}
	yaml.WriteString("          cat > \"${RUNNER_TEMP}/gh-aw/mcp-scripts/tools.json\" << '" + toolsDelimiter + "'\n")
	for line := range strings.SplitSeq(toolsJSON, "\n") {
		yaml.WriteString("          " + line + "\n")
	}
	yaml.WriteString("          " + toolsDelimiter + "\n")

	mcpScriptsMCPServer := GenerateMCPScriptsMCPServerScript(workflowData.MCPScripts)
	serverDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_SERVER", workflowData.FrontmatterHash)
	if err := ValidateHeredocContent(mcpScriptsMCPServer, serverDelimiter); err != nil {
		return fmt.Errorf("mcp-scripts mcp-server.cjs: %w", err)
	}
	yaml.WriteString("          cat > \"${RUNNER_TEMP}/gh-aw/mcp-scripts/mcp-server.cjs\" << '" + serverDelimiter + "'\n")
	for _, line := range FormatJavaScriptForYAML(mcpScriptsMCPServer) {
		yaml.WriteString(line)
	}
	yaml.WriteString("          " + serverDelimiter + "\n")
	yaml.WriteString("          chmod +x \"${RUNNER_TEMP}/gh-aw/mcp-scripts/mcp-server.cjs\"\n")
	yaml.WriteString("          \n")

	yaml.WriteString("      - name: Write MCP Scripts Tool Files\n")
	yaml.WriteString("        run: |\n")
	mcpScriptToolNames := sliceutil.MapToSlice(workflowData.MCPScripts.Tools)
	sort.Strings(mcpScriptToolNames)
	for _, toolName := range mcpScriptToolNames {
		toolConfig := workflowData.MCPScripts.Tools[toolName]
		if err := appendMCPScriptToolFile(yaml, workflowData, toolName, toolConfig); err != nil {
			return err
		}
	}
	yaml.WriteString("          \n")
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
	yaml.WriteString("      - name: Start MCP Scripts HTTP Server\n")
	yaml.WriteString("        id: mcp-scripts-start\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          DEBUG: '*'\n")
	yaml.WriteString("          GH_AW_MCP_SCRIPTS_PORT: ${{ steps.mcp-scripts-config.outputs.mcp_scripts_port }}\n")
	yaml.WriteString("          GH_AW_MCP_SCRIPTS_API_KEY: ${{ steps.mcp-scripts-config.outputs.mcp_scripts_api_key }}\n")
	mcpScriptsSecrets := collectMCPScriptsSecrets(workflowData.MCPScripts)
	if len(mcpScriptsSecrets) > 0 {
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
	yaml.WriteString("          bash \"${RUNNER_TEMP}/gh-aw/actions/start_mcp_scripts_server.sh\"\n")
	yaml.WriteString("          \n")
	return nil
}

func appendMCPScriptToolFile(yaml *strings.Builder, workflowData *WorkflowData, toolName string, toolConfig *MCPScriptToolConfig) error {
	if toolConfig.Script != "" {
		toolScript := GenerateMCPScriptJavaScriptToolScript(toolConfig)
		jsDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_JS_"+strings.ToUpper(toolName), workflowData.FrontmatterHash)
		if err := ValidateHeredocContent(toolScript, jsDelimiter); err != nil {
			return fmt.Errorf("mcp-scripts tool %q (js): %w", toolName, err)
		}
		fmt.Fprintf(yaml, "          cat > \"${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.cjs\" << '%s'\n", toolName, jsDelimiter)
		for _, line := range FormatJavaScriptForYAML(toolScript) {
			yaml.WriteString(line)
		}
		fmt.Fprintf(yaml, "          %s\n", jsDelimiter)
		return nil
	}
	if toolConfig.Run != "" {
		toolScript := GenerateMCPScriptShellToolScript(toolConfig)
		shDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_SH_"+strings.ToUpper(toolName), workflowData.FrontmatterHash)
		if err := ValidateHeredocContent(toolScript, shDelimiter); err != nil {
			return fmt.Errorf("mcp-scripts tool %q (sh): %w", toolName, err)
		}
		fmt.Fprintf(yaml, "          cat > \"${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.sh\" << '%s'\n", toolName, shDelimiter)
		for line := range strings.SplitSeq(toolScript, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		fmt.Fprintf(yaml, "          %s\n", shDelimiter)
		fmt.Fprintf(yaml, "          chmod +x \"${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.sh\"\n", toolName)
		return nil
	}
	if toolConfig.Py != "" {
		toolScript := GenerateMCPScriptPythonToolScript(toolConfig)
		pyDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_PY_"+strings.ToUpper(toolName), workflowData.FrontmatterHash)
		if err := ValidateHeredocContent(toolScript, pyDelimiter); err != nil {
			return fmt.Errorf("mcp-scripts tool %q (py): %w", toolName, err)
		}
		fmt.Fprintf(yaml, "          cat > \"${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.py\" << '%s'\n", toolName, pyDelimiter)
		for line := range strings.SplitSeq(toolScript, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		fmt.Fprintf(yaml, "          %s\n", pyDelimiter)
		fmt.Fprintf(yaml, "          chmod +x \"${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.py\"\n", toolName)
		return nil
	}
	if toolConfig.Go != "" {
		toolScript := GenerateMCPScriptGoToolScript(toolConfig)
		goDelimiter := GenerateHeredocDelimiterFromSeed("MCP_SCRIPTS_GO_"+strings.ToUpper(toolName), workflowData.FrontmatterHash)
		if err := ValidateHeredocContent(toolScript, goDelimiter); err != nil {
			return fmt.Errorf("mcp-scripts tool %q (go): %w", toolName, err)
		}
		fmt.Fprintf(yaml, "          cat > \"${RUNNER_TEMP}/gh-aw/mcp-scripts/%s.go\" << '%s'\n", toolName, goDelimiter)
		for line := range strings.SplitSeq(toolScript, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		fmt.Fprintf(yaml, "          %s\n", goDelimiter)
	}
	return nil
}

func generateMCPGatewaySetup(yaml *strings.Builder, tools map[string]any, mcpTools []string, engine CodingAgentEngine, workflowData *WorkflowData, hasAgenticWorkflows bool) error {
	yaml.WriteString("      - name: Start MCP Gateway\n")
	yaml.WriteString("        id: start-mcp-gateway\n")
	mcpEnvVars := collectMCPEnvironmentVariables(tools, mcpTools, workflowData, hasAgenticWorkflows)
	writeMCPGatewayStepEnv(yaml, mcpEnvVars)
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          set -eo pipefail\n")
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/mcp-config\"\n")
	if slices.Contains(mcpTools, "playwright") {
		yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-logs/playwright\n")
		yaml.WriteString("          chmod 777 /tmp/gh-aw/mcp-logs/playwright\n")
	}
	ensureDefaultMCPGatewayConfig(workflowData)
	gatewayConfig := workflowData.SandboxConfig.MCP
	port, domain, payloadDir, payloadPathPrefix, payloadSizeThreshold := resolveMCPGatewayValues(workflowData, gatewayConfig)
	githubTool, hasGitHub := tools["github"]
	writeMCPGatewayExports(yaml, tools, engine, workflowData, gatewayConfig, hasGitHub, githubTool, port, domain, payloadDir, payloadPathPrefix, payloadSizeThreshold)
	containerCmd := buildMCPGatewayContainerCommand(engine, workflowData, gatewayConfig, mcpEnvVars, payloadDir, payloadPathPrefix, hasGitHub, githubTool, tools)
	yaml.WriteString("          MCP_GATEWAY_UID=$(id -u 2>/dev/null || echo '0')\n")
	yaml.WriteString("          MCP_GATEWAY_GID=$(id -g 2>/dev/null || echo '0')\n")
	yaml.WriteString("          DOCKER_SOCK_GID=$(stat -c '%g' /var/run/docker.sock 2>/dev/null || echo '0')\n")
	cmdWithExpandableVars := buildDockerCommandWithExpandableVars(containerCmd)
	yaml.WriteString("          export MCP_GATEWAY_DOCKER_COMMAND=" + cmdWithExpandableVars + "\n")
	yaml.WriteString("          \n")
	return engine.RenderMCPConfig(yaml, tools, mcpTools, workflowData)
}

func writeMCPGatewayStepEnv(yaml *strings.Builder, mcpEnvVars map[string]string) {
	if len(mcpEnvVars) == 0 {
		return
	}
	yaml.WriteString("        env:\n")
	envVarNames := sliceutil.MapToSlice(mcpEnvVars)
	sort.Strings(envVarNames)
	for _, envVarName := range envVarNames {
		fmt.Fprintf(yaml, "          %s: %s\n", envVarName, mcpEnvVars[envVarName])
	}
}

func resolveMCPGatewayValues(workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig) (int, string, string, string, int) {
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
	payloadDir := gatewayConfig.PayloadDir
	if payloadDir == "" {
		payloadDir = constants.DefaultMCPGatewayPayloadDir
	}
	payloadSizeThreshold := gatewayConfig.PayloadSizeThreshold
	if payloadSizeThreshold == 0 {
		payloadSizeThreshold = constants.DefaultMCPGatewayPayloadSizeThreshold
	}
	return port, domain, payloadDir, gatewayConfig.PayloadPathPrefix, payloadSizeThreshold
}

func writeMCPGatewayExports(yaml *strings.Builder, tools map[string]any, engine CodingAgentEngine, workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig, hasGitHub bool, githubTool any, port int, domain string, payloadDir string, payloadPathPrefix string, payloadSizeThreshold int) {
	yaml.WriteString("          \n")
	yaml.WriteString("          # Export gateway environment variables for MCP config and gateway script\n")
	yaml.WriteString("          export MCP_GATEWAY_PORT=\"" + strconv.Itoa(port) + "\"\n")
	yaml.WriteString("          export MCP_GATEWAY_DOMAIN=\"" + domain + "\"\n")
	if gatewayConfig.APIKey == "" {
		yaml.WriteString("          MCP_GATEWAY_API_KEY=$(openssl rand -base64 45 | tr -d '/+=')\n")
		yaml.WriteString("          echo \"::add-mask::${MCP_GATEWAY_API_KEY}\"\n")
		yaml.WriteString("          export MCP_GATEWAY_API_KEY\n")
	} else {
		yaml.WriteString("          export MCP_GATEWAY_API_KEY=\"" + gatewayConfig.APIKey + "\"\n")
		yaml.WriteString("          echo \"::add-mask::${MCP_GATEWAY_API_KEY}\"\n")
	}
	yaml.WriteString("          export MCP_GATEWAY_PAYLOAD_DIR=\"" + payloadDir + "\"\n")
	yaml.WriteString("          mkdir -p \"${MCP_GATEWAY_PAYLOAD_DIR}\"\n")
	if payloadPathPrefix != "" {
		yaml.WriteString("          export MCP_GATEWAY_PAYLOAD_PATH_PREFIX=\"" + payloadPathPrefix + "\"\n")
	}
	yaml.WriteString("          export MCP_GATEWAY_PAYLOAD_SIZE_THRESHOLD=\"" + strconv.Itoa(payloadSizeThreshold) + "\"\n")
	yaml.WriteString("          export DEBUG=\"*\"\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          export GH_AW_ENGINE=\"" + engine.GetID() + "\"\n")
	if cliServers := getMCPCLIExcludeFromAgentConfig(workflowData); len(cliServers) > 0 {
		cliServersJSON, err := json.Marshal(cliServers)
		if err == nil {
			yaml.WriteString("          export GH_AW_MCP_CLI_SERVERS='" + string(cliServersJSON) + "'\n")
			yaml.WriteString("          echo 'GH_AW_MCP_CLI_SERVERS=" + string(cliServersJSON) + "' >> \"$GITHUB_ENV\"\n")
		}
	}
	if hasGitHub && getGitHubType(githubTool) == "remote" && engine.GetID() == "copilot" {
		yaml.WriteString("          export GITHUB_PERSONAL_ACCESS_TOKEN=\"$GITHUB_MCP_SERVER_TOKEN\"\n")
	}
	if len(gatewayConfig.Env) > 0 {
		envVarNames := sliceutil.MapToSlice(gatewayConfig.Env)
		sort.Strings(envVarNames)
		for _, envVarName := range envVarNames {
			fmt.Fprintf(yaml, "          export %s=%s\n", envVarName, gatewayConfig.Env[envVarName])
		}
	}
}

func buildMCPGatewayContainerCommand(engine CodingAgentEngine, workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig, mcpEnvVars map[string]string, payloadDir string, payloadPathPrefix string, hasGitHub bool, githubTool any, tools map[string]any) string {
	containerImage := gatewayConfig.Container
	if gatewayConfig.Version != "" {
		containerImage += ":" + gatewayConfig.Version
	} else {
		containerImage += ":" + string(constants.DefaultMCPGatewayVersion)
	}
	var containerCmd strings.Builder
	containerCmd.WriteString("docker run -i --rm --network host")
	containerCmd.WriteString(" --add-host host.docker.internal:127.0.0.1")
	containerCmd.WriteString(" --user ${MCP_GATEWAY_UID}:${MCP_GATEWAY_GID}")
	containerCmd.WriteString(" --group-add ${DOCKER_SOCK_GID}")
	containerCmd.WriteString(" -v /var/run/docker.sock:/var/run/docker.sock")
	appendMCPGatewayBaseEnvFlags(&containerCmd, payloadPathPrefix)
	appendMCPGatewayConditionalEnvFlags(&containerCmd, workflowData, engine, hasGitHub, githubTool, tools)
	appendMCPGatewayCustomAndHTTPEnvFlags(&containerCmd, workflowData, gatewayConfig, mcpEnvVars, hasGitHub, githubTool, tools, engine)
	if payloadDir != "" {
		containerCmd.WriteString(" -v " + payloadDir + ":" + payloadDir + ":rw")
	}
	for _, mount := range gatewayConfig.Mounts {
		containerCmd.WriteString(" -v " + mount)
	}
	if gatewayConfig.Entrypoint != "" {
		containerCmd.WriteString(" --entrypoint " + shellEscapeArg(gatewayConfig.Entrypoint))
	}
	containerCmd.WriteString(" " + containerImage)
	for _, arg := range gatewayConfig.EntrypointArgs {
		containerCmd.WriteString(" " + shellEscapeArg(arg))
	}
	for _, arg := range gatewayConfig.Args {
		containerCmd.WriteString(" " + shellEscapeArg(arg))
	}
	return containerCmd.String()
}

func appendMCPGatewayBaseEnvFlags(containerCmd *strings.Builder, payloadPathPrefix string) {
	containerCmd.WriteString(" -e MCP_GATEWAY_PORT")
	containerCmd.WriteString(" -e MCP_GATEWAY_DOMAIN")
	containerCmd.WriteString(" -e MCP_GATEWAY_API_KEY")
	containerCmd.WriteString(" -e MCP_GATEWAY_PAYLOAD_DIR")
	if payloadPathPrefix != "" {
		containerCmd.WriteString(" -e MCP_GATEWAY_PAYLOAD_PATH_PREFIX")
	}
	containerCmd.WriteString(" -e MCP_GATEWAY_PAYLOAD_SIZE_THRESHOLD")
	containerCmd.WriteString(" -e DEBUG")
	containerCmd.WriteString(" -e MCP_GATEWAY_LOG_DIR")
	containerCmd.WriteString(" -e GH_AW_MCP_LOG_DIR")
	containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS")
	containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_CONFIG_PATH")
	containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_TOOLS_PATH")
	containerCmd.WriteString(" -e GH_AW_ASSETS_BRANCH")
	containerCmd.WriteString(" -e GH_AW_ASSETS_MAX_SIZE_KB")
	containerCmd.WriteString(" -e GH_AW_ASSETS_ALLOWED_EXTS")
	containerCmd.WriteString(" -e DEFAULT_BRANCH")
	containerCmd.WriteString(" -e GITHUB_MCP_SERVER_TOKEN")
	containerCmd.WriteString(" -e GITHUB_MCP_GUARD_MIN_INTEGRITY")
	containerCmd.WriteString(" -e GITHUB_MCP_GUARD_REPOS")
	containerCmd.WriteString(" -e GITHUB_REPOSITORY")
	containerCmd.WriteString(" -e GITHUB_SERVER_URL")
	containerCmd.WriteString(" -e GITHUB_SHA")
	containerCmd.WriteString(" -e GITHUB_WORKSPACE")
	containerCmd.WriteString(" -e GITHUB_TOKEN")
	containerCmd.WriteString(" -e GITHUB_RUN_ID")
	containerCmd.WriteString(" -e GITHUB_RUN_NUMBER")
	containerCmd.WriteString(" -e GITHUB_RUN_ATTEMPT")
	containerCmd.WriteString(" -e GITHUB_JOB")
	containerCmd.WriteString(" -e GITHUB_ACTION")
	containerCmd.WriteString(" -e GITHUB_EVENT_NAME")
	containerCmd.WriteString(" -e GITHUB_EVENT_PATH")
	containerCmd.WriteString(" -e GITHUB_ACTOR")
	containerCmd.WriteString(" -e GITHUB_ACTOR_ID")
	containerCmd.WriteString(" -e GITHUB_TRIGGERING_ACTOR")
	containerCmd.WriteString(" -e GITHUB_WORKFLOW")
	containerCmd.WriteString(" -e GITHUB_WORKFLOW_REF")
	containerCmd.WriteString(" -e GITHUB_WORKFLOW_SHA")
	containerCmd.WriteString(" -e GITHUB_REF")
	containerCmd.WriteString(" -e GITHUB_REF_NAME")
	containerCmd.WriteString(" -e GITHUB_REF_TYPE")
	containerCmd.WriteString(" -e GITHUB_HEAD_REF")
	containerCmd.WriteString(" -e GITHUB_BASE_REF")
}

func appendMCPGatewayConditionalEnvFlags(containerCmd *strings.Builder, workflowData *WorkflowData, engine CodingAgentEngine, hasGitHub bool, githubTool any, tools map[string]any) {
	if hasGitHub && getGitHubType(githubTool) == "remote" && engine.GetID() == "copilot" {
		containerCmd.WriteString(" -e GITHUB_PERSONAL_ACCESS_TOKEN")
	}
	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		containerCmd.WriteString(" -e GH_AW_MCP_SCRIPTS_PORT")
		containerCmd.WriteString(" -e GH_AW_MCP_SCRIPTS_API_KEY")
	}
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_PORT")
		containerCmd.WriteString(" -e GH_AW_SAFE_OUTPUTS_API_KEY")
	}
	if workflowData.OTLPEndpoint != "" {
		containerCmd.WriteString(" -e GITHUB_AW_OTEL_TRACE_ID")
		containerCmd.WriteString(" -e GITHUB_AW_OTEL_PARENT_SPAN_ID")
	}
	if hasGitHubOIDCAuthInTools(tools) {
		containerCmd.WriteString(" -e ACTIONS_ID_TOKEN_REQUEST_URL")
		containerCmd.WriteString(" -e ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	}
}

func appendMCPGatewayCustomAndHTTPEnvFlags(containerCmd *strings.Builder, workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig, mcpEnvVars map[string]string, hasGitHub bool, githubTool any, tools map[string]any, engine CodingAgentEngine) {
	if len(gatewayConfig.Env) > 0 {
		envVarNames := sliceutil.MapToSlice(gatewayConfig.Env)
		sort.Strings(envVarNames)
		for _, envVarName := range envVarNames {
			containerCmd.WriteString(" -e " + envVarName)
		}
	}
	if len(mcpEnvVars) == 0 {
		return
	}
	addedEnvVars := buildAddedGatewayEnvVarSet(workflowData, gatewayConfig, hasGitHub, githubTool, tools, engine)
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

func buildAddedGatewayEnvVarSet(workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig, hasGitHub bool, githubTool any, tools map[string]any, engine CodingAgentEngine) map[string]bool {
	addedEnvVars := make(map[string]bool)
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
	if hasGitHubOIDCAuthInTools(tools) {
		addedEnvVars["ACTIONS_ID_TOKEN_REQUEST_URL"] = true
		addedEnvVars["ACTIONS_ID_TOKEN_REQUEST_TOKEN"] = true
	}
	for envVarName := range gatewayConfig.Env {
		addedEnvVars[envVarName] = true
	}
	return addedEnvVars
}
