package workflow

import "strings"

// Planning and applying the work queue checkpoint run in one trusted job.
func buildMaintenanceWorkQueueCompactionJobs(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
  work_queue_compaction:
    name: Compact work queue
    if: ${{ ` + RenderCondition(buildNotForkAndScheduleOnly()) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      contents: write
    steps:
`)
	writeMaintenanceConditionalActionsCheckoutStep(&b, opts)
	writeMaintenanceSetupScriptsStep(&b, setupActionRef)
	b.WriteString(`      - name: Plan and apply work queue compaction
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          GH_AW_WORK_QUEUE_COMPACTION_PLAN_FILE: ${{ runner.temp }}/gh-aw/work-queue-compaction/plan.json
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main: plan } = require('${{ runner.temp }}/gh-aw/actions/work_queue_compaction_plan.cjs');
            const result = await plan();
            if (result.status === 'planned') {
              const { main: apply } = require('${{ runner.temp }}/gh-aw/actions/work_queue_compaction_apply.cjs');
              await apply();
            }
`)
	return b.String()
}

func buildMaintenanceWorkQueueIssueClosureJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
  work_queue_issue_closure:
    name: Cancel Work for closed related Issue
    if: ${{ github.event_name == 'issues' && github.event.action == 'closed' && github.event.issue.pull_request == null && github.event.repository.fork == false }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      contents: write
      issues: write
      actions: read
    steps:
`)
	writeMaintenanceConditionalActionsCheckoutStep(&b, opts)
	writeMaintenanceSetupScriptsStep(&b, setupActionRef)
	b.WriteString(`      - name: Record Issue closure cancellation
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/cancel_work_queue_closed_issue.cjs');
            await main();
`)
	return b.String()
}
