package workflow

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var notifyCommentLog = logger.New("workflow:notify_comment")

// buildConclusionJob creates a job that handles workflow completion tasks
// This job is generated when safe-outputs are configured and handles:
// - Updating status comments (if status-comment: true)
// - Processing noop messages
// - Handling agent failures
// - Recording missing tools
// This job runs when:
// 1. always() - runs even if agent fails
// 2. Agent job was not skipped
// 3. NO add_comment output was produced by the agent (avoids duplicate updates)
// This job depends on all safe output jobs to ensure it runs last
func (c *Compiler) buildConclusionJob(data *WorkflowData, mainJobName string, safeOutputJobNames []string) (*Job, error) {
	notifyCommentLog.Printf("Building conclusion job: main_job=%s, safe_output_jobs_count=%d", mainJobName, len(safeOutputJobNames))
	// Always create this job when safe-outputs exist (because noop is always enabled)
	// This ensures noop messages can be handled even without reactions
	if data.SafeOutputs == nil {
		notifyCommentLog.Printf("Skipping job: no safe-outputs configured")
		return nil, nil // No safe-outputs configured, no need for conclusion job
	}
	steps := c.buildConclusionSetupSteps(data)
	steps = append(steps, c.buildConclusionNoOpStep(data, mainJobName)...)
	steps = append(steps, c.buildConclusionDetectionRunsStep(data, mainJobName)...)
	steps = append(steps, c.buildConclusionMissingToolStep(data, mainJobName)...)
	steps = append(steps, c.buildConclusionReportIncompleteStep(data, mainJobName)...)
	messagesJSON := serializeConclusionMessagesJSON(data)
	agentFailureSteps, err := c.buildAgentFailureStep(data, mainJobName, messagesJSON)
	if err != nil {
		return nil, err
	}
	steps = append(steps, agentFailureSteps...)
	steps = append(steps, c.buildConclusionReportFailedJobsStep(data, mainJobName)...)
	customEnvVars := c.buildConclusionScriptEnvVars(data, mainJobName, safeOutputJobNames, messagesJSON)
	var token string
	if data.SafeOutputs != nil && data.SafeOutputs.AddComments != nil {
		token = data.SafeOutputs.AddComments.GitHubToken
	}
	// Only add the conclusion update step if status comments are explicitly enabled
	if data.StatusComment != nil && *data.StatusComment {
		steps = append(steps, c.buildGitHubScriptStepWithoutDownload(data, GitHubScriptStepConfig{
			StepName:      "Update reaction comment with completion status",
			StepID:        "conclusion",
			MainJobName:   mainJobName,
			CustomEnvVars: customEnvVars,
			Script:        getNotifyCommentErrorScript(),
			ScriptFile:    "notify_comment_error.cjs",
			CustomToken:   token,
		})...)
	}
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}
	needs := buildConclusionJobNeeds(data, mainJobName, safeOutputJobNames)
	// If any message template references needs.pre_activation.outputs.*, add pre_activation
	// as a dependency so that GitHub Actions can resolve the expression at runtime.
	if data.SafeOutputs != nil && messagesContainPreActivationRef(data.SafeOutputs.Messages) {
		if _, exists := c.jobManager.GetJob(string(constants.PreActivationJobName)); exists {
			preActName := string(constants.PreActivationJobName)
			if !slices.Contains(needs, preActName) {
				needs = append(needs, preActName)
				notifyCommentLog.Print("Added pre_activation dependency to conclusion job (messages reference pre_activation outputs)")
			}
		}
	}
	notifyCommentLog.Printf("Job built successfully: dependencies_count=%d", len(needs))
	conclusionPerms := ComputePermissionsForSafeOutputs(data.SafeOutputs)
	// When observability.otlp.github-app is configured without app-id/private-key
	// credentials, id-token: write is needed so the conclusion job can mint the OTLP
	// OIDC token via core.getIDToken(audience) (mirrors threat_detection_job.go).
	if hasOTLPGitHubOIDCAuth(data.ParsedFrontmatter, data.RawFrontmatter) {
		conclusionPerms.Set(PermissionIdToken, PermissionWrite)
	}
	// The daily-AIC usage cache save step must not run with a fully read-only GITHUB_TOKEN.
	// If safe-outputs already granted some writable scope (for example issues: write for
	// comment updates), reuse that existing write access instead of broadening the job.
	if needsDailyAICCachePermission(data) && !conclusionPerms.HasAnyWriteScope() {
		conclusionPerms.Set(PermissionActions, PermissionWrite)
	}
	// The report-failed-jobs step lists workflow run jobs (actions: read) and creates issues
	// (issues: write). Ensure the conclusion job has at least those permissions when the
	// feature is enabled (default: true).
	if data.SafeOutputs == nil || data.SafeOutputs.ReportFailedJobs == nil || *data.SafeOutputs.ReportFailedJobs {
		if level, ok := conclusionPerms.Get(PermissionActions); !ok || level == PermissionNone {
			conclusionPerms.Set(PermissionActions, PermissionRead)
		}
		if level, ok := conclusionPerms.Get(PermissionIssues); !ok || level == PermissionNone {
			conclusionPerms.Set(PermissionIssues, PermissionWrite)
		}
	}
	return &Job{
		Name:        "conclusion",
		If:          RenderCondition(c.buildConclusionJobCondition(data, mainJobName, safeOutputJobNames)),
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		Environment: c.indentYAMLLines(resolveSafeOutputsEnvironment(data), "    "),
		Permissions: conclusionPerms.RenderToYAML(),
		Concurrency: c.buildConclusionJobConcurrency(data),
		Steps:       steps,
		Needs:       needs,
		Outputs:     buildConclusionJobOutputs(data),
	}, nil
}

