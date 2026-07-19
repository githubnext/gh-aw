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

	var steps []string

	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" && !c.actionMode.IsScript() {
		return nil, errors.New("setup action reference is required but could not be resolved")
	}

	steps = append(steps, c.buildUnlockSetupSteps(data, setupActionRef)...)
	steps = append(steps, buildUnlockIssueStep(data)...)

	compilerUnlockJobLog.Print("Added unlock issue step to dedicated unlock job")

	needs := unlockJobNeeds(threatDetectionEnabled)
	permissions := c.unlockJobPermissions(data)

	compilerUnlockJobLog.Printf("Job built successfully: dependencies=%v", needs)

	// In script mode, explicitly add a cleanup step (mirrors post.js in dev/release/action mode).
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}

	job := &Job{
		Name:           "unlock",
		Needs:          needs,
		If:             unlockJobCondition(),
		RunsOn:         c.formatFrameworkJobRunsOn(data),
		Permissions:    permissions,
		Steps:          steps,
		TimeoutMinutes: 5, // Short timeout - unlock is a quick operation
	}

	return job, nil
}

func (c *Compiler) buildUnlockSetupSteps(data *WorkflowData, setupActionRef string) []string {
	var steps []string
	steps = append(steps, c.generateCheckoutActionsFolder(data)...)
	unlockTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
	unlockParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
	return append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, unlockTraceID, unlockParentSpanID)...)
}

func buildUnlockIssueStep(data *WorkflowData) []string {
	unlockCondition := BuildAnd(
		BuildOr(BuildEventTypeEquals("issues"), BuildEventTypeEquals("issue_comment")),
		BuildEquals(BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.issue_locked", constants.ActivationJobName)), BuildStringLiteral("true")),
	)
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

func unlockJobCondition() string {
	alwaysFunc := BuildFunctionCall("always")
	activationNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.ActivationJobName)),
		BuildStringLiteral("skipped"),
	)
	return RenderCondition(BuildAnd(alwaysFunc, activationNotSkipped))
}

func unlockJobNeeds(threatDetectionEnabled bool) []string {
	needs := []string{string(constants.ActivationJobName), string(constants.AgentJobName)}
	if threatDetectionEnabled {
		needs = append(needs, string(constants.DetectionJobName))
		compilerUnlockJobLog.Print("Added detection job dependency to unlock job")
	}
	return needs
}

func (c *Compiler) unlockJobPermissions(data *WorkflowData) string {
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
