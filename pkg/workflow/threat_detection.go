package workflow

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

//go:embed templates/threat_detection.md
var defaultThreatDetectionPrompt string

// ThreatDetectionConfig holds configuration for threat detection in agent output
type ThreatDetectionConfig struct {
	Prompt       string        `yaml:"prompt,omitempty"`        // Additional custom prompt instructions to append
	Steps        []any         `yaml:"steps,omitempty"`         // Array of extra job steps
	EngineConfig *EngineConfig `yaml:"engine-config,omitempty"` // Extended engine configuration for threat detection
}

// parseThreatDetectionConfig handles threat-detection configuration
func (c *Compiler) parseThreatDetectionConfig(outputMap map[string]any) *ThreatDetectionConfig {
	if configData, exists := outputMap["threat-detection"]; exists {
		// Handle boolean values
		if boolVal, ok := configData.(bool); ok {
			if !boolVal {
				// When explicitly disabled, return nil
				return nil
			}
			// When enabled as boolean, return empty config
			return &ThreatDetectionConfig{}
		}

		// Handle object configuration
		if configMap, ok := configData.(map[string]any); ok {
			// Check for enabled field
			if enabled, exists := configMap["enabled"]; exists {
				if enabledBool, ok := enabled.(bool); ok {
					if !enabledBool {
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

			// Parse engine field (supports both string and object formats)
			if engine, exists := configMap["engine"]; exists {
				// Handle string format
				if engineStr, ok := engine.(string); ok {
					threatConfig.EngineConfig = &EngineConfig{ID: engineStr}
				} else if engineObj, ok := engine.(map[string]any); ok {
					// Handle object format - use extractEngineConfig logic
					_, engineConfig := c.ExtractEngineConfig(map[string]any{"engine": engineObj})
					threatConfig.EngineConfig = engineConfig
				}
			}

			return threatConfig
		}
	}

	// Default behavior: enabled if any safe-outputs are configured
	return &ThreatDetectionConfig{}
}

// buildThreatDetectionJob creates the detection job
func (c *Compiler) buildThreatDetectionJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.ThreatDetection == nil {
		return nil, fmt.Errorf("threat detection is not enabled")
	}

	// Build steps using a more structured approach
	steps := c.buildThreatDetectionSteps(data, mainJobName)

	// Generate agent concurrency configuration (same as main agent job)
	agentConcurrency := GenerateJobConcurrencyConfig(data)

	job := &Job{
		Name:           constants.DetectionJobName,
		If:             "",
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    NewPermissionsReadAll().RenderToYAML(),
		Concurrency:    c.indentYAMLLines(agentConcurrency, "    "),
		TimeoutMinutes: 10,
		Steps:          steps,
		Needs:          []string{mainJobName},
	}

	return job, nil
}

// buildThreatDetectionSteps builds the steps for the threat detection job
func (c *Compiler) buildThreatDetectionSteps(data *WorkflowData, mainJobName string) []string {
	var steps []string

	// Step 1: Download agent artifacts
	steps = append(steps, c.buildDownloadArtifactStep()...)

	// Step 2: Echo agent outputs for debugging
	steps = append(steps, c.buildEchoAgentOutputsStep(mainJobName)...)

	// Step 3: Setup and run threat detection
	steps = append(steps, c.buildThreatDetectionAnalysisStep(data)...)

	// Step 4: Add custom steps if configured
	if len(data.SafeOutputs.ThreatDetection.Steps) > 0 {
		steps = append(steps, c.buildCustomThreatDetectionSteps(data.SafeOutputs.ThreatDetection.Steps)...)
	}

	// Step 5: Upload detection log artifact
	steps = append(steps, c.buildUploadDetectionLogStep()...)

	return steps
}

// buildDownloadArtifactStep creates the artifact download step
func (c *Compiler) buildDownloadArtifactStep() []string {
	var steps []string

	// Download prompt artifact
	steps = append(steps, buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName: "prompt.txt",
		DownloadPath: "/tmp/gh-aw/threat-detection/",
		SetupEnvStep: false,
		StepName:     "Download prompt artifact",
	})...)

	// Download agent output artifact
	steps = append(steps, buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName: "agent_output.json",
		DownloadPath: "/tmp/gh-aw/threat-detection/",
		SetupEnvStep: false,
		StepName:     "Download agent output artifact",
	})...)

	// Download patch artifact
	steps = append(steps, buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName: "aw.patch",
		DownloadPath: "/tmp/gh-aw/threat-detection/",
		SetupEnvStep: false,
		StepName:     "Download patch artifact",
	})...)

	return steps
}

