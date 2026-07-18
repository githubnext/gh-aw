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
 *   - agentic_engine_timeout: A timeout signature was detected in engine logs.
 *     This includes process termination by signal (SIGTERM/SIGKILL/SIGINT),
 *     typically due to step timeout-minutes, and SDK idle-timeout messages
 *     ("Timeout after <n>ms waiting for session.idle").
 *   - model_not_supported_error: The configured model is invalid or unsupported
 *     for the selected engine/account (for example unknown model name, model not
 *     found, or model unavailable for the plan).
 *   - http_400_response_error: The engine surfaced a generic HTTP 400 Bad Request
 *     response (for example "Response status code does not indicate success: 400 (Bad Request)").
 *   - capi_quota_exceeded_error: The Copilot CAPI quota has been exhausted
 *     or rate-limited (e.g., "CAPIError: 429 429 quota exceeded",
 *     "CAPIError: Too Many Requests"). All matched forms are treated as
 *     non-retryable because the Copilot SDK has already retried internally
 *     before surfacing the error.
 *   - invocation_cap_exceeded: The per-run pooled LLM invocation cap is
 *     fully exhausted (e.g., "CAPIError: 429 Maximum LLM invocations exceeded (N/N)"
 *     or `"type":"max_runs_exceeded"`). This is more specific than generic
 *     CAPI quota exhaustion and takes precedence in step outputs.
 * This replaces the individual bash scripts (detect_inference_access_error.sh,
 * detect_mcp_policy_error.sh) with a single JavaScript step.
 *
 * Exit codes:
 *   0 — Always succeeds (uses continue-on-error in the workflow step)
 */

"use strict";

const fs = require("fs");
const { MAX_RUNS_EXCEEDED_PATTERNS, isMaxRunsExceededError } = require("./harness_retry_guard.cjs");

const LOG_FILE = "/tmp/gh-aw/agent-stdio.log";

// Pattern: Copilot CLI inference access denied
const INFERENCE_ACCESS_ERROR_PATTERN = /Access denied by policy settings|invalid access to inference/;

// Pattern: MCP servers blocked by enterprise/organization policy
const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;

// Pattern: Agentic engine timeout.
// Covers both timeout signatures observed in engine logs:
//   1) Process killed by signal after step timeout-minutes:
//      [copilot-harness] ... process closed exitCode=1 signal=SIGTERM ...
//   2) Copilot SDK idle-timeout while waiting for session.idle:
//      [sdk-driver] error: Timeout after 870000ms waiting for session.idle
// The second form can occur even when the driver collected output, and should
// still be classified as a timeout for conclusion/reporting purposes.
const AGENTIC_ENGINE_TIMEOUT_PATTERN = /(?:signal=SIG(?:TERM|KILL|INT)|Timeout after \d+ms waiting for session\.idle)/;

