// @ts-check

/**
 * Detect Copilot CLI errors in the agent stdio log.
 *
 * Scans the agent stdio log for known error patterns and sets GitHub Actions
 * output variables for each detected error class:
 *
 *   - inference_access_error: The COPILOT_GITHUB_TOKEN does not have valid
 *     access to inference (e.g., "Access denied by policy settings").
 *   - mcp_policy_error: MCP servers were blocked by enterprise/organization
 *     policy (e.g., "MCP servers were blocked by policy: 'github', 'safeoutputs'").
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

/**
 * Detect known error patterns in a log string and return detection results.
 * @param {string} logContent - Contents of the agent stdio log
 * @returns {{ inferenceAccessError: boolean, mcpPolicyError: boolean }}
 */
function detectErrors(logContent) {
  return {
    inferenceAccessError: INFERENCE_ACCESS_ERROR_PATTERN.test(logContent),
    mcpPolicyError: MCP_POLICY_BLOCKED_PATTERN.test(logContent),
  };
}

/**
 * Write GitHub Actions outputs to $GITHUB_OUTPUT.
 * @param {{ inferenceAccessError: boolean, mcpPolicyError: boolean }} results
 */
function writeOutputs(results) {
  const outputFile = process.env.GITHUB_OUTPUT;
  if (!outputFile) {
    process.stderr.write("[detect-copilot-errors] GITHUB_OUTPUT not set — skipping output\n");
    return;
  }

  const lines = [`inference_access_error=${results.inferenceAccessError}`, `mcp_policy_error=${results.mcpPolicyError}`];
  fs.appendFileSync(outputFile, lines.join("\n") + "\n");
}

function main() {
  let logContent = "";

  if (fs.existsSync(LOG_FILE)) {
    logContent = fs.readFileSync(LOG_FILE, "utf8");
  } else {
    process.stderr.write(`[detect-copilot-errors] Log file not found: ${LOG_FILE}\n`);
  }

  const results = detectErrors(logContent);

  if (results.inferenceAccessError) {
    process.stderr.write("[detect-copilot-errors] Detected inference access error in agent log\n");
  }
  if (results.mcpPolicyError) {
    process.stderr.write("[detect-copilot-errors] Detected MCP policy error in agent log\n");
  }

  writeOutputs(results);
}

main();

module.exports = { detectErrors, INFERENCE_ACCESS_ERROR_PATTERN, MCP_POLICY_BLOCKED_PATTERN };
