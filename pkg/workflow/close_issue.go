package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var closeIssueLog = logger.New("workflow:close_issue")

// CloseIssuesConfig holds configuration for closing GitHub issues from agent output
type CloseIssuesConfig = CloseEntityConfig

// parseCloseIssuesConfig handles close-issue configuration
func (c *Compiler) parseCloseIssuesConfig(outputMap map[string]any) *CloseIssuesConfig {
	params := CloseEntityJobParams{
		EntityType: CloseEntityIssue,
		ConfigKey:  "close-issue",
	}
	return c.parseCloseEntityConfig(outputMap, params, closeIssueLog)
}

// buildCreateOutputCloseIssueJob creates the close_issue job
func (c *Compiler) buildCreateOutputCloseIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	params := CloseEntityJobParams{
		EntityType:       CloseEntityIssue,
		ConfigKey:        "close-issue",
		EnvVarPrefix:     "GH_AW_CLOSE_ISSUE",
		JobName:          "close_issue",
		StepName:         "Close Issue",
		OutputNumberKey:  "issue_number",
		OutputURLKey:     "issue_url",
		EventNumberPath1: "github.event.issue.number",
		EventNumberPath2: "github.event.comment.issue.number",
		ScriptGetter:     getCloseIssueScript,
		PermissionsFunc:  NewPermissionsContentsReadIssuesWrite,
	}
	return c.buildCloseEntityJob(data, mainJobName, data.SafeOutputs.CloseIssues, params, closeIssueLog)
}
