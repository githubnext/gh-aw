package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var threatLog = logger.New("workflow:threat_detection")

// ThreatDetectionConfig holds configuration for threat detection in agent output
type ThreatDetectionConfig struct {
	Prompt              string        `yaml:"prompt,omitempty"`            // Additional custom prompt instructions to append
	Steps               []any         `yaml:"steps,omitempty"`             // Array of extra job steps to run before engine execution
	PostSteps           []any         `yaml:"post-steps,omitempty"`        // Array of extra job steps to run after engine execution
	EngineConfig        *EngineConfig `yaml:"engine-config,omitempty"`     // Extended engine configuration for threat detection
	EngineDisabled      bool          `yaml:"-"`                           // Internal flag: true when engine is explicitly set to false
	RunsOn              string        `yaml:"runs-on,omitempty"`           // Runner override for the detection job
	ContinueOnError     *bool         `yaml:"continue-on-error,omitempty"` // When true (default), detection failures produce warnings instead of blocking safe outputs
	EnabledExpr         *string       `yaml:"-"`                           // Expression form of the enabled flag, e.g. "${{ inputs.enable-threat-detection }}"
	ContinueOnErrorExpr *string       `yaml:"-"`                           // Expression form of continue-on-error, e.g. "${{ inputs.coe }}"
}

// IsContinueOnError reports whether detection failures should produce warnings instead of errors.
// Defaults to true (continue) when not explicitly set.
// Note: when ContinueOnErrorExpr is set, the value is determined at runtime; this method returns
// true as a safe compile-time default (matches the default behaviour).
func (td *ThreatDetectionConfig) IsContinueOnError() bool {
	return td.ContinueOnError == nil || *td.ContinueOnError
}

// HasRunnableDetection reports whether this config will produce a detection job
// that actually executes. Returns false when the engine is disabled and no
// custom steps are configured, since the job would have nothing to run.
// When EnabledExpr is set, detection is conditionally enabled at runtime so we always
// compile the detection job.
func (td *ThreatDetectionConfig) HasRunnableDetection() bool {
	if td.EnabledExpr != nil {
		return true
	}
	return !td.EngineDisabled || len(td.Steps) > 0 || len(td.PostSteps) > 0
}

// IsConditional reports whether detection is expression-controlled (enabled/disabled at runtime).
// When true the detection job is always compiled but its if: condition includes the caller
// expression so GitHub Actions evaluates it at runtime.
func (td *ThreatDetectionConfig) IsConditional() bool {
	return td.EnabledExpr != nil
}

// IsDetectionJobEnabled reports whether a detection job should be created for
// the given safe-outputs configuration. This is the single source of truth
// used by all codepaths that decide whether to create, depend on, or reference
// the detection job.
func IsDetectionJobEnabled(so *SafeOutputsConfig) bool {
	return so != nil && so.ThreatDetection != nil && so.ThreatDetection.HasRunnableDetection()
}

// IsConditionalDetection reports whether the safe-outputs configuration uses an expression
// to control threat detection at runtime. When true, the detection job is always compiled
// but may be skipped at runtime; downstream jobs must handle the skipped result.
func IsConditionalDetection(so *SafeOutputsConfig) bool {
	return so != nil && so.ThreatDetection != nil && so.ThreatDetection.IsConditional()
}

// isThreatDetectionExplicitlyDisabledInConfigs checks whether any of the provided
// safe-outputs config JSON strings has threat-detection explicitly set to disabled.
// Supports both the boolean form (threat-detection: false) and the object form
// (threat-detection: { enabled: false }), mirroring parseThreatDetectionConfig.
// This is used to determine whether the default detection should be applied when
// safe-outputs comes from imports/includes (i.e. no safe-outputs: section in the
// main workflow frontmatter).
func isThreatDetectionExplicitlyDisabledInConfigs(configs []string) bool {
	for _, configJSON := range configs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}
		var config map[string]any
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			continue
		}
		if tdVal, exists := config["threat-detection"]; exists {
			// Boolean form: threat-detection: false
			if tdBool, ok := tdVal.(bool); ok && !tdBool {
				return true
			}
			// Object form: threat-detection: { enabled: false }
			if tdMap, ok := tdVal.(map[string]any); ok {
				if enabled, exists := tdMap["enabled"]; exists {
					if enabledBool, ok := enabled.(bool); ok && !enabledBool {
						return true
					}
				}
			}
		}
	}
	return false
}

