package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// buildAddReactionJob creates the add_reaction job
func (c *Compiler) buildAddReactionJob(data *WorkflowData, activationJobCreated bool, frontmatter map[string]any) (*Job, error) {
	reactionCondition := buildReactionCondition()

	var steps []string

	steps = append(steps, fmt.Sprintf("      - name: Add %s reaction to the triggering item\n", data.AIReaction))
	steps = append(steps, "        id: react\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_REACTION: %s\n", data.AIReaction))
	if data.Command != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_COMMAND: %s\n", data.Command))
	}
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_WORKFLOW_NAME: %q\n", data.Name))

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(addReactionAndEditCommentScript)
	steps = append(steps, formattedScript...)

	outputs := map[string]string{
		"reaction_id": "${{ steps.react.outputs.reaction-id }}",
		"comment_id":  "${{ steps.react.outputs.comment-id }}",
		"comment_url": "${{ steps.react.outputs.comment-url }}",
	}

	var depends []string
	if activationJobCreated {
		depends = []string{constants.ActivationJobName} // Depend on the activation job only if it exists
	}

	// Set base permissions
	permissions := "permissions:\n      discussions: write\n      issues: write\n      pull-requests: write"

	// Add actions: write permission if team member checks are present for command workflows
	_, hasExplicitRoles := frontmatter["roles"]
	requiresWorkflowCancellation := data.Command != "" ||
		(!activationJobCreated && c.needsRoleCheck(data, frontmatter) && hasExplicitRoles)

	if requiresWorkflowCancellation {
		permissions = "permissions:\n      actions: write  # Required for github.rest.actions.cancelWorkflowRun()\n      contents: read\n      discussions: write\n      issues: write\n      pull-requests: write"
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
