package workflow

import (
	"fmt"
)

// buildCreateOutputPushToBranchJob creates the push_to_branch job
func (c *Compiler) buildCreateOutputPushToBranchJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.PushToBranch == nil {
		return nil, fmt.Errorf("safe-outputs.push-to-branch configuration is required")
	}

	// Branch should have a default value of "triggering" set by the parser
	if data.SafeOutputs.PushToBranch.Branch == "" {
		return nil, fmt.Errorf("safe-outputs.push-to-branch branch configuration is invalid")
	}

	var steps []string

	// Step 1: Download patch artifact
	steps = append(steps, "      - name: Download patch artifact\n")
	steps = append(steps, "        uses: actions/download-artifact@v4\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: aw.patch\n")
	steps = append(steps, "          path: /tmp/\n")

	// Step 2: Checkout repository
	steps = append(steps, "      - name: Checkout repository\n")
	steps = append(steps, "        uses: actions/checkout@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          fetch-depth: 0\n")

	// Step 3: Push to branch
	steps = append(steps, "      - name: Push to Branch\n")
	steps = append(steps, "        id: push_to_branch\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the branch configuration
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_PUSH_BRANCH: %q\n", data.SafeOutputs.PushToBranch.Branch))
	// Pass the target configuration
	if data.SafeOutputs.PushToBranch.Target != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PUSH_TARGET: %q\n", data.SafeOutputs.PushToBranch.Target))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(pushToBranchScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"branch_name": "${{ steps.push_to_branch.outputs.branch_name }}",
		"commit_sha":  "${{ steps.push_to_branch.outputs.commit_sha }}",
		"push_url":    "${{ steps.push_to_branch.outputs.push_url }}",
	}

	// Determine the job condition based on target configuration
	var jobCondition string
	if data.SafeOutputs.PushToBranch.Target == "*" {
		// Allow pushing to any pull request - no specific context required
		jobCondition = "if: always()"
	} else {
		// Default behavior: only run in pull request context
		jobCondition = "if: github.event.pull_request.number"
	}

	job := &Job{
		Name:           "push_to_branch",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: write\n      pull-requests: read",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
