// @ts-check

/**
 * Detect agent engine errors in the agent stdio log.
 *
 * Scans the agent stdio log for known error patterns and sets GitHub Actions
 * output variables for each detected error class:
 *
 *   - inference_access_error: The COPILOT_GITHUB_TOKEN does not have valid
 *     access to inference (e.g., "Access denied by policy settings").
 *   - mcp_policy_error: MCP servers were blocked by enterprise/organization
 *     policy (e.g., "MCP servers were blocked by policy: 'github', 'safeoutputs'").
 *   - agentic_engine_timeout: The agentic engine process was killed by a
 *     signal (SIGTERM/SIGKILL/SIGINT), typically due to the step
 *     timeout-minutes limit being reached.
 *   - model_not_supported_error: The configured model is invalid or unsupported
 *     for the selected engine/account (for example unknown model name, model not
 *     found, or model unavailable for the plan).
 *   - capi_quota_exceeded_error: The Copilot CAPI quota has been exhausted
 *     (e.g., "CAPIError: 429 429 quota exceeded").
 *
 * This replaces the individual bash scripts (detect_inference_access_error.sh,
 * detect_mcp_policy_error.sh) with a single JavaScript step.
 *
 * Exit codes:
 *   0 — Always succeeds (uses continue-on-error in the workflow step)
 */

"use strict";

const fs = require("fs");

const LOG_FILE = "/tmp/gh-aw/agent-stdio.log";

// Pattern: Copilot CLI inference access denied
const INFERENCE_ACCESS_ERROR_PATTERN = /Access denied by policy settings|invalid access to inference/;

// Pattern: MCP servers blocked by enterprise/organization policy
const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;

// Pattern: Agentic engine process killed by signal (timeout).
// When GitHub Actions cancels a step due to timeout-minutes, the runner sends
// SIGINT/SIGTERM/SIGKILL to the process group.  The copilot_harness.cjs (and
// other engine wrappers) log the signal in their close handlers:
//   [copilot-harness] attempt 1: process closed exitCode=1 signal=SIGTERM ...
// The pattern matches any "signal=SIG(TERM|KILL|INT)" occurrence in the log,
// making it engine-agnostic.
const AGENTIC_ENGINE_TIMEOUT_PATTERN = /signal=SIG(?:TERM|KILL|INT)/;

// Pattern: Configured model is invalid or unavailable.
// Covers common engine/provider variants:
//   - "The requested model is not supported"
//   - "invalid model name '...'"
//   - "unknown model <id>"
//   - "model ... not found"
//   - "model ... does not exist"
const MODEL_NOT_SUPPORTED_PATTERN =
  /(?:The requested model is not supported|invalid model(?:\s+name)?\s+['"`]?[a-z0-9._:/@-]+['"`]?(?=(?:\s*$|\s*[\n\r.,;:!?)]))|unknown model\s+['"`]?[a-z0-9._:/@-]+['"`]?(?=(?:\s*$|\s*[\n\r.,;:!?)]))|model(?:\s+name)?\s+['"`]?[a-z0-9._:/@-]+['"`]?\s+(?:is\s+)?(?:not found|does not exist|not supported|not available|unavailable))/i;

// Pattern: Copilot/CAPI quota exhaustion.
// Matches the observed error: "CAPIError: 429 429 quota exceeded".
// Quota exhaustion is a persistent, non-retryable condition.
const CAPI_QUOTA_EXCEEDED_PATTERN = /CAPIError:\s*429\s+429\s+quota exceeded/i;

/**
 * Determines if the collected output contains the observed Copilot/CAPI quota exhaustion error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isCAPIQuotaExceededError(output) {
  return CAPI_QUOTA_EXCEEDED_PATTERN.test(output);
}

/**
 * Detect known error patterns in a log string and return detection results.
 * @param {string} logContent - Contents of the agent stdio log
 * @returns {{ inferenceAccessError: boolean, mcpPolicyError: boolean, agenticEngineTimeout: boolean, modelNotSupportedError: boolean, capiQuotaExceededError: boolean }}
 */
function detectErrors(logContent) {
  return {
    inferenceAccessError: INFERENCE_ACCESS_ERROR_PATTERN.test(logContent),
    mcpPolicyError: MCP_POLICY_BLOCKED_PATTERN.test(logContent),
    agenticEngineTimeout: AGENTIC_ENGINE_TIMEOUT_PATTERN.test(logContent),
    modelNotSupportedError: MODEL_NOT_SUPPORTED_PATTERN.test(logContent),
    capiQuotaExceededError: isCAPIQuotaExceededError(logContent),
  };
}

/**
 * Write GitHub Actions outputs to $GITHUB_OUTPUT.
 * @param {{ inferenceAccessError: boolean, mcpPolicyError: boolean, agenticEngineTimeout: boolean, modelNotSupportedError: boolean, capiQuotaExceededError: boolean }} results
 */
function writeOutputs(results) {
  const outputFile = process.env.GITHUB_OUTPUT;
  if (!outputFile) {
    process.stderr.write("[detect-agent-errors] GITHUB_OUTPUT not set — skipping output\n");
    return;
  }

  const lines = [
    `inference_access_error=${results.inferenceAccessError}`,
    `mcp_policy_error=${results.mcpPolicyError}`,
    `agentic_engine_timeout=${results.agenticEngineTimeout}`,
    `model_not_supported_error=${results.modelNotSupportedError}`,
    `capi_quota_exceeded_error=${results.capiQuotaExceededError}`,
  ];
  fs.appendFileSync(outputFile, lines.join("\n") + "\n");
}

function main() {
  let logContent = "";

  if (fs.existsSync(LOG_FILE)) {
    logContent = fs.readFileSync(LOG_FILE, "utf8");
  } else {
    process.stderr.write(`[detect-agent-errors] Log file not found: ${LOG_FILE}\n`);
  }

  const results = detectErrors(logContent);

  if (results.inferenceAccessError) {
    process.stderr.write("[detect-agent-errors] Detected inference access error in agent log\n");
  }
  if (results.mcpPolicyError) {
    process.stderr.write("[detect-agent-errors] Detected MCP policy error in agent log\n");
  }
  if (results.agenticEngineTimeout) {
    process.stderr.write("[detect-agent-errors] Detected timeout: engine process was killed by signal (step timeout-minutes likely exceeded)\n");
  }
  if (results.modelNotSupportedError) {
    process.stderr.write("[detect-agent-errors] Detected model configuration error: configured model is invalid or unavailable for this engine/account\n");
  }
  if (results.capiQuotaExceededError) {
    process.stderr.write("[detect-agent-errors] Detected CAPI quota exhaustion: Copilot quota has been exceeded\n");
  }

  writeOutputs(results);
}

if (require.main === module) {
  main();
}

module.exports = { detectErrors, isCAPIQuotaExceededError, INFERENCE_ACCESS_ERROR_PATTERN, MCP_POLICY_BLOCKED_PATTERN, AGENTIC_ENGINE_TIMEOUT_PATTERN, MODEL_NOT_SUPPORTED_PATTERN, CAPI_QUOTA_EXCEEDED_PATTERN };
