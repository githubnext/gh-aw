package workflow

import (
	"fmt"
)

// buildCreateOutputPushToOrphanedBranchJob creates the push_to_orphaned_branch job
func (c *Compiler) buildCreateOutputPushToOrphanedBranchJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.PushToOrphanedBranch == nil {
		return nil, fmt.Errorf("safe-outputs.push-to-orphaned-branch configuration is required")
	}

	var steps []string

	// Step 1: Checkout repository
	steps = append(steps, "      - name: Checkout repository\n")
	steps = append(steps, "        uses: actions/checkout@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          fetch-depth: 0\n")

	// Step 2: Configure Git credentials
	steps = append(steps, c.generateGitConfigurationSteps()...)

	// Step 3: Push to Orphaned Branch
	steps = append(steps, "      - name: Push to Orphaned Branch\n")
	steps = append(steps, "        id: push_to_orphaned_branch\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Add GH_TOKEN for authentication
	steps = append(steps, "          GH_TOKEN: ${{ github.token }}\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the max count configuration
	maxCount := 1
	if data.SafeOutputs.PushToOrphanedBranch.Max > 0 {
		maxCount = data.SafeOutputs.PushToOrphanedBranch.Max
	}
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_ORPHANED_BRANCH_MAX_COUNT: %d\n", maxCount))

	// Pass the staged flag if it's set to true
	if data.SafeOutputs.Staged != nil && *data.SafeOutputs.Staged {
		steps = append(steps, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	c.addSafeOutputGitHubToken(&steps, data)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(pushToOrphanedBranchScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"uploaded_files": "${{ steps.push_to_orphaned_branch.outputs.uploaded_files }}",
		"file_urls":      "${{ steps.push_to_orphaned_branch.outputs.file_urls }}",
	}

	// This job can run in any context since it only uploads files to orphaned branches
	jobCondition := "always()"

	// If this is a command workflow, add the command trigger condition
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		jobCondition = commandConditionStr
	}

	job := &Job{
		Name:           "push_to_orphaned_branch",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: write\n      actions: read",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}