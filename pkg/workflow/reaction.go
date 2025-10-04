package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// buildAddReactionJob creates the add_reaction job
func (c *Compiler) buildAddReactionJob(data *WorkflowData, activationJobCreated bool, frontmatter map[string]any) (*Job, error) {
	reactionCondition := buildReactionCondition()

	// Prepare environment variables for the step
	env := make(map[string]string)
	env["GITHUB_AW_REACTION"] = data.AIReaction
	if data.Command != "" {
		env["GITHUB_AW_COMMAND"] = data.Command
	}

	// Build the github-script step using the helper with script included
	steps := BuildGitHubScriptStepLines(
		fmt.Sprintf("Add %s reaction to the triggering item", data.AIReaction),
		"react",
		addReactionAndEditCommentScript, // script is now included directly
		env,
		nil, // no with parameters other than script
	)

	outputs := map[string]string{
		"reaction_id": "${{ steps.react.outputs.reaction-id }}",
	}

	var depends []string
	if activationJobCreated {
		depends = []string{constants.ActivationJobName} // Depend on the activation job only if it exists
	}

	// Set base permissions
	permissions := "permissions:\n      issues: write\n      pull-requests: write"

	// Add actions: write permission if team member checks are present for command workflows
	_, hasExplicitRoles := frontmatter["roles"]
	requiresWorkflowCancellation := data.Command != "" ||
		(!activationJobCreated && c.needsRoleCheck(data, frontmatter) && hasExplicitRoles)

	if requiresWorkflowCancellation {
		permissions = "permissions:\n      actions: write  # Required for github.rest.actions.cancelWorkflowRun()\n      issues: write\n      pull-requests: write\n      contents: read"
	}

	job := &Job{
		Name:        "add_reaction",
		If:          reactionCondition.Render(),
		RunsOn:      "runs-on: ubuntu-latest",
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
		Needs:       depends,
	}

	return job, nil
}
