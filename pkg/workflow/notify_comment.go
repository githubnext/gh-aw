package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// buildConclusionJob creates a job that updates the activation comment with workflow completion status
// AND updates the commit status if create-commit-status is configured
// This job runs when:
// 1. always() - runs even if agent fails
// 2. For comment updates: A comment was created in activation job AND no conflicting outputs
// 3. For commit status updates: create-commit-status is configured (always runs)
// This job depends on all safe output jobs to ensure it runs last
func (c *Compiler) buildConclusionJob(data *WorkflowData, mainJobName string, safeOutputJobNames []string) (*Job, error) {
	// Create this job when:
	// 1. add-comment is configured with a reaction, OR
	// 2. command is configured with a reaction (which auto-creates a comment in activation), OR
	// 3. create-commit-status is configured (needs final status update)

	hasAddComment := data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil
	hasCommand := data.Command != ""
	hasReaction := data.AIReaction != "" && data.AIReaction != "none"
	hasCommitStatus := data.SafeOutputs != nil && data.SafeOutputs.CreateCommitStatus != nil

	// Determine if we need this job at all
	needsCommentUpdate := hasReaction && (hasAddComment || hasCommand)
	needsStatusUpdate := hasCommitStatus

	if !needsCommentUpdate && !needsStatusUpdate {
		return nil, nil // No updates needed
	}

	// Build the job steps
	var steps []string

	// Add debug step
	steps = append(steps, "      - name: Debug job inputs\n")
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
	steps = append(steps, fmt.Sprintf("          COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
	steps = append(steps, fmt.Sprintf("          AGENT_OUTPUT_TYPES: ${{ needs.%s.outputs.output_types }}\n", mainJobName))
	steps = append(steps, fmt.Sprintf("          AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))
	if hasCommitStatus {
		steps = append(steps, fmt.Sprintf("          STATUS_CONTEXT: ${{ needs.%s.outputs.status_context }}\n", constants.ActivationJobName))
		steps = append(steps, fmt.Sprintf("          STATUS_SHA: ${{ needs.%s.outputs.status_sha }}\n", constants.ActivationJobName))
	}
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          echo \"Comment ID: $COMMENT_ID\"\n")
	steps = append(steps, "          echo \"Comment Repo: $COMMENT_REPO\"\n")
	steps = append(steps, "          echo \"Agent Output Types: $AGENT_OUTPUT_TYPES\"\n")
	steps = append(steps, "          echo \"Agent Conclusion: $AGENT_CONCLUSION\"\n")
	if hasCommitStatus {
		steps = append(steps, "          echo \"Status Context: $STATUS_CONTEXT\"\n")
		steps = append(steps, "          echo \"Status SHA: $STATUS_SHA\"\n")
	}

	// Add commit status update step if create-commit-status is configured
	// This step ALWAYS runs (no conditional) and updates the final status
	if hasCommitStatus {
		steps = append(steps, "      - name: Update final commit status\n")
		steps = append(steps, "        id: update_status\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_STATUS_CONTEXT: ${{ needs.%s.outputs.status_context }}\n", constants.ActivationJobName))
		steps = append(steps, fmt.Sprintf("          GH_AW_STATUS_SHA: ${{ needs.%s.outputs.status_sha }}\n", constants.ActivationJobName))
		steps = append(steps, fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
		steps = append(steps, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add the final status update script
		formattedScript := FormatJavaScriptForYAML(getUpdateFinalCommitStatusScript())
		steps = append(steps, formattedScript...)
	}

	// Add comment update step if reactions are configured
	// This step is conditional on comment existence and no conflicting outputs
	if needsCommentUpdate {
		// Build environment variables for the comment update script
		var customEnvVars []string
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
		customEnvVars = append(customEnvVars, "          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n")
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_AGENT_CONCLUSION: ${{ needs.%s.result }}\n", mainJobName))

		// Get token from config
		var token string
		if data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil {
			token = data.SafeOutputs.AddComments.GitHubToken
		}

		// Build the conditional comment update steps using the common helper
		scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
			StepName:      "Update reaction comment with completion status",
			StepID:        "conclusion_comment",
			MainJobName:   mainJobName,
			CustomEnvVars: customEnvVars,
			Script:        notifyCommentErrorScript,
			Token:         token,
		})
		
		// Add condition to the first step line (the "- name:" line)
		// Find and modify the first line to add the condition
		for i, line := range scriptSteps {
			if strings.Contains(line, "- name:") {
				// Insert the condition after the id line
				// Find the line with "id:" and insert condition after it
				for j := i; j < len(scriptSteps); j++ {
					if strings.Contains(scriptSteps[j], "id:") {
						// Insert condition line after id
						condition := fmt.Sprintf("        if: %s\n", buildCommentUpdateCondition(mainJobName).Render())
						scriptSteps = append(scriptSteps[:j+1], append([]string{condition}, scriptSteps[j+1:]...)...)
						break
					}
				}
				break
			}
		}
		
		steps = append(steps, scriptSteps...)
	}

	// Build the condition for this job:
	// For commit status: always() && agent not skipped
	// For comments: same as before (always() && agent not skipped && comment exists && no conflicting outputs)
	
	// Base condition: always() && agent not skipped
	alwaysFunc := BuildFunctionCall("always")
	agentNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)
	
	baseCondition := buildAnd(alwaysFunc, agentNotSkipped)

	// Build dependencies - this job depends on all safe output jobs to ensure it runs last
	needs := []string{mainJobName, constants.ActivationJobName}
	needs = append(needs, safeOutputJobNames...)

	// Set permissions - need statuses:write if commit status is configured
	// Also need comment permissions if reactions are configured
	permsMap := map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead,
	}
	
	if needsCommentUpdate {
		permsMap[PermissionIssues] = PermissionWrite
		permsMap[PermissionPullRequests] = PermissionWrite
		permsMap[PermissionDiscussions] = PermissionWrite
	}
	
	if hasCommitStatus {
		permsMap[PermissionStatuses] = PermissionWrite
	}

	job := &Job{
		Name:        "conclusion",
		If:          baseCondition.Render(),
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: NewPermissionsFromMap(permsMap).RenderToYAML(),
		Steps:       steps,
		Needs:       needs,
	}

	return job, nil
}

// buildCommentUpdateCondition builds the condition for when to update the comment
// This is a helper function to keep the logic clear
func buildCommentUpdateCondition(mainJobName string) ConditionNode {
	// Check that a comment was created in activation
	commentIdExists := BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.comment_id", constants.ActivationJobName))

	// Check that output_types doesn't contain add_comment
	noAddComment := &NotNode{
		Child: BuildFunctionCall("contains",
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.output_types", constants.AgentJobName)),
			BuildStringLiteral("add_comment"),
		),
	}

	// Check that output_types doesn't contain create_pull_request
	noCreatePR := &NotNode{
		Child: BuildFunctionCall("contains",
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.output_types", constants.AgentJobName)),
			BuildStringLiteral("create_pull_request"),
		),
	}

	// Check that output_types doesn't contain push_to_pull_request_branch
	noPushToBranch := &NotNode{
		Child: BuildFunctionCall("contains",
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.output_types", constants.AgentJobName)),
			BuildStringLiteral("push_to_pull_request_branch"),
		),
	}

	// Combine all conditions with AND
	return buildAnd(
		buildAnd(
			buildAnd(
				commentIdExists,
				noAddComment,
			),
			noCreatePR,
		),
		noPushToBranch,
	)
}
