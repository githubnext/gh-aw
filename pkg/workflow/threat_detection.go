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
	Enabled      bool          `yaml:"enabled,omitempty"`       // Whether threat detection is enabled
	Prompt       string        `yaml:"prompt,omitempty"`        // Additional custom prompt instructions to append
	Steps        []any         `yaml:"steps,omitempty"`         // Array of extra job steps
	Engine       string        `yaml:"engine,omitempty"`        // Engine ID for threat detection (overrides main engine)
	EngineConfig *EngineConfig `yaml:"engine-config,omitempty"` // Extended engine configuration for threat detection
}

// parseThreatDetectionConfig handles threat-detection configuration
func (c *Compiler) parseThreatDetectionConfig(outputMap map[string]any) *ThreatDetectionConfig {
	if configData, exists := outputMap["threat-detection"]; exists {
		// Handle boolean values
		if boolVal, ok := configData.(bool); ok {
			return &ThreatDetectionConfig{
				Enabled: boolVal,
			}
		}

		// Handle object configuration
		if configMap, ok := configData.(map[string]any); ok {
			threatConfig := &ThreatDetectionConfig{
				Enabled: true, // Default to enabled when object is provided
			}

			// Parse enabled field
			if enabled, exists := configMap["enabled"]; exists {
				if enabledBool, ok := enabled.(bool); ok {
					threatConfig.Enabled = enabledBool
				}
			}

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
					threatConfig.Engine = engineStr
					threatConfig.EngineConfig = &EngineConfig{ID: engineStr}
				} else if engineObj, ok := engine.(map[string]any); ok {
					// Handle object format - use extractEngineConfig logic
					engineSetting, engineConfig := c.extractEngineConfig(map[string]any{"engine": engineObj})
					threatConfig.Engine = engineSetting
					threatConfig.EngineConfig = engineConfig
				}
			}

			return threatConfig
		}
	}

	// Default behavior: enabled if any safe-outputs are configured
	return &ThreatDetectionConfig{
		Enabled: true,
	}
}

// buildThreatDetectionJob creates the detection job
func (c *Compiler) buildThreatDetectionJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.ThreatDetection == nil || !data.SafeOutputs.ThreatDetection.Enabled {
		return nil, fmt.Errorf("threat detection is not enabled")
	}

	// Build steps using a more structured approach
	steps := c.buildThreatDetectionSteps(data, mainJobName)

	job := &Job{
		Name:           constants.DetectionJobName,
		If:             "",
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions: read-all",
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

	// Step 2: Setup and run threat detection
	steps = append(steps, c.buildThreatDetectionAnalysisStep(data, mainJobName)...)

	// Step 3: Add custom steps if configured
	if len(data.SafeOutputs.ThreatDetection.Steps) > 0 {
		steps = append(steps, c.buildCustomThreatDetectionSteps(data.SafeOutputs.ThreatDetection.Steps)...)
	}

	return steps
}

// buildDownloadArtifactStep creates the artifact download step
func (c *Compiler) buildDownloadArtifactStep() []string {
	return []string{
		"      - name: Download agent output artifact\n",
		"        continue-on-error: true\n",
		"        uses: actions/download-artifact@v5\n",
		"        with:\n",
		"          name: agent_output.json\n",
		"          path: /tmp/threat-detection/\n",
		"      - name: Download patch artifact\n",
		"        continue-on-error: true\n",
		"        uses: actions/download-artifact@v5\n",
		"        with:\n",
		"          name: aw.patch\n",
		"          path: /tmp/threat-detection/\n",
	}
}

