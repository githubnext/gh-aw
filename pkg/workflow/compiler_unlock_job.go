package workflow

import (
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

	// Add setup step to copy scripts
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" && !c.actionMode.IsScript() {
		return nil, fmt.Errorf("setup action reference is required but could not be resolved")
	}

	// For dev mode (local action path), checkout the actions folder first
	steps = append(steps, c.generateCheckoutActionsFolder(data)...)

	// Unlock job doesn't need project support
	steps = append(steps, c.generateSetupStep(setupActionRef, SetupActionDestination, false)...)

	// Add unlock step
	// Build condition: only unlock if issue was locked by activation job
	// Must match lock condition: event type is 'issues' or 'issue_comment'
	eventTypeCheck := BuildOr(
		BuildEventTypeEquals("issues"),
		BuildEventTypeEquals("issue_comment"),
	)
	lockedOutputCheck := BuildEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.issue_locked", constants.ActivationJobName)),
		BuildStringLiteral("true"),
	)

	unlockCondition := BuildAnd(eventTypeCheck, lockedOutputCheck)

	steps = append(steps, "      - name: Unlock issue after agent workflow\n")
	steps = append(steps, "        id: unlock-issue\n")
	steps = append(steps, fmt.Sprintf("        if: %s\n", unlockCondition.Render()))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")
	steps = append(steps, generateGitHubScriptWithRequire("unlock-issue.cjs"))

	compilerUnlockJobLog.Print("Added unlock issue step to dedicated unlock job")

	// Build the condition for this job:
	// 1. always() - run even if agent or other jobs fail
	// 2. issue was locked (checked at step level for clarity in workflow YAML)
	alwaysFunc := BuildFunctionCall("always")

	// Create the unlock job
	// This job depends on activation (for issue_locked output), agent (to run after workflow),
	// and detection (if enabled, to run after threat detection completes)
	needs := []string{string(constants.ActivationJobName), string(constants.AgentJobName)}

	// Add detection job dependency if threat detection is enabled
	if threatDetectionEnabled {
		needs = append(needs, string(constants.DetectionJobName))
		compilerUnlockJobLog.Print("Added detection job dependency to unlock job")
	}

	// Determine permissions - need contents: read for dev mode checkout, issues: write for unlocking
	var permissions string
	needsContentsRead := (c.actionMode.IsDev() || c.actionMode.IsScript()) && len(c.generateCheckoutActionsFolder(data)) > 0
	if needsContentsRead {
		perms := NewPermissionsContentsRead()
		// Add issues write permission for unlocking
		perms.Set(PermissionIssues, PermissionWrite)
		permissions = perms.RenderToYAML()
	} else {
		// Only need issues write permission
		perms := NewPermissions()
		perms.Set(PermissionIssues, PermissionWrite)
		permissions = perms.RenderToYAML()
	}

	compilerUnlockJobLog.Printf("Job built successfully: dependencies=%v", needs)

	job := &Job{
		Name:           "unlock",
		Needs:          needs,
		If:             alwaysFunc.Render(),
		RunsOn:         data.RunsOn,
		Permissions:    permissions,
		Steps:          steps,
		TimeoutMinutes: 5, // Short timeout - unlock is a quick operation
	}

	return job, nil
}
