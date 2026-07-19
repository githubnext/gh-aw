package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// generateEngineExecutionSteps generates the GitHub Actions steps for executing the AI engine
func (c *Compiler) generateEngineExecutionSteps(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine, logFile string) {
	// --use-samples (hidden) replaces the agent step with a deterministic driver
	// that replays the workflow's safe-outputs `samples` frontmatter entries
	// through the safe-outputs MCP server. The engine is never invoked.
	if data.UseSamples {
		compilerYamlLog.Printf("Replacing engine execution with samples replay driver: engine=%s", engine.GetID())
		c.generateSamplesReplayStep(yaml, data, logFile)
		return
	}

	steps := engine.GetExecutionSteps(data, logFile)
	compilerYamlLog.Printf("Generating engine execution steps: engine=%s, steps=%d", engine.GetID(), len(steps))

	for _, step := range steps {
		for _, line := range step {
			yaml.WriteString(line)
			yaml.WriteByte('\n')
		}
	}
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
	// GH_AW_SAFE_OUTPUTS lets the log parser detect safe-output entries written by the agent
	// so it can downgrade a "no structured log entries" failure to a warning when the agent
	// demonstrably completed (e.g. emitted a noop). Without this, runs where the container
	// debug log overwrites the JSON stream in agent-stdio.log are permanently red even though
	// the agent ran to completion.
	if data.SafeOutputs != nil {
		yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ steps.set-runtime-paths.outputs.GH_AW_SAFE_OUTPUTS }}\n")
	}
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

// generateAWFReflectSummary generates a step that reads the AWF /reflect payload
// persisted by copilot_harness.cjs and appends a provider/model table to $GITHUB_STEP_SUMMARY.
//
// The /reflect endpoint (served by the AWF api-proxy sidecar on port 10000) returns the
// list of configured LLM providers together with their available model lists. The harness
// fetches this data from inside the AWF container and writes it to /tmp/gh-aw/awf-reflect.json
// so this step can include it in the summary after the agent has completed.
func (c *Compiler) generateAWFReflectSummary(yaml *strings.Builder, data *WorkflowData) {
	compilerYamlLog.Print("Generating AWF reflect summary step")

	yaml.WriteString("      - name: Print AWF reflect summary\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('" + SetupActionDestination + "/awf_reflect_summary.cjs');\n")
	yaml.WriteString("            await main();\n")
}

// generateDetectAgentErrorsStep emits a host-runner step that runs the engine's error detection
// script after the AWF container exits. This step must run on the host runner (not inside the
// container) because GITHUB_OUTPUT is not mounted into the AWF sandbox.
// The step is only emitted when the engine provides a detection script via GetErrorDetectionScriptId.
func (c *Compiler) generateDetectAgentErrorsStep(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine) {
	scriptId := engine.GetErrorDetectionScriptId()
	if scriptId == "" {
		compilerYamlLog.Printf("Skipping error detection step: engine %s has no detection script", engine.GetID())
		return
	}

	compilerYamlLog.Printf("Generating error detection step for engine: %s (script=%s)", engine.GetID(), scriptId)

	yaml.WriteString("      - name: Detect agent errors\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        id: %s\n", constants.DetectAgentErrorsStepID)
	yaml.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(yaml, "        run: node \"${RUNNER_TEMP}/gh-aw/actions/%s.cjs\"\n", scriptId)
}

// generateEngineInstallAndPreAgentSteps emits git credential configuration, the PR-ready-for-review
// checkout, engine installation steps, GitHub MCP app token minting, MCP lockdown detection, guard
// variable parsing, DIFC proxy stop, base-.github-folder restore, pre-agent steps, MCP gateway
// setup, and MCP CLI mount.
// The activation artifact download and comment-memory file preparation are emitted earlier (in
// generateActivationArtifactAndCommentMemorySteps) so that user steps: can access prior
// comment-memory state.
// It returns the resolved CodingAgentEngine for use in subsequent phases.
func (c *Compiler) generateEngineInstallAndPreAgentSteps(yaml *strings.Builder, data *WorkflowData, needsGitConfig bool) (CodingAgentEngine, error) {
	compilerYamlLog.Printf("Git credential configuration needed: %t", needsGitConfig)
	if needsGitConfig {
		gitConfigSteps := c.generateGitConfigurationSteps()
		for _, line := range gitConfigSteps {
			yaml.WriteString(line)
		}
	}

	c.generatePRReadyForReviewCheckout(yaml, data)

	engine, err := c.getAgenticEngine(data.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve agentic engine from AI configuration: %w", err)
	}

	if HasMCPServers(data) {
		ensureDefaultMCPGatewayConfig(data)
	}

	writeEngineInstallSteps(yaml, data, engine)
	for _, step := range c.generateGitHubMCPAppTokenMintingSteps(data) {
		yaml.WriteString(step)
	}

	c.generateGitHubMCPLockdownDetectionStep(yaml, data)
	c.generateParseGuardVarsStep(yaml, data)
	c.generateStopDIFCProxyStep(yaml, data)
	writePreAgentRestoreSteps(yaml, data)
	c.generatePreAgentSteps(yaml, data)

	if err := c.generateMCPSetup(yaml, data.Tools, engine, data); err != nil {
		return nil, fmt.Errorf("failed to generate MCP setup: %w", err)
	}
	c.generateMCPCLIMountStep(yaml, data)

	return engine, nil
}