// parseThreatDetectionConfig handles threat-detection configuration
func (c *Compiler) parseThreatDetectionConfig(outputMap map[string]any) *ThreatDetectionConfig {
	if configData, exists := outputMap["threat-detection"]; exists {
		threatLog.Print("Found threat-detection configuration")
		// Handle boolean values
		if boolVal, ok := configData.(bool); ok {
			if !boolVal {
				threatLog.Print("Threat detection explicitly disabled")
				// When explicitly disabled, return nil
				return nil
			}
			threatLog.Print("Threat detection enabled with default settings")
			// When enabled as boolean, return empty config
			return &ThreatDetectionConfig{}
		}

		// Handle expression string values (e.g. "${{ inputs.enable-threat-detection }}")
		if strVal, ok := configData.(string); ok {
			if isExpression(strVal) {
				threatLog.Printf("Threat detection controlled by runtime expression: %s", strVal)
				// Detection is conditionally enabled at runtime; always compile the detection job.
				return &ThreatDetectionConfig{EnabledExpr: &strVal}
			}
			// Non-expression strings are rejected by the JSON schema validator; log and fall through.
			threatLog.Printf("Ignoring invalid non-expression string for threat-detection: %s", strVal)
		}

		// Handle object configuration
		if configMap, ok := configData.(map[string]any); ok {
			// Check for enabled field – supports both literal bool and expression string.
			if enabled, exists := configMap["enabled"]; exists {
				switch v := enabled.(type) {
				case bool:
					if !v {
						threatLog.Print("Threat detection disabled via enabled field")
						// When explicitly disabled, return nil
						return nil
					}
				case string:
					if isExpression(v) {
						threatLog.Printf("Threat detection enabled field is a runtime expression: %s", v)
						// Parse remaining fields but record the expression for runtime evaluation.
						config := c.parseThreatDetectionObjectConfig(configMap)
						config.EnabledExpr = &v
						return config
					}
					// Non-expression strings are invalid; fall through to parse remaining fields.
					threatLog.Printf("Ignoring invalid non-expression string for enabled: %s", v)
				}
			}

			return c.parseThreatDetectionObjectConfig(configMap)
		}
	}

	// Default behavior: enabled if any safe-outputs are configured
	threatLog.Print("Using default threat detection configuration")
	return &ThreatDetectionConfig{}
}

// parseThreatDetectionObjectConfig parses the object form of threat-detection config,
// assuming enabled has already been checked and is truthy. It extracts prompt, steps,
// post-steps, runs-on, continue-on-error, and engine fields.
func (c *Compiler) parseThreatDetectionObjectConfig(configMap map[string]any) *ThreatDetectionConfig {
	threatConfig := &ThreatDetectionConfig{}

	// Parse prompt field
	if prompt, exists := configMap["prompt"]; exists {
		if promptStr, ok := prompt.(string); ok {
			threatConfig.Prompt = promptStr
		}
	}

	// Parse steps field (pre-execution steps, run before engine execution)
	if steps, exists := configMap["steps"]; exists {
		if stepsArray, ok := steps.([]any); ok {
			threatConfig.Steps = stepsArray
		}
	}

	// Parse post-steps field (post-execution steps, run after engine execution)
	if postSteps, exists := configMap["post-steps"]; exists {
		if postStepsArray, ok := postSteps.([]any); ok {
			threatConfig.PostSteps = postStepsArray
		}
	}

	// Parse runs-on field
	if runOn, exists := configMap["runs-on"]; exists {
		if runOnStr, ok := runOn.(string); ok {
			threatConfig.RunsOn = runOnStr
		}
	}

	// Parse continue-on-error field (default: true).
	// Accepts a literal bool or a GitHub Actions expression string.
	if coe, exists := configMap["continue-on-error"]; exists {
		switch v := coe.(type) {
		case bool:
			threatConfig.ContinueOnError = &v
			threatLog.Printf("Threat detection continue-on-error set to: %v", v)
		case string:
			if isExpression(v) {
				threatLog.Printf("Threat detection continue-on-error is a runtime expression: %s", v)
				threatConfig.ContinueOnErrorExpr = &v
			}
		}
	}

	// Parse engine field (supports string, object, and boolean false formats)
	if engine, exists := configMap["engine"]; exists {
		// Handle boolean false to disable AI engine
		if engineBool, ok := engine.(bool); ok {
			if !engineBool {
				threatLog.Print("Threat detection AI engine disabled")
				// engine: false means no AI engine steps
				threatConfig.EngineConfig = nil
				threatConfig.EngineDisabled = true
			}
		} else if engineStr, ok := engine.(string); ok {
			threatLog.Printf("Threat detection engine set to: %s", engineStr)
			// Handle string format
			threatConfig.EngineConfig = &EngineConfig{ID: engineStr}
		} else if engineObj, ok := engine.(map[string]any); ok {
			threatLog.Print("Parsing threat detection engine configuration")
			// Handle object format - use extractEngineConfig logic
			_, engineConfig := c.ExtractEngineConfig(map[string]any{"engine": engineObj})
			threatConfig.EngineConfig = engineConfig
		}
	}

	threatLog.Printf("Threat detection configured with custom prompt: %v, custom pre-steps: %v, custom post-steps: %v", threatConfig.Prompt != "", len(threatConfig.Steps) > 0, len(threatConfig.PostSteps) > 0)
	return threatConfig
}

// extractRawExpression strips the "${{" prefix and "}}" suffix from a GitHub Actions
// expression string (e.g. "${{ inputs.flag }}" → "inputs.flag"). The result can be
// embedded directly into a YAML if: condition expression tree.
// Callers must ensure the input is a valid expression (verified by isExpression()) before
// calling this function; non-expression strings are returned with no modification.
func extractRawExpression(expr string) string {
	s := strings.TrimPrefix(expr, "${{")
	s = strings.TrimSuffix(s, "}}")
	return strings.TrimSpace(s)
}

// detectionStepCondition is the if condition applied to inline detection steps.
// Detection steps only run when the detection guard determines there's output to analyze.
const detectionStepCondition = "always() && steps.detection_guard.outputs.run_detection == 'true'"

