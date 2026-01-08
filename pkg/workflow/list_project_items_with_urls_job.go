package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var listProjectItemsWithUrlsJobLog = logger.New("workflow:list_project_items_with_urls_job")

// buildListProjectItemsWithUrlsJob creates the list_project_items_with_urls job
func (c *Compiler) buildListProjectItemsWithUrlsJob(data *WorkflowData, mainJobName string) (*Job, error) {
	listProjectItemsWithUrlsJobLog.Printf("Building list_project_items_with_urls job for workflow: %s", data.Name)

	if data.SafeOutputs == nil || data.SafeOutputs.ListProjectItemsWithUrls == nil {
		return nil, fmt.Errorf("safe-outputs.list-project-items-with-urls configuration is required")
	}

	// Build custom environment variables specific to list-project-items-with-urls
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
	if data.SafeOutputs.ListProjectItemsWithUrls != nil {
		token = data.SafeOutputs.ListProjectItemsWithUrls.GitHubToken
	}

	// Get the effective token using the Projects-specific precedence chain
	effectiveToken := getEffectiveProjectGitHubToken(token, data.GitHubToken)

	// Always expose the effective token as GH_AW_PROJECT_GITHUB_TOKEN environment variable
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_GITHUB_TOKEN: %s\n", effectiveToken))

	jobCondition := BuildSafeOutputType("list_project_items_with_urls")
	permissions := NewPermissionsContentsRead() // Read-only operation

	// Use buildSafeOutputJob helper to get common scaffolding
	job, err := c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:       "list_project_items_with_urls",
		StepName:      "List Project Items",
		StepID:        "list_items",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        "const { main } = require('/opt/gh-aw/actions/list_project_items_with_urls.cjs'); await main();",
		ScriptName:    "list_project_items_with_urls",
		Permissions:   permissions,
		Outputs:       map[string]string{
			"items": "${{ steps.list_items.outputs.items }}",
			"count": "${{ steps.list_items.outputs.count }}",
		},
		Condition: jobCondition,
		Needs:     []string{mainJobName},
		Token:     effectiveToken,
	})

	listProjectItemsWithUrlsJobLog.Printf("Created list_project_items_with_urls job with condition: %s", jobCondition)
	return job, err
}