// buildUsageArtifactUploadSteps creates steps that collect and upload a compact usage artifact.
// The artifact includes aw_info.json, aw-info.jsonl, agent_usage.json, agent_usage.jsonl, detection_usage.jsonl,
// evals.jsonl, and agent/detection token usage JSONL files (when present).
// It also downloads the safe-outputs-items artifact so that generate_usage_activity_summary.cjs
// can include safe-output item counts in the activity summary without requiring a separate artifact download.
func buildUsageArtifactUploadSteps(prefix string, hasEvals bool, pinAction func(string) string) []string {
	usageArtifactName := prefix + "usage"
	safeOutputsItemsArtifactName := prefix + constants.SafeOutputItemsArtifactName
	steps := []string{
		"      - name: Download Safe Outputs Items Manifest\n",
		"        id: download-safe-outputs-manifest\n",
		"        if: always()\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", pinAction("actions/download-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %s\n", safeOutputsItemsArtifactName),
		"          path: /tmp/gh-aw/\n",
	}
	if hasEvals {
		evalsArtifactName := prefix + constants.EvalsArtifactName
		steps = append(steps,
			"      - name: Download evals artifact\n",
			"        id: download-evals-artifact\n",
			"        if: always()\n",
			"        continue-on-error: true\n",
			fmt.Sprintf("        uses: %s\n", pinAction("actions/download-artifact")),
			"        with:\n",
			fmt.Sprintf("          name: %s\n", evalsArtifactName),
			"          path: /tmp/gh-aw/evals/\n",
			"      - name: Report unavailable evals artifact\n",
			"        if: always() && steps.download-evals-artifact.outcome == 'failure'\n",
			"        run: echo \"::notice::Evals artifact unavailable; evals were skipped or failed upstream.\"\n",
		)
	}
	steps = append(steps,
		"      - name: Collect usage artifact files\n",
		"        if: always()\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        run: bash \"%s/collect_usage_artifact_files.sh\"\n", SetupActionDestinationShell),
		"      - name: Upload usage artifact\n",
		"        if: always()\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", pinAction("actions/upload-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %s\n", usageArtifactName),
		"          path: |\n",
		"            /tmp/gh-aw/usage/aw_info.json\n",
		"            /tmp/gh-aw/usage/aw-info.jsonl\n",
		"            /tmp/gh-aw/usage/agent_usage.json\n",
		"            /tmp/gh-aw/usage/agent_usage.jsonl\n",
		"            /tmp/gh-aw/usage/detection_usage.jsonl\n",
		"            /tmp/gh-aw/usage/evals.jsonl\n",
		"            /tmp/gh-aw/usage/github_rate_limits.jsonl\n",
		"            /tmp/gh-aw/usage/agent/token_usage.jsonl\n",
		"            /tmp/gh-aw/usage/detection/token_usage.jsonl\n",
		"            /tmp/gh-aw/usage/activity/summary.json\n",
		"          if-no-files-found: ignore\n",
	)
	return steps
}

