package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var prSafeOutputsLog = logger.New("workflow:compiler_safe_outputs_prs")

// buildCreatePullRequestStepConfig builds the configuration for creating a pull request
func (c *Compiler) buildCreatePullRequestStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreatePullRequests
	prSafeOutputsLog.Printf("Building create pull request step config: draft=%v, if_no_changes=%s",
		cfg.Draft != nil && *cfg.Draft, cfg.IfNoChanges)

	var customEnvVars []string
	// Pass the base branch from GitHub context (required by create_pull_request.cjs)
	// Note: GH_AW_WORKFLOW_ID is now set at the job level and inherited by all steps
	customEnvVars = append(customEnvVars, "          GH_AW_BASE_BRANCH: ${{ github.ref_name }}\n")
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_PR_TITLE_PREFIX", cfg.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_PR_LABELS", cfg.Labels)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_PR_ALLOWED_LABELS", cfg.AllowedLabels)...)
	// Add draft setting - always set with default to true for backwards compatibility
	draftValue := true // Default value
	if cfg.Draft != nil {
		draftValue = *cfg.Draft
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_DRAFT: %q\n", fmt.Sprintf("%t", draftValue)))
	// Add if-no-changes setting - always set with default to "warn"
	ifNoChanges := cfg.IfNoChanges
	if ifNoChanges == "" {
		ifNoChanges = "warn" // Default value
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_IF_NO_CHANGES: %q\n", ifNoChanges))
	// Add allow-empty setting
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_ALLOW_EMPTY: %q\n", fmt.Sprintf("%t", cfg.AllowEmpty)))
	// Add max patch size setting
	maxPatchSize := 1024 // default 1024 KB
	if data.SafeOutputs.MaximumPatchSize > 0 {
		maxPatchSize = data.SafeOutputs.MaximumPatchSize
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MAX_PATCH_SIZE: %d\n", maxPatchSize))
	// Add activation comment information if available (for updating the comment with PR link)
	if data.AIReaction != "" && data.AIReaction != "none" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_ID: ${{ needs.%s.outputs.comment_id }}\n", constants.ActivationJobName))
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_REPO: ${{ needs.%s.outputs.comment_repo }}\n", constants.ActivationJobName))
	}
	// Add expires value if set (only for same-repo PRs - when target-repo is not set)
	if cfg.Expires > 0 && cfg.TargetRepoSlug == "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PR_EXPIRES: \"%d\"\n", cfg.Expires))
	}
	// Add temporary ID map from handler manager output (if handler manager is enabled)
	// This allows PRs to reference issues created by the handler manager using temporary IDs
	customEnvVars = append(customEnvVars, "          GH_AW_TEMPORARY_ID_MAP: ${{ steps.process_safe_outputs.outputs.temporary_id_map }}\n")
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("create_pull_request")

	// Build pre-steps for checkout and git config
	preSteps := c.buildCreatePullRequestPreStepsConsolidated(data, cfg, condition)

	return SafeOutputStepConfig{
		StepName:      "Create Pull Request",
		StepID:        "create_pull_request",
		ScriptName:    "create_pull_request",
		Script:        getCreatePullRequestScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
		PreSteps:      preSteps,
	}
}

// buildUpdatePullRequestStepConfig builds the configuration for updating a pull request
func (c *Compiler) buildUpdatePullRequestStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdatePullRequests
	prSafeOutputsLog.Print("Building update pull request step config")

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("update_pull_request")

	return SafeOutputStepConfig{
		StepName:      "Update Pull Request",
		StepID:        "update_pull_request",
		ScriptName:    "update_pull_request",
		Script:        getUpdatePullRequestScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildClosePullRequestStepConfig builds the configuration for closing a pull request
func (c *Compiler) buildClosePullRequestStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.ClosePullRequests

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("close_pull_request")

	return SafeOutputStepConfig{
		StepName:      "Close Pull Request",
		StepID:        "close_pull_request",
		ScriptName:    "close_pull_request",
		Script:        getClosePullRequestScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildMarkPullRequestAsReadyForReviewStepConfig builds the configuration for marking a PR as ready for review
func (c *Compiler) buildMarkPullRequestAsReadyForReviewStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.MarkPullRequestAsReadyForReview

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("mark_pull_request_as_ready_for_review")

	return SafeOutputStepConfig{
		StepName:      "Mark Pull Request as Ready for Review",
		StepID:        "mark_pull_request_as_ready_for_review",
		ScriptName:    "mark_pull_request_as_ready_for_review",
		Script:        getMarkPullRequestAsReadyForReviewScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildPushToPullRequestBranchStepConfig builds the configuration for pushing to a pull request branch
func (c *Compiler) buildPushToPullRequestBranchStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.PushToPullRequestBranch

	var customEnvVars []string
	// Add target config if set
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PUSH_TARGET: %q\n", cfg.Target))
	}
	// Add if-no-changes config if set
	if cfg.IfNoChanges != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PUSH_IF_NO_CHANGES: %q\n", cfg.IfNoChanges))
	}
	// Add title prefix if set (using same env var as create-pull-request)
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_PR_TITLE_PREFIX", cfg.TitlePrefix)...)
	// Add labels if set
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_PR_LABELS", cfg.Labels)...)
	// Add commit title suffix if set
	if cfg.CommitTitleSuffix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMIT_TITLE_SUFFIX: %q\n", cfg.CommitTitleSuffix))
	}
	// Add max patch size setting
	maxPatchSize := 1024 // default 1024 KB
	if data.SafeOutputs.MaximumPatchSize > 0 {
		maxPatchSize = data.SafeOutputs.MaximumPatchSize
	}
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MAX_PATCH_SIZE: %d\n", maxPatchSize))
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("push_to_pull_request_branch")

	// Build pre-steps for checkout and git config
	preSteps := c.buildPushToPullRequestBranchPreStepsConsolidated(data, cfg, condition)

	return SafeOutputStepConfig{
		StepName:      "Push To Pull Request Branch",
		StepID:        "push_to_pull_request_branch",
		ScriptName:    "push_to_pull_request_branch",
		Script:        getPushToPullRequestBranchScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
		PreSteps:      preSteps,
	}
}

