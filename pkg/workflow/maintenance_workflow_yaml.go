package workflow

import (
	"context"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var maintenanceWorkflowYAMLLog = logger.New("workflow:maintenance_workflow_yaml")

// buildMaintenanceWorkflowYAMLOptions configures the maintenance workflow YAML builder.
type buildMaintenanceWorkflowYAMLOptions struct {
	cronSchedule        string
	scheduleDesc        string
	minExpiresDays      int
	runsOnValue         string
	actionMode          ActionMode
	version             string
	actionTag           string
	resolver            SHAResolver
	configuredRunsOn    RunsOnValue
	defaultBranch       string
	disableLabelTrigger bool
	maintenanceConfig   *MaintenanceConfig
	compileGitHubToken  string
	createCompilePR     bool
	copilotOrgBilling   bool // all Copilot workflows use copilot-requests: write (GITHUB_TOKEN); COPILOT_GITHUB_TOKEN is not required
}

// buildMaintenanceWorkflowYAML generates the complete YAML content for the
// agentics-maintenance.yml workflow. It is called by GenerateMaintenanceWorkflow
// after the cron schedule and setup parameters have been resolved.
const maintenanceWorkflowCustomInstructions = `This file defines the generated agentic maintenance workflow for this repository.
It runs scheduled cleanup for expiring safe outputs and supports manual maintenance operations.

This workflow is generated automatically when workflows use expiring safe outputs
or when repository maintenance features are enabled in .github/workflows/aw.json.

To disable maintenance workflow generation, set in .github/workflows/aw.json:
  {"maintenance": false}

Agentic maintenance docs:
  https://github.github.com/gh-aw/reference/ephemerals/#manual-maintenance-operations`

const maintenanceWorkflowEntrypointsPrefix = `  workflow_dispatch:
    inputs:
      operation:
        description: 'Optional maintenance operation to run'
        required: false
        type: choice
        default: ''
        options:
          - ''
          - 'disable'
          - 'enable'
          - 'update'
          - 'upgrade'
          - 'safe_outputs'
          - 'create_labels'
          - 'activity_report'
          - 'close_agentic_workflows_issues'
          - 'clean_cache_memories'
          - 'update_pull_request_branches'
          - 'validate'
          - 'forecast'
      run_url:
        description: 'Run URL or run ID to replay safe outputs from (e.g. https://github.com/owner/repo/actions/runs/12345 or 12345). Required when operation is safe_outputs.'
        required: false
        type: string
        default: ''
  workflow_call:
    inputs:
      operation:
        description: 'Optional maintenance operation to run (disable, enable, update, upgrade, safe_outputs, create_labels, activity_report, close_agentic_workflows_issues, clean_cache_memories, update_pull_request_branches, validate, forecast)'
        required: false
        type: string
        default: ''
      run_url:
        description: 'Run URL or run ID to replay safe outputs from (e.g. https://github.com/owner/repo/actions/runs/12345 or 12345). Required when operation is safe_outputs.'
        required: false
        type: string
        default: ''
    outputs:
      operation_completed:
        description: 'The maintenance operation that was completed (empty when none ran or a scheduled job ran)'
        value: ${{ jobs.run_operation.outputs.operation || inputs.operation }}
      applied_run_url:
`

func buildMaintenanceWorkflowYAML(
	ctx context.Context,
	opts buildMaintenanceWorkflowYAMLOptions,
) string {
	maintenanceWorkflowYAMLLog.Printf("Building maintenance workflow YAML: actionMode=%s minExpiresDays=%d cronSchedule=%q defaultBranch=%q disableLabelTrigger=%v createCompilePR=%v copilotOrgBilling=%v", opts.actionMode, opts.minExpiresDays, opts.cronSchedule, opts.defaultBranch, opts.disableLabelTrigger, opts.createCompilePR, opts.copilotOrgBilling)
	labelDisableJobEnabled := !opts.disableLabelTrigger && !opts.maintenanceConfig.IsJobDisabled("label_disable_agentic_workflow")
	labelApplySafeOutputsJobEnabled := !opts.disableLabelTrigger && !opts.maintenanceConfig.IsJobDisabled("label_apply_safe_outputs")
	setupActionRef := ResolveSetupActionReference(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver)

	var yaml strings.Builder
	writeMaintenanceWorkflowHeader(&yaml)
	writeMaintenanceOnTriggers(&yaml, opts, labelDisableJobEnabled, labelApplySafeOutputsJobEnabled)
	writeMaintenanceEntrypointsAndPermissions(&yaml, opts.maintenanceConfig)
	writeMaintenanceCloseExpiredJobs(&yaml, opts, setupActionRef)
	writeMaintenanceCleanupCacheMemoryJob(&yaml, opts, setupActionRef)
	writeMaintenanceRunOperationJob(&yaml, ctx, opts, setupActionRef)
	writeMaintenanceUpdatePullRequestBranchesJob(&yaml, opts, setupActionRef)
	if !opts.maintenanceConfig.IsJobDisabled("apply_safe_outputs") {
		writeMaintenanceApplySafeOutputsJob(&yaml, opts, setupActionRef)
	}
	writeMaintenanceCreateLabelsJob(&yaml, ctx, opts, setupActionRef)
	writeMaintenanceActivityReportJob(&yaml, ctx, opts, setupActionRef)
	writeMaintenanceForecastReportJob(&yaml, ctx, opts, setupActionRef)
	writeMaintenanceCloseAgenticIssuesJob(&yaml, opts, setupActionRef)
	writeMaintenanceValidateWorkflowsJob(&yaml, ctx, opts, setupActionRef)
	writeMaintenanceLabelJobs(&yaml, opts, setupActionRef, labelDisableJobEnabled, labelApplySafeOutputsJobEnabled)
	if opts.actionMode == ActionModeDev {
		writeMaintenanceDevJobs(&yaml, ctx, opts, setupActionRef)
	}
	return yaml.String()
}

func writeMaintenanceWorkflowHeader(yaml *strings.Builder) {
	header := GenerateWorkflowHeader("", "pkg/workflow/maintenance_workflow.go", maintenanceWorkflowCustomInstructions)
	yaml.WriteString(header)
	yaml.WriteString("name: Agentic Maintenance\n\non:\n")
}

func writeMaintenanceOnTriggers(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, labelDisableJobEnabled, labelApplySafeOutputsJobEnabled bool) {
	yaml.WriteString(`  schedule:
    - cron: "` + opts.cronSchedule + `"  # ` + opts.scheduleDesc + ` (based on minimum expires: ` + strconv.Itoa(opts.minExpiresDays) + ` days)
`)
	if opts.actionMode == ActionModeDev {
		maintenanceWorkflowYAMLLog.Printf("Adding dev-mode push trigger for branch %q", opts.defaultBranch)
		yaml.WriteString(`  push:
    branches:
      - ` + opts.defaultBranch + `
    paths:
      - '.github/workflows/*.md'
`)
	}
	if labelDisableJobEnabled || labelApplySafeOutputsJobEnabled {
		maintenanceWorkflowYAMLLog.Print("Adding issues:labeled trigger for label-triggered maintenance jobs")
		yaml.WriteString(`  issues:
    types: [labeled]
`)
	}
}

func writeMaintenanceEntrypointsAndPermissions(yaml *strings.Builder, maintenanceConfig *MaintenanceConfig) {
	appliedRunURLValue := "${{ jobs.apply_safe_outputs.outputs.run_url }}"
	appliedRunURLDescription := "The run URL that safe outputs were applied from"
	if maintenanceConfig.IsJobDisabled("apply_safe_outputs") {
		appliedRunURLValue = "${{ inputs.run_url }}"
		appliedRunURLDescription = "The run URL that safe outputs were applied from (workflow_call falls back to inputs.run_url when apply_safe_outputs is disabled; other triggers leave this empty)"
	}
	yaml.WriteString(maintenanceWorkflowEntrypointsPrefix)
	yaml.WriteString("        description: '" + appliedRunURLDescription + "'\n")
	yaml.WriteString("        value: " + appliedRunURLValue + "\n\npermissions: {}\n\njobs:\n")
}

func writeMaintenanceCloseExpiredJobs(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	if opts.maintenanceConfig.IsJobDisabled("close-expired-entities") {
		return
	}
	writeMaintenanceCloseExpiredJob(yaml, opts, setupActionRef, "close-expired-discussions", "discussions: write", "Close expired discussions", "close_expired_discussions")
	writeMaintenanceCloseExpiredJob(yaml, opts, setupActionRef, "close-expired-issues", "issues: write", "Close expired issues", "close_expired_issues")
	writeMaintenanceCloseExpiredJob(yaml, opts, setupActionRef, "close-expired-pull-requests", "pull-requests: write", "Close expired pull requests", "close_expired_pull_requests")
}

func writeMaintenanceCloseExpiredJob(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef, jobName, permissionLine, stepName, scriptName string) {
	yaml.WriteString(`  ` + jobName + `:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduleOnly()) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      ` + permissionLine + `
    steps:
`)
	if opts.actionMode == ActionModeDev || opts.actionMode == ActionModeScript {
		maintenanceWorkflowYAMLLog.Printf("Adding checkout step for %s (actionMode=%s)", jobName, opts.actionMode)
		writeMaintenanceActionsCheckoutStep(yaml)
	}
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceScriptStep(yaml, opts.resolver, stepName, "", "", scriptName)
}

func writeMaintenanceCleanupCacheMemoryJob(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	cleanupCacheCondition := buildNotForkAndScheduleOnlyOrOperation("clean_cache_memories")
	yaml.WriteString(`
  cleanup-cache-memory:
    if: ${{ ` + RenderCondition(cleanupCacheCondition) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      actions: write
    steps:
`)
	if opts.actionMode == ActionModeDev || opts.actionMode == ActionModeScript {
		writeMaintenanceActionsCheckoutStep(yaml)
	}
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceScriptStep(yaml, opts.resolver, "Cleanup outdated cache-memory entries", "", "", "cleanup_cache_memory")
}

func writeMaintenanceRunOperationJob(yaml *strings.Builder, ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	runOperationCondition := buildRunOperationCondition("safe_outputs", "create_labels", "activity_report", "close_agentic_workflows_issues", "clean_cache_memories", "update_pull_request_branches", "validate", "forecast")
	yaml.WriteString(`
  run_operation:
    if: ${{ ` + RenderCondition(runOperationCondition) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      actions: write
      contents: write
      pull-requests: write
    outputs:
      operation: ${{ steps.record.outputs.operation }}
    steps:
`)
	writeMaintenanceRepositoryCheckoutStep(yaml)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "")
	yaml.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	yaml.WriteString(`      - name: Run operation
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_AW_OPERATION: ${{ inputs.operation }}
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(opts.actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/run_operation_update_upgrade.cjs');
            await main();

      - name: Record outputs
        id: record
        env:
          GH_AW_OPERATION: ${{ inputs.operation }}
        run: echo "operation=$GH_AW_OPERATION" >> "$GITHUB_OUTPUT"
`)
}

func writeMaintenanceUpdatePullRequestBranchesJob(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	yaml.WriteString(`
  update_pull_request_branches:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("update_pull_request_branches")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      contents: write
      pull-requests: write
    steps:
`)
	if opts.actionMode == ActionModeDev || opts.actionMode == ActionModeScript {
		writeMaintenanceActionsCheckoutStep(yaml)
	}
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "")
	yaml.WriteString(`      - name: Update pull request branches
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/update_pull_request_branches.cjs');
            await main();
`)
}

func writeMaintenanceApplySafeOutputsJob(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	yaml.WriteString(`
  apply_safe_outputs:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("safe_outputs")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      actions: read
      contents: write
      discussions: write
      issues: write
      pull-requests: write
    outputs:
      run_url: ${{ steps.record.outputs.run_url }}
    steps:
`)
	writeMaintenanceActionsCheckoutStep(yaml)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "")
	yaml.WriteString(`      - name: Apply Safe Outputs
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_AW_RUN_URL: ${{ inputs.run_url }}
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/apply_safe_outputs_replay.cjs');
            await main();

      - name: Record outputs
        id: record
        env:
          GH_AW_RUN_URL: ${{ inputs.run_url }}
        run: echo "run_url=$GH_AW_RUN_URL" >> "$GITHUB_OUTPUT"
`)
}

