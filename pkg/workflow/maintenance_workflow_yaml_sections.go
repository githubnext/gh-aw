package workflow

import (
	"context"
	"fmt"
	"strings"
)

type maintenanceWorkflowYAMLBuilder struct {
	ctx                             context.Context
	opts                            buildMaintenanceWorkflowYAMLOptions
	yaml                            *strings.Builder
	setupActionRef                  string
	labelDisableJobEnabled          bool
	labelApplySafeOutputsJobEnabled bool
}

type maintenanceCloseExpiredJob struct {
	jobName        string
	permissionLine string
	stepName       string
	scriptName     string
}

const maintenanceTriggersTemplate = `name: Agentic Maintenance

on:
  schedule:
    - cron: "%s"  # %s (based on minimum expires: %d days)
`

const maintenanceDevPushTriggerTemplate = `  push:
    branches:
      - %s
    paths:
      - '.github/workflows/*.md'
`

const maintenanceIssuesTrigger = `  issues:
    types: [labeled]
`

const maintenanceDispatchSectionTemplate = `  workflow_dispatch:
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
        description: '%s'
        value: %s

permissions: {}

jobs:
`

const maintenanceActionsCheckoutTemplate = `      - name: Checkout actions folder
        uses: %s
        with:
          sparse-checkout: |
            actions
          clean: false
          persist-credentials: false

`

const maintenanceRepoCheckoutTemplate = `      - name: Checkout repository
        uses: %s
        with:
          persist-credentials: false

`

const maintenanceSetupScriptsTemplate = `      - name: Setup Scripts
        uses: %s
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

`

const maintenanceAdminPermissionsTemplate = `      - name: Check admin/maintainer permissions
%s        uses: %s
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`

const maintenanceCloseExpiredJobTemplate = `  %s:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      %s
    steps:
%s%s      - name: %s
        uses: %s
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/%s.cjs');
            await main();
`

const maintenanceCleanupCacheJobTemplate = `
  cleanup-cache-memory:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      actions: write
    steps:
%s%s      - name: Cleanup outdated cache-memory entries
        uses: %s
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/cleanup_cache_memory.cjs');
            await main();
`

const maintenanceRunOperationJobTemplate = `
  run_operation:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      actions: write
      contents: write
      pull-requests: write
    outputs:
      operation: ${{ steps.record.outputs.operation }}
    steps:
%s%s%s%s      - name: Run operation
        uses: %s
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_AW_OPERATION: ${{ inputs.operation }}
          GH_AW_CMD_PREFIX: %s
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
`

const maintenanceUpdatePRBranchesJobTemplate = `
  update_pull_request_branches:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      contents: write
      pull-requests: write
    steps:
%s%s%s      - name: Update pull request branches
        uses: %s
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/update_pull_request_branches.cjs');
            await main();
`

const maintenanceApplySafeOutputsJobTemplate = `
  apply_safe_outputs:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      actions: read
      contents: write
      discussions: write
      issues: write
      pull-requests: write
    outputs:
      run_url: ${{ steps.record.outputs.run_url }}
    steps:
%s%s%s      - name: Apply Safe Outputs
        uses: %s
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
`

const maintenanceCreateLabelsJobTemplate = `
  create_labels:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      contents: read
      issues: write
    steps:
%s%s%s%s      - name: Create missing labels
        uses: %s
        env:
          GH_AW_CMD_PREFIX: %s
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/create_labels.cjs');
            await main();
`

const maintenanceActivityReportJobTemplate = `
  activity_report:
    if: ${{ %s }}
    runs-on: %s
    timeout-minutes: 120
    permissions:
      actions: read
      contents: read
      issues: write
    steps:
%s%s%s%s      - name: Restore activity report logs cache
        id: activity_report_logs_cache
        uses: %s
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
          GH_AW_CMD_PREFIX: %s
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
        uses: %s
        with:
          path: ./.cache/gh-aw/activity-report-logs
          key: ${{ steps.activity_report_logs_cache.outputs.cache-primary-key }}

      - name: Generate activity report issue
        uses: %s
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
%s`

const maintenanceActivityReportIssueScript = `            const fs = require('node:fs');
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

const maintenanceForecastReportJobTemplate = `
  forecast_report:
    if: ${{ %s }}
    runs-on: %s
    timeout-minutes: 60
    permissions:
      actions: read
      contents: read
      issues: write
    steps:
