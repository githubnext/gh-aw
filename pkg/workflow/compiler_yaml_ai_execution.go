package workflow

import (
	"fmt"
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

// generateLogParsing generates a step that parses the agent's logs and adds them to the step summary
func (c *Compiler) generateLogParsing(yaml *strings.Builder, engine CodingAgentEngine) {
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
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_AW_AGENT_OUTPUT: %s\n", logFileForParsing)
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
	// Load log parser script from external file using require()
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/" + parserScriptName + ".cjs');\n")
	yaml.WriteString("            await main();\n")
}

// generateMCPScriptsLogParsing generates a step that parses mcp-scripts logs and adds them to the step summary
func (c *Compiler) generateMCPScriptsLogParsing(yaml *strings.Builder) {
	compilerYamlLog.Print("Generating mcp-scripts log parsing step")

	yaml.WriteString("      - name: Parse MCP Scripts logs for step summary\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
	// Load mcp-scripts log parser script from external file using require()
	yaml.WriteString("            const { main } = require('${{ runner.temp }}/gh-aw/actions/parse_mcp_scripts_logs.cjs');\n")
	yaml.WriteString("            await main();\n")
}

// generateMCPGatewayLogParsing generates a step that parses MCP gateway logs and adds them to the step summary
func (c *Compiler) generateMCPGatewayLogParsing(yaml *strings.Builder) {
	compilerYamlLog.Print("Generating MCP gateway log parsing step")

	yaml.WriteString("      - name: Parse MCP Gateway logs for step summary\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        id: %s\n", constants.ParseMCPGatewayStepID)
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
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
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
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
func (c *Compiler) generateTokenUsageSummary(yaml *strings.Builder) {
	compilerYamlLog.Print("Generating token usage summary step")

	yaml.WriteString("      - name: Parse token usage for step summary\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
	yaml.WriteString("            const { main } = require('" + SetupActionDestination + "/parse_token_usage.cjs');\n")
	yaml.WriteString("            await main();\n")
}
