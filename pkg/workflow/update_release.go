package workflow

import (
	"fmt"
)

// UpdateReleasesConfig holds configuration for updating GitHub releases from agent output
type UpdateReleasesConfig struct {
	ReleaseID   string `yaml:"release-id,omitempty"`   // Explicit release ID or GitHub expression
	Target      string `yaml:"target,omitempty"`       // Target for updates: "triggering" (default), "*" (any release), or explicit release number
	Max         int    `yaml:"max,omitempty"`          // Maximum number of releases to update (default: 1)
	GitHubToken string `yaml:"github-token,omitempty"` // GitHub token for this specific output type
}

// buildCreateOutputUpdateReleaseJob creates the update_release job
func (c *Compiler) buildCreateOutputUpdateReleaseJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateReleases == nil {
		return nil, fmt.Errorf("safe-outputs.update-release configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Update Release\n")
	steps = append(steps, "        id: update_release\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))

	// Pass the target configuration
	if data.SafeOutputs.UpdateReleases.Target != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_UPDATE_TARGET: %q\n", data.SafeOutputs.UpdateReleases.Target))
	}

	// Pass the release ID configuration if specified
	if data.SafeOutputs.UpdateReleases.ReleaseID != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_RELEASE_ID: %q\n", data.SafeOutputs.UpdateReleases.ReleaseID))
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.UpdateReleases != nil {
		token = data.SafeOutputs.UpdateReleases.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(updateReleaseScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"release_id":  "${{ steps.update_release.outputs.release_id }}",
		"release_url": "${{ steps.update_release.outputs.release_url }}",
	}

	// Determine the job condition based on target configuration
	var baseCondition string
	if data.SafeOutputs.UpdateReleases.Target == "*" {
		// Allow updates to any release - no specific context required
		baseCondition = "always()"
	} else if data.SafeOutputs.UpdateReleases.Target != "" || data.SafeOutputs.UpdateReleases.ReleaseID != "" {
		// Explicit release ID specified - no specific context required
		baseCondition = "always()"
	} else {
		// Default behavior: only update triggering release (requires release event)
		baseCondition = "github.event.release.id"
	}

	// If this is a command workflow, combine the command trigger condition with the base condition
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()

		// Combine command condition with base condition using AND
		if baseCondition == "always()" {
			// If base condition is always(), just use the command condition
			jobCondition = commandConditionStr
		} else {
			// Combine both conditions with AND
			jobCondition = fmt.Sprintf("(%s) && (%s)", commandConditionStr, baseCondition)
		}
	} else {
		// No command trigger, just use the base condition
		jobCondition = baseCondition
	}

	job := &Job{
		Name:           "update_release",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parseUpdateReleasesConfig handles update-release configuration
func (c *Compiler) parseUpdateReleasesConfig(outputMap map[string]any) *UpdateReleasesConfig {
	if configData, exists := outputMap["update-release"]; exists {
		updateReleasesConfig := &UpdateReleasesConfig{Max: 1} // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse max
			if max, exists := configMap["max"]; exists {
				if maxInt, ok := parseIntValue(max); ok {
					updateReleasesConfig.Max = maxInt
				}
			}

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					updateReleasesConfig.Target = targetStr
				}
			}

			// Parse release-id
			if releaseID, exists := configMap["release-id"]; exists {
				if releaseIDStr, ok := releaseID.(string); ok {
					updateReleasesConfig.ReleaseID = releaseIDStr
				}
			}

			// Parse github-token
			if githubToken, exists := configMap["github-token"]; exists {
				if githubTokenStr, ok := githubToken.(string); ok {
					updateReleasesConfig.GitHubToken = githubTokenStr
				}
			}
		}

		return updateReleasesConfig
	}

	return nil
}