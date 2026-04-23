package workflow

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var maintenanceWorkflowYAMLLog = logger.New("workflow:maintenance_workflow_yaml")

// buildMaintenanceWorkflowYAML generates the complete YAML content for the
// agentics-maintenance.yml workflow. It is called by GenerateMaintenanceWorkflow
// after the cron schedule and setup parameters have been resolved.
func buildMaintenanceWorkflowYAML(
	cronSchedule, scheduleDesc string,
	minExpiresDays int,
	runsOnValue string,
	actionMode ActionMode,
	version, actionTag string,
	resolver ActionSHAResolver,
	configuredRunsOn RunsOnValue,
) string {
	maintenanceWorkflowYAMLLog.Printf("Building maintenance workflow YAML: actionMode=%s minExpiresDays=%d cronSchedule=%q", actionMode, minExpiresDays, cronSchedule)

	var yaml strings.Builder

	// Add workflow header with logo and instructions
	customInstructions := `Alternative regeneration methods:
  make recompile

Or use the gh-aw CLI directly:
  ./gh-aw compile --validate --verbose

The workflow is generated when any workflow uses the 'expires' field
in create-discussions, create-issues, or create-pull-request safe-outputs configuration.
Schedule frequency is automatically determined by the shortest expiration time.`

	header := GenerateWorkflowHeader("", "pkg/workflow/maintenance_workflow.go", customInstructions)
	yaml.WriteString(header)

	yaml.WriteString(`name: Agentic Maintenance

on:
  schedule:
    - cron: "` + cronSchedule + `"  # ` + scheduleDesc + ` (based on minimum expires: ` + strconv.Itoa(minExpiresDays) + ` days)
  workflow_dispatch:
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
          - 'validate'
      run_url:
        description: 'Run URL or run ID to replay safe outputs from (e.g. https://github.com/owner/repo/actions/runs/12345 or 12345). Required when operation is safe_outputs.'
        required: false
        type: string
        default: ''
  workflow_call:
    inputs:
      operation:
        description: 'Optional maintenance operation to run (disable, enable, update, upgrade, safe_outputs, create_labels, activity_report, close_agentic_workflows_issues, clean_cache_memories, validate)'
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
  close-expired-entities:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduled()) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      discussions: write
      issues: write
      pull-requests: write
    steps:
`)

	setupActionRef := ResolveSetupActionReference(actionMode, version, actionTag, resolver)

	// Add checkout step only in dev/script mode (for local action paths)
	if actionMode == ActionModeDev || actionMode == ActionModeScript {
		maintenanceWorkflowYAMLLog.Printf("Adding checkout step for close-expired-entities (actionMode=%s)", actionMode)
		yaml.WriteString("      - name: Checkout actions folder\n")
		yaml.WriteString("        uses: " + getActionPin("actions/checkout") + "\n")
		yaml.WriteString("        with:\n")
		yaml.WriteString("          sparse-checkout: |\n")
		yaml.WriteString("            actions\n")
		yaml.WriteString("          persist-credentials: false\n\n")
	}

	// Add setup step with the resolved action reference
	yaml.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Close expired discussions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          script: |
`)

	// Add the close expired discussions script using require()
	yaml.WriteString(`            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_discussions.cjs');
            await main();

      - name: Close expired issues
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          script: |
`)

	// Add the close expired issues script using require()
	yaml.WriteString(`            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_issues.cjs');
            await main();

      - name: Close expired pull requests
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          script: |
`)

	// Add the close expired pull requests script using require()
	yaml.WriteString(`            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_pull_requests.cjs');
            await main();
`)

	// Add cleanup-cache-memory job for scheduled runs and clean_cache_memories operation
	// This job lists all caches starting with "memory-", groups them by key prefix,
	// keeps the latest run ID per group, and deletes the rest.
	cleanupCacheCondition := buildNotForkAndScheduledOrOperation("clean_cache_memories")
	yaml.WriteString(`
  cleanup-cache-memory:
    if: ${{ ` + RenderCondition(cleanupCacheCondition) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      actions: write
    steps:
`)

	// Add checkout step only in dev/script mode (for local action paths)
	if actionMode == ActionModeDev || actionMode == ActionModeScript {
		yaml.WriteString("      - name: Checkout actions folder\n")
		yaml.WriteString("        uses: " + getActionPin("actions/checkout") + "\n")
		yaml.WriteString("        with:\n")
		yaml.WriteString("          sparse-checkout: |\n")
		yaml.WriteString("            actions\n")
		yaml.WriteString("          persist-credentials: false\n\n")
	}

	yaml.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Cleanup outdated cache-memory entries
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/cleanup_cache_memory.cjs');
            await main();
`)

	// Add unified run_operation job for all dispatch operations except those with dedicated jobs
	// (safe_outputs, create_labels, activity_report, close_agentic_workflows_issues, clean_cache_memories, validate)
	runOperationCondition := buildRunOperationCondition("safe_outputs", "create_labels", "activity_report", "close_agentic_workflows_issues", "clean_cache_memories", "validate")
	yaml.WriteString(`
  run_operation:
    if: ${{ ` + RenderCondition(runOperationCondition) + ` }}
    runs-on: ` + runsOnValue + `
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)

	yaml.WriteString(generateInstallCLISteps(actionMode, version, actionTag, resolver))
	yaml.WriteString(`      - name: Run operation
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_AW_OPERATION: ${{ inputs.operation }}
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/run_operation_update_upgrade.cjs');
            await main();

      - name: Record outputs
        id: record
        run: echo "operation=${{ inputs.operation }}" >> "$GITHUB_OUTPUT"
`)

	// Add apply_safe_outputs job for workflow_dispatch with operation == 'safe_outputs'
	yaml.WriteString(`
  apply_safe_outputs:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("safe_outputs")) + ` }}
    runs-on: ` + runsOnValue + `
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Apply Safe Outputs
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
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
        run: echo "run_url=${{ inputs.run_url }}" >> "$GITHUB_OUTPUT"