func writeEngineInstallSteps(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine) {
	installSteps := engine.GetInstallationSteps(data)
	compilerYamlLog.Printf("Adding %d engine installation steps for %s", len(installSteps), engine.GetID())
	writeNestedLineSteps(yaml, installSteps)
	writeNestedLineSteps(yaml, generatePlaywrightCLIInstallSteps(data))
}

func writePreAgentRestoreSteps(yaml *strings.Builder, data *WorkflowData) {
	if ShouldGeneratePRCheckoutStep(data) {
		registry := GetGlobalEngineRegistry()
		generateRestoreBaseGitHubFoldersStep(yaml,
			registry.GetAllAgentManifestFolders(),
			registry.GetAllAgentManifestFiles(),
		)
	}
	if isFeatureEnabled(constants.FeatureFlag("inline-agents"), data) {
		generateRestoreInlineSubAgentsStep(yaml, data)
	}
	if isFeatureEnabled(constants.FeatureFlag("inline-agents"), data) || len(data.Skills) > 0 {
		generateRestoreInlineSkillsStep(yaml, data)
	}
}

func writeNestedLineSteps(yaml *strings.Builder, steps []GitHubActionStep) {
	for _, step := range steps {
		for _, line := range step {
			yaml.WriteString(line)
			yaml.WriteByte('\n')
		}
	}
}

// generateAgentRunSteps emits the git credentials cleaner, engine config steps, CLI proxy start,
// AI execution, CLI proxy stop, Copilot error detection, agent-execution-complete marker,
// post-agent git credential regeneration, firewall log collection, engine pre-bundle steps,
// MCP gateway stop, secret redaction, agent step summary append, and output collection.
// It returns the initial set of artifact paths (to be extended by the caller) and the
// agent stdio log path constant.
func (c *Compiler) generateAgentRunSteps(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine, needsGitConfig bool) ([]string, string, error) {
	var artifactPaths []string
	artifactPaths = append(artifactPaths, constants.AwPromptsFile)

	logFileFull := constants.AgentStdioLogPath

	for _, line := range c.generateCredentialsCleanerStep(data.KnownActionCredentialEnvVars) {
		yaml.WriteString(line)
	}
	c.generatePreAgentAuditStep(yaml)

	if err := writeEngineConfigSteps(yaml, data, engine); err != nil {
		return nil, "", err
	}
	c.generateStartCliProxyStep(yaml, data)
	writeDockerSbxCredentialRefreshStep(yaml, data)

	compilerYamlLog.Printf("Generating engine execution steps for %s", engine.GetID())
	c.generateEngineExecutionSteps(yaml, data, engine, logFileFull)
	c.generateStopCliProxyStep(yaml, data)
	c.generateDetectAgentErrorsStep(yaml, data, engine)

	compilerYamlLog.Print("Marking agent execution as complete for step order tracking")
	c.stepOrderTracker.MarkAgentExecutionComplete()

	if needsGitConfig {
		gitConfigStepsAfterAgent := c.generateGitConfigurationSteps()
		for _, line := range gitConfigStepsAfterAgent {
			yaml.WriteString(line)
		}
	}

	writeNestedLineSteps(yaml, engine.GetFirewallLogsCollectionStep(data))
	writeNestedLineSteps(yaml, engine.GetPreBundleSteps(data))

	c.generateStopMCPGateway(yaml, data)
	c.generateSecretRedactionStep(yaml, yaml.String(), data)
	c.generateAgentStepSummaryAppend(yaml)

	if data.SafeOutputs != nil {
		if err := c.generateOutputCollectionStep(yaml, data); err != nil {
			return nil, "", err
		}
	}

	return artifactPaths, logFileFull, nil
}

func writeEngineConfigSteps(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine) error {
	if len(data.EngineConfigSteps) == 0 {
		return nil
	}
	compilerYamlLog.Printf("Adding %d engine config steps for %s", len(data.EngineConfigSteps), engine.GetID())
	for _, step := range data.EngineConfigSteps {
		stepYAML, err := ConvertStepToYAML(step)
		if err != nil {
			return fmt.Errorf("failed to render engine config step: %w", err)
		}
		yaml.WriteString(stepYAML)
	}
	return nil
}

func writeDockerSbxCredentialRefreshStep(yaml *strings.Builder, data *WorkflowData) {
	if !isDockerSbxRuntime(data) {
		return
	}
	for _, line := range generateDockerSbxCredentialRefreshStep() {
		yaml.WriteString(line)
		yaml.WriteString("\n")
	}
}
