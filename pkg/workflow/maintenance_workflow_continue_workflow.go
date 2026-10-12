package workflow

import "strings"

// maintenanceDispatchContinuationsOperation is the Agentic Maintenance operation that
// dispatches queued continue-workflow hops. The continue_workflow handler requests it right
// after recording a hop; the schedule retries dispatches that failed.
const maintenanceDispatchContinuationsOperation = "dispatch_continuations"

// buildMaintenanceContinueWorkflowJob dispatches queued continuation hops.
//
// Each hop is claimed in the continuation ledger (compare-and-swap) before the workflow is
// dispatched, so overlapping maintenance runs cannot start the same hop twice.
func buildMaintenanceContinueWorkflowJob(opts buildMaintenanceWorkflowYAMLOptions, setupActionRef string) string {
	var b strings.Builder
	b.WriteString(`
  continue_workflow_dispatch:
    name: Dispatch workflow continuations
    if: ${{ ` + RenderCondition(buildNotForkAndScheduleOnlyOrOperation(maintenanceDispatchContinuationsOperation)) + ` }}
    runs-on: ` + opts.runsOnValue + `
    # contents: write updates the continuation ledger branch; actions: write dispatches the next run.
    permissions:
      actions: write
      contents: write
    steps:
`)
	writeMaintenanceConditionalActionsCheckoutStep(&b, opts)
	writeMaintenanceSetupScriptsStep(&b, setupActionRef)
	b.WriteString(`      - name: Dispatch queued continuation runs
        uses: ` + getCachedActionPinFromResolver("actions/github-script", opts.resolver) + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/dispatch_continue_workflow.cjs');
            await main();
`)
	return b.String()
}