func writeMaintenanceCreateLabelsJob(yaml *strings.Builder, ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	yaml.WriteString(`
  create_labels:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("create_labels")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      contents: read
      issues: write
    steps:
`)
	writeMaintenanceRepositoryCheckoutStep(yaml)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "")
	yaml.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	writeMaintenanceScriptStep(yaml, opts.resolver, "Create missing labels", "          GH_AW_CMD_PREFIX: "+getCLICmdPrefix(opts.actionMode)+"\n", "", "create_labels")
}

func writeMaintenanceActivityReportJob(yaml *strings.Builder, ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	yaml.WriteString(`
  activity_report:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("activity_report")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    timeout-minutes: 120
    permissions:
      actions: read
      contents: read
      issues: write
    steps:
`)
	writeMaintenanceRepositoryCheckoutStep(yaml)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "")
	yaml.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	writeMaintenanceActivityReportCacheSteps(yaml, opts)
	writeMaintenanceActivityReportIssueStep(yaml, opts.resolver)
}

func writeMaintenanceActivityReportCacheSteps(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions) {
	yaml.WriteString(`      - name: Restore activity report logs cache
        id: activity_report_logs_cache
        uses: ` + getActionPin("actions/cache/restore") + `
        with:
          path: ./.cache/gh-aw/activity-report-logs
          key: ${{ runner.os }}-activity-report-logs-${{ github.repository }}-${{ github.ref_name }}-${{ github.run_id }}
          restore-keys: |
            ${{ runner.os }}-activity-report-logs-${{ github.repository }}-
            ${{ runner.os }}-activity-report-logs-
`)
	yaml.WriteString(`      - name: Download activity report logs
        timeout-minutes: 20
        shell: bash
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(opts.actionMode) + `
        run: |
          ${GH_AW_CMD_PREFIX} logs \
            --repo "$GITHUB_REPOSITORY" \
            --start-date -1w \
            --count 500 \
            --output ./.cache/gh-aw/activity-report-logs \
            --format markdown \
            --report-file ./.cache/gh-aw/activity-report-logs/report.md

      - name: Save activity report logs cache
        if: ${{ always() }}
        uses: ` + getActionPin("actions/cache/save") + `
        with:
          path: ./.cache/gh-aw/activity-report-logs
          key: ${{ steps.activity_report_logs_cache.outputs.cache-primary-key }}

`)
}

func writeMaintenanceActivityReportIssueStep(yaml *strings.Builder, resolver SHAResolver) {
	yaml.WriteString(`      - name: Generate activity report issue
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const fs = require('node:fs');
            const reportPath = './.cache/gh-aw/activity-report-logs/report.md';
            if (!fs.existsSync(reportPath)) {
              core.warning('Activity report markdown not found at ' + reportPath + '; skipping issue creation.');
              return;
            }
            let reportBody = '';
            try {
              reportBody = fs.readFileSync(reportPath, 'utf8').trim();
            } catch (error) {
              core.warning('Failed to read activity report markdown at ' + reportPath + ': ' + error.message);
              return;
            }
            if (!reportBody) {
              core.warning('Activity report markdown is empty at ' + reportPath + '; skipping issue creation.');
              return;
            }
            const repoSlug = context.repo.owner + '/' + context.repo.repo;
            const body = [
              '### Agentic workflow activity report',
              '',
              'Repository: ' + repoSlug,
              'Generated at: ' + new Date().toISOString(),
              '',
              reportBody,
            ].join('\n');
            const createdIssue = await github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: '[aw] agentic status report',
              body,
              labels: ['agentic-workflows'],
            });
            core.info('Created issue #' + createdIssue.data.number + ': ' + createdIssue.data.html_url);
