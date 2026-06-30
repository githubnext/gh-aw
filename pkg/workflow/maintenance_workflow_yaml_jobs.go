package workflow

import (
	"context"
	"strconv"
	"strings"
)

// maintenanceWorkflowDispatchBlock is the static YAML for the workflow_dispatch and workflow_call
// trigger blocks (no Go-level string interpolation needed).
const maintenanceWorkflowDispatchBlock = `  workflow_dispatch:
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
        description: 'The run URL that safe outputs were applied from'
        value: ${{ jobs.apply_safe_outputs.outputs.run_url }}

permissions: {}

jobs:
`

// buildMaintenanceWorkflowPreamble returns the workflow header, name, and triggers section.
func buildMaintenanceWorkflowPreamble(opts buildMaintenanceWorkflowYAMLOptions) string {
	customInstructions := `This file defines the generated agentic maintenance workflow for this repository.
It runs scheduled cleanup for expiring safe outputs and supports manual maintenance operations.

This workflow is generated automatically when workflows use expiring safe outputs
or when repository maintenance features are enabled in .github/workflows/aw.json.

To disable maintenance workflow generation, set in .github/workflows/aw.json:
  {"maintenance": false}

Agentic maintenance docs:
  https://github.github.com/gh-aw/reference/ephemerals/#manual-maintenance-operations`

	header := GenerateWorkflowHeader("", "pkg/workflow/maintenance_workflow.go", customInstructions)
	var b strings.Builder
	b.WriteString(header)
	b.WriteString("name: Agentic Maintenance\n\non:\n")
	b.WriteString(`  schedule:
    - cron: "` + opts.cronSchedule + `"  # ` + opts.scheduleDesc + ` (based on minimum expires: ` + strconv.Itoa(opts.minExpiresDays) + ` days)
`)
	if opts.actionMode == ActionModeDev {
		maintenanceWorkflowYAMLLog.Printf("Adding dev-mode push trigger for branch %q", opts.defaultBranch)
		b.WriteString(`  push:
    branches:
      - ` + opts.defaultBranch + `
    paths:
      - '.github/workflows/*.md'
`)
	}
	if !opts.disableLabelTrigger {
		maintenanceWorkflowYAMLLog.Print("Adding issues:labeled trigger for label-triggered maintenance jobs")
		b.WriteString("  issues:\n    types: [labeled]\n")
	}
	b.WriteString(maintenanceWorkflowDispatchBlock)
	return b.String()
}

// buildOptionalDevCheckout emits the checkout step for dev/script action modes.
func buildOptionalDevCheckout(opts buildMaintenanceWorkflowYAMLOptions) string {
	if opts.actionMode != ActionModeDev && opts.actionMode != ActionModeScript {
		return ""
	}
	return "      - name: Checkout actions folder\n" +
		"        uses: " + getActionPin("actions/checkout") + "\n" +
		"        with:\n" +
		"          sparse-checkout: |\n" +
		"            actions\n" +
		"          persist-credentials: false\n\n"
}

// buildCloseExpiredEntitiesJob returns the close-expired-entities job YAML.
func buildCloseExpiredEntitiesJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`  close-expired-entities:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduleOnly()) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      discussions: write
      issues: write
      pull-requests: write
    steps:
`)
	b.WriteString(buildOptionalDevCheckout(opts))
	b.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Close expired discussions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_discussions.cjs');
            await main();

      - name: Close expired issues
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_issues.cjs');
            await main();

      - name: Close expired pull requests
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_pull_requests.cjs');
            await main();
`)
	return b.String()
}

// buildCleanupCacheMemoryJob returns the cleanup-cache-memory job YAML.
func buildCleanupCacheMemoryJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	cleanupCacheCondition := buildNotForkAndScheduleOnlyOrOperation("clean_cache_memories")
	b.WriteString(`
  cleanup-cache-memory:
    if: ${{ ` + RenderCondition(cleanupCacheCondition) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      actions: write
    steps:
`)
	b.WriteString(buildOptionalDevCheckout(opts))
	b.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Cleanup outdated cache-memory entries
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/cleanup_cache_memory.cjs');
            await main();
`)
	return b.String()
}

// buildRunOperationJob returns the run_operation job YAML.
func buildRunOperationJob(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	runOperationCondition := buildRunOperationCondition("safe_outputs", "create_labels", "activity_report", "close_agentic_workflows_issues", "clean_cache_memories", "update_pull_request_branches", "validate", "forecast")
	var b strings.Builder
	b.WriteString(`
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
      - name: Checkout repository
        uses: ` + getActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)
	b.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	b.WriteString(`      - name: Run operation
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
	return b.String()
}

