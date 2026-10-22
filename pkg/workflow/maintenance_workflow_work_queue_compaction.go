package workflow

import "strings"

// Planning and applying the work queue checkpoint run in one trusted job.
func buildMaintenanceWorkQueueCompactionJobs(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
  work_queue_compaction:
    name: Reconcile and compact work queue
    if: ${{ ` + RenderCondition(buildNotForkAndScheduleOnly()) + ` }}
    runs-on: ` + opts.runsOnValue + `
    permissions:
      actions: read
      contents: write
      issues: write
    steps:
`)
	writeMaintenanceConditionalActionsCheckoutStep(&b, opts)
	writeMaintenanceSetupScriptsStep(&b, setupActionRef)
	b.WriteString(`      - name: Reconcile stranded workers and compact work queue
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        env:
          GH_AW_WORK_QUEUE_COMPACTION_PLAN_FILE: ${{ runner.temp }}/gh-aw/work-queue-compaction/plan.json
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { maintenance: reconcile } = require('${{ runner.temp }}/gh-aw/actions/reconcile_work_queue_run.cjs');
            await reconcile({ projectIssues: true });
            const { main: plan } = require('${{ runner.temp }}/gh-aw/actions/work_queue_compaction_plan.cjs');
            const result = await plan();
            if (result.status === 'planned') {
              const { main: apply } = require('${{ runner.temp }}/gh-aw/actions/work_queue_compaction_apply.cjs');
              await apply();
            }
`)
	return b.String()
}