%s%s%s%s      - name: Restore forecast report logs cache
        id: forecast_report_logs_cache
        uses: %s
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
          GH_AW_CMD_PREFIX: %s
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

      - name: Save forecast report logs cache
        if: ${{ always() }}
        uses: %s
        with:
          path: ./.github/aw/logs
          key: ${{ runner.os }}-forecast-report-logs-${{ github.repository }}-${{ github.ref_name }}-${{ github.run_id }}

      - name: Generate forecast issue
        if: ${{ always() }}
        uses: %s
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

const maintenanceCloseAWIssuesJobTemplate = `
  close_agentic_workflows_issues:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      issues: write
    steps:
%s%s%s      - name: Close no-repro agentic-workflows issues
        uses: %s
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_agentic_workflows_issues.cjs');
            await main();
`

const maintenanceValidateWorkflowsJobTemplate = `
  validate_workflows:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      contents: read
      issues: write
    steps:
%s%s%s%s      - name: Validate workflows and file issue on findings
        uses: %s
        env:
          GH_AW_CMD_PREFIX: %s
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/run_validate_workflows.cjs');
            await main();
`

const maintenanceLabelDisableJobTemplate = `
  label_disable_agentic_workflow:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      actions: write
      contents: read
      issues: write
    steps:
%s%s      - name: Disable agentic workflow
        if: ${{ steps.check_permissions.outcome == 'success' }}
        uses: %s
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/disable_agentic_workflow.cjs');
            await main();
`

const maintenanceLabelApplySafeOutputsJobTemplate = `
  label_apply_safe_outputs:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      actions: read
      contents: write
      discussions: write
      issues: write
      pull-requests: write
    steps:
%s%s      - name: Apply safe outputs from referenced run
        if: ${{ steps.check_permissions.outcome == 'success' }}
        uses: %s
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

const maintenanceCompileWorkflowsJobTemplate = `
  compile-workflows:
    if: ${{ %s }}
    runs-on: %s
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

%s      - name: Pre-compile validation
        run: |
          %s compile --validate --no-emit --verbose
          echo "✓ Pre-compile validation passed"

      - name: Compile workflows
        run: |
          %s compile --validate --verbose
          echo "✓ All workflows compiled successfully"

%s      - name: Check for out-of-sync workflows and create issue or pull request if needed
        uses: %s
%s        with:
%s          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_workflow_recompile_needed.cjs');
            await main();
`

const maintenanceSecretValidationJobTemplate = `
  secret-validation:
    if: ${{ %s }}
    runs-on: %s
    permissions:
      contents: read
    steps:
%s      - name: Setup Node.js
        uses: actions/setup-node@39370e3970a6d050c480ffad4ff0ed4d3fdee5af # v4.1.0
        with:
          node-version: '22'

%s      - name: Validate Secrets
        uses: %s
        env:
          # GitHub tokens
          GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
          GH_AW_GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}
          GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
          GH_AW_COPILOT_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN }}
%s          # AI Engine API keys
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
        uses: %s
        with:
          name: secret-validation-report
          path: secret-validation-report.md
          retention-days: 30
          if-no-files-found: warn
