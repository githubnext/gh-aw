package workflow

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/sliceutil"
)

// compiler_activation_daily_aic contains daily AIC guardrail token and step builders.

const dailyAICAppTokenStepID = "daily-aic-app-token"

// buildDailyAICAppTokenMintStep generates a GitHub App token mint step dedicated
// to the daily AIC guardrail. The minted token is used only for the guardrail API
// calls, avoiding depletion of credentials held by the main activation app or
// GITHUB_TOKEN.
//
// The step is gated on maxDailyAICreditsConfiguredIfExpr so it is skipped when
// the guardrail is not active at runtime.
func (c *Compiler) buildDailyAICAppTokenMintStep(app *GitHubAppConfig) []string {
	var steps []string
	steps = append(steps, "      - name: Generate GitHub App token for daily AIC guardrail\n")
	steps = append(steps, fmt.Sprintf("        id: %s\n", dailyAICAppTokenStepID))
	if app.shouldIgnoreMissingKey() {
		guard := buildIgnoreIfMissingCondition(app)
		steps = appendStepEnvAssignments(steps, guard.EnvAssignments)
		if condition := combineGitHubIfExpressions(maxDailyAICreditsConfiguredIfExpr, guard.Condition); condition != "" {
			steps = append(steps, fmt.Sprintf("        if: %s\n", condition))
		} else {
			steps = append(steps, fmt.Sprintf("        if: %s\n", maxDailyAICreditsConfiguredIfExpr))
		}
	} else {
		steps = append(steps, fmt.Sprintf("        if: %s\n", maxDailyAICreditsConfiguredIfExpr))
	}
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          client-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))
	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))
	if len(app.Repositories) == 1 && app.Repositories[0] == "*" {
		// Org-wide access: omit repositories field entirely
	} else if len(app.Repositories) == 1 {
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", app.Repositories[0]))
	} else if len(app.Repositories) > 1 {
		steps = append(steps, "          repositories: |-\n")
		for _, repo := range app.Repositories {
			steps = append(steps, fmt.Sprintf("            %s\n", repo))
		}
	} else {
		steps = append(steps, "          repositories: ${{ github.event.repository.name }}\n")
	}
	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")
	// Build permission fields: baseline is actions: read (required for guardrail script to read
	// workflow run data). Merge any user-configured app.Permissions on top so callers can extend
	// or override the scope without changing the compiler. Sort keys for deterministic output.
	basePerms := NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionActions: PermissionRead,
	})
	permissionFields := convertPermissionsToAppTokenFields(basePerms)
	for key, val := range app.Permissions {
		scope := convertStringToPermissionScope(key)
		if scope == "" {
			safeOutputsAppLog.Printf("Skipping unknown permission scope %q in max-daily-ai-credits github-app.permissions", key)
			continue
		}
		level := strings.ToLower(strings.TrimSpace(val))
		tempPerms := NewPermissionsFromMap(map[PermissionScope]PermissionLevel{scope: PermissionLevel(level)})
		maps.Copy(permissionFields, convertPermissionsToAppTokenFields(tempPerms))
	}
	for _, key := range sliceutil.SortedKeys(permissionFields) {
		steps = append(steps, fmt.Sprintf("          %s: %s\n", key, permissionFields[key]))
	}
	return steps
}

// resolveDailyAICToken returns the GitHub token to use for daily AIC guardrail steps.
// When a dedicated MaxDailyAICreditsGitHubApp is configured, it references the
// minted token from that step. Otherwise it falls back to the activation token.
func (c *Compiler) resolveDailyAICToken(data *WorkflowData) string {
	if data.MaxDailyAICreditsGitHubApp != nil {
		if data.MaxDailyAICreditsGitHubApp.shouldIgnoreMissingKey() {
			return combineTokenExpressions(
				fmt.Sprintf("${{ steps.%s.outputs.token }}", dailyAICAppTokenStepID),
				c.resolveActivationToken(data),
			)
		}
		return fmt.Sprintf("${{ steps.%s.outputs.token }}", dailyAICAppTokenStepID)
	}
	return c.resolveActivationToken(data)
}

func (c *Compiler) buildActivationDailyAICGuardrailStep(data *WorkflowData) []string {
	compilerActivationJobLog.Printf("Building daily AIC guardrail step: dedicated_app=%t, cache_enabled=%t", data.MaxDailyAICreditsGitHubApp != nil, data.WorkflowID != "")
	var steps []string
	// When a dedicated GitHub App is configured for the daily AIC guardrail, mint
	// its token first so the subsequent steps can reference it.
	if data.MaxDailyAICreditsGitHubApp != nil {
		compilerActivationJobLog.Print("Prepending dedicated daily-AIC app-token mint step")
		steps = append(steps, c.buildDailyAICAppTokenMintStep(data.MaxDailyAICreditsGitHubApp)...)
	}
	if data.WorkflowID != "" {
		steps = append(steps, c.buildDailyAICCacheRestoreSteps(data)...)
	}
	steps = append(steps, c.buildDailyAICGuardrailScriptStep(data)...)
	return steps
}