`)

	// Add create_labels job for workflow_dispatch with operation == 'create_labels'
	yaml.WriteString(`
  create_labels:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("create_labels")) + ` }}
    runs-on: ` + runsOnValue + `
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)

	yaml.WriteString(generateInstallCLISteps(actionMode, version, actionTag, resolver))
	yaml.WriteString(`      - name: Create missing labels
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        env:
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/create_labels.cjs');
            await main();
`)

	// Add activity_report job for workflow_dispatch with operation == 'activity_report'
	yaml.WriteString(`
  activity_report:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("activity_report")) + ` }}
    runs-on: ` + runsOnValue + `
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)

	yaml.WriteString(generateInstallCLISteps(actionMode, version, actionTag, resolver))
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
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(actionMode) + `
        run: |
          ${GH_AW_CMD_PREFIX} logs \
            --repo "${{ github.repository }}" \
            --start-date -1w \
            --count 100 \
            --output ./.cache/gh-aw/activity-report-logs \
            --format markdown \
            > ./.cache/gh-aw/activity-report-logs/report.md

      - name: Save activity report logs cache
        if: ${{ always() }}
        uses: ` + getActionPin("actions/cache/save") + `
        with:
          path: ./.cache/gh-aw/activity-report-logs
          key: ${{ steps.activity_report_logs_cache.outputs.cache-primary-key }}

      - name: Generate activity report issue
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

	// Add close_agentic_workflows_issues job for workflow_dispatch with operation == 'close_agentic_workflows_issues'
	yaml.WriteString(`
  close_agentic_workflows_issues:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("close_agentic_workflows_issues")) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      issues: write
    steps:
`)

	// Add checkout step only in dev/script mode (for local action paths)
	if actionMode == ActionModeDev || actionMode == ActionModeScript {
		yaml.WriteString("      - name: Checkout actions folder\n")
		yaml.WriteString("        uses: " + getActionPin("actions/checkout") + "\n")
		yaml.WriteString("        with:\n")
		yaml.WriteString("          sparse-checkout: |\n")
		yaml.WriteString("            actions\n")
		yaml.WriteString("          persist-credentials: false\n\n")
	}

	yaml.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Close no-repro agentic-workflows issues
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_agentic_workflows_issues.cjs');
            await main();
`)

	// Add validate_workflows job for workflow_dispatch with operation == 'validate'
	// This job uses ubuntu-latest by default (needs full runner for CLI installation).
	validateRunsOnValue := FormatRunsOn(configuredRunsOn, "ubuntu-latest")
	yaml.WriteString(`
  validate_workflows:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("validate")) + ` }}
    runs-on: ` + validateRunsOnValue + `
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)

	yaml.WriteString(generateInstallCLISteps(actionMode, version, actionTag, resolver))

	yaml.WriteString(`      - name: Validate workflows and file issue on findings
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        env:
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/run_validate_workflows.cjs');
            await main();
`)

	// Add compile-workflows and zizmor-scan jobs only in dev mode
	// These jobs are specific to the gh-aw repository and require go.mod, make build, etc.
	// User repositories won't have these dependencies, so we skip them in release mode
	if actionMode == ActionModeDev {
		maintenanceWorkflowYAMLLog.Printf("Adding dev-only jobs: compile-workflows and secret-validation")
		// Add compile-workflows job
		yaml.WriteString(`
  compile-workflows:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduled()) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      contents: read
      issues: write
    steps:
`)

		// Dev mode: checkout entire repository (no sparse checkout, but no credentials)
		yaml.WriteString(`      - name: Checkout repository
        uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
        with:
          persist-credentials: false

`)

		yaml.WriteString(generateInstallCLISteps(actionMode, version, actionTag, resolver))
		yaml.WriteString(`      - name: Compile workflows
        run: |
          ` + getCLICmdPrefix(actionMode) + ` compile --validate --validate-images --verbose
          echo "✓ All workflows compiled successfully"

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check for out-of-sync workflows and create issue if needed
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_workflow_recompile_needed.cjs');
            await main();

  secret-validation:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduled()) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      contents: read
    steps:
`)

		// Add checkout step only in dev mode (for local action paths)
		yaml.WriteString(`      - name: Checkout actions folder
        uses: ` + getActionPin("actions/checkout") + `
        with:
          sparse-checkout: |
            actions
          persist-credentials: false

`)

		yaml.WriteString(`      - name: Setup Node.js
        uses: actions/setup-node@39370e3970a6d050c480ffad4ff0ed4d3fdee5af # v4.1.0
        with:
          node-version: '22'

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Validate Secrets
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        env:
          # GitHub tokens
          GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
          GH_AW_GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}
          GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
          GH_AW_COPILOT_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN }}
          # AI Engine API keys
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

	return yaml.String()
}