`)
}

func writeMaintenanceForecastReportJob(yaml *strings.Builder, ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	yaml.WriteString(`
  forecast_report:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("forecast")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    timeout-minutes: 60
    permissions:
      actions: read
      contents: read
      issues: write
    steps:
`)
	writeMaintenanceRepositoryCheckoutStep(yaml)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "")
	yaml.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	writeMaintenanceForecastGenerateSteps(yaml, opts)
	writeMaintenanceForecastIssueStep(yaml, opts.resolver)
}

func writeMaintenanceForecastGenerateSteps(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions) {
	yaml.WriteString(`      - name: Restore forecast report logs cache
        id: forecast_report_logs_cache
        uses: ` + getActionPin("actions/cache/restore") + `
        with:
          path: ./.github/aw/logs
          key: ${{ runner.os }}-forecast-report-logs-${{ github.repository }}-${{ github.ref_name }}-${{ github.run_id }}
          restore-keys: |
            ${{ runner.os }}-forecast-report-logs-${{ github.repository }}-
            ${{ runner.os }}-forecast-report-logs-

      - name: Generate forecast report
        id: generate_forecast_report
        timeout-minutes: 30
        shell: bash
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          DEBUG: "*"
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(opts.actionMode) + `
        run: |
          mkdir -p ./.cache/gh-aw/forecast
          set +e
          ${GH_AW_CMD_PREFIX} forecast --repo "$GITHUB_REPOSITORY" --timeout 30 --verbose --json > ./.cache/gh-aw/forecast/report.json
          forecast_exit_code=$?
          set -e
          if [ "${forecast_exit_code}" -eq 124 ]; then
            echo '{"outcome":"timeout","message":"Forecast computation timed out after 30 minutes."}' > ./.cache/gh-aw/forecast/error.json
            echo "::error::Forecast computation timed out after 30 minutes."
            exit 1
          fi
          if [ "${forecast_exit_code}" -ne 0 ]; then
            echo '{"outcome":"error","message":"Forecast computation failed before producing a report."}' > ./.cache/gh-aw/forecast/error.json
            echo "::error::Forecast computation failed with exit code ${forecast_exit_code}."
            exit 1
          fi

      - name: Debug forecast logs folder
        if: ${{ always() }}
        shell: bash
        run: |
          if [ ! -d ./.github/aw/logs ]; then
            echo "Logs directory not found: ./.github/aw/logs"
            exit 0
          fi
          echo "Files under ./.github/aw/logs:"
          find ./.github/aw/logs -type f | sort

`)
	writeMaintenanceForecastSaveCacheStep(yaml)
}

