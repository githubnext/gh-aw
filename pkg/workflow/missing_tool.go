package workflow

import (
	"fmt"
)

// MissingToolConfig holds configuration for reporting missing tools or functionality
type MissingToolConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// buildCreateOutputMissingToolJob creates the missing_tool job
func (c *Compiler) buildCreateOutputMissingToolJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.MissingTool == nil {
		return nil, fmt.Errorf("safe-outputs.missing-tool configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Record Missing Tool\n")
	steps = append(steps, "        id: missing_tool\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))

	// Pass the max configuration if set
	if data.SafeOutputs.MissingTool.Max > 0 {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_MISSING_TOOL_MAX: %d\n", data.SafeOutputs.MissingTool.Max))
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.MissingTool != nil {
		token = data.SafeOutputs.MissingTool.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(missingToolScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"tools_reported": "${{ steps.missing_tool.outputs.tools_reported }}",
		"total_count":    "${{ steps.missing_tool.outputs.total_count }}",
	}

	// Build the job condition using BuildSafeOutputType
	jobCondition := BuildSafeOutputType("missing-tool").Render()

	// Create the job
	job := &Job{
		Name:           "missing_tool",
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		If:             jobCondition,
		Permissions:    "permissions:\n      contents: read", // Only needs read access for logging
		TimeoutMinutes: 5,                                    // Short timeout since it's just processing output
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parseMissingToolConfig handles missing-tool configuration
func (c *Compiler) parseMissingToolConfig(outputMap map[string]any) *MissingToolConfig {
	if configData, exists := outputMap["missing-tool"]; exists {
		// Handle the case where configData is false (explicitly disabled)
		if configBool, ok := configData.(bool); ok && !configBool {
			return nil
		}

		missingToolConfig := &MissingToolConfig{} // Default: no max limit

		// Handle the case where configData is nil (missing-tool: with no value)
		if configData == nil {
			return missingToolConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &missingToolConfig.BaseSafeOutputConfig)
		}

		return missingToolConfig
	}

	return nil
}