// Pattern: Configured model is invalid or unavailable.
// Covers common engine/provider variants:
//   - "The requested model is not supported"
//   - "invalid model name '...'"
//   - "unknown model <id>"
//   - "model ... not found"
//   - "model ... does not exist"
//   - "Model not found" (standalone, e.g. AIC api-proxy 404: "404 Not Found: Model not found")
const MODEL_NOT_SUPPORTED_PATTERN =
  /(?:The requested model is not supported|invalid model(?:\s+name)?\s+['"`]?[a-z0-9._:/@-]+['"`]?(?=(?:\s*$|\s*[\n\r.,;:!?)]))|unknown model\s+['"`]?[a-z0-9._:/@-]+['"`]?(?=(?:\s*$|\s*[\n\r.,;:!?)]))|model(?:\s+name)?\s+['"`]?[a-z0-9._:/@-]+['"`]?\s+(?:is\s+)?(?:not found|does not exist|not supported|not available|unavailable)|404\b[^\n]*\bModel\s+not\s+found)/i;

// Pattern: Generic HTTP 400 Bad Request responses emitted by engine / SDK wrappers.
// NOTE: keep in sync with HTTP_400_RESPONSE_ERROR_PATTERN in copilot_harness.cjs.
// Also matches "400 400 400 no model endpoints available given user constraints" which is emitted
// by the Copilot SDK when no model endpoints are available for the user's configured constraints.
// Also matches "400 400 400 stream_options: Extra inputs are not permitted" which is emitted when
// the Copilot SDK sends an OpenAI-only field to an Anthropic-type provider.
// The non-first alternatives are anchored to a leading "400" to avoid false positives from unrelated
// diagnostic or informational messages that might contain the phrase.
const HTTP_400_RESPONSE_ERROR_PATTERN =
  /(?:Response status code does not indicate success:\s*400(?:\s*\(Bad Request\))?|400[^\n]*no model endpoints available given user constraints|400[^\n]*stream_options:\s*Extra inputs are not permitted)/i;

// Pattern: Copilot/CAPI quota exhaustion and rate-limit responses.
// Matches all observed forms:
//   "CAPIError: 429 429 quota exceeded"  (original observed form)
//   "CAPIError: 429 Too Many Requests"   (HTTP 429 form)
//   "CAPIError: Too Many Requests"       (no status code in message)
// All forms are treated as non-retryable; the Copilot SDK has already retried
// internally before surfacing this error (evidenced by "retried 5 times" context).
const CAPI_QUOTA_EXCEEDED_PATTERN = /CAPIError:\s*(?:429\s+)?(?:429\s+quota exceeded|Too Many Requests)/i;

/**
 * Build a case-insensitive merged RegExp from literal/regex patterns.
 * @param {(RegExp|string)[]} patterns
 * @returns {RegExp}
 */
function buildCombinedPattern(patterns) {
  const patternSources = patterns.map(pattern => (pattern instanceof RegExp ? pattern.source : String(pattern))).filter(Boolean);
  return new RegExp(patternSources.join("|"), "i");
}

// Pattern: per-run LLM invocation cap exhausted.
// Matches both the Anthropic JSON error type ("max_runs_exceeded") and the
// human-readable message form ("Maximum LLM invocations exceeded") seen in
// both CAPI (Copilot CLI: "CAPIError: 429 Maximum LLM invocations exceeded (N/N)")
// and direct Anthropic API responses ("max_runs_exceeded").
// The pooled per-run invocation budget is saturated — retries cannot make progress.
const INVOCATION_CAP_EXCEEDED_PATTERN = buildCombinedPattern(MAX_RUNS_EXCEEDED_PATTERNS);

/**
 * Determines if the collected output contains the observed Copilot/CAPI quota exhaustion error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isCAPIQuotaExceededError(output) {
  return CAPI_QUOTA_EXCEEDED_PATTERN.test(output);
}

/**
 * Determines if the collected output indicates the per-run LLM invocation cap is exhausted.
 * This covers both the CAPI form ("CAPIError: 429 Maximum LLM invocations exceeded (N/N)")
 * and the Anthropic JSON form ("max_runs_exceeded"). The pooled budget cannot be recovered
 * within the current run — retrying is pointless.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isInvocationCapExceededError(output) {
  return isMaxRunsExceededError(output);
}

/**
 * Detect known error patterns in a log string and return detection results.
 * @param {string} logContent - Contents of the agent stdio log
 * @returns {{ inferenceAccessError: boolean, mcpPolicyError: boolean, agenticEngineTimeout: boolean, modelNotSupportedError: boolean, http400ResponseError: boolean, capiQuotaExceededError: boolean, invocationCapExceeded: boolean }}
 */
function detectErrors(logContent) {
  return {
    inferenceAccessError: INFERENCE_ACCESS_ERROR_PATTERN.test(logContent),
    mcpPolicyError: MCP_POLICY_BLOCKED_PATTERN.test(logContent),
    agenticEngineTimeout: AGENTIC_ENGINE_TIMEOUT_PATTERN.test(logContent),
    modelNotSupportedError: MODEL_NOT_SUPPORTED_PATTERN.test(logContent),
    http400ResponseError: HTTP_400_RESPONSE_ERROR_PATTERN.test(logContent),
    capiQuotaExceededError: isCAPIQuotaExceededError(logContent),
    invocationCapExceeded: isInvocationCapExceededError(logContent),
  };
}

/**
 * Build GitHub Actions output lines from detection results.
 * @param {{ inferenceAccessError: boolean, mcpPolicyError: boolean, agenticEngineTimeout: boolean, modelNotSupportedError: boolean, http400ResponseError: boolean, capiQuotaExceededError: boolean, invocationCapExceeded: boolean }} results
 * @returns {string[]}
 */
function buildOutputLines(results) {
  const effectiveCAPIQuotaExceeded = results.capiQuotaExceededError && !results.invocationCapExceeded;
  return [
    `inference_access_error=${results.inferenceAccessError}`,
    `mcp_policy_error=${results.mcpPolicyError}`,
    `agentic_engine_timeout=${results.agenticEngineTimeout}`,
    `model_not_supported_error=${results.modelNotSupportedError}`,
    `http_400_response_error=${results.http400ResponseError}`,
    `capi_quota_exceeded_error=${effectiveCAPIQuotaExceeded}`,
    `invocation_cap_exceeded=${results.invocationCapExceeded}`,
  ];
}

/**
 * Write GitHub Actions outputs to $GITHUB_OUTPUT.
 * @param {{ inferenceAccessError: boolean, mcpPolicyError: boolean, agenticEngineTimeout: boolean, modelNotSupportedError: boolean, http400ResponseError: boolean, capiQuotaExceededError: boolean, invocationCapExceeded: boolean }} results
 */
function writeOutputs(results) {
  const outputFile = process.env.GITHUB_OUTPUT;
  if (!outputFile) {
    process.stderr.write("[detect-agent-errors] GITHUB_OUTPUT not set — skipping output\n");
    return;
  }

  const lines = buildOutputLines(results);
  try {
    fs.appendFileSync(outputFile, lines.join("\n") + "\n");
  } catch (err) {
    process.stderr.write(`[detect-agent-errors] Failed to write to GITHUB_OUTPUT: ${String(err)}\n`);
  }
}

function main() {
  let logContent = "";

  if (fs.existsSync(LOG_FILE)) {
    try {
      logContent = fs.readFileSync(LOG_FILE, "utf8");
    } catch (err) {
      throw new Error(`Failed to read file ${LOG_FILE}: ${String(err)}`, { cause: err });
    }
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
    process.stderr.write("[detect-agent-errors] Detected agentic engine timeout signature in agent log\n");
  }
  if (results.modelNotSupportedError) {
    process.stderr.write("[detect-agent-errors] Detected model configuration error: configured model is invalid or unavailable for this engine/account\n");
  }
  if (results.http400ResponseError) {
    process.stderr.write("[detect-agent-errors] Detected HTTP 400 response error in agent log\n");
  }
  if (results.capiQuotaExceededError) {
    process.stderr.write("[detect-agent-errors] Detected CAPI quota exhaustion: Copilot quota has been exceeded\n");
  }
  if (results.invocationCapExceeded) {
    process.stderr.write("[detect-agent-errors] Detected invocation cap exhaustion: the pooled per-run LLM invocation budget is fully saturated\n");
  }

  writeOutputs(results);
}

if (require.main === module) {
  main();
}

module.exports = {
  detectErrors,
  isCAPIQuotaExceededError,
  isInvocationCapExceededError,
  INFERENCE_ACCESS_ERROR_PATTERN,
  MCP_POLICY_BLOCKED_PATTERN,
  AGENTIC_ENGINE_TIMEOUT_PATTERN,
  MODEL_NOT_SUPPORTED_PATTERN,
  HTTP_400_RESPONSE_ERROR_PATTERN,
  CAPI_QUOTA_EXCEEDED_PATTERN,
  INVOCATION_CAP_EXCEEDED_PATTERN,
  buildOutputLines,
};
