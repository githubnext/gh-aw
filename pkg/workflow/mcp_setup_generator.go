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
//   - Setting up safe-outputs MCP server runtime files and container config
//   - Setting up mcp-scripts MCP server (config, tool files, HTTP server)
//   - Starting the MCP gateway with proper environment variables
//   - Rendering MCP configuration for the selected AI engine
//
// Setup sequence:
//  1. Download required Docker images
//  2. Install gh-aw extension (if agentic-workflows enabled)
//  3. Write safe-outputs config.json (may contain template expressions; kept small)
//  4. Write safe-outputs tools.json and validation.json (large, no template expressions)
//  5. Prepare safe-outputs runtime files for containerized MCP execution
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
//   - Mount safe-outputs runtime files into the gh-aw node MCP container
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
	generateAgenticWorkflowsInstallStep(c, yaml, hasAgenticWorkflows, workflowData)

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
			// Playwright in CLI mode is not an MCP server; skip it here.
			if toolName == "playwright" && isPlaywrightCLIMode(workflowData.Tools) {
				mcpSetupGeneratorLog.Print("Skipping playwright MCP registration: tools.playwright.mode is cli")
				continue
			}
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
	if IsMCPScriptsEnabled(workflowData.MCPScripts) {
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

func generateAgenticWorkflowsInstallStep(c *Compiler, yaml *strings.Builder, hasAgenticWorkflows bool, workflowData *WorkflowData) {
	if !hasAgenticWorkflows {
		return
	}

	cliVersion := resolveAgenticWorkflowsCLIVersion(c, workflowData)
	effectiveToken := getEffectiveGitHubToken("")
	actionRepo := GitHubOrgRepo + "/actions/setup-cli"
	installStep, err := generateGhAwSetupStep(ghAwSetupStepConfig{
		actionMode:           c.actionMode,
		cliVersion:           cliVersion,
		actionRepo:           actionRepo,
		fallbackActionRefTag: cliVersion,
		workflowData:         workflowData,
		withFields: map[string]string{
			"github-token": effectiveToken,
		},
	})
	if err != nil {
		mcpSetupGeneratorLog.Printf("Failed to resolve pinned setup-cli action reference for %s@%s: %v", actionRepo, cliVersion, err)
	}
	for _, line := range installStep {
		yaml.WriteString(line + "\n")
	}
	yaml.WriteString("      - name: Copy gh-aw binary for MCP server\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          gh aw --version\n")
	yaml.WriteString("          # Copy the gh-aw binary to ${RUNNER_TEMP}/gh-aw for MCP server containerization\n")
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw\"\n")
	yaml.WriteString("          GH_AW_BIN=\"\"\n")
	yaml.WriteString("          GH_AW_BIN=$(command -v gh-aw 2>/dev/null) || true\n")
	yaml.WriteString("          if [ -z \"$GH_AW_BIN\" ]; then\n")
	yaml.WriteString("            GH_AW_BIN=$(find \"${HOME}/.local/share/gh/extensions/gh-aw\" -name 'gh-aw' -type f 2>/dev/null | head -1) || true\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          if [ -z \"$GH_AW_BIN\" ] && [ -n \"${GH_CONFIG_DIR:-}\" ]; then\n")
	yaml.WriteString("            GH_AW_BIN=$(find \"${GH_CONFIG_DIR}/extensions/gh-aw\" -name 'gh-aw' -type f 2>/dev/null | head -1) || true\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          if [ -z \"$GH_AW_BIN\" ] && [ -f \"${GITHUB_WORKSPACE}/gh-aw\" ]; then\n")
	yaml.WriteString("            GH_AW_BIN=\"${GITHUB_WORKSPACE}/gh-aw\"\n")
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

func resolveAgenticWorkflowsCLIVersion(c *Compiler, workflowData *WorkflowData) string {
	cliVersion := c.actionTag
	if cliVersion == "" {
		cliVersion = getActionTagFromFeatures(workflowData)
	}
	if cliVersion == "" {
		cliVersion = c.version
	}
	// "dev" and empty versions are not valid release pins; fall back to the
	// current compiler runtime version so setup-cli always receives a concrete
	// pinned release tag in non-dev modes.
	if cliVersion == "" || cliVersion == "dev" {
		cliVersion = getDefaultGhAWRuntimeVersion()
	}
	return cliVersion
}

func getActionTagFromFeatures(workflowData *WorkflowData) string {
	if workflowData == nil || workflowData.Features == nil {
		return ""
	}
	actionTagVal, exists := workflowData.Features["action-tag"]
	if !exists {
		return ""
	}
	actionTagStr, ok := actionTagVal.(string)
	if !ok || actionTagStr == "" {
		return ""
	}
	return actionTagStr
}

func generateSafeOutputsSetup(c *Compiler, yaml *strings.Builder, safeOutputConfig string, workflowData *WorkflowData) {
	if !HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		return
	}
	yaml.WriteString("      - name: Generate Safe Outputs Config\n")
	sanitizedConfig, envKeys, envValues := buildSafeOutputsConfigRuntimeData(safeOutputConfig)
	if len(envKeys) > 0 {
		yaml.WriteString("        env:\n")
		writeStepEnvVars(yaml, envKeys, envValues)
	}
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/safeoutputs\"\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/safeoutputs\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-logs/safeoutputs\n")
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.UploadArtifact != nil {
		yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts\"\n")
	}

	delimiter := GenerateHeredocDelimiterFromContent("SAFE_OUTPUTS_CONFIG", sanitizedConfig)
	if safeOutputConfig != "" {
		yaml.WriteString("          cat > \"${RUNNER_TEMP}/gh-aw/safeoutputs/config.json\" << '" + delimiter + "'\n")
		yaml.WriteString("          " + sanitizedConfig + "\n")
		yaml.WriteString("          " + delimiter + "\n")
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
	// Propagate mentions config to the collection pass so that allowed @-mentions
	// (e.g. "@copilot") are not backtick-escaped before publish-side handlers run.
	var mentionsBlock map[string]any
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.Mentions != nil {
		mentionsBlock = buildMentionsHandlerConfig(workflowData.SafeOutputs.Mentions)
	}
	validationConfigJSON, err := GetValidationConfigJSON(enabledTypes, mentionsBlock)
	if err != nil {
		mcpSetupGeneratorLog.Printf("CRITICAL: Error generating validation config JSON: %v - validation will not work correctly", err)
		validationConfigJSON = "{}"
	}

	yaml.WriteString("      - name: Generate Safe Outputs Tools\n")
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
}