// buildDailyAICUsageCacheSteps creates steps that compute AIC for the current run and persist
// it to a per-workflow JSONL cache via actions/cache/save.  The cache is restored by the
// activation job so that subsequent guardrail checks can skip artifact downloads for known runs.
//
// The sequence is: restore latest snapshot → append current run entry → save updated snapshot.
// The restore step uses a prefix restore-key so it picks up the most recent snapshot even when
// the exact key (which includes the current run ID) does not exist yet.
func buildDailyAICUsageCacheSteps(data *WorkflowData, pinAction func(string) string) []string {
	sanitized := SanitizeWorkflowIDForCacheKey(data.WorkflowID)
	cacheKeyPrefix := fmt.Sprintf("agentic-workflow-usage-%s-", sanitized)
	cacheKey := cacheKeyPrefix + "${{ github.run_id }}"
	return []string{
		"      - name: Restore daily AIC usage cache\n",
		"        id: restore-daily-aic-cache-conclusion\n",
		"        if: always()\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", pinAction("actions/cache/restore")),
		"        with:\n",
		fmt.Sprintf("          key: %s\n", cacheKey),
		fmt.Sprintf("          restore-keys: %s\n", cacheKeyPrefix),
		"          path: /tmp/gh-aw/agentic-workflow-usage-cache.jsonl\n",
		"      - name: Write daily AIC usage cache entry\n",
		"        id: write-daily-aic-cache\n",
		"        if: always()\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", pinAction("actions/github-script")),
		"        with:\n",
		"          github-token: ${{ github.token }}\n",
		"          script: |\n",
		"            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n",
		"            setupGlobals(core, github, context);\n",
		"            const { main } = require('" + SetupActionDestination + "/write_daily_aic_usage_cache.cjs');\n",
		"            await main();\n",
		"      - name: Save daily AIC usage cache\n",
		"        id: save-daily-aic-cache\n",
		"        if: always()\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", pinAction("actions/cache/save")),
		"        with:\n",
		fmt.Sprintf("          key: %s\n", cacheKey),
		"          path: /tmp/gh-aw/agentic-workflow-usage-cache.jsonl\n",
		// Upload the cache file as an artifact so the activation job's artifact-based
		// fallback can retrieve it on a different PR branch where actions/cache is
		// branch-scoped and would otherwise always miss.
		"      - name: Upload daily AIC usage cache artifact\n",
		"        id: upload-daily-aic-cache\n",
		"        if: always()\n",
		"        continue-on-error: true\n",
		fmt.Sprintf("        uses: %s\n", pinAction("actions/upload-artifact")),
		"        with:\n",
		"          name: aic-usage-cache\n",
		"          path: /tmp/gh-aw/agentic-workflow-usage-cache.jsonl\n",
		"          if-no-files-found: ignore\n",
		"          retention-days: 7\n",
	}
}

// isGroupConcurrencyQueueEnabled reports whether compiler-generated concurrency groups
// should include queue: max. The feature is enabled by default and can be disabled
// with features.group-concurrency-queue: false.
func isGroupConcurrencyQueueEnabled(data *WorkflowData) bool {
	flag := strings.ToLower(strings.TrimSpace(string(constants.GroupConcurrencyQueueFeatureFlag)))
	if data != nil && data.Features != nil {
		for key, value := range data.Features {
			if strings.EqualFold(key, flag) {
				return parseGroupConcurrencyQueueFeatureValue(value)
			}
		}
	}
	return true
}

func parseGroupConcurrencyQueueFeatureValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		switch normalized {
		case "false", "0", "off", "no":
			return false
		default:
			return true
		}
	default:
		return true
	}
}