func writeMaintenanceForecastSaveCacheStep(yaml *strings.Builder) {
	yaml.WriteString(`      - name: Save forecast report logs cache
        if: ${{ always() }}
        uses: ` + getActionPin("actions/cache/save") + `
        with:
          path: ./.github/aw/logs
          key: ${{ runner.os }}-forecast-report-logs-${{ github.repository }}-${{ github.ref_name }}-${{ github.run_id }}

`)
}

func writeMaintenanceForecastIssueStep(yaml *strings.Builder, resolver SHAResolver) {
	yaml.WriteString(`      - name: Generate forecast issue
        if: ${{ always() }}
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        env:
          FORECAST_STEP_OUTCOME: ${{ steps.generate_forecast_report.outcome }}
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/create_forecast_issue.cjs');
            await main();
`)
}

func writeMaintenanceCloseAgenticIssuesJob(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	yaml.WriteString(`
  close_agentic_workflows_issues:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("close_agentic_workflows_issues")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      issues: write
    steps:
`)
	if opts.actionMode == ActionModeDev || opts.actionMode == ActionModeScript {
		writeMaintenanceActionsCheckoutStep(yaml)
	}
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "")
	writeMaintenanceScriptStep(yaml, opts.resolver, "Close no-repro agentic-workflows issues", "", "", "close_agentic_workflows_issues")
}