// safeOutputsSecretEnvPrefix is prepended to secret names when generating step env var names for
// safe-outputs config placeholders. The prefix avoids accidental collisions between a workflow
// secret name and a pre-existing step env var (e.g. a secret named DEBUG or
// GH_AW_SAFE_OUTPUTS_CONFIG_PATH would silently override those step vars without the prefix).
// The prefixed env vars are written into the step env: block and resolved in memory at runtime
// by the JavaScript safe-outputs loader (resolveEnvPlaceholders in safe_outputs_config.cjs).
const safeOutputsSecretEnvPrefix = "GH_AW_SECRET_"

func buildSafeOutputsConfigRuntimeEnvVars(safeOutputConfig string) ([]string, map[string]string) {
	configSecrets := ExtractSecretsFromValue(safeOutputConfig)
	configContextVars := ExtractGitHubContextExpressionsFromValue(safeOutputConfig)
	configWorkflowInputs := ExtractWorkflowInputExpressionsFromValue(safeOutputConfig)
	envValues := make(map[string]string, safeAllocationCapacity(len(configSecrets), len(configContextVars), len(configWorkflowInputs)))
	addEnvValue := func(key, value string) {
		envValues[key] = value
	}
	for k, v := range configWorkflowInputs {
		addEnvValue(k, v)
	}
	for k, v := range configContextVars {
		addEnvValue(k, v)
	}
	for k, v := range configSecrets {
		// Prefix secret env vars to avoid colliding with reserved/known step env var names.
		addEnvValue(safeOutputsSecretEnvPrefix+k, v)
	}
	return sortedMapKeys(envValues), envValues
}