// buildEchoAgentOutputsStep creates a step that echoes the agent outputs
func (c *Compiler) buildEchoAgentOutputsStep(mainJobName string) []string {
	return []string{
		"      - name: Echo agent output types\n",
		"        env:\n",
		fmt.Sprintf("          AGENT_OUTPUT_TYPES: ${{ needs.%s.outputs.output_types }}\n", mainJobName),
		"        run: |\n",
		"          echo \"Agent output-types: $AGENT_OUTPUT_TYPES\"\n",
	}
}

// buildThreatDetectionAnalysisStep creates the main threat analysis step
func (c *Compiler) buildThreatDetectionAnalysisStep(data *WorkflowData) []string {
	var steps []string

	// Setup step
	steps = append(steps, []string{
		"      - name: Setup threat detection\n",
		"        uses: actions/github-script@v8\n",
		"        env:\n",
	}...)
	steps = append(steps, c.buildWorkflowContextEnvVars(data)...)

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

	// Add the setup script
	setupScript := c.buildSetupScript()
	formattedSetupScript := FormatJavaScriptForYAML(setupScript)
	steps = append(steps, formattedSetupScript...)

	// Add a small shell step in YAML to ensure the output directory and log file exist
	steps = append(steps, []string{
		"      - name: Ensure threat-detection directory and log\n",
		"        run: |\n",
		"          mkdir -p /tmp/gh-aw/threat-detection\n",
		"          mkdir -p /tmp/gh-aw/agent\n",
		"          mkdir -p /tmp/gh-aw/.copilot/logs\n",
		"          touch /tmp/gh-aw/threat-detection/detection.log\n",
	}...)

	// Add engine execution steps
	steps = append(steps, c.buildEngineSteps(data)...)

	// Add results parsing step
	steps = append(steps, c.buildParsingStep()...)

	return steps
}

// buildSetupScript creates the setup portion
func (c *Compiler) buildSetupScript() string {
	// Build the JavaScript code with proper handling of backticks for markdown code blocks
	script := `const fs = require('fs');

// Check if prompt file exists
const promptPath = '/tmp/gh-aw/threat-detection/prompt.txt';
let promptFileInfo = 'No prompt file found';
if (fs.existsSync(promptPath)) {
  try {
    const stats = fs.statSync(promptPath);
    promptFileInfo = promptPath + ' (' + stats.size + ' bytes)';
    core.info('Prompt file found: ' + promptFileInfo);
  } catch (error) {
    core.warning('Failed to stat prompt file: ' + error.message);
  }
} else {
  core.info('No prompt file found at: ' + promptPath);
}

// Check if agent output file exists
const agentOutputPath = '/tmp/gh-aw/threat-detection/agent_output.json';
let agentOutputFileInfo = 'No agent output file found';
if (fs.existsSync(agentOutputPath)) {
  try {
    const stats = fs.statSync(agentOutputPath);
    agentOutputFileInfo = agentOutputPath + ' (' + stats.size + ' bytes)';
    core.info('Agent output file found: ' + agentOutputFileInfo);
  } catch (error) {
    core.warning('Failed to stat agent output file: ' + error.message);
  }
} else {
  core.info('No agent output file found at: ' + agentOutputPath);
}

// Check if patch file exists
const patchPath = '/tmp/gh-aw/threat-detection/aw.patch';
let patchFileInfo = 'No patch file found';
if (fs.existsSync(patchPath)) {
  try {
    const stats = fs.statSync(patchPath);
    patchFileInfo = patchPath + ' (' + stats.size + ' bytes)';
    core.info('Patch file found: ' + patchFileInfo);
  } catch (error) {
    core.warning('Failed to stat patch file: ' + error.message);
  }
} else {
  core.info('No patch file found at: ' + patchPath);
}

// Create threat detection prompt with embedded template
const templateContent = %s;
let promptContent = templateContent
  .replace(/{WORKFLOW_NAME}/g, process.env.WORKFLOW_NAME || 'Unnamed Workflow')
  .replace(/{WORKFLOW_DESCRIPTION}/g, process.env.WORKFLOW_DESCRIPTION || 'No description provided')
  .replace(/{WORKFLOW_PROMPT_FILE}/g, promptFileInfo)
  .replace(/{AGENT_OUTPUT_FILE}/g, agentOutputFileInfo)
  .replace(/{AGENT_PATCH_FILE}/g, patchFileInfo);

// Append custom prompt instructions if provided
const customPrompt = process.env.CUSTOM_PROMPT;
if (customPrompt) {
  promptContent += '\n\n## Additional Instructions\n\n' + customPrompt;
}

// Write prompt file
fs.mkdirSync('/tmp/gh-aw/aw-prompts', { recursive: true });
fs.writeFileSync('/tmp/gh-aw/aw-prompts/prompt.txt', promptContent);
core.exportVariable('GH_AW_PROMPT', '/tmp/gh-aw/aw-prompts/prompt.txt');

// Note: creation of /tmp/gh-aw/threat-detection and detection.log is handled by a separate shell step

// Write rendered prompt to step summary using HTML details/summary
await core.summary
  .addRaw('<details>\n<summary>Threat Detection Prompt</summary>\n\n' + '` + "`" + `` + "`" + `` + "`" + `` + "`" + `` + "`" + `` + "`" + `markdown\n' + promptContent + '\n' + '` + "`" + `` + "`" + `` + "`" + `` + "`" + `` + "`" + `` + "`" + `\n\n</details>\n')
  .write();

core.info('Threat detection setup completed');`

	return fmt.Sprintf(script, c.formatStringAsJavaScriptLiteral(defaultThreatDetectionPrompt))
}

