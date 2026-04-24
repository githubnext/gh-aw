package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// generateEngineExecutionSteps generates the GitHub Actions steps for executing the AI engine
func (c *Compiler) generateEngineExecutionSteps(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine, logFile string) {

	steps := engine.GetExecutionSteps(data, logFile)
	compilerYamlLog.Printf("Generating engine execution steps: engine=%s, steps=%d", engine.GetID(), len(steps))

	for _, step := range steps {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}
}

// generateModelsCheckStep generates a step that calls the engine's /models endpoint before the
// agent runs. The step verifies that the API key is valid, reports available models to
// GITHUB_STEP_SUMMARY, and sets models_check_failed=true (then exits 1) when the request fails.
func (c *Compiler) generateModelsCheckStep(yaml *strings.Builder, route *ModelsRoute) {
	compilerYamlLog.Printf("Generating models check step: url=%s", route.URL)

	yaml.WriteString("      - name: Verify engine API access\n")
	fmt.Fprintf(yaml, "        id: %s\n", constants.ModelsCheckStepID)
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          %s: %s\n", route.SecretEnvVar, route.SecretExpr)
	// Pass the custom base URL env var to the step when the user has configured one in engine.env.
	// This allows the bash script to override the default API endpoint at runtime.
	if route.BaseURLEnvVar != "" && route.BaseURLEnvExpr != "" {
		fmt.Fprintf(yaml, "          %s: %s\n", route.BaseURLEnvVar, route.BaseURLEnvExpr)
	}
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw\n")

	// When BaseURLEnvVar is set, resolve the models URL at runtime to support custom API endpoints.
	// The user may have configured ANTHROPIC_BASE_URL or OPENAI_BASE_URL in engine.env to point to
	// a custom API proxy (e.g., enterprise gateway, Azure OpenAI, internal LLM router).
	// We strip any trailing slash from the base URL and append the models path.
	// If the env var is empty/unset, we fall back to the default compiled-in URL.
	if route.BaseURLEnvVar != "" && route.ModelsPath != "" {
		fmt.Fprintf(yaml, "          if [ -n \"${%s:-}\" ]; then\n", route.BaseURLEnvVar)
		fmt.Fprintf(yaml, "            MODELS_URL=\"${%s%%/}%s\"\n", route.BaseURLEnvVar, route.ModelsPath)
		yaml.WriteString("          else\n")
		fmt.Fprintf(yaml, "            MODELS_URL=\"%s\"\n", route.URL)
		yaml.WriteString("          fi\n")
	} else {
		fmt.Fprintf(yaml, "          MODELS_URL=\"%s\"\n", route.URL)
	}

	// Build the curl command with authentication and any extra headers
	yaml.WriteString("          HTTP_CODE=$(curl -s -o /tmp/gh-aw/models-response.json -w \"%{http_code}\" \\\n")
	fmt.Fprintf(yaml, "            -H \"%s: %s$%s\" \\\n", route.AuthHeader, route.AuthScheme, route.SecretEnvVar)

	// Add extra headers in deterministic order
	extraHeaderKeys := make([]string, 0, len(route.ExtraHeaders))
	for k := range route.ExtraHeaders {
		extraHeaderKeys = append(extraHeaderKeys, k)
	}
	sort.Strings(extraHeaderKeys)
	for _, k := range extraHeaderKeys {
		fmt.Fprintf(yaml, "            -H \"%s: %s\" \\\n", k, route.ExtraHeaders[k])
	}

	yaml.WriteString("            \"$MODELS_URL\")\n")
	yaml.WriteString("          echo \"## Engine API Access\" >> \"$GITHUB_STEP_SUMMARY\"\n")
	yaml.WriteString("          if [ \"$HTTP_CODE\" -lt 200 ] || [ \"$HTTP_CODE\" -ge 300 ]; then\n")
	yaml.WriteString("            echo \"models_check_failed=true\" >> \"$GITHUB_OUTPUT\"\n")
	yaml.WriteString("            echo \"❌ Models request failed (HTTP ${HTTP_CODE}). The engine secret may be incorrect or outdated.\" >> \"$GITHUB_STEP_SUMMARY\"\n")
	yaml.WriteString("            exit 1\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          echo \"models_check_failed=false\" >> \"$GITHUB_OUTPUT\"\n")
	yaml.WriteString("          echo \"✅ Engine API access verified (HTTP ${HTTP_CODE})\" >> \"$GITHUB_STEP_SUMMARY\"\n")
	yaml.WriteString("          echo \"\" >> \"$GITHUB_STEP_SUMMARY\"\n")
	yaml.WriteString("          echo \"### Available Models\" >> \"$GITHUB_STEP_SUMMARY\"\n")
	yaml.WriteString("          if command -v jq &>/dev/null && [ -s /tmp/gh-aw/models-response.json ]; then\n")
	yaml.WriteString("            jq -r '(.data // .models // []) | .[] | \"- \" + (.id // .name // .model // \"unknown\")' \\\n")
	yaml.WriteString("              /tmp/gh-aw/models-response.json >> \"$GITHUB_STEP_SUMMARY\" 2>/dev/null \\\n")
	yaml.WriteString("              || cat /tmp/gh-aw/models-response.json >> \"$GITHUB_STEP_SUMMARY\"\n")
	yaml.WriteString("          fi\n")
}

