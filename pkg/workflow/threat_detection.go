package workflow

import (
	"fmt"
	"strings"

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

// buildInlineDetectionSteps builds the threat detection steps to be inlined in the agent job.
// These steps run after the output collection step (collect_output) and analyze agent output
// for threats using the same agentic engine with sandbox.agent and fully blocked network.
func (c *Compiler) buildInlineDetectionSteps(data *WorkflowData) []string {
	threatLog.Print("Building inline threat detection steps for agent job")
	if data.SafeOutputs == nil || data.SafeOutputs.ThreatDetection == nil {
		return nil
	}

	var steps []string

	// Comment separator
	steps = append(steps, "      # --- Threat Detection (inline) ---\n")

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

	// Step 7: Parse threat detection results
	steps = append(steps, c.buildParsingStep()...)

	// Step 8: Upload detection log artifact
	steps = append(steps, c.buildUploadDetectionLogStep()...)

	// Step 9: Detection conclusion - sets final detection_success and detection_conclusion outputs
	steps = append(steps, c.buildDetectionConclusionStep()...)

	threatLog.Printf("Generated %d inline detection step lines", len(steps))
	return steps
}

// buildDetectionGuardStep creates a guard step that checks if detection should run.
// Uses always() to run even if the agent step failed (collect_output also runs always()).
func (c *Compiler) buildDetectionGuardStep() []string {
	return []string{
		"      - name: Check if detection needed\n",
		"        id: detection_guard\n",
		"        if: always()\n",
		"        env:\n",
		"          OUTPUT_TYPES: ${{ steps.collect_output.outputs.output_types }}\n",
		"          HAS_PATCH: ${{ steps.collect_output.outputs.has_patch }}\n",
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

// buildClearMCPConfigStep creates a step that removes MCP configuration files written by
// the main agent job. This ensures the detection engine runs without any MCP servers,
// even if the main agent had MCP servers configured.
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
// Since detection now runs inline in the agent job, files are already local and just need copying.
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
		"          echo \"Prepared threat detection files:\"\n",
		"          ls -la /tmp/gh-aw/threat-detection/ 2>/dev/null || true\n",
	}
}

// buildDetectionConclusionStep creates a step that sets the final detection outputs.
// Runs with always() to ensure outputs are set regardless of detection step outcomes.
func (c *Compiler) buildDetectionConclusionStep() []string {
	return []string{
		"      - name: Set detection conclusion\n",
		"        id: detection_conclusion\n",
		"        if: always()\n",
		"        env:\n",
		"          RUN_DETECTION: ${{ steps.detection_guard.outputs.run_detection }}\n",
		"          DETECTION_SUCCESS: ${{ steps.parse_detection_results.outputs.success }}\n",
		"        run: |\n",
		"          if [[ \"$RUN_DETECTION\" != \"true\" ]]; then\n",
		"            echo \"conclusion=skipped\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"success=true\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"Detection was not needed, marking as skipped\"\n",
		"          elif [[ \"$DETECTION_SUCCESS\" == \"true\" ]]; then\n",
		"            echo \"conclusion=success\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"success=true\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"Detection passed successfully\"\n",
		"          else\n",
		"            echo \"conclusion=failure\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"success=false\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"Detection found issues\"\n",
		"          fi\n",
	}
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

	// Add HAS_PATCH environment variable from collect_output step (inline in agent job)
	steps = append(steps, "          HAS_PATCH: ${{ steps.collect_output.outputs.has_patch }}\n")

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

	// Apply default detection model if the engine provides one and no model is specified
	detectionEngineConfig := engineConfig
	if detectionEngineConfig == nil {
		detectionEngineConfig = &EngineConfig{ID: engineSetting}
	} else {
		detectionEngineConfig = &EngineConfig{
			ID:      detectionEngineConfig.ID,
			Model:   detectionEngineConfig.Model,
			Version: detectionEngineConfig.Version,
			Env:     detectionEngineConfig.Env,
			Config:  detectionEngineConfig.Config,
			Args:    detectionEngineConfig.Args,
		}
	}

	// Create minimal WorkflowData for threat detection with network fully blocked.
	// SandboxConfig with AWF enabled ensures the engine runs inside the firewall.
	// NetworkPermissions with empty Allowed list blocks all network egress.
	// No MCP servers are configured for detection.
	threatDetectionData := &WorkflowData{
		Tools: map[string]any{
			"bash": []any{"cat", "head", "tail", "wc", "grep", "ls", "jq"},
		},
		SafeOutputs:  nil,
		EngineConfig: detectionEngineConfig,
		AI:           engineSetting,
		Features:     data.Features,
		NetworkPermissions: &NetworkPermissions{
			Allowed: []string{}, // deny-all: no network access
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Type: SandboxTypeAWF,
			},
		},
	}

	var steps []string

	// Skip engine installation - engine is already installed in the agent job.
	// Only generate execution steps.
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

// buildParsingStep creates the results parsing step
func (c *Compiler) buildParsingStep() []string {
	steps := []string{
		"      - name: Parse threat detection results\n",
		"        id: parse_detection_results\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")),
		"        with:\n",
		"          script: |\n",
	}

	// Use require() to load script from the separate .cjs file
	parsingScript := c.buildResultsParsingScriptRequire()
	formattedParsingScript := FormatJavaScriptForYAML(parsingScript)
	steps = append(steps, formattedParsingScript...)

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

// buildUploadDetectionLogStep creates the step to upload the detection log
func (c *Compiler) buildCustomThreatDetectionSteps(steps []any) []string {
	var result []string
	for _, step := range steps {
		if stepMap, ok := step.(map[string]any); ok {
			if stepYAML, err := c.convertStepToYAML(stepMap); err == nil {
				result = append(result, stepYAML)
			}
		}
	}
	return result
}

// buildUploadDetectionLogStep creates the step to upload the detection log
func (c *Compiler) buildUploadDetectionLogStep() []string {
	return []string{
		"      - name: Upload threat detection log\n",
		fmt.Sprintf("        if: %s\n", detectionStepCondition),
		fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")),
		"        with:\n",
		"          name: threat-detection.log\n",
		"          path: /tmp/gh-aw/threat-detection/detection.log\n",
		"          if-no-files-found: ignore\n",
	}
}