// buildEngineSteps creates the engine execution steps
func (c *Compiler) buildEngineSteps(data *WorkflowData) []string {
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
		// Return a fallback if engine not found
		return []string{"      # Engine not found, skipping execution\n"}
	}

	// Create minimal WorkflowData for threat detection
	// Configure bash read tools for accessing the agent output file
	threatDetectionData := &WorkflowData{
		Tools: map[string]any{
			"bash": []any{"cat", "head", "tail", "wc", "grep", "ls", "jq"},
		},
		SafeOutputs:  nil,
		Network:      "",
		EngineConfig: engineConfig,
		AI:           engineSetting,
	}

	var steps []string

	// Add engine installation steps (includes Node.js setup for npm-based engines)
	installSteps := engine.GetInstallationSteps(threatDetectionData)
	for _, step := range installSteps {
		for _, line := range step {
			steps = append(steps, line+"\n")
		}
	}

	// Add verification step for engine installation
	steps = append(steps, c.buildEngineVerificationStep(engine)...)

	// Add verification step for prompt file
	steps = append(steps, c.buildPromptVerificationStep()...)

	// Add engine execution steps
	logFile := "/tmp/gh-aw/threat-detection/detection.log"
	executionSteps := engine.GetExecutionSteps(threatDetectionData, logFile)
	for _, step := range executionSteps {
		for _, line := range step {
			steps = append(steps, line+"\n")
		}
	}

	return steps
}

// buildEngineVerificationStep creates a step to verify the engine CLI is installed and working
func (c *Compiler) buildEngineVerificationStep(engine CodingAgentEngine) []string {
	versionCmd := engine.GetVersionCommand()
	if versionCmd == "" {
		// No version command available, skip verification
		return []string{}
	}

	return []string{
		"      - name: Verify engine installation\n",
		"        run: |\n",
		fmt.Sprintf("          echo \"Verifying %s installation...\"\n", engine.GetDisplayName()),
		fmt.Sprintf("          if ! command -v %s &> /dev/null; then\n", getCommandName(versionCmd)),
		fmt.Sprintf("            echo \"Error: %s command not found\"\n", getCommandName(versionCmd)),
		"            exit 1\n",
		"          fi\n",
		fmt.Sprintf("          %s\n", versionCmd),
		fmt.Sprintf("          echo \"%s is installed and working\"\n", engine.GetDisplayName()),
	}
}

