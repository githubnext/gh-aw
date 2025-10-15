package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// buildPreActivationJob creates the pre-activation job that combines permission checks and stop-time validation
// This job replaces both check_membership and stop_time_check jobs
func (c *Compiler) buildPreActivationJob(data *WorkflowData, needsPermissionCheck bool) (*Job, error) {
	outputs := map[string]string{}
	var steps []string
	var permissions string

	// Add permission check steps if needed
	if needsPermissionCheck {
		// Add team member check that only sets outputs
		steps = c.generateMembershipCheck(data, steps)

		// Add membership outputs
		outputs["is_team_member"] = fmt.Sprintf("${{ steps.%s.outputs.is_team_member }}", constants.CheckMembershipJobName)
		outputs["result"] = fmt.Sprintf("${{ steps.%s.outputs.result }}", constants.CheckMembershipJobName)
		outputs["user_permission"] = fmt.Sprintf("${{ steps.%s.outputs.user_permission }}", constants.CheckMembershipJobName)
		outputs["error_message"] = fmt.Sprintf("${{ steps.%s.outputs.error_message }}", constants.CheckMembershipJobName)
	}

	// Add stop-time check steps if stop-time is configured
	if data.StopTime != "" {
		steps = append(steps, "      - name: Check stop-time limit\n")
		steps = append(steps, "        id: stop_time_check\n")
		steps = append(steps, "        run: |\n")
		steps = append(steps, "          set -e\n")
		steps = append(steps, "          echo \"Checking stop-time limit...\"\n")

		// Extract workflow name for gh workflow commands
		workflowName := data.Name
		steps = append(steps, fmt.Sprintf("          WORKFLOW_NAME=\"%s\"\n", workflowName))

		// Add stop-time check logic
		steps = append(steps, "          \n")
		steps = append(steps, fmt.Sprintf("          STOP_TIME=\"%s\"\n", data.StopTime))
		steps = append(steps, "          echo \"Stop-time limit: $STOP_TIME\"\n")
		steps = append(steps, "          \n")
		steps = append(steps, "          # Convert stop time to epoch seconds\n")
		steps = append(steps, "          STOP_EPOCH=$(date -d \"$STOP_TIME\" +%s 2>/dev/null || echo \"invalid\")\n")
		steps = append(steps, "          if [ \"$STOP_EPOCH\" = \"invalid\" ]; then\n")
		steps = append(steps, "            echo \"Warning: Invalid stop-time format: $STOP_TIME. Expected format: YYYY-MM-DD HH:MM:SS\"\n")
		steps = append(steps, "            echo \"stop_time_expired=false\" >> $GITHUB_OUTPUT\n")
		steps = append(steps, "          else\n")
		steps = append(steps, "            CURRENT_EPOCH=$(date +%s)\n")
		steps = append(steps, "            echo \"Current time: $(date)\"\n")
		steps = append(steps, "            echo \"Stop time: $STOP_TIME\"\n")
		steps = append(steps, "            \n")
		steps = append(steps, "            if [ \"$CURRENT_EPOCH\" -ge \"$STOP_EPOCH\" ]; then\n")
		steps = append(steps, "              echo \"Stop time reached. Attempting to disable workflow to prevent cost overrun.\"\n")
		steps = append(steps, "              gh workflow disable \"$WORKFLOW_NAME\" || true\n")
		steps = append(steps, "              echo \"stop_time_expired=true\" >> $GITHUB_OUTPUT\n")
		steps = append(steps, "            else\n")
		steps = append(steps, "              echo \"Stop-time check passed. Workflow can proceed.\"\n")
		steps = append(steps, "              echo \"stop_time_expired=false\" >> $GITHUB_OUTPUT\n")
		steps = append(steps, "            fi\n")
		steps = append(steps, "          fi\n")
		steps = append(steps, "        env:\n")
		steps = append(steps, "          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")

		// Add stop-time output
		outputs["stop_time_expired"] = "${{ steps.stop_time_check.outputs.stop_time_expired }}"

		// Set permissions - need actions: write for workflow disable
		permissions = "permissions:\n      actions: write  # Required for gh workflow disable"
	}

	// If no steps have been added, this shouldn't happen but handle gracefully
	if len(steps) == 0 {
		return nil, fmt.Errorf("pre-activation job has no steps - this should not happen")
	}

	job := &Job{
		Name:        constants.PreActivationJobName,
		If:          data.If, // Use the existing condition (which may include alias checks)
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}
