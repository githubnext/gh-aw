package workflow

import (
	"fmt"
)

// MissingToolConfig holds configuration for reporting missing tools or functionality
type MissingToolConfig struct {
	Max         int    `yaml:"max,omitempty"`          // Maximum number of missing tool reports (default: unlimited)
	GitHubToken string `yaml:"github-token,omitempty"` // GitHub token for this specific output type
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

	// Create the job
	job := &Job{
		Name:           "missing_tool",
		RunsOn:         "runs-on: ubuntu-latest",
		If:             "${{ always() }}",                    // Always run to capture missing tools
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
		missingToolConfig := &MissingToolConfig{} // Default: no max limit

		// Handle the case where configData is nil (missing-tool: with no value)
		if configData == nil {
			return missingToolConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse max (optional)
			if max, exists := configMap["max"]; exists {
				// Handle different numeric types that YAML parsers might return
				var maxInt int
				var validMax bool
				switch v := max.(type) {
				case int:
					maxInt = v
					validMax = true
				case int64:
					maxInt = int(v)
					validMax = true
				case uint64:
					maxInt = int(v)
					validMax = true
				case float64:
					maxInt = int(v)
					validMax = true
				}
				if validMax {
					missingToolConfig.Max = maxInt
				}
			}

			// Parse github-token
			if githubToken, exists := configMap["github-token"]; exists {
				if githubTokenStr, ok := githubToken.(string); ok {
					missingToolConfig.GitHubToken = githubTokenStr
				}
			}
		}

		return missingToolConfig
	}

	return nil
}