func buildSafeOutputsConfigRuntimeData(safeOutputConfig string) (string, []string, map[string]string) {
	sanitizedConfig := safeOutputConfig
	envKeys, envValues := buildSafeOutputsConfigRuntimeEnvVars(safeOutputConfig)
	for _, varName := range envKeys {
		value := envValues[varName]
		sanitizedConfig = strings.ReplaceAll(sanitizedConfig, value, "${"+varName+"}")
	}
	return sanitizedConfig, envKeys, envValues
}

func writeStepEnvVars(yaml *strings.Builder, envKeys []string, envValues map[string]string) {
	for _, varName := range envKeys {
		yaml.WriteString("          " + varName + ": " + envValues[varName] + "\n")
	}
}

func generateMCPScriptsSetup(yaml *strings.Builder, workflowData *WorkflowData) error {
	if !IsMCPScriptsEnabled(workflowData.MCPScripts) {
		return nil
	}
	yaml.WriteString("      - name: Write MCP Scripts Config\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p \"${RUNNER_TEMP}/gh-aw/mcp-scripts/logs\"\n")

	toolsJSON := GenerateMCPScriptsToolsConfig(workflowData.MCPScripts)
	toolsDelimiter := GenerateHeredocDelimiterFromContent("MCP_SCRIPTS_TOOLS", toolsJSON)
	if err := ValidateHeredocContent(toolsJSON, toolsDelimiter); err != nil {
		return fmt.Errorf("mcp-scripts tools.json: %w", err)
	}
	yaml.WriteString("          cat > \"${RUNNER_TEMP}/gh-aw/mcp-scripts/tools.json\" << '" + toolsDelimiter + "'\n")
	for line := range strings.SplitSeq(toolsJSON, "\n") {
		yaml.WriteString("          " + line + "\n")
	}
	yaml.WriteString("          " + toolsDelimiter + "\n")

	mcpScriptsMCPServer := GenerateMCPScriptsMCPServerScript(workflowData.MCPScripts)
	serverDelimiter := GenerateHeredocDelimiterFromContent("MCP_SCRIPTS_SERVER", mcpScriptsMCPServer)
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
	mcpScriptToolNames := sliceutil.MapKeys(workflowData.MCPScripts.Tools)
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
		envVarNames := sliceutil.MapKeys(mcpScriptsSecrets)
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
		jsDelimiter := GenerateHeredocDelimiterFromContent("MCP_SCRIPTS_JS_"+strings.ToUpper(toolName), toolScript)
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
		shDelimiter := GenerateHeredocDelimiterFromContent("MCP_SCRIPTS_SH_"+strings.ToUpper(toolName), toolScript)
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
		pyDelimiter := GenerateHeredocDelimiterFromContent("MCP_SCRIPTS_PY_"+strings.ToUpper(toolName), toolScript)
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
		goDelimiter := GenerateHeredocDelimiterFromContent("MCP_SCRIPTS_GO_"+strings.ToUpper(toolName), toolScript)
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
	githubToolRaw, hasGitHub := tools["github"]
	githubTool, _ := githubToolRaw.(map[string]any)
	writeMCPGatewayExports(yaml, writeMCPGatewayExportsOptions{
		engine:               engine,
		workflowData:         workflowData,
		gatewayConfig:        gatewayConfig,
		hasGitHub:            hasGitHub,
		githubTool:           githubTool,
		port:                 port,
		domain:               domain,
		payloadDir:           payloadDir,
		payloadPathPrefix:    payloadPathPrefix,
		payloadSizeThreshold: payloadSizeThreshold,
	})
	containerCmd := buildMCPGatewayContainerCommand(buildMCPGatewayContainerCommandOptions{
		engine:            engine,
		workflowData:      workflowData,
		gatewayConfig:     gatewayConfig,
		mcpEnvVars:        mcpEnvVars,
		payloadDir:        payloadDir,
		payloadPathPrefix: payloadPathPrefix,
		hasGitHub:         hasGitHub,
		githubTool:        githubTool,
		tools:             tools,
	})
	yaml.WriteString("          MCP_GATEWAY_UID=$(id -u 2>/dev/null || echo '0')\n")
	yaml.WriteString("          MCP_GATEWAY_GID=$(id -g 2>/dev/null || echo '0')\n")
	// Resolve the Docker socket path from DOCKER_HOST (supports ARC/dind custom socket paths).
	// Only unix:// and bare absolute paths are treated as socket paths; all other schemes
	// (tcp://, ssh://, npipe://, etc.) fall back to /var/run/docker.sock.
	yaml.WriteString("          case \"${DOCKER_HOST:-}\" in\n")
	yaml.WriteString("            unix://* ) DOCKER_SOCK_PATH=\"${DOCKER_HOST#unix://}\" ;;\n")
	yaml.WriteString("            /* ) DOCKER_SOCK_PATH=\"$DOCKER_HOST\" ;;\n")
	yaml.WriteString("            * ) DOCKER_SOCK_PATH=/var/run/docker.sock ;;\n")
	yaml.WriteString("          esac\n")
	yaml.WriteString("          DOCKER_SOCK_GID=$(stat -c '%g' \"$DOCKER_SOCK_PATH\" 2>/dev/null || echo '0')\n")
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
	envVarNames := sliceutil.MapKeys(mcpEnvVars)
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

// writeMCPGatewayExportsOptions holds configuration for writeMCPGatewayExports.
type writeMCPGatewayExportsOptions struct {
	engine               CodingAgentEngine
	workflowData         *WorkflowData
	gatewayConfig        *MCPGatewayRuntimeConfig
	hasGitHub            bool
	githubTool           map[string]any
	port                 int
	domain               string
	payloadDir           string
	payloadPathPrefix    string
	payloadSizeThreshold int
}

func writeMCPGatewayExports(yaml *strings.Builder, opts writeMCPGatewayExportsOptions) {
	engine := opts.engine
	workflowData := opts.workflowData
	gatewayConfig := opts.gatewayConfig
	hasGitHub := opts.hasGitHub
	githubTool := opts.githubTool
	port := opts.port
	domain := opts.domain
	payloadDir := opts.payloadDir
	payloadPathPrefix := opts.payloadPathPrefix
	payloadSizeThreshold := opts.payloadSizeThreshold
	yaml.WriteString("          \n")
	yaml.WriteString("          # Export gateway environment variables for MCP config and gateway script\n")
	yaml.WriteString("          export MCP_GATEWAY_PORT=\"" + strconv.Itoa(port) + "\"\n")
	yaml.WriteString("          export MCP_GATEWAY_DOMAIN=\"" + domain + "\"\n")
	// MCP_GATEWAY_HOST_DOMAIN is the domain used by host-side clients (e.g. Gemini CLI).
	// When MCP_GATEWAY_DOMAIN is host.docker.internal (only reachable from containers),
	// use localhost instead; otherwise inherit the configured domain as-is.
	hostDomain := domain
	if domain == "host.docker.internal" {
		hostDomain = "localhost"
	}
	yaml.WriteString("          export MCP_GATEWAY_HOST_DOMAIN=\"" + hostDomain + "\"\n")
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
			escapedCLIServersJSON := shellEscapeArg(string(cliServersJSON))
			yaml.WriteString("          export GH_AW_MCP_CLI_SERVERS=" + escapedCLIServersJSON + "\n")
		}
	}
	if hasGitHub && getGitHubType(githubTool) == GitHubMCPModeRemote && engine.GetID() == "copilot" {
		yaml.WriteString("          export GITHUB_PERSONAL_ACCESS_TOKEN=\"$GITHUB_MCP_SERVER_TOKEN\"\n")
	}
	if len(gatewayConfig.Env) > 0 {
		envVarNames := sliceutil.MapKeys(gatewayConfig.Env)
		sort.Strings(envVarNames)
		for _, envVarName := range envVarNames {
			fmt.Fprintf(yaml, "          export %s=%s\n", envVarName, gatewayConfig.Env[envVarName])
		}
	}
}