// buildUpdatePRBranchesJob returns the update_pull_request_branches job YAML.
func buildUpdatePRBranchesJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
  update_pull_request_branches:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("update_pull_request_branches")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      contents: write
      pull-requests: write
    steps:
`)
	b.WriteString(buildOptionalDevCheckout(opts))
	b.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Update pull request branches
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
	return b.String()
}

// buildApplySafeOutputsJob returns the apply_safe_outputs job YAML.
func buildApplySafeOutputsJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
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
      - name: Checkout actions folder
        uses: ` + getActionPin("actions/checkout") + `
        with:
          sparse-checkout: |
            actions
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Apply Safe Outputs
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
	return b.String()
}

// buildCreateLabelsJob returns the create_labels job YAML.
func buildCreateLabelsJob(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
  create_labels:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("create_labels")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      contents: read
      issues: write
    steps:
      - name: Checkout repository
        uses: ` + getActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)
	b.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	b.WriteString(`      - name: Create missing labels
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(opts.actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/create_labels.cjs');
            await main();
`)
	return b.String()
}

// buildActivityReportJobHeader returns the activity_report job header and setup steps.
func buildActivityReportJobHeader(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	return `
  activity_report:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("activity_report")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    timeout-minutes: 120
    permissions:
      actions: read
      contents: read
      issues: write
    steps:
      - name: Checkout repository
        uses: ` + getActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`
}

// buildActivityReportJobSteps returns the activity_report data steps after setup.
func buildActivityReportJobSteps(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions) string {
	var b strings.Builder
	b.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	b.WriteString(`      - name: Restore activity report logs cache
        id: activity_report_logs_cache
        uses: ` + getActionPin("actions/cache/restore") + `
        with:
          path: ./.cache/gh-aw/activity-report-logs
          key: ${{ runner.os }}-activity-report-logs-${{ github.repository }}-${{ github.ref_name }}-${{ github.run_id }}
          restore-keys: |
            ${{ runner.os }}-activity-report-logs-${{ github.repository }}-
            ${{ runner.os }}-activity-report-logs-
      - name: Download activity report logs
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
	b.WriteString(buildActivityReportGenerateIssueStep(opts))
	return b.String()
}

// buildActivityReportGenerateIssueStep returns the generate activity report issue step.
func buildActivityReportGenerateIssueStep(opts buildMaintenanceWorkflowYAMLOptions) string {
	return `      - name: Generate activity report issue
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
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
`
}

// buildActivityReportJob returns the full activity_report job YAML.
func buildActivityReportJob(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	return buildActivityReportJobHeader(opts, setupActionRef) + buildActivityReportJobSteps(ctx, opts)
}

// buildForecastReportJobHeader returns the forecast_report job header and setup steps.
func buildForecastReportJobHeader(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	return `
  forecast_report:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("forecast")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    timeout-minutes: 60
    permissions:
      actions: read
      contents: read
      issues: write
    steps:
      - name: Checkout repository
        uses: ` + getActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`
}

// buildForecastReportJobSteps returns the forecast data generation and issue-creation steps.
func buildForecastReportJobSteps(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions) string {
	var b strings.Builder
	b.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	b.WriteString(`      - name: Restore forecast report logs cache
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

`)
	b.WriteString(buildForecastReportFinalSteps(opts))
	return b.String()
}

// buildForecastReportFinalSteps returns the debug, cache-save, and issue steps for the forecast job.
func buildForecastReportFinalSteps(opts buildMaintenanceWorkflowYAMLOptions) string {
	return `      - name: Debug forecast logs folder
        if: ${{ always() }}
        shell: bash
        run: |
          if [ ! -d ./.github/aw/logs ]; then
            echo "Logs directory not found: ./.github/aw/logs"
            exit 0
          fi
          echo "Files under ./.github/aw/logs:"
          find ./.github/aw/logs -type f | sort

      - name: Save forecast report logs cache
        if: ${{ always() }}
        uses: ` + getActionPin("actions/cache/save") + `
        with:
          path: ./.github/aw/logs
          key: ${{ runner.os }}-forecast-report-logs-${{ github.repository }}-${{ github.ref_name }}-${{ github.run_id }}

      - name: Generate forecast issue
        if: ${{ always() }}
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          FORECAST_STEP_OUTCOME: ${{ steps.generate_forecast_report.outcome }}
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/create_forecast_issue.cjs');
            await main();
`
}

// buildForecastReportJob returns the full forecast_report job YAML.
func buildForecastReportJob(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	return buildForecastReportJobHeader(opts, setupActionRef) + buildForecastReportJobSteps(ctx, opts)
}

// buildCloseAgenticWorkflowsIssuesJob returns the close_agentic_workflows_issues job YAML.
func buildCloseAgenticWorkflowsIssuesJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
  close_agentic_workflows_issues:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("close_agentic_workflows_issues")) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      issues: write
    steps:
`)
	b.WriteString(buildOptionalDevCheckout(opts))
	b.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Close no-repro agentic-workflows issues
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_agentic_workflows_issues.cjs');
            await main();
