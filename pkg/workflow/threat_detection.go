package workflow

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed templates/threat_detection.md
var defaultThreatDetectionPrompt string

// ThreatDetectionConfig holds configuration for threat detection in agent output
type ThreatDetectionConfig struct {
	Enabled bool     `yaml:"enabled,omitempty"`        // Whether threat detection is enabled
	Prompt  string   `yaml:"prompt,omitempty"`         // Custom prompt text content (defaults to bundled template)
	Steps   []any    `yaml:"steps,omitempty"`          // Array of extra job steps
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

	var steps []string

	// Step 1: Download agent output artifacts
	steps = append(steps, "      - name: Download agent output artifact\n")
	steps = append(steps, "        continue-on-error: true\n")
	steps = append(steps, "        uses: actions/download-artifact@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: agent_output.json\n")
	steps = append(steps, "          path: /tmp/threat-detection/\n")

	// Step 2: Setup threat detection environment and prompt file
	steps = append(steps, "      - name: Setup threat detection environment\n")
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          mkdir -p /tmp/threat-detection/prompts\n")
	steps = append(steps, "          \n")
	
	// Create threat detection prompt (use custom text if provided, otherwise use default)
	steps = append(steps, "          # Create threat detection prompt\n")
	steps = append(steps, "          cat > /tmp/threat-detection/prompts/detection.md << 'THREAT_DETECTION_EOF'\n")
	
	// Determine which prompt content to use
	var promptContent string
	if data.SafeOutputs.ThreatDetection.Prompt != "" {
		// Use custom prompt text provided inline
		promptContent = data.SafeOutputs.ThreatDetection.Prompt
	} else {
		// Use default embedded prompt
		promptContent = defaultThreatDetectionPrompt
	}
	
	// Include the prompt content
	for _, line := range strings.Split(promptContent, "\n") {
		steps = append(steps, "          "+line+"\n")
	}
	steps = append(steps, "          THREAT_DETECTION_EOF\n")

	steps = append(steps, "          \n")
	steps = append(steps, "          # Prepare agent output files for analysis\n")
	steps = append(steps, fmt.Sprintf("          AGENT_OUTPUT_TEXT=\"${{ needs.%s.outputs.text }}\"\n", mainJobName))
	steps = append(steps, fmt.Sprintf("          AGENT_OUTPUT_PATCH=\"${{ needs.%s.outputs.patch }}\"\n", mainJobName))
	steps = append(steps, "          \n")
	steps = append(steps, "          # Replace placeholders in detection prompt\n")
	steps = append(steps, "          sed -i \"s/{AGENT_OUTPUT}/${AGENT_OUTPUT_TEXT//\\/\\\\/}/g\" /tmp/threat-detection/prompts/detection.md\n")
	steps = append(steps, "          sed -i \"s/{AGENT_PATCH}/${AGENT_OUTPUT_PATCH//\\/\\\\/}/g\" /tmp/threat-detection/prompts/detection.md\n")
	steps = append(steps, "          \n")
	steps = append(steps, "          echo \"GITHUB_AW_PROMPT=/tmp/threat-detection/prompts/detection.md\" >> $GITHUB_ENV\n")

	// Step 3: Get the agentic engine and generate its execution steps
	engineSetting := data.AI
	if data.EngineConfig != nil {
		engineSetting = data.EngineConfig.ID
	}
	if engineSetting == "" {
		engineSetting = "claude" // Default engine
	}

	// Get the engine instance
	engine, err := c.getAgenticEngine(engineSetting)
	if err != nil {
		return nil, fmt.Errorf("failed to get agentic engine for threat detection: %w", err)
	}

	// Create a minimal WorkflowData for threat detection (no tools, no network, no safe-outputs)
	threatDetectionData := &WorkflowData{
		MarkdownContent: "", // The prompt file will be used instead
		Tools:           map[string]any{}, // No tools for threat detection
		SafeOutputs:     nil, // No safe-outputs for threat detection
		Network:         "", // No network access
		EngineConfig:    data.EngineConfig, // Use same engine config
		AI:              engineSetting,
	}

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

	// Step 4: Parse threat detection results
	steps = append(steps, "      - name: Parse Threat Detection Results\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add the embedded threat detection parsing script
	formattedScript := FormatJavaScriptForYAML(parseThreatDetectionScript)
	steps = append(steps, formattedScript...)

	// Add any custom steps from the threat detection configuration
	if len(data.SafeOutputs.ThreatDetection.Steps) > 0 {
		for _, step := range data.SafeOutputs.ThreatDetection.Steps {
			if stepMap, ok := step.(map[string]any); ok {
				stepYAML, err := c.convertStepToYAML(stepMap)
				if err != nil {
					return nil, fmt.Errorf("failed to convert custom threat detection step to YAML: %w", err)
				}
				steps = append(steps, stepYAML)
			}
		}
	}

	// Determine the job condition for command workflows
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		jobCondition = commandConditionStr
	} else {
		jobCondition = "" // No conditional execution
	}

	job := &Job{
		Name:           "detection",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions: read-all",
		TimeoutMinutes: 10, // 10-minute timeout
		Steps:          steps,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}