// buildDetectionJobSteps builds the threat detection steps to be run in the separate detection job.
// These steps run after the agent job completes and analyze agent output for threats using the
// same agentic engine with sandbox.agent and fully blocked network.
// The detection job downloads the agent artifact to access the output files.
func (c *Compiler) buildDetectionJobSteps(data *WorkflowData) []string {
	threatLog.Print("Building threat detection steps for detection job")
	if data.SafeOutputs == nil || data.SafeOutputs.ThreatDetection == nil {
		return nil
	}

	var steps []string

	// Comment separator
	steps = append(steps, "      # --- Threat Detection ---\n")

	// Step 0: Clean stale firewall files left by the agent artifact download.
	// The agent artifact populates sandbox/firewall/logs and sandbox/firewall/audit
	// with files that cause the squid container to crash on start-up.
	steps = append(steps, c.buildCleanFirewallDirsStep()...)

	// Step 1: Pull AWF container images - the detection engine runs inside AWF (firewall),
	// so pre-pulling the containers speeds up execution and avoids on-demand pulls.
	//
	// For Codex detection, MCP setup generation already emits this step, so skip here
	// to avoid duplicate step IDs/names in the detection job.
	if c.getThreatDetectionEngineID(data) != "codex" {
		steps = append(steps, c.buildPullAWFContainersStep(data)...)
	}

	// Step 2: Detection guard - determines whether detection should run
	steps = append(steps, c.buildDetectionGuardStep()...)

	// Step 3: Clear MCP configuration files so the detection engine runs without MCP servers
	steps = append(steps, c.buildClearMCPConfigStep()...)

	// Step 4: Prepare files - copies agent output files to expected paths
	steps = append(steps, c.buildPrepareDetectionFilesStep()...)

	// Step 5: Custom pre-steps if configured (run before engine execution)
	if len(data.SafeOutputs.ThreatDetection.Steps) > 0 {
		steps = append(steps, c.buildCustomThreatDetectionSteps(data.SafeOutputs.ThreatDetection.Steps)...)
	}

	// Step 6: Setup threat detection (github-script)
	steps = append(steps, c.buildThreatDetectionAnalysisStep(data)...)

	// Step 7: Engine execution (AWF, no network)
	steps = append(steps, c.buildDetectionEngineExecutionStep(data)...)

	// Step 8: Custom post-steps if configured (run after engine execution)
	if len(data.SafeOutputs.ThreatDetection.PostSteps) > 0 {
		steps = append(steps, c.buildCustomThreatDetectionSteps(data.SafeOutputs.ThreatDetection.PostSteps)...)
	}

	// Step 9: Upload detection-artifact
	steps = append(steps, c.buildUploadDetectionLogStep(data)...)

	// Step 10: Parse results, log extensively, and set job conclusion (single JS step)
	steps = append(steps, c.buildDetectionConclusionStep(data)...)

	threatLog.Printf("Generated %d detection job step lines", len(steps))
	return steps
}

// buildPullAWFContainersStep creates a step that pre-pulls AWF (agent workflow firewall)
// container images in the detection job. The detection engine runs inside AWF, which uses
// three containers (squid, agent, api-proxy). Pre-pulling avoids on-demand pulls at runtime.
// Only AWF images are pulled here; MCP server images are not needed for detection.
func (c *Compiler) buildPullAWFContainersStep(data *WorkflowData) []string {
	// Build a minimal WorkflowData that represents the detection engine context so
	// collectDockerImages returns only the AWF firewall images (no MCP tool images).
	engineSetting := data.AI
	if engineSetting == "" {
		engineSetting = "claude"
	}
	detectionData := &WorkflowData{
		Tools: map[string]any{},
		AI:    engineSetting,
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Type: SandboxTypeAWF,
			},
		},
		ActionCache: data.ActionCache, // Propagate cache so container digest pins are applied
		Features:    data.Features,    // Propagate features so cli-proxy image is included when enabled
	}

	images := collectDockerImages(detectionData.Tools, detectionData, c.actionMode)
	if len(images) == 0 {
		return nil
	}

	var b strings.Builder
	generateDownloadDockerImagesStep(&b, images)
	if b.Len() == 0 {
		return nil
	}

	// Split the generated YAML into individual lines so each is a separate entry
	lines := strings.Split(b.String(), "\n")
	var steps []string
	for _, line := range lines {
		if line != "" {
			steps = append(steps, line+"\n")
		}
	}
	return steps
}

// buildDetectionGuardStep creates a guard step that checks if detection should run.
// Uses always() to run even if the agent job failed (detection still analyzes whatever output exists).
// In the separate detection job, output metadata is read from the agent job's outputs.
func (c *Compiler) buildDetectionGuardStep() []string {
	return []string{
		"      - name: Check if detection needed\n",
		"        id: detection_guard\n",
		"        if: always()\n",
		"        env:\n",
		"          OUTPUT_TYPES: ${{ needs.agent.outputs.output_types }}\n",
		"          HAS_PATCH: ${{ needs.agent.outputs.has_patch }}\n",
		"        run: |\n",
		"          if [[ -n \"$OUTPUT_TYPES\" || \"$HAS_PATCH\" == \"true\" ]]; then\n",
		"            echo \"run_detection=true\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"Detection will run: output_types=$OUTPUT_TYPES, has_patch=$HAS_PATCH\"\n",
		"          else\n",
		"            echo \"run_detection=false\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"Detection skipped: no agent outputs or patches to analyze\"\n",
		"          fi\n",
	}
}

// buildClearMCPConfigStep creates a step that removes MCP configuration files.
// This ensures the detection engine runs without any MCP servers.
func (c *Compiler) buildClearMCPConfigStep() []string {
	return []string{
		"      - name: Clear MCP Config for detection\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		"        run: |\n",
		"          rm -f \"${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json\"\n",
		"          rm -f /home/runner/.copilot/mcp-config.json\n",
		"          rm -f \"$GITHUB_WORKSPACE/.gemini/settings.json\"\n",
	}
}

