package workflow

import (
	"fmt"
	"strings"
)

// PushToPullRequestBranchConfig holds configuration for pushing changes to a specific branch from agent output
type PushToPullRequestBranchConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               string   `yaml:"target,omitempty"`        // Target for push-to-pull-request-branch: like add-comment but for pull requests
	TitlePrefix          string   `yaml:"title-prefix,omitempty"`  // Required title prefix for pull request validation
	Labels               []string `yaml:"labels,omitempty"`        // Required labels for pull request validation
	IfNoChanges          string   `yaml:"if-no-changes,omitempty"` // Behavior when no changes to push: "warn", "error", or "ignore" (default: "warn")
}

// buildCreateOutputPushToPullRequestBranchJob creates the push_to_pull_request_branch job
func (c *Compiler) buildCreateOutputPushToPullRequestBranchJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.PushToPullRequestBranch == nil {
		return nil, fmt.Errorf("safe-outputs.push-to-pull-request-branch configuration is required")
	}

	var steps []string

	// Step 1: Download patch artifact
	steps = append(steps, "      - name: Download patch artifact\n")
	steps = append(steps, "        continue-on-error: true\n")
	steps = append(steps, "        uses: actions/download-artifact@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: aw.patch\n")
	steps = append(steps, "          path: /tmp/gh-aw/\n")

	// Step 2: Checkout repository
	steps = buildCheckoutRepository(steps, c)

	// Step 3: Configure Git credentials
	steps = append(steps, c.generateGitConfigurationSteps()...)

	// Step 4: Push to Branch
	steps = append(steps, "      - name: Push to Branch\n")
	steps = append(steps, "        id: push_to_pull_request_branch\n")
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
	// Pass the title prefix configuration
	if data.SafeOutputs.PushToPullRequestBranch.TitlePrefix != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_TITLE_PREFIX: %q\n", data.SafeOutputs.PushToPullRequestBranch.TitlePrefix))
	}
	// Pass the labels configuration
	if len(data.SafeOutputs.PushToPullRequestBranch.Labels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.PushToPullRequestBranch.Labels, ",")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_LABELS: %q\n", labelsStr))
	}
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
		"branch_name": "${{ steps.push_to_pull_request_branch.outputs.branch_name }}",
		"commit_sha":  "${{ steps.push_to_pull_request_branch.outputs.commit_sha }}",
		"push_url":    "${{ steps.push_to_pull_request_branch.outputs.push_url }}",
	}

	safeOutputCondition := BuildSafeOutputType("push_to_pull_request_branch", data.SafeOutputs.PushToPullRequestBranch.Min)
	issueWithPR := &AndNode{
		Left:  &ExpressionNode{Expression: "github.event.issue.number"},
		Right: &ExpressionNode{Expression: "github.event.issue.pull_request"},
	}
	baseCondition := &OrNode{
		Left:  issueWithPR,
		Right: &ExpressionNode{Expression: "github.event.pull_request"},
	}
	jobCondition := &AndNode{
		Left:  safeOutputCondition,
		Right: baseCondition,
	}

	job := &Job{
		Name:           "push_to_pull_request_branch",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: write\n      pull-requests: read\n      issues: read",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

func buildCheckoutRepository(steps []string, c *Compiler) []string {
	steps = append(steps, "      - name: Checkout repository\n")
	steps = append(steps, "        uses: actions/checkout@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          fetch-depth: 0\n")
	if c.trialMode {
		if c.trialLogicalRepoSlug != "" {
			steps = append(steps, fmt.Sprintf("          repository: %s\n", c.trialLogicalRepoSlug))
			// trialTargetRepoName := strings.Split(c.trialLogicalRepoSlug, "/")
			// if len(trialTargetRepoName) == 2 {
			// 	steps = append(steps, fmt.Sprintf("          path: %s\n", trialTargetRepoName[1]))
			// }
		}
		steps = append(steps, "          token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}\n")
	}
	return steps
}

// parsePushToPullRequestBranchConfig handles push-to-pull-request-branch configuration
func (c *Compiler) parsePushToPullRequestBranchConfig(outputMap map[string]any) *PushToPullRequestBranchConfig {
	if configData, exists := outputMap["push-to-pull-request-branch"]; exists {
		pushToBranchConfig := &PushToPullRequestBranchConfig{
			IfNoChanges: "warn", // Default behavior: warn when no changes
		}

		// Handle the case where configData is nil (push-to-pull-request-branch: with no value)
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

			// Parse title-prefix
			if titlePrefix, exists := configMap["title-prefix"]; exists {
				if titlePrefixStr, ok := titlePrefix.(string); ok {
					pushToBranchConfig.TitlePrefix = titlePrefixStr
				}
			}

			// Parse labels
			if labels, exists := configMap["labels"]; exists {
				if labelsArray, ok := labels.([]any); ok {
					var labelStrings []string
					for _, label := range labelsArray {
						if labelStr, ok := label.(string); ok {
							labelStrings = append(labelStrings, labelStr)
						}
					}
					pushToBranchConfig.Labels = labelStrings
				}
			}

			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &pushToBranchConfig.BaseSafeOutputConfig)
		}

		return pushToBranchConfig
	}

	return nil
}
