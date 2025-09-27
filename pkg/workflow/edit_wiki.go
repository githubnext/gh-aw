package workflow

import (
	"fmt"
	"strings"
)

// EditWikiConfig holds configuration for editing GitHub wiki pages from agent output
type EditWikiConfig struct {
	Path        []string `yaml:"path,omitempty"`         // Optional path restriction (defaults to workflowid/)
	Max         int      `yaml:"max,omitempty"`          // Maximum number of wiki edits to perform
	GitHubToken string   `yaml:"github-token,omitempty"` // GitHub token for this specific output type
}

// parseEditWikiConfig handles edit-wiki configuration
func (c *Compiler) parseEditWikiConfig(outputMap map[string]any) *EditWikiConfig {
	if configData, exists := outputMap["edit-wiki"]; exists {
		editWikiConfig := &EditWikiConfig{Max: 1} // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse path
			if path, exists := configMap["path"]; exists {
				if pathArray, ok := path.([]any); ok {
					var pathStrings []string
					for _, pathItem := range pathArray {
						if pathStr, ok := pathItem.(string); ok {
							pathStrings = append(pathStrings, pathStr)
						}
					}
					editWikiConfig.Path = pathStrings
				}
			}

			// Parse max
			if max, exists := configMap["max"]; exists {
				if maxInt, ok := parseIntValue(max); ok {
					editWikiConfig.Max = maxInt
				}
			}

			// Parse github-token
			if githubToken, exists := configMap["github-token"]; exists {
				if githubTokenStr, ok := githubToken.(string); ok {
					editWikiConfig.GitHubToken = githubTokenStr
				}
			}
		} else if configData == nil {
			// Handle null case: create empty config with defaults
			editWikiConfig = &EditWikiConfig{Max: 1}
		}

		return editWikiConfig
	}
	return nil
}

// buildEditWikiJob creates the edit_wiki job
func (c *Compiler) buildEditWikiJob(data *WorkflowData, mainJobName string, taskJobCreated bool, frontmatter map[string]any) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.EditWiki == nil {
		return nil, fmt.Errorf("safe-outputs.edit-wiki configuration is required")
	}

	var steps []string

	steps = append(steps, "      - name: Edit Wiki Pages\n")
	steps = append(steps, "        id: edit_wiki\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	steps = append(steps, fmt.Sprintf("          GITHUB_WORKFLOW_NAME: %s\n", data.Name))

	// Add path restriction if configured
	if len(data.SafeOutputs.EditWiki.Path) > 0 {
		pathsStr := strings.Join(data.SafeOutputs.EditWiki.Path, ",")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_WIKI_ALLOWED_PATHS: %q\n", pathsStr))
	}

	// Add max configuration
	if data.SafeOutputs.EditWiki.Max > 0 {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_WIKI_MAX: %d\n", data.SafeOutputs.EditWiki.Max))
	}

	// Pass the staged flag if it's set to true
	if data.SafeOutputs.Staged != nil && *data.SafeOutputs.Staged {
		steps = append(steps, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	if data.SafeOutputs.EditWiki.GitHubToken != "" {
		steps = append(steps, fmt.Sprintf("          github-token: %s\n", data.SafeOutputs.EditWiki.GitHubToken))
	} else if data.SafeOutputs.GitHubToken != "" {
		steps = append(steps, fmt.Sprintf("          github-token: %s\n", data.SafeOutputs.GitHubToken))
	}
	steps = append(steps, "          script: |\n")

	// Use the embedded JavaScript from edit_wiki.cjs
	formattedScript := FormatJavaScriptForYAML(editWikiScript)
	for _, line := range formattedScript {
		if strings.TrimSpace(line) != "" {
			steps = append(steps, fmt.Sprintf("            %s\n", line))
		}
	}

	// Create outputs for the job
	outputs := map[string]string{
		"wiki_pages_edited": "${{ steps.edit_wiki.outputs.wiki_pages_edited }}",
		"wiki_pages_failed": "${{ steps.edit_wiki.outputs.wiki_pages_failed }}",
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
		Name:           "edit_wiki",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: write", // Need contents write for wiki access
		TimeoutMinutes: 10,                                    // 10-minute timeout
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}