// buildCleanFirewallDirsStep creates a step that removes stale firewall files
// from the directories populated by the agent artifact download. When the agent
// artifact is extracted to /tmp/gh-aw/, it pre-populates the sandbox/firewall/logs
// and sandbox/firewall/audit directories with files from the agent job (squid.conf,
// cache.log, access.log, etc.). If these files are present when AWF starts the
// squid container in the detection job, squid fails to initialise (exit code 1).
// Cleaning these directories before pulling containers avoids the crash.
func (c *Compiler) buildCleanFirewallDirsStep() []string {
	return []string{
		"      - name: Clean stale firewall files from agent artifact\n",
		"        run: |\n",
		fmt.Sprintf("          rm -rf %s\n", constants.AWFProxyLogsDir),
		fmt.Sprintf("          rm -rf %s\n", constants.AWFAuditDir),
	}
}

// buildPrepareDetectionFilesStep creates a step that copies agent output files
// to the /tmp/gh-aw/threat-detection/ directory expected by the detection JS scripts.
// In the separate detection job, files are available after downloading the agent artifact.
func (c *Compiler) buildPrepareDetectionFilesStep() []string {
	return []string{
		"      - name: Prepare threat detection files\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		"        run: |\n",
		"          mkdir -p /tmp/gh-aw/threat-detection/aw-prompts\n",
		"          cp /tmp/gh-aw/aw-prompts/prompt.txt /tmp/gh-aw/threat-detection/aw-prompts/prompt.txt 2>/dev/null || true\n",
		"          if [ ! -s /tmp/gh-aw/threat-detection/aw-prompts/prompt.txt ]; then\n",
		"            echo \"::warning::ERR_VALIDATION: Missing or empty detection context prompt at /tmp/gh-aw/threat-detection/aw-prompts/prompt.txt. Ensure the agent artifact includes /tmp/gh-aw/aw-prompts/prompt.txt. Detection will continue with fallback workflow context.\"\n",
		"          fi\n",
		"          cp /tmp/gh-aw/agent_output.json /tmp/gh-aw/threat-detection/agent_output.json 2>/dev/null || true\n",
		"          for f in /tmp/gh-aw/aw-*.patch; do\n",
		"            [ -f \"$f\" ] && cp \"$f\" /tmp/gh-aw/threat-detection/ 2>/dev/null || true\n",
		"          done\n",
		"          for f in /tmp/gh-aw/aw-*.bundle; do\n",
		"            [ -f \"$f\" ] && cp \"$f\" /tmp/gh-aw/threat-detection/ 2>/dev/null || true\n",
		"          done\n",
		"          echo \"Prepared threat detection files:\"\n",
		"          ls -la /tmp/gh-aw/threat-detection/ 2>/dev/null || true\n",
	}
}

// buildDetectionConclusionStep creates the combined parse-and-conclude step for threat detection.
// This single JS step consolidates what was previously two steps:
//  1. Parsing the detection log (parse_detection_results)
//  2. Setting the final job conclusion (detection_conclusion)
//
// It always runs (always()) so that job outputs are set regardless of prior step outcomes.
// The RUN_DETECTION env var lets the script short-circuit with conclusion=skipped when
// the detection guard determined there was no output to analyze.
func (c *Compiler) buildDetectionConclusionStep(data *WorkflowData) []string {
	// Determine continue-on-error mode (default: true — detection failures produce warnings).
	// When ContinueOnErrorExpr is set the value is resolved at runtime; compile-time we use
	// true as a safe default so the step-level continue-on-error is included (permissive).
	continueOnError := true
	var continueOnErrorExpr *string
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil {
		continueOnError = data.SafeOutputs.ThreatDetection.IsContinueOnError()
		continueOnErrorExpr = data.SafeOutputs.ThreatDetection.ContinueOnErrorExpr
	}

	steps := []string{
		"      - name: Parse and conclude threat detection\n",
		"        id: detection_conclusion\n",
		"        if: always()\n",
	}
	// In warn mode (continue-on-error: true), add continue-on-error to the parse step so that
	// an unexpected exception in the parse script never causes the detection job to fail. The
	// script already handles all expected error cases via setDetectionFailure(), but adding
	// continue-on-error here as a defence-in-depth measure prevents the detection job from
	// blocking safe_outputs due to an unanticipated runtime error in the parse step.
	// In strict mode (continue-on-error: false), we intentionally leave this off so that
	// a parse failure in strict mode keeps the detection job result as failure.
	// When the value is an expression, emit it unquoted; when the value is a literal, only
	// emit if true (permissive default). In either expression or literal-true case the step
	// is included, so the two paths are distinct.
	if continueOnErrorExpr != nil {
		// Expression form: GitHub Actions evaluates this at runtime.
		steps = append(steps, fmt.Sprintf("        continue-on-error: %s\n", *continueOnErrorExpr))
	} else if continueOnError {
		steps = append(steps, "        continue-on-error: true\n")
	}

	// Build the GH_AW_DETECTION_CONTINUE_ON_ERROR env var.
	var coeEnvLine string
	if continueOnErrorExpr != nil {
		// Pass the expression unquoted so GitHub Actions evaluates it at runtime.
		coeEnvLine = fmt.Sprintf("          GH_AW_DETECTION_CONTINUE_ON_ERROR: %s\n", *continueOnErrorExpr)
	} else {
		coeEnvLine = fmt.Sprintf("          GH_AW_DETECTION_CONTINUE_ON_ERROR: %q\n", strconv.FormatBool(continueOnError))
	}

	steps = append(steps, []string{
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		"          RUN_DETECTION: ${{ steps.detection_guard.outputs.run_detection }}\n",
		"          DETECTION_AGENTIC_EXECUTION_OUTCOME: ${{ steps.detection_agentic_execution.outcome }}\n",
		coeEnvLine,
		"        with:\n",
		"          script: |\n",
	}...)

	script := c.buildResultsParsingScriptRequire()
	formattedScript := FormatJavaScriptForYAML(script)
	steps = append(steps, formattedScript...)

	return steps
}

