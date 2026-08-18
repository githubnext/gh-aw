package workflow

import "fmt"

const preCreatePullRequestAppTokenStepID = "pre-create-pull-request-app-token"

func isPreCreatePullRequestEnabled(data *WorkflowData) bool {
	return data != nil &&
		data.SafeOutputs != nil &&
		data.SafeOutputs.CreatePullRequests != nil &&
		data.SafeOutputs.CreatePullRequests.PreCreate
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

func (c *Compiler) addActivationPreCreatePullRequestStep(ctx *activationJobBuildContext) {
	if !isPreCreatePullRequestEnabled(ctx.data) {
		return
	}

	app := preCreatePullRequestApp(ctx.data)
	token := preCreatePullRequestFallbackToken(ctx.data)

	if app != nil {
		permissions := NewPermissionsContentsWritePRWrite()
		permissions.Set(PermissionChecks, PermissionWrite)
		ctx.steps = append(ctx.steps, c.buildGitHubAppTokenMintStepWithMeta(
			app,
			permissions,
			"",
			"",
			"Generate GitHub App token (pre-create pull request)",
			preCreatePullRequestAppTokenStepID,
		)...)
		appToken := fmt.Sprintf("${{ steps.%s.outputs.token }}", preCreatePullRequestAppTokenStepID)
		if app.shouldIgnoreMissingKey() {
			token = combineTokenExpressions(appToken, token)
		} else {
			token = appToken
		}
	}

	ctx.steps = append(ctx.steps,
		"      - name: Pre-create pull request\n",
		"        id: pre-create-pull-request\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", ctx.data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", ctx.data.Name),
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

	var steps []string
	app := preCreatePullRequestApp(data)
	token := preCreatePullRequestFallbackToken(data)
	if app != nil {
		stepID := "safe-outputs-app-token"
		if data.SafeOutputs.CreatePullRequests.GitHubApp != nil {
			stepID = preCreatePullRequestAppTokenStepID
			steps = append(steps, c.buildGitHubAppTokenMintStepWithMeta(
				app,
				NewPermissionsChecksWrite(),
				"",
				"",
				"Generate GitHub App token (complete pre-created check)",
				stepID,
			)...)
		}
		appToken := fmt.Sprintf("${{ steps.%s.outputs.token }}", stepID)
		if app.shouldIgnoreMissingKey() {
			token = combineTokenExpressions(appToken, token)
		} else {
			token = appToken
		}
	}

	steps = append(steps,
		"      - name: Complete pre-created pull request check\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		"          GH_AW_PRE_CREATED_CHECK_RUN_ID: ${{ needs.activation.outputs.pre_created_pull_request_check_run_id }}\n",
		"          GH_AW_NEEDS: ${{ toJSON(needs) }}\n",
		"        with:\n",
		fmt.Sprintf("          github-token: %s\n", token),
		"          script: |\n",
		generateGitHubScriptWithRequire("complete_pre_created_check_run.cjs"),
	)
	return steps
}