func writeMaintenanceValidateWorkflowsJob(yaml *strings.Builder, ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	formattedRunsOn := FormatRunsOn(opts.configuredRunsOn, "ubuntu-latest")
	yaml.WriteString(`
  validate_workflows:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("validate")) + ` }}
    runs-on: ` + formattedRunsOn + `
    permissions:
      contents: read
      issues: write
    steps:
`)
	writeMaintenanceRepositoryCheckoutStep(yaml)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "")
	yaml.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	writeMaintenanceScriptStep(yaml, opts.resolver, "Validate workflows and file issue on findings", "          GH_AW_CMD_PREFIX: "+getCLICmdPrefix(opts.actionMode)+"\n", "", "run_validate_workflows")
}

func writeMaintenanceLabelJobs(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string, labelDisableJobEnabled, labelApplySafeOutputsJobEnabled bool) {
	if !labelDisableJobEnabled && !labelApplySafeOutputsJobEnabled {
		return
	}
	maintenanceWorkflowYAMLLog.Print("Adding label-triggered jobs")
	if labelDisableJobEnabled {
		writeMaintenanceLabelDisableJob(yaml, opts, setupActionRef)
	}
	if labelApplySafeOutputsJobEnabled {
		writeMaintenanceLabelApplySafeOutputsJob(yaml, opts, setupActionRef)
	}
}

func writeMaintenanceLabelDisableJob(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	disableLabelCondition := buildLabeledDisableCondition()
	yaml.WriteString(`
  label_disable_agentic_workflow:
    if: ${{ ` + RenderCondition(disableLabelCondition) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      actions: write
      contents: read
      issues: write
    steps:
`)
	writeMaintenanceActionsCheckoutStep(yaml)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "check_permissions")
	writeMaintenanceScriptStep(yaml, opts.resolver, "Disable agentic workflow", "", "        if: ${{ steps.check_permissions.outcome == 'success' }}\n", "disable_agentic_workflow")
}