// buildThreatDetectionAnalysisStep creates the main threat analysis step
func (c *Compiler) buildThreatDetectionAnalysisStep(data *WorkflowData) []string {
	var steps []string

	// Setup step
	steps = append(steps, []string{
		"      - name: Setup threat detection\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
	}...)
	steps = append(steps, c.buildWorkflowContextEnvVars(data)...)

	// Add HAS_PATCH environment variable from the agent job output (detection runs in a separate job)
	steps = append(steps, "          HAS_PATCH: ${{ needs.agent.outputs.has_patch }}\n")

	// Add custom prompt instructions if configured
	customPrompt := ""
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil {
		customPrompt = data.SafeOutputs.ThreatDetection.Prompt
	}
	if customPrompt != "" {
		steps = append(steps, fmt.Sprintf("          CUSTOM_PROMPT: %q\n", customPrompt))
	}

	steps = append(steps, []string{
		"        with:\n",
		"          script: |\n",
	}...)

	// Require the setup_threat_detection.cjs module and call main with the template
	setupScript := c.buildSetupScriptRequire()
	formattedSetupScript := FormatJavaScriptForYAML(setupScript)
	steps = append(steps, formattedSetupScript...)

	// Add a small shell step in YAML to ensure the output directory and log file exist
	steps = append(steps, []string{
		"      - name: Ensure threat-detection directory and log\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		"        run: |\n",
		"          mkdir -p /tmp/gh-aw/threat-detection\n",
		"          touch /tmp/gh-aw/threat-detection/detection.log\n",
	}...)

	return steps
}

// buildSetupScriptRequire creates the setup script that requires the .cjs module
func (c *Compiler) buildSetupScriptRequire() string {
	// Build a simple require statement that calls the main function
	// The template is now read from file at runtime by the JavaScript module
	script := `const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
setupGlobals(core, github, context, exec, io, getOctokit);
const { main } = require('` + SetupActionDestination + `/setup_threat_detection.cjs');
await main();`

	return script
}