// buildMCPGatewayContainerCommandOptions holds configuration for buildMCPGatewayContainerCommand.
type buildMCPGatewayContainerCommandOptions struct {
	engine            CodingAgentEngine
	workflowData      *WorkflowData
	gatewayConfig     *MCPGatewayRuntimeConfig
	mcpEnvVars        map[string]string
	payloadDir        string
	payloadPathPrefix string
	hasGitHub         bool
	githubTool        map[string]any
	tools             map[string]any
}

func buildMCPGatewayContainerCommand(opts buildMCPGatewayContainerCommandOptions) string {
	engine := opts.engine
	workflowData := opts.workflowData
	gatewayConfig := opts.gatewayConfig
	mcpEnvVars := opts.mcpEnvVars
	payloadDir := opts.payloadDir
	payloadPathPrefix := opts.payloadPathPrefix
	hasGitHub := opts.hasGitHub
	githubTool := opts.githubTool
	tools := opts.tools
	containerImage := gatewayConfig.Container
	if gatewayConfig.Version != "" {
		containerImage += ":" + gatewayConfig.Version
	} else {
		containerImage += ":" + string(constants.DefaultMCPGatewayVersion)
	}
	var containerCmd strings.Builder
	// Pre-size the builder to avoid reallocations. The base flags from
	// appendMCPGatewayBaseEnvFlags alone write ~2KB of -e flags; allocating
	// 2048 bytes upfront covers the common case without overcommitting.
	containerCmd.Grow(2048)
	containerCmd.WriteString("docker run -i --rm --network host")
	containerCmd.WriteString(" --add-host host.docker.internal:127.0.0.1")
	containerCmd.WriteString(" --user ${MCP_GATEWAY_UID}:${MCP_GATEWAY_GID}")
	containerCmd.WriteString(" --group-add ${DOCKER_SOCK_GID}")
	containerCmd.WriteString(" -v ${DOCKER_SOCK_PATH}:/var/run/docker.sock")
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
	// Override DOCKER_HOST inside the gateway to match the fixed mount destination,
	// regardless of what the runner's DOCKER_HOST was (custom path, tcp://, etc.).
	containerCmd.WriteString(" -e DOCKER_HOST=unix:///var/run/docker.sock")
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
	containerCmd.WriteString(" -e RUNNER_TEMP")
}