// buildAddReviewerStepConfig builds the configuration for adding a reviewer
func (c *Compiler) buildAddReviewerStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AddReviewer

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("add_reviewer")

	return SafeOutputStepConfig{
		StepName:      "Add Reviewer",
		StepID:        "add_reviewer",
		ScriptName:    "add_reviewer",
		Script:        getAddReviewerScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildCreatePullRequestPreStepsConsolidated builds the pre-steps for create-pull-request
// in the consolidated safe outputs job
func (c *Compiler) buildCreatePullRequestPreStepsConsolidated(data *WorkflowData, cfg *CreatePullRequestsConfig, condition ConditionNode) []string {
	prSafeOutputsLog.Printf("Building create PR pre-steps: app_configured=%v, trial_mode=%v",
		data.SafeOutputs.App != nil, c.trialMode)
	var preSteps []string

	// Determine which token to use for checkout
	// If an app is configured, use the app token; otherwise use the default github.token
	var checkoutToken string
	var gitRemoteToken string
	if data.SafeOutputs.App != nil {
		checkoutToken = "${{ steps.app-token.outputs.token }}"
		gitRemoteToken = "${{ steps.app-token.outputs.token }}"
	} else {
		checkoutToken = "${{ github.token }}"
		gitRemoteToken = "${{ github.token }}"
	}

	// Step 1: Checkout repository with conditional execution
	preSteps = append(preSteps, "      - name: Checkout repository\n")
	// Add the condition to only checkout if create_pull_request will run
	preSteps = append(preSteps, fmt.Sprintf("        if: %s\n", condition.Render()))
	preSteps = append(preSteps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")))
	preSteps = append(preSteps, "        with:\n")
	preSteps = append(preSteps, fmt.Sprintf("          token: %s\n", checkoutToken))
	preSteps = append(preSteps, "          persist-credentials: false\n")
	preSteps = append(preSteps, "          fetch-depth: 1\n")
	if c.trialMode {
		if c.trialLogicalRepoSlug != "" {
			preSteps = append(preSteps, fmt.Sprintf("          repository: %s\n", c.trialLogicalRepoSlug))
		}
	}

	// Step 2: Configure Git credentials with conditional execution
	gitConfigSteps := []string{
		"      - name: Configure Git credentials\n",
		fmt.Sprintf("        if: %s\n", condition.Render()),
		"        env:\n",
		"          REPO_NAME: ${{ github.repository }}\n",
		"          SERVER_URL: ${{ github.server_url }}\n",
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          # Re-authenticate git with GitHub token\n",
		"          SERVER_URL_STRIPPED=\"${SERVER_URL#https://}\"\n",
		fmt.Sprintf("          git remote set-url origin \"https://x-access-token:%s@${SERVER_URL_STRIPPED}/${REPO_NAME}.git\"\n", gitRemoteToken),
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	}
	preSteps = append(preSteps, gitConfigSteps...)

	return preSteps
}

// buildPushToPullRequestBranchPreStepsConsolidated builds the pre-steps for push-to-pull-request-branch
// in the consolidated safe outputs job
func (c *Compiler) buildPushToPullRequestBranchPreStepsConsolidated(data *WorkflowData, cfg *PushToPullRequestBranchConfig, condition ConditionNode) []string {
	var preSteps []string

	// Determine which token to use for checkout
	// If an app is configured, use the app token; otherwise use the default github.token
	var checkoutToken string
	var gitRemoteToken string
	if data.SafeOutputs.App != nil {
		checkoutToken = "${{ steps.app-token.outputs.token }}"
		gitRemoteToken = "${{ steps.app-token.outputs.token }}"
	} else {
		checkoutToken = "${{ github.token }}"
		gitRemoteToken = "${{ github.token }}"
	}

	// Step 1: Checkout repository with conditional execution
	preSteps = append(preSteps, "      - name: Checkout repository\n")
	// Add the condition to only checkout if push_to_pull_request_branch will run
	preSteps = append(preSteps, fmt.Sprintf("        if: %s\n", condition.Render()))
	preSteps = append(preSteps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")))
	preSteps = append(preSteps, "        with:\n")
	preSteps = append(preSteps, fmt.Sprintf("          token: %s\n", checkoutToken))
	preSteps = append(preSteps, "          persist-credentials: false\n")
	preSteps = append(preSteps, "          fetch-depth: 1\n")
	if c.trialMode {
		if c.trialLogicalRepoSlug != "" {
			preSteps = append(preSteps, fmt.Sprintf("          repository: %s\n", c.trialLogicalRepoSlug))
		}
	}

	// Step 2: Configure Git credentials with conditional execution
	gitConfigSteps := []string{
		"      - name: Configure Git credentials\n",
		fmt.Sprintf("        if: %s\n", condition.Render()),
		"        env:\n",
		"          REPO_NAME: ${{ github.repository }}\n",
		"          SERVER_URL: ${{ github.server_url }}\n",
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          # Re-authenticate git with GitHub token\n",
		"          SERVER_URL_STRIPPED=\"${SERVER_URL#https://}\"\n",
		fmt.Sprintf("          git remote set-url origin \"https://x-access-token:%s@${SERVER_URL_STRIPPED}/${REPO_NAME}.git\"\n", gitRemoteToken),
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	}
	preSteps = append(preSteps, gitConfigSteps...)

	return preSteps
}
