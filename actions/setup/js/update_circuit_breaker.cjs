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
 *   - FAILURE/CANCELLED: increment consecutive_failures
 *
 * The updated state is written to /tmp/gh-aw/circuit-breaker-state.json,
 * which is then uploaded as the 'circuit-breaker-state' artifact.
 */
async function main() {
  const jobStatus = (process.env.GH_AW_CB_JOB_STATUS || "").toLowerCase();
  const maxFailures = parseInt(process.env.GH_AW_CB_MAX_FAILURES?.trim() || "5", 10);
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Unknown Workflow";

  const {
    repo: { owner, repo },
    runId,
  } = context;

  core.info(`🔌 Updating circuit breaker state for workflow '${workflowName}'`);
  core.info(`   Job status: ${jobStatus}`);

  // Load the previous state from the artifact downloaded in the check step (if available).
  // The check step would have written the state to /tmp/gh-aw/circuit-breaker-state.json
  // if one was found; otherwise we start fresh.
  let previousState = { consecutive_failures: 0 };

  const stateDir = "/tmp/gh-aw";
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
    // Failure or cancellation — increment the failure counter
    const newCount = (previousState.consecutive_failures ?? 0) + 1;
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
