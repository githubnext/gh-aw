package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var closePullRequestLog = logger.New("workflow:close_pull_request")

// ClosePullRequestsConfig holds configuration for closing GitHub pull requests from agent output
type ClosePullRequestsConfig = CloseEntityConfig

// parseClosePullRequestsConfig handles close-pull-request configuration
func (c *Compiler) parseClosePullRequestsConfig(outputMap map[string]any) *ClosePullRequestsConfig {
	params := CloseEntityJobParams{
		EntityType: CloseEntityPullRequest,
		ConfigKey:  "close-pull-request",
	}
	return c.parseCloseEntityConfig(outputMap, params, closePullRequestLog)
}

// buildCreateOutputClosePullRequestJob creates the close_pull_request job
func (c *Compiler) buildCreateOutputClosePullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	params := CloseEntityJobParams{
		EntityType:       CloseEntityPullRequest,
		ConfigKey:        "close-pull-request",
		EnvVarPrefix:     "GH_AW_CLOSE_PR",
		JobName:          "close_pull_request",
		StepName:         "Close Pull Request",
		OutputNumberKey:  "pull_request_number",
		OutputURLKey:     "pull_request_url",
		EventNumberPath1: "github.event.pull_request.number",
		EventNumberPath2: "github.event.comment.pull_request.number",
		ScriptGetter:     getClosePullRequestScript,
		PermissionsFunc:  NewPermissionsContentsReadPRWrite,
	}
	return c.buildCloseEntityJob(data, mainJobName, data.SafeOutputs.ClosePullRequests, params, closePullRequestLog)
}
