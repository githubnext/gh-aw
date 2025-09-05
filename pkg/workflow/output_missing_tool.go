package workflow

import (
	"fmt"
)

// buildCreateOutputMissingToolJob creates the missing_tool job
func (c *Compiler) buildCreateOutputMissingToolJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.MissingTool == nil {
		return nil, fmt.Errorf("safe-outputs.missing-tool configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Record Missing Tool\n")
	steps = append(steps, "        id: missing_tool\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))

	// Pass the max configuration if set
	if data.SafeOutputs.MissingTool.Max > 0 {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_MISSING_TOOL_MAX: %d\n", data.SafeOutputs.MissingTool.Max))
	}

	steps = append(steps, "        with:\n")
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
		If:             "if: ${{ always() }}",                // Always run to capture missing tools
		Permissions:    "permissions:\n      contents: read", // Only needs read access for logging
		TimeoutMinutes: 5,                                    // Short timeout since it's just processing output
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