// buildDetectionEngineExecutionStep creates the engine execution step for inline threat detection.
// It uses the same agentic engine already installed in the agent job, but runs it through
// sandbox.agent (AWF) with no allowed domains (network fully blocked) and no MCP configured.
func (c *Compiler) buildDetectionEngineExecutionStep(data *WorkflowData) []string {
	// Check if threat detection has engine explicitly disabled
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil {
		if data.SafeOutputs.ThreatDetection.EngineDisabled {
			// Engine explicitly disabled with engine: false
			return []string{
				"      # AI engine disabled for threat detection (engine: false)\n",
			}
		}
	}

	// Determine which engine to use: threat detection engine from frontmatter,
	// otherwise main engine.
	engineSetting := c.getThreatDetectionEngineID(data)

	engineConfig := data.EngineConfig
	hasThreatDetectionEngineConfig := data.SafeOutputs != nil &&
		data.SafeOutputs.ThreatDetection != nil &&
		data.SafeOutputs.ThreatDetection.EngineConfig != nil
	if hasThreatDetectionEngineConfig {
		engineConfig = data.SafeOutputs.ThreatDetection.EngineConfig
	}

	// Get the engine instance
	engine, err := c.getAgenticEngine(engineSetting)
	if err != nil {
		return []string{"      # Engine not found, skipping execution\n"}
	}

	// Build a detection engine config inheriting ID, Model, Version, Env, Config, Args, APITarget.
	// MaxTurns, Concurrency, UserAgent, Firewall, and Agent are intentionally omitted —
	// the detection job is a simple threat-analysis invocation and must never run as a
	// custom agent (no repo checkout, agent file unavailable).
	detectionEngineConfig := engineConfig
	if detectionEngineConfig == nil {
		detectionEngineConfig = &EngineConfig{ID: engineSetting}
	} else {
		detectionEngineConfig = &EngineConfig{
			ID:                 detectionEngineConfig.ID,
			Model:              detectionEngineConfig.Model,
			Version:            detectionEngineConfig.Version,
			Env:                detectionEngineConfig.Env,
			Config:             detectionEngineConfig.Config,
			Args:               detectionEngineConfig.Args,
			APITarget:          detectionEngineConfig.APITarget,
			MaxEffectiveTokens: detectionEngineConfig.MaxEffectiveTokens,
			HarnessScript:      detectionEngineConfig.HarnessScript,
			CopilotSDKDriver:   detectionEngineConfig.CopilotSDKDriver,
		}
	}
	if detectionEngineConfig.ID == "" {
		detectionEngineConfig.ID = engineSetting
	}

	// Apply enterprise and engine default detection models when no model was explicitly configured.
	// GetDefaultDetectionModel() returns a cost-effective model optimised for detection
	// (e.g. "gpt-5.1-codex-mini" for Copilot). Other engines return "" (no default).
	// This was accidentally removed in commit a93e36ea4 while fixing engine.agent propagation.
	if detectionEngineConfig.Model == "" {
		if defaultModel := compilerenv.ResolveDefaultDetectionModel(""); defaultModel != "" {
			detectionEngineConfig.Model = defaultModel
		} else if defaultModel := engine.GetDefaultDetectionModel(); defaultModel != "" {
			detectionEngineConfig.Model = defaultModel
		}
	}

	// Inherit APITarget from the main engine config for GHE/custom endpoints if not already set.
	// This ensures the threat detection AWF invocation receives the same --copilot-api-target
	// and GHE-specific domains in --allow-domains as the main agent AWF invocation.
	if detectionEngineConfig.APITarget == "" && data.EngineConfig != nil && data.EngineConfig.APITarget != "" {
		detectionEngineConfig.APITarget = data.EngineConfig.APITarget
	}

	// Create minimal WorkflowData for threat detection.
	// SandboxConfig with AWF enabled ensures the engine runs inside the firewall.
	// NetworkPermissions.Allowed is empty so no user-specified domains are added on top of
	// the engine's minimal detection domain list (see GetThreatDetectionAllowedDomains).
	// No MCP servers are configured for detection.
	// bash: ["*"] allows all shell commands — AWF's network firewall is the primary
	// constraint, so restricting individual bash commands inside the sandbox adds friction
	// without meaningful security benefit.
	threatDetectionData := &WorkflowData{
		Tools: map[string]any{
			"bash": []any{"*"},
		},
		SafeOutputs:       nil,
		EngineConfig:      detectionEngineConfig,
		AI:                engineSetting,
		Features:          data.Features,
		Permissions:       data.Permissions,
		CachedPermissions: data.CachedPermissions,
		IsDetectionRun:    true, // Mark as detection run for phase tagging
		NetworkPermissions: &NetworkPermissions{
			Allowed: []string{}, // no user-specified additional domains; engine provides its own minimal set
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Type: SandboxTypeAWF,
			},
		},
	}

	var steps []string

	// Install the engine in the detection job. The detection job runs on a separate fresh
	// runner where the agent's installed tools are not available, so we must install them here.
	installSteps := engine.GetInstallationSteps(threatDetectionData)

	// Ensure node is on PATH when the engine's execution wraps the CLI with a harness
	// script (see engineRequiresNodeHarness). The detection job does not go through
	// DetectRuntimeRequirements, so the setup must be emitted here explicitly. Guard
	// against engines whose install steps already bundle Setup Node.js (Claude/Codex
	// via BuildStandardNpmEngineInstallSteps) — a duplicate would trip
	// JobManager.ValidateDuplicateSteps and hard-fail the compile.
	if engineRequiresNodeHarness(engine) && !installStepsContainNodeSetup(installSteps) {
		for _, line := range GenerateNodeJsSetupStep() {
			steps = append(steps, line+"\n")
		}
	}

	for _, step := range installSteps {
		for _, line := range step {
			steps = append(steps, line+"\n")
		}
	}

	// Codex detection runs with no MCP tools, but still needs MCP gateway/config bootstrap
	// so config.toml includes the OpenAI proxy provider used by AWF API proxy mode.
	if engine.GetID() == "codex" {
		var mcpSetup strings.Builder
		if err := c.generateMCPSetup(&mcpSetup, threatDetectionData.Tools, engine, threatDetectionData); err == nil {
			for line := range strings.SplitSeq(mcpSetup.String(), "\n") {
				if line != "" {
					steps = append(steps, line+"\n")
				}
			}
		} else {
			threatLog.Printf("Failed to generate MCP setup for Codex detection; OpenAI proxy configuration may be incomplete: %v", err)
		}
	}

	logFile := "/tmp/gh-aw/threat-detection/detection.log"
	executionSteps := engine.GetExecutionSteps(threatDetectionData, logFile)
	for _, step := range executionSteps {
		for i, line := range step {
			// Prefix step IDs with "detection_" to avoid conflicts with agent job steps
			// (e.g., "agentic_execution" is already used by the main engine execution step)
			prefixed := strings.Replace(line, "id: agentic_execution", "id: detection_agentic_execution", 1)
			steps = append(steps, prefixed+"\n")
			// Inject the if condition and continue-on-error after the first line (- name:).
			// continue-on-error: true ensures that infrastructure failures (e.g. unhealthy
			// AWF container, Claude API errors) do not mark the detection job as failed.
			// The "Parse and conclude" step always runs (if: always()) and handles the
			// missing/incomplete detection log as parse_error in warn mode (exit 0).
			if i == 0 {
				steps = append(steps, fmt.Sprintf("        if: %s\n", detectionStepCondition))
				steps = append(steps, "        continue-on-error: true\n")
			}
		}
	}

	return steps
}

// getThreatDetectionEngineID returns the effective engine ID for the detection job.
// It mirrors threat-detection engine resolution: threat-detection.engine overrides main engine.
func (c *Compiler) getThreatDetectionEngineID(data *WorkflowData) string {
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil &&
		data.SafeOutputs.ThreatDetection.EngineConfig != nil &&
		data.SafeOutputs.ThreatDetection.EngineConfig.ID != "" {
		return data.SafeOutputs.ThreatDetection.EngineConfig.ID
	}

	mainEngineID := data.AI
	if mainEngineID == "" && data.EngineConfig != nil && data.EngineConfig.ID != "" {
		mainEngineID = data.EngineConfig.ID
	}

	if mainEngineID != "" {
		return mainEngineID
	}

	return "claude"
}

