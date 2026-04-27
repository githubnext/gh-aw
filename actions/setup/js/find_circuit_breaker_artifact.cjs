// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Find the most recent completed workflow run (other than the current one)
 * that has a 'circuit-breaker-state' artifact, and output its run ID.
 *
 * Output: previous_run_id — the run ID string, or empty string if not found.
 */
async function main() {
  const {
    repo: { owner, repo },
    runId,
  } = context;

  core.info(`🔌 Looking for previous circuit-breaker-state artifact`);

  try {
    // Get the workflow ID of the current run
    const { data: runData } = await github.rest.actions.getWorkflowRun({
      owner,
      repo,
      run_id: runId,
    });
    const workflowId = runData.workflow_id;
    core.info(`   Workflow ID: ${workflowId}`);

    // List recent completed runs for this workflow (excluding the current one)
    const { data: runsData } = await github.rest.actions.listWorkflowRuns({
      owner,
      repo,
      workflow_id: workflowId,
      status: "completed",
      per_page: 20,
    });

    core.info(`   Found ${runsData.workflow_runs.length} recent completed runs`);

    for (const run of runsData.workflow_runs) {
      if (run.id === runId) continue;

      try {
        const { data: artifactsData } = await github.rest.actions.listWorkflowRunArtifacts({
          owner,
          repo,
          run_id: run.id,
        });

        const artifact = artifactsData.artifacts.find(a => a.name === "circuit-breaker-state" && !a.expired);
        if (artifact) {
          core.info(`   Found circuit-breaker-state artifact in run #${run.id}`);
          core.setOutput("previous_run_id", String(run.id));
          return;
        }
      } catch (error) {
        core.debug(`   Could not list artifacts for run #${run.id}: ${getErrorMessage(error)}`);
        continue;
      }
    }

    core.info(`   No previous circuit-breaker-state artifact found`);
    core.setOutput("previous_run_id", "");
  } catch (error) {
    core.warning(`Could not search for previous circuit breaker state: ${getErrorMessage(error)}`);
    core.setOutput("previous_run_id", "");
  }
}

module.exports = { main };