// buildPromptVerificationStep creates a step to verify the prompt file exists and has content
func (c *Compiler) buildPromptVerificationStep() []string {
	return []string{
		"      - name: Verify prompt file\n",
		"        run: |\n",
		"          echo \"Verifying prompt file...\"\n",
		"          if [ ! -f /tmp/gh-aw/aw-prompts/prompt.txt ]; then\n",
		"            echo \"Error: Prompt file not found at /tmp/gh-aw/aw-prompts/prompt.txt\"\n",
		"            exit 1\n",
		"          fi\n",
		"          PROMPT_SIZE=$(wc -c < /tmp/gh-aw/aw-prompts/prompt.txt)\n",
		"          if [ \"$PROMPT_SIZE\" -eq 0 ]; then\n",
		"            echo \"Error: Prompt file is empty\"\n",
		"            exit 1\n",
		"          fi\n",
		"          echo \"Prompt file exists and has content ($PROMPT_SIZE bytes)\"\n",
	}
}

// getCommandName extracts the command name from a version command string
func getCommandName(versionCmd string) string {
	// Extract the first word from the version command (e.g., "copilot --version" -> "copilot")
	parts := strings.Split(versionCmd, " ")
	if len(parts) > 0 {
		return parts[0]
	}
	return versionCmd
}

// buildParsingStep creates the results parsing step
func (c *Compiler) buildParsingStep() []string {
	steps := []string{
		"      - name: Parse threat detection results\n",
		"        uses: actions/github-script@v8\n",
		"        with:\n",
		"          script: |\n",
	}

	parsingScript := c.buildResultsParsingScript()
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

// formatStringAsJavaScriptLiteral properly formats a Go string as a JavaScript template literal
func (c *Compiler) formatStringAsJavaScriptLiteral(s string) string {
	// Use template literals with proper escaping
	escaped := strings.ReplaceAll(s, "`", "\\`")
	escaped = strings.ReplaceAll(escaped, "${", "\\${")
	return "`" + escaped + "`"
}

// buildResultsParsingScript creates the results parsing portion
func (c *Compiler) buildResultsParsingScript() string {
	return `const fs = require('fs');
// Parse threat detection results
let verdict = { prompt_injection: false, secret_leak: false, malicious_patch: false, reasons: [] };

try {
  const outputPath = '/tmp/gh-aw/threat-detection/agent_output.json';
  if (fs.existsSync(outputPath)) {
    const outputContent = fs.readFileSync(outputPath, 'utf8');
    const lines = outputContent.split('\n');
    
    for (const line of lines) {
      const trimmedLine = line.trim();
      if (trimmedLine.startsWith('THREAT_DETECTION_RESULT:')) {
        const jsonPart = trimmedLine.substring('THREAT_DETECTION_RESULT:'.length);
        verdict = { ...verdict, ...JSON.parse(jsonPart) };
        break;
      }
    }
  }
} catch (error) {
  core.warning('Failed to parse threat detection results: ' + error.message);
}

core.info('Threat detection verdict: ' + JSON.stringify(verdict));

// Fail if threats detected
if (verdict.prompt_injection || verdict.secret_leak || verdict.malicious_patch) {
  const threats = [];
  if (verdict.prompt_injection) threats.push('prompt injection');
  if (verdict.secret_leak) threats.push('secret leak');
  if (verdict.malicious_patch) threats.push('malicious patch');
  
  const reasonsText = verdict.reasons && verdict.reasons.length > 0 
    ? '\\nReasons: ' + verdict.reasons.join('; ')
    : '';
  
  core.setFailed('❌ Security threats detected: ' + threats.join(', ') + reasonsText);
} else {
  core.info('✅ No security threats detected. Safe outputs may proceed.');
}`
}

// buildCustomThreatDetectionSteps adds custom user-defined steps
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
		"        if: always()\n",
		"        uses: actions/upload-artifact@v4\n",
		"        with:\n",
		"          name: threat-detection.log\n",
		"          path: /tmp/gh-aw/threat-detection/detection.log\n",
		"          if-no-files-found: ignore\n",
	}
}