func appendMCPGatewayConditionalEnvFlags(containerCmd *strings.Builder, workflowData *WorkflowData, engine CodingAgentEngine, hasGitHub bool, githubTool map[string]any, tools map[string]any) {
	if hasGitHub && getGitHubType(githubTool) == GitHubMCPModeRemote && engine.GetID() == "copilot" {
		containerCmd.WriteString(" -e GITHUB_PERSONAL_ACCESS_TOKEN")
	}
	if IsMCPScriptsEnabled(workflowData.MCPScripts) {
		containerCmd.WriteString(" -e GH_AW_MCP_SCRIPTS_PORT")
		containerCmd.WriteString(" -e GH_AW_MCP_SCRIPTS_API_KEY")
	}
	if workflowData.OTLPEndpoint != "" {
		containerCmd.WriteString(" -e GITHUB_AW_OTEL_TRACE_ID")
		containerCmd.WriteString(" -e GITHUB_AW_OTEL_PARENT_SPAN_ID")
		// Pass OTEL_EXPORTER_OTLP_HEADERS as an env var so that auth credentials
		// are not embedded in the stdin JSON config pipe. mcpg reads this env var
		// as the standard OTel mechanism for providing OTLP authentication headers.
		containerCmd.WriteString(" -e OTEL_EXPORTER_OTLP_HEADERS")
	}
	if hasGitHubOIDCAuthInTools(tools) {
		containerCmd.WriteString(" -e ACTIONS_ID_TOKEN_REQUEST_URL")
		containerCmd.WriteString(" -e ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	}
}

func appendMCPGatewayCustomAndHTTPEnvFlags(containerCmd *strings.Builder, workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig, mcpEnvVars map[string]string, hasGitHub bool, githubTool map[string]any, tools map[string]any, engine CodingAgentEngine) {
	if len(gatewayConfig.Env) > 0 {
		envVarNames := sliceutil.MapKeys(gatewayConfig.Env)
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

func buildAddedGatewayEnvVarSet(workflowData *WorkflowData, gatewayConfig *MCPGatewayRuntimeConfig, hasGitHub bool, githubTool map[string]any, tools map[string]any, engine CodingAgentEngine) map[string]bool {
	addedEnvVars := make(map[string]bool)
	standardEnvVars := []string{
		"MCP_GATEWAY_PORT", "MCP_GATEWAY_DOMAIN", "MCP_GATEWAY_API_KEY", "MCP_GATEWAY_PAYLOAD_DIR", "DEBUG",
		"MCP_GATEWAY_LOG_DIR", "GH_AW_MCP_LOG_DIR", "GH_AW_SAFE_OUTPUTS",
		"GH_AW_SAFE_OUTPUTS_CONFIG_PATH", "GH_AW_SAFE_OUTPUTS_TOOLS_PATH",
		"GH_AW_ASSETS_BRANCH", "GH_AW_ASSETS_MAX_SIZE_KB", "GH_AW_ASSETS_ALLOWED_EXTS",
		"DEFAULT_BRANCH", "GITHUB_MCP_SERVER_TOKEN", "GITHUB_MCP_GUARD_MIN_INTEGRITY", "GITHUB_MCP_GUARD_REPOS",
		"GITHUB_REPOSITORY", "GITHUB_SERVER_URL", "GITHUB_SHA", "GITHUB_WORKSPACE",
		"RUNNER_TEMP",
		"GITHUB_TOKEN", "GITHUB_RUN_ID", "GITHUB_RUN_NUMBER", "GITHUB_RUN_ATTEMPT",
		"GITHUB_JOB", "GITHUB_ACTION", "GITHUB_EVENT_NAME", "GITHUB_EVENT_PATH",
		"GITHUB_ACTOR", "GITHUB_ACTOR_ID", "GITHUB_TRIGGERING_ACTOR",
		"GITHUB_WORKFLOW", "GITHUB_WORKFLOW_REF", "GITHUB_WORKFLOW_SHA",
		"GITHUB_REF", "GITHUB_REF_NAME", "GITHUB_REF_TYPE", "GITHUB_HEAD_REF", "GITHUB_BASE_REF",
	}
	for _, envVar := range standardEnvVars {
		addedEnvVars[envVar] = true
	}
	if hasGitHub && getGitHubType(githubTool) == GitHubMCPModeRemote && engine.GetID() == "copilot" {
		addedEnvVars["GITHUB_PERSONAL_ACCESS_TOKEN"] = true
	}
	if IsMCPScriptsEnabled(workflowData.MCPScripts) {
		addedEnvVars["GH_AW_MCP_SCRIPTS_PORT"] = true
		addedEnvVars["GH_AW_MCP_SCRIPTS_API_KEY"] = true
	}
	if workflowData.OTLPEndpoint != "" {
		addedEnvVars["GITHUB_AW_OTEL_TRACE_ID"] = true
		addedEnvVars["GITHUB_AW_OTEL_PARENT_SPAN_ID"] = true
		addedEnvVars["OTEL_EXPORTER_OTLP_HEADERS"] = true
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