func (c *Compiler) buildDailyAICCacheRestoreSteps(data *WorkflowData) []string {
	sanitized := SanitizeWorkflowIDForCacheKey(data.WorkflowID)
	cacheKeyPrefix := fmt.Sprintf("agentic-workflow-usage-%s-", sanitized)
	steps := []string{
		"      - name: Restore daily AIC usage cache\n",
		"        id: restore-daily-aic-cache\n",
		fmt.Sprintf("        if: %s\n", maxDailyAICreditsConfiguredIfExpr),
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/cache/restore", data)),
		"        with:\n",
		fmt.Sprintf("          key: %s${{ github.run_id }}\n", cacheKeyPrefix),
		fmt.Sprintf("          restore-keys: %s\n", cacheKeyPrefix),
		"          path: /tmp/gh-aw/agentic-workflow-usage-cache.jsonl\n",
	}
	return append(steps, c.buildDailyAICCacheFallbackSteps(data)...)
}

func (c *Compiler) buildDailyAICCacheFallbackSteps(data *WorkflowData) []string {
	return []string{
		"      - name: Restore daily AIC usage cache (artifact fallback)\n",
		"        id: restore-daily-aic-cache-fallback\n",
		fmt.Sprintf("        if: %s\n", maxDailyAICreditsConfiguredIfExpr),
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		"          GH_AW_RESTORE_DAILY_AIC_CACHE_HIT: ${{ steps.restore-daily-aic-cache.outputs.cache-hit }}\n",
		"          GH_AW_RESTORE_DAILY_AIC_CACHE_MATCHED_KEY: ${{ steps.restore-daily-aic-cache.outputs.cache-matched-key }}\n",
		"        with:\n",
		fmt.Sprintf("          github-token: %s\n", c.resolveDailyAICToken(data)),
		"          script: |\n",
		"            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('" + SetupActionDestination + "/restore_aic_usage_cache_fallback.cjs');\n",
		"            await main();\n",
	}
}

func (c *Compiler) buildDailyAICGuardrailScriptStep(data *WorkflowData) []string {
	steps := []string{
		"      - name: Check daily workflow token guardrail\n",
		"        id: daily-effective-workflow-guardrail\n",
		fmt.Sprintf("        if: %s\n", maxDailyAICreditsConfiguredIfExpr),
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)),
		"        env:\n",
		fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name),
		fmt.Sprintf("          GH_AW_WORKFLOW_ID: %q\n", data.WorkflowID),
		"          GH_AW_RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}\n",
		"          GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT: ${{ github.event.inputs.aw_context || '' }}\n",
		fmt.Sprintf("          GH_AW_HAS_SLASH_COMMAND: %q\n", strconv.FormatBool(len(data.Command) > 0)),
		fmt.Sprintf("          GH_AW_HAS_LABEL_COMMAND: %q\n", strconv.FormatBool(len(data.LabelCommand) > 0)),
		fmt.Sprintf("          GH_AW_GITHUB_TOKEN: %s\n", c.resolveDailyAICToken(data)),
	}
	steps = append(steps, buildTemplatableIntEnvVar(maxDailyAICreditsEnvVar, data.MaxDailyAICredits)...)
	return append(steps, c.buildDailyAICGuardrailScriptWith(data)...)
}

func (c *Compiler) buildDailyAICGuardrailScriptWith(data *WorkflowData) []string {
	return []string{
		"        with:\n",
		fmt.Sprintf("          github-token: %s\n", c.resolveDailyAICToken(data)),
		"          script: |\n",
		"            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		"            const { main } = require('" + SetupActionDestination + "/check_daily_aic_workflow_guardrail.cjs');\n",
		"            await main();\n",
	}
}

func buildDailyAICActivationJobEnv(data *WorkflowData) map[string]string {
	if !hasMaxDailyAICGuardrail(data) || !hasMaxDailyAICFrontmatterConfig(data) {
		return nil
	}
	value := strings.TrimSpace(*data.MaxDailyAICredits)
	if value == "" {
		compilerActivationJobLog.Print("Daily AIC guardrail configured but max-daily-ai-credits value is empty; omitting activation job env")
		return nil
	}
	if isExpression(value) {
		return map[string]string{maxDailyAICreditsEnvVar: value}
	}
	return map[string]string{maxDailyAICreditsEnvVar: strconv.Quote(value)}
}
