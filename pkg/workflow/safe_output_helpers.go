package workflow

import (
	"fmt"
)

// GitHubScriptStepConfig holds configuration for building a GitHub Script step
type GitHubScriptStepConfig struct {
	// Step metadata
	StepName string // e.g., "Create Output Issue"
	StepID   string // e.g., "create_issue"

	// Main job reference for agent output
	MainJobName string

	// Environment variables specific to this safe output type
	// These are added after GITHUB_AW_AGENT_OUTPUT
	CustomEnvVars []string

	// JavaScript script constant to format and include
	Script string

	// Token configuration (passed to addSafeOutputGitHubTokenForConfig)
	Token string
}

// buildGitHubScriptStep creates a GitHub Script step with common scaffolding
// This extracts the repeated pattern found across safe output job builders
func (c *Compiler) buildGitHubScriptStep(data *WorkflowData, config GitHubScriptStepConfig) []string {
	var steps []string

	// Add step to download agent output artifact
	steps = append(steps, "      - name: Download agent output artifact\n")
	steps = append(steps, "        continue-on-error: true\n")
	steps = append(steps, "        uses: actions/download-artifact@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          name: ${{ needs.%s.outputs.output-artifact }}\n", config.MainJobName))
	steps = append(steps, "          path: /tmp/gh-aw/safe-outputs/\n")

	// Step name and metadata
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Environment variables section - only add if there are custom env vars
	if len(config.CustomEnvVars) > 0 || (data.SafeOutputs != nil && len(data.SafeOutputs.Env) > 0) {
		steps = append(steps, "        env:\n")

		// Add custom environment variables specific to this safe output type
		steps = append(steps, config.CustomEnvVars...)

		// Add custom environment variables from safe-outputs.env
		c.addCustomSafeOutputEnvVars(&steps, data)
	}

	// With section for github-token
	steps = append(steps, "        with:\n")
	c.addSafeOutputGitHubTokenForConfig(&steps, data, config.Token)
	steps = append(steps, "          script: |\n")

	// Add the formatted JavaScript script
	formattedScript := FormatJavaScriptForYAML(config.Script)
	steps = append(steps, formattedScript...)

	return steps
}