`

func newMaintenanceWorkflowYAMLBuilder(ctx context.Context, opts buildMaintenanceWorkflowYAMLOptions, yaml *strings.Builder, setupActionRef string, labelDisableJobEnabled, labelApplySafeOutputsJobEnabled bool) *maintenanceWorkflowYAMLBuilder {
	return &maintenanceWorkflowYAMLBuilder{
		ctx:                             ctx,
		opts:                            opts,
		yaml:                            yaml,
		setupActionRef:                  setupActionRef,
		labelDisableJobEnabled:          labelDisableJobEnabled,
		labelApplySafeOutputsJobEnabled: labelApplySafeOutputsJobEnabled,
	}
}

func (b *maintenanceWorkflowYAMLBuilder) writeMaintenanceWorkflowTriggersSection() {
	fmt.Fprintf(b.yaml, maintenanceTriggersTemplate, b.opts.cronSchedule, b.opts.scheduleDesc, b.opts.minExpiresDays)
	if b.opts.actionMode == ActionModeDev {
		maintenanceWorkflowYAMLLog.Printf("Adding dev-mode push trigger for branch %q", b.opts.defaultBranch)
		fmt.Fprintf(b.yaml, maintenanceDevPushTriggerTemplate, b.opts.defaultBranch)
	}
	if b.labelDisableJobEnabled || b.labelApplySafeOutputsJobEnabled {
		maintenanceWorkflowYAMLLog.Print("Adding issues:labeled trigger for label-triggered maintenance jobs")
		b.yaml.WriteString(maintenanceIssuesTrigger)
	}
	description, value := b.resolveAppliedRunURLOutput()
	fmt.Fprintf(b.yaml, maintenanceDispatchSectionTemplate, description, value)
}

func (b *maintenanceWorkflowYAMLBuilder) writeCloseExpiredJobs() {
	if b.opts.maintenanceConfig.IsJobDisabled("close-expired-entities") {
		return
	}
	jobs := []maintenanceCloseExpiredJob{{"close-expired-discussions", "discussions: write", "Close expired discussions", "close_expired_discussions"}, {"close-expired-issues", "issues: write", "Close expired issues", "close_expired_issues"}, {"close-expired-pull-requests", "pull-requests: write", "Close expired pull requests", "close_expired_pull_requests"}}
	for _, job := range jobs {
		b.writeCloseExpiredJob(job)
	}
}

func (b *maintenanceWorkflowYAMLBuilder) writeCleanupCacheJob() {
	fmt.Fprintf(b.yaml, maintenanceCleanupCacheJobTemplate, RenderCondition(buildNotForkAndScheduleOnlyOrOperation("clean_cache_memories")), b.opts.runsOnValue, b.actionsFolderCheckoutBlock(), b.setupScriptsBlock(), b.githubScriptPin())
}

func (b *maintenanceWorkflowYAMLBuilder) writeRunOperationJob() {
	fmt.Fprintf(b.yaml, maintenanceRunOperationJobTemplate, RenderCondition(buildRunOperationCondition("safe_outputs", "create_labels", "activity_report", "close_agentic_workflows_issues", "clean_cache_memories", "update_pull_request_branches", "validate", "forecast")), b.opts.runsOnValue, b.repositoryCheckoutBlock(), b.setupScriptsBlock(), b.adminPermissionsBlock(""), generateInstallCLISteps(b.ctx, b.opts.actionMode, b.opts.version, b.opts.actionTag, b.opts.resolver), b.githubScriptPin(), getCLICmdPrefix(b.opts.actionMode))
}

func (b *maintenanceWorkflowYAMLBuilder) writeUpdatePRBranchesJob() {
	fmt.Fprintf(b.yaml, maintenanceUpdatePRBranchesJobTemplate, RenderCondition(buildDispatchOperationCondition("update_pull_request_branches")), b.opts.runsOnValue, b.actionsFolderCheckoutBlock(), b.setupScriptsBlock(), b.adminPermissionsBlock(""), b.githubScriptPin())
}

func (b *maintenanceWorkflowYAMLBuilder) writeApplySafeOutputsJob() {
	if b.opts.maintenanceConfig.IsJobDisabled("apply_safe_outputs") {
		return
	}
	fmt.Fprintf(b.yaml, maintenanceApplySafeOutputsJobTemplate, RenderCondition(buildDispatchOperationCondition("safe_outputs")), b.opts.runsOnValue, b.actionsCheckoutBlock(), b.setupScriptsBlock(), b.adminPermissionsBlock(""), b.githubScriptPin())
}

func (b *maintenanceWorkflowYAMLBuilder) writeCreateLabelsJob() {
	fmt.Fprintf(b.yaml, maintenanceCreateLabelsJobTemplate, RenderCondition(buildDispatchOperationCondition("create_labels")), b.opts.runsOnValue, b.repositoryCheckoutBlock(), b.setupScriptsBlock(), b.adminPermissionsBlock(""), generateInstallCLISteps(b.ctx, b.opts.actionMode, b.opts.version, b.opts.actionTag, b.opts.resolver), b.githubScriptPin(), getCLICmdPrefix(b.opts.actionMode))
}

func (b *maintenanceWorkflowYAMLBuilder) writeActivityReportJob() {
	fmt.Fprintf(b.yaml, maintenanceActivityReportJobTemplate, RenderCondition(buildDispatchOperationCondition("activity_report")), b.opts.runsOnValue, b.repositoryCheckoutBlock(), b.setupScriptsBlock(), b.adminPermissionsBlock(""), generateInstallCLISteps(b.ctx, b.opts.actionMode, b.opts.version, b.opts.actionTag, b.opts.resolver), getActionPin("actions/cache/restore"), getCLICmdPrefix(b.opts.actionMode), getActionPin("actions/cache/save"), b.githubScriptPin(), maintenanceActivityReportIssueScript)
}

func (b *maintenanceWorkflowYAMLBuilder) writeForecastReportJob() {
	fmt.Fprintf(b.yaml, maintenanceForecastReportJobTemplate, RenderCondition(buildDispatchOperationCondition("forecast")), b.opts.runsOnValue, b.repositoryCheckoutBlock(), b.setupScriptsBlock(), b.adminPermissionsBlock(""), generateInstallCLISteps(b.ctx, b.opts.actionMode, b.opts.version, b.opts.actionTag, b.opts.resolver), getActionPin("actions/cache/restore"), getCLICmdPrefix(b.opts.actionMode), getActionPin("actions/cache/save"), b.githubScriptPin())
}

func (b *maintenanceWorkflowYAMLBuilder) writeCloseAWIssuesJob() {
	fmt.Fprintf(b.yaml, maintenanceCloseAWIssuesJobTemplate, RenderCondition(buildDispatchOperationCondition("close_agentic_workflows_issues")), b.opts.runsOnValue, b.actionsFolderCheckoutBlock(), b.setupScriptsBlock(), b.adminPermissionsBlock(""), b.githubScriptPin())
}

func (b *maintenanceWorkflowYAMLBuilder) writeValidateWorkflowsJob() {
	fmt.Fprintf(b.yaml, maintenanceValidateWorkflowsJobTemplate, RenderCondition(buildDispatchOperationCondition("validate")), FormatRunsOn(b.opts.configuredRunsOn, "ubuntu-latest"), b.repositoryCheckoutBlock(), b.setupScriptsBlock(), b.adminPermissionsBlock(""), generateInstallCLISteps(b.ctx, b.opts.actionMode, b.opts.version, b.opts.actionTag, b.opts.resolver), b.githubScriptPin(), getCLICmdPrefix(b.opts.actionMode))
}

func (b *maintenanceWorkflowYAMLBuilder) writeLabelTriggeredJobs() {
	if !b.labelDisableJobEnabled && !b.labelApplySafeOutputsJobEnabled {
		return
	}
	maintenanceWorkflowYAMLLog.Print("Adding label-triggered jobs")
	if b.labelDisableJobEnabled {
		fmt.Fprintf(b.yaml, maintenanceLabelDisableJobTemplate, RenderCondition(buildLabeledDisableCondition()), b.opts.runsOnValue, b.actionsCheckoutBlock(), b.labelAdminPermissionsBlock(), b.githubScriptPin())
	}
	if b.labelApplySafeOutputsJobEnabled {
		fmt.Fprintf(b.yaml, maintenanceLabelApplySafeOutputsJobTemplate, RenderCondition(buildLabeledApplySafeOutputsCondition()), b.opts.runsOnValue, b.actionsCheckoutBlock(), b.labelAdminPermissionsBlock(), b.githubScriptPin())
	}
}

func (b *maintenanceWorkflowYAMLBuilder) writeDevModeJobs() {
	if b.opts.actionMode != ActionModeDev {
		return
	}
	maintenanceWorkflowYAMLLog.Printf("Adding dev-only jobs: compile-workflows and secret-validation")
	fmt.Fprintf(b.yaml, maintenanceCompileWorkflowsJobTemplate, RenderCondition(buildNotForkAndScheduled()), b.opts.runsOnValue, generateInstallCLISteps(b.ctx, b.opts.actionMode, b.opts.version, b.opts.actionTag, b.opts.resolver), getCLICmdPrefix(b.opts.actionMode), getCLICmdPrefix(b.opts.actionMode), b.setupScriptsBlock(), b.githubScriptPin(), b.compileGitHubTokenEnvBlock(), b.compileGitHubTokenWithBlock())
	copilotOrgBillingLine := ""
	if b.opts.copilotOrgBilling {
		maintenanceWorkflowYAMLLog.Print("Copilot org billing mode detected: adding GH_AW_COPILOT_ORG_BILLING=true to secret-validation step")
		copilotOrgBillingLine = "          GH_AW_COPILOT_ORG_BILLING: \"true\"\n"
	}
	fmt.Fprintf(b.yaml, maintenanceSecretValidationJobTemplate, RenderCondition(buildNotForkAndScheduleOnly()), b.opts.runsOnValue, b.actionsCheckoutBlock(), b.setupScriptsBlock(), b.githubScriptPin(), copilotOrgBillingLine, getActionPin("actions/upload-artifact"))
}

func (b *maintenanceWorkflowYAMLBuilder) resolveAppliedRunURLOutput() (string, string) {
	if b.opts.maintenanceConfig.IsJobDisabled("apply_safe_outputs") {
		return "The run URL that safe outputs were applied from (workflow_call falls back to inputs.run_url when apply_safe_outputs is disabled; other triggers leave this empty)", "${{ inputs.run_url }}"
	}
	return "The run URL that safe outputs were applied from", "${{ jobs.apply_safe_outputs.outputs.run_url }}"
}

func (b *maintenanceWorkflowYAMLBuilder) writeCloseExpiredJob(job maintenanceCloseExpiredJob) {
	fmt.Fprintf(b.yaml, maintenanceCloseExpiredJobTemplate, job.jobName, RenderCondition(buildNotForkAndScheduleOnly()), b.opts.runsOnValue, job.permissionLine, b.actionsFolderCheckoutBlock(), b.setupScriptsBlock(), job.stepName, b.githubScriptPin(), job.scriptName)
}

func (b *maintenanceWorkflowYAMLBuilder) actionsFolderCheckoutBlock() string {
	if b.opts.actionMode == ActionModeDev || b.opts.actionMode == ActionModeScript {
		maintenanceWorkflowYAMLLog.Printf("Adding checkout step for local actions (actionMode=%s)", b.opts.actionMode)
		return fmt.Sprintf(maintenanceActionsCheckoutTemplate, getActionPin("actions/checkout"))
	}
	return ""
}

func (b *maintenanceWorkflowYAMLBuilder) actionsCheckoutBlock() string {
	return fmt.Sprintf(maintenanceActionsCheckoutTemplate, getActionPin("actions/checkout"))
}

func (b *maintenanceWorkflowYAMLBuilder) repositoryCheckoutBlock() string {
	return fmt.Sprintf(maintenanceRepoCheckoutTemplate, getActionPin("actions/checkout"))
}

func (b *maintenanceWorkflowYAMLBuilder) setupScriptsBlock() string {
	return fmt.Sprintf(maintenanceSetupScriptsTemplate, b.setupActionRef)
}

func (b *maintenanceWorkflowYAMLBuilder) adminPermissionsBlock(id string) string {
	idBlock := ""
	if id != "" {
		idBlock = "        id: " + id + "\n"
	}
	return fmt.Sprintf(maintenanceAdminPermissionsTemplate, idBlock, b.githubScriptPin())
}

func (b *maintenanceWorkflowYAMLBuilder) labelAdminPermissionsBlock() string {
	return b.setupScriptsBlock() + b.adminPermissionsBlock("check_permissions")
}

func (b *maintenanceWorkflowYAMLBuilder) compileGitHubTokenEnvBlock() string {
	if b.opts.compileGitHubToken == "" {
		return ""
	}
	return "        env:\n          GH_AW_MAINTENANCE_GITHUB_TOKEN: " + b.opts.compileGitHubToken + "\n"
}

func (b *maintenanceWorkflowYAMLBuilder) compileGitHubTokenWithBlock() string {
	if b.opts.compileGitHubToken == "" {
		return ""
	}
	return "          github-token: ${{ env.GH_AW_MAINTENANCE_GITHUB_TOKEN }}\n"
}

func (b *maintenanceWorkflowYAMLBuilder) githubScriptPin() string {
	return getCachedActionPinFromResolver("actions/github-script", b.opts.resolver)
}