// buildThreatDetectionAnalysisStep creates the main threat analysis step
func (c *Compiler) buildThreatDetectionAnalysisStep(data *WorkflowData, mainJobName string) []string {
	var steps []string

	// Setup step
	steps = append(steps, []string{
		"      - name: Setup threat detection\n",
		"        uses: actions/github-script@v8\n",
		"        env:\n",
		fmt.Sprintf("          AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName),
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
		"          mkdir -p /tmp/threat-detection\n",
		"          touch /tmp/threat-detection/detection.log\n",
	}...)

	// Add engine execution steps
	steps = append(steps, c.buildEngineSteps(data)...)

	// Add results parsing step
	steps = append(steps, c.buildParsingStep()...)

	return steps
}

// buildSetupScript creates the setup portion
func (c *Compiler) buildSetupScript() string {
	return fmt.Sprintf(`const fs = require('fs');

// Read patch file if it exists
let patchContent = '';
const patchPath = '/tmp/threat-detection/aw.patch';
if (fs.existsSync(patchPath)) {
  try {
    patchContent = fs.readFileSync(patchPath, 'utf8');
    core.info('Patch file loaded: ' + patchPath);
  } catch (error) {
    core.warning('Failed to read patch file: ' + error.message);
  }
} else {
  core.info('No patch file found at: ' + patchPath);
}

// Create threat detection prompt with embedded template
const templateContent = %s;
let promptContent = templateContent
  .replace(/{WORKFLOW_NAME}/g, process.env.WORKFLOW_NAME || 'Unnamed Workflow')
  .replace(/{WORKFLOW_DESCRIPTION}/g, process.env.WORKFLOW_DESCRIPTION || 'No description provided')
  .replace(/{WORKFLOW_MARKDOWN}/g, process.env.WORKFLOW_MARKDOWN || 'No content provided')
  .replace(/{AGENT_OUTPUT}/g, process.env.AGENT_OUTPUT || '')
  .replace(/{AGENT_PATCH}/g, patchContent);

// Append custom prompt instructions if provided
const customPrompt = process.env.CUSTOM_PROMPT;
if (customPrompt) {
  promptContent += '\n\n## Additional Instructions\n\n' + customPrompt;
}

// Write prompt file
fs.mkdirSync('/tmp/aw-prompts', { recursive: true });
fs.writeFileSync('/tmp/aw-prompts/prompt.txt', promptContent);
core.exportVariable('GITHUB_AW_PROMPT', '/tmp/aw-prompts/prompt.txt');

// Note: creation of /tmp/threat-detection and detection.log is handled by a separate shell step

// Write rendered prompt to step summary
await core.summary
  .addHeading('Threat Detection Prompt', 2)
  .addRaw('\n')
  .addCodeBlock(promptContent, 'text')
  .write();

core.info('Threat detection setup completed');`,
		c.formatStringAsJavaScriptLiteral(defaultThreatDetectionPrompt))
}

// buildEngineSteps creates the engine execution steps
func (c *Compiler) buildEngineSteps(data *WorkflowData) []string {
	// Determine which engine to use - threat detection engine if specified, otherwise main engine
	engineSetting := data.AI
	engineConfig := data.EngineConfig

	// Check if threat detection has its own engine configuration
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil {
		if data.SafeOutputs.ThreatDetection.Engine != "" {
			engineSetting = data.SafeOutputs.ThreatDetection.Engine
			engineConfig = data.SafeOutputs.ThreatDetection.EngineConfig
		}
	}

	// Fall back to main engine config if no threat detection override
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
	// Empty tools map and nil SafeOutputs ensures:
	// 1. No MCP servers are configured
	// 2. No --allow-tool arguments are generated (all tools denied)
	threatDetectionData := &WorkflowData{
		Tools:        map[string]any{},
		SafeOutputs:  nil,
		Network:      "",
		EngineConfig: engineConfig,
		AI:           engineSetting,
	}

	var steps []string

	// Add engine installation steps
	installSteps := engine.GetInstallationSteps(threatDetectionData)
	for _, step := range installSteps {
		for _, line := range step {
			steps = append(steps, line+"\n")
		}
	}

	// Add engine execution steps
	logFile := "/tmp/threat-detection/detection.log"
	executionSteps := engine.GetExecutionSteps(threatDetectionData, logFile)
	for _, step := range executionSteps {
		for _, line := range step {
			steps = append(steps, line+"\n")
		}
	}

	return steps
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

	workflowMarkdown := data.MarkdownContent
	if workflowMarkdown == "" {
		workflowMarkdown = "No content provided"
	}

	return []string{
		fmt.Sprintf("          WORKFLOW_NAME: %q\n", workflowName),
		fmt.Sprintf("          WORKFLOW_DESCRIPTION: %q\n", workflowDescription),
		fmt.Sprintf("          WORKFLOW_MARKDOWN: %q\n", workflowMarkdown),
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
	return `// Parse threat detection results
let verdict = { prompt_injection: false, secret_leak: false, malicious_patch: false, reasons: [] };

try {
  const outputPath = '/tmp/threat-detection/agent_output.json';
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
