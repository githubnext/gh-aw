// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Circuit breaker state update — runs with if: always() after agent execution.
 *
 * Reads the current job status and updates the circuit breaker state:
 *   - SUCCESS: reset consecutive_failures to 0
 *   - FAILURE/CANCELLED: increment consecutive_failures (but only count failures
 *     within the configured time window; older failures are discarded)
 *
 * The updated state is written to <stateDir>/circuit-breaker-state.json,
 * which is then uploaded as the 'circuit-breaker-state' artifact.
 *
 * GH_AW_CB_STATE_DIR overrides the default state directory (/tmp/gh-aw) for tests.
 */
async function main() {
  const jobStatus = (process.env.GH_AW_CB_JOB_STATUS || "").toLowerCase();
  const maxFailures = parseInt(process.env.GH_AW_CB_MAX_FAILURES?.trim() || "5", 10);
  const timeWindowMinutes = parseInt(process.env.GH_AW_CB_TIME_WINDOW_MINUTES?.trim() || "1440", 10);
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Unknown Workflow";
  const stateDir = process.env.GH_AW_CB_STATE_DIR || "/tmp/gh-aw";

  core.info(`🔌 Updating circuit breaker state for workflow '${workflowName}'`);
  core.info(`   Job status: ${jobStatus}`);

  // Load the previous state from the artifact downloaded in the check step (if available).
  // The check step would have written the state to <stateDir>/circuit-breaker-state.json
  // if one was found; otherwise we start fresh.
  let previousState = { consecutive_failures: 0 };

  const stateFile = path.join(stateDir, "circuit-breaker-state.json");

  try {
    if (fs.existsSync(stateFile)) {
      const content = fs.readFileSync(stateFile, "utf8");
      previousState = JSON.parse(content);
      core.info(`   Loaded previous state: consecutive_failures=${previousState.consecutive_failures}`);
    }
  } catch (error) {
    core.warning(`Could not load existing circuit breaker state: ${getErrorMessage(error)}. Starting fresh.`);
  }

  const nowISO = new Date().toISOString();
  const nowMs = Date.now();
  const windowMs = timeWindowMinutes * 60 * 1000;

  // If the last failure is outside the time window, the accumulated count no longer
  // applies and we treat the previous state as if it were a fresh start.
  const lastFailureMs = previousState.last_failure ? new Date(previousState.last_failure).getTime() : null;
  const previousCountInWindow = lastFailureMs !== null && nowMs - lastFailureMs <= windowMs ? (previousState.consecutive_failures ?? 0) : 0;

  let newState;

  if (jobStatus === "success") {
    // Success — reset the circuit breaker
    newState = {
      consecutive_failures: 0,
      last_success: nowISO,
      last_failure: previousState.last_failure ?? null,
      circuit_opened_at: null,
    };
    core.info(`✅ Job succeeded — resetting circuit breaker (was ${previousState.consecutive_failures} failures)`);
  } else {
    // Failure or cancellation — increment the failure counter (using only in-window count)
    const newCount = previousCountInWindow + 1;
    // Preserve the original circuit_opened_at timestamp from when the circuit first opened.
    // Using ?? ensures we only record the timestamp on the first opening (newCount === maxFailures),
    // and keep that value on all subsequent failures without overwriting it.
    const circuitOpenedAt = newCount >= maxFailures ? (previousState.circuit_opened_at ?? nowISO) : null;

    newState = {
      consecutive_failures: newCount,
      last_failure: nowISO,
      last_success: previousState.last_success ?? null,
      circuit_opened_at: circuitOpenedAt,
    };

    core.info(`❌ Job failed — consecutive failures: ${newCount} / ${maxFailures}`);
    if (newCount >= maxFailures) {
      core.warning(`🔴 Circuit breaker threshold reached (${newCount} consecutive failures). Circuit is now OPEN.`);
    }
  }

  // Write the updated state to disk so the upload-artifact step can find it
  try {
    if (!fs.existsSync(stateDir)) {
      fs.mkdirSync(stateDir, { recursive: true });
    }
    fs.writeFileSync(stateFile, JSON.stringify(newState, null, 2), "utf8");
    core.info(`   State written to ${stateFile}`);
  } catch (error) {
    core.error(`Failed to write circuit breaker state: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