func writeMaintenanceLabelApplySafeOutputsJob(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	applySafeOutputsCondition := buildLabeledApplySafeOutputsCondition()
	yaml.WriteString(`
  label_apply_safe_outputs:
    if: ${{ ` + RenderCondition(applySafeOutputsCondition) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      actions: read
      contents: write
      discussions: write
      issues: write
      pull-requests: write
    steps:
`)
	writeMaintenanceActionsCheckoutStep(yaml)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceAdminCheckStep(yaml, opts.resolver, "check_permissions")
	writeMaintenanceScriptStep(yaml, opts.resolver, "Apply safe outputs from referenced run", "          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n", "        if: ${{ steps.check_permissions.outcome == 'success' }}\n", "label_apply_safe_outputs")
}

func writeMaintenanceDevJobs(yaml *strings.Builder, ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	maintenanceWorkflowYAMLLog.Printf("Adding dev-only jobs: compile-workflows and secret-validation")
	writeMaintenanceCompileWorkflowsJob(yaml, ctx, opts, setupActionRef)
	writeMaintenanceSecretValidationJob(yaml, opts, setupActionRef)
}

func writeMaintenanceCompileWorkflowsJob(yaml *strings.Builder, ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	yaml.WriteString(`
  compile-workflows:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduled()) + ` }}
    runs-on: ` + opts.runsOnValue + `
    concurrency:
      group: ${{ github.workflow }}-compile-workflows-${{ github.repository }}
      cancel-in-progress: true
    permissions:
      contents: read
      issues: write
    steps:
`)
	writeMaintenanceDevRepositoryCheckoutStep(yaml)
	yaml.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	writeMaintenanceCompileWorkflowsSteps(yaml, opts, setupActionRef)
}

func writeMaintenanceCompileWorkflowsSteps(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	yaml.WriteString(`      - name: Pre-compile validation
        run: |
          ` + getCLICmdPrefix(opts.actionMode) + ` compile --validate --no-emit --verbose
          echo "✓ Pre-compile validation passed"

      - name: Compile workflows
        run: |
          ` + getCLICmdPrefix(opts.actionMode) + ` compile --validate --verbose
          echo "✓ All workflows compiled successfully"

`)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	yaml.WriteString(`      - name: Check for out-of-sync workflows and create issue or pull request if needed
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
`)
	if opts.compileGitHubToken != "" {
		yaml.WriteString(`        env:
          GH_AW_MAINTENANCE_GITHUB_TOKEN: ` + opts.compileGitHubToken + `
`)
	}
	yaml.WriteString(`        with:
`)
	if opts.compileGitHubToken != "" {
		yaml.WriteString(`          github-token: ${{ env.GH_AW_MAINTENANCE_GITHUB_TOKEN }}
`)
	}
	yaml.WriteString(`          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_workflow_recompile_needed.cjs');
            await main();
`)
}

