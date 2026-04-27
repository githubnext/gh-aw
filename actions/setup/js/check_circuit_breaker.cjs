// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { writeDenialSummary } = require("./pre_activation_summary.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Circuit breaker check for agentic workflows.
 *
 * Reads the circuit-breaker state from <stateDir>/circuit-breaker-state.json
 * (downloaded by the preceding actions/download-artifact step) and implements
 * the standard closed → open → half-open state machine pattern:
 *
 *   CLOSED    → normal execution (consecutive_failures < max)
 *   OPEN      → execution blocked (consecutive_failures >= max AND cooldown not elapsed)
 *   HALF-OPEN → one retry allowed (consecutive_failures >= max AND cooldown elapsed)
 *
 * GH_AW_CB_STATE_DIR overrides the default state directory (/tmp/gh-aw) for tests.
 */
async function main() {
  const maxFailures = parseInt(process.env.GH_AW_CB_MAX_FAILURES?.trim() || "5", 10);
  const timeWindowMinutes = parseInt(process.env.GH_AW_CB_TIME_WINDOW_MINUTES?.trim() || "1440", 10);
  const cooldownMinutes = parseInt(process.env.GH_AW_CB_COOLDOWN_MINUTES?.trim() || "60", 10);
  const notify = (process.env.GH_AW_CB_NOTIFY?.trim() || "true") === "true";
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Unknown Workflow";
  const stateDir = process.env.GH_AW_CB_STATE_DIR || "/tmp/gh-aw";

  core.info(`🔌 Circuit breaker check for workflow '${workflowName}'`);
  core.info(`   Configuration: max=${maxFailures} failures, window=${timeWindowMinutes}m, cooldown=${cooldownMinutes}m`);

  // Read the circuit breaker state downloaded by the preceding download-artifact step.
  const stateFile = path.join(stateDir, "circuit-breaker-state.json");
  let state = { consecutive_failures: 0 };

  try {
    if (fs.existsSync(stateFile)) {
      const content = fs.readFileSync(stateFile, "utf8");
      state = JSON.parse(content);
      core.info(`   Loaded state: consecutive_failures=${state.consecutive_failures}`);
    } else {
      core.info(`   No previous state found — starting fresh (circuit CLOSED)`);
    }
  } catch (error) {
    // If we can't load the previous state, assume circuit is closed (fail-open for availability)
    core.warning(`Could not read previous circuit breaker state from ${stateFile}: ${getErrorMessage(error)}. Assuming circuit is closed.`);
  }

  const consecutiveFailures = state.consecutive_failures ?? 0;
  const lastFailure = state.last_failure ? new Date(state.last_failure) : null;

  core.info(`   Consecutive failures: ${consecutiveFailures} / ${maxFailures}`);

  const nowMs = Date.now();
  const windowMs = timeWindowMinutes * 60 * 1000;
  const failureIsRecent = lastFailure !== null && nowMs - lastFailure.getTime() <= windowMs;

  // CLOSED state: fewer failures than threshold, or failures are outside the time window
  if (consecutiveFailures < maxFailures || !failureIsRecent) {
    core.info(`✅ Circuit breaker is CLOSED — workflow execution allowed`);
    core.setOutput("circuit_breaker_ok", "true");
    core.setOutput("consecutive_failures", String(consecutiveFailures));
    return;
  }

  // Circuit is OPEN — check if cooldown has elapsed (HALF-OPEN state)
  const cooldownMs = cooldownMinutes * 60 * 1000;
  const cooldownElapsed = lastFailure !== null && nowMs - lastFailure.getTime() >= cooldownMs;

  if (cooldownElapsed) {
    core.info(`🔄 Circuit breaker is HALF-OPEN — cooldown elapsed, allowing one retry`);
    core.setOutput("circuit_breaker_ok", "true");
    core.setOutput("consecutive_failures", String(consecutiveFailures));
    return;
  }

  // OPEN state: block execution
  const minutesSinceFail = lastFailure ? Math.floor((nowMs - lastFailure.getTime()) / 60000) : 0;
  const minutesUntilRetry = Math.max(0, cooldownMinutes - minutesSinceFail);

  core.warning(`🔴 Circuit breaker is OPEN — workflow execution blocked`);
  core.warning(`   ${consecutiveFailures} consecutive failures in the last ${timeWindowMinutes} minutes`);
  core.warning(`   Retry allowed in approximately ${minutesUntilRetry} minute(s)`);

  core.setOutput("circuit_breaker_ok", "false");
  core.setOutput("consecutive_failures", String(consecutiveFailures));

  if (notify) {
    core.error(
      `Circuit breaker OPEN for '${workflowName}': ${consecutiveFailures} consecutive failures detected. ` +
        `Workflow execution is blocked until the cooldown period expires (≈${minutesUntilRetry} min remaining). ` +
        `Fix the underlying issue and wait for the cooldown, or manually reset by deleting the 'circuit-breaker-state' artifact.`
    );
  }

  await writeDenialSummary(
    `Circuit breaker OPEN for workflow '${workflowName}': ${consecutiveFailures} consecutive failures detected within the ${timeWindowMinutes}-minute window.`,
    `The circuit breaker will allow a retry after the cooldown period (≈${minutesUntilRetry} min remaining). Fix the underlying issue and wait, or delete the \`circuit-breaker-state\` artifact to reset manually.`
  );
}

module.exports = { main };
