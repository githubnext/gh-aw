package workflow

import (
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var compilerUnlockJobLog = logger.New("workflow:compiler_unlock_job")

// buildUnlockJob creates a dedicated job that unlocks issues after agent workflow execution.
// This job is separate from the conclusion job to ensure it always runs, even if other jobs fail.
// The job runs when:
// 1. always() - runs even if agent or other jobs fail
// 2. activation.outputs.issue_locked == 'true' - only if issue was actually locked
// 3. Event type is 'issues' or 'issue_comment' - only for applicable events
// The job depends on agent and detection (if enabled) to ensure unlock happens after workflow execution.
func (c *Compiler) buildUnlockJob(data *WorkflowData, threatDetectionEnabled bool) (*Job, error) {
	compilerUnlockJobLog.Print("Building dedicated unlock job")
	if !data.LockForAgent {
		compilerUnlockJobLog.Print("Lock-for-agent not enabled, skipping unlock job")
		return nil, nil
	}
	steps, err := c.buildUnlockJobSteps(data)
	if err != nil {
		return nil, err
	}
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}
	needs := buildUnlockJobNeeds(threatDetectionEnabled)
	permissions := c.buildUnlockJobPermissions(data)
	jobCondition := buildUnlockJobCondition()
	compilerUnlockJobLog.Printf("Job built successfully: dependencies=%v", needs)

	job := &Job{
		Name:           "unlock",
		Needs:          needs,
		If:             RenderCondition(jobCondition),
		RunsOn:         c.formatFrameworkJobRunsOn(data),
		Permissions:    permissions,
		Steps:          steps,
		TimeoutMinutes: 5, // Short timeout - unlock is a quick operation
	}

	return job, nil
}

func (c *Compiler) buildUnlockJobSteps(data *WorkflowData) ([]string, error) {
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" && !c.actionMode.IsScript() {
		return nil, errors.New("setup action reference is required but could not be resolved")
	}
	steps := append([]string{}, c.generateCheckoutActionsFolder(data)...)
	unlockTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
	unlockParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
	steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, unlockTraceID, unlockParentSpanID)...)
	steps = append(steps, buildUnlockIssueStep(data)...)
	return steps, nil
}

func buildUnlockIssueStep(data *WorkflowData) []string {
	unlockCondition := BuildAnd(
		BuildOr(BuildEventTypeEquals("issues"), BuildEventTypeEquals("issue_comment")),
		BuildEquals(BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.issue_locked", constants.ActivationJobName)), BuildStringLiteral("true")),
	)
	compilerUnlockJobLog.Print("Added unlock issue step to dedicated unlock job")
	return []string{
		"      - name: Unlock issue after agentic workflow\n",
		"        id: unlock-issue\n",
		fmt.Sprintf("        if: %s\n", RenderCondition(unlockCondition)),
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        with:\n",
		"          script: |\n",
		generateGitHubScriptWithRequire("unlock-issue.cjs"),
	}
}

func buildUnlockJobCondition() ConditionNode {
	return BuildAnd(
		BuildFunctionCall("always"),
		BuildNotEquals(BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.ActivationJobName)), BuildStringLiteral("skipped")),
	)
}

func buildUnlockJobNeeds(threatDetectionEnabled bool) []string {
	needs := []string{string(constants.ActivationJobName), string(constants.AgentJobName)}
	if threatDetectionEnabled {
		needs = append(needs, string(constants.DetectionJobName))
		compilerUnlockJobLog.Print("Added detection job dependency to unlock job")
	}
	return needs
}

func (c *Compiler) buildUnlockJobPermissions(data *WorkflowData) string {
	needsContentsRead := (c.actionMode.IsDev() || c.actionMode.IsScript()) && len(c.generateCheckoutActionsFolder(data)) > 0
	if needsContentsRead {
		perms := NewPermissionsContentsRead()
		perms.Set(PermissionIssues, PermissionWrite)
		return perms.RenderToYAML()
	}
	perms := NewPermissions()
	perms.Set(PermissionIssues, PermissionWrite)
	return perms.RenderToYAML()
}