func writeMaintenanceSecretValidationJob(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) {
	yaml.WriteString(`
  secret-validation:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduleOnly()) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      contents: read
    steps:
`)
	writeMaintenanceActionsCheckoutStep(yaml)
	yaml.WriteString(`      - name: Setup Node.js
        uses: actions/setup-node@39370e3970a6d050c480ffad4ff0ed4d3fdee5af # v4.1.0
        with:
          node-version: '22'

`)
	writeMaintenanceSetupScriptsStep(yaml, setupActionRef)
	writeMaintenanceValidateSecretsStep(yaml, opts)
}

func writeMaintenanceValidateSecretsStep(yaml *strings.Builder, opts buildMaintenanceWorkflowYAMLOptions) {
	copilotOrgBillingLine := ""
	if opts.copilotOrgBilling {
		maintenanceWorkflowYAMLLog.Print("Copilot org billing mode detected: adding GH_AW_COPILOT_ORG_BILLING=true to secret-validation step")
		copilotOrgBillingLine = `          GH_AW_COPILOT_ORG_BILLING: "true"
`
	}
	yaml.WriteString(`      - name: Validate Secrets
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          # GitHub tokens
          GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
          GH_AW_GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}
          GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
          GH_AW_COPILOT_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN }}
` + copilotOrgBillingLine + `          # AI Engine API keys
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          BRAVE_API_KEY: ${{ secrets.BRAVE_API_KEY }}
          # Integration tokens
          NOTION_API_TOKEN: ${{ secrets.NOTION_API_TOKEN }}
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/validate_secrets.cjs');
            await main();

      - name: Upload secret validation report
        if: always()
        uses: ` + getActionPin("actions/upload-artifact") + `
        with:
          name: secret-validation-report
          path: secret-validation-report.md
          retention-days: 30
          if-no-files-found: warn
`)
}

func writeMaintenanceActionsCheckoutStep(yaml *strings.Builder) {
	yaml.WriteString(`      - name: Checkout actions folder
        uses: ` + getActionPin("actions/checkout") + `
        with:
          sparse-checkout: |
            actions
          clean: false
          persist-credentials: false

`)
}

func writeMaintenanceRepositoryCheckoutStep(yaml *strings.Builder) {
	yaml.WriteString(`      - name: Checkout repository
        uses: ` + getActionPin("actions/checkout") + `
        with:
          persist-credentials: false

`)
}

func writeMaintenanceDevRepositoryCheckoutStep(yaml *strings.Builder) {
	yaml.WriteString(`      - name: Checkout repository
        uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
        with:
          persist-credentials: false

`)
}

func writeMaintenanceSetupScriptsStep(yaml *strings.Builder, setupActionRef string) {
	yaml.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

`)
}

func writeMaintenanceAdminCheckStep(yaml *strings.Builder, resolver SHAResolver, id string) {
	yaml.WriteString(`      - name: Check admin/maintainer permissions
`)
	if id != "" {
		yaml.WriteString(`        id: ` + id + `
`)
	}
	yaml.WriteString(`        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)
}

func writeMaintenanceScriptStep(yaml *strings.Builder, resolver SHAResolver, name, envLines, ifLine, scriptName string) {
	yaml.WriteString(`      - name: ` + name + `
`)
	if ifLine != "" {
		yaml.WriteString(ifLine)
	}
	yaml.WriteString(`        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
`)
	if envLines != "" {
		yaml.WriteString("        env:\n" + envLines)
	}
	yaml.WriteString(`        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/` + scriptName + `.cjs');
            await main();
`)
}
