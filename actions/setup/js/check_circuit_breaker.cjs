// @ts-check
/// <reference types="@actions/github-script" />

const { writeDenialSummary } = require("./pre_activation_summary.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Circuit breaker check for agentic workflows.
 *
 * Implements the standard closed → open → half-open state machine pattern:
 *   CLOSED  → normal execution (consecutive_failures < max)
 *   OPEN    → execution blocked (consecutive_failures >= max AND cooldown not elapsed)
 *   HALF-OPEN → one retry allowed (consecutive_failures >= max AND cooldown elapsed)
 *
 * State is persisted as a GitHub Actions artifact ("circuit-breaker-state") from
 * the previous run and is read here at pre-activation time.
 */
async function main() {
  const maxFailures = parseInt(process.env.GH_AW_CB_MAX_FAILURES?.trim() || "5", 10);
  const timeWindowMinutes = parseInt(process.env.GH_AW_CB_TIME_WINDOW_MINUTES?.trim() || "1440", 10);
  const cooldownMinutes = parseInt(process.env.GH_AW_CB_COOLDOWN_MINUTES?.trim() || "60", 10);
  const notify = (process.env.GH_AW_CB_NOTIFY?.trim() || "true") === "true";
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Unknown Workflow";

  const {
    repo: { owner, repo },
    runId,
  } = context;

  core.info(`🔌 Circuit breaker check for workflow '${workflowName}'`);
  core.info(`   Configuration: max=${maxFailures} failures, window=${timeWindowMinutes}m, cooldown=${cooldownMinutes}m`);

  // Fetch the circuit-breaker-state artifact from a recent workflow run.
  // We look at completed runs for this same workflow file that have a circuit-breaker-state artifact.
  let state = { consecutive_failures: 0 };

  try {
    // Get the workflow ID from the current run
    const { data: runData } = await github.rest.actions.getWorkflowRun({
      owner,
      repo,
      run_id: runId,
    });
    const workflowId = runData.workflow_id;

    // List recent completed runs (excluding the current one) to find a state artifact
    const { data: runsData } = await github.rest.actions.listWorkflowRuns({
      owner,
      repo,
      workflow_id: workflowId,
      status: "completed",
      per_page: 10,
    });

    core.info(`   Found ${runsData.workflow_runs.length} recent completed runs`);

    // Find the most recent run that has a circuit-breaker-state artifact
    const nowMs = Date.now();
    const windowMs = timeWindowMinutes * 60 * 1000;

    for (const run of runsData.workflow_runs) {
      if (run.id === runId) continue;

      try {
        const { data: artifactsData } = await github.rest.actions.listWorkflowRunArtifacts({
          owner,
          repo,
          run_id: run.id,
        });

        const artifact = artifactsData.artifacts.find(a => a.name === "circuit-breaker-state");
        if (!artifact) continue;

        core.info(`   Found circuit-breaker-state artifact in run #${run.id}`);

        // Download the artifact ZIP
        const { data: zipData } = await github.rest.actions.downloadArtifact({
          owner,
          repo,
          artifact_id: artifact.id,
          archive_format: "zip",
        });

        // Parse the artifact contents — zipData is a Buffer of the ZIP
        const AdmZip = require("adm-zip");
        const zip = new AdmZip(Buffer.from(zipData));
        const entry = zip.getEntries().find(e => e.entryName === "circuit-breaker-state.json");

        if (entry) {
          const content = entry.getData().toString("utf8");
          state = JSON.parse(content);
          core.info(`   Loaded state: consecutive_failures=${state.consecutive_failures}`);
        }
        break;
      } catch {
        // Artifact not found or couldn't be parsed — continue to next run
        continue;
      }
    }
  } catch (error) {
    // If we can't load previous state, assume circuit is closed (fail-open for availability)
    core.warning(`Could not load previous circuit breaker state: ${getErrorMessage(error)}. Assuming circuit is closed.`);
  }

  const consecutiveFailures = state.consecutive_failures ?? 0;
  const lastFailure = state.last_failure ? new Date(state.last_failure) : null;

  core.info(`   Consecutive failures: ${consecutiveFailures} / ${maxFailures}`);

  // Check if we're within the time window
  const nowMs = Date.now();
  const windowMs = timeWindowMinutes * 60 * 1000;
  const failureIsRecent = lastFailure && nowMs - lastFailure.getTime() <= windowMs;

  // CLOSED state: fewer failures than threshold, or failures are outside the time window
  if (consecutiveFailures < maxFailures || !failureIsRecent) {
    core.info(`✅ Circuit breaker is CLOSED — workflow execution allowed`);
    core.setOutput("circuit_breaker_ok", "true");
    core.setOutput("consecutive_failures", String(consecutiveFailures));
    return;
  }

  // Circuit is OPEN — check if cooldown has elapsed (HALF-OPEN state)
  const cooldownMs = cooldownMinutes * 60 * 1000;
  const cooldownElapsed = lastFailure && nowMs - lastFailure.getTime() >= cooldownMs;

  if (cooldownElapsed) {
    core.info(`🔄 Circuit breaker is HALF-OPEN — cooldown elapsed, allowing one retry`);
    core.setOutput("circuit_breaker_ok", "true");
    core.setOutput("consecutive_failures", String(consecutiveFailures));
    return;
  }

  // OPEN state: block execution
  const minutesSinceFail = lastFailure ? Math.floor((nowMs - lastFailure.getTime()) / 60000) : 0;
  const minutesUntilRetry = cooldownMinutes - minutesSinceFail;

  core.warning(`🔴 Circuit breaker is OPEN — workflow execution blocked`);
  core.warning(`   ${consecutiveFailures} consecutive failures in the last ${timeWindowMinutes} minutes`);
  core.warning(`   Retry allowed in approximately ${minutesUntilRetry} minute(s)`);

  core.setOutput("circuit_breaker_ok", "false");
  core.setOutput("consecutive_failures", String(consecutiveFailures));

  if (notify) {
    core.error(
      `Circuit breaker OPEN for '${workflowName}': ${consecutiveFailures} consecutive failures detected. ` +
        `Workflow execution is blocked until the cooldown period expires (≈${minutesUntilRetry} min remaining). ` +
        `Fix the underlying issue and wait for the cooldown, or manually reset the circuit breaker by deleting the '${
          // eslint-disable-next-line no-undef
          "circuit-breaker-state"
        }' artifact.`
    );
  }

  await writeDenialSummary(
    `Circuit breaker OPEN for workflow '${workflowName}': ${consecutiveFailures} consecutive failures detected within the ${timeWindowMinutes}-minute window.`,
    `The circuit breaker will allow a retry after the cooldown period (≈${minutesUntilRetry} min remaining). ` + `Fix the underlying issue and wait, or delete the \`circuit-breaker-state\` artifact to reset manually.`
  );
}

module.exports = { main };
