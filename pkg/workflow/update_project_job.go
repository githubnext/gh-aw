package workflow

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// buildUpdateProjectJob creates the update_project job
func (c *Compiler) buildUpdateProjectJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.UpdateProjects == nil {
		return nil, fmt.Errorf("safe-outputs.update-project configuration is required")
	}

	// Build custom environment variables specific to update-project
	var customEnvVars []string

	// Add common safe output job environment variables (staged/target repo)
	// Note: Project operations always work on the current repo, so targetRepoSlug is ""
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		"", // targetRepoSlug - projects always work on current repo
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.UpdateProjects != nil {
		token = data.SafeOutputs.UpdateProjects.GitHubToken
	}

	// Get the effective token using the Projects-specific precedence chain
	// This includes fallback to GH_AW_PROJECT_GITHUB_TOKEN if no custom token is configured
	// Note: Projects v2 requires a PAT or GitHub App - the default GITHUB_TOKEN cannot work
	effectiveToken := getEffectiveProjectGitHubToken(token, data.GitHubToken)

	// Always expose the effective token as GH_AW_PROJECT_GITHUB_TOKEN environment variable
	// The JavaScript code checks process.env.GH_AW_PROJECT_GITHUB_TOKEN to provide helpful error messages
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_GITHUB_TOKEN: %s\n", effectiveToken))

	// If views are configured in frontmatter, pass them to the JavaScript via environment variable
	if data.SafeOutputs.UpdateProjects != nil && len(data.SafeOutputs.UpdateProjects.Views) > 0 {
		viewsJSON, err := json.Marshal(data.SafeOutputs.UpdateProjects.Views)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal views configuration: %w", err)
		}
		// Use base64 encoding to safely pass JSON data through YAML without any quoting concerns
		viewsBase64 := base64.StdEncoding.EncodeToString(viewsJSON)
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_VIEWS_BASE64: %q\n", viewsBase64))
	}

	jobCondition := BuildSafeOutputType("update_project")
	permissions := NewPermissionsContentsReadProjectsWrite()

	// Use buildSafeOutputJob helper to get common scaffolding including app token minting
	job, err := c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:       "update_project",
		StepName:      "Update Project",
		StepID:        "update_project",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        "", // Script is now handled by project handler manager
		ScriptName:    "update_project",
		Permissions:   permissions,
		Outputs:       nil,
		Condition:     jobCondition,
		Needs:         []string{mainJobName},
		Token:         effectiveToken,
	})

	return job, err
}
