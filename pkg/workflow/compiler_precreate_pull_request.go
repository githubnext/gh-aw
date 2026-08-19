package workflow

import "fmt"

const preCreatePullRequestAppTokenStepID = "pre-create-pull-request-app-token"

func isPreCreatePullRequestEnabled(data *WorkflowData) bool {
	if data == nil || data.SafeOutputs == nil || data.SafeOutputs.CreatePullRequests == nil || !data.SafeOutputs.CreatePullRequests.PreCreate {
		return false
	}
	// Staged mode is preview-only and must not perform any API side effects, so
	// pre-creation (branch, draft pull request, check run) is disabled when the
	// handler or the whole safe-outputs block compiles as staged.
	return !isPreCreatePullRequestStaged(data)
}

// isPreCreatePullRequestStaged reports whether create-pull-request compiles in staged
// mode, either through the global safe-outputs staged flag or the handler-level one.
func isPreCreatePullRequestStaged(data *WorkflowData) bool {
	return isHandlerStaged(templatableBoolIsTrue(data.SafeOutputs.Staged), data.SafeOutputs.CreatePullRequests.Staged)
}

func preCreatePullRequestApp(data *WorkflowData) *GitHubAppConfig {
	if !isPreCreatePullRequestEnabled(data) {
		return nil
	}
	if app := data.SafeOutputs.CreatePullRequests.GitHubApp; app != nil {
		return app
	}
	return data.SafeOutputs.GitHubApp
}

func preCreatePullRequestFallbackToken(data *WorkflowData) string {
	token := data.SafeOutputs.CreatePullRequests.GitHubToken
	if token == "" {
		token = data.SafeOutputs.GitHubToken
	}
	return getEffectiveSafeOutputGitHubToken(token)
}

// buildPreCreatePullRequestTokenSteps resolves the token used by the pre-create steps.
// When a GitHub App is configured and stepName is non-empty, it also returns the steps
// that mint the app token; passing an empty stepName reuses a token minted by an earlier
// step with the same stepID.
func (c *Compiler) buildPreCreatePullRequestTokenSteps(data *WorkflowData, app *GitHubAppConfig, permissions *Permissions, stepName string, stepID string) ([]string, string) {
	token := preCreatePullRequestFallbackToken(data)
	if app == nil {
		return nil, token
	}

	var steps []string
	if stepName != "" {
		steps = c.buildGitHubAppTokenMintStepWithMeta(app, permissions, "", "", stepName, stepID)
	}
	appToken := fmt.Sprintf("${{ steps.%s.outputs.token }}", stepID)
	if app.shouldIgnoreMissingKey() {
		return steps, combineTokenExpressions(appToken, token)
	}
	return steps, appToken
}

func (c *Compiler) addActivationPreCreatePullRequestStep(ctx *activationJobBuildContext) {
	if !isPreCreatePullRequestEnabled(ctx.data) {
		return
	}

	permissions := NewPermissionsContentsWritePRWrite()
	permissions.Set(PermissionChecks, PermissionWrite)
	tokenSteps, token := c.buildPreCreatePullRequestTokenSteps(
		ctx.data,
		preCreatePullRequestApp(ctx.data),
		permissions,
		"Generate GitHub App token (pre-create pull request)",
		preCreatePullRequestAppTokenStepID,
	)
	ctx.steps = append(ctx.steps, tokenSteps...)

	ctx.steps = append(ctx.steps,
		"      - name: Pre-create pull request\n",
		"        id: pre-create-pull-request\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", ctx.data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", ctx.data.Name),
	)
	if baseBranch := ctx.data.SafeOutputs.CreatePullRequests.BaseBranch; baseBranch != "" {
		ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_CUSTOM_BASE_BRANCH: %q\n", baseBranch))
	}
	if titlePrefix := ctx.data.SafeOutputs.CreatePullRequests.TitlePrefix; titlePrefix != "" {
		ctx.steps = append(ctx.steps, fmt.Sprintf("          GH_AW_PR_TITLE_PREFIX: %q\n", titlePrefix))
	}
	ctx.steps = append(ctx.steps,
		"        with:\n",
		fmt.Sprintf("          github-token: %s\n", token),
		"          script: |\n",
		generateGitHubScriptWithRequire("pre_create_pull_request.cjs"),
	)
	ctx.outputs["pre_created_pull_request_number"] = "${{ steps.pre-create-pull-request.outputs.pull_request_number }}"
	ctx.outputs["pre_created_pull_request_url"] = "${{ steps.pre-create-pull-request.outputs.pull_request_url }}"
	ctx.outputs["pre_created_pull_request_branch"] = "${{ steps.pre-create-pull-request.outputs.branch }}"
	ctx.outputs["pre_created_pull_request_check_run_id"] = "${{ steps.pre-create-pull-request.outputs.check_run_id }}"
}

func (c *Compiler) buildConclusionPreCreatedCheckRunStep(data *WorkflowData) []string {
	if !isPreCreatePullRequestEnabled(data) {
		return nil
	}

	app := preCreatePullRequestApp(data)
	// A handler-level app must be minted here; a safe-outputs-wide app token is already
	// minted by an earlier conclusion step, so only its step ID is reused.
	stepID := "safe-outputs-app-token"
	stepName := ""
	if data.SafeOutputs.CreatePullRequests.GitHubApp != nil {
		stepID = preCreatePullRequestAppTokenStepID
		stepName = "Generate GitHub App token (complete pre-created check)"
	}
	steps, token := c.buildPreCreatePullRequestTokenSteps(data, app, NewPermissionsChecksWriteContentsWritePRWrite(), stepName, stepID)

	steps = append(steps,
		"      - name: Complete pre-created pull request check\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		"          GH_AW_PRE_CREATED_CHECK_RUN_ID: ${{ needs.activation.outputs.pre_created_pull_request_check_run_id }}\n",
		"          GH_AW_PRE_CREATED_PULL_REQUEST_NUMBER: ${{ needs.activation.outputs.pre_created_pull_request_number }}\n",
		"          GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH: ${{ needs.activation.outputs.pre_created_pull_request_branch }}\n",
		"          GH_AW_NEEDS: ${{ toJSON(needs) }}\n",
		"        with:\n",
		fmt.Sprintf("          github-token: %s\n", token),
		"          script: |\n",
		generateGitHubScriptWithRequire("complete_pre_created_check_run.cjs"),
	)
	return steps
}
