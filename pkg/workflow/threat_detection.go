package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var threatLog = logger.New("workflow:threat_detection")

// ThreatDetectionConfig holds configuration for threat detection in agent output
type ThreatDetectionConfig struct {
	Prompt         string        `yaml:"prompt,omitempty"`        // Additional custom prompt instructions to append
	Steps          []any         `yaml:"steps,omitempty"`         // Array of extra job steps
	EngineConfig   *EngineConfig `yaml:"engine-config,omitempty"` // Extended engine configuration for threat detection
	EngineDisabled bool          `yaml:"-"`                       // Internal flag: true when engine is explicitly set to false
	RunsOn         string        `yaml:"runs-on,omitempty"`       // Runner override for the detection job
}

// HasRunnableDetection reports whether this config will produce a detection job
// that actually executes. Returns false when the engine is disabled and no
// custom steps are configured, since the job would have nothing to run.
func (td *ThreatDetectionConfig) HasRunnableDetection() bool {
	return !td.EngineDisabled || len(td.Steps) > 0
}

// IsDetectionJobEnabled reports whether a detection job should be created for
// the given safe-outputs configuration. This is the single source of truth
// used by all codepaths that decide whether to create, depend on, or reference
// the detection job.
func IsDetectionJobEnabled(so *SafeOutputsConfig) bool {
	return so != nil && so.ThreatDetection != nil && so.ThreatDetection.HasRunnableDetection()
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

		// Handle object configuration
		if configMap, ok := configData.(map[string]any); ok {
			// Check for enabled field
			if enabled, exists := configMap["enabled"]; exists {
				if enabledBool, ok := enabled.(bool); ok {
					if !enabledBool {
						threatLog.Print("Threat detection disabled via enabled field")
						// When explicitly disabled, return nil
						return nil
					}
				}
			}

			// Build the config (enabled by default when object is provided)
			threatConfig := &ThreatDetectionConfig{}

			// Parse prompt field
			if prompt, exists := configMap["prompt"]; exists {
				if promptStr, ok := prompt.(string); ok {
					threatConfig.Prompt = promptStr
				}
			}

			// Parse steps field
			if steps, exists := configMap["steps"]; exists {
				if stepsArray, ok := steps.([]any); ok {
					threatConfig.Steps = stepsArray
				}
			}

			// Parse runs-on field
			if runOn, exists := configMap["runs-on"]; exists {
				if runOnStr, ok := runOn.(string); ok {
					threatConfig.RunsOn = runOnStr
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

			threatLog.Printf("Threat detection configured with custom prompt: %v, custom steps: %v", threatConfig.Prompt != "", len(threatConfig.Steps) > 0)
			return threatConfig
		}
	}

	// Default behavior: enabled if any safe-outputs are configured
	threatLog.Print("Using default threat detection configuration")
	return &ThreatDetectionConfig{}
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

	// Step 0: Pull AWF container images - the detection engine runs inside AWF (firewall),
	// so pre-pulling the containers speeds up execution and avoids on-demand pulls.
	steps = append(steps, c.buildPullAWFContainersStep(data)...)

	// Step 1: Detection guard - determines whether detection should run
	steps = append(steps, c.buildDetectionGuardStep()...)

	// Step 2: Clear MCP configuration files so the detection engine runs without MCP servers
	steps = append(steps, c.buildClearMCPConfigStep()...)

	// Step 3: Prepare files - copies agent output files to expected paths
	steps = append(steps, c.buildPrepareDetectionFilesStep()...)

	// Step 4: Setup threat detection (github-script)
	steps = append(steps, c.buildThreatDetectionAnalysisStep(data)...)

	// Step 5: Engine execution (AWF, no network)
	steps = append(steps, c.buildDetectionEngineExecutionStep(data)...)

	// Step 6: Custom steps if configured
	if len(data.SafeOutputs.ThreatDetection.Steps) > 0 {
		steps = append(steps, c.buildCustomThreatDetectionSteps(data.SafeOutputs.ThreatDetection.Steps)...)
	}

	// Step 7: Upload detection-artifact
	steps = append(steps, c.buildUploadDetectionLogStep(data)...)

	// Step 8: Parse results, log extensively, and set job conclusion (single JS step)
	steps = append(steps, c.buildDetectionConclusionStep()...)

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
		"      - name: Clear MCP configuration for detection\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		"        run: |\n",
		"          rm -f /tmp/gh-aw/mcp-config/mcp-servers.json\n",
		"          rm -f /home/runner/.copilot/mcp-config.json\n",
		"          rm -f \"$GITHUB_WORKSPACE/.gemini/settings.json\"\n",
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
func (c *Compiler) buildDetectionConclusionStep() []string {
	steps := []string{
		"      - name: Parse and conclude threat detection\n",
		"        id: detection_conclusion\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")),
		"        env:\n",
		"          RUN_DETECTION: ${{ steps.detection_guard.outputs.run_detection }}\n",
		"        with:\n",
		"          script: |\n",
	}

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
		fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")),
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
setupGlobals(core, github, context, exec, io);
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

	// Determine which engine to use - threat detection engine if specified, otherwise main engine
	engineSetting := data.AI
	engineConfig := data.EngineConfig

	// Check if threat detection has its own engine configuration
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil {
		if data.SafeOutputs.ThreatDetection.EngineConfig != nil {
			engineConfig = data.SafeOutputs.ThreatDetection.EngineConfig
		}
	}

	// Use engine config ID if available
	if engineConfig != nil {
		engineSetting = engineConfig.ID
	}
	if engineSetting == "" {
		engineSetting = "claude"
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
			ID:        detectionEngineConfig.ID,
			Model:     detectionEngineConfig.Model,
			Version:   detectionEngineConfig.Version,
			Env:       detectionEngineConfig.Env,
			Config:    detectionEngineConfig.Config,
			Args:      detectionEngineConfig.Args,
			APITarget: detectionEngineConfig.APITarget,
		}
	}

	// Apply the engine's default detection model when no model was explicitly configured.
	// GetDefaultDetectionModel() returns a cost-effective model optimised for detection
	// (e.g. "gpt-5.1-codex-mini" for Copilot). Other engines return "" (no default).
	// This was accidentally removed in commit a93e36ea4 while fixing engine.agent propagation.
	if detectionEngineConfig.Model == "" {
		if defaultModel := engine.GetDefaultDetectionModel(); defaultModel != "" {
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
		SafeOutputs:    nil,
		EngineConfig:   detectionEngineConfig,
		AI:             engineSetting,
		Features:       data.Features,
		IsDetectionRun: true, // Mark as detection run for phase tagging
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
	for _, step := range installSteps {
		for _, line := range step {
			steps = append(steps, line+"\n")
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
			// Inject the if condition after the first line (- name:)
			if i == 0 {
				steps = append(steps, fmt.Sprintf("        if: %s\n", detectionStepCondition))
			}
		}
	}

	return steps
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

// buildResultsParsingScriptRequire creates the parsing script that requires the .cjs module
func (c *Compiler) buildResultsParsingScriptRequire() string {
	// Build a simple require statement that calls the main function
	script := `const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
setupGlobals(core, github, context, exec, io);
const { main } = require('` + SetupActionDestination + `/parse_threat_detection_results.cjs');
await main();`

	return script
}

// buildCustomThreatDetectionSteps builds YAML steps from user-configured threat detection steps.
func (c *Compiler) buildCustomThreatDetectionSteps(steps []any) []string {
	var result []string
	for _, step := range steps {
		if stepMap, ok := step.(map[string]any); ok {
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
		fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")),
		"        with:\n",
		"          name: " + detectionArtifactName + "\n",
		"          path: /tmp/gh-aw/threat-detection/detection.log\n",
		"          if-no-files-found: ignore\n",
	}
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
		steps = append(steps, c.generateSetupStep(setupActionRef, SetupActionDestination, false)...)
	}

	// Download agent output artifact to access output files (prompt.txt, agent_output.json, patches).
	// Use agent-downstream prefix since this job depends on the agent job.
	agentArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	steps = append(steps, buildAgentOutputDownloadSteps(agentArtifactPrefix)...)

	// Add all threat detection steps
	detectionStepsContent := c.buildDetectionJobSteps(data)
	steps = append(steps, detectionStepsContent...)

	// Build job outputs
	outputs := map[string]string{
		"detection_success":    "${{ steps.detection_conclusion.outputs.success }}",
		"detection_conclusion": "${{ steps.detection_conclusion.outputs.conclusion }}",
	}

	// Detection job depends on agent job
	needs := []string{string(constants.AgentJobName)}

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
	jobCondition := RenderCondition(jobConditionNode)

	// Determine permissions for the detection job.
	// - In dev/script mode, need contents: read if the actions folder checkout is needed.
	// - When the copilot-requests feature is enabled, the detection job runs the Copilot CLI
	//   and requires copilot-requests: write for authentication.
	needsContentsRead := (c.actionMode.IsDev() || c.actionMode.IsScript()) && len(c.generateCheckoutActionsFolder(data)) > 0
	copilotRequestsEnabled := isFeatureEnabled(constants.CopilotRequestsFeatureFlag, data)
	var perms *Permissions
	if needsContentsRead {
		perms = NewPermissionsContentsRead()
	} else {
		perms = NewPermissions()
	}
	if copilotRequestsEnabled {
		perms.Set(PermissionCopilotRequests, PermissionWrite)
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
