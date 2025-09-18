package workflow

import (
	"fmt"
	"strings"
)

// buildAddReactionJob creates the add_reaction job
func (c *Compiler) buildAddReactionJob(data *WorkflowData, taskJobCreated bool, frontmatter map[string]any) (*Job, error) {
	reactionCondition := buildReactionCondition()

	var steps []string

	// Add permission checks if no task job was created but permission checks are needed
	if !taskJobCreated && c.needsRoleCheck(data, frontmatter) {
		// Add team member check step
		steps = append(steps, "      - name: Check team membership for workflow\n")
		steps = append(steps, "        id: check-team-member\n")
		steps = append(steps, "        uses: actions/github-script@v8\n")

		// Add environment variables for permission check
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_REQUIRED_ROLES: %s\n", strings.Join(data.Roles, ",")))

		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Generate the JavaScript code for the permission check
		scriptContent := c.generateRoleCheckScript(data.Roles)
		scriptLines := strings.Split(scriptContent, "\n")
		for _, line := range scriptLines {
			if strings.TrimSpace(line) != "" {
				steps = append(steps, fmt.Sprintf("            %s\n", line))
			}
		}
	}

	steps = append(steps, fmt.Sprintf("      - name: Add %s reaction to the triggering item\n", data.AIReaction))
	steps = append(steps, "        id: react\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_REACTION: %s\n", data.AIReaction))
	if data.Command != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_COMMAND: %s\n", data.Command))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(addReactionAndEditCommentScript)
	steps = append(steps, formattedScript...)

	outputs := map[string]string{
		"reaction_id": "${{ steps.react.outputs.reaction-id }}",
	}

	var depends []string
	if taskJobCreated {
		depends = []string{"task"} // Depend on the task job only if it exists
	}

	// Set base permissions
	permissions := "permissions:\n      issues: write\n      pull-requests: write"

	// Add actions: write permission if team member checks are present for command workflows
	_, hasExplicitRoles := frontmatter["roles"]
	requiresWorkflowCancellation := data.Command != "" ||
		(!taskJobCreated && c.needsRoleCheck(data, frontmatter) && hasExplicitRoles)

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