`)
	return b.String()
}

// buildValidateWorkflowsJob returns the validate_workflows job YAML.
func buildValidateWorkflowsJob(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	formattedRunsOn := FormatRunsOn(opts.configuredRunsOn, "ubuntu-latest")
	var b strings.Builder
	b.WriteString(`
  validate_workflows:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("validate")) + ` }}
    runs-on: ` + formattedRunsOn + `
    permissions:
      contents: read
      issues: write
    steps:
      - name: Checkout repository
        uses: ` + getActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)
	b.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	b.WriteString(`      - name: Validate workflows and file issue on findings
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(opts.actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/run_validate_workflows.cjs');
            await main();
`)
	return b.String()
}

// buildLabelDisableAgenticWorkflowJob returns the label_disable_agentic_workflow job YAML.
func buildLabelDisableAgenticWorkflowJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	disableLabelCondition := buildLabeledDisableCondition()
	return `
  label_disable_agentic_workflow:
    if: ${{ ` + RenderCondition(disableLabelCondition) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      actions: write
      contents: read
      issues: write
    steps:
      - name: Checkout actions folder
        uses: ` + getActionPin("actions/checkout") + `
        with:
          sparse-checkout: |
            actions
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        id: check_permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Disable agentic workflow
        if: ${{ steps.check_permissions.outcome == 'success' }}
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/disable_agentic_workflow.cjs');
            await main();
`
}

// buildLabelApplySafeOutputsJob returns the label_apply_safe_outputs job YAML.
func buildLabelApplySafeOutputsJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	applySafeOutputsCondition := buildLabeledApplySafeOutputsCondition()
	return `
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
      - name: Checkout actions folder
        uses: ` + getActionPin("actions/checkout") + `
        with:
          sparse-checkout: |
            actions
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        id: check_permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Apply safe outputs from referenced run
        if: ${{ steps.check_permissions.outcome == 'success' }}
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/label_apply_safe_outputs.cjs');
            await main();
`
}

// buildDevCompileWorkflowsJob returns the compile-workflows dev-only job YAML.
func buildDevCompileWorkflowsJob(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
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
      - name: Checkout repository
        uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
        with:
          persist-credentials: false

`)
	b.WriteString(generateInstallCLISteps(ctx, opts.actionMode, opts.version, opts.actionTag, opts.resolver))
	b.WriteString(`      - name: Pre-compile validation
        run: |
          ` + getCLICmdPrefix(opts.actionMode) + ` compile --validate --no-emit --verbose
          echo "✓ Pre-compile validation passed"

      - name: Compile workflows
        run: |
          ` + getCLICmdPrefix(opts.actionMode) + ` compile --validate --verbose
          echo "✓ All workflows compiled successfully"

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check for out-of-sync workflows and create issue or pull request if needed
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
`)
	if opts.compileGitHubToken != "" {
		b.WriteString(`        env:
          GH_AW_MAINTENANCE_GITHUB_TOKEN: ` + opts.compileGitHubToken + `
`)
	}
	b.WriteString("        with:\n")
	if opts.compileGitHubToken != "" {
		b.WriteString(`          github-token: ${{ env.GH_AW_MAINTENANCE_GITHUB_TOKEN }}
`)
	}
	b.WriteString(`          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_workflow_recompile_needed.cjs');
            await main();
`)
	return b.String()
}

// buildDevSecretValidationJob returns the secret-validation dev-only job YAML.
func buildDevSecretValidationJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
  secret-validation:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduleOnly()) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      contents: read
    steps:
      - name: Checkout actions folder
        uses: ` + getActionPin("actions/checkout") + `
        with:
          sparse-checkout: |
            actions
          persist-credentials: false

      - name: Setup Node.js
        uses: actions/setup-node@39370e3970a6d050c480ffad4ff0ed4d3fdee5af # v4.1.0
        with:
          node-version: '22'

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

`)
	copilotOrgBillingLine := ""
	if opts.copilotOrgBilling {
		maintenanceWorkflowYAMLLog.Print("Copilot org billing mode detected: adding GH_AW_COPILOT_ORG_BILLING=true to secret-validation step")
		copilotOrgBillingLine = "          GH_AW_COPILOT_ORG_BILLING: \"true\"\n"
	}
	b.WriteString(`      - name: Validate Secrets
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
	return b.String()
}
