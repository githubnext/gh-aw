package workflow

import (
	"fmt"
)

// buildCreateOutputPushToPullRequestBranchJob creates the push_to_pr_branch job
func (c *Compiler) buildCreateOutputPushToPullRequestBranchJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.PushToPullRequestBranch == nil {
		return nil, fmt.Errorf("safe-outputs.push-to-pr-branch configuration is required")
	}

	var steps []string

	// Step 1: Download patch artifact
	steps = append(steps, "      - name: Download patch artifact\n")
	steps = append(steps, "        continue-on-error: true\n")
	steps = append(steps, "        uses: actions/download-artifact@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: aw.patch\n")
	steps = append(steps, "          path: /tmp/\n")

	// Step 2: Checkout repository
	steps = append(steps, "      - name: Checkout repository\n")
	steps = append(steps, "        uses: actions/checkout@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          fetch-depth: 0\n")

	// Step 3: Configure Git credentials
	steps = append(steps, c.generateGitConfigurationSteps()...)

	// Step 4: Push to Branch
	steps = append(steps, "      - name: Push to Branch\n")
	steps = append(steps, "        id: push_to_pr_branch\n")
	steps = append(steps, "        uses: actions/github-script@v8\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Add GH_TOKEN for authentication, because we shell out to 'gh' commands
	steps = append(steps, "          GH_TOKEN: ${{ github.token }}\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the target configuration
	if data.SafeOutputs.PushToPullRequestBranch.Target != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PUSH_TARGET: %q\n", data.SafeOutputs.PushToPullRequestBranch.Target))
	}
	// Pass the if-no-changes configuration
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_PUSH_IF_NO_CHANGES: %q\n", data.SafeOutputs.PushToPullRequestBranch.IfNoChanges))
	// Pass the maximum patch size configuration
	maxPatchSize := 1024 // Default value
	if data.SafeOutputs != nil && data.SafeOutputs.MaximumPatchSize > 0 {
		maxPatchSize = data.SafeOutputs.MaximumPatchSize
	}
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_MAX_PATCH_SIZE: %d\n", maxPatchSize))

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	steps = append(steps, "        with:\n")
	// Add github-token if specified
	var token string
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		token = data.SafeOutputs.PushToPullRequestBranch.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, token)
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(pushToBranchScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"branch_name": "${{ steps.push_to_pr_branch.outputs.branch_name }}",
		"commit_sha":  "${{ steps.push_to_pr_branch.outputs.commit_sha }}",
		"push_url":    "${{ steps.push_to_pr_branch.outputs.push_url }}",
	}

	// Determine the job condition based on target configuration
	var baseCondition string
	if data.SafeOutputs.PushToPullRequestBranch.Target == "*" {
		// Allow pushing to any pull request - no specific context required
		baseCondition = "always()"
	} else {
		// Default behavior: only run in pull request context, or issue context with a linked PR
		baseCondition = "(github.event.issue.number && github.event.issue.pull_request) || github.event.pull_request"
	}

	// If this is a command workflow, combine the command trigger condition with the base condition
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()

		// Combine command condition with base condition using AND
		if baseCondition == "always()" {
			// If base condition is always(), just use the command condition
			jobCondition = commandConditionStr
		} else {
			// Combine both conditions with AND
			jobCondition = fmt.Sprintf("(%s) && (%s)", commandConditionStr, baseCondition)
		}
	} else {
		// No command trigger, just use the base condition
		jobCondition = baseCondition
	}

	job := &Job{
		Name:           "push_to_pr_branch",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: write\n      pull-requests: read\n      issues: read",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parsePushToPullRequestBranchConfig handles push-to-pr-branch configuration
func (c *Compiler) parsePushToPullRequestBranchConfig(outputMap map[string]any) *PushToPullRequestBranchConfig {
	if configData, exists := outputMap["push-to-pr-branch"]; exists {
		pushToBranchConfig := &PushToPullRequestBranchConfig{
			IfNoChanges: "warn", // Default behavior: warn when no changes
		}

		// Handle the case where configData is nil (push-to-pr-branch: with no value)
		if configData == nil {
			return pushToBranchConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse target (optional, similar to add-comment)
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					pushToBranchConfig.Target = targetStr
				}
			}

			// Parse if-no-changes (optional, defaults to "warn")
			if ifNoChanges, exists := configMap["if-no-changes"]; exists {
				if ifNoChangesStr, ok := ifNoChanges.(string); ok {
					// Validate the value
					switch ifNoChangesStr {
					case "warn", "error", "ignore":
						pushToBranchConfig.IfNoChanges = ifNoChangesStr
					default:
						// Invalid value, use default and log warning
						if c.verbose {
							fmt.Printf("Warning: invalid if-no-changes value '%s', using default 'warn'\n", ifNoChangesStr)
						}
						pushToBranchConfig.IfNoChanges = "warn"
					}
				}
			}

			// Parse github-token
			if githubToken, exists := configMap["github-token"]; exists {
				if githubTokenStr, ok := githubToken.(string); ok {
					pushToBranchConfig.GitHubToken = githubTokenStr
				}
			}
		}

		return pushToBranchConfig
	}

	return nil
}