// generateLogParsing generates a step that parses the agent's logs and adds them to the step summary
func (c *Compiler) generateLogParsing(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine) {
	parserScriptName := engine.GetLogParserScriptId()
	if parserScriptName == "" {
		// Skip log parsing if engine doesn't provide a parser
		compilerYamlLog.Printf("Skipping log parsing: engine %s has no parser script", engine.GetID())
		return
	}

	compilerYamlLog.Printf("Generating log parsing step for engine: %s (parser=%s)", engine.GetID(), parserScriptName)

	logParserScript := GetLogParserScript(parserScriptName)
	if logParserScript == "" {
		// Skip if parser script not found
		compilerYamlLog.Printf("Warning: parser script %s not found, skipping log parsing", parserScriptName)
		return
	}

	// Get the log file path for parsing (may be different from stdout/stderr log)
	logFileForParsing := engine.GetLogFileForParsing()

	yaml.WriteString("      - name: Parse agent logs for step summary\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_AW_AGENT_OUTPUT: %s\n", logFileForParsing)
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	// Load log parser script from external file using require()
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/" + parserScriptName + ".cjs');\n")
	yaml.WriteString("            await main();\n")
}

// generateMCPScriptsLogParsing generates a step that parses mcp-scripts logs and adds them to the step summary
func (c *Compiler) generateMCPScriptsLogParsing(yaml *strings.Builder, data *WorkflowData) {
	compilerYamlLog.Print("Generating mcp-scripts log parsing step")

	yaml.WriteString("      - name: Parse MCP Scripts logs for step summary\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	// Load mcp-scripts log parser script from external file using require()
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/parse_mcp_scripts_logs.cjs');\n")
	yaml.WriteString("            await main();\n")
}

// generateMCPGatewayLogParsing generates a step that parses MCP gateway logs and adds them to the step summary
func (c *Compiler) generateMCPGatewayLogParsing(yaml *strings.Builder, data *WorkflowData) {
	compilerYamlLog.Print("Generating MCP gateway log parsing step")

	yaml.WriteString("      - name: Parse MCP Gateway logs for step summary\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        id: %s\n", constants.ParseMCPGatewayStepID)
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	// Load MCP gateway log parser script from external file using require()
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/parse_mcp_gateway_log.cjs');\n")
	yaml.WriteString("            await main();\n")
}

// generateObservabilitySummary generates a step that synthesizes a compact
// observability section for the GitHub Actions step summary from existing runtime files.
// The step is only emitted when OTLP is configured in the workflow.
func (c *Compiler) generateObservabilitySummary(yaml *strings.Builder, data *WorkflowData) {
	if !isOTLPEnabled(data) {
		return
	}

	compilerYamlLog.Print("Generating observability step summary")

	yaml.WriteString("      - name: Generate observability summary\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/generate_observability_summary.cjs');\n")
	yaml.WriteString("            await main(core);\n")
}

// isOTLPEnabled returns true when OTLP has been configured in the workflow (including
// imported frontmatter). It checks whether injectOTLPConfig has already written the
// OTEL_EXPORTER_OTLP_ENDPOINT env var into workflowData.Env, which is the authoritative
// result of OTLP detection after all frontmatter (main + imports) has been processed.
func isOTLPEnabled(data *WorkflowData) bool {
	if data == nil {
		return false
	}
	return strings.Contains(data.Env, "OTEL_EXPORTER_OTLP_ENDPOINT")
}

// generateStopMCPGateway generates a step that stops the MCP gateway process using its PID from step output
// It passes the gateway port and API key to enable graceful shutdown via /close endpoint
func (c *Compiler) generateStopMCPGateway(yaml *strings.Builder, data *WorkflowData) {
	compilerYamlLog.Print("Generating MCP gateway stop step")

	yaml.WriteString("      - name: Stop MCP Gateway\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")

	// Add environment variables for graceful shutdown via /close endpoint
	// These values come from the Start MCP Gateway step outputs
	// Security: Pass all step outputs through environment variables to prevent template injection
	yaml.WriteString("        env:\n")
	yaml.WriteString("          MCP_GATEWAY_PORT: ${{ steps.start-mcp-gateway.outputs.gateway-port }}\n")
	yaml.WriteString("          MCP_GATEWAY_API_KEY: ${{ steps.start-mcp-gateway.outputs.gateway-api-key }}\n")
	yaml.WriteString("          GATEWAY_PID: ${{ steps.start-mcp-gateway.outputs.gateway-pid }}\n")

	yaml.WriteString("        run: |\n")
	yaml.WriteString("          bash \"${RUNNER_TEMP}/gh-aw/actions/stop_mcp_gateway.sh\" \"$GATEWAY_PID\"\n")
}

// generateAgentOutputPlaceholderStep generates a step that writes a minimal {"items":[]}
// placeholder to agent_output.json when the engine exits before producing any safe outputs.
// This prevents downstream safe_outputs and conclusion jobs from receiving an ENOENT error
// when loading the agent output file, making it easier to surface the real engine failure
// reason (e.g. quota exceeded) instead of an unhelpful file-not-found message.
func (c *Compiler) generateAgentOutputPlaceholderStep(yaml *strings.Builder) {
	compilerYamlLog.Print("Generating agent output placeholder step")

	yaml.WriteString("      - name: Write agent output placeholder if missing\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          if [ ! -f /tmp/gh-aw/agent_output.json ]; then\n")
	yaml.WriteString("            echo '{\"items\":[]}' > /tmp/gh-aw/agent_output.json\n")
	yaml.WriteString("          fi\n")
}

// generateAgentStepSummaryAppend generates a step that appends the agent's GITHUB_STEP_SUMMARY
// file to the real $GITHUB_STEP_SUMMARY. This runs after secret redaction so the content
// is already sanitised before being published to the workflow step summary.
// The step is a no-op when the file is empty (agent wrote nothing).
func (c *Compiler) generateAgentStepSummaryAppend(yaml *strings.Builder) {
	compilerYamlLog.Print("Generating agent step summary append step")

	yaml.WriteString("      - name: Append agent step summary\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/append_agent_step_summary.sh\"\n")
}

// generateTokenUsageSummary generates a step that parses the firewall proxy's
// token-usage.jsonl and appends a markdown table to $GITHUB_STEP_SUMMARY.
// The step also writes aggregated token totals to /tmp/gh-aw/agent_usage.json
// so they are bundled in the agent artifact for third-party tools.
func (c *Compiler) generateTokenUsageSummary(yaml *strings.Builder, data *WorkflowData) {
	compilerYamlLog.Print("Generating token usage summary step")

	yaml.WriteString("      - name: Parse token usage for step summary\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('" + SetupActionDestination + "/parse_token_usage.cjs');\n")
	yaml.WriteString("            await main();\n")
}
