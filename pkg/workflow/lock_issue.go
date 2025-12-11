package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var lockIssueLog = logger.New("workflow:lock_issue")

// LockIssuesConfig holds configuration for locking GitHub issues from agent output
type LockIssuesConfig = CloseEntityConfig

// parseLockIssuesConfig handles lock-issue configuration
func (c *Compiler) parseLockIssuesConfig(outputMap map[string]any) *LockIssuesConfig {
	params := CloseEntityJobParams{
		EntityType: CloseEntityIssue,
		ConfigKey:  "lock-issue",
	}
	return c.parseCloseEntityConfig(outputMap, params, lockIssueLog)
}

// buildCreateOutputLockIssueJob creates the lock_issue job
func (c *Compiler) buildCreateOutputLockIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	params := CloseEntityJobParams{
		EntityType:       CloseEntityIssue,
		ConfigKey:        "lock-issue",
		EnvVarPrefix:     "GH_AW_LOCK_ISSUE",
		JobName:          "lock_issue",
		StepName:         "Lock Issue",
		OutputNumberKey:  "issue_number",
		OutputURLKey:     "issue_url",
		EventNumberPath1: "github.event.issue.number",
		EventNumberPath2: "github.event.comment.issue.number",
		ScriptGetter:     getLockIssueScript,
		PermissionsFunc:  NewPermissionsContentsReadIssuesWrite,
	}
	return c.buildCloseEntityJob(data, mainJobName, data.SafeOutputs.LockIssues, params, lockIssueLog)
}

// getLockIssueScript returns the bundled lock_issue script
func getLockIssueScript() string {
	return DefaultScriptRegistry.GetWithMode("lock_issue", RuntimeModeGitHubScript)
}