// buildWorkflowContextEnvVars creates environment variables for workflow context
func (c *Compiler) buildWorkflowContextEnvVars(data *WorkflowData) []string {
	workflowName := data.Name
	if workflowName == "" {
		workflowName = "Unnamed Workflow"
	}

	workflowDescription := data.Description
	if workflowDescription == "" {
		workflowDescription = "No description provided"
	}

	return []string{
		fmt.Sprintf("          WORKFLOW_NAME: %q\n", workflowName),
		fmt.Sprintf("          WORKFLOW_DESCRIPTION: %q\n", workflowDescription),
	}
}

// buildResultsParsingScriptRequire creates the parsing script that requires the .cjs module.
// The generated code wraps the require() and main() calls in a try/catch so that module load
// failures (e.g. parse_threat_detection_results.cjs not found, setup_globals.cjs missing) still
// set the detection_* outputs to a safe "warning" state instead of leaving them unset.  Unset
// outputs would cause downstream conditions that reference steps.detection_conclusion.outputs.*
// to evaluate to empty strings and could silently bypass the detection gate.
func (c *Compiler) buildResultsParsingScriptRequire() string {
	script := `try {
  const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
  setupGlobals(core, github, context, exec, io, getOctokit);
  const { main } = require('` + SetupActionDestination + `/parse_threat_detection_results.cjs');
  await main();
} catch (loadErr) {
  const continueOnError = process.env.GH_AW_DETECTION_CONTINUE_ON_ERROR !== 'false';
  const detectionExecutionFailed = process.env.DETECTION_AGENTIC_EXECUTION_OUTCOME === 'failure';
  const msg = 'ERR_SYSTEM: \u274C Unexpected error loading threat detection module: ' + (loadErr && loadErr.message ? loadErr.message : String(loadErr));
  core.error(msg);
  core.setOutput('reason', 'parse_error');
  if (continueOnError && !detectionExecutionFailed) {
    core.warning('\u26A0\uFE0F ' + msg);
    core.setOutput('conclusion', 'warning');
    core.setOutput('success', 'false');
  } else {
    core.setOutput('conclusion', 'failure');
    core.setOutput('success', 'false');
    core.setFailed(msg);
  }
}`

	return script
}

// buildCustomThreatDetectionSteps builds YAML steps from user-configured threat detection steps.
// It injects the detection guard condition into each step unless an explicit if: condition is
// already set, ensuring custom steps only run when the detection_guard determines that detection
// should proceed and preventing unexpected side effects in runs with no agent outputs to analyze.
func (c *Compiler) buildCustomThreatDetectionSteps(steps []any) []string {
	var result []string
	for _, step := range steps {
		if stepMap, ok := step.(map[string]any); ok {
			// Inject the detection guard condition unless the user already provided an if: condition.
			if _, hasIf := stepMap["if"]; !hasIf {
				// Clone the map to avoid mutating the original config.
				injected := make(map[string]any, safeAllocationCapacity(len(stepMap), 1))
				maps.Copy(injected, stepMap)
				injected["if"] = detectionStepCondition
				stepMap = injected
			}
			if stepYAML, err := ConvertStepToYAML(stepMap); err == nil {
				result = append(result, stepYAML)
			}
		}
	}
	return result
}

// buildUploadDetectionLogStep creates the step to upload the detection-artifact.
// In workflow_call context, the artifact name is prefixed to avoid name clashes when the
// same reusable workflow is called multiple times within a single workflow run.
// The prefix comes from the agent job output since the detection job depends on the agent job.
func (c *Compiler) buildUploadDetectionLogStep(data *WorkflowData) []string {
	detectionArtifactName := artifactPrefixExprForAgentDownstreamJob(data) + constants.DetectionArtifactName
	return []string{
		"      - name: Upload threat detection log\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/upload-artifact")),
		"        with:\n",
		"          name: " + detectionArtifactName + "\n",
		"          path: /tmp/gh-aw/threat-detection/detection.log\n",
		"          if-no-files-found: ignore\n",
	}
}

// buildWorkspaceCheckoutForDetectionStep creates a checkout step for the detection job.
// It runs only when the agent job produced a patch, so the detection engine can
// analyze code changes in the context of the surrounding codebase.
func (c *Compiler) buildWorkspaceCheckoutForDetectionStep(data *WorkflowData) []string {
	checkoutPin := getActionPin("actions/checkout")
	if checkoutPin == "" {
		threatLog.Print("No action pin found for actions/checkout, skipping workspace checkout step")
		return nil
	}

	steps := []string{
		"      - name: Checkout repository for patch context\n",
		fmt.Sprintf("        if: needs.%s.outputs.has_patch == 'true'\n", constants.AgentJobName),
		fmt.Sprintf("        uses: %s\n", checkoutPin),
		"        with:\n",
		"          persist-credentials: false\n",
	}

	threatLog.Print("Added conditional workspace checkout step for patch context")
	return steps
}

// buildDetectionJob creates a separate detection job that runs after the agent job.
// The job downloads the agent artifact to access output files, then runs all threat detection
// steps. It outputs detection_success and detection_conclusion for downstream jobs.
// Returns nil if threat detection is not configured.
func (c *Compiler) buildDetectionJob(data *WorkflowData) (*Job, error) {
	threatLog.Print("Building separate detection job")
	if data.SafeOutputs == nil || data.SafeOutputs.ThreatDetection == nil {
		threatLog.Print("Threat detection not configured, skipping detection job")
		return nil, nil
	}

	// When the engine is explicitly disabled and there are no custom steps,
	// there is nothing to run in the detection job — skip it entirely.
	// The detection job would only create an empty detection.log and the parser
	// would correctly fail with "No THREAT_DETECTION_RESULT found".
	if !IsDetectionJobEnabled(data.SafeOutputs) {
		threatLog.Print("Threat detection engine disabled with no custom steps, skipping detection job")
		return nil, nil
	}

	var steps []string

	// Add setup action steps (same as agent job - installs the agentic engine)
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		// For dev mode (local action path), checkout the actions folder first
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		// Detection job depends on agent job; reuse the agent's trace ID so all jobs share one OTLP trace
		detectionTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		detectionParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, detectionTraceID, detectionParentSpanID)...)
	}

	// Download agent output artifact to access output files (prompt.txt, agent_output.json, patches).
	// Use agent-downstream prefix since this job depends on the agent job.
	agentArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	steps = append(steps, buildAgentOutputDownloadSteps(agentArtifactPrefix, c.getActionPin)...)

	// Download experiment artifact so the detection agent can read the current variant assignments.
	// The experiment artifact is uploaded by the activation job.
	steps = append(steps, buildExperimentArtifactDownloadSteps(data, c.getActionPin)...)

	// Conditionally checkout the target repository so the detection engine can
	// analyze patches in the context of the surrounding codebase.
	steps = append(steps, c.buildWorkspaceCheckoutForDetectionStep(data)...)

	// Add all threat detection steps
	detectionStepsContent := c.buildDetectionJobSteps(data)
	steps = append(steps, detectionStepsContent...)

	// Build job outputs
	outputs := map[string]string{
		"detection_success":    "${{ steps.detection_conclusion.outputs.success }}",
		"detection_conclusion": "${{ steps.detection_conclusion.outputs.conclusion }}",
		"detection_reason":     "${{ steps.detection_conclusion.outputs.reason }}",
	}

	// Detection job depends on agent job and activation job (for trace ID)
	needs := []string{string(constants.AgentJobName), string(constants.ActivationJobName)}

	// Determine runs-on: use threat detection override if set, otherwise ubuntu-latest.
	// The detection job runs on a fresh runner separate from the agent job, so it does
	// not need the same custom runner as safe-outputs.
	runsOn := "runs-on: ubuntu-latest"
	if data.SafeOutputs.ThreatDetection.RunsOn != "" {
		runsOn = "runs-on: " + data.SafeOutputs.ThreatDetection.RunsOn
	}

	// Detection job condition: always run if agent job was not skipped AND produced outputs or a patch.
	// Skip the detection job entirely (result = 'skipped') when there is nothing to detect against,
	// so downstream jobs (safe_outputs) are also correctly skipped.
	alwaysFunc := BuildFunctionCall("always")
	agentNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)
	outputTypesNotEmpty := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.output_types", constants.AgentJobName)),
		BuildStringLiteral(""),
	)
	hasPatchTrue := BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.has_patch", constants.AgentJobName)),
		BuildStringLiteral("true"),
	)
	hasContent := BuildOr(outputTypesNotEmpty, hasPatchTrue)
	jobConditionNode := BuildAnd(BuildAnd(alwaysFunc, agentNotSkipped), hasContent)

	// When detection is expression-controlled, add the caller expression to the condition so
	// GitHub Actions skips the detection job at runtime when the expression evaluates to false.
	if data.SafeOutputs.ThreatDetection.EnabledExpr != nil {
		rawExpr := extractRawExpression(*data.SafeOutputs.ThreatDetection.EnabledExpr)
		jobConditionNode = BuildAnd(jobConditionNode, &ExpressionNode{Expression: rawExpr})
		threatLog.Printf("Detection job condition includes runtime expression: %s", rawExpr)
	}

	jobCondition := RenderCondition(jobConditionNode)

	// Determine permissions for the detection job.
	// - Always grant contents: read because the workspace checkout (for patch context)
	//   requires it, and contents: read is a minimal read-only permission.
	//   The checkout is conditional on has_patch at runtime, but permissions cannot
	//   be set conditionally in GitHub Actions.
	// - In dev/script mode, contents: read is also needed for the actions folder checkout.
	// - When permissions.copilot-requests is set to write, the detection job runs the Copilot CLI
	//   and requires copilot-requests: write for authentication.
	// - When the engine uses GitHub OIDC (WIF) auth, the detection job's api-proxy also needs
	//   to mint a GitHub OIDC token for the token exchange. Without id-token: write,
	//   ACTIONS_ID_TOKEN_REQUEST_URL/TOKEN are not set in the runner environment and the
	//   api-proxy returns HTTP 401 on every request (mirrors validateOIDCPermissions logic).
	// - When observability.otlp.github-app is configured without app-id/private-key
	//   credentials, id-token: write is also needed (mirrors validateOIDCPermissions).
	copilotRequestsEnabled := hasCopilotRequestsWritePermission(data)
	perms := NewPermissionsContentsRead()
	if copilotRequestsEnabled {
		perms.Set(PermissionCopilotRequests, PermissionWrite)
	}
	if data.EngineConfig != nil && data.EngineConfig.Auth != nil && data.EngineConfig.Auth.Type == "github-oidc" {
		perms.Set(PermissionIdToken, PermissionWrite)
	}
	if hasOTLPGitHubOIDCAuth(data.ParsedFrontmatter, data.RawFrontmatter) {
		perms.Set(PermissionIdToken, PermissionWrite)
	}
	permissions := perms.RenderToYAML()

	job := &Job{
		Name:        string(constants.DetectionJobName),
		Needs:       needs,
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(runsOn, "    "),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
	}

	threatLog.Printf("Built detection job with %d steps, depends on: %v", len(steps), needs)
	return job, nil
}